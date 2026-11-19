package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type acpAttachment struct {
	Name         string       `json:"name"`
	Ref          relayFileRef `json:"fileRef"`
	UploadID     string       `json:"uploadId,omitempty"`
	URI          string       `json:"uri,omitempty"`
	CanvasNodeID string       `json:"canvasNodeId,omitempty"`
}

type acpCanvasNode struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	NodeType string `json:"nodeType"`
}

type desktopQueuedMessage struct {
	ID          string          `json:"id"`
	OperationID string          `json:"operationId"`
	Text        string          `json:"text"`
	Attachments []acpAttachment `json:"attachments"`
	CanvasNodes []acpCanvasNode `json:"canvasNodes,omitempty"`
	CreatedAt   int64           `json:"createdAt"`
	editID      string
}

type desktopSessionQueue struct {
	Messages []desktopQueuedMessage `json:"messages,omitempty"`
	Revision int                    `json:"revision,omitempty"`
	Paused   bool                   `json:"paused,omitempty"`
}

func (s *acpSession) advanceQueue() {
	s.Queue.Revision = max(s.Queue.Revision+1, s.historySequence())
}

func (s acpSession) queueSnapshot() map[string]any {
	messages := make([]map[string]any, 0, len(s.Queue.Messages))
	var editing []string
	for _, message := range s.Queue.Messages {
		if message.editID != "" {
			editing = append(editing, message.ID)
		}
		files := make([]map[string]any, 0, len(message.Attachments))
		for _, file := range message.Attachments {
			part := map[string]any{"name": file.Name, "path": queueAttachmentPath(file), "isImage": file.CanvasNodeID != ""}
			if file.URI == "" {
				part["fileRef"] = file.Ref
			}
			files = append(files, part)
		}
		messages = append(messages, map[string]any{"id": message.ID, "text": message.Text,
			"createdAt": message.CreatedAt, "attachments": files, "canvasNodes": message.CanvasNodes})
	}
	queue := map[string]any{"paused": s.Queue.Paused, "messages": messages}
	if len(editing) != 0 {
		queue["editingIds"] = editing
	}
	return map[string]any{"sessionId": s.ID, "eventSeq": max(s.Queue.Revision, s.historySequence()), "queue": queue}
}

func queueAttachmentPath(file acpAttachment) string {
	if file.URI != "" {
		return file.URI
	}
	return file.Ref.Path
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
	return s.queueSnapshot(), nil
}

