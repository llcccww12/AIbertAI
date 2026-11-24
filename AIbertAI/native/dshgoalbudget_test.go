package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestDSHGoalBudgetAdmissionAndSettlement(t *testing.T) {
	const session = "11111111-1111-4111-8111-111111111111"
	const goalID = "22222222-2222-4222-8222-222222222222"
	const first = "33333333-3333-4333-8333-333333333333"
	const second = "44444444-4444-4444-8444-444444444444"
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	now := time.Now().UTC()
	row := acpSession{ID: session, Operation: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Model: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", CWD: root.Name(), State: "idle", Created: now, Updated: now, Engine: "dsh", Goal: &desktopGoal{ID: goalID, Status: "active", Current: true, Limits: desktopGoalLimit{Tokens: 10}}}
	live := &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{session: row}}
	if err = live.admitDSHGoalInvocation(session, goalID, first); err != nil {
		t.Fatal(err)
	}
	if err = live.admitDSHGoalInvocation(session, goalID, first); err == nil {
		t.Fatal("replayed invocation admitted")
	}
	if err = live.admitDSHGoalInvocation(session, goalID, second); err == nil {
		t.Fatal("admitted while usage pending")
	}
	persisted, err := readDesktopSession(root, session)
	if err != nil || persisted.Goal == nil || !persisted.Goal.hasPendingDSHInvocation() {
		t.Fatalf("reservation not durable: %+v %v", persisted.Goal, err)
	}
	input, output := int64(3), int64(2)
	usage := modelUsage{Input: &input, Output: &output}
	if err = live.settleDSHGoalInvocation(session, goalID, first, usage); err != nil {
		t.Fatal(err)
	}
	if err = live.settleDSHGoalInvocation(session, goalID, first, usage); err != nil {
		t.Fatal(err)
	}
	output++
	if err = live.settleDSHGoalInvocation(session, goalID, first, usage); err == nil {
		t.Fatal("accepted conflicting usage")
	}
	if live.sessions[session].Goal.Usage.Tokens != 5 {
		t.Fatal("settlement charged twice")
	}
	if err = live.admitDSHGoalInvocation(session, goalID, second); err != nil {
		t.Fatal(err)
	}
	input, output = 3, 2
	if err = live.settleDSHGoalInvocation(session, goalID, second, usage); err != nil {
		t.Fatal(err)
	}
	if got := live.sessions[session].Goal; got.Status != "budget_limited" || got.Reason != "token_budget" || got.Usage.Tokens != 10 {
		t.Fatalf("budget not enforced: %+v", got)
	}
	if err = live.admitDSHGoalInvocation(session, goalID, "55555555-5555-4555-8555-555555555555"); err == nil {
		t.Fatal("admitted after budget")
	}
}

func TestDSHGoalUnknownUsageRequiresReconciliation(t *testing.T) {
	const session = "11111111-1111-4111-8111-111111111111"
	const goalID = "22222222-2222-4222-8222-222222222222"
	const callID = "33333333-3333-4333-8333-333333333333"
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	now := time.Now().UTC()
	row := acpSession{ID: session, Operation: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Model: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", CWD: root.Name(), State: "idle", Created: now, Updated: now, Engine: "dsh", Goal: &desktopGoal{ID: goalID, Status: "active", Current: true, Generation: 1, Limits: desktopGoalLimit{Tokens: 100}}}
	live := &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{session: row}}
	if err = live.admitDSHGoalInvocation(session, goalID, callID); err != nil {
		t.Fatal(err)
	}
	if err = live.settleDSHGoalInvocation(session, goalID, callID, modelUsage{}); err == nil {
		t.Fatal("accepted unknown usage")
	}
	if err = live.admitDSHGoalInvocation(session, goalID, "44444444-4444-4444-8444-444444444444"); err == nil || err.Error() != "goal_usage_reconciliation_required" {
		t.Fatalf("unknown outcome treated as a clean denial: %v", err)
	}
	if err = live.admitDSHGoalInvocation(session, goalID, callID); err == nil || err.Error() != "goal_invocation_already_admitted" {
		t.Fatalf("existing invocation treated as a clean denial: %v", err)
	}
	got, err := readDesktopSession(root, session)
	if err != nil || got.Goal.Reason != "usage_unknown" || !got.Goal.Uncertain {
		t.Fatalf("unknown state: %+v %v", got.Goal, err)
	}
	resume := json.RawMessage(`{"sessionId":"` + session + `","action":"resume","requestId":"55555555-5555-4555-8555-555555555555","expectedGoalId":"` + goalID + `","expectedGeneration":1}`)
	if _, err = live.saveGoalAction(resume, true); err != nil || live.sessions[session].Goal.Status != "reconciling" {
		t.Fatalf("resume did not start reconciliation: %v", err)
	}
	in, out := int64(4), int64(1)
	if err = live.settleDSHGoalInvocation(session, goalID, callID, modelUsage{Input: &in, Output: &out}); err != nil {
		t.Fatal(err)
	}
	if live.sessions[session].Goal.Uncertain {
		t.Fatal("reconciled usage remains uncertain")
	}
}

