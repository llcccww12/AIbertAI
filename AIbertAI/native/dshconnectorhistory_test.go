package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

const historySession = "11111111-1111-4111-8111-111111111111"
const historyOperation = "22222222-2222-4222-8222-222222222222"

type historyRow map[string]any

func historyFixture() []historyRow {
	user := map[string]any{"id": "message-1", "role": "user", "source": map[string]any{"kind": "user", "aibertaiOperationId": historyOperation}, "content": []any{map[string]any{"type": "text", "text": "prompt"}}}
	rows := []historyRow{
		{"type": "agent/inbox/spliced", "data": map[string]any{"target": "next-turn", "start": 0, "inserted": []any{user}}},
		{"type": "turn/start", "data": map[string]any{"turn": 1}},
		{"type": "step/start", "data": map[string]any{"turn": 1, "step": 1}},
		{"type": "user/message", "data": user},
		{"type": "tool/call", "data": map[string]any{"turn": 1, "step": 1, "callId": "call-1", "name": "connector_test", "arguments": `{"n":9007199254740993}`}},
		{"type": "tool/result", "sourceEventSeqs": []int{4}, "data": map[string]any{"turn": 1, "step": 1, "message": map[string]any{"role": "tool", "toolCallId": "call-1", "source": map[string]any{"kind": "tool", "callId": "call-1"}, "content": []any{map[string]any{"type": "text", "text": "result"}}, "isError": false}}},
		{"type": "step/end", "data": map[string]any{"turn": 1, "step": 1}},
		{"type": "turn/end", "data": map[string]any{"turn": 1}},
	}
	for i := range rows {
		rows[i]["seq"] = i
		rows[i]["time"] = i + 1
	}
	return rows
}
func historyPager(t *testing.T, rows []historyRow, mode string) dshConnectorHistoryFetch {
	t.Helper()
	return func(_ context.Context, params map[string]any) (json.RawMessage, error) {
		after := params["afterSeq"].(int64)
		through := int64(len(rows) - 1)
		if after >= 0 {
			if params["throughSeq"] != through {
				t.Errorf("ceiling missing or changed: %+v", params)
			}
		}
		if mode == "rpc_error" {
			return nil, errors.New("history unavailable")
		}
		start := int(after + 1)
		end := min(start+3, len(rows))
		page := append([]historyRow(nil), rows[start:end]...)
		if mode == "gap" && after >= 0 && len(page) > 0 {
			copied := historyRow{}
			for k, v := range page[0] {
				copied[k] = v
			}
			copied["seq"] = after + 2
			page[0] = copied
		}
		if mode == "stall" && after >= 0 {
			page = nil
		}
		if mode == "ceiling" && after >= 0 {
			through++
		}
		sid := historySession
		if mode == "session" {
			sid = "other"
		}
		return json.Marshal(map[string]any{"sessionId": sid, "throughSeq": through, "events": page})
	}
}
func matchFixture(t *testing.T, rows []historyRow, mode string) (json.RawMessage, error) {
	t.Helper()
	return dshMatchConnectorHistory(t.Context(), historySession, historyOperation, "call-1", "connector_test", json.RawMessage(`{"n":9007199254740993}`), historyPager(t, rows, mode))
}
func TestDSHConnectorHistoryReconcile(t *testing.T) {
	rows := historyFixture()
	raw, err := matchFixture(t, rows, "")
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		SessionID string
		Event     struct {
			Seq  int
			Type string
		}
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.SessionID != historySession || envelope.Event.Seq != 5 || envelope.Event.Type != "tool/result" {
		t.Fatalf("wrong result: %s", raw)
	}
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err = dshConnectorDispatch(root, historySession, historyOperation, "call-1"); err != nil {
		t.Fatal(err)
	}
	if err = dshConnectorRecordResult(root, historySession, historyOperation, "call-1", "result"); err != nil {
		t.Fatal(err)
	}
	read := func(session, operation, call string) (json.RawMessage, error) {
		return dshMatchConnectorHistory(t.Context(), session, operation, call, "connector_test", json.RawMessage(`{"n":9007199254740993}`), historyPager(t, rows, ""))
	}
	if err = dshConnectorReconcile(root, historySession, historyOperation, "call-1", read); err != nil {
		t.Fatal(err)
	}
	if err = dshConnectorAssertSettled(root, historySession); err != nil {
		t.Fatal(err)
	}
}

