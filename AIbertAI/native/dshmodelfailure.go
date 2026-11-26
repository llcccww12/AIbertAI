package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const dshNotDispatchedHeader = "X-Aibertai-Model-Not-Dispatched"

// This error may only be created before the transport sends POST /v1/*.
type dshPreflightError struct{ cause error }

func (e *dshPreflightError) Error() string { return e.cause.Error() }
func (e *dshPreflightError) Unwrap() error { return e.cause }

func dshFailureCode(status int) string {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return "dsh_model_access_denied"
	case http.StatusPaymentRequired:
		return "dsh_model_quota"
	case http.StatusTooManyRequests:
		return "dsh_model_rate_limited"
	case http.StatusBadRequest, http.StatusNotFound, http.StatusUnprocessableEntity, http.StatusRequestEntityTooLarge:
		return "dsh_model_request_rejected"
	default:
		return "dsh_model_service_unavailable"
	}
}

func dshPreflightStatus(err error) int {
	var status *backendHTTPError
	if errors.As(err, &status) {
		return status.Status
	}
	if errors.Is(err, context.Canceled) {
		return http.StatusConflict
	}
	return http.StatusBadGateway
}

func dshTerminalModelFailure(code string) bool {
	switch code {
	case "dsh_model_stream_ended", "dsh_model_output_limit", "dsh_model_content_refused", "dsh_model_access_denied", "dsh_model_quota", "dsh_model_rate_limited", "dsh_model_request_rejected", "dsh_model_service_unavailable", "dsh_model_unsettled":
		return true
	}
	return false
}

// Keep the original invocation id and request body. A refused request is
// retried only if our platform explicitly attests that it was never forwarded,
// or if the connection to the platform could not be opened at all.
func (g *dshLocalGateway) invokeWithPreflightRetry(ctx context.Context, call dshGatewayCall) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		resp, err := g.gateway.invoke(ctx, call)
		if err != nil && dshDialFailed(err) {
			// Nothing reached the platform, so the call is a proven non-dispatch.
			if attempt == 2 || ctx.Err() != nil {
				return nil, &dshPreflightError{err}
			}
			select {
			case <-ctx.Done():
				return nil, &dshPreflightError{err}
			case <-time.After(time.Duration(attempt+1) * 500 * time.Millisecond):
			}
			continue
		}
		if err != nil || attempt == 2 || resp.Header.Get(dshNotDispatchedHeader) != call.CallID || resp.Header.Get("X-Billing-Transaction-ID") != "" || resp.StatusCode != 429 && resp.StatusCode != 502 && resp.StatusCode != 503 && resp.StatusCode != 504 {
			return resp, err
		}
		delay := time.Duration(attempt+1) * 500 * time.Millisecond
		if after := strings.TrimSpace(resp.Header.Get("Retry-After")); after != "" {
			if seconds, err := strconv.Atoi(after); err == nil && seconds >= 0 && seconds <= 3 {
				delay = max(delay, time.Duration(seconds)*time.Second)
			} else if at, err := http.ParseTime(after); err == nil && time.Until(at) <= 3*time.Second {
				delay = max(delay, time.Until(at))
			} else {
				return resp, nil
			}
		}
		select {
		case <-ctx.Done():
			return resp, nil
		case <-time.After(delay):
		}
		if err := resp.Body.Close(); err != nil {
			return nil, &dshPreflightError{err}
		}
	}
}

func writeDSHModelError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": code, "code": code}})
}

// The journal continues to block all dispatch until the SDK commits the
// failure. Its ordinary call-id archive also rejects any subsequent replay.
func (g *dshLocalGateway) noteModelRejection(record *dshLocalRecord, status int, code string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.pending != record || !dshTerminalModelFailure(code) {
		return errors.New("dsh_reconciliation_required")
	}
	next := *record
	next.Status = status
	next.State = "rejected"
	next.FailureCode = code
	if err := g.writePending(&next); err != nil {
		return err
	}
	*record = next
	return nil
}

func (g *dshLocalGateway) archivedFailure(operation, call, finish string) string {
	if !validUUID(call) || !validUUID(operation) {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(g.journal, call+".json"))
	var r dshLocalRecord
	if err != nil || json.Unmarshal(data, &r) != nil || r.SessionID != g.session || r.OperationID != operation || r.CallID != call {
		return ""
	}
	if r.State == "rejected" && dshTerminalModelFailure(r.FailureCode) {
		return r.FailureCode
	}
	if r.State == "delivered" && r.Finish == "length" && finish == "max-tokens" {
		return "dsh_model_output_limit"
	}
	return ""
}

