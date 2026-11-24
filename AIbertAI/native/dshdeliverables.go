package main

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
)

type dshPresentedFile struct {
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
}

func (t *acpTurn) recordDeliverables(data json.RawMessage, turn int64) {
	var event struct {
		Turn   int64
		CallID string
		Files  []dshPresentedFile
	}
	if turn <= 0 || json.Unmarshal(data, &event) != nil || event.Turn != turn || event.CallID == "" {
		return
	}
	// A goal snapshot may still hold the previous slice.
	t.DSHDeliverables = append(slices.Clip(t.DSHDeliverables), event.Files...)
}

func (t acpTurn) presentedFiles(cwd string) []dshPresentedFile {
	if !filepath.IsAbs(cwd) {
		return nil
	}
	var files []dshPresentedFile
	index := make(map[string]int)
	add := func(f dshPresentedFile) {
		rel := f.Path
		if filepath.IsAbs(rel) {
			var err error
			if rel, err = filepath.Rel(cwd, rel); err != nil {
				return
			}
		}
		safe, err := workspaceRelative(filepath.ToSlash(rel))
		if err != nil || safe == "." {
			return
		}
		f.Path = filepath.ToSlash(safe)
		if i, ok := index[f.Path]; ok {
			files[i] = f
			return
		}
		index[f.Path] = len(files)
		files = append(files, f)
	}
	for _, f := range t.DSHDeliverables {
		add(f)
	}
	if len(t.DSHDeliverables) != 0 {
		return files
	}
	// Older clients saved the SDK's successful present result, before recording
	// deliverables/presented. Match its exact per-file confirmation to the call.
	for _, run := range t.workspaceTools() {
		if run.Pending || run.Report.Failed != 0 || run.DSHCallID == "" || run.Call == nil || run.Call.Function.Name != "present" {
			continue
		}
		var input struct{ Files []dshPresentedFile }
		if json.Unmarshal([]byte(run.Call.Function.Arguments), &input) != nil {
			continue
		}
		lines := strings.Split(run.Output, "\n")
		if len(input.Files) != len(lines) {
			continue
		}
		for i, f := range input.Files {
			target := f.Path
			if !filepath.IsAbs(target) {
				target = filepath.Join(cwd, target)
			}
			if lines[i] == "Presented "+target {
				f.Path = target
				add(f)
			}
		}
	}
	return files
}

// mutatedFiles lists the workspace files this turn changed, in the shape the
// client's changed-file list expects.
func (t acpTurn) mutatedFiles(cwd string) []map[string]any {
	var files []map[string]any
	seen := make(map[string]int)
	for _, run := range t.ToolRuns {
		if run.Call == nil || run.Report.Failed != 0 {
			continue
		}
		if name := run.Call.Function.Name; name != "write" && name != "edit" {
			continue
		}
		var args struct {
			Path string `json:"file_path"`
		}
		if json.Unmarshal([]byte(run.Call.Function.Arguments), &args) != nil || args.Path == "" {
			continue
		}
		path := args.Path
		if filepath.IsAbs(path) && filepath.IsAbs(cwd) {
			relative, err := filepath.Rel(cwd, path)
			if err != nil {
				continue
			}
			path = relative
		}
		path = filepath.ToSlash(path)
		if index, ok := seen[path]; ok {
			files[index]["added"] = files[index]["added"].(int) + run.FileAdded
			files[index]["deleted"] = files[index]["deleted"].(int) + run.FileDeleted
			continue
		}
		seen[path] = len(files)
		operation := run.FileOperation
		if operation == "" {
			operation = "updated"
		}
		files = append(files, map[string]any{"path": path, "operation": operation, "added": run.FileAdded, "deleted": run.FileDeleted})
	}
	return files
}
