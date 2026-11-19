package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"slices"
	"time"
)

type toolConversationMessage struct {
	Role          string            `json:"role"`
	Content       *string           `json:"content"`
	Reasoning     string            `json:"reasoning_content,omitempty"`
	ToolCalls     []modelToolUse    `json:"tool_calls,omitempty"`
	ToolCallID    string            `json:"tool_call_id,omitempty"`
	ResponseItems []json.RawMessage `json:"-"`
}

type modelToolUse struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

const workspaceReadTool = "read_text_file"
const workspaceWriteTool = "write_text_file"

// Accept one invocation of the declared read-only tool. The model cannot change
// workspace permissions or trigger repeated tool execution in this verifier.
func (c *client) runToolConversation(ctx context.Context, key, session, approved string, call func(string) (string, error)) (modelUsage, *reportedTool, error) {
	const prompt = "Use the provided tool once to read the approved test file and acknowledge the result without quoting its contents."
	_, usage, fact, err := c.completeToolTurn(ctx, "openai_chat_completions", key, session, []string{approved}, prompt, nil, true, false, func(_ modelToolUse, args workspaceToolArguments, _ []json.RawMessage) (string, error) {
		return call(args.Path)
	})
	return usage, fact, err
}

type workspaceToolArguments struct {
	Path    string
	Content *string
}

type toolArguments struct {
	Workspace workspaceToolArguments
	Connector *connectorTool
	Compat    *workspaceCompatCall
	Design    bool
	JSON      json.RawMessage
}

type workspaceToolRun struct {
	Call          *modelToolUse     `json:"call,omitempty"`
	Report        reportedTool      `json:"report"`
	Output        string            `json:"output,omitempty"`
	ResponseItems []json.RawMessage `json:"responseItems,omitempty"`
}

func (c *client) completeToolTurn(ctx context.Context, protocol, key, session string, approved []string, prompt string, history []toolConversationMessage, requireTool, allowWrite bool, call func(modelToolUse, workspaceToolArguments, []json.RawMessage) (string, error)) (string, modelUsage, *reportedTool, error) {
	answer, usage, runs, err := c.completeWorkspaceToolTurn(ctx, protocol, key, session, approved, prompt, history, requireTool, allowWrite, 1, nil, nil, defaultModelInputBudget,
		func(message toolConversationMessage, args toolArguments, _ int) (string, error) {
			return call(message.ToolCalls[0], args.Workspace, message.ResponseItems)
		})
	if len(runs) == 0 {
		return answer, usage, nil, err
	}
	return answer, usage, &runs[0].Report, err
}

