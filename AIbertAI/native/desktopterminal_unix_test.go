//go:build darwin || linux

package main

import (
	"context"
	"encoding/base64"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestEmployeeTerminalOpensAndReturnsShellOutput(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	const serverID = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	const machineID = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	const operationID = "c3bd985c-9f71-4b85-937a-c41b1b104d38"
	const terminalID = "c4bd985c-9f71-4b85-937a-c41b1b104d38"
	profile := t.TempDir()
	live, err := openEmployeeSessions(profile, serverID, machineID, "")
	if err != nil {
		t.Fatal(err)
	}
	defer live.store.Close()
	defer live.workspaces.root.Close()
	id, err := live.createInWorkspace(operationID, "", "", serverID, expertSelection{}, nil, "", desktopMediaModels{})
	if err != nil {
		t.Fatal(err)
	}
	terminals, err := newDesktopTerminals()
	if err != nil {
		t.Fatal(err)
	}
	defer terminals.shutdown()
	host := &relaySessionHost{profile: profile, live: live, terminals: terminals, terminalLifetime: t.Context(),
		local: &desktopRuntimeAccess{protocol: "monkeyai.token.test"}}
	server := httptest.NewServer(host)
	defer server.Close()
	host.local.address = strings.TrimPrefix(server.URL, "http://")
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, server.URL+"/acp", &websocket.DialOptions{Subprotocols: []string{"monkeyai.v1", "monkeyai.token.test"}})
	if err != nil {
		t.Fatal(err)
	}
	defer ws.CloseNow()
	call := func(method string, params any) terminalInfo {
		t.Helper()
		if err := wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params}); err != nil {
			t.Fatal(err)
		}
		var response struct {
			Result terminalInfo
			Error  *struct{ Message string }
		}
		if err := wsjson.Read(ctx, ws, &response); err != nil {
			t.Fatal(err)
		}
		if response.Error != nil {
			t.Fatalf("%s: %s", method, response.Error.Message)
		}
		return response.Result
	}
	call("initialize", map[string]int{"protocolVersion": 1})
	empty := call("_monkeyai/terminal/get", map[string]string{"sessionId": id})
	started := call("_monkeyai/terminal/open", map[string]any{
		"sessionId": id, "runtimeInstanceId": empty.Instance, "terminalId": terminalID,
		"expectedRevision": empty.Revision, "cols": 80, "rows": 24,
	})
	if started.State != "running" || started.CWD != live.workspaces.defaultPath(id) {
		t.Fatalf("unexpected terminal: %+v", started)
	}
	terminals.mu.Lock()
	term := terminals.slots[terminalSlot{Session: id}]
	terminals.mu.Unlock()
	if _, err := term.process.Write([]byte("printf '%s%s\\n' actual-shell- output; exit\n")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-term.done:
	case <-ctx.Done():
		t.Fatal("shell did not finish")
	}
	var output strings.Builder
	for _, frame := range term.frames {
		data, err := base64.StdEncoding.DecodeString(frame.Data)
		if err != nil {
			t.Fatal(err)
		}
		output.Write(data)
	}
	if !strings.Contains(output.String(), "actual-shell-output") {
		t.Fatalf("missing shell output: %q", output.String())
	}
}
