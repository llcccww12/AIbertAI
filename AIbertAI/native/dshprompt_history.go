package main

import (
	"bytes"
	"encoding/json"
	"errors"
)

// dshPromptHistory binds one persisted platform operation to its unique DSH
// user message and turn. It never infers ownership from prompt text or timing.
func dshPromptHistory(events []dshConnectorHistoryEvent, operation string) (int64, int64, error) {
	fail := func() (int64, int64, error) { return 0, 0, errors.New("dsh_prompt_history_unverified") }
	if !validUUID(operation) {
		return fail()
	}
	type message struct {
		ID     string `json:"id"`
		Role   string `json:"role"`
		Source struct {
			Kind      string `json:"kind"`
			Operation string `json:"aibertaiOperationId"`
		} `json:"source"`
	}
	var admitted json.RawMessage
	var id string
	var admissionSeq int64
	admissions := 0
	for _, event := range events {
		if event.Seq == nil {
			return fail()
		}
		if event.Type == "aibertai/query-binding" {
			var binding struct {
				OperationID string `json:"operationId"`
			}
			if json.Unmarshal(event.Data, &binding) != nil || binding.OperationID == operation {
				return fail()
			}
		}
		if event.Type != "agent/inbox/spliced" {
			continue
		}
		var splice struct {
			Inserted []json.RawMessage `json:"inserted"`
		}
		if json.Unmarshal(event.Data, &splice) != nil {
			return fail()
		}
		for _, raw := range splice.Inserted {
			var m message
			if json.Unmarshal(raw, &m) != nil {
				return fail()
			}
			if m.Source.Operation != operation {
				continue
			}
			if m.ID == "" || m.Role != "user" || m.Source.Kind != "user" {
				return fail()
			}
			admissions++
			id = m.ID
			admitted = raw
			admissionSeq = *event.Seq
		}
	}
	if admissions != 1 {
		return fail()
	}
	canonical, err := strictDSHJSON(admitted)
	if err != nil {
		return fail()
	}
	var active, last, step, lastStep int64
	var open, stepOpen bool
	var target, userSeq int64
	matches, humans := 0, 0
	for _, event := range events {
		switch event.Type {
		case "turn/start":
			var data struct{ Turn *int64 }
			if json.Unmarshal(event.Data, &data) != nil || data.Turn == nil || *data.Turn <= last || open {
				return fail()
			}
			active = *data.Turn
			last = active
			lastStep = 0
			open = true
		case "turn/end":
			var data struct{ Turn *int64 }
			if json.Unmarshal(event.Data, &data) != nil || data.Turn == nil || !open || stepOpen || *data.Turn != active {
				return fail()
			}
			open = false
		case "step/start":
			var data struct{ Turn, Step *int64 }
			if json.Unmarshal(event.Data, &data) != nil || data.Turn == nil || data.Step == nil || !open || stepOpen || *data.Turn != active || *data.Step <= lastStep {
				return fail()
			}
			step = *data.Step
			lastStep = step
			stepOpen = true
		case "step/end":
			var data struct{ Turn, Step *int64 }
			if json.Unmarshal(event.Data, &data) != nil || data.Turn == nil || data.Step == nil || !open || !stepOpen || *data.Turn != active || *data.Step != step {
				return fail()
			}
			stepOpen = false
		case "tool/call", "tool/result":
			var data struct{ Turn, Step *int64 }
			if json.Unmarshal(event.Data, &data) != nil || data.Turn == nil || data.Step == nil || !open || !stepOpen || *data.Turn != active || *data.Step != step {
				return fail()
			}
		case "user/message":
			var m message
			if json.Unmarshal(event.Data, &m) != nil {
				return fail()
			}
			if m.ID != id && m.Source.Operation != operation {
				continue
			}
			if !open || !stepOpen || m.ID != id || m.Role != "user" || m.Source.Kind != "user" || m.Source.Operation != operation || *event.Seq <= admissionSeq {
				return fail()
			}
			actual, err := strictDSHJSON(event.Data)
			if err != nil || !bytes.Equal(actual, canonical) {
				return fail()
			}
			matches++
			target = active
			userSeq = *event.Seq
		}
	}
	if matches != 1 || target < 1 {
		return fail()
	}
	active = 0
	for _, event := range events {
		if event.Type == "turn/start" {
			var data struct{ Turn int64 }
			_ = json.Unmarshal(event.Data, &data)
			active = data.Turn
		}
		if event.Type == "user/message" && active == target {
			var m message
			if json.Unmarshal(event.Data, &m) != nil {
				return fail()
			}
			if m.Source.Kind == "user" {
				humans++
			}
		}
	}
	if humans != 1 {
		return fail()
	}
	return target, userSeq, nil
}

