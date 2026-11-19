package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
)

func (h *relaySessionHost) desktopDevices(ctx context.Context) (any, error) {
	auth, err := h.live.authorization(ctx)
	if err != nil {
		return nil, err
	}
	devices := []any{}
	if !slices.Contains(strings.Fields(auth.Scope), "device:connect") {
		return map[string]any{"serverId": h.local.serverID, "supported": false, "devices": devices}, nil
	}
	for page := 1; page <= 10; page++ {
		var result struct {
			Total int
			Items []struct {
				MachineID          string `json:"machine_id"`
				Name               string `json:"display_name"`
				Platform           string
				Status             string
				ExecutionSupported bool       `json:"execution_supported"`
				RemoteAccessState  string     `json:"remote_access_state"`
				LastSeenAt         *time.Time `json:"last_seen_at"`
				Protocols          map[string][]int
			}
		}
		path := fmt.Sprintf("/api/v1/endpoints?capability=execute&page=%d&page_size=100", page)
		if err := h.live.client.json(ctx, http.MethodGet, path, auth.Access, "", nil, &result); err != nil {
			return nil, errors.New("device_directory_unavailable")
		}
		for _, device := range result.Items {
			if !validUUID(device.MachineID) || device.MachineID == h.local.reportMachine ||
				device.MachineID == h.local.installedMachine || device.Status == "revoked" || device.Name == "" {
				continue
			}
			devices = append(devices, map[string]any{
				"target": map[string]string{"serverId": h.local.serverID, "userId": h.local.userID,
					"machineId": device.MachineID, "name": device.Name, "transport": "relay"},
				"platform": device.Platform, "status": device.Status, "accessState": device.RemoteAccessState,
				"lastSeenAt": device.LastSeenAt, "executionSupported": device.ExecutionSupported,
				"compatible": slices.Contains(device.Protocols["acp"], 1) && slices.Contains(device.Protocols["runtime_api"], 1),
			})
		}
		if page*100 >= result.Total || len(result.Items) == 0 {
			break
		}
	}
	return map[string]any{"serverId": h.local.serverID, "supported": true, "devices": devices}, nil
}

func (h *relaySessionHost) desktopConnect(ctx context.Context, raw json.RawMessage) (any, error) {
	var p struct{ ServerID, MachineID, ConnectionID string }
	if json.Unmarshal(raw, &p) != nil || p.ServerID != h.local.serverID ||
		p.MachineID == h.local.reportMachine || p.MachineID == h.local.installedMachine {
		return nil, errors.New("device_target_changed")
	}
	auth, err := h.live.authorization(ctx)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(strings.Fields(auth.Scope), "device:connect") {
		return nil, errors.New("permission_denied")
	}
	return h.live.client.remoteConnection(ctx, auth, p.ServerID, p.MachineID, p.ConnectionID)
}
