package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type desktopSharingRecipients struct {
	Users  []json.RawMessage `json:"users"`
	Groups []json.RawMessage `json:"groups"`
}

func (h *relaySessionHost) desktopSharing(ctx context.Context, method string, raw json.RawMessage) (any, error) {
	var p struct {
		Kind, ID, Q string
		Limit       int
		Users       []string `json:"user_ids"`
		Groups      []string `json:"group_ids"`
		Remove      bool
	}
	if json.Unmarshal(raw, &p) != nil {
		return nil, errors.New("invalid_request")
	}
	c := h.live.client
	auth, err := h.live.authorization(ctx)
	if err != nil {
		return nil, err
	}
	if method == "_monkeyai/users/search" || method == "_monkeyai/groups/search" {
		kind := "users"
		if method == "_monkeyai/groups/search" {
			kind = "groups"
		}
		query := strings.TrimSpace(p.Q)
		if p.Limit == 0 {
			p.Limit = 20
		}
		if len(query) > 200 || p.Limit < 1 || p.Limit > 100 {
			return nil, errors.New("invalid_request")
		}
		if kind == "users" && query == "" {
			return []json.RawMessage{}, nil
		}
		params := url.Values{"q": {query}, "limit": {strconv.Itoa(p.Limit)}}
		var response map[string][]json.RawMessage
		if err := c.json(ctx, http.MethodGet, "/api/v1/"+kind+"?"+params.Encode(), auth.Access, "", nil, &response); err != nil {
			return nil, err
		}
		if response[kind] == nil {
			return []json.RawMessage{}, nil
		}
		return response[kind], nil
	}
	switch p.Kind {
	case "model", "rule", "expert", "connector", "skill", "kb":
	default:
		return nil, errors.New("resource_operation_unavailable")
	}
	if !validUUID(p.ID) {
		return nil, errors.New("invalid_request")
	}
	if method == "_monkeyai/resources/recipients" {
		return c.desktopSharingRecipients(ctx, auth, p.Kind, p.ID)
	}
	if len(p.Users)+len(p.Groups) < 1 || len(p.Users)+len(p.Groups) > 100 {
		return nil, errors.New("sharing_invalid")
	}
	for _, targets := range [][]string{p.Users, p.Groups} {
		for _, id := range targets {
			if !validUUID(id) {
				return nil, errors.New("sharing_invalid")
			}
		}
	}
	verb := http.MethodPost
	if p.Remove {
		verb = http.MethodDelete
	}
	body := map[string]any{"resources": []map[string]string{{"type": p.Kind, "id": p.ID}}, "user_ids": p.Users, "group_ids": p.Groups}
	_, err = c.knowledgeChange(ctx, auth, knowledgeOperation{Verb: verb, Path: "/api/v1/resources/shares", Body: body})
	if err != nil {
		return nil, err
	}
	return map[string]any{"result": nil}, nil
}

func (c *client) desktopSharingRecipients(ctx context.Context, auth authorization, kind, id string) (desktopSharingRecipients, error) {
	var item struct {
		ID     string            `json:"id"`
		Users  []json.RawMessage `json:"shared_users"`
		Groups []json.RawMessage `json:"shared_groups"`
	}
	if kind == "model" {
		// The owner-only detail omits user profiles. The catalog returns direct
		// recipients for the owner and omits them for everyone else.
		models, err := c.desktopCatalog(ctx, auth, "models")
		if err != nil {
			return desktopSharingRecipients{}, err
		}
		for _, model := range models {
			item.ID, item.Users, item.Groups = "", nil, nil
			if json.Unmarshal(model, &item) == nil && item.ID == id && item.Users != nil {
				break
			}
			item.ID = ""
		}
		if item.ID == "" {
			return desktopSharingRecipients{}, errors.New("not_found")
		}
	} else {
		path := "/api/v1/" + kind + "s/" + id
		if kind == "kb" {
			path = "/api/v1/kb/" + id
		}
		if err := c.json(ctx, http.MethodGet, path, auth.Access, "", nil, &item); err != nil {
			return desktopSharingRecipients{}, err
		}
		if item.ID != id || item.Users == nil {
			return desktopSharingRecipients{}, errors.New("not_found")
		}
	}
	if item.Groups == nil {
		item.Groups = []json.RawMessage{}
	}
	return desktopSharingRecipients{Users: item.Users, Groups: item.Groups}, nil
}
