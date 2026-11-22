package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

type dshHistoryEvent struct {
	Seq  int64           `json:"seq"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
type dshHistoryPage struct {
	SessionID string            `json:"sessionId"`
	Through   int64             `json:"throughSeq"`
	Events    []dshHistoryEvent `json:"events"`
}

type dshGoalCompletionEvidence struct {
	CommandID    string `json:"commandId"`
	GoalID       string `json:"goalId"`
	GoalRevision int64  `json:"goalRevision"`
	GoalSeq      int64  `json:"goalSeq"`
	AssistantSeq int64  `json:"assistantSeq"`
	InvocationID string `json:"invocationId"`
	Turn         int64  `json:"turn"`
	TurnEndSeq   int64  `json:"turnEndSeq"`
	FinishReason string `json:"finishReason"`
}

type dshGoalQueryBinding struct {
	OperationID   string `json:"operationId"`
	CommandID     string `json:"commandId"`
	CommandRunSeq int64  `json:"commandRunSeq"`
	GoalID        string `json:"goalId"`
	GoalRevision  int64  `json:"goalRevision"`
	GoalSeq       int64  `json:"goalSeq"`
}

// readDSHGoalCompletion accepts only SDK history pages verified against the
// flushed session log. It binds the completion marker to the platform command,
// goal revision, committed final invocation, assistant text and turn ending.
func (l *acpLive) readDSHGoalCompletion(ctx context.Context, p *dshProcess, sessionID, goalID string) (string, error) {
	row, err := l.load(sessionID)
	if err != nil || row.Goal == nil || row.Goal.ID != goalID || row.Goal.Status != "reconciling" {
		return "", errors.New("goal_reconciliation_changed")
	}
	goal := *row.Goal
	if goal.DSHPhase != "complete" || goal.DSHGoalID == "" || goal.DSHCommandID == "" || goal.DSHRevision < 1 || goal.DSHEventSeq < 0 || goal.Uncertain || goal.hasPendingDSHInvocation() {
		return "", errors.New("dsh_completion_evidence_unavailable")
	}
	read := func(after int64, through *int64) (dshHistoryPage, error) {
		params := map[string]any{"sessionId": sessionID, "afterSeq": after}
		if through != nil {
			params["throughSeq"] = *through
		}
		raw, err := p.call(ctx, "aibertai/history", params, nil)
		if err != nil {
			return dshHistoryPage{}, err
		}
		var page dshHistoryPage
		if json.Unmarshal(raw, &page) != nil || page.SessionID != sessionID || len(page.Events) > 128 || page.Through < after || through != nil && page.Through != *through {
			return page, errors.New("dsh_invalid_history")
		}
		previous := after
		for _, event := range page.Events {
			if event.Seq != previous+1 || event.Seq > page.Through {
				return page, errors.New("dsh_invalid_history")
			}
			previous = event.Seq
		}
		if previous < page.Through && len(page.Events) == 0 {
			return page, errors.New("dsh_invalid_history")
		}
		return page, nil
	}
	return verifyDSHGoalCompletion(row, read)
}

func verifyDSHGoalCompletion(row acpSession, read func(int64, *int64) (dshHistoryPage, error)) (string, error) {
	goal := *row.Goal
	if !validUUID(row.GoalRun) || !validUUID(goal.DSHQueryOp) {
		return "", errors.New("dsh_completion_evidence_unavailable")
	}
	after := int64(-1)
	var ceiling *int64
	var marker *dshGoalCompletionEvidence
	var markerSeq int64
	var binding *dshGoalQueryBinding
	var bindingSeq int64
	for {
		page, err := read(after, ceiling)
		if err != nil {
			return "", err
		}
		if ceiling == nil {
			value := page.Through
			ceiling = &value
		}
		for _, event := range page.Events {
			after = event.Seq
			if event.Type == "aibertai/query-binding" {
				if binding != nil && event.Seq < goal.DSHEventSeq {
					return "", errors.New("dsh_completion_changed")
				}
				var candidate dshGoalQueryBinding
				if json.Unmarshal(event.Data, &candidate) != nil {
					return "", errors.New("dsh_invalid_completion_evidence")
				}
				if candidate.OperationID == goal.DSHQueryOp || candidate.CommandID == goal.DSHCommandID {
					if binding != nil || candidate.OperationID != goal.DSHQueryOp || candidate.CommandID != goal.DSHCommandID {
						return "", errors.New("dsh_invalid_completion_evidence")
					}
					binding, bindingSeq = &candidate, event.Seq
				}
			}
			if event.Type == "goal/change" && event.Seq > goal.DSHEventSeq {
				return "", errors.New("dsh_completion_changed")
			}
			if binding != nil && event.Type == "goal/change" && event.Seq > bindingSeq && event.Seq < goal.DSHEventSeq {
				return "", errors.New("dsh_completion_changed")
			}
			if event.Type != "aibertai/query-completed" {
				continue
			}
			var candidate dshGoalCompletionEvidence
			if json.Unmarshal(event.Data, &candidate) != nil {
				return "", errors.New("dsh_invalid_completion_evidence")
			}
			if candidate.CommandID != goal.DSHCommandID {
				continue
			}
			if marker != nil {
				return "", errors.New("dsh_invalid_completion_evidence")
			}
			marker = &candidate
			markerSeq = event.Seq
		}
		if after == *ceiling {
			break
		}
	}
	if marker == nil || binding == nil || binding.GoalID != goal.DSHGoalID || binding.GoalRevision < 1 || binding.CommandRunSeq < 0 || binding.CommandRunSeq >= binding.GoalSeq || binding.GoalSeq >= bindingSeq || bindingSeq >= marker.GoalSeq || binding.GoalRevision >= marker.GoalRevision || marker.GoalID != goal.DSHGoalID || marker.GoalRevision != goal.DSHRevision || marker.GoalSeq != goal.DSHEventSeq || marker.FinishReason != "completed" || marker.Turn <= 0 || !validUUID(marker.InvocationID) || marker.GoalSeq >= marker.AssistantSeq || marker.AssistantSeq >= marker.TurnEndSeq || marker.TurnEndSeq >= markerSeq {
		return "", errors.New("dsh_invalid_completion_evidence")
	}
	commandDone := 0
	commandAfter := binding.GoalSeq
	for commandAfter < bindingSeq {
		page, err := read(commandAfter, &bindingSeq)
		if err != nil {
			return "", err
		}
		if len(page.Events) == 0 {
			return "", errors.New("dsh_invalid_history")
		}
		for _, event := range page.Events {
			commandAfter = event.Seq
			if event.Type == "command/done" {
				var done struct{ CommandID, Kind string }
				if json.Unmarshal(event.Data, &done) != nil {
					return "", errors.New("dsh_invalid_completion_evidence")
				}
				if done.CommandID == binding.CommandID {
					if done.Kind != "success" || event.Seq <= binding.GoalSeq || event.Seq >= bindingSeq {
						return "", errors.New("dsh_invalid_completion_evidence")
					}
					commandDone++
				}
			}
		}
	}
	if commandDone != 1 {
		return "", errors.New("dsh_invalid_completion_evidence")
	}
	if call, ok := goal.Invocations[marker.InvocationID]; !ok || !call.Settled {
		return "", errors.New("goal_usage_reconciliation_required")
	}
	eventAt := func(seq int64) (dshHistoryEvent, error) {
		page, err := read(seq-1, &seq)
		if err != nil {
			return dshHistoryEvent{}, err
		}
		if len(page.Events) != 1 || page.Events[0].Seq != seq {
			return dshHistoryEvent{}, errors.New("dsh_invalid_history")
		}
		return page.Events[0], nil
	}
	command, err := eventAt(binding.CommandRunSeq)
	if err != nil {
		return "", err
	}
	var run struct{ CommandID, Name string }
	if command.Type != "command/run" || json.Unmarshal(command.Data, &run) != nil || run.CommandID != binding.CommandID || run.Name != "goal" {
		return "", errors.New("dsh_invalid_completion_evidence")
	}
	initial, err := eventAt(binding.GoalSeq)
	if err != nil {
		return "", err
	}
	var start struct {
		Operation string `json:"operation"`
		Goal      struct {
			ID       string `json:"id"`
			Revision int64  `json:"revision"`
			Phase    string `json:"phase"`
		} `json:"goal"`
	}
	if initial.Type != "goal/change" || json.Unmarshal(initial.Data, &start) != nil || (start.Operation != "create" && start.Operation != "resume") || start.Goal.ID != binding.GoalID || start.Goal.Revision != binding.GoalRevision || start.Goal.Phase != "active" {
		return "", errors.New("dsh_invalid_completion_evidence")
	}
	goalEvent, err := eventAt(marker.GoalSeq)
	if err != nil {
		return "", err
	}
	var change struct {
		Operation string `json:"operation"`
		Goal      struct {
			ID       string `json:"id"`
			Revision int64  `json:"revision"`
			Phase    string `json:"phase"`
		} `json:"goal"`
	}
	if goalEvent.Type != "goal/change" || json.Unmarshal(goalEvent.Data, &change) != nil || change.Operation != "complete" || change.Goal.ID != marker.GoalID || change.Goal.Revision != marker.GoalRevision || change.Goal.Phase != "complete" {
		return "", errors.New("dsh_invalid_completion_evidence")
	}
	assistant, err := eventAt(marker.AssistantSeq)
	if err != nil {
		return "", err
	}
	var message struct {
		Turn             int64  `json:"turn"`
		InvocationID     string `json:"invocationId"`
		InvocationFinish string `json:"invocationFinish"`
		Message          struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"message"`
	}
	if assistant.Type != "assistant/message" || json.Unmarshal(assistant.Data, &message) != nil || message.Turn != marker.Turn || message.InvocationID != marker.InvocationID || message.InvocationFinish != "stop" {
		return "", errors.New("dsh_invalid_completion_evidence")
	}
	var text strings.Builder
	for _, block := range message.Message.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}
	if strings.TrimSpace(text.String()) == "" {
		return "", errors.New("dsh_invalid_completion_evidence")
	}
	end, err := eventAt(marker.TurnEndSeq)
	if err != nil {
		return "", err
	}
	var terminal struct {
		Turn   int64 `json:"turn"`
		Reason struct {
			Kind string `json:"kind"`
		} `json:"reason"`
	}
	if end.Type != "turn/end" || json.Unmarshal(end.Data, &terminal) != nil || terminal.Turn != marker.Turn || terminal.Reason.Kind != "completed" {
		return "", errors.New("dsh_invalid_completion_evidence")
	}
	if len(row.Turns) == 0 || row.Turns[len(row.Turns)-1].Operation != row.GoalRun {
		return "", errors.New("dsh_completion_evidence_unavailable")
	}
	if _, err := goalReportedUsage(&goal, row.Turns[len(row.Turns)-1].Usage); err != nil {
		return "", errors.New("goal_usage_reconciliation_required")
	}
	return text.String(), nil
}
