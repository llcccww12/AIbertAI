package main

import (
	"encoding/json"
	"os"
	"testing"
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
