package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type invocationKey struct {
	ID    string `json:"id"`
	Value string `json:"api_key"`
}

type modelUsage struct {
	Input  *int64 `json:"prompt_tokens"`
	Output *int64 `json:"completion_tokens"`
}

const defaultModelInputBudget = 64 << 10

type selectedModelConfig struct {
	ID                  string  `json:"id"`
	Model               string  `json:"model"`
	DisplayName         string  `json:"display_name"`
	CreditMultiplier    float64 `json:"credit_multiplier"`
	Protocol            string  `json:"protocol"`
	Kind                string  `json:"kind"`
	ContextWindowTokens int     `json:"context_window_tokens"`
	MaxOutputTokens     int     `json:"max_output_tokens"`
	SupportsReasoning   bool    `json:"supports_reasoning"`
	StreamMode          string  `json:"stream_mode"`
}

func (m selectedModelConfig) inputBudget() int {
	if m.ContextWindowTokens <= 0 {
		return defaultModelInputBudget
	}
	output := m.MaxOutputTokens
	if output <= 0 || output >= m.ContextWindowTokens {
		output = m.ContextWindowTokens / 4
	}
	return max(0, m.ContextWindowTokens-output)
}

func (c *client) selectedModel(ctx context.Context, auth authorization) (string, error) {
	config, err := c.selectedModelConfig(ctx, auth)
	return config.Protocol, err
}

func (c *client) selectedModelConfig(ctx context.Context, auth authorization) (selectedModelConfig, error) {
	return c.modelConfig(ctx, auth, modelID)
}

func (c *client) modelConfig(ctx context.Context, auth authorization, requested string) (selectedModelConfig, error) {
	for page := 1; page <= 100; page++ {
		var catalog struct {
			Total  int                   `json:"total_count"`
			Models []selectedModelConfig `json:"models"`
		}
		if err := c.json(ctx, "GET", fmt.Sprintf("/api/v1/models?page=%d&page_size=100", page), auth.Access, "", nil, &catalog); err != nil {
			return selectedModelConfig{}, err
		}
		for _, model := range catalog.Models {
			if !validUUID(model.ID) || (requested != "" && model.ID != requested) {
				continue
			}
			if model.Kind != "text" || (model.Protocol != "openai_chat_completions" && model.Protocol != "openai_responses") {
				if requested == "" {
					continue
				}
				return selectedModelConfig{}, errors.New("指定模型必须支持 OpenAI Responses 或 Chat Completions")
			}
			return model, nil
		}
		if page*100 >= catalog.Total || len(catalog.Models) == 0 {
			break
		}
	}
	return selectedModelConfig{}, errors.New("指定模型未开放给当前登录用户")
}

func (c *client) createModelKey(ctx context.Context, auth authorization) (invocationKey, error) {
	return c.createInvocationKey(ctx, auth, "model:invoke")
}

func (c *client) createInvocationKey(ctx context.Context, auth authorization, scope string) (invocationKey, error) {
	var key invocationKey
	err := c.json(ctx, "POST", "/api/v1/api-keys", auth.Access, "", map[string]any{
		"name": "AIbertAI Desktop Runtime", "scopes": []string{scope}, "expires_in_days": 1,
	}, &key)
	if err == nil && (key.Value == "" || !validUUID(key.ID)) {
		err = errors.New("私有后端没有签发有效的调用密钥")
	}
	return key, err
}

