package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestDSHGoalEventsFenceIdentityAndRequireCompletionVerification(t *testing.T) {
	const session = "11111111-1111-4111-8111-111111111111"
	const goalID = "22222222-2222-4222-8222-222222222222"
	const remote = "goal-33333333-3333-4333-8333-333333333333"
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	live := &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{session: {ID: session, Engine: "dsh", Goal: &desktopGoal{ID: goalID, Objective: "Make report", Status: "active", Current: true}}}}
	create := json.RawMessage(`{"sessionId":"` + session + `","event":{"type":"goal/change","seq":4,"data":{"kind":"goal/change","version":1,"operation":"create","goal":{"id":"` + remote + `","revision":1,"objective":"Make report","phase":"active"},"roundsStarted":0}}}`)
	if err = live.recordDSHGoalEvent(session, goalID, create); err != nil {
		t.Fatal(err)
	}
	if err = live.recordDSHGoalEvent(session, goalID, create); err != nil {
		t.Fatal(err)
	}
	round := json.RawMessage(`{"sessionId":"` + session + `","event":{"type":"user/message","seq":5,"data":{"source":{"kind":"goal","goalId":"` + remote + `","revision":1,"round":1}}}}`)
	if err = live.recordDSHGoalEvent(session, goalID, round); err != nil {
		t.Fatal(err)
	}
	if live.sessions[session].Goal.RoundsStarted != 1 {
		t.Fatal("round was not persisted")
	}
	foreign := json.RawMessage(`{"sessionId":"` + session + `","event":{"type":"user/message","seq":6,"data":{"source":{"kind":"goal","goalId":"goal-44444444-4444-4444-8444-444444444444","revision":1,"round":2}}}}`)
	if err = live.recordDSHGoalEvent(session, goalID, foreign); err == nil {
		t.Fatal("accepted foreign goal")
	}
	complete := json.RawMessage(`{"sessionId":"` + session + `","event":{"type":"goal/change","seq":7,"data":{"kind":"goal/change","version":1,"operation":"complete","goal":{"id":"` + remote + `","revision":2,"objective":"Make report","phase":"complete"},"roundsStarted":1}}}`)
	if err = live.recordDSHGoalEvent(session, goalID, complete); err != nil {
		t.Fatal(err)
	}
	g := live.sessions[session].Goal
	if !g.Current || g.Status != "active" || g.DSHPhase != "complete" {
		t.Fatalf("goal closing activity was stopped: %+v", g)
	}
	if err = live.prepareDSHGoalCompletion(session, goalID); err != nil {
		t.Fatal(err)
	}
	g = live.sessions[session].Goal
	if !g.Current || g.Status != "paused" || g.Reason != "verification_required" {
		t.Fatalf("unverified goal marked complete: %+v", g)
	}
}

// The model can create a goal in DSH without the platform. The conversation
// showed nothing before, so the employee saw a goal the model kept describing
// and could not see, pause or dismiss. Mirror it without letting it own the
// session: a paused goal that keeps GoalRun froze every prompt once already.
func TestDSHModelCreatedGoalIsMirroredWithoutTakingTheSession(t *testing.T) {
	const server = "84ec3d37-ec01-4348-aa35-7e1007c6c731"
	const machine = "c1207fc3-d5c2-4e26-a2f5-1c2dd0f49f65"
	const id = "6a1b2c3d-4e5f-4a6b-8c9d-1e2f3a4b5c6d"
	const sessionOp = "d9618877-384d-4ef6-8092-d78242d4b703"
	const promptOp = "1a2b3c4d-5e6f-4a7b-8c9d-2e3f4a5b6c7d"
	const modelGoal = "goal-339eb5af-c2b3-439e-9027-1769eb94ef4f"
	profile := t.TempDir()
	cwd := filepath.Join(profile, "workspace")
	now := time.Now().UTC()
	live, err := openDesktopSessions(profile, server, machine, cwd)
	if err != nil {
		t.Fatal(err)
	}
	defer live.store.Close()
	if err := live.save(acpSession{ID: id, Operation: sessionOp, Engine: "dsh", Model: "6321dabc-2eae-4469-88cf-096d20414cee",
		CWD: cwd, Created: now, Updated: now, State: "idle"}); err != nil {
		t.Fatal(err)
	}
	create := json.RawMessage(`{"sessionId":"` + id + `","event":{"type":"goal/change","seq":4,"data":{"kind":"goal/change","version":1,"operation":"create","goal":{"id":"` + modelGoal + `","revision":1,"objective":"盘点工作区","phase":"active"},"roundsStarted":1}}}`)
	if err := live.adoptDSHGoalEvent(id, create); err != nil {
		t.Fatal(err)
	}
	row, err := live.load(id)
	if err != nil {
		t.Fatal(err)
	}
	if goal := row.Goal; goal == nil || !goal.Observed || goal.Status != "active" || goal.Objective != "盘点工作区" || goal.RoundsStarted != 1 || goal.DSHGoalID != modelGoal {
		t.Fatalf("model goal was not mirrored: %+v", row.Goal)
	}
	if desktopGoalAdvancing(row.Goal) {
		t.Fatal("observed goal claimed the session")
	}
	if _, _, _, err := live.reserveTurn(id, promptOp, promptOp, "你还在做吗", nil, nil, false, nil); err != nil {
		t.Fatalf("observed goal blocked the prompt: %v", err)
	}
	// A repeated event must not move the mirror.
	if err := live.adoptDSHGoalEvent(id, create); err != nil {
		t.Fatal(err)
	}
	if got := live.sessions[id].Goal.Version; got != row.Goal.Version {
		t.Fatalf("duplicate event bumped the version to %d", got)
	}
	complete := json.RawMessage(`{"sessionId":"` + id + `","event":{"type":"goal/change","seq":9,"data":{"kind":"goal/change","version":1,"operation":"complete","goal":{"id":"` + modelGoal + `","revision":2,"objective":"盘点工作区","phase":"complete"},"roundsStarted":3}}}`)
	if err := live.adoptDSHGoalEvent(id, complete); err != nil {
		t.Fatal(err)
	}
	if goal := live.sessions[id].Goal; goal.Status != "completed" || goal.Current || goal.RoundsStarted != 3 {
		t.Fatalf("completion was not mirrored: %+v", goal)
	}
	// A goal the platform created is recorded by its own observer only.
	platform := live.sessions[id]
	platform.Goal = &desktopGoal{ID: sessionOp, Objective: "平台目标", Status: "active", Current: true}
	live.sessions[id] = platform
	if err := live.adoptDSHGoalEvent(id, create); err != nil {
		t.Fatal(err)
	}
	if goal := live.sessions[id].Goal; goal.Observed || goal.ID != sessionOp {
		t.Fatalf("adoption replaced a platform goal: %+v", goal)
	}
}

