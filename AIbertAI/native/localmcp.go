package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type desktopMCP struct {
	proc   *exec.Cmd
	in     io.WriteCloser
	out    io.ReadCloser
	scan   *bufio.Scanner
	cancel context.CancelFunc
	seq    int
}

func localMCPExecutable(command string) (string, error) {
	if strings.TrimSpace(command) != command || strings.ContainsRune(command, 0) {
		return "", errors.New("local_mcp_command_invalid")
	}
	executable, err := exec.LookPath(command)
	if err != nil || !filepath.IsAbs(executable) {
		return "", errors.New("local_mcp_command_unavailable")
	}
	return executable, nil
}

func localMCPEnvironment(raw json.RawMessage) ([]string, error) {
	env := make(map[string]string)
	// Deliberately inherit only platform essentials. Account tokens and other
	// application environment variables are never inherited by a connector.
	for _, key := range []string{"PATH", "HOME", "USERPROFILE", "SystemRoot", "TEMP", "TMP", "TMPDIR", "LANG"} {
		if value, ok := os.LookupEnv(key); ok {
			env[key] = value
		}
	}
	extra, err := localMCPEnvValues(raw)
	if err != nil {
		return nil, err
	}
	for key, value := range extra {
		env[key] = value
	}
	out := make([]string, 0, len(env))
	for _, key := range slices.Sorted(maps.Keys(env)) {
		out = append(out, key+"="+env[key])
	}
	return out, nil
}

func localMCPEnvValues(raw json.RawMessage) (map[string]string, error) {
	var extra map[string]string
	if len(raw) != 0 && json.Unmarshal(raw, &extra) != nil {
		var rows []struct{ Name, Value string }
		if json.Unmarshal(raw, &rows) != nil {
			return nil, errors.New("local_mcp_environment_invalid")
		}
		extra = make(map[string]string, len(rows))
		for _, row := range rows {
			extra[row.Name] = row.Value
		}
	}
	for key, value := range extra {
		if key == "" || strings.ContainsAny(key, "=\x00") || strings.ContainsRune(value, 0) {
			return nil, errors.New("local_mcp_environment_invalid")
		}
	}
	return extra, nil
}

func openLocalMCP(ctx context.Context, entry legacyMCP, executable, cwd string) (*desktopMCP, error) {
	env, err := localMCPEnvironment(entry.Env)
	if err != nil {
		return nil, err
	}
	lifetime, cancel := context.WithTimeout(ctx, 2*time.Minute)
	cmd := exec.CommandContext(lifetime, executable, entry.Args...)
	cmd.Dir, cmd.Env, cmd.WaitDelay = cwd, env, 2*time.Second
	in, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, errors.New("local_mcp_start_failed")
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		_ = in.Close()
		return nil, errors.New("local_mcp_start_failed")
	}
	if err := cmd.Start(); err != nil {
		cancel()
		_ = in.Close()
		_ = out.Close()
		return nil, errors.New("local_mcp_start_failed")
	}
	context.AfterFunc(lifetime, func() {
		_ = in.Close()
		_ = out.Close()
	})
	server := &desktopMCP{proc: cmd, in: in, out: out, scan: bufio.NewScanner(io.LimitReader(out, 16<<20)), cancel: cancel}
	server.scan.Buffer(make([]byte, 4096), 1<<20)
	var initialized struct {
		Protocol string `json:"protocolVersion"`
	}
	data, err := server.rpc("initialize", map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{}, "clientInfo": map[string]string{"name": "AIbertAI", "version": "1"}})
	if err != nil || json.Unmarshal(data, &initialized) != nil || !slices.Contains([]string{"2024-11-05", "2025-03-26", "2025-06-18", "2025-11-25"}, initialized.Protocol) ||
		json.NewEncoder(in).Encode(map[string]string{"jsonrpc": "2.0", "method": "notifications/initialized"}) != nil {
		server.close()
		return nil, errors.New("local_mcp_handshake_failed")
	}
	return server, nil
}

func (m *desktopMCP) close() {
	m.cancel()
	_ = m.in.Close()
	_ = m.out.Close()
	_ = m.proc.Wait()
}

