package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDSHMemberTurnStaysInItsConversation(t *testing.T) {
	const secret = "private follow-up"
	session := acpSession{ID: "11111111-1111-4111-8111-111111111111", Engine: "dsh", Turns: []acpTurn{
		{Prompt: "root A", Answer: "answer A"},
		{Prompt: secret, Answer: secret, DSHMemberID: "member"},
		{Prompt: "root B", Answer: "answer B"},
	}, Queue: desktopSessionQueue{Messages: []desktopQueuedMessage{
		{ID: "a", Text: "root queued"}, {ID: "b", Text: secret, DSHMemberID: "member"},
	}}}
	if n := session.historySequence(); n != 4 {
		t.Fatalf("root history length = %d", n)
	}
	for _, q := range []historyQuery{{SessionID: session.ID, Limit: 4}, {SessionID: session.ID, Limit: 1}, {SessionID: session.ID, Limit: 1, Cursor: session.ID + "/source/2", Direction: "newer"}} {
		fitted, err := fitRelayDSHHistoryQuery(session, q, nil)
		if err != nil {
			t.Fatal(err)
		}
		page, err := session.historyWindow(fitted, nil)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(page)
		if strings.Contains(string(raw), secret) || len(page["messages"].([]map[string]any)) != q.Limit {
			t.Fatalf("wrong root page %s", raw)
		}
	}
	for _, v := range []any{session.queueSnapshot(), session.modelHistory()} {
		raw, _ := json.Marshal(v)
		if strings.Contains(string(raw), secret) {
			t.Fatal("member chat appeared in parent context")
		}
	}
}

func TestDSHModelFailureRemainsVisibleAfterFollowup(t *testing.T) {
	const reason = "dsh_model_rate_limited"
	session := acpSession{ID: "11111111-1111-4111-8111-111111111111", Engine: "dsh", Turns: []acpTurn{
		{Prompt: "first", Failed: true, FailureCode: reason},
		{Prompt: "continue", Answer: "done"},
	}}
	page, err := session.historyWindow(historyQuery{SessionID: session.ID, Limit: 4}, nil)
	if err != nil {
		t.Fatal(err)
	}
	messages := page["messages"].([]map[string]any)
	if len(messages) != 4 {
		t.Fatalf("lost model failure from conversation, got %d messages", len(messages))
	}
	parts := messages[1]["parts"].([]map[string]string)
	if len(parts) != 1 || parts[0]["type"] != "error" || parts[0]["reason"] != reason {
		t.Fatalf("missing saved failure reason: %v", parts)
	}
}
