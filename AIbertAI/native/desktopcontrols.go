package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type desktopSessionPresentation struct {
	Title      string     `json:"title,omitempty"`
	Pinned     bool       `json:"pinned,omitempty"`
	ArchivedAt *time.Time `json:"archivedAt,omitempty"`
	ReadAt     *time.Time `json:"readAt,omitempty"`
}

func promptTitle(text string) string {
	text = strings.Join(strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsControl(r) || unicode.IsSpace(r)
	}), " ")
	runes := []rune(text)
	return string(runes[:min(len(runes), 40)])
}

func (row acpSession) sessionTitle() (string, string) {
	if row.Title != "" {
		return row.Title, "user"
	}
	if len(row.Turns) != 0 {
		first := row.Turns[0]
		if title := promptTitle(first.Prompt); title != "" {
			return title, "prompt"
		}
		for _, node := range first.CanvasNodes {
			if title := promptTitle(node.Title); title != "" {
				return title, "prompt"
			}
		}
		for _, file := range first.Attachments {
			if title := promptTitle(file.Name); title != "" {
				return title, "prompt"
			}
		}
	}
	if row.Goal != nil {
		if title := promptTitle(row.Goal.Objective); title != "" {
			return title, "prompt"
		}
	}
	return "AIbertAI 会话", ""
}

func (h *relaySessionHost) updateDesktopSessionPresentation(method string, raw json.RawMessage) (any, error) {
	var p struct {
		SessionID, Title       string
		Pinned, Archived, Read *bool
	}
	if json.Unmarshal(raw, &p) != nil || !validUUID(p.SessionID) {
		return nil, errors.New("invalid_session_parameters")
	}
	l := h.live
	l.mu.Lock()
	defer l.mu.Unlock()
	row, ok := l.sessions[p.SessionID]
	if !ok {
		return nil, errors.New("session_not_found")
	}
	previous := row.desktopSessionPresentation
	if method == "_monkeyai/session/rename" {
		title := strings.TrimSpace(p.Title)
		if title == "" || utf8.RuneCountInString(title) > 200 || strings.ContainsAny(title, "\r\n\x00") {
			return nil, errors.New("invalid_session_title")
		}
		row.Title = title
	} else {
		if p.Archived != nil {
			if !*p.Archived {
				row.ArchivedAt = nil
			} else if row.ArchivedAt == nil {
				now := time.Now().UTC()
				row.ArchivedAt = &now
			}
		}
		if p.Pinned != nil {
			row.Pinned = *p.Pinned
		}
		if row.ArchivedAt != nil {
			row.Pinned = false
		}
		if p.Read != nil && *p.Read && (row.ReadAt == nil || row.Updated.After(*row.ReadAt)) {
			row.ReadAt = &row.Updated
		}
	}
	if row.desktopSessionPresentation != previous {
		if err := l.save(row); err != nil {
			return nil, errors.New("session_write_failed")
		}
	}
	title, source := row.sessionTitle()
	return map[string]any{"title": title, "titleSource": source, "pinned": row.Pinned, "archivedAt": row.ArchivedAt, "readAt": row.ReadAt}, nil
}

func validACPMode(mode string) bool {
	return mode == "auto" || mode == "bypassPermissions"
}

func (row acpSession) permissionMode() string {
	if row.Mode == "bypassPermissions" {
		return row.Mode
	}
	return "auto"
}

func (h *relaySessionHost) setACPPermissionMode(r *http.Request, raw json.RawMessage, waitingID string) (map[string]any, error) {
	if h.local == nil || h.live == nil || h.acpScope(r, "session:control") != nil {
		return nil, errRelayAccess
	}
	var p struct{ SessionID, ModeID string }
	if json.Unmarshal(raw, &p) != nil || !validUUID(p.SessionID) || !validACPMode(p.ModeID) ||
		waitingID != "" && waitingID != p.SessionID {
		return nil, errors.New("invalid_session_mode")
	}
	l := h.live
	l.mu.Lock()
	defer l.mu.Unlock()
	row, ok := l.sessions[p.SessionID]
	if !ok {
		return nil, errors.New("session_not_found")
	}
	row.Mode = p.ModeID
	if err := l.save(row); err != nil {
		return nil, errors.New("session_write_failed")
	}
	return map[string]any{"modeId": row.Mode, "resolvedPermission": waitingID != "" && row.Mode == "bypassPermissions"}, nil
}