func (c *client) chat(ctx context.Context, protocol, key, session, selected, prompt string, stream bool, delta func(string) error) (modelUsage, string, error) {
	var usage modelUsage
	request := map[string]any{"model": selected, "stream": stream}
	path := "/v1/chat/completions"
	if protocol == "openai_responses" {
		path = "/v1/responses"
		request["input"] = prompt
	} else {
		request["messages"] = []map[string]string{{"role": "user", "content": prompt}}
		if stream {
			request["stream_options"] = map[string]bool{"include_usage": true}
		}
	}
	resp, err := c.requestModelRound(ctx, path, key, session, request)
	if err != nil {
		return usage, "", err
	}
	defer resp.Body.Close()
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		usage, content, finish, err := readBufferedModelText(resp.Body, protocol)
		if err != nil {
			return usage, "", err
		}
		return usage, finish, delta(content)
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	if protocol == "openai_responses" {
		return readResponses(scanner, delta)
	}
	finish := ""
	var receivedText, done bool
	for scanner.Scan() {
		if done {
			continue
		}
		line, ok := strings.CutPrefix(scanner.Text(), "data:")
		if !ok {
			continue
		}
		line = strings.TrimSpace(line)
		if line == "[DONE]" {
			done = true
			continue
		}
		var event struct {
			Usage   *modelUsage `json:"usage"`
			Choices []struct {
				Finish string `json:"finish_reason"`
				Delta  struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return usage, "", errors.New("指定模型返回了无效事件")
		}
		if event.Usage != nil {
			usage = *event.Usage
		}
		if len(event.Choices) == 0 {
			continue
		}
		choice := event.Choices[0]
		if choice.Finish != "" {
			finish = choice.Finish
		}
		if choice.Delta.Content != "" {
			receivedText = true
			if err := delta(choice.Delta.Content); err != nil {
				return usage, "", err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return usage, "", err
	}
	if done && receivedText && (finish == "stop" || finish == "length") {
		return usage, finish, nil
	}
	return usage, "", errors.New("模型事件流没有正常结束或未返回完整文本")
}

func readBufferedModelText(body io.Reader, protocol string) (modelUsage, string, string, error) {
	var usage modelUsage
	var message toolConversationMessage
	var finish string
	var err error
	if protocol == "openai_responses" {
		var result responsesToolResult
		if json.NewDecoder(io.LimitReader(body, 2<<20)).Decode(&result) != nil {
			return usage, "", "", errors.New("invalid_responses_tool_output")
		}
		usage, message, finish, err = result.message()
	} else {
		usage, message, finish, err = readChatToolResponse(body)
	}
	if err != nil {
		return usage, "", "", err
	}
	if message.Content == nil || *message.Content == "" || (finish != "stop" && finish != "length") {
		return usage, "", "", errors.New("指定模型没有完成文本输出")
	}
	return usage, *message.Content, finish, nil
}

func readResponses(scanner *bufio.Scanner, delta func(string) error) (modelUsage, string, error) {
	var usage modelUsage
	var receivedText bool
	finish := ""
	for scanner.Scan() {
		// Read through EOF so the backend can finalize the upstream result and usage.
		if finish != "" {
			continue
		}
		raw, ok := strings.CutPrefix(scanner.Text(), "data:")
		if !ok {
			continue
		}
		if strings.TrimSpace(raw) == "[DONE]" {
			break
		}
		var event struct {
			Type     string `json:"type"`
			Delta    string `json:"delta"`
			Response struct {
				Usage struct {
					Input  *int64 `json:"input_tokens"`
					Output *int64 `json:"output_tokens"`
				} `json:"usage"`
				Incomplete struct {
					Reason string `json:"reason"`
				} `json:"incomplete_details"`
			} `json:"response"`
		}
		// Responses can include delta objects for tools; this client consumes text events only.
		var kind struct{ Type string }
		if err := json.Unmarshal([]byte(raw), &kind); err != nil {
			return usage, "", errors.New("指定模型返回了无效 Responses 事件")
		}
		switch kind.Type {
		case "response.output_text.delta", "response.completed", "response.incomplete":
			if err := json.Unmarshal([]byte(raw), &event); err != nil {
				return usage, "", errors.New("指定模型返回了无效 Responses 文本事件")
			}
		case "response.failed", "error":
			return usage, "", errors.New("指定模型返回失败事件")
		default:
			continue
		}
		if event.Type == "response.output_text.delta" && event.Delta != "" {
			receivedText = true
			if err := delta(event.Delta); err != nil {
				return usage, "", err
			}
		}
		if event.Type == "response.completed" || event.Type == "response.incomplete" {
			usage = modelUsage{Input: event.Response.Usage.Input, Output: event.Response.Usage.Output}
			if receivedText && event.Type == "response.completed" {
				finish = "stop"
				continue
			}
			if receivedText && event.Response.Incomplete.Reason == "max_output_tokens" {
				finish = "length"
				continue
			}
			return usage, "", errors.New("指定模型没有完成文本输出")
		}
	}
	if err := scanner.Err(); err != nil {
		return usage, "", err
	}
	if finish != "" {
		return usage, finish, nil
	}
	return usage, "", errors.New("Responses 流在结束前断开")
}
