package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// dshStuckModelOnlyTurn reports a failed turn that ran no tools and is blocked
// only by an unproven model call. Turns with tool runs, workspace writes,
// connectors, canvas or team members keep their stricter recovery.
func dshStuckModelOnlyTurn(t acpTurn) bool {
	if !t.Failed || t.Interrupted || t.DSHReconciled || t.DSHModelFailureSettled || t.DSHMemberID != "" || len(t.ToolRuns) != 0 || !validUUID(t.Operation) {
		return false
	}
	switch t.FailureCode {
	case "dsh_model_outcome_unknown", "dsh_model_response_interrupted", "dsh_reconciliation_required":
		return true
	}
	return false
}

// settleStuckDSHModelTurn applies the DSH failure semantics to a turn recorded
// before them: if the SDK durably recorded the model call as failed, the call
// is archived as unverified and the session accepts the next prompt.
func (h *relaySessionHost) settleStuckDSHModelTurn(ctx context.Context, id string) error {
	fail := errors.New("dsh_reconciliation_required")
	live := h.live
	if live == nil || live.store == nil || live.store.Root == nil || live.dsh == nil || h.local == nil || !validUUID(id) {
		return fail
	}
	live.mu.Lock()
	row, ok := live.sessions[id]
	if !ok || live.restarting || live.compacting[id] || live.activeTurns[id] != nil || row.Engine != "dsh" || row.State == "running" || desktopGoalAdvancing(row.Goal) || len(row.Turns) == 0 || !dshStuckModelOnlyTurn(row.Turns[len(row.Turns)-1]) {
		live.mu.Unlock()
		return fail
	}
	last := row.Turns[len(row.Turns)-1]
	snapshot, err := json.Marshal(row)
	var detached acpSession
	if err != nil || json.Unmarshal(snapshot, &detached) != nil {
		live.mu.Unlock()
		return fail
	}
	if live.compacting == nil {
		live.compacting = make(map[string]bool)
	}
	live.compacting[id] = true
	live.mu.Unlock()
	defer func() { live.mu.Lock(); delete(live.compacting, id); live.mu.Unlock() }()

	if err := dshConnectorAssertSettled(live.store.Root, id); err != nil {
		return fail
	}
	var code string
	check := func(ctx context.Context, p *dshProcess, local *dshLocalGateway) error {
		local.mu.Lock()
		pending := local.pending
		var record dshLocalRecord
		if pending != nil {
			record = *pending
		}
		local.mu.Unlock()
		if pending == nil {
			code = local.abandonedFailure(last.Operation)
			if code == "" {
				return fail
			}
			return nil
		}
		if record.OperationID != last.Operation {
			return fail
		}
		raw, err := p.call(ctx, "aibertai/reconcile", map[string]string{"sessionId": detached.DSHSessionID, "invocationId": record.CallID}, nil)
		var receipt struct {
			State string `json:"state"`
			Seq   int64  `json:"seq"`
		}
		if err != nil || json.Unmarshal(raw, &receipt) != nil || receipt.State != "failed" || receipt.Seq <= 0 {
			// The SDK says the reply reached it. Repair the journal from the
			// ledger when it can be proven; when it cannot, this still must not
			// block the session forever.
			if receipt.State == "committed" {
				local.recoverCommittedDelivery(ctx, detached.DSHSessionID, record.CallID, receipt.Seq)
			}
			// Neither the SDK nor the ledger could prove the outcome. Once the
			// release window has passed, stop blocking the employee: archive the
			// call as unsettled, charged nothing, with its receipt kept on disk.
			if unsettled, abandonErr := local.abandonUnsettled(record.CallID); abandonErr == nil {
				code = unsettled
				return nil
			}
			return fail
		}
		if err := local.acceptModelSettlement(ctx, detached.ID, record.CallID, receipt.State, receipt.Seq); err != nil {
			return err
		}
		if code = local.archivedFailure(last.Operation, record.CallID, "error"); code == "" {
			return fail
		}
		return nil
	}
	if err := reconcileDSHSession(context.WithValue(ctx, dshSDKOptionsKey{}, dshSDKOptions{CheckStopped: check}), live.dsh, detached); err != nil {
		return err
	}
	live.mu.Lock()
	defer live.mu.Unlock()
	current, ok := live.sessions[id]
	currentBytes, err := json.Marshal(current)
	if !ok || err != nil || string(currentBytes) != string(snapshot) {
		return fail
	}
	current.Turns = append([]acpTurn(nil), current.Turns...)
	item := &current.Turns[len(current.Turns)-1]
	item.FailureCode, item.DSHModelFailureSettled = code, true
	return live.save(current)
}

// dshModelCallPending reports whether the session's model-call journal still
// holds an unsettled invocation. Such a call binds the session to its model
// route and must be reconciled (its receipt rechecked) before another prompt
// can be dispatched; any other earlier failure never blocks the next message.
func (l *acpLive) dshModelCallPending(id string) bool {
	if l.dsh == nil || l.dsh.JournalRoot == "" || !validUUID(id) {
		return false
	}
	_, err := os.Lstat(filepath.Join(l.dsh.JournalRoot, id, "pending.json"))
	return err == nil || !errors.Is(err, os.ErrNotExist)
}
