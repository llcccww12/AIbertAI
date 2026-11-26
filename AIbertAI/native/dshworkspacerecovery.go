package main

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
)

// Explicit, authorized recovery uses only durable evidence, not the model or a
// user assertion. The interrupted operation remains failed and non-replayable.
func (h *relaySessionHost) recoverDSHWorkspaceSession(ctx context.Context, id string) error {
	return h.recoverDSHPromptSession(ctx, id, true)
}

// A normal prompt can be recovered from a verified user stop or a provable
// prompt history. Neither path may introduce a new model or tool dispatch.
func (h *relaySessionHost) recoverDSHPromptSession(ctx context.Context, id string, workspace bool) error {
	fail := func() error {
		if workspace {
			return errors.New("dsh_workspace_reconciliation_required")
		}
		return errors.New("dsh_reconciliation_required")
	}
	if h.live == nil || h.live.store == nil || h.live.store.Root == nil || h.live.dsh == nil || !validUUID(id) {
		return fail()
	}
	live := h.live
	live.mu.Lock()
	row, ok := live.sessions[id]
	if !ok || live.restarting || live.compacting[id] || live.activeTurns[id] != nil || row.ID != id || row.Engine != "dsh" || row.State == "running" || row.GoalRun != "" || !workspace && row.Goal != nil || len(row.Turns) == 0 {
		live.mu.Unlock()
		return fail()
	}
	if workspace {
		if handled, err := h.finishRecordedDSHManualReviewLocked(id); handled {
			live.mu.Unlock()
			return err
		}
	}
	last := row.Turns[len(row.Turns)-1]
	native := workspace && row.platformGoal() == nil && (dshNativeRecoveryCandidate(last) || dshUnexecutedBash(last))
	modelFailure := workspace && row.platformGoal() == nil && dshPendingModelFailure(last)
	stoppedTools := !native && !modelFailure && row.platformGoal() == nil && dshStoppedToolsCandidate(last)
	if !last.Failed || last.DSHReconciled || !last.Acked ||
		workspace && !native && !stoppedTools && !modelFailure && !dshRecoverableWorkspaceFailure(last.FailureCode) ||
		!workspace && !stoppedTools && (!dshRecoverablePromptFailure(last.FailureCode) || len(last.ToolRuns) != 0) {
		live.mu.Unlock()
		return fail()
	}
	if live.compacting == nil {
		live.compacting = make(map[string]bool)
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
	live.compacting[id] = true
	live.mu.Unlock()
	defer func() { live.mu.Lock(); delete(live.compacting, id); live.mu.Unlock() }()
	var rebuilt acpTurn
	var projectionSeq int64
	var projectionAvailable bool
	var cancelledRecovery bool
	options := dshSDKOptions{AfterReconcile: func(ctx context.Context, p *dshProcess) error {
		var events []dshConnectorHistoryEvent
		var rawEvents []json.RawMessage
		fetch := func(ctx context.Context, params map[string]any) (json.RawMessage, error) {
			return p.call(ctx, "aibertai/history", params, nil)
		}
		through, err := dshRepairHistory(ctx, detached.DSHSessionID, -1, fetch, func(_ int64, raw json.RawMessage) error {
			var event dshConnectorHistoryEvent
			if json.Unmarshal(raw, &event) != nil {
				return errors.New("dsh_invalid_history")
			}
			events = append(events, event)
			rawEvents = append(rawEvents, raw)
			return nil
		})
		if err != nil {
			return err
		}
		verify := func(run workspaceToolRun) error {
			if !workspace || run.Call == nil {
				return fail()
			}
			claim, err := readConnectorClaim(live.store.Root, connectorClaimPath(id, last.Operation, run.DSHCallID), id)
			if err != nil || claim.OperationID != last.Operation || claim.CallID != run.DSHCallID {
				return fail()
			}
			if dshRecoverableCanvasTool(run.Call.Function.Name) {
				return dshVerifyCanvasResult(ctx, claim, run, fetch)
			}
			seq, err := dshVerifyWorkspaceProof(ctx, claim, run.Call.Function.Name, json.RawMessage(run.Call.Function.Arguments), fetch)
			if err != nil || claim.State == "reconciled" && (claim.SDKSeq == nil || *claim.SDKSeq != seq) {
				return fail()
			}
			return nil
		}
		var projected int64
		rebuilt, projected, err = dshRebuildPromptTurn(detached.DSHSessionID, last, events, rawEvents, verify)
		if err != nil || projected != through || !workspace && len(rebuilt.ToolRuns) != 0 {
			if !workspace {
				return fail()
			}
			// Keep the existing proof-only recovery for histories this projection
			// does not support; do not publish a partial projection or watermark.
			return nil
		}
		projectionAvailable = true
		projectionSeq = projected
		return nil
	}}
	if !workspace {
		options.CheckCancelled = func(ctx context.Context, p *dshProcess, local *dshLocalGateway, pending *dshLocalRecord, seq int64) error {
			marker, err := dshReadCancelledAccounting(local.journal, last.Operation)
			if err != nil {
				return fail()
			}
			// A local account can prove its own billing receipt. Cloud grants and
			// Goal billing require separate authorization and remain unchanged.
			if h.local == nil {
				if pending != nil || marker != nil {
					return fail()
				}
				return nil
			}
			if pending == nil && marker == nil {
				return nil
			}
			var record dshLocalRecord
			if pending != nil {
				record = *pending
			} else {
				record, seq = marker.Record, marker.ReceiptSeq
				raw, err := p.call(ctx, "aibertai/reconcile", map[string]string{"sessionId": detached.DSHSessionID, "invocationId": record.CallID}, nil)
				var receipt struct {
					State string `json:"state"`
					Seq   int64  `json:"seq"`
				}
				if err != nil || json.Unmarshal(raw, &receipt) != nil || receipt.State != "failed" || receipt.Seq != seq {
					return fail()
				}
			}
			if marker != nil && (!reflect.DeepEqual(marker.Record, record) || marker.ReceiptSeq != seq) {
				return fail()
			}
			receipt, err := h.verifiedDSHCancelReceipt(ctx, p, local, detached, last, record, seq)
			if err != nil || marker != nil && marker.receipt() != receipt {
				return fail()
			}
			cancelledRecovery = true
			if pending != nil {
				return local.archiveCancelled(record, receipt, seq)
			}
			return nil
		}
	}
	ctx = context.WithValue(ctx, dshSDKOptionsKey{}, options)
	var modelFailureCode string
	if modelFailure {
		modelFailureCode, err = h.recoverModelFailure(ctx, detached, last)
	} else if stoppedTools {
		err = h.verifyStoppedTools(ctx, detached, last)
		cancelledRecovery = err == nil
	} else if native {
		err = h.recoverDSHNativeTurn(ctx, detached)
	} else if workspace {
		err = reconcileDSHWorkspaceSession(ctx, live.dsh, live.store.Root, detached)
	} else {
		err = reconcileDSHSession(ctx, live.dsh, detached)
	}
	accountingPending := errors.Is(err, errDSHStoppedAccounting)
	if err != nil && (!accountingPending || last.DSHStopObserved) {
		return err
	}
	if err = dshConnectorAssertSettled(live.store.Root, id); err != nil {
		return fail()
	}
	live.mu.Lock()
	defer live.mu.Unlock()
	current, ok := live.sessions[id]
	currentBytes, e := json.Marshal(current)
	if !ok || e != nil || string(currentBytes) != string(snapshot) {
		return fail()
	}
	current.Turns = append([]acpTurn(nil), current.Turns...)
	if accountingPending {
		current.Turns[len(current.Turns)-1].DSHStopObserved = true
		if err := live.save(current); err != nil {
			return fail()
		}
		return errDSHStoppedAccounting
	}
	if modelFailure {
		item := &current.Turns[len(current.Turns)-1]
		item.FailureCode, item.DSHModelFailureSettled = modelFailureCode, true
		return live.save(current)
	}
	if projectionAvailable {
		rebuilt.DSHProjectionSeq = &projectionSeq
		current.Turns[len(current.Turns)-1] = rebuilt
	}
	current.Turns[len(current.Turns)-1].DSHReconciled = true
	if native {
		current.State = "idle"
		if last.Interrupted || dshUnexecutedBash(last) {
			current.Turns[len(current.Turns)-1].Interrupted = true
			current.Turns[len(current.Turns)-1].DSHCancelConfirmed = true
		}
	}
	if cancelledRecovery {
		current.Turns[len(current.Turns)-1].Interrupted = true
		current.Turns[len(current.Turns)-1].DSHCancelConfirmed = true
		current.State = "idle"
	}
	if err = live.save(current); err != nil {
		return fail()
	}
	return nil
}

func dshRecoverablePromptFailure(code string) bool {
	return code == "dsh_runtime_exited" || code == "dsh_reconciliation_required" || code == "backend_response_interrupted" || code == "session_turn_failed"
}

func dshPromptRecoveryAvailable(row acpSession) bool {
	if row.Engine != "dsh" || row.State == "running" || row.GoalRun != "" || row.Goal != nil || len(row.Turns) == 0 {
		return false
	}
	last := row.Turns[len(row.Turns)-1]
	return last.Failed && !last.DSHReconciled && last.Acked && (len(last.ToolRuns) == 0 || dshStoppedToolsCandidate(last)) && dshRecoverablePromptFailure(last.FailureCode)
}

func dshRecoverableWorkspaceFailure(code string) bool {
	return code == "" || code == "dsh_write_outcome_unknown" || code == "dsh_runtime_exited"
}

func dshWorkspaceRecoveryAvailable(row acpSession) bool {
	if row.Engine != "dsh" || row.State == "running" || row.GoalRun != "" || len(row.Turns) == 0 {
		return false
	}
	last := row.Turns[len(row.Turns)-1]
	if row.platformGoal() == nil && (dshPendingModelFailure(last) || dshStoppedToolsCandidate(last) || dshNativeRecoveryCandidate(last) || dshUnexecutedBash(last)) {
		return true
	}
	if !last.Failed || (last.DSHReconciled || last.DSHUserReviewed) || !last.Acked || !dshRecoverableWorkspaceFailure(last.FailureCode) || len(last.ToolRuns) == 0 {
		return false
	}
	run := last.ToolRuns[len(last.ToolRuns)-1]
	return run.Call != nil && (run.Call.Function.Name == "write" || run.Call.Function.Name == "edit")
}
