package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDSHNativeFailedTail(t *testing.T) {
	record := dshLocalRecord{SessionID: historySession, CallID: "33333333-3333-4333-8333-333333333333"}
	const resultSeq = 24
	for _, tc := range []string{"disposed", "user", "resumed SDK", "invalid resume marker", "wrong invocation", "later model", "other tool", "wrong turn", "failed receipt mismatch", "normal end", "foreign abort", "unfinished"} {
		t.Run(tc, func(t *testing.T) {
			kind := "disposed"
			if tc == "user" || tc == "foreign abort" {
				kind = tc
			}
			rows := []historyRow{
				{"type": "step/end", "data": historyRow{"turn": 1, "step": 2}},
				{"type": "step/start", "data": historyRow{"turn": 1, "step": 3}},
				{"type": "aibertai/invocation", "data": historyRow{"invocationId": record.CallID, "purpose": "conversation"}},
				{"type": "assistant/attempt", "data": historyRow{"turn": 1, "step": 3, "invocationId": record.CallID}},
				{"type": "step/end", "data": historyRow{"turn": 1, "step": 3}},
				{"type": "turn/end", "data": historyRow{"turn": 1, "reason": historyRow{"kind": "aborted", "reason": historyRow{"kind": kind}}}},
			}
			seq := int64(resultSeq + 4)
			switch tc {
			case "wrong invocation":
				rows[2]["data"].(historyRow)["invocationId"] = "other"
			case "invalid resume marker":
				rows = append(rows, historyRow{"type": "session/end-seed", "data": historyRow{"unexpected": true}})
			case "resumed SDK":
				rows = append(rows, historyRow{"type": "session/end-seed", "data": historyRow{}})
			case "later model":
				rows = append(rows, historyRow{"type": "aibertai/invocation", "data": historyRow{}})
			case "other tool":
				rows[2]["type"] = "tool/call"
			case "wrong turn":
				rows[3]["data"].(historyRow)["turn"] = 2
			case "failed receipt mismatch":
				seq++
			case "normal end":
				rows[5]["data"].(historyRow)["reason"] = historyRow{"kind": "completed"}
			case "unfinished":
				rows = rows[:5]
			}
			for i, r := range rows {
				r["seq"] = resultSeq + 1 + i
			}
			fetch := func(_ context.Context, params map[string]any) (json.RawMessage, error) {
				if params["sessionId"] != historySession || params["afterSeq"] != int64(resultSeq) {
					return nil, fmt.Errorf("wrong history request")
				}
				return json.Marshal(historyRow{"sessionId": historySession, "throughSeq": resultSeq + len(rows), "events": rows})
			}
			err := dshNativeFailedTail(t.Context(), record, resultSeq, seq, 1, 2, fetch)
			want := tc == "disposed" || tc == "user" || tc == "resumed SDK"
			if (err == nil) != want {
				t.Fatalf("verified=%v want=%v: %v", err == nil, want, err)
			}
		})
	}
}

func TestDSHNativeRecoveryRequiresTerminalAccounting(t *testing.T) {
	const transaction = "44444444-4444-4444-8444-444444444444"
	record := dshLocalRecord{SessionID: historySession, OperationID: historyOperation, CallID: "33333333-3333-4333-8333-333333333333", ModelID: "55555555-5555-4555-8555-555555555555", Protocol: "openai_responses", Body: []byte(`{"model":"fixture"}`)}
	for _, tc := range []struct {
		result, code string
		recorded, ok bool
	}{
		{"failed", "upstream_connection_failed", true, true}, {"cancelled", "", true, true}, {"failed", "upstream_connection_failed", false, false}, {"succeeded", "", true, false}, {"failed", "different_error", true, false}, {"", "", true, false},
	} {
		t.Run(fmt.Sprintf("%s_%s_%t", tc.result, tc.code, tc.recorded), func(t *testing.T) {
			client, err := newBackendClient("https://receipt.invalid", false)
			if err != nil {
				t.Fatal(err)
			}
			client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != "POST" || r.URL.Path != "/api/v1/sessions/"+record.SessionID+"/model-invocations/"+record.CallID+"/receipt" {
					t.Fatalf("unexpected request %s", r.URL.Path)
				}
				w := httptest.NewRecorder()
				_ = json.NewEncoder(w).Encode(map[string]any{"recorded": tc.recorded, "transaction_id": transaction, "result": tc.result, "error_code": tc.code})
				return w.Result(), nil
			})
			g := dshLocalGateway{session: record.SessionID, model: record.ModelID, gateway: &dshGateway{client: client, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}}
			_, err = g.verifyCancelledAccounting(t.Context(), record, true)
			if (err == nil) != tc.ok {
				t.Fatalf("verified=%t want %t", err == nil, tc.ok)
			}
		})
	}
}

func TestDSHUnavailableBashApprovalMustEndTheRecordedTurn(t *testing.T) {
	for _, tc := range []struct {
		name, outcome, code string
		extra, want         bool
	}{
		{"unavailable", "unavailable", "SANDBOX_ESCALATION_UNAVAILABLE", false, true},
		{"approved", "approved", "SANDBOX_ESCALATION_UNAVAILABLE", false, false},
		{"untyped failure", "unavailable", "OTHER", false, false},
		{"later activity", "unavailable", "SANDBOX_ESCALATION_UNAVAILABLE", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := historyFixture()
			call := rows[4]["data"].(map[string]any)
			call["name"] = "bash"
			result := rows[5]["data"].(map[string]any)
			result["error"] = map[string]string{"code": tc.code}
			result["message"].(map[string]any)["isError"] = true
			end := rows[7]["data"].(map[string]any)
			end["reason"] = map[string]any{"kind": "aborted", "reason": map[string]string{"kind": "disposed"}}
			decision := []historyRow{
				{"type": "approval/asked", "data": map[string]string{"id": historySession, "callId": "call-1", "toolName": "bash"}},
				{"type": "approval/decided", "data": map[string]string{"id": historySession, "outcome": tc.outcome}},
			}
			tail := append(decision, rows[5:]...)
			rows = append(rows[:5], tail...)
			if tc.extra {
				rows = append(rows, historyRow{"type": "job/registered", "data": map[string]any{}})
			}
			for i, row := range rows {
				row["seq"] = i
			}
			_, err := dshMatchToolHistory(t.Context(), historySession, historyOperation, "call-1", "bash", json.RawMessage(`{"n":9007199254740993}`), historyPager(t, rows, ""), true)
			if (err == nil) != tc.want {
				t.Fatalf("verified=%v, err=%v", err == nil, err)
			}
		})
	}
}
