package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestDSHGoalReadProjectionProof(t *testing.T) {
	const goalID = "goal-33333333-3333-4333-8333-333333333333"
	const platformGoal = "44444444-4444-4444-8444-444444444444"
	const invocation = "55555555-5555-4555-8555-555555555555"
	for _, scenario := range []string{"valid", "append_after_ceiling", "large_history", "history_limit", "large_bytes", "projection_capacity", "checkpoint_only", "unexecuted_tool", "unsupported_tool", "unprojected_text", "wrong_usage", "wrong_binding", "wrong_purpose", "pre_binding_collision", "pre_binding_unique", "pre_target_activity", "late_binding", "late_result", "late_assistant", "late_goal_change", "late_gap", "moving_ceiling", "save_failure", "concurrent_change"} {
		t.Run(scenario, func(t *testing.T) {
			root, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			input, output := int64(3), int64(1)
			goal := &desktopGoal{ID: platformGoal, Current: true, Status: "reconciling", DSHGoalID: goalID, DSHQueryOp: historyOperation, DSHCommandID: "command-1", DSHRevision: 1, DSHEventSeq: 6, DSHPhase: "active", RoundsStarted: 1, Usage: desktopGoalUsage{Tokens: 4}, Invocations: map[string]dshGoalInvocation{invocation: {Settled: true, Tokens: 4, Input: &input, Output: &output}}}
			turn := acpTurn{Operation: platformGoal, GoalCheckpoint: &goalCheckpoint{}}
			row := acpSession{ID: historySession, DSHSessionID: historySession, Engine: "dsh", Goal: goal, GoalRun: platformGoal, Turns: []acpTurn{turn}}
			live := &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{historySession: row}}
			rows := goalHistoryFixture()
			call := rows[7]["data"].(map[string]any)
			call["name"] = "read"
			content := []any{map[string]any{"type": "tool-call", "id": "call-1", "name": "read", "arguments": `{"n":9007199254740993}`}}
			if scenario == "unsupported_tool" {
				call["name"] = "write"
				content[0].(map[string]any)["name"] = "write"
			}
			if scenario == "unprojected_text" {
				content = append(content, map[string]any{"type": "text", "text": "Earlier progress"})
			}
			if scenario == "unexecuted_tool" {
				content = append(content, map[string]any{"type": "tool-call", "id": "unexecuted", "name": "write", "arguments": `{}`})
			}
			if scenario == "projection_capacity" {
				for i := 2; i <= 18; i++ {
					content = append(content, map[string]any{"type": "tool-call", "id": fmt.Sprintf("call-%d", i), "name": "read", "arguments": `{}`})
				}
			}
			if scenario == "wrong_usage" {
				input = 99
			}
			if scenario == "wrong_binding" {
				goal.DSHCommandID = "other"
			}
			extra := []historyRow{
				{"type": "aibertai/invocation", "data": map[string]any{"invocationId": invocation, "purpose": "conversation"}},
				{"type": "assistant/message", "data": map[string]any{"turn": 1, "step": 1, "invocationId": invocation, "invocationFinish": "tool-calls", "usage": map[string]any{"inputTokens": 3, "outputTokens": 1}, "message": map[string]any{"id": "assistant-1", "role": "assistant", "source": map[string]any{"kind": "model"}, "content": content}}},
			}
			if scenario == "wrong_purpose" {
				extra[0]["data"].(map[string]any)["purpose"] = "compaction"
			}
			rows = append(rows[:7], append(extra, rows[7:]...)...)
			for i := range rows {
				rows[i]["seq"], rows[i]["time"] = i, i+1
			}
			rows[10]["sourceEventSeqs"] = []int{9}
			if scenario == "pre_binding_collision" || scenario == "pre_binding_unique" {
				prior := invocation
				if scenario == "pre_binding_unique" {
					prior = "66666666-6666-4666-8666-666666666666"
					zero := int64(0)
					goal.Invocations[prior] = dshGoalInvocation{Settled: true, Input: &zero, Output: &zero}
				}
				prefix := []historyRow{
					{"type": "turn/start", "data": map[string]any{"turn": 1}},
					{"type": "aibertai/invocation", "data": map[string]any{"invocationId": prior, "purpose": "conversation"}},
					{"type": "turn/end", "data": map[string]any{"turn": 1}},
				}
				rows = append(prefix, rows...)
				for i := range rows {
					rows[i]["seq"], rows[i]["time"] = i, i+1
				}
				rows[6]["data"].(map[string]any)["commandRunSeq"] = 3
				rows[6]["data"].(map[string]any)["goalSeq"] = 4
				rows[13]["sourceEventSeqs"] = []int{12}
				goal.DSHEventSeq = 9
			}
			if scenario == "pre_target_activity" {
				goalUser := rows[6]
				rows[6] = historyRow{"seq": 6, "type": "user/message", "data": map[string]any{"id": "context", "role": "user", "source": map[string]any{"kind": "time-context"}}}
				rows = append(rows[:11], append([]historyRow{goalUser}, rows[11:]...)...)
				for i := range rows {
					rows[i]["seq"], rows[i]["time"] = i, i+1
				}
				goal.DSHEventSeq = 11
			}
			if scenario == "projection_capacity" {
				payload := strings.Repeat("x", 480<<10)
				rows[10]["data"].(map[string]any)["message"].(map[string]any)["content"] = []any{map[string]any{"type": "text", "text": payload}}
				var extra []historyRow
				for i := 2; i <= 18; i++ {
					id := fmt.Sprintf("call-%d", i)
					seq := 11 + len(extra)
					extra = append(extra,
						historyRow{"type": "tool/call", "data": map[string]any{"turn": 1, "step": 1, "callId": id, "name": "read", "arguments": `{}`}},
						historyRow{"type": "tool/result", "sourceEventSeqs": []int{seq}, "data": map[string]any{"turn": 1, "step": 1, "message": map[string]any{"role": "tool", "toolCallId": id, "source": map[string]any{"kind": "tool", "callId": id}, "content": []any{map[string]any{"type": "text", "text": payload}}, "isError": false}}})
				}
				rows = append(rows[:11], append(extra, rows[11:]...)...)
				for i := range rows {
					rows[i]["seq"], rows[i]["time"] = i, i+1
				}
			}
			if scenario == "late_binding" || scenario == "late_result" || scenario == "late_assistant" {
				source := rows[3]
				if scenario == "late_result" {
					source = rows[10]
				}
				if scenario == "late_assistant" {
					source = rows[8]
				}
				duplicate := historyRow{}
				for k, v := range source {
					duplicate[k] = v
				}
				duplicate["seq"] = len(rows)
				rows = append(rows, duplicate)
			}
			if scenario == "late_goal_change" {
				rows = append(rows, historyRow{"seq": len(rows), "type": "goal/change", "data": map[string]any{"operation": "pause", "goal": map[string]any{"id": goalID, "revision": 2, "phase": "paused"}}})
			}
			if scenario == "large_history" || scenario == "history_limit" {
				count := 10001
				if scenario == "history_limit" {
					count = dshConnectorHistoryMaxEvents
				}
				for range count {
					rows = append(rows, historyRow{"seq": len(rows), "type": "user/message", "data": map[string]any{"id": "time-context", "role": "user", "source": map[string]any{"kind": "time-context"}}})
				}
			}
			if scenario == "large_bytes" {
				payload := strings.Repeat("x", 16<<10)
				for range 2100 {
					rows = append(rows, historyRow{"seq": len(rows), "type": "user/message", "data": map[string]any{"id": "time-context", "role": "user", "source": map[string]any{"kind": "time-context"}, "content": []any{map[string]any{"type": "text", "text": payload}}}})
				}
			}
			if scenario == "checkpoint_only" {
				for _, i := range []int{9, 10} {
					frame, _ := json.Marshal(map[string]any{"sessionId": historySession, "event": rows[i]})
					if _, err := projectDSHTool(&turn, frame); err != nil {
						t.Fatal(err)
					}
				}
				row.Turns[0] = turn
				live.sessions[historySession] = row
			}
			fetch := historyPager(t, rows, "")
			fetchCount := 0
			if scenario == "append_after_ceiling" {
				first := true
				fetch = func(_ context.Context, params map[string]any) (json.RawMessage, error) {
					after := params["afterSeq"].(int64)
					through := int64(len(rows) - 1)
					if fixed, ok := params["throughSeq"]; ok {
						through = fixed.(int64)
					}
					end := min(after+3, through)
					page := append([]historyRow(nil), rows[after+1:end+1]...)
					if first {
						first = false
						duplicate := historyRow{}
						for k, v := range rows[3] {
							duplicate[k] = v
						}
						duplicate["seq"] = len(rows)
						rows = append(rows, duplicate)
					}
					return json.Marshal(map[string]any{"sessionId": historySession, "throughSeq": through, "events": page})
				}
			}
			if scenario == "late_gap" || scenario == "moving_ceiling" {
				base := fetch
				fetch = func(ctx context.Context, params map[string]any) (json.RawMessage, error) {
					raw, err := base(ctx, params)
					if err != nil || params["afterSeq"].(int64) < 8 {
						return raw, err
					}
					var page map[string]json.RawMessage
					if json.Unmarshal(raw, &page) != nil {
						t.Fatal("invalid fixture page")
					}
					if scenario == "moving_ceiling" {
						page["throughSeq"] = json.RawMessage("999")
					} else {
						var items []map[string]json.RawMessage
						if json.Unmarshal(page["events"], &items) != nil || len(items) == 0 {
							t.Fatal("missing fixture events")
						}
						items[0]["seq"] = json.RawMessage("999")
						page["events"], _ = json.Marshal(items)
					}
					return json.Marshal(page)
				}
			}
			if scenario == "large_history" || scenario == "history_limit" || scenario == "large_bytes" {
				fetch = func(_ context.Context, params map[string]any) (json.RawMessage, error) {
					fetchCount++
					after := params["afterSeq"].(int64)
					through := int64(len(rows) - 1)
					if fixed, ok := params["throughSeq"]; ok && fixed != through {
						t.Fatalf("history ceiling moved: %v", fixed)
					}
					pageSize := int64(128)
					if scenario == "large_bytes" {
						pageSize = 32
					}
					end := min(after+pageSize, through)
					return json.Marshal(map[string]any{"sessionId": historySession, "throughSeq": through, "events": rows[after+1 : end+1]})
				}
			}
			if scenario == "save_failure" || scenario == "concurrent_change" {
				base := fetch
				first := true
				fetch = func(ctx context.Context, params map[string]any) (json.RawMessage, error) {
					raw, err := base(ctx, params)
					if first {
						first = false
						if scenario == "save_failure" {
							_ = root.Close()
						} else {
							live.mu.Lock()
							current := live.sessions[historySession]
							changed := *current.Goal
							changed.Version++
							current.Goal = &changed
							live.sessions[historySession] = current
							live.mu.Unlock()
						}
					}
					return raw, err
				}
			}
			err = live.repairDSHGoalReadProjection(t.Context(), fetch, historySession, platformGoal)
			if scenario != "pre_binding_unique" && scenario != "valid" && scenario != "append_after_ceiling" && scenario != "large_history" && scenario != "history_limit" && scenario != "large_bytes" && scenario != "checkpoint_only" {
				invalidPage := scenario == "late_gap" || scenario == "moving_ceiling"
				if err == nil || invalidPage && !errors.Is(err, errDSHInvalidHistory) || scenario == "projection_capacity" && !errors.Is(err, errDSHHistoryTooLarge) || scenario != "save_failure" && scenario != "concurrent_change" && scenario != "projection_capacity" && !invalidPage && !errors.Is(err, errDSHGoalProjectionUnverified) || len(live.sessions[historySession].Turns[0].ToolRuns) != 0 || live.sessions[historySession].Goal.DSHReadProjSeq != nil {
					t.Fatalf("unverified history was projected: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if (scenario == "large_history" || scenario == "history_limit") && fetchCount != (len(rows)+127)/128 {
				t.Fatalf("history was scanned more than once: pages=%d events=%d", fetchCount, len(rows))
			}
			if scenario == "large_bytes" && fetchCount != (len(rows)+31)/32 {
				t.Fatalf("large history was rescanned: pages=%d", fetchCount)
			}
			saved := live.sessions[historySession]
			if len(saved.Turns[0].ToolRuns) != 1 || len(saved.Turns[0].GoalCheckpoint.Runs) != 1 || saved.Goal.DSHReadProjSeq == nil || *saved.Goal.DSHReadProjSeq != map[bool]int64{true: 13, false: 10}[scenario == "pre_binding_unique"] {
				t.Fatalf("incorrect projection: %+v", saved)
			}
			if scenario == "append_after_ceiling" {
				if err = live.repairDSHGoalReadProjection(t.Context(), historyPager(t, rows, ""), historySession, platformGoal); !errors.Is(err, errDSHGoalProjectionUnverified) {
					t.Fatalf("later conflicting prefix accepted: %v", err)
				}
				return
			}
			if scenario == "large_bytes" {
				return
			}
			if err = live.repairDSHGoalReadProjection(t.Context(), fetch, historySession, platformGoal); err != nil {
				t.Fatal(err)
			}
			if len(live.sessions[historySession].Turns[0].ToolRuns) != 1 {
				t.Fatal("duplicate recovery")
			}
		})
	}
}