// Dismissing the mirror is the only control the window has over a goal DSH
// owns; before this the delete request was refused and nothing happened.
func TestDSHModelCreatedGoalCanBeDismissed(t *testing.T) {
	const session = "11111111-1111-4111-8111-111111111111"
	const request = "22222222-2222-4222-8222-222222222222"
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	goal := &desktopGoal{ID: request, Objective: "盘点工作区", Status: "active", Current: true, Generation: 3, Observed: true,
		DSHGoalID: "goal-33333333-3333-4333-8333-333333333333"}
	live := &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{session: {ID: session, Engine: "dsh", Goal: goal}}}
	clear := json.RawMessage(`{"sessionId":"` + session + `","action":"clear","requestId":"44444444-4444-4444-8444-444444444444","expectedGoalId":"` + request + `","expectedGeneration":3}`)
	pause := json.RawMessage(`{"sessionId":"` + session + `","action":"pause","requestId":"55555555-5555-4555-8555-555555555555","expectedGoalId":"` + request + `","expectedGeneration":3}`)
	if _, err := live.saveGoalAction(pause, false); err == nil {
		// DSH has no platform-side pause: reporting "paused" while the SDK kept
		// advancing the goal would be a lie.
		t.Fatal("window paused a goal DSH owns")
	}
	if live.sessions[session].Goal.Status != "active" {
		t.Fatalf("refused pause still changed the record: %+v", live.sessions[session].Goal)
	}
	// The window always asks the background goal runtime to clear, so this must
	// take the same branch production does: DSH is asked to clear the goal.
	view, err := live.saveGoalAction(clear, true)
	if err != nil {
		t.Fatalf("dismiss refused: %v", err)
	}
	if got := view.(map[string]any)["goal"].(*desktopGoal); got.Status != "clearing" || got.DSHPhase == "cleared" {
		t.Fatalf("dismiss did not ask DSH to clear: %+v", got)
	}
	if live.sessions[session].GoalRun != "44444444-4444-4444-8444-444444444444" {
		t.Fatalf("dismiss did not queue the clear: %+v", live.sessions[session])
	}
	// Pressing delete again must not raise a red "the goal changed" warning.
	second, err := live.saveGoalAction(json.RawMessage(`{"sessionId":"`+session+`","action":"clear","requestId":"66666666-6666-4666-8666-666666666666","expectedGoalId":"`+request+`","expectedGeneration":4}`), true)
	if err != nil || second.(map[string]any)["goal"].(*desktopGoal).Status != "clearing" {
		t.Fatalf("second dismiss failed: %v %+v", err, second)
	}
	// DSH confirms the clear with a goal event; only then is the record gone.
	if err := live.recordDSHGoalEvent(session, request, json.RawMessage(`{"sessionId":"`+session+`","event":{"type":"goal/change","seq":5,"data":{"kind":"goal/change","version":1,"operation":"clear","cleared":{"id":"goal-33333333-3333-4333-8333-333333333333","revision":2},"roundsStarted":0}}}`)); err != nil {
		t.Fatal(err)
	}
	if live.sessions[session].Goal.DSHPhase != "cleared" {
		t.Fatalf("DSH clear was not recorded: %+v", live.sessions[session].Goal)
	}
	// Editing still belongs to DSH.
	if view, err := live.saveGoalAction(json.RawMessage(`{"sessionId":"`+session+`","action":"edit","requestId":"`+request+`","objective":"换个目标","expectedGoalId":"`+request+`","expectedGeneration":4}`), false); err == nil {
		t.Fatalf("window edited a goal it does not drive: %+v", view)
	}
	// Dismissing hides the record; a goal DSH keeps advancing must come back
	// rather than run invisibly.
	resumed := json.RawMessage(`{"sessionId":"` + session + `","event":{"type":"goal/change","seq":7,"data":{"kind":"goal/change","version":1,"operation":"resume","goal":{"id":"goal-33333333-3333-4333-8333-333333333333","revision":2,"objective":"盘点工作区","phase":"active"},"roundsStarted":2}}}`)
	if err := live.adoptDSHGoalEvent(session, resumed); err != nil {
		t.Fatal(err)
	}
	if goal := live.sessions[session].Goal; goal.Status != "active" || !goal.Current || goal.RoundsStarted != 2 {
		t.Fatalf("a goal DSH resumed stayed hidden: %+v", goal)
	}
}

