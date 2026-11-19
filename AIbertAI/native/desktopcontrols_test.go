package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestDesktopSessionPresentationSurvivesReopen(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	const id = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	const operation = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	const model = "91c29565-0bb8-4e70-9278-032c847bddcf"
	now := time.Now().UTC()
	host := &relaySessionHost{live: &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{
		id: {ID: id, Operation: operation, CWD: root.Name(), Model: model, Created: now, Updated: now, State: "idle"},
	}}}
	for _, request := range []struct{ method, fields string }{
		{"rename", `"title":"  会话回归  "`},
		{"flags", `"pinned":true`},
		{"flags", `"archived":true,"read":true`},
	} {
		raw := json.RawMessage(`{"sessionId":"` + id + `",` + request.fields + `}`)
		if _, err := host.updateDesktopSessionPresentation("_monkeyai/session/"+request.method, raw); err != nil {
			t.Fatal(err)
		}
	}
	reopened, err := readDesktopSession(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.displayTitle() != "会话回归" || reopened.titleSource() != "user" || reopened.Pinned ||
		reopened.ArchivedAt == nil || reopened.ReadAt == nil || !reopened.ReadAt.Equal(now) {
		t.Fatalf("metadata lost or inconsistent: %v", reopened.workspaceMetadata())
	}
}

func TestTrustPendingToolAppliesOnlyToCurrentLocalSession(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	const id = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	const other = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	host := &relaySessionHost{live: &acpLive{store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{
		id: {ID: id, State: "running"}, other: {ID: other},
	}}, local: &desktopRuntimeAccess{protocol: "monkeyai.token.test"}}
	done := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"monkeyai.v1"}})
		if err != nil {
			done <- err
			return
		}
		defer ws.CloseNow()
		_, err = host.requestACPToolApproval(r.Context(), ws, r, id, "fixture", "Read", "Read", "read", map[string]string{"path": "test.txt"})
		done <- err
	}))
	defer server.Close()
	host.local.address = strings.TrimPrefix(server.URL, "http://")
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, server.URL, &websocket.DialOptions{Subprotocols: []string{"monkeyai.v1", "monkeyai.token.test"}})
	if err != nil {
		t.Fatal(err)
	}
	defer ws.CloseNow()
	var message map[string]any
	for range 2 {
		if err := wsjson.Read(ctx, ws, &message); err != nil {
			t.Fatal(err)
		}
	}
	for _, target := range []string{other, id} {
		params := map[string]string{"sessionId": target, "modeId": "bypassPermissions"}
		if err := wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": target, "method": "session/set_mode", "params": params}); err != nil {
			t.Fatal(err)
		}
		message = nil
		if err := wsjson.Read(ctx, ws, &message); err != nil {
			t.Fatal(err)
		}
		if (message["error"] != nil) != (target == other) {
			t.Fatalf("unexpected eligibility: %v", message)
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	first, _ := host.live.load(id)
	second, _ := host.live.load(other)
	if first.permissionMode() != "bypassPermissions" || second.permissionMode() != "auto" {
		t.Fatalf("mode leaked between sessions")
	}
}

func TestDesktopMediaSelectionPreservesOtherKindAndChecksCatalog(t *testing.T) {
	const id = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	const image = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	const video = "c3bd985c-9f71-4b85-937a-c41b1b104d38"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"id":"` + image + `","kind":"image"},{"id":"` + video + `","kind":"video"}],"total_count":2}`))
	}))
	defer server.Close()
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	h := &relaySessionHost{live: &acpLive{client: &client{server.URL, server.Client()}, store: &desktopSessionStore{Root: root}, sessions: map[string]acpSession{id: {ID: id, MediaImageModelID: image}}}}
	_, err = h.setDesktopSessionMedia(t.Context(), json.RawMessage(`{"sessionId":"`+id+`","video":"`+video+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	row, _ := h.live.load(id)
	if row.MediaVideoModelID != video || row.MediaImageModelID != image {
		t.Fatalf("wrong media selections: %v", row.workspaceMetadata())
	}
	if _, err := h.setDesktopSessionMedia(t.Context(), json.RawMessage(`{"sessionId":"`+id+`","video":"`+image+`"}`)); err == nil {
		t.Fatal("wrong kind accepted")
	}
	if _, err := h.setDesktopSessionMedia(t.Context(), json.RawMessage(`{"sessionId":"`+id+`","video":null}`)); err != nil {
		t.Fatal(err)
	}
	row, _ = h.live.load(id)
	if row.MediaVideoModelID != "" || row.MediaImageModelID != image {
		t.Fatal("reset changed the other kind")
	}
}

