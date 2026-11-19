package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type workspaceFileStat struct {
	Exists  bool  `json:"exists"`
	Size    int64 `json:"size"`
	MtimeMS int64 `json:"mtimeMs"`
}

func statWorkspaceFile(rootName, name string) (workspaceFileStat, error) {
	var result workspaceFileStat
	if !filepath.IsAbs(rootName) || name == "" || strings.ContainsRune(name, 0) {
		return result, errors.New("invalid_path")
	}
	relative := name
	if filepath.IsAbs(name) {
		var err error
		relative, err = filepath.Rel(rootName, name)
		if err != nil {
			return result, errors.New("invalid_path")
		}
	}
	// Reject parent components even where joining them would yield a local path.
	if !filepath.IsLocal(relative) || relative == "." {
		return result, errors.New("invalid_path")
	}
	for _, part := range strings.Split(filepath.ToSlash(relative), "/") {
		if part == ".." {
			return result, errors.New("invalid_path")
		}
	}
	root, err := os.OpenRoot(rootName)
	if err != nil {
		return result, errors.New("invalid_path")
	}
	defer root.Close()
	info, err := root.Stat(relative)
	if err != nil || !info.Mode().IsRegular() {
		return result, errors.New("invalid_path")
	}
	return workspaceFileStat{Exists: true, Size: info.Size(), MtimeMS: info.ModTime().UnixMilli()}, nil
}
