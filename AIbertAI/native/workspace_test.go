package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceWriteKeepsCancelledAndOutOfScopeFilesIntact(t *testing.T) {
	base := t.TempDir()
	directory := filepath.Join(base, "project")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := writeWorkspaceText(t.Context(), root, "notes.txt", "initial"); err != nil {
		t.Fatal(err)
	}
	if err := writeWorkspaceText(t.Context(), root, "notes.txt", "replaced"); err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	for _, attempt := range []struct {
		ctx  context.Context
		path string
	}{{cancelled, "notes.txt"}, {t.Context(), "../outside.txt"}} {
		if err := writeWorkspaceText(attempt.ctx, root, attempt.path, "must remain unwritten"); err == nil {
			t.Fatalf("accepted invalid write to %q", attempt.path)
		}
	}
	outside := filepath.Join(base, "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(directory, "alias.txt")); err == nil {
		if err := writeWorkspaceText(t.Context(), root, "alias.txt", "must remain unwritten"); err == nil {
			t.Fatal("accepted replacing a symlink")
		}
	}
	for file, wanted := range map[string]string{outside: "outside", filepath.Join(directory, "notes.txt"): "replaced"} {
		data, err := os.ReadFile(file)
		if err != nil || string(data) != wanted {
			t.Fatalf("%s: got %q, %v", file, data, err)
		}
	}
}

func TestWorkspaceStatKeepsFileInsideTrustedRoot(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "workspace")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "memo.txt")
	outside := filepath.Join(base, "outside.txt")
	for _, name := range []string{file, outside} {
		if err := os.WriteFile(name, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{file, "memo.txt"} {
		stat, err := statWorkspaceFile(root, name)
		if err != nil || !stat.Exists || stat.Size != 7 || stat.MtimeMS <= 0 {
			t.Fatalf("stat=%+v err=%v", stat, err)
		}
	}
	for _, name := range []string{outside, "../outside.txt", root, "missing.txt"} {
		if _, err := statWorkspaceFile(root, name); err == nil {
			t.Errorf("accepted %s", name)
		}
	}
	if err := os.Symlink(outside, filepath.Join(root, "link.txt")); err == nil {
		if _, err := statWorkspaceFile(root, "link.txt"); err == nil {
			t.Fatal("accepted symlink outside authorized workspace")
		}
	}
}
