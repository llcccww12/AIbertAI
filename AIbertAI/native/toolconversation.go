package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"slices"
	"strings"
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
	Workspace   workspaceToolArguments
	Connector   *connectorTool
	Compat      *workspaceCompatCall
	Design      bool
	SkillScript *skillScriptCall
	JSON        json.RawMessage
}

type workspaceToolRun struct {
	DSHResultMeta json.RawMessage   `json:"dshResultMeta,omitempty"`
	DSHCallID     string            `json:"dshCallId,omitempty"`
	DisplayName   string            `json:"displayName,omitempty"`
	DisplayKind   string            `json:"displayKind,omitempty"`
	Pending       bool              `json:"pending,omitempty"`
	Preamble      string            `json:"preamble,omitempty"`
	DSHParts      []dshMessagePart  `json:"dshParts,omitempty"`
	Call          *modelToolUse     `json:"call,omitempty"`
	Report        reportedTool      `json:"report"`
	Output        string            `json:"output,omitempty"`
	ResponseItems []json.RawMessage `json:"responseItems,omitempty"`

	dshSourceInvocation string
	dshStreamAttempt    string
	DSHThoughtProof     *dshGoalThoughtProof `json:"dshThoughtProof,omitempty"`
}

func (c *client) completeToolTurn(ctx context.Context, protocol, key, session string, approved []string, prompt string, history []toolConversationMessage, requireTool, allowWrite bool, call func(modelToolUse, workspaceToolArguments, []json.RawMessage) (string, error)) (string, modelUsage, *reportedTool, error) {
	answer, usage, runs, err := c.completeWorkspaceToolTurn(ctx, protocol, key, session, modelID, approved, prompt, history, requireTool, allowWrite, 1, nil, nil, defaultModelInputBudget, nil,
		func(message toolConversationMessage, args toolArguments, _ int) (string, error) {
			return call(message.ToolCalls[0], args.Workspace, message.ResponseItems)
		})
	if len(runs) == 0 {
		return answer, usage, nil, err
	}
	return answer, usage, &runs[0].Report, err
}

