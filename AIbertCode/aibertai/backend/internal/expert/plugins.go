package expert

import (
	"encoding/json"
	"slices"
)

var dshPlugins = []string{
	"agent-instructions",
	"time-context",
	"tool-bash",
	"tool-pwsh",
	"tool-fs",
	"tool-fs-search",
	"tool-jobs",
	"skill-filesystem",
	"tool-skill",
	"tool-ask-user",
	"tool-todo",
	"tool-web",
	"present",
	"tool-office",
	"tool-agent-team",
	"tool-subagent-control",
	"tool-subagent",
	"tool-subagent-fork",
	"tool-plugin-manager",
	"tool-workflow",
}

var canvasTools = []string{
	"canvas_get_document",
	"canvas_get_selection",
	"canvas_create_text",
	"canvas_create_html",
	"canvas_commit_html",
	"canvas_prepare_pptx",
	"design_catalog_search",
	"design_catalog_detail",
	"design_await_style_choice",
	"media_get_image_model",
	"media_generate_image",
	"canvas_create_image",
}

func validToolSelection(value any, allowed []string) bool {
	if value == nil {
		return true
	}
	data, err := json.Marshal(value)
	var names []string
	if err != nil || json.Unmarshal(data, &names) != nil || len(names) > len(allowed) {
		return false
	}
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if seen[name] || !slices.Contains(allowed, name) {
			return false
		}
		seen[name] = true
	}
	return true
}
