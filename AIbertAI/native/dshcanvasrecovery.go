package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// History-only recovery. The caller must authorize and exclude concurrent turns.
func reconcileDSHCanvasSession(ctx context.Context, cfg *dshRuntimeConfig, root *os.Root, session acpSession) error {
	options, _ := ctx.Value(dshSDKOptionsKey{}).(dshSDKOptions)
	previous := options.AfterReconcile
	options.AfterReconcile = func(ctx context.Context, p *dshProcess) error {
		if previous != nil {
			if err := previous(ctx, p); err != nil {
				return err
			}
		}
		return dshRecoverCanvasClaim(ctx, root, session, func(ctx context.Context, params map[string]any) (json.RawMessage, error) {
			return p.call(ctx, "aibertai/history", params, nil)
		})
	}
	return reconcileDSHSession(context.WithValue(ctx, dshSDKOptionsKey{}, options), cfg, session)
}

func dshRecoverCanvasClaim(ctx context.Context, root *os.Root, session acpSession, fetch dshConnectorHistoryFetch) error {
	fail := func() error { return errors.New("dsh_canvas_reconciliation_required") }
	if root == nil || session.ID != session.DSHSessionID || !validUUID(session.ID) || fetch == nil {
		return fail()
	}
	marker, err := readConnectorClaim(root, "dsh-connector-dispatch/"+session.ID+"/.pending", session.ID)
	pending := err == nil
	var claim dshConnectorClaim
	if pending {
		claim, err = connectorPending(root, session.ID, marker.OperationID, marker.CallID)
	} else if errors.Is(err, os.ErrNotExist) && len(session.Turns) > 0 {
		last := session.Turns[len(session.Turns)-1]
		if len(last.ToolRuns) == 0 {
			return fail()
		}
		run := last.ToolRuns[len(last.ToolRuns)-1]
		claim, err = readConnectorClaim(root, connectorClaimPath(session.ID, last.Operation, run.DSHCallID), session.ID)
		if err == nil && (claim.State != "reconciled" || dshConnectorAssertSettled(root, session.ID) != nil) {
			return fail()
		}
	} else {
		return fail()
	}
	if err != nil || claim.Kind != "canvas" || (claim.State != "result_ready" && claim.State != "reconciled") {
		return fail()
	}
	if len(session.Turns) == 0 {
		return fail()
	}
	last := session.Turns[len(session.Turns)-1]
	if last.Operation != claim.OperationID || len(last.ToolRuns) == 0 || last.ToolRuns[len(last.ToolRuns)-1].DSHCallID != claim.CallID {
		return fail()
	}
	var matched *workspaceToolRun
	count := 0
	for _, turn := range session.Turns {
		if turn.Operation != claim.OperationID {
			continue
		}
		for i := range turn.ToolRuns {
			if turn.ToolRuns[i].DSHCallID == claim.CallID {
				matched = &turn.ToolRuns[i]
				count++
			}
		}
	}
	if count != 1 || matched == nil || matched.Call == nil || !designWriteTool(matched.Call.Function.Name) {
		return fail()
	}
	read := func(s, o, c string) (json.RawMessage, error) {
		return dshMatchConnectorHistory(ctx, s, o, c, matched.Call.Function.Name, json.RawMessage(matched.Call.Function.Arguments), fetch)
	}
	if pending {
		return dshConnectorReconcile(root, session.ID, claim.OperationID, claim.CallID, read)
	}
	raw, err := read(session.ID, claim.OperationID, claim.CallID)
	var event struct {
		Event struct {
			Seq  *int64 `json:"seq"`
			Data struct {
				Message struct {
					IsError bool `json:"isError"`
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content"`
				} `json:"message"`
			} `json:"data"`
		} `json:"event"`
	}
	if err != nil || json.Unmarshal(raw, &event) != nil || event.Event.Seq == nil || claim.SDKSeq == nil || *event.Event.Seq != *claim.SDKSeq || event.Event.Data.Message.IsError || len(event.Event.Data.Message.Content) != 1 || event.Event.Data.Message.Content[0].Type != "text" || claim.Output == nil || dshCanvasResultDigest(event.Event.Data.Message.Content[0].Text) != *claim.Output {
		return fail()
	}
	return nil
}

