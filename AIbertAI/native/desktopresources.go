package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (c *client) desktopCatalog(ctx context.Context, auth authorization, kind string) ([]json.RawMessage, error) {
	items := []json.RawMessage{}
	for page := 1; page <= 100; page++ {
		var result struct {
			Models, Experts, Skills []json.RawMessage
			Total                   int `json:"total_count"`
		}
		path := fmt.Sprintf("/api/v1/%s?page=%d&page_size=100", kind, page)
		if err := c.json(ctx, http.MethodGet, path, auth.Access, "", nil, &result); err != nil {
			return nil, err
		}
		rows := map[string][]json.RawMessage{
			"models": result.Models, "experts": result.Experts, "skills": result.Skills,
		}[kind]
		items = append(items, rows...)
		if len(rows) == 0 || len(items) >= result.Total {
			return items, nil
		}
	}
	return nil, errors.New("resource_catalog_too_large")
}

func (h *relaySessionHost) desktopModelSettings(ctx context.Context, method string, raw json.RawMessage) (any, error) {
	l := h.live
	auth, err := l.authorization(ctx)
	if err != nil {
		return nil, err
	}
	var p struct {
		ID     string
		Input  json.RawMessage
		Inputs []json.RawMessage
	}
	if json.Unmarshal(raw, &p) != nil || p.ID != "" && !validUUID(p.ID) {
		return nil, errors.New("invalid_request")
	}
	const base = "/api/v1/models"
	op := knowledgeOperation{Path: base}
	if p.ID != "" {
		op.Path += "/" + p.ID
	}
	switch method {
	case "_monkeyai/config/models/discover":
		return requestModelDraft(ctx, raw, true)
	case "_monkeyai/config/models/test":
		if p.ID == "" {
			return requestModelDraft(ctx, raw, false)
		}
		model, err := l.client.modelConfig(ctx, auth, p.ID)
		if err != nil {
			return nil, err
		}
		probe, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		_, _, err = l.client.chat(probe, model.Protocol, l.key, "", model.ID, "Reply OK.", model.StreamMode != "off", func(string) error { return nil })
		if err != nil {
			return map[string]any{"ok": false, "message": err.Error()}, nil
		}
		return map[string]any{"ok": true, "model": model.Model}, nil
	case "_monkeyai/config/models/get":
		if p.ID == "" {
			return nil, errors.New("invalid_request")
		}
		op.Verb = http.MethodGet
	case "_monkeyai/config/models/save":
		if !desktopModelInput(p.Input) {
			return nil, errors.New("model_config_invalid")
		}
		op.Verb, op.Body = http.MethodPost, p.Input
		if p.ID != "" {
			op.Verb = http.MethodPut
		}
	case "_monkeyai/config/models/delete":
		if p.ID == "" {
			return nil, errors.New("invalid_request")
		}
		if p.ID == modelID {
			return nil, errors.New("model_in_use")
		}
		op.Verb = http.MethodDelete
	case "_monkeyai/config/models/create-many":
		if len(p.Inputs) == 0 || len(p.Inputs) > 50 {
			return nil, errors.New("invalid_request")
		}
		for _, input := range p.Inputs {
			if !desktopModelInput(input) {
				return nil, errors.New("model_config_invalid")
			}
		}
		results := make([]map[string]any, 0, len(p.Inputs))
		for i, input := range p.Inputs {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			_, saveErr := l.client.knowledgeChange(ctx, auth, knowledgeOperation{Verb: http.MethodPost, Path: base, Body: input})
			row := map[string]any{"index": i}
			if saveErr != nil {
				row["error"] = saveErr.Error()
			}
			results = append(results, row)
		}
		config, err := h.employeeConfig(ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{"config": config, "results": results}, nil
	default:
		return nil, errors.New("method_not_found")
	}
	result, err := l.client.knowledgeChange(ctx, auth, op)
	if err != nil || op.Verb == http.MethodGet {
		return result, err
	}
	return h.employeeConfig(ctx)
}

func desktopModelInput(input json.RawMessage) bool {
	var p struct {
		ModelID     string `json:"model_id"`
		DisplayName string `json:"display_name"`
		BaseURL     string `json:"base_url"`
	}
	return json.Unmarshal(input, &p) == nil && strings.TrimSpace(p.ModelID) != "" &&
		strings.TrimSpace(p.DisplayName) != "" && strings.TrimSpace(p.BaseURL) != ""
}

func (h *relaySessionHost) desktopResource(ctx context.Context, method string, raw json.RawMessage) (any, error) {
	var p struct {
		Kind, ID, ETag, CredentialID, Name string
		Input                              json.RawMessage
	}
	if json.Unmarshal(raw, &p) != nil || p.ID != "" && !validUUID(p.ID) {
		return nil, errors.New("invalid_request")
	}
	if p.CredentialID != "" && !validUUID(p.CredentialID) {
		return nil, errors.New("invalid_request")
	}
	op := knowledgeOperation{Verb: http.MethodGet}
	refresh := false
	switch method {
	case "_monkeyai/resources/connectors/test", "_monkeyai/resources/connectors/authorize",
		"_monkeyai/resources/connectors/authorization", "_monkeyai/resources/connectors/credentials",
		"_monkeyai/resources/connectors/credentials/get", "_monkeyai/resources/connectors/credentials/create",
		"_monkeyai/resources/connectors/credentials/update", "_monkeyai/resources/connectors/credentials/revoke":
		if p.ID == "" {
			return nil, errors.New("invalid_request")
		}
		op.Path = connectorPath("/api/v1/connectors/", p.ID, p.CredentialID)
		switch method {
		case "_monkeyai/resources/connectors/test":
			op.Verb, op.Path, refresh = http.MethodPost, op.Path+"/test", true
		case "_monkeyai/resources/connectors/authorize":
			op.Verb, op.Path, op.ETag = http.MethodPost, op.Path+"/oauth/authorizations", p.ETag
			op.Body = map[string]string{"name": p.Name}
		case "_monkeyai/resources/connectors/authorization":
			op.Path = "/api/v1/connector-authorizations/" + p.ID
		case "_monkeyai/resources/connectors/credentials":
			op.Path = "/api/v1/connectors/" + p.ID + "/credentials"
		case "_monkeyai/resources/connectors/credentials/get":
			if p.CredentialID == "" {
				return nil, errors.New("invalid_request")
			}
		case "_monkeyai/resources/connectors/credentials/create":
			op.Verb, op.Path, op.Body, refresh = http.MethodPost, "/api/v1/connectors/"+p.ID+"/credentials", p.Input, true
		case "_monkeyai/resources/connectors/credentials/update":
			if p.CredentialID == "" {
				return nil, errors.New("invalid_request")
			}
			op.Verb, op.Body, op.ETag, refresh = http.MethodPatch, p.Input, p.ETag, true
		case "_monkeyai/resources/connectors/credentials/revoke":
			if p.CredentialID == "" {
				return nil, errors.New("invalid_request")
			}
			op.Verb, op.ETag, refresh = http.MethodDelete, p.ETag, true
		}
	case "_monkeyai/resources/experts/details":
		if p.ID == "" {
			return nil, errors.New("invalid_request")
		}
		op.Path = "/api/v1/experts/" + p.ID + "/manifest"
	case "_monkeyai/resources/experts/tags":
		op.Path = "/api/v1/experts/tags"
	case "_monkeyai/resources/connectors/tags":
		op.Path = "/api/v1/connectors/tags"
	case "_monkeyai/resources/skills/content", "_monkeyai/resources/skills/manifest":
		if p.ID == "" {
			return nil, errors.New("invalid_request")
		}
		op.Path = "/api/v1/skills/" + p.ID + "/" + strings.TrimPrefix(method, "_monkeyai/resources/skills/")
	case "_monkeyai/resources/skills/tags":
		op.Path = "/api/v1/skills/tags"
	case "_monkeyai/resources/get", "_monkeyai/resources/save", "_monkeyai/resources/delete":
		switch p.Kind {
		case "expert", "connector", "rule":
			op.Path = "/api/v1/" + p.Kind + "s"
		default:
			return nil, errors.New("resource_operation_unavailable")
		}
		if p.ID != "" {
			op.Path += "/" + p.ID
		}
		switch method {
		case "_monkeyai/resources/get":
			op.Detail = true
		case "_monkeyai/resources/save":
			var input map[string]json.RawMessage
			if json.Unmarshal(p.Input, &input) != nil || input == nil {
				return nil, errors.New("invalid_request")
			}
			op.Verb, op.Body, op.ETag, refresh = http.MethodPost, p.Input, p.ETag, true
			op.Detail = true
			if p.ID != "" {
				op.Verb = http.MethodPut
			}
		case "_monkeyai/resources/delete":
			op.Verb, op.ETag, refresh = http.MethodDelete, p.ETag, true
		}
		if p.ID == "" && op.Verb != http.MethodPost {
			return nil, errors.New("invalid_request")
		}
	default:
		return nil, errors.New("method_not_found")
	}
	auth, err := h.live.authorization(ctx)
	if err != nil {
		return nil, err
	}
	result, err := h.live.client.knowledgeChange(ctx, auth, op)
	if err != nil {
		return nil, err
	}
	if method == "_monkeyai/resources/connectors/credentials" {
		var list struct{ Items []json.RawMessage }
		payload, ok := result.(json.RawMessage)
		if !ok || json.Unmarshal(payload, &list) != nil || list.Items == nil {
			return nil, errors.New("connector_credentials_invalid")
		}
		return list.Items, nil
	}
	if !refresh {
		return result, nil
	}
	config, err := h.employeeConfig(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"result": result, "config": config}, nil
}

func (h *relaySessionHost) desktopSelectedModel(ctx context.Context, id string) (string, error) {
	l := h.live
	if !validUUID(id) || l.modelLocked && id != modelID {
		return "", errors.New("model_unavailable")
	}
	if id == modelID {
		return id, nil
	}
	auth, err := l.authorization(ctx)
	if err != nil {
		return "", err
	}
	model, err := l.client.modelConfig(ctx, auth, id)
	if err != nil {
		return "", err
	}
	return model.ID, nil
}

func (h *relaySessionHost) setDesktopSessionModel(ctx context.Context, method string, raw json.RawMessage) (any, error) {
	var p struct {
		SessionID, Model string
		Thinking         *modelThinking
	}
	if json.Unmarshal(raw, &p) != nil || !validUUID(p.SessionID) {
		return nil, errors.New("invalid_session_parameters")
	}
	if p.Thinking != nil && !p.Thinking.valid() {
		return nil, errors.New("invalid_model_thinking")
	}
	changeModel := method == "_monkeyai/session/model/set"
	if changeModel {
		if _, err := h.desktopSelectedModel(ctx, p.Model); err != nil {
			return nil, err
		}
	}
	l := h.live
	l.mu.Lock()
	defer l.mu.Unlock()
	row, ok := l.sessions[p.SessionID]
	if !ok {
		return nil, errors.New("session_not_found")
	}
	if row.State == "running" || len(row.Turns) > 0 && !row.Turns[len(row.Turns)-1].Acked {
		return nil, errors.New("session_unavailable_for_model_change")
	}
	if changeModel && row.Engine == "dsh" && row.Model != p.Model && len(row.Turns) > 0 {
		last := row.Turns[len(row.Turns)-1]
		if last.Failed || last.Interrupted {
			return nil, errors.New("dsh_reconciliation_required")
		}
	}
	if changeModel {
		row.Model = p.Model
	}
	row.Thinking = p.Thinking
	if err := l.save(row); err != nil {
		return nil, err
	}
	return map[string]any{"model": row.Model, "thinking": row.Thinking}, nil
}