// recoverCommittedDelivery repairs a call left "dispatched" by builds that
// recorded delivery only when the upstream closed: if the SDK committed the
// reply and the platform ledger settled it as succeeded, the SDK did receive
// a complete response. Without that receipt the record stays pending.
func (g *dshLocalGateway) recoverCommittedDelivery(ctx context.Context, session, call string, seq int64) {
	if !g.requireInvocationID || session != g.session || !validUUID(call) || seq <= 0 {
		return
	}
	g.mu.Lock()
	if !g.waitServedLocked(call) || g.pending == nil || g.pending.CallID != call || g.pending.State != "dispatched" || g.pending.Status < 200 || g.pending.Status >= 300 || g.pending.GatewayCode != "" {
		g.mu.Unlock()
		return
	}
	record := *g.pending
	g.mu.Unlock()
	check, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	receipt, err := g.verifyModelAccounting(check, record, func(result, _, status string, _ dshCancelledReceipt) bool {
		return result == "succeeded" && dshFinalAccountStatus(status)
	})
	if err != nil {
		return
	}
	g.mu.Lock()
	current := g.pending
	g.mu.Unlock()
	if current == nil || !reflect.DeepEqual(*current, record) {
		return
	}
	// Keep the ledger's transaction on the repaired record: it is the only
	// proof the charge belongs to this call once the record leaves pending.
	transaction := record.Transaction
	if transaction == "" {
		transaction = receipt.TransactionID
	}
	g.markDelivered(current, record.Finish, record.Status, transaction)
}

// An HTTP status describes the transport, not whether the model was charged.
// Require both the exact persisted SDK failure and the account's terminal record.
func (g *dshLocalGateway) verifyModelSettlement(ctx context.Context, session, call, state string, seq int64) error {
	if state == "committed" {
		g.recoverCommittedDelivery(ctx, session, call, seq)
	}
	if state != "failed" || !g.requireInvocationID || session != g.session || !validUUID(call) || seq <= 0 {
		return g.acceptSDKSettlement(session, call, state, seq)
	}
	g.mu.Lock()
	pending := g.pending
	var record dshLocalRecord
	if pending != nil {
		record = *pending
	}
	g.mu.Unlock()
	if pending == nil || record.CallID != call || record.State == "rejected" || record.GatewayCode != "" {
		return g.acceptSDKSettlement(session, call, state, seq)
	}
	filtered := record.State == "delivered" && record.Finish == "content_filter"
	brokenStream := record.State == "dispatched" && record.Status >= 200 && record.Status < 300
	if !filtered && !brokenStream && record.Status < 400 {
		return g.acceptSDKSettlement(session, call, state, seq)
	}
	code := dshFailureCode(record.Status)
	if filtered {
		code = "dsh_model_content_refused"
	} else if brokenStream {
		code = "dsh_model_stream_ended"
	}
	// Proxy ledger writes can settle just after an HTTP rejection. All retries
	// here read the same receipt; none issues a model request.
	check, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	for _, delay := range []time.Duration{0, 120 * time.Millisecond, 400 * time.Millisecond} {
		if delay > 0 {
			select {
			case <-check.Done():
				return errors.New("dsh_model_outcome_unknown")
			case <-time.After(delay):
			}
		}
		_, err := g.verifyModelAccounting(check, record, func(result, code, status string, _ dshCancelledReceipt) bool {
			if !dshFinalAccountStatus(status) {
				return false
			}
			if brokenStream {
				return result == "failed" && (code == "upstream_error_event" || code == "upstream_connection_failed" || code == "upstream_timeout" || code == "usage_read_failed" || code == "usage_terminal_event_missing")
			}
			if filtered {
				return result == "succeeded" || result == "failed" && code == "upstream_http_error"
			}
			return result == "failed" && (code == "upstream_http_error" || code == "upstream_connection_failed" || code == "upstream_timeout")
		})
		if err != nil {
			continue
		}
		if err = g.noteModelRejection(pending, record.Status, code); err != nil {
			return err
		}
		return g.acceptSDKSettlement(session, call, state, seq)
	}
	if brokenStream {
		return errors.New("dsh_model_response_interrupted")
	}
	return errors.New("dsh_model_outcome_unknown")
}

type dshMemberInterruptReceipt struct {
	OperationID      string `json:"operationId"`
	Turn             int64  `json:"turn"`
	Step             int64  `json:"step"`
	TurnEndSeq       int64  `json:"turnEndSeq"`
	BeforeFirstToken bool   `json:"beforeFirstToken"`
}

