package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const maxToolText = 128 << 10

// Only the file selected by the local operator belongs to this tool's scope.
func readApprovedToolFile(root *os.Root, approved, requested string) (string, error) {
	invalid := errors.New("tool_file_unavailable")
	if approved == "." || approved != requested || filepath.Base(requested) != requested || !filepath.IsLocal(requested) || strings.ContainsAny(requested, "\\\x00") {
		return "", invalid
	}
	info, err := root.Lstat(requested)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxToolText {
		return "", invalid
	}
	file, err := root.Open(requested)
	if err != nil {
		return "", invalid
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return "", invalid
	}
	data, err := io.ReadAll(io.LimitReader(file, maxToolText+1))
	if err != nil || len(data) > maxToolText || !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
		return "", invalid
	}
	return string(data), nil
}
