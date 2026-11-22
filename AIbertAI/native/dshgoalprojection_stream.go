package main

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
)

// scanDSHGoalPages consumes a fixed durable prefix without retaining pages.
// The caller owns all state derived by visit and must not publish it before
// this function returns successfully.
func scanDSHGoalPages(ctx context.Context, session string, fetch dshConnectorHistoryFetch, visit func(dshConnectorHistoryEvent, json.RawMessage) error) error {
	if !validUUID(session) || fetch == nil || visit == nil {
		return errDSHInvalidHistory
	}
	after := int64(-1)
	var ceiling *int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		page, _, err := dshFetchHistoryPage(ctx, session, after, ceiling, fetch)
		if err != nil {
			return err
		}
		if ceiling == nil {
			value := *page.Through
			ceiling = &value
		}
		previous := after
		for _, rawEvent := range page.Events {
			var e dshConnectorHistoryEvent
			if (len(rawEvent) > 512<<10 && page.Large == nil) || json.Unmarshal(rawEvent, &e) != nil || e.Seq == nil || *e.Seq != previous+1 || *e.Seq > *ceiling || e.Type == "" || len(e.Data) == 0 {
				return errDSHInvalidHistory
			}
			previous = *e.Seq
		}
		if previous < *ceiling && len(page.Events) == 0 {
			return errDSHInvalidHistory
		}
		for _, rawEvent := range page.Events {
			var e dshConnectorHistoryEvent
			_ = json.Unmarshal(rawEvent, &e)
			if err := visit(e, rawEvent); err != nil {
				return err
			}
			after = *e.Seq
		}
		if after == *ceiling {
			return nil
		}
		if after == int64(^uint64(0)>>1) {
			return errors.New("dsh_invalid_history")
		}
	}
}

type dshGoalReadRecord struct {
	id                 string
	callSeq, resultSeq int64
	call, result       json.RawMessage
}

type dshGoalReadPlan struct {
	records                         []dshGoalReadRecord
	needsRepair, checkpointMismatch bool
}

