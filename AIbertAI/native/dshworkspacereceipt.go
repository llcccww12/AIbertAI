package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// A prepared value is validated before acquiring the shared claim and before
// writing the file. Only its digest and encoded length are retained. No
// text-only connector receipt may release this claim.
type dshWorkspaceReceiptValue struct {
	Path      string  `json:"path"`
	Operation string  `json:"operation,omitempty"`
	Before    *string `json:"before"`
	After     string  `json:"after"`
}

func prepareDSHWorkspaceReceipt(value dshWorkspaceReceiptValue) (string, error) {
	if value.Path == "" || !utf8.ValidString(value.Path) || !utf8.ValidString(value.After) || strings.ContainsRune(value.After, 0) || len(value.After) > maxDSHWorkspaceText || value.Before != nil && (!utf8.ValidString(*value.Before) || strings.ContainsRune(*value.Before, 0) || len(*value.Before) > maxDSHWorkspaceText) || value.Operation != "" && value.Operation != "create" && value.Operation != "update" || value.Operation == "create" && value.Before != nil || value.Operation == "update" && value.Before == nil || value.Operation == "" && value.Before == nil {
		return "", errors.New("dsh_workspace_receipt_invalid")
	}
	// Existing history consumers accept events up to 512 KiB. Reject before
	// acquiring a claim when the SDK's complete diff cannot fit that bound.
	encoded, err := json.Marshal(value)
	path, perr := json.Marshal(value.Path)
	if err != nil || perr != nil || len(encoded)+2*len(path) > (512<<10)-(64<<10) {
		return "", errors.New("dsh_workspace_receipt_invalid")
	}
	// Versioned binary encoding: four length-prefixed UTF-8 fields, with a
	// one-byte presence marker for before. Lengths are unsigned big-endian.
	h := sha256.New()
	var length uint64
	put := func(text string) {
		var n [8]byte
		binary.BigEndian.PutUint64(n[:], uint64(len(text)))
		h.Write(n[:])
		h.Write([]byte(text))
		length += uint64(len(n) + len(text))
	}
	put(value.Path)
	put(value.Operation)
	if value.Before == nil {
		h.Write([]byte{0})
		length++
	} else {
		h.Write([]byte{1})
		length++
		put(*value.Before)
	}
	put(value.After)
	return fmt.Sprintf("v1:%d:%x", length, h.Sum(nil)), nil
}

// completeWorkspace records the exact host value after the actual mutation.
// A crash or persistence failure leaves the exclusive claim unresolved.
func (effect *dshSideEffect) completeWorkspace(value dshWorkspaceReceiptValue) error {
	if effect == nil {
		return errors.New("dsh_connector_outcome_unknown")
	}
	prepared, err := prepareDSHWorkspaceReceipt(value)
	if err != nil {
		return errors.New("dsh_connector_outcome_unknown")
	}
	claim, err := connectorPending(effect.root, effect.session, effect.operation, effect.call)
	if err != nil || claim.State != "dispatch_claimed" {
		return errors.New("dsh_connector_outcome_unknown")
	}
	claim.State, claim.Kind, claim.Output = "result_ready", "workspace", &prepared
	if err = writeConnectorClaim(effect.root, claim); err != nil {
		return errors.Join(errors.New("dsh_connector_outcome_unknown"), err)
	}
	return nil
}

// Reconcile only against the owning SDK's flushed, fixed-ceiling history. The
// existing matcher proves operation -> turn -> unique call -> durable result;
// the new event separately binds the exact persisted host-value digest.
func dshWorkspaceReconcile(ctx context.Context, root *os.Root, session, operation, call, name string, args json.RawMessage, fetch dshConnectorHistoryFetch) error {
	return dshWorkspaceReconcileBound(ctx, root, session, operation, operation, call, name, args, fetch)
}

func dshWorkspaceReconcileBound(ctx context.Context, root *os.Root, session, operation, historyOperation, call, name string, args json.RawMessage, fetch dshConnectorHistoryFetch) error {
	claim, err := connectorPending(root, session, operation, call)
	if err != nil {
		return errors.New("dsh_workspace_reconciliation_required")
	}
	seq, err := dshVerifyWorkspaceProofBound(ctx, claim, historyOperation, name, args, fetch)
	if err != nil {
		return err
	}
	return dshReleaseSideEffect(root, claim, seq)
}

