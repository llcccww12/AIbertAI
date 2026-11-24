package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestPresentedFilesUseScopedDSHEventAndCompleteTurn(t *testing.T) {
	cwd := t.TempDir()
	s := acpSession{Engine: "dsh", CWD: cwd, Turns: []acpTurn{{}}}
	turn := &s.Turns[0]
	for i, n := range []int64{1, 2, 2} {
		name := []string{"上一轮.docx", "报告.docx", "报告.docx"}[i]
		description := []string{"", "第一版", "修改版"}[i]
		event, err := json.Marshal(map[string]any{"turn": n, "callId": "workflow:ptc:3", "files": []dshPresentedFile{
			{Path: filepath.Join(cwd, name), Description: description},
			{Path: filepath.Join(filepath.Dir(cwd), "another-session.docx")},
		}})
		if err != nil {
			t.Fatal(err)
		}
		turn.recordDeliverables(event, 2)
	}
	if _, ok := s.responseMessage(0, map[string]any{})["dshDeliverables"]; ok {
		t.Fatal("exposed deliverables before turn end")
	}
	turn.Ended = time.Now()
	got, _ := s.responseMessage(0, map[string]any{})["dshDeliverables"].([]dshPresentedFile)
	if len(got) != 1 || got[0] != (dshPresentedFile{Path: "报告.docx", Description: "修改版"}) {
		t.Fatalf("deliverables: %+v", got)
	}
}

func TestPresentedFilesReadOlderSuccessfulResult(t *testing.T) {
	cwd := t.TempDir()
	call := &modelToolUse{}
	call.Function.Name = "present"
	call.Function.Arguments = `{"files":[{"path":"难忘的一天.docx","description":"约400字作文"}]}`
	run := workspaceToolRun{Call: call, DSHCallID: "sdk-present", Output: "Presented " + filepath.Join(cwd, "难忘的一天.docx")}
	for _, tc := range []struct {
		name, output string
		pending      bool
		failed       int
		count        int
	}{
		{"success", run.Output, false, 0, 1}, {"unconfirmed", "", false, 0, 0}, {"incorrect", "Presented other.docx", false, 0, 0}, {"pending", run.Output, true, 0, 0}, {"failed", run.Output, false, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			saved := run
			saved.Output, saved.Pending, saved.Report.Failed = tc.output, tc.pending, tc.failed
			files := (acpTurn{ToolRuns: []workspaceToolRun{saved}}).presentedFiles(cwd)
			if len(files) != tc.count || len(files) > 0 && files[0].Path != "难忘的一天.docx" {
				t.Fatalf("deliverables: %+v", files)
			}
		})
	}
}
