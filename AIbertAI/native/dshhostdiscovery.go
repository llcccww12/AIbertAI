package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"

	"github.com/coder/websocket"
)

func (h *relaySessionHost) dshSelectedKnowledge(session acpSession) []string {
	if h.live == nil {
		return nil
	}
	h.live.mu.Lock()
	prefs := h.live.connectorPrefs
	h.live.mu.Unlock()
	if session.ConnectorPreferences != nil {
		prefs = *session.ConnectorPreferences
	}
	return slices.Clone(prefs.KnowledgeBaseIDs)
}

func (h *relaySessionHost) dshExpertCanvasTools(ctx context.Context, expertID string) []string {
	if expert, ok := ctx.Value(dshExpertManifestKey{}).(desktopExpert); ok && expert.ID == expertID && expert.CanvasTools != nil {
		return expert.CanvasTools
	}
	h.importedResourcesMu.RLock()
	defer h.importedResourcesMu.RUnlock()
	if h.importedResources == nil {
		return nil
	}
	return slices.Clone(h.importedResources.Experts[expertID].DesignTools)
}

func dshExpertAllowsCanvas(expertID string, configured []string, selected bool, name string) bool {
	return expertID == "" || expertID == designExpertID || slices.Contains(configured, name) || selected && name == designSelectionTool
}

var errDSHCanvasUnavailable = errors.New("dsh_canvas_unavailable")

