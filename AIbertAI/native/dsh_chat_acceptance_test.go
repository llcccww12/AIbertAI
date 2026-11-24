package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Opt-in: the entry must be an isolated build of the pinned SDK with the
// platform invocation patch. The enterprise backend is entirely local.
func TestDSHChatAcceptance(t *testing.T) {
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Skip("patched SDK build not configured")
	}
	const session = "11111111-1111-4111-8111-111111111111"
	const operation = "22222222-2222-4222-8222-222222222222"
	const modelID = "33333333-3333-4333-8333-333333333333"
	const keyID = "44444444-4444-4444-8444-444444444444"
	for _, tc := range []struct{ protocol, disruption string }{
		{"openai_chat_completions", "crash"}, {"openai_chat_completions", "disconnect"},
		{"openai_responses", "crash"}, {"openai_responses", "disconnect"},
	} {
		protocol, disruption := tc.protocol, tc.disruption
		t.Run(protocol+"/"+disruption, func(t *testing.T) {
			directory := t.TempDir()
			if err := os.WriteFile(filepath.Join(directory, "input.txt"), []byte("unique acceptance content\n"), 0600); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(directory)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			var calls, reports atomic.Int32
			modelBlocked := make(chan struct{})
			modelRelease := make(chan struct{})
			processes := make(chan *exec.Cmd, 2)
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == "/api/v1/settings":
					fmt.Fprint(w, `{"settings":{"session_reporting":{"enabled":true,"level":"full"}}}`)
				case r.URL.Path == "/api/v1/models":
					fmt.Fprintf(w, `{"models":[{"id":%q,"kind":"text","protocol":%q}],"total_count":1}`, modelID, protocol)
				case r.URL.Path == "/api/v1/api-keys" && r.Method == "POST":
					fmt.Fprintf(w, `{"id":%q,"api_key":"mock-key"}`, keyID)
				case r.URL.Path == "/api/v1/api-keys/"+keyID && r.Method == "DELETE":
				case r.URL.Path == "/api/v1/sessions/"+session:
					w.WriteHeader(http.StatusNoContent)
				case r.URL.Path == "/api/v1/sessions/"+session+"/turns":
					n := reports.Add(1)
					var report struct {
						Turns []struct {
							Model string `json:"model_id"`
							Usage struct {
								Input  int `json:"input_tokens"`
								Output int `json:"output_tokens"`
							} `json:"usage"`
							Tools []reportedTool `json:"tools"`
						} `json:"turns"`
					}
					if err := json.NewDecoder(r.Body).Decode(&report); err != nil || len(report.Turns) != 1 {
						t.Errorf("invalid turn report: %+v %v", report, err)
					} else if n == 1 && (report.Turns[0].Model != modelID || report.Turns[0].Usage.Input != 93 || report.Turns[0].Usage.Output != 31 || len(report.Turns[0].Tools) != 30) {
						t.Errorf("incorrect first turn report: %+v", report.Turns[0])
					}
					fmt.Fprintf(w, `{"acked_turn":%d}`, n)
				case r.URL.Path == "/v1/chat/completions" || r.URL.Path == "/v1/responses":
					var request map[string]any
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					n := calls.Add(1)
					if n == 32 {
						close(modelBlocked)
						select {
						case <-r.Context().Done():
							return
						case <-modelRelease:
						case <-time.After(20 * time.Second):
							t.Error("model release timed out")
							return
						}
					}
					encoded, _ := json.Marshal(request)
					if n > 1 && !strings.Contains(string(encoded), "unique acceptance content") {
						t.Errorf("model request %d lacks prior read result", n)
					}
					w.Header().Set("Content-Type", "text/event-stream")
					w.Header().Set("X-Billing-Transaction-ID", fmt.Sprintf("tx-%d", n))
					emit := func(v any) { b, _ := json.Marshal(v); fmt.Fprintf(w, "data: %s\n\n", b) }
					answer := "acceptance complete"
					if n == 32 {
						answer = "continued after disconnect"
					}
					if protocol == "openai_chat_completions" {
						delta := map[string]any{"role": "assistant", "content": answer}
						finish := "stop"
						if n <= 30 {
							delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("call-%d", n), "type": "function", "function": map[string]string{"name": "read", "arguments": `{"file_path":"input.txt"}`}}}}
							finish = "tool_calls"
						}
						emit(map[string]any{"id": "chat-1", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": nil}}})
						emit(map[string]any{"id": "chat-1", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": finish}}, "usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 1, "total_tokens": 4}})
						fmt.Fprint(w, "data: [DONE]\n\n")
					} else {
						item := map[string]any{"type": "message", "id": "msg1", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": answer, "annotations": []any{}}}}
						if n <= 30 {
							item = map[string]any{"type": "function_call", "id": fmt.Sprintf("item-%d", n), "call_id": fmt.Sprintf("call-%d", n), "name": "read", "arguments": `{"file_path":"input.txt"}`, "status": "completed"}
						}
						emit(map[string]any{"type": "response.created", "response": map[string]any{"id": "resp1"}})
						emit(map[string]any{"type": "response.output_item.added", "output_index": 0, "item": item})
						emit(map[string]any{"type": "response.output_item.done", "output_index": 0, "item": item})
						emit(map[string]any{"type": "response.completed", "response": map[string]any{"id": "resp1", "status": "completed", "output": []any{item}, "usage": map[string]int{"input_tokens": 3, "output_tokens": 1, "total_tokens": 4}}})
					}
				default:
					t.Errorf("unexpected backend request: %s %s", r.Method, r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer backend.Close()
			client := &client{backend.URL, backend.Client()}
			gateway := &dshGateway{client: client, authorization: func(context.Context) (authorization, error) { return authorization{Access: "account"}, nil }}
			selected := selectedModelConfig{ID: modelID, Protocol: protocol, DisplayName: "Test", ContextWindowTokens: 100000, MaxOutputTokens: 4096}
			cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, Home: filepath.Join(directory, "home"), Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(directory, "journal"), gateway, func(context.Context, acpSession) (selectedModelConfig, error) { return selected, nil })
			if err != nil {
				t.Fatal(err)
			}
			prepare := cfg.Prepare
			cfg.Prepare = func(ctx context.Context, s acpSession) (*exec.Cmd, string, string, *dshLocalGateway, func(), error) {
				cmd, provider, model, local, cleanup, err := prepare(ctx, s)
				if err == nil {
					processes <- cmd
				}
				return cmd, provider, model, local, cleanup, err
			}
			now := time.Now().UTC()
			host := &relaySessionHost{root: root, terminalLifetime: t.Context(), local: &desktopRuntimeAccess{protocol: "monkeyai.token.test"}, live: &acpLive{client: client, auth: authorization{Access: "account"}, store: &desktopSessionStore{Root: root}, cwd: directory, dsh: cfg, sessions: map[string]acpSession{session: {ID: session, DSHSessionID: session, Engine: "dsh", CWD: directory, Model: modelID, State: "idle", Created: now, Updated: now}}}}
			desktop := httptest.NewServer(host)
			defer desktop.Close()
			host.local.address = strings.TrimPrefix(desktop.URL, "http://")
			ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
			defer cancel()
			connect := func() *websocket.Conn {
				ws, _, err := websocket.Dial(ctx, desktop.URL+"/acp", &websocket.DialOptions{Subprotocols: []string{"monkeyai.v1", "monkeyai.token.test"}})
				if err != nil {
					t.Fatal(err)
				}
				return ws
			}
			ws := connect()
			defer ws.CloseNow()
			send := func(ws *websocket.Conn, id int, method string, params any) {
				t.Helper()
				if err := wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
					t.Fatal(err)
				}
			}
			receive := func(ws *websocket.Conn, id int) []map[string]any {
				t.Helper()
				var messages []map[string]any
				for {
					var msg map[string]any
					if err := wsjson.Read(ctx, ws, &msg); err != nil {
						t.Fatal(err)
					}
					messages = append(messages, msg)
					if msg["id"] == float64(id) {
						if msg["error"] != nil {
							t.Fatalf("request %d: %v", id, msg)
						}
						return messages
					}
				}
			}
			send(ws, 1, "initialize", map[string]int{"protocolVersion": 1})
			receive(ws, 1)
			params := map[string]any{"sessionId": session, "_meta": map[string]string{"operationId": operation}, "prompt": []map[string]string{{"type": "text", "text": "Read input"}}}
			send(ws, 2, "session/prompt", params)
			receive(ws, 2)
			row, err := host.live.load(session)
			if err != nil {
				t.Fatal(err)
			}
			if len(row.Turns) != 1 || len(row.Turns[0].ToolRuns) != 30 || row.Turns[0].Answer != "acceptance complete" || !row.Turns[0].Acked {
				t.Fatalf("turn not retained: %+v", row.Turns)
			}
			for i, run := range row.Turns[0].ToolRuns {
				if run.Pending || run.Report.Failed != 0 || !strings.Contains(run.Output, "unique acceptance content") {
					t.Fatalf("tool %d failed: %+v", i, run)
				}
			}
			if row.Turns[0].Usage.Input == nil || row.Turns[0].Usage.Output == nil || *row.Turns[0].Usage.Input != 93 || *row.Turns[0].Usage.Output != 31 {
				t.Fatalf("unexpected usage: %+v", row.Turns[0].Usage)
			}
			send(ws, 3, "session/prompt", params)
			receive(ws, 3)
			if calls.Load() != 31 || reports.Load() != 1 {
				t.Fatalf("retry replayed model or report: calls=%d reports=%d", calls.Load(), reports.Load())
			}
			<-processes // the completed first turn
			ws.CloseNow()
			reopened := connect()
			defer reopened.CloseNow()
			send(reopened, 4, "initialize", map[string]int{"protocolVersion": 1})
			receive(reopened, 4)
			send(reopened, 5, "session/load", map[string]string{"sessionId": session, "cwd": directory})
			receive(reopened, 5)
			send(reopened, 6, "_monkeyai/session/messages", map[string]string{"sessionId": session})
			history := receive(reopened, 6)
			result, ok := history[len(history)-1]["result"].(map[string]any)
			if !ok {
				t.Fatalf("missing history result: %v", history)
			}
			toolCalls, ok := result["toolCalls"].(map[string]any)
			data, _ := json.Marshal(result)
			if !ok || len(toolCalls) != 30 || !strings.Contains(string(data), "acceptance complete") {
				t.Fatalf("reconnected history missing response or 30 tools: %s", data)
			}
			crashParams := map[string]any{"sessionId": session, "_meta": map[string]string{"operationId": "55555555-5555-4555-8555-555555555555"}, "prompt": []map[string]string{{"type": "text", "text": "crash during response"}}}
			send(reopened, 7, "session/prompt", crashParams)
			var child *exec.Cmd
			select {
			case child = <-processes:
			case <-ctx.Done():
				t.Fatal("SDK did not start")
			}
			select {
			case <-modelBlocked:
			case <-ctx.Done():
				t.Fatal("gateway did not receive second request")
			}
			if disruption == "disconnect" {
				reopened.CloseNow()
				// Leave the request in flight long enough for the viewer to be removed.
				time.Sleep(200 * time.Millisecond)
				close(modelRelease)
				deadline := time.Now().Add(20 * time.Second)
				for {
					row, err = host.live.load(session)
					if err == nil && len(row.Turns) == 2 && row.Turns[1].Acked {
						break
					}
					if err == nil && len(row.Turns) == 2 && row.Turns[1].Failed {
						t.Fatalf("disconnected turn failed: %+v", row.Turns[1])
					}
					if time.Now().After(deadline) {
						t.Fatalf("disconnected turn did not settle: state=%s turns=%d error=%v", row.State, len(row.Turns), err)
					}
					time.Sleep(20 * time.Millisecond)
				}
				reopened = connect()
				defer reopened.CloseNow()
				send(reopened, 20, "initialize", map[string]int{"protocolVersion": 1})
				receive(reopened, 20)
				send(reopened, 21, "_monkeyai/session/messages", map[string]string{"sessionId": session})
				history := receive(reopened, 21)
				data, _ := json.Marshal(history)
				historyResult, ok := history[len(history)-1]["result"].(map[string]any)
				historyTools, toolsOK := historyResult["toolCalls"].(map[string]any)
				if !ok || !toolsOK || len(historyTools) != 30 || !strings.Contains(string(data), "acceptance complete") || !strings.Contains(string(data), "continued after disconnect") {
					t.Fatalf("history lost after disconnected turn: %s", data)
				}
				if row.Turns[1].Failed || row.Turns[1].Answer != "continued after disconnect" {
					t.Fatalf("disconnected turn did not complete: %+v", row.Turns[1])
				}
				send(reopened, 22, "session/prompt", crashParams)
				receive(reopened, 22)
				if calls.Load() != 32 || reports.Load() != 2 {
					t.Fatalf("disconnected operation repeated: calls=%d reports=%d", calls.Load(), reports.Load())
				}
				return
			} else {
				if err := child.Process.Kill(); err != nil {
					t.Fatal(err)
				}
				for {
					var msg map[string]any
					if err := wsjson.Read(ctx, reopened, &msg); err != nil {
						t.Fatal(err)
					}
					if msg["id"] == float64(7) {
						if msg["error"] == nil {
							t.Fatalf("crashed turn succeeded: %v", msg)
						}
						break
					}
				}
			}
			row, err = host.live.load(session)
			if err != nil || len(row.Turns) != 2 || !row.Turns[1].Failed {
				t.Fatalf("crashed turn not retained as failed: %+v %v", row.Turns, err)
			}
			if calls.Load() != 32 {
				t.Fatalf("crashed request repeated: %d", calls.Load())
			}
			// The same failed operation and a new operation must both fail closed.
			for i, p := range []map[string]any{crashParams, {"sessionId": session, "_meta": map[string]string{"operationId": "66666666-6666-4666-8666-666666666666"}, "prompt": []map[string]string{{"type": "text", "text": "continue"}}}} {
				id := 8 + i
				send(reopened, id, "session/prompt", p)
				for {
					var msg map[string]any
					if err := wsjson.Read(ctx, reopened, &msg); err != nil {
						t.Fatal(err)
					}
					if msg["id"] == float64(id) {
						if msg["error"] == nil {
							t.Fatalf("unsafe retry accepted: %v", msg)
						}
						break
					}
				}
			}
			if calls.Load() != 32 {
				t.Fatalf("failure retry reached gateway: %d", calls.Load())
			}
		})
	}
}
