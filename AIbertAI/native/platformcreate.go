package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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

// createPlatformSkill saves a personal skill and refreshes the DSH copy, whose
// watcher makes it loadable in running conversations without an expert.
func (h *relaySessionHost) createPlatformSkill(ctx context.Context, name, description, content string) (json.RawMessage, error) {
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
	if h.live.dsh != nil && h.live.dsh.SyncPlatformSkills != nil {
		_ = h.live.dsh.SyncPlatformSkills(ctx)
	}
	return result, nil
}