func (c *client) completeWorkspaceToolTurn(ctx context.Context, protocol, key, session, selected string, approved []string, prompt string, history []toolConversationMessage, requireTool, allowWrite bool, maxCalls int, connectors map[string]connectorTool, workspace *workspaceToolPolicy, inputBudget int, modelOptions map[string]any, call func(toolConversationMessage, toolArguments, int) (string, error), deltas ...func(string) error) (string, modelUsage, []workspaceToolRun, error) {
	talk := append(slices.Clone(history), toolConversationMessage{Role: "user", Content: &prompt})
	fileParameter := map[string]any{"type": "string", "description": "A workspace-relative path or the exact name of a file attached to this turn"}
	if len(approved) != 0 {
		fileParameter["enum"] = approved
	}
	var preferences builtinToolPreferences
	var instructions string
	var goal *desktopGoalRun
	if workspace != nil {
		preferences = workspace.Preferences
		instructions = workspace.Instructions
		goal = workspace.Goal
	}
	var delta func(string) error
	if len(deltas) != 0 && goal == nil {
		delta = deltas[0]
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
		if instructions != "" {
			system := toolConversationMessage{Role: "system", Content: &instructions}
			messages = fitModelHistory(messages, len(history), budget-modelMessageSize(system))
			messages = append([]toolConversationMessage{system}, messages...)
		} else {
			messages = fitModelHistory(messages, len(history), budget)
		}
		if protocol == "openai_responses" {
			return c.responsesToolRound(ctx, key, session, selected, messages, tools, choice, modelOptions, delta)
		}
		request := map[string]any{"model": selected, "messages": messages, "stream": false}
		maps.Copy(request, modelOptions)
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
		if delta != nil {
			request["stream"] = true
			request["stream_options"] = map[string]bool{"include_usage": true}
		}
		return c.toolRound(ctx, key, session, request, delta)
	}
	var usage modelUsage
	var runs []workspaceToolRun
	seen := make(map[string]bool, maxCalls)
	checkpoints := 0
	lastProgress := "目标已暂停。"
	if goal != nil && goal.save != nil {
		lastProgress = ""
	}
	var checkpoint goalCheckpoint
	var recovered []workspaceToolRun
	promptEnd := len(talk)
	saveProgress := func(pending *toolConversationMessage) error {
		if goal == nil || goal.save == nil {
			return nil
		}
		checkpoint.Runs, checkpoint.Usage, checkpoint.Progress = runs, usage, lastProgress
		checkpoint.Pending = nil
		if pending != nil {
			checkpoint.Pending = &goalReplayMessage{*pending, pending.ResponseItems}
		}
		checkpoint.Tail = make([]goalReplayMessage, 0, len(talk)-promptEnd)
		for _, message := range talk[promptEnd:] {
			checkpoint.Tail = append(checkpoint.Tail, goalReplayMessage{message, message.ResponseItems})
		}
		return goal.save(checkpoint)
	}
	if goal != nil && goal.checkpoint != nil {
		checkpoint = *goal.checkpoint
		checkpoint.Unknown = slices.Clone(checkpoint.Unknown)
		runs, usage, lastProgress = slices.Clone(checkpoint.Runs), checkpoint.Usage, checkpoint.Progress
		recovered = slices.Clone(runs)
		for _, saved := range checkpoint.Tail {
			message := saved.toolConversationMessage
			message.ResponseItems = saved.Items
			talk = append(talk, message)
			for _, invocation := range message.ToolCalls {
				seen[invocation.ID] = true
			}
		}
		if pending := checkpoint.Pending; pending != nil {
			if len(pending.ToolCalls) != 1 {
				return lastProgress, usage, runs, errors.New("invalid_goal_checkpoint")
			}
			message := pending.toolConversationMessage
			message.ResponseItems = pending.Items
			invocation := message.ToolCalls[0]
			seen[invocation.ID] = true
			checkpoint.Unknown = append(checkpoint.Unknown, invocation)
			status := "This tool call was interrupted after submission and its outcome is unknown. Do not repeat it. Verify its result independently before proceeding."
			talk = append(talk, message, toolConversationMessage{Role: "tool", Content: &status, ToolCallID: invocation.ID})
			if err := saveProgress(nil); err != nil {
				return lastProgress, usage, runs, err
			}
		}
		if len(runs) >= maxCalls {
			choice = ""
		}
	}
	for roundIndex := 0; ; roundIndex++ {
		if goal == nil && workspace != nil && workspace.SlideDeck && len(runs) >= maxCalls {
			return "", usage, runs, errors.New("model_tool_continuation_incomplete")
		}
		if goal != nil && !goal.active() {
			goal.outcome = "user_paused"
			return lastProgress, usage, runs, nil
		}
		if goal != nil && goal.save != nil && checkpoint.Rounds >= maxBackgroundGoalRounds {
			goal.outcome = "auto_turn_budget"
			return lastProgress, usage, runs, nil
		}
		nextUsage, answer, finish, err := round(talk, choice)
		if err == nil || nextUsage.Input != nil || nextUsage.Output != nil {
			if roundIndex == 0 && checkpoint.Rounds == 0 {
				usage = nextUsage
			} else {
				usage = addToolUsage(usage, nextUsage)
			}
		}
		partial := ""
		if delta != nil && answer.Content != nil {
			partial = *answer.Content
		}
		if err != nil {
			if goal != nil && goal.save != nil {
				return lastProgress, usage, runs, err
			}
			return partial, usage, runs, err
		}
		checkpoint.Rounds++
		if choice != "required" && finish == "stop" && len(answer.ToolCalls) == 0 && answer.Content != nil && *answer.Content != "" {
			progress := strings.TrimSpace(*answer.Content)
			if goal == nil {
				return *answer.Content, usage, runs, nil
			}
			const complete = "\n[GOAL_COMPLETE]"
			if !goal.active() {
				goal.outcome = "user_paused"
				return strings.TrimSuffix(progress, complete), usage, runs, nil
			}
			if strings.HasSuffix(progress, complete) {
				goal.outcome = "completed"
				return strings.TrimSuffix(progress, complete), usage, runs, nil
			}
			lastProgress = progress
			if goal.stop(usage) {
				return lastProgress, usage, runs, nil
			}
			checkpoints++
			if goal.save == nil && checkpoints >= 3 {
				goal.outcome = "auto_turn_budget"
				return lastProgress, usage, runs, nil
			}
			next := goalContinuePrompt
			answer.Role = "assistant"
			talk = append(talk, answer, toolConversationMessage{Role: "user", Content: &next})
			if err := saveProgress(nil); err != nil {
				return lastProgress, usage, runs, err
			}
			continue
		}
		if goal != nil && goal.stop(usage) {
			return lastProgress, usage, runs, nil
		}
		if len(runs) == maxCalls {
			return partial, usage, runs, errors.New("model_tool_continuation_incomplete")
		}
		if finish != "tool_calls" || len(answer.ToolCalls) != 1 {
			return partial, usage, runs, errors.New("expected_one_model_tool_call")
		}
		invocation := answer.ToolCalls[0]
		args, err := parseModelToolArguments(invocation, approved, allowWrite, connectors, builtins)
		if err != nil || seen[invocation.ID] {
			return partial, usage, runs, errors.New("invalid_model_tool_call")
		}
		seen[invocation.ID] = true
		answer.Role = "assistant"
		if goal != nil && goal.save != nil {
			var recorded *string
			for _, unknown := range checkpoint.Unknown {
				if sameGoalAction(unknown, invocation) {
					recorded = new("The outcome of this submitted action is unknown. The runtime did not submit it again.")
					break
				}
			}
			if recorded == nil && !safeGoalRead(invocation.Function.Name) {
				for _, previous := range recovered {
					if previous.Call != nil && previous.Report.Failed == 0 && sameGoalAction(*previous.Call, invocation) {
						recorded = &previous.Output
						break
					}
				}
			}
			if recorded != nil {
				talk = append(talk, answer, toolConversationMessage{Role: "tool", Content: recorded, ToolCallID: invocation.ID})
				if err := saveProgress(nil); err != nil {
					return lastProgress, usage, runs, err
				}
				continue
			}
			var pending *toolConversationMessage
			if !safeGoalRead(invocation.Function.Name) {
				pending = &answer
			}
			if err := saveProgress(pending); err != nil {
				return lastProgress, usage, runs, err
			}
		}
		started := time.Now()
		value, err := call(answer, args, len(runs))
		if err != nil && goal != nil && goal.save != nil && !safeGoalRead(invocation.Function.Name) &&
			(ctx.Err() != nil || strings.Contains(err.Error(), "already_submitted")) {
			goal.outcome = "resume_confirmation"
			return lastProgress, usage, runs, err
		}
		fact := reportedTool{Category: "builtin", Name: invocation.Function.Name, ResourceID: invocation.Function.Name,
			Calls: 1, DurationMS: time.Since(started).Milliseconds()}
		if args.Connector != nil {
			fact.Category, fact.Name, fact.ResourceID = "connector", args.Connector.Name, args.Connector.ConnectorID
			if args.Connector.Local {
				fact.Category, fact.Name = "local_mcp", args.Connector.ConnectorName
			}
		}
		if err != nil {
			fact.Failed = 1
			if !requireTool && ctx.Err() == nil && err.Error() != "tool_permission_cancelled" && err.Error() != "tool_approval_unavailable" {
				value = "Tool failed (" + safeDesktopTurnFailure(err) + "). Use the observed result to correct the request, choose another suitable action, or explain the failure."
			}
		}
		runs = append(runs, workspaceToolRun{Preamble: partial, Call: &invocation, Report: fact, Output: value, ResponseItems: answer.ResponseItems})
		if err != nil && (err.Error() == "tool_permission_cancelled" || err.Error() == "tool_approval_unavailable") && goal != nil && goal.save != nil {
			goal.outcome = "permission_denied"
			// No action happened. Resume from the same model checkpoint when an operator is available.
			runs = runs[:len(runs)-1]
			if saveErr := saveProgress(nil); saveErr != nil {
				return lastProgress, usage, runs, saveErr
			}
			return lastProgress, usage, runs, err
		}
		talk = append(talk, answer, toolConversationMessage{Role: "tool", Content: &value, ToolCallID: invocation.ID})
		if saveErr := saveProgress(nil); saveErr != nil {
			return lastProgress, usage, runs, saveErr
		}
		if err != nil && (requireTool || ctx.Err() != nil || err.Error() == "tool_permission_cancelled" || err.Error() == "tool_approval_unavailable") {
			return "", usage, runs, err
		}
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
	if invocation.Function.Name == skillScriptTool {
		raw := json.RawMessage(invocation.Function.Arguments)
		var script skillScriptCall
		if len(raw) > 16<<10 || decodeWorkspaceTool(raw, &script) != nil || !validSkillScript(script) {
			return toolArguments{}, invalid
		}
		return toolArguments{SkillScript: &script, JSON: raw}, nil
	}
	if designTool(invocation.Function.Name) {
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

func (c *client) toolRound(ctx context.Context, key, session string, request map[string]any, deltas ...func(string) error) (modelUsage, toolConversationMessage, string, error) {
	resp, err := c.requestModelRound(ctx, "/v1/chat/completions", key, session, request)
	if err != nil {
		return modelUsage{}, toolConversationMessage{}, "", err
	}
	defer resp.Body.Close()
	var delta func(string) error
	if len(deltas) != 0 {
		delta = deltas[0]
	}
	if delta != nil && strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		return readChatToolStream(resp.Body, delta)
	}
	usage, message, finish, err := readChatToolResponse(resp.Body)
	if err == nil && delta != nil && message.Content != nil && *message.Content != "" {
		err = delta(*message.Content)
	}
	return usage, message, finish, err
}

func readChatToolResponse(body io.Reader) (modelUsage, toolConversationMessage, string, error) {
	var result struct {
		Usage   modelUsage
		Choices []struct {
			Finish  string                  `json:"finish_reason"`
			Message toolConversationMessage `json:"message"`
		}
	}
	if json.NewDecoder(io.LimitReader(body, 2<<20)).Decode(&result) != nil || len(result.Choices) != 1 {
		return result.Usage, toolConversationMessage{}, "", errors.New("invalid_tool_model_response")
	}
	return result.Usage, result.Choices[0].Message, result.Choices[0].Finish, nil
}
