package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode"
)

func (h *relaySessionHost) sessionFilePreview(ctx context.Context, raw json.RawMessage) (any, error) {
	var p struct {
		workspaceRequest
		MetadataOnly bool
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, errors.New("invalid_request")
	}
	if !p.MetadataOnly {
		return h.workspaceQuery(ctx, "_monkeyai/workspace/read", raw)
	}
	return h.workspaceQuery(ctx, "_monkeyai/workspace/stat", raw)
}

func (h *relaySessionHost) sessionState(id string) (map[string]any, error) {
	if h.live == nil || !validUUID(id) {
		return nil, errors.New("session_not_found")
	}
	row, err := h.live.load(id)
	state := map[string]any{
		"files": []any{}, "tasks": []any{}, "plan": []any{}, "resourceIds": []string{},
		"resourcePreferences": (connectorPreferences{}).normalized(),
	}
	if err != nil {
		if h.local == nil && !h.shareDesktopSessions && h.sessionID != id {
			return nil, err
		}
		archive, archiveErr := readLegacySessionArchive(h.profile, id)
		if archiveErr != nil {
			return nil, err
		}
		page, archiveErr := legacyHistoryPage(archive)
		if archiveErr != nil {
			return nil, archiveErr
		}
		state["status"] = page.Session["status"]
		return map[string]any{"sessionId": id, "eventSeq": page.EventSeq, "state": state, "finish": false}, nil
	}
	if row.ConnectorPreferences != nil {
		state["resourcePreferences"] = row.ConnectorPreferences.normalized()
	}
	state["status"], state["goal"] = row.State, goalView(row)
	if row.GoalRun != "" && row.State == "running" {
		state["executionId"] = row.GoalRun
	}
	return map[string]any{"sessionId": id, "eventSeq": row.historySequence(), "state": state, "finish": false}, nil
}

func (l *acpLive) cancelTurn(id string) bool {
	l.mu.Lock()
	cancel := l.activeTurns[id]
	l.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (h *relaySessionHost) sessionDetail(r *http.Request, method string, raw json.RawMessage) (any, error) {
	var p struct {
		SessionID, TaskID, RequestID, Prompt, Action string
	}
	if json.Unmarshal(raw, &p) != nil || !validUUID(p.SessionID) {
		return nil, errors.New("invalid_session_parameters")
	}
	if method == "_monkeyai/session/state/get" {
		return h.sessionState(p.SessionID)
	}
	row, err := h.live.load(p.SessionID)
	if err != nil {
		return nil, err
	}
	switch method {
	case "_monkeyai/session/subscribe":
		return map[string]any{"sessionId": row.ID, "eventSeq": row.historySequence()}, nil
	case "_monkeyai/session/generate-title":
		title := strings.Join(strings.FieldsFunc(p.Prompt, func(r rune) bool {
			return unicode.IsControl(r) || unicode.IsSpace(r)
		}), " ")
		if title == "" {
			title = row.displayTitle()
		}
		runes := []rune(title)
		return map[string]any{"title": string(runes[:min(len(runes), 40)]), "titleSource": "prompt"}, nil
	case "_monkeyai/session/task":
		if p.Action == "cancel" {
			if h.acpScope(r, "session:control") != nil {
				return nil, errRelayAccess
			}
			n := len(row.Turns)
			if n > 0 && row.State == "running" && row.Turns[n-1].Operation == p.TaskID && h.live.cancelTurn(row.ID) {
				return map[string]any{"task": map[string]string{"id": p.TaskID, "status": "cancelled"}}, nil
			}
		} else if p.Action == "get" {
			if raw, err := h.historyToolOutput(historyQuery{SessionID: row.ID, ToolCallID: p.TaskID}); err == nil {
				output := raw.(map[string]any)["output"]
				return map[string]any{"task": map[string]any{"id": p.TaskID, "status": "completed", "output": output}, "output": output}, nil
			}
		}
		return nil, errors.New("session_task_not_found")
	case "_monkeyai/session/interaction/get", "_monkeyai/session/interaction/answer":
		// Pending tool approvals are handled over standard ACP. A saved
		// question from an earlier process cannot authorize a new action.
		return nil, errors.New("session_interaction_closed")
	}
	return nil, errors.New("invalid_session_parameters")
}
