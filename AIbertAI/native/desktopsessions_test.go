package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"os"
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
