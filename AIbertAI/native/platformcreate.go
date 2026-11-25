package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"
)

// personalSkillPackage builds the one-file package the backend parses: a
// SKILL.md whose frontmatter carries the platform name and description.
func personalSkillPackage(name, description, content string) ([]byte, error) {
	if !skillCommandName.MatchString(name) || strings.TrimSpace(description) == "" || len(description) > 1024 ||
		strings.TrimSpace(content) == "" || len(content) > 128<<10 || !utf8.ValidString(description+content) {
		return nil, errors.New("skill_input_invalid")
	}
	quoted := func(s string) string { data, _ := json.Marshal(s); return string(data) }
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	w, err := z.Create(name + "/SKILL.md")
	if err == nil {
		_, err = io.WriteString(w, "---\nname: "+quoted(name)+"\ndescription: "+quoted(strings.TrimSpace(description))+"\n---\n"+content)
	}
	if err = errors.Join(err, z.Close()); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// createPersonalSkill saves a skill owned by the current account; it then
// appears in the account's skill catalog like one uploaded as a ZIP.
func (c *client) createPersonalSkill(ctx context.Context, auth authorization, name, description, content string) (json.RawMessage, error) {
	pkg, err := personalSkillPackage(name, description, content)
	if err != nil {
		return nil, err
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("package", name+".zip")
	if err == nil {
		_, err = file.Write(pkg)
	}
	if err = errors.Join(err, form.Close()); err != nil {
		return nil, err
	}
	resp, err := c.request(ctx, http.MethodPost, "/api/v1/skills", auth.Access, "", "", form.FormDataContentType(), &body, http.StatusBadRequest, http.StatusConflict, http.StatusUnprocessableEntity)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var payload json.RawMessage
	if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload) != nil {
		return nil, errors.New("skill_create_failed")
	}
	if resp.StatusCode >= 300 {
		var problem struct{ Error struct{ Code string } }
		if json.Unmarshal(payload, &problem) == nil && knowledgeErrorCode.MatchString(problem.Error.Code) {
			return nil, errors.New(problem.Error.Code)
		}
		return nil, errors.New("skill_create_failed")
	}
	return payload, nil
}

// remoteConnectorInput is the personal HTTP connector the chat tool may save.
// Credentials are never accepted here: the user adds them in Settings.
func remoteConnectorInput(name, description, endpoint string) (map[string]any, error) {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if strings.TrimSpace(name) == "" || len(name) > 200 || len(description) > 2000 || err != nil ||
		(u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.Fragment != "" {
		return nil, errors.New("connector_input_invalid")
	}
	return map[string]any{"name": strings.TrimSpace(name), "description": strings.TrimSpace(description), "url": u.String(),
		"authorization_mode": "none", "authorization_method": nil}, nil
}

// createPlatformSkill saves a personal skill and refreshes the DSH copy. A
// skill created from a conversation is enabled there, so the folder watcher
// makes it loadable in the same turn.
func (h *relaySessionHost) createPlatformSkill(ctx context.Context, sessionID, name, description, content string) (json.RawMessage, error) {
	if h.live == nil || h.live.client == nil {
		return nil, errors.New("skill_create_unavailable")
	}
	auth, err := h.live.authorization(ctx)
	if err != nil {
		return nil, err
	}
	result, err := h.live.client.createPersonalSkill(ctx, auth, name, description, content)
	if err != nil {
		return nil, err
	}
	var created struct{ ID string }
	if sessionID != "" && json.Unmarshal(result, &created) == nil && validUUID(created.ID) {
		_ = h.live.enableSessionSkill(sessionID, created.ID)
	}
	if dsh := h.live.dsh; dsh != nil && dsh.SyncPlatformSkills != nil {
		_ = dsh.SyncPlatformSkills(ctx)
		if session, err := h.live.load(sessionID); sessionID != "" && err == nil && dsh.SessionSkills != nil {
			_ = dsh.SessionSkills(session)
		}
	}
	return result, nil
}

func (l *acpLive) enableSessionSkill(sessionID, skillID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	session, ok := l.sessions[sessionID]
	if !ok {
		return errors.New("session_not_found")
	}
	prefs := l.connectorPrefs
	if session.ConnectorPreferences != nil {
		prefs = *session.ConnectorPreferences
	}
	prefs.Enabled = maps.Clone(prefs.Enabled)
	prefs = prefs.normalized()
	prefs.Enabled[dshSkillPreferencePrefix+skillID] = true
	session.ConnectorPreferences = &prefs
	session.ResourceRevision++
	return l.save(session)
}

type expertConnectorLink struct {
	ID       string `json:"id"`
	Required bool   `json:"required"`
}

// personalExpertInput is the backend body for POST /api/v1/experts. DSH
// plugins are left unset, so the expert keeps every default capability.
func personalExpertInput(name, description, prompt string, skills []string, connectors []expertConnectorLink) (map[string]any, error) {
	name, description = strings.TrimSpace(name), strings.TrimSpace(description)
	if name == "" || utf8.RuneCountInString(name) > 200 || utf8.RuneCountInString(description) > 2000 ||
		strings.TrimSpace(prompt) == "" || len(prompt) > 32<<10 || !utf8.ValidString(name+description+prompt) ||
		len(skills) > 50 || len(connectors) > 50 {
		return nil, errors.New("expert_input_invalid")
	}
	seen := map[string]bool{}
	skillIDs := make([]string, 0, len(skills))
	for _, id := range skills {
		if !validUUID(id) || seen[id] {
			return nil, errors.New("expert_input_invalid")
		}
		seen[id] = true
		skillIDs = append(skillIDs, id)
	}
	links := make([]map[string]any, 0, len(connectors))
	for _, link := range connectors {
		if !validUUID(link.ID) || seen[link.ID] {
			return nil, errors.New("expert_input_invalid")
		}
		seen[link.ID] = true
		links = append(links, map[string]any{"connector_id": link.ID, "required": link.Required})
	}
	return map[string]any{"name": name, "description": description, "prompt": prompt, "skill_ids": skillIDs, "connectors": links}, nil
}

// platformResourceCatalog lists what a new expert may reference: the
// account's skills and remote connectors, with names for the model.
func (h *relaySessionHost) platformResourceCatalog(ctx context.Context) (map[string]any, error) {
	auth, err := h.live.authorization(ctx)
	if err != nil {
		return nil, err
	}
	skills, err := h.live.client.desktopSkillCatalog(ctx, auth)
	if err != nil {
		return nil, err
	}
	connectors, err := h.live.client.employeeConnectors(ctx, auth)
	if err != nil {
		return nil, err
	}
	type entry struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	out := map[string]any{"skills": []entry{}, "connectors": []entry{}}
	skillRows := make([]entry, 0, len(skills))
	for _, s := range skills {
		id, _ := s["id"].(string)
		name, _ := s["name"].(string)
		description, _ := s["description"].(string)
		skillRows = append(skillRows, entry{id, name, description})
	}
	connectorRows := make([]entry, 0, len(connectors))
	for _, raw := range connectors {
		var c entry
		if json.Unmarshal(raw, &c) == nil && validUUID(c.ID) {
			connectorRows = append(connectorRows, c)
		}
	}
	out["skills"], out["connectors"] = skillRows, connectorRows
	return out, nil
}
