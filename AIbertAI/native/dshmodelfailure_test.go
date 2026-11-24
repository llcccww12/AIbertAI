package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDSHScopedMemberInterruptRequiresFinalLedger(t *testing.T) {
	for _, tc := range []struct {
		name, status, result string
		first, wrongOp, ok   bool
	}{
		{"settled cancellation", "released", "cancelled", true, false, true},
		{"known usage", "settled", "succeeded", false, false, true},
		{"first token success", "settled", "succeeded", true, false, false},
		{"uncertain account", "unknown", "cancelled", true, false, false},
		{"settling account", "settling", "cancelled", false, false, false},
		{"older receipt", "", "cancelled", false, false, false},
		{"another operator", "released", "cancelled", false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := "12121212-1212-4212-8212-121212121212"
			tx := "45454545-4545-4545-8545-454545454545"
			m := "56565656-5656-4656-8656-565656565656"
			dir := t.TempDir()
			rec := dshLocalRecord{SessionID: historySession, OperationID: historyOperation, CallID: id, ModelID: m, Protocol: "openai_chat_completions", Body: []byte(fmt.Sprintf(`{"model":%q,"messages":[]}`, m)), State: "dispatched", Status: 200, Transaction: tx}
			c, err := newClient("http://127.0.0.1:8077")
			if err != nil {
				t.Fatal(err)
			}
			c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/api/v1/sessions/"+historySession+"/model-invocations/"+id+"/receipt" {
					t.Fatal("unexpected account request")
				}
				w := httptest.NewRecorder()
				fmt.Fprintf(w, `{"recorded":true,"status":%q,"result":%q,"transaction_id":%q}`, tc.status, tc.result, tx)
				return w.Result(), nil
			})
			g := dshLocalGateway{journal: dir, session: historySession, model: m, gateway: &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "test"}, nil }}, pending: &rec}
			if err := g.writePending(&rec); err != nil {
				t.Fatal(err)
			}
			proof := &dshMemberInterruptReceipt{OperationID: historyOperation, Turn: 1, Step: 5, TurnEndSeq: 11, BeforeFirstToken: tc.first}
			if tc.wrongOp {
				proof.OperationID = cancelledChild
			}
			err = g.acceptMemberInterrupt(t.Context(), historyOperation, dshFailedCompactionReceipt{SessionID: cancelledChild, InvocationID: id, State: "failed", Seq: 9, MemberInterrupt: proof})
			if (err == nil) != tc.ok {
				t.Fatalf("outcome=%v", err)
			}
			if (g.pending == nil) != tc.ok {
				t.Fatal("incorrect journal admission")
			}
			if _, err := os.Stat(filepath.Join(dir, id+".json")); (err == nil) != tc.ok {
				t.Fatal("incorrect call receipt")
			}
			if _, err := os.Stat(dshCancelledAccountingPath(dir, historyOperation)); !os.IsNotExist(err) {
				t.Fatal("child receipt changed root operation")
			}
		})
	}
}

func TestDSHWireTerminalReasons(t *testing.T) {
	for _, tc := range []struct {
		name, protocol, body, finish string
		complete                     bool
	}{
		{"chat limit", "openai_chat_completions", "data: {\"choices\":[{\"finish_reason\":\"length\"}]}\n\ndata: [DONE]\n\n", "length", true},
		{"chat filter", "openai_chat_completions", "data: {\"choices\":[{\"finish_reason\":\"content_filter\"}]}\n\ndata: [DONE]\n\n", "content_filter", true},
		{"missing done", "openai_chat_completions", "data: {\"choices\":[{\"finish_reason\":\"content_filter\"}]}\n\n", "content_filter", false},
		{"responses limit", "openai_responses", "data: {\"type\":\"response.incomplete\",\"response\":{\"status\":\"incomplete\",\"incomplete_details\":{\"reason\":\"max_output_tokens\"}}}\n\n", "length", true},
		{"responses filter", "openai_responses", "data: {\"type\":\"response.incomplete\",\"response\":{\"status\":\"incomplete\",\"incomplete_details\":{\"reason\":\"content_filter\"}}}\n\n", "content_filter", true},
		{"unknown cause", "openai_responses", "data: {\"type\":\"response.incomplete\",\"response\":{\"status\":\"incomplete\",\"incomplete_details\":{\"reason\":\"unknown\"}}}\n\n", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := dshStreamCompletion{valid: true, protocol: tc.protocol}
			s.add([]byte(tc.body))
			if s.terminal != tc.finish || s.complete() != tc.complete {
				t.Fatalf("complete=%v terminal=%q", s.complete(), s.terminal)
			}
		})
	}
}
