package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/coder/websocket"
)

type dshCanvasBridgeKey struct{}

// Catalog queries and waiting for an explicit style decision do not mutate the canvas.
// Canvas reads and writes require the worker's guarded protocol.
func dshCanvasReadOnly(name string) bool {
	switch name {
	case "media_get_image_model", "design_catalog_search", "design_catalog_detail", designStyleChoiceTool:
		return true
	default:
		return false
	}
}

func dshCanvasGuarded(name string) bool {
	switch name {
	case designReadTool, designSelectionTool, designTextTool, designCreateHTMLTool, designCommitHTMLTool, designPreparePPTXTool, "canvas_create_image", "media_generate_image":
		return true
	default:
		return false
	}
}

func dshCanvasHasGuard(raw string, name string) bool {
	var guard struct {
		Version int
		Tools   []string
	}
	return json.Unmarshal([]byte(raw), &guard) == nil && guard.Version == 4 && slices.Contains(guard.Tools, name)
}

// dshCanvasArtifacts hashes HTML contents independently of the canvas revision.
func dshCanvasArtifacts(root *os.Root, canvasID string, canvas []byte) (string, error) {
	var doc struct{ Nodes []struct{ ID, Type string } }
	if json.Unmarshal(canvas, &doc) != nil || len(doc.Nodes) > 10000 {
		return "", errors.New("dsh_canvas_changed")
	}
	entries := make([]string, 0)
	for _, node := range doc.Nodes {
		if node.Type != "html" {
			continue
		}
		if !validUUID(node.ID) || len(entries) >= 512 {
			return "", errors.New("dsh_canvas_changed")
		}
		base := filepath.ToSlash(filepath.Join(workspaceDataDir, "design", canvasID, "artifacts", node.ID))
		indexFound := false
		var total int64
		err := fs.WalkDir(root.FS(), base, func(name string, item fs.DirEntry, walkErr error) error {
			if walkErr != nil || item.Type()&os.ModeSymlink != 0 {
				return errors.New("unsafe")
			}
			if item.IsDir() {
				return nil
			}
			if !item.Type().IsRegular() || len(entries) >= 4096 {
				return errors.New("unsafe")
			}
			state, err := dshWorkspaceTarget(root, name)
			if err != nil || state.File == nil || state.File.Size() > 8<<20 {
				return errors.New("unsafe")
			}
			total += state.File.Size()
			if total > 128<<20 {
				return errors.New("unsafe")
			}
			file, err := openAttachmentFile(root, name, 8<<20)
			if err != nil {
				return err
			}
			data, readErr := io.ReadAll(io.LimitReader(file, (8<<20)+1))
			if err = errors.Join(readErr, file.Close()); err != nil || len(data) > 8<<20 {
				return errors.New("unsafe")
			}
			relative := strings.TrimPrefix(name, base+"/")
			if relative == "index.html" {
				indexFound = true
			}
			entries = append(entries, node.ID+"/"+relative+":"+fmt.Sprintf("%x", sha256.Sum256(data)))
			return nil
		})
		if err != nil {
			return "", errors.New("dsh_canvas_changed")
		}
		if !indexFound {
			entries = append(entries, node.ID+"/index.html:missing")
		}
	}
	slices.Sort(entries)
	hasher := sha256.New()
	for _, entry := range entries {
		_, _ = io.WriteString(hasher, entry)
		_, _ = hasher.Write([]byte{0})
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// dshCanvasBridge dispatches only calls projected from the active SDK turn.
// catalog must be the same restricted catalog exposed to that SDK session.
func (h *relaySessionHost) dshCanvasBridge(ctx context.Context, ws *websocket.Conn, request *http.Request, session acpSession, turn *acpTurn, design json.RawMessage, catalog []desktopDesignTool) func(json.RawMessage) (any, error) {
	allowed := make(map[string]bool, len(catalog))
	for _, tool := range catalog {
		var schema struct{ Type string }
		if designTool(tool.Name) && json.Unmarshal(tool.Schema, &schema) == nil && schema.Type == "object" {
			allowed[tool.Name] = true
		}
	}
	design = slices.Clone(design)
	selection := slices.Clone(turn.CanvasNodes)
	var mu sync.Mutex
	seen := map[string]bool{}
	return func(raw json.RawMessage) (any, error) {
		// Serialize canvas operations and protect against duplicate host requests.
		mu.Lock()
		defer mu.Unlock()
		var req dshMutation
		if json.Unmarshal(raw, &req) != nil || req.SessionID != session.DSHSessionID || req.CallID == "" || seen[req.CallID] || !allowed[req.Name] || !validDesignToolArguments(req.Name, req.Arguments) {
			return nil, errors.New("dsh_canvas_request_invalid")
		}
		guarded := dshCanvasGuarded(req.Name)
		if !dshCanvasReadOnly(req.Name) && !guarded {
			return nil, errors.New("dsh_canvas_atomic_guard_unavailable")
		}
		seen[req.CallID] = true
		var run *workspaceToolRun
		for i := range turn.ToolRuns {
			if turn.ToolRuns[i].DSHCallID == req.CallID {
				run = &turn.ToolRuns[i]
				break
			}
		}
		if run == nil || !run.Pending || run.Call == nil || run.Call.ID == "" || run.Call.Function.Name != req.Name {
			return nil, errors.New("dsh_canvas_request_invalid")
		}
		callID := run.Call.ID
		var original, requested any
		if json.Unmarshal([]byte(run.Call.Function.Arguments), &original) != nil || json.Unmarshal(req.Arguments, &requested) != nil {
			return nil, errors.New("dsh_canvas_request_invalid")
		}
		a, _ := json.Marshal(original)
		b, _ := json.Marshal(requested)
		if !bytes.Equal(a, b) {
			return nil, errors.New("dsh_canvas_request_invalid")
		}
		args := req.Arguments
		var imageSource, sourceDigest string
		if req.Name == "canvas_create_image" {
			var err error
			imageSource, err = dshCanvasImageSource(args)
			if err != nil {
				return nil, err
			}
		}
		if req.Name == "media_generate_image" {
			var image struct {
				Prompt     string
				Parameters json.RawMessage
			}
			decoder := json.NewDecoder(bytes.NewReader(args))
			decoder.DisallowUnknownFields()
			if decoder.Decode(&image) != nil || strings.TrimSpace(image.Prompt) == "" {
				return nil, errors.New("dsh_canvas_request_invalid")
			}
		}
		if req.Name == designSelectionTool && len(selection) != 0 {
			ids := make([]string, 0, len(selection))
			for _, node := range selection {
				ids = append(ids, node.ID)
			}
			args, _ = json.Marshal(map[string][]string{"node_ids": ids})
		}
		var revision int64
		var artifacts, contextDigest string
		check := func() ([32]byte, error) {
			var empty [32]byte
			matched := false
			for i := range turn.ToolRuns {
				item := &turn.ToolRuns[i]
				if item.DSHCallID != req.CallID {
					continue
				}
				var value any
				if item.Pending && item.Call != nil && item.Call.ID == callID && item.Call.Function.Name == req.Name && json.Unmarshal([]byte(item.Call.Function.Arguments), &value) == nil {
					encoded, _ := json.Marshal(value)
					matched = bytes.Equal(encoded, b)
				}
				break
			}
			if !matched {
				return empty, errors.New("dsh_canvas_request_invalid")
			}
			if h.local == nil || h.designBridge == nil || h.live == nil || h.live.workspaces == nil || request == nil || h.acpScope(request.WithContext(ctx), "files:read") != nil || designWriteTool(req.Name) && h.acpScope(request.WithContext(ctx), "files:write") != nil {
				return empty, errRelayAccess
			}
			current, err := h.live.load(session.ID)
			if err != nil || current.Engine != "dsh" || current.DSHSessionID != session.DSHSessionID || current.CWD != session.CWD || current.Workspace != session.Workspace || current.ProjectID != session.ProjectID || current.ExpertID != session.ExpertID || current.ResourceRevision != session.ResourceRevision || !dshExpertAllowsCanvas(current.ExpertID, h.dshExpertCanvasTools(ctx, current.ExpertID), len(selection) != 0, req.Name) || len(current.Turns) == 0 || current.Turns[len(current.Turns)-1].Operation != turn.Operation || !slices.Equal(current.Turns[len(current.Turns)-1].CanvasNodes, selection) || !slices.Equal(turn.CanvasNodes, selection) {
				return empty, errors.New("dsh_canvas_session_changed")
			}
			if req.Name == "media_generate_image" && current.MediaImageModelID != session.MediaImageModelID {
				return empty, errors.New("dsh_canvas_session_changed")
			}
			root, err := h.live.workspaces.sessionRoot(session)
			if err != nil {
				return empty, err
			}
			defer root.Close()
			if imageSource != "" {
				if h.live.store == nil {
					return empty, errDSHCanvasImageSource
				}
				sourceDigest, err = dshGeneratedImageDigest(h.live.store.Root, root, current, imageSource)
				if err != nil {
					return empty, err
				}
			}
			currentDesign := workspaceDesign(root, session.ID)
			if !bytes.Equal(currentDesign, design) {
				return empty, errors.New("dsh_canvas_changed")
			}
			if len(currentDesign) == 0 {
				if guarded {
					return empty, errors.New("dsh_canvas_changed")
				}
				return sha256.Sum256(nil), nil
			}
			var marker struct{ CanvasID string }
			if json.Unmarshal(currentDesign, &marker) != nil {
				return empty, errors.New("dsh_canvas_changed")
			}
			canvas := readDesignJSON(root, filepath.Join(workspaceDataDir, "design", marker.CanvasID, "canvas.json"), maxDesktopSessionBytes)
			if canvas == nil {
				return empty, errors.New("dsh_canvas_changed")
			}
			var document struct {
				Revision *int64 `json:"revision"`
			}
			if json.Unmarshal(canvas, &document) != nil || document.Revision != nil && (*document.Revision < 0 || *document.Revision > 9007199254740991) {
				return empty, errors.New("dsh_canvas_changed")
			}
			revision = 0
			if document.Revision != nil {
				revision = *document.Revision
			}
			if guarded && req.Name != designTextTool {
				contextFile := readDesignJSON(root, filepath.Join(workspaceDataDir, "design", marker.CanvasID, "design-context.json"), maxDesktopSessionBytes)
				if contextFile == nil {
					return empty, errors.New("dsh_canvas_changed")
				}
				contextDigest = fmt.Sprintf("%x", sha256.Sum256(contextFile))
				artifacts, err = dshCanvasArtifacts(root, marker.CanvasID, canvas)
				if err != nil {
					return empty, err
				}
			}
			return sha256.Sum256(canvas), nil
		}
		before, err := check()
		if err != nil {
			return nil, err
		}
		if req.Name == designStyleChoiceTool {
			// The viewer's existing style picker records the decision. The SDK
			// only waits for it and supplies no input that can change the canvas.
			if string(b) != "{}" || len(design) == 0 {
				return nil, errors.New("dsh_canvas_request_invalid")
			}
			result, err := h.designBridge.call(ctx, session, design, req.Name, args)
			if err != nil {
				return nil, err
			}
			return map[string]string{"output": result}, nil
		}
		beforeArtifacts, beforeContext := artifacts, contextDigest
		if guarded {
			capabilities, err := h.designBridge.call(ctx, session, design, "design_guard_capabilities", json.RawMessage(`{}`))
			if err != nil || !dshCanvasHasGuard(capabilities, req.Name) {
				return nil, errors.New("dsh_canvas_atomic_guard_unavailable")
			}
		}
		if req.Name != designReadTool {
			approvalCtx := context.WithValue(ctx, dshNativeApprovalWaitKey{}, true)
			counts, err := h.requestDSHToolApproval(approvalCtx, ws, session.ID, callID, req.Name, req.Name, workspaceToolKind(req.Name), args, true)
			turn.Permissions.Asked += counts.Asked
			turn.Permissions.Allowed += counts.Allowed
			turn.Permissions.Denied += counts.Denied
			if err != nil {
				return nil, err
			}
			after, err := check()
			if err != nil {
				return nil, err
			}
			if before != after || beforeArtifacts != artifacts || beforeContext != contextDigest {
				return nil, errors.New("dsh_canvas_changed")
			}
		}
		var effect *dshSideEffect
		if designWriteTool(req.Name) {
			effect, err = h.beginDSHSideEffect(ctx, session.ID, turn.Operation, req.CallID, func(current acpSession) bool {
				return current.DSHSessionID == session.DSHSessionID && current.CWD == session.CWD && current.Workspace == session.Workspace && current.ProjectID == session.ProjectID && (req.Name != "media_generate_image" || current.MediaImageModelID == session.MediaImageModelID) && slices.Equal(current.Turns[len(current.Turns)-1].CanvasNodes, selection)
			})
			if err != nil {
				if ctx.Err() != nil && strings.Contains(err.Error(), "dsh_connector_outcome_unknown") {
					return nil, errors.Join(errors.New("dsh_canvas_outcome_unknown"), err)
				}
				return nil, err
			}
			if err = dshCanvasMarkClaim(effect, fmt.Sprintf("%x", before)); err != nil {
				return nil, errors.Join(errors.New("dsh_canvas_outcome_unknown"), err)
			}
		}
		if effect != nil && ctx.Err() != nil {
			return nil, errors.Join(errors.New("dsh_canvas_outcome_unknown"), errors.New("dsh_connector_outcome_unknown"), ctx.Err())
		}
		var result string
		if guarded {
			result, err = h.designBridge.callGuarded(ctx, session, design, req.Name, args, &revision, fmt.Sprintf("%x", before), beforeArtifacts, beforeContext, sourceDigest)
		} else {
			result, err = h.designBridge.call(ctx, session, design, req.Name, args)
		}
		if err != nil {
			if effect != nil {
				return nil, errors.Join(errors.New("dsh_canvas_outcome_unknown"), errors.New("dsh_connector_outcome_unknown"), err)
			}
			return nil, err
		}
		if effect != nil {
			if err = dshCanvasRecordResult(effect, result); err != nil {
				return nil, errors.Join(errors.New("dsh_canvas_outcome_unknown"), err)
			}
		}
		return map[string]string{"output": result}, nil
	}
}
