package main

import (
	"encoding/json"
	"testing"
)

// A live call and its reloaded turn must present the same recorded change.
// A failed edit and a context-pruning replacement carry no applied edit.
func TestDSHFileDiffProjection(t *testing.T) {
	for _, tc := range []struct {
		name, tool, surface string
		failed, wantMeta    bool
	}{
		{"edit", "edit", "append", false, true},
		{"failed", "edit", "append", true, false},
		{"pruned", "edit", "replace", false, false},
		{"external", "custom_connector", "append", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			turn := acpTurn{Operation: "turn"}
			call := map[string]any{"type": "tool/call", "data": map[string]any{"name": tc.tool, "callId": "sdk-call", "arguments": "{}"}}
			meta := map[string]any{"path": "/work/a", "diffs": []any{map[string]any{"path": "/work/a", "oldText": "a", "newText": "b"}}}
			var surface any = "append"
			if tc.surface == "replace" {
				surface = map[string]any{"op": "replace", "startSeq": 1, "endSeq": 1}
			}
			result := map[string]any{"type": "tool/result", "surfaceOp": surface, "data": map[string]any{
				"message": map[string]any{"toolCallId": "sdk-call", "isError": tc.failed, "content": []map[string]string{{"type": "text", "text": "result"}}}, "meta": meta,
			}}
			send := func(event any) map[string]any {
				data, _ := json.Marshal(map[string]any{"event": event})
				update, err := projectDSHTool(&turn, data)
				if err != nil {
					t.Fatal(err)
				}
				return update
			}
			id := send(call)["toolCallId"].(string)
			update := send(result)
			if tc.surface == "replace" && update != nil {
				t.Fatal("pruning settled an edit")
			}
			live, _ := update["_meta"].(map[string]any)
			bytes, _ := json.Marshal(turn)
			var persisted acpTurn
			if err := json.Unmarshal(bytes, &persisted); err != nil {
				t.Fatal(err)
			}
			calls := map[string]any{}
			(acpSession{Engine: "dsh", Turns: []acpTurn{persisted}}).responseMessage(0, calls)
			for _, view := range []map[string]any{live, calls[id].(map[string]any)} {
				value, ok := view["dshResultMeta"]
				if ok != tc.wantMeta {
					t.Fatalf("diff presence %v want %v", ok, tc.wantMeta)
				}
				if ok {
					got, _ := json.Marshal(value)
					want, _ := json.Marshal(meta)
					if string(got) != string(want) {
						t.Fatalf("diff changed: %s", got)
					}
				}
			}
		})
	}
}
