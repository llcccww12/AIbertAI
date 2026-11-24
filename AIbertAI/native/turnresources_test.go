package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestTurnResourcesAndSkillEvents(t *testing.T) {
	const expert, skill, rule, model = "a1bd985c-9f71-4b85-937a-c41b1b104d38", "a2bd985c-9f71-4b85-937a-c41b1b104d38", "a3bd985c-9f71-4b85-937a-c41b1b104d38", "a4bd985c-9f71-4b85-937a-c41b1b104d38"
	manifest := desktopExpert{expertSelection: expertSelection{ID: expert}, Skills: []desktopExpertSkill{{ID: skill}}}
	ctx := context.WithValue(t.Context(), dshExpertManifestKey{}, manifest)
	ctx = context.WithValue(ctx, dshAppliedRulesKey{}, []string{rule})
	turn := acpTurn{Model: model}
	turn.recordDSHResources(ctx, acpSession{ExpertID: expert})
	turn.useResource("connector", "not-a-uuid")
	turn.useResource("rule", rule)
	if len(turn.Resources) != 4 {
		t.Fatalf("resources = %+v", turn.Resources)
	}
	id, items := resourceSnapshot(turn.Resources)
	if len(items) != 4 || items[0]["resource_id"] != expert || id == fmt.Sprintf("sha256:%x", sha256.Sum256([]byte("[]"))) {
		t.Fatalf("snapshot %s %+v", id, items)
	}
	if empty, _ := resourceSnapshot(nil); empty != fmt.Sprintf("sha256:%x", sha256.Sum256([]byte("[]"))) {
		t.Fatalf("empty snapshot changed: %s", empty)
	}

	call := func(args string, failed int) workspaceToolRun {
		c := &modelToolUse{}
		c.Function.Name, c.Function.Arguments = "skill", args
		return workspaceToolRun{DSHCallID: "call", Call: c, Report: reportedTool{Failed: failed}}
	}
	turn.ToolRuns = []workspaceToolRun{call(`{"name":"aibert-`+skill+`"}`, 0), call(`{"name":"office-docx"}`, 1), call(`{}`, 0)}
	events := turn.skillEvents()
	if len(events) != 2 || events[0]["skill_id"] != skill || events[0]["origin"] != "server" ||
		events[1]["skill_id"] != "office-docx" || events[1]["origin"] != "builtin" || events[1]["ok"] != false || events[1]["ordinal"] != 1 {
		t.Fatalf("skill events = %+v", events)
	}
}
