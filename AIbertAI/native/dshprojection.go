package main

import (
	"encoding/json"
	"fmt"
	"slices"
)

// DSH call IDs correlate its events; platform IDs stay stable in live and
// history views. A result is never invented for a missing or incomplete call.
func projectDSHTool(turn *acpTurn, raw json.RawMessage) (map[string]any, error) {
	var e struct {
		Event struct {
			Type      string              `json:"type"`
			SurfaceOp dshHistorySurfaceOp `json:"surfaceOp"`
			Data      struct {
				CallID    string          `json:"callId"`
				Name      string          `json:"name"`
				Arguments string          `json:"arguments"`
				Meta      json.RawMessage `json:"meta"`
				Message   struct {
					IsError    bool   `json:"isError"`
					ToolCallID string `json:"toolCallId"`
					Source     struct {
						CallID string `json:"callId"`
					} `json:"source"`
					Content []struct {
						Type       string          `json:"type"`
						Text       string          `json:"text"`
						ToolCallID string          `json:"toolCallId"`
						IsError    bool            `json:"isError"`
						Content    json.RawMessage `json:"content"`
					} `json:"content"`
				} `json:"message"`
				Error *struct {
					Reason string `json:"reason"`
				} `json:"error"`
			} `json:"data"`
		} `json:"event"`
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil, err
	}
	if e.Event.Type == "tool/result" && e.Event.SurfaceOp.Op == "replace" {
		return nil, nil
	}
	d := e.Event.Data
	if e.Event.Type == "tool/call" {
		if d.CallID == "" || d.Name == "" {
			return nil, fmt.Errorf("invalid_dsh_tool_call")
		}
		for _, r := range turn.ToolRuns {
			if r.DSHCallID == d.CallID {
				return nil, nil
			}
		}
		call := &modelToolUse{ID: fmt.Sprintf("%s-tool-%d", turn.Operation, len(turn.ToolRuns)+1), Type: "function"}
		call.Function.Name = d.Name
		call.Function.Arguments = d.Arguments
		category := "builtin"
		if dshAgentTools[d.Name] {
			category = "agent"
		}
		turn.ToolRuns = append(turn.ToolRuns, workspaceToolRun{DSHCallID: d.CallID, Pending: true, Call: call, Report: reportedTool{Category: category, Name: d.Name, ResourceID: d.Name, Calls: 1}})
		var input any
		if json.Unmarshal([]byte(d.Arguments), &input) != nil {
			input = d.Arguments
		}
		return map[string]any{"sessionUpdate": "tool_call", "toolCallId": call.ID, "name": d.Name, "title": d.Name, "kind": workspaceToolKind(d.Name), "status": "in_progress", "rawInput": input, "_meta": map[string]string{"dshCallId": d.CallID}}, nil
	}
	if e.Event.Type != "tool/result" {
		return nil, nil
	}
	id := d.Message.ToolCallID
	if id == "" {
		id = d.Message.Source.CallID
	} else if d.Message.Source.CallID != "" && d.Message.Source.CallID != id {
		return nil, nil
	}
	for _, b := range d.Message.Content {
		if b.Type != "tool-result" || b.ToolCallID == "" {
			continue
		}
		if id == "" {
			id = b.ToolCallID
		} else if b.ToolCallID != id {
			return nil, nil
		}
	}
	if id == "" {
		return nil, nil
	}
	turn.ToolRuns = slices.Clone(turn.ToolRuns)
	for i := range turn.ToolRuns {
		r := &turn.ToolRuns[i]
		if r.DSHCallID != id || !r.Pending {
			continue
		}
		r.Pending = false
		if d.Message.IsError {
			r.Report.Failed = 1
		}
		for _, b := range d.Message.Content {
			if b.Type == "text" {
				r.Output += b.Text
				continue
			}
			if b.Type != "tool-result" {
				continue
			}
			if b.IsError {
				r.Report.Failed = 1
			}
			var blocks []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if json.Unmarshal(b.Content, &blocks) == nil {
				for _, block := range blocks {
					if block.Type == "text" {
						r.Output += block.Text
					}
				}
			} else {
				r.Output = string(b.Content)
			}
		}
		if d.Error != nil {
			r.Report.Failed = 1
			if r.Output == "" {
				r.Output = d.Error.Reason
			}
		}
		status := "completed"
		if r.Report.Failed > 0 {
			status = "failed"
		}
		update := map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": r.Call.ID, "name": r.Call.Function.Name, "status": status, "rawOutput": r.Output}
		if r.Report.Failed == 0 {
			if meta := dshToolViewMeta(r.Call.Function.Name, d.Meta); meta != nil {
				r.DSHResultMeta = meta
				update["_meta"] = map[string]any{"dshResultMeta": meta}
			}
		}
		return update, nil
	}
	return nil, nil
}

// DSH subagent and agent-team tools delegate work to other agents.
var dshAgentTools = map[string]bool{
	"subagent": true, "subagent_fork": true, "list_subagent_models": true,
	"send_message": true, "wait_agent": true, "interrupt_agent": true, "list_agents": true,
	"team_task_create": true, "team_task_get": true, "team_task_list": true, "team_task_update": true,
}

// These fields belong to DSH's display cards. Private connector or plugin
// metadata stays outside the employee transcript and its persisted history.
var dshToolViewFields = map[string][]string{
	"edit": {"path", "diffs"}, "write": {"path", "diffs"}, "apply_patch": {"diffs", "stopped"},
	"read": {"path", "offset", "lines", "totalLines", "lang"},
	"glob": {"shape", "truncated", "total", "paths"}, "grep": {"shape", "truncated", "total", "files"},
	"bash": {"cwd"}, "pwsh": {"cwd"},
	"web_fetch": {"url", "statusCode", "truncated"}, "web_search": {"sources", "answer", "truncated"},
	"subagent": {"kind", "name", "subagentId"}, "subagent_fork": {"kind", "name", "subagentId"},
}

func dshToolViewMeta(name string, raw json.RawMessage) json.RawMessage {
	keys := dshToolViewFields[name]
	if len(keys) == 0 || len(raw) == 0 {
		return nil
	}
	var recorded map[string]json.RawMessage
	if json.Unmarshal(raw, &recorded) != nil {
		return nil
	}
	selected := make(map[string]json.RawMessage)
	for _, key := range keys {
		if value, ok := recorded[key]; ok {
			selected[key] = value
		}
	}
	if len(selected) == 0 {
		return nil
	}
	result, _ := json.Marshal(selected)
	return result
}
