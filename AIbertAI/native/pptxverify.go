package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

const maxEditableDeckBytes = 128 << 20

func verifyOfficePreview(input io.Reader, format string) error {
	const partLimit = 2 << 20
	document := map[string]string{"pptx": "ppt/presentation.xml", "docx": "word/document.xml", "xlsx": "xl/workbook.xml"}[format]
	invalid := errors.New("unsafe_office_preview_document")
	if document == "" {
		return invalid
	}
	data, err := io.ReadAll(io.LimitReader(input, maxEditableDeckBytes+1))
	if err != nil || len(data) > maxEditableDeckBytes {
		return invalid
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(archive.File) > 10000 {
		return invalid
	}
	var total uint64
	var hasDocument bool
	seen := make(map[string]bool, len(archive.File))
	for _, file := range archive.File {
		name := strings.ToLower(file.Name)
		clean := strings.TrimSuffix(name, "/")
		if clean == "" || path.IsAbs(clean) ||
			path.Clean(clean) != clean || clean == ".." || strings.HasPrefix(clean, "../") ||
			strings.ContainsAny(clean, "\\:") || seen[clean] || file.UncompressedSize64 > maxEditableDeckBytes {
			return invalid
		}
		seen[clean] = true
		if file.FileInfo().IsDir() {
			continue
		}
		if strings.Contains(name, "vbaproject") ||
			strings.Contains(name, "/activex/") || strings.Contains(name, "/externallinks/") ||
			strings.Contains(name, "/querytables/") || name == "xl/connections.xml" {
			return invalid
		}
		total += file.UncompressedSize64
		if total > 512<<20 {
			return invalid
		}
		if strings.Contains(name, "/embeddings/") {
			if format != "pptx" || path.Dir(name) != "ppt/embeddings" || path.Ext(name) != ".xlsx" ||
				file.UncompressedSize64 > 8<<20 {
				return invalid
			}
			book, err := file.Open()
			if err != nil {
				return invalid
			}
			err = verifyOfficePreview(io.LimitReader(book, (8<<20)+1), "xlsx")
			book.Close()
			if err != nil {
				return invalid
			}
			continue
		}
		hasDocument = hasDocument || file.Name == document
		if !strings.HasSuffix(name, ".rels") {
			continue
		}
		if file.UncompressedSize64 > partLimit {
			return invalid
		}
		part, err := file.Open()
		if err != nil {
			return invalid
		}
		decoder := xml.NewDecoder(io.LimitReader(part, partLimit+1))
		for {
			token, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				part.Close()
				return invalid
			}
			if start, ok := token.(xml.StartElement); ok && start.Name.Local == "Relationship" {
				for _, attr := range start.Attr {
					if attr.Name.Local == "TargetMode" && strings.EqualFold(attr.Value, "external") {
						part.Close()
						return invalid
					}
				}
			}
		}
		part.Close()
	}
	if !hasDocument {
		return invalid
	}
	return nil
}

func verifyEditableDeck(input io.Reader, expected int) error {
	const drawingNS = "http://schemas.openxmlformats.org/drawingml/2006/main"
	const presentationNS = "http://schemas.openxmlformats.org/presentationml/2006/main"
	const chartNS = "http://schemas.openxmlformats.org/drawingml/2006/chart"
	const relationshipsNS = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	invalidChart := errors.New("invalid_pptx_chart_data")
	if expected < 1 || expected > 512 {
		return errors.New("invalid_pptx_slide_order")
	}
	data, err := io.ReadAll(io.LimitReader(input, maxEditableDeckBytes+1))
	if err != nil || len(data) > maxEditableDeckBytes {
		return errors.New("invalid_pptx_result")
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(archive.File) > 10000 {
		return errors.New("invalid_pptx_result")
	}
	files := make(map[string]*zip.File)
	for _, file := range archive.File {
		if _, duplicate := files[file.Name]; duplicate {
			return errors.New("invalid_pptx_result")
		}
		files[file.Name] = file
	}
	parse := func(name string, scan func(*xml.Decoder, xml.StartElement) error) error {
		const maxPartBytes = 12 << 20
		file := files[name]
		if file == nil || file.UncompressedSize64 > maxPartBytes {
			return errors.New("invalid_pptx_result")
		}
		part, err := file.Open()
		if err != nil {
			return err
		}
		defer part.Close()
		decoder := xml.NewDecoder(io.LimitReader(part, maxPartBytes+1))
		for {
			token, err := decoder.Token()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			if item, ok := token.(xml.StartElement); ok {
				if err := scan(decoder, item); err != nil {
					return err
				}
			}
		}
	}
	pages := 0
	if err := parse("ppt/presentation.xml", func(_ *xml.Decoder, item xml.StartElement) error {
		if item.Name.Space == presentationNS && item.Name.Local == "sldId" {
			pages++
		}
		return nil
	}); err != nil || pages != expected || files["[Content_Types].xml"] == nil {
		return errors.New("invalid_pptx_result")
	}
	relatedPart := func(owner, id, kind, prefix, extension string) string {
		if id == "" {
			return ""
		}
		var target string
		relationships := path.Join(path.Dir(owner), "_rels", path.Base(owner)+".rels")
		err := parse(relationships, func(_ *xml.Decoder, item xml.StartElement) error {
			if item.Name.Local != "Relationship" {
				return nil
			}
			fields := make(map[string]string, len(item.Attr))
			for _, attribute := range item.Attr {
				fields[attribute.Name.Local] = attribute.Value
			}
			if fields["Id"] == id && fields["Type"] == relationshipsNS+"/"+kind && fields["TargetMode"] == "" {
				raw := fields["Target"]
				if strings.HasPrefix(raw, "/") {
					target = path.Clean(strings.TrimPrefix(raw, "/"))
				} else {
					target = path.Join(path.Dir(owner), raw)
				}
			}
			return nil
		})
		if err != nil || !strings.HasPrefix(target, prefix) || path.Ext(target) != extension || files[target] == nil {
			return ""
		}
		return target
	}
	relationID := func(item xml.StartElement) string {
		for _, attribute := range item.Attr {
			if attribute.Name.Space == relationshipsNS && attribute.Name.Local == "id" {
				return attribute.Value
			}
		}
		return ""
	}
	for i := 1; i <= expected; i++ {
		editable := false
		slide := fmt.Sprintf("ppt/slides/slide%d.xml", i)
		err := parse(slide, func(decoder *xml.Decoder, item xml.StartElement) error {
			if item.Name.Space == chartNS && item.Name.Local == "chart" {
				chart := relatedPart(slide, relationID(item), "chart", "ppt/charts/", ".xml")
				var workbook string
				if parse(chart, func(_ *xml.Decoder, item xml.StartElement) error {
					if item.Name.Space == chartNS && item.Name.Local == "externalData" {
						workbook = relatedPart(chart, relationID(item), "package", "ppt/embeddings/", ".xlsx")
					}
					return nil
				}) != nil || workbook == "" {
					return invalidChart
				}
				editable = true
				return nil
			}
			if item.Name.Space != drawingNS || item.Name.Local != "t" {
				return nil
			}
			var value string
			if err := decoder.DecodeElement(&value, &item); err != nil {
				return err
			}
			editable = editable || strings.TrimSpace(value) != ""
			return nil
		})
		if errors.Is(err, invalidChart) {
			return err
		}
		if err != nil || !editable {
			return fmt.Errorf("slide_%d_lost_editable_text", i)
		}
	}
	return nil
}
