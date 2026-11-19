package main

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestOfficeAttachmentText(t *testing.T) {
	for _, tc := range []struct {
		name  string
		parts map[string]string
		want  string
	}{
		{
			name:  ".docx",
			parts: map[string]string{"word/document.xml": `<w:document xmlns:w="test"><w:body><w:p><w:r><w:t>文档正文</w:t><w:tab/><w:t>第二列</w:t></w:r></w:p></w:body></w:document>`},
			want:  "文档正文\t第二列\n",
		},
		{
			name: ".xlsx",
			parts: map[string]string{
				"xl/sharedStrings.xml":     `<sst><si><r><t>表格</t></r><r><t>正文</t></r></si></sst>`,
				"xl/worksheets/sheet1.xml": `<worksheet><sheetData><row><c r="A1" t="s"><v>0</v></c><c r="B1" t="inlineStr"><is><t>相邻文本</t></is></c><c r="C1"><f>40+2</f><v>42</v></c></row></sheetData></worksheet>`,
			},
			want: "\n[Sheet 1]\nA1=表格正文\tB1=相邻文本\tC1=42\t\n",
		},
		{
			name: ".pptx",
			parts: map[string]string{
				"ppt/slides/slide10.xml": `<p:sld xmlns:p="test"><p:p><p:t>第十页</p:t></p:p></p:sld>`,
				"ppt/slides/slide2.xml":  `<p:sld xmlns:p="test"><p:p><p:t>第二页</p:t></p:p></p:sld>`,
			},
			want: "\n[Slide 2]\n第二页\n\n[Slide 10]\n第十页\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractOfficeAttachment(tc.name, officeTestArchive(t, tc.parts))
			if err != nil || got.Text != tc.want || got.Truncated {
				t.Fatalf("extract: %+v, %v", got, err)
			}
		})
	}
}

func TestOfficeAttachmentLimits(t *testing.T) {
	part := func(body string) []byte {
		return officeTestArchive(t, map[string]string{"word/document.xml": body})
	}
	if got, err := extractOfficeAttachment(".docx", part("<p><t>partial</t>")); err == nil || got.Text != "" {
		t.Fatalf("invalid XML exposed partial text: %+v, %v", got, err)
	}
	if _, err := extractOfficeAttachment(".docx", part(strings.Repeat(" ", (8<<20)+1))); err == nil {
		t.Fatal("oversized ZIP part accepted")
	}
	got, err := extractOfficeAttachment(".docx", part("<p><t>"+strings.Repeat("文", maxAttachmentText/3+1)+"</t></p>"))
	if err != nil || !got.Truncated || len(got.Text) != maxAttachmentText/3*3 || !utf8.ValidString(got.Text) {
		t.Fatalf("UTF-8 limit: bytes=%d, truncated=%v, err=%v", len(got.Text), got.Truncated, err)
	}
}

func officeTestArchive(t *testing.T, parts map[string]string) []byte {
	t.Helper()
	var data bytes.Buffer
	writer := zip.NewWriter(&data)
	for name, body := range parts {
		part, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}
