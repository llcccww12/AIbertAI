package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
)

// The host only writes this marker before it has sent an approval grant.
func dshUnexecutedBash(turn acpTurn) bool {
	if !turn.Failed || !turn.Acked || turn.DSHReconciled || turn.DSHUserReviewed ||
		turn.FailureCode != "dsh_reconciliation_required" || len(turn.ToolRuns) == 0 {
		return false
	}
	for _, run := range turn.ToolRuns {
		if run.Pending {
			return false
		}
	}
	last := turn.ToolRuns[len(turn.ToolRuns)-1]
	return last.Call != nil && last.Call.Function.Name == "bash" && last.DSHCallID != "" &&
		last.Report.Failed == 1 && last.Output == dshApprovalNotExecuted
}

// Require the SDK's own approval decision, typed sandbox error and terminal
// turn boundary. A tool's textual output cannot prove that it never ran.
func dshAbortedApprovalTail(events []dshConnectorHistoryEvent, i int, callSeq, turn, step int64, call string) bool {
	end := len(events)
	if end > 0 && events[end-1].Type == "session/end-seed" {
		end--
	}
	if i < 3 || i+3 != end || events[i-3].Seq == nil || *events[i-3].Seq != callSeq ||
		events[i-2].Type != "approval/asked" || events[i-1].Type != "approval/decided" ||
		events[i+1].Type != "step/end" || events[i+2].Type != "turn/end" {
		return false
	}
	var asked struct{ ID, ToolName, CallID string }
	var decision struct{ ID, Outcome string }
	var result struct{ Error struct{ Code string } }
	var stepEnd struct{ Turn, Step int64 }
	var turnEnd struct {
		Turn   int64
		Reason struct {
			Kind   string
			Reason struct{ Kind string }
		}
	}
	return json.Unmarshal(events[i-2].Data, &asked) == nil && validUUID(asked.ID) && asked.ToolName == "bash" && asked.CallID == call &&
		json.Unmarshal(events[i-1].Data, &decision) == nil && decision.ID == asked.ID && decision.Outcome == "unavailable" &&
		json.Unmarshal(events[i].Data, &result) == nil && result.Error.Code == "SANDBOX_ESCALATION_UNAVAILABLE" &&
		json.Unmarshal(events[i+1].Data, &stepEnd) == nil && stepEnd.Turn == turn && stepEnd.Step == step &&
		json.Unmarshal(events[i+2].Data, &turnEnd) == nil && turnEnd.Turn == turn && turnEnd.Reason.Kind == "aborted" &&
		(turnEnd.Reason.Reason.Kind == "disposed" || turnEnd.Reason.Reason.Kind == "user")
}

// This recovery covers an acknowledged native shell tool followed immediately
// by an aborted model attempt. Every earlier call remains in the original turn.
func dshNativeRecoveryCandidate(turn acpTurn) bool {
	if !turn.Failed || !turn.Acked || turn.DSHReconciled || turn.DSHUserReviewed || turn.FailureCode != "dsh_connector_reconciliation_required" || len(turn.ToolRuns) == 0 {
		return false
	}
	last := turn.ToolRuns[len(turn.ToolRuns)-1]
	return !last.Pending && last.Call != nil && last.Call.Function.Name == "bash" && last.DSHCallID != ""
}

