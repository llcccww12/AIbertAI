package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"path"
	"strings"
	"unicode/utf8"
)

// Uploaded knowledge files are stored as their extracted text, like chat
// attachments, so personal knowledge bases accept documents and folders.
const (
	maxKnowledgeUpload     = 10 << 20
	maxKnowledgeUploadText = 1 << 20
)

type knowledgeUpload struct {
	ID       string  `json:"id"`
	ParentID *string `json:"parentId"`
	FileName string  `json:"fileName"`
	Data     string  `json:"data"`
}

func (h *relaySessionHost) uploadKnowledgeFile(ctx context.Context, auth authorization, raw json.RawMessage) (any, error) {
	var in knowledgeUpload
	invalid := errors.New("invalid_request")
	if json.Unmarshal(raw, &in) != nil || !validUUID(in.ID) || in.ParentID != nil && !validUUID(*in.ParentID) {
		return nil, invalid
	}
	name := strings.TrimSpace(in.FileName)
	if name == "" || len(name) > 255 || !utf8.ValidString(name) || strings.ContainsAny(name, `/\`) {
		return nil, invalid
	}
	if base64.StdEncoding.DecodedLen(len(in.Data)) > maxKnowledgeUpload+3 {
		return nil, errors.New("kb_upload_too_large")
	}
	data, err := base64.StdEncoding.DecodeString(in.Data)
	if err != nil || len(data) == 0 {
		return nil, invalid
	}
	if len(data) > maxKnowledgeUpload {
		return nil, errors.New("kb_upload_too_large")
	}
	mime := http.DetectContentType(data)
	var text attachmentText
	switch extension := strings.ToLower(path.Ext(name)); {
	case mime == "application/pdf" || mime == "image/png" || mime == "image/jpeg":
		text, err = h.live.client.extractVisualAttachment(ctx, auth, data)
	case extension == ".docx" || extension == ".xlsx" || extension == ".pptx":
		text, err = extractOfficeAttachmentWithLimit(extension, data, maxKnowledgeUploadText)
	case isAttachmentText(data):
		text = limitAttachmentText(string(data), maxKnowledgeUploadText)
	default:
		return nil, errors.New("kb_upload_format_unsupported")
	}
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(text.Text) == "" {
		return nil, errors.New("kb_upload_empty")
	}
	input := map[string]any{"title": strings.TrimSuffix(name, path.Ext(name)), "source_type": "file", "file_name": name,
		"mime_type": mime, "size_bytes": len(data), "content": text.Text, "parent_id": in.ParentID}
	if input["title"] == "" {
		input["title"] = name
	}
	body, _ := json.Marshal(input)
	return h.live.client.knowledgeChange(ctx, auth, knowledgeOperation{Verb: http.MethodPost, Path: "/api/v1/kb/" + in.ID + "/documents", Body: json.RawMessage(body)})
}
