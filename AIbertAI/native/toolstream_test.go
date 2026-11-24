package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"
)

func TestResponsesToolStreamKeepsCompletedCallAfterReadTimeout(t *testing.T) {
	completed := eventStream(`{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":10,"output_tokens":2},"output":[{"type":"function_call","call_id":"call_fixture","name":"canvas_create_html","arguments":"{}","status":"completed"}]}}`)
	for _, tc := range []struct {
		name, events string
		wantErr      error
	}{
		{"completion received", completed, nil},
		{"completion missing", eventStream(`{"type":"response.output_text.delta","delta":"Working"}`), context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stream := io.MultiReader(strings.NewReader(tc.events), iotest.ErrReader(context.DeadlineExceeded))
			usage, message, finish, err := readResponsesToolStream(stream, func(string) error { return nil })
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err=%v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				if safeDesktopTurnFailure(err) != "backend_request_timeout" || len(message.ToolCalls) != 0 {
					t.Fatalf("incomplete output was accepted or timeout was hidden: %v", message)
				}
				return
			}
			if finish != "tool_calls" || len(message.ToolCalls) != 1 || message.ToolCalls[0].ID != "call_fixture" || usage.Output == nil || *usage.Output != 2 {
				t.Fatalf("lost completed tool call or usage: %s, %v, %v", finish, message, usage)
			}
		})
	}
}

func TestStreamedToolTurnForBothModelProtocols(t *testing.T) {
	type protocol struct {
		name, first, final string
	}
	for _, tc := range []protocol{
		{
			name: "openai_chat_completions",
			first: eventStream(
				`{"choices":[{"delta":{"content":"Before the tool. ","reasoning_content":"private"}}]}`,
				`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_fixture","type":"function","function":{"name":"read_text_file","arguments":"{\"path\":"}}]}}]}`,
				`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"notes.txt\"}"}}]},"finish_reason":"tool_calls"}]}`,
				`{"choices":[],"usage":{"prompt_tokens":10,"completion_tokens":2}}`, "[DONE]",
			),
			final: eventStream(
				`{"choices":[{"delta":{"content":"Complete"}}]}`,
				`{"choices":[{"delta":{"content":" answer."},"finish_reason":"stop"}]}`,
				`{"choices":[],"usage":{"prompt_tokens":11,"completion_tokens":3}}`, "[DONE]",
			),
		},
		{
			name: "openai_responses",
			first: eventStream(
				`{"type":"response.output_text.delta","delta":"Before the tool. "}`,
				`{"type":"response.function_call_arguments.delta","delta":"fixture"}`,
				`{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":10,"output_tokens":2},"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Before the tool. "}]},{"type":"reasoning","encrypted_content":"fixture"},{"type":"function_call","call_id":"call_fixture","name":"read_text_file","arguments":"{\"path\":\"notes.txt\"}","status":"completed"}]}}`,
			),
			final: eventStream(
				`{"type":"response.output_text.delta","delta":"Complete"}`,
				`{"type":"response.output_text.delta","delta":" answer."}`,
				`{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":11,"output_tokens":3},"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Complete answer."}]}]}}`,
			),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := newClient("https://private.example.test")
			if err != nil {
				t.Fatal(err)
			}
			requests := 0
			c.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				var request map[string]json.RawMessage
				if err := json.NewDecoder(req.Body).Decode(&request); err != nil || string(request["stream"]) != "true" {
					t.Fatalf("invalid streaming request: %v", err)
				}
				content := tc.first
				if requests == 2 {
					content = tc.final
					field := "messages"
					if tc.name == "openai_responses" {
						field = "input"
					}
					if !strings.Contains(string(request[field]), "fixture output") {
						t.Fatal("tool result missing from the continuation")
					}
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": {"text/event-stream"}},
					Body:       io.NopCloser(strings.NewReader(content)),
				}, nil
			})
			var shown strings.Builder
			answer, usage, runs, err := c.completeWorkspaceToolTurn(t.Context(), tc.name, "fixture", "session", "model", nil, "Read the notes.", nil,
				false, false, 1, nil, nil, defaultModelInputBudget, nil,
				func(message toolConversationMessage, args toolArguments, _ int) (string, error) {
					if shown.String() != "Before the tool. " || args.Workspace.Path != "notes.txt" || message.ToolCalls[0].ID != "call_fixture" {
						t.Fatalf("text or tool arguments were not assembled: %q %v", shown.String(), args.Workspace)
					}
					if tc.name == "openai_chat_completions" && message.Reasoning != "private" {
						t.Fatal("reasoning replay context missing")
					}
					return "fixture output", nil
				},
				func(text string) error { shown.WriteString(text); return nil })
			if err != nil || requests != 2 || answer != "Complete answer." ||
				shown.String() != "Before the tool. Complete answer." || len(runs) != 1 || runs[0].Preamble != "Before the tool. " ||
				usage.Input == nil || *usage.Input != 21 || usage.Output == nil || *usage.Output != 5 {
				t.Fatalf("lost text, tools or usage: %q %q %v %v", answer, shown.String(), usage, err)
			}
		})
	}
}

func eventStream(events ...string) string {
	return "data: " + strings.Join(events, "\n\ndata: ") + "\n\n"
}

func TestRequestModelRoundOnlyRetriesExplicitStreamRejection(t *testing.T) {
	for _, tc := range []struct {
		name, errorBody string
		stream          bool
		requests        int
	}{
		{"off", "", false, 1},
		{"unsupported stream", `{"error":{"param":"stream","message":"Streaming is not supported"}}`, true, 2},
		{"unrelated failure", `{"error":{"param":"model","message":"unsupported model"}}`, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := newClient("https://private.example.test")
			if err != nil {
				t.Fatal(err)
			}
			requests := 0
			c.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				var request map[string]any
				if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
					t.Fatal(err)
				}
				if got := request["stream"]; got != (tc.stream && requests == 1) {
					t.Fatalf("request %d stream = %v", requests, got)
				}
				if request["stream"] == false && request["stream_options"] != nil {
					t.Fatal("non-streaming request retained stream_options")
				}
				status, body := http.StatusOK, `{"choices":[{"message":{"content":"answer"},"finish_reason":"stop"}]}`
				if tc.stream && requests == 1 {
					status, body = http.StatusBadRequest, tc.errorBody
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			input := map[string]any{"stream": tc.stream}
			if tc.stream {
				input["stream_options"] = map[string]bool{"include_usage": true}
			}
			_, message, _, err := c.toolRound(t.Context(), "key", "session", input)
			if requests != tc.requests {
				t.Fatalf("requests = %d; want %d", requests, tc.requests)
			}
			if tc.name == "unrelated failure" {
				if err == nil {
					t.Fatal("suppressed unrelated HTTP error")
				}
			} else if err != nil || message.Content == nil || *message.Content != "answer" {
				t.Fatalf("invalid model response: %+v %v", message, err)
			}
		})
	}
}
