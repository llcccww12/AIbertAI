package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	legacyReadTool  = "Read"
	legacyWriteTool = "Write"
	legacyEditTool  = "Edit"
	legacyGlobTool  = "Glob"
	legacyGrepTool  = "Grep"
	directoryTool   = "list_directory"
)

type builtinToolPreferences struct {
	Whitelist  []string `json:"whitelist,omitempty"`
	Blacklist  []string `json:"blacklist,omitempty"`
	additional *builtinToolPreferences
}

func (p builtinToolPreferences) allows(name string) bool {
	if p.additional != nil && !p.additional.allows(name) {
		return false
	}
	alias := name
	switch name {
	case workspaceReadTool:
		alias = legacyReadTool
	case workspaceWriteTool:
		alias = legacyWriteTool
	}
	return !slices.Contains(p.Blacklist, name) && !slices.Contains(p.Blacklist, alias) &&
		(len(p.Whitelist) == 0 || slices.Contains(p.Whitelist, name) || slices.Contains(p.Whitelist, alias))
}

func (p builtinToolPreferences) intersect(other builtinToolPreferences) builtinToolPreferences {
	p.additional = &other
	return p
}

type workspaceToolPolicy struct {
	Preferences  builtinToolPreferences
	CanWrite     bool
	DesignTools  []desktopDesignTool
	Instructions string
	Goal         *desktopGoalRun
	SkillScript  bool
}

func (p *workspaceToolPolicy) extraDefinitions() []map[string]any {
	if p == nil {
		return nil
	}
	var tools []map[string]any
	add := func(name, description string, properties map[string]any, required ...string) {
		if !p.Preferences.allows(name) {
			return
		}
		if required == nil {
			required = []string{}
		}
		// Preserve optional parameters in the legacy schema for both model APIs.
		schema, _ := json.Marshal(map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false})
		tools = append(tools, map[string]any{"name": name, "description": description, "parameters": json.RawMessage(schema)})
	}
	str := func() any { return map[string]string{"type": "string"} }
	if p.SkillScript {
		add(skillScriptTool, "Execute a packaged Python, shell, or JavaScript script from the selected skill after approval. The working directory is the session workspace.",
			map[string]any{"path": str(), "arguments": map[string]any{"type": "array", "items": str()}}, "path")
	}
	boolean := map[string]string{"type": "boolean"}
	integer := map[string]any{"type": "integer", "minimum": 0}
	add(legacyReadTool, "Read a UTF-8 file in this workspace. Offset counts lines from zero.",
		map[string]any{"file_path": str(), "offset": integer, "limit": integer}, "file_path")
	add(legacyGlobTool, "Find workspace paths by glob, such as **/*.txt. Symlinks are not followed.",
		map[string]any{"pattern": str(), "path": str(), "hidden": boolean, "limit": integer}, "pattern")
	add(legacyGrepTool, "Find text in the workspace using a regular expression or literal string.",
		map[string]any{"pattern": str(), "path": str(), "include": str(), "output_mode": map[string]any{"type": "string", "enum": []string{"content", "files"}},
			"literal": boolean, "case_sensitive": boolean, "context_lines": integer, "limit": integer}, "pattern")
	add(directoryTool, "List one directory in the workspace. Use . to list the workspace root.",
		map[string]any{"path": str(), "offset": integer}, "path")
	for _, tool := range p.DesignTools {
		var schema struct{ Type string }
		if !designTool(tool.Name) || designWriteTool(tool.Name) && !p.CanWrite ||
			json.Unmarshal(tool.Schema, &schema) != nil || schema.Type != "object" {
			continue
		}
		tools = append(tools, map[string]any{"name": tool.Name, "description": tool.Description, "parameters": tool.Schema})
	}
	if p.CanWrite {
		add(legacyWriteTool, "Write complete UTF-8 text to a workspace file.", map[string]any{"file_path": str(), "content": str()}, "file_path", "content")
		edit := map[string]any{"type": "object", "properties": map[string]any{"old_string": str(), "new_string": str(), "replace_all": boolean},
			"required": []string{"old_string", "new_string"}, "additionalProperties": false}
		add(legacyEditTool, "Edit existing UTF-8 text by exact match. Pass one replacement or an atomic batch of replacements against the original file.",
			map[string]any{"file_path": str(), "old_string": str(), "new_string": str(), "replace_all": boolean,
				"edits": map[string]any{"type": "array", "items": edit}}, "file_path")
		add(legacyPatchTool, "Apply an OpenAI-style workspace patch. Use *** Begin Patch and *** End Patch, with Add File, Update File, Delete File, or Move to sections. Context lines start with a space, added lines with +, and removed lines with -.", map[string]any{"patch": str()}, "patch")
	}
	return tools
}

