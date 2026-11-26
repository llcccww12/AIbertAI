package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"time"
)

var errDSHCancelProof = errors.New("dsh_cancel_proof_unverified")

// dshCancelProof verifies an ordinary prompt's exact interrupted invocation.
// It never replays a model request or treats its usage as known.
func dshCancelProof(record dshLocalRecord, original acpTurn, events []dshConnectorHistoryEvent, receiptSeq int64) error {
	if !validUUID(record.SessionID) || !validUUID(record.CallID) || !validUUID(record.ModelID) || record.OperationID != original.Operation || !validUUID(record.OperationID) || len(original.ToolRuns) != 0 || (record.State != "dispatched" && record.State != "delivered") || record.Status < 200 || record.Status >= 300 || record.GatewayCode != "" || len(record.Body) == 0 || len(record.Body) > dshGatewayMaxRequestBytes || !json.Valid(record.Body) {
		return errDSHCancelProof
	}
	target, userSeq, err := dshPromptHistory(events, record.OperationID)
	if err != nil {
		return errDSHCancelProof
	}
	var active int64
	var invocations, matching, messages, ends int
	var targetEnded bool
	var invocationSeq, messageSeq int64 = -1, -1
	for i, event := range events {
		if event.Seq == nil || *event.Seq != int64(i) {
			return errDSHCancelProof
		}
		switch event.Type {
		case "turn/start":
			var data struct{ Turn int64 }
			if json.Unmarshal(event.Data, &data) != nil {
				return errDSHCancelProof
			}
			if targetEnded {
				return errDSHCancelProof
			}
			active = data.Turn
		case "aibertai/invocation":
			var data struct{ InvocationID, Purpose string }
			if json.Unmarshal(event.Data, &data) != nil {
				return errDSHCancelProof
			}
			if data.InvocationID == record.CallID {
				matching++
			}
			if active == target {
				invocations++
				if data.InvocationID != record.CallID || data.Purpose != "conversation" || *event.Seq <= userSeq {
					return errDSHCancelProof
				}
				invocationSeq = *event.Seq
			}
		case "assistant/message", "assistant/attempt":
			var data struct {
				Turn         int64
				InvocationID string
				Interrupted  bool
			}
			if json.Unmarshal(event.Data, &data) != nil {
				return errDSHCancelProof
			}
			if data.Turn == target && data.InvocationID != record.CallID {
				return errDSHCancelProof
			}
			if data.InvocationID == record.CallID {
				messages++
				if event.Type != "assistant/message" || data.Turn != target || !data.Interrupted || *event.Seq <= invocationSeq {
					return errDSHCancelProof
				}
				messageSeq = *event.Seq
			}
		case "tool/call", "tool/result":
			var data struct{ Turn *int64 }
			if json.Unmarshal(event.Data, &data) != nil || data.Turn == nil || *data.Turn <= 0 || *data.Turn == target || active == target {
				return errDSHCancelProof
			}
		case "turn/end":
			var data struct {
				Turn   int64
				Reason struct {
					Kind   string
					Reason struct{ Kind string }
				}
			}
			if json.Unmarshal(event.Data, &data) != nil {
				return errDSHCancelProof
			}
			if data.Turn == target {
				ends++
				if data.Reason.Kind != "aborted" || data.Reason.Reason.Kind != "user" || messageSeq < 0 || *event.Seq <= messageSeq {
					return errDSHCancelProof
				}
				targetEnded = true
			}
			active = 0
		}
	}
	if invocations != 1 || matching != 1 || messages != 1 || ends != 1 || messageSeq != receiptSeq {
		return errDSHCancelProof
	}
	return nil
}

// verify the same SDK history, tool boundary, and account receipt for both
// immediate cancellation and later recovery of an interrupted desktop run.
func (h *relaySessionHost) verifiedDSHCancelReceipt(ctx context.Context, p *dshProcess, local *dshLocalGateway, session acpSession, turn acpTurn, record dshLocalRecord, seq int64) (dshCancelledReceipt, error) {
	if err := h.dshCancelHistoryProof(ctx, p, session, turn, record, seq); err != nil {
		return dshCancelledReceipt{}, err
	}
	return local.verifyCancelledAccounting(ctx, record, false)
}

// dshCancelHistoryProof checks only the SDK's durable record of the stop.
func (h *relaySessionHost) dshCancelHistoryProof(ctx context.Context, p *dshProcess, session acpSession, turn acpTurn, record dshLocalRecord, seq int64) error {
	if err := dshConnectorAssertSettled(h.live.store.Root, session.ID); err != nil {
		return err
	}
	var events []dshConnectorHistoryEvent
	_, err := dshRepairHistory(ctx, session.DSHSessionID, -1, p.historyCall, func(_ int64, raw json.RawMessage) error {
		var event dshConnectorHistoryEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			return err
		}
		events = append(events, event)
		return nil
	})
	if err != nil || dshCancelProof(record, turn, events, seq) != nil {
		return errDSHCancelProof
	}
	return nil
}

