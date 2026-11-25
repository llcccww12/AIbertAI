package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUploadKnowledgeFile(t *testing.T) {
	const kb = "11111111-1111-4111-8111-111111111111"
	const folder = "22222222-2222-4222-8222-222222222222"
	c, err := newBackendClient("https://gateway.test", false)
	if err != nil {
		t.Fatal(err)
	}
	var saved []map[string]any
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/kb/"+kb+"/documents" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		saved = append(saved, body)
		w.WriteHeader(http.StatusAccepted)
		w.WriteString(`{"id":"33333333-3333-4333-8333-333333333333"}`)
		return w.Result(), nil
	})
	h := &relaySessionHost{live: &acpLive{client: c}}
	upload := func(name string, data []byte, parent *string) error {
		raw, _ := json.Marshal(knowledgeUpload{ID: kb, ParentID: parent, FileName: name, Data: base64.StdEncoding.EncodeToString(data)})
		_, err := h.uploadKnowledgeFile(t.Context(), authorization{Access: "test"}, raw)
		return err
	}
	var doc bytes.Buffer
	archive := zip.NewWriter(&doc)
	part, _ := archive.Create("word/document.xml")
	part.Write([]byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>季度报告正文</w:t></w:r></w:p></w:body></w:document>`))
	archive.Close()
	parent := folder
	if err = upload("季度报告.docx", doc.Bytes(), &parent); err != nil {
		t.Fatal(err)
	}
	if err = upload("notes.md", []byte("# 笔记\n内容"), nil); err != nil {
		t.Fatal(err)
	}
	if len(saved) != 2 || saved[0]["title"] != "季度报告" || saved[0]["parent_id"] != folder || saved[0]["file_name"] != "季度报告.docx" ||
		!strings.Contains(saved[0]["content"].(string), "季度报告正文") || saved[1]["parent_id"] != nil || saved[1]["content"] != "# 笔记\n内容" {
		t.Fatalf("saved documents: %v", saved)
	}
	if err = upload("tool.exe", []byte{0x4d, 0x5a, 0, 0}, nil); err == nil || err.Error() != "kb_upload_format_unsupported" {
		t.Fatalf("binary file accepted: %v", err)
	}
	if err = upload("../escape.md", []byte("x"), nil); err == nil {
		t.Fatal("path in file name accepted")
	}
}
