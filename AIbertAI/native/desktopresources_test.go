package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestPrivateDesktopResources(t *testing.T) {
	const selected = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	const other = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	const sessionID = "c3bd985c-9f71-4b85-937a-c41b1b104d38"
	const userID = "c4bd985c-9f71-4b85-937a-c41b1b104d38"
	saved := false
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer account" {
			t.Error("resource call must use account authorization")
		}
		var body string
		switch r.Method + " " + r.URL.Path {
		case "GET /api/v1/models":
			body = `{"models":[{"id":"` + selected + `","protocol":"openai_chat_completions","kind":"text"},` +
				`{"id":"` + other + `","display_name":"My model","kind":"text","protocol":"openai_responses","ownership_type":"user","user":{"id":"` + userID + `"}}],"total_count":2}`
		case "PUT /api/v1/models/" + other:
			var input struct {
				ModelID string `json:"model_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.ModelID != "personal" {
				t.Error("model input was not forwarded")
			}
			saved = true
			body = `{}`
		case "GET /api/v1/skills":
			body = `{"skills":[{"id":"` + other + `","name":"Uploaded skill","ownership_type":"system","package_sha256":"fixture","tags":null}],"total_count":1}`
		case "GET /api/v1/skills/" + other + "/content":
			body = `{"content":"Uploaded instructions"}`
		case "DELETE /api/v1/skills/" + other:
			deleted = true
			body = `{}`
		case "GET /api/v1/experts":
			body = `{"experts":[{"id":"` + other + `","name":"Private expert"}],"total_count":1}`
		case "PUT /api/v1/experts/" + other:
			var change struct{ Name string }
			if r.Header.Get("If-Match") != `"4"` || json.NewDecoder(r.Body).Decode(&change) != nil || change.Name != "Updated expert" {
				t.Error("expert update did not preserve the resource ID and ETag")
			}
			w.Header().Set("ETag", `"5"`)
			body = `{"id":"` + other + `","name":"Updated expert"}`
		case "GET /api/v1/rules":
			body = `{"rules":[{"id":"` + other + `","name":"My rule","content":"Be brief"}],"total_count":1}`
		case "GET /api/v1/connectors":
			body = `{"connectors":[],"total_count":0}`
		case "GET /api/v1/connectors/" + other + "/credentials":
			body = `{"items":[{"id":"` + other + `"}]}`
		case "POST /api/v1/connectors/" + other + "/test":
			body = `{"tool_count":2}`
		default:
			t.Errorf("unexpected route: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	old := modelID
	modelID = selected
	defer func() { modelID = old }()
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	live := &acpLive{
		client: &client{server: server.URL, http: server.Client()},
		auth:   authorization{Access: "account"}, protocol: "openai_chat_completions",
		store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{sessionID: {ID: sessionID, Model: selected}},
	}
	host := &relaySessionHost{live: live, local: &desktopRuntimeAccess{userID: userID}}
	cfg, err := host.localQuery(t.Context(), "_monkeyai/config/models/save", json.RawMessage(`{"id":"`+other+`","input":{"model_id":"personal","display_name":"My model","base_url":"https://models.example"}}`))
	if err != nil || !saved {
		t.Fatal("save failed", err)
	}
	raw, err := json.Marshal(cfg)
	if err != nil || !strings.Contains(string(raw), `"source":"mine"`) || !strings.Contains(string(raw), "Private expert") {
		t.Fatalf("catalog not refreshed: %s (%v)", raw, err)
	}
	rules := cfg.(map[string]any)["server"].(map[string]any)["rules"].([]json.RawMessage)
	if len(rules) != 1 || !strings.Contains(string(rules[0]), "My rule") {
		t.Fatalf("personal rules missing from catalog: %s", rules)
	}
	skills := cfg.(map[string]any)["server"].(map[string]any)["skills"].([]map[string]any)
	if len(skills) != 1 || skills[0]["id"] != other || skills[0]["resourceId"] != other ||
		skills[0]["ownership_type"] != "system" || skills[0]["description"] != "" || len(skills[0]["tags"].([]any)) != 0 {
		t.Fatalf("invalid visible skill catalog: %+v", skills)
	}
	content, err := host.localQuery(t.Context(), "_monkeyai/resources/skills/content",
		json.RawMessage(`{"id":"`+other+`"}`))
	if err != nil || string(content.(json.RawMessage)) != `{"content":"Uploaded instructions"}` {
		t.Fatalf("uploaded skill body unavailable: %v", err)
	}
	_, err = host.setDesktopSessionModel(t.Context(), "_monkeyai/session/model/set", json.RawMessage(`{"sessionId":"`+sessionID+`","model":"`+other+`","thinking":{"enabled":true,"effort":"low"}}`))
	if err != nil || live.sessions[sessionID].Model != other {
		t.Fatal("model change not retained", err)
	}
	change, err := host.setDesktopSessionModel(t.Context(), "_monkeyai/session/thinking/set", json.RawMessage(`{"sessionId":"`+sessionID+`","thinking":{"enabled":false}}`))
	if err != nil || live.sessions[sessionID].Model != other || *change.(map[string]any)["thinking"].(*modelThinking).Enabled {
		t.Fatal("session thinking choice not retained", err)
	}
	credentials, err := host.localQuery(t.Context(), "_monkeyai/resources/connectors/credentials", json.RawMessage(`{"id":"`+other+`"}`))
	if err != nil || len(credentials.([]json.RawMessage)) != 1 {
		t.Fatal("credential list not adapted", err)
	}
	result, err := host.localQuery(t.Context(), "_monkeyai/resources/connectors/test", json.RawMessage(`{"id":"`+other+`"}`))
	if err != nil || string(result.(map[string]any)["result"].(json.RawMessage)) != `{"tool_count":2}` {
		t.Fatal("connector test failed", err)
	}
	_, err = host.desktopResource(t.Context(), "_monkeyai/resources/save",
		json.RawMessage(`{"kind":"expert","id":"`+other+`","etag":"\"4\"","input":{"name":"Updated expert"}}`))
	if err != nil {
		t.Fatal("expert edit failed through the private resource API", err)
	}
	_, err = host.desktopResource(t.Context(), "_monkeyai/resources/delete",
		json.RawMessage(`{"kind":"skill","id":"`+other+`","etag":"\"7\""}`))
	if err != nil || !deleted {
		t.Fatal("skill delete failed through the private resource API", err)
	}
}

func TestNativeDraftModelProvider(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer example-key" {
			t.Error("provider key was not set")
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /v1/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"fixture"}]}`))
		case "POST /v1/chat/completions":
			var input struct{ Model string }
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Model != "fixture" {
				t.Error("incorrect draft model")
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"OK"}}]}`))
		default:
			t.Errorf("unexpected provider route %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	draft := json.RawMessage(`{"protocol":"openai_chat_completions","base_url":"` + provider.URL + `/v1","api_key":"example-key","model_id":"fixture"}`)
	for _, tc := range []struct {
		discover bool
		want     string
	}{
		{true, `{"models":[{"id":"fixture"}]}`},
		{false, `{"model":"fixture","ok":true}`},
	} {
		result, err := requestModelDraft(t.Context(), draft, tc.discover)
		if err != nil {
			t.Fatal(err)
		}
		actual, err := json.Marshal(result)
		if err != nil || string(actual) != tc.want {
			t.Fatalf("got %s, want %s (%v)", actual, tc.want, err)
		}
	}
}

func TestDSHRulesFollowSettings(t *testing.T) {
	const required, enabled, off = "d1bd985c-9f71-4b85-937a-c41b1b104d38", "d2bd985c-9f71-4b85-937a-c41b1b104d38", "d3bd985c-9f71-4b85-937a-c41b1b104d38"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/rules" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"rules":[{"id":"` + required + `","content":"Required rule","required":true},` +
			`{"id":"` + enabled + `","content":" Enabled rule "},{"id":"` + off + `","content":"Off rule"}],"total_count":3}`))
	}))
	defer server.Close()
	live := &acpLive{client: &client{server: server.URL, http: server.Client()}, auth: authorization{Access: "account"},
		connectorPrefs: connectorPreferences{Enabled: map[string]bool{"rule/server/" + enabled: true, "rule/server/" + off: false}}}
	ctx, err := (&relaySessionHost{live: live}).bindDSHRules(t.Context(), acpSession{})
	if got, _ := ctx.Value(dshRuleInstructionsKey{}).(string); err != nil || got != "Required rule\n\nEnabled rule" {
		t.Fatalf("rules = %q (%v)", got, err)
	}
	// A conversation's own choice overrides the default from Settings.
	chat := acpSession{ID: "chat", ConnectorPreferences: &connectorPreferences{Enabled: map[string]bool{"rule/server/" + enabled: false, "rule/server/" + off: true}}}
	ctx, err = (&relaySessionHost{live: live}).bindDSHRules(t.Context(), chat)
	if got, _ := ctx.Value(dshRuleInstructionsKey{}).(string); err != nil || got != "Required rule\n\nOff rule" {
		t.Fatalf("conversation rules = %q (%v)", got, err)
	}
	missing := httptest.NewServer(http.NotFoundHandler())
	defer missing.Close()
	live.client = &client{server: missing.URL, http: missing.Client()}
	if ctx, err := (&relaySessionHost{live: live}).bindDSHRules(t.Context(), acpSession{}); err != nil || ctx.Value(dshRuleInstructionsKey{}) != nil {
		t.Fatalf("backend without rules must not block turns: %v", err)
	}
}
