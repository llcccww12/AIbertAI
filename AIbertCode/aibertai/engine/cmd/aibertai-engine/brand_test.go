package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestHostBrandMigrationPreservesFilesAndResumes(t *testing.T) {
	root := t.TempDir()
	previous := hostBrandPaths{filepath.Join(root, "old-etc"), filepath.Join(root, "old-data"), filepath.Join(root, "monkeyai-engine.service")}
	current := hostBrandPaths{filepath.Join(root, "new-etc"), filepath.Join(root, "new-data"), filepath.Join(root, "aibertai-engine.service")}
	receipt := filepath.Join(root, "brand")
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	read := func(path, want string) {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil || string(data) != want {
			t.Fatalf("%s: %q %v", path, data, err)
		}
	}
	write(filepath.Join(previous.config, "config.json"), `{"host_id":"host","data_dir":"/external/volume"}`)
	write(filepath.Join(previous.state, "control", "identity.key"), "existing key")
	write(filepath.Join(previous.state, "data", "volumes", "worktree", "notes"), "existing work")
	write(previous.unit, "[Service]\nExecStart="+installedProgram+" serve\n")
	fail := true
	systemctl := func(args ...string) error {
		if fail && args[0] == "daemon-reload" {
			return errors.New("simulated interruption")
		}
		return nil
	}
	if err := moveHostBrand(previous, current, receipt, false, systemctl); err == nil {
		t.Fatal("expected first startup to be interrupted")
	}
	fail = false
	for _, rollback := range []bool{false, false, true, true, false} {
		if err := moveHostBrand(previous, current, receipt, rollback, systemctl); err != nil {
			t.Fatal(err)
		}
		active, inactive := current, previous
		if rollback {
			active, inactive = inactive, active
		}
		read(filepath.Join(active.config, "config.json"), `{"host_id":"host","data_dir":"/external/volume"}`)
		read(filepath.Join(active.state, "control", "identity.key"), "existing key")
		read(filepath.Join(inactive.state, "data", "volumes", "worktree", "notes"), "existing work")
		write(filepath.Join(active.state, "data", "sessions", "new"), "new session")
		read(filepath.Join(inactive.state, "data", "sessions", "new"), "new session")
		if _, err := os.Lstat(inactive.unit); !os.IsNotExist(err) {
			t.Fatalf("duplicate unit: %v", err)
		}
	}
}
