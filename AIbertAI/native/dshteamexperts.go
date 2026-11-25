package main

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"time"
)

// Platform experts are offered to DSH subagents as agent presets, so a lead
// agent (with or without Team mode) can delegate work to a matching expert.
// A preset carries the expert's role prompt; the child shares the
// conversation's skills and tools.
type dshTeamExpert struct {
	PresetID, Name, Description, Prompt string
}

const dshTeamExpertLimit = 20

func dshTeamPresetID(expertID string) string {
	return "expert-" + strings.ReplaceAll(expertID, "-", "")[:8]
}

// The persona prefix is a template; a literal "{{" in a role prompt must not
// be read as a prompt variable.
func dshPersonaText(prompt string) string {
	return strings.ReplaceAll(prompt, "{{", "{ {")
}

func (c *client) dshTeamExperts(ctx context.Context, auth authorization) ([]dshTeamExpert, error) {
	rows, err := c.desktopCatalog(ctx, auth, "experts")
	if err != nil {
		return nil, err
	}
	type summary struct {
		ID, Name, Description string
		Available             *bool
	}
	var candidates []summary
	for _, raw := range rows {
		var s summary
		if json.Unmarshal(raw, &s) != nil || !validUUID(s.ID) || s.Available != nil && !*s.Available {
			continue
		}
		candidates = append(candidates, s)
		if len(candidates) == dshTeamExpertLimit {
			break
		}
	}
	out := make([]dshTeamExpert, len(candidates))
	var wg sync.WaitGroup
	for i, s := range candidates {
		wg.Go(func() {
			manifest, err := c.desktopExpertManifest(ctx, auth, s.ID)
			// An expert that disables subagents or skills keeps its own shape;
			// only experts with the default capability set join as presets.
			if err != nil || strings.TrimSpace(manifest.Prompt) == "" || manifest.DSHPlugins != nil {
				return
			}
			out[i] = dshTeamExpert{PresetID: dshTeamPresetID(s.ID), Name: s.Name, Description: s.Description, Prompt: manifest.Prompt}
		})
	}
	wg.Wait()
	seen := map[string]bool{}
	return slices.DeleteFunc(out, func(e dshTeamExpert) bool {
		drop := e.PresetID == "" || seen[e.PresetID]
		seen[e.PresetID] = true
		return drop
	}), nil
}

// dshTeamExpertCache keeps the expert list for a few minutes so starting a
// conversation does not wait on the catalog each time.
type dshTeamExpertCache struct {
	mu      sync.Mutex
	fetched time.Time
	value   []dshTeamExpert
	load    func(context.Context) ([]dshTeamExpert, error)
}

func (c *dshTeamExpertCache) get(ctx context.Context) []dshTeamExpert {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.fetched.IsZero() && time.Since(c.fetched) < 3*time.Minute {
		return c.value
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if value, err := c.load(ctx); err == nil {
		c.value, c.fetched = value, time.Now()
	}
	return c.value
}

func dshTeamExpertInstructions(experts []dshTeamExpert) string {
	if len(experts) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Platform experts are available as subagent presets. When you delegate work that matches one of them, pass its id as agent_preset; otherwise omit agent_preset.")
	for _, e := range experts {
		b.WriteString("\n- " + e.PresetID + ": " + e.Name)
		if d := strings.TrimSpace(e.Description); d != "" {
			b.WriteString(" — " + d)
		}
	}
	return b.String()
}

// dshTeamPresetRows declares the registry, an empty standard preset for the
// lead, and one preset per expert.
func dshTeamPresetRows(experts []dshTeamExpert) []any {
	rows := []any{
		map[string]any{"id": "agent-preset-registry", "name": "@deepseek-ai/dsh-agent-preset-registry", "config": map[string]any{"default": "standard"}},
		map[string]any{"id": "preset-standard", "name": "@deepseek-ai/dsh-agent-preset", "config": map[string]any{"id": "standard", "plugins": []any{}}},
	}
	for _, e := range experts {
		rows = append(rows, map[string]any{"id": "preset-" + e.PresetID, "name": "@deepseek-ai/dsh-agent-preset", "config": map[string]any{
			"id": e.PresetID, "name": e.Name, "description": e.Description,
			"plugins": []any{map[string]any{"id": "persona", "name": "@deepseek-ai/dsh-persona", "config": map[string]any{"prefix": dshPersonaText(e.Prompt)}}},
		}})
	}
	return rows
}
