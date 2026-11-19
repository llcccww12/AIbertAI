package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func connectorPath(prefix, id, credential string) string {
	path := prefix + id
	if credential != "" {
		path += "/credentials/" + credential
	}
	return path
}

func (c *client) connectorTools(ctx context.Context, auth authorization, id, credentialID string) (any, error) {
	if !validUUID(id) || (credentialID != "" && !validUUID(credentialID)) {
		return nil, errors.New("invalid_request")
	}
	path := connectorPath("/api/v1/connectors/", id, credentialID)
	return c.knowledgeChange(ctx, auth, knowledgeOperation{Verb: http.MethodGet, Path: path + "/tools"})
}

type connectorCatalogRow struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Authorization string `json:"authorization_mode"`
	Credential    string `json:"credential_id"`
	Status        string `json:"authorization_status"`
	Credentials   []struct {
		ID     string `json:"id"`
		Status string `json:"authorization_status"`
	} `json:"credentials"`
}

func (c *client) employeeConnectors(ctx context.Context, auth authorization) ([]json.RawMessage, error) {
	all := make([]json.RawMessage, 0)
	for page := 1; ; page++ {
		var response struct {
			Connectors []json.RawMessage `json:"connectors"`
			Total      int               `json:"total_count"`
		}
		path := fmt.Sprintf("/api/v1/connectors?page=%d&page_size=100", page)
		if err := c.json(ctx, http.MethodGet, path, auth.Access, "", nil, &response); err != nil {
			return nil, err
		}
		for _, raw := range response.Connectors {
			var entry map[string]json.RawMessage
			var id string
			if json.Unmarshal(raw, &entry) != nil || json.Unmarshal(entry["id"], &id) != nil || !validUUID(id) {
				return nil, errors.New("connector_catalog_invalid")
			}
			entry["runtimeName"], _ = json.Marshal("server/" + id)
			data, err := json.Marshal(entry)
			if err != nil {
				return nil, err
			}
			all = append(all, data)
		}
		if page*100 >= response.Total || len(response.Connectors) == 0 {
			return all, nil
		}
	}
}

type connectorTool struct {
	ConnectorID, CredentialID string
	ConnectorName, Name       string
	Description               string
	Schema                    json.RawMessage
}

func (t connectorTool) alias() string {
	fingerprint := sha256.Sum256([]byte(t.ConnectorID + "\x00" + t.Name))
	return fmt.Sprintf("connector_%x", fingerprint[:16])
}

func (c *client) activeConnectorTools(ctx context.Context, auth authorization, p connectorPreferences) (map[string]connectorTool, error) {
	if !p.hasEnabledConnector() {
		return nil, nil
	}
	catalog, err := c.employeeConnectors(ctx, auth)
	if err != nil {
		return nil, err
	}
	result := make(map[string]connectorTool)
	for _, raw := range catalog {
		var entry connectorCatalogRow
		if json.Unmarshal(raw, &entry) != nil || !p.Enabled["mcp/server/"+entry.ID] {
			continue
		}
		credential := ""
		switch entry.Authorization {
		case "none":
		case "centralized":
			if entry.Status == "authorized" && validUUID(entry.Credential) {
				credential = entry.Credential
			}
		case "independent":
			chosen := p.Bindings[entry.ID]
			for _, candidate := range entry.Credentials {
				if candidate.ID == chosen && candidate.Status == "authorized" && validUUID(candidate.ID) {
					credential = candidate.ID
					break
				}
			}
		default:
			continue
		}
		if entry.Authorization != "none" && credential == "" {
			continue
		}
		var available struct {
			Items []struct {
				Name, Description string
				Schema            json.RawMessage `json:"input_schema"`
				Enabled           bool
			}
		}
		path := connectorPath("/api/v1/connectors/", entry.ID, credential) + "/tools"
		if err := c.json(ctx, http.MethodGet, path, auth.Access, "", nil, &available); err != nil {
			return nil, err
		}
		states := p.Tools["server/"+entry.ID]
		for _, tool := range available.Items {
			optIn, hasChoice := states[tool.Name]
			var object map[string]json.RawMessage
			if !tool.Enabled || hasChoice && !optIn || strings.TrimSpace(tool.Name) == "" ||
				len(tool.Schema) > 64<<10 || json.Unmarshal(tool.Schema, &object) != nil || object == nil || string(object["type"]) != `"object"` {
				continue
			}
			selected := connectorTool{entry.ID, credential, entry.Name, tool.Name, tool.Description, tool.Schema}
			result[selected.alias()] = selected
		}
	}
	return result, nil
}

func (c *client) callConnector(ctx context.Context, auth authorization, session, operation string, step int, tool connectorTool, args json.RawMessage) (string, error) {
	key, err := c.createInvocationKey(ctx, auth, "mcp:invoke")
	if validUUID(key.ID) {
		defer func() {
			cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 4*time.Second)
			defer stop()
			_ = c.json(cleanup, http.MethodDelete, "/api/v1/api-keys/"+key.ID, auth.Access, "", nil, nil)
		}()
	}
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": tool.Name, "arguments": args}})
	if err != nil {
		return "", err
	}
	idempotency := fmt.Sprintf("%s-%d", operation, step)
	path := connectorPath("/mcp/connectors/", tool.ConnectorID, tool.CredentialID)
	resp, err := c.requestWithHeaders(ctx, http.MethodPost, path, key.Value, "", session, "application/json",
		bytes.NewReader(payload), http.Header{"Idempotency-Key": []string{idempotency}})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var reply struct {
		Version string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  json.RawMessage
		Error   json.RawMessage
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 256<<10)).Decode(&reply) != nil || reply.Version != "2.0" ||
		reply.ID != 1 || len(reply.Result) == 0 || len(reply.Error) != 0 {
		return "", errors.New("connector_call_failed")
	}
	var result struct{ IsError bool }
	if json.Unmarshal(reply.Result, &result) != nil || result.IsError {
		return string(reply.Result), errors.New("connector_tool_failed")
	}
	return string(reply.Result), nil
}