type desktopMediaModels struct{ Image, Video string }

func (h *relaySessionHost) desktopMediaGateway(ctx context.Context, raw json.RawMessage) (any, error) {
	var input struct{ SessionID, Kind, ModelID string }
	if json.Unmarshal(raw, &input) != nil || input.Kind != "image" && input.Kind != "video" ||
		input.ModelID != "" && !validUUID(input.ModelID) {
		return nil, errors.New("invalid_media_model")
	}
	row, err := h.live.load(input.SessionID)
	if err != nil {
		return nil, err
	}
	if h.live.modelLocked || h.live.key == "" {
		return nil, errors.New("media_model_unavailable")
	}
	selected := input.ModelID
	if selected == "" {
		selected = row.MediaImageModelID
		if input.Kind == "video" {
			selected = row.MediaVideoModelID
		}
	}
	auth, err := h.live.authorization(ctx)
	if err != nil {
		return nil, err
	}
	items, err := h.live.client.desktopCatalog(ctx, auth, "models")
	if err != nil {
		return nil, err
	}
	for _, raw := range items {
		var model struct {
			ID, Kind, Model string
			ImageConfig     json.RawMessage `json:"image_config"`
			VideoConfig     json.RawMessage `json:"video_config"`
		}
		if json.Unmarshal(raw, &model) != nil || model.Kind != input.Kind ||
			!validUUID(model.ID) || model.Model == "" || selected != "" && model.ID != selected {
			continue
		}
		// Current catalogs supply the gateway identifier. Older catalogs supplied
		// the upstream name and need the catalog UUID appended once.
		modelRef := model.Model
		if suffix := "@" + model.ID; !strings.HasSuffix(modelRef, suffix) {
			modelRef += suffix
		}
		return map[string]any{"id": model.ID, "model": modelRef,
			"baseUrl": h.live.client.server + "/v1", "apiKey": h.live.key,
			"imageConfig": model.ImageConfig, "videoConfig": model.VideoConfig,
			"headers": map[string]string{"X-MAI-Session-ID": row.ID}}, nil
	}
	return nil, errors.New("media_model_unavailable")
}

func (h *relaySessionHost) validateDesktopMedia(ctx context.Context, selection desktopMediaModels) error {
	if selection.Image == "" && selection.Video == "" {
		return nil
	}
	if h.live.modelLocked || selection.Image != "" && !validUUID(selection.Image) || selection.Video != "" && !validUUID(selection.Video) {
		return errors.New("media_model_unavailable")
	}
	auth, err := h.live.authorization(ctx)
	if err != nil {
		return err
	}
	items, err := h.live.client.desktopCatalog(ctx, auth, "models")
	if err != nil {
		return err
	}
	for _, raw := range items {
		var model struct{ ID, Kind string }
		if json.Unmarshal(raw, &model) != nil {
			continue
		}
		if model.Kind == "image" && selection.Image == model.ID {
			selection.Image = ""
		}
		if model.Kind == "video" && selection.Video == model.ID {
			selection.Video = ""
		}
	}
	if selection.Image != "" || selection.Video != "" {
		return errors.New("media_model_unavailable")
	}
	return nil
}

func (h *relaySessionHost) setDesktopSessionMedia(ctx context.Context, raw json.RawMessage) (any, error) {
	var p struct {
		SessionID    string
		Image, Video json.RawMessage
	}
	if json.Unmarshal(raw, &p) != nil || !validUUID(p.SessionID) || len(p.Image)+len(p.Video) == 0 {
		return nil, errors.New("invalid_session_parameters")
	}
	var requested desktopMediaModels
	for _, field := range []struct {
		raw json.RawMessage
		id  *string
	}{{p.Image, &requested.Image}, {p.Video, &requested.Video}} {
		if len(field.raw) != 0 && json.Unmarshal(field.raw, field.id) != nil {
			return nil, errors.New("invalid_media_model")
		}
	}
	if err := h.validateDesktopMedia(ctx, requested); err != nil {
		return nil, err
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
	if len(p.Image) != 0 {
		row.MediaImageModelID = requested.Image
	}
	if len(p.Video) != 0 {
		row.MediaVideoModelID = requested.Video
	}
	if err := l.save(row); err != nil {
		return nil, errors.New("session_write_failed")
	}
	return map[string]string{"mediaImageModelId": row.MediaImageModelID, "mediaVideoModelId": row.MediaVideoModelID}, nil
}