func dshVerifyWorkspaceProof(ctx context.Context, claim dshConnectorClaim, name string, args json.RawMessage, fetch dshConnectorHistoryFetch) (int64, error) {
	return dshVerifyWorkspaceProofBound(ctx, claim, claim.OperationID, name, args, fetch)
}

func dshVerifyWorkspaceProofBound(ctx context.Context, claim dshConnectorClaim, historyOperation, name string, args json.RawMessage, fetch dshConnectorHistoryFetch) (int64, error) {
	fail := func() error { return errors.New("dsh_workspace_reconciliation_required") }
	if claim.Kind != "workspace" || claim.Output == nil || (claim.State != "result_ready" && claim.State != "reconciled") || !validUUID(historyOperation) || fetch == nil || name != "write" && name != "edit" {
		return 0, fail()
	}
	if claim.WorkspaceBoundary != nil {
		if (claim.WorkspaceBoundary.HistoryOperation == "" && historyOperation != claim.OperationID) || (claim.WorkspaceBoundary.HistoryOperation != "" && historyOperation != claim.WorkspaceBoundary.HistoryOperation) {
			return 0, fail()
		}
		return dshVerifyBoundaryClaim(ctx, claim, name, args)
	}
	session, operation, call := claim.SessionID, claim.OperationID, claim.CallID
	var proofs []dshConnectorHistoryEvent
	var fetchErr error
	collect := func(ctx context.Context, params map[string]any) (json.RawMessage, error) {
		raw, err := fetch(ctx, params)
		if err != nil {
			fetchErr = err
			return nil, err
		}
		if len(raw) > 1<<20 {
			return nil, fail()
		}
		var page struct {
			Events []json.RawMessage `json:"events"`
		}
		if json.Unmarshal(raw, &page) != nil {
			return nil, fail()
		}
		for _, item := range page.Events {
			var e dshConnectorHistoryEvent
			if json.Unmarshal(item, &e) != nil {
				return nil, fail()
			}
			if e.Type == "aibertai/workspace-result" {
				proofs = append(proofs, e)
			}
		}
		return raw, nil
	}
	raw, err := dshMatchConnectorHistory(ctx, session, historyOperation, call, name, args, collect)
	var result struct {
		SessionID string `json:"sessionId"`
		Event     struct {
			Seq             *int64  `json:"seq"`
			SourceEventSeqs []int64 `json:"sourceEventSeqs"`
		} `json:"event"`
	}
	if err != nil && fetchErr != nil {
		return 0, errors.Join(fail(), fetchErr)
	}
	if err != nil || json.Unmarshal(raw, &result) != nil || result.SessionID != session || result.Event.Seq == nil || len(result.Event.SourceEventSeqs) != 1 {
		return 0, fail()
	}
	matches := 0
	for _, proof := range proofs {
		var d struct {
			OperationID   string `json:"operationId"`
			CallID        string `json:"callId"`
			ToolName      string `json:"toolName"`
			ToolCallSeq   *int64 `json:"toolCallSeq"`
			ToolResultSeq *int64 `json:"toolResultSeq"`
			Digest        string `json:"digest"`
		}
		var fields map[string]json.RawMessage
		if proof.Seq == nil || json.Unmarshal(proof.Data, &fields) != nil || len(fields) != 6 {
			return 0, fail()
		}
		for _, key := range []string{"operationId", "callId", "toolName", "toolCallSeq", "toolResultSeq", "digest"} {
			if _, ok := fields[key]; !ok {
				return 0, fail()
			}
		}
		if json.Unmarshal(proof.Data, &d) != nil {
			return 0, fail()
		}
		if d.OperationID != operation || d.CallID != call {
			if d.ToolCallSeq != nil && *d.ToolCallSeq == result.Event.SourceEventSeqs[0] || d.ToolResultSeq != nil && *d.ToolResultSeq == *result.Event.Seq {
				return 0, fail()
			}
			continue
		}
		matches++
		if d.ToolName != name || d.ToolCallSeq == nil || *d.ToolCallSeq != result.Event.SourceEventSeqs[0] || d.ToolResultSeq == nil || *d.ToolResultSeq != *result.Event.Seq || *proof.Seq <= *result.Event.Seq || d.Digest != *claim.Output {
			return 0, fail()
		}
	}
	if matches != 1 {
		return 0, fail()
	}
	return *result.Event.Seq, nil
}

