package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	settleSession   = "11111111-1111-4111-8111-111111111111"
	settleModel     = "22222222-2222-4222-8222-222222222222"
	settleCall      = "33333333-3333-4333-8333-333333333333"
	settleOperation = "55555555-5555-4555-8555-555555555555"
)

func pendingSettleGateway(t *testing.T, status int) *dshLocalGateway {
	t.Helper()
	journal := filepath.Join(t.TempDir(), "journal")
	if err := os.MkdirAll(journal, 0700); err != nil {
		t.Fatal(err)
	}
	record := dshLocalRecord{SessionID: settleSession, OperationID: settleOperation, CallID: settleCall, ModelID: settleModel, Protocol: "openai_chat_completions", Body: []byte(`{}`), State: "dispatched", Status: status}
	data, _ := json.Marshal(record)
	if err := os.WriteFile(filepath.Join(journal, "pending.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	g, err := newDSHLocalGateway(nil, settleSession, selectedModelConfig{ID: settleModel, Protocol: "openai_chat_completions"}, journal)
	if err != nil {
		t.Fatal(err)
	}
	g.requireInvocationID = true
	return g
}

// Like DSH, a model call the SDK recorded as failed ends only its own turn,
// even when the platform ledger cannot prove the outcome.
func TestDSHFailedModelCallSettlesWithoutLedgerProof(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
	}{{200, "dsh_model_stream_ended"}, {502, "dsh_model_service_unavailable"}, {0, "dsh_model_service_unavailable"}} {
		g := pendingSettleGateway(t, tc.status)
		if err := g.acceptModelSettlement(context.Background(), settleSession, settleCall, "failed", 7); err != nil {
			t.Fatalf("status %d: %v", tc.status, err)
		}
		if g.pending != nil {
			t.Fatalf("status %d: pending call was not archived", tc.status)
		}
		if _, err := os.Stat(filepath.Join(g.journal, "pending.json")); !os.IsNotExist(err) {
			t.Fatalf("status %d: pending journal remains: %v", tc.status, err)
		}
		if code := g.archivedFailure(settleOperation, settleCall, "error"); code != tc.code {
			t.Fatalf("status %d: archived code %q", tc.status, code)
		}
		if code := g.abandonedFailure(settleOperation); code != tc.code {
			t.Fatalf("status %d: abandoned code %q", tc.status, code)
		}
		if err := g.resetTurnState(); err != nil {
			t.Fatalf("status %d: next turn blocked: %v", tc.status, err)
		}
	}
}

func TestDSHUnprovenModelCallStaysPendingWithoutSDKFailure(t *testing.T) {
	g := pendingSettleGateway(t, 200)
	if err := g.acceptModelSettlement(context.Background(), settleSession, settleCall, "unknown", 7); err == nil {
		t.Fatal("unknown SDK state settled the call")
	}
	if err := g.acceptModelSettlement(context.Background(), settleSession, settleCall, "failed", 0); err == nil {
		t.Fatal("failure without a durable SDK sequence settled the call")
	}
	if g.pending == nil || g.resetTurnState() == nil {
		t.Fatal("unproven call no longer blocks dispatch")
	}
}

func TestDSHStuckModelOnlyTurn(t *testing.T) {
	base := acpTurn{Failed: true, Operation: settleOperation, FailureCode: "dsh_model_outcome_unknown"}
	if !dshStuckModelOnlyTurn(base) {
		t.Fatal("model-only failure not recognised")
	}
	for name, turn := range map[string]acpTurn{
		"tools":     func() acpTurn { t := base; t.ToolRuns = []workspaceToolRun{{}}; return t }(),
		"settled":   func() acpTurn { t := base; t.DSHModelFailureSettled = true; return t }(),
		"member":    func() acpTurn { t := base; t.DSHMemberID = "m"; return t }(),
		"write":     func() acpTurn { t := base; t.FailureCode = "dsh_write_outcome_unknown"; return t }(),
		"connector": func() acpTurn { t := base; t.FailureCode = "dsh_connector_outcome_unknown"; return t }(),
	} {
		if dshStuckModelOnlyTurn(turn) {
			t.Fatalf("%s turn treated as model-only", name)
		}
	}
}

// A connection to the platform that never opened carried no request, so the
// gateway retries it like DSH's provider retry policy.
func TestDSHGatewayRetriesPlatformDialFailure(t *testing.T) {
	const keyID = "44444444-4444-4444-8444-444444444444"
	c, err := newBackendClient("https://gateway.test", false)
	if err != nil {
		t.Fatal(err)
	}
	posts := 0
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch {
		case r.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, settleModel)
		case r.URL.Path == "/api/v1/api-keys" && r.Method == "POST":
			fmt.Fprintf(w, `{"id":%q,"api_key":"key"}`, keyID)
		case r.URL.Path == "/v1/chat/completions":
			posts++
			if posts < 3 {
				return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
			}
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
		}
		return w.Result(), nil
	})
	gateway := &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}
	g, err := newDSHLocalGateway(gateway, settleSession, selectedModelConfig{ID: settleModel, Protocol: "openai_chat_completions"}, filepath.Join(t.TempDir(), "journal"))
	if err != nil {
		t.Fatal(err)
	}
	g.requireInvocationID = true
	r := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(fmt.Sprintf(`{"model":%q,"stream":true}`, settleModel)))
	r.Header.Set("Authorization", "Bearer "+g.token)
	r.Header.Set("X-AIBERTAI-Invocation-ID", settleCall)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, r)
	if w.Code != http.StatusOK || posts != 3 || g.pending == nil || g.pending.State != "delivered" {
		t.Fatalf("status=%d posts=%d pending=%+v", w.Code, posts, g.pending)
	}
}

func TestDSHGatewayDialFailureIsProvenNonDispatch(t *testing.T) {
	const keyID = "44444444-4444-4444-8444-444444444444"
	c, err := newBackendClient("https://gateway.test", false)
	if err != nil {
		t.Fatal(err)
	}
	posts := 0
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch {
		case r.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, settleModel)
		case r.URL.Path == "/api/v1/api-keys" && r.Method == "POST":
			fmt.Fprintf(w, `{"id":%q,"api_key":"key"}`, keyID)
		case r.URL.Path == "/v1/chat/completions":
			posts++
			return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
		}
		return w.Result(), nil
	})
	gateway := &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}
	g, err := newDSHLocalGateway(gateway, settleSession, selectedModelConfig{ID: settleModel, Protocol: "openai_chat_completions"}, filepath.Join(t.TempDir(), "journal"))
	if err != nil {
		t.Fatal(err)
	}
	g.requireInvocationID = true
	r := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(fmt.Sprintf(`{"model":%q,"stream":true}`, settleModel)))
	r.Header.Set("Authorization", "Bearer "+g.token)
	r.Header.Set("X-AIBERTAI-Invocation-ID", settleCall)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, r)
	if posts != 3 || w.Code != http.StatusBadGateway || g.pending == nil || g.pending.State != "rejected" || !dshTerminalModelFailure(g.pending.FailureCode) {
		t.Fatalf("status=%d posts=%d pending=%+v", w.Code, posts, g.pending)
	}
}
