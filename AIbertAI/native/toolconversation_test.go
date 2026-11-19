package main

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestToolConversationRejectsModelRequestForDifferentFile(t *testing.T) {
	c, err := newClient("https://private.example.test")
	if err != nil {
		t.Fatal(err)
	}
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("incorrect model endpoint: %s", r.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
			`{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_text_file","arguments":"{\"path\":\"other.txt\"}"}}]}}]}`,
		))}, nil
	})
	_, report, err := c.runToolConversation(t.Context(), "fixture", "", "approved.txt", func(string) (string, error) {
		t.Error("model override reached the local file tool")
		return "", errors.New("unexpected invocation")
	})
	if err == nil || err.Error() != "invalid_model_tool_call" || report != nil {
		t.Fatalf("unexpected denial result: report=%+v err=%v", report, err)
	}
}
