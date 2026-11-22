package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// The synthetic model reports the tools offered by the real SDK for this call.
// Task content on lookup must come from a tool reply in the current turn.
func dshFixtureTeamSwitch(request map[string]any, sequence int64) (map[string]any, string) {
	lastUser, lastResult := "", ""
	messages, _ := request["messages"].([]any)
	for _, raw := range messages {
		message, _ := raw.(map[string]any)
		value, _ := json.Marshal(message["content"])
		switch message["role"] {
		case "user":
			// DSH injects time context as another user-role message. Only a
			// fixture prompt starts a new synthetic scenario.
			if input := string(value); strings.Contains(input, "保存任务") || strings.Contains(input, "找回任务") || strings.Contains(input, "核对") {
				lastUser, lastResult = input, ""
			}
		case "tool":
			lastResult = string(value)
		}
	}
	tools, _ := request["tools"].([]any)
	enabled := false
	for _, raw := range tools {
		tool, _ := raw.(map[string]any)
		fn, _ := tool["function"].(map[string]any)
		if fn["name"] == "team_task_create" {
			enabled = true
		}
	}
	answer := "模型未收到共享任务工具。"
	if enabled {
		answer = "模型收到了共享任务工具。"
		tool, args := "", "{}"
		if strings.Contains(lastUser, "保存任务") {
			tool, args = "team_task_create", `{"subject":"switch-persisted","description":"shared board acceptance"}`
		} else if strings.Contains(lastUser, "找回任务") {
			tool = "team_task_list"
		}
		if lastResult == "" && tool != "" {
			call := map[string]any{"index": 0, "id": fmt.Sprintf("switch-%d", sequence), "type": "function", "function": map[string]string{"name": tool, "arguments": args}}
			return map[string]any{"role": "assistant", "tool_calls": []any{call}}, tool
		}
		if tool == "team_task_list" && strings.Contains(lastResult, "switch-persisted") {
			answer = "已从真实任务板找到原任务。"
		}
	}
	return map[string]any{"role": "assistant", "content": answer}, ""
}
