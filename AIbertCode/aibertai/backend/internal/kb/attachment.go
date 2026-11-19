package kb

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"aibertai.local/backend/internal/identity"
	"aibertai.local/backend/internal/resource"
)

const attachmentExtractTextLimit = 128 << 10

type attachmentExtraction struct {
	Text      string `json:"text"`
	Truncated bool   `json:"truncated,omitempty"`
}

func (s *Service) extractAttachment(w http.ResponseWriter, r *http.Request) {
	principal, ok := identity.PrincipalFromContext(r.Context())
	if !ok || principal.ClientID != "monkeyai-desktop" {
		resource.Fail(r.Context(), w, &resource.Error{Status: http.StatusForbidden, Code: "attachment_permission_denied", Message: "此接口供桌面客户端读取附件"})
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, visualDocumentMaxBytes))
	if err != nil || len(body) == 0 {
		resource.Fail(r.Context(), w, resource.InvalidCode("attachment_upload_invalid", "附件为空或超过 10 MiB"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 75*time.Second)
	defer cancel()
	contentType := http.DetectContentType(body)
	if contentType != "application/pdf" && contentType != "image/png" && contentType != "image/jpeg" {
		resource.Fail(r.Context(), w, resource.InvalidCode("attachment_format_unsupported", "此接口解析 PDF、PNG 和 JPEG"))
		return
	}
	if contentType == "application/pdf" {
		if result, err := extractPDFText(ctx, body); err == nil && result.Text != "" {
			resource.JSON(r.Context(), w, http.StatusOK, result)
			return
		}
	}
	if contentType != "application/pdf" {
		if err := validateVisualImage(body); err != nil {
			resource.Fail(r.Context(), w, err)
			return
		}
	}
	if s.documentVision == nil {
		resource.Fail(r.Context(), w, visionNotConfigured())
		return
	}
	config, err := s.documentVision(ctx)
	if err != nil {
		resource.Fail(r.Context(), w, err)
		return
	}
	if config.APIKey == "" || config.BaseURL == "" || config.ModelID == "" {
		resource.Fail(r.Context(), w, visionNotConfigured())
		return
	}
	text, err := parseVisualDocument(ctx, config, contentType, body)
	if err != nil {
		resource.Fail(r.Context(), w, err)
		return
	}
	resource.JSON(r.Context(), w, http.StatusOK, attachmentResult(text, false))
}

func attachmentResult(text string, truncated bool) attachmentExtraction {
	text = strings.ToValidUTF8(strings.ReplaceAll(text, "\x00", ""), "\uFFFD")
	if len(text) > attachmentExtractTextLimit {
		end := attachmentExtractTextLimit
		for !utf8.RuneStart(text[end]) {
			end--
		}
		text, truncated = text[:end], true
	}
	return attachmentExtraction{strings.TrimSpace(text), truncated}
}

// Keep the extractor's stdout bounded even for highly compressed pages. An
// output overflow closes the pipe and the child is reaped by Cmd.Run.
type attachmentTextWriter struct {
	buffer    bytes.Buffer
	truncated bool
}

var errAttachmentOutputFull = errors.New("attachment text limit")

func (w *attachmentTextWriter) Write(p []byte) (int, error) {
	remaining := attachmentExtractTextLimit + utf8.UTFMax - w.buffer.Len()
	if len(p) > remaining {
		w.truncated = true
		n, _ := w.buffer.Write(p[:remaining])
		return n, errAttachmentOutputFull
	}
	return w.buffer.Write(p)
}

func extractPDFText(ctx context.Context, data []byte) (attachmentExtraction, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var command *exec.Cmd
	if renderer, err := exec.LookPath("pdftotext"); err == nil {
		command = exec.CommandContext(ctx, renderer, "-layout", "-enc", "UTF-8", "-", "-")
		command.Stdin = bytes.NewReader(data)
	} else if renderer, err := exec.LookPath("pdf2txt.py"); err == nil {
		// Some native private installations already provide PDFMiner.
		dir, err := os.MkdirTemp("", "aibert-attachment-")
		if err != nil {
			return attachmentExtraction{}, err
		}
		defer os.RemoveAll(dir)
		input := filepath.Join(dir, "input.pdf")
		if err := os.WriteFile(input, data, 0600); err != nil {
			return attachmentExtraction{}, err
		}
		command = exec.CommandContext(ctx, renderer, "-t", "text", input)
	} else {
		return attachmentExtraction{}, errors.New("PDF text parser unavailable")
	}
	var out attachmentTextWriter
	command.Env = []string{"LANG=C.UTF-8", "PYTHONIOENCODING=utf-8"}
	command.Stdout = &out
	if err := command.Run(); err != nil && !out.truncated {
		return attachmentExtraction{}, err
	}
	return attachmentResult(out.buffer.String(), out.truncated), nil
}