func (m *desktopMCP) rpc(method string, params any) (json.RawMessage, error) {
	m.seq++
	if err := json.NewEncoder(m.in).Encode(map[string]any{"jsonrpc": "2.0", "id": m.seq, "method": method, "params": params}); err != nil {
		return nil, errors.New("local_mcp_request_failed")
	}
	for m.scan.Scan() {
		var response struct {
			Version       string `json:"jsonrpc"`
			ID            int
			Method        string
			Result, Error json.RawMessage
		}
		if json.Unmarshal(m.scan.Bytes(), &response) != nil || response.Version != "2.0" {
			return nil, errors.New("local_mcp_protocol_invalid")
		}
		if response.Method != "" || response.ID != m.seq {
			continue
		}
		if len(response.Error) != 0 || len(response.Result) == 0 {
			return nil, errors.New("local_mcp_request_failed")
		}
		return response.Result, nil
	}
	return nil, errors.New("local_mcp_stream_closed")
}

func (m *desktopMCP) tools(id string, entry legacyMCP, states map[string]bool) (map[string]connectorTool, error) {
	tools := make(map[string]connectorTool)
	seen := make(map[string]bool)
	cursor := ""
	for range 16 {
		params := map[string]string{}
		if cursor != "" {
			params["cursor"] = cursor
		}
		raw, err := m.rpc("tools/list", params)
		var page struct {
			Tools []struct {
				Name, Description string
				InputSchema       json.RawMessage
			}
			NextCursor string
		}
		if err != nil || json.Unmarshal(raw, &page) != nil || len(page.Tools) > 128 {
			return nil, errors.New("local_mcp_tool_catalog_invalid")
		}
		for _, item := range page.Tools {
			name := item.Name
			next, explicit := states[name]
			if !explicit {
				next, explicit = entry.ToolStates[name]
			}
			var schema struct{ Type string }
			if name == "" || len(name) > 256 || !entry.Filter.allows(name) || explicit && !next ||
				len(item.InputSchema) > 64<<10 || json.Unmarshal(item.InputSchema, &schema) != nil || schema.Type != "object" {
				continue
			}
			tool := connectorTool{ConnectorID: id, ConnectorName: entry.Name, Name: name, Description: item.Description, Schema: item.InputSchema, Local: true}
			tools[tool.alias()] = tool
		}
		cursor = page.NextCursor
		if cursor == "" {
			return tools, nil
		}
		if seen[cursor] || len(tools) > 128 {
			break
		}
		seen[cursor] = true
	}
	return nil, errors.New("local_mcp_tool_catalog_invalid")
}

func (m *desktopMCP) invoke(tool connectorTool, args json.RawMessage) (string, error) {
	raw, err := m.rpc("tools/call", map[string]any{"name": tool.Name, "arguments": args})
	var outcome struct{ IsError bool }
	if err != nil || json.Unmarshal(raw, &outcome) != nil || outcome.IsError || len(raw) > 256<<10 {
		return "", errors.New("local_mcp_tool_failed")
	}
	return string(raw), nil
}

func (h *relaySessionHost) testSavedLocalConnector(ctx context.Context, raw json.RawMessage) (any, error) {
	var params struct {
		Name   string
		Server *legacyMCP
	}
	if json.Unmarshal(raw, &params) != nil {
		return nil, errors.New("local_mcp_not_found")
	}
	imported := h.runtimeResources()
	var existing legacyMCP
	if imported != nil && params.Name != "" {
		existing = imported.Local[localMCPToolID(params.Name)]
	}
	entry := existing
	if params.Server != nil {
		var err error
		entry, err = localConnectorInput(*params.Server, existing)
		if err != nil {
			return nil, err
		}
	}
	if entry.Name == "" {
		return nil, errors.New("local_mcp_not_found")
	}
	id := localMCPToolID(entry.Name)
	failure := func(err error) (any, error) {
		return map[string]any{"ok": false, "error": err.Error()}, nil
	}
	executable, err := localMCPExecutable(entry.Command)
	if err != nil {
		return failure(err)
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	server, err := openLocalMCP(ctx, entry, executable, h.profile)
	if err != nil {
		return failure(err)
	}
	defer server.close()
	tools, err := server.tools(id, entry, nil)
	if err != nil {
		return failure(err)
	}
	result := make([]map[string]any, 0, len(tools))
	for _, alias := range slices.Sorted(maps.Keys(tools)) {
		tool := tools[alias]
		result = append(result, map[string]any{"name": tool.Name, "description": tool.Description, "inputSchema": tool.Schema})
	}
	return map[string]any{"ok": true, "tools": result}, nil
}
