package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestKnowledgeUsesPrivateBackendAndAccount(t *testing.T) {
	c, err := newClient("https://private.example.test")
	if err != nil {
		t.Fatal(err)
	}
	const id = "d38ff8ce-b949-422d-a080-488b4c89c20a"
	const doc = "6bfc669b-55d4-40eb-afb3-6a9df8247430"
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "private.example.test" || r.Method != "GET" || r.Header.Get("Authorization") != "Bearer active" {
			t.Fatalf("wrong knowledge request: %v", r.URL)
		}
		var body string
		switch r.URL.Path {
		case "/api/v1/kb":
			if r.URL.Query().Get("q") != "a&b" || r.URL.Query().Get("owner") != "mine" || r.URL.Query().Get("tag_ids") != id {
				t.Fatalf("filters not encoded: %v", r.URL)
			}
			body = `{"knowledge_bases":[{"id":"` + id + `"}],"total_count":3}`
		case "/api/v1/kb/" + id + "/documents":
			if r.URL.Query().Get("status") != "learned" {
				t.Fatal(r.URL)
			}
			body = `{"items":[{"id":"` + doc + `"}],"total_count":1}`
		case "/api/v1/kb/" + id + "/documents/" + doc:
			body = `{"id":"` + doc + `","content":"example"}`
		default:
			t.Fatal(r.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	ctx, auth := t.Context(), authorization{Access: "active"}
	catalog, err := c.knowledgeBases(ctx, auth, knowledgeFilter{Page: 1, Size: 10, Query: "a&b", Owner: "mine", Tags: []string{id}})
	if err != nil || len(catalog.Bases) != 1 || catalog.TotalCount != 3 {
		t.Fatalf("catalog=%v err=%v", catalog, err)
	}
	list, err := c.knowledgeDocuments(ctx, auth, id, knowledgeFilter{Page: 1, Size: 10, Status: "learned"})
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("list=%v err=%v", list, err)
	}
	document, err := c.knowledgeDocument(ctx, auth, id, doc)
	if err != nil || !strings.Contains(string(document), "example") {
		t.Fatalf("document=%s err=%v", document, err)
	}
	if _, err := c.knowledgeDocument(ctx, auth, "../models", doc); err == nil {
		t.Fatal("non-UUID path was accepted")
	}
}
