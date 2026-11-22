package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestDesktopDSHReattachesPendingUserInput(t *testing.T) {
	for _, method := range []string{"_monkeyai/session/question", "session/request_permission"} {
		t.Run(method, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			h := &relaySessionHost{local: &desktopRuntimeAccess{origin: "http://workbench.test", protocol: "monkeyai.token.test"}, goals: newGoalRuntime(), interactions: newDesktopDSHInteractions()}
			server := httptest.NewServer(h)
			defer server.Close()
			h.local.address = strings.TrimPrefix(server.URL, "http://")
			connect := func() *websocket.Conn {
				t.Helper()
				ws, _, err := websocket.Dial(ctx, server.URL+"/acp", &websocket.DialOptions{Subprotocols: []string{"monkeyai.v1", h.local.protocol}, HTTPHeader: map[string][]string{"Origin": {h.local.origin}}})
				if err != nil {
					t.Fatal(err)
				}
				if err := wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]int{"protocolVersion": 1}}); err != nil {
					t.Fatal(err)
				}
				var init map[string]any
				if err := wsjson.Read(ctx, ws, &init); err != nil || init["result"] == nil {
					t.Fatalf("initialize %v %v", init, err)
				}
				return ws
			}
			read := func(ws *websocket.Conn) string {
				t.Helper()
				var frame struct {
					ID, Method string
					Params     struct{ SessionID string }
				}
				if err := wsjson.Read(ctx, ws, &frame); err != nil || frame.Method != method || frame.Params.SessionID != "original" || !validUUID(frame.ID) {
					t.Fatalf("request %+v %v", frame, err)
				}
				return frame.ID
			}
			done := make(chan error, 1)
			go func() {
				result, err := h.interactions.ask(ctx, nil, method, map[string]any{"sessionId": "original"})
				if err == nil && string(result.Result) != "true" {
					t.Errorf("result %s", result.Result)
				}
				done <- err
			}()
			first := connect()
			id := read(first)
			first.CloseNow()
			second := connect()
			defer second.CloseNow()
			if next := read(second); next != id {
				t.Fatalf("replacement request %s != %s", next, id)
			}
			if err := wsjson.Write(ctx, second, map[string]any{"jsonrpc": "2.0", "id": id, "result": true}); err != nil {
				t.Fatal(err)
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if h.interactions.answer(second, id, acpPermissionReply{Result: json.RawMessage("true")}) {
				t.Fatal("completed request accepted twice")
			}
		})
	}
}
