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
)

// A platform expert is offered as a subagent preset: the lead delegates to it
// by agent_preset and the child runs with the expert's role prompt.
func TestDSHTeamExpertPresetPatchedSDK(t *testing.T) {
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Skip("patched SDK build not configured")
	}
	const id = "11111111-1111-4111-8111-111111111111"
	const model = "33333333-3333-4333-8333-333333333333"
	const expertID = "abcdef12-9999-4999-8999-999999999999"
	preset := dshTeamPresetID(expertID)
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	offered := make(chan string, 1)
	child := make(chan bool, 8)
	client, _ := newBackendClient("https://gateway.test", false)
	client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch {
		case r.URL.Path == "/api/v1/experts" && r.Method == http.MethodGet:
			fmt.Fprintf(w, `{"total_count":1,"experts":[{"id":%q,"name":"财务专家","description":"审核报销","available":true}]}`, expertID)
		case r.URL.Path == "/api/v1/experts/"+expertID+"/manifest":
			fmt.Fprintf(w, `{"available":true,"expert_id":%q,"prompt":"EXPERT-ROLE-MARKER: 你是财务专家 {{not-a-variable}}"}`, expertID)
		case r.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, model)
		case r.URL.Path == "/api/v1/api-keys" && r.Method == http.MethodPost:
			fmt.Fprint(w, `{"id":"44444444-4444-4444-8444-444444444444","api_key":"fixture"}`)
		case r.URL.Path == "/v1/chat/completions":
			var request any
			_ = json.NewDecoder(r.Body).Decode(&request)
			wire, _ := json.Marshal(request)
			body := string(wire)
			n := calls.Add(1)
			delta, finish := map[string]any{"role": "assistant", "content": "done"}, "stop"
			switch {
			case strings.Contains(body, "EXPERT-ROLE-MARKER"):
				child <- !strings.Contains(body, "You are a coding agent")
			case n == 1:
				offered <- body
				args, _ := json.Marshal(map[string]string{"description": "审核报销", "prompt": "Check the expense report", "agent_preset": preset})
				delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "call-1", "type": "function", "function": map[string]string{"name": "subagent", "arguments": string(args)}}}}
				finish = "tool_calls"
			}
			w.Header().Set("Content-Type", "text/event-stream")
			emit := func(v any) { b, _ := json.Marshal(v); fmt.Fprintf(w, "data: %s\n\n", b) }
			emit(map[string]any{"id": "chat", "object": "chat.completion.chunk", "created": 1, "model": model, "choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": nil}}})
			emit(map[string]any{"id": "chat", "object": "chat.completion.chunk", "created": 1, "model": model, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": finish}}, "usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 1, "total_tokens": 4}})
			fmt.Fprint(w, "data: [DONE]\n\n")
		default:
			fmt.Fprint(w, `{}`)
		}
		return w.Result(), nil
	})
	selected := selectedModelConfig{ID: model, Protocol: "openai_chat_completions", ContextWindowTokens: 100000, MaxOutputTokens: 4096}
	gateway := &dshGateway{client: client, authorization: func(context.Context) (authorization, error) { return authorization{Access: "test"}, nil }}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, Home: home, Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(root, "journal"), gateway, func(context.Context, acpSession) (selectedModelConfig, error) { return selected, nil })
	if err != nil {
		t.Fatal(err)
	}
	cfg.LeasePool = &dshLeasePool{}
	defer cfg.LeasePool.close(context.Background())
	session := acpSession{ID: id, DSHSessionID: id, Engine: "dsh", CWD: root, Model: model}
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	turn := acpTurn{Operation: "22222222-2222-4222-8222-000000000001", Prompt: "Have the finance expert check my expenses"}
	if err := runDSHTurn(ctx, cfg, session, &turn, false, func(map[string]any) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if body := <-offered; !strings.Contains(body, preset) {
		t.Fatal("expert preset not offered in the subagent tool")
	} else if !strings.Contains(body, "You are a coding agent") || strings.Contains(body, "EXPERT-ROLE-MARKER") {
		t.Fatal("the lead's own prompt changed")
	}
	select {
	case replaced := <-child:
		if !replaced {
			t.Fatal("expert child still carries the default persona")
		}
	case <-time.After(30 * time.Second):
		t.Fatalf("expert child never ran; model calls %d; runs %+v", calls.Load(), turn.ToolRuns)
	}
}
