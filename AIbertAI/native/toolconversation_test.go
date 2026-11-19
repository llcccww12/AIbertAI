package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSlideDeckToolsStopAtLimitWithoutInvitingFalseCompletion(t *testing.T) {
	for _, limit := range []int{64, maxACPDeckToolCalls} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			c, err := newClient("https://private.example.test")
			if err != nil {
				t.Fatal(err)
			}
			const invocations = 72
			requests := 0
			c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
				requests++
				body := `{"choices":[{"finish_reason":"stop","message":{"content":"Complete"}}]}`
				if requests <= invocations {
					body = fmt.Sprintf(`{"choices":[{"finish_reason":"tool_calls","message":{"tool_calls":[{"id":"call_%d","type":"function","function":{"name":"read_text_file","arguments":"{\"path\":\"slide.html\"}"}}]}}]}`, requests)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			answer, _, runs, err := c.completeWorkspaceToolTurn(t.Context(), "openai_chat_completions", "fixture", "session", "model", nil, "Finish 19 slides.", nil,
				false, true, limit, nil, &workspaceToolPolicy{SlideDeck: true}, defaultModelInputBudget, nil,
				func(toolConversationMessage, toolArguments, int) (string, error) { return "Slide content", nil })
			if limit < invocations {
				if err == nil || err.Error() != "model_tool_continuation_incomplete" || requests != limit || len(runs) != limit || answer != "" {
					t.Fatalf("tool limit left an opportunity for false completion: requests=%d runs=%d answer=%q err=%v", requests, len(runs), answer, err)
				}
				return
			}
			if err != nil || requests != invocations+1 || len(runs) != invocations || answer != "Complete" {
				t.Fatalf("deck could not finish: requests=%d runs=%d answer=%q err=%v", requests, len(runs), answer, err)
			}
		})
	}
}

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
