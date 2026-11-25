package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"os"
	"slices"
	"strings"
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
	if err != nil {
		// The code alone gave no clue why a first start failed on Windows.
		fmt.Fprintf(os.Stderr, "model config unavailable: %v\n", err)
		return nil, errors.New("model_unavailable")
	}
	if h.local == nil && l.protocol != "" && l.protocol != m.Protocol {
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
	rules, ruleErr := l.client.desktopCatalog(ctx, auth, "rules")
	if ruleErr != nil {
		rules = []json.RawMessage{}
		syncStatus["rules"] = map[string]any{"state": "error", "error": ruleErr.Error()}
	}
	// The conversation rule picker reads resourceRules, keyed like the
	// preferences a turn applies (rule/server/<id>).
	resourceRules := []any{}
	for _, raw := range rules {
		var rule map[string]any
		if json.Unmarshal(raw, &rule) != nil || rule == nil || rule["enabled"] == false {
			continue
		}
		id, _ := rule["id"].(string)
		if !validUUID(id) {
			continue
		}
		required, _ := rule["required"].(bool)
		item := map[string]any{"id": "rule/server/" + id, "referenceId": id, "name": rule["name"], "content": rule["content"],
			"description": rule["description"], "required": required, "source": "team"}
		for _, key := range []string{"scope", "origin", "user", "ownership_type", "name_i18n", "description_i18n"} {
			if value, ok := rule[key]; ok {
				item[key] = value
			}
		}
		if rule["ownership_type"] == "user" {
			item["source"] = "personal"
		}
		resourceRules = append(resourceRules, item)
	}
	skills, skillErr := l.client.desktopSkillCatalog(ctx, auth)
	if skillErr != nil {
		syncStatus["skills"] = map[string]any{"state": "error", "error": skillErr.Error()}
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
			"id": designExpertID, "name": "创作模式", "description": "在画布上生成图片、视频、网页原型、演示文稿和 Logo", "skills": empty, "builtin": true,
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
		"resourceRules": resourceRules, "resourceSkills": empty, "resourceSyncStatus": syncStatus,
		"resourcePreferences": preferences,
		"server":              map[string]any{"connectors": connectors, "experts": experts, "rules": rules, "skills": skills},
		"workspace":           map[string]any{"additional_directories": empty, "last_workspace": l.cwd},
		"capabilities":        map[string]any{}, "display": settings.display(),
		"goals": map[string]any{"default_token_budget": settings.goalBudget(), "execution_supported": false},
		"rules": "", "rulesReady": true, "engineDir": "", "skillsDir": "",
	}, nil
}

func (c *client) desktopSkillCatalog(ctx context.Context, auth authorization) ([]map[string]any, error) {
	rows, err := c.desktopCatalog(ctx, auth, "skills")
	skills := make([]map[string]any, 0, len(rows))
	if err != nil {
		return skills, err
	}
	for _, raw := range rows {
		var item map[string]any
		if json.Unmarshal(raw, &item) != nil {
			return []map[string]any{}, errors.New("skill_catalog_invalid")
		}
		id, _ := item["id"].(string)
		name, _ := item["name"].(string)
		if !validUUID(id) || name == "" {
			return []map[string]any{}, errors.New("skill_catalog_invalid")
		}
		item["resourceId"] = id
		item["version"] = item["package_sha256"]
		if skillCommandName.MatchString(name) && !strings.EqualFold(name, "goal") {
			item["command_name"] = name
			item["invokable"] = true
		} else {
			item["invokable"] = false
		}
		if _, ok := item["description"].(string); !ok {
			item["description"] = ""
		}
		if _, ok := item["tags"].([]any); !ok {
			item["tags"] = []any{}
		}
		skills = append(skills, item)
	}
	return skills, nil
}