func (h *relaySessionHost) handleQueue(ctx context.Context, ws *websocket.Conn, r *http.Request, raw json.RawMessage, accept func(any) error) (any, error) {
	var p struct {
		SessionID, Action, ID, Text, EditID string
		BeforeID                            *string
		KeepAttachments                     []string
		SendIfIdle                          bool
		Attachments                         []acpAttachment
		CanvasNodes                         []acpCanvasNode
		Meta                                struct{ OperationID string } `json:"_meta"`
	}
	if json.Unmarshal(raw, &p) != nil || !validUUID(p.SessionID) {
		return nil, errors.New("invalid_session_queue")
	}
	if p.Action == "get" {
		return h.readQueue(p.SessionID)
	}
	if h.live == nil {
		return nil, errors.New("session_queue_action_unavailable")
	}
	switch p.Action {
	case "add":
		if !validUUID(p.ID) || (strings.TrimSpace(p.Text) == "" && len(p.Attachments) == 0 && len(p.CanvasNodes) == 0) || len(p.Text) > 16<<10 || len(p.Attachments) > 4 ||
			len(p.CanvasNodes) > 20 || (len(p.CanvasNodes) != 0 && h.local == nil) {
			return nil, errors.New("session_queue_action_unavailable")
		}
		for _, node := range p.CanvasNodes {
			if node.ID == "" || len(node.ID) > 128 || len(node.Title) > 512 ||
				!slices.Contains([]string{"text", "image", "video", "audio", "html", "media-project", "group"}, node.NodeType) {
				return nil, errors.New("invalid_canvas_selection")
			}
		}
		if p.Meta.OperationID == "" {
			var err error
			p.Meta.OperationID, err = newUUID()
			if err != nil {
				return nil, err
			}
		}
		if !validUUID(p.Meta.OperationID) {
			return nil, errors.New("invalid_operationId")
		}
	case "update":
		if !validUUID(p.EditID) || len(p.Text) > 16<<10 || len(p.Attachments)+len(p.KeepAttachments) > 4 {
			return nil, errors.New("session_queue_action_unavailable")
		}
	case "beginEdit", "endEdit":
		if !validUUID(p.EditID) {
			return nil, errors.New("session_queue_action_unavailable")
		}
	case "resume", "remove", "move", "sendNow":
	default:
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
			p.Attachments[index] = acpAttachment{Name: row.Name, Ref: *row.Ref, CanvasNodeID: file.CanvasNodeID}
		}
		h.uploads.mu.Unlock()
	}
	live := h.live
	live.mu.Lock()
	row, ok := live.sessions[p.SessionID]
	if !ok || live.restarting {
		live.mu.Unlock()
		return nil, errors.New("session_unavailable_for_prompt")
	}
	result := func(row acpSession) map[string]any {
		value := row.queueSnapshot()
		value["clientMessageId"], value["action"] = p.ID, p.Action
		return value
	}
	messages := row.Queue.Messages
	index := slices.IndexFunc(messages, func(m desktopQueuedMessage) bool { return m.ID == p.ID })
	switch p.Action {
	case "add":
		if index >= 0 || slices.ContainsFunc(row.Turns, func(t acpTurn) bool { return t.ClientID == p.ID }) {
			live.mu.Unlock()
			return result(row), nil
		}
		if slices.ContainsFunc(messages, func(m desktopQueuedMessage) bool { return m.OperationID == p.Meta.OperationID }) ||
			slices.ContainsFunc(row.Turns, func(t acpTurn) bool { return t.Operation == p.Meta.OperationID }) {
			live.mu.Unlock()
			return nil, errors.New("session_operation_conflict")
		}
		if len(messages) >= 16 {
			live.mu.Unlock()
			return nil, errors.New("session_queue_full")
		}
		messages = append(slices.Clone(messages), desktopQueuedMessage{
			ID: p.ID, OperationID: p.Meta.OperationID, Text: p.Text, Attachments: p.Attachments, CanvasNodes: p.CanvasNodes,
			CreatedAt: time.Now().UnixMilli(),
		})
	case "resume":
		row.Queue.Paused = false
	case "beginEdit", "endEdit", "update":
		if index < 0 || p.Action != "beginEdit" && messages[index].editID != p.EditID {
			live.mu.Unlock()
			return nil, errors.New("session_queue_message_not_found")
		}
		messages = slices.Clone(messages)
		message := &messages[index]
		switch p.Action {
		case "beginEdit":
			message.editID = p.EditID
		case "endEdit":
			message.editID = ""
		case "update":
			files := slices.Clone(p.Attachments)
			for _, previous := range message.Attachments {
				if slices.Contains(p.KeepAttachments, queueAttachmentPath(previous)) {
					files = append(files, previous)
				}
			}
			if strings.TrimSpace(p.Text) == "" && len(files) == 0 && len(message.CanvasNodes) == 0 {
				live.mu.Unlock()
				return nil, errors.New("invalid_prompt")
			}
			message.Text, message.Attachments, message.editID = p.Text, files, ""
		}
	default:
		if index < 0 {
			live.mu.Unlock()
			return result(row), nil
		}
		message := messages[index]
		messages = slices.Delete(slices.Clone(messages), index, index+1)
		if p.Action == "sendNow" {
			messages = slices.Insert(messages, 0, message)
			row.Queue.Paused = false
		} else if p.Action == "move" {
			next := len(messages)
			if p.BeforeID != nil {
				next = slices.IndexFunc(messages, func(m desktopQueuedMessage) bool { return m.ID == *p.BeforeID })
				if next < 0 {
					live.mu.Unlock()
					return nil, errors.New("session_queue_message_not_found")
				}
			}
			messages = slices.Insert(messages, next, message)
		}
	}
	row.Queue.Messages = messages
	row.advanceQueue()
	if err := live.save(row); err != nil {
		live.mu.Unlock()
		return nil, errors.New("session_write_failed")
	}
	if p.Action == "sendNow" && row.GoalRun == "" {
		if cancel := live.activeTurns[row.ID]; cancel != nil {
			cancel()
		}
	}
	start := (p.Action != "add" || p.SendIfIdle) && live.claimQueue(row)
	live.mu.Unlock()
	if !start {
		return result(row), nil
	}

	accepted := func(current acpSession) error { return accept(result(current)) }
	return h.runSessionQueue(ctx, ws, r, row.ID, result(row), accepted)
}