func TestDesktopMediaGatewayUsesSessionModelAndAccountCatalog(t *testing.T) {
	const session = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	const image = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	const video = "c3bd985c-9f71-4b85-937a-c41b1b104d38"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/models" || r.Header.Get("Authorization") != "Bearer current-account" {
			t.Error("gateway did not use the current account's catalog")
		}
		_, _ = w.Write([]byte(`{"models":[{"id":"` + image + `","kind":"image","model":"picture"},{"id":"` + video + `","kind":"video","model":"motion"}],"total_count":2}`))
	}))
	defer server.Close()
	h := &relaySessionHost{live: &acpLive{client: &client{server.URL, server.Client()}, auth: authorization{Access: "current-account"},
		key: "runtime-invocation-key", store: &desktopSessionStore{}, sessions: map[string]acpSession{session: {ID: session, MediaVideoModelID: video}}}}
	result, err := h.desktopMediaGateway(t.Context(), json.RawMessage(`{"sessionId":"`+session+`","kind":"video"}`))
	if err != nil {
		t.Fatal(err)
	}
	config := result.(map[string]any)
	if config["model"] != "motion@"+video || config["baseUrl"] != server.URL+"/v1" ||
		config["headers"].(map[string]string)["X-MAI-Session-ID"] != session {
		t.Fatal("media call lost the selected model, private backend, or session attribution")
	}
	if _, err := h.desktopMediaGateway(t.Context(), json.RawMessage(`{"sessionId":"`+session+`","kind":"video","modelId":"`+image+`"}`)); err == nil {
		t.Fatal("an image model was issued as a video model")
	}
}

