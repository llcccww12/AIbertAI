package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestResponsesToolRoundCarriesDeclaredCallAndResult(t *testing.T) {
	c, err := newClient("https://private.example.test")
	if err != nil {
		t.Fatal(err)
	}
	rounds := 0
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		rounds++
		var req struct {
			Input []map[string]any
			Tools []map[string]any
			Store bool
		}
		if r.URL.Path != "/v1/responses" || json.NewDecoder(r.Body).Decode(&req) != nil || req.Store {
			t.Fatalf("unexpected Responses request %s", r.URL.Path)
		}
		reply := `{"status":"completed","output":[{"type":"reasoning","id":"rs","summary":[],"encrypted_content":"opaque"},{"type":"function_call","call_id":"call_file","name":"read_text_file","arguments":"{\"path\":\"attachment.txt\"}"}],"usage":{"input_tokens":10,"output_tokens":5}}`
		if rounds == 1 {
			if len(req.Tools) != 1 || req.Tools[0]["type"] != "function" || req.Tools[0]["name"] != workspaceReadTool {
				t.Fatal("Responses tool was not declared as a function")
			}
		} else {
			if len(req.Input) != 4 || req.Input[1]["encrypted_content"] != "opaque" ||
				req.Input[2]["call_id"] != "call_file" || req.Input[3]["type"] != "function_call_output" ||
				req.Input[3]["call_id"] != "call_file" || req.Input[3]["output"] != "expected bytes" || len(req.Tools) != 0 {
				t.Fatal("continuation lost the approved tool invocation or exact result")
			}
			reply = `{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"result ok"}]}],"usage":{"input_tokens":4,"output_tokens":3}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(reply))}, nil
	})
	answer, usage, report, err := c.completeToolTurn(t.Context(), "openai_responses", "test-key", "", []string{"approved.txt", "attachment.txt"}, "read the attachment", nil, true, false,
		func(call modelToolUse, args workspaceToolArguments, replay []json.RawMessage) (string, error) {
			if call.ID != "call_file" || args.Path != "attachment.txt" || len(replay) != 2 {
				t.Fatal("invalid invocation reached the local tool")
			}
			return "expected bytes", nil
		})
	if err != nil || answer != "result ok" || report == nil || report.Calls != 1 ||
		report.ResourceID != workspaceReadTool ||
		usage.Input == nil || *usage.Input != 14 || usage.Output == nil || *usage.Output != 8 || rounds != 2 {
		t.Fatalf("invalid response %q usage=%+v report=%+v err=%v", answer, usage, report, err)
	}
}