func goalHistoryFixture() []historyRow {
	goalID := "goal-33333333-3333-4333-8333-333333333333"
	rows := []historyRow{
		{"type": "command/run", "data": map[string]any{"commandId": "command-1", "name": "goal"}},
		{"type": "goal/change", "data": map[string]any{"operation": "create", "goal": map[string]any{"id": goalID, "revision": 1, "phase": "active"}}},
		{"type": "command/done", "data": map[string]any{"commandId": "command-1", "kind": "success"}},
		{"type": "aibertai/query-binding", "data": map[string]any{"operationId": historyOperation, "commandId": "command-1", "commandRunSeq": 0, "goalId": goalID, "goalRevision": 1, "goalSeq": 1}},
		{"type": "turn/start", "data": map[string]any{"turn": 1}},
		{"type": "step/start", "data": map[string]any{"turn": 1, "step": 1}},
		{"type": "user/message", "data": map[string]any{"id": "goal-message", "role": "user", "source": map[string]any{"kind": "goal", "goalId": goalID, "revision": 1, "round": 1}}},
		{"type": "tool/call", "data": map[string]any{"turn": 1, "step": 1, "callId": "call-1", "name": "connector_test", "arguments": `{"n":9007199254740993}`}},
		{"type": "tool/result", "sourceEventSeqs": []int{7}, "data": map[string]any{"turn": 1, "step": 1, "message": map[string]any{"role": "tool", "toolCallId": "call-1", "source": map[string]any{"kind": "tool", "callId": "call-1"}, "content": []any{map[string]any{"type": "text", "text": "result"}}, "isError": false}}},
		{"type": "step/end", "data": map[string]any{"turn": 1, "step": 1}},
		{"type": "turn/end", "data": map[string]any{"turn": 1}},
	}
	for i := range rows {
		rows[i]["seq"], rows[i]["time"] = i, i+1
	}
	return rows
}

