package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type acpAttachment struct {
	Name     string       `json:"name"`
	Ref      relayFileRef `json:"fileRef"`
	UploadID string       `json:"uploadId,omitempty"`
	URI      string       `json:"uri,omitempty"`
}

func sessionQueue(id string, seq int) map[string]any {
	return map[string]any{"sessionId": id, "eventSeq": seq, "queue": map[string]any{"paused": false, "messages": []any{}}}
}

func (h *relaySessionHost) readQueue(id string) (map[string]any, error) {
	if id == h.sessionID || h.local != nil || h.shareDesktopSessions {
		archive, err := readLegacySessionArchive(h.profile, id)
		if err == nil {
			page, err := legacyHistoryPage(archive)
			if err != nil {
				return nil, err
			}
			return sessionQueue(id, int(page.EventSeq)), nil
		} else if id == h.sessionID {
			return nil, err
		}
	}
	if h.live == nil {
		return nil, errors.New("session_not_found")
	}
	s, err := h.live.load(id)
	if err != nil {
		return nil, err
	}
	seq := s.historySequence()
	return sessionQueue(id, seq), nil
}

func (h *relaySessionHost) handleQueue(ctx context.Context, ws *websocket.Conn, r *http.Request, raw json.RawMessage) (any, error) {
	var p struct {
		SessionID, Action, ID, Text string
		SendIfIdle                  bool
		Attachments                 []acpAttachment
		CanvasNodes                 []json.RawMessage
		Meta                        struct{ OperationID string } `json:"_meta"`
	}
	if json.Unmarshal(raw, &p) != nil || !validUUID(p.SessionID) {
		return nil, errors.New("invalid_session_queue")
	}
	if p.Action == "get" {
		return h.readQueue(p.SessionID)
	}
	if p.Action != "add" || !p.SendIfIdle || !validUUID(p.ID) || strings.TrimSpace(p.Text) == "" || len(p.Text) > 16<<10 || len(p.Attachments) > 4 || len(p.CanvasNodes) != 0 || h.live == nil {
		return nil, errors.New("session_queue_action_unavailable")
	}
	if err := h.acpScope(r, "session:control"); err != nil {
		return nil, errRelayAccess
	}
	if err := h.acpScope(r, "files:read"); err != nil {
		return nil, errRelayAccess
	}
	if len(p.Attachments) != 0 && h.local == nil {
		if h.uploads == nil {
			return nil, errInvalidUpload
		}
		readAccess, err := h.access.authorizeAccess(r, "files:read")
		if err != nil {
			return nil, errRelayAccess
		}
		h.uploads.mu.Lock()
		for index, file := range p.Attachments {
			var row *relayUpload
			var err error
			if file.URI != "" {
				h.uploads.mu.Unlock()
				return nil, errInvalidUpload
			}
			if file.UploadID != "" && file.Ref == (relayFileRef{}) {
				row, err = h.uploads.slot(file.UploadID, p.SessionID, readAccess.Auth)
			} else if file.UploadID == "" {
				row, err = h.uploads.completed(file.Ref, readAccess.Auth)
			}
			if err != nil || row == nil || row.Ref == nil || row.SessionID != p.SessionID || row.Name != file.Name {
				h.uploads.mu.Unlock()
				return nil, errInvalidUpload
			}
			p.Attachments[index] = acpAttachment{Name: row.Name, Ref: *row.Ref}
		}
		h.uploads.mu.Unlock()
	}
	if _, err := h.promptACP(ctx, ws, r, p.SessionID, p.Meta.OperationID, p.ID, p.Text, p.Attachments...); err != nil {
		return nil, err
	}
	result, err := h.readQueue(p.SessionID)
	if err != nil {
		return nil, err
	}
	result["clientMessageId"], result["action"] = p.ID, p.Action
	if err := wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "method": "_monkeyai/session/history_changed",
		"params": map[string]any{"sessionId": p.SessionID, "eventSeq": result["eventSeq"]}}); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *relaySessionHost) attachmentPrompt(ctx context.Context, session, prompt string, files []acpAttachment) (string, map[string]string, error) {
	if len(files) == 0 {
		return prompt, nil, nil
	}
	var payload strings.Builder
	payload.WriteString(prompt)
	remainingBytes := int64(relayMaxUpload)
	remainingText := maxAttachmentText
	counts := make(map[string]int, len(files))
	contents := make(map[string]string, len(files))
	for _, file := range files {
		var original *os.File
		var err error
		if h.local != nil {
			original, err = openDesktopAttachment(h.local.attachments, file, remainingBytes)
		} else if h.uploads != nil {
			original, err = h.uploads.promptFile(file.Ref, session, file.Name, remainingBytes)
		}
		if err != nil || original == nil {
			return "", nil, errors.New("attachment_unavailable_or_too_large")
		}
		data, err := io.ReadAll(io.LimitReader(original, remainingBytes+1))
		_ = original.Close()
		if err != nil || int64(len(data)) > remainingBytes {
			return "", nil, errors.New("attachment_unavailable_or_too_large")
		}
		remainingBytes -= int64(len(data))
		counts[file.Name]++
		if counts[file.Name] > 1 {
			delete(contents, file.Name)
		}
		payload.WriteString("\n\n[Attachment: " + file.Name + "]\n")
		if remainingText == 0 {
			payload.WriteString("[Attachment text limit reached. The original file remains available.]")
			continue
		}
		extracted, err := h.extractAttachment(ctx, file.Name, data)
		if ctx.Err() != nil {
			return "", nil, ctx.Err()
		}
		if err != nil || strings.TrimSpace(extracted.Text) == "" {
			payload.WriteString("[No readable text was extracted. The original file remains available.]")
			continue
		}
		part := limitAttachmentText(extracted.Text, remainingText)
		remainingText -= len(part.Text)
		payload.WriteString(part.Text)
		if extracted.Truncated || part.Truncated {
			payload.WriteString("\n[Attachment text truncated.]")
		}
		if file.Name != h.approvedFile && counts[file.Name] == 1 && part.Text != "" {
			contents[file.Name] = part.Text
		}
	}
	return payload.String(), contents, nil
}

func (h *relaySessionHost) sessionOperation(operation string, a relayAuthorization) (any, error) {
	if h.live == nil || !validUUID(operation) {
		return nil, errUploadOperationMissing
	}
	l := h.live
	l.mu.Lock()
	defer l.mu.Unlock()
	target := map[string]string{"serverId": a.ServerID, "userId": a.OwnerUserID, "machineId": a.MachineID}
	for id, session := range l.sessions {
		if session.Operation == operation {
			return map[string]any{"version": 1, "operationId": operation, "target": target, "sessionId": id, "method": "session/new", "state": "completed", "result": map[string]string{"sessionId": id}}, nil
		}
		for _, turn := range session.Turns {
			if turn.Operation != operation || turn.ClientID == "" {
				continue
			}
			snapshot := map[string]any{"version": 1, "operationId": operation, "target": target, "sessionId": id, "method": "_monkeyai/session/queue", "clientMessageId": turn.ClientID}
			if turn.Ended.IsZero() {
				snapshot["state"] = "running"
				return snapshot, nil
			}
			if turn.Failed {
				snapshot["state"] = "failed"
				snapshot["error"] = map[string]any{"code": -32000, "message": "session_turn_failed"}
				return snapshot, nil
			}
			seq := session.historySequence()
			result := sessionQueue(id, seq)
			result["clientMessageId"], result["action"] = turn.ClientID, "add"
			snapshot["state"], snapshot["result"] = "completed", result
			return snapshot, nil
		}
	}
	return nil, errUploadOperationMissing
}
