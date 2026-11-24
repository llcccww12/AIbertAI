package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestDSHPluginApprovalBoundary(t *testing.T) {
	for _, args := range []map[string]any{
		{"action": "install_bundle", "target": "file:/other"},
		{"action": "remove_bundle", "target": "example"},
		{"action": "set_plugin", "target": "include:hmr", "enabled": true},
		{"action": "set_plugin", "target": "include:tool-todo", "enabled": false, "profile": "/other"},
		{"action": "list_plugins", "cwd": "/other"},
		{"action": "list_bundles", "limit": json.Number("101")},
	} {
		if title := dshPluginApprovalTitle(args); title != "" {
			t.Fatalf("unexpected approval %q for %v", title, args)
		}
	}
	h := &relaySessionHost{}
	session := acpSession{Engine: "dsh", ID: "11111111-1111-4111-8111-111111111111"}
	session.DSHSessionID = session.ID
	if h.dshPluginManagerBound(t.Context(), session) {
		t.Fatal("allowed nonlocal session")
	}
	h.profile = t.TempDir()
	h.local = &desktopRuntimeAccess{userID: session.ID, serverID: session.ID, reportMachine: session.ID}
	h.live = &acpLive{dsh: &dshRuntimeConfig{LeasePool: &dshLeasePool{}, SharedPluginHome: t.TempDir()}}
	if h.dshPluginManagerBound(t.Context(), session) {
		t.Fatal("allowed another profile")
	}
	h.live.dsh.SharedPluginHome = filepath.Join(h.profile, "source-runtime", session.ID, session.ID, "dsh", "home")
	session.ExpertID = "33333333-3333-4333-8333-333333333333"
	h.live.dsh.ResolveDisabledPlugins = func(context.Context, string) ([]string, error) { return nil, nil }
	if h.dshPluginManagerBound(t.Context(), session) {
		t.Fatal("allowed unbound expert")
	}
	ctx := context.WithValue(t.Context(), dshDisabledPluginsKey{}, []string{"tool-plugin-manager"})
	if h.dshPluginManagerBound(ctx, session) {
		t.Fatal("allowed disabled tool")
	}
	ctx = context.WithValue(t.Context(), dshDisabledPluginsKey{}, []string{})
	if !h.dshPluginManagerBound(ctx, session) {
		t.Fatal("denied bound expert")
	}
	h.live.dsh.ResolveDisabledPlugins = func(context.Context, string) ([]string, error) { return []string{"tool-plugin-manager"}, nil }
	if h.dshPluginManagerBound(ctx, session) {
		t.Fatal("allowed revoked tool")
	}
}

