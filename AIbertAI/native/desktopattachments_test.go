package main

import (
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func attachmentURI(path string) string {
	path = filepath.ToSlash(path)
	if runtime.GOOS == "windows" {
		path = "/" + path
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}

func TestDesktopAttachmentReadRequiresStagedFiles(t *testing.T) {
	profile := t.TempDir()
	root, err := openDesktopAttachments(profile)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	id := "1ad55c3b-8b36-44ed-8526-35b22356fe9a"
	if err := root.Mkdir(id, 0700); err != nil {
		t.Fatal(err)
	}
	name := "验收附件.txt"
	path := filepath.Join(root.Name(), id, name)
	want := "attachment line\n"
	if err := os.WriteFile(path, []byte(want), 0600); err != nil {
		t.Fatal(err)
	}
	file := acpAttachment{Name: name, URI: attachmentURI(path)}
	if actual, err := readDesktopAttachment(root, file); err != nil || actual != want {
		t.Fatalf("staged attachment returned %q, %v", actual, err)
	}
	other := filepath.Join(profile, name)
	if err := os.WriteFile(other, []byte("outside scope"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, file := range []acpAttachment{
		{Name: name, URI: attachmentURI(other)},
		{Name: name, URI: attachmentURI(path), UploadID: id},
		{Name: "wrong.txt", URI: attachmentURI(path)},
		{Name: name, URI: attachmentURI(path) + "?extra=true"},
	} {
		if data, err := readDesktopAttachment(root, file); err == nil || data != "" {
			t.Fatalf("invalid attachment unexpectedly returned %q", data)
		}
	}
}
