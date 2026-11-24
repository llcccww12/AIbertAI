package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
)

const dshPluginManagerTool = "plugin_manager"

var dshManagedEntries = []string{"include:time-context", "include:tool-todo", "include:tool-fs-search", "include:tool-ask-user", "include:present"}

// A live desktop credential and its authenticated employee profile own every management call.
func (h *relaySessionHost) dshPluginManagerBound(ctx context.Context, session acpSession) bool {
	if h.local == nil || h.live == nil || h.live.dsh == nil || h.live.dsh.LeasePool == nil ||
		!validUUID(h.local.serverID) || !validUUID(h.local.reportMachine) || !validUUID(h.local.userID) ||
		!filepath.IsAbs(h.profile) || session.Engine != "dsh" || session.ID != session.DSHSessionID {
		return false
	}
	home := filepath.Join(h.profile, "source-runtime", h.local.serverID, h.local.reportMachine, "dsh", "home")
	if h.live.dsh.SharedPluginHome != home {
		return false
	}
	if session.ExpertID == "" || session.ExpertID == designExpertID {
		return true
	}
	disabled, bound := ctx.Value(dshDisabledPluginsKey{}).([]string)
	if !bound || slices.Contains(disabled, "tool-plugin-manager") || h.live.dsh.ResolveDisabledPlugins == nil {
		return false
	}
	current, err := h.live.dsh.ResolveDisabledPlugins(ctx, session.ExpertID)
	return err == nil && !slices.Contains(current, "tool-plugin-manager")
}

// The SDK's native manager also supports package installation and profile-wide service changes.
// This employee approval surface only lists the inventory or toggles selected built-in entries.
func dshPluginApprovalTitle(input map[string]any) string {
	const scope = " · 当前账号在本机的所有 DSH 会话"
	switch input["action"] {
	case "list_plugins", "list_bundles":
		for key, value := range input {
			if key == "action" {
				continue
			}
			number, ok := value.(json.Number)
			if !ok || key != "offset" && key != "limit" {
				return ""
			}
			n, err := number.Int64()
			if err != nil || n < 0 || key == "limit" && (n < 1 || n > 100) {
				return ""
			}
		}
		if input["action"] == "list_bundles" {
			return "查看插件包" + scope
		}
		return "查看插件" + scope
	case "set_plugin":
		target, _ := input["target"].(string)
		enabled, ok := input["enabled"].(bool)
		if !ok || len(input) != 3 {
			return ""
		}
		if slices.Contains(dshManagedEntries, target) {
			if enabled {
				return "启用 " + target + scope
			}
			return "停用 " + target + scope
		}
	}
	return ""
}
