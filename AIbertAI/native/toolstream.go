package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

func scanModelEvents(body io.Reader, consume func([]byte) error) error {
	// Limit the whole stream as well as any individual SSE event.
	reader := &io.LimitedReader{R: body, N: 32 << 20}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var data []byte
	flush := func() error {
		if len(data) == 0 {
			return nil
		}
		event := data
		data = nil
		return consume(event)
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		value, ok := strings.CutPrefix(line, "data:")
		if !ok {
			continue
		}
		if len(data)+len(value) > 1<<20 {
			return errors.New("model_event_too_large")
		}
		if len(data) != 0 {
			data = append(data, '\n')
		}
		data = append(data, strings.TrimPrefix(value, " ")...)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if reader.N == 0 {
		return errors.New("model_stream_too_large")
	}
	return flush()
}

func readChatToolStream(body io.Reader, delta func(string) error) (modelUsage, toolConversationMessage, string, error) {
	var usage modelUsage
	var text, reasoning, arguments strings.Builder
	message := toolConversationMessage{Role: "assistant"}
	var tool modelToolUse
	foundTool, done := false, false
	finish := ""
	invalid := errors.New("invalid_tool_model_response")
	err := scanModelEvents(body, func(data []byte) error {
		if done {
			return nil
		}
		if string(data) == "[DONE]" {
			done = true
			return nil
		}
		var event struct {
			Error   json.RawMessage `json:"error"`
			Usage   *modelUsage     `json:"usage"`
			Choices []struct {
				Finish string `json:"finish_reason"`
				Delta  struct {
					Content   string `json:"content"`
					Reasoning string `json:"reasoning_content"`
					ToolCalls []struct {
						Index int `json:"index"`
						modelToolUse
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if json.Unmarshal(data, &event) != nil || len(event.Choices) > 1 ||
			len(event.Error) != 0 && string(event.Error) != "null" {
			return invalid
		}
		if event.Usage != nil {
			usage = *event.Usage
		}
		if len(event.Choices) == 0 {
			return nil
		}
		choice := event.Choices[0]
		if choice.Finish != "" {
			finish = choice.Finish
		}
		if text.Len()+len(choice.Delta.Content) > 128<<10 ||
			reasoning.Len()+len(choice.Delta.Reasoning) > 1<<20 {
			return errors.New("session_answer_too_large")
		}
		reasoning.WriteString(choice.Delta.Reasoning)
		for _, part := range choice.Delta.ToolCalls {
			if part.Index != 0 || len(choice.Delta.ToolCalls) != 1 ||
				arguments.Len()+len(part.Function.Arguments) > 1<<20 ||
				part.Type != "" && tool.Type != "" && part.Type != tool.Type {
				return invalid
			}
			foundTool = true
			tool.ID += part.ID
			tool.Function.Name += part.Function.Name
			if len(tool.ID) > 256 || len(tool.Function.Name) > 256 {
				return invalid
			}
			if part.Type != "" {
				tool.Type = part.Type
			}
			arguments.WriteString(part.Function.Arguments)
		}
		if choice.Delta.Content != "" {
			text.WriteString(choice.Delta.Content)
			return delta(choice.Delta.Content)
		}
		return nil
	})
	if text.Len() != 0 {
		message.Content = new(text.String())
	}
	message.Reasoning = reasoning.String()
	if foundTool {
		tool.Function.Arguments = arguments.String()
		message.ToolCalls = []modelToolUse{tool}
	}
	if err == nil && finish == "" {
		err = invalid
	}
	return usage, message, finish, err
}
