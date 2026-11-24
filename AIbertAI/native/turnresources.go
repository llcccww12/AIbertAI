package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// turnResource is one resource in effect for a turn. Reported as the turn's
// resource snapshot, it drives the "enabled resources" view of AI 行为.
type turnResource struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

func (t *acpTurn) useResource(kind, id string) {
	if t == nil || !validUUID(id) || slices.Contains(t.Resources, turnResource{id, kind}) {
		return
	}
	t.Resources = append(t.Resources, turnResource{id, kind})
}

// recordDSHResources notes the model, expert, expert skills and applied rules.
// Connectors are added where DSH receives their tools.
func (t *acpTurn) recordDSHResources(ctx context.Context, session acpSession) {
	model := t.Model
	if model == "" {
		model = modelID
	}
	t.useResource("model", model)
	if session.ExpertID != designExpertID {
		t.useResource("expert", session.ExpertID)
	}
	if expert, ok := ctx.Value(dshExpertManifestKey{}).(desktopExpert); ok && expert.ID == session.ExpertID {
		for _, skill := range expert.Skills {
			t.useResource("skill", skill.ID)
		}
	}
	rules, _ := ctx.Value(dshAppliedRulesKey{}).([]string)
	for _, id := range rules {
		t.useResource("rule", id)
	}
}

// resourceSnapshot matches the backend's normalized form: items sorted by ID,
// identified by the sha256 of their JSON.
func resourceSnapshot(resources []turnResource) (string, []map[string]any) {
	type item struct {
		ID        string `json:"id"`
		Kind      string `json:"kind"`
		Enabled   bool   `json:"enabled"`
		Available bool   `json:"available"`
	}
	items := make([]item, 0, len(resources))
	seen := map[string]bool{}
	for _, r := range resources {
		if !seen[r.ID] {
			seen[r.ID] = true
			items = append(items, item{r.ID, r.Kind, true, true})
		}
	}
	slices.SortFunc(items, func(a, b item) int { return strings.Compare(a.ID, b.ID) })
	data, _ := json.Marshal(items)
	out := make([]map[string]any, len(items))
	for i, it := range items {
		out[i] = map[string]any{"resource_id": it.ID, "kind": it.Kind, "enabled": true, "available": true}
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data)), out
}

// skillEvents turns DSH skill tool calls into skill usage. Expert skills are
// installed as aibert-<skill ID>; other names are DSH's bundled skills.
func (t acpTurn) skillEvents() []map[string]any {
	var events []map[string]any
	for _, run := range t.ToolRuns {
		if run.DSHCallID == "" || run.Pending || run.Call == nil || run.Call.Function.Name != "skill" {
			continue
		}
		var args struct{ Name string }
		name := ""
		if json.Unmarshal([]byte(run.Call.Function.Arguments), &args) == nil {
			name = strings.TrimSpace(args.Name)
		}
		if name == "" || len(name) > 256 {
			continue
		}
		if len(events) == 200 { // backend limit per turn
			break
		}
		event := map[string]any{"ordinal": len(events), "skill_id": name, "name": name, "origin": "builtin", "trigger": "model", "ok": run.Report.Failed == 0}
		if id := strings.TrimPrefix(name, "aibert-"); id != name && validUUID(id) {
			event["skill_id"], event["origin"] = id, "server"
		}
		events = append(events, event)
	}
	return events
}