// The authenticated SDK has already bound this child invocation to its root.
// Only this invocation is settled; other members and the root operation stay active.
func (g *dshLocalGateway) acceptMemberInterrupt(ctx context.Context, operation string, r dshFailedCompactionReceipt) error {
	m := r.MemberInterrupt
	if m == nil || r.State != "failed" || !validUUID(r.SessionID) || r.SessionID == g.session || !validUUID(r.InvocationID) || !validUUID(operation) || m.OperationID != operation || m.Turn <= 0 || m.Step <= 0 || r.Seq <= 0 || m.TurnEndSeq <= r.Seq {
		return errDSHCancelProof
	}
	g.mu.Lock()
	if g.pending == nil {
		g.mu.Unlock()
		return errDSHCancelProof
	}
	record := *g.pending
	g.mu.Unlock()
	if record.SessionID != g.session || record.OperationID != operation || record.CallID != r.InvocationID || record.GatewayCode != "" || record.State != "dispatched" && record.State != "delivered" {
		return errDSHCancelProof
	}
	beforeHeaders := record.Status == 0 && record.State == "dispatched" && record.Transaction == ""
	if !beforeHeaders && (record.Status < 200 || record.Status >= 300) {
		return errDSHCancelProof
	}
	check, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	receipt, err := g.verifyModelAccounting(check, record, func(result, code, status string, receipt dshCancelledReceipt) bool {
		if !dshFinalAccountStatus(status) {
			return false
		}
		if m.BeforeFirstToken || beforeHeaders {
			return receipt.TransactionID != "" && (result == "cancelled" || result == "failed" && code == "upstream_connection_failed")
		}
		return result == "cancelled" || result == "succeeded" || result == "failed"
	})
	if err != nil {
		return errDSHStoppedAccounting
	}
	data, err := json.Marshal(struct {
		Version    int                        `json:"version"`
		Record     dshLocalRecord             `json:"record"`
		MemberID   string                     `json:"memberId"`
		Seq        int64                      `json:"seq"`
		Interrupt  *dshMemberInterruptReceipt `json:"interrupt"`
		Accounting dshCancelledReceipt        `json:"accounting"`
	}{1, record, r.SessionID, r.Seq, m, receipt})
	if err != nil {
		return err
	}
	return g.archiveVerifiedModelRecord(record, filepath.Join(g.journal, r.InvocationID+".json"), data)
}

// acceptModelSettlement follows DSH: a model request the SDK durably recorded as
// failed ends only its own turn. When the platform ledger cannot prove the
// outcome, the call is archived as unverified so the next prompt can proceed.
// The ledger keeps its unknown transaction for administrative reconciliation.
func (g *dshLocalGateway) acceptModelSettlement(ctx context.Context, session, call, state string, seq int64) error {
	err := g.verifyModelSettlement(ctx, session, call, state, seq)
	if err == nil || state != "failed" || seq <= 0 || !g.requireInvocationID || session != g.session || !validUUID(call) {
		return err
	}
	if code := err.Error(); code != "dsh_model_outcome_unknown" && code != "dsh_model_response_interrupted" {
		return err
	}
	if abandonErr := g.abandonFailed(call); abandonErr != nil {
		return err
	}
	return nil
}

// abandonFailed archives the pending call as an unverified terminal failure.
// It waits for the call's HTTP handler so a late journal write cannot recreate
// the pending record after it is archived.
func (g *dshLocalGateway) abandonFailed(call string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.waitServedLocked(call) {
		return errors.New("dsh_model_outcome_unknown")
	}
	if g.pending == nil || g.pending.CallID != call || g.pending.State == "rejected" {
		return errors.New("dsh_model_outcome_unknown")
	}
	next := *g.pending
	next.State, next.Unverified = "rejected", true
	next.FailureCode = "dsh_model_service_unavailable"
	if next.Status >= 200 && next.Status < 300 {
		next.FailureCode = "dsh_model_stream_ended"
	}
	if err := g.writePending(&next); err != nil {
		return err
	}
	*g.pending = next
	return g.archivePendingLocked(call)
}

// dshModelUnsettledRelease bounds how long an unprovable model call may block
// the session. The archived record stays on disk for later repair; the turn is
// charged nothing, which is visible to the employee as an unknown result.
const dshModelUnsettledRelease = 60 * time.Second

