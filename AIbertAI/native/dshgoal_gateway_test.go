package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDSHGoalGatewayAdmissionPrecedesDispatch(t *testing.T) {
	const session = "11111111-1111-4111-8111-111111111111"
	const modelID = "22222222-2222-4222-8222-222222222222"
	const invocation = "33333333-3333-4333-8333-333333333333"
	c, err := newBackendClient("https://gateway.test", false)
	if err != nil {
		t.Fatal(err)
	}
	sent := 0
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) { sent++; return nil, errors.New("unexpected dispatch") })
	gateway := &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}
	g, err := newDSHLocalGateway(gateway, session, selectedModelConfig{ID: modelID, Protocol: "openai_chat_completions"}, filepath.Join(t.TempDir(), "journal"))
	if err != nil {
		t.Fatal(err)
	}
	g.requireInvocationID = true
	g.admitGoalInvocation = func(id string) error {
		if id != invocation {
			t.Error("wrong invocation")
		}
		return errors.New("goal_token_budget")
	}
	request := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(fmt.Sprintf(`{"model":%q}`, modelID)))
		r.Header.Set("Authorization", "Bearer "+g.token)
		r.Header.Set("X-AIBERTAI-Invocation-ID", invocation)
		w := httptest.NewRecorder()
		g.ServeHTTP(w, r)
		return w
	}
	if w := request(); w.Code != http.StatusConflict || sent != 0 || g.pending != nil {
		t.Fatalf("denied request sent: status=%d sent=%d", w.Code, sent)
	}
	if _, err = os.Stat(filepath.Join(g.journal, "pending.json")); !os.IsNotExist(err) {
		t.Fatalf("denied request changed journal: %v", err)
	}
	if g.deniedGoalSettlement(session, invocation, "committed", 1, false) || g.deniedGoalSettlement(session, invocation, "failed", 0, false) || g.deniedGoalSettlement("foreign", invocation, "failed", 1, false) {
		t.Fatal("accepted mismatched denial evidence")
	}
	if !g.deniedGoalSettlement(session, invocation, "failed", 1, true) || g.deniedGoalSettlement(session, invocation, "failed", 1, true) {
		t.Fatal("denial evidence was not single-use")
	}
	g.admitGoalInvocation = func(id string) error { return nil }
	if w := request(); w.Code != http.StatusConflict || sent != 0 {
		t.Fatalf("request after denial was dispatched: status=%d sent=%d", w.Code, sent)
	}
	if err := g.resetTurnState(); err != nil {
		t.Fatal(err)
	}
	if w := request(); w.Code != http.StatusBadGateway || sent != 1 || g.pending == nil {
		t.Fatalf("admitted request not forwarded: status=%d sent=%d", w.Code, sent)
	}
}

func TestDSHActiveGoalRequiresBudgetAdmission(t *testing.T) {
	const session = "11111111-1111-4111-8111-111111111111"
	const modelID = "22222222-2222-4222-8222-222222222222"
	c, err := newBackendClient("https://gateway.test", false)
	if err != nil {
		t.Fatal(err)
	}
	gateway := &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{}, nil }}
	model := selectedModelConfig{ID: modelID, Protocol: "openai_chat_completions"}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{}, filepath.Join(t.TempDir(), "journal"), gateway, func(context.Context, acpSession) (selectedModelConfig, error) { return model, nil })
	if err != nil {
		t.Fatal(err)
	}
	row := acpSession{ID: session, DSHSessionID: session, Model: modelID, Goal: &desktopGoal{Current: true, Status: "active"}}
	if _, _, _, _, _, err = cfg.Prepare(context.Background(), row); err == nil || err.Error() != "dsh_goal_budget_unavailable" {
		t.Fatalf("active goal without budget accepted: %v", err)
	}
}