func TestDSHGoalRestartRequiresExplicitResumeAndReconciliation(t *testing.T) {
	const server = "84ec3d37-ec01-4348-aa35-7e1007c6c731"
	const machine = "c1207fc3-d5c2-4e26-a2f5-1c2dd0f49f65"
	const session = "11111111-1111-4111-8111-111111111111"
	const goalID = "22222222-2222-4222-8222-222222222222"
	const call = "33333333-3333-4333-8333-333333333333"
	for _, tc := range []struct {
		name                      string
		pendingUsage, pendingTool bool
		reason                    string
	}{
		{"clean", false, false, "host_restarted"},
		{"unknown_usage", true, false, "usage_unknown"},
		{"unknown_tool", false, true, "resume_confirmation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile := t.TempDir()
			cwd := profile
			now := time.Now().UTC()
			live, err := openDesktopSessions(profile, server, machine, cwd)
			if err != nil {
				t.Fatal(err)
			}
			g := &desktopGoal{ID: goalID, Objective: "fixture", Status: "active", Current: true, Generation: 1, Limits: desktopGoalLimit{Tokens: 100}}
			if tc.pendingUsage {
				g.Invocations = map[string]dshGoalInvocation{call: {}}
			}
			turn := acpTurn{Operation: goalID, Started: now, GoalCheckpoint: &goalCheckpoint{}}
			if tc.pendingTool {
				turn.ToolRuns = []workspaceToolRun{{Pending: true, DSHCallID: "tool-1"}}
			}
			row := acpSession{ID: session, Operation: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Model: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", CWD: cwd, State: "running", Engine: "dsh", DSHSessionID: session, GoalRun: goalID, Goal: g, Created: now, Updated: now, Turns: []acpTurn{turn}}
			if err = live.save(row); err != nil {
				t.Fatal(err)
			}
			if err = live.store.Close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := openDesktopSessions(profile, server, machine, cwd)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.store.Close()
			row, err = reopened.load(session)
			if err != nil || row.Goal.Status != "paused" || row.Goal.Reason != tc.reason || row.State != "idle" {
				t.Fatalf("bad restored goal: %+v %v", row.Goal, err)
			}
			resume := json.RawMessage(`{"sessionId":"` + session + `","action":"resume","requestId":"55555555-5555-4555-8555-555555555555","expectedGoalId":"` + goalID + `","expectedGeneration":1}`)
			_, err = reopened.saveGoalAction(resume, true)
			if tc.pendingTool {
				if err == nil {
					t.Fatal("resumed unresolved work")
				}
			} else if err != nil {
				t.Fatal(err)
			} else if tc.pendingUsage && reopened.sessions[session].Goal.Status != "reconciling" {
				t.Fatal("resume did not start reconciliation")
			}
		})
	}
}

func TestDSHGoalViewIncludesCurrentAttemptTime(t *testing.T) {
	started := time.Now().Add(-2 * time.Second)
	row := acpSession{Engine: "dsh", State: "running", GoalRun: "operation", Goal: &desktopGoal{Current: true, ActiveSeconds: 7, DSHActiveSince: &started}, Turns: []acpTurn{{Operation: "operation", GoalCheckpoint: &goalCheckpoint{ActiveTime: 3 * time.Second}}}}
	view := goalView(row)
	if view.ActiveSeconds < 12 || view.ActiveSeconds > 13 {
		t.Fatalf("wrong elapsed time: %d", view.ActiveSeconds)
	}
	row.State = "idle"
	if got := goalView(row).ActiveSeconds; got != 10 {
		t.Fatalf("idle goal added running time: %d", got)
	}
	row.State, row.GoalRun = "running", ""
	if got := goalView(row).ActiveSeconds; got != 7 {
		t.Fatalf("unrelated turn added goal time: %d", got)
	}
}

func TestDSHGoalCannotReplaceClearedUnknownOutcome(t *testing.T) {
	const session = "11111111-1111-4111-8111-111111111111"
	const previous = "22222222-2222-4222-8222-222222222222"
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	goal := &desktopGoal{ID: previous, Current: false, Status: "cleared", Uncertain: true}
	live := &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{session: {ID: session, Engine: "dsh", Goal: goal}}}
	create := json.RawMessage(`{"sessionId":"` + session + `","action":"create","requestId":"33333333-3333-4333-8333-333333333333","objective":"Continue work"}`)
	if _, err = live.saveGoalAction(create, true); err == nil {
		t.Fatal("replaced goal with unknown prior outcome")
	}
}

func TestDSHGoalCannotResumeWithoutRecordedObjective(t *testing.T) {
	const session = "11111111-1111-4111-8111-111111111111"
	const goalID = "22222222-2222-4222-8222-222222222222"
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	row := acpSession{ID: session, Engine: "dsh", Goal: &desktopGoal{ID: goalID, Current: true, Status: "active", Objective: "New objective", DSHGoalID: "goal-33333333-3333-4333-8333-333333333333"}}
	live := &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{session: row}}
	host := &relaySessionHost{live: live}
	turn := acpTurn{}
	if err = host.runDSHGoalTurn(context.Background(), nil, nil, row, &turn, true, false, func(map[string]any) error { return nil }); err == nil || err.Error() != "dsh_goal_state_unverified" {
		t.Fatalf("unverified goal accepted: %v", err)
	}
}

