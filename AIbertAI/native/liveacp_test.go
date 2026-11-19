package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestACPCreateKeepsOneSessionAndLoadsTheSavedSnapshot(t *testing.T) {
	dir, workspace := t.TempDir(), t.TempDir()
	store := &acpLive{dir: dir, cwd: workspace, operations: map[string]string{}, sessions: map[string]acpSession{}}
	op, err := newUUID()
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.create(op, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if retried, err := store.create(op, workspace); err != nil || retried != id {
		t.Fatalf("duplicate operation was not idempotent: %v", err)
	}
	if _, err := store.create(op, dir); err == nil {
		t.Fatal("session creation accepted another workspace")
	}
	if saved, err := store.load(id); err != nil || saved.ID != id || saved.Operation != op || saved.State != "idle" {
		t.Fatalf("expected saved snapshot: %+v, %v", saved, err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".json"), []byte(`{"sessionId":"other"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.load(id); err == nil {
		t.Fatal("accepted a snapshot belonging to another session")
	}
}
