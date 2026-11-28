package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDSHFourMembersWithSlowModel(t *testing.T) {
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Skip("patched SDK build not configured")
	}
	for _, timeoutWorker := range []bool{false, true} {
		t.Run(fmt.Sprintf("member-timeout=%t", timeoutWorker), func(t *testing.T) {
			testDSHFourMembers(t, node, entry, timeoutWorker)
		})
	}
}

func testDSHFourMembers(t *testing.T, node, entry string, timeoutWorker bool) {
	t.Helper()
	const sessionID = "11111111-1111-4111-8111-111111111111"
	const modelID = "22222222-2222-4222-8222-222222222222"
	const keyID = "33333333-3333-4333-8333-333333333333"
	root := t.TempDir()
	c, err := newBackendClient("https://gateway.test", false)
	if err != nil {
		t.Fatal(err)
	}
	const firstPrompt, nextPrompt = "coordinate-four-fixture-workers", "continue-four-fixture-workers"
	var spawned atomic.Bool
	var waits atomic.Int32
	var workerCalls [4]atomic.Int32
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch r.URL.Path {
		case "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, modelID)
		case "/api/v1/api-keys":
			fmt.Fprintf(w, `{"id":%q,"api_key":"fixture"}`, keyID)
		case "/v1/chat/completions":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			var request struct {
				Messages []struct {
					Role    string
					Content json.RawMessage
				}
			}
			if err := json.Unmarshal(body, &request); err != nil {
				return nil, err
			}
			toolCall := func(index int, id, name, args string) map[string]any {
				return map[string]any{"index": index, "id": id, "type": "function", "function": map[string]string{"name": name, "arguments": args}}
			}
			emit := func(dst io.Writer, delta map[string]any, finish any, usage map[string]int) {
				chunk := map[string]any{"id": "chat", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}}}
				if usage != nil {
					chunk["usage"] = usage
				}
				data, _ := json.Marshal(chunk)
				fmt.Fprintf(dst, "data: %s\n\n", data)
			}
			complete := func(dst io.Writer, delta map[string]any, finish string) {
				emit(dst, delta, nil, nil)
				emit(dst, map[string]any{}, finish, map[string]int{"prompt_tokens": 3, "completion_tokens": 1, "total_tokens": 4})
				fmt.Fprint(dst, "data: [DONE]\n\n")
			}
			w.Header().Set("Content-Type", "text/event-stream")
			for i := range workerCalls {
				marker := fmt.Sprintf("worker-input-%d", i)
				for _, message := range request.Messages {
					if message.Role != "user" || !strings.Contains(string(message.Content), marker) {
						continue
					}
					workerCalls[i].Add(1)
					reader, writer := io.Pipe()
					go func() {
						defer writer.Close()
						if timeoutWorker && i == 0 {
							<-r.Context().Done()
							return
						}
						// Three individually healthy streams together exceed the SDK's idle budget.
						select {
						case <-time.After(850 * time.Millisecond):
							complete(writer, map[string]any{"role": "assistant", "content": fmt.Sprintf("worker-result-%d", i)}, "stop")
						case <-r.Context().Done():
						}
					}()
					return &http.Response{StatusCode: 200, Header: w.Header(), Body: reader}, nil
				}
			}
			if strings.Contains(string(body), nextPrompt) && strings.Contains(string(body), "lead collected the results") {
				complete(w, map[string]any{"role": "assistant", "content": "original session replied"}, "stop")
			} else if spawned.CompareAndSwap(false, true) {
				calls := make([]map[string]any, 0, len(workerCalls))
				for i := range workerCalls {
					args := fmt.Sprintf(`{"name":"reviewer-%d","description":"Review %d","prompt":"worker-input-%d"}`, i, i, i)
					calls = append(calls, toolCall(i, fmt.Sprintf("spawn-%d", i), "subagent", args))
				}
				complete(w, map[string]any{"role": "assistant", "tool_calls": calls}, "tool_calls")
			} else {
				allResults := !timeoutWorker || strings.Contains(string(body), "failed before it finished")
				for i := range workerCalls {
					if !timeoutWorker || i != 0 {
						allResults = allResults && strings.Contains(string(body), fmt.Sprintf("worker-result-%d", i))
					}
				}
				if allResults {
					complete(w, map[string]any{"role": "assistant", "content": "lead collected the results"}, "stop")
				} else {
					calls := []any{toolCall(0, fmt.Sprintf("wait-%d", waits.Add(1)), "wait_agent", `{"timeout_ms":5000}`)}
					complete(w, map[string]any{"role": "assistant", "tool_calls": calls}, "tool_calls")
				}
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
		return w.Result(), nil
	})
	model := selectedModelConfig{ID: modelID, Protocol: "openai_chat_completions", ContextWindowTokens: 100000, MaxOutputTokens: 4096}
	gateway := &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, Home: filepath.Join(root, "home"), Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(root, "journal"), gateway, func(context.Context, acpSession) (selectedModelConfig, error) { return model, nil })
	if err != nil {
		t.Fatal(err)
	}
	prepare := cfg.Prepare
	cfg.Prepare = func(ctx context.Context, session acpSession) (*exec.Cmd, string, string, *dshLocalGateway, func(), error) {
		cmd, provider, model, local, cleanup, err := prepare(ctx, session)
		if err == nil {
			for i, arg := range cmd.Args {
				if arg == "--patch" {
					path := cmd.Args[i+1]
					data, readErr := os.ReadFile(path)
					if readErr != nil {
						cleanup()
						return nil, "", "", nil, nil, readErr
					}
					patch := strings.Replace(string(data), `"apiKeyEnv":"DSH_AIBERTAI_LOCAL_TOKEN"`, `"apiKeyEnv":"DSH_AIBERTAI_LOCAL_TOKEN","streamIdleTimeoutMs":2000`, 1)
					if patch == string(data) {
						cleanup()
						return nil, "", "", nil, nil, fmt.Errorf("missing model route in SDK profile patch")
					}
					if err = os.WriteFile(path, []byte(patch), 0600); err != nil {
						cleanup()
						return nil, "", "", nil, nil, err
					}
					return cmd, provider, model, local, cleanup, nil
				}
			}
			cleanup()
			err = fmt.Errorf("missing SDK profile patch")
		}
		return nil, "", "", nil, nil, err
	}
	cfg.LeasePool = &dshLeasePool{}
	t.Cleanup(func() { _ = cfg.LeasePool.close(context.Background()) })
	session := acpSession{ID: sessionID, DSHSessionID: sessionID, CWD: root, Engine: "dsh", Model: modelID}
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Second)
	defer cancel()
	first := acpTurn{Operation: "44444444-4444-4444-8444-444444444441", Prompt: firstPrompt}
	err = runDSHTurn(ctx, cfg, session, &first, false, nil)
	var dispatched [4]int32
	for i := range workerCalls {
		dispatched[i] = workerCalls[i].Load()
	}
	if err != nil {
		t.Fatalf("team turn failed: %v; worker requests dispatched: %v", err, dispatched)
	}
	if first.Answer != "lead collected the results" || dispatched != [4]int32{1, 1, 1, 1} {
		t.Fatalf("lead answer %q; worker requests dispatched: %v", first.Answer, dispatched)
	}
	completed, failed := 0, 0
	for _, notice := range first.DSHMemberNotices {
		switch notice.StopReason {
		case "completed":
			completed++
		case "error":
			if strings.Contains(notice.Text, "reviewer-0") {
				failed++
			}
		}
	}
	wantFailed := 0
	if timeoutWorker {
		wantFailed = 1
	}
	if completed != len(workerCalls)-wantFailed || failed != wantFailed {
		t.Fatalf("expected %d completed and %d failed member outcomes, got %+v", len(workerCalls)-wantFailed, wantFailed, first.DSHMemberNotices)
	}
	next := acpTurn{Operation: "44444444-4444-4444-8444-444444444442", Prompt: nextPrompt}
	if err := runDSHTurn(ctx, cfg, session, &next, true, nil); err != nil || next.Answer != "original session replied" {
		t.Fatalf("follow-up failed in original session: %v, answer %q", err, next.Answer)
	}
	t.Logf("dispatched %v, collected %d successful and %d failed member outcomes, then replied in the same session", dispatched, completed, failed)
}
