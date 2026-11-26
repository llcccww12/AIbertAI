package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type dshUserSideEffectReview struct {
	Decision    string    `json:"decision"`
	Fingerprint string    `json:"fingerprint"`
	ReviewedAt  time.Time `json:"reviewedAt"`
}

func validDSHUserSideEffectReview(r dshUserSideEffectReview) bool {
	if (r.Decision != "observed_applied" && r.Decision != "observed_not_applied") || len(r.Fingerprint) != 64 || r.ReviewedAt.IsZero() {
		return false
	}
	b, e := hex.DecodeString(r.Fingerprint)
	return e == nil && len(b) == 32 && hex.EncodeToString(b) == r.Fingerprint
}

// Caller must hold live.mu. A recorded stop exists only after this host has
// killed and waited for its own SDK child. The exclusive store lock prevents
// another host from operating on the same account profile concurrently.
func (h *relaySessionHost) dshManualReviewLocked(id string, reserved bool) (dshSideEffectReview, acpSession, error) {
	fail := func() (dshSideEffectReview, acpSession, error) {
		return dshSideEffectReview{}, acpSession{}, errors.New("dsh_manual_review_unavailable")
	}
	l := h.live
	if l == nil || l.store == nil || l.store.Root == nil || l.store.lock == nil || l.dsh == nil || !validUUID(id) {
		return fail()
	}
	if _, err := l.store.lock.Stat(); err != nil {
		return fail()
	}
	row, ok := l.sessions[id]
	if !ok || l.restarting || l.compacting[id] && !reserved || l.activeTurns[id] != nil || row.Engine != "dsh" || row.State == "running" || desktopGoalAdvancing(row.Goal) || len(row.Turns) == 0 {
		return fail()
	}
	last := row.Turns[len(row.Turns)-1]
	if !last.Failed || !last.Acked || !dshRecoverableWorkspaceFailure(last.FailureCode) || last.DSHReconciled || last.DSHUserReviewed || l.dshStoppedTurns[id] != last.Operation {
		return fail()
	}
	review, err := dshInspectSideEffect(l.store.Root, id)
	if err != nil || review.Target != nil || review.OperationID != last.Operation || (review.ToolName != "write" && review.ToolName != "edit") || review.WorkspaceRoot != row.CWD {
		return fail()
	}
	state, content, err := h.snapshotDSHReviewFile(row, review.Arguments)
	if err != nil {
		return fail()
	}
	review.ObservedFileState, review.ObservedContent = state, content
	original := review.Fingerprint
	review.Fingerprint = ""
	data, err := json.Marshal(struct {
		Previous string              `json:"previous"`
		Review   dshSideEffectReview `json:"review"`
	}{original, review})
	if err != nil || len(data) > (5<<20)-(64<<10) {
		return fail()
	}
	digest := sha256.Sum256(data)
	review.Fingerprint = fmt.Sprintf("%x", digest[:])
	return review, row, nil
}
func (h *relaySessionHost) inspectDSHManualReview(id string) (dshSideEffectReview, error) {
	if h.live == nil {
		return dshSideEffectReview{}, errors.New("dsh_manual_review_unavailable")
	}
	h.live.mu.Lock()
	defer h.live.mu.Unlock()
	review, _, err := h.dshManualReviewLocked(id, false)
	return review, err
}

// confirmDSHManualReview records the user's observation, never an SDK success.
// Rechecking after gateway reconciliation rejects stale snapshots/late results.
func (h *relaySessionHost) confirmDSHManualReview(ctx context.Context, id, fingerprint, decision string) error {
	fail := func() error { return errors.New("dsh_manual_review_unavailable") }
	if decision != "observed_applied" && decision != "observed_not_applied" || h.live == nil {
		return fail()
	}
	l := h.live
	l.mu.Lock()
	review, row, err := h.dshManualReviewLocked(id, false)
	if err != nil || review.Fingerprint != fingerprint {
		l.mu.Unlock()
		return fail()
	}
	snapshot, err := json.Marshal(row)
	if err != nil {
		l.mu.Unlock()
		return fail()
	}
	var detached acpSession
	if json.Unmarshal(snapshot, &detached) != nil {
		l.mu.Unlock()
		return fail()
	}
	if l.compacting == nil {
		l.compacting = map[string]bool{}
	}
	l.compacting[id] = true
	l.mu.Unlock()
	defer func() { l.mu.Lock(); delete(l.compacting, id); l.mu.Unlock() }()
	// The SDK is resumed only to reconcile model accounting. No prompt or tool
	// is issued; an unknown model dispatch makes this return an error.
	if err = reconcileDSHSession(ctx, l.dsh, detached); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	current, ok := l.sessions[id]
	nowBytes, e := json.Marshal(current)
	if !ok || e != nil || string(nowBytes) != string(snapshot) || l.activeTurns[id] != nil || l.dshStoppedTurns[id] != review.OperationID {
		return fail()
	}
	fresh, _, err := h.dshManualReviewLocked(id, true)
	if err != nil || fresh.Fingerprint != fingerprint || fresh.Target != nil || fresh.OperationID != review.OperationID {
		return fail()
	}
	claim, err := connectorPending(l.store.Root, id, review.OperationID, review.CallID)
	if err != nil || claim.Target != nil || claim.State != "dispatch_claimed" && claim.State != "result_ready" {
		return fail()
	}
	claim.Kind = "workspace"
	claim.State = "user_reviewed"
	claim.Review = &dshUserSideEffectReview{Decision: decision, Fingerprint: fingerprint, ReviewedAt: time.Now().UTC()}
	if err = writeConnectorClaim(l.store.Root, claim); err != nil {
		return fail()
	}
	_, err = h.finishRecordedDSHManualReviewLocked(id)
	return err
}

