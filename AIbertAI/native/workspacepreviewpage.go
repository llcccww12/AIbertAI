package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// The workspace read route already authorizes the session and resolves its root.
type workspaceTextCursor struct {
	Offset  int64
	Version string
}

const workspaceTextPageBytes = 64 << 10
const workspaceTextPageLines = 500

func previewWorkspaceTextPage(root *os.Root, relative, requested string, cursor workspaceTextCursor) (any, error) {
	if cursor.Offset < 0 || (cursor.Offset == 0) != (cursor.Version == "") {
		return nil, errors.New("invalid_request")
	}
	file, err := openAttachmentFile(root, relative, 1<<40)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	version := fmt.Sprintf("%x-%x", info.Size(), info.ModTime().UnixNano())
	if cursor.Offset > info.Size() || cursor.Offset > 0 && version != cursor.Version {
		return nil, errors.New("workspace_file_changed")
	}
	data, err := io.ReadAll(io.NewSectionReader(file, cursor.Offset, min(workspaceTextPageBytes, info.Size()-cursor.Offset)))
	if err != nil {
		return nil, err
	}
	// Formats with their own complete-document renderer must never receive a
	// partial HTML document or binary container in its place.
	if cursor.Offset == 0 && !strings.HasPrefix(http.DetectContentType(data), "text/") {
		return map[string]any{"path": requested, "kind": "unavailable", "reason": "too_large"}, nil
	}
	count, end := 0, 0
	for count < workspaceTextPageLines {
		newline := bytes.IndexByte(data[end:], '\n')
		if newline < 0 {
			break
		}
		count++
		end += newline + 1
	}
	if count == workspaceTextPageLines {
		data = data[:end]
	} else if cursor.Offset+int64(len(data)) < info.Size() && len(data) > 0 {
		start := len(data) - 1
		for start > 0 && !utf8.RuneStart(data[start]) {
			start--
		}
		if !utf8.FullRune(data[start:]) {
			data = data[:start]
		}
	}
	if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return map[string]any{"path": requested, "kind": "unavailable", "reason": "binary"}, nil
	}
	after, err := file.Stat()
	if err != nil || !sameWorkspaceVersion(info, after) {
		return nil, errors.New("workspace_file_changed")
	}
	kind := "text"
	switch strings.ToLower(filepath.Ext(relative)) {
	case ".md", ".markdown":
		kind = "markdown"
	}
	page := map[string]any{"path": requested, "kind": kind, "content": string(data), "version": version, "lineBreaks": count}
	if next := cursor.Offset + int64(len(data)); next < info.Size() {
		page["nextOffset"] = next
	}
	return page, nil
}
