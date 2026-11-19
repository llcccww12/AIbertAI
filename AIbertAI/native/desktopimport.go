package main

import (
	"context"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"slices"
	"strings"
)

const desktopImportReceipt = "legacy-resource-import-v3.json"

type desktopImportResult struct {
	Experts    int      `json:"experts"`
	Connectors int      `json:"connectors"`
	Pending    []string `json:"pending,omitempty"`
}

type legacyResourceIndex struct {
	Kind      string
	Resources []struct {
		ID, Name, Origin, Scope, Path string
	}
}

type legacyMCP struct {
	Name, Type, Transport, URL, Command, Description string
	DisplayName                                      string `json:"display_name"`
	Namespace                                        string `json:"tool_namespace"`
	Args                                             []string
	Env                                              json.RawMessage
	Headers                                          json.RawMessage
	Enabled                                          *bool
	Filter                                           builtinToolPreferences `json:"tool_filter"`
	ToolStates                                       map[string]bool        `json:"tool_states"`
	InsecureTLS                                      bool                   `json:"tls_insecure_skip_verify"`
	Connector                                        json.RawMessage
}

type desktopImportedResource struct {
	ID          string
	Credentials []struct {
		ID, Name string
	}
}

type legacyExpertBundle struct {
	ID, Name, Description string
	Version               int    `json:"schema_version"`
	Prompt                string `json:"prompt_file"`
	Rules, Skills         []struct{ Path string }
	Builtin               builtinToolPreferences `json:"builtin_tools"`
}

func readLegacyResource(root *os.Root, file string, limit int64) ([]byte, error) {
	info, err := root.Stat(file)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("legacy_resource_invalid")
	}
	f, err := root.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, errors.New("legacy_resource_invalid")
	}
	return data, nil
}

func legacyIndex(root *os.Root, kind string) (legacyResourceIndex, error) {
	var index legacyResourceIndex
	data, err := readLegacyResource(root, kind+"/.catalog.json", 1<<20)
	if errors.Is(err, os.ErrNotExist) {
		return index, nil
	}
	if err != nil || json.Unmarshal(data, &index) != nil || index.Kind != kind {
		return index, errors.New("legacy_catalog_invalid")
	}
	return index, nil
}

func (c *client) importDesktopResource(ctx context.Context, auth authorization, kind, legacyID string, input any) (desktopImportedResource, error) {
	sum := sha256.Sum256([]byte(legacyID))
	var saved desktopImportedResource
	route := "/api/v1/" + kind + "/desktop-import/" + hex.EncodeToString(sum[:])
	if err := c.json(ctx, http.MethodPut, route, auth.Access, "", input, &saved); err != nil {
		return saved, err
	}
	if !validUUID(saved.ID) {
		return saved, errors.New("legacy_import_response_invalid")
	}
	return saved, nil
}

