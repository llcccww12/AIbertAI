package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Like DSH's first-use default workspace, new conversations without a project
// share one visible folder, keep it as the default in the sidebar, and leave
// its files in place when a conversation is deleted.
func TestSharedDefaultWorkspace(t *testing.T) {
	const serverID = "d1bd985c-9f71-4b85-937a-c41b1b104d38"
	const machineID = "d2bd985c-9f71-4b85-937a-c41b1b104d38"
	profile := t.TempDir()
	shared := filepath.Join(t.TempDir(), "Documents", "AIbertAI", "default-workspace")
	live, err := openEmployeeSessions(profile, serverID, machineID, "")
	if err != nil {
		t.Fatal(err)
	}
	defer live.workspaces.root.Close()
	defer live.store.Close()
	live.workspaces.shared = shared
	create := func() acpSession {
		t.Helper()
		op, err := newUUID()
		if err != nil {
			t.Fatal(err)
		}
		id, err := live.createInWorkspace(op, "", "", serverID, expertSelection{}, nil, "", desktopMediaModels{})
		if err != nil {
			t.Fatal(err)
		}
		return live.sessions[id]
	}
	first, second := create(), create()
	for _, row := range []acpSession{first, second} {
		if row.CWD != shared || row.Workspace != "selected" || !row.SharedDefault || row.ProjectID != "" {
			t.Fatalf("session did not open the shared default: %+v", row)
		}
		if meta := row.workspaceMetadata(); meta["isDefaultWorkspace"] != true || meta["cwd"] != shared {
			t.Fatalf("metadata %v", meta)
		}
	}
	// Both conversations reach the same folder through the granted root.
	root, err := live.workspaces.sessionRoot(first)
	if err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile("site/index.html", nil, 0o600); err == nil {
		t.Fatal("expected a missing subdirectory to fail")
	}
	if err := root.Mkdir("site", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile("site/index.html", []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	root.Close()
	other, err := live.workspaces.sessionRoot(second)
	if err != nil {
		t.Fatal(err)
	}
	if data, err := other.ReadFile("site/index.html"); err != nil || string(data) != "ok" {
		t.Fatalf("shared file %q %v", data, err)
	}
	other.Close()

	// Deleting a conversation, even with its workspace, keeps the shared files.
	host := &relaySessionHost{profile: profile, live: live}
	if ok, err := host.trashDesktopSession(first.ID, true); err != nil || !ok {
		t.Fatalf("delete %v %v", ok, err)
	}
	if _, err := os.Stat(filepath.Join(shared, "site", "index.html")); err != nil {
		t.Fatal("shared workspace file removed with a conversation")
	}

	// The record survives a restart and still validates.
	if err := live.store.Close(); err != nil {
		t.Fatal(err)
	}
	live.workspaces.root.Close()
	reloaded, err := openEmployeeSessions(profile, serverID, machineID, "")
	if err != nil {
		t.Fatal(err)
	}
	defer reloaded.workspaces.root.Close()
	defer reloaded.store.Close()
	if row := reloaded.sessions[second.ID]; row.CWD != shared || !row.SharedDefault {
		t.Fatalf("reloaded %+v", row)
	}
}

// If the shared folder cannot be prepared, a conversation still opens in a
// private per-session folder rather than failing.
func TestSharedDefaultWorkspaceFallsBack(t *testing.T) {
	const serverID = "e1bd985c-9f71-4b85-937a-c41b1b104d38"
	const machineID = "e2bd985c-9f71-4b85-937a-c41b1b104d38"
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	live, err := openEmployeeSessions(t.TempDir(), serverID, machineID, "")
	if err != nil {
		t.Fatal(err)
	}
	defer live.workspaces.root.Close()
	defer live.store.Close()
	live.workspaces.shared = filepath.Join(blocked, "default-workspace")
	op, _ := newUUID()
	id, err := live.createInWorkspace(op, "", "", serverID, expertSelection{}, nil, "", desktopMediaModels{})
	if err != nil {
		t.Fatal(err)
	}
	if row := live.sessions[id]; row.Workspace != "default" || row.SharedDefault || row.CWD != live.workspaces.defaultPath(id) {
		t.Fatalf("fallback %+v", row)
	}
}

// As DSH lists its first-use workspace, the shared folder appears once as the
// 默认工作区 project: new conversations without a project and earlier ones in
// the folder are filed under it, and a removed project is not added back.
func TestSharedDefaultWorkspaceProject(t *testing.T) {
	const serverID = "a7bd985c-9f71-4b85-937a-c41b1b104d38"
	const machineID = "a8bd985c-9f71-4b85-937a-c41b1b104d38"
	profile := t.TempDir()
	shared := filepath.Join(t.TempDir(), "default-workspace")
	open := func() *acpLive {
		t.Helper()
		live, err := openEmployeeSessions(profile, serverID, machineID, "")
		if err != nil {
			t.Fatal(err)
		}
		live.workspaces.shared = shared
		return live
	}
	create := func(live *acpLive) acpSession {
		t.Helper()
		op, _ := newUUID()
		id, err := live.createInWorkspace(op, "", "", serverID, expertSelection{}, nil, "", desktopMediaModels{})
		if err != nil {
			t.Fatal(err)
		}
		return live.sessions[id]
	}
	live := open()
	earlier := create(live)
	if !earlier.SharedDefault || earlier.ProjectID != "" {
		t.Fatalf("before seeding %+v", earlier)
	}
	project, err := live.workspaces.seedDefaultProject()
	if err != nil || project == "" {
		t.Fatalf("seed %q %v", project, err)
	}
	listed, _ := live.workspaces.projects("_monkeyai/projects/list", nil)
	if rows := listed.([]employeeProject); len(rows) != 1 || rows[0].ID != project || rows[0].Name != "默认工作区" || rows[0].CWD != shared || !rows[0].Default {
		t.Fatalf("projects %+v", rows)
	}
	if err := live.fileSharedDefaultSessions(project); err != nil {
		t.Fatal(err)
	}
	if row := live.sessions[earlier.ID]; row.ProjectID != project || row.SharedDefault || row.CWD != shared {
		t.Fatalf("earlier session not filed %+v", row)
	}
	if row := create(live); row.ProjectID != project || row.SharedDefault || row.CWD != shared || row.Workspace != "selected" {
		t.Fatalf("new session not filed %+v", row)
	}
	if again, err := live.workspaces.seedDefaultProject(); err != nil || again != project {
		t.Fatalf("reseed %q %v", again, err)
	}
	if _, err := live.workspaces.projects("_monkeyai/projects/remove", json.RawMessage(`{"id":"`+project+`"}`)); err != nil {
		t.Fatal(err)
	}
	live.store.Close()
	live.workspaces.root.Close()

	// After a restart the filed records still validate, and the removed
	// project stays removed; new conversations still share the folder.
	live = open()
	defer live.workspaces.root.Close()
	defer live.store.Close()
	if row := live.sessions[earlier.ID]; row.ProjectID != project {
		t.Fatalf("reloaded %+v", row)
	}
	if again, err := live.workspaces.seedDefaultProject(); err != nil || again != "" {
		t.Fatalf("removed project returned %q %v", again, err)
	}
	if row := create(live); row.ProjectID != "" || !row.SharedDefault || row.CWD != shared {
		t.Fatalf("after removal %+v", row)
	}
}
