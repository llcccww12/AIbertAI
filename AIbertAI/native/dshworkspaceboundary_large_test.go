package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

// A long earlier reply (>512 KiB) is delivered as history fragments. The
// workspace audit must still verify the pending write instead of refusing it.
func TestDSHWorkspaceAuditAcceptsLargeHistoryEvent(t *testing.T) {
	user := map[string]any{"id": "message-2", "role": "user", "source": map[string]any{"kind": "user", "aibertaiOperationId": historyOperation}, "content": []any{map[string]any{"type": "text", "text": "继续"}}}
	args := `{"file_path":"tank.html","content":"<html></html>"}`
	big := strings.Repeat("坦克", 200<<10)
	rows := []historyRow{
		{"type": "turn/start", "data": map[string]any{"turn": 1}},
		{"type": "step/start", "data": map[string]any{"turn": 1, "step": 1}},
		{"type": "assistant/message", "data": map[string]any{"turn": 1, "step": 1, "message": map[string]any{"role": "assistant", "source": map[string]any{"kind": "model"}, "content": []any{map[string]any{"type": "text", "text": big}}}}},
		{"type": "step/end", "data": map[string]any{"turn": 1, "step": 1}},
		{"type": "turn/end", "data": map[string]any{"turn": 1}},
		{"type": "agent/inbox/spliced", "data": map[string]any{"target": "next-turn", "start": 0, "inserted": []any{user}}},
		{"type": "turn/start", "data": map[string]any{"turn": 2}},
		{"type": "step/start", "data": map[string]any{"turn": 2, "step": 1}},
		{"type": "user/message", "data": user},
		{"type": "assistant/message", "data": map[string]any{"turn": 2, "step": 1, "message": map[string]any{"role": "assistant", "source": map[string]any{"kind": "model"}, "content": []any{map[string]any{"type": "tool-call", "id": "call-1", "name": "write", "arguments": args}}}}},
		{"type": "tool/call", "data": map[string]any{"turn": 2, "step": 1, "callId": "call-1", "name": "write", "arguments": args}},
	}
	encoded := make([][]byte, len(rows))
	for i := range rows {
		rows[i]["seq"], rows[i]["time"] = i, i+1
		encoded[i], _ = json.Marshal(rows[i])
	}
	if len(encoded[2]) <= 512<<10 {
		t.Fatal("fixture event must exceed the large-event threshold")
	}
	last := int64(len(rows) - 1)
	fragments := 0
	fetch := func(_ context.Context, params map[string]any) (json.RawMessage, error) {
		after := params["afterSeq"].(int64)
		through := last
		if c, ok := params["throughSeq"].(int64); ok {
			through = c
		}
		if seq, ok := params["fragmentSeq"].(int64); ok {
			fragments++
			offset := params["offset"].(int)
			data := encoded[seq]
			end := min(offset+512<<10, len(data))
			sum := sha256.Sum256(data)
			return json.Marshal(map[string]any{"sessionId": historySession, "throughSeq": through, "seq": seq, "offset": offset, "total": len(data), "sha256": hex.EncodeToString(sum[:]), "data": base64.StdEncoding.EncodeToString(data[offset:end])})
		}
		page := map[string]any{"sessionId": historySession, "throughSeq": through}
		events := []json.RawMessage{}
		for seq := after + 1; seq <= through && len(events) < 3; seq++ {
			if len(encoded[seq]) > 512<<10 {
				if len(events) == 0 {
					sum := sha256.Sum256(encoded[seq])
					page["largeEvent"] = map[string]any{"seq": seq, "total": len(encoded[seq]), "sha256": hex.EncodeToString(sum[:])}
				}
				break
			}
			events = append(events, encoded[seq])
		}
		page["events"] = events
		return json.Marshal(page)
	}
	w, err := dshAuditWorkspacePrefix(t.Context(), historySession, historyOperation, "call-1", "write", json.RawMessage(args), fetch)
	if err != nil {
		t.Fatalf("large history event broke the workspace audit: %v", err)
	}
	if w.CallSeq != last || fragments == 0 {
		t.Fatalf("unexpected witness %+v after %d fragment reads", w, fragments)
	}
}