// Import only material belonging to this profile. The profile lock is already
// held by the new runtime. The old directory is opened for reads only.
func (h *relaySessionHost) importLegacyResources(ctx context.Context, auth authorization, profileKey cipher.AEAD) desktopImportResult {
	var report desktopImportResult
	state := h.live.store.Root
	c := h.live.client
	imported, err := loadImportedDesktopResources(state, profileKey, c.server)
	if err != nil {
		report.Pending = []string{"imported_runtime_resources_unreadable"}
		return report
	}
	h.importedResources = imported
	h.importedResourcesKey = profileKey
	if data, err := readLegacyResource(state, desktopImportReceipt, 4096); err == nil && json.Unmarshal(data, &report) == nil {
		return report
	}
	report = desktopImportResult{}
	root, err := os.OpenRoot(h.profile)
	if err != nil {
		report.Pending = []string{"legacy_profile_unavailable"}
		return report
	}
	defer root.Close()
	connectors := map[string]string{}
	savedConnectors := map[[32]byte]string{}
	prefs := h.live.connectorPrefs.normalized()
	prefs.Bindings = maps.Clone(prefs.Bindings)
	prefs.Enabled = maps.Clone(prefs.Enabled)
	prefs.Tools = maps.Clone(prefs.Tools)
	prefsChanged, importedChanged := false, false
	localCount := map[string]bool{}
	saveConnector := func(key string, entry legacyMCP) {
		if key == "" {
			report.Pending = append(report.Pending, "legacy_connector_config_invalid")
			return
		}
		if connectors[key] != "" {
			return
		}
		if entry.Command != "" {
			if strings.TrimSpace(entry.Command) == "" || len(entry.Command) > 4096 || strings.TrimSpace(entry.Name) == "" || strings.Contains(entry.Name, "/") || len(entry.Name) > 128 {
				report.Pending = append(report.Pending, "legacy_stdio_config_invalid")
				return
			}
			id := localMCPToolID(entry.Name)
			if _, exists := imported.Local[id]; !exists {
				imported.Local[id] = entry
				importedChanged = true
			}
			connectors[key], localCount[id] = id, true
			return
		}
		input, err := legacyConnectorInput(entry)
		if err != nil {
			report.Pending = append(report.Pending, err.Error())
			return
		}
		data, _ := json.Marshal(input)
		signature := sha256.Sum256(data)
		clear(data)
		if id := savedConnectors[signature]; id != "" {
			connectors[key] = id
			return
		}
		saved, err := c.importDesktopResource(ctx, auth, "connectors", key, input)
		if err != nil {
			report.Pending = append(report.Pending, "legacy_connector_import_failed")
			return
		}
		connectors[key] = saved.ID
		savedConnectors[signature] = saved.ID
		if len(entry.Filter.Whitelist)+len(entry.Filter.Blacklist) != 0 {
			if _, exists := imported.Filters[saved.ID]; !exists {
				imported.Filters[saved.ID] = entry.Filter
				importedChanged = true
			}
		}
		if entry.Enabled != nil {
			prefKey := "mcp/server/" + saved.ID
			if _, exists := prefs.Enabled[prefKey]; !exists {
				prefs.Enabled[prefKey] = *entry.Enabled
				prefsChanged = true
			}
		}
		if len(entry.ToolStates) > 0 && prefs.Tools["server/"+saved.ID] == nil {
			prefs.Tools["server/"+saved.ID] = entry.ToolStates
			prefsChanged = true
		}
		if input["http_headers"] != nil && prefs.Bindings[saved.ID] == "" {
			for _, credential := range saved.Credentials {
				if credential.Name == "旧本机配置" && validUUID(credential.ID) {
					prefs.Bindings[saved.ID] = credential.ID
					prefsChanged = true
					break
				}
			}
		}
	}
	for _, file := range []string{"runtime/ohmyagent/connectors/user/local.enc.json", "mcp.enc.json"} {
		entries, err := readLegacyMCPArchive(root, file, profileKey)
		if err != nil {
			report.Pending = append(report.Pending, "legacy_mcp_archive_unreadable")
		}
		for _, entry := range entries {
			saveConnector(entry.Name, entry)
		}
	}
	if data, err := readLegacyResource(root, "mcp.json", 1<<20); err == nil {
		entries, err := decodeLegacyMCP(data)
		if err != nil {
			report.Pending = append(report.Pending, "legacy_mcp_catalog_invalid")
		}
		for _, entry := range entries {
			saveConnector(entry.Name, entry)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		report.Pending = append(report.Pending, "legacy_mcp_catalog_invalid")
	}
	engine, err := root.OpenRoot("runtime/ohmyagent")
	if err == nil {
		defer engine.Close()
		oldConnectors, err := legacyIndex(engine, "connectors")
		if err != nil {
			report.Pending = append(report.Pending, "legacy_mcp_catalog_invalid")
		}
		for _, item := range oldConnectors.Resources {
			if item.Origin != "local" || item.Scope != "mine" {
				continue
			}
			if _, exists := connectors[item.ID]; exists {
				continue
			}
			var entry legacyMCP
			file := path.Clean(item.Path)
			if !strings.HasPrefix(file, "connectors/") {
				report.Pending = append(report.Pending, "legacy_connector_config_unavailable")
				continue
			}
			data, err := readLegacyResource(engine, file, 1<<20)
			if err != nil || json.Unmarshal(data, &entry) != nil {
				report.Pending = append(report.Pending, "legacy_connector_config_unavailable")
				continue
			}
			if entry.Name == "" {
				entry.Name = item.Name
			}
			saveConnector(item.ID, entry)
		}

		experts, err := legacyIndex(engine, "experts")
		if err != nil {
			report.Pending = append(report.Pending, "legacy_expert_catalog_invalid")
		}
		seen := map[string]bool{}
		for _, item := range experts.Resources {
			if item.Origin != "local" || item.Scope != "mine" || item.ID == "" || seen[item.ID] {
				continue
			}
			seen[item.ID] = true
			input, runtime, err := legacyExpertInput(engine, item.Path, item.ID, connectors, imported.Local)
			if err != nil {
				report.Pending = append(report.Pending, err.Error())
				continue
			}
			saved, err := c.importDesktopResource(ctx, auth, "experts", item.ID, input)
			if err != nil {
				report.Pending = append(report.Pending, "legacy_expert_import_failed")
				continue
			}
			if len(runtime.LocalTools)+len(runtime.DesignTools)+len(runtime.Builtin.Whitelist)+len(runtime.Builtin.Blacklist) != 0 && !reflect.DeepEqual(imported.Experts[saved.ID], runtime) {
				imported.Experts[saved.ID] = runtime
				importedChanged = true
			}
			report.Experts++
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		report.Pending = append(report.Pending, "legacy_profile_unavailable")
	}
	report.Connectors = len(savedConnectors) + len(localCount)
	if importedChanged {
		if err := imported.save(state, profileKey, c.server); err != nil {
			report.Pending = append(report.Pending, "imported_runtime_resources_save_failed")
			h.importedResources = nil
		}
	}
	if prefsChanged {
		data, _ := json.Marshal(prefs)
		if err := h.live.saveConnectorPreferences(data); err != nil {
			report.Pending = append(report.Pending, "legacy_mcp_binding_save_failed")
		}
	}
	slices.Sort(report.Pending)
	report.Pending = slices.Compact(report.Pending)
	if len(report.Pending) == 0 && report.Connectors+report.Experts > 0 {
		data, _ := json.Marshal(report)
		if err := writeSourceState(state, desktopImportReceipt, data); err != nil {
			report.Pending = []string{"legacy_import_receipt_unavailable"}
		}
	}
	return report
}

func decodeLegacyMCP(data []byte) ([]legacyMCP, error) {
	var rows []legacyMCP
	if json.Unmarshal(data, &rows) == nil && rows != nil {
		return rows, nil
	}
	var input struct {
		Servers json.RawMessage `json:"mcpServers"`
	}
	if json.Unmarshal(data, &input) != nil || len(input.Servers) == 0 {
		return nil, errors.New("legacy_mcp_catalog_invalid")
	}
	if json.Unmarshal(input.Servers, &rows) == nil && rows != nil {
		return rows, nil
	}
	var named map[string]legacyMCP
	if json.Unmarshal(input.Servers, &named) != nil || named == nil {
		return nil, errors.New("legacy_mcp_catalog_invalid")
	}
	for name, row := range named {
		row.Name = name
		rows = append(rows, row)
	}
	return rows, nil
}

func legacyConnectorInput(entry legacyMCP) (map[string]any, error) {
	transport := entry.Transport
	if transport == "" {
		transport = entry.Type
	}
	if entry.Command != "" || transport != "" && transport != "http" && transport != "sse" && transport != "streamable-http" {
		return nil, errors.New("legacy_mcp_transport_unsupported")
	}
	u, err := url.Parse(entry.URL)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" ||
		(u.Scheme != "https" && u.Scheme != "http") || strings.TrimSpace(entry.Name) == "" {
		return nil, errors.New("legacy_connector_config_invalid")
	}
	name := entry.DisplayName
	if strings.TrimSpace(name) == "" {
		name = entry.Name
	}
	input := map[string]any{"name": name, "description": entry.Description, "url": entry.URL, "authorization_mode": "none", "skip_tls_verify": entry.InsecureTLS}
	if len(entry.Headers) == 0 {
		return input, nil
	}
	headers := map[string]string{}
	if json.Unmarshal(entry.Headers, &headers) != nil {
		var list []struct{ Name, Value string }
		if json.Unmarshal(entry.Headers, &list) != nil {
			return nil, errors.New("legacy_mcp_headers_invalid")
		}
		for _, header := range list {
			name := http.CanonicalHeaderKey(header.Name)
			if _, duplicate := headers[name]; duplicate {
				return nil, errors.New("legacy_mcp_headers_invalid")
			}
			headers[name] = header.Value
		}
	}
	if len(headers) > 0 {
		input["http_headers"] = headers
	}
	return input, nil
}

func legacyExpertInput(root *os.Root, location, id string, connectors map[string]string, local map[string]legacyMCP) (map[string]any, importedExpertRuntime, error) {
	var runtime importedExpertRuntime
	location = path.Clean(location)
	badBundle := errors.New("legacy_expert_bundle_invalid")
	if !strings.HasPrefix(location, "experts/") {
		return nil, runtime, badBundle
	}
	bundle, err := root.OpenRoot(location)
	if err != nil {
		return nil, runtime, badBundle
	}
	defer bundle.Close()
	var manifest struct {
		Version  int `json:"schema_version"`
		Kind, ID string
		Entry    string
		Requires struct {
			HostTools []string `json:"host_tools"`
			MCP       []string `json:"agent_mcp_servers"`
		}
	}
	data, err := readLegacyResource(bundle, "bundle.json", 1<<20)
	if err != nil || json.Unmarshal(data, &manifest) != nil || manifest.Version != 1 || manifest.Kind != "expert" || manifest.ID != id {
		return nil, runtime, badBundle
	}
	var expert legacyExpertBundle
	data, err = readLegacyResource(bundle, manifest.Entry, 1<<20)
	if err != nil || json.Unmarshal(data, &expert) != nil || expert.Version != 1 || expert.ID != id {
		return nil, runtime, badBundle
	}
	runtime.Builtin = expert.Builtin
	for _, name := range manifest.Requires.HostTools {
		if designTool(name) {
			runtime.DesignTools = append(runtime.DesignTools, name)
			continue
		}
		if !slices.Contains(legacyWorkspaceToolNames, name) && name != workspaceReadTool && name != workspaceWriteTool && name != legacyPatchTool {
			return nil, runtime, errors.New("legacy_expert_host_tools_unsupported")
		}
	}
	links := []map[string]any{}
	for _, key := range manifest.Requires.MCP {
		connectorID := connectors[key]
		if connectorID == "" {
			return nil, runtime, errors.New("legacy_expert_connector_unavailable")
		}
		if _, exists := local[connectorID]; exists {
			runtime.LocalTools = append(runtime.LocalTools, connectorID)
			continue
		}
		links = append(links, map[string]any{"connector_id": connectorID, "required": true})
	}
	files := []string{expert.Prompt}
	for _, rule := range expert.Rules {
		files = append(files, rule.Path)
	}
	for _, skill := range expert.Skills {
		files = append(files, path.Join(skill.Path, "SKILL.md"))
	}
	var instructions strings.Builder
	for _, file := range files {
		body, err := readLegacyResource(bundle, file, 256<<10)
		if err != nil || instructions.Len()+len(body)+2 > 256<<10 {
			return nil, runtime, badBundle
		}
		if instructions.Len() > 0 {
			instructions.WriteString("\n\n")
		}
		instructions.Write(body)
	}
	return map[string]any{"name": expert.Name, "description": expert.Description, "prompt": instructions.String(), "connectors": links}, runtime, nil
}
