package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/coder/websocket"
)

const dshKnowledgeSearchTool = "knowledge_search"

// Empty IDs must never reach the backend: there they mean all accessible libraries.
func dshKnowledgeCatalog(ids []string) []desktopDesignTool {
	if len(ids) == 0 || !validKnowledgeSelection(ids) {
		return nil
	}
	return []desktopDesignTool{{Name: dshKnowledgeSearchTool, Description: "Search the knowledge bases selected for this conversation and return citations.", Schema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","minLength":1,"maxLength":4096}},"required":["query"],"additionalProperties":false}`)}}
}

func (h *relaySessionHost) dshKnowledgeBridge(ctx context.Context, ws *websocket.Conn, request *http.Request, session acpSession, turn *acpTurn, ids []string) func(json.RawMessage) (any, error) {
	selected := slices.Clone(ids)
	var mu sync.Mutex
	seen := map[string]bool{}
	return func(raw json.RawMessage) (any, error) {
		mu.Lock()
		defer mu.Unlock()
		var req dshMutation
		if len(selected) == 0 || !validKnowledgeSelection(selected) || json.Unmarshal(raw, &req) != nil || req.SessionID != session.DSHSessionID || req.CallID == "" || seen[req.CallID] || req.Name != dshKnowledgeSearchTool {
			return nil, errors.New("dsh_knowledge_request_invalid")
		}
		seen[req.CallID] = true
		var run *workspaceToolRun
		for i := range turn.ToolRuns {
			if turn.ToolRuns[i].DSHCallID == req.CallID {
				run = &turn.ToolRuns[i]
				break
			}
		}
		if run == nil || !run.Pending || run.Call == nil || run.Call.ID == "" || run.Call.Function.Name != req.Name {
			return nil, errors.New("dsh_knowledge_request_invalid")
		}
		callID := run.Call.ID
		var original, requested any
		if json.Unmarshal([]byte(run.Call.Function.Arguments), &original) != nil || json.Unmarshal(req.Arguments, &requested) != nil {
			return nil, errors.New("dsh_knowledge_request_invalid")
		}
		a, _ := json.Marshal(original)
		b, _ := json.Marshal(requested)
		if !bytes.Equal(a, b) {
			return nil, errors.New("dsh_knowledge_request_invalid")
		}
		var input struct {
			Query string `json:"query"`
		}
		if decodeWorkspaceTool(req.Arguments, &input) != nil || strings.TrimSpace(input.Query) == "" || len(input.Query) > 4096 {
			return nil, errors.New("dsh_knowledge_request_invalid")
		}
		check := func() error {
			if h.local == nil || h.live == nil || h.live.client == nil || request == nil || h.acpScope(request.WithContext(ctx), "resources:manage") != nil {
				return errRelayAccess
			}
			current, err := h.live.load(session.ID)
			if err != nil || current.Engine != "dsh" || current.DSHSessionID != session.DSHSessionID || current.CWD != session.CWD || current.Workspace != session.Workspace || current.ProjectID != session.ProjectID || current.ResourceRevision != session.ResourceRevision || len(current.Turns) == 0 || current.Turns[len(current.Turns)-1].Operation != turn.Operation {
				return errors.New("dsh_knowledge_session_changed")
			}
			h.live.mu.Lock()
			prefs := h.live.connectorPrefs
			h.live.mu.Unlock()
			if current.ConnectorPreferences != nil {
				prefs = *current.ConnectorPreferences
			}
			if !slices.Equal(prefs.KnowledgeBaseIDs, selected) {
				return errors.New("dsh_knowledge_session_changed")
			}
			matched := false
			for i := range turn.ToolRuns {
				item := &turn.ToolRuns[i]
				if item.DSHCallID != req.CallID {
					continue
				}
				var value any
				if item.Pending && item.Call != nil && item.Call.ID == callID && item.Call.Function.Name == req.Name && json.Unmarshal([]byte(item.Call.Function.Arguments), &value) == nil {
					encoded, _ := json.Marshal(value)
					matched = bytes.Equal(encoded, b)
				}
				break
			}
			if !matched {
				return errors.New("dsh_knowledge_request_invalid")
			}
			return ctx.Err()
		}
		if err := check(); err != nil {
			return nil, err
		}
		// Selection and the backend's per-library authorization govern this
		// read, matching the existing workspace read experience.
		if err := check(); err != nil {
			return nil, err
		}
		auth, err := h.live.authorization(ctx)
		if err != nil {
			return nil, err
		}
		var response struct {
			Items         []json.RawMessage `json:"items"`
			Query         string            `json:"query"`
			RetrievalMode string            `json:"retrieval_mode"`
			Channels      json.RawMessage   `json:"channels_used"`
			Fusion        json.RawMessage   `json:"fusion"`
			Degraded      bool              `json:"degraded"`
			Warnings      json.RawMessage   `json:"warnings"`
		}
		if err = h.live.client.json(ctx, http.MethodPost, "/api/v1/kb/search", auth.Access, "", map[string]any{"query": input.Query, "kb_ids": selected}, &response); err != nil {
			return nil, err
		}
		if len(response.Items) > 20 {
			return nil, errors.New("dsh_knowledge_response_invalid")
		}
		for _, item := range response.Items {
			var record struct {
				KBID       string          `json:"kb_id"`
				DocumentID string          `json:"document_id"`
				ChunkID    string          `json:"chunk_id"`
				Citation   json.RawMessage `json:"citation"`
			}
			var citation map[string]json.RawMessage
			if json.Unmarshal(item, &record) != nil || !slices.Contains(selected, record.KBID) || !validUUID(record.DocumentID) || !validUUID(record.ChunkID) || json.Unmarshal(record.Citation, &citation) != nil || citation == nil || citation["heading_path"] == nil || citation["page_no"] == nil || citation["source_url"] == nil {
				return nil, errors.New("dsh_knowledge_response_invalid")
			}
		}
		if err = check(); err != nil {
			return nil, err
		}
		output, err := json.Marshal(response)
		if err != nil || len(output) > 256<<10 {
			return nil, errors.New("dsh_knowledge_response_invalid")
		}
		return map[string]string{"output": string(output)}, nil
	}
}
