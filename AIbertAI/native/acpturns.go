package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const maxACPToolCalls = 4

var errSessionReportPending = errors.New("session_report_pending")

type acpTurn struct {
	Operation     string              `json:"operationId"`
	ClientID      string              `json:"clientMessageId,omitempty"`
	Attachments   []acpAttachment     `json:"attachments,omitempty"`
	Started       time.Time           `json:"startedAt"`
	Ended         time.Time           `json:"endedAt,omitzero"`
	Prompt        string              `json:"prompt"`
	ModelInput    string              `json:"modelInput,omitempty"`
	ResponseItems []json.RawMessage   `json:"responseItems,omitempty"`
	Answer        string              `json:"answer,omitempty"`
	Tool          *reportedTool       `json:"tool,omitempty"`
	Call          *modelToolUse       `json:"modelToolCall,omitempty"`
	Output        string              `json:"toolOutput,omitempty"`
	ToolRuns      []workspaceToolRun  `json:"toolRuns,omitempty"`
	Usage         modelUsage          `json:"usage"`
	Permissions   reportedPermissions `json:"permissions,omitzero"`
	Failed        bool                `json:"failed,omitempty"`
	Acked         bool                `json:"acked,omitempty"`
}

// Older sessions recorded a single tool directly on the turn.
func (t acpTurn) workspaceTools() []workspaceToolRun {
	if len(t.ToolRuns) != 0 || t.Tool == nil {
		return t.ToolRuns
	}
	return []workspaceToolRun{{Call: t.Call, Report: *t.Tool, Output: t.Output, ResponseItems: t.ResponseItems}}
}

func (s acpSession) modelHistory() []toolConversationMessage {
	messages := make([]toolConversationMessage, 0, 2*len(s.Turns))
	for _, turn := range s.Turns {
		if turn.Answer != "" {
			input := turn.Prompt
			if turn.ModelInput != "" {
				input = turn.ModelInput
			}
			messages = append(messages, toolConversationMessage{Role: "user", Content: &input})
			for _, run := range turn.workspaceTools() {
				if run.Call != nil && run.Report.Failed == 0 {
					messages = append(messages, toolConversationMessage{Role: "assistant", ToolCalls: []modelToolUse{*run.Call}, ResponseItems: run.ResponseItems},
						toolConversationMessage{Role: "tool", Content: &run.Output, ToolCallID: run.Call.ID})
				}
			}
			messages = append(messages, toolConversationMessage{Role: "assistant", Content: &turn.Answer})
		}
	}
	return messages
}

func (l *acpLive) beginTurn(id, op, clientID, prompt string, attachments []acpAttachment) (acpSession, int, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	session, ok := l.sessions[id]
	if !ok || l.workspaces == nil && session.CWD != l.cwd || session.State == "running" || !validUUID(op) || (l.store != nil && session.Model != modelID) {
		return session, 0, false, errors.New("session_unavailable_for_prompt")
	}
	for index, turn := range session.Turns {
		if turn.Operation != op {
			continue
		}
		if turn.Prompt != prompt || turn.ClientID != clientID || !slices.Equal(turn.Attachments, attachments) || turn.Ended.IsZero() {
			return session, 0, false, errors.New("session_operation_conflict")
		}
		if !turn.Acked {
			session.State = "running"
			if err := l.save(session); err != nil {
				return session, 0, false, errors.New("session_write_failed")
			}
		}
		return session, index, true, nil
	}
	if n := len(session.Turns); n > 0 && !session.Turns[n-1].Acked {
		return session, 0, false, errSessionReportPending
	}
	started := time.Now().UTC()
	if started.Before(session.Updated) {
		started = session.Updated
	}
	session.State, session.Updated = "running", started
	if session.ConnectorPreferences == nil {
		prefs := l.connectorPrefs
		session.ConnectorPreferences = &prefs
	}
	index := len(session.Turns)
	session.Turns = append(slices.Clone(session.Turns), acpTurn{Operation: op, ClientID: clientID, Started: started, Prompt: prompt, Attachments: attachments})
	if err := l.save(session); err != nil {
		return session, 0, false, errors.New("session_write_failed")
	}
	return session, index, false, nil
}