func dshVerifyCanvasResult(ctx context.Context, claim dshConnectorClaim, run workspaceToolRun, fetch dshConnectorHistoryFetch) error {
	fail := func() error { return errors.New("dsh_canvas_reconciliation_required") }
	if claim.Kind != "canvas" || (claim.State != "result_ready" && claim.State != "reconciled") || claim.Output == nil || run.Call == nil || !dshRecoverableCanvasTool(run.Call.Function.Name) || run.DSHCallID != claim.CallID {
		return fail()
	}
	raw, err := dshMatchConnectorHistory(ctx, claim.SessionID, claim.OperationID, claim.CallID, run.Call.Function.Name, json.RawMessage(run.Call.Function.Arguments), fetch)
	var data struct {
		Event struct {
			Seq  *int64 `json:"seq"`
			Data struct {
				Message struct {
					IsError bool `json:"isError"`
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content"`
				} `json:"message"`
			} `json:"data"`
		} `json:"event"`
	}
	if err != nil || json.Unmarshal(raw, &data) != nil || data.Event.Seq == nil || data.Event.Data.Message.IsError || len(data.Event.Data.Message.Content) != 1 || data.Event.Data.Message.Content[0].Type != "text" || dshCanvasResultDigest(data.Event.Data.Message.Content[0].Text) != *claim.Output || claim.State == "reconciled" && (claim.SDKSeq == nil || *claim.SDKSeq != *data.Event.Seq) {
		return fail()
	}
	return nil
}

// Derived from the dispatch allowlist so a newly guarded canvas write can never
// succeed at the bridge yet stay unreconcilable after an unknown outcome.
func dshRecoverableCanvasTool(name string) bool {
	return dshCanvasGuarded(name) && designWriteTool(name) || dshCanvasEvidenceWrite(name)
}

// A guarded write that never produced a result cannot be proven by SDK history.
// The local canvas is the remaining evidence: if it still matches the digest
// captured before dispatch, nothing was applied.
func dshCanvasSettlementDecision(root *os.Root, session acpSession, claim dshConnectorClaim) (string, error) {
	fail := errors.New("dsh_canvas_reconciliation_required")
	if root == nil || claim.Kind != "canvas" || claim.State != "dispatch_claimed" || claim.CanvasBefore == nil {
		return "", fail
	}
	var marker struct{ CanvasID string }
	if design := workspaceDesign(root, session.ID); len(design) == 0 || json.Unmarshal(design, &marker) != nil || marker.CanvasID == "" || filepath.Base(marker.CanvasID) != marker.CanvasID {
		return "", fail
	}
	canvas := readDesignJSON(root, filepath.Join(workspaceDataDir, "design", marker.CanvasID, "canvas.json"), maxDesktopSessionBytes)
	if canvas == nil {
		return "", fail
	}
	if fmt.Sprintf("%x", sha256.Sum256(canvas)) == *claim.CanvasBefore {
		return "observed_not_applied", nil
	}
	return "observed_applied", nil
}

// Admission-time settlement. The user's next message is refused while a canvas
// write is unresolved, so the same evidence check runs before rejecting it.
func (h *relaySessionHost) settleStaleCanvasTurn(id string) error {
	fail := errors.New("dsh_canvas_reconciliation_required")
	if h.live == nil || h.live.store == nil || h.live.store.Root == nil || !validUUID(id) {
		return fail
	}
	live := h.live
	live.mu.Lock()
	defer live.mu.Unlock()
	row, ok := live.sessions[id]
	if !ok || live.restarting || live.compacting[id] || live.activeTurns[id] != nil || !dshCanvasRecoveryAvailable(row) || !row.Turns[len(row.Turns)-1].Acked {
		return fail
	}
	if live.workspaces == nil {
		return fail
	}
	workspace, err := live.workspaces.sessionRoot(row)
	if err != nil {
		return fail
	}
	settled, err := dshSettleStaleCanvasWrite(live.store.Root, workspace, row)
	workspace.Close()
	if err != nil || !settled {
		return fail
	}
	row.Turns = append([]acpTurn(nil), row.Turns...)
	row.Turns[len(row.Turns)-1].DSHUserReviewed = true
	return live.save(row)
}

// Settles an unrecoverable canvas write into user_reviewed so the session can
// accept the next message. It never replays the write and never invents a
// result: the decision records what the canvas showed.
// root holds the dispatch claims; workspace holds the canvas evidence.
func dshSettleStaleCanvasWrite(root, workspace *os.Root, session acpSession) (bool, error) {
	if root == nil || workspace == nil || len(session.Turns) == 0 || !validUUID(session.ID) {
		return false, nil
	}
	last := session.Turns[len(session.Turns)-1]
	if !last.Failed || !last.Acked || last.DSHReconciled || last.DSHUserReviewed || last.FailureCode != "dsh_canvas_outcome_unknown" || len(last.ToolRuns) == 0 || !validUUID(last.Operation) {
		return false, nil
	}
	run := last.ToolRuns[len(last.ToolRuns)-1]
	if run.Call == nil || run.DSHCallID == "" || !dshRecoverableCanvasTool(run.Call.Function.Name) {
		return false, nil
	}
	claim, err := connectorPending(root, session.ID, last.Operation, run.DSHCallID)
	if err != nil || claim.State != "dispatch_claimed" || claim.Kind != "canvas" || claim.CanvasBefore == nil {
		return false, nil
	}
	decision, err := dshCanvasSettlementDecision(workspace, session, claim)
	if err != nil {
		return false, err
	}
	if err = dshCanvasSettleClaim(root, session.ID, claim, decision); err != nil {
		return false, err
	}
	return true, nil
}

