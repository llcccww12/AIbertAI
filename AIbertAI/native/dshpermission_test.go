package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func launchedPermission(t *testing.T, cfg *dshRuntimeConfig, ctx context.Context, mode string) []string {
	t.Helper()
	cmd, err := cfg.Command(ctx, acpSession{CWD: "/work", Mode: mode})
	if err != nil {
		t.Fatal(err)
	}
	var modes []string
	for _, e := range cmd.Env {
		if value, ok := strings.CutPrefix(e, "DSH_PERMISSION_MODE="); ok {
			modes = append(modes, value)
		}
	}
	return modes
}

// A local, interactive turn with files:write starts the SDK in the DSH preset
// of the conversation's level; every other launch stays read-only. Records
// from before the three levels keep the level they behaved as.
func TestDSHSessionPermissionPreset(t *testing.T) {
	runtime := dshInstalledRuntime{Executable: "/bin/node", Home: "/tmp/dsh-home",
		Provider: func(context.Context, acpSession) (string, string, error) { return "", "", nil }}
	cfg, err := runtime.config()
	if err != nil {
		t.Fatal(err)
	}
	trusted := context.WithValue(t.Context(), dshTrustedLaunchKey{}, true)
	for _, tc := range []struct {
		name string
		ctx  context.Context
		mode string
		want string
	}{
		{"trusted read-only", trusted, "read-only", "read-only"},
		{"trusted workspace-write", trusted, "workspace-write", "workspace-write"},
		{"trusted full access", trusted, "full-access", "danger-full-access"},
		{"trusted legacy 按需确认", trusted, "auto", "workspace-write"},
		{"trusted legacy 完全信任", trusted, "bypassPermissions", "danger-full-access"},
		{"untrusted full access (relay, goal, no files:write)", t.Context(), "full-access", "read-only"},
		{"untrusted workspace-write", t.Context(), "workspace-write", "read-only"},
	} {
		if got := launchedPermission(t, cfg, tc.ctx, tc.mode); !slices.Equal(got, []string{tc.want}) {
			t.Fatalf("%s: launched with %v, want %s", tc.name, got, tc.want)
		}
	}
	// The ambient environment still cannot choose the preset.
	runtime.Env = []string{"DSH_PERMISSION_MODE=danger-full-access"}
	if _, err := runtime.config(); err == nil {
		t.Fatal("ambient permission mode accepted")
	}
}

// New conversations default to 工作区内修改, and the level can change at any
// time: a DSH session that already has turns queues its SDK preset switch.
func TestDSHPermissionLevelSwitchesDuringConversation(t *testing.T) {
	const id = "f4bd985c-9f71-4b85-937a-c41b1b104d38"
	live := &acpLive{dir: t.TempDir(), cwd: "/work", operations: map[string]string{}, sessions: map[string]acpSession{}}
	op, _ := newUUID()
	created, err := live.createInWorkspace(op, "", "", id, expertSelection{}, nil, "", desktopMediaModels{})
	if err != nil {
		t.Fatal(err)
	}
	if mode := live.sessions[created].permissionMode(); mode != "workspace-write" {
		t.Fatalf("default level %q", mode)
	}
	host := &relaySessionHost{local: &desktopRuntimeAccess{address: "example.com", protocol: "token"}, live: live}
	request := httptest.NewRequest("POST", "/acp", nil)
	request.Header.Set("Sec-WebSocket-Protocol", "monkeyai.v1, token")
	set := func(mode string) error {
		_, err := host.setACPPermissionMode(request, json.RawMessage(`{"sessionId":"`+created+`","modeId":"`+mode+`"}`))
		return err
	}
	if err := set("bypassPermissions"); err == nil {
		t.Fatal("legacy mode accepted")
	}
	if err := set("read-only"); err != nil || live.sessions[created].permissionMode() != "read-only" {
		t.Fatalf("switch before the first message: %v", err)
	}
	row := live.sessions[created]
	row.Engine, row.Turns = "dsh", []acpTurn{{Operation: op}}
	live.sessions[created] = row
	if err := set("full-access"); err != nil {
		t.Fatalf("switch after the first message: %v", err)
	}
	if mode := live.sessions[created].permissionMode(); mode != "full-access" {
		t.Fatalf("level after the switch %q", mode)
	}
	if preset := live.sessions[created].DSHPendingMode; preset != "danger-full-access" {
		t.Fatalf("pending preset %q", preset)
	}
	if err := set("full-access"); err != nil {
		t.Fatalf("reselecting the current level: %v", err)
	}
	// A session without the SDK keeps the level without a queued preset.
	row = live.sessions[created]
	row.Engine, row.DSHPendingMode = "", ""
	live.sessions[created] = row
	if err := set("read-only"); err != nil || live.sessions[created].DSHPendingMode != "" {
		t.Fatalf("non-DSH switch: %v %q", err, live.sessions[created].DSHPendingMode)
	}
}

func TestDSHReadOnlyWriteReason(t *testing.T) {
	if reason := dshWorkspaceToolReason(errWorkspaceReadOnly); !strings.HasPrefix(reason, "read_only:") || !strings.Contains(reason, "sandbox_permissions") {
		t.Fatalf("reason %q", reason)
	}
}

