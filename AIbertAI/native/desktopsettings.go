package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const desktopSettingsFile = "desktop-settings.json"
const defaultGoalBudget = 200_000_000
const maxGoalBudget = 1<<53 - 1

type desktopSettings struct {
	Revision         int
	GoalBudget       *int64
	ToolSummaries    *bool
	ToolUseSummaries *bool
	ModelThinking    map[string]modelThinking `json:",omitempty"`
}

type modelThinking struct {
	Enabled *bool  `json:"enabled"`
	Effort  string `json:"effort,omitempty"`
}

func (p *modelThinking) valid() bool {
	if p == nil || p.Enabled == nil || !*p.Enabled && p.Effort != "" {
		return false
	}
	switch p.Effort {
	case "", "minimal", "low", "medium", "high", "xhigh", "max":
		return true
	}
	return false
}

func (p *modelThinking) requestFields(model selectedModelConfig) map[string]any {
	if !p.valid() || !model.SupportsReasoning {
		return nil
	}
	effort := p.Effort
	if !*p.Enabled {
		effort = "none"
	}
	if model.Protocol == "openai_chat_completions" && strings.HasPrefix(strings.ToLower(model.Model), "deepseek-") {
		if !*p.Enabled {
			return map[string]any{"thinking": map[string]string{"type": "disabled"}}
		}
		fields := map[string]any{"thinking": map[string]string{"type": "enabled"}}
		if effort != "" {
			fields["reasoning_effort"] = effort
		}
		return fields
	}
	if effort == "" {
		effort = "medium"
	}
	if model.Protocol == "openai_responses" {
		return map[string]any{"reasoning": map[string]string{"effort": effort}}
	}
	if effort == "max" {
		effort = "xhigh"
	}
	return map[string]any{"reasoning_effort": effort}
}

func (s desktopSettings) goalBudget() int64 {
	if s.GoalBudget == nil {
		return defaultGoalBudget
	}
	return *s.GoalBudget
}

func (s desktopSettings) display() map[string]bool {
	return map[string]bool{
		"tool_summaries":     s.ToolSummaries == nil || *s.ToolSummaries,
		"tool_use_summaries": s.ToolUseSummaries == nil || *s.ToolUseSummaries,
	}
}

