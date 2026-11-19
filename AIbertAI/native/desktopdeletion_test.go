package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestDesktopSessionDeletionThroughACP(t *testing.T) {
	const serverID = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	const machineID = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	const oldID = "c3bd985c-9f71-4b85-937a-c41b1b104d38"
	const missingID = "c4bd985c-9f71-4b85-937a-c41b1b104d38"
	profile := t.TempDir()
	live, err := openEmployeeSessions(profile, serverID, machineID, "")
	if err != nil {
		t.Fatal(err)
	}
	defer live.workspaces.root.Close()
	defer func() {
		if live.store != nil {
			live.store.Close()
		}
	}()
	create := func() string {
		t.Helper()
		operation, err := newUUID()
		if err != nil {
			t.Fatal(err)
		}
		id, err := live.createInWorkspace(operation, "", "", serverID, expertSelection{}, nil, "", desktopMediaModels{})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(live.workspaces.defaultPath(id), "fixture.txt"), []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
		return id
	}
	single, batch, running := create(), create(), create()
	row := live.sessions[running]
	row.State = "running"
	live.sessions[running] = row
	legacy := filepath.Join(profile, "runtime", "sessions", oldID)
	if err := os.MkdirAll(legacy, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "meta.json"), []byte(`{"id":"`+oldID+`"}`), 0600); err != nil {
		t.Fatal(err)
	}

	host := &relaySessionHost{profile: profile, live: live, local: &desktopRuntimeAccess{protocol: "monkeyai.token.test"}}
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
	call := func(method string, params any) (json.RawMessage, []string) {
		t.Helper()
		if err := wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params}); err != nil {
			t.Fatal(err)
		}
		var deleted []string
		for {
			var reply struct {
				ID     int
				Method string
				Params struct{ SessionID string }
				Result json.RawMessage
				Error  json.RawMessage
			}
			if err := wsjson.Read(ctx, ws, &reply); err != nil {
				t.Fatal(err)
			}
			if len(reply.Error) > 0 {
				t.Fatalf("%s: %s", method, reply.Error)
			}
			if reply.Method == "_monkeyai/session/deleted" {
				deleted = append(deleted, reply.Params.SessionID)
			}
			if reply.ID == 1 {
				return reply.Result, deleted
			}
		}
	}
	call("initialize", map[string]int{"protocolVersion": 1})
	if _, deleted := call("session/delete", map[string]string{"sessionId": single}); !reflect.DeepEqual(deleted, []string{single}) {
		t.Fatalf("single deletion events: %v", deleted)
	}
	result, events := call("_monkeyai/sessions/delete", map[string]any{
		"sessionIds": []string{batch, running, oldID, batch, missingID}, "deleteWorkspace": true,
	})
	var response struct {
		Deleted []string
		Failed  []struct{ SessionID, Reason string }
	}
	if err := json.Unmarshal(result, &response); err != nil ||
		!reflect.DeepEqual(response.Deleted, []string{batch, oldID, missingID}) ||
		len(response.Failed) != 1 || response.Failed[0].SessionID != running || response.Failed[0].Reason != "session_busy" ||
		!reflect.DeepEqual(events, []string{batch, oldID}) {
		t.Fatalf("batch result %s, events %v", result, events)
	}
	for _, path := range []string{
		filepath.Join(live.workspaces.defaultPath(single), "fixture.txt"),
		filepath.Join(live.workspaces.root.Name(), "deleted-workspaces", batch, "fixture.txt"),
		filepath.Join(profile, "runtime", "deleted-sessions", oldID, "meta.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{live.workspaces.defaultPath(batch), legacy} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("original path still visible: %s", path)
		}
	}
	ws.CloseNow()
	if err := live.store.Close(); err != nil {
		t.Fatal(err)
	}
	live.store = nil
	reloaded, err := openEmployeeSessions(profile, serverID, machineID, "")
	if err != nil {
		t.Fatal(err)
	}
	defer reloaded.workspaces.root.Close()
	defer reloaded.store.Close()
	if len(reloaded.sessions) != 1 || reloaded.sessions[running].ID != running {
		t.Fatalf("deleted sessions reappeared after restart: %v", reloaded.sessions)
	}
}
