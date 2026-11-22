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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestDSHPatchedSDKWebSocketReconnect(t *testing.T) {
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Skip("patched SDK build not configured")
	}
	const modelID = "22222222-2222-4222-8222-222222222222"
	const keyID = "33333333-3333-4333-8333-333333333333"
	const op = "44444444-4444-4444-8444-444444444444"
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "input.txt"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := newBackendClient("https://gateway.test", false)
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	release := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	var mu sync.Mutex
	var report map[string]any
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch {
		case r.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, modelID)
		case r.URL.Path == "/api/v1/api-keys" && r.Method == "POST":
			fmt.Fprintf(w, `{"id":%q,"api_key":"key"}`, keyID)
		case r.URL.Path == "/api/v1/api-keys/"+keyID && r.Method == "DELETE":
		case strings.HasSuffix(r.URL.Path, "/turns"):
			mu.Lock()
			_ = json.NewDecoder(r.Body).Decode(&report)
			mu.Unlock()
			fmt.Fprint(w, `{"acked_turn":1}`)
		case strings.HasPrefix(r.URL.Path, "/api/v1/sessions/"):
		case r.URL.Path == "/v1/chat/completions":
			n := calls.Add(1)
			if n == 2 {
				<-release
			}
			w.Header().Set("Content-Type", "text/event-stream")
			var delta map[string]any
			finish := "stop"
			if n <= 30 {
				args, _ := json.Marshal(map[string]string{"file_path": "input.txt"})
				delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("call-%d", n), "type": "function", "function": map[string]string{"name": "read", "arguments": string(args)}}}}
				finish = "tool_calls"
			} else {
				delta = map[string]any{"role": "assistant", "content": "finished"}
			}
			emit := func(v any) { data, _ := json.Marshal(v); fmt.Fprintf(w, "data: %s\n\n", data) }
			emit(map[string]any{"id": "chat-1", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": nil}}})
			emit(map[string]any{"id": "chat-1", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": finish}}, "usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 1, "total_tokens": 4}})
			fmt.Fprint(w, "data: [DONE]\n\n")
		default:
			fmt.Fprint(w, `{}`)
		}
		return w.Result(), nil
	})
	selected := selectedModelConfig{ID: modelID, Protocol: "openai_chat_completions", ContextWindowTokens: 100000, MaxOutputTokens: 4096}
	gw := &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "test"}, nil }}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, Home: filepath.Join(root, "home"), Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(root, "journal"), gw, func(context.Context, acpSession) (selectedModelConfig, error) { return selected, nil })
	if err != nil {
		t.Fatal(err)
	}
	live := &acpLive{client: c, auth: authorization{Access: "test"}, dir: root, cwd: root, dsh: cfg, operations: map[string]string{}, sessions: map[string]acpSession{}}
	id, err := live.createInWorkspace("55555555-5555-4555-8555-555555555555", root, "", modelID, expertSelection{}, nil, "", desktopMediaModels{})
	if err != nil {
		t.Fatal(err)
	}

	host := &relaySessionHost{profile: root, live: live, local: &desktopRuntimeAccess{origin: "http://desktop.test", protocol: "monkeyai.token.test", reportMachine: "machine"}, goals: newGoalRuntime(), terminalLifetime: t.Context()}
	server := httptest.NewUnstartedServer(host)
	host.local.address = server.Listener.Addr().String()
	server.Start()
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	dial := func() *websocket.Conn {
		t.Helper()
		ws, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/acp", &websocket.DialOptions{Subprotocols: []string{"monkeyai.v1", "monkeyai.token.test"}, HTTPHeader: http.Header{"Origin": []string{"http://desktop.test"}}})
		if err != nil {
			t.Fatal(err)
		}
		if err = wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]int{"protocolVersion": 1}}); err != nil {
			t.Fatal(err)
		}
		var response map[string]any
		if err = wsjson.Read(ctx, ws, &response); err != nil || response["error"] != nil {
			t.Fatalf("initialize: %v %v", response, err)
		}
		return ws
	}
	ws := dial()
	params := map[string]any{"sessionId": id, "prompt": []map[string]string{{"type": "text", "text": "read input"}}, "_meta": map[string]string{"operationId": op}}
	if err = wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "session/prompt", "params": params}); err != nil {
		t.Fatal(err)
	}
	for {
		var frame map[string]any
		if err = wsjson.Read(ctx, ws, &frame); err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(frame)
		if strings.Contains(string(b), `"sessionUpdate":"tool_call"`) {
			break
		}
	}
	_ = ws.CloseNow()
	ws = dial()
	defer ws.CloseNow()
	releaseOnce.Do(func() { close(release) })
	for {
		var frame map[string]any
		if err = wsjson.Read(ctx, ws, &frame); err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(frame)
		if strings.Contains(string(b), "_monkeyai/session/history_changed") {
			break
		}
	}
	for {
		row, _ := live.load(id)
		if len(row.Turns) > 0 && !row.Turns[0].Ended.IsZero() && row.Turns[0].Acked {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("turn did not finish")
		case <-time.After(10 * time.Millisecond):
		}
	}
	if err = wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "_monkeyai/session/messages", "params": map[string]any{"sessionId": id, "limit": 20}}); err != nil {
		t.Fatal(err)
	}
	for {
		var frame struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
		}
		if err = wsjson.Read(ctx, ws, &frame); err != nil {
			t.Fatal(err)
		}
		if frame.ID != 3 {
			continue
		}
		var page struct {
			ToolCalls map[string]any `json:"toolCalls"`
		}
		if json.Unmarshal(frame.Result, &page) != nil || len(frame.Error) != 0 || len(page.ToolCalls) != 30 || !strings.Contains(string(frame.Result), "finished") || !strings.Contains(string(frame.Result), "hello") {
			t.Fatalf("history: %s %s", frame.Result, frame.Error)
		}
		break
	}
	if err = wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": 4, "method": "session/prompt", "params": params}); err != nil {
		t.Fatal(err)
	}
	for {
		var frame struct {
			ID    int             `json:"id"`
			Error json.RawMessage `json:"error"`
		}
		if err = wsjson.Read(ctx, ws, &frame); err != nil {
			t.Fatal(err)
		}
		if frame.ID == 4 {
			if len(frame.Error) != 0 {
				t.Fatalf("retry: %s", frame.Error)
			}
			break
		}
	}
	if calls.Load() != 31 {
		t.Fatalf("duplicate request: %d", calls.Load())
	}
	mu.Lock()
	if len(report) == 0 {
		t.Error("missing report")
	}
	mu.Unlock()
}
