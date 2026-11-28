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
	"time"
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
	}{{200, "dsh_model_stream_ended"}, {400, "dsh_model_request_rejected"}, {502, "dsh_model_service_unavailable"}, {0, "dsh_model_service_unavailable"}} {
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

// An unsettled model call blocked the session until the process restarted.
// After the release window it is archived as unsettled: charged nothing, still
// available for a later receipt check, and no longer holding the next prompt.
func TestDSHUnsettledModelCallReleasesAfterWindow(t *testing.T) {
	g := pendingSettleGateway(t, http.StatusOK)
	if _, err := g.abandonUnsettled(settleCall); err == nil {
		t.Fatal("released a call that is still inside the window")
	}
	old := time.Now().Add(-2 * dshModelUnsettledRelease)
	if err := os.Chtimes(filepath.Join(g.journal, "pending.json"), old, old); err != nil {
		t.Fatal(err)
	}
	code, err := g.abandonUnsettled(settleCall)
	if err != nil || code != "dsh_model_unsettled" {
		t.Fatalf("release=%q err=%v", code, err)
	}
	if g.pending != nil {
		t.Fatal("released call still holds the session")
	}
	if _, err := os.Stat(filepath.Join(g.journal, "pending.json")); !os.IsNotExist(err) {
		t.Fatalf("pending journal remains: %v", err)
	}
	if got := g.abandonedFailure(settleOperation); got != "dsh_model_unsettled" {
		t.Fatalf("archived code %q", got)
	}
	if err := g.resetTurnState(); err != nil {
		t.Fatalf("next turn still blocked: %v", err)
	}
}

// Only a ledger receipt that proves the call succeeded may repair a record
// left "dispatched" by builds that recorded delivery only at stream close.
func committedRecoveryGateway(t *testing.T, receipt string) *dshLocalGateway {
	t.Helper()
	c, err := newBackendClient("https://gateway.test", false)
	if err != nil {
		t.Fatal(err)
	}
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		if r.URL.Path == "/api/v1/sessions/"+settleSession+"/model-invocations/"+settleCall+"/receipt" {
			fmt.Fprint(w, receipt)
		}
		return w.Result(), nil
	})
	gateway := &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}
	journal := filepath.Join(t.TempDir(), "journal")
	if err := os.MkdirAll(journal, 0700); err != nil {
		t.Fatal(err)
	}
	record := dshLocalRecord{SessionID: settleSession, OperationID: settleOperation, CallID: settleCall, ModelID: settleModel,
		Protocol: "openai_chat_completions", Body: []byte(`{}`), State: "dispatched", Status: http.StatusOK, Finish: "stop"}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(journal, "pending.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	g, err := newDSHLocalGateway(gateway, settleSession, selectedModelConfig{ID: settleModel, Protocol: "openai_chat_completions"}, journal)
	if err != nil {
		t.Fatal(err)
	}
	g.requireInvocationID = true
	return g
}

func TestDSHCommittedDeliveryRepairsDispatchedCall(t *testing.T) {
	const tx = "44444444-4444-4444-8444-444444444444"
	g := committedRecoveryGateway(t, fmt.Sprintf(`{"status":"settled","recorded":true,"result":"succeeded","transaction_id":%q}`, tx))
	g.recoverCommittedDelivery(context.Background(), settleSession, settleCall, 7)
	if g.pending == nil || g.pending.State != "delivered" || g.pending.Transaction != tx || g.pending.Finish != "stop" {
		t.Fatalf("committed call not repaired: %+v", g.pending)
	}
	data, err := os.ReadFile(filepath.Join(g.journal, "pending.json"))
	var saved dshLocalRecord
	if err != nil || json.Unmarshal(data, &saved) != nil || saved.State != "delivered" || saved.Transaction != tx {
		t.Fatalf("repaired record not durable: %s, %v", data, err)
	}
}

func TestDSHCommittedDeliveryRequiresProof(t *testing.T) {
	for _, tc := range []struct {
		name    string
		receipt string
	}{
		{"failed result", `{"status":"settled","recorded":true,"result":"failed"}`},
		{"no final status", `{"status":"pending","recorded":true,"result":"succeeded","transaction_id":"44444444-4444-4444-8444-444444444444"}`},
		{"not recorded", `{"status":"settled","recorded":false,"result":"succeeded"}`},
		{"missing", `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := committedRecoveryGateway(t, tc.receipt)
			g.recoverCommittedDelivery(context.Background(), settleSession, settleCall, 7)
			if g.pending == nil || g.pending.State != "dispatched" {
				t.Fatalf("unproven call was delivered: %+v", g.pending)
			}
		})
	}
}

func TestDSHArchivesOnlyAnEndedPredecessorBeforeNextPrompt(t *testing.T) {
	const next = "66666666-6666-4666-8666-666666666666"
	for _, tc := range []struct {
		name     string
		previous acpTurn
		liveCall bool
		archived bool
	}{
		{"failed predecessor", acpTurn{Failed: true, Ended: time.Now()}, false, true},
		{"incomplete predecessor", acpTurn{Failed: true}, false, false},
		{"live handler", acpTurn{Failed: true, Ended: time.Now()}, true, false},
		{"goal accounting", acpTurn{Failed: true, Ended: time.Now(), GoalCheckpoint: &goalCheckpoint{}}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gateway := pendingSettleGateway(t, 200)
			if tc.liveCall {
				gateway.serving = make(chan struct{})
			}
			tc.previous.Operation = settleOperation
			row := acpSession{Turns: []acpTurn{tc.previous, {Operation: next}}}
			if err := gateway.archivePreviousFailure(row, next); err != nil {
				t.Fatal(err)
			}
			if !tc.archived {
				if gateway.pending == nil {
					t.Fatal("an invocation without a finished owner was archived")
				}
				return
			}
			if gateway.pending != nil {
				t.Fatal("the previous invocation still blocks the next message")
			}
			raw, err := os.ReadFile(filepath.Join(gateway.journal, settleCall+".json"))
			var archived dshLocalRecord
			if err != nil || json.Unmarshal(raw, &archived) != nil || !archived.Unverified || archived.CallID != settleCall ||
				archived.OperationID != settleOperation {
				t.Fatalf("the previous invocation's durable identity was lost: %+v, %v", archived, err)
			}
		})
	}
}