func (h *relaySessionHost) submitACPTurn(ctx context.Context, id string, index int, turn acpTurn, register bool) error {
	live := h.live
	auth, err := live.authorization(ctx)
	if err != nil {
		return err
	}
	if register {
		if err := live.client.registerSession(ctx, auth, h.reportMachine(), id, turn.Started); err != nil {
			return err
		}
	}
	stop := "complete"
	if turn.Failed {
		stop = "error"
	}
	tools := turn.workspaceTools()
	facts := make([]reportedTool, 0, len(tools))
	for _, tool := range tools {
		facts = append(facts, tool.Report)
	}
	return live.client.reportTurn(ctx, auth, h.reportMachine(), id, index, index-1, turn.Started, turn.Ended, stop, turn.Usage, &turn.Permissions, facts...)
}

func (h *relaySessionHost) retryPendingReport(ctx context.Context, id string) error {
	live := h.live
	live.mu.Lock()
	session, ok := live.sessions[id]
	index := len(session.Turns) - 1
	if !ok || index < 0 || session.State == "running" || session.Turns[index].Ended.IsZero() {
		live.mu.Unlock()
		return errSessionReportPending
	}
	turn := session.Turns[index]
	if turn.Acked {
		live.mu.Unlock()
		return nil
	}
	// Reserve the session while replaying its durable facts after a disconnect.
	session.State = "running"
	if err := live.save(session); err != nil {
		live.mu.Unlock()
		return err
	}
	live.mu.Unlock()
	reportErr := h.submitACPTurn(ctx, id, index+1, turn, index == 0)
	live.mu.Lock()
	defer live.mu.Unlock()
	session = live.sessions[id]
	turn.Acked = reportErr == nil
	session.State = completedState(turn)
	session.Turns = slices.Clone(session.Turns)
	session.Turns[index] = turn
	return errors.Join(reportErr, live.save(session))
}

func (h *relaySessionHost) promptACP(ctx context.Context, ws *websocket.Conn, request *http.Request, id, op, clientID, prompt string, files ...acpAttachment) (any, error) {
	if op == "" {
		var err error
		op, err = newUUID()
		if err != nil {
			return nil, errors.New("session_operation_unavailable")
		}
	}
	live := h.live
	session, index, retry, err := live.beginTurn(id, op, clientID, prompt, files)
	if errors.Is(err, errSessionReportPending) && h.retryPendingReport(ctx, id) == nil {
		session, index, retry, err = live.beginTurn(id, op, clientID, prompt, files)
	}
	if err != nil {
		return nil, err
	}
	turn := session.Turns[index]
	register := index == 0 && retry
	if !retry {
		if index == 0 {
			var auth authorization
			auth, err = live.authorization(ctx)
			if err == nil {
				err = live.client.registerSession(ctx, auth, h.reportMachine(), id, turn.Started)
			}
			register = err != nil
		}
		if err == nil {
			err = h.runACPModelTurn(ctx, ws, request, session, &turn, files)
		}
		if len(turn.Answer) > 128<<10 {
			turn.Answer = ""
			err = errors.New("session_answer_too_large")
		}
		turn.Failed, turn.Ended = err != nil, time.Now().UTC()
		if turn.Ended.Before(turn.Started) {
			turn.Ended = turn.Started
		}
		live.mu.Lock()
		session.Turns = slices.Clone(session.Turns)
		session.Turns[index] = turn
		writeErr := live.save(session)
		live.mu.Unlock()
		if writeErr != nil {
			return nil, errors.New("session_write_failed")
		}
	}
	if !turn.Acked {
		reportErr := h.submitACPTurn(ctx, id, index+1, turn, register)
		turn.Acked = reportErr == nil
		session.State, session.Updated = completedState(turn), turn.Ended
		live.mu.Lock()
		session.Turns = slices.Clone(session.Turns)
		session.Turns[index] = turn
		writeErr := live.save(session)
		live.mu.Unlock()
		if writeErr != nil {
			return nil, errors.New("session_write_failed")
		}
	}
	if turn.Failed {
		return nil, errors.New("session_turn_failed")
	}
	if err := sendACPText(ctx, ws, id, "agent_message_chunk", turn.Answer); err != nil {
		return nil, err
	}
	return map[string]string{"stopReason": "end_turn"}, nil
}

