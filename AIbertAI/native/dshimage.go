package main

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"path"
	"strings"
)

type dshImageInputKey struct{}

type dshImageAdmission uint8

const (
	dshImageNotSent dshImageAdmission = iota
	dshImageOutcomeUnknown
	dshImageAdmitted
)

type dshImageInput struct {
	blocks []map[string]string
}

// prepareDSHImageInput reads only attachments already authorized for this
// session. The ordered blocks are ephemeral; the SDK persists admitted images.
func (h *relaySessionHost) prepareDSHImageInput(ctx context.Context, session, prompt string, files []acpAttachment, vision bool) (dshImageInput, bool, error) {
	if err := ctx.Err(); err != nil {
		return dshImageInput{}, false, err
	}
	blocks := []map[string]string{{"type": "text", "text": prompt}}
	remaining, textRemaining := int64(relayMaxUpload), maxAttachmentText
	hasImage := false
	for _, file := range files {
		data, err := h.readPromptAttachment(session, file, remaining)
		if err != nil {
			return dshImageInput{}, false, err
		}
		if err := ctx.Err(); err != nil {
			return dshImageInput{}, false, err
		}
		remaining -= int64(len(data))
		mime := http.DetectContentType(data)
		image := mime == "image/png" || mime == "image/jpeg" || mime == "image/webp" || mime == "image/gif"
		ext := strings.ToLower(path.Ext(file.Name))
		if !image && (ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".webp" || ext == ".gif" || strings.HasPrefix(mime, "image/")) {
			return dshImageInput{}, false, errors.New("attachment_image_invalid")
		}
		if image {
			if !vision {
				// 当前模型看不了图。整轮直接失败会把会话卡死：之后每条消息都会撞在
				// 同一张图上，用户只能退出重开。这里把图换成一句说明，让这轮照常进行。
				blocks = append(blocks, map[string]string{"type": "text", "text": "\n\n[附件 " + file.Name + "：当前模型不支持图片，这张图没有发送。换成支持图片的模型后重新提问即可。]"})
				continue
			}
			hasImage = true
			blocks = append(blocks, map[string]string{"type": "image", "data": base64.StdEncoding.EncodeToString(data), "mimeType": mime, "name": file.Name})
			continue
		}
		if mime == "application/pdf" {
			extracted, err := h.extractAttachment(context.WithValue(ctx, dshAttachmentExtractionKey{}, true), file.Name, data)
			if err := ctx.Err(); err != nil {
				return dshImageInput{}, false, err
			}
			if err == nil && strings.TrimSpace(extracted.Text) != "" {
				text := extracted.Text
				if extracted.Truncated {
					text += "\n[Attachment text truncated.]"
				}
				blocks = append(blocks, map[string]string{"type": "text", "text": "\n\n[Attachment: " + file.Name + "]\n" + text})
				continue
			}
			blocks = append(blocks, map[string]string{"type": "text", "text": "\n\n[Attachment: " + file.Name + "]\n[No readable text was extracted. The original file remains available.]"})
			continue
		}
		// Plain text is already usable by the SDK; the legacy extraction
		// preview limit must not silently cut off the model's attachment.
		if mime != "application/pdf" && ext != ".docx" && ext != ".xlsx" && ext != ".pptx" && isAttachmentText(data) {
			blocks = append(blocks, map[string]string{"type": "text", "text": "\n\n[Attachment: " + file.Name + "]\n" + string(data)})
			continue
		}
		if mime != "application/pdf" && (ext == ".docx" || ext == ".xlsx" || ext == ".pptx") {
			extracted, err := extractOfficeAttachmentWithLimit(ext, data, relayMaxUpload)
			if err := ctx.Err(); err != nil {
				return dshImageInput{}, false, err
			}
			if err == nil && strings.TrimSpace(extracted.Text) != "" {
				text := extracted.Text
				if extracted.Truncated {
					text += "\n[Attachment text truncated.]"
				}
				blocks = append(blocks, map[string]string{"type": "text", "text": "\n\n[Attachment: " + file.Name + "]\n" + text})
				continue
			}
			blocks = append(blocks, map[string]string{"type": "text", "text": "\n\n[Attachment: " + file.Name + "]\n[No readable text was extracted. The original file remains available.]"})
			continue
		}
		text := "[No readable text was extracted. The original file remains available.]"
		if textRemaining > 0 {
			extracted, err := h.extractAttachment(ctx, file.Name, data)
			if ctx.Err() != nil {
				return dshImageInput{}, false, ctx.Err()
			}
			if err == nil && strings.TrimSpace(extracted.Text) != "" {
				part := limitAttachmentText(extracted.Text, textRemaining)
				textRemaining -= len(part.Text)
				text = part.Text
				if extracted.Truncated || part.Truncated {
					text += "\n[Attachment text truncated.]"
				}
			}
		} else {
			text = "[Attachment text limit reached. The original file remains available.]"
		}
		blocks = append(blocks, map[string]string{"type": "text", "text": "\n\n[Attachment: " + file.Name + "]\n" + text})
	}
	return dshImageInput{blocks: blocks}, hasImage, nil
}