func TestDesktopTrustTurnExecutesToolAndReportsAnswer(t *testing.T) {
	const model = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	const machine = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	const session = "c3bd985c-9f71-4b85-937a-c41b1b104d38"
	const operation = "c4bd985c-9f71-4b85-937a-c41b1b104d38"
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.WriteFile("fixture.txt", []byte("tool result fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	var rounds int
	var reported bool
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/settings":
			_, _ = w.Write([]byte(`{"settings":{"session_reporting":{"enabled":true,"level":"full"}}}`))
		case "/api/v1/models":
			_, _ = w.Write([]byte(`{"models":[{"id":"` + model + `","kind":"text","protocol":"openai_chat_completions"}],"total_count":1}`))
		case "/v1/chat/completions":
			rounds++
			if r.Header.Get("Authorization") != "Bearer model-credential" {
				t.Error("missing model authorization")
			}
			if rounds == 1 {
				_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"tool_calls","message":{"content":null,"tool_calls":[{"id":"call-1","type":"function","function":{"name":"read_text_file","arguments":"{\"path\":\"fixture.txt\"}"}}]}}],"usage":{"prompt_tokens":2,"completion_tokens":1}}`))
				return
			}
			var body struct{ Messages []toolConversationMessage }
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Messages) < 1 {
				t.Errorf("tool result could not be decoded: %v", err)
			} else if last := body.Messages[len(body.Messages)-1]; last.Role != "tool" || last.Content == nil || *last.Content != "tool result fixture" {
				t.Errorf("incorrect tool result: %+v", last)
			}
			_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"content":"fixture completed"}}],"usage":{"prompt_tokens":3,"completion_tokens":4}}`))
		case "/api/v1/sessions/" + session:
			w.WriteHeader(http.StatusNoContent)
		case "/api/v1/sessions/" + session + "/turns":
			var body struct {
				Turns []struct {
					Tools       []reportedTool
					Permissions reportedPermissions
					Input       reportedInput
				}
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Turns) != 1 ||
				len(body.Turns[0].Tools) != 1 || body.Turns[0].Permissions.Allowed != 1 || body.Turns[0].Input.Text == "" {
				t.Errorf("report lost tool or approval data: %+v (%v)", body, err)
			}
			reported = true
			_, _ = w.Write([]byte(`{"acked_turn":1}`))
		default:
			t.Errorf("unexpected backend path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer backend.Close()
	host := &relaySessionHost{root: root, approvedFile: "fixture.txt",
		local: &desktopRuntimeAccess{protocol: "monkeyai.token.test", reportMachine: machine},
		live: &acpLive{client: &client{backend.URL, backend.Client()}, auth: authorization{Access: "account"}, key: "model-credential",
			store: &desktopSessionStore{Root: root}, cwd: root.Name(),
			sessions: map[string]acpSession{session: {ID: session, CWD: root.Name(), Model: model, State: "idle"}}}}
	desktop := httptest.NewServer(host)
	defer desktop.Close()
	host.local.address = strings.TrimPrefix(desktop.URL, "http://")
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, desktop.URL+"/acp", &websocket.DialOptions{Subprotocols: []string{"monkeyai.v1", "monkeyai.token.test"}})
	if err != nil {
		t.Fatal(err)
	}
	defer ws.CloseNow()
	send := func(id int, method string, params any) {
		t.Helper()
		if err := wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
			t.Fatal(err)
		}
	}
	read := func() map[string]any {
		t.Helper()
		var message map[string]any
		if err := wsjson.Read(ctx, ws, &message); err != nil {
			t.Fatal(err)
		}
		return message
	}
	send(1, "initialize", map[string]int{"protocolVersion": 1})
	read()
	send(2, "session/prompt", map[string]any{"sessionId": session, "_meta": map[string]string{"operationId": operation},
		"prompt": []map[string]string{{"type": "text", "text": "Read the fixture"}}})
	answered, allowed := false, false
	for {
		message := read()
		if message["method"] == "session/request_permission" {
			send(3, "session/set_mode", map[string]string{"sessionId": session, "modeId": "bypassPermissions"})
		}
		if message["id"] == float64(3) {
			allowed = message["error"] == nil
		}
		if params, ok := message["params"].(map[string]any); ok {
			if update, ok := params["update"].(map[string]any); ok {
				if content, ok := update["content"].(map[string]any); ok && content["text"] == "fixture completed" {
					answered = true
				}
			}
		}
		if message["id"] == float64(2) {
			if !answered || !reported || !allowed || message["error"] != nil {
				t.Fatalf("turn did not finish through the real ACP handler: allowed=%v answered=%v reported=%v result=%v", allowed, answered, reported, message)
			}
			break
		}
	}
	row, err := host.live.load(session)
	if err != nil || row.permissionMode() != "bypassPermissions" || !row.Turns[0].Acked {
		t.Fatalf("turn or mode not retained: %v", err)
	}
}