func (l *acpLive) loadDesktopSettings() error {
	err := readDesktopSessionRecord(l.store.Root, desktopSettingsFile, &l.settings)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (l *acpLive) saveDesktopSettings(method string, raw json.RawMessage) error {
	var fields map[string]json.RawMessage
	if l.store == nil || len(raw) > 4096 || json.Unmarshal(raw, &fields) != nil || len(fields) == 0 {
		return errors.New("invalid_desktop_settings")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	next := l.settings
	switch method {
	case "_monkeyai/config/goals/set":
		budget, ok := fields["default_token_budget"]
		var value *int64
		if !ok || json.Unmarshal(budget, &value) != nil ||
			value != nil && (*value < 1 || *value > maxGoalBudget) {
			return errors.New("invalid_goal_budget")
		}
		next.GoalBudget = value
	case "_monkeyai/config/display/set":
		for _, field := range []struct {
			name string
			dest **bool
		}{{"tool_summaries", &next.ToolSummaries}, {"tool_use_summaries", &next.ToolUseSummaries}} {
			if v, ok := fields[field.name]; ok {
				var value bool
				if string(v) == "null" || json.Unmarshal(v, &value) != nil {
					return errors.New("invalid_desktop_settings")
				}
				*field.dest = &value
			}
		}
	case "_monkeyai/preferences/model-thinking/set":
		var request struct {
			Model    string
			Thinking *modelThinking
		}
		if json.Unmarshal(raw, &request) != nil || !validUUID(request.Model) || !request.Thinking.valid() {
			return errors.New("invalid_model_thinking")
		}
		next.ModelThinking = maps.Clone(next.ModelThinking)
		if next.ModelThinking == nil {
			next.ModelThinking = make(map[string]modelThinking)
		}
		next.ModelThinking[request.Model] = *request.Thinking
	default:
		return errors.New("invalid_desktop_settings")
	}
	next.Revision++
	data, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err := writeSourceState(l.store.Root, desktopSettingsFile, data); err != nil {
		return err
	}
	l.settings = next
	return nil
}

func (h *relaySessionHost) desktopBilling(ctx context.Context, method string, raw json.RawMessage) (any, error) {
	auth, err := h.live.authorization(ctx)
	if err != nil {
		return nil, err
	}
	var data json.RawMessage
	path := "/api/v1/billing/account"
	if method == "_monkeyai/billing/entries" {
		var filter struct {
			Page                                 int
			PageSize                             int `json:"page_size"`
			Content, Category, Mode, From, Until string
			EntryType                            string `json:"entry_type"`
		}
		if json.Unmarshal(raw, &filter) != nil || filter.Page < 0 || filter.Page > 100000 ||
			filter.PageSize < 0 || filter.PageSize > 100 {
			return nil, errors.New("invalid_billing_filters")
		}
		query := url.Values{}
		for key, value := range map[string]string{
			"content": filter.Content, "category": filter.Category, "mode": filter.Mode, "entry_type": filter.EntryType,
			"from": filter.From, "until": filter.Until,
		} {
			if value != "" {
				query.Set(key, value)
			}
		}
		if filter.Page > 0 {
			query.Set("page", strconv.Itoa(filter.Page))
		}
		if filter.PageSize > 0 {
			query.Set("page_size", strconv.Itoa(filter.PageSize))
		}
		path = "/api/v1/billing/entries?" + query.Encode()
	}
	if err := h.live.client.json(ctx, http.MethodGet, path, auth.Access, "", nil, &data); err != nil {
		return nil, err
	}
	if method == "_monkeyai/billing/account" {
		overview := map[string]any{"account": data}
		var config struct {
			Settings struct {
				Billing json.RawMessage
			}
		}
		if err := h.live.client.json(ctx, http.MethodGet, "/api/v1/settings", auth.Access, "", nil, &config); err == nil && config.Settings.Billing != nil {
			overview["policy"] = config.Settings.Billing
		}
		overview["fetchedAt"] = time.Now().UTC()
		return map[string]any{"data": overview}, nil
	}
	return map[string]any{"data": data}, nil
}

// desktopAvatar uploads or clears the signed-in user's custom avatar. The web
// client sends an already resized image as base64 so the ACP frame stays small.
func (h *relaySessionHost) desktopAvatar(ctx context.Context, raw json.RawMessage) (any, error) {
	var input struct {
		Image string `json:"image"`
		MIME  string `json:"mime"`
		Clear bool   `json:"clear"`
	}
	if json.Unmarshal(raw, &input) != nil {
		return nil, errors.New("invalid_avatar")
	}
	auth, err := h.live.authorization(ctx)
	if err != nil {
		return nil, err
	}
	method, mime, body := http.MethodDelete, "", io.Reader(nil)
	if !input.Clear {
		if input.MIME != "image/webp" && input.MIME != "image/png" && input.MIME != "image/jpeg" {
			return nil, errors.New("invalid_avatar")
		}
		data, err := base64.StdEncoding.DecodeString(input.Image)
		if err != nil || len(data) == 0 {
			return nil, errors.New("invalid_avatar")
		}
		if len(data) > 5<<20 {
			return nil, errors.New("avatar_too_large")
		}
		method, mime, body = http.MethodPut, input.MIME, bytes.NewReader(data)
	}
	resp, err := h.live.client.request(ctx, method, "/api/v1/me/avatar", auth.Access, "", "", mime, body)
	if err != nil {
		var status *backendHTTPError
		if errors.As(err, &status) {
			switch status.Status {
			case http.StatusBadRequest:
				return nil, errors.New("invalid_avatar")
			case http.StatusRequestEntityTooLarge:
				return nil, errors.New("avatar_too_large")
			case http.StatusNotFound, http.StatusMethodNotAllowed:
				return nil, errors.New("avatar_unsupported")
			}
		}
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		User account `json:"user"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return nil, err
	}
	return map[string]any{"user": out.User}, nil
}

func (l *acpLive) desktopDrafts(method string, raw json.RawMessage) (any, error) {
	const file = "desktop-drafts.json"
	if l.store == nil {
		return nil, errors.New("drafts_unavailable")
	}
	type draft struct {
		Text  string            `json:"text"`
		Files []json.RawMessage `json:"files"`
	}
	drafts := struct {
		Sessions    map[string]draft `json:"sessions"`
		NewSessions map[string]draft `json:"newSessions"`
	}{make(map[string]draft), make(map[string]draft)}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := readDesktopSessionRecord(l.store.Root, file, &drafts); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if method == "_monkeyai/drafts" {
		return drafts, nil
	}
	var request struct {
		Target struct{ SessionID, NewSession string }
		Draft  json.RawMessage
	}
	if len(raw) > 256<<10 || json.Unmarshal(raw, &request) != nil || len(request.Draft) == 0 {
		return nil, errors.New("invalid_draft")
	}
	target, id := drafts.Sessions, request.Target.SessionID
	if id != "" {
		if !validUUID(id) || request.Target.NewSession != "" {
			return nil, errors.New("invalid_draft")
		}
	} else {
		id, target = request.Target.NewSession, drafts.NewSessions
		if id == "" || len(id) > 8192 {
			return nil, errors.New("invalid_draft")
		}
	}
	if string(request.Draft) == "null" {
		delete(target, id)
	} else {
		var value draft
		if json.Unmarshal(request.Draft, &value) != nil || value.Files == nil {
			return nil, errors.New("invalid_draft")
		}
		target[id] = value
	}
	return map[string]bool{"ok": true}, saveDesktopSessionRecord(l.store.Root, file, drafts)
}