func TestDSHGoalConnectorHistoryRequiresExactBinding(t *testing.T) {
	if _, err := matchFixture(t, goalHistoryFixture(), ""); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		mutate func([]historyRow) []historyRow
	}{
		{"wrong_operation", func(r []historyRow) []historyRow { r[3]["data"].(map[string]any)["operationId"] = "other"; return r }},
		{"wrong_command", func(r []historyRow) []historyRow { r[3]["data"].(map[string]any)["commandId"] = "other"; return r }},
		{"wrong_goal", func(r []historyRow) []historyRow {
			r[3]["data"].(map[string]any)["goalId"] = "goal-44444444-4444-4444-8444-444444444444"
			return r
		}},
		{"wrong_revision", func(r []historyRow) []historyRow {
			r[6]["data"].(map[string]any)["source"].(map[string]any)["revision"] = 2
			return r
		}},
		{"missing_binding", func(r []historyRow) []historyRow { r[3]["type"] = "other"; return r }},
		{"duplicate_binding", func(r []historyRow) []historyRow {
			r = append(r, historyRow{"seq": len(r), "type": "aibertai/query-binding", "data": r[3]["data"]})
			return r
		}},
		{"failed_result", func(r []historyRow) []historyRow {
			r[8]["data"].(map[string]any)["message"].(map[string]any)["isError"] = true
			return r
		}},
		{"goal_revision_changed", func(r []historyRow) []historyRow {
			change := historyRow{"type": "goal/change", "data": map[string]any{"operation": "edit", "goal": map[string]any{"id": "goal-33333333-3333-4333-8333-333333333333", "revision": 2, "phase": "active"}}}
			r = append(r[:4], append([]historyRow{change}, r[4:]...)...)
			for i := range r {
				r[i]["seq"] = i
			}
			r[9]["sourceEventSeqs"] = []int{8}
			return r
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := matchFixture(t, tc.mutate(goalHistoryFixture()), ""); err == nil {
				t.Fatal("accepted unverified goal history")
			}
		})
	}
}
func TestDSHConnectorHistoryRejectsIncompleteOrAmbiguous(t *testing.T) {
	for _, mode := range []string{"rpc_error", "gap", "stall", "ceiling", "session"} {
		t.Run(mode, func(t *testing.T) {
			if _, err := matchFixture(t, historyFixture(), mode); err == nil {
				t.Fatal("accepted incomplete history")
			}
		})
	}
	for _, tc := range []struct {
		name   string
		mutate func([]historyRow) []historyRow
	}{
		{"wrong_operation", func(r []historyRow) []historyRow {
			r[0]["data"].(map[string]any)["inserted"].([]any)[0].(map[string]any)["source"].(map[string]any)["aibertaiOperationId"] = "other"
			return r
		}},
		{"user_differs", func(r []historyRow) []historyRow {
			r[3]["data"] = map[string]any{"id": "message-1", "role": "user", "source": map[string]any{"kind": "user", "aibertaiOperationId": historyOperation}, "content": []any{map[string]any{"type": "text", "text": "tampered"}}}
			return r
		}},
		{"wrong_name", func(r []historyRow) []historyRow { r[4]["data"].(map[string]any)["name"] = "other"; return r }},
		{"wrong_number", func(r []historyRow) []historyRow {
			r[4]["data"].(map[string]any)["arguments"] = `{"n":9007199254740992}`
			return r
		}},
		{"duplicate_argument_key", func(r []historyRow) []historyRow {
			r[4]["data"].(map[string]any)["arguments"] = `{"n":1,"n":9007199254740993}`
			return r
		}},
		{"wrong_step", func(r []historyRow) []historyRow { r[5]["data"].(map[string]any)["step"] = 2; return r }},
		{"wrong_source_seq", func(r []historyRow) []historyRow { r[5]["sourceEventSeqs"] = []int{3}; return r }},
		{"wrong_call", func(r []historyRow) []historyRow {
			r[5]["data"].(map[string]any)["message"].(map[string]any)["toolCallId"] = "other"
			return r
		}},
		{"missing_result", func(r []historyRow) []historyRow { r[5]["type"] = "other"; return r }},
		{"duplicate_admission", func(r []historyRow) []historyRow {
			r = append(r, historyRow{"seq": len(r), "type": "agent/inbox/spliced", "data": r[0]["data"]})
			return r
		}},
		{"duplicate_call", func(r []historyRow) []historyRow {
			copy := historyRow{}
			for k, v := range r[4] {
				copy[k] = v
			}
			r = append(r[:5], append([]historyRow{copy}, r[5:]...)...)
			for i := range r {
				r[i]["seq"] = i
			}
			return r
		}},
		{"duplicate_result", func(r []historyRow) []historyRow {
			copy := historyRow{}
			for k, v := range r[5] {
				copy[k] = v
			}
			r = append(r[:6], append([]historyRow{copy}, r[6:]...)...)
			for i := range r {
				r[i]["seq"] = i
			}
			return r
		}},
		{"multiple_human_inputs", func(r []historyRow) []historyRow {
			other := historyRow{"type": "user/message", "data": map[string]any{"id": "second", "role": "user", "source": map[string]any{"kind": "user"}}}
			r = append(r[:4], append([]historyRow{other}, r[4:]...)...)
			for i := range r {
				r[i]["seq"] = i
			}
			r[6]["sourceEventSeqs"] = []int{5}
			return r
		}},
		{"wrong_turn", func(r []historyRow) []historyRow { r[4]["data"].(map[string]any)["turn"] = 2; return r }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := matchFixture(t, tc.mutate(historyFixture()), ""); err == nil {
				t.Fatal("accepted ambiguous or tampered history")
			}
		})
	}
}

func TestStrictDSHJSONRejectsAmbiguityAndDepth(t *testing.T) {
	for _, raw := range [][]byte{[]byte(`{"a":1,"a":2}`), []byte(`{"a":1} {}`), append([]byte(`{"a":"`), 0xff, '"', '}'), []byte(strings.Repeat("[", 130) + "0" + strings.Repeat("]", 130))} {
		if _, err := strictDSHJSON(raw); err == nil {
			t.Fatalf("accepted invalid JSON: %q", raw)
		}
	}
}