// Caller holds live.mu. A durable user_reviewed record is a checkpoint that the
// stop, authorization, gateway, and explicit decision checks already passed.
func (h *relaySessionHost) finishRecordedDSHManualReviewLocked(id string) (bool, error) {
	l := h.live
	if l == nil || l.store == nil || l.store.Root == nil {
		return false, nil
	}
	row, ok := l.sessions[id]
	if !ok || len(row.Turns) == 0 {
		return false, nil
	}
	last := row.Turns[len(row.Turns)-1]
	var claim dshConnectorClaim
	matches := 0
	for _, run := range last.ToolRuns {
		if run.Call == nil || run.DSHCallID == "" || (run.Call.Function.Name != "write" && run.Call.Function.Name != "edit") {
			continue
		}
		c, err := readConnectorClaim(l.store.Root, connectorClaimPath(id, last.Operation, run.DSHCallID), id)
		if err == nil && c.State == "user_reviewed" {
			claim = c
			matches++
		}
	}
	if matches == 0 {
		return false, nil
	}
	if l.store.lock == nil {
		return true, errors.New("dsh_manual_review_unavailable")
	}
	if _, err := l.store.lock.Stat(); err != nil {
		return true, errors.New("dsh_manual_review_unavailable")
	}
	if matches != 1 || len(last.ToolRuns) == 0 || last.ToolRuns[len(last.ToolRuns)-1].DSHCallID != claim.CallID || !last.Failed || !last.Acked || !dshRecoverableWorkspaceFailure(last.FailureCode) {
		return true, errors.New("dsh_manual_review_unavailable")
	}
	for i, run := range last.ToolRuns {
		if run.Pending && i != len(last.ToolRuns)-1 {
			return true, errors.New("dsh_manual_review_unavailable")
		}
	}
	marker, err := readConnectorClaim(l.store.Root, "dsh-connector-dispatch/"+id+"/.pending", id)
	if err == nil {
		if marker.OperationID != claim.OperationID || marker.CallID != claim.CallID || marker.State != "dispatch_claimed" {
			return true, errors.New("dsh_manual_review_unavailable")
		}
		if err = l.store.Root.Remove("dsh-connector-dispatch/" + id + "/.pending"); err != nil {
			return true, err
		}
		if err = syncConnectorDir(l.store.Root, "dsh-connector-dispatch/"+id); err != nil {
			return true, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return true, err
	}
	if err = dshConnectorAssertSettled(l.store.Root, id); err != nil {
		return true, err
	}
	row.Turns = append([]acpTurn(nil), row.Turns...)
	row.Turns[len(row.Turns)-1].DSHUserReviewed = true
	if err = l.save(row); err != nil {
		return true, err
	}
	return true, nil
}

// The SDK may project tool/result before its separate proof arrives. In that
// case Pending is already false even though the shared claim is unresolved.
func dshUnsettledWorkspaceWrite(root *os.Root, session string, turn acpTurn) bool {
	if root == nil {
		return false
	}
	claims, err := dshConnectorClaims(root, session)
	if err != nil {
		for _, run := range turn.ToolRuns {
			if run.Call != nil && (run.Call.Function.Name == "write" || run.Call.Function.Name == "edit") {
				return true
			}
		}
		return false
	}
	for _, claim := range claims {
		if claim.OperationID != turn.Operation || claim.State == "reconciled" || claim.State == "user_reviewed" {
			continue
		}
		for _, run := range turn.ToolRuns {
			if run.DSHCallID == claim.CallID && run.Call != nil && (run.Call.Function.Name == "write" || run.Call.Function.Name == "edit") {
				return true
			}
		}
	}
	return false
}

// Snapshot the approved workspace path without following symlinks, and bind
// what the user sees to the subsequent confirmation. A concurrent change fails.
func (h *relaySessionHost) snapshotDSHReviewFile(row acpSession, raw json.RawMessage) (string, *string, error) {
	if h.live.workspaces == nil {
		return "", nil, errors.New("dsh_manual_review_unavailable")
	}
	var args struct {
		Path string `json:"file_path"`
	}
	if json.Unmarshal(raw, &args) != nil || args.Path == "" {
		return "", nil, errors.New("dsh_manual_review_unavailable")
	}
	relative := args.Path
	if filepath.IsAbs(relative) {
		var err error
		relative, err = filepath.Rel(row.CWD, relative)
		if err != nil {
			return "", nil, err
		}
		relative = filepath.ToSlash(relative)
	}
	if _, err := workspaceRelative(relative); err != nil || relative == "." {
		return "", nil, errors.New("dsh_manual_review_unavailable")
	}
	root, err := h.live.workspaces.openSessionRoot(row)
	if err != nil {
		return "", nil, err
	}
	defer root.Close()
	before, err := dshWorkspaceTarget(root, relative)
	if err != nil {
		return "", nil, err
	}
	if before.File == nil {
		return "missing", nil, nil
	}
	f, err := openAttachmentFile(root, filepath.FromSlash(relative), maxDSHWorkspaceText)
	if err != nil {
		return "", nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(f, maxDSHWorkspaceText+1))
	if err = errors.Join(readErr, f.Close()); err != nil {
		return "", nil, err
	}
	content := string(data)
	if !validDSHWorkspaceText(&content, maxDSHWorkspaceText) || strings.ContainsRune(content, 0) {
		return "", nil, errors.New("dsh_manual_review_unavailable")
	}
	after, err := dshWorkspaceTarget(root, relative)
	if err != nil || !sameDSHTarget(before, after) {
		return "", nil, errors.New("dsh_manual_review_unavailable")
	}
	return "file", &content, nil
}