// The window merges goal state by state_version alone, so a goal that replaces
// another one must start above it or every update for it is dropped.
func TestDSHModelCreatedGoalOutranksTheRecordItReplaces(t *testing.T) {
	const session = "11111111-1111-4111-8111-111111111111"
	const first = "goal-33333333-3333-4333-8333-333333333333"
	const second = "goal-77777777-7777-4777-8777-777777777777"
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	live := &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{session: {ID: session, Engine: "dsh"}}}
	event := func(id string, seq int64, operation, phase string) json.RawMessage {
		return json.RawMessage(`{"sessionId":"` + session + `","event":{"type":"goal/change","seq":` + strconv.FormatInt(seq, 10) + `,"data":{"kind":"goal/change","version":1,"operation":"` + operation + `","goal":{"id":"` + id + `","revision":1,"objective":"盘点工作区","phase":"` + phase + `"},"roundsStarted":1}}}`)
	}
	if err := live.adoptDSHGoalEvent(session, event(first, 1, "create", "active")); err != nil {
		t.Fatal(err)
	}
	if err := live.adoptDSHGoalEvent(session, event(first, 2, "complete", "complete")); err != nil {
		t.Fatal(err)
	}
	before := live.sessions[session].Goal.Version
	if err := live.adoptDSHGoalEvent(session, event(second, 3, "create", "active")); err != nil {
		t.Fatal(err)
	}
	goal := live.sessions[session].Goal
	if goal.DSHGoalID != second || goal.Status != "active" {
		t.Fatalf("second goal was not mirrored: %+v", goal)
	}
	if goal.Version <= before {
		t.Fatalf("replacement started at version %d, not above %d: the window would keep the old goal", goal.Version, before)
	}
}

// A finished platform goal no longer owns the session, so a goal the model
// opens afterwards must still reach the window.
func TestDSHModelCreatedGoalFollowsAFinishedPlatformGoal(t *testing.T) {
	const session = "11111111-1111-4111-8111-111111111111"
	const platform = "22222222-2222-4222-8222-222222222222"
	const modelGoal = "goal-33333333-3333-4333-8333-333333333333"
	create := json.RawMessage(`{"sessionId":"` + session + `","event":{"type":"goal/change","seq":4,"data":{"kind":"goal/change","version":1,"operation":"create","goal":{"id":"` + modelGoal + `","revision":1,"objective":"盘点工作区","phase":"active"},"roundsStarted":1}}}`)
	for _, status := range []string{"completed", "cleared"} {
		root, err := os.OpenRoot(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		live := &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{session: {ID: session, Engine: "dsh",
			Goal: &desktopGoal{ID: platform, Objective: "平台目标", Status: status, Current: true, Generation: 1, Version: 2}}}}
		if err := live.adoptDSHGoalEvent(session, create); err != nil {
			root.Close()
			t.Fatal(err)
		}
		goal := live.sessions[session].Goal
		root.Close()
		if !goal.Observed || goal.DSHGoalID != modelGoal {
			t.Fatalf("platform goal %q kept the model's goal hidden: %+v", status, goal)
		}
	}
	// A platform goal that can still be resumed keeps the session.
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	live := &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{session: {ID: session, Engine: "dsh",
		Goal: &desktopGoal{ID: platform, Objective: "平台目标", Status: "paused", Current: true, Generation: 1, Version: 2}}}}
	if err := live.adoptDSHGoalEvent(session, create); err != nil {
		t.Fatal(err)
	}
	if goal := live.sessions[session].Goal; goal.Observed {
		t.Fatalf("mirror replaced a resumable platform goal: %+v", goal)
	}
}
