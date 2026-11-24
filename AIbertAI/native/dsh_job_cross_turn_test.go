package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDSHBackgroundJobSurvivesNextTurn(t *testing.T) {
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
	var jobID string
	idPattern := regexp.MustCompile(`bash-[a-f0-9]{16}`)
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch {
		case r.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, modelID)
		case r.URL.Path == "/api/v1/api-keys" && r.Method == "POST":
			fmt.Fprintf(w, `{"id":%q,"api_key":"fixture"}`, keyID)
		case r.URL.Path == "/api/v1/api-keys/"+keyID && r.Method == "DELETE":
		case r.URL.Path == "/v1/chat/completions":
			_, _ = io.ReadAll(r.Body)
			n := calls.Add(1)
			wantKey := "Bearer fixture"
			if r.Header.Get("Authorization") != wantKey {
				t.Errorf("model call %d used stale authority", n)
			}
			w.Header().Set("Content-Type", "text/event-stream")
			name, args := "", ""
			finish := "tool_calls"
			switch n {
			case 1:
				name = "bash"
				args = `{"description":"Run isolated background fixture","command":"printf 'synthetic-job-output\\n'; sleep 30","run_in_background":true}`
			case 5:
				name = "job_kill"
				encoded, _ := json.Marshal(map[string]any{"job_id": jobID, "reason": "fixture complete"})
				args = string(encoded)
			case 3, 7:
				name = "job_output"
				encoded, _ := json.Marshal(map[string]any{"job_id": jobID, "wait": true, "timeout_ms": 1000})
				args = string(encoded)
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
	model := selectedModelConfig{ID: modelID, Protocol: "openai_chat_completions", ContextWindowTokens: 100000, MaxOutputTokens: 4096, SupportsVision: true}
	gateway := &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, Home: filepath.Join(root, "home"), Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(root, "journal"), gateway, func(context.Context, acpSession) (selectedModelConfig, error) { return model, nil })
	if err != nil {
		t.Fatal(err)
	}
	cfg.LeasePool = &dshLeasePool{}
	t.Cleanup(func() { _ = cfg.LeasePool.close(context.Background()) })
	row := acpSession{ID: session, DSHSessionID: session, Engine: "dsh", Model: modelID, CWD: root}
	turn := acpTurn{Operation: "44444444-4444-4444-8444-444444444444", Prompt: "Run the synthetic background job and stop it."}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if err = runDSHTurn(ctx, cfg, row, &turn, false, nil); err != nil {
		t.Fatal(err)
	}
	if len(turn.ToolRuns) != 1 {
		t.Fatalf("missing job creation: %+v", turn.ToolRuns)
	}
	jobID = idPattern.FindString(turn.ToolRuns[0].Output)
	if jobID == "" {
		t.Fatalf("missing job id: %+v", turn.ToolRuns[0])
	}
	turn = acpTurn{Operation: "55555555-5555-4555-8555-555555555555", Prompt: "Read the background job output."}
	if err = runDSHTurn(ctx, cfg, row, &turn, true, nil); err != nil {
		t.Fatal(err)
	}
	if len(turn.ToolRuns) != 1 || turn.ToolRuns[0].Report.Failed != 0 || !strings.Contains(turn.ToolRuns[0].Output, "synthetic-job-output") {
		t.Fatalf("background job unavailable in next turn: %+v", turn.ToolRuns)
	}
	turn = acpTurn{Operation: "66666666-6666-4666-8666-666666666666", Prompt: "Cancel the background job."}
	if err = runDSHTurn(ctx, cfg, row, &turn, true, nil); err != nil {
		t.Fatal(err)
	}
	if len(turn.ToolRuns) != 1 || turn.ToolRuns[0].Report.Failed != 0 || !strings.Contains(turn.ToolRuns[0].Output, "cancellation") {
		t.Fatalf("background job cancellation failed: %+v", turn.ToolRuns)
	}
}

func TestDSHLeaseConcurrentSessions(t *testing.T) {
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Skip("patched SDK build not configured")
	}
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
			body, _ := io.ReadAll(r.Body)
			answer := "unexpected"
			if strings.Contains(string(body), "only-session-one") {
				answer = "answer-one"
			}
			if strings.Contains(string(body), "only-session-two") {
				if answer != "unexpected" {
					answer = "mixed-session"
				} else {
					answer = "answer-two"
				}
			}
			calls.Add(1)
			w.Header().Set("Content-Type", "text/event-stream")
			emit := func(v any) { b, _ := json.Marshal(v); fmt.Fprintf(w, "data: %s\n\n", b) }
			emit(map[string]any{"id": "fixture", "object": "chat.completion.chunk", "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{"role": "assistant", "content": answer}, "finish_reason": nil}}})
			emit(map[string]any{"id": "fixture", "object": "chat.completion.chunk", "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": "stop"}}, "usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 1, "total_tokens": 4}})
			fmt.Fprint(w, "data: [DONE]\n\n")
		default:
			w.WriteHeader(404)
		}
		return w.Result(), nil
	})
	model := selectedModelConfig{ID: modelID, Protocol: "openai_chat_completions", ContextWindowTokens: 100000, MaxOutputTokens: 4096}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, Home: filepath.Join(root, "home"), Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(root, "journal"), &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}, func(context.Context, acpSession) (selectedModelConfig, error) { return model, nil })
	if err != nil {
		t.Fatal(err)
	}
	cfg.LeasePool = &dshLeasePool{}
	t.Cleanup(func() { _ = cfg.LeasePool.close(context.Background()) })
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	defer cancel()
	results := make(chan error, 2)
	for _, fixture := range []struct{ id, op, prompt, answer string }{
		{"11111111-1111-4111-8111-111111111111", "44444444-4444-4444-8444-444444444444", "only-session-one", "answer-one"},
		{"77777777-7777-4777-8777-777777777777", "55555555-5555-4555-8555-555555555555", "only-session-two", "answer-two"},
	} {
		go func() {
			row := acpSession{ID: fixture.id, DSHSessionID: fixture.id, Engine: "dsh", Model: modelID, CWD: root}
			turn := acpTurn{Operation: fixture.op, Prompt: fixture.prompt}
			if err := runDSHTurn(ctx, cfg, row, &turn, false, nil); err != nil {
				results <- err
				return
			}
			if turn.Answer != fixture.answer {
				results <- fmt.Errorf("session %s received %q", fixture.id, turn.Answer)
				return
			}
			results <- nil
		}()
	}
	for range 2 {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("unexpected model call count: %d", calls.Load())
	}
}
