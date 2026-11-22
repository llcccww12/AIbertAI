package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestDSHRealWebSocketDisconnectHistory(t *testing.T) {
	root := t.TempDir()
	counter := filepath.Join(root, "prompts")
	cfg := mockDSH()
	base := cfg.Command
	cfg.Command = func(ctx context.Context, s acpSession) (*exec.Cmd, error) {
		cmd, err := base(ctx, s)
		cmd.Env = append(cmd.Env, "DSH_MOCK_DISCONNECT="+counter)
		return cmd, err
	}
	c, _ := newBackendClient("https://gateway.test", false)
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch {
		case strings.HasSuffix(r.URL.Path, "/turns"):
			fmt.Fprint(w, `{"acked_turn":1}`)
		default:
			fmt.Fprint(w, `{}`)
		}
		return w.Result(), nil
	})
	live := &acpLive{client: c, auth: authorization{Access: "test"}, dir: root, cwd: root, dsh: cfg, operations: map[string]string{}, sessions: map[string]acpSession{}}
	id, err := live.createInWorkspace("11111111-1111-4111-8111-111111111111", root, "", "22222222-2222-4222-8222-222222222222", expertSelection{}, nil, "", desktopMediaModels{})
	if err != nil {
		t.Fatal(err)
	}
	host := &relaySessionHost{profile: root, live: live, local: &desktopRuntimeAccess{origin: "http://desktop.test", protocol: "monkeyai.token.test", reportMachine: "machine"}, goals: newGoalRuntime(), terminalLifetime: t.Context()}
	server := httptest.NewUnstartedServer(host)
	host.local.address = server.Listener.Addr().String()
	server.Start()
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	dial := func() *websocket.Conn {
		t.Helper()
		ws, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/acp", &websocket.DialOptions{Subprotocols: []string{"monkeyai.v1", "monkeyai.token.test"}, HTTPHeader: http.Header{"Origin": []string{"http://desktop.test"}}})
		if err != nil {
			t.Fatal(err)
		}
		if err = wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]int{"protocolVersion": 1}}); err != nil {
			t.Fatal(err)
		}
		var response map[string]any
		if err = wsjson.Read(ctx, ws, &response); err != nil || response["error"] != nil {
			t.Fatalf("initialize: %v %v", response, err)
		}
		return ws
	}
	ws := dial()
	operation := "33333333-3333-4333-8333-333333333333"
	params := map[string]any{"sessionId": id, "prompt": []map[string]string{{"type": "text", "text": "read"}}, "_meta": map[string]string{"operationId": operation}}
	if err = wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "session/prompt", "params": params}); err != nil {
		t.Fatal(err)
	}
	for {
		var frame map[string]any
		if err = wsjson.Read(ctx, ws, &frame); err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(frame)
		if strings.Contains(string(b), `"sessionUpdate":"tool_call"`) {
			break
		}
	}
	_ = ws.CloseNow()
	ws = dial()
	defer ws.CloseNow()
	// Observe a history invalidation after reconnect, then read a coherent snapshot.
	for {
		var frame map[string]any
		if err = wsjson.Read(ctx, ws, &frame); err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(frame)
		if strings.Contains(string(b), "_monkeyai/session/history_changed") {
			break
		}
	}
	for {
		row, _ := live.load(id)
		if len(row.Turns) > 0 && !row.Turns[0].Ended.IsZero() {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("turn did not finish")
		case <-time.After(10 * time.Millisecond):
		}
	}
	if err = wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "_monkeyai/session/messages", "params": map[string]any{"sessionId": id, "limit": 20}}); err != nil {
		t.Fatal(err)
	}
	for {
		var frame struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
		}
		if err = wsjson.Read(ctx, ws, &frame); err != nil {
			t.Fatal(err)
		}
		if frame.ID != 3 {
			continue
		}
		if len(frame.Error) != 0 || !strings.Contains(string(frame.Result), "read output") || !strings.Contains(string(frame.Result), "final") {
			t.Fatalf("history: %s %s", frame.Result, frame.Error)
		}
		break
	}
	if err = wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": 4, "method": "session/prompt", "params": params}); err != nil {
		t.Fatal(err)
	}
	for {
		var frame struct {
			ID int `json:"id"`
		}
		if err = wsjson.Read(ctx, ws, &frame); err != nil {
			t.Fatal(err)
		}
		if frame.ID == 4 {
			break
		}
	}
	data, err := os.ReadFile(counter)
	if err != nil || string(data) != "prompt\n" {
		t.Fatalf("prompt replayed: %q %v", data, err)
	}
}
