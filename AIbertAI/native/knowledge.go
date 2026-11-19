package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
)

type knowledgeCatalog struct {
	Bases      []json.RawMessage `json:"knowledge_bases"`
	TotalCount int               `json:"total_count"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
}

type knowledgePage struct {
	Items      []json.RawMessage `json:"items"`
	TotalCount int               `json:"total_count"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
}

type knowledgeFilter struct {
	Owner, Query, Status string
	Page, Size           int
	Tags                 []string
}

func (f knowledgeFilter) values() (url.Values, error) {
	if f.Page < 1 || f.Size < 1 || f.Size > 100 {
		return nil, errors.New("无效的知识库分页")
	}
	values := url.Values{"page": {strconv.Itoa(f.Page)}, "page_size": {strconv.Itoa(f.Size)}}
	if f.Query != "" {
		values.Set("q", f.Query)
	}
	return values, nil
}

func (c *client) knowledgeBases(ctx context.Context, auth authorization, filter knowledgeFilter) (knowledgeCatalog, error) {
	var page knowledgeCatalog
	values, err := filter.values()
	if err != nil {
		return page, err
	}
	switch filter.Owner {
	case "", "all", "team", "mine", "shared":
	default:
		return page, errors.New("无效的知识库归属筛选")
	}
	if filter.Owner != "" {
		values.Set("owner", filter.Owner)
	}
	for _, id := range filter.Tags {
		if !validUUID(id) {
			return page, errors.New("无效的知识库标签")
		}
		values.Add("tag_ids", id)
	}
	err = c.json(ctx, "GET", "/api/v1/kb?"+values.Encode(), auth.Access, "", nil, &page)
	return page, err
}

func (c *client) knowledgeDocuments(ctx context.Context, auth authorization, id string, filter knowledgeFilter) (knowledgePage, error) {
	var page knowledgePage
	if !validUUID(id) {
		return page, errors.New("无效的知识库标识")
	}
	values, err := filter.values()
	if err != nil {
		return page, err
	}
	if filter.Status != "" {
		values.Set("status", filter.Status)
	}
	err = c.json(ctx, "GET", "/api/v1/kb/"+id+"/documents?"+values.Encode(), auth.Access, "", nil, &page)
	return page, err
}

func (c *client) knowledgeDocument(ctx context.Context, auth authorization, id, documentID string) (json.RawMessage, error) {
	if !validUUID(id) || !validUUID(documentID) {
		return nil, errors.New("无效的知识库或文档标识")
	}
	var document json.RawMessage
	err := c.json(ctx, "GET", "/api/v1/kb/"+id+"/documents/"+documentID, auth.Access, "", nil, &document)
	return document, err
}