// dshRebuildPromptTurn reconstructs one interrupted ordinary turn from a
// validated fixed history. verifyWrite must prove every successful mutation;
// this function never dispatches a tool or changes the original turn.
func dshRebuildPromptTurn(session string, original acpTurn, events []dshConnectorHistoryEvent, rawEvents []json.RawMessage, verifyWrite func(workspaceToolRun) error) (acpTurn, int64, error) {
	fail := func() (acpTurn, int64, error) { return acpTurn{}, 0, errors.New("dsh_prompt_history_unverified") }
	if len(events) == 0 || len(events) != len(rawEvents) || verifyWrite == nil {
		return fail()
	}
	target, userSeq, err := dshPromptHistory(events, original.Operation)
	if err != nil {
		return fail()
	}
	rebuilt := original
	rebuilt.ToolRuns = nil
	rebuilt.DSHTodos = nil
	rebuilt.DSHCompactions = nil
	rebuilt.DSHDeliverables = nil
	rebuilt.Answer = ""
	rebuilt.DSHParts = nil
	rebuilt.DSHStreamed = false
	var answer string
	var ended bool
	var lastSeq int64 = -1
	for i, event := range events {
		if event.Seq == nil || *event.Seq != int64(i) {
			return fail()
		}
		lastSeq = *event.Seq
		if event.Type == "turn/start" && *event.Seq > userSeq {
			var d struct{ Turn int64 }
			if json.Unmarshal(event.Data, &d) != nil || d.Turn != target {
				return fail()
			}
		}
		if *event.Seq <= userSeq {
			continue
		}
		switch event.Type {
		case "turn/start", "todo/write":
			rebuilt.applyDSHTodo(event)
		case "compaction/start", "compaction/end":
			rebuilt.applyDSHCompaction(event, target)
		case "deliverables/presented":
			rebuilt.recordDeliverables(event.Data, target)
		case "assistant/message":
			var d struct {
				Turn    int64 `json:"turn"`
				Message struct {
					Content []struct{ Type, Text string } `json:"content"`
				} `json:"message"`
			}
			if json.Unmarshal(event.Data, &d) != nil {
				return fail()
			}
			if d.Turn != target {
				continue
			}
			answer = ""
			rebuilt.DSHParts = nil
			hasThought := false
			for _, block := range d.Message.Content {
				hasThought = hasThought || block.Type == "reasoning"
			}
			for _, block := range d.Message.Content {
				if block.Type == "text" {
					answer += block.Text
				}
				if hasThought && block.Type == "text" {
					rebuilt.DSHParts = append(rebuilt.DSHParts, dshMessagePart{Type: "text", End: len(answer)})
				} else if block.Type == "reasoning" {
					rebuilt.DSHParts = append(rebuilt.DSHParts, dshMessagePart{Type: "thought", Text: block.Text})
				}
			}
			rebuilt.Answer = answer
		case "assistant/attempt":
			var d struct{ Turn int64 }
			if json.Unmarshal(event.Data, &d) != nil {
				return fail()
			}
			if d.Turn == target {
				answer = ""
				rebuilt.Answer = ""
				rebuilt.DSHParts = nil
			}
		case "tool/call", "tool/result":
			var d struct{ Turn int64 }
			if json.Unmarshal(event.Data, &d) != nil {
				return fail()
			}
			if d.Turn != target {
				continue
			}
			envelope, err := json.Marshal(map[string]any{"sessionId": session, "event": rawEvents[i]})
			if err != nil {
				return fail()
			}
			previous := len(rebuilt.ToolRuns)
			update, err := projectDSHTool(&rebuilt, envelope)
			if err != nil || update == nil {
				return fail()
			}
			if event.Type == "tool/call" && len(rebuilt.ToolRuns) > previous {
				run := &rebuilt.ToolRuns[len(rebuilt.ToolRuns)-1]
				if run.Call == nil || run.Call.Function.Name != "write" && run.Call.Function.Name != "edit" && !dshRecoverableCanvasTool(run.Call.Function.Name) {
					return fail()
				}
				if answer != "" || len(rebuilt.DSHParts) != 0 {
					run.Preamble = answer
					run.DSHParts = rebuilt.DSHParts
					answer = ""
					rebuilt.Answer = ""
					rebuilt.DSHParts = nil
				}
				for _, old := range original.ToolRuns {
					if old.DSHCallID == run.DSHCallID {
						run.DisplayName = old.DisplayName
						run.DisplayKind = old.DisplayKind
						run.Report.Category = old.Report.Category
					}
				}
			}
		case "turn/end":
			var d struct{ Turn int64 }
			if json.Unmarshal(event.Data, &d) != nil {
				return fail()
			}
			if d.Turn == target {
				ended = true
			}
		}
	}
	if !ended {
		return fail()
	}
	for _, run := range rebuilt.ToolRuns {
		if run.Pending {
			return fail()
		}
		if run.Report.Failed == 0 {
			if err := verifyWrite(run); err != nil {
				return fail()
			}
		}
	}
	return rebuilt, lastSeq, nil
}