type workspaceReplacement struct {
	Old string  `json:"old_string"`
	New *string `json:"new_string"`
	All bool    `json:"replace_all"`
}

type workspaceCompatCall struct {
	Path      string
	Offset    int
	Limit     *int
	Pattern   string
	Include   string
	Mode      string
	Hidden    bool
	Literal   bool
	Sensitive bool
	Context   int
	Content   *string
	Edits     []workspaceReplacement
	Patch     []workspacePatchFile
	PatchPlan *workspacePatchPlan
}

func builtinWorkspaceWrite(name string) bool {
	return name == workspaceWriteTool || name == legacyWriteTool || name == legacyEditTool || name == legacyPatchTool
}

func decodeWorkspaceTool(raw json.RawMessage, into any) error {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil || decoder.Decode(new(any)) != io.EOF {
		return errors.New("invalid_model_tool_call")
	}
	return nil
}

func parseWorkspaceCompatCall(name string, raw json.RawMessage) (*workspaceCompatCall, error) {
	invalid := errors.New("invalid_model_tool_call")
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || len(raw) > 8*maxToolText || raw[0] != '{' {
		return nil, invalid
	}
	switch name {
	case legacyReadTool:
		var in struct {
			Path   string `json:"file_path"`
			Offset int    `json:"offset"`
			Limit  *int   `json:"limit"`
		}
		if decodeWorkspaceTool(raw, &in) != nil || in.Path == "" || in.Offset < 0 || in.Limit != nil && *in.Limit < 1 {
			return nil, invalid
		}
		return &workspaceCompatCall{Path: in.Path, Offset: in.Offset, Limit: in.Limit}, nil
	case legacyWriteTool:
		var in struct {
			Path    string  `json:"file_path"`
			Content *string `json:"content"`
		}
		if decodeWorkspaceTool(raw, &in) != nil || in.Path == "" || !validWorkspaceToolText(in.Content) {
			return nil, invalid
		}
		return &workspaceCompatCall{Path: in.Path, Content: in.Content}, nil
	case legacyEditTool:
		var in struct {
			Path  string                 `json:"file_path"`
			Old   *string                `json:"old_string"`
			New   *string                `json:"new_string"`
			All   *bool                  `json:"replace_all"`
			Edits []workspaceReplacement `json:"edits"`
		}
		if decodeWorkspaceTool(raw, &in) != nil || in.Path == "" {
			return nil, invalid
		}
		if in.Edits == nil {
			if in.Old == nil {
				return nil, invalid
			}
			in.Edits = []workspaceReplacement{{*in.Old, in.New, in.All != nil && *in.All}}
		} else if in.Old != nil || in.New != nil || in.All != nil || len(in.Edits) == 0 {
			return nil, invalid
		}
		for _, edit := range in.Edits {
			if edit.Old == "" || !validWorkspaceToolText(&edit.Old) || !validWorkspaceToolText(edit.New) {
				return nil, invalid
			}
		}
		return &workspaceCompatCall{Path: in.Path, Edits: in.Edits}, nil
	case legacyGlobTool, legacyGrepTool, directoryTool:
		return parseWorkspaceSearch(name, raw)
	case legacyPatchTool:
		var input struct{ Patch string }
		if decodeWorkspaceTool(raw, &input) != nil {
			return nil, invalid
		}
		patch, err := parseWorkspacePatch(input.Patch)
		if err != nil {
			return nil, err
		}
		return &workspaceCompatCall{Patch: patch}, nil
	default:
		return nil, invalid
	}
}

func validWorkspaceToolText(text *string) bool {
	return text != nil && len(*text) <= maxToolText && utf8.ValidString(*text) && !strings.ContainsRune(*text, 0)
}

func resolveWorkspaceToolPath(cwd, requested string) (string, error) {
	if filepath.IsAbs(requested) {
		var err error
		requested, err = filepath.Rel(cwd, requested)
		if err != nil {
			return "", errWorkspacePath
		}
		requested = filepath.ToSlash(requested)
	}
	_, err := workspaceRelative(requested)
	return requested, err
}

