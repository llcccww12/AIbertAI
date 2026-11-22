package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
)

func TestDSHGoalThoughtRequiresExactBindingAndAttempt(t *testing.T) {
	const goal = "11111111-1111-4111-8111-111111111111"
	const query = "22222222-2222-4222-8222-222222222222"
	const invocation = "33333333-3333-4333-8333-333333333333"
	const sdkGoal = "goal-44444444-4444-4444-8444-444444444444"
	for _, tc := range []struct {
		name   string
		mutate func(string, map[string]any)
		want   bool
	}{
		{"verified", nil, true},
		{"wrong query", func(kind string, d map[string]any) {
			if kind == "aibertai/query-binding" {
				d["operationId"] = "55555555-5555-4555-8555-555555555555"
			}
		}, false},
		{"wrong goal", func(kind string, d map[string]any) {
			if kind == "user/message" {
				d["source"] = map[string]any{"kind": "goal", "goalId": "goal-55555555-5555-4555-8555-555555555555", "revision": 1, "round": 1}
			}
		}, false},
		{"wrong invocation", func(kind string, d map[string]any) {
			if kind == "assistant/message" {
				d["invocationId"] = "55555555-5555-4555-8555-555555555555"
			}
		}, false},
		{"wrong step", func(kind string, d map[string]any) {
			if kind == "assistant/message" {
				d["step"] = 2
			}
		}, false},
		{"wrong purpose", func(kind string, d map[string]any) {
			if kind == "aibertai/invocation" {
				d["purpose"] = "compaction"
			}
		}, false},
		{"missing usage", func(kind string, d map[string]any) {
			if kind == "assistant/message" {
				delete(d, "usage")
			}
		}, false},
		{"wrong source", func(kind string, d map[string]any) {
			if kind == "assistant/message" {
				d["message"].(map[string]any)["source"] = map[string]any{"kind": "user"}
			}
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := acpSession{Goal: &desktopGoal{ID: goal, DSHGoalID: sdkGoal, DSHQueryOp: query, DSHRevision: 1, RoundsStarted: 1}}
			tracker := dshGoalThoughtTracker{}
			seq := int64(0)
			emit := func(kind string, data map[string]any) {
				t.Helper()
				if tc.mutate != nil {
					tc.mutate(kind, data)
				}
				raw, _ := json.Marshal(map[string]any{"event": map[string]any{"type": kind, "seq": seq, "data": data}})
				tracker.event(raw, row)
				seq++
			}
			emit("command/run", map[string]any{"commandId": "cmd-1", "name": "goal"})
			emit("goal/change", map[string]any{"operation": "create", "goal": map[string]any{"id": sdkGoal, "phase": "active", "revision": 1}})
			emit("command/done", map[string]any{"commandId": "cmd-1", "kind": "success"})
			emit("aibertai/query-binding", map[string]any{"operationId": query, "commandId": "cmd-1", "commandRunSeq": 0, "goalId": sdkGoal, "goalRevision": 1, "goalSeq": 1})
			emit("turn/start", map[string]any{"turn": 1})
			emit("step/start", map[string]any{"turn": 1, "step": 1})
			emit("user/message", map[string]any{"role": "user", "source": map[string]any{"kind": "goal", "goalId": sdkGoal, "revision": 1, "round": 1}})
			emit("aibertai/invocation", map[string]any{"invocationId": invocation, "purpose": "conversation"})
			emit("assistant/message", map[string]any{"turn": 1, "step": 1, "invocationId": invocation, "invocationFinish": "stop", "usage": map[string]any{"inputTokens": 3, "outputTokens": 1}, "message": map[string]any{"id": "message", "role": "assistant", "source": map[string]any{"kind": "model"}, "content": []any{map[string]any{"type": "reasoning", "text": "verified thought"}, map[string]any{"type": "text", "text": "answer"}}}})
			candidate, ok := tracker.candidates[invocation]
			if ok != tc.want {
				t.Fatalf("candidate=%+v present=%v", candidate, ok)
			}
			if ok && !validDSHGoalThoughtProof(&candidate.proof, goal, candidate.text, candidate.parts) {
				t.Fatal("invalid content proof")
			}
		})
	}
}

func TestDSHGoalThoughtProofRejectsContentChanges(t *testing.T) {
	const goal = "11111111-1111-4111-8111-111111111111"
	parts := []dshMessagePart{{Type: "thought", Text: "safe"}, {Type: "text", End: 6}}
	proof := &dshGoalThoughtProof{GoalID: goal, QueryOperation: "22222222-2222-4222-8222-222222222222", InvocationID: "33333333-3333-4333-8333-333333333333", MessageSeq: 7, ContentSHA256: dshGoalThoughtDigest("answer", parts)}
	if !validDSHGoalThoughtProof(proof, goal, "answer", parts) || validDSHGoalThoughtProof(proof, goal, "edited", parts) {
		t.Fatal("answer binding failed")
	}
	parts[0].Text = "tampered"
	if validDSHGoalThoughtProof(proof, goal, "answer", parts) {
		t.Fatal("tampered thought accepted")
	}
}

