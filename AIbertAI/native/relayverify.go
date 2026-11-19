package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func verifyRemoteRuntime(ctx context.Context, profile, sessionID, file string, preview bool, emit func(string, string) error) (err error) {
	if !filepath.IsAbs(file) {
		return errors.New("expected an absolute path for the approved fixture")
	}
	archive, err := readLegacySessionArchive(profile, sessionID)
	if err != nil || len(archive.Replay) == 0 || len(archive.Replay) > 20 {
		return errors.New("expected a completed session fixture with 1 to 20 messages")
	}
	expectedHistory, err := legacyHistoryPage(archive)
	if err != nil {
		return errors.New("expected a completed session fixture")
	}
	root, err := os.OpenRoot(filepath.Dir(file))
	if err != nil {
		return err
	}
	defer root.Close()
	name := filepath.Base(file)
	expected, err := readApprovedToolFile(root, name, name)
	if err != nil || expected == "" {
		return errors.New("expected a nonempty UTF-8 file fixture")
	}
	c, err := newClient(backendOrigin)
	if err != nil {
		return err
	}
	scopes := "device:bind device:publish device:connect session:read files:read"
	if validUUID(modelID) {
		scopes += " session:control session:report files:write"
	}
	loginCtx, stopLogin := context.WithTimeout(ctx, 5*time.Minute)
	auth, user, err := c.loginWithScopes(loginCtx, scopes, func(link string) error { return emit("authorize", link) })
	stopLogin()
	if auth.Refresh != "" {
		defer func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err = errors.Join(err, c.revoke(cleanup, auth))
		}()
	}
	if err != nil {
		return err
	}
	duration, label := time.Minute, "AIbertAI Remote Runtime Acceptance"
	if preview {
		duration, label = 10*time.Minute, "AIbertAI 源码远程验收设备"
	}
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	serverID, keys, err := c.relaySigningKeys(ctx)
	if err != nil {
		return err
	}
	device, err := newRelayDevice(label)
	if err != nil {
		return err
	}
	controlWS, ready, err := c.relayHandshake(ctx, auth, user, serverID, device, "publish", "control", 0)
	if err != nil {
		return fmt.Errorf("relay control handshake: %w", err)
	}
	defer controlWS.CloseNow()
	control, err := relayMux(ctx, controlWS)
	if err != nil {
		return err
	}
	defer control.Close()
	access := &relayAccess{issuer: c.server, owner: user.ID, boot: device.boot, ready: ready, keys: keys, control: control}
	host := &relaySessionHost{access: access, profile: profile, sessionID: sessionID, root: root, approvedFile: name, trace: preview}
	if validUUID(modelID) {
		protocol, protocolErr := c.selectedModel(ctx, auth)
		if protocolErr != nil {
			return protocolErr
		}
		key, keyErr := c.createModelKey(ctx, auth)
		if validUUID(key.ID) {
			defer func() {
				cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
				defer stop()
				err = errors.Join(err, c.json(cleanup, "DELETE", "/api/v1/api-keys/"+key.ID, auth.Access, "", nil, nil))
			}()
		}
		if keyErr != nil {
			return keyErr
		}
		dir, dirErr := os.MkdirTemp("", "aibertai-acp-acceptance-")
		if dirErr != nil {
			return dirErr
		}
		defer os.RemoveAll(dir)
		host.live = &acpLive{client: c, auth: auth, key: key.Value, protocol: protocol, dir: dir, cwd: filepath.Dir(file),
			operations: map[string]string{}, sessions: map[string]acpSession{}}
		uploadRoot, err := os.OpenRoot(dir)
		if err != nil {
			return err
		}
		defer uploadRoot.Close()
		host.uploads = openRelayUploads(uploadRoot, device.private, false)
	}
	hostCtx, stopHost := context.WithCancel(ctx)
	server := &http.Server{Handler: host, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 15 * time.Second, MaxHeaderBytes: 32 << 10, BaseContext: func(_ net.Listener) context.Context { return hostCtx }}
	defer func() {
		stopHost()
		_ = server.Close()
	}()
	go server.Serve(dataRelayListener{mux: control})
	closeData := func() {}
	defer func() { closeData() }()
	attachData := func() error {
		closeData()
		ws, attached, err := c.relayHandshake(ctx, auth, user, serverID, device, "attach", "data", ready.Epoch)
		if err != nil {
			return fmt.Errorf("relay data handshake: %w", err)
		}
		if attached.GrantVersion != ready.GrantVersion || attached.RegistrationVersion != ready.RegistrationVersion {
			_ = ws.CloseNow()
			return errors.New("relay_attach_changed")
		}
		mux, err := relayMux(ctx, ws)
		if err != nil {
			_ = ws.CloseNow()
			return err
		}
		closeData = func() { _ = mux.Close(); _ = ws.CloseNow() }
		go server.Serve(dataRelayListener{mux: mux})
		return nil
	}
	if err := attachData(); err != nil {
		return err
	}
	if preview {
		if err := emit("preview_device", device.machine); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-control.CloseChan():
			return errors.New("preview_control_channel_closed")
		}
	}
	return c.verifyRelayRequests(ctx, auth, serverID, device.machine, sessionID, name, filepath.Dir(file), expected, expectedHistory, attachData, emit)
}

