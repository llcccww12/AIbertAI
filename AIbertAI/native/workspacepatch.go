package main

import (
	"errors"
	"slices"
	"strings"
	"unicode/utf8"
)

const legacyPatchTool = "apply_patch"

var errWorkspacePatch = errors.New("workspace_invalid_patch")

type workspacePatchFile struct {
	Kind, Path, Move string
	Added            []string
	Hunks            []workspacePatchHunk
}

type workspacePatchHunk struct {
	Anchor   string
	Old, New []string
	EOF      bool
}

func parseWorkspacePatch(text string) ([]workspacePatchFile, error) {
	if len(text) > maxToolText || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return nil, errWorkspacePatch
	}
	lines := strings.Split(strings.TrimSuffix(strings.ReplaceAll(text, "\r\n", "\n"), "\n"), "\n")
	if len(lines) < 3 || lines[0] != "*** Begin Patch" || lines[len(lines)-1] != "*** End Patch" {
		return nil, errWorkspacePatch
	}
	lines = lines[1 : len(lines)-1]
	var files []workspacePatchFile
	for len(lines) > 0 {
		operation := workspacePatchFile{}
		header := lines[0]
		lines = lines[1:]
		switch {
		case strings.HasPrefix(header, "*** Add File: "):
			operation.Kind, operation.Path = "add", strings.TrimPrefix(header, "*** Add File: ")
			for len(lines) > 0 && strings.HasPrefix(lines[0], "+") {
				operation.Added = append(operation.Added, lines[0][1:])
				lines = lines[1:]
			}
		case strings.HasPrefix(header, "*** Delete File: "):
			operation.Kind, operation.Path = "delete", strings.TrimPrefix(header, "*** Delete File: ")
		case strings.HasPrefix(header, "*** Update File: "):
			operation.Kind, operation.Path = "update", strings.TrimPrefix(header, "*** Update File: ")
			if len(lines) > 0 && strings.HasPrefix(lines[0], "*** Move to: ") {
				operation.Move, lines = strings.TrimPrefix(lines[0], "*** Move to: "), lines[1:]
				if operation.Move == "" {
					return nil, errWorkspacePatch
				}
			}
			var err error
			operation.Hunks, lines, err = parseWorkspacePatchHunks(lines)
			if err != nil || operation.Move == "" && len(operation.Hunks) == 0 {
				return nil, errWorkspacePatch
			}
		default:
			return nil, errWorkspacePatch
		}
		if operation.Path == "" {
			return nil, errWorkspacePatch
		}
		files = append(files, operation)
	}
	return files, nil
}

func parseWorkspacePatchHunks(lines []string) ([]workspacePatchHunk, []string, error) {
	var hunks []workspacePatchHunk
	for len(lines) > 0 && !strings.HasPrefix(lines[0], "*** ") {
		var hunk workspacePatchHunk
		if lines[0] == "@@" || strings.HasPrefix(lines[0], "@@ ") {
			hunk.Anchor, lines = strings.TrimPrefix(lines[0][2:], " "), lines[1:]
		}
		changed := false
		for len(lines) > 0 && len(lines[0]) > 0 && strings.ContainsRune(" +-", rune(lines[0][0])) {
			line := lines[0]
			lines = lines[1:]
			if line[0] != '+' {
				hunk.Old = append(hunk.Old, line[1:])
			}
			if line[0] != '-' {
				hunk.New = append(hunk.New, line[1:])
			}
			changed = changed || line[0] != ' '
		}
		if !changed {
			return nil, nil, errWorkspacePatch
		}
		if len(lines) > 0 && lines[0] == "*** End of File" {
			hunk.EOF, lines = true, lines[1:]
		}
		hunks = append(hunks, hunk)
	}
	return hunks, lines, nil
}

func applyWorkspacePatchHunks(original string, hunks []workspacePatchHunk) (string, error) {
	if !validWorkspaceToolText(&original) {
		return "", errWorkspacePatch
	}
	lineEnd := "\n"
	if strings.Contains(original, "\r\n") {
		plain := strings.ReplaceAll(original, "\r\n", "\n")
		if strings.Count(plain, "\n") != strings.Count(original, "\r\n") {
			return "", errors.New("workspace_patch_mixed_line_endings")
		}
		lineEnd, original = "\r\n", plain
	}
	hasFinalNewline := strings.HasSuffix(original, "\n") || original == ""
	var lines []string
	if original != "" {
		lines = strings.Split(strings.TrimSuffix(original, "\n"), "\n")
	}
	cursor := 0
	for _, hunk := range hunks {
		start, err := workspacePatchMatch(lines, cursor, hunk)
		if err != nil {
			return "", err
		}
		lines = slices.Replace(lines, start, start+len(hunk.Old), hunk.New...)
		cursor = start + len(hunk.New)
	}
	result := strings.Join(lines, lineEnd)
	if hasFinalNewline && len(lines) > 0 {
		result += lineEnd
	}
	if len(result) > maxToolText {
		return "", errors.New("workspace_patch_output_too_large")
	}
	return result, nil
}

func workspacePatchMatch(lines []string, cursor int, hunk workspacePatchHunk) (int, error) {
	if hunk.Anchor != "" {
		index := slices.Index(lines[cursor:], hunk.Anchor)
		if index < 0 {
			return 0, errors.New("workspace_patch_anchor_not_found")
		}
		cursor += index + 1
	}
	last := len(lines) - len(hunk.Old)
	if hunk.EOF || len(hunk.Old) == 0 && hunk.Anchor == "" {
		if last >= cursor && slices.Equal(lines[last:], hunk.Old) {
			return last, nil
		}
		return 0, errors.New("workspace_patch_context_not_found")
	}
	if len(hunk.Old) == 0 {
		return cursor, nil
	}
	match := -1
	for index := cursor; index <= last; index++ {
		if slices.Equal(lines[index:index+len(hunk.Old)], hunk.Old) {
			if match >= 0 {
				return 0, errors.New("workspace_patch_ambiguous_context")
			}
			match = index
		}
	}
	if match < 0 {
		return 0, errors.New("workspace_patch_context_not_found")
	}
	return match, nil
}
