package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const cancelledChild = "33333333-3333-4333-8333-333333333333"

func stoppedRootFixture(t *testing.T) ([]dshConnectorHistoryEvent, acpTurn) {
	t.Helper()
	rows := historyFixture()
	rows[7]["data"] = historyRow{"turn": 1, "reason": historyRow{"kind": "aborted", "reason": historyRow{"kind": "user"}}}
	bytes, err := json.Marshal(rows)
	var events []dshConnectorHistoryEvent
	if err != nil || json.Unmarshal(bytes, &events) != nil {
		t.Fatal("invalid fixture")
	}
	turn := acpTurn{Operation: historyOperation, Acked: true, Failed: true, FailureCode: "dsh_reconciliation_required"}
	for _, entry := range rows[4:6] {
		raw, err := json.Marshal(map[string]any{"event": entry})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := projectDSHTool(&turn, raw); err != nil {
			t.Fatal(err)
		}
	}
	return events, turn
}

func TestDSHStoppedRootToolOwnership(t *testing.T) {
	for _, change := range []string{"original", "foreign operation", "missing receipt", "wrong tool", "changed arguments", "pending", "wrong outcome", "changed end reason", "new turn"} {
		t.Run(change, func(t *testing.T) {
			rows, turn := stoppedRootFixture(t)
			switch change {
			case "foreign operation":
				turn.Operation = cancelledChild
			case "missing receipt":
				rows[5].Type = "other"
			case "wrong tool":
				turn.ToolRuns[0].Call.Function.Name = "bash"
			case "changed arguments":
				turn.ToolRuns[0].Call.Function.Arguments = "{}"
			case "pending":
				turn.ToolRuns[0].Pending = true
			case "wrong outcome":
				turn.ToolRuns[0].Report.Failed = 1
			case "changed end reason":
				rows[7].Data = json.RawMessage(`{"turn":1,"reason":{"kind":"aborted","reason":{"kind":"parent"}}}`)
			case "new turn":
				s := int64(8)
				rows = append(rows, dshConnectorHistoryEvent{Seq: &s, Type: "turn/start", Data: json.RawMessage(`{"turn":2}`)})
			}
			tg, end, err := dshStoppedToolTurn(rows, turn)
			if change == "original" {
				if err != nil || tg != 1 || end != 7 || !dshStoppedToolsCandidate(turn) {
					t.Fatal(tg, end, err)
				}
			} else if err == nil {
				t.Fatal("accepted", change)
			}
		})
	}
}

func TestDSHStoppedMemberProofBinding(t *testing.T) {
	record := dshLocalRecord{SessionID: historySession, OperationID: historyOperation, CallID: cancelledChild}
	for _, bad := range []string{"", "unknown", "root as child", "invalid step", "different root end", "no child stop", "committed root"} {
		t.Run(bad, func(t *testing.T) {
			calls := 0
			rpc := func(_ context.Context, method string, params any) (json.RawMessage, error) {
				calls++
				args := params.(map[string]string)
				if args["sessionId"] != record.SessionID || args["invocationId"] != record.CallID {
					t.Fatal("wrong owner")
				}
				if method == "aibertai/reconcile" {
					if bad == "committed root" {
						return json.RawMessage(`{"state":"committed","seq":5}`), nil
					}
					return json.RawMessage(`{"state":"unknown"}`), nil
				}
				if method != "aibertai/team-cancel-proof" || args["operationId"] != record.OperationID {
					t.Fatal("unexpected RPC")
				}
				data := historyRow{"state": "cancelled", "memberId": cancelledChild, "seq": 74, "turn": 1, "step": 3, "rootEndSeq": 7, "childEndSeq": 76}
				switch bad {
				case "unknown":
					data["state"] = "unknown"
				case "root as child":
					data["memberId"] = record.SessionID
				case "invalid step":
					delete(data, "step")
				case "different root end":
					data["rootEndSeq"] = 8
				case "no child stop":
					data["childEndSeq"] = 74
				}
				return json.Marshal(data)
			}
			rows, _ := stoppedRootFixture(t)
			seq, member, err := dshStoppedModel(t.Context(), rpc, record, rows, 1, 7)
			if bad == "" {
				if err != nil || seq != 74 || member == nil || member.MemberID != cancelledChild {
					t.Fatal("missing proof", err)
				}
			} else if err == nil {
				t.Fatal("unproven child")
			}
			if bad == "committed root" && calls != 1 {
				t.Fatal("foreign branch queried for settled root")
			}
		})
	}
}

func TestDSHStoppedMemberReceiptRetry(t *testing.T) {
	dir := t.TempDir()
	record := dshLocalRecord{OperationID: historyOperation, SessionID: historySession, CallID: cancelledChild}
	receipt := dshCancelledReceipt{TransactionID: "44444444-4444-4444-8444-444444444444"}
	g := dshLocalGateway{journal: dir, pending: &record}
	proof := dshStoppedMember{MemberID: cancelledChild, Seq: 74, Turn: 1, Step: 3, RootEndSeq: 7, ChildEndSeq: 76}
	path := filepath.Join(dir, "pending.json")
	if err := os.WriteFile(path, []byte("pending"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := g.archiveStopped(record, receipt, 73, &proof); err == nil {
		t.Fatal("accepted mismatched sequence")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("lost pending receipt")
	}
	if err := g.archiveStopped(record, receipt, 74, &proof); err != nil {
		t.Fatal(err)
	}
	m, err := dshReadCancelledAccounting(dir, historyOperation)
	if err != nil || m.Version != 3 || m.Member == nil || *m.Member != proof || m.receipt() != receipt {
		t.Fatal("invalid archived proof", err)
	}
	// Simulate a crash between archiving the exact receipt and removing pending.json.
	if err := os.WriteFile(path, []byte("pending"), 0600); err != nil {
		t.Fatal(err)
	}
	g.pending = &record
	if err := g.archiveStopped(record, receipt, 74, &proof); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("journal not cleared")
	}
}

func TestDSHStoppedPendingAccountingProjection(t *testing.T) {
	_, turn := stoppedRootFixture(t)
	row := acpSession{ID: historySession, Engine: "dsh", Turns: []acpTurn{turn}}
	project := func(want bool) {
		t.Helper()
		_, meta := row.workspaceMetadata()["dshStopAccountingPending"]
		message := row.responseMessage(0, map[string]any{})
		_, waiting := message["dshStopAccountingPending"]
		if meta != want || waiting != want {
			t.Fatal("wrong accounting projection", meta, waiting, want)
		}
	}
	project(false)
	row.Turns[0].DSHStopObserved = true
	project(true)
	if !dshWorkspaceRecoveryAvailable(row) || !dshPromptRecoveryAvailable(row) {
		t.Fatal("receipt retry unavailable")
	}
	row.Turns[0].DSHReconciled = true
	row.Turns[0].Interrupted = true
	row.Turns[0].DSHCancelConfirmed = true
	project(false)
	if row.responseMessage(0, map[string]any{})["interrupted"] != true {
		t.Fatal("accounted stop not shown")
	}
}
