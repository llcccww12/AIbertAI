package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceGlobAndGrepRespectWorkspaceAndOptions(t *testing.T) {
	dir := t.TempDir()
	write := func(path, text string) {
		t.Helper()
		filename := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("docs/sample.txt", "first\nneedle Value\nlast\n")
	write("docs/sub/sample.txt", "other Needle\n")
	write(".hidden.txt", "needle Value\n")
	write(".git/config.txt", "needle Value\n")
	write("docs/outside.md", "needle Value\n")
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("needle external\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_ = os.Symlink(outside, filepath.Join(dir, "docs", "alias.txt"))
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	call := func(name, query string) workspaceSearchOutput {
		t.Helper()
		input, err := parseWorkspaceCompatCall(name, json.RawMessage(query))
		if err != nil {
			t.Fatal(err)
		}
		folder := input.Path
		if folder == "" {
			folder = "."
		}
		value, err := searchWorkspace(t.Context(), root, folder, name, *input)
		if err != nil {
			t.Fatal(err)
		}
		var result workspaceSearchOutput
		if err := json.Unmarshal([]byte(value), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	glob := call(legacyGlobTool, `{"pattern":"**/*.txt"}`)
	if strings.Join(glob.Paths, ",") != "docs/sample.txt,docs/sub/sample.txt" {
		t.Fatalf("unexpected glob paths: %v", glob.Paths)
	}
	grep := call(legacyGrepTool, `{"pattern":"NEEDLE Value","path":"docs","include":"*.txt","literal":true,"case_sensitive":false,"context_lines":1}`)
	if len(grep.Lines) != 3 || grep.Lines[1].Line != 2 || !grep.Lines[1].Match || grep.Lines[0].Match || grep.Lines[2].Text != "last" {
		t.Fatalf("unexpected grep output: %v", grep.Lines)
	}
	if _, err := resolveWorkspaceToolPath(dir, outside); err == nil {
		t.Fatal("outside absolute path became a workspace path")
	}
}

func TestWorkspaceEditUsesOriginalFileAndRejectsOverlaps(t *testing.T) {
	old := "left beetle left"
	replacement := func(before, after string, all bool) workspaceReplacement {
		return workspaceReplacement{before, &after, all}
	}
	value, count, err := replaceWorkspaceText(old, []workspaceReplacement{replacement("left", "right", true), replacement("beetle", "wasp", false)})
	if err != nil || count != 3 || value != "right wasp right" {
		t.Fatalf("bad exact edit: %q, %d, %v", value, count, err)
	}
	if _, _, err := replaceWorkspaceText(old, []workspaceReplacement{replacement("left", "right", false)}); err == nil {
		t.Fatal("ambiguous edit was accepted")
	}
	if _, _, err := replaceWorkspaceText(old, []workspaceReplacement{replacement("beetle", "wasp", false), replacement("etle", "ant", false)}); err == nil {
		t.Fatal("overlapping edits were accepted")
	}
	if got := workspaceTextLines("one\r\ntwo\r\nthree", 1, new(1)); got != "2\ttwo\r\n" {
		t.Fatalf("read offset changed line numbering: %q", got)
	}
}
