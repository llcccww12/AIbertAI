package main

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"sync"

	"github.com/coder/websocket"
)

type dshConnectorDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}
type dshConnectorCatalog struct {
	tools       map[string]connectorTool
	local       map[string]*desktopMCP
	reauthorize func(context.Context, connectorTool) error
	closeOnce   sync.Once
}

func (c *dshConnectorCatalog) definitions() []dshConnectorDefinition {
	out := make([]dshConnectorDefinition, 0, len(c.tools))
	for _, name := range slices.Sorted(maps.Keys(c.tools)) {
		t := c.tools[name]
		out = append(out, dshConnectorDefinition{Name: name, Description: t.Description, InputSchema: slices.Clone(t.Schema)})
	}
	return out
}
func (c *dshConnectorCatalog) close() {
	c.closeOnce.Do(func() {
		for _, server := range c.local {
			server.close()
		}
	})
}

// dshConnectorDiscovery owns all processes it starts. The caller must close
// the returned catalog on every completion, cancellation and setup failure.
func (h *relaySessionHost) dshConnectorDiscovery(ctx context.Context, ws *websocket.Conn, request *http.Request, session acpSession, turn *acpTurn, includeLocal bool) (_ *dshConnectorCatalog, err error) {
	if h.live == nil || h.live.store == nil || h.live.workspaces == nil || h.acpScope(request.WithContext(ctx), "resources:manage") != nil {
		return nil, errRelayAccess
	}
	current := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if h.acpScope(request.WithContext(ctx), "resources:manage") != nil {
			return errRelayAccess
		}
		row, err := h.live.load(session.ID)
		if err != nil {
			return err
		}
		if row.Engine != "dsh" || row.State != "running" || row.DSHSessionID != session.DSHSessionID || row.CWD != session.CWD || row.Workspace != session.Workspace || row.ProjectID != session.ProjectID || row.ExpertID != session.ExpertID || row.ResourceRevision != session.ResourceRevision || !reflect.DeepEqual(row.ConnectorPreferences, session.ConnectorPreferences) || len(row.Turns) == 0 || row.Turns[len(row.Turns)-1].Operation != turn.Operation {
			return errors.New("dsh_connector_session_changed")
		}
		if session.GoalRun != "" && (row.GoalRun != session.GoalRun || row.Goal == nil || session.Goal == nil || !row.Goal.Current || row.Goal.ID != session.Goal.ID || row.Goal.Status != "active") {
			return errors.New("dsh_connector_session_changed")
		}
		return nil
	}
	if err = current(); err != nil {
		return nil, err
	}
	catalog := &dshConnectorCatalog{tools: map[string]connectorTool{}, local: map[string]*desktopMCP{}}
	defer func() {
		if err != nil {
			catalog.close()
		}
	}()
	// Copy resources under lock to bind approvals to immutable commands.
	snapshot := func() (*importedDesktopResources, error) {
		h.importedResourcesMu.RLock()
		defer h.importedResourcesMu.RUnlock()
		if h.importedResources == nil {
			return nil, nil
		}
		data, e := json.Marshal(h.importedResources)
		if e != nil {
			return nil, e
		}
		var copy importedDesktopResources
		if e = json.Unmarshal(data, &copy); e != nil {
			return nil, e
		}
		return &copy, nil
	}
	imported, err := snapshot()
	if err != nil {
		return nil, err
	}
	prefs := connectorPreferences{}.normalized()
	if session.ConnectorPreferences != nil {
		prefs = session.ConnectorPreferences.normalized()
		prefs.Enabled = maps.Clone(prefs.Enabled)
	}
	cloud := func(ctx context.Context) (map[string]connectorTool, error) {
		if err := current(); err != nil {
			return nil, err
		}
		enabled := prefs
		enabled.Enabled = maps.Clone(prefs.Enabled)
		expert, err := h.resolveDesktopExpert(ctx, session.ExpertID)
		if err != nil {
			return nil, err
		}
		for _, link := range expert.Connectors {
			if validUUID(link.ID) && link.Required {
				enabled.Enabled["mcp/server/"+link.ID] = true
			}
		}
		if !enabled.hasEnabledConnector() {
			return nil, nil
		}
		auth, err := h.live.authorization(ctx)
		if err != nil {
			return nil, err
		}
		tools, err := h.live.client.activeConnectorTools(ctx, auth, enabled)
		if err != nil {
			return nil, err
		}
		for alias, tool := range tools {
			for _, link := range expert.Connectors {
				if tool.ConnectorID == link.ID && !link.allows(tool.Name) {
					delete(tools, alias)
				}
			}
			if imported != nil && !imported.Filters[tool.ConnectorID].allows(tool.Name) {
				delete(tools, alias)
			}
		}
		return tools, nil
	}
	cloudTools, err := cloud(ctx)
	if err != nil {
		return nil, err
	}
	maps.Copy(catalog.tools, cloudTools)
	if includeLocal && h.local != nil && imported != nil {
		profile := imported.Experts[session.ExpertID]
		for _, id := range slices.Sorted(maps.Keys(imported.Local)) {
			entry := imported.Local[id]
			enabled, set := prefs.Enabled["mcp/"+entry.Name]
			selected := enabled || !set && (entry.Enabled == nil || *entry.Enabled)
			if !selected && !slices.Contains(profile.LocalTools, id) {
				continue
			}
			executable, e := localMCPExecutable(entry.Command)
			if e != nil {
				return nil, e
			}
			callID := turn.Operation + "-local-mcp-" + id
			counts, e := h.requestACPToolApproval(ctx, ws, session.ID, callID, "启动本机连接器 "+entry.Name, entry.Name, "execute", map[string]any{"command": executable, "arguments": entry.Args})
			turn.Permissions.Asked += counts.Asked
			turn.Permissions.Allowed += counts.Allowed
			turn.Permissions.Denied += counts.Denied
			if e != nil {
				_ = sendACPToolUpdate(ctx, ws, session.ID, map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": callID, "status": "failed"})
				if counts.Denied > 0 {
					continue
				}
				return nil, e
			}
			if e = current(); e != nil {
				return nil, e
			}
			latest, e := snapshot()
			if e != nil || !reflect.DeepEqual(latest, imported) {
				_ = sendACPToolUpdate(ctx, ws, session.ID, map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": callID, "status": "failed"})
				return nil, errors.New("dsh_connector_configuration_changed")
			}
			server, e := openLocalMCP(ctx, entry, executable, session.CWD)
			if e != nil {
				_ = sendACPToolUpdate(ctx, ws, session.ID, map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": callID, "status": "failed"})
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				continue
			}
			catalog.local[id] = server
			discovered, e := server.tools(id, entry, prefs.Tools[entry.Name])
			if e != nil {
				_ = sendACPToolUpdate(ctx, ws, session.ID, map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": callID, "status": "failed"})
				server.close()
				delete(catalog.local, id)
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				continue
			}
			_ = sendACPToolUpdate(ctx, ws, session.ID, map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": callID, "status": "completed"})
			for alias, tool := range discovered {
				if _, exists := catalog.tools[alias]; exists {
					return nil, errors.New("dsh_connector_alias_conflict")
				}
				catalog.tools[alias] = tool
			}
		}
	}
	catalog.reauthorize = func(ctx context.Context, tool connectorTool) error {
		if err := current(); err != nil {
			return err
		}
		latest, err := snapshot()
		if err != nil {
			return err
		}
		if tool.Local {
			if imported == nil || latest == nil || !reflect.DeepEqual(latest.Local[tool.ConnectorID], imported.Local[tool.ConnectorID]) || !reflect.DeepEqual(latest.Experts[session.ExpertID], imported.Experts[session.ExpertID]) {
				return errors.New("dsh_connector_configuration_changed")
			}
			server := catalog.local[tool.ConnectorID]
			if server == nil {
				return errors.New("local_mcp_unavailable")
			}
			entry := latest.Local[tool.ConnectorID]
			available, err := server.tools(tool.ConnectorID, entry, prefs.Tools[entry.Name])
			if err != nil {
				return err
			}
			if got, ok := available[tool.alias()]; !ok || !reflect.DeepEqual(got, tool) {
				return errors.New("dsh_connector_configuration_changed")
			}
			return current()
		}
		if !reflect.DeepEqual(latest, imported) {
			return errors.New("dsh_connector_configuration_changed")
		}
		available, err := cloud(ctx)
		if err != nil {
			return err
		}
		if got, ok := available[tool.alias()]; !ok || !reflect.DeepEqual(got, tool) {
			return errors.New("dsh_connector_configuration_changed")
		}
		return current()
	}
	context.AfterFunc(ctx, catalog.close)
	return catalog, nil
}