func (h *relaySessionHost) runACPModelTurn(ctx context.Context, ws *websocket.Conn, request *http.Request, session acpSession, turn *acpTurn, files []acpAttachment) error {
	live := h.live
	id := session.ID
	modelPrompt, attachments, err := h.attachmentPrompt(ctx, id, turn.Prompt, files)
	if err != nil {
		return err
	}
	if modelPrompt != turn.Prompt {
		turn.ModelInput = modelPrompt
	}
	approved := append([]string{h.approvedFile}, slices.Sorted(maps.Keys(attachments))...)
	limit, allowWrite := 1, false
	var connectors map[string]connectorTool
	var workspace *workspaceToolPolicy
	var design json.RawMessage
	if live.workspaces != nil {
		approved = nil
		limit = maxACPToolCalls
		allowWrite = h.acpScope(request.WithContext(ctx), "files:write") == nil
		workspace = &workspaceToolPolicy{CanWrite: allowWrite}
		if h.local != nil && h.designBridge != nil && h.acpScope(request.WithContext(ctx), "files:read") == nil {
			if root, err := live.workspaces.sessionRoot(session); err == nil {
				design = workspaceDesign(root, id)
				root.Close()
			}
			workspace.HasDesign = design != nil
		}
		if session.ConnectorPreferences != nil {
			workspace.Preferences = session.ConnectorPreferences.builtinTools()
		}
		if session.ConnectorPreferences != nil && session.ConnectorPreferences.hasEnabledConnector() &&
			h.acpScope(request.WithContext(ctx), "resources:manage") == nil {
			auth, err := live.authorization(ctx)
			if err != nil {
				return err
			}
			connectors, err = live.client.activeConnectorTools(ctx, auth, *session.ConnectorPreferences)
			if err != nil {
				return err
			}
		}
	}
	budget := live.modelBudget
	if live.store == nil {
		budget = defaultModelInputBudget
	}
	turn.Answer, turn.Usage, turn.ToolRuns, err = live.client.completeWorkspaceToolTurn(ctx, live.protocol, live.key, id, approved, modelPrompt,
		session.modelHistory(), false, allowWrite, limit, connectors, workspace, budget, func(message toolConversationMessage, args toolArguments, step int) (output string, toolErr error) {
			call := message.ToolCalls[0]
			call.ID = fmt.Sprintf("%s-tool-%d", turn.Operation, step+1)
			defer func() {
				update := map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": call.ID,
					"status": "completed", "rawOutput": output}
				if toolErr != nil {
					update["status"], update["rawOutput"] = "failed", toolErr.Error()
				}
				_ = sendACPToolUpdate(ctx, ws, id, update)
			}()
			var permissions reportedPermissions
			var permissionErr error
			if tool := args.Connector; tool != nil {
				title := tool.ConnectorName + " / " + tool.Name
				permissions, permissionErr = requestACPToolApproval(ctx, ws, id, call.ID, title, tool.Name, "other",
					map[string]any{"connector": tool.ConnectorName, "tool": tool.Name, "arguments": args.JSON})
			} else if args.Design {
				permissions, permissionErr = requestACPToolApproval(ctx, ws, id, call.ID, call.Function.Name, call.Function.Name, workspaceToolKind(call.Function.Name), args.JSON)
			} else if args.Compat != nil {
				if args.Compat.Patch != nil {
					if h.acpScope(request.WithContext(ctx), "files:read") != nil ||
						h.acpScope(request.WithContext(ctx), "files:write") != nil {
						return "", errRelayAccess
					}
					plan, err := h.prepareToolWorkspacePatch(session, turn.Operation, args.Compat.Patch)
					if err != nil {
						return "", err
					}
					args.Compat.PatchPlan = &plan
				}
				permissions, permissionErr = requestACPToolApproval(ctx, ws, id, call.ID, workspaceCompatTitle(call.Function.Name), call.Function.Name, workspaceToolKind(call.Function.Name), args.JSON)
			} else {
				permissions, permissionErr = approveACPTool(ctx, ws, id, call, args.Workspace)
			}
			turn.Permissions.Asked += permissions.Asked
			turn.Permissions.Allowed += permissions.Allowed
			turn.Permissions.Denied += permissions.Denied
			if permissionErr != nil {
				return "", permissionErr
			}
			if args.Design {
				return h.designBridge.call(ctx, session, design, call.Function.Name, args.JSON)
			}
			return h.executeACPModelTool(ctx, request, session, turn.Operation, step, call.Function.Name, args, attachments)
		})
	return err
}

