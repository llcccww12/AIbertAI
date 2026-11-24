package main

import (
	"context"
	"encoding/json"
	"fmt"

	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"sync/atomic"
	"testing"
	"time"
)

func TestDSHNativeEscalationApprovalIsRejected(t *testing.T) {
	if os.Getenv("DSH_AUDIT_REJECTED_APPROVAL") != "1" {
		t.Skip("historical approval rejection audit")
	}
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Skip("patched SDK build not configured")
	}
	const session = "11111111-1111-4111-8111-111111111111"
	const modelID = "22222222-2222-4222-8222-222222222222"
	const keyID = "33333333-3333-4333-8333-333333333333"
	root := t.TempDir()
	c, err := newBackendClient("https://gateway.test", false)
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch {
		case r.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, modelID)
		case r.URL.Path == "/api/v1/api-keys" && r.Method == "POST":
			fmt.Fprintf(w, `{"id":%q,"api_key":"fixture"}`, keyID)
		case r.URL.Path == "/api/v1/api-keys/"+keyID && r.Method == "DELETE":
		case r.URL.Path == "/v1/chat/completions":
			n := calls.Add(1)
			w.Header().Set("Content-Type", "text/event-stream")
			name, args := "", ""
			finish := "tool_calls"
			switch n {
			case 1:
				name = "bash"
				args = `{"description":"Run harmless fixture","command":"printf synthetic-approved","sandbox_permissions":"danger-full-access","justification":"Synthetic approval fixture"}`
			default:
				finish = "stop"
			}

			var delta map[string]any
			if name != "" {
				delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("call-%d", n), "type": "function", "function": map[string]string{"name": name, "arguments": args}}}}
			} else {
				delta = map[string]any{"role": "assistant", "content": "Background fixture stopped."}
			}
			emit := func(v any) { b, _ := json.Marshal(v); fmt.Fprintf(w, "data: %s\n\n", b) }
			emit(map[string]any{"id": "chat-1", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": nil}}})
			emit(map[string]any{"id": "chat-1", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": finish}}, "usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 1, "total_tokens": 4}})
			fmt.Fprint(w, "data: [DONE]\n\n")
		default:
			w.WriteHeader(404)
		}
		return w.Result(), nil
	})
	model := selectedModelConfig{ID: modelID, Protocol: "openai_chat_completions", ContextWindowTokens: 100000, MaxOutputTokens: 4096}
	gateway := &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, Home: filepath.Join(root, "home"), Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(root, "journal"), gateway, func(context.Context, acpSession) (selectedModelConfig, error) { return model, nil })
	if err != nil {
		t.Fatal(err)
	}
	row := acpSession{ID: session, DSHSessionID: session, Engine: "dsh", Model: modelID, CWD: root}
	turn := acpTurn{Operation: "44444444-4444-4444-8444-444444444444", Prompt: "Run the synthetic background job and stop it."}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if err = runDSHTurn(ctx, cfg, row, &turn, false, nil); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 || len(turn.ToolRuns) != 1 || turn.Permissions.Asked != 1 || turn.Permissions.Denied != 1 || turn.ToolRuns[0].Report.Failed != 1 {
		t.Fatalf("unexpected approval behavior: calls=%d permissions=%+v tools=%+v", calls.Load(), turn.Permissions, turn.ToolRuns)
	}
	t.Logf("SDK requested approval and platform rejected it: %s", turn.ToolRuns[0].Output)
}
