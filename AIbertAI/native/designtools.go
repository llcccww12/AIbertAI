package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"time"
)

const designReadTool = "canvas_get_document"
const designSelectionTool = "canvas_get_selection"
const designTextTool = "canvas_create_text"
const designCreateHTMLTool = "canvas_create_html"
const designCommitHTMLTool = "canvas_commit_html"

const designProductionTool = "canvas_production"
const designPreparePPTXTool = "canvas_prepare_pptx"
const designStyleChoiceTool = "design_await_style_choice"

func designTool(name string) bool {
	switch name {
	case designReadTool, designSelectionTool, designTextTool, designCreateHTMLTool, designCommitHTMLTool,
		designProductionTool, designPreparePPTXTool, "media_get_image_model", "media_generate_image", "media_edit_image", designStyleChoiceTool, "canvas_add_artifact_asset",
		"canvas_capture_preview", "design_jury_submit", "canvas_create_image", "canvas_create_video",
		"canvas_compose_media_project", "canvas_create_layer_group", "canvas_group_story_assets", "canvas_update_node",
		"canvas_delete_node", "design_catalog_search", "design_catalog_detail", "design_catalog_logo_batch":
		return true
	}
	return false
}

func designWriteTool(name string) bool {
	switch name {
	case designReadTool, designSelectionTool, "media_get_image_model", designStyleChoiceTool, "design_catalog_search", "design_catalog_detail":
		return false
	}
	return designTool(name)
}

type desktopDesignTool struct {
	Name, Description string
	Schema            json.RawMessage `json:"inputSchema"`
}

func (b *desktopDesignBridge) tools(ctx context.Context, row acpSession, design json.RawMessage) ([]desktopDesignTool, error) {
	out, err := b.call(ctx, row, design, "design_tool_catalog", json.RawMessage(`{}`))
	var tools []desktopDesignTool
	if err != nil || json.Unmarshal([]byte(out), &tools) != nil || len(tools) == 0 || len(tools) > 32 {
		return nil, errors.New("design_tool_catalog_unavailable")
	}
	return tools, nil
}

func (b *desktopDesignBridge) ensureCanvas(ctx context.Context, row acpSession) (json.RawMessage, error) {
	args := json.RawMessage(`{"templateRef":null,"designSystemRef":null,"mediaModelId":null}`)
	result, err := b.call(ctx, row, nil, "design_session_open", args)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(result), nil
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
	return b.callGuarded(ctx, row, design, name, args, nil, "", "", "", "")
}

func (b *desktopDesignBridge) callGuarded(ctx context.Context, row acpSession, design json.RawMessage, name string, args json.RawMessage, revision *int64, digest, artifacts, contextDigest, sourceDigest string) (string, error) {
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
	event := map[string]any{"event": "design_tool", "id": id, "sessionId": row.ID, "cwd": row.CWD, "design": design, "name": name, "arguments": args}
	if revision != nil {
		event["expectedRevision"], event["expectedCanvasDigest"] = *revision, digest
		if sourceDigest != "" {
			event["expectedSourceDigest"] = sourceDigest
		}
		if artifacts != "" {
			event["expectedArtifactsDigest"], event["expectedContextDigest"] = artifacts, contextDigest
		}
	}
	if err := b.send(event); err != nil {
		return "", err
	}
	timeout := 20 * time.Second
	if name == "media_generate_image" || name == "media_edit_image" {
		timeout = 21 * time.Minute
	} else if name == "canvas_capture_preview" || name == designStyleChoiceTool || name == "design_session_open" {
		timeout = 125 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	select {
	case reply := <-ch:
		if reply.Error != "" || len(reply.Result) == 0 {
			if strings.HasPrefix(reply.Error, "design_") {
				return "", errors.New(reply.Error)
			}
			return "", errors.New("design_tool_failed")
		}
		return string(reply.Result), nil
	case <-ctx.Done():
		if name == "media_generate_image" || name == "media_edit_image" || name == designStyleChoiceTool {
			_ = b.send(map[string]string{"event": "design_tool_cancel", "id": id})
		}
		return "", errors.New("design_tool_unavailable")
	}
}

func validDesignToolArguments(name string, raw json.RawMessage) bool {
	raw = bytes.TrimSpace(raw)
	var object map[string]json.RawMessage
	return designTool(name) && len(raw) <= 48<<10 && json.Unmarshal(raw, &object) == nil && object != nil
}
