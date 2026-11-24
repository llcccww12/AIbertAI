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
		{Call: call("write", `{"file_path":"notes.md","content":"x"}`), FileOperation: "created"},
		{Call: call("edit", `{"file_path":"/work/app/index.html","old_string":"a","new_string":"b"}`), FileOperation: "updated"},
		{Call: call("write", `{"file_path":"notes.md","content":"y"}`), FileOperation: "updated"},
		{Call: call("read", `{"file_path":"readme.md"}`)},
		{Call: call("write", `{"file_path":"failed.md","content":"z"}`), Report: reportedTool{Failed: 1}},
	}}
	files := turn.mutatedFiles("/work/app")
	if len(files) != 2 {
		t.Fatalf("files=%v", files)
	}
	encoded, _ := json.Marshal(files)
	if string(encoded) != `[{"operation":"created","path":"notes.md"},{"operation":"updated","path":"index.html"}]` {
		t.Fatalf("unexpected files: %s", encoded)
	}
}