func TestDSHGoalCreateRejectsCommandLikeObjectives(t *testing.T) {
	for _, tc := range []struct {
		objective string
		reserved  bool
	}{
		{"pause", true}, {"RESUME", true}, {"Clear", true}, {"edit", true}, {"EDIT report", true}, {"edit\u3000report", true}, {"\ufeffpause\ufeff", true}, {"editor report", false}, {"pause after verification", false},
	} {
		t.Run(tc.objective, func(t *testing.T) {
			if dshReservedGoalObjective(tc.objective) != tc.reserved {
				t.Fatalf("incorrect command classification: %q", tc.objective)
			}
		})
	}
}

func TestDSHGoalRestartBetweenCompletionAndTurnCommit(t *testing.T) {
	const server = "84ec3d37-ec01-4348-aa35-7e1007c6c731"
	const machine = "c1207fc3-d5c2-4e26-a2f5-1c2dd0f49f65"
	const session = "11111111-1111-4111-8111-111111111111"
	const goalID = "22222222-2222-4222-8222-222222222222"
	profile := t.TempDir()
	now := time.Now().UTC()
	live, err := openDesktopSessions(profile, server, machine, profile)
	if err != nil {
		t.Fatal(err)
	}
	row := acpSession{ID: session, Operation: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Model: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", CWD: profile, State: "running", Engine: "dsh", DSHSessionID: session, GoalRun: goalID, Goal: &desktopGoal{ID: goalID, Generation: 1, Status: "completed", DSHPhase: "complete", DSHCommandID: "cmd-test", Limits: desktopGoalLimit{Tokens: 100}}, Created: now, Updated: now, Turns: []acpTurn{{Operation: goalID, Started: now}}}
	if err = live.save(row); err != nil {
		t.Fatal(err)
	}
	if err = live.store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := openDesktopSessions(profile, server, machine, profile)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.store.Close()
	got, err := reopened.load(session)
	if err != nil || !got.Goal.Current || got.Goal.Status != "paused" || got.Goal.Reason != "completion_unverified" || got.State != "idle" {
		t.Fatalf("unfinished commit was not held: %+v %v", got.Goal, err)
	}
}

func TestDSHGoalReportedUsageIncludesCompaction(t *testing.T) {
	a, b, c, d := int64(3), int64(1), int64(5), int64(2)
	goal := &desktopGoal{Usage: desktopGoalUsage{Tokens: 11}, Invocations: map[string]dshGoalInvocation{"one": {Settled: true, Tokens: 4, Input: &a, Output: &b}, "compaction": {Settled: true, Tokens: 7, Input: &c, Output: &d}}}
	usage, err := goalReportedUsage(goal, modelUsage{Input: &a, Output: &b})
	if err != nil || *usage.Input != 8 || *usage.Output != 3 {
		t.Fatalf("compaction omitted: %+v %v", usage, err)
	}
	goal.Invocations["compaction"] = dshGoalInvocation{Settled: true, Tokens: 7}
	if _, err = goalReportedUsage(goal, modelUsage{Input: &a, Output: &b}); err == nil {
		t.Fatal("accepted incomplete legacy total")
	}
	invalid := int64(maxGoalBudget + 1)
	goal.Invocations["compaction"] = dshGoalInvocation{Settled: true, Tokens: 7, Input: &invalid, Output: &d}
	if _, err = goalReportedUsage(goal, modelUsage{Input: &c, Output: &d}); err == nil {
		t.Fatal("accepted malformed invocation usage")
	}
}

func TestDSHGoalCannotEditDuringRecoveryOrAfterCompletion(t *testing.T) {
	const session = "11111111-1111-4111-8111-111111111111"
	const goalID = "22222222-2222-4222-8222-222222222222"
	for _, tc := range []struct{ status, phase string }{{"reconciling", "active"}, {"clearing", "paused"}, {"paused", "complete"}} {
		t.Run(tc.status+tc.phase, func(t *testing.T) {
			goal := &desktopGoal{ID: goalID, Current: true, Status: tc.status, DSHPhase: tc.phase, Generation: 1}
			live := &acpLive{sessions: map[string]acpSession{session: {ID: session, Engine: "dsh", State: "idle", Goal: goal}}}
			edit := json.RawMessage(`{"sessionId":"` + session + `","action":"edit","requestId":"33333333-3333-4333-8333-333333333333","expectedGoalId":"` + goalID + `","expectedGeneration":1,"objective":"Different task"}`)
			if _, err := live.saveGoalAction(edit, true); err == nil {
				t.Fatal("edited goal during completion or reconciliation")
			}
		})
	}
}
