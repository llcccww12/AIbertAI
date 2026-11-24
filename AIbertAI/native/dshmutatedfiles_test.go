package main

import (
	"encoding/json"
	"testing"
)

func TestAcpTurnMutatedFiles(t *testing.T) {
	call := func(name, args string) *modelToolUse {
		use := &modelToolUse{ID: "call-" + name}
		use.Function.Name, use.Function.Arguments = name, args
		return use
	}
	turn := acpTurn{ToolRuns: []workspaceToolRun{
		{Call: call("write", `{"file_path":"notes.md","content":"x"}`), FileOperation: "created", FileAdded: 1},
		{Call: call("edit", `{"file_path":"/work/app/index.html","old_string":"a","new_string":"b"}`), FileOperation: "updated", FileAdded: 1, FileDeleted: 1},
		{Call: call("write", `{"file_path":"notes.md","content":"y"}`), FileOperation: "updated", FileAdded: 1, FileDeleted: 1},
		{Call: call("read", `{"file_path":"readme.md"}`)},
		{Call: call("write", `{"file_path":"failed.md","content":"z"}`), Report: reportedTool{Failed: 1}},
	}}
	files := turn.mutatedFiles("/work/app")
	if len(files) != 2 {
		t.Fatalf("files=%v", files)
	}
	encoded, _ := json.Marshal(files)
	if string(encoded) != `[{"added":2,"deleted":1,"operation":"created","path":"notes.md"},{"added":1,"deleted":1,"operation":"updated","path":"index.html"}]` {
		t.Fatalf("unexpected files: %s", encoded)
	}
}

func TestWorkspaceLineChanges(t *testing.T) {
	for _, tc := range []struct {
		before, after  string
		added, deleted int
	}{
		{"", "a\nb\n", 2, 0},
		{"a\nb\nc\n", "a\nB\nc\nd\n", 2, 1},
		{"a\nb\n", "", 0, 2},
		{"x\ny\nz", "x\ny\nz", 0, 0},
		{"a\nb\nc\nd", "d\nc\nb\na", 3, 3},
	} {
		added, deleted := workspaceLineChanges(tc.before, tc.after)
		if added != tc.added || deleted != tc.deleted {
			t.Fatalf("%q -> %q: +%d -%d, want +%d -%d", tc.before, tc.after, added, deleted, tc.added, tc.deleted)
		}
	}
}
