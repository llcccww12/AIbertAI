package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDesktopOutlineAndCompactionPreserveTranscript(t *testing.T) {
	const id = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	const operation = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	const model = "c3bd985c-9f71-4b85-937a-c41b1b104d38"
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	now := time.Now().UTC()
	turn := acpTurn{Operation: operation, Model: model, Prompt: "original prompt", Answer: "original answer",
		Started: now, Ended: now, Acked: true, Attachments: []acpAttachment{{Name: "fixture.txt"}}}
	calls := 0
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/models":
			_, _ = w.Write([]byte(`{"models":[{"id":"` + model + `","kind":"text","protocol":"openai_chat_completions","context_window_tokens":16000}],"total_count":1}`))
		case "/v1/chat/completions":
			var request struct {
				Messages []toolConversationMessage
				Tools    []json.RawMessage
			}
			if json.NewDecoder(r.Body).Decode(&request) != nil || len(request.Messages) != 1 ||
				len(request.Tools) != 0 || request.Messages[0].Content == nil ||
				!strings.Contains(*request.Messages[0].Content, "original answer") {
				t.Error("summary request lost the transcript or exposed tools")
			}
			calls++
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Saved context"},"finish_reason":"stop"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer gateway.Close()
	live := &acpLive{client: &client{gateway.URL, gateway.Client()}, store: &desktopSessionStore{Root: root}, cwd: root.Name(),
		sessions: map[string]acpSession{id: {ID: id, Operation: operation, Model: model, CWD: root.Name(),
			Created: now, Updated: now, State: "idle", Turns: []acpTurn{turn}}}}
	host := &relaySessionHost{live: live}
	result, err := host.desktopOutline(id)
	if err != nil {
		t.Fatal(err)
	}
	entry := result.(map[string]any)["entries"].([]desktopOutlineEntry)[0]
	page, err := live.sessions[id].historyWindow(historyQuery{SessionID: id, Cursor: entry.Cursor, Limit: 20}, nil)
	if err != nil || page["messages"].([]map[string]any)[0]["id"] != entry.ID ||
		entry.Text != turn.Prompt || entry.Attachments != 1 {
		t.Fatalf("outline cursor mismatch: %+v, %v", entry, err)
	}
	for range 2 {
		if _, err := host.compactDesktopSession(t.Context(), id); err != nil {
			t.Fatal(err)
		}
	}
	reopened, err := readDesktopSession(root, id)
	if err != nil {
		t.Fatal(err)
	}
	context := reopened.modelHistory()
	if calls != 1 || reopened.Turns[0].Answer != turn.Answer || len(context) != 2 ||
		context[1].Content == nil || *context[1].Content != "Saved context" {
		t.Fatal("compaction did not replace model context or changed the transcript")
	}
	live.compacting[id] = true
	if _, err := live.prepareDesktopRestart(); err == nil {
		t.Fatal("allowed restart during compaction")
	}
	delete(live.compacting, id)
	if _, err := live.prepareDesktopRestart(); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := live.beginTurn(id, model, "", "next", nil, nil); err == nil {
		t.Fatal("accepted a turn after reserving the runtime for restart")
	}
}