func dshCanvasSettleClaim(root *os.Root, sessionID string, claim dshConnectorClaim, decision string) error {
	sum := sha256.Sum256([]byte(strings.Join([]string{sessionID, claim.OperationID, claim.CallID, decision, *claim.CanvasBefore}, "\x00")))
	claim.Review = &dshUserSideEffectReview{Decision: decision, Fingerprint: fmt.Sprintf("%x", sum), ReviewedAt: time.Now().UTC()}
	claim.State = "user_reviewed"
	if err := writeConnectorClaim(root, claim); err != nil {
		return err
	}
	if err := root.Remove("dsh-connector-dispatch/" + sessionID + "/.pending"); err != nil {
		return err
	}
	return syncConnectorDir(root, "dsh-connector-dispatch/"+sessionID)
}

// dshCanvasDefiniteRejection reports an explicit answer from the design worker
// that it did not complete the call. The worker saves media and applies canvas
// changes only on success, and canvas changes are still checked against the
// pre-dispatch digest. A timeout, an opaque failure or an oversized result is
// not definite: the call may have completed.
func dshCanvasDefiniteRejection(err error) bool {
	if err == nil {
		return false
	}
	switch code := err.Error(); code {
	case "design_tool_unavailable", "design_tool_failed", "design_tool_result_too_large":
		return false
	default:
		return strings.HasPrefix(code, "design_") && !strings.ContainsAny(code, " \n")
	}
}

// dshCanvasReleaseRejected settles the claim of a rejected guarded write as not
// applied, but only when the canvas still matches the digest taken before dispatch.
// The claim lives under effect.root; the canvas is read from workspace.
func dshCanvasReleaseRejected(effect *dshSideEffect, workspace *os.Root, session acpSession) error {
	fail := errors.New("dsh_canvas_reconciliation_required")
	if effect == nil || workspace == nil {
		return fail
	}
	claim, err := connectorPending(effect.root, effect.session, effect.operation, effect.call)
	if err != nil || claim.State != "dispatch_claimed" || claim.Kind != "canvas" || claim.CanvasBefore == nil {
		return fail
	}
	if *claim.CanvasBefore == fmt.Sprintf("%x", sha256.Sum256(nil)) {
		// Dispatched without a canvas: there is no canvas to compare, and the
		// refusal itself is the evidence that nothing was generated.
		if len(workspaceDesign(workspace, session.ID)) != 0 {
			return fail
		}
	} else if decision, err := dshCanvasSettlementDecision(workspace, session, claim); err != nil || decision != "observed_not_applied" {
		return fail
	}
	return dshCanvasSettleClaim(effect.root, effect.session, claim, "observed_not_applied")
}

func dshPendingCanvasWrite(turn acpTurn) bool {
	for _, run := range turn.ToolRuns {
		if run.Pending && run.Call != nil && dshRecoverableCanvasTool(run.Call.Function.Name) {
			return true
		}
	}
	return false
}

func dshCanvasRecoveryAvailable(row acpSession) bool {
	if row.Engine != "dsh" || row.State == "running" || row.GoalRun != "" || row.Goal != nil || len(row.Turns) == 0 {
		return false
	}
	last := row.Turns[len(row.Turns)-1]
	if !last.Failed || last.DSHReconciled || (last.FailureCode != "dsh_canvas_outcome_unknown" && last.FailureCode != "dsh_runtime_exited") || len(last.ToolRuns) == 0 {
		return false
	}
	run := last.ToolRuns[len(last.ToolRuns)-1]
	return run.Call != nil && run.DSHCallID != "" && designWriteTool(run.Call.Function.Name)
}

