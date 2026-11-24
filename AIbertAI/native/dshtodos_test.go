package main

import (
	"encoding/json"
	"testing"
)

func TestDSHTodosReplaceClearAndRejectStale(t *testing.T) {
	var turn acpTurn
	apply := func(seq int64, kind, data string) bool {
		return turn.applyDSHTodo(dshConnectorHistoryEvent{Seq: &seq, Type: kind, Data: json.RawMessage(data)})
	}
	if !apply(2, "todo/write", `{"todos":[{"content":"first","status":"in_progress"},{"content":"second","status":"in_progress"}]}`) || len(turn.DSHTodos.Items) != 2 {
		t.Fatal("parallel activities were lost")
	}
	if !apply(3, "todo/write", `{"todos":[]}`) || len(turn.DSHTodos.Items) != 0 || apply(2, "todo/write", `{"todos":[{"content":"stale","status":"pending"}]}`) {
		t.Fatal("clear or revision fence lost")
	}
	if apply(4, "todo/write", `{"todos":[{"content":"bad","status":"made_up"}]}`) || apply(5, "tool/call", `{}`) {
		t.Fatal("non-authoritative data accepted")
	}
	if !apply(6, "turn/start", `{}`) || turn.DSHTodos.Seq != 6 {
		t.Fatal("next native turn did not clear todos")
	}
}