func TestDSHGoalThoughtRecoveryRejectsHistoricalConflicts(t *testing.T) {
	const platform = "44444444-4444-4444-8444-444444444444"
	const sdkGoal = "goal-33333333-3333-4333-8333-333333333333"
	const invocation = "55555555-5555-4555-8555-555555555555"
	for _, scenario := range []string{"valid", "prior_invocation", "prior_call", "bad_receipt", "bad_usage", "duplicate_binding"} {
		t.Run(scenario, func(t *testing.T) {
			root, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			in, out := int64(3), int64(1)
			goal := &desktopGoal{ID: platform, Current: true, Status: "reconciling", DSHGoalID: sdkGoal, DSHQueryOp: historyOperation, DSHCommandID: "cmd-1", DSHRevision: 1, RoundsStarted: 1, Invocations: map[string]dshGoalInvocation{invocation: {Settled: true, Tokens: 4, Input: &in, Output: &out}}}
			modelCall := &modelToolUse{ID: "platform-call", Type: "function"}
			modelCall.Function.Name, modelCall.Function.Arguments = "read", "{}"
			run := workspaceToolRun{DSHCallID: "call-1", Call: modelCall}
			turn := acpTurn{Operation: platform, DSHGoalID: platform, ToolRuns: []workspaceToolRun{run}, GoalCheckpoint: &goalCheckpoint{Runs: []workspaceToolRun{run}}}
			row := acpSession{ID: historySession, DSHSessionID: historySession, Engine: "dsh", Goal: goal, GoalRun: platform, Turns: []acpTurn{turn}}
			live := &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{historySession: row}}
			rows := []historyRow{}
			if scenario == "prior_invocation" {
				rows = append(rows, historyRow{"type": "aibertai/invocation", "data": map[string]any{"invocationId": invocation, "purpose": "conversation"}})
			}
			if scenario == "prior_call" {
				rows = append(rows, historyRow{"type": "tool/call", "data": map[string]any{"turn": 0, "step": 0, "callId": "call-1", "name": "read", "arguments": "{}"}})
			}
			base := len(rows)
			binding := map[string]any{"operationId": historyOperation, "commandId": "cmd-1", "commandRunSeq": base, "goalId": sdkGoal, "goalRevision": 1, "goalSeq": base + 1}
			rows = append(rows,
				historyRow{"type": "command/run", "data": map[string]any{"commandId": "cmd-1", "name": "goal"}},
				historyRow{"type": "goal/change", "data": map[string]any{"operation": "create", "goal": map[string]any{"id": sdkGoal, "phase": "active", "revision": 1}}},
				historyRow{"type": "command/done", "data": map[string]any{"commandId": "cmd-1", "kind": "success"}},
				historyRow{"type": "aibertai/query-binding", "data": binding},
				historyRow{"type": "turn/start", "data": map[string]any{"turn": 1}},
				historyRow{"type": "step/start", "data": map[string]any{"turn": 1, "step": 1}},
				historyRow{"type": "user/message", "data": map[string]any{"role": "user", "source": map[string]any{"kind": "goal", "goalId": sdkGoal, "revision": 1, "round": 1}}},
				historyRow{"type": "aibertai/invocation", "data": map[string]any{"invocationId": invocation, "purpose": "conversation"}},
				historyRow{"type": "assistant/message", "data": map[string]any{"turn": 1, "step": 1, "invocationId": invocation, "invocationFinish": "tool-calls", "usage": map[string]any{"inputTokens": 3, "outputTokens": 1}, "message": map[string]any{"id": "m", "role": "assistant", "source": map[string]any{"kind": "model"}, "content": []any{map[string]any{"type": "reasoning", "text": "safe"}, map[string]any{"type": "tool-call", "id": "call-1", "name": "read", "arguments": "{}"}}}}},
				historyRow{"type": "tool/call", "data": map[string]any{"turn": 1, "step": 1, "callId": "call-1", "name": "read", "arguments": "{}"}},
				historyRow{"type": "step/end", "data": map[string]any{"turn": 1, "step": 1}},
				historyRow{"type": "turn/end", "data": map[string]any{"turn": 1}},
			)
			if scenario == "duplicate_binding" {
				rows = append(rows, historyRow{"type": "aibertai/query-binding", "data": binding})
			}
			for i := range rows {
				rows[i]["seq"] = i
			}
			pager := historyPager(t, rows, "")
			calls := 0
			rpc := func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "aibertai/history" {
					return pager(ctx, params.(map[string]any))
				}
				if method != "aibertai/reconcile" {
					return nil, errors.New("unexpected rpc")
				}
				calls++
				seq := base + 8
				if scenario == "bad_receipt" {
					seq++
				}
				input := 3
				if scenario == "bad_usage" {
					input = 9
				}
				return json.Marshal(map[string]any{"state": "committed", "seq": seq, "usageSeq": seq, "usage": map[string]any{"inputTokens": input, "outputTokens": 1}})
			}
			if err := live.recoverDSHGoalThoughtsWith(t.Context(), historySession, platform, rpc); err != nil {
				t.Fatal(err)
			}
			got := live.sessions[historySession].Turns[0].GoalCheckpoint.Runs[0]
			if scenario == "valid" {
				if calls != 1 || !validDSHGoalThoughtProof(got.DSHThoughtProof, platform, got.Preamble, got.DSHParts) || got.DSHParts[0].Text != "safe" {
					t.Fatalf("valid recovery missing: %+v calls=%d", got, calls)
				}
				before, _ := json.Marshal(live.sessions[historySession])
				if err := live.recoverDSHGoalThoughtsWith(t.Context(), historySession, platform, rpc); err != nil {
					t.Fatal(err)
				}
				after, _ := json.Marshal(live.sessions[historySession])
				if calls != 2 || !bytes.Equal(before, after) {
					t.Fatal("repeated recovery changed checkpoint")
				}
			} else if len(got.DSHParts) != 0 || got.DSHThoughtProof != nil {
				t.Fatalf("%s accepted: %+v", scenario, got)
			}
		})
	}
}
