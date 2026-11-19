package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDesktopGoalPersistsAndExecutesWithBudget(t *testing.T) {
	const session = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	const operation = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	const model = "c3bd985c-9f71-4b85-937a-c41b1b104d38"
	const createID = "c4bd985c-9f71-4b85-937a-c41b1b104d38"
	const editID = "c5bd985c-9f71-4b85-937a-c41b1b104d38"
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	now, budget := time.Now().UTC(), int64(100)
	row := acpSession{ID: session, Operation: operation, Model: model, CWD: root.Name(), State: "idle", Created: now, Updated: now}
	l := &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{session: row}, settings: desktopSettings{GoalBudget: &budget}}
	create := json.RawMessage(`{"sessionId":"` + session + `","action":"create","requestId":"` + createID + `","objective":"Summarize the fixture"}`)
	for range 2 {
		if _, err := l.saveGoalAction(create, false); err != nil {
			t.Fatal(err)
		}
	}
	row = l.sessions[session]
	if row.Goal == nil || row.Goal.ID != createID || row.Goal.Version != 1 || row.Goal.Limits.Tokens != budget {
		t.Fatalf("wrong goal snapshot %+v", row.Goal)
	}
	stale := json.RawMessage(`{"sessionId":"` + session + `","action":"edit","requestId":"` + editID + `","expectedGoalId":"` + createID + `","expectedGeneration":2,"objective":"Discard"}`)
	if _, err := l.saveGoalAction(stale, false); err == nil {
		t.Fatal("accepted a stale goal edit")
	}

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var request struct{ Messages []toolConversationMessage }
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || len(request.Messages) < 2 {
			t.Error("missing goal and user prompt")
		}
		answer := "Partial progress"
		if calls > 1 {
			last := request.Messages[len(request.Messages)-1].Content
			if last == nil || !strings.HasPrefix(*last, "Continue the current goal") {
				t.Error("continuation lost previous progress")
			}
			answer = "Fixture summarized\n[GOAL_COMPLETE]"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"usage": map[string]int{"prompt_tokens": 20, "completion_tokens": 5},
			"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]string{"role": "assistant", "content": answer}}}})
	}))
	defer server.Close()
	c := &client{server.URL, server.Client()}
	g := &desktopGoalRun{remaining: budget, active: func() bool { return true }}
	answer, usage, _, err := c.completeWorkspaceToolTurn(t.Context(), "openai_chat_completions", "fixture", session, model,
		nil, row.Goal.Objective, nil, false, false, 4, nil, &workspaceToolPolicy{Instructions: row.Goal.Objective, Goal: g}, 4096, nil, nil)
	if err != nil || answer != "Fixture summarized" || g.outcome != "completed" || calls != 2 {
		t.Fatalf("goal did not continue and complete: %q %s %d %v", answer, g.outcome, calls, err)
	}
	settleDesktopGoal(&row, row.Goal, acpTurn{Usage: usage, Started: now, Ended: now.Add(time.Second), goalOutcome: g.outcome})
	if err := l.save(row); err != nil {
		t.Fatal(err)
	}
	reopened, err := readDesktopSession(root, session)
	if err != nil || reopened.Goal == nil || reopened.Goal.Current || reopened.Goal.Status != "completed" || reopened.Goal.Usage.Tokens != 50 {
		t.Fatalf("goal result not persisted: %+v %v", reopened.Goal, err)
	}
	calls = 0
	g = &desktopGoalRun{remaining: 25, active: func() bool { return true }}
	_, _, _, err = c.completeWorkspaceToolTurn(t.Context(), "openai_chat_completions", "fixture", session, model,
		nil, row.Goal.Objective, nil, false, false, 4, nil, &workspaceToolPolicy{Instructions: row.Goal.Objective, Goal: g}, 4096, nil, nil)
	if err != nil || g.outcome != "token_budget" || calls != 1 {
		t.Fatalf("goal exceeded budget: %s %d %v", g.outcome, calls, err)
	}
}