func (h *relaySessionHost) recoverDSHNativeTurn(ctx context.Context, session acpSession) error {
	fail := errors.New("dsh_connector_reconciliation_required")
	if h.local == nil || session.ID != session.DSHSessionID || !validUUID(session.ID) {
		return fail
	}
	turn := session.Turns[len(session.Turns)-1]
	run := turn.ToolRuns[len(turn.ToolRuns)-1]
	root := h.live.store.Root
	if dshUnexecutedBash(turn) {
		options := dshSDKOptions{AfterReconcile: func(ctx context.Context, p *dshProcess) error {
			if dshUnsettledWorkspaceWrite(root, session.ID, turn) {
				return fail
			}
			_, err := dshMatchToolHistory(ctx, session.ID, turn.Operation, run.DSHCallID, "bash", json.RawMessage(run.Call.Function.Arguments), p.historyCall, true)
			return err
		}}
		return reconcileDSHSession(context.WithValue(ctx, dshSDKOptionsKey{}, options), h.live.dsh, session)
	}
	verified := false
	options := dshSDKOptions{
		Observe: func(method string, raw json.RawMessage) error {
			if method == "aibertai/model-settled" {
				var result struct{ State string }
				if json.Unmarshal(raw, &result) != nil || result.State != "failed" {
					return fail
				}
			}
			return nil
		},
		AfterReconcile: func(context.Context, *dshProcess) error {
			if !verified {
				return fail
			}
			return nil
		},
	}
	options.CheckCancelled = func(ctx context.Context, p *dshProcess, local *dshLocalGateway, pending *dshLocalRecord, seq int64) error {
		marker, err := dshReadCancelledAccounting(local.journal, turn.Operation)
		if err != nil || pending == nil && marker == nil {
			return fail
		}
		var record dshLocalRecord
		if pending != nil {
			record = *pending
		} else {
			record, seq = marker.Record, marker.ReceiptSeq
			raw, err := p.call(ctx, "aibertai/reconcile", map[string]string{"sessionId": session.ID, "invocationId": record.CallID}, nil)
			var settlement struct {
				State string
				Seq   *int64
			}
			if err != nil || json.Unmarshal(raw, &settlement) != nil || settlement.State != "failed" || settlement.Seq == nil || *settlement.Seq != seq {
				return fail
			}
		}
		if record.SessionID != session.ID || record.OperationID != turn.Operation || !validUUID(record.CallID) || record.State != "dispatched" || marker != nil && (!reflect.DeepEqual(record, marker.Record) || seq != marker.ReceiptSeq) {
			return fail
		}
		claim, err := readConnectorClaim(root, connectorClaimPath(session.ID, turn.Operation, run.DSHCallID), session.ID)
		if err != nil || claim.OperationID != turn.Operation || claim.CallID != run.DSHCallID || claim.Kind != "native" || claim.Output == nil || run.Output != *claim.Output || claim.State != "result_ready" && claim.State != "reconciled" {
			return fail
		}
		raw, err := dshMatchConnectorHistory(ctx, session.ID, turn.Operation, run.DSHCallID, run.Call.Function.Name, json.RawMessage(run.Call.Function.Arguments), p.historyCall)
		if err != nil {
			return err
		}
		resultSeq, err := dshVerifiedConnectorResult(claim, raw)
		if err != nil || claim.SDKSeq != nil && *claim.SDKSeq != resultSeq {
			return fail
		}
		var result struct {
			Event struct{ Data struct{ Turn, Step int64 } }
		}
		if json.Unmarshal(raw, &result) != nil {
			return fail
		}
		if err = dshNativeFailedTail(ctx, record, resultSeq, seq, result.Event.Data.Turn, result.Event.Data.Step, p.historyCall); err != nil {
			return err
		}
		receipt, err := local.verifyCancelledAccounting(ctx, record, true)
		if err != nil || marker != nil && marker.receipt() != receipt {
			return fail
		}
		// Either half may have been committed before a crash. On retry, match the
		// exact history, output, archived record and accounting receipt again.
		if _, err = root.Lstat("dsh-connector-dispatch/" + session.ID + "/.pending"); err == nil {
			if err = dshReleaseSideEffect(root, claim, resultSeq); err != nil {
				return err
			}
		} else if !errors.Is(err, os.ErrNotExist) || claim.SDKSeq == nil {
			return fail
		}
		if err = dshConnectorAssertSettled(root, session.ID); err != nil {
			return err
		}
		if pending != nil {
			if err = local.archiveCancelled(record, receipt, seq); err != nil {
				return err
			}
		}
		verified = true
		return nil
	}
	return reconcileDSHSession(context.WithValue(ctx, dshSDKOptionsKey{}, options), h.live.dsh, session)
}

// The session's final six events must close the completed tool's step, start
// one conversation invocation, record its failed attempt, and abort the turn.
func dshNativeFailedTail(ctx context.Context, record dshLocalRecord, resultSeq, receiptSeq, turn, step int64, fetch dshConnectorHistoryFetch) error {
	fail := errors.New("dsh_native_failed_tail_unverified")
	if turn < 1 || step < 1 || receiptSeq != resultSeq+4 {
		return fail
	}
	kinds := [...]string{"step/end", "step/start", "aibertai/invocation", "assistant/attempt", "step/end", "turn/end"}
	end, err := dshRepairHistory(ctx, record.SessionID, resultSeq, fetch, func(seq int64, raw json.RawMessage) error {
		var e struct {
			Type string
			Data struct {
				Turn, Step            int64
				InvocationID, Purpose string
				Reason                struct {
					Kind   string
					Reason struct{ Kind string }
				}
			}
		}
		i := seq - resultSeq - 1
		if json.Unmarshal(raw, &e) != nil {
			return fail
		}
		// A fresh SDK process restoring persisted history may append this lifecycle
		// marker. It contains no invocation, prompt, or tool action.
		if i == int64(len(kinds)) && e.Type == "session/end-seed" {
			var marker struct{ Data map[string]json.RawMessage }
			if json.Unmarshal(raw, &marker) != nil || marker.Data == nil || len(marker.Data) != 0 {
				return fail
			}
			return nil
		}
		if i < 0 || i >= int64(len(kinds)) || e.Type != kinds[i] {
			return fail
		}
		d := e.Data
		switch i {
		case 0:
			if d.Turn != turn || d.Step != step {
				return fail
			}
		case 1, 4:
			if d.Turn != turn || d.Step != step+1 {
				return fail
			}
		case 2:
			if d.Purpose != "conversation" || d.InvocationID != record.CallID {
				return fail
			}
		case 3:
			if d.Turn != turn || d.Step != step+1 || d.InvocationID != record.CallID {
				return fail
			}
		case 5:
			if d.Turn != turn || d.Reason.Kind != "aborted" || (d.Reason.Reason.Kind != "disposed" && d.Reason.Reason.Kind != "user") {
				return fail
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if end != resultSeq+int64(len(kinds)) && end != resultSeq+int64(len(kinds))+1 {
		return fail
	}
	return nil
}
