package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDesktopJournalMigratesAndRestoresLargeSession(t *testing.T) {
	prior := modelID
	modelID = "91c29565-0bb8-4e70-9278-032c847bddcf"
	t.Cleanup(func() { modelID = prior })
	const server = "729437d1-1ee3-4181-b0a6-3d4399016be7"
	const machine = "3825bddd-81da-493a-a040-7544a606a3f4"
	const sessionID = "1ad55c3b-8b36-44ed-8526-35b22356fe9a"
	const create = "56340dbb-8f3c-432e-a69b-3c58267a5f55"
	profile, cwd := t.TempDir(), t.TempDir()
	open := func() *acpLive {
		t.Helper()
		live, err := openDesktopSessions(profile, server, machine, cwd)
		if err != nil {
			t.Fatal(err)
		}
		return live
	}
	closeStore := func(live *acpLive) {
		t.Helper()
		if err := live.store.Close(); err != nil {
			t.Fatal(err)
		}
	}
	allocate := func() string {
		t.Helper()
		id, err := newUUID()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	live := open()
	originalOp, now := allocate(), time.Now().UTC()
	original := acpSession{ID: sessionID, CWD: cwd, Model: modelID, Operation: create, Created: now, Updated: now, State: "idle",
		Turns: []acpTurn{{Operation: originalOp, Started: now, Ended: now, Prompt: "before", Answer: "old result", Acked: true}}}
	if err := saveDesktopSessionRecord(live.store.Root, sessionID+".json", original); err != nil {
		t.Fatal(err)
	}
	old, err := live.store.ReadFile(sessionID + ".json")
	if err != nil {
		t.Fatal(err)
	}
	if other, err := openDesktopSessions(profile, server, machine, cwd); err == nil {
		other.store.Close()
		t.Fatal("a second writer acquired the same profile")
	}
	closeStore(live)
	live = open()
	defer func() { closeStore(live) }()
	const persistedTurns = 19
	payload := strings.Repeat("x", 480<<10)
	for range persistedTurns - 1 {
		next, index, _, err := live.beginTurn(sessionID, allocate(), "", "after", nil)
		if err != nil {
			t.Fatal(err)
		}
		next.Turns[index].ToolRuns = []workspaceToolRun{{Output: payload}}
		next.Turns[index].Ended = next.Updated
		next.Turns[index].Acked, next.State = true, "idle"
		if err := live.save(next); err != nil {
			t.Fatal(err)
		}
	}
	before, err := live.store.ReadFile(sessionID + ".json")
	if err != nil || !bytes.Equal(old, before) {
		t.Fatalf("legacy record was overwritten: %v", err)
	}
	// Simulate a worker stopping after it persisted a started turn. Startup
	// should restore it as failed and leave it pending for backend reporting.
	interrupted := allocate()
	if _, _, _, err := live.beginTurn(sessionID, interrupted, "", "interrupted", nil); err != nil {
		t.Fatal(err)
	}
	closeStore(live)
	live = open()
	row, err := live.load(sessionID)
	if err != nil || row.archivedTurns != persistedTurns || row.Turns[0].Answer != "old result" ||
		row.Turns[persistedTurns-1].ToolRuns[0].Output != payload || row.State != "error" ||
		len(row.Turns) != persistedTurns+1 || row.Turns[persistedTurns].Acked {
		t.Fatalf("unexpected restored session: turns=%d, state=%q, err=%v", len(row.Turns), row.State, err)
	}
	if _, index, retry, err := live.beginTurn(sessionID, originalOp, "", "before", nil); err != nil || !retry || index != 0 {
		t.Fatalf("migration lost the original operation identity: index=%d retry=%v err=%v", index, retry, err)
	}
	if _, _, _, err := live.beginTurn(sessionID, allocate(), "", "too early", nil); !errors.Is(err, errSessionReportPending) {
		t.Fatalf("expected the interrupted report before accepting another turn, got %v", err)
	}
}

func TestModelContextKeepsWholeToolInteractions(t *testing.T) {
	text := func(value string) *string { return &value }
	dialog := []toolConversationMessage{
		{Role: "user", Content: text(strings.Repeat("old", 40))},
		{Role: "assistant", Content: text("earliest answer")},
		{Role: "user", Content: text("recent request")},
		{Role: "assistant", ToolCalls: []modelToolUse{{ID: "recent-call"}}},
		{Role: "tool", Content: text("recent output"), ToolCallID: "recent-call"},
		{Role: "assistant", Content: text("recent answer")},
		{Role: "user", Content: text("current request")},
		{Role: "assistant", ToolCalls: []modelToolUse{{ID: "current-call"}}},
		{Role: "tool", Content: text("current output"), ToolCallID: "current-call"},
	}
	got := fitModelHistory(dialog, 6, 375)
	if len(got) != 7 || got[0].Content == nil || *got[0].Content != "recent request" || got[len(got)-1].ToolCallID != "current-call" {
		t.Fatalf("tool interaction was split: %+v", got)
	}
	got = fitModelHistory(dialog, 6, 24)
	if len(got) != 3 || got[0].Content == nil || *got[0].Content != "current request" || got[2].ToolCallID != "current-call" {
		t.Fatalf("current tool interaction was lost: %+v", got)
	}
}
