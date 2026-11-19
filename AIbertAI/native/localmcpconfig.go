package main

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"strings"
	"unicode"
)

func localConnectorInput(input, existing legacyMCP) (legacyMCP, error) {
	invalid := errors.New("local_mcp_configuration_invalid")
	if input.Name == "" || len(input.Name) > 128 || strings.ContainsAny(input.Name, "/\\\x00") ||
		strings.IndexFunc(input.Name, unicode.IsControl) >= 0 ||
		existing.Name == "" && (strings.Contains(input.Name, "__") || strings.IndexFunc(input.Name, unicode.IsSpace) >= 0) ||
		input.Transport != "stdio" || strings.TrimSpace(input.Command) == "" || len(input.Command) > 4096 ||
		strings.ContainsRune(input.Command, 0) || len(input.Args) > 128 ||
		existing.Name != "" && input.Name != existing.Name {
		return legacyMCP{}, invalid
	}
	for _, arg := range input.Args {
		if len(arg) > 8192 || strings.ContainsRune(arg, 0) {
			return legacyMCP{}, invalid
		}
	}
	env, err := localMCPEnvValues(input.Env)
	if err != nil || len(env) > 128 {
		return legacyMCP{}, invalid
	}
	previous, err := localMCPEnvValues(existing.Env)
	if err != nil {
		return legacyMCP{}, err
	}
	for key, value := range env {
		if value == "" {
			if previousValue, ok := previous[key]; ok {
				env[key] = previousValue
			}
		}
	}
	result := existing
	result.Name, result.Transport, result.Command, result.Args = input.Name, "stdio", strings.TrimSpace(input.Command), input.Args
	result.Env, err = json.Marshal(env)
	return result, err
}

func (h *relaySessionHost) saveLocalConnector(ctx context.Context, raw json.RawMessage) (any, error) {
	var request struct {
		Name   string
		Server legacyMCP
		Delete bool
	}
	if json.Unmarshal(raw, &request) != nil || h.live == nil || h.live.store == nil {
		return nil, errors.New("local_mcp_configuration_invalid")
	}
	err := func() error {
		h.importedResourcesMu.Lock()
		defer h.importedResourcesMu.Unlock()
		store := h.live.store
		old := h.importedResources
		if old == nil {
			return errors.New("imported_runtime_resources_unreadable")
		}
		next := *old
		next.Revision++
		next.Local = maps.Clone(old.Local)
		if request.Delete {
			id := localMCPToolID(request.Name)
			if _, exists := old.Local[id]; !exists {
				return errors.New("local_mcp_not_found")
			}
			delete(next.Local, id)
		} else {
			id := localMCPToolID(request.Server.Name)
			existing, exists := old.Local[id]
			if request.Name == "" && exists {
				return errors.New("local_mcp_name_exists")
			}
			if request.Name != "" && (!exists || existing.Name != request.Name) {
				return errors.New("local_mcp_not_found")
			}
			entry, err := localConnectorInput(request.Server, existing)
			if err != nil {
				return err
			}
			next.Local[id] = entry
		}
		if err := next.save(store.Root, h.importedResourcesKey, h.live.client.server); err != nil {
			return err
		}
		h.importedResources = &next
		return nil
	}()
	if err != nil {
		return nil, err
	}
	return h.employeeConfig(ctx)
}
