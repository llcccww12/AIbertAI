package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

// Opt-in long-running SDK goal exercise. It uses only a synthetic backend and
// read-only fixture. It deliberately stops the unfinished goal at the end.
func TestDSHGoalSoak(t *testing.T) {
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	duration, err := time.ParseDuration(os.Getenv("DSH_GOAL_SOAK_DURATION"))
	if node == "" || entry == "" || err != nil {
		t.Skip("goal soak not configured")
	}
	delay, err := time.ParseDuration(os.Getenv("DSH_GOAL_SOAK_DELAY"))
	if err != nil {
		t.Fatal(err)
	}
	minimum, err := strconv.Atoi(os.Getenv("DSH_GOAL_SOAK_MIN_TOOLS"))
	if err != nil || minimum < 1 {
		t.Fatal("invalid minimum")
	}
	const session = "11111111-1111-4111-8111-111111111111"
	const modelID = "22222222-2222-4222-8222-222222222222"
	const keyID = "33333333-3333-4333-8333-333333333333"
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "input.txt"), []byte("isolated goal fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := newBackendClient("https://gateway.test", false)
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int64
	var remoteGoal atomic.Value
	var completionEvidence bool
	completionMode := os.Getenv("DSH_GOAL_SOAK_COMPLETE")
	complete := completionMode == "1" || completionMode == "after_duration"
	var completionRequested atomic.Bool
	var completionStep atomic.Int32
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
			var delta map[string]any
			finish := "stop"
			step := int32(0)
			if completionMode == "1" && n >= 3 {
				step = int32(n - 2)
			}
			if completionMode == "after_duration" && completionRequested.Load() {
				step = completionStep.Add(1)
			}
			if complete && (step == 1 || step == 2) {
				name, args := "get_goal", `{}`
				if step == 2 {
					id, ok := remoteGoal.Load().(string)
					if !ok {
						return nil, fmt.Errorf("goal id was not observed")
					}
					name = "update_goal"
					encoded, _ := json.Marshal(map[string]any{"goal_id": id, "revision": 1, "action": "complete"})
					args = string(encoded)
				}
				delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("goal-%d", n), "type": "function", "function": map[string]string{"name": name, "arguments": args}}}}
				finish = "tool_calls"
			} else if complete && step > 2 {
				delta = map[string]any{"role": "assistant", "content": "Fixture inspection completed."}
			} else if n%2 == 1 {
				select {
				case <-time.After(delay):
				case <-r.Context().Done():
					return nil, r.Context().Err()
				}
				args, _ := json.Marshal(map[string]string{"file_path": "input.txt"})
				delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("read-%d", n), "type": "function", "function": map[string]string{"name": "read", "arguments": string(args)}}}}
				finish = "tool_calls"
			} else {
				delta = map[string]any{"role": "assistant", "content": "Progress recorded; continue."}
			}
			emit := func(v any) { data, _ := json.Marshal(v); fmt.Fprintf(w, "data: %s\n\n", data) }
			emit(map[string]any{"id": "chat-1", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": nil}}})
			emit(map[string]any{"id": "chat-1", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": finish}}, "usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 1, "total_tokens": 4}})
			fmt.Fprint(w, "data: [DONE]\n\n")
		default:
			w.WriteHeader(404)
		}
		return w.Result(), nil
	})
	model := selectedModelConfig{ID: modelID, Protocol: "openai_chat_completions", DisplayName: "Test", ContextWindowTokens: 100000, MaxOutputTokens: 4096}
	gateway := &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}
	local, err := newDSHLocalGateway(gateway, session, model, filepath.Join(root, "journal"))
	if err != nil {
		t.Fatal(err)
	}
	local.requireInvocationID = true
	var observer *dshGoalObserver
	if os.Getenv("DSH_GOAL_SOAK_BUDGET") == "1" {
		storeRoot, err := os.OpenRoot(filepath.Join(root, "journal"))
		if err != nil {
			t.Fatal(err)
		}
		defer storeRoot.Close()
		const goalID = "44444444-4444-4444-8444-444444444444"
		row := acpSession{ID: session, Engine: "dsh", GoalRun: goalID, Turns: []acpTurn{{Operation: goalID}}, Goal: &desktopGoal{ID: goalID, Objective: "Repeatedly inspect the isolated fixture and record progress.", Status: "active", Current: true, Limits: desktopGoalLimit{Tokens: 10000000}}}
		live := &acpLive{store: &desktopSessionStore{Root: storeRoot}, sessions: map[string]acpSession{session: row}}
		observer = &dshGoalObserver{live: live, sessionID: session, goalID: goalID}
		local.admitGoalInvocation = func(invocationID string) error { return live.admitDSHGoalInvocation(session, goalID, invocationID) }
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
	if err := os.WriteFile(patchPath, patch, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
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
	defer func() { cancel(); stdin.Close(); cmd.Wait() }()
	frames := make(chan map[string]json.RawMessage, 32)
	go func() {
		defer close(frames)
		s := bufio.NewScanner(stdout)
		s.Buffer(make([]byte, 4096), 8<<20)
		for s.Scan() {
			var f map[string]json.RawMessage
			if json.Unmarshal(s.Bytes(), &f) == nil {
				select {
				case frames <- f:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	send := func(id int, method string, params any) {
		t.Helper()
		if err := json.NewEncoder(stdin).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
			t.Fatal(err)
		}
	}
	send(1, "initialize", map[string]any{"cwd": root, "provider": "aibertai", "model": modelID})
	started := time.Now()
	deadline := time.NewTimer(duration)
	defer deadline.Stop()
	timeout := time.NewTimer(duration + 20*time.Minute)
	defer timeout.Stop()
	ready, elapsed, goalCreated := false, false, false
	var finalResponse *string
	tools := 0
	for {
		select {
		case <-deadline.C:
			elapsed = true
			if completionMode == "after_duration" {
				completionRequested.Store(true)
			}
		case <-timeout.C:
			t.Fatalf("soak timed out: goal=%v tools=%d calls=%d", goalCreated, tools, calls.Load())
		case f, ok := <-frames:
			if !ok {
				t.Fatalf("SDK exited: goal=%v tools=%d calls=%d", goalCreated, tools, calls.Load())
			}
			var id int
			_ = json.Unmarshal(f["id"], &id)
			if id != 0 && len(f["error"]) != 0 && string(f["error"]) != "null" {
				t.Fatalf("RPC %d: %s", id, f["error"])
			}
			if id == 1 && !ready {
				ready = true
				send(2, "session/query", map[string]any{"sessionId": session, "contentBlocks": []map[string]string{{"type": "text", "text": "/goal Repeatedly inspect the isolated fixture and record progress."}}})
			}
			var method string
			_ = json.Unmarshal(f["method"], &method)
			if observer != nil && (method == "session.event" || method == "aibertai/model-settled") {
				if err := observer.observe(method, f["params"]); err != nil {
					t.Fatalf("goal observation: %v: %s", err, f["params"])
				}
			}
			if method == "session.query.completed" && complete {
				var result struct{ SessionID, FinishReason, FinalResponse string }
				if json.Unmarshal(f["params"], &result) != nil || result.SessionID != session || result.FinishReason != "completed" || observer == nil {
					t.Fatalf("unexpected query result: %s", f["params"])
				}
				finalResponse = &result.FinalResponse
			}
			if method == "aibertai/model-settled" {
				var receipt struct {
					SessionID, InvocationID, State string
					Seq                            int64
				}
				if json.Unmarshal(f["params"], &receipt) != nil {
					t.Fatal("invalid receipt")
				}
				if err := local.acceptSDKSettlement(receipt.SessionID, receipt.InvocationID, receipt.State, receipt.Seq); err != nil {
					t.Fatal(err)
				}
			}
			if method == "session.event" {
				var e struct {
					Event struct {
						Type string `json:"type"`
					} `json:"event"`
				}
				_ = json.Unmarshal(f["params"], &e)
				if e.Event.Type == "aibertai/query-completed" {
					completionEvidence = true
				}
				if e.Event.Type == "goal/change" {
					goalCreated = true
					if observer != nil {
						observer.live.mu.Lock()
						remote := observer.live.sessions[session].Goal.DSHGoalID
						observer.live.mu.Unlock()
						remoteGoal.Store(remote)
					}
					t.Logf("goal change: %s", f["params"])
				}
				if e.Event.Type == "tool/result" {
					tools++
					t.Logf("goal progress: elapsed=%s tools=%d calls=%d", time.Since(started).Round(time.Second), tools, calls.Load())
				}
			}
		}
		if finalResponse != nil {
			if !completionEvidence {
				t.Fatal("goal completion lacked durable evidence event")
			}
			observer.live.mu.Lock()
			pending := observer.live.sessions[session].Goal.hasPendingDSHInvocation()
			observer.live.mu.Unlock()
			if !pending {
				if completionMode == "after_duration" && (!elapsed || tools < minimum) {
					t.Fatalf("completed before soak criteria: elapsed=%v tools=%d", elapsed, tools)
				}
				if err := observer.live.prepareDSHGoalCompletion(session, observer.goalID); err != nil {
					t.Fatal(err)
				}
				if err := observer.live.finishDSHGoal(session, observer.goalID, acpTurn{Operation: observer.goalID, Answer: *finalResponse}); err != nil {
					t.Fatal(err)
				}
				t.Logf("goal completed: elapsed=%s tools=%d calls=%d", time.Since(started), tools, calls.Load())
				return
			}
		}
		if !complete && elapsed && goalCreated && tools >= minimum {
			if observer != nil {
				observer.live.mu.Lock()
				charged := observer.live.sessions[session].Goal.Usage.Tokens
				observer.live.mu.Unlock()
				if charged < 4 {
					t.Fatal("goal usage was not persisted")
				}
			}
			t.Logf("soak complete: elapsed=%s tools=%d calls=%d", time.Since(started), tools, calls.Load())
			return
		}
		if ready && calls.Load() == 0 && time.Since(started) > 30*time.Second && delay < time.Second {
			t.Fatal("goal did not start")
		}
	}
}
