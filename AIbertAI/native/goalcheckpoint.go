package main

import (
	"encoding/json"
	"reflect"
)

// Only tools whose implementations are known to be read-only can be safely
// reconsidered after a crash. Connector annotations are not a guarantee.
func safeGoalRead(name string) bool {
	switch name {
	case workspaceReadTool, legacyReadTool, legacyGlobTool, legacyGrepTool, directoryTool,
		designReadTool, designSelectionTool, "media_get_image_model", "design_catalog_search", "design_catalog_detail":
		return true
	}
	return false
}

func sameGoalAction(before, after modelToolUse) bool {
	if before.Function.Name != after.Function.Name {
		return false
	}
	var first, second any
	return json.Unmarshal([]byte(before.Function.Arguments), &first) == nil &&
		json.Unmarshal([]byte(after.Function.Arguments), &second) == nil && reflect.DeepEqual(first, second)
}
