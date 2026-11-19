package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
)

type expertSelection struct {
	ID   string
	Name string
}

type desktopExpert struct {
	expertSelection
	Available bool
	ExpertID  string `json:"expert_id"`
	Prompt    string
	Rules     []struct{ Name, Content string }
	Skills    []struct {
		ID, Name string
		Digest   string `json:"package_sha256"`
	}
	Connectors []expertConnector
}

type expertConnector struct {
	ID       string
	Required bool
	Allow    []string `json:"tool_allowlist"`
	Deny     []string `json:"tool_denylist"`
}

func (h *relaySessionHost) resolveDesktopExpert(ctx context.Context, id string) (desktopExpert, error) {
	if id == "" || id == designExpertID {
		return desktopExpert{}, nil
	}
	if !validUUID(id) || h.live == nil {
		return desktopExpert{}, errors.New("expert_unavailable")
	}
	var expert desktopExpert
	auth, err := h.live.authorization(ctx)
	if err != nil {
		return expert, err
	}
	if err = h.live.client.json(ctx, http.MethodGet, "/api/v1/experts/"+id+"/manifest", auth.Access, "", nil, &expert); err != nil {
		return expert, err
	}
	if expert.ExpertID != id || !expert.Available {
		return expert, errors.New("expert_unavailable")
	}
	expert.ID = id
	return expert, nil
}

func (h *relaySessionHost) setDesktopSessionExpert(ctx context.Context, raw json.RawMessage) (any, error) {
	var p struct {
		SessionID, ExpertID string
		Meta                struct{ ExpectedRevision *int } `json:"_meta"`
	}
	if json.Unmarshal(raw, &p) != nil || !validUUID(p.SessionID) {
		return nil, errors.New("invalid_session_parameters")
	}
	expert, err := h.resolveDesktopExpert(ctx, p.ExpertID)
	if err != nil {
		return nil, err
	}
	if p.ExpertID == designExpertID {
		if h.local == nil || h.designBridge == nil {
			return nil, errors.New("expert_unavailable")
		}
		root, _, err := h.workspaceRoot(p.SessionID)
		if err != nil {
			return nil, err
		}
		defer root.Close()
		if workspaceDesign(root, p.SessionID) == nil {
			return nil, errors.New("invalid_design_workspace")
		}
		expert.expertSelection = expertSelection{ID: designExpertID, Name: "设计专家"}
	}
	l := h.live
	l.mu.Lock()
	defer l.mu.Unlock()
	row, ok := l.sessions[p.SessionID]
	if !ok {
		return nil, errors.New("session_not_found")
	}
	if row.State == "running" || p.Meta.ExpectedRevision != nil && *p.Meta.ExpectedRevision != row.ResourceRevision {
		return nil, errors.New("configuration_conflict")
	}
	if row.ExpertID != expert.ID {
		row.ExpertID, row.ExpertName = expert.ID, expert.Name
		row.ResourceRevision++
		if err := l.save(row); err != nil {
			return nil, err
		}
	}
	return map[string]any{"expertId": row.ExpertID, "_meta": map[string]int{"configurationRevision": row.ResourceRevision}}, nil
}

func (c *client) expertInstructions(ctx context.Context, auth authorization, expert desktopExpert) (string, error) {
	var text strings.Builder
	text.WriteString(expert.Prompt)
	for _, rule := range expert.Rules {
		text.WriteString("\n\n")
		text.WriteString(rule.Content)
	}
	if text.Len() > 256<<10 {
		return "", errors.New("expert_instructions_too_large")
	}
	for _, skill := range expert.Skills {
		if !validUUID(skill.ID) {
			return "", errors.New("invalid_expert_skill")
		}
		digest := strings.TrimPrefix(skill.Digest, "sha256:")
		if _, err := hex.DecodeString(digest); err != nil || len(digest) != 64 {
			return "", errors.New("invalid_expert_skill")
		}
		query := url.Values{"sha256": {skill.Digest}}
		route := "/api/v1/experts/" + expert.ID + "/skills/" + skill.ID + "/package?" + query.Encode()
		content, err := c.expertSkill(ctx, auth, route, digest)
		if err != nil {
			return "", err
		}
		text.WriteString("\n\n")
		text.WriteString(content)
		if text.Len() > 256<<10 {
			return "", errors.New("expert_instructions_too_large")
		}
	}
	return text.String(), nil
}

func (c *client) expertSkill(ctx context.Context, auth authorization, route, digest string) (string, error) {
	const maxPackage = 16 << 20
	const maxInstruction = 128 << 10
	resp, err := c.request(ctx, http.MethodGet, route, auth.Access, "", "", "", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxPackage+1))
	sum := sha256.Sum256(data)
	if err != nil || len(data) > maxPackage || hex.EncodeToString(sum[:]) != digest {
		return "", errors.New("invalid_expert_skill")
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", errors.New("invalid_expert_skill")
	}
	var instructions *zip.File
	for _, file := range archive.File {
		if path.Base(file.Name) == "SKILL.md" && file.Mode().IsRegular() {
			if instructions != nil || file.UncompressedSize64 > maxInstruction {
				return "", errors.New("invalid_expert_skill")
			}
			instructions = file
		}
	}
	if instructions == nil {
		return "", errors.New("invalid_expert_skill")
	}
	reader, err := instructions.Open()
	if err != nil {
		return "", err
	}
	defer reader.Close()
	body, err := io.ReadAll(io.LimitReader(reader, maxInstruction+1))
	if err != nil || len(body) > maxInstruction {
		return "", errors.New("invalid_expert_skill")
	}
	return string(body), nil
}

func (link expertConnector) allows(tool string) bool {
	return !slices.Contains(link.Deny, tool) && (len(link.Allow) == 0 || slices.Contains(link.Allow, tool))
}