func (c *client) completeWorkspaceToolTurn(ctx context.Context, protocol, key, session string, approved []string, prompt string, history []toolConversationMessage, requireTool, allowWrite bool, maxCalls int, connectors map[string]connectorTool, workspace *workspaceToolPolicy, inputBudget int, call func(toolConversationMessage, toolArguments, int) (string, error)) (string, modelUsage, []workspaceToolRun, error) {
	talk := append(slices.Clone(history), toolConversationMessage{Role: "user", Content: &prompt})
	fileParameter := map[string]any{"type": "string", "description": "A workspace-relative path or the exact name of a file attached to this turn"}
	if len(approved) != 0 {
		fileParameter["enum"] = approved
	}
	var preferences builtinToolPreferences
	if workspace != nil {
		preferences = workspace.Preferences
	}
	tools := workspace.extraDefinitions()
	if preferences.allows(workspaceReadTool) {
		tools = append(tools, map[string]any{"name": workspaceReadTool,
			"description": "Read a UTF-8 file from the current workspace or opened attachment",
			"parameters": map[string]any{"type": "object", "additionalProperties": false,
				"properties": map[string]any{"path": fileParameter}, "required": []string{"path"}}})
	}
	if allowWrite && preferences.allows(workspaceWriteTool) {
		tools = append(tools, map[string]any{
			"name": workspaceWriteTool, "description": "Create or replace a UTF-8 file in the current workspace, after user approval",
			"parameters": map[string]any{"type": "object", "additionalProperties": false,
				"properties": map[string]any{
					"path":    map[string]string{"type": "string", "description": "Workspace-relative file path"},
					"content": map[string]string{"type": "string", "description": "Complete UTF-8 content"},
				}, "required": []string{"path", "content"}},
		})
	}
	builtins := make(map[string]bool, len(tools))
	for _, tool := range tools {
		builtins[tool["name"].(string)] = true
	}
	for _, alias := range slices.Sorted(maps.Keys(connectors)) {
		tool := connectors[alias]
		tools = append(tools, map[string]any{"name": alias,
			"description": tool.ConnectorName + " / " + tool.Name + ": " + tool.Description, "parameters": tool.Schema})
	}
	choice := "auto"
	if requireTool {
		choice = "required"
	}
	if len(tools) == 0 {
		choice = ""
	}
	schemas, err := json.Marshal(tools)
	if err != nil {
		return "", modelUsage{}, nil, err
	}
	round := func(messages []toolConversationMessage, choice string) (modelUsage, toolConversationMessage, string, error) {
		// JSON framing and tool definitions also take space from the input window.
		budget := inputBudget - 256
		if choice != "" {
			budget -= len(schemas)
		}
		messages = fitModelHistory(messages, len(history), budget)
		if protocol == "openai_responses" {
			return c.responsesToolRound(ctx, key, session, messages, tools, choice)
		}
		request := map[string]any{"model": modelID, "messages": messages, "stream": false}
		if choice != "" {
			declared := make([]map[string]any, 0, len(tools))
			for _, tool := range tools {
				declared = append(declared, map[string]any{"type": "function", "function": tool})
			}
			request["tools"] = declared
			request["tool_choice"] = choice
			if maxCalls > 1 {
				request["parallel_tool_calls"] = false
			}
		}
		return c.toolRound(ctx, key, session, request)
	}
	var usage modelUsage
	var runs []workspaceToolRun
	seen := make(map[string]bool, maxCalls)
	for {
		nextUsage, answer, finish, err := round(talk, choice)
		if len(runs) == 0 {
			usage = nextUsage
		} else {
			usage = addToolUsage(usage, nextUsage)
		}
		if err != nil {
			return "", usage, runs, err
		}
		if choice != "required" && finish == "stop" && len(answer.ToolCalls) == 0 && answer.Content != nil && *answer.Content != "" {
			return *answer.Content, usage, runs, nil
		}
		if len(runs) == maxCalls {
			return "", usage, runs, errors.New("model_tool_continuation_incomplete")
		}
		if finish != "tool_calls" || len(answer.ToolCalls) != 1 {
			return "", usage, runs, errors.New("expected_one_model_tool_call")
		}
		invocation := answer.ToolCalls[0]
		args, err := parseModelToolArguments(invocation, approved, allowWrite, connectors, builtins)
		if err != nil || seen[invocation.ID] {
			return "", usage, runs, errors.New("invalid_model_tool_call")
		}
		seen[invocation.ID] = true
		answer.Role = "assistant"
		started := time.Now()
		value, err := call(answer, args, len(runs))
		fact := reportedTool{Category: "builtin", Name: invocation.Function.Name, ResourceID: invocation.Function.Name,
			Calls: 1, DurationMS: time.Since(started).Milliseconds()}
		if args.Connector != nil {
			fact.Category, fact.Name, fact.ResourceID = "connector", args.Connector.Name, args.Connector.ConnectorID
		}
		if err != nil {
			fact.Failed = 1
		}
		runs = append(runs, workspaceToolRun{Call: &invocation, Report: fact, Output: value, ResponseItems: answer.ResponseItems})
		if err != nil {
			return "", usage, runs, err
		}
		talk = append(talk, answer, toolConversationMessage{Role: "tool", Content: &value, ToolCallID: invocation.ID})
		choice = "auto"
		if len(runs) == maxCalls {
			choice = ""
		}
	}
}

// Keep a suffix of complete previous turns. The newest user prompt, its tool
// calls, and their results always travel together on each continuation request.
func fitModelHistory(messages []toolConversationMessage, current, budget int) []toolConversationMessage {
	for _, message := range messages[current:] {
		budget -= modelMessageSize(message)
	}
	start := current
	for index := current - 1; index >= 0 && budget > 0; index-- {
		budget -= modelMessageSize(messages[index])
		if budget >= 0 && messages[index].Role == "user" {
			start = index
		}
	}
	return messages[start:]
}

// Byte counts conservatively approximate input tokens across configured model
// providers when their tokenizer is unavailable locally.
func modelMessageSize(message toolConversationMessage) int {
	size := 24 + len(message.ToolCallID) + len(message.Reasoning)
	if message.Content != nil {
		size += len(*message.Content)
	}
	for _, call := range message.ToolCalls {
		size += 32 + len(call.ID) + len(call.Function.Name) + len(call.Function.Arguments)
	}
	for _, item := range message.ResponseItems {
		size += len(item)
	}
	return size
}

