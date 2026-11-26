package main

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
)

var errDSHStoppedAccounting = errors.New("dsh_stopped_accounting_pending")

// Identify the exact user stop and prove that every tool projected for its
// root turn has a matching terminal SDK result. No tool is replayed.
func dshStoppedToolTurn(events []dshConnectorHistoryEvent, turn acpTurn) (target, end int64, err error) {
	return dshTerminalToolTurn(events, turn, "aborted")
}

func dshTerminalToolTurn(events []dshConnectorHistoryEvent, turn acpTurn, kind string) (target, end int64, err error) {
	target, _, err = dshPromptHistory(events, turn.Operation)
	if err != nil {
		return 0, 0, errDSHCancelProof
	}
	end = -1
	projected := acpTurn{Operation: turn.Operation}
	for i, e := range events {
		if e.Seq == nil || *e.Seq != int64(i) {
			return 0, 0, errDSHCancelProof
		}
		var d struct {
			Turn   int64
			Reason struct {
				Kind   string
				Reason struct{ Kind string }
			}
		}
		switch e.Type {
		case "turn/start", "turn/end", "tool/call", "tool/result":
			if json.Unmarshal(e.Data, &d) != nil || end >= 0 {
				return 0, 0, errDSHCancelProof
			}
		default:
			continue
		}
		if d.Turn != target {
			continue
		}
		switch e.Type {
		case "turn/end":
			if d.Reason.Kind != kind || kind == "aborted" && d.Reason.Reason.Kind != "user" {
				return 0, 0, errDSHCancelProof
			}
			end = *e.Seq
		case "tool/call", "tool/result":
			if e.SurfaceOp.Op == "replace" {
				if e.Type != "tool/result" || !dshValidPrunedResult(events, i) {
					return 0, 0, errDSHCancelProof
				}
				continue
			}
			raw, err := json.Marshal(map[string]any{"event": map[string]any{"type": e.Type, "data": e.Data}})
			if err != nil {
				return 0, 0, errDSHCancelProof
			}
			update, err := projectDSHTool(&projected, raw)
			if update == nil || err != nil {
				return 0, 0, errDSHCancelProof
			}
		}
	}
	if end < 0 || len(projected.ToolRuns) != len(turn.ToolRuns) {
		return 0, 0, errDSHCancelProof
	}
	for i, r := range projected.ToolRuns {
		old := turn.ToolRuns[i]
		if r.Pending || old.Pending || r.Call == nil || old.Call == nil || r.DSHCallID != old.DSHCallID ||
			r.Call.Function.Name != old.Call.Function.Name || r.Call.Function.Arguments != old.Call.Function.Arguments || r.Report.Failed != old.Report.Failed {
			return 0, 0, errDSHCancelProof
		}
	}
	return target, end, nil
}

type dshStoppedMember struct {
	BeforeFirstToken bool   `json:"beforeFirstToken,omitempty"`
	MemberID         string `json:"memberId"`
	Seq              int64  `json:"seq"`
	Turn             int64  `json:"turn"`
	Step             int64  `json:"step"`
	RootEndSeq       int64  `json:"rootEndSeq"`
	ChildEndSeq      int64  `json:"childEndSeq"`
}

func (m dshStoppedMember) valid(root string) bool {
	return validUUID(m.MemberID) && m.MemberID != root && m.Seq > 0 && m.Turn > 0 && m.Step > 0 && m.RootEndSeq > 0 && m.ChildEndSeq > m.Seq
}

// The root SDK independently verifies the durable delegation chain and the
// child's exact interrupted invocation. A root reconciliation miss alone is
// never a settlement, and a member sequence is never treated as a root one.
func dshStoppedModel(ctx context.Context, call func(context.Context, string, any) (json.RawMessage, error), record dshLocalRecord, events []dshConnectorHistoryEvent, target, rootEnd int64) (int64, *dshStoppedMember, error) {
	params := map[string]string{"sessionId": record.SessionID, "invocationId": record.CallID}
	raw, err := call(ctx, "aibertai/reconcile", params)
	var receipt struct {
		State string
		Seq   int64
	}
	if err != nil || json.Unmarshal(raw, &receipt) != nil {
		return 0, nil, errDSHCancelProof
	}
	if receipt.State == "failed" {
		// The root receipt alone cannot establish the terminal state of delegates.
		for _, e := range events {
			if e.Type == "subagent/catalog" {
				return 0, nil, errDSHCancelProof
			}
		}
		if receipt.Seq <= 0 || receipt.Seq >= rootEnd {
			return 0, nil, errDSHCancelProof
		}
		event := events[receipt.Seq]
		var interrupted struct {
			Turn         int64
			InvocationID string
			Interrupted  bool
		}
		if event.Type != "assistant/message" || json.Unmarshal(event.Data, &interrupted) != nil || interrupted.Turn != target || interrupted.InvocationID != record.CallID || !interrupted.Interrupted {
			return 0, nil, errDSHCancelProof
		}
		for _, later := range events[receipt.Seq+1 : rootEnd] {
			switch later.Type {
			case "aibertai/invocation", "assistant/attempt", "assistant/message", "tool/result", "tool/call", "step/start":
				return 0, nil, errDSHCancelProof
			}
		}
		return receipt.Seq, nil, nil
	}
	if receipt.State != "unknown" {
		return 0, nil, errDSHCancelProof
	}
	params["operationId"] = record.OperationID
	raw, err = call(ctx, "aibertai/team-cancel-proof", params)
	var proof struct {
		State string
		dshStoppedMember
	}
	if err != nil || json.Unmarshal(raw, &proof) != nil || proof.State != "cancelled" || !proof.valid(record.SessionID) || proof.RootEndSeq != rootEnd {
		return 0, nil, errDSHCancelProof
	}
	return proof.Seq, &proof.dshStoppedMember, nil
}

