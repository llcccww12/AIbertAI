package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestDSHGoalCompletionLargeHistory(t *testing.T) {
	const operation = "44444444-4444-4444-8444-444444444444"
	const invocation = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	const goalID = "goal-22222222-2222-4222-8222-222222222222"
	events := make([]dshHistoryEvent, 0, 10100)
	add := func(kind string, data any) int64 {
		b, _ := json.Marshal(data)
		seq := int64(len(events))
		events = append(events, dshHistoryEvent{Seq: seq, Type: kind, Data: b})
		return seq
	}
	add("command/run", map[string]any{"commandId": "cmd-1", "name": "goal"})
	add("goal/change", map[string]any{"operation": "create", "goal": map[string]any{"id": goalID, "revision": 1, "phase": "active"}})
	add("command/done", map[string]any{"commandId": "cmd-1", "kind": "success"})
	add("aibertai/query-binding", map[string]any{"operationId": operation, "commandId": "cmd-1", "commandRunSeq": 0, "goalId": goalID, "goalRevision": 1, "goalSeq": 1})
	for i := 0; i < 10100; i++ {
		add("user/message", map[string]any{"round": i})
	}
	goalSeq := add("goal/change", map[string]any{"operation": "complete", "goal": map[string]any{"id": goalID, "revision": 2, "phase": "complete"}})
	assistantSeq := add("assistant/message", map[string]any{"turn": 1, "invocationId": invocation, "invocationFinish": "stop", "message": map[string]any{"content": []any{map[string]any{"type": "text", "text": "finished"}}}})
	endSeq := add("turn/end", map[string]any{"turn": 1, "reason": map[string]any{"kind": "completed"}})
	add("aibertai/query-completed", map[string]any{"commandId": "cmd-1", "goalId": goalID, "goalRevision": 2, "goalSeq": goalSeq, "assistantSeq": assistantSeq, "invocationId": invocation, "turn": 1, "turnEndSeq": endSeq, "finishReason": "completed"})
	zero := int64(0)
	row := acpSession{GoalRun: operation, Goal: &desktopGoal{DSHGoalID: goalID, DSHCommandID: "cmd-1", DSHQueryOp: operation, DSHRevision: 2, DSHEventSeq: goalSeq, Invocations: map[string]dshGoalInvocation{invocation: {Settled: true, Input: &zero, Output: &zero}}}, Turns: []acpTurn{{Operation: operation}}}
	pages := 0
	read := func(after int64, through *int64) (dshHistoryPage, error) {
		pages++
		ceiling := int64(len(events) - 1)
		if through != nil {
			ceiling = *through
		} else {
			// A later append must not move the first page's fixed watermark.
			add("goal/change", map[string]any{"operation": "clear"})
		}
		if after > ceiling || ceiling >= int64(len(events)) {
			return dshHistoryPage{}, fmt.Errorf("invalid range")
		}
		end := min(after+128, ceiling)
		return dshHistoryPage{Through: ceiling, Events: events[after+1 : end+1]}, nil
	}
	text, err := verifyDSHGoalCompletion(row, read)
	if err != nil || text != "finished" || pages < 80 {
		t.Fatalf("large history: text=%q pages=%d err=%v", text, pages, err)
	}
	events = events[:len(events)-1]
	events[10000].Type = "goal/change"
	if _, err := verifyDSHGoalCompletion(row, read); err == nil {
		t.Fatal("accepted a goal change late in the paginated history")
	}
}