// collectDSHGoalReads validates the binding and target turns in one pass. It
// retains only target read pairs, bounded by the desktop session capacity.
func collectDSHGoalReads(ctx context.Context, row acpSession, original acpTurn, fetch dshConnectorHistoryFetch) (dshGoalReadPlan, error) {
	fail := errDSHGoalProjectionUnverified
	goal := row.Goal
	plan := dshGoalReadPlan{checkpointMismatch: len(original.ToolRuns) != len(original.GoalCheckpoint.Runs) || len(original.ToolRuns) > 0 && !reflect.DeepEqual(original.ToolRuns, original.GoalCheckpoint.Runs)}
	plan.needsRepair = plan.checkpointMismatch
	existing := make(map[string]bool, len(original.ToolRuns))
	invalid, conflict := false, false
	for _, run := range original.ToolRuns {
		if run.DSHCallID == "" || existing[run.DSHCallID] {
			invalid = true
		}
		if run.Pending {
			plan.needsRepair = true
		}
		existing[run.DSHCallID] = true
	}
	commandSeq, goalSeq, doneSeq, bindingSeq := int64(-1), int64(-1), int64(-1), int64(-1)
	var goalRevision int64
	var latestRevision int64
	var latestPhase string
	var lastTurn, turn, lastStep, step int64
	var turnOpen, stepOpen, target, human, preTargetActivity, changedGoal bool
	var lastRound int
	var invocation, invocationPurpose string
	var assistantSeen, preBindingActivity bool
	var observedSettled bool
	type declaration struct{ name, arguments string }
	declared := map[string]declaration{}
	seenInvocation := map[string]bool{}
	priorInvocation := map[string]bool{}
	seenCalls := map[string]bool{}
	pending := map[string]int{}
	retained := 0
	tooLarge := false
	err := scanDSHGoalPages(ctx, row.DSHSessionID, fetch, func(e dshConnectorHistoryEvent, raw json.RawMessage) error {
		seq := *e.Seq
		if goalSeq >= 0 && bindingSeq < 0 && (e.Type == "turn/start" || e.Type == "step/start" || e.Type == "tool/call") {
			preBindingActivity = true
		}
		switch e.Type {
		case "command/run":
			var d struct{ CommandID, Name string }
			if json.Unmarshal(e.Data, &d) != nil {
				return fail
			}
			if d.CommandID == goal.DSHCommandID {
				if commandSeq >= 0 || d.Name != "goal" {
					return fail
				}
				commandSeq = seq
			}
		case "goal/change":
			var d struct {
				Operation string
				Goal      struct {
					ID, Phase string
					Revision  int64
				}
				Cleared struct{ ID string }
			}
			if json.Unmarshal(e.Data, &d) != nil {
				return fail
			}
			if bindingSeq < 0 && commandSeq >= 0 && (d.Goal.ID == goal.DSHGoalID || d.Cleared.ID == goal.DSHGoalID) {
				if goalSeq >= 0 || (d.Operation != "create" && d.Operation != "resume") || d.Goal.Phase != "active" || d.Goal.Revision < 1 {
					return fail
				}
				goalSeq, goalRevision, latestRevision, latestPhase = seq, d.Goal.Revision, d.Goal.Revision, d.Goal.Phase
			} else if bindingSeq >= 0 && (d.Goal.ID == goal.DSHGoalID || d.Cleared.ID == goal.DSHGoalID) {
				if d.Cleared.ID == goal.DSHGoalID || d.Goal.Revision < latestRevision || d.Goal.Revision > goal.DSHRevision || seq > goal.DSHEventSeq {
					return fail
				}
				switch d.Operation {
				case "edit":
				case "resume":
					if d.Goal.Phase != "active" {
						return fail
					}
				case "pause":
					if d.Goal.Phase != "paused" {
						return fail
					}
				case "block":
					if d.Goal.Phase != "blocked" {
						return fail
					}
				case "complete":
					if d.Goal.Phase != "complete" {
						return fail
					}
				default:
					return fail
				}
				latestRevision, latestPhase = d.Goal.Revision, d.Goal.Phase
				if d.Goal.Revision != goalRevision || d.Goal.Phase != "active" {
					changedGoal = true
				}
			}
		case "command/done":
			var d struct{ CommandID, Kind string }
			if json.Unmarshal(e.Data, &d) != nil {
				return fail
			}
			if d.CommandID == goal.DSHCommandID {
				if doneSeq >= 0 || d.Kind != "success" || goalSeq < 0 || seq <= goalSeq {
					return fail
				}
				doneSeq = seq
			}
		case "aibertai/query-binding":
			var b dshGoalQueryBinding
			if json.Unmarshal(e.Data, &b) != nil {
				return fail
			}
			if b.OperationID == goal.DSHQueryOp || b.CommandID == goal.DSHCommandID {
				if bindingSeq >= 0 || preBindingActivity || b.OperationID != goal.DSHQueryOp || b.CommandID != goal.DSHCommandID || b.GoalID != goal.DSHGoalID || b.GoalRevision != goalRevision || b.GoalRevision > goal.DSHRevision || b.CommandRunSeq != commandSeq || b.GoalSeq != goalSeq || commandSeq < 0 || goalSeq <= commandSeq || doneSeq <= goalSeq || seq <= doneSeq {
					return fail
				}
				bindingSeq = seq
			} else if bindingSeq >= 0 {
				return fail
			}
		}
		if bindingSeq < 0 || seq <= bindingSeq {
			if e.Type == "aibertai/invocation" || e.Type == "assistant/message" {
				var prior struct{ InvocationID string }
				if json.Unmarshal(e.Data, &prior) != nil {
					return fail
				}
				if _, ok := goal.Invocations[prior.InvocationID]; ok {
					priorInvocation[prior.InvocationID] = true
				}
			}
			return nil
		}
		if turnOpen && !target && (e.Type == "aibertai/invocation" || e.Type == "assistant/message" || e.Type == "tool/call" || e.Type == "tool/result") {
			preTargetActivity = true
		}
		switch e.Type {
		case "turn/start":
			var d struct{ Turn int64 }
			if json.Unmarshal(e.Data, &d) != nil || turnOpen || d.Turn <= lastTurn {
				return fail
			}
			turn, lastTurn, lastStep, turnOpen, target, human, preTargetActivity = d.Turn, d.Turn, 0, true, false, false, false
		case "turn/end":
			var d struct{ Turn int64 }
			if json.Unmarshal(e.Data, &d) != nil || !turnOpen || stepOpen || d.Turn != turn {
				return fail
			}
			turnOpen, target = false, false
		case "step/start":
			var d struct{ Turn, Step int64 }
			if json.Unmarshal(e.Data, &d) != nil || !turnOpen || stepOpen || d.Turn != turn || d.Step <= lastStep {
				return fail
			}
			step, lastStep, stepOpen, invocation, invocationPurpose, assistantSeen = d.Step, d.Step, true, "", "", false
			declared = map[string]declaration{}
		case "step/end":
			var d struct{ Turn, Step int64 }
			if json.Unmarshal(e.Data, &d) != nil || !turnOpen || !stepOpen || d.Turn != turn || d.Step != step {
				return fail
			}
			if target && (len(declared) != 0 || len(pending) != 0) {
				invalid = true
				if len(pending) != 0 {
					conflict = true
				}
			}
			stepOpen = false
		case "user/message":
			var d struct {
				ID, Role string
				Source   struct {
					Kind, GoalID string
					Revision     int64
					Round        int
				}
			}
			if json.Unmarshal(e.Data, &d) != nil {
				return fail
			}
			if d.Source.Kind == "user" && turnOpen {
				human = true
				if target {
					conflict = true
				}
			}
			if d.Source.Kind == "goal" && d.Source.GoalID == goal.DSHGoalID {
				if !turnOpen || !stepOpen || target || human || preTargetActivity || d.ID == "" || d.Role != "user" || d.Source.Revision != goalRevision || d.Source.Round <= lastRound || d.Source.Round > goal.RoundsStarted || seq > goal.DSHEventSeq {
					return fail
				}
				target, lastRound = true, d.Source.Round
			} else if target && d.Source.Kind != "runtime-context" && d.Source.Kind != "skill-catalog" && d.Source.Kind != "time-context" {
				conflict = true
			}
		case "compaction/summary":
			invalid = true
		case "aibertai/invocation":
			var d struct{ InvocationID, Purpose string }
			if json.Unmarshal(e.Data, &d) != nil {
				return fail
			}
			if !target {
				if seenInvocation[d.InvocationID] {
					conflict = true
				}
				break
			}
			if !stepOpen || !validUUID(d.InvocationID) || d.Purpose == "" || invocation != "" || seenInvocation[d.InvocationID] || priorInvocation[d.InvocationID] {
				conflict = true
				break
			}
			if len(seenInvocation) >= 4096 {
				tooLarge = true
				break
			}
			invocation, invocationPurpose, seenInvocation[d.InvocationID] = d.InvocationID, d.Purpose, true
		case "assistant/message":
			var d struct {
				Turn, Step       int64
				InvocationID     string
				InvocationFinish string
				Usage            *struct {
					Input  *int64 `json:"inputTokens"`
					Output *int64 `json:"outputTokens"`
				}
				Message struct {
					ID, Role string
					Source   struct{ Kind string }
					Content  []struct{ Type, ID, Name, Arguments, Text string }
				}
			}
			if json.Unmarshal(e.Data, &d) != nil {
				return fail
			}
			if !target {
				if seenInvocation[d.InvocationID] {
					conflict = true
				}
				break
			}
			use, ok := goal.Invocations[d.InvocationID]
			if !stepOpen || d.Turn != turn || d.Step != step || assistantSeen || d.Message.ID == "" || d.Message.Role != "assistant" || d.Message.Source.Kind != "model" || d.InvocationID != invocation || invocationPurpose != "conversation" || !ok || !use.Settled || d.Usage == nil || d.Usage.Input == nil || d.Usage.Output == nil || use.Input == nil || use.Output == nil || *d.Usage.Input != *use.Input || *d.Usage.Output != *use.Output {
				conflict = true
				break
			}
			assistantSeen = true
			observedSettled = true
			for _, block := range d.Message.Content {
				if block.Type == "text" && block.Text != "" {
					invalid = true
				}
				if block.Type != "tool-call" {
					continue
				}
				if block.ID == "" || declared[block.ID].name != "" {
					invalid = true
					continue
				}
				declared[block.ID] = declaration{block.Name, block.Arguments}
			}
			if len(declared) > 0 && d.InvocationFinish != "tool-calls" {
				invalid = true
			}
		case "tool/call":
			var d struct {
				Turn, Step              int64
				CallID, Name, Arguments string
			}
			if json.Unmarshal(e.Data, &d) != nil {
				return fail
			}
			if !target {
				if seenCalls[d.CallID] {
					conflict = true
				}
				break
			}
			if d.Turn != turn {
				invalid = true
				break
			}
			if !existing[d.CallID] {
				plan.needsRepair = true
			}
			want := declared[d.CallID]
			if seenCalls[d.CallID] {
				conflict = true
			}
			canonical, argsErr := strictDSHJSON([]byte(d.Arguments))
			if !stepOpen || d.Step != step || changedGoal || d.CallID == "" || len(d.CallID) > 1024 || d.Name != "read" || len(d.Arguments) > 64<<10 || argsErr != nil || len(canonical) == 0 || canonical[0] != '{' || want.name != d.Name || want.arguments != d.Arguments || seenCalls[d.CallID] {
				invalid = true
				break
			}
			delete(declared, d.CallID)
			if len(seenCalls) >= 4096 || retained+len(raw) > maxDesktopSessionBytes {
				tooLarge = true
				break
			}
			seenCalls[d.CallID] = true
			pending[d.CallID] = len(plan.records)
			plan.records = append(plan.records, dshGoalReadRecord{id: d.CallID, callSeq: seq, call: append(json.RawMessage(nil), raw...)})
			retained += len(raw)
		case "tool/result":
			var d struct {
				Turn, Step int64
				Message    struct {
					Role, ToolCallID string
					IsError          bool
					Source           struct{ Kind, CallID string }
				}
			}
			if json.Unmarshal(e.Data, &d) != nil {
				return fail
			}
			if !target {
				if seenCalls[d.Message.Source.CallID] {
					conflict = true
				}
				break
			}
			if d.Turn != turn {
				invalid = true
				break
			}
			i, ok := pending[d.Message.Source.CallID]
			if !ok || !stepOpen || d.Step != step || d.Message.Role != "tool" || d.Message.IsError || d.Message.Source.Kind != "tool" || d.Message.ToolCallID != d.Message.Source.CallID || len(e.SourceEventSeqs) != 1 || e.SourceEventSeqs[0] != plan.records[i].callSeq || seq <= plan.records[i].callSeq {
				if seenCalls[d.Message.Source.CallID] {
					conflict = true
				}
				invalid = true
				break
			}
			if retained+len(raw) > maxDesktopSessionBytes {
				tooLarge = true
				break
			}
			plan.records[i].result, plan.records[i].resultSeq = append(json.RawMessage(nil), raw...), seq
			retained += len(raw)
			delete(pending, d.Message.Source.CallID)
		}
		return nil
	})
	if err != nil {
		return dshGoalReadPlan{}, err
	}
	if bindingSeq < 0 || turnOpen || stepOpen {
		return dshGoalReadPlan{}, fail
	}
	if observedSettled {
		if _, err := goalReportedUsage(goal, original.Usage); err != nil {
			conflict = true
		}
	}
	if !plan.needsRepair {
		if conflict {
			return dshGoalReadPlan{}, fail
		}
		return plan, nil
	}
	if latestRevision != goal.DSHRevision || latestPhase != goal.DSHPhase {
		return dshGoalReadPlan{}, fail
	}
	if tooLarge {
		return dshGoalReadPlan{}, errDSHHistoryTooLarge
	}
	if invalid || conflict || len(pending) != 0 {
		return dshGoalReadPlan{}, fail
	}
	if _, err := goalReportedUsage(goal, original.Usage); err != nil {
		return dshGoalReadPlan{}, fail
	}
	return plan, nil
}