func TestDSHHistoryAfterPrunedToolResult(t *testing.T) {
	for _, mode := range []string{"original", "later call", "missing marker", "wrong range", "wrong source", "changed identity", "unpriced duplicate", "missing original"} {
		t.Run(mode, func(t *testing.T) {
			rows := historyFixture()[:7] // The first step ended, but the turn stays open.
			rows[5]["surfaceOp"] = "append"
			prune := historyRow{"type": "compaction/prune", "data": map[string]any{"shadowedRange": map[string]int{"start": 5, "end": 5}, "shadowedSeqs": []int{5}, "shadowedTokenCount": 1800}}
			message := map[string]any{"role": "tool", "toolCallId": "call-1", "source": map[string]string{"kind": "tool", "callId": "call-1"}, "content": []any{map[string]string{"type": "text", "text": "pruned"}}, "isError": false}
			replacement := historyRow{"type": "tool/result", "sourceEventSeqs": []int{5}, "surfaceOp": map[string]any{"op": "replace", "startSeq": 5, "endSeq": 5}, "data": map[string]any{"turn": 1, "step": 1, "message": message}}
			rows = append(rows, prune, replacement,
				historyRow{"type": "step/start", "data": map[string]any{"turn": 1, "step": 2}},
				historyRow{"type": "tool/call", "data": map[string]any{"turn": 1, "step": 2, "callId": "call-2", "name": "bash", "arguments": `{"command":"echo fixture"}`}},
				historyRow{"type": "tool/result", "surfaceOp": "append", "sourceEventSeqs": []int{10}, "data": map[string]any{"turn": 1, "step": 2, "message": map[string]any{"role": "tool", "toolCallId": "call-2", "source": map[string]string{"kind": "tool", "callId": "call-2"}, "content": []any{map[string]string{"type": "text", "text": "fixture"}}, "isError": false}}},
			)
			for i := range rows {
				rows[i]["seq"] = i
			}
			switch mode {
			case "missing marker":
				prune["type"] = "compaction/start"
			case "wrong range":
				prune["data"].(map[string]any)["shadowedRange"] = map[string]int{"start": 4, "end": 5}
			case "wrong source":
				replacement["sourceEventSeqs"] = []int{4}
			case "changed identity":
				message["toolCallId"] = "call-2"
			case "unpriced duplicate":
				delete(replacement, "surfaceOp")
				prune["type"] = "compaction/start"
			case "missing original":
				rows[5]["type"] = "agent/status"
			}
			target, name, args, wantSeq := "call-1", "connector_test", json.RawMessage(`{"n":9007199254740993}`), 5
			if mode == "later call" {
				target, name, args, wantSeq = "call-2", "bash", json.RawMessage(`{"command":"echo fixture"}`), 11
			}
			raw, err := dshMatchConnectorHistory(t.Context(), historySession, historyOperation, target, name, args, historyPager(t, rows, ""))
			if mode != "original" && mode != "later call" {
				if err == nil {
					t.Fatal("accepted a malformed replacement")
				}
				return
			}
			var result struct{ Event struct{ Seq int } }
			if err != nil || json.Unmarshal(raw, &result) != nil || result.Event.Seq != wantSeq {
				t.Fatalf("receipt seq=%d, want %d: %v", result.Event.Seq, wantSeq, err)
			}
		})
	}
}

func TestDSHPrunedResultDoesNotCompletePendingTool(t *testing.T) {
	rows := historyFixture()
	var turn acpTurn
	project := func(row historyRow) map[string]any {
		t.Helper()
		raw, err := json.Marshal(map[string]any{"event": row})
		if err != nil {
			t.Fatal(err)
		}
		update, err := projectDSHTool(&turn, raw)
		if err != nil {
			t.Fatal(err)
		}
		return update
	}
	project(rows[4])
	result := rows[5]
	result["surfaceOp"] = map[string]any{"op": "replace", "startSeq": 5, "endSeq": 5}
	if update := project(result); update != nil || !turn.ToolRuns[0].Pending {
		t.Fatal("pruned context completed a pending tool")
	}
	result["surfaceOp"] = "append"
	if update := project(result); update["status"] != "completed" || turn.ToolRuns[0].Output != "result" {
		t.Fatal("original execution result was not projected")
	}
}