// dshTurnHostTools exposes only account/session-authorized tools. Canvas writes
// are restricted to those with verified guards and durable receipt handling.
func (h *relaySessionHost) dshTurnHostTools(ctx context.Context, ws *websocket.Conn, request *http.Request, session acpSession, turn *acpTurn) ([]dshHostToolBridge, string, error) {
	selected := h.dshSelectedKnowledge(session)
	requiresCanvas := session.ExpertID == designExpertID || len(turn.CanvasNodes) > 0
	if h.live == nil || request == nil {
		if len(selected) > 0 {
			return nil, "", errors.New("dsh_knowledge_unavailable")
		}
		if requiresCanvas {
			return nil, "", errDSHCanvasUnavailable
		}
		return nil, "", nil
	}
	if h.local == nil {
		if len(selected) > 0 {
			return nil, "", errors.New("dsh_knowledge_unavailable")
		}
		if requiresCanvas {
			return nil, "", errDSHCanvasUnavailable
		}
	}
	var bridges []dshHostToolBridge
	hasSelection := false
	var instructions string
	configured := h.dshExpertCanvasTools(ctx, session.ExpertID)
	defaultCanvas := session.ExpertID == "" || session.ExpertID == designExpertID
	if (defaultCanvas || len(configured) > 0 || len(turn.CanvasNodes) > 0) && h.local != nil && h.live.workspaces != nil && h.designBridge != nil && h.acpScope(request.WithContext(ctx), "files:read") == nil {
		root, err := h.live.workspaces.sessionRoot(session)
		if err != nil {
			return nil, "", err
		}
		design := workspaceDesign(root, session.ID)
		canWrite := h.acpScope(request.WithContext(ctx), "files:write") == nil
		if len(design) == 0 && len(configured) > 0 && len(turn.CanvasNodes) == 0 && canWrite {
			_, err = h.designBridge.ensureCanvas(ctx, session)
			if err == nil {
				design = workspaceDesign(root, session.ID)
			}
		}
		root.Close()
		if err != nil {
			return nil, "", err
		}
		if len(design) > 0 {
			catalog, err := h.designBridge.tools(ctx, session, design)
			if err != nil {
				return nil, "", err
			}
			capabilities, _ := h.designBridge.call(ctx, session, design, "design_guard_capabilities", json.RawMessage(`{}`))
			catalog = dshCanvasRestrictedCatalog(catalog, canWrite, capabilities)
			catalog = slices.DeleteFunc(catalog, func(item desktopDesignTool) bool {
				return !dshExpertAllowsCanvas(session.ExpertID, configured, len(turn.CanvasNodes) != 0, item.Name)
			})
			tools := make([]dshHostTool, 0, len(catalog))
			for _, item := range catalog {
				tools = append(tools, dshHostTool{Name: item.Name, Description: item.Description, InputSchema: item.Schema, Title: item.Name, Kind: workspaceToolKind(item.Name), Category: "canvas"})
				hasSelection = hasSelection || item.Name == designSelectionTool
			}
			if len(tools) > 0 {
				if defaultCanvas {
					instructions = designExpertInstructions(design)
				}
				invoke := h.dshCanvasBridge(ctx, ws, request, session, turn, design, catalog)
				// These two advertised schemas accept only an empty object. The
				// bound user selection is added by the host after this check.
				handle := func(raw json.RawMessage) (any, error) {
					var call dshMutation
					if json.Unmarshal(raw, &call) != nil {
						return nil, errors.New("dsh_canvas_request_invalid")
					}
					args, err := strictDSHJSON(call.Arguments)
					if err != nil || (call.Name == designReadTool || call.Name == designSelectionTool) && string(args) != "{}" {
						return nil, errors.New("dsh_canvas_request_invalid")
					}
					return invoke(raw)
				}
				bridges = append(bridges, dshHostToolBridge{Tools: tools, Handle: handle})
			}
		} else if !requiresCanvas {
			// Without a canvas, ordinary conversations can still generate
			// images into the workspace. Best effort: an unavailable design
			// worker only leaves these tools out of this turn.
			if bridge, hint := h.dshWorkspaceImageTools(ctx, ws, request, session, turn, configured, canWrite); bridge != nil {
				bridges = append(bridges, *bridge)
				if session.ExpertID == "" {
					instructions = hint
				}
			}
		}
	}
	if (requiresCanvas && !hasSelection) || (session.ExpertID == designExpertID && instructions == "") {
		return nil, "", errDSHCanvasUnavailable
	}
	if len(selected) > 0 {
		if h.acpScope(request.WithContext(ctx), "resources:manage") != nil {
			return nil, "", errRelayAccess
		}
		catalog := dshKnowledgeCatalog(selected)
		if len(catalog) == 0 {
			return nil, "", errors.New("dsh_knowledge_selection_invalid")
		}
		tools := make([]dshHostTool, 0, len(catalog))
		for _, item := range catalog {
			schema, err := dshStrictToolSchema(item.Schema)
			if err != nil {
				return nil, "", err
			}
			tools = append(tools, dshHostTool{Name: item.Name, Description: item.Description, InputSchema: schema, Title: "检索已选知识库", Kind: "read", Category: "knowledge"})
		}
		bridges = append(bridges, dshHostToolBridge{Tools: tools, Handle: h.dshKnowledgeBridge(ctx, ws, request, session, turn, selected)})
	}
	remote, err := h.dshRemoteTurnConnectorTools(ctx, ws, request, session, turn)
	if err != nil {
		return nil, "", err
	}
	bridges = append(bridges, remote...)
	return bridges, instructions, nil
}

// dshLocalMCPSelected reports whether discovery would start a local MCP
// server for this session, using the same selection as dshConnectorDiscovery.
func (h *relaySessionHost) dshLocalMCPSelected(session acpSession) bool {
	if h.local == nil {
		return false
	}
	prefs := connectorPreferences{}.normalized()
	if session.ConnectorPreferences != nil {
		prefs = session.ConnectorPreferences.normalized()
	}
	h.importedResourcesMu.RLock()
	defer h.importedResourcesMu.RUnlock()
	if h.importedResources == nil {
		return false
	}
	profile := h.importedResources.Experts[session.ExpertID]
	for id, entry := range h.importedResources.Local {
		enabled, set := prefs.Enabled["mcp/"+entry.Name]
		if enabled || !set && (entry.Enabled == nil || *entry.Enabled) || slices.Contains(profile.LocalTools, id) {
			return true
		}
	}
	return false
}

