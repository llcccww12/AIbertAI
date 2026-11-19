package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"
)

const designReadTool = "canvas_get_document"
const designTextTool = "canvas_create_text"
const designCreateHTMLTool = "canvas_create_html"
const designCommitHTMLTool = "canvas_commit_html"

func designWriteTool(name string) bool {
	return name == designTextTool || name == designCreateHTMLTool || name == designCommitHTMLTool
}

type desktopDesignBridge struct {
	send    func(any) error
	mu      sync.Mutex
	pending map[string]chan designToolResult
}

type designToolResult struct {
	Type, ID, Error string
	Result          json.RawMessage
}

func (b *desktopDesignBridge) read(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 64<<10)
	for scanner.Scan() {
		var reply designToolResult
		if json.Unmarshal(scanner.Bytes(), &reply) != nil || reply.Type != "design_tool_result" || len(reply.Result) > 48<<10 {
			continue
		}
		b.mu.Lock()
		ch := b.pending[reply.ID]
		delete(b.pending, reply.ID)
		b.mu.Unlock()
		if ch != nil {
			ch <- reply
		}
	}
}

func (b *desktopDesignBridge) call(ctx context.Context, row acpSession, design json.RawMessage, name string, args json.RawMessage) (string, error) {
	id, err := newUUID()
	if err != nil {
		return "", err
	}
	ch := make(chan designToolResult, 1)
	b.mu.Lock()
	b.pending[id] = ch
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(b.pending, id)
		b.mu.Unlock()
	}()
	if err := b.send(map[string]any{"event": "design_tool", "id": id, "sessionId": row.ID, "cwd": row.CWD, "design": design, "name": name, "arguments": args}); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	select {
	case reply := <-ch:
		if reply.Error != "" || len(reply.Result) == 0 {
			return "", errors.New("design_tool_failed")
		}
		return string(reply.Result), nil
	case <-ctx.Done():
		return "", errors.New("design_tool_unavailable")
	}
}

func validDesignToolArguments(name string, raw json.RawMessage) bool {
	raw = bytes.TrimSpace(raw)
	if len(raw) < 2 || len(raw) > 12<<10 || raw[0] != '{' {
		return false
	}
	switch name {
	case designReadTool:
		return decodeWorkspaceTool(raw, &struct{}{}) == nil
	case designTextTool:
		var p struct {
			Title   string
			Content string
		}
		return decodeWorkspaceTool(raw, &p) == nil && len(p.Title) <= 160 && len(p.Content) > 0 && len(p.Content) <= 8000
	case designCreateHTMLTool:
		var p struct {
			Title         string
			Width, Height *int
			SlideIndex    *int `json:"slide_index"`
		}
		return decodeWorkspaceTool(raw, &p) == nil && len(p.Title) <= 160 &&
			(p.Width == nil || *p.Width > 0 && *p.Width <= 16384) &&
			(p.Height == nil || *p.Height > 0 && *p.Height <= 16384) &&
			(p.SlideIndex == nil || *p.SlideIndex > 0 && *p.SlideIndex <= 512)
	case designCommitHTMLTool:
		var p struct {
			NodeID string `json:"node_id"`
		}
		return decodeWorkspaceTool(raw, &p) == nil && validUUID(p.NodeID)
	default:
		return false
	}
}
