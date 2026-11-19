package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"strings"
)

func (c *client) responsesToolRound(ctx context.Context, key, session string, messages []toolConversationMessage, tools []map[string]any, choice string) (modelUsage, toolConversationMessage, string, error) {
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
	request := map[string]any{"model": modelID, "input": input, "stream": false, "store": false,
		"include": []string{"reasoning.encrypted_content"}}
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
	data, err := json.Marshal(request)
	if err != nil {
		return usage, message, "", err
	}
	resp, err := c.request(ctx, "POST", "/v1/responses", key, "", session, "application/json", bytes.NewReader(data))
	if err != nil {
		return usage, message, "", err
	}
	defer resp.Body.Close()
	var result struct {
		Status string
		Output []json.RawMessage
		Usage  struct {
			Input  *int64 `json:"input_tokens"`
			Output *int64 `json:"output_tokens"`
		}
	}
	invalid := errors.New("invalid_responses_tool_output")
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 2<<20))
	if decoder.Decode(&result) != nil {
		return usage, message, "", invalid
	}
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
