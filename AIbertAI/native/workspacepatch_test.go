package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func patchFixtureFiles(t *testing.T, root *os.Root, files map[string]string) {
	t.Helper()
	for filename, text := range files {
		if err := root.WriteFile(filepath.FromSlash(filename), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWorkspacePatchUpdatesMovesAddsAndDeletes(t *testing.T) {
	base, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer base.Close()
	dir, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	workspace := desktopWorkspaces{root: base}
	row := acpSession{ID: "4992201e-7b78-4f9e-a4fb-5bad992a9e15", CWD: dir.Name()}
	const operation = "bb6d45dd-8bbf-4cc5-8112-4ef2fd7868fe"
	patchFixtureFiles(t, dir, map[string]string{"setting.txt": "header\r\nkeep\r\nold\r\nlast\r\n", "discard.txt": "discard"})
	patch := `*** Begin Patch
*** Update File: setting.txt
*** Move to: moved/settings.txt
@@ header
 keep
-old
+new
 last
*** End of File
*** Add File: release/notes/intro.txt
+ready
*** Delete File: discard.txt
*** End Patch`
	files, err := parseWorkspacePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := prepareWorkspacePatch(dir, row, operation, files)
	if err != nil {
		t.Fatal(err)
	}
	if err := workspace.applyPatch(t.Context(), dir, plan); err != nil {
		t.Fatal(err)
	}
	for filename, want := range map[string]string{"moved/settings.txt": "header\r\nkeep\r\nnew\r\nlast\r\n", "release/notes/intro.txt": "ready\n"} {
		result, err := dir.ReadFile(filepath.FromSlash(filename))
		if err != nil || string(result) != want {
			t.Fatalf("%s: %q, %v", filename, result, err)
		}
	}
	for _, filename := range []string{"setting.txt", "discard.txt"} {
		if _, err := dir.Lstat(filename); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected %s removed: %v", filename, err)
		}
	}
	if _, err := base.Lstat(workspacePatchJournalFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("patch journal was not cleaned: %v", err)
	}
}

func TestWorkspacePatchReturnsBeforeMutatingOnConflict(t *testing.T) {
	base, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer base.Close()
	dir, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	row := acpSession{ID: "4992201e-7b78-4f9e-a4fb-5bad992a9e15", CWD: dir.Name()}
	const operation = "bb6d45dd-8bbf-4cc5-8112-4ef2fd7868fe"
	patchFixtureFiles(t, dir, map[string]string{"safe.txt": "old\n", "duplicate.txt": "x\nx\n"})
	tests := []struct {
		text string
		want error
	}{
		{"*** Add File: extra.txt\n+first\n*** Update File: safe.txt\n@@\n-bad\n+new\n", errors.New("workspace_patch_context_not_found")},
		{"*** Add File: extra.txt\n+first\n*** Update File: duplicate.txt\n@@\n-x\n+y\n", errors.New("workspace_patch_ambiguous_context")},
		{"*** Add File: ../extra.txt\n+first\n", errPatchConflict},
	}
	for _, tc := range tests {
		files, err := parseWorkspacePatch("*** Begin Patch\n" + tc.text + "*** End Patch")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := prepareWorkspacePatch(dir, row, operation, files); err == nil || err.Error() != tc.want.Error() {
			t.Fatalf("preflight error = %v, want %v", err, tc.want)
		}
	}
	files, err := parseWorkspacePatch("*** Begin Patch\n*** Add File: extra.txt\n+first\n*** Delete File: safe.txt\n*** End Patch")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := prepareWorkspacePatch(dir, row, operation, files)
	if err != nil {
		t.Fatal(err)
	}
	patchFixtureFiles(t, dir, map[string]string{"safe.txt": "edit made during approval\n"})
	workspace := desktopWorkspaces{root: base}
	if err := workspace.applyPatch(t.Context(), dir, plan); !errors.Is(err, errPatchConflict) {
		t.Fatalf("changed target was accepted: %v", err)
	}
	if _, err := dir.Lstat("extra.txt"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("first file changed before the conflict was noticed: %v", err)
	}
	result, err := dir.ReadFile("safe.txt")
	if err != nil || string(result) != "edit made during approval\n" {
		t.Fatalf("conflict did not preserve the externally edited file: %q %v", result, err)
	}
}

func TestWorkspacePatchRecoversAfterInterruptedCommit(t *testing.T) {
	oldModel := modelID
	modelID = "dd628a82-5cdd-4b15-a21c-ec84e681c420"
	t.Cleanup(func() { modelID = oldModel })
	profile := t.TempDir()
	open := func() *acpLive {
		t.Helper()
		live, err := openEmployeeSessions(profile, "4992201e-7b78-4f9e-a4fb-5bad992a9e15",
			"bb6d45dd-8bbf-4cc5-8112-4ef2fd7868fe", "")
		if err != nil {
			t.Fatal(err)
		}
		return live
	}
	live := open()
	closeStore := func() {
		t.Helper()
		if err := errors.Join(live.workspaces.root.Close(), live.store.Close()); err != nil {
			t.Fatal(err)
		}
	}
	id, err := live.create("fbf14bee-2398-47be-a398-53703866e11b", "")
	if err != nil {
		t.Fatal(err)
	}
	row, err := live.load(id)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := live.workspaces.sessionRoot(row)
	if err != nil {
		t.Fatal(err)
	}
	patchFixtureFiles(t, dir, map[string]string{"safe.txt": "old\n"})
	files, err := parseWorkspacePatch("*** Begin Patch\n*** Update File: safe.txt\n@@\n-old\n+new\n*** Add File: later.txt\n+later\n*** End Patch")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := prepareWorkspacePatch(dir, row, "fe8bed87-2c00-48af-ae08-b56daeb36237", files)
	if err != nil {
		t.Fatal(err)
	}
	plan.FilesStarted = 1
	if err := live.workspaces.savePatchPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := setWorkspacePatchFile(dir, plan.Changes[0], workspacePatchTemp(plan.Operation, 0, false), false); err != nil {
		t.Fatal(err)
	}
	dir.Close()
	closeStore()
	live = open()
	defer closeStore()
	dir, err = live.workspaces.sessionRoot(row)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	result, err := dir.ReadFile("safe.txt")
	if err != nil || string(result) != "old\n" {
		t.Fatalf("restart failed to restore original bytes: %q %v", result, err)
	}
	if _, err := dir.Lstat("later.txt"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("restart applied an uncommitted file: %v", err)
	}
}
