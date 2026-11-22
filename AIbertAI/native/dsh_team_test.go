package main

import (
	"context"
	"encoding/json"
	"fmt"

	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDSHTeamSharesTasksMessagesAndBilling(t *testing.T) {
	for _, mode := range []string{"chat", "goal"} {
		t.Run(mode, func(t *testing.T) { testDSHTeamSession(t, mode == "goal", "") })
	}
}

func TestDSHMemberFailureReturnsWorkToLead(t *testing.T) {
	for _, outcome := range []string{"error", "max-tokens"} {
		t.Run(outcome, func(t *testing.T) { testDSHTeamSession(t, false, outcome) })
	}
}

func testDSHTeamSession(t *testing.T, goal bool, memberOutcome string) {
	memberEndsEarly := memberOutcome != ""
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
	if memberEndsEarly {
		if err := os.WriteFile(filepath.Join(root, "draft.txt"), []byte("existing-reviewed-artifact"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var leadCheckedFile atomic.Bool
	var goalID atomic.Value
	var revision atomic.Int64
	var admitted sync.Map
	var settled atomic.Int32
	var calls, rootCalls, childCalls atomic.Int32
	var childRead, rootReceived, reopened atomic.Bool
	var reopenCalls atomic.Int32
	childStarted, releaseChild := make(chan struct{}), make(chan struct{})
	var unblock sync.Once
	resumeChild := func() { unblock.Do(func() { close(releaseChild) }) }
	defer resumeChild()
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch {
		case r.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, modelID)
		case r.URL.Path == "/api/v1/api-keys" && r.Method == "POST":
			fmt.Fprintf(w, `{"id":%q,"api_key":"fixture"}`, keyID)
		case r.URL.Path == "/api/v1/api-keys/"+keyID && r.Method == "DELETE":
		case r.URL.Path == "/v1/chat/completions":
			var req struct {
				Messages []struct {
					Role    string
					Content json.RawMessage
				}
				Tools []struct{ Function struct{ Name string } }
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error(err)
			}
			child, leadGuidance := false, false
			for _, msg := range req.Messages {
				content := string(msg.Content)
				if msg.Role == "system" && strings.Contains(content, "check the existing files") {
					leadGuidance = true
				}
				if msg.Role == "user" && strings.Contains(content, "worker-isolated-prompt") {
					child = true
				}
				if msg.Role == "user" && strings.Contains(content, "checked-by-teammate") {
					rootReceived.Store(true)
				}
			}
			name, args, finish, answer := "", "", "stop", "team finished"
			n := calls.Add(1)
			if memberEndsEarly {
				if !child && !leadGuidance {
					t.Error("the lead did not receive member recovery guidance")
				}
				if child {
					childCalls.Add(1)
					if memberOutcome == "error" {
						w.Header().Set(dshNotDispatchedHeader, r.Header.Get("Idempotency-Key"))
						w.WriteHeader(http.StatusForbidden)
						return w.Result(), nil
					}
					answer, finish = "Partial work is in draft.txt", "length"
				} else {
					recovered, verified := false, false
					for _, m := range req.Messages {
						if m.Role == "user" && (strings.Contains(string(m.Content), "Partial work is in draft.txt") || strings.Contains(string(m.Content), "failed before it finished")) {
							recovered = true
						}
						if m.Role == "tool" && strings.Contains(string(m.Content), "existing-reviewed-artifact") {
							verified = true
						}
					}
					switch {
					case rootCalls.Add(1) == 1:
						name, args = "subagent", `{"name":"reviewer","description":"Review draft","prompt":"worker-isolated-prompt"}`
					case verified:
						leadCheckedFile.Store(true)
						answer = "Confirmed the existing draft."
					case recovered:
						rootReceived.Store(true)
						name, args = "read", `{"file_path":"draft.txt"}`
					default:
						answer = "Delegated the review."
					}
				}
			} else if reopened.Load() {
				if reopenCalls.Add(1) == 1 {
					name, args = "team_task_list", "{}"
				}
			} else if child {
				switch childCalls.Add(1) {
				case 1:
					if !goal {
						close(childStarted)
						select {
						case <-releaseChild:
						case <-r.Context().Done():
							return nil, r.Context().Err()
						}
					}
					name, args = "team_task_list", "{}"
				case 2:
					for _, msg := range req.Messages {
						if msg.Role == "tool" && strings.Contains(string(msg.Content), "fixture shared work") {
							childRead.Store(true)
						}
					}
					name, args = "team_task_update", `{"task_id":"task-1","expected_revision":1,"action":"claim"}`
				case 3:
					name, args = "team_task_update", `{"task_id":"task-1","expected_revision":2,"action":"complete"}`
				case 4:
					name, args = "send_message", `{"agent":"lead","message":"checked-by-teammate"}`
				}
			} else {
				switch rootCalls.Add(1) {
				case 1:
					names := []string{}
					for _, tool := range req.Tools {
						names = append(names, tool.Function.Name)
					}
					for _, expected := range []string{"team_task_create", "team_task_get", "team_task_update", "subagent", "send_message", "wait_agent"} {
						if !slices.Contains(names, expected) {
							t.Errorf("missing native team tool %q", expected)
						}
					}
					name, args = "team_task_create", `{"subject":"fixture shared work","description":"Confirm receipt"}`
				case 2:
					name, args = "subagent", `{"name":"reviewer","description":"Inspect task board","prompt":"worker-isolated-prompt"}`
				case 3:
					name, args = "wait_agent", `{"timeout_ms":10000}`
				case 4:
					if goal {
						name = "update_goal"
						token, _ := goalID.Load().(string)
						args = fmt.Sprintf(`{"goal_id":%q,"revision":%d,"action":"complete"}`, token, revision.Load())
					}
				}
			}
			if n > 16 {
				return nil, fmt.Errorf("unexpected extra model call")
			}
			w.Header().Set("Content-Type", "text/event-stream")
			if name != "" {
				finish = "tool_calls"
			}
			var delta map[string]any
			if name != "" {
				delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("call-%d", n), "type": "function", "function": map[string]string{"name": name, "arguments": args}}}}
			} else {
				delta = map[string]any{"role": "assistant", "content": answer}
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
	if memberEndsEarly {
		enabled := true
		row.AgentTeamEnabled = &enabled
	}
	turn := acpTurn{Operation: "44444444-4444-4444-8444-444444444444", Prompt: "Coordinate our isolated fixture."}
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	turnCtx := ctx
	if goal {
		turn.Prompt = "/goal " + turn.Prompt
		turnCtx = context.WithValue(turnCtx, dshGoalAdmissionKey{}, func(id string) error { admitted.Store(id, true); return nil })
		turnCtx = context.WithValue(turnCtx, dshSDKOptionsKey{}, dshSDKOptions{Query: true, Observe: func(method string, raw json.RawMessage) error {
			switch method {
			case "session.event":
				var ev struct {
					Event struct {
						Type string
						Data struct {
							Goal struct {
								ID       string
								Revision int64
							}
						}
					}
				}
				if json.Unmarshal(raw, &ev) == nil && ev.Event.Type == "goal/change" && ev.Event.Data.Goal.ID != "" {
					goalID.Store(ev.Event.Data.Goal.ID)
					revision.Store(ev.Event.Data.Goal.Revision)
				}
			case "aibertai/model-settled":
				var receipt dshFailedCompactionReceipt
				if json.Unmarshal(raw, &receipt) != nil || receipt.SessionID != session || receipt.State != "committed" {
					return fmt.Errorf("invalid goal settlement")
				}
				if _, found := admitted.LoadAndDelete(receipt.InvocationID); !found {
					return fmt.Errorf("unadmitted goal settlement")
				}
				settled.Add(1)
			}
			return nil
		}})
	}
	done := make(chan error, 1)
	go func() { done <- runDSHTurn(turnCtx, cfg, row, &turn, false, nil) }()
	if !goal && !memberEndsEarly {
		select {
		case <-childStarted:
		case err := <-done:
			t.Fatalf("turn ended before member started: %v", err)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		host := &relaySessionHost{live: &acpLive{dsh: cfg, store: &desktopSessionStore{}, sessions: map[string]acpSession{row.ID: row}}}
		req, _ := json.Marshal(dshMemberRequest{SessionID: row.ID})
		response, err := host.sessionMembers(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		var members []struct{ ID, Name, Activity string }
		if json.Unmarshal(response.(map[string]json.RawMessage)["members"], &members) != nil || len(members) != 1 || members[0].Name != "reviewer" || members[0].Activity != "running" {
			t.Fatalf("running child missing: %+v", members)
		}
		req, _ = json.Marshal(dshMemberRequest{SessionID: row.ID, MemberID: members[0].ID})
		response, err = host.sessionMembers(ctx, req)
		if err != nil || !strings.Contains(string(response.(map[string]json.RawMessage)["entries"]), "worker-isolated-prompt") {
			t.Fatalf("could not open running member: %v", err)
		}
		resumeChild()
	}
	if err := <-done; err != nil {
		t.Fatalf("team session failed after %d root / %d child calls: %v", rootCalls.Load(), childCalls.Load(), err)
	}
	if memberEndsEarly {
		if !rootReceived.Load() || !leadCheckedFile.Load() || childCalls.Load() != 1 || turn.Answer != "Confirmed the existing draft." {
			t.Fatalf("handoff incomplete: lead=%d, child=%d, received=%t, verified=%t, reply=%q", rootCalls.Load(), childCalls.Load(), rootReceived.Load(), leadCheckedFile.Load(), turn.Answer)
		}
		if len(turn.DSHMemberNotices) != 1 || turn.DSHMemberNotices[0].StopReason != memberOutcome || turn.DSHMemberNotices[0].Text == "" {
			t.Fatalf("missing durable member outcome: %+v", turn.DSHMemberNotices)
		}
		row.Turns = []acpTurn{turn}
		old, _ := json.Marshal(row.responseMessage(0, map[string]any{}))
		var restored acpSession
		encoded, _ := json.Marshal(row)
		if err := json.Unmarshal(encoded, &restored); err != nil {
			t.Fatal(err)
		}
		history, _ := json.Marshal(restored.responseMessage(0, map[string]any{}))
		if string(old) != string(history) || !strings.Contains(string(history), "dshMemberNotices") {
			t.Fatal("member notice lost on reopen")
		}
		return
	}
	if goal && settled.Load() != calls.Load() {
		t.Fatalf("goal only accounted for %d of %d calls", settled.Load(), calls.Load())
	}
	if !childRead.Load() || !rootReceived.Load() || rootCalls.Load() < 4 || childCalls.Load() != 5 {
		t.Fatalf("shared tasks/messages incomplete: read=%t received=%t calls(root=%d child=%d)", childRead.Load(), rootReceived.Load(), rootCalls.Load(), childCalls.Load())
	}
	if turn.Usage.Input == nil || *turn.Usage.Input != 3*int64(calls.Load()) || turn.Usage.Output == nil || *turn.Usage.Output != int64(calls.Load()) {
		t.Fatalf("expected all %d billed calls, got %+v", calls.Load(), turn.Usage)
	}
	for _, run := range turn.ToolRuns {
		if run.Report.Failed != 0 {
			t.Fatalf("tool failed: %+v", run)
		}
	}
	if !turn.DSHTeamBoard.visible() || len(turn.DSHTeamBoard.Tasks) != 1 {
		t.Fatal("missing shared task board snapshot")
	}
	var task struct{ Status, OwnerName string }
	if json.Unmarshal(turn.DSHTeamBoard.Tasks[0], &task) != nil || task.Status != "completed" || task.OwnerName != "reviewer" {
		t.Fatalf("wrong completed task: %+v", task)
	}
	row.Turns = []acpTurn{turn}
	checkDSHMemberView(t, ctx, cfg, row, true)
	if row.responseMessage(0, map[string]any{})["dshTeamBoard"] != turn.DSHTeamBoard {
		t.Fatal("snapshot missing from employee chat")
	}
	if !goal {
		testDSHRestoredAgentTeamMode(t, ctx, cfg, row)
	}
	// Closing the entire SDK process is stricter than closing the employee chat.
	if err := cfg.LeasePool.close(ctx); err != nil {
		t.Fatal(err)
	}
	cfg.LeasePool = &dshLeasePool{}
	checkDSHMemberView(t, ctx, cfg, row, false)
	reopened.Store(true)
	followup := acpTurn{Operation: "55555555-5555-4555-8555-555555555555", Prompt: "Show the task board."}
	if err := runDSHTurn(ctx, cfg, row, &followup, true, nil); err != nil {
		t.Fatal(err)
	}
	if len(followup.ToolRuns) != 1 || !strings.Contains(followup.ToolRuns[0].Output, `"status":"completed"`) || !strings.Contains(followup.ToolRuns[0].Output, "reviewer") {
		t.Fatalf("reopened session lost team result: %+v", followup.ToolRuns)
	}
}

func TestDSHTeamReceiptsBoundToRootAndOperation(t *testing.T) {
	const root = "11111111-1111-4111-8111-111111111111"
	const op = "22222222-2222-4222-8222-222222222222"
	const call = "33333333-3333-4333-8333-333333333333"
	team := dshTeamSessions{}
	team.begin(root, op)
	mark := func(member, owner, operation string) {
		frame, _ := json.Marshal(map[string]any{"sessionId": member, "aibertaiTeamRoot": owner, "aibertaiOperationId": operation, "event": map[string]any{"type": "aibertai/invocation", "data": map[string]string{"invocationId": call}}})
		team.track("session.event", frame)
	}
	team.track("subagent.started", json.RawMessage(`{"parentSessionId":"11111111-1111-4111-8111-111111111111","childSessionId":"child"}`))
	mark("child", "other", op)
	mark("child", root, "expired")
	mark("child", "", "")
	if len(team.calls) != 0 {
		t.Fatal("accepted unrelated ownership or stale lease")
	}
	mark("cold-descendant", root, op)
	if team.calls[call] != "cold-descendant" {
		t.Fatal("missing SDK-owned descendant")
	}
	team.begin(root, "next")
	if len(team.calls) != 0 {
		t.Fatal("invocation leaked across leases")
	}
}