func TestDSHPluginManagerApprovalPatchedSDK(t *testing.T) {
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Skip("patched SDK build not configured")
	}
	const serverID, machine, user = "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222", "33333333-3333-4333-8333-333333333333"
	const model, key, operation = "44444444-4444-4444-8444-444444444444", "55555555-5555-4555-8555-555555555555", "66666666-6666-4666-8666-666666666666"
	profile := t.TempDir()
	home := filepath.Join(profile, "source-runtime", serverID, machine, "dsh", "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	// Prior profile preferences and the UI's normally trusted mode must never grant a manager call.
	if err := os.WriteFile(filepath.Join(home, "cordis.patch.yml"), []byte(`- id: sandbox-policy
  config:
    mode: danger-full-access
- id: permission
  config:
    defaultPreset: managed
    presets:
      managed: { sandbox: danger-full-access, approval: ask }
`), 0600); err != nil {
		t.Fatal(err)
	}
	actions := []string{`{"action":"list_plugins","limit":1}`, `{"action":"set_plugin","target":"include:tool-todo","enabled":false}`, `{"action":"set_plugin","target":"include:tool-todo","enabled":true}`, `{"action":"set_plugin","target":"include:tool-todo","enabled":true}`}
	var calls atomic.Int32
	c, err := newBackendClient("https://gateway.test", false)
	if err != nil {
		t.Fatal(err)
	}
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch r.URL.Path {
		case "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, model)
		case "/api/v1/api-keys":
			fmt.Fprintf(w, `{"id":%q,"api_key":"fixture"}`, key)
		case "/api/v1/api-keys/" + key:
		case "/v1/chat/completions":
			n := int(calls.Add(1))
			var body struct {
				Tools []struct {
					Function struct {
						Name       string
						Parameters struct {
							Properties struct{ Action struct{ Enum []string } }
						}
					}
				}
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			found := false
			for _, tool := range body.Tools {
				if tool.Function.Name == dshPluginManagerTool {
					found = strings.Join(tool.Function.Parameters.Properties.Action.Enum, ",") == "list_plugins,list_bundles,set_plugin"
				}
			}
			if !found {
				t.Error("model did not receive the restricted manager action schema")
			}
			delta, finish := map[string]any{"role": "assistant", "content": "done"}, "stop"
			if n <= len(actions) {
				delta, finish = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("plugin-%d", n), "type": "function", "function": map[string]string{"name": dshPluginManagerTool, "arguments": actions[n-1]}}}}, "tool_calls"
			}
			w.Header().Set("Content-Type", "text/event-stream")
			for i, data := range []map[string]any{delta, {}} {
				var reason any
				message := map[string]any{"id": "chat", "object": "chat.completion.chunk", "created": 1, "model": model}
				if i == 1 {
					reason = finish
					message["usage"] = map[string]int{"prompt_tokens": 10, "completion_tokens": 3, "total_tokens": 13}
				}
				message["choices"] = []any{map[string]any{"index": 0, "delta": data, "finish_reason": reason}}
				encoded, _ := json.Marshal(message)
				fmt.Fprintf(w, "data: %s\n\n", encoded)
			}
			fmt.Fprint(w, "data: [DONE]\n\n")
		default:
			w.WriteHeader(404)
		}
		return w.Result(), nil
	})
	selected := selectedModelConfig{ID: model, Protocol: "openai_chat_completions", ContextWindowTokens: 100000, MaxOutputTokens: 4096}
	gw := &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "test"}, nil }}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, Home: home, Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(profile, "journal"), gw, func(context.Context, acpSession) (selectedModelConfig, error) { return selected, nil })
	if err != nil {
		t.Fatal(err)
	}
	cfg.LeasePool = &dshLeasePool{}
	defer cfg.LeasePool.close(context.Background())
	store, err := os.OpenRoot(profile)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	workspaces, err := openDesktopWorkspaces(store, filepath.Join(profile, "workspaces"), "")
	if err != nil {
		t.Fatal(err)
	}
	defer workspaces.root.Close()
	live := &acpLive{dsh: cfg, store: &desktopSessionStore{Root: store}, workspaces: workspaces, operations: map[string]string{}, sessions: map[string]acpSession{}, cwd: profile}
	id, err := live.createInWorkspace(operation, "", "", "", expertSelection{}, nil, "", desktopMediaModels{})
	if err != nil {
		t.Fatal(err)
	}
	row, err := live.load(id)
	if err != nil {
		t.Fatal(err)
	}
	row.Engine, row.DSHSessionID, row.Mode, row.State, row.Model = "dsh", id, "bypassPermissions", "running", model
	turn := acpTurn{Operation: operation, Prompt: "List current plugins, then change the built-in todo plugin.", Started: time.Now().UTC()}
	row.Turns = []acpTurn{turn}
	if err = live.save(row); err != nil {
		t.Fatal(err)
	}
	h := &relaySessionHost{profile: profile, live: live, local: &desktopRuntimeAccess{serverID: serverID, reportMachine: machine, userID: user, origin: "http://workbench.test", protocol: "monkeyai.token.fixture"}, interactions: newDesktopDSHInteractions(), goals: newGoalRuntime()}
	server := httptest.NewServer(h)
	defer server.Close()
	h.local.address = strings.TrimPrefix(server.URL, "http://")
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, server.URL+"/acp", &websocket.DialOptions{Subprotocols: []string{"monkeyai.v1", h.local.protocol}, HTTPHeader: http.Header{"Origin": {h.local.origin}}})
	if err != nil {
		t.Fatal(err)
	}
	defer ws.CloseNow()
	if err = wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]int{"protocolVersion": 1}}); err != nil {
		t.Fatal(err)
	}
	var init map[string]any
	if err = wsjson.Read(ctx, ws, &init); err != nil || init["result"] == nil {
		t.Fatalf("initialize: %v %v", init, err)
	}
	request := &http.Request{Host: h.local.address, Header: http.Header{"Sec-Websocket-Protocol": {"monkeyai.v1, " + h.local.protocol}}}
	ctx = context.WithValue(ctx, acpViewerDoneKey{}, (<-chan struct{})(make(chan struct{})))
	ctx = context.WithValue(ctx, dshSideEffectRootKey{}, store)
	ctx = context.WithValue(ctx, dshNativeApprovalKey{}, h.dshNativeApproval(ctx, nil, request, row))
	done := make(chan error, 1)
	go func() {
		err := runDSHTurn(ctx, cfg, row, &turn, false, func(map[string]any) error {
			live.mu.Lock()
			defer live.mu.Unlock()
			updated := row
			updated.Turns = []acpTurn{turn}
			return live.save(updated)
		})
		done <- err
		if err != nil {
			ws.CloseNow()
		}
	}()
	for i, choice := range []string{"allow", "allow", "deny", "allow"} {
		var frame struct {
			ID, Method string
			Params     struct {
				ToolCall struct {
					Title, Name string
					RawInput    json.RawMessage
				}
			}
		}
		if err = wsjson.Read(ctx, ws, &frame); err != nil {
			select {
			case err = <-done:
				t.Fatal(err)
			default:
				t.Fatalf("permission %d: %v", i, err)
			}
		}
		title := []string{"查看插件", "停用 include:tool-todo", "启用 include:tool-todo", "启用 include:tool-todo"}[i]
		if frame.Method != "session/request_permission" || frame.Params.ToolCall.Name != dshPluginManagerTool || frame.Params.ToolCall.Title != title+" · 当前账号在本机的所有 DSH 会话" {
			t.Fatalf("unexpected request %+v", frame)
		}
		if i == 3 {
			if err := assertTodoSetting(home, true); err != nil {
				t.Fatalf("denied enable changed the saved profile: %v", err)
			}
		}
		if err = wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": frame.ID, "result": map[string]any{"outcome": map[string]string{"outcome": "selected", "optionId": choice}}}); err != nil {
			t.Fatal(err)
		}
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if turn.Permissions.Asked != 4 || turn.Permissions.Allowed != 3 || turn.Permissions.Denied != 1 || len(turn.ToolRuns) != 4 || turn.ToolRuns[2].Report.Failed != 1 || !strings.Contains(turn.ToolRuns[0].Output, "entries") || !strings.Contains(turn.ToolRuns[1].Output, "restart-required") {
		t.Fatalf("approval results %+v, %+v", turn.Permissions, turn.ToolRuns)
	}
	if err := assertTodoSetting(home, false); err != nil {
		t.Fatalf("approved enable did not update the profile: %v", err)
	}
}

func assertTodoSetting(home string, disabled bool) error {
	data, err := os.ReadFile(filepath.Join(home, "profiles", "sdk", "cordis.patch.yml"))
	if err != nil {
		return err
	}
	// Enabling the shipped default removes the redundant override from the user's patch.
	if !disabled && strings.HasSuffix(strings.TrimSpace(string(data)), "[]") {
		return nil
	}
	if !disabled || !strings.Contains(string(data), "tool-todo") || !strings.Contains(string(data), "disabled: true") {
		return fmt.Errorf("unexpected saved profile %s", data)
	}
	return nil
}