// claimQueue is called with the live session lock held.
func (l *acpLive) claimQueue(row acpSession) bool {
	if l.restarting || l.queueWorkers[row.ID] != nil || row.State == "running" && row.GoalRun == "" || row.Queue.Paused || len(row.Queue.Messages) == 0 || row.Queue.Messages[0].editID != "" {
		return false
	}
	if l.queueWorkers == nil {
		l.queueWorkers = make(map[string]chan struct{})
	}
	l.queueWorkers[row.ID] = make(chan struct{}, 1)
	return true
}

func (h *relaySessionHost) runSessionQueue(ctx context.Context, ws *websocket.Conn, r *http.Request, id string, initial map[string]any, accepted func(acpSession) error) (any, error) {
	live := h.live
	for {
		live.mu.Lock()
		row := live.sessions[id]
		if ctx.Err() != nil || live.restarting || row.Queue.Paused || len(row.Queue.Messages) == 0 || row.Queue.Messages[0].editID != "" {
			delete(live.queueWorkers, id)
			live.mu.Unlock()
			return initial, nil
		}
		if row.GoalRun != "" || row.State == "running" {
			wake := live.queueWorkers[id]
			live.mu.Unlock()
			// The goal owns the session. Acknowledge the persisted message before waiting.
			if accepted != nil {
				err := accepted(row)
				accepted = nil
				if err != nil {
					live.mu.Lock()
					delete(live.queueWorkers, id)
					live.mu.Unlock()
					return nil, err
				}
			}
			select {
			case <-ctx.Done():
			case <-wake:
			}
			continue
		}
		message := row.Queue.Messages[0]
		live.mu.Unlock()

		onStart := func(current acpSession) error {
			if accepted != nil {
				err := accepted(current)
				accepted = nil
				return err
			}
			return notifySessionQueue(ctx, ws, current)
		}
		_, err := h.promptACP(ctx, ws, r, id, message.OperationID, message.ID, message.Text, onStart, &message, message.Attachments...)
		if errors.Is(err, errSessionQueueChanged) || errors.Is(err, errGoalSessionBusy) {
			continue
		}
		if err != nil {
			live.mu.Lock()
			row = live.sessions[id]
			if len(row.Queue.Messages) != 0 {
				row.Queue.Paused = true
				row.advanceQueue()
				if saveErr := live.save(row); saveErr != nil {
					err = errors.Join(err, saveErr)
				}
			}
			delete(live.queueWorkers, id)
			live.mu.Unlock()
			_ = notifySessionQueue(ctx, ws, row)
			return initial, err
		}
		current, err := live.load(id)
		if err == nil {
			_ = wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "method": "_monkeyai/session/history_changed",
				"params": map[string]any{"sessionId": id, "eventSeq": current.historySequence()}})
		}
	}
}

var errSessionQueueChanged = errors.New("session_queue_changed")

func notifySessionQueue(ctx context.Context, ws *websocket.Conn, row acpSession) error {
	params := row.queueSnapshot()
	params["state"], params["finish"] = map[string]any{}, false
	return wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "method": "_monkeyai/session/state", "params": params})
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
		for _, pending := range session.Queue.Messages {
			if pending.OperationID == operation {
				result := session.queueSnapshot()
				result["clientMessageId"], result["action"] = pending.ID, "add"
				return map[string]any{"version": 1, "operationId": operation, "target": target, "sessionId": id,
					"method": "_monkeyai/session/queue", "clientMessageId": pending.ID, "state": "completed", "result": result}, nil
			}
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
