package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestKnowledgeMovePreservesOptimisticRevision(t *testing.T) {
	const id = "d38ff8ce-b949-422d-a080-488b4c89c20a"
	const doc = "6bfc669b-55d4-40eb-afb3-6a9df8247430"
	var in desktopKnowledgeInput
	in.Method = "_monkeyai/kb/documents/move"
	in.Params.ID, in.Params.DocumentID = id, doc
	in.Params.Input = json.RawMessage(`{"expected_revision":3,"parent_id":null}`)
	op, err := desktopKnowledgeOperation(in)
	if err != nil || op.Verb != http.MethodPost || op.Path != "/api/v1/kb/"+id+"/documents/move" || !op.Single {
		t.Fatalf("op=%+v, err=%v", op, err)
	}
	encoded, err := json.Marshal(op.Body)
	want := `{"expected_revisions":{"` + doc + `":3},"ids":["` + doc + `"],"parent_id":null}`
	if err != nil || string(encoded) != want {
		t.Fatalf("body=%s, want=%s, err=%v", encoded, want, err)
	}
}

func TestKnowledgeRejectsFailedSingleOperation(t *testing.T) {
	c, err := newClient("https://private.example.test")
	if err != nil {
		t.Fatal(err)
	}
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("If-Match") != `"42"` {
			t.Fatal("If-Match was not forwarded")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"succeeded":[],"failed":[{"id":"a","reason":"document_revision_conflict"}]}`)),
		}, nil
	})
	_, err = c.knowledgeChange(t.Context(), authorization{Access: "token"}, knowledgeOperation{
		Verb: http.MethodPost, Path: "/api/v1/kb/test/documents/move", ETag: `"42"`, Single: true,
	})
	if err == nil || err.Error() != "document_revision_conflict" {
		t.Fatalf("incorrect result for stale document: %v", err)
	}
}
