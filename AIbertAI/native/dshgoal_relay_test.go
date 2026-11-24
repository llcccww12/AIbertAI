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
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// The WebSocket is deliberately closed after creating the goal. Only the
// synthetic backend is contacted while the independent SDK continues.
func TestDSHGoalRelayDisconnectCompletion(t *testing.T) {
	t.Run("automatic_compaction", func(t *testing.T) { testDSHGoalRelay(t, 100000, "automatic_compaction") })
	t.Run("completion", func(t *testing.T) { testDSHGoalRelay(t, 10000, "completed") })
	t.Run("budget", func(t *testing.T) { testDSHGoalRelay(t, 4, "budget_limited") })
	t.Run("pause_edit_resume", func(t *testing.T) { testDSHGoalRelay(t, 10000, "pause_edit_resume") })
	t.Run("clear_active", func(t *testing.T) { testDSHGoalRelay(t, 10000, "clear_active") })
}

func testDSHGoalRelay(t *testing.T, budget int64, expected string) {
	autoCompact := expected == "automatic_compaction"
	if autoCompact {
		expected = "completed"
	}
	pauseMode := expected == "pause_edit_resume" || expected == "clear_active"
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Skip("patched SDK build not configured")
	}
	const session = "11111111-1111-4111-8111-111111111111"
	const modelID = "22222222-2222-4222-8222-222222222222"
	const keyID = "33333333-3333-4333-8333-333333333333"
	const goalID = "44444444-4444-4444-8444-444444444444"
	rootDir := t.TempDir()
	evidence := "fixture"
	if autoCompact {
		evidence = strings.Repeat("fixture evidence line\n", 300)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "input.txt"), []byte(evidence), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	var calls, summaries atomic.Int32
	var resumedAfterSummary atomic.Bool
	var admissions atomic.Int32
	admissionReached, admissionRelease := make(chan struct{}), make(chan struct{})
	var live *acpLive
	gate := make(chan struct{})
	pattern := regexp.MustCompile(`goal-[0-9a-f-]{36}`)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/settings":
			fmt.Fprint(w, `{"settings":{"session_reporting":{"enabled":true,"level":"full"}}}`)
		case r.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}],"total_count":1}`, modelID)
		case r.URL.Path == "/api/v1/api-keys" && r.Method == "POST":
			fmt.Fprintf(w, `{"id":%q,"api_key":"fixture"}`, keyID)
		case r.URL.Path == "/api/v1/api-keys/"+keyID && r.Method == "DELETE":
		case r.URL.Path == "/api/v1/sessions/"+session:
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/api/v1/sessions/"+session+"/turns":
			fmt.Fprint(w, `{"acked_turn":1}`)
		case r.URL.Path == "/v1/chat/completions":
			data, _ := io.ReadAll(r.Body)
			var n int32
			if autoCompact && dshSyntheticCompactionRequest(data) {
				summaries.Add(1)
			} else {
				n = calls.Add(1)
				if autoCompact && strings.Contains(string(data), "GOAL_CHECKPOINT") {
					resumedAfterSummary.Store(true)
				}
			}
			if n == 1 {
				select {
				case <-gate:
				case <-r.Context().Done():
					return
				}
			}
			w.Header().Set("Content-Type", "text/event-stream")
			var delta map[string]any
			finish := "stop"
			inputTokens, outputTokens := 3, 1
			switch n {
			case 0:
				delta = map[string]any{"role": "assistant", "content": "GOAL_CHECKPOINT: fixture evidence has been read."}
			case 1:
				delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "read-1", "type": "function", "function": map[string]string{"name": "read", "arguments": `{"file_path":"input.txt"}`}}}}
				finish = "tool_calls"
			case 2:
				progress := "Progress recorded."
				if autoCompact {
					// The meter combines the recorded provider usage with new input.
					progress = strings.Repeat("Goal progress. ", 1150)
					inputTokens, outputTokens = 18000, 4300
				}
				delta = map[string]any{"role": "assistant", "content": progress}
			case 3:
				delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "get-goal", "type": "function", "function": map[string]string{"name": "get_goal", "arguments": "{}"}}}}
				finish = "tool_calls"
			case 4:
				id := pattern.FindString(string(data))
				if id == "" {
					t.Error("model input lacked goal id")
				}
				revision := int64(1)
				if pauseMode {
					live.mu.Lock()
					revision = live.sessions[session].Goal.DSHRevision
					live.mu.Unlock()
				}
				args, _ := json.Marshal(map[string]any{"goal_id": id, "revision": revision, "action": "complete"})
				delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "complete-goal", "type": "function", "function": map[string]string{"name": "update_goal", "arguments": string(args)}}}}
				finish = "tool_calls"
			default:
				delta = map[string]any{"role": "assistant", "content": "Fixture inspection completed."}
			}
			emit := func(v any) { b, _ := json.Marshal(v); fmt.Fprintf(w, "data: %s\n\n", b) }
			emit(map[string]any{"id": "chat-1", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": nil}}})
			emit(map[string]any{"id": "chat-1", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": finish}}, "usage": map[string]int{"prompt_tokens": inputTokens, "completion_tokens": outputTokens, "total_tokens": inputTokens + outputTokens}})
			fmt.Fprint(w, "data: [DONE]\n\n")
		default:
			t.Errorf("unexpected backend %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer backend.Close()
	client := &client{backend.URL, backend.Client()}
	gateway := &dshGateway{client: client, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}
	selected := selectedModelConfig{ID: modelID, Protocol: "openai_chat_completions", DisplayName: "Test", ContextWindowTokens: 100000, MaxOutputTokens: 4096}
	if autoCompact {
		selected.ContextWindowTokens, selected.MaxOutputTokens = 32768, 8192
	}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, Home: filepath.Join(rootDir, "home"), Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(rootDir, "journal"), gateway, func(context.Context, acpSession) (selectedModelConfig, error) { return selected, nil })
	if err != nil {
		t.Fatal(err)
	}
	if pauseMode {
		prepare := cfg.Prepare
		cfg.Prepare = func(ctx context.Context, row acpSession) (*exec.Cmd, string, string, *dshLocalGateway, func(), error) {
			cmd, provider, model, local, cleanup, err := prepare(ctx, row)
			if err == nil && local.admitGoalInvocation != nil {
				admit := local.admitGoalInvocation
				local.admitGoalInvocation = func(id string) error {
					if admissions.Add(1) == 2 {
						close(admissionReached)
						<-admissionRelease
					}
					return admit(id)
				}
			}
			return cmd, provider, model, local, cleanup, err
		}
	}
	if !pauseMode {
		cfg.LeasePool = &dshLeasePool{}
		t.Cleanup(func() { _ = cfg.LeasePool.close(context.Background()) })
	}
	now := time.Now().UTC()
	live = &acpLive{client: client, auth: authorization{Access: "fixture"}, store: &desktopSessionStore{Root: root}, cwd: rootDir, dsh: cfg, settings: desktopSettings{GoalBudget: &budget}, sessions: map[string]acpSession{session: {ID: session, DSHSessionID: session, Engine: "dsh", CWD: rootDir, Model: modelID, State: "idle", Created: now, Updated: now}}}
	host := &relaySessionHost{root: root, local: &desktopRuntimeAccess{protocol: "monkeyai.token.test"}, live: live, goals: newGoalRuntime(), terminalLifetime: t.Context()}
	desktop := httptest.NewServer(host)
	defer desktop.Close()
	host.local.address = strings.TrimPrefix(desktop.URL, "http://")
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	go host.superviseGoals(ctx)
	ws, _, err := websocket.Dial(ctx, desktop.URL+"/acp", &websocket.DialOptions{Subprotocols: []string{"monkeyai.v1", "monkeyai.token.test"}})
	if err != nil {
		t.Fatal(err)
	}
	send := func(id int, method string, params any) {
		t.Helper()
		if err := wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
			t.Fatal(err)
		}
		for {
			var frame map[string]any
			if err := wsjson.Read(ctx, ws, &frame); err != nil {
				t.Fatal(err)
			}
			if frame["id"] == float64(id) {
				if frame["error"] != nil {
					t.Fatalf("request failed: %v", frame)
				}
				return
			}
		}
	}
	send(1, "initialize", map[string]int{"protocolVersion": 1})
	send(2, "_monkeyai/session/goal", map[string]any{"sessionId": session, "action": "create", "requestId": goalID, "objective": "Inspect the fixture and report the result."})
	close(gate)
	if pauseMode {
		select {
		case <-admissionReached:
		case <-ctx.Done():
			t.Fatal("second admission not reached")
		}
		if expected == "clear_active" {
			send(3, "_monkeyai/session/goal", map[string]any{"sessionId": session, "action": "clear", "requestId": "55555555-5555-4555-8555-555555555555", "expectedGoalId": goalID, "expectedGeneration": 1})
			close(admissionRelease)
			for {
				row, err := live.load(session)
				if err != nil {
					t.Fatal(err)
				}
				if row.Goal.Status == "cleared" && !row.Goal.Current && row.GoalRun == "" {
					return
				}
				if row.Goal.Reason == "dsh_goal_clear_failed" {
					t.Fatalf("active clear failed: %+v", row.Goal)
				}
				select {
				case <-time.After(50 * time.Millisecond):
				case <-ctx.Done():
					t.Fatalf("active clear timed out: %+v", row.Goal)
				}
			}
		}
		send(3, "_monkeyai/session/goal", map[string]any{"sessionId": session, "action": "pause", "requestId": "55555555-5555-4555-8555-555555555555", "expectedGoalId": goalID, "expectedGeneration": 1})
		close(admissionRelease)
		for {
			row, err := live.load(session)
			if err != nil {
				t.Fatal(err)
			}
			if row.State == "idle" && row.Goal.Status == "paused" {
				if row.Goal.Uncertain || row.Goal.Reason != "user_paused" {
					t.Fatalf("pause did not settle cleanly: %+v", row.Goal)
				}
				break
			}
			select {
			case <-time.After(50 * time.Millisecond):
			case <-ctx.Done():
				t.Fatalf("pause did not complete: %+v", row.Goal)
			}
		}
		send(4, "_monkeyai/session/goal", map[string]any{"sessionId": session, "action": "edit", "requestId": "66666666-6666-4666-8666-666666666666", "expectedGoalId": goalID, "expectedGeneration": 1, "objective": "Inspect the fixture and summarize it."})
		send(5, "_monkeyai/session/goal", map[string]any{"sessionId": session, "action": "resume", "requestId": "77777777-7777-4777-8777-777777777777", "expectedGoalId": goalID, "expectedGeneration": 2})
		expected = "completed"
	}
	ws.CloseNow()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		row, err := live.load(session)
		if err != nil {
			t.Fatal(err)
		}
		if row.Goal != nil && row.Goal.Status == expected && row.State == "idle" {
			if expected == "budget_limited" {
				if row.Goal.Usage.Tokens != 4 || row.Goal.Reason != "token_budget" || calls.Load() != 1 {
					t.Fatalf("incorrect budget stop: %+v calls=%d", row.Goal, calls.Load())
				}
				ws, _, err = websocket.Dial(ctx, desktop.URL+"/acp", &websocket.DialOptions{Subprotocols: []string{"monkeyai.v1", "monkeyai.token.test"}})
				if err != nil {
					t.Fatal(err)
				}
				defer ws.CloseNow()
				send(3, "initialize", map[string]int{"protocolVersion": 1})
				send(4, "_monkeyai/session/goal", map[string]any{"sessionId": session, "action": "clear", "requestId": "55555555-5555-4555-8555-555555555555", "expectedGoalId": goalID, "expectedGeneration": 1})
				for {
					cleared, err := live.load(session)
					if err != nil {
						t.Fatal(err)
					}
					if cleared.Goal.Status == "cleared" && !cleared.Goal.Current && cleared.GoalRun == "" {
						return
					}
					if cleared.Goal.Reason == "dsh_goal_clear_failed" {
						t.Fatalf("SDK clear failed: %+v", cleared.Goal)
					}
					select {
					case <-ticker.C:
					case <-ctx.Done():
						t.Fatalf("SDK goal not cleared: %+v", cleared.Goal)
					}
				}
			}
			expectedTokens := int64(20)
			if autoCompact {
				expectedTokens += 22300 - 4 + int64(summaries.Load())*4
				if summaries.Load() < 1 || !resumedAfterSummary.Load() {
					t.Fatalf("background goal did not continue from automatic checkpoint: summaries=%d resumed=%v", summaries.Load(), resumedAfterSummary.Load())
				}
			}
			if row.Goal.Usage.Tokens != expectedTokens || row.Goal.RoundsStarted < 2 || len(row.Turns) != 1 || row.Turns[0].Answer != "Fixture inspection completed." {
				t.Fatalf("incorrect completed goal: %+v turns=%+v", row.Goal, row.Turns)
			}
			return
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatalf("goal did not complete: %+v calls=%d", row.Goal, calls.Load())
		}
	}
}

func dshSyntheticCompactionRequest(body []byte) bool {
	var request struct {
		Messages []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if json.Unmarshal(body, &request) != nil || len(request.Messages) == 0 {
		return false
	}
	last := request.Messages[len(request.Messages)-1]
	var content string
	return last.Role == "user" && json.Unmarshal(last.Content, &content) == nil && strings.HasPrefix(content, "You are now acting as a compaction engine")
}
