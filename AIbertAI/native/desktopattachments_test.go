package main

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func attachmentURI(path string) string {
	path = filepath.ToSlash(path)
	if runtime.GOOS == "windows" {
		path = "/" + path
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}

func TestDSHDocumentAttachmentOriginals(t *testing.T) {
	profile := t.TempDir()
	root, err := os.OpenRoot(profile)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	attachments, err := openDesktopAttachments(profile)
	if err != nil {
		t.Fatal(err)
	}
	defer attachments.Close()
	workspaces, err := openDesktopWorkspaces(root, filepath.Join(profile, "workspaces"), "")
	if err != nil {
		t.Fatal(err)
	}
	session := acpSession{ID: "a43ce4d4-581c-4fb4-a818-33d72f79642a", Workspace: "default"}
	session.CWD = workspaces.defaultPath(session.ID)
	if err := workspaces.allocate(session.ID); err != nil {
		t.Fatal(err)
	}
	host := &relaySessionHost{local: &desktopRuntimeAccess{attachments: attachments}, live: &acpLive{workspaces: workspaces}}
	const filename = "成本 表.xlsx"
	files := make([]acpAttachment, 0, 2)
	originals := make([][]byte, 0, 2)
	for i, cell := range []string{"模板", "替换来源"} {
		id := fmt.Sprintf("70fee8d6-059d-4f42-8a7d-%012d", i)
		data := officeTestArchive(t, map[string]string{
			"xl/worksheets/sheet1.xml": "<worksheet><sheetData><row><c r=\"A1\" t=\"inlineStr\"><is><t>" + cell + "</t></is></c><c r=\"B1\"><f>40+2</f><v>42</v></c></row></sheetData></worksheet>",
		})
		if err := attachments.Mkdir(id, 0700); err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(attachments.Name(), id, filename)
		if err := os.WriteFile(file, data, 0600); err != nil {
			t.Fatal(err)
		}
		originals = append(originals, data)
		files = append(files, acpAttachment{Name: filename, URI: attachmentURI(file)})
	}
	for range 2 {
		input, image, err := host.prepareDSHImageInput(t.Context(), session, "编辑工作簿", files, false)
		if err != nil || image || len(input.blocks) != 3 {
			t.Fatalf("invalid prepared input: %v, image %t, %d blocks", err, image, len(input.blocks))
		}
		paths := make(map[string]bool)
		for i, block := range input.blocks[1:] {
			_, tail, ok := strings.Cut(block["text"], `Original file in workspace: "`)
			relative, _, found := strings.Cut(tail, `"`)
			if !ok || !found || paths[relative] || !strings.Contains(block["text"], "B1=42") {
				t.Fatalf("invalid attachment block: %q", block["text"])
			}
			paths[relative] = true
			data, err := os.ReadFile(filepath.Join(session.CWD, filepath.FromSlash(relative)))
			if err != nil || !bytes.Equal(data, originals[i]) {
				t.Fatalf("workspace file does not match attachment %d: %v", i, err)
			}
		}
	}
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
