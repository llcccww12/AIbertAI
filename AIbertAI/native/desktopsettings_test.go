package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestDesktopSettingsPersistAndReset(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	live := &acpLive{store: &desktopSessionStore{Root: root}}
	const goals = "_monkeyai/config/goals/set"
	if live.settings.goalBudget() != defaultGoalBudget {
		t.Fatal("missing default")
	}
	if err := live.saveDesktopSettings(goals, json.RawMessage(`{"default_token_budget":12345}`)); err != nil {
		t.Fatal(err)
	}
	if err := live.saveDesktopSettings(goals, json.RawMessage(`{"default_token_budget":0}`)); err == nil || live.settings.goalBudget() != 12345 {
		t.Fatal("invalid value changed the setting")
	}
	if err := live.saveDesktopSettings("_monkeyai/config/display/set", json.RawMessage(`{"tool_summaries":false}`)); err != nil {
		t.Fatal(err)
	}
	reopened := &acpLive{store: live.store}
	if err := reopened.loadDesktopSettings(); err != nil {
		t.Fatal(err)
	}
	if reopened.settings.goalBudget() != 12345 || reopened.settings.display()["tool_summaries"] {
		t.Fatal("settings were not retained")
	}
	if err := reopened.saveDesktopSettings(goals, json.RawMessage(`{"default_token_budget":null}`)); err != nil || reopened.settings.goalBudget() != defaultGoalBudget {
		t.Fatal("reset failed", err)
	}
}

func TestDesktopBillingUsesPrivateAccountAndFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fixture" || r.Method != "GET" {
			t.Errorf("request must use the logged-in account")
		}
		switch r.URL.Path {
		case "/api/v1/billing/account":
			_, _ = w.Write([]byte(`{"balance":"125.000000"}`))
		case "/api/v1/settings":
			_, _ = w.Write([]byte(`{"settings":{"billing":{"enabled":true}}}`))
		case "/api/v1/billing/entries":
			q := r.URL.Query()
			if q.Get("entry_type") != "refund" || q.Get("page_size") != "25" || q.Get("content") != "中文 & 条目" || q.Has("user_id") {
				t.Errorf("filters = %s", q)
			}
			_, _ = w.Write([]byte(`{"total":1,"page_size":25,"items":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	host := &relaySessionHost{live: &acpLive{client: &client{server.URL, server.Client()}, auth: authorization{Access: "fixture"}}}
	for _, tc := range []struct{ method, input, want string }{
		{"_monkeyai/billing/account", `{}`, `{"data":{"account":{"balance":"125.000000"},"policy":{"enabled":true}}}`},
		{"_monkeyai/billing/entries", `{"page_size":25,"entry_type":"refund","content":"中文 & 条目","user_id":"ignored"}`, `{"data":{"total":1,"page_size":25,"items":[]}}`},
	} {
		result, err := host.localQuery(t.Context(), tc.method, json.RawMessage(tc.input))
		if err != nil {
			t.Fatal(err)
		}
		if tc.method == "_monkeyai/billing/account" {
			overview := result.(map[string]any)["data"].(map[string]any)
			if fetchedAt, ok := overview["fetchedAt"].(time.Time); !ok || fetchedAt.IsZero() {
				t.Fatalf("missing billing fetch time: %v", overview["fetchedAt"])
			}
			delete(overview, "fetchedAt")
		}
		if b, err := json.Marshal(result); err != nil || string(b) != tc.want {
			t.Fatalf("response %s, want %s (%v)", b, tc.want, err)
		}
	}
}

func TestDesktopThinkingAndDraftsSurviveRestart(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	const model = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	original := &acpLive{store: &desktopSessionStore{Root: root}, cwd: root.Name(), operations: map[string]string{}, sessions: map[string]acpSession{}}
	h := &relaySessionHost{live: original}
	const method = "_monkeyai/preferences/model-thinking/set"
	if _, err := h.localQuery(t.Context(), method, json.RawMessage(`{"model":"`+model+`","thinking":{"enabled":true,"effort":"low"}}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := h.localQuery(t.Context(), method, json.RawMessage(`{"model":"`+model+`","thinking":{"enabled":false,"effort":"low"}}`)); err == nil {
		t.Fatal("invalid thinking choice accepted")
	}
	draft := json.RawMessage(`{"target":{"newSession":"general"},"draft":{"text":"unfinished","files":[]}}`)
	if _, err := h.localQuery(t.Context(), "_monkeyai/draft/set", draft); err != nil {
		t.Fatal(err)
	}
	reopened := &acpLive{store: original.store, cwd: original.cwd, operations: map[string]string{}, sessions: map[string]acpSession{}}
	if err := reopened.loadDesktopSettings(); err != nil {
		t.Fatal(err)
	}
	id, err := reopened.createInWorkspace(model, "", "", model, expertSelection{}, nil, "", desktopMediaModels{})
	if err != nil {
		t.Fatal(err)
	}
	session, err := reopened.load(id)
	if err != nil || session.Thinking == nil || session.Thinking.Effort != "low" {
		t.Fatalf("saved model default was not restored: %v", err)
	}
	opts := session.Thinking.requestFields(selectedModelConfig{Protocol: "openai_responses", SupportsReasoning: true})
	if opts["reasoning"].(map[string]string)["effort"] != "low" {
		t.Fatal("missing reasoning effort")
	}
	h.live = reopened
	drafts, err := h.localQuery(t.Context(), "_monkeyai/drafts", json.RawMessage(`{}`))
	if b, _ := json.Marshal(drafts); err != nil || string(b) != `{"sessions":{},"newSessions":{"general":{"text":"unfinished","files":[]}}}` {
		t.Fatalf("draft not restored: %s (%v)", b, err)
	}
}