func (h *relaySessionHost) verifyStoppedTools(ctx context.Context, session acpSession, turn acpTurn) error {
	if h.local == nil || h.live == nil || h.live.store == nil || h.live.store.Root == nil || session.platformGoal() != nil || session.ID != session.DSHSessionID || !validUUID(session.ID) {
		return errDSHCancelProof
	}
	check := func(ctx context.Context, p *dshProcess, local *dshLocalGateway) error {
		root := h.live.store.Root
		if dshConnectorAssertSettled(root, session.ID) != nil {
			return errDSHCancelProof
		}
		marker, err := dshReadCancelledAccounting(local.journal, turn.Operation)
		if err != nil {
			return err
		}
		local.mu.Lock()
		pending := local.pending != nil
		var record dshLocalRecord
		if pending {
			record = *local.pending
		}
		local.mu.Unlock()
		if !pending && marker == nil {
			return errDSHCancelProof
		}
		if !pending {
			record = marker.Record
		}
		// A cancellation before the first gateway response has no status or
		// transaction in the journal. It needs an explicit terminal backend receipt.
		beforeHeaders := record.State == "dispatched" && record.Status == 0 && record.Transaction == ""
		if record.SessionID != session.ID || record.OperationID != turn.Operation || !validUUID(record.CallID) || !validUUID(record.ModelID) || record.State != "dispatched" && record.State != "delivered" || record.GatewayCode != "" || !beforeHeaders && (record.Status < 200 || record.Status >= 300) || len(record.Body) == 0 || len(record.Body) > dshGatewayMaxRequestBytes || !json.Valid(record.Body) {
			return errDSHCancelProof
		}
		var events []dshConnectorHistoryEvent
		_, err = dshRepairHistory(ctx, session.ID, -1, p.historyCall, func(_ int64, raw json.RawMessage) error {
			var e dshConnectorHistoryEvent
			if err := json.Unmarshal(raw, &e); err != nil {
				return err
			}
			events = append(events, e)
			return nil
		})
		if err != nil {
			return err
		}
		target, end, err := dshStoppedToolTurn(events, turn)
		if err != nil {
			return err
		}
		seq, child, err := dshStoppedModel(ctx, func(ctx context.Context, m string, params any) (json.RawMessage, error) {
			return p.call(ctx, m, params, nil)
		}, record, events, target, end)
		if err != nil {
			return err
		}
		receipt, err := local.verifyCancelledAccounting(ctx, record, beforeHeaders || child != nil && child.BeforeFirstToken)
		if err != nil {
			return errDSHStoppedAccounting
		}
		if marker != nil && (!reflect.DeepEqual(marker.Record, record) || marker.receipt() != receipt || marker.ReceiptSeq != seq || !reflect.DeepEqual(marker.Member, child)) {
			return errDSHCancelProof
		}
		if err := dshConnectorAssertSettled(root, session.ID); err != nil {
			return err
		}
		if !pending {
			return nil
		}
		return local.archiveStopped(record, receipt, seq, child)
	}
	return reconcileDSHSession(context.WithValue(ctx, dshSDKOptionsKey{}, dshSDKOptions{CheckStopped: check}), h.live.dsh, session)
}

func dshStoppedToolsCandidate(turn acpTurn) bool {
	if !turn.Failed || !turn.Acked || turn.DSHReconciled || turn.DSHUserReviewed || turn.FailureCode != "dsh_reconciliation_required" || len(turn.ToolRuns) == 0 || !validUUID(turn.Operation) {
		return false
	}
	for _, run := range turn.ToolRuns {
		if run.Pending || run.Call == nil || run.DSHCallID == "" {
			return false
		}
	}
	return true
}
