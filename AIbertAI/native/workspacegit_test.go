package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceGitFindsStagedUnstagedAndUntracked(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("Git unavailable")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		command := exec.Command(git, args...)
		command.Dir = dir
		if data, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, data)
		}
	}
	write := func(path, value string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, path), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q")
	write("report.txt", "baseline\n")
	run("add", "report.txt")
	run("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-qm", "base")
	write("report.txt", "changed\n")
	write("new.txt", "staged\n")
	run("add", "new.txt")
	write("notes.txt", "untracked\n")
	write(".gitattributes", "report.txt filter=unsafe\n")
	run("config", "filter.unsafe.clean", "exit 9")
	run("config", "filter.unsafe.required", "true")
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	result, err := workspaceGit(t.Context(), root, "_monkeyai/workspace/changes", "")
	if err != nil {
		t.Fatal(err)
	}
	files := result.(map[string]any)["files"].([]map[string]any)
	staged := map[string]bool{}
	for _, file := range files {
		staged[file["path"].(string)] = file["staged"].(bool)
	}
	if len(staged) != 4 || !staged["new.txt"] || staged["report.txt"] || staged["notes.txt"] {
		t.Fatalf("unexpected status: %+v", files)
	}
	patch, err := workspaceGit(t.Context(), root, "_monkeyai/workspace/diff", "report.txt")
	if err != nil || !strings.Contains(patch.(map[string]any)["diff"].(string), "+changed") {
		t.Fatalf("unexpected worktree diff: %v %v", patch, err)
	}
	if _, err := workspaceGit(t.Context(), root, "_monkeyai/workspace/diff", "../other"); err == nil {
		t.Fatal("allowed path outside the workspace")
	}
}