func TestDesktopCancelInterruptsModelAndReportsTurn(t *testing.T) {
	const model = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	const machine = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	const session = "c3bd985c-9f71-4b85-937a-c41b1b104d38"
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	started := make(chan struct{})
	reported := make(chan string, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/settings":
			_, _ = w.Write([]byte(`{"settings":{"session_reporting":{"enabled":true,"level":"stats"}}}`))
		case "/api/v1/models":
			_, _ = w.Write([]byte(`{"models":[{"id":"` + model + `","kind":"text","protocol":"openai_chat_completions"}],"total_count":1}`))
		case "/v1/chat/completions":
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			close(started)
			select {
			case <-r.Context().Done():
			case <-ctx.Done():
			}
		case "/api/v1/sessions/" + session:
			w.WriteHeader(http.StatusNoContent)
		case "/api/v1/sessions/" + session + "/turns":
			var body struct {
				Turns []struct {
					Stop  string `json:"stop_reason"`
					Input reportedInput
				}
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Turns) != 1 || body.Turns[0].Input.Text != "" {
				t.Errorf("missing turn report: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			reported <- body.Turns[0].Stop
			_, _ = w.Write([]byte(`{"acked_turn":1}`))
		default:
			t.Errorf("unexpected backend path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer backend.Close()
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	host := &relaySessionHost{root: root, local: &desktopRuntimeAccess{protocol: "monkeyai.token.test", reportMachine: machine},
		live: &acpLive{client: &client{backend.URL, backend.Client()}, auth: authorization{Access: "account"}, key: "model-credential",
			store: &desktopSessionStore{Root: root}, cwd: root.Name(),
			sessions: map[string]acpSession{session: {ID: session, CWD: root.Name(), Model: model, State: "idle"}}}}
	desktop := httptest.NewServer(host)
	defer desktop.Close()
	host.local.address = strings.TrimPrefix(desktop.URL, "http://")
	send := func(ws *websocket.Conn, id int, method string, params any) {
		t.Helper()
		if err := wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
			t.Fatal(err)
		}
	}
	read := func(ws *websocket.Conn, id int) map[string]any {
		t.Helper()
		for {
			var reply struct {
				ID     int
				Result map[string]any
				Error  json.RawMessage
			}
			if err := wsjson.Read(ctx, ws, &reply); err != nil {
				t.Fatal(err)
			}
			if reply.ID == id {
				if len(reply.Error) != 0 {
					t.Fatalf("request failed: %s", reply.Error)
				}
				return reply.Result
			}
		}
	}
	dial := func() *websocket.Conn {
		t.Helper()
		ws, _, err := websocket.Dial(ctx, desktop.URL+"/acp", &websocket.DialOptions{Subprotocols: []string{"monkeyai.v1", "monkeyai.token.test"}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { ws.CloseNow() })
		send(ws, 1, "initialize", map[string]int{"protocolVersion": 1})
		read(ws, 1)
		return ws
	}
	turn, controls := dial(), dial()
	defer turn.CloseNow()
	defer controls.CloseNow()
	send(turn, 2, "session/prompt", map[string]any{"sessionId": session, "prompt": []map[string]string{{"type": "text", "text": "Fixture"}}})
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("model never started")
	}
	send(controls, 4, "_monkeyai/draft/set", map[string]any{"target": map[string]string{"sessionId": session},
		"draft": map[string]any{"text": "draft during the turn", "files": []any{}}})
	if read(controls, 4)["ok"] != true {
		t.Fatal("draft did not save during the turn")
	}
	send(controls, 5, "_monkeyai/session/flags", map[string]any{"sessionId": session, "pinned": true})
	if read(controls, 5)["pinned"] != true {
		t.Fatal("session flags did not update during the turn")
	}
	send(controls, 2, "session/cancel", map[string]string{"sessionId": machine})
	if read(controls, 2)["stopped"] != false {
		t.Fatal("cancel affected the wrong session")
	}
	send(controls, 3, "session/cancel", map[string]string{"sessionId": session})
	if read(controls, 3)["stopped"] != true || read(turn, 2)["stopReason"] != "cancelled" {
		t.Fatal("cancel did not interrupt the running turn")
	}
	row, err := host.live.load(session)
	if err != nil || row.State != "idle" || !row.Pinned || !row.Turns[0].Acked || !row.Turns[0].Interrupted {
		t.Fatal("interrupted turn was not persisted")
	}
	if stop := <-reported; stop != "interrupted" {
		t.Fatalf("reported stop reason: %s", stop)
	}
}