// abandonUnsettled releases a call whose outcome neither the SDK nor the
// platform ledger could prove within the release window. It only acts on a
// record nothing is serving any more, so a live stream is never abandoned.
func (g *dshLocalGateway) abandonUnsettled(call string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.waitServedLocked(call) {
		return "", errors.New("dsh_model_outcome_unknown")
	}
	if g.pending == nil || g.pending.CallID != call || g.pending.State != "dispatched" {
		return "", errors.New("dsh_model_outcome_unknown")
	}
	info, err := os.Stat(filepath.Join(g.journal, "pending.json"))
	if err != nil || time.Since(info.ModTime()) < dshModelUnsettledRelease {
		return "", errors.New("dsh_model_outcome_unknown")
	}
	next := *g.pending
	next.State, next.Unverified = "rejected", true
	next.FailureCode = "dsh_model_unsettled"
	if err := g.writePending(&next); err != nil {
		return "", err
	}
	*g.pending = next
	if err := g.archivePendingLocked(call); err != nil {
		return "", err
	}
	return next.FailureCode, nil
}

// abandonedFailure returns the code of an unverified failure archived for the
// operation, or "" when there is none.
func (g *dshLocalGateway) abandonedFailure(operation string) string {
	if !validUUID(operation) {
		return ""
	}
	entries, err := os.ReadDir(g.journal)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.Type().IsRegular() || !strings.HasSuffix(name, ".json") || !validUUID(strings.TrimSuffix(name, ".json")) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(g.journal, name))
		var r dshLocalRecord
		if err != nil || json.Unmarshal(data, &r) != nil {
			continue
		}
		if r.SessionID == g.session && r.OperationID == operation && r.CallID+".json" == name && r.State == "rejected" && r.Unverified && dshTerminalModelFailure(r.FailureCode) {
			return r.FailureCode
		}
	}
	return ""
}

func dshDialFailed(err error) bool {
	var op *net.OpError
	return errors.As(err, &op) && (op.Op == "dial" || op.Op == "proxyconnect")
}

func dshPendingModelFailure(t acpTurn) bool {
	return t.Failed && t.Acked && !t.DSHReconciled && !t.DSHModelFailureSettled && (t.FailureCode == "dsh_model_outcome_unknown" || t.FailureCode == "dsh_model_response_interrupted")
}

// Reopening a failed conversation may recheck its account receipt. Every tool
// in the turn must already have a matching final result in the SDK journal.
func (h *relaySessionHost) recoverModelFailure(ctx context.Context, session acpSession, t acpTurn) (string, error) {
	if h.local == nil || session.ID != session.DSHSessionID {
		return "", errors.New("dsh_model_outcome_unknown")
	}
	var failure string
	check := func(ctx context.Context, p *dshProcess, local *dshLocalGateway) error {
		var events []dshConnectorHistoryEvent
		_, err := dshRepairHistory(ctx, session.ID, -1, p.historyCall, func(_ int64, raw json.RawMessage) error {
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
		target, end, err := dshTerminalToolTurn(events, t, "error")
		if err != nil {
			return err
		}
		var call string
		var at int64
		for _, e := range events[:end] {
			if e.Type != "assistant/attempt" && e.Type != "assistant/message" {
				continue
			}
			var data struct {
				Turn         int64
				InvocationID string
			}
			if err := json.Unmarshal(e.Data, &data); err != nil {
				return err
			}
			if data.Turn == target {
				call, at = data.InvocationID, *e.Seq
			}
		}
		if !validUUID(call) || at <= 0 {
			return errors.New("dsh_model_outcome_unknown")
		}
		raw, err := p.call(ctx, "aibertai/reconcile", map[string]string{"sessionId": session.ID, "invocationId": call}, nil)
		var receipt struct {
			State string
			Seq   int64
		}
		if err != nil || json.Unmarshal(raw, &receipt) != nil || receipt.State != "failed" || receipt.Seq != at {
			return errors.New("dsh_model_outcome_unknown")
		}
		local.mu.Lock()
		pending := local.pending
		owned := pending != nil && pending.SessionID == session.ID && pending.OperationID == t.Operation && pending.CallID == call
		local.mu.Unlock()
		if pending != nil {
			if !owned {
				return errors.New("dsh_model_outcome_unknown")
			}
			if err = local.acceptModelSettlement(ctx, session.ID, call, receipt.State, receipt.Seq); err != nil {
				return err
			}
		}
		failure = local.archivedFailure(t.Operation, call, "error")
		if !dshTerminalModelFailure(failure) {
			return errors.New("dsh_model_outcome_unknown")
		}
		return nil
	}
	err := reconcileDSHSession(context.WithValue(ctx, dshSDKOptionsKey{}, dshSDKOptions{CheckStopped: check}), h.live.dsh, session)
	return failure, err
}

func dshFinalAccountStatus(s string) bool {
	return s == "released" || s == "settled" || s == "rejected"
}