func TestDSHOutsideWorkspaceReason(t *testing.T) {
	reason := dshWorkspaceToolReason(errWorkspacePath)
	if !strings.HasPrefix(reason, "outside_workspace:") || !strings.Contains(reason, "bash") {
		t.Fatalf("reason %q", reason)
	}
}

// An unsettled model call in the gateway journal is the one failure that still
// waits: the next message is refused while that call is the last turn, and a
// model switch is refused while the journal holds it.
func TestDSHPendingModelCallGate(t *testing.T) {
	const id = "f1bd985c-9f71-4b85-937a-c41b1b104d38"
	const op = "f2bd985c-9f71-4b85-937a-c41b1b104d38"
	const prior = "f3bd985c-9f71-4b85-937a-c41b1b104d38"
	journal := t.TempDir()
	last := acpTurn{Operation: prior, Failed: true, FailureCode: "dsh_model_outcome_unknown", Acked: true, Ended: time.Now().UTC()}
	live := &acpLive{dir: t.TempDir(), dsh: &dshRuntimeConfig{JournalRoot: journal},
		sessions: map[string]acpSession{id: {ID: id, Engine: "dsh", State: "error", Turns: []acpTurn{last}}}}
	if live.dshModelCallPending(id) {
		t.Fatal("empty journal reported pending")
	}
	if err := os.MkdirAll(filepath.Join(journal, id), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(journal, id, "pending.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := live.beginTurn(id, op, "", "next", nil, nil); err == nil || err.Error() != "dsh_reconciliation_required" {
		t.Fatalf("pending model call admitted the next message: %v", err)
	}
	// A different failure with the same journal state is not this call's turn.
	row := live.sessions[id]
	row.Turns = []acpTurn{{Operation: prior, Failed: true, FailureCode: "dsh_canvas_outcome_unknown", Acked: true, Ended: time.Now().UTC(), ToolRuns: []workspaceToolRun{{}}}}
	live.sessions[id] = row
	if _, index, _, err := live.beginTurn(id, op, "", "next", nil, nil); err != nil || index != 1 {
		t.Fatalf("tool failure blocked the next message: %d %v", index, err)
	}
}

// write and edit follow DSH's file sandbox for the conversation's level:
// 仅可查看 refuses a plain write and asks for an escalated one, 工作区内修改
// writes inside the workspace without asking, and 完全权限 never asks.
func TestDSHWorkspaceWriteFollowsPermissionLevel(t *testing.T) {
	const serverID = "f5bd985c-9f71-4b85-937a-c41b1b104d38"
	const machineID = "f6bd985c-9f71-4b85-937a-c41b1b104d38"
	live, err := openEmployeeSessions(t.TempDir(), serverID, machineID, "")
	if err != nil {
		t.Fatal(err)
	}
	defer live.workspaces.root.Close()
	defer live.store.Close()
	live.workspaces.shared = filepath.Join(t.TempDir(), "default-workspace")
	host := &relaySessionHost{local: &desktopRuntimeAccess{}, live: live}
	plain := `{"file_path":"a.txt","content":"x"}`
	escalated := `{"file_path":"a.txt","content":"x","sandbox_permissions":"workspace-write","justification":"save a.txt"}`
	// A new file in folders that do not exist yet passes the path check like DSH.
	nested := `{"file_path":"newdir/sub/a.txt","content":"x"}`
	for _, tc := range []struct {
		mode, args string
		want       error
	}{
		{"read-only", plain, errWorkspaceReadOnly},
		{"read-only", escalated, errors.New("tool_approval_unavailable")},
		{"workspace-write", plain, errRelayAccess},
		{"workspace-write", nested, errRelayAccess},
		{"workspace-write", escalated, errors.New("tool_approval_unavailable")},
		{"full-access", escalated, errRelayAccess},
	} {
		op, _ := newUUID()
		id, err := live.createInWorkspace(op, "", "", serverID, expertSelection{}, nil, tc.mode, desktopMediaModels{})
		if err != nil {
			t.Fatal(err)
		}
		turn := acpTurn{Operation: op, ToolRuns: []workspaceToolRun{{DSHCallID: "call-1", Pending: true, Call: &modelToolUse{ID: "call-1"}}}}
		turn.ToolRuns[0].Call.Function.Name, turn.ToolRuns[0].Call.Function.Arguments = "write", tc.args
		session := live.sessions[id]
		session.Engine, session.DSHSessionID, session.Turns = "dsh", id, []acpTurn{turn}
		if err := live.save(session); err != nil {
			t.Fatal(err)
		}
		// No approval channel: a request that asks fails as unavailable; one
		// that does not ask reaches the files:write check, absent here.
		bridge := host.dshWorkspaceBridge(t.Context(), nil, session, &turn)
		_, err = bridge(json.RawMessage(`{"sessionId":"` + id + `","callId":"call-1","name":"write","arguments":` + tc.args + `}`))
		if err == nil || err.Error() != tc.want.Error() {
			t.Fatalf("%s %s: %v, want %v", tc.mode, tc.args, err, tc.want)
		}
	}
	if _, err := os.Stat(filepath.Join(live.workspaces.shared, "newdir")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("folders must not be created before the write is authorized: %v", err)
	}
}
