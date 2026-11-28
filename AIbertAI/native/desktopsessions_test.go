package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestEmployeeSessionsMigratePackageWorkspaceToProfile(t *testing.T) {
	prior := modelID
	modelID = "91c29565-0bb8-4e70-9278-032c847bddcf"
	t.Cleanup(func() { modelID = prior })
	const server = "729437d1-1ee3-4181-b0a6-3d4399016be7"
	const machine = "3825bddd-81da-493a-a040-7544a606a3f4"
	const operation = "1ad55c3b-8b36-44ed-8526-35b22356fe9a"
	profile := t.TempDir()
	old, err := openDesktopSessions(profile, server, machine, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id, err := old.create(operation, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := old.store.Close(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		current, err := openEmployeeSessions(profile, server, machine, "")
		if err != nil {
			t.Fatal(err)
		}
		row, err := current.load(id)
		if err != nil || row.Workspace != "default" || row.CWD != current.workspaces.defaultPath(id) {
			t.Fatalf("stored workspace did not migrate: %+v, %v", row, err)
		}
		retry, err := current.create(operation, "")
		if err != nil || retry != id {
			t.Fatalf("existing operation changed session identity: %q, %v", retry, err)
		}
		info, err := os.Stat(row.CWD)
		if err != nil || !info.IsDir() {
			t.Fatal("expected the new workspace directory in the profile")
		}
		if err := current.workspaces.root.Close(); err != nil {
			t.Fatal(err)
		}
		if err := current.store.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDesktopDeviceIdentitySurvivesRestart(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x51}, 32))
	server := "91c29565-0bb8-4e70-9278-032c847bddcf"
	user := "b37c1ef7-9d57-47b3-bcc1-e62612fa72d7"
	origin := "https://private.example"
	first, err := desktopRelayDevice(key, origin, server, user)
	if err != nil {
		t.Fatal(err)
	}
	again, err := desktopRelayDevice(key, origin, server, user)
	if err != nil {
		t.Fatal(err)
	}
	other, err := desktopRelayDevice(key, "https://other.example", server, user)
	if err != nil {
		t.Fatal(err)
	}
	if !validUUID(first.machine) || first.machine[14] != '4' || first.machine != again.machine ||
		first.boot == again.boot || !first.public.Equal(again.public) {
		t.Fatal("desktop identity changed across boots")
	}
	if first.machine == other.machine || first.public.Equal(other.public) {
		t.Fatal("nonmatching issuer can reproduce the installed device")
	}
}

func TestDesktopRetriesDisconnectedReportBeforeAcceptingNextTurn(t *testing.T) {
	prior := modelID
	modelID = "91c29565-0bb8-4e70-9278-032c847bddcf"
	t.Cleanup(func() { modelID = prior })
	server, machine := "729437d1-1ee3-4181-b0a6-3d4399016be7", "3825bddd-81da-493a-a040-7544a606a3f4"
	create, first, next := "1ad55c3b-8b36-44ed-8526-35b22356fe9a", "98ba1a94-b19b-4e79-9f86-6192f72d5768", "7ad85231-aca4-432a-8299-ae29691cc356"
	live, err := openDesktopSessions(t.TempDir(), server, machine, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer live.store.Close()
	live.client, err = newClient("https://private.example.test")
	if err != nil {
		t.Fatal(err)
	}
	live.auth.Access = "test"
	id, err := live.create(create, "")
	if err != nil {
		t.Fatal(err)
	}
	row, index, _, err := live.beginTurn(id, first, first, "First turn", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	row.State, row.Updated = "idle", now
	row.Turns[index].Ended, row.Turns[index].Answer = now, "A completed answer"
	if err := live.save(row); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := live.beginTurn(id, next, "", "After reconnect", nil, nil); err != errSessionReportPending {
		t.Fatalf("expected the previous report to be pending, got %v", err)
	}
	allowReport := false
	live.client.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, status := "", http.StatusNoContent
		if req.URL.Path == "/api/v1/sessions/"+id+"/turns" {
			if !allowReport {
				status = http.StatusServiceUnavailable
			} else {
				body, status = `{"acked_turn":1}`, http.StatusOK
			}
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	host := &relaySessionHost{live: live, local: &desktopRuntimeAccess{reportMachine: machine}}
	if err := host.retryPendingReport(t.Context(), id); err == nil || live.sessions[id].State == "running" {
		t.Fatalf("failed upload must stay retryable, got %v", err)
	}
	snapshot, err := host.sessionOperation(first, relayAuthorization{MachineID: machine})
	if err != nil || snapshot.(map[string]any)["state"] != "completed" {
		t.Fatalf("completed answer should remain available while reporting is pending: %v, %v", snapshot, err)
	}
	allowReport = true
	// Recovery covers sessions whose model is no longer the default.
	modelID = "5b0b6a1e-2c3d-4e5f-8a9b-0c1d2e3f4a5b"
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		host.superviseDesktopReports(ctx, 10*time.Millisecond)
	}()
	defer func() {
		cancel()
		<-done
	}()
	deadline, stopDeadline := context.WithTimeout(t.Context(), 3*time.Second)
	defer stopDeadline()
	poll := time.NewTicker(5 * time.Millisecond)
	defer poll.Stop()
	for {
		row, err = live.load(id)
		if err == nil && row.Turns[0].Acked {
			break
		}
		select {
		case <-deadline.Done():
			t.Fatal("the idle runtime did not replay the pending report")
		case <-poll.C:
		}
	}
	if row.Turns[0].Answer != "A completed answer" {
		t.Fatal("background report recovery changed the user's answer")
	}
	stored, err := readDesktopSession(live.store.Root, id)
	if err != nil || !stored.Turns[0].Acked {
		t.Fatalf("report acknowledgment was not saved to the journal: %v", err)
	}
	if _, index, replay, err := live.beginTurn(id, next, "", "After reconnect", nil, nil); err != nil || replay || index != 1 {
		t.Fatalf("expected second turn after retry, got index %d, replay %t, error %v", index, replay, err)
	}
}

func TestDesktopSessionRecoversInterruptedTurn(t *testing.T) {
	prior := modelID
	modelID = "91c29565-0bb8-4e70-9278-032c847bddcf"
	t.Cleanup(func() { modelID = prior })
	profile, cwd := t.TempDir(), t.TempDir()
	server, machine := "729437d1-1ee3-4181-b0a6-3d4399016be7", "3825bddd-81da-493a-a040-7544a606a3f4"
	create, prompt := "1ad55c3b-8b36-44ed-8526-35b22356fe9a", "98ba1a94-b19b-4e79-9f86-6192f72d5768"
	live, err := openDesktopSessions(profile, server, machine, cwd)
	if err != nil {
		t.Fatal(err)
	}
	id, err := live.create(create, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := live.beginTurn(id, prompt, "", "test prompt", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := live.store.Close(); err != nil {
		t.Fatal(err)
	}
	restored, err := openDesktopSessions(profile, server, machine, cwd)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.store.Close()
	restoredID, err := restored.create(create, cwd)
	if err != nil || restoredID != id {
		t.Fatalf("creation replay gave %q, %v", restoredID, err)
	}
	session, index, replay, err := restored.beginTurn(id, prompt, "", "test prompt", nil, nil)
	if err != nil || !replay || index != 0 || len(session.Turns) != 1 {
		t.Fatalf("interrupted turn was not replay-safe: index=%d replay=%t err=%v", index, replay, err)
	}
	turn := session.Turns[index]
	if !turn.Failed || turn.Acked || turn.Ended.IsZero() {
		t.Fatal("interrupted turn lost its local failure record or pending report")
	}
}

// Like DSH, an earlier failure never locks the conversation. Only a turn whose
// usage report has not reached the platform waits, and promptACP retries it.
func TestDSHFailedTurnAdmitsNextMessage(t *testing.T) {
	const id = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	const op = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	now := time.Now().UTC()
	for _, tc := range []struct {
		name string
		last acpTurn
		want string
	}{
		{"connector unverified", acpTurn{Failed: true, FailureCode: "dsh_connector_reconciliation_required", Acked: true}, ""},
		{"connector reviewed", acpTurn{Failed: true, FailureCode: "dsh_connector_reconciliation_required", DSHUserReviewed: true, Acked: true}, ""},
		{"canvas outcome unknown", acpTurn{Failed: true, FailureCode: "dsh_canvas_outcome_unknown", Acked: true}, ""},
		{"model outcome unknown", acpTurn{Failed: true, FailureCode: "dsh_model_outcome_unknown", Acked: true}, ""},
		{"ordinary failure", acpTurn{Failed: true, FailureCode: "session_turn_failed", Acked: true}, ""},
		{"runtime exited", acpTurn{Failed: true, FailureCode: "dsh_runtime_exited", Acked: true}, ""},
		{"stop not confirmed", acpTurn{Failed: true, FailureCode: "dsh_reconciliation_required", Acked: true}, ""},
		{"report pending", acpTurn{Failed: true, FailureCode: "dsh_connector_reconciliation_required", DSHReconciled: true}, "session_report_pending"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			last := tc.last
			last.Ended = now
			live := &acpLive{dir: t.TempDir(), sessions: map[string]acpSession{id: {ID: id, Engine: "dsh", State: "error", Turns: []acpTurn{last}}}}
			session, index, replay, err := live.beginTurn(id, op, "", "next", nil, nil)
			if tc.want != "" {
				if err == nil || err.Error() != tc.want {
					t.Fatalf("expected %s, got %v", tc.want, err)
				}
				return
			}
			if err != nil || replay || index != 1 {
				t.Fatalf("expected the next message to start a turn, index=%d replay=%v err=%v", index, replay, err)
			}
			// The failed turn stays on disk unchanged as evidence.
			if prior := session.Turns[0]; !prior.Failed || prior.FailureCode != last.FailureCode || prior.DSHReconciled != last.DSHReconciled {
				t.Fatalf("failed turn was rewritten: %+v", prior)
			}
		})
	}
}

func TestDesktopQueueRecoversEmptyPause(t *testing.T) {
	const session = "11111111-1111-4111-8111-111111111111"
	const message = "22222222-2222-4222-8222-222222222222"
	for _, tc := range []struct {
		name, action string
		messages     []desktopQueuedMessage
		orphaned     bool
		wantCount    int
	}{
		{"add to an old empty paused queue", "add", nil, false, 1},
		{"remove the last paused message", "remove", []desktopQueuedMessage{{ID: message}}, false, 0},
		{"add after an ended turn lost its worker", "add", nil, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			live := &acpLive{dir: t.TempDir(), sessions: map[string]acpSession{
				session: {ID: session, Engine: "dsh", State: "error", Queue: desktopSessionQueue{
					Messages: tc.messages, Paused: true, PauseReason: "session_report_pending",
				}},
			}}
			if tc.orphaned {
				row := live.sessions[session]
				row.State = "running"
				row.Turns = []acpTurn{{Ended: time.Now(), Acked: true, Failed: true}}
				live.sessions[session] = row
			}
			host := &relaySessionHost{live: live, local: &desktopRuntimeAccess{address: "127.0.0.1:12345", protocol: "fixture"}}
			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://127.0.0.1:12345/agent", nil)
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Sec-WebSocket-Protocol", "monkeyai.v1, fixture")
			raw := json.RawMessage(fmt.Sprintf(`{"sessionId":%q,"action":%q,"id":%q,"text":"next"}`, session, tc.action, message))
			if _, err := host.handleQueue(t.Context(), nil, request, raw, nil); err != nil {
				t.Fatal(err)
			}
			row, err := live.load(session)
			if err != nil || row.Queue.Paused || row.Queue.PauseReason != "" || len(row.Queue.Messages) != tc.wantCount {
				t.Fatalf("queue was left paused or changed unexpectedly: %+v, %v", row.Queue, err)
			}
			if tc.wantCount != 0 && !live.claimQueue(row) {
				t.Fatal("a new message in the formerly empty queue could not start")
			}
		})
	}
}

func TestDesktopQueueContinuesOnlyAfterSettledFailure(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	const op = "22222222-2222-4222-8222-222222222222"
	root := t.TempDir()
	live := &acpLive{dsh: &dshRuntimeConfig{JournalRoot: root}}
	row := acpSession{ID: id, Engine: "dsh", State: "error", Turns: []acpTurn{
		{Operation: op, Failed: true, Acked: true, FailureCode: "session_turn_failed"},
	}}
	if !live.canContinueQueueAfterFailure(row, op) {
		t.Fatal("an ordinary failure stopped the next queued message")
	}
	row.Turns[0].Acked = false
	if live.canContinueQueueAfterFailure(row, op) {
		t.Fatal("advanced while the previous report was pending")
	}
	row.Turns[0].Acked = true
	row.Turns[0].FailureCode = "dsh_canvas_outcome_unknown"
	if live.canContinueQueueAfterFailure(row, op) {
		t.Fatal("advanced after an unverified canvas change")
	}
	row.Turns[0].FailureCode = "dsh_model_outcome_unknown"
	if err := os.Mkdir(filepath.Join(root, id), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, id, "pending.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if live.canContinueQueueAfterFailure(row, op) {
		t.Fatal("advanced while the model receipt was pending")
	}
}

func TestDesktopTurnFailureSurvivesReloadAndOperationLookup(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	const op = "22222222-2222-4222-8222-222222222222"
	live := &acpLive{dir: t.TempDir(), sessions: map[string]acpSession{id: {
		ID: id, Engine: "dsh", State: "error", Turns: []acpTurn{{
			Operation: op, ClientID: op, Failed: true, Ended: time.Now(), Acked: true,
			FailureCode: safeDesktopTurnFailure(fmt.Errorf("session resume: %w", errDSHSessionNotFound)),
		}},
	}}}
	if err := live.save(live.sessions[id]); err != nil {
		t.Fatal(err)
	}
	host := &relaySessionHost{live: live}
	result, err := host.sessionState(id)
	if err != nil || result["state"].(map[string]any)["lastError"] != "dsh_session_not_found" {
		t.Fatalf("failed to load the current failure: %v, %v", result, err)
	}
	snapshot, err := host.sessionOperation(op, relayAuthorization{})
	if err != nil || snapshot.(map[string]any)["error"].(map[string]any)["message"] != "dsh_session_not_found" {
		t.Fatalf("failed to look up the current failure: %v, %v", snapshot, err)
	}
	if got := safeDesktopTurnFailure(errors.New("unclassified failure")); got != "session_turn_failed" {
		t.Fatalf("unexpected fallback reason: %q", got)
	}
}

func TestSessionsContinueWithReportsOffline(t *testing.T) {
	t.Run("dsh", func(t *testing.T) { testTurnReportBacklog(t, "dsh") })
	t.Run("cloud-legacy", func(t *testing.T) { testTurnReportBacklog(t, "") })
}

func testTurnReportBacklog(t *testing.T, engine string) {
	t.Helper()
	const id = "11111111-1111-4111-8111-111111111111"
	const model = "22222222-2222-4222-8222-222222222222"
	const first = "33333333-3333-4333-8333-333333333333"
	const second = "44444444-4444-4444-8444-444444444444"
	const third = "55555555-5555-4555-8555-555555555555"
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	now := time.Now().UTC()
	cwd := t.TempDir()
	c, err := newClient("https://private.example.test")
	if err != nil {
		t.Fatal(err)
	}
	live := &acpLive{client: c, cwd: cwd, store: &desktopSessionStore{Root: root},
		reportWake: make(chan struct{}, 1), sessions: map[string]acpSession{}}
	row := acpSession{ID: id, Model: model, Operation: id, Engine: engine, CWD: cwd,
		Created: now, Updated: now, State: "error", Turns: []acpTurn{{
			Operation: first, Model: model, Started: now, Ended: now, Failed: true, FailureCode: "session_turn_failed",
		}}}
	if err = live.save(row); err != nil {
		t.Fatal(err)
	}
	var cursors []string
	offline := true
	c.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		status, body := http.StatusNoContent, ""
		if strings.HasSuffix(req.URL.Path, "/turns") {
			cursor := req.URL.Query().Get("after_turn")
			cursors = append(cursors, cursor)
			if offline {
				status = http.StatusServiceUnavailable
			} else {
				status = http.StatusOK
				previous, _ := strconv.Atoi(cursor)
				body = fmt.Sprintf(`{"acked_turn":%d}`, previous+1)
			}
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	host := &relaySessionHost{live: live, local: &desktopRuntimeAccess{reportMachine: id}}
	if err := host.reportTurnBacklog(t.Context(), id); err == nil {
		t.Fatal("expected the test report to fail")
	}

	row, index, _, err := live.beginTurn(id, second, second, "After failure", nil, func() {})
	if err != nil || index != 1 {
		t.Fatalf("could not start next message while reporting was offline: %d, %v", index, err)
	}
	row.Turns[index].Ended = row.Turns[index].Started
	row.Turns[index].Answer = "Same chat"
	row.State = "idle"
	delete(live.activeTurns, id)
	if err = live.save(row); err != nil {
		t.Fatal(err)
	}
	row, err = readDesktopSession(root, id)
	if err != nil || len(row.Turns) != 2 || row.Turns[0].Acked || row.Turns[1].Acked {
		t.Fatalf("report backlog did not survive reload: %+v, %v", row.Turns, err)
	}
	live.sessions[id] = row
	if _, index, _, err = live.beginTurn(id, third, third, "Keep working", nil, func() {}); err != nil || index != 2 {
		t.Fatalf("could not start third turn: %d, %v", index, err)
	}
	offline = false
	if err := host.reportTurnBacklog(t.Context(), id); err != nil {
		t.Fatal(err)
	}
	row, err = readDesktopSession(root, id)
	if err != nil || row.State != "running" || row.archivedTurns != 2 || row.Turns[1].Answer != "Same chat" ||
		!row.Turns[0].Acked || !row.Turns[1].Acked || row.Turns[2].Acked {
		t.Fatalf("reporting disturbed a live turn: %+v, %v", row, err)
	}
	if !slices.Equal(cursors, []string{"0", "0", "1"}) {
		t.Fatalf("out-of-order report cursors: %v", cursors)
	}
}