func (h *relaySessionHost) confirmStoppedDSHReply(ctx context.Context, session acpSession, turn *acpTurn) bool {
	if h.local == nil || h.live.store == nil || h.live.store.Root == nil || session.platformGoal() != nil {
		return false
	}
	check, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if len(turn.ToolRuns) != 0 {
		err := h.verifyStoppedTools(check, session, *turn)
		turn.DSHStopObserved = errors.Is(err, errDSHStoppedAccounting)
		return err == nil
	}
	confirmed := false
	options := dshSDKOptions{CheckCancelled: func(ctx context.Context, p *dshProcess, local *dshLocalGateway, pending *dshLocalRecord, seq int64) error {
		if pending == nil {
			// The SDK's failed settlement may already have archived the call.
			if local.abandonedFailure(turn.Operation) == "" {
				return errDSHCancelProof
			}
			confirmed = true
			return nil
		}
		if err := h.dshCancelHistoryProof(ctx, p, session, *turn, *pending, seq); err != nil {
			return err
		}
		receipt, err := local.verifyCancelledAccounting(ctx, *pending, false)
		if err != nil {
			// DSH ends a stopped request with its turn. The SDK has proved the
			// stop; the platform ledger may never reach a final state.
			if err := local.abandonFailed(pending.CallID); err != nil {
				return err
			}
			confirmed = true
			return nil
		}
		if err := local.archiveCancelled(*pending, receipt, seq); err != nil {
			return err
		}
		confirmed = true
		return nil
	}}
	err := reconcileDSHSession(context.WithValue(check, dshSDKOptionsKey{}, options), h.live.dsh, session)
	return err == nil && confirmed
}

type dshCancelledAccounting struct {
	Member         *dshStoppedMember `json:"member,omitempty"`
	Version        int               `json:"version"`
	Record         dshLocalRecord    `json:"record"`
	TransactionID  string            `json:"transaction_id,omitempty"`
	AccountingID   string            `json:"accounting_id,omitempty"`
	AccountingKind string            `json:"accounting_kind,omitempty"`
	ReceiptSeq     int64             `json:"receipt_seq"`
}

type dshCancelledReceipt struct{ TransactionID, AccountingID, AccountingKind string }

func (r dshCancelledReceipt) valid(record dshLocalRecord) bool {
	billed := validUUID(r.TransactionID) && r.AccountingID == "" && r.AccountingKind == "" && (record.Transaction == "" || record.Transaction == r.TransactionID)
	owned := record.Transaction == "" && r.TransactionID == "" && validUUID(r.AccountingID) && r.AccountingKind == "owned_model_call"
	return billed || owned
}

func (m dshCancelledAccounting) receipt() dshCancelledReceipt {
	return dshCancelledReceipt{TransactionID: m.TransactionID, AccountingID: m.AccountingID, AccountingKind: m.AccountingKind}
}

func dshCancelledAccountingPath(journal, operation string) string {
	return filepath.Join(journal, "cancelled-"+operation+".json")
}

func dshReadCancelledAccounting(journal, operation string) (*dshCancelledAccounting, error) {
	if !validUUID(operation) {
		return nil, errDSHCancelProof
	}
	path := dshCancelledAccountingPath(journal, operation)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > 48<<20 {
		return nil, errDSHCancelProof
	}
	data, err := os.ReadFile(path)
	var marker dshCancelledAccounting
	if err != nil || json.Unmarshal(data, &marker) != nil || marker.Record.OperationID != operation ||
		marker.ReceiptSeq < 0 || !marker.receipt().valid(marker.Record) {
		return nil, errDSHCancelProof
	}
	switch marker.Version {
	case 1:
		if marker.Member != nil || marker.AccountingID != "" {
			return nil, errDSHCancelProof
		}
	case 2:
		if marker.Member != nil || marker.AccountingKind != "owned_model_call" {
			return nil, errDSHCancelProof
		}
	case 3:
		if marker.Member == nil || !marker.Member.valid(marker.Record.SessionID) || marker.Member.Seq != marker.ReceiptSeq {
			return nil, errDSHCancelProof
		}
	default:
		return nil, errDSHCancelProof
	}
	return &marker, nil
}

func (g *dshLocalGateway) verifyCancelledAccounting(ctx context.Context, record dshLocalRecord, requireFailedCall bool) (dshCancelledReceipt, error) {
	return g.verifyModelAccounting(ctx, record, func(result, code, _ string, receipt dshCancelledReceipt) bool {
		return !requireFailedCall || receipt.TransactionID != "" && (result == "cancelled" || result == "failed" && code == "upstream_connection_failed")
	})
}

