package main

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestOfficePreviewPackageAcceptsEmptyDirectoriesAndRejectsExternalResources(t *testing.T) {
	for _, test := range []struct {
		name, path, content string
		wantError           bool
	}{
		{"empty_directory", "ppt/embeddings/", "", false},
		{"embedded_object", "ppt/embeddings/object.bin", "fixture", true},
		{"external_resource", "ppt/_rels/presentation.xml.rels", `<Relationships><Relationship TargetMode="External" Target="https://example.invalid/example.png"/></Relationships>`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var data bytes.Buffer
			archive := zip.NewWriter(&data)
			if _, err := archive.Create("ppt/presentation.xml"); err != nil {
				t.Fatal(err)
			}
			part, err := archive.Create(test.path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := part.Write([]byte(test.content)); err != nil {
				t.Fatal(err)
			}
			if err := archive.Close(); err != nil {
				t.Fatal(err)
			}
			if err := verifyOfficePreview(&data, "pptx"); (err != nil) != test.wantError {
				t.Fatalf("verifyOfficePreview: %v", err)
			}
		})
	}
}

func TestOfficePreviewRequiresDocumentPartForTheRequestedFormat(t *testing.T) {
	for format, entry := range map[string]string{
		"docx": "word/document.xml",
		"xlsx": "xl/workbook.xml",
		"pptx": "ppt/presentation.xml",
	} {
		t.Run(format, func(t *testing.T) {
			var data bytes.Buffer
			archive := zip.NewWriter(&data)
			if _, err := archive.Create(entry); err != nil {
				t.Fatal(err)
			}
			if err := archive.Close(); err != nil {
				t.Fatal(err)
			}
			if err := verifyOfficePreview(bytes.NewReader(data.Bytes()), format); err != nil {
				t.Fatal(err)
			}
			for _, other := range []string{"pptx", "docx", "xlsx"} {
				if other != format && verifyOfficePreview(bytes.NewReader(data.Bytes()), other) == nil {
					t.Fatalf("%s accepted as %s", format, other)
				}
			}
		})
	}
}
