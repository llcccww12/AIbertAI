package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReadDSHContextInjectionMatchesDSHProjection(t *testing.T) {
	frame := func(kind, op, text string) json.RawMessage {
		// The SDK encodes an append as the bare string and a replacement as an object.
		var surface any = "append"
		if op != "append" {
			surface = map[string]any{"op": op, "startSeq": 0, "endSeq": 0}
		}
		raw, _ := json.Marshal(map[string]any{"event": map[string]any{
			"type": "user/message", "time": 1, "surfaceOp": surface,
			"data": map[string]any{"id": "m-" + kind, "role": "user", "source": map[string]string{"kind": kind},
				"content": []map[string]string{{"type": "text", "text": text}}},
		}})
		return raw
	}
	for kind, want := range map[string]bool{
		"runtime-context": true, "skill-catalog": true, "time-context": true,
		"user": false, "compact-checkpoint": false, "subagent-settled": false, "agent-message": false,
	} {
		_, ok := readDSHContextInjection(frame(kind, "append", "x"))
		if ok != want {
			t.Errorf("%s: injected = %v, want %v", kind, ok, want)
		}
	}
	if _, ok := readDSHContextInjection(frame("time-context", "replace", "x")); ok {
		t.Error("a replaced surface is not a new injection")
	}
	developer, _ := json.Marshal(map[string]any{"event": map[string]any{"type": "developer/message", "time": 2,
		"data": map[string]any{"message": map[string]any{"id": "d1", "source": map[string]string{"kind": "agent-instructions"},
			"content": []map[string]string{{"type": "text", "text": "rules"}}}}}})
	if entry, ok := readDSHContextInjection(developer); !ok || entry.Text != "rules" || entry.ID != "d1" {
		t.Fatalf("developer message: %+v %v", entry, ok)
	}
	long, ok := readDSHContextInjection(frame("skill-catalog", "append", strings.Repeat("技", dshContextTextLimit)))
	if !ok || !long.Truncated || len(long.Text) > dshContextTextLimit || !json.Valid([]byte(`"`+long.Text+`"`)) {
		t.Fatalf("long injection not truncated on a rune boundary: %d %v", len(long.Text), long.Truncated)
	}
	var turn acpTurn
	entry, _ := readDSHContextInjection(frame("time-context", "append", "now"))
	if !turn.addContextInjection(entry) || turn.addContextInjection(entry) {
		t.Fatal("the same injection must be recorded once")
	}
}