func (h *relaySessionHost) recoverDSHCanvasSession(ctx context.Context, id string) error {
	fail := func() error { return errors.New("dsh_canvas_reconciliation_required") }
	if h.live == nil || h.live.store == nil || h.live.store.Root == nil || h.live.dsh == nil || !validUUID(id) {
		return fail()
	}
	live := h.live
	// The interrupted turn must be reported before evidence can release it.
	live.mu.Lock()
	row, ok := live.sessions[id]
	needsReport := ok && dshCanvasRecoveryAvailable(row) && !row.Turns[len(row.Turns)-1].Acked
	live.mu.Unlock()
	if needsReport && h.retryPendingReport(ctx, id) != nil {
		return fail()
	}
	live.mu.Lock()
	row, ok = live.sessions[id]
	if !ok || live.restarting || live.compacting[id] || live.activeTurns[id] != nil || !dshCanvasRecoveryAvailable(row) || !row.Turns[len(row.Turns)-1].Acked {
		live.mu.Unlock()
		return fail()
	}
	snapshot, err := json.Marshal(row)
	if err != nil {
		live.mu.Unlock()
		return fail()
	}
	var detached acpSession
	if json.Unmarshal(snapshot, &detached) != nil {
		live.mu.Unlock()
		return fail()
	}
	if live.compacting == nil {
		live.compacting = map[string]bool{}
	}
	live.compacting[id] = true
	live.mu.Unlock()
	defer func() { live.mu.Lock(); delete(live.compacting, id); live.mu.Unlock() }()
	// A write that never produced a result cannot be proven by SDK history. The
	// canvas decides instead, so the session is never locked without an exit.
	if live.workspaces == nil {
		return fail()
	}
	workspace, settleErr := live.workspaces.sessionRoot(detached)
	if settleErr != nil {
		return fail()
	}
	settled, settleErr := dshSettleStaleCanvasWrite(live.store.Root, workspace, detached)
	workspace.Close()
	if settleErr != nil {
		return fail()
	}
	if settled {
		live.mu.Lock()
		defer live.mu.Unlock()
		current, ok := live.sessions[id]
		currentBytes, err := json.Marshal(current)
		if !ok || err != nil || string(currentBytes) != string(snapshot) {
			return fail()
		}
		current.Turns = append([]acpTurn(nil), current.Turns...)
		current.Turns[len(current.Turns)-1].DSHUserReviewed = true
		if live.save(current) != nil {
			return fail()
		}
		return nil
	}
	var rebuilt acpTurn
	var projectionSeq int64
	options := dshSDKOptions{AfterReconcile: func(ctx context.Context, p *dshProcess) error {
		fetch := func(ctx context.Context, params map[string]any) (json.RawMessage, error) {
			return p.call(ctx, "aibertai/history", params, nil)
		}
		var events []dshConnectorHistoryEvent
		var raws []json.RawMessage
		through, e := dshRepairHistory(ctx, detached.DSHSessionID, -1, fetch, func(_ int64, raw json.RawMessage) error {
			var event dshConnectorHistoryEvent
			if json.Unmarshal(raw, &event) != nil {
				return fail()
			}
			events = append(events, event)
			raws = append(raws, raw)
			return nil
		})
		if e != nil {
			return fail()
		}
		last := detached.Turns[len(detached.Turns)-1]
		verify := func(run workspaceToolRun) error {
			claim, e := readConnectorClaim(live.store.Root, connectorClaimPath(id, last.Operation, run.DSHCallID), id)
			if e != nil {
				return fail()
			}
			if run.Call != nil && (run.Call.Function.Name == "write" || run.Call.Function.Name == "edit") {
				seq, e := dshVerifyWorkspaceProof(ctx, claim, run.Call.Function.Name, json.RawMessage(run.Call.Function.Arguments), fetch)
				if e != nil || claim.State == "reconciled" && (claim.SDKSeq == nil || *claim.SDKSeq != seq) {
					return fail()
				}
				return nil
			}
			return dshVerifyCanvasResult(ctx, claim, run, fetch)
		}
		rebuilt, projectionSeq, e = dshRebuildPromptTurn(detached.DSHSessionID, last, events, raws, verify)
		if e != nil || projectionSeq != through {
			return fail()
		}
		return nil
	}}
	ctx = context.WithValue(ctx, dshSDKOptionsKey{}, options)
	if err = reconcileDSHCanvasSession(ctx, live.dsh, live.store.Root, detached); err != nil {
		return fail()
	}
	if dshConnectorAssertSettled(live.store.Root, id) != nil {
		return fail()
	}
	live.mu.Lock()
	defer live.mu.Unlock()
	current, ok := live.sessions[id]
	currentBytes, err := json.Marshal(current)
	if !ok || err != nil || string(currentBytes) != string(snapshot) {
		return fail()
	}
	current.Turns = append([]acpTurn(nil), current.Turns...)
	rebuilt.DSHProjectionSeq = &projectionSeq
	rebuilt.DSHReconciled = true
	current.Turns[len(current.Turns)-1] = rebuilt
	if live.save(current) != nil {
		return fail()
	}
	return nil
}
