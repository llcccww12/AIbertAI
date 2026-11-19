package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
)

var knowledgeErrorCode = regexp.MustCompile(`^[a-z][a-z0-9_]{0,79}$`)

type knowledgeOperation struct {
	Verb, Path, ETag string
	Body             any
	Detail, Single   bool
}

func (c *client) knowledgeChange(ctx context.Context, auth authorization, op knowledgeOperation) (any, error) {
	var body io.Reader
	if op.Body != nil {
		data, err := json.Marshal(op.Body)
		if err != nil {
			return nil, errors.New("invalid_request")
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, op.Verb, c.server+op.Path, body)
	if err != nil {
		return nil, errors.New("invalid_request")
	}
	req.Header.Set("Authorization", "Bearer "+auth.Access)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if op.ETag != "" {
		req.Header.Set("If-Match", op.ETag)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, errors.New("knowledge_request_failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var problem struct{ Error struct{ Code string } }
		if decoder.Decode(&problem) == nil && knowledgeErrorCode.MatchString(problem.Error.Code) {
			if resp.StatusCode != http.StatusUnauthorized || problem.Error.Code == "connection_expired" {
				return nil, errors.New(problem.Error.Code)
			}
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, errDesktopCredential
		}
		return nil, errors.New("knowledge_request_failed")
	}
	var payload json.RawMessage
	if decoder.Decode(&payload) != nil {
		return nil, errors.New("knowledge_request_failed")
	}
	if op.Single {
		var operationResult struct {
			Succeeded []string
			Failed    []struct{ Reason string }
		}
		if json.Unmarshal(payload, &operationResult) != nil || len(operationResult.Succeeded) != 1 {
			for _, item := range operationResult.Failed {
				if knowledgeErrorCode.MatchString(item.Reason) {
					return nil, errors.New(item.Reason)
				}
			}
			return nil, errors.New("knowledge_request_failed")
		}
	}
	if op.Detail {
		return map[string]any{"resource": payload, "etag": resp.Header.Get("ETag")}, nil
	}
	if tag := resp.Header.Get("ETag"); tag != "" {
		var fields map[string]json.RawMessage
		if json.Unmarshal(payload, &fields) != nil || fields == nil {
			return nil, errors.New("knowledge_request_failed")
		}
		fields["etag"], _ = json.Marshal(tag)
		return fields, nil
	}
	return payload, nil
}

func desktopKnowledgeOperation(in desktopKnowledgeInput) (knowledgeOperation, error) {
	p := in.Params
	invalid := errors.New("invalid_request")
	base := "/api/v1/kb"
	if in.Method == "_monkeyai/kb/save" && p.ID == "" && p.Input != nil {
		return knowledgeOperation{Verb: "POST", Path: base, Body: p.Input, Detail: true}, nil
	}
	if !validUUID(p.ID) {
		return knowledgeOperation{}, invalid
	}
	kb := base + "/" + p.ID
	documents := kb + "/documents"
	switch in.Method {
	case "_monkeyai/kb/get":
		return knowledgeOperation{Verb: "GET", Path: kb, Detail: true}, nil
	case "_monkeyai/kb/save", "_monkeyai/kb/delete", "_monkeyai/kb/permissions/update":
		if p.ETag == "" {
			return knowledgeOperation{}, errors.New("precondition_required")
		}
		switch in.Method {
		case "_monkeyai/kb/save":
			if p.Input != nil {
				return knowledgeOperation{Verb: "PUT", Path: kb, Body: p.Input, ETag: p.ETag, Detail: true}, nil
			}
		case "_monkeyai/kb/delete":
			return knowledgeOperation{Verb: "DELETE", Path: kb, ETag: p.ETag}, nil
		case "_monkeyai/kb/permissions/update":
			if p.Grants != nil {
				return knowledgeOperation{Verb: "PUT", Path: base + "/permissions/update", ETag: p.ETag,
					Body: map[string]any{"kb_id": p.ID, "grants": p.Grants}}, nil
			}
		}
	case "_monkeyai/kb/permissions/list":
		return knowledgeOperation{Verb: "GET", Path: base + "/permissions/list?kb_id=" + url.QueryEscape(p.ID)}, nil
	case "_monkeyai/kb/documents/save":
		if p.Input != nil && p.DocumentID == "" {
			return knowledgeOperation{Verb: "POST", Path: documents, Body: p.Input}, nil
		}
		if p.Input != nil && validUUID(p.DocumentID) {
			return knowledgeOperation{Verb: "PUT", Path: documents + "/" + p.DocumentID, Body: p.Input}, nil
		}
	case "_monkeyai/kb/documents/batch":
		var batch map[string]json.RawMessage
		if json.Unmarshal(p.Input, &batch) == nil && batch != nil {
			var action string
			if json.Unmarshal(batch["action"], &action) == nil && (action == "move" || action == "delete") {
				delete(batch, "action")
				return knowledgeOperation{Verb: "POST", Path: documents + "/" + action, Body: batch}, nil
			}
		}
	case "_monkeyai/kb/documents/restudy", "_monkeyai/kb/documents/move", "_monkeyai/kb/documents/delete":
		if !validUUID(p.DocumentID) {
			return knowledgeOperation{}, invalid
		}
		ids := []string{p.DocumentID}
		if in.Method == "_monkeyai/kb/documents/restudy" {
			return knowledgeOperation{Verb: "POST", Path: base + "/documents/restudy", Body: map[string]any{"kb_id": p.ID, "ids": ids}}, nil
		}
		if in.Method == "_monkeyai/kb/documents/delete" {
			input := map[string]any{"ids": ids}
			if p.ExpectedRevision != nil {
				input["expected_revisions"] = map[string]int64{p.DocumentID: *p.ExpectedRevision}
			}
			return knowledgeOperation{Verb: "POST", Path: documents + "/delete", Body: input, Single: true}, nil
		}
		var move map[string]any
		if json.Unmarshal(p.Input, &move) == nil && move != nil {
			if revision, ok := move["expected_revision"]; ok {
				move["expected_revisions"] = map[string]any{p.DocumentID: revision}
				delete(move, "expected_revision")
			}
			move["ids"] = ids
			return knowledgeOperation{Verb: "POST", Path: documents + "/move", Body: move, Single: true}, nil
		}
	}
	return knowledgeOperation{}, invalid
}