func (c *client) verifyRelayRequests(ctx context.Context, auth authorization, serverID, machine, sessionID, name, cwd, expected string, expectedHistory archivedHistory, reattach func() error, emit func(string, string) error) error {
	var ticket remoteTicket
	var err error
	// Route readiness is published after the final handshake acknowledgement.
	for tries := 0; tries < 6; tries++ {
		ticket, err = c.remoteConnection(ctx, auth, serverID, machine, "")
		if err == nil || err.Error() != "device_offline" {
			break
		}
		select {
		case <-time.After(150 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err != nil {
		return fmt.Errorf("relay ticket: %w", err)
	}
	prefix := "/api/v1/endpoints/" + machine + "/proxy"
	var history struct{ Result archivedHistory }
	if err := c.json(ctx, "POST", prefix+"/read/session", ticket.Token, "", map[string]any{
		"method": "_monkeyai/session/messages", "params": map[string]any{"sessionId": sessionID, "limit": 20},
	}, &history); err != nil {
		return err
	}
	want, wantErr := json.Marshal(expectedHistory)
	got, gotErr := json.Marshal(history.Result)
	if wantErr != nil || gotErr != nil || !bytes.Equal(want, got) {
		return errors.New("relay_history_mismatch")
	}
	call := func(path string) map[string]any {
		return map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call",
			"params": map[string]any{"name": workspaceReadTool, "arguments": map[string]string{"path": path}}}
	}
	var tool struct {
		Result struct {
			Content []struct{ Type, Text string }
		}
	}
	if err := c.json(ctx, "POST", prefix+"/cap/workspace/mcp", ticket.Token, "", call(name), &tool); err != nil {
		return err
	}
	if len(tool.Result.Content) != 1 || tool.Result.Content[0].Type != "text" || tool.Result.Content[0].Text != expected {
		return errors.New("relay_tool_result_mismatch")
	}
	var denied struct {
		Result json.RawMessage
		Error  struct{ Code int }
	}
	if err := c.json(ctx, "POST", prefix+"/cap/workspace/mcp", ticket.Token, "", call("./"+name), &denied); err != nil {
		return err
	}
	if denied.Result != nil || denied.Error.Code != -32602 {
		return errors.New("relay_tool_scope_failed")
	}
	if err := reattach(); err != nil {
		return err
	}
	if err := c.verifyRelayACP(ctx, prefix, machine, ticket.Token, sessionID, cwd, expected, emit); err != nil {
		return err
	}
	return emit("remote_runtime", "Verified archived session and approved workspace tool through the private relay")
}

func (c *client) verifyRelayACP(ctx context.Context, prefix, machine, ticket, archiveID, cwd, expectedOutput string, emit func(string, string) error) error {
	connect := func() (*websocket.Conn, error) {
		address := "ws" + strings.TrimPrefix(c.server, "http") + prefix + "/acp"
		ws, response, err := websocket.Dial(ctx, address, &websocket.DialOptions{HTTPClient: c.http,
			Subprotocols: []string{"monkeyai.v1", "monkeyai.token." + ticket}})
		if err != nil {
			if response != nil {
				return nil, fmt.Errorf("ACP upgrade: HTTP %d", response.StatusCode)
			}
			return nil, errors.New("ACP upgrade failed")
		}
		if ws.Subprotocol() != "monkeyai.v1" {
			ws.CloseNow()
			return nil, errors.New("ACP subprotocol mismatch")
		}
		return ws, nil
	}
	ws, err := connect()
	if err != nil {
		return err
	}
	defer ws.CloseNow()
	var list struct{ Sessions []struct{ SessionID string } }
	if _, err := acpVerifyRPC(ctx, ws, 1, "initialize", map[string]int{"protocolVersion": 1}, nil); err != nil {
		return err
	}
	if _, err := acpVerifyRPC(ctx, ws, 2, "session/list", map[string]any{}, &list); err != nil {
		return err
	}
	if len(list.Sessions) != 1 || list.Sessions[0].SessionID != archiveID {
		return errors.New("ACP did not list the archived session")
	}
	if !validUUID(modelID) {
		return nil
	}
	op, err := newUUID()
	if err != nil {
		return err
	}
	args := map[string]any{"cwd": cwd, "_meta": map[string]string{"operationId": op}}
	var created struct{ SessionID string }
	if _, err := acpVerifyRPC(ctx, ws, 3, "session/new", args, &created); err != nil {
		return err
	}
	firstOperation, err := newUUID()
	if err != nil {
		return err
	}
	prompt := func(operation, text string) map[string]any {
		return map[string]any{"sessionId": created.SessionID, "_meta": map[string]string{"operationId": operation},
			"prompt": []map[string]string{{"type": "text", "text": text}}}
	}
	firstPrompt := prompt(firstOperation, "Use the approved file-reading tool once and then acknowledge the result.")
	answer, err := acpVerifyRPC(ctx, ws, 4, "session/prompt", firstPrompt, nil, "allow")
	if err != nil || answer == "" || !validUUID(created.SessionID) {
		return errors.Join(errors.New("ACP live prompt did not complete"), err)
	}
	_ = ws.CloseNow()
	second, err := connect()
	if err != nil {
		return err
	}
	defer second.CloseNow()
	if _, err := acpVerifyRPC(ctx, second, 1, "initialize", map[string]int{"protocolVersion": 1}, nil); err != nil {
		return err
	}
	var retried struct{ SessionID string }
	if _, err := acpVerifyRPC(ctx, second, 2, "session/new", args, &retried); err != nil || retried.SessionID != created.SessionID {
		return errors.Join(errors.New("session creation was not idempotent across reconnect"), err)
	}
	restored, err := acpVerifyRPC(ctx, second, 3, "session/load", map[string]string{"sessionId": created.SessionID, "cwd": cwd}, nil)
	if err != nil || restored != answer {
		return errors.Join(errors.New("ACP reconnect transcript mismatch"), err)
	}
	duplicate, err := acpVerifyRPC(ctx, second, 4, "session/prompt", firstPrompt, nil)
	if err != nil || duplicate != answer {
		return errors.Join(errors.New("ACP completed operation replay mismatch"), err)
	}
	nextOperation, err := newUUID()
	if err != nil {
		return err
	}
	continuation, err := acpVerifyRPC(ctx, second, 5, "session/prompt", prompt(nextOperation,
		"Use the previous turn and tool result as context and confirm what happened. Do not call the tool again."), nil)
	if err != nil || continuation == "" {
		return errors.Join(errors.New("ACP second turn did not complete"), err)
	}
	rejectedOperation, err := newUUID()
	if err != nil {
		return err
	}
	_, err = acpVerifyRPC(ctx, second, 6, "session/prompt", prompt(rejectedOperation,
		"Request the file-reading tool again for the permission denial test."), nil, "deny")
	if err == nil || err.Error() != "session_turn_failed" {
		return errors.Join(errors.New("ACP permission denial did not stop the tool"), err)
	}
	var response struct {
		Result struct {
			Session   struct{ ID string }
			Messages  []json.RawMessage
			ToolCalls map[string]struct {
				Output  string
				IsError bool
			}
		}
	}
	if err := c.json(ctx, "POST", prefix+"/read/session", ticket, "", map[string]any{
		"method": "_monkeyai/session/messages", "params": map[string]any{"sessionId": created.SessionID, "limit": 20},
	}, &response); err != nil {
		return err
	}
	snapshot := response.Result
	refused := snapshot.ToolCalls[created.SessionID+"-3-tool"]
	if snapshot.Session.ID != created.SessionID || len(snapshot.Messages) != 6 || len(snapshot.ToolCalls) != 2 ||
		snapshot.ToolCalls[created.SessionID+"-1-tool"].Output != expectedOutput || !refused.IsError || refused.Output != "" {
		return errors.New("ACP persisted prompt or tool output mismatch")
	}
	if err := c.verifyRelayFiles(ctx, second, connect, machine, created.SessionID, ticket, expectedOutput); err != nil {
		return err
	}
	if err := emit("relay_files", "Verified resumed upload, checksum, signed download and Range response"); err != nil {
		return err
	}
	return emit("tool_session_reported", created.SessionID)
}

func acpVerifyRPC(ctx context.Context, ws *websocket.Conn, id int, method string, params any, result any, permission ...string) (string, error) {
	if err := wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
		return "", err
	}
	var text strings.Builder
	for {
		var reply struct {
			ID     json.RawMessage
			Method string
			Result json.RawMessage
			Error  *struct{ Message string }
			Params struct {
				Update struct {
					SessionUpdate string
					Content       struct{ Text string }
				}
			}
		}
		if err := wsjson.Read(ctx, ws, &reply); err != nil {
			return "", err
		}
		if reply.Method == "session/request_permission" {
			if len(permission) != 1 {
				return "", errors.New("unexpected approval request")
			}
			outcome := map[string]string{"outcome": "selected", "optionId": permission[0]}
			permission = nil
			if err := wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": reply.ID,
				"result": map[string]any{"outcome": outcome}}); err != nil {
				return "", err
			}
			continue
		}
		if reply.Method == "session/update" && reply.Params.Update.SessionUpdate == "agent_message_chunk" {
			text.WriteString(reply.Params.Update.Content.Text)
		}
		if string(reply.ID) != fmt.Sprint(id) {
			continue
		}
		if len(permission) != 0 {
			return "", errors.New("expected approval request was missing")
		}
		if reply.Error != nil {
			return "", errors.New(reply.Error.Message)
		}
		if result != nil && json.Unmarshal(reply.Result, result) != nil {
			return "", errors.New("invalid ACP response")
		}
		return text.String(), nil
	}
}
