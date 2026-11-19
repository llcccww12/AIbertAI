package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadApprovedToolFileRejectsOtherFilesAndSymlinks(t *testing.T) {
	dir := t.TempDir()
	const approved = "报告.txt"
	const expected = "合成验收内容\n"
	for _, name := range []string{approved, "other.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(expected), 0600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if value, err := readApprovedToolFile(root, approved, approved); err != nil || value != expected {
		t.Fatalf("approved file: %q, %v", value, err)
	}
	for _, name := range []string{"./" + approved, "other.txt", "../" + approved} {
		if _, err := readApprovedToolFile(root, approved, name); err == nil {
			t.Errorf("accepted unapproved path %q", name)
		}
	}
	if err := os.Symlink("other.txt", filepath.Join(dir, "link.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := readApprovedToolFile(root, "link.txt", "link.txt"); err == nil {
		t.Error("accepted a symlink in place of an approved file")
	}
}
