package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDSHTeamPresetPatch(t *testing.T) {
	g := &dshLocalGateway{model: "m", protocol: "openai_chat_completions", standardMode: true,
		teamExperts: []dshTeamExpert{{PresetID: "expert-abcdef12", Name: "财务专家", Description: "审核报销", Prompt: "role {{x}}"}}}
	data, err := g.Patch("http://127.0.0.1:9/v1", selectedModelConfig{ID: "m", Protocol: "openai_chat_completions"})
	if err != nil {
		t.Fatal(err)
	}
	patch := string(data)
	for _, want := range []string{`"allowedAgentPresets":["standard","expert-abcdef12"]`, `"id":"agent-preset-registry"`, `"prefix":"role { {x}}"`} {
		if !strings.Contains(patch, want) {
			t.Fatalf("patch missing %s: %s", want, patch)
		}
	}
	// An expert session keeps its own composition.
	g.standardMode = false
	data, _ = g.Patch("http://127.0.0.1:9/v1", selectedModelConfig{ID: "m", Protocol: "openai_chat_completions"})
	var rows []map[string]any
	if json.Unmarshal(data, &rows) != nil || strings.Contains(string(data), "agent-preset-registry") {
		t.Fatal("expert session received team presets")
	}
	if got := dshTeamExpertInstructions(g.teamExperts); !strings.Contains(got, "expert-abcdef12: 财务专家 — 审核报销") {
		t.Fatalf("instructions = %q", got)
	}
	if dshTeamExpertInstructions(nil) != "" {
		t.Fatal("instructions without experts")
	}
}
