package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

const workspaceSearchDefault = 100
const workspaceSearchMaxMatches = 1000
const workspaceSearchMaxBytes = 32 << 20

func parseWorkspaceSearch(name string, raw json.RawMessage) (*workspaceCompatCall, error) {
	invalid := errors.New("invalid_model_tool_call")
	switch name {
	case directoryTool:
		var in struct {
			Path   string
			Offset int
		}
		if decodeWorkspaceTool(raw, &in) != nil || in.Path == "" || in.Offset < 0 {
			return nil, invalid
		}
		return &workspaceCompatCall{Path: in.Path, Offset: in.Offset}, nil
	case legacyGlobTool:
		var in struct {
			Path, Pattern string
			Hidden        bool
			Limit         *int
		}
		if decodeWorkspaceTool(raw, &in) != nil || validateWorkspaceGlob(in.Pattern) != nil || in.Limit != nil && *in.Limit < 1 {
			return nil, invalid
		}
		return &workspaceCompatCall{Path: in.Path, Pattern: in.Pattern, Hidden: in.Hidden, Limit: in.Limit}, nil
	default:
		var in struct {
			Path, Pattern, Include string
			Mode                   string `json:"output_mode"`
			Literal                bool
			Sensitive              *bool `json:"case_sensitive"`
			Context                int   `json:"context_lines"`
			Limit                  *int
		}
		if decodeWorkspaceTool(raw, &in) != nil || in.Pattern == "" || len(in.Pattern) > 4096 ||
			(in.Include != "" && validateWorkspaceGlob(in.Include) != nil) || (in.Mode != "" && in.Mode != "content" && in.Mode != "files") ||
			in.Context < 0 || in.Context > workspaceSearchMaxMatches || in.Limit != nil && *in.Limit < 1 {
			return nil, invalid
		}
		return &workspaceCompatCall{Path: in.Path, Pattern: in.Pattern, Include: in.Include, Mode: in.Mode, Hidden: true, Literal: in.Literal,
			Sensitive: in.Sensitive == nil || *in.Sensitive, Context: in.Context, Limit: in.Limit}, nil
	}
}

func readSearchableText(root *os.Root, name string) (string, error) {
	file, err := openAttachmentFile(root, filepath.FromSlash(name), maxWorkspaceTextPreview)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxWorkspaceTextPreview+1))
	if err != nil || len(data) > maxWorkspaceTextPreview || !utf8.Valid(data) || slices.Contains(data, 0) {
		return "", errors.New("tool_file_unavailable")
	}
	return string(data), nil
}

func validateWorkspaceGlob(pattern string) error {
	if pattern == "" || len(pattern) > 4096 || strings.ContainsAny(pattern, "\\:\x00") || path.IsAbs(pattern) {
		return path.ErrBadPattern
	}
	for _, part := range strings.Split(pattern, "/") {
		if part == "" || part == "." || part == ".." {
			return path.ErrBadPattern
		}
		if _, err := path.Match(part, ""); err != nil {
			return err
		}
	}
	return nil
}

func matchesWorkspaceGlob(pattern, name string) bool {
	if !strings.Contains(pattern, "/") && pattern != "**" {
		ok, _ := path.Match(pattern, path.Base(name))
		return ok
	}
	parts := strings.Split(name, "/")
	states := make([]bool, len(parts)+1)
	states[0] = true
	for _, pattern := range strings.Split(pattern, "/") {
		if pattern == "**" {
			for index := 1; index < len(states); index++ {
				states[index] = states[index] || states[index-1]
			}
			continue
		}
		for index := len(parts); index > 0; index-- {
			match, _ := path.Match(pattern, parts[index-1])
			states[index] = states[index-1] && match
		}
		states[0] = false
	}
	return states[len(parts)]
}

type workspaceSearchLine struct {
	Path  string `json:"path"`
	Line  int    `json:"line"`
	Text  string `json:"text"`
	Match bool   `json:"match"`
}

type workspaceSearchOutput struct {
	Paths     []string              `json:"paths,omitempty"`
	Lines     []workspaceSearchLine `json:"lines,omitempty"`
	Truncated bool                  `json:"truncated"`
	Size      int                   `json:"-"`
}

var errWorkspaceSearchLimit = errors.New("workspace_search_limit")

func searchWorkspace(ctx context.Context, root *os.Root, directory, name string, call workspaceCompatCall) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	isGlob := name == legacyGlobTool
	if !isGlob && name != legacyGrepTool {
		return "", errors.New("invalid_model_tool_call")
	}
	limit := workspaceSearchDefault
	if call.Limit != nil {
		limit = min(*call.Limit, workspaceSearchMaxMatches)
	}
	var matcher *regexp.Regexp
	if !isGlob {
		pattern := call.Pattern
		if call.Literal {
			pattern = regexp.QuoteMeta(pattern)
		}
		if !call.Sensitive {
			pattern = "(?i)" + pattern
		}
		var err error
		matcher, err = regexp.Compile(pattern)
		if err != nil {
			return "", errors.New("workspace_invalid_search_pattern")
		}
	}
	var result workspaceSearchOutput
	scanned, matched := 0, 0
	err := fs.WalkDir(root.FS(), directory, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		base := entry.Name()
		if name != directory && (base == ".git" || !call.Hidden && strings.HasPrefix(base, ".")) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		relative := strings.TrimPrefix(name, directory+"/")
		if _, err := workspaceRelative(name); err != nil {
			return nil
		}
		if isGlob {
			if matchesWorkspaceGlob(call.Pattern, relative) {
				return result.addPath(name, limit)
			}
			return nil
		}
		if call.Include != "" && !matchesWorkspaceGlob(call.Include, relative) {
			return nil
		}
		text, err := readSearchableText(root, name)
		if err != nil {
			return nil
		}
		scanned += len(text)
		if scanned > workspaceSearchMaxBytes {
			return errWorkspaceSearchLimit
		}
		lines := strings.Split(text, "\n")
		if call.Mode == "files" {
			for _, line := range lines {
				if matcher.MatchString(line) {
					return result.addPath(name, limit)
				}
			}
			return nil
		}
		last := -1
		for index, line := range lines {
			if !matcher.MatchString(line) {
				continue
			}
			start, end := max(last+1, index-call.Context), min(len(lines), index+call.Context+1)
			for next := start; next < end; next++ {
				value := strings.TrimSuffix(lines[next], "\r")
				if result.Size+6*(len(name)+len(value))+96 > maxToolText {
					return errWorkspaceSearchLimit
				}
				result.Lines = append(result.Lines, workspaceSearchLine{name, next + 1, value, matcher.MatchString(lines[next])})
				result.Size += 6*(len(name)+len(value)) + 96
				last = next
			}
			matched++
			if matched >= limit {
				return errWorkspaceSearchLimit
			}
		}
		return nil
	})
	if errors.Is(err, errWorkspaceSearchLimit) {
		result.Truncated = true
	} else if err != nil {
		return "", err
	}
	data, err := json.Marshal(result)
	return string(data), err
}

func (o *workspaceSearchOutput) addPath(name string, limit int) error {
	size := 6*len(name) + 4
	if o.Size+size > maxToolText {
		return errWorkspaceSearchLimit
	}
	o.Paths = append(o.Paths, name)
	o.Size += size
	if len(o.Paths) >= limit {
		return errWorkspaceSearchLimit
	}
	return nil
}
