package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchivedHistoryLocalAndRelayPages(t *testing.T) {
	const id = "06bfc669-55d4-40eb-afb3-6a9df8247430"
	const messages = 53
	profile := t.TempDir()
	session := filepath.Join(profile, "runtime", "sessions", id)
	if err := os.MkdirAll(session, 0700); err != nil {
		t.Fatal(err)
	}
	var replay strings.Builder
	for i := 1; i <= messages; i++ {
		fmt.Fprintf(&replay, `{"version":11,"from_seq":%d,"to_seq":%d,"message":{"id":"message-%d","role":"assistant","parts":[{"type":"tool_call","toolCallId":"call-%d"}]},"toolCalls":{"call-%d":{"id":"call-%d","status":"completed","output":"result-%d"}}}`+"\n", i, i, i, i, i, i, i)
	}
	meta := fmt.Sprintf(`{"id":%q,"status":"idle","schema_version":8,"replay_version":11,"view_seq":53,"replay_generation":"stable-generation","replay_applied_seq":53,"last_event_seq":53,"message_count":53,"replay_bytes":%d}`, id, replay.Len())
	for name, data := range map[string]string{"meta.json": meta, "replay.jsonl": replay.String()} {
		if err := os.WriteFile(filepath.Join(session, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	host := relaySessionHost{profile: profile, live: &acpLive{}, shareDesktopSessions: true}
	local := func(q historyQuery) archivedHistory {
		t.Helper()
		params, _ := json.Marshal(q)
		value, err := host.localQuery(t.Context(), "_monkeyai/session/messages", params)
		if err != nil {
			t.Fatal(err)
		}
		return value.(archivedHistory)
	}
	assertPage := func(page archivedHistory, first, last int) {
		t.Helper()
		if page.EventSeq != messages || len(page.Messages) != last-first+1 || len(page.ToolCalls) != len(page.Messages) {
			t.Fatalf("unexpected page size or sequence: %d, %d", len(page.Messages), page.EventSeq)
		}
		for index, row := range page.Messages {
			var item struct{ ID string }
			if json.Unmarshal(row, &item) != nil || item.ID != fmt.Sprintf("message-%d", first+index) {
				t.Fatalf("unexpected message at index %d: %s", index, row)
			}
			call := page.ToolCalls[fmt.Sprintf("call-%d", first+index)]
			if !strings.Contains(string(call), fmt.Sprintf(`"output":"result-%d"`, first+index)) {
				t.Fatalf("missing tool result for %s", item.ID)
			}
		}
	}
	q := historyQuery{SessionID: id, Limit: 20}
	latest := local(q)
	assertPage(latest, 34, 53)
	if latest.OlderCursor == nil || latest.NewerCursor != nil {
		t.Fatal("latest page cursor is invalid")
	}
	q.Cursor, q.Direction = *latest.OlderCursor, "older"
	middle := local(q)
	assertPage(middle, 14, 33)
	q.Cursor = *middle.OlderCursor
	first := local(q)
	assertPage(first, 1, 13)
	if first.OlderCursor != nil || first.NewerCursor == nil {
		t.Fatal("first page cursor is invalid")
	}
	q.Cursor, q.Direction = *first.NewerCursor, "newer"
	body, _ := json.Marshal(map[string]any{"method": "_monkeyai/session/messages", "params": q})
	w := httptest.NewRecorder()
	host.history(w, httptest.NewRequest(http.MethodPost, "/read/session", strings.NewReader(string(body))), relayAuthorization{})
	var relay struct{ Result archivedHistory }
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &relay) != nil {
		t.Fatalf("relay pagination failed: %d, %s", w.Code, w.Body)
	}
	assertPage(relay.Result, 14, 33)
	outputQuery, _ := json.Marshal(historyQuery{SessionID: id, ToolCallID: "call-3"})
	output, err := host.localQuery(t.Context(), "_monkeyai/session/tool_output", outputQuery)
	if err != nil || output.(map[string]any)["output"] != "result-3" {
		t.Fatalf("old tool output did not survive pagination: %v, %v", output, err)
	}
	q.Cursor = id + "/other-generation/1"
	data, _ := json.Marshal(q)
	if _, err := host.localQuery(t.Context(), "_monkeyai/session/messages", data); !errors.Is(err, errHistoryCursor) {
		t.Fatalf("accepted a cursor from an outdated archive: %v", err)
	}
}