func (h *relaySessionHost) executeACPModelTool(ctx context.Context, request *http.Request, session acpSession, operation string, step int, name string, args toolArguments, attachments map[string]string) (string, error) {
	if tool := args.Connector; tool != nil {
		if err := h.acpScope(request.WithContext(ctx), "resources:manage"); err != nil {
			return "", errRelayAccess
		}
		auth, err := h.live.authorization(ctx)
		if err != nil {
			return "", err
		}
		return h.live.client.callConnector(ctx, auth, session.ID, operation, step, *tool, args.JSON)
	}
	if builtinWorkspaceWrite(name) {
		if err := h.acpScope(request.WithContext(ctx), "files:write"); err != nil {
			return "", errRelayAccess
		}
	}
	if args.Compat != nil {
		if err := h.acpScope(request.WithContext(ctx), "files:read"); err != nil {
			return "", errRelayAccess
		}
		return h.executeWorkspaceCompat(ctx, session, name, *args.Compat, attachments)
	}
	if name == workspaceWriteTool {
		return h.writeToolWorkspaceFile(ctx, session, args.Workspace.Path, *args.Workspace.Content)
	}
	if data, exists := attachments[args.Workspace.Path]; exists {
		return data, nil
	}
	return h.readToolWorkspaceFile(session, args.Workspace.Path)
}

func (h *relaySessionHost) readToolWorkspaceFile(row acpSession, path string) (string, error) {
	if h.approvedFile != "" && path == h.approvedFile && row.Workspace != "selected" {
		return readApprovedToolFile(h.root, h.approvedFile, path)
	}
	if h.live.workspaces == nil {
		return "", errWorkspacePath
	}
	root, err := h.live.workspaces.sessionRoot(row)
	if err != nil {
		return "", err
	}
	defer root.Close()
	return readWorkspaceText(root, path)
}

func workspaceToolKind(name string) string {
	if builtinWorkspaceWrite(name) || designWriteTool(name) {
		return "edit"
	}
	if name == legacyGrepTool || name == legacyGlobTool || name == directoryTool {
		return "search"
	}
	return "read"
}

func workspaceCompatTitle(name string) string {
	switch name {
	case legacyWriteTool:
		return "写入工作区文件"
	case legacyEditTool:
		return "修改工作区文件"
	case legacyPatchTool:
		return "应用工作区补丁"
	case legacyGrepTool:
		return "搜索工作区内容"
	case legacyGlobTool, directoryTool:
		return "查找工作区文件"
	default:
		return "读取工作区文件"
	}
}

func sendACPToolUpdate(ctx context.Context, ws *websocket.Conn, id string, update map[string]any) error {
	return wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "method": "session/update", "params": map[string]any{
		"sessionId": id, "update": update,
	}})
}

func approveACPTool(ctx context.Context, ws *websocket.Conn, sessionID string, call modelToolUse, args workspaceToolArguments) (reportedPermissions, error) {
	title := "读取会话文件"
	input := map[string]string{"path": args.Path}
	if args.Content != nil {
		title = "写入工作区文件"
		input["content"] = *args.Content
	}
	return requestACPToolApproval(ctx, ws, sessionID, call.ID, title, call.Function.Name, workspaceToolKind(call.Function.Name), input)
}

func requestACPToolApproval(ctx context.Context, ws *websocket.Conn, sessionID, callID, title, name, kind string, input any) (reportedPermissions, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	permissionID := "permission-" + callID
	if err := sendACPToolUpdate(ctx, ws, sessionID, map[string]any{"sessionUpdate": "tool_call", "toolCallId": callID,
		"title": title, "name": name, "kind": kind, "status": "pending", "rawInput": input}); err != nil {
		return reportedPermissions{}, errors.New("tool_approval_unavailable")
	}
	err := wsjson.Write(ctx, ws, map[string]any{
		"jsonrpc": "2.0", "id": permissionID, "method": "session/request_permission",
		"params": map[string]any{"sessionId": sessionID,
			"toolCall": map[string]any{"toolCallId": callID, "title": title, "name": name,
				"kind": kind, "status": "pending", "rawInput": input},
			"options": []map[string]string{
				{"optionId": "allow", "kind": "allow_once", "name": "允许一次"},
				{"optionId": "deny", "kind": "reject_once", "name": "拒绝"},
			},
		},
	})
	if err != nil {
		return reportedPermissions{}, errors.New("tool_approval_unavailable")
	}
	counts := reportedPermissions{Asked: 1}
	for {
		var reply struct {
			JSONRPC string `json:"jsonrpc"`
			ID      json.RawMessage
			Method  string
			Result  struct {
				Outcome struct{ Outcome, OptionID string }
			}
			Error json.RawMessage
		}
		if wsjson.Read(ctx, ws, &reply) != nil {
			return counts, errors.New("tool_approval_unavailable")
		}
		if reply.JSONRPC != "2.0" {
			continue
		}
		if reply.Method != "" && len(reply.ID) != 0 && string(reply.ID) != "null" {
			if wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": reply.ID,
				"error": map[string]any{"code": -32000, "message": "session_waiting_for_permission"}}) != nil {
				return counts, errors.New("tool_approval_unavailable")
			}
			continue
		}
		var id string
		if reply.Method != "" || json.Unmarshal(reply.ID, &id) != nil || id != permissionID {
			continue
		}
		if len(reply.Error) != 0 || reply.Result.Outcome.Outcome != "selected" {
			return counts, errors.New("tool_permission_cancelled")
		}
		if reply.Result.Outcome.OptionID == "deny" {
			counts.Denied = 1
			return counts, errors.New("tool_permission_denied")
		}
		if reply.Result.Outcome.OptionID != "allow" {
			return counts, errors.New("invalid_tool_permission")
		}
		counts.Allowed = 1
		return counts, nil
	}
}

