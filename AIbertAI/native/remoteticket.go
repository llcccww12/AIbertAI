package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"
)

type remoteTicket struct {
	MachineID    string   `json:"machineId"`
	ConnectionID string   `json:"connectionId"`
	HTTPBase     string   `json:"httpBase"`
	Token        string   `json:"token"`
	Version      int64    `json:"version"`
	RemainingMS  int64    `json:"remainingMs"`
	Scopes       []string `json:"scopes"`
	FilesOrigin  string   `json:"filesOrigin,omitempty"`
}

func (c *client) remoteConnection(ctx context.Context, auth authorization, serverID, machineID, connectionID string) (remoteTicket, error) {
	var ticket remoteTicket
	if !validUUID(serverID) || !validUUID(machineID) || (connectionID != "" && !validUUID(connectionID)) {
		return ticket, errors.New("invalid_request")
	}
	var metadata struct {
		Issuer  string `json:"issuer"`
		Runtime struct {
			ServerID    string `json:"server_id"`
			FilesOrigin string `json:"files_origin"`
			Features    struct {
				Relay bool `json:"runtime_relay"`
			} `json:"features"`
		} `json:"monkeyai_runtime"`
	}
	if err := c.json(ctx, "GET", "/.well-known/oauth-authorization-server", "", "", nil, &metadata); err != nil {
		return ticket, errDesktopCredential
	}
	if metadata.Issuer != c.server || metadata.Runtime.ServerID != serverID || !metadata.Runtime.Features.Relay {
		return ticket, errors.New("device_target_changed")
	}
	if metadata.Runtime.FilesOrigin != "" {
		origin, err := url.Parse(metadata.Runtime.FilesOrigin)
		server, _ := url.Parse(c.server)
		if err != nil || origin.Hostname() == "" || origin.Scheme != server.Scheme || origin.User != nil ||
			(origin.Path != "" && origin.Path != "/") || origin.RawQuery != "" || origin.ForceQuery || origin.Fragment != "" ||
			(backendOrigin != "" && !sameOrigin(origin, server)) {
			return ticket, errors.New("invalid_device_ticket")
		}
	}
	// A renewal must keep the existing granted scopes. For a new ticket the
	// backend computes the intersection with the desktop OAuth grant.
	message := struct {
		ConnectionID       string           `json:"connection_id,omitempty"`
		SupportedProtocols map[string][]int `json:"supported_protocols"`
	}{connectionID, map[string][]int{"acp": {1}, "runtime_api": {1}}}
	path := "/api/v1/endpoints/" + machineID
	result, err := c.knowledgeChange(ctx, auth, knowledgeOperation{Verb: http.MethodPost, Path: path + "/connections", Body: message})
	if err != nil {
		return ticket, err
	}
	body, ok := result.(json.RawMessage)
	if !ok {
		return ticket, errors.New("invalid_device_ticket")
	}
	var response struct {
		ConnectionID string    `json:"connection_id"`
		MachineID    string    `json:"machine_id"`
		HTTPBase     string    `json:"http_base"`
		Ticket       string    `json:"ticket"`
		Version      int64     `json:"ticket_version"`
		Expires      time.Time `json:"expires_at"`
		ServerTime   time.Time `json:"server_time"`
		Scopes       []string  `json:"scopes"`
	}
	if json.Unmarshal(body, &response) != nil || response.HTTPBase != c.server+path+"/proxy/" ||
		response.MachineID != machineID || !validUUID(response.ConnectionID) || response.Version < 1 ||
		response.ServerTime.IsZero() || len(response.Scopes) == 0 {
		return ticket, errors.New("invalid_device_ticket")
	}
	remaining := response.Expires.Sub(response.ServerTime)
	if remaining <= 0 || remaining > 15*time.Minute {
		return ticket, errors.New("invalid_device_ticket")
	}
	return remoteTicket{
		MachineID: response.MachineID, ConnectionID: response.ConnectionID,
		HTTPBase: response.HTTPBase, Token: response.Ticket, Version: response.Version,
		RemainingMS: remaining.Milliseconds(), Scopes: response.Scopes, FilesOrigin: metadata.Runtime.FilesOrigin,
	}, nil
}
