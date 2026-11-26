package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// A goal paused mid-run cancels the turn, and the finalize path leaves that turn
// unfinished on purpose so a restart can resume it. While the app keeps running
// nothing else closes it: the window showed "running", the stop key answered
// "unconfirmed", and every later prompt was refused.
func TestCancelledGoalTurnDoesNotStrandTheSession(t *testing.T) {
	const session = "11111111-1111-4111-8111-111111111111"
	const goalOp = "22222222-2222-4222-8222-222222222222"
	const turnOp = "33333333-3333-4333-8333-333333333333"
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	now := time.Now().UTC()
	live := &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{}, operations: map[string]string{}}
	live.sessions[session] = acpSession{ID: session, Operation: goalOp, Model: goalOp, CWD: t.TempDir(), State: "running",
		Created: now, Updated: now, GoalRun: turnOp,
		Goal:  &desktopGoal{ID: goalOp, Objective: "盘点工作区", Status: "paused", Current: true},
		Turns: []acpTurn{{Operation: turnOp, Started: now, Prompt: "盘点工作区"}}}

	if !live.cancelTurn(session) {
		t.Fatal("stop key reported failure with nothing left running")
	}
	row := live.sessions[session]
	if row.State != "idle" || row.GoalRun != "" || row.Turns[0].Ended.IsZero() || !row.Turns[0].Interrupted {
		t.Fatalf("turn was not closed: state=%q goalRun=%q turn=%+v", row.State, row.GoalRun, row.Turns[0])
	}
	// The closed record must still load on the next start.
	if _, err := readDesktopSession(root, session); err != nil {
		t.Fatalf("closed record cannot be restored: %v", err)
	}
}

// The same recovery runs before a prompt is admitted, so the employee never has
// to hit the dead stop key first.
func TestInterruptedTurnIsClosedBeforeTheNextPrompt(t *testing.T) {
	const session = "44444444-4444-4444-8444-444444444444"
	const op = "55555555-5555-4555-8555-555555555555"
	const turnOp = "66666666-6666-4666-8666-666666666666"
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	now := time.Now().UTC()
	cwd := t.TempDir()
	live := &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{}, operations: map[string]string{},
		cwd: cwd, workspaces: &desktopWorkspaces{}}
	live.sessions[session] = acpSession{ID: session, Operation: op, Model: op, CWD: cwd, State: "running",
		Created: now, Updated: now, GoalRun: turnOp,
		Goal:  &desktopGoal{ID: op, Objective: "盘点工作区", Status: "paused", Current: true},
		Turns: []acpTurn{{Operation: turnOp, Started: now, Prompt: "盘点工作区"}}}

	// The report for the closed turn is still pending, which is the ordinary
	// gate for any finished turn. What matters is that the session is no longer
	// "running": before this fix the prompt was refused as unavailable.
	_, _, _, err = live.reserveTurn(session, "77777777-7777-4777-8777-777777777777", "", "继续", nil, nil, false, nil)
	if err == nil || err.Error() != errSessionReportPending.Error() {
		t.Fatalf("prompt gate did not move past the interrupted turn: %v", err)
	}
	if row := live.sessions[session]; row.State != "idle" || row.Turns[0].Ended.IsZero() || !row.Turns[0].Interrupted {
		t.Fatalf("recovered record is inconsistent: state=%q turns=%d", row.State, len(row.Turns))
	}
}

// A goal that failed keeps the attempt id of the turn that failed so the same
// turn can be resumed. Handing that id to the goal runtime made the prompt look
// like a retry of a finished turn, so resuming sent no model request and the
// goal sat active without advancing.
func TestResumedGoalTakesAFreshAttempt(t *testing.T) {
	const session = "88888888-8888-4888-8888-888888888888"
	const goalID = "99999999-9999-4999-8999-999999999999"
	const failedOp = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	const resumeOp = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	now := time.Now().UTC()
	live := &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{}}
	live.sessions[session] = acpSession{ID: session, Operation: goalID, Model: goalID, CWD: t.TempDir(),
		State: "idle", Created: now, Updated: now, GoalRun: failedOp,
		Goal:  &desktopGoal{ID: goalID, Objective: "盘点工作区", Status: "paused", Reason: "execution_failure", Current: true, Generation: 1,
			Limits: desktopGoalLimit{Tokens: 200000000}},
		Turns: []acpTurn{{Operation: failedOp, Started: now, Ended: now.Add(time.Second), Failed: true, Acked: true, Prompt: "盘点工作区"}},
	}
	resume := json.RawMessage(`{"sessionId":"` + session + `","action":"resume","requestId":"` + resumeOp + `","expectedGoalId":"` + goalID + `","expectedGeneration":1}`)
	if _, err := live.saveGoalAction(resume, true); err != nil {
		t.Fatalf("resume refused: %v", err)
	}
	row := live.sessions[session]
	if row.GoalRun != resumeOp {
		t.Fatalf("resumed goal kept attempt %q; the runtime would retry a finished turn", row.GoalRun)
	}
	if row.Goal == nil || row.Goal.Status != "active" {
		t.Fatalf("goal did not resume: %+v", row.Goal)
	}
}