func (g *dshLocalGateway) verifyModelAccounting(ctx context.Context, record dshLocalRecord, accept func(string, string, string, dshCancelledReceipt) bool) (dshCancelledReceipt, error) {
	if g.gateway == nil || g.gateway.client == nil || g.gateway.authorization == nil || record.SessionID != g.session || record.ModelID != g.model {
		return dshCancelledReceipt{}, errDSHCancelProof
	}
	hash, err := dshCancelRequestHash(record)
	if err != nil {
		return dshCancelledReceipt{}, err
	}
	auth, err := g.gateway.authorization(ctx)
	if err != nil {
		return dshCancelledReceipt{}, err
	}
	var out struct {
		Status         string `json:"status"`
		Recorded       bool   `json:"recorded"`
		Result         string `json:"result"`
		ErrorCode      string `json:"error_code"`
		TransactionID  string `json:"transaction_id"`
		AccountingID   string `json:"accounting_id"`
		AccountingKind string `json:"accounting_kind"`
	}
	path := "/api/v1/sessions/" + record.SessionID + "/model-invocations/" + record.CallID + "/receipt"
	input := map[string]string{"model_id": record.ModelID, "request_hash": hash}
	if record.Transaction != "" {
		input["transaction_id"] = record.Transaction
	}
	err = g.gateway.client.json(ctx, "POST", path, auth.Access, "", input, &out)
	receipt := dshCancelledReceipt{TransactionID: out.TransactionID, AccountingID: out.AccountingID, AccountingKind: out.AccountingKind}
	if err != nil || !out.Recorded || !receipt.valid(record) || !accept(out.Result, out.ErrorCode, out.Status, receipt) {
		return dshCancelledReceipt{}, errDSHCancelProof
	}
	return receipt, nil
}

// archiveCancelled keeps the original unknown usage record durable before
// freeing the single pending slot. It is safe to repeat after a crash.
func (g *dshLocalGateway) archiveCancelled(record dshLocalRecord, receipt dshCancelledReceipt, seq int64) error {
	return g.archiveStopped(record, receipt, seq, nil)
}

func (g *dshLocalGateway) archiveStopped(record dshLocalRecord, receipt dshCancelledReceipt, seq int64, member *dshStoppedMember) error {
	if member != nil && (!member.valid(record.SessionID) || member.Seq != seq) {
		return errDSHCancelProof
	}
	if !validUUID(record.OperationID) || !validUUID(record.CallID) || !receipt.valid(record) || seq < 0 {
		return errDSHCancelProof
	}
	version := 1
	if receipt.AccountingKind == "owned_model_call" {
		version = 2
	}
	if member != nil {
		version = 3
	}
	data, err := json.Marshal(dshCancelledAccounting{Member: member, Version: version, Record: record, TransactionID: receipt.TransactionID, AccountingID: receipt.AccountingID, AccountingKind: receipt.AccountingKind, ReceiptSeq: seq})
	if err != nil {
		return err
	}
	return g.archiveVerifiedModelRecord(record, dshCancelledAccountingPath(g.journal, record.OperationID), data)
}

func (g *dshLocalGateway) archiveVerifiedModelRecord(record dshLocalRecord, path string, data []byte) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.pending == nil || !reflect.DeepEqual(*g.pending, record) {
		return errDSHCancelProof
	}
	tmp, err := os.CreateTemp(g.journal, ".cancelled-")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(data)
	}
	err = errors.Join(err, tmp.Sync(), tmp.Close())
	if err != nil {
		return err
	}
	if err = os.Link(tmp.Name(), path); errors.Is(err, os.ErrExist) {
		existing, readErr := os.ReadFile(path)
		if readErr != nil || !bytes.Equal(existing, data) {
			return errDSHCancelProof
		}
	} else if err != nil {
		return err
	}
	if err = syncDSHJournal(g.journal); err != nil {
		return err
	}
	if err = os.Remove(filepath.Join(g.journal, "pending.json")); err != nil {
		return err
	}
	if err = syncDSHJournal(g.journal); err != nil {
		return err
	}
	g.pending = nil
	if g.ackSignal != nil {
		close(g.ackSignal)
		g.ackSignal = nil
	}
	return nil
}

func dshCancelRequestHash(record dshLocalRecord) (string, error) {
	path := "/v1/chat/completions"
	if record.Protocol == "openai_responses" {
		path = "/v1/responses"
	} else if record.Protocol != "openai_chat_completions" {
		return "", errDSHCancelProof
	}
	request := json.RawMessage(record.Body)
	if record.UpstreamModel != "" && record.UpstreamModel != record.ModelID {
		var object map[string]json.RawMessage
		var originalModel string
		if json.Unmarshal(request, &object) != nil || object == nil || json.Unmarshal(object["model"], &originalModel) != nil || originalModel != record.ModelID {
			return "", errDSHCancelProof
		}
		object["model"], _ = json.Marshal(record.UpstreamModel)
		rewritten, err := json.Marshal(object)
		if err != nil {
			return "", errDSHCancelProof
		}
		request = rewritten
	}
	body, err := json.Marshal(map[string]any{"model_id": record.ModelID, "path": path, "body": request, "session_id": record.SessionID})
	if err != nil {
		return "", errDSHCancelProof
	}
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