func parseModelToolArguments(invocation modelToolUse, approved []string, allowWrite bool, connectors map[string]connectorTool, builtins map[string]bool) (toolArguments, error) {
	invalid := errors.New("invalid_model_tool_call")
	if invocation.ID == "" || len(invocation.ID) > 256 || invocation.Type != "function" {
		return toolArguments{}, invalid
	}
	if connector, ok := connectors[invocation.Function.Name]; ok {
		raw := json.RawMessage(invocation.Function.Arguments)
		var object map[string]json.RawMessage
		if len(raw) > 48<<10 || json.Unmarshal(raw, &object) != nil || object == nil {
			return toolArguments{}, invalid
		}
		return toolArguments{Connector: &connector, JSON: raw}, nil
	}
	if !builtins[invocation.Function.Name] {
		return toolArguments{}, invalid
	}
	if invocation.Function.Name == designReadTool || designWriteTool(invocation.Function.Name) {
		raw := json.RawMessage(invocation.Function.Arguments)
		if !validDesignToolArguments(invocation.Function.Name, raw) || designWriteTool(invocation.Function.Name) && !allowWrite {
			return toolArguments{}, invalid
		}
		return toolArguments{Design: true, JSON: raw}, nil
	}
	if invocation.Function.Name != workspaceReadTool && invocation.Function.Name != workspaceWriteTool {
		raw := json.RawMessage(invocation.Function.Arguments)
		parsed, err := parseWorkspaceCompatCall(invocation.Function.Name, raw)
		return toolArguments{Compat: parsed, JSON: raw}, err
	}
	workspace, err := parseWorkspaceToolArguments(invocation, approved, allowWrite)
	return toolArguments{Workspace: workspace}, err
}

func parseWorkspaceToolArguments(invocation modelToolUse, approved []string, allowWrite bool) (workspaceToolArguments, error) {
	invalid := errors.New("invalid_model_tool_call")
	var arguments workspaceToolArguments
	parse := json.NewDecoder(bytes.NewBufferString(invocation.Function.Arguments))
	parse.DisallowUnknownFields()
	var extra json.RawMessage
	if invocation.ID == "" || len(invocation.ID) > 256 || invocation.Type != "function" ||
		len(invocation.Function.Arguments) > 8*maxToolText || parse.Decode(&arguments) != nil || parse.Decode(&extra) != io.EOF {
		return arguments, invalid
	}
	if relative, err := workspaceRelative(arguments.Path); err != nil || relative == "." {
		return arguments, invalid
	}
	switch invocation.Function.Name {
	case workspaceReadTool:
		if arguments.Content != nil || len(approved) != 0 && !slices.Contains(approved, arguments.Path) {
			return arguments, invalid
		}
	case workspaceWriteTool:
		if !allowWrite || arguments.Content == nil || len(*arguments.Content) > maxToolText || bytes.ContainsRune([]byte(*arguments.Content), 0) {
			return arguments, invalid
		}
	default:
		return arguments, invalid
	}
	return arguments, nil
}

func addToolUsage(usage, next modelUsage) modelUsage {
	combine := func(a, b *int64) *int64 {
		if a == nil || b == nil || *a < 0 || *b < 0 || *a > 1<<52 || *b > 1<<52 {
			return nil
		}
		total := *a + *b
		return &total
	}
	return modelUsage{Input: combine(usage.Input, next.Input), Output: combine(usage.Output, next.Output)}
}

func (c *client) toolRound(ctx context.Context, key, session string, request map[string]any) (modelUsage, toolConversationMessage, string, error) {
	var result struct {
		Usage   modelUsage
		Choices []struct {
			Finish  string                  `json:"finish_reason"`
			Message toolConversationMessage `json:"message"`
		}
	}
	data, err := json.Marshal(request)
	if err != nil {
		return result.Usage, toolConversationMessage{}, "", err
	}
	resp, err := c.request(ctx, "POST", "/v1/chat/completions", key, "", session, "application/json", bytes.NewReader(data))
	if err != nil {
		return result.Usage, toolConversationMessage{}, "", err
	}
	defer resp.Body.Close()
	if json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&result) != nil || len(result.Choices) != 1 {
		return result.Usage, toolConversationMessage{}, "", errors.New("invalid_tool_model_response")
	}
	return result.Usage, result.Choices[0].Message, result.Choices[0].Finish, nil
}
