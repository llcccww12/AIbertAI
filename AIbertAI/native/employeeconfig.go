package main

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"
)

func (h *relaySessionHost) readConfig(w http.ResponseWriter, r *http.Request, auth relayAuthorization) {
	var request struct{ Method string }
	if h.live == nil || json.NewDecoder(r.Body).Decode(&request) != nil || request.Method != "_monkeyai/config/get" {
		http.NotFound(w, r)
		return
	}
	result, err := h.employeeConfig(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	writeRelayResult(w, auth, result)
}

func (h *relaySessionHost) employeeConfig(ctx context.Context) (any, error) {
	l := h.live
	if l == nil {
		return nil, errors.New("config_unavailable")
	}
	auth, err := l.authorization(ctx)
	if err != nil {
		return nil, errors.New("model_catalog_unavailable")
	}
	m, err := l.client.selectedModelConfig(ctx, auth)
	if err != nil || (h.local == nil && l.protocol != "" && l.protocol != m.Protocol) {
		return nil, errors.New("model_unavailable")
	}
	models, err := l.client.desktopCatalog(ctx, auth, "models")
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(models))
	for _, raw := range models {
		var item map[string]any
		if json.Unmarshal(raw, &item) != nil || item == nil {
			return nil, errors.New("model_catalog_invalid")
		}
		item["ready"] = item["id"] == m.ID || !l.modelLocked && item["kind"] == "text" &&
			(item["protocol"] == "openai_chat_completions" || item["protocol"] == "openai_responses")
		if h.local != nil && !l.modelLocked && (item["kind"] == "image" || item["kind"] == "video") {
			item["ready"] = true
		}
		item["name"] = item["display_name"]
		if item["ownership_type"] == "user" {
			if owner, ok := item["user"].(map[string]any); ok && h.local != nil && owner["id"] == h.local.userID {
				item["source"] = "mine"
			}
		}
		items = append(items, item)
	}
	empty := []any{}
	syncStatus := map[string]any{}
	for _, name := range []string{"connectors", "experts", "rules", "skills"} {
		syncStatus[name] = map[string]any{"state": "ok", "error": nil}
	}
	connectors, connectorErr := l.client.employeeConnectors(ctx, auth)
	if connectorErr != nil {
		connectors = []json.RawMessage{}
		syncStatus["connectors"] = map[string]any{"state": "error", "error": connectorErr.Error()}
	}
	experts, expertErr := l.client.desktopCatalog(ctx, auth, "experts")
	if expertErr != nil {
		experts = []json.RawMessage{}
		syncStatus["experts"] = map[string]any{"state": "error", "error": expertErr.Error()}
	}
	l.mu.Lock()
	preferences := l.connectorPrefs.normalized()
	settings := l.settings
	l.mu.Unlock()
	thinking := make(map[string]modelThinking, len(settings.ModelThinking))
	maps.Copy(thinking, settings.ModelThinking)
	expertSummaries := []any{}
	for _, raw := range experts {
		var summary map[string]any
		if json.Unmarshal(raw, &summary) != nil || summary == nil || summary["available"] == false {
			continue
		}
		summary["skills"] = empty
		expertSummaries = append(expertSummaries, summary)
	}
	if h.local != nil && h.designBridge != nil {
		expertSummaries = append(expertSummaries, map[string]any{
			"id": designExpertID, "name": "设计专家", "description": "网页原型、演示文稿、海报、Logo 与视频分镜", "skills": empty, "builtin": true,
		})
	}
	localMCP := []any{}
	localRevision := 0
	if imported := h.runtimeResources(); h.local != nil && imported != nil {
		localRevision = imported.Revision
		for _, id := range slices.Sorted(maps.Keys(imported.Local)) {
			entry := imported.Local[id]
			env, _ := localMCPEnvValues(entry.Env)
			localMCP = append(localMCP, map[string]any{"name": entry.Name, "display_name": entry.DisplayName,
				"transport": "stdio", "command": entry.Command, "args": entry.Args,
				"maskedKeys": map[string]any{"env": slices.Sorted(maps.Keys(env))},
				"enabled":    entry.Enabled == nil || *entry.Enabled, "tool_states": entry.ToolStates, "tool_filter": entry.Filter})
		}
	}
	return map[string]any{
		"viewRevision": preferences.Revision + settings.Revision + localRevision + 1, "sourceRuntime": true, "models": map[string]any{"all": items, "default_model_id": m.ID,
			"thinking": thinking, "status": map[string]any{"state": "ok", "gatewayBaseUrl": l.client.server, "error": nil}},
		"mcp": localMCP, "mcpError": nil, "skills": empty, "skillPackages": empty, "experts": empty,
		"expertSummaries": expertSummaries, "expertIssues": map[string]any{}, "expertMissingSkills": map[string]any{},
		"resourceRules": empty, "resourceSkills": empty, "resourceSyncStatus": syncStatus,
		"resourcePreferences": preferences,
		"server":              map[string]any{"connectors": connectors, "experts": experts, "rules": empty, "skills": empty},
		"workspace":           map[string]any{"additional_directories": empty, "last_workspace": l.cwd},
		"capabilities":        map[string]any{}, "display": settings.display(),
		"goals": map[string]any{"default_token_budget": settings.goalBudget(), "execution_supported": false},
		"rules": "", "rulesReady": true, "engineDir": "", "skillsDir": "",
	}, nil
}