// dshRemoteTurnConnectorTools starts local MCP servers only on this device,
// each after its own startup approval, as the ACP engine does.
func (h *relaySessionHost) dshRemoteTurnConnectorTools(ctx context.Context, ws *websocket.Conn, request *http.Request, session acpSession, turn *acpTurn) ([]dshHostToolBridge, error) {
	var bridges []dshHostToolBridge
	selected := session.ConnectorPreferences != nil && session.ConnectorPreferences.hasEnabledConnector() || h.dshLocalMCPSelected(session)
	if !selected && session.ExpertID != "" && session.ExpertID != designExpertID {
		expert, ok := ctx.Value(dshExpertManifestKey{}).(desktopExpert)
		if !ok || expert.ID != session.ExpertID {
			var err error
			expert, err = h.resolveDesktopExpert(ctx, session.ExpertID)
			if err != nil {
				return nil, err
			}
		}
		for _, link := range expert.Connectors {
			if link.Required && validUUID(link.ID) {
				selected = true
				break
			}
		}
	}
	if selected {
		catalog, err := h.dshConnectorDiscovery(ctx, ws, request, session, turn, h.local != nil)
		if err != nil {
			return nil, err
		}
		tools := make([]dshHostTool, 0, len(catalog.tools))
		for _, item := range catalog.definitions() {
			tool := catalog.tools[item.Name]
			schema, err := dshStrictToolSchema(item.InputSchema)
			if err != nil && tool.Local {
				// A third-party local server's unsupported schema drops only that tool.
				delete(catalog.tools, item.Name)
				continue
			}
			if err != nil {
				catalog.close()
				return nil, err
			}
			turn.useResource("connector", tool.ConnectorID)
			tools = append(tools, dshHostTool{Name: item.Name, Description: item.Description, InputSchema: schema, Title: tool.ConnectorName + " / " + tool.Name, Kind: "other", Category: "connector", ResourceID: tool.ConnectorID})
		}
		if len(tools) > 0 {
			bridges = append(bridges, dshHostToolBridge{Tools: tools, Handle: h.dshConnectorBridge(ctx, ws, request, session, turn, catalog.tools, catalog.local, catalog.reauthorize)})
		} else {
			catalog.close()
		}
	}
	return bridges, nil
}

const dshWorkspaceImageInstructions = "Image generation is available in this conversation. When the user asks for a picture, illustration, poster or photo, call media_get_image_model to check the selected model and its parameters, then media_generate_image with a concrete prompt. The result gives a workspace-relative source_path under .aibertai/media/generated/, a folder the file panel hides: copy the image to a visible, descriptively named path in the workspace (for example with cp) and tell the user that path; reference the visible path from any document or web page you write instead of embedding base64. If the tool reports that no image model is available or the generation failed, do not retry it in this turn; tell the user plainly and offer an alternative such as an SVG illustration."

// dshWorkspaceImageTools exposes only image generation when the session has no
// canvas. The worker runs these tools without a design marker; the canvas
// bridge dispatches them unguarded under the host's durable claim.
func (h *relaySessionHost) dshWorkspaceImageTools(ctx context.Context, ws *websocket.Conn, request *http.Request, session acpSession, turn *acpTurn, configured []string, canWrite bool) (*dshHostToolBridge, string) {
	catalog, err := h.designBridge.tools(ctx, session, nil)
	if err != nil {
		return nil, ""
	}
	catalog = slices.DeleteFunc(catalog, func(item desktopDesignTool) bool {
		return item.Name != "media_get_image_model" && item.Name != "media_generate_image" || !dshExpertAllowsCanvas(session.ExpertID, configured, false, item.Name)
	})
	capabilities, _ := h.designBridge.call(ctx, session, nil, "design_guard_capabilities", json.RawMessage(`{}`))
	catalog = dshCanvasRestrictedCatalog(catalog, canWrite, capabilities)
	if !slices.ContainsFunc(catalog, func(item desktopDesignTool) bool { return item.Name == "media_generate_image" }) {
		return nil, ""
	}
	tools := make([]dshHostTool, 0, len(catalog))
	for _, item := range catalog {
		tools = append(tools, dshHostTool{Name: item.Name, Description: item.Description, InputSchema: item.Schema, Title: item.Name, Kind: workspaceToolKind(item.Name), Category: "canvas"})
	}
	return &dshHostToolBridge{Tools: tools, Handle: h.dshCanvasBridge(ctx, ws, request, session, turn, nil, catalog)}, dshWorkspaceImageInstructions
}
