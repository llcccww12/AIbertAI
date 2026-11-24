package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type dshExpertSkill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Directory   string `json:"directory"`
}

type dshExpertManifestKey struct{}
type dshExpertSkillsKey struct{}
type dshDisabledPluginsKey struct{}

// These are entry IDs in the pinned SDK's base and sdk-app Cordis profiles.
// The gateway patch only disables entries; platform-specific defaults still apply.
var dshOptionalPlugins = []string{
	"agent-instructions", "time-context", "tool-bash", "tool-pwsh", "tool-fs", "tool-fs-search",
	"tool-jobs", "skill-filesystem", "tool-skill", "tool-ask-user", "tool-todo",
	"tool-web", "present", "tool-office", "tool-agent-team", "tool-subagent-control", "tool-subagent", "tool-subagent-fork",
	"tool-plugin-manager", "tool-workflow",
}

func dshDisabledPlugins(enabled []string) ([]string, error) {
	if enabled == nil {
		return nil, nil
	}
	for _, id := range enabled {
		// Older manifests may still select scheduling. The host disables it for every session.
		if id != "tool-schedule" && !slices.Contains(dshOptionalPlugins, id) {
			return nil, errors.New("dsh_expert_plugin_unsupported")
		}
	}
	var disabled []string
	for _, id := range dshOptionalPlugins {
		if !slices.Contains(enabled, id) {
			disabled = append(disabled, id)
		}
	}
	return disabled, nil
}

// The backend manifest controls this turn's role and skills. The connector
// bridge separately enforces the manifest's connector filters before dispatch.
func (h *relaySessionHost) bindDSHExpert(ctx context.Context, session acpSession) (context.Context, func(), error) {
	noop := func() {}
	if session.ExpertID == "" {
		return ctx, noop, nil
	}
	if session.ExpertID == designExpertID {
		return ctx, noop, errDSHCanvasUnavailable
	}
	if h.live == nil || h.live.client == nil {
		return ctx, noop, errors.New("expert_unavailable")
	}
	expert, err := h.resolveDesktopExpert(ctx, session.ExpertID)
	if err != nil {
		return ctx, noop, err
	}
	instructions, err := expert.roleInstructions()
	if err != nil {
		return ctx, noop, err
	}
	disabled, err := dshDisabledPlugins(expert.DSHPlugins)
	if err != nil {
		return ctx, noop, err
	}
	bound := context.WithValue(ctx, dshExpertManifestKey{}, expert)
	bound = context.WithValue(bound, dshDisabledPluginsKey{}, disabled)
	bound = context.WithValue(bound, dshExpertInstructionsKey{}, instructions)
	if len(expert.Skills) == 0 {
		return bound, noop, nil
	}

	auth, err := h.live.authorization(ctx)
	if err != nil {
		return ctx, noop, err
	}
	root, err := os.MkdirTemp("", "aibertai-expert-skills-")
	if err != nil {
		return ctx, noop, err
	}
	cleanup := func() { _ = os.RemoveAll(root) }
	var skills []dshExpertSkill
	total := len(instructions)
	for _, item := range expert.Skills {
		if !validUUID(item.ID) {
			cleanup()
			return ctx, noop, errors.New("invalid_expert_skill")
		}
		dir := filepath.Join(root, item.ID)
		if err := os.Mkdir(dir, 0700); err != nil {
			cleanup()
			return ctx, noop, err
		}
		content, err := h.live.client.expertSkill(ctx, auth, expert.ID, item, dir)
		if err != nil {
			cleanup()
			return ctx, noop, err
		}
		total += len(content) + len(item.Name) + len(item.Description)
		if total > 256<<10 {
			cleanup()
			return ctx, noop, errors.New("expert_instructions_too_large")
		}
		name := "aibert-" + strings.ToLower(item.ID)
		description := strings.TrimSpace(strings.TrimSpace(item.Name) + " " + strings.TrimSpace(item.Description))
		if description == "" {
			description = name
		}
		skills = append(skills, dshExpertSkill{Name: name, Description: description, Content: content, Directory: dir})
	}
	return context.WithValue(bound, dshExpertSkillsKey{}, skills), cleanup, nil
}