func (session acpSession) historySequence() int {
	count := len(session.Turns)
	for _, turn := range session.Turns {
		if turn.hasHistoryResponse() {
			count++
		}
	}
	return count
}

func (t acpTurn) hasHistoryResponse() bool {
	return t.Answer != "" || len(t.ToolRuns) > 0 || t.Tool != nil
}

func (session acpSession) toolHistoryID(turn acpTurn, index, step int) string {
	if len(turn.ToolRuns) == 0 {
		return fmt.Sprintf("%s-%d-tool", session.ID, index+1)
	}
	return fmt.Sprintf("%s-tool-%d", turn.Operation, step+1)
}

func (session acpSession) historyWindow(query historyQuery, auth *relayAuthorization) (map[string]any, error) {
	count := session.historySequence()
	start, end, older, newer, err := query.window("source", count)
	if err != nil {
		return nil, err
	}
	messages := []map[string]any{}
	calls := map[string]any{}
	position := 0
	for index, turn := range session.Turns {
		userPosition := position
		position++
		if turn.hasHistoryResponse() {
			position++
		}
		if position <= start || userPosition >= end {
			continue
		}
		id := fmt.Sprintf("%s-%d", session.ID, index+1)
		if userPosition >= start {
			userParts := []map[string]any{{"type": "text", "text": turn.Prompt}}
			for _, file := range turn.Attachments {
				part := map[string]any{"type": "file", "name": file.Name}
				if file.URI != "" {
					if auth == nil {
						part["uri"] = file.URI
						part["employeeAttachment"] = true
					} else if path := historyFilePath(file.URI, file.Name); path != "" {
						part["fileRef"] = historyRelayRef(*auth, session.ID, path)
					}
				} else {
					part["fileRef"] = file.Ref
				}
				userParts = append(userParts, part)
			}
			user := map[string]any{"id": id + "-prompt", "role": "user", "at": turn.Started.UnixMilli(), "parts": userParts}
			if turn.ClientID != "" {
				user["queueMessageId"] = turn.ClientID
			}
			messages = append(messages, user)
		}
		if !turn.hasHistoryResponse() || userPosition+1 >= end {
			continue
		}
		parts := []map[string]string{}
		for step, run := range turn.workspaceTools() {
			callID, state := session.toolHistoryID(turn, index, step), "completed"
			if run.Report.Failed > 0 {
				state = "failed"
			}
			kind := workspaceToolKind(run.Report.Name)
			if run.Report.Category == "connector" {
				kind = "other"
			}
			calls[callID] = map[string]any{"id": callID, "name": run.Report.Name, "title": run.Report.Name, "kind": kind, "status": state,
				"output": run.Output, "isError": run.Report.Failed > 0}
			parts = append(parts, map[string]string{"type": "tool_call", "toolCallId": callID})
		}
		if turn.Answer != "" {
			parts = append(parts, map[string]string{"type": "text", "text": turn.Answer})
		}
		if len(parts) > 0 {
			messages = append(messages, map[string]any{"id": id + "-response", "role": "assistant", "at": turn.Ended.UnixMilli(), "parts": parts})
		}
	}
	model := session.Model
	if model == "" {
		model = modelID
	}
	return map[string]any{"session": map[string]any{"id": session.ID, "cwd": session.CWD, "title": "AIbertAI 会话", "status": session.State,
		"createdAt": session.Created, "updatedAt": session.Updated, "model": model, "mode": "auto",
		"isDefaultWorkspace": session.Workspace != "selected", "projectId": session.ProjectID}, "messages": messages,
		"toolCalls": calls, "eventSeq": count, "olderCursor": older, "newerCursor": newer}, nil
}
