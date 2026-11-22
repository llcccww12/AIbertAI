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

// dshRemoteTurnConnectorTools never starts imported local MCP processes.
func (h *relaySessionHost) dshRemoteTurnConnectorTools(ctx context.Context, ws *websocket.Conn, request *http.Request, session acpSession, turn *acpTurn) ([]dshHostToolBridge, error) {
	var bridges []dshHostToolBridge
	selected := session.ConnectorPreferences != nil && session.ConnectorPreferences.hasEnabledConnector()
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
		catalog, err := h.dshConnectorDiscovery(ctx, ws, request, session, turn, false)
		if err != nil {
			return nil, err
		}
		tools := make([]dshHostTool, 0, len(catalog.tools))
		for _, item := range catalog.definitions() {
			schema, err := dshStrictToolSchema(item.InputSchema)
			if err != nil {
				catalog.close()
				return nil, err
			}
			tool := catalog.tools[item.Name]
			tools = append(tools, dshHostTool{Name: item.Name, Description: item.Description, InputSchema: schema, Title: tool.ConnectorName + " / " + tool.Name, Kind: "other", Category: "connector"})
		}
		if len(tools) > 0 {
			bridges = append(bridges, dshHostToolBridge{Tools: tools, Handle: h.dshConnectorBridge(ctx, ws, request, session, turn, catalog.tools, catalog.local, catalog.reauthorize)})
		} else {
			catalog.close()
		}
	}
	return bridges, nil
}
