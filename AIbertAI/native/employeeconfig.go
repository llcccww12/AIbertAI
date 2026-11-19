package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
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
	if err != nil || (l.protocol != "" && l.protocol != m.Protocol) {
		return nil, errors.New("model_unavailable")
	}
	item := map[string]any{"id": m.ID, "model": m.Model, "name": m.DisplayName, "display_name": m.DisplayName, "kind": m.Kind, "protocol": m.Protocol,
		"ready": true, "credit_multiplier": m.CreditMultiplier, "context_window_tokens": m.ContextWindowTokens}
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
	l.mu.Lock()
	preferences := l.connectorPrefs.normalized()
	l.mu.Unlock()
	return map[string]any{
		"viewRevision": preferences.Revision + 1, "models": map[string]any{"all": []any{item}, "default_model_id": m.ID,
			"thinking": map[string]any{}, "status": map[string]any{"state": "ok", "gatewayBaseUrl": l.client.server, "error": nil}},
		"mcp": empty, "mcpError": nil, "skills": empty, "skillPackages": empty, "experts": empty,
		"expertSummaries": empty, "expertIssues": map[string]any{}, "expertMissingSkills": map[string]any{},
		"resourceRules": empty, "resourceSkills": empty, "resourceSyncStatus": syncStatus,
		"resourcePreferences": preferences,
		"server":              map[string]any{"connectors": connectors, "experts": empty, "rules": empty, "skills": empty},
		"workspace":           map[string]any{"additional_directories": empty, "last_workspace": l.cwd},
		"capabilities":        map[string]any{}, "display": map[string]bool{"tool_summaries": true, "tool_use_summaries": true},
		"goals": map[string]int{"default_token_budget": 0}, "rules": "", "rulesReady": true, "engineDir": "", "skillsDir": "",
	}, nil
}
