package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestDesktopGoalRunsAfterDisconnectAndRecoversOnRestart(t *testing.T) {
	const serverID = "dbf6ae3e-fe84-45b8-afde-3fc42265df07"
	const machine = "e19939a6-518e-491e-a119-7c221e4c30e8"
	const model = "020f4590-bdd7-4df8-b584-c0cd32601ad4"
	const operation = "b65f2c45-2e7e-412d-a071-2afdf434a8d8"
	const goalID = "bf0b238c-c079-4d0f-9b26-4c7e5354eacd"
	const queuedID = "99133b41-63bf-4b4f-96b5-47ffbfdac3b1"
	const origin = "http://renderer.test"
	const protocol = "monkeyai.token.fixture"
	profile := t.TempDir()
	allowModel := make(chan struct{})
	allowRecovery := make(chan struct{})
	checkpointReached := make(chan struct{})
	var calls atomic.Int32
	toolCall := func(id string) string {
		return `{"usage":{"prompt_tokens":20,"completion_tokens":5},"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","tool_calls":[{"id":"` + id + `","type":"function","function":{"name":"write_text_file","arguments":"{\"path\":\"report.txt\",\"content\":\"saved progress\"}"}}]}}]}`
	}
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/models":
			_, _ = io.WriteString(w, `{"models":[{"id":"`+model+`","kind":"text","protocol":"openai_chat_completions","stream_mode":"off"}],"total_count":1}`)
		case strings.HasSuffix(r.URL.Path, "/turns"):
			acked := 1
			if r.URL.Query().Get("after_turn") == "1" {
				acked = 2
			}
			_ = json.NewEncoder(w).Encode(map[string]int{"acked_turn": acked})
		case r.URL.Path == "/api/v1/settings", strings.HasPrefix(r.URL.Path, "/api/v1/sessions/"):
			_, _ = io.WriteString(w, "{}")
		case r.URL.Path == "/v1/chat/completions":
			body, _ := io.ReadAll(r.Body)
			switch calls.Add(1) {
			case 1:
				select {
				case <-allowModel:
					_, _ = io.WriteString(w, toolCall("first"))
				case <-r.Context().Done():
				}
			case 2:
				close(checkpointReached)
				<-r.Context().Done()
			case 3:
				if !strings.Contains(string(body), `"role":"tool"`) || !strings.Contains(string(body), `"tool_call_id":"first"`) {
					t.Error("model did not receive the recorded tool result after restart")
				}
				select {
				case <-allowRecovery:
					_, _ = io.WriteString(w, toolCall("retried"))
				case <-r.Context().Done():
				}
			case 4:
				_, _ = io.WriteString(w, `{"usage":{"prompt_tokens":20,"completion_tokens":5},"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"Report complete\n[GOAL_COMPLETE]"}}]}`)
			case 5:
				if !strings.Contains(string(body), "Queued after the goal") || !strings.Contains(string(body), "Fixture canvas node") {
					t.Error("queued message or its canvas reference was lost")
				}
				_, _ = io.WriteString(w, `{"choices":[{"finish_reason":"stop","message":{"content":"Queue complete"}}]}`)
			default:
				t.Error("unexpected model request")
				http.Error(w, "unexpected request", http.StatusBadRequest)
			}
		default:
			t.Errorf("unexpected gateway request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer gateway.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
	defer cancel()
	open := func() *acpLive {
		live, err := openEmployeeSessions(profile, serverID, machine, "")
		if err != nil {
			t.Fatal(err)
		}
		live.client = &client{gateway.URL, gateway.Client()}
		live.auth = authorization{Access: "fixture"}
		live.key = "fixture"
		return live
	}
	hostFor := func(live *acpLive) *relaySessionHost {
		return &relaySessionHost{live: live, goals: newGoalRuntime(), local: &desktopRuntimeAccess{
			origin: origin, protocol: protocol, reportMachine: machine,
		}}
	}
	start := func(host *relaySessionHost) func() {
		runtime, stop := context.WithCancel(ctx)
		done := make(chan struct{})
		go func() {
			defer close(done)
			host.superviseGoals(runtime)
		}()
		return func() {
			stop()
			<-done
		}
	}

	first := open()
	id, err := first.createInWorkspace(operation, "", "", model, expertSelection{}, nil, "default", desktopMediaModels{})
	if err != nil {
		t.Fatal(err)
	}
	initial, err := first.load(id)
	if err != nil {
		t.Fatal(err)
	}
	host := hostFor(first)
	server := httptest.NewServer(http.HandlerFunc(host.acp))
	defer server.Close()
	host.local.address = strings.TrimPrefix(server.URL, "http://")
	stopFirst := start(host)
	defer stopFirst()
	ws, _, err := websocket.Dial(ctx, "ws://"+host.local.address, &websocket.DialOptions{
		Subprotocols: []string{"monkeyai.v1", protocol}, HTTPHeader: http.Header{"Origin": {origin}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer ws.CloseNow()
	rpc := func(id int, method string, params map[string]any) json.RawMessage {
		t.Helper()
		if err := wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
			t.Fatal(err)
		}
		for {
			var response struct {
				ID            int
				Result, Error json.RawMessage
			}
			if err := wsjson.Read(ctx, ws, &response); err != nil {
				t.Fatal(err)
			}
			if response.ID == id {
				if len(response.Error) > 0 {
					t.Fatalf("%s: %s", method, response.Error)
				}
				return response.Result
			}
		}
	}
	rpc(1, "initialize", map[string]any{"protocolVersion": 1})
	result := rpc(2, "_monkeyai/session/goal", map[string]any{
		"sessionId": id, "action": "create", "requestId": goalID, "objective": "Write a report",
	})
	if !strings.Contains(string(result), `"background":true`) {
		t.Fatalf("goal was not accepted for background execution: %s", result)
	}
	close(allowModel)
	for {
		var request struct {
			ID     json.RawMessage
			Method string
		}
		if err := wsjson.Read(ctx, ws, &request); err != nil {
			t.Fatal(err)
		}
		if request.Method == "session/request_permission" {
			if err := wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": request.ID,
				"result": map[string]any{"outcome": map[string]string{"outcome": "selected", "optionId": "allow"}}}); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	select {
	case <-checkpointReached:
	case <-ctx.Done():
		t.Fatal("background turn did not reach its second model request")
	}
	if err := ws.Close(websocket.StatusNormalClosure, "window closed"); err != nil {
		t.Fatal(err)
	}
	stopFirst()
	before, err := first.load(id)
	if err != nil || len(before.Turns) != 1 || before.Turns[0].GoalCheckpoint == nil ||
		len(before.Turns[0].GoalCheckpoint.Runs) != 1 || before.Turns[0].GoalCheckpoint.Pending != nil {
		t.Fatalf("completed tool was not persisted before shutdown: %+v, %v", before.Turns, err)
	}
	path := filepath.Join(initial.CWD, "report.txt")
	if contents, err := os.ReadFile(path); err != nil || string(contents) != "saved progress" {
		t.Fatalf("tool did not run with the window closed: %q, %v", contents, err)
	}
	if err := os.WriteFile(path, []byte("edited during restart"), 0600); err != nil {
		t.Fatal(err)
	}
	first.store.Close()
	first.workspaces.root.Close()

	second := open()
	defer second.store.Close()
	defer second.workspaces.root.Close()
	resumedHost := hostFor(second)
	resumedServer := httptest.NewServer(http.HandlerFunc(resumedHost.acp))
	defer resumedServer.Close()
	resumedHost.local.address = strings.TrimPrefix(resumedServer.URL, "http://")
	stopSecond := start(resumedHost)
	defer stopSecond()
	ws, _, err = websocket.Dial(ctx, "ws://"+resumedHost.local.address, &websocket.DialOptions{
		Subprotocols: []string{"monkeyai.v1", protocol}, HTTPHeader: http.Header{"Origin": {origin}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer ws.CloseNow()
	rpc(3, "initialize", map[string]any{"protocolVersion": 1})
	queued := rpc(4, "_monkeyai/session/queue", map[string]any{
		"sessionId": id, "action": "add", "id": queuedID, "sendIfIdle": true,
		"text":        "Queued after the goal",
		"canvasNodes": []acpCanvasNode{{ID: queuedID, Title: "Fixture canvas node", NodeType: "text"}},
	})
	if !strings.Contains(string(queued), queuedID) || calls.Load() > 3 {
		t.Fatalf("follow-up was not acknowledged while the goal owned the session: %s", queued)
	}
	close(allowRecovery)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		row, err := second.load(id)
		if err != nil {
			t.Fatal(err)
		}
		if row.Goal.Status == "completed" && len(row.Turns) == 2 && row.Turns[1].Acked {
			if !row.Turns[0].Acked || row.Turns[0].Operation != before.Turns[0].Operation ||
				row.Turns[0].Answer != "Report complete" || row.Goal.Usage.Tokens != 75 || row.GoalRun != "" ||
				row.Turns[0].Permissions != (reportedPermissions{Asked: 1, Allowed: 1}) ||
				row.Turns[1].ClientID != queuedID || row.Turns[1].Answer != "Queue complete" ||
				len(row.Queue.Messages) != 0 {
				t.Fatalf("incorrect recovered turn: %+v", row)
			}
			break
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatalf("goal and its queued follow-up did not complete after restart: %+v", row.Goal)
		}
	}
	if contents, err := os.ReadFile(path); err != nil || string(contents) != "edited during restart" {
		t.Fatalf("tool was repeated after restart: %q, %v", contents, err)
	}
}

func TestDesktopGoalPausesWithUnconfirmedAction(t *testing.T) {
	const server = "84ec3d37-ec01-4348-aa35-7e1007c6c731"
	const machine = "c1207fc3-d5c2-4e26-a2f5-1c2dd0f49f65"
	const id = "8c81b546-39db-48ae-a41c-857bc4ac99a8"
	const sessionOp = "d9618877-384d-4ef6-8092-d78242d4b703"
	const runOp = "bef35947-ef29-4cba-8dae-e3aebd029257"
	const pauseOp = "77a51b58-f658-47e5-b2ec-7d9ee510c129"
	profile := t.TempDir()
	cwd := filepath.Join(profile, "workspace")
	now := time.Now().UTC()
	live, err := openDesktopSessions(profile, server, machine, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if err := live.save(acpSession{
		ID: id, Operation: sessionOp, Model: "6321dabc-2eae-4469-88cf-096d20414cee", CWD: cwd, Created: now, Updated: now, State: "idle",
		GoalRun: runOp, Goal: &desktopGoal{ID: runOp, Objective: "Run a tool", Generation: 1, Current: true, Status: "active"},
	}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	if _, _, _, err := live.reserveTurn(id, runOp, runOp, "Run a tool", nil, cancel, true, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := live.checkpointGoal(id, runOp, goalCheckpoint{Pending: &goalReplayMessage{}}); err != nil {
		t.Fatal(err)
	}
	pause := json.RawMessage(`{"sessionId":"` + id + `","action":"pause","requestId":"` + pauseOp + `","expectedGoalId":"` + runOp + `","expectedGeneration":1}`)
	if _, err := live.saveGoalAction(pause, true); err != nil || ctx.Err() != context.Canceled {
		t.Fatalf("pause did not cancel the active attempt: %v, %v", err, ctx.Err())
	}
	if err := live.store.Close(); err != nil {
		t.Fatal(err)
	}

	resumed, err := openDesktopSessions(profile, server, machine, cwd)
	if err != nil {
		t.Fatal(err)
	}
	defer resumed.store.Close()
	row, err := resumed.load(id)
	if err != nil || row.State != "idle" || row.Goal.Status != "paused" || row.Goal.Reason != "resume_confirmation" ||
		row.GoalRun != runOp || row.Turns[0].GoalCheckpoint.Pending == nil {
		t.Fatalf("unconfirmed action was not held for confirmation: %+v, %v", row, err)
	}
	if _, _, _, err := resumed.reserveTurn(id, runOp, runOp, "Run a tool", nil, nil, true, nil); err == nil {
		t.Fatal("started an unconfirmed action automatically")
	}
}
