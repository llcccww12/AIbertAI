package main

import (
	"bufio"
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

// Opt-in integration for an isolated SDK built from the experimental patch.
func TestDSHGatewayPatchedSDKThirtyTools(t *testing.T) {
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Skip("patched SDK build not configured")
	}
	const session = "11111111-1111-4111-8111-111111111111"
	const modelID = "22222222-2222-4222-8222-222222222222"
	const keyID = "33333333-3333-4333-8333-333333333333"
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "input.txt"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := newBackendClient("https://gateway.test", false)
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	firstChunk := make(chan struct{})
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch {
		case r.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, modelID)
		case r.URL.Path == "/api/v1/api-keys" && r.Method == "POST":
			fmt.Fprintf(w, `{"id":%q,"api_key":"key"}`, keyID)
		case r.URL.Path == "/api/v1/api-keys/"+keyID && r.Method == "DELETE":
		case r.URL.Path == "/v1/chat/completions":
			n := calls.Add(1)
			if n == 38 {
				w.WriteHeader(http.StatusServiceUnavailable)
				fmt.Fprint(w, `{"error":{"message":"unavailable"}}`)
				return w.Result(), nil
			}
			if n == 34 {
				reader, writer := io.Pipe()
				go func() {
					_, _ = fmt.Fprint(writer, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"partial\"},\"finish_reason\":null}]}\n\n")
					close(firstChunk)
					<-r.Context().Done()
					_ = writer.CloseWithError(r.Context().Err())
				}()
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: reader}, nil
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("X-Billing-Transaction-ID", fmt.Sprintf("tx-%d", n))
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
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
		}
		return w.Result(), nil
	})
	model := selectedModelConfig{ID: modelID, Protocol: "openai_chat_completions", DisplayName: "Test", ContextWindowTokens: 100000, MaxOutputTokens: 4096}
	gateway := &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "account"}, nil }}
	local, err := newDSHLocalGateway(gateway, session, model, filepath.Join(root, "journal"))
	if err != nil {
		t.Fatal(err)
	}
	local.requireInvocationID = true
	var goalObserver *dshGoalObserver
	if os.Getenv("DSH_GOAL_VERIFY_COMPACTION") == "1" {
		storeRoot, err := os.OpenRoot(root)
		if err != nil {
			t.Fatal(err)
		}
		defer storeRoot.Close()
		const goalID = "55555555-5555-4555-8555-555555555555"
		row := acpSession{ID: session, Engine: "dsh", Goal: &desktopGoal{ID: goalID, Current: true, Status: "active", Limits: desktopGoalLimit{Tokens: 10000}}}
		live := &acpLive{store: &desktopSessionStore{Root: storeRoot}, sessions: map[string]acpSession{session: row}}
		goalObserver = &dshGoalObserver{live: live, sessionID: session, goalID: goalID}
		local.admitGoalInvocation = func(id string) error { return live.admitDSHGoalInvocation(session, goalID, id) }
	}
	base, stop, err := local.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer stop(context.Background())
	patch, err := local.Patch(base, model)
	if err != nil {
		t.Fatal(err)
	}
	patchPath := filepath.Join(root, "patch.json")
	if err = os.WriteFile(patchPath, patch, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, entry, "--profile", "sdk", "--patch", patchPath)
	cmd.Dir = root
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "DSH_HOME=" + filepath.Join(root, "home"), "DSH_TELEMETRY_DISABLED=1", "DSH_PERMISSION_MODE=read-only", "DSH_AIBERTAI_LOCAL_TOKEN=" + local.token}
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { stdin.Close(); cmd.Process.Kill(); cmd.Wait() }()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), 8<<20)
	receipts, tools := 0, 0
	var terminal, answer string
	var discardReceipt bool
	var expectFailure bool
	var failedReceipt bool
	var lostID string
	var lastResult json.RawMessage
	var diagnostic []string
	call := func(id int, method string, params any) {
		t.Helper()
		if err := json.NewEncoder(stdin).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
			t.Fatal(err)
		}
		for scanner.Scan() {
			var frame struct {
				ID     int             `json:"id"`
				Method string          `json:"method"`
				Params json.RawMessage `json:"params"`
				Error  json.RawMessage `json:"error"`
			}
			if json.Unmarshal(scanner.Bytes(), &frame) != nil {
				continue
			}
			if frame.ID == id {
				var reply struct {
					Result json.RawMessage `json:"result"`
				}
				if err := json.Unmarshal(scanner.Bytes(), &reply); err != nil {
					t.Fatal(err)
				}
				lastResult = reply.Result
				if len(frame.Error) > 0 {
					t.Fatalf("%s: %s", method, frame.Error)
				}
				return
			}
			if frame.Method == "aibertai/flush-probe" {
				diagnostic = append(diagnostic, string(frame.Params))
			}
			if frame.Method == "aibertai/model-settled" {
				if goalObserver != nil {
					if err := goalObserver.observe(frame.Method, frame.Params); err != nil {
						t.Fatalf("goal settlement: %v", err)
					}
				}
				var receipt struct {
					SessionID    string `json:"sessionId"`
					InvocationID string `json:"invocationId"`
					State        string `json:"state"`
					Seq          int64  `json:"seq"`
				}
				if err := json.Unmarshal(frame.Params, &receipt); err != nil {
					t.Fatal(err)
				}
				if discardReceipt {
					lostID = receipt.InvocationID
					continue
				}
				if expectFailure && receipt.State == "failed" {
					failedReceipt = true
					if err := local.acceptSDKSettlement(receipt.SessionID, receipt.InvocationID, receipt.State, receipt.Seq); err == nil {
						t.Fatal("released failed invocation")
					}
					continue
				}
				if err := local.acceptSDKSettlement(receipt.SessionID, receipt.InvocationID, receipt.State, receipt.Seq); err != nil {
					t.Fatalf("receipt: %+v: %v", receipt, err)
				}
				receipts++
			}
			if frame.Method == "session.event" {
				if goalObserver != nil {
					if err := goalObserver.observe(frame.Method, frame.Params); err != nil {
						t.Fatalf("goal event: %v", err)
					}
				}
				if strings.Contains(string(frame.Params), "aibertai/invocation") || strings.Contains(string(frame.Params), "assistant/message") {
					diagnostic = append(diagnostic, string(frame.Params))
				}
				var e struct {
					Event struct {
						Type string `json:"type"`
						Data struct {
							Reason struct {
								Kind string `json:"kind"`
							} `json:"reason"`
							Message struct {
								Content []struct {
									Type string `json:"type"`
									Text string `json:"text"`
								} `json:"content"`
							} `json:"message"`
						} `json:"data"`
					} `json:"event"`
				}
				if json.Unmarshal(frame.Params, &e) == nil {
					switch e.Event.Type {
					case "tool/result":
						tools++
					case "turn/end":
						terminal = e.Event.Data.Reason.Kind
					case "assistant/message":
						for _, b := range e.Event.Data.Message.Content {
							if b.Type == "text" {
								answer = b.Text
							}
						}
					}
				}
			}
		}
		t.Fatalf("%s SDK exited: %v", method, scanner.Err())
	}
	call(1, "initialize", map[string]any{"cwd": root, "provider": "aibertai", "model": modelID})
	call(2, "session/prompt", map[string]any{"sessionId": session, "contentBlocks": []map[string]string{{"type": "text", "text": "read input"}}})
	call(3, "session/wait", map[string]string{"sessionId": session})
	if calls.Load() != 31 || receipts != 31 || tools != 30 || terminal != "completed" || answer != "finished" {
		t.Fatalf("calls=%d receipts=%d tools=%d terminal=%s answer=%s diagnostic=%v", calls.Load(), receipts, tools, terminal, answer, diagnostic)
	}
	entries, err := os.ReadDir(filepath.Join(root, "journal"))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			count++
		}
	}
	if count != 31 || local.pending != nil {
		t.Fatalf("journal count=%d pending=%v", count, local.pending != nil)
	}
	// Exercise a real DSH compaction transaction over the same gateway.
	call(40, "session/query", map[string]any{"sessionId": session, "contentBlocks": []map[string]string{{"type": "text", "text": "/compact"}}})
	if calls.Load() != 32 || receipts != 32 || local.pending != nil {
		t.Fatalf("compaction calls=%d receipts=%d pending=%v", calls.Load(), receipts, local.pending != nil)
	}
	if goalObserver != nil {
		goalObserver.live.mu.Lock()
		goal := *goalObserver.live.sessions[session].Goal
		goalObserver.live.mu.Unlock()
		if goal.Usage.Tokens != 128 || goal.hasPendingDSHInvocation() {
			t.Fatalf("compaction usage not settled: %+v", goal)
		}
		goalObserver = nil
		local.admitGoalInvocation = nil
	}
	// Lose an already-flushed receipt, restart both components, and reconcile.
	discardReceipt = true
	call(4, "session/prompt", map[string]any{"sessionId": session, "contentBlocks": []map[string]string{{"type": "text", "text": "one more"}}})
	call(5, "session/wait", map[string]string{"sessionId": session})
	if lostID == "" || local.pending == nil || local.pending.CallID != lostID || local.pending.State != "delivered" {
		t.Fatal("expected an unacknowledged delivered call")
	}
	stdin.Close()
	cmd.Process.Kill()
	cmd.Wait()
	local, err = newDSHLocalGateway(gateway, session, model, filepath.Join(root, "journal"))
	if err != nil {
		t.Fatal(err)
	}
	local.requireInvocationID = true
	base2, stop2, err := local.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer stop2(context.Background())
	patch2, err := local.Patch(base2, model)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(patchPath, patch2, 0600); err != nil {
		t.Fatal(err)
	}
	cmd = exec.CommandContext(ctx, node, entry, "--profile", "sdk", "--patch", patchPath)
	cmd.Dir = root
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "DSH_HOME=" + filepath.Join(root, "home"), "DSH_TELEMETRY_DISABLED=1", "DSH_PERMISSION_MODE=read-only", "DSH_AIBERTAI_LOCAL_TOKEN=" + local.token}
	cmd.Stderr = os.Stderr
	stdin, err = cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	scanner = bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), 8<<20)
	discardReceipt = false
	call(6, "initialize", map[string]any{"cwd": root, "provider": "aibertai", "model": modelID})
	call(7, "session/resume", map[string]string{"sessionId": session})
	call(8, "aibertai/reconcile", map[string]string{"sessionId": session, "invocationId": lostID})
	var reconciled struct {
		State string `json:"state"`
		Seq   int64  `json:"seq"`
	}
	if err = json.Unmarshal(lastResult, &reconciled); err != nil || reconciled.State != "committed" {
		t.Fatalf("reconcile=%s err=%v", lastResult, err)
	}
	if err = local.acceptSDKSettlement(session, lostID, reconciled.State, reconciled.Seq); err != nil {
		t.Fatal(err)
	}
	if local.pending != nil || calls.Load() != 33 {
		t.Fatalf("recovery pending=%v calls=%d", local.pending != nil, calls.Load())
	}
	request := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"`+modelID+`","stream":true}`))
	request.Header.Set("Authorization", "Bearer "+local.token)
	request.Header.Set("X-AIBERTAI-Invocation-ID", lostID)
	w := httptest.NewRecorder()
	local.ServeHTTP(w, request)
	if w.Code != http.StatusConflict || calls.Load() != 33 {
		t.Fatal("recovery replay reached gateway")
	}
	// Crash during an incomplete model stream; preserve unknown status and do
	// not resend, even if a fabricated committed receipt is supplied.
	call(9, "session/prompt", map[string]any{"sessionId": session, "contentBlocks": []map[string]string{{"type": "text", "text": "start unfinished"}}})
	if err = json.NewEncoder(stdin).Encode(map[string]any{"jsonrpc": "2.0", "id": 10, "method": "session/wait", "params": map[string]string{"sessionId": session}}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-firstChunk:
	case <-ctx.Done():
		t.Fatal("stream did not start")
	}
	stdin.Close()
	cmd.Process.Kill()
	cmd.Wait()
	local.mu.Lock()
	unfinished := local.pending
	local.mu.Unlock()
	if unfinished == nil {
		t.Fatal("missing interrupted pending call")
	}
	unknownID := unfinished.CallID
	restored, err := newDSHLocalGateway(gateway, session, model, filepath.Join(root, "journal"))
	if err != nil {
		t.Fatal(err)
	}
	restored.requireInvocationID = true
	if restored.pending == nil || restored.pending.CallID != unknownID || restored.pending.State != "dispatched" {
		t.Fatal("unknown call not retained")
	}
	if err = restored.acceptSDKSettlement(session, unknownID, "committed", 999); err == nil {
		t.Fatal("released an incomplete stream")
	}
	request = httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"`+modelID+`","stream":true}`))
	request.Header.Set("Authorization", "Bearer "+restored.token)
	request.Header.Set("X-AIBERTAI-Invocation-ID", unknownID)
	w = httptest.NewRecorder()
	restored.ServeHTTP(w, request)
	if w.Code != http.StatusConflict || calls.Load() != 34 {
		t.Fatal("interrupted request replayed")
	}
	// A separate session accumulates compactable history. The mock returns 503
	// for its compaction invocation; the durable failure must not release it.
	const failedSession = "55555555-5555-4555-8555-555555555555"
	local, err = newDSHLocalGateway(gateway, failedSession, model, filepath.Join(root, "failure-journal"))
	if err != nil {
		t.Fatal(err)
	}
	local.requireInvocationID = true
	base3, stop3, err := local.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer stop3(context.Background())
	patch3, err := local.Patch(base3, model)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(patchPath, patch3, 0600); err != nil {
		t.Fatal(err)
	}
	cmd = exec.CommandContext(ctx, node, entry, "--profile", "sdk", "--patch", patchPath)
	cmd.Dir = root
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "DSH_HOME=" + filepath.Join(root, "home"), "DSH_TELEMETRY_DISABLED=1", "DSH_PERMISSION_MODE=read-only", "DSH_AIBERTAI_LOCAL_TOKEN=" + local.token}
	cmd.Stderr = os.Stderr
	stdin, err = cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	scanner = bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), 8<<20)
	call(11, "initialize", map[string]any{"cwd": root, "provider": "aibertai", "model": modelID})
	for i := 0; i < 3; i++ {
		call(12+i*2, "session/prompt", map[string]any{"sessionId": failedSession, "contentBlocks": []map[string]string{{"type": "text", "text": strings.Repeat("history ", 100)}}})
		call(13+i*2, "session/wait", map[string]string{"sessionId": failedSession})
	}
	expectFailure = true
	call(20, "session/query", map[string]any{"sessionId": failedSession, "contentBlocks": []map[string]string{{"type": "text", "text": "/compact"}}})
	if calls.Load() != 38 || !failedReceipt || local.pending == nil || local.pending.State != "dispatched" {
		t.Fatalf("failed compaction calls=%d receipt=%v pending=%v", calls.Load(), failedReceipt, local.pending != nil)
	}

}