// dshRecoverWorkspaceClaim can run after a fresh SDK process has resumed the
// same session and exposes its flushed history. It never re-executes a tool.
// A missing host result or durable proof remains blocked for human review.
func dshRecoverWorkspaceClaim(ctx context.Context, root *os.Root, session acpSession, fetch dshConnectorHistoryFetch) error {
	fail := func() error { return errors.New("dsh_workspace_reconciliation_required") }
	if root == nil || !validUUID(session.ID) || session.ID != session.DSHSessionID || fetch == nil {
		return fail()
	}
	pending, err := readConnectorClaim(root, "dsh-connector-dispatch/"+session.ID+"/.pending", session.ID)
	var claim dshConnectorClaim
	var pendingMarker bool
	if err == nil {
		pendingMarker = true
		claim, err = connectorPending(root, session.ID, pending.OperationID, pending.CallID)
	} else if errors.Is(err, os.ErrNotExist) && len(session.Turns) > 0 {
		last := session.Turns[len(session.Turns)-1]
		if len(last.ToolRuns) == 0 {
			return fail()
		}
		finalRun := last.ToolRuns[len(last.ToolRuns)-1]
		if finalRun.Call == nil || (finalRun.Call.Function.Name != "write" && finalRun.Call.Function.Name != "edit") || finalRun.DSHCallID == "" {
			return fail()
		}
		for i, candidate := range last.ToolRuns {
			if candidate.Pending && i != len(last.ToolRuns)-1 {
				return fail()
			}
		}
		candidate := finalRun.DSHCallID
		claim, err = readConnectorClaim(root, connectorClaimPath(session.ID, last.Operation, candidate), session.ID)
		if err == nil && (claim.State != "reconciled" || claim.SDKSeq == nil || dshConnectorAssertSettled(root, session.ID) != nil) {
			return fail()
		}
	} else {
		return fail()
	}
	if err != nil || claim.Kind != "workspace" || claim.State != "result_ready" && claim.State != "reconciled" {
		return fail()
	}
	var run *workspaceToolRun
	count := 0
	for _, turn := range session.Turns {
		if turn.Operation != claim.OperationID {
			continue
		}
		for i := range turn.ToolRuns {
			if turn.ToolRuns[i].DSHCallID == claim.CallID {
				run = &turn.ToolRuns[i]
				count++
			}
		}
	}
	if count != 1 || run == nil || run.Call == nil {
		return fail()
	}
	historyOperation := claim.OperationID
	if w := claim.WorkspaceBoundary; w != nil && w.HistoryOperation != "" {
		if session.Goal == nil || session.Goal.ID != w.GoalID || session.Goal.DSHGoalID != w.SDKGoalID || session.Goal.DSHQueryOp != w.HistoryOperation || session.GoalRun != claim.OperationID {
			return fail()
		}
		historyOperation = w.HistoryOperation
	}
	if pendingMarker {
		return dshWorkspaceReconcileBound(ctx, root, session.ID, claim.OperationID, historyOperation, claim.CallID, run.Call.Function.Name, json.RawMessage(run.Call.Function.Arguments), fetch)
	}
	seq, err := dshVerifyWorkspaceProofBound(ctx, claim, historyOperation, run.Call.Function.Name, json.RawMessage(run.Call.Function.Arguments), fetch)
	if err != nil || claim.SDKSeq == nil || seq != *claim.SDKSeq {
		return fail()
	}
	return nil
}

// reconcileDSHWorkspaceSession resumes only for history inspection; it does not
// enqueue a prompt or re-execute the write. The caller must authorize the
// session and keep further side effects excluded while it runs.
func reconcileDSHWorkspaceSession(ctx context.Context, cfg *dshRuntimeConfig, root *os.Root, session acpSession) error {
	options, _ := ctx.Value(dshSDKOptionsKey{}).(dshSDKOptions)
	previous := options.AfterReconcile
	options.AfterReconcile = func(ctx context.Context, p *dshProcess) error {
		if previous != nil {
			if err := previous(ctx, p); err != nil {
				return err
			}
		}
		return dshRecoverWorkspaceClaim(ctx, root, session, func(ctx context.Context, params map[string]any) (json.RawMessage, error) {
			return p.call(ctx, "aibertai/history", params, nil)
		})
	}
	return reconcileDSHSession(context.WithValue(ctx, dshSDKOptionsKey{}, options), cfg, session)
}
