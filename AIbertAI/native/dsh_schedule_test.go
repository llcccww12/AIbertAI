package main

import (
	"encoding/json"
	"testing"
)

func TestDSHSessionProcessDisablesHostScheduler(t *testing.T) {
	model := selectedModelConfig{ID: "test-model", Protocol: "openai_chat_completions"}
	gateway := &dshLocalGateway{model: model.ID, protocol: model.Protocol}
	for _, plugins := range [][]string{nil, {}, {"tool-schedule", "tool-fs-search"}} {
		disabled, err := dshDisabledPlugins(plugins)
		if err != nil {
			t.Fatal(err)
		}
		patch, err := gateway.Patch("http://127.0.0.1:12345/v1", model, disabled...)
		if err != nil {
			t.Fatal(err)
		}
		var rows []struct {
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		}
		if err := json.Unmarshal(patch, &rows); err != nil {
			t.Fatal(err)
		}
		entries := map[string]bool{}
		for _, row := range rows {
			entries[row.ID] = row.Disabled
		}
		if !entries["schedule"] || !entries["tool-schedule"] || !entries["hmr"] {
			t.Fatalf("session selection %v enables a host-wide scheduler: %v", plugins, entries)
		}
	}
}
