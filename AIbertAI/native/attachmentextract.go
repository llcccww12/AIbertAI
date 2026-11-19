package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

func isAttachmentText(data []byte) bool {
	return utf8.Valid(data) && bytes.IndexByte(data, 0) < 0
}

type attachmentText struct {
	Text      string `json:"text"`
	Truncated bool   `json:"truncated"`
}

func (c *client) extractVisualAttachment(ctx context.Context, auth authorization, data []byte) (attachmentText, error) {
	var result attachmentText
	resp, err := c.request(ctx, http.MethodPost, "/api/v1/attachments/extract", auth.Access, "", "", "application/octet-stream", bytes.NewReader(data))
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(io.LimitReader(resp.Body, 8*maxAttachmentText)).Decode(&result)
	return result, err
}

func limitAttachmentText(text string, limit int) attachmentText {
	if len(text) <= limit {
		return attachmentText{Text: text}
	}
	for limit > 0 && !utf8.RuneStart(text[limit]) {
		limit--
	}
	return attachmentText{Text: text[:limit], Truncated: true}
}

type officeAttachment struct {
	entries   map[string]*zip.File
	remaining int64
	text      strings.Builder
	truncated bool
}

func extractOfficeAttachment(extension string, data []byte) (attachmentText, error) {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return attachmentText{}, err
	}
	doc := officeAttachment{entries: make(map[string]*zip.File, len(archive.File)), remaining: 8 << 20}
	for _, entry := range archive.File {
		if doc.entries[entry.Name] != nil {
			return attachmentText{}, errors.New("duplicate office archive entry")
		}
		doc.entries[entry.Name] = entry
	}
	switch extension {
	case ".docx":
		err = doc.paragraphFile("word/document.xml")
	case ".pptx":
		err = doc.numberedFiles("ppt/slides/slide", "Slide", doc.paragraphFile)
	case ".xlsx":
		err = doc.spreadsheet()
	}
	if err != nil {
		return attachmentText{}, err
	}
	return attachmentText{Text: doc.text.String(), Truncated: doc.truncated}, nil
}

func (d *officeAttachment) append(text string) {
	if d.truncated || text == "" {
		return
	}
	part := limitAttachmentText(text, maxAttachmentText-d.text.Len())
	d.text.WriteString(part.Text)
	d.truncated = part.Truncated
}

func (d *officeAttachment) read(name string) ([]byte, error) {
	entry := d.entries[name]
	if entry == nil || entry.UncompressedSize64 > uint64(d.remaining) {
		return nil, errors.New("office attachment part unavailable or too large")
	}
	file, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, d.remaining+1))
	if err != nil || int64(len(data)) > d.remaining {
		return nil, errors.New("office attachment part unavailable or too large")
	}
	d.remaining -= int64(len(data))
	return data, nil
}

func (d *officeAttachment) paragraphFile(name string) error {
	raw, err := d.read(name)
	if err != nil {
		return err
	}
	dec := xml.NewDecoder(bytes.NewReader(raw))
	for !d.truncated {
		token, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch tag := token.(type) {
		case xml.StartElement:
			switch tag.Name.Local {
			case "t":
				var text string
				if err := dec.DecodeElement(&text, &tag); err != nil {
					return err
				}
				d.append(text)
			case "tab":
				d.append("\t")
			case "br":
				d.append("\n")
			}
		case xml.EndElement:
			if tag.Name.Local == "p" {
				d.append("\n")
			}
		}
	}
	return nil
}

func (d *officeAttachment) numberedFiles(prefix, label string, extract func(string) error) error {
	ordered := make(map[int]string)
	for name := range d.entries {
		suffix, ok := strings.CutPrefix(name, prefix)
		if !ok || !strings.HasSuffix(suffix, ".xml") {
			continue
		}
		number := strings.TrimSuffix(suffix, ".xml")
		value, err := strconv.Atoi(number)
		if err == nil && value > 0 && strconv.Itoa(value) == number {
			ordered[value] = name
		}
	}
	if len(ordered) == 0 {
		return errors.New("office attachment has no content parts")
	}
	for _, value := range slices.Sorted(maps.Keys(ordered)) {
		if d.truncated {
			break
		}
		d.append(fmt.Sprintf("\n[%s %d]\n", label, value))
		if err := extract(ordered[value]); err != nil {
			return err
		}
	}
	return nil
}

type officeRichText struct {
	Text string `xml:"t"`
	Runs []struct {
		Text string `xml:"t"`
	} `xml:"r"`
}

func (r officeRichText) value() string {
	var builder strings.Builder
	builder.WriteString(r.Text)
	for _, part := range r.Runs {
		builder.WriteString(part.Text)
	}
	return builder.String()
}

func (d *officeAttachment) spreadsheet() error {
	var shared struct {
		Strings []officeRichText `xml:"si"`
	}
	if d.entries["xl/sharedStrings.xml"] != nil {
		data, err := d.read("xl/sharedStrings.xml")
		if err != nil {
			return err
		}
		if err := xml.Unmarshal(data, &shared); err != nil {
			return err
		}
	}
	return d.numberedFiles("xl/worksheets/sheet", "Sheet", func(name string) error {
		data, err := d.read(name)
		if err != nil {
			return err
		}
		decoder := xml.NewDecoder(bytes.NewReader(data))
		for !d.truncated {
			token, err := decoder.Token()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			switch element := token.(type) {
			case xml.StartElement:
				if element.Name.Local != "c" {
					continue
				}
				var cell struct {
					Type  string         `xml:"t,attr"`
					Ref   string         `xml:"r,attr"`
					Value string         `xml:"v"`
					Rich  officeRichText `xml:"is"`
				}
				if err := decoder.DecodeElement(&cell, &element); err != nil {
					return err
				}
				value := cell.Value
				switch cell.Type {
				case "s":
					index, err := strconv.Atoi(value)
					if err != nil || index < 0 || index >= len(shared.Strings) {
						return errors.New("invalid shared spreadsheet value")
					}
					value = shared.Strings[index].value()
				case "inlineStr":
					value = cell.Rich.value()
				}
				if value != "" {
					d.append(cell.Ref + "=" + value + "\t")
				}
			case xml.EndElement:
				if element.Name.Local == "row" {
					d.append("\n")
				}
			}
		}
		return nil
	})
}

func (h *relaySessionHost) extractAttachment(ctx context.Context, name string, data []byte) (attachmentText, error) {
	switch http.DetectContentType(data) {
	case "application/pdf", "image/png", "image/jpeg":
		if h.live == nil || h.live.client == nil {
			return attachmentText{}, errors.New("attachment extractor unavailable")
		}
		auth, err := h.live.authorization(ctx)
		if err != nil {
			return attachmentText{}, err
		}
		return h.live.client.extractVisualAttachment(ctx, auth, data)
	}
	extension := strings.ToLower(path.Ext(name))
	switch extension {
	case ".docx", ".xlsx", ".pptx":
		return extractOfficeAttachment(extension, data)
	}
	if isAttachmentText(data) {
		return limitAttachmentText(string(data), maxAttachmentText), nil
	}
	return attachmentText{}, errors.New("no text available for this file")
}
