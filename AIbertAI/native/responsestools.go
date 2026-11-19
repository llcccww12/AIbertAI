package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"strings"
)

func (c *client) responsesToolRound(ctx context.Context, key, session, selected string, messages []toolConversationMessage, tools []map[string]any, choice string, modelOptions map[string]any, deltas ...func(string) error) (modelUsage, toolConversationMessage, string, error) {
	input := make([]any, 0, len(messages))
	for _, message := range messages {
		if len(message.ResponseItems) != 0 {
			for _, item := range message.ResponseItems {
				input = append(input, item)
			}
			continue
		}
		if message.Role == "tool" && message.Content != nil {
			input = append(input, map[string]string{"type": "function_call_output", "call_id": message.ToolCallID, "output": *message.Content})
			continue
		}
		if message.Content != nil {
			input = append(input, map[string]string{"role": message.Role, "content": *message.Content})
		}
		for _, call := range message.ToolCalls {
			input = append(input, map[string]string{"type": "function_call", "call_id": call.ID,
				"name": call.Function.Name, "arguments": call.Function.Arguments})
		}
	}
	request := map[string]any{"model": selected, "input": input, "stream": false, "store": false,
		"include": []string{"reasoning.encrypted_content"}}
	maps.Copy(request, modelOptions)
	var delta func(string) error
	if len(deltas) != 0 {
		delta = deltas[0]
	}
	if delta != nil {
		request["stream"] = true
	}
	if choice != "" {
		declared := make([]map[string]any, 0, len(tools))
		for _, tool := range tools {
			item := maps.Clone(tool)
			_, external := item["parameters"].(json.RawMessage)
			item["type"], item["strict"] = "function", !external
			declared = append(declared, item)
		}
		request["tools"], request["tool_choice"], request["parallel_tool_calls"] = declared, choice, false
	}
	var usage modelUsage
	var message toolConversationMessage
	resp, err := c.requestModelRound(ctx, "/v1/responses", key, session, request)
	if err != nil {
		return usage, message, "", err
	}
	defer resp.Body.Close()
	if delta != nil && strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		return readResponsesToolStream(resp.Body, delta)
	}
	var result responsesToolResult
	if json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&result) != nil {
		return usage, message, "", errors.New("invalid_responses_tool_output")
	}
	usage, message, finish, err := result.message()
	if err == nil && delta != nil && message.Content != nil {
		err = delta(*message.Content)
	}
	return usage, message, finish, err
}

type responsesToolResult struct {
	Status string
	Output []json.RawMessage
	Usage  struct {
		Input  *int64 `json:"input_tokens"`
		Output *int64 `json:"output_tokens"`
	}
}

func (result responsesToolResult) message() (modelUsage, toolConversationMessage, string, error) {
	var usage modelUsage
	var message toolConversationMessage
	invalid := errors.New("invalid_responses_tool_output")
	usage = modelUsage{Input: result.Usage.Input, Output: result.Usage.Output}
	if result.Status != "completed" || len(result.Output) == 0 {
		return usage, message, "", invalid
	}
	message.Role = "assistant"
	var content strings.Builder
	for _, raw := range result.Output {
		var item struct {
			Type, Role, Status, Name, Arguments string
			CallID                              string `json:"call_id"`
			Content                             []struct{ Type, Text string }
		}
		if json.Unmarshal(raw, &item) != nil {
			return usage, message, "", invalid
		}
		switch item.Type {
		case "message":
			if item.Role != "assistant" {
				return usage, message, "", invalid
			}
			for _, chunk := range item.Content {
				if chunk.Type == "output_text" {
					content.WriteString(chunk.Text)
				}
			}
		case "function_call":
			if item.Status != "" && item.Status != "completed" {
				return usage, message, "", invalid
			}
			invocation := modelToolUse{ID: item.CallID, Type: "function"}
			invocation.Function.Name, invocation.Function.Arguments = item.Name, item.Arguments
			message.ToolCalls = append(message.ToolCalls, invocation)
		case "reasoning":
			// Preserve replay context without exposing it as assistant text.
		default:
			return usage, message, "", invalid
		}
	}
	if content.Len() != 0 {
		message.Content = new(content.String())
	}
	if len(message.ToolCalls) != 0 {
		message.ResponseItems = result.Output
		return usage, message, "tool_calls", nil
	}
	return usage, message, "stop", nil
}

func readResponsesToolStream(body io.Reader, delta func(string) error) (modelUsage, toolConversationMessage, string, error) {
	var result *responsesToolResult
	var text strings.Builder
	invalid := errors.New("invalid_responses_tool_output")
	err := scanModelEvents(body, func(data []byte) error {
		if string(data) == "[DONE]" || result != nil {
			return nil
		}
		var event struct {
			Type     string
			Delta    json.RawMessage
			Response *responsesToolResult
		}
		if json.Unmarshal(data, &event) != nil {
			return invalid
		}
		switch event.Type {
		case "response.output_text.delta":
			var chunk string
			if json.Unmarshal(event.Delta, &chunk) != nil || text.Len()+len(chunk) > 128<<10 {
				return invalid
			}
			if chunk != "" {
				text.WriteString(chunk)
				return delta(chunk)
			}
		case "response.completed":
			result = event.Response
			if result == nil {
				return invalid
			}
		case "error", "response.failed", "response.incomplete":
			return invalid
		}
		return nil
	})
	if err == nil && result != nil {
		usage, message, finish, err := result.message()
		if err == nil && message.Content != nil {
			if remainder, prefix := strings.CutPrefix(*message.Content, text.String()); prefix && remainder != "" {
				err = delta(remainder)
			}
		}
		return usage, message, finish, err
	}
	if err == nil {
		err = invalid
	}
	var partial toolConversationMessage
	if text.Len() != 0 {
		partial.Content = new(text.String())
	}
	return modelUsage{}, partial, "", err
}