func (h *relaySessionHost) executeWorkspaceCompat(ctx context.Context, row acpSession, name string, call workspaceCompatCall, attachments map[string]string) (string, error) {
	if name == legacyPatchTool && call.PatchPlan != nil {
		return h.applyToolWorkspacePatch(ctx, row, *call.PatchPlan)
	}
	path := call.Path
	if path == "" {
		path = "."
	}
	relative, err := resolveWorkspaceToolPath(row.CWD, path)
	if err != nil {
		return "", err
	}
	if name == legacyReadTool {
		if text, exists := attachments[relative]; exists {
			return workspaceTextLines(text, call.Offset, call.Limit), nil
		}
	}
	switch name {
	case legacyWriteTool:
		return h.writeToolWorkspaceFile(ctx, row, relative, *call.Content)
	case legacyEditTool:
		return h.editToolWorkspaceFile(ctx, row, relative, call.Edits)
	}
	root, err := h.live.workspaces.sessionRoot(row)
	if err != nil {
		return "", err
	}
	defer root.Close()
	switch name {
	case legacyReadTool:
		text, err := readSearchableText(root, relative)
		if err != nil {
			return "", err
		}
		return workspaceTextLines(text, call.Offset, call.Limit), nil
	case directoryTool:
		page, err := listWorkspaceDirectory(root, filepath.FromSlash(relative), relative, call.Offset)
		if err != nil {
			return "", err
		}
		data, err := json.Marshal(page)
		return string(data), err
	default:
		return searchWorkspace(ctx, root, relative, name, call)
	}
}

func workspaceTextLines(text string, offset int, limit *int) string {
	var output strings.Builder
	written := 0
	for index, line := range strings.SplitAfter(text, "\n") {
		if index < offset || line == "" {
			continue
		}
		if limit != nil && written == *limit {
			break
		}
		if output.Len()+len(line)+24 > maxToolText {
			if written == 0 {
				return "[Line too large for a text preview]"
			}
			output.WriteString("\n[Output truncated. Read from a later offset.]")
			break
		}
		fmt.Fprintf(&output, "%d\t%s", index+1, line)
		written++
	}
	return output.String()
}

func (h *relaySessionHost) editToolWorkspaceFile(ctx context.Context, row acpSession, path string, edits []workspaceReplacement) (string, error) {
	workspaces := h.live.workspaces
	workspaces.writes.Lock()
	defer workspaces.writes.Unlock()
	root, err := workspaces.sessionRoot(row)
	if err != nil {
		return "", err
	}
	defer root.Close()
	relative := filepath.FromSlash(path)
	file, err := openAttachmentFile(root, relative, maxToolText)
	if err != nil {
		return "", err
	}
	before, err := file.Stat()
	if err != nil {
		file.Close()
		return "", err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxToolText+1))
	if err := errors.Join(readErr, file.Close()); err != nil {
		return "", err
	}
	original := string(data)
	if !validWorkspaceToolText(&original) {
		return "", errors.New("tool_file_unavailable")
	}
	edited, replacements, err := replaceWorkspaceText(original, edits)
	if err != nil {
		return "", err
	}
	if err := writeWorkspaceTextIfUnchanged(ctx, root, path, edited, before); err != nil {
		return "", err
	}
	return fmt.Sprintf("Applied %d replacements in %s", replacements, path), nil
}

func replaceWorkspaceText(original string, edits []workspaceReplacement) (string, int, error) {
	type match struct {
		start, end int
		value      string
	}
	var matches []match
	occupied := make([]bool, len(original))
	for _, edit := range edits {
		if edit.Old == "" || edit.New == nil {
			return "", 0, errors.New("invalid_model_tool_call")
		}
		offset, count := 0, 0
		for {
			found := strings.Index(original[offset:], edit.Old)
			if found < 0 {
				break
			}
			start := offset + found
			offset = start + len(edit.Old)
			count++
			if count > 1 && !edit.All {
				return "", 0, errors.New("workspace_edit_ambiguous")
			}
			for index := start; index < offset; index++ {
				if occupied[index] {
					return "", 0, errors.New("workspace_edits_overlap")
				}
				occupied[index] = true
			}
			matches = append(matches, match{start, offset, *edit.New})
		}
		if count == 0 {
			return "", 0, errors.New("workspace_edit_text_not_found")
		}
	}
	slices.SortFunc(matches, func(a, b match) int { return a.start - b.start })
	var output strings.Builder
	offset := 0
	for _, item := range matches {
		if output.Len()+item.start-offset+len(item.value) > maxToolText {
			return "", 0, errors.New("workspace_edit_output_too_large")
		}
		output.WriteString(original[offset:item.start])
		output.WriteString(item.value)
		offset = item.end
	}
	if output.Len()+len(original)-offset > maxToolText {
		return "", 0, errors.New("workspace_edit_output_too_large")
	}
	output.WriteString(original[offset:])
	return output.String(), len(matches), nil
}
