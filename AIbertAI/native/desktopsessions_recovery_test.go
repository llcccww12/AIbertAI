package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Losing every conversation because one record cannot be read is not a recovery
// path the employee can act on. Start without it, keep the file, and move it
// aside so the next launch does not trip over it again.
func TestUnreadableSessionRecordDoesNotBlockStartup(t *testing.T) {
	const server = "84ec3d37-ec01-4348-aa35-7e1007c6c731"
	const machine = "c1207fc3-d5c2-4e26-a2f5-1c2dd0f49f65"
	const good = "11111111-1111-4111-8111-111111111111"
	const broken = "22222222-2222-4222-8222-222222222222"
	const operation = "33333333-3333-4333-8333-333333333333"

	profile := t.TempDir()
	cwd := filepath.Join(profile, "workspace")
	if err := os.MkdirAll(cwd, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	live, err := openDesktopSessions(profile, server, machine, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if err := live.save(acpSession{ID: good, Operation: operation, Model: operation, CWD: cwd,
		Created: now, Updated: now, State: "idle"}); err != nil {
		t.Fatal(err)
	}
	sessions := filepath.Join(profile, "source-runtime", server, machine, "sessions")
	if err := os.WriteFile(filepath.Join(sessions, broken+".json"), []byte(`{"id":"`+broken+`"`), 0o600); err != nil {
		t.Fatal(err)
	}
	live.store.Close()

	restored, err := openDesktopSessions(profile, server, machine, cwd)
	if err != nil {
		t.Fatalf("one unreadable record stopped the host: %v", err)
	}
	defer restored.store.Close()
	if len(restored.sessions) != 1 || restored.sessions[good].ID != good {
		t.Fatalf("the readable conversation was not restored: %d sessions", len(restored.sessions))
	}
	entries, err := os.ReadDir(sessions)
	if err != nil {
		t.Fatal(err)
	}
	var kept bool
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), broken+".json.corrupt-") {
			kept = true
		}
	}
	if !kept {
		t.Fatal("the unreadable record was not kept for repair")
	}
}
