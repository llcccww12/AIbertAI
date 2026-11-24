package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceTextPages(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	name := "资料.md"
	source := strings.Repeat("a", workspaceTextPageBytes-1) + "界\r\n" + strings.Repeat("more lines\n", 100000)
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(source)
	var output strings.Builder
	var cursor, saved workspaceTextCursor
	for {
		raw, err := previewWorkspaceTextPage(root, name, name, cursor)
		if err != nil {
			t.Fatal(err)
		}
		page := raw.(map[string]any)
		body, ok := page["content"].(string)
		if !ok || page["kind"] != "markdown" || len(body) > workspaceTextPageBytes || strings.Count(body, "\n") > workspaceTextPageLines {
			t.Fatalf("invalid page offset=%d size=%d kind=%v", cursor.Offset, len(body), page["kind"])
		}
		output.WriteString(body)
		next, more := page["nextOffset"].(int64)
		if !more {
			break
		}
		if next != int64(output.Len()) || next <= cursor.Offset {
			t.Fatalf("invalid offset %d", next)
		}
		cursor = workspaceTextCursor{Offset: next, Version: page["version"].(string)}
		saved = cursor
	}
	if output.String() != source {
		t.Fatal("paging changed text or CRLF/UTF-8 boundaries")
	}
	write(source + "different")
	if _, err := previewWorkspaceTextPage(root, name, name, saved); err == nil {
		t.Fatal("mixed two file versions")
	}
	if err := os.Symlink(name, filepath.Join(dir, "link.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := previewWorkspaceTextPage(root, "link.md", "link.md", workspaceTextCursor{}); err == nil {
		t.Fatal("preview followed symlink")
	}
}
