package main

import (
	"encoding/json"
	"errors"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
)

const connectorPreferencesFile = "connector-preferences.json"

type connectorPreferences struct {
	Enabled  map[string]bool            `json:"enabled"`
	Tools    map[string]map[string]bool `json:"tool_states"`
	Bindings map[string]string          `json:"connector_bindings"`
	Builtin  json.RawMessage            `json:"builtin_tools"`
	Revision int                        `json:"-"`
}

func (p connectorPreferences) hasEnabledConnector() bool {
	for key, enabled := range p.Enabled {
		if enabled && validUUID(strings.TrimPrefix(key, "mcp/server/")) && strings.HasPrefix(key, "mcp/server/") {
			return true
		}
	}
	return false
}

func (p connectorPreferences) normalized() connectorPreferences {
	if p.Enabled == nil {
		p.Enabled = map[string]bool{}
	}
	if p.Tools == nil {
		p.Tools = map[string]map[string]bool{}
	}
	if p.Bindings == nil {
		p.Bindings = map[string]string{}
	}
	if p.Builtin == nil {
		p.Builtin = json.RawMessage("{}")
	}
	return p
}

func (p connectorPreferences) builtinTools() builtinToolPreferences {
	var parsed builtinToolPreferences
	_ = json.Unmarshal(p.Builtin, &parsed)
	return parsed
}

var legacyWorkspaceToolNames = []string{legacyReadTool, legacyWriteTool, legacyEditTool, legacyPatchTool, legacyGlobTool, legacyGrepTool, directoryTool}

type savedConnectorPreferences struct {
	Revision int
	Value    connectorPreferences
}

func (l *acpLive) loadConnectorPreferences() error {
	file, err := l.store.Open(connectorPreferencesFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	var saved savedConnectorPreferences
	decoder := json.NewDecoder(io.LimitReader(file, 64<<10))
	if err := decoder.Decode(&saved); err != nil {
		return err
	}
	if saved.Revision < 0 || decoder.Decode(new(any)) != io.EOF {
		return errors.New("connector_preferences_invalid")
	}
	l.connectorPrefs = saved.Value
	l.connectorPrefs.Revision = saved.Revision
	return nil
}

func (l *acpLive) saveConnectorPreferences(raw json.RawMessage) error {
	var next connectorPreferences
	if l.store == nil || len(raw) > 48<<10 || json.Unmarshal(raw, &next) != nil || next.Enabled == nil || next.Bindings == nil || next.Tools == nil {
		return errors.New("connector_preferences_invalid")
	}
	if next.Builtin != nil && json.Unmarshal(next.Builtin, new(builtinToolPreferences)) != nil {
		return errors.New("connector_preferences_invalid")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	next.Revision = l.connectorPrefs.Revision + 1
	data, err := json.Marshal(savedConnectorPreferences{next.Revision, next})
	if err != nil {
		return err
	}
	if err := writeSourceState(l.store.Root, connectorPreferencesFile, data); err != nil {
		return err
	}
	l.connectorPrefs = next
	return nil
}

func (h *relaySessionHost) sessionConnectorResources(raw json.RawMessage, write bool) (any, error) {
	var request struct {
		SessionID string
		Meta      struct {
			ExpectedRevision *int
		} `json:"_meta"`
		Change struct {
			Kind, ID, Name, Server, ConnectorID, CredentialID string
			Enabled                                           *bool
		}
	}
	if json.Unmarshal(raw, &request) != nil || !validUUID(request.SessionID) {
		return nil, errors.New("invalid_session_resources")
	}
	l := h.live
	l.mu.Lock()
	defer l.mu.Unlock()
	session, ok := l.sessions[request.SessionID]
	if !ok {
		return nil, errors.New("session_not_found")
	}
	prefs := l.connectorPrefs
	if session.ConnectorPreferences != nil {
		prefs = *session.ConnectorPreferences
	}
	if write {
		if session.State == "running" {
			return nil, errors.New("session_running")
		}
		if request.Meta.ExpectedRevision != nil && *request.Meta.ExpectedRevision != session.ResourceRevision {
			return nil, errors.New("configuration_conflict")
		}
		prefs.Enabled = maps.Clone(prefs.Enabled)
		prefs.Tools = maps.Clone(prefs.Tools)
		prefs.Bindings = maps.Clone(prefs.Bindings)
		prefs = prefs.normalized()
		change := request.Change
		switch change.Kind {
		case "resource":
			id, ok := strings.CutPrefix(change.ID, "mcp/server/")
			if !ok || !validUUID(id) || change.Enabled == nil {
				return nil, errors.New("invalid_session_resources")
			}
			prefs.Enabled[change.ID] = *change.Enabled
		case "mcp_tool":
			id, ok := strings.CutPrefix(change.Server, "server/")
			if !ok || !validUUID(id) || change.Enabled == nil || change.Name == "" || len(change.Name) > 256 {
				return nil, errors.New("invalid_session_resources")
			}
			states := maps.Clone(prefs.Tools[change.Server])
			if states == nil {
				states = make(map[string]bool)
			}
			states[change.Name] = *change.Enabled
			prefs.Tools[change.Server] = states
		case "connector_credential":
			if !validUUID(change.ConnectorID) || !validUUID(change.CredentialID) {
				return nil, errors.New("invalid_session_resources")
			}
			if prefs.Bindings[change.ConnectorID] != change.CredentialID {
				delete(prefs.Tools, "server/"+change.ConnectorID)
			}
			prefs.Bindings[change.ConnectorID] = change.CredentialID
			prefs.Enabled["mcp/server/"+change.ConnectorID] = true
		case "builtin_tool":
			if !slices.Contains(legacyWorkspaceToolNames, change.Name) || change.Enabled == nil {
				return nil, errors.New("invalid_session_resources")
			}
			builtins := prefs.builtinTools()
			if *change.Enabled {
				builtins.Blacklist = slices.DeleteFunc(builtins.Blacklist, func(name string) bool { return name == change.Name })
				if len(builtins.Whitelist) > 0 && !slices.Contains(builtins.Whitelist, change.Name) {
					builtins.Whitelist = append(builtins.Whitelist, change.Name)
				}
			} else if !slices.Contains(builtins.Blacklist, change.Name) {
				builtins.Blacklist = append(builtins.Blacklist, change.Name)
			}
			prefs.Builtin, _ = json.Marshal(builtins)
		default:
			return nil, errors.New("invalid_session_resources")
		}
		session.ConnectorPreferences = &prefs
		session.ResourceRevision++
		if err := l.save(session); err != nil {
			return nil, err
		}
	}
	builtin := prefs.builtinTools()
	items := make([]map[string]any, 0, len(legacyWorkspaceToolNames))
	for _, name := range legacyWorkspaceToolNames {
		items = append(items, map[string]any{"kind": "tool", "id": "builtin-tool/" + name, "name": name, "available": true, "enabled": builtin.allows(name)})
	}
	return map[string]any{"sessionId": session.ID, "instance": 1, "preferences": prefs.normalized(),
		"preferencesRevision": session.ResourceRevision, "configurationRevision": session.ResourceRevision,
		"snapshot": map[string]any{"revision": session.ResourceRevision, "state_version": session.ResourceRevision, "items": items}}, nil
}
