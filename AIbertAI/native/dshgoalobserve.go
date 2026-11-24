package main

import (
	"encoding/json"
	"errors"
	"strings"
)

type dshGoalUsageRecord struct {
	usage      modelUsage
	seq        int64
	compaction bool
	progress   string
}

// dshGoalObserver is consumed serially in SDK stdio order. The gateway's
// settlement acknowledgment must run only after observe returns successfully.
type dshGoalObserver struct {
	live        *acpLive
	sessionID   string
	goalID      string
	usage       map[string]dshGoalUsageRecord
	failedUsage map[string]dshFailedCompactionReceipt
	thoughts    dshGoalThoughtTracker
	onThought   func(dshGoalThoughtCandidate) error
}

func (o *dshGoalObserver) observe(method string, raw json.RawMessage) error {
	switch method {
	case "session.event":
		var e struct {
			SessionID string `json:"sessionId"`
			Event     struct {
				Type string `json:"type"`
				Seq  int64  `json:"seq"`
				Data struct {
					InvocationID string `json:"invocationId"`
					OperationID  string `json:"operationId"`
					Message      struct {
						Content []struct {
							Type string `json:"type"`
							Text string `json:"text"`
						} `json:"content"`
					} `json:"message"`
					Usage *dshTokenUsage `json:"usage"`
				} `json:"data"`
			} `json:"event"`
		}
		if json.Unmarshal(raw, &e) != nil || e.SessionID != o.sessionID || e.Event.Seq < 0 {
			return errors.New("dsh_invalid_goal_event")
		}
		if e.Event.Type == "goal/change" || e.Event.Type == "user/message" {
			if err := o.live.recordDSHGoalEvent(o.sessionID, o.goalID, raw); err != nil {
				return err
			}
		}
		if o.onThought != nil {
			if row, err := o.live.load(o.sessionID); err == nil {
				o.thoughts.event(raw, row)
			}
		}
		if e.Event.Type == "aibertai/compaction-usage" {
			id := e.Event.Data.InvocationID
			if !validUUID(id) || !validUUID(e.Event.Data.OperationID) || e.Event.Data.Usage == nil || e.Event.Data.Usage.Input == nil || e.Event.Data.Usage.Output == nil || *e.Event.Data.Usage.Input < 0 || *e.Event.Data.Usage.Output < 0 {
				return errors.New("dsh_invalid_goal_event")
			}
			if o.failedUsage == nil {
				o.failedUsage = make(map[string]dshFailedCompactionReceipt)
			}
			if _, exists := o.failedUsage[id]; exists {
				return errors.New("goal_duplicate_usage")
			}
			seq := e.Event.Seq
			value := dshFailedCompactionReceipt{InvocationID: id, OperationID: e.Event.Data.OperationID, UsageSeq: &seq}
			value.Usage = e.Event.Data.Usage
			o.failedUsage[id] = value
			return nil
		}
		if e.Event.Type != "assistant/message" && e.Event.Type != "compaction/summary" {
			return nil
		}
		id := e.Event.Data.InvocationID
		if !validUUID(id) {
			return errors.New("goal_invalid_invocation")
		}
		if o.usage == nil {
			o.usage = make(map[string]dshGoalUsageRecord)
		}
		if _, duplicate := o.usage[id]; duplicate {
			return errors.New("goal_duplicate_usage")
		}
		var usage modelUsage
		if e.Event.Data.Usage != nil {
			usage.Input, usage.Output = e.Event.Data.Usage.Input, e.Event.Data.Usage.Output
		}
		var progress strings.Builder
		for _, block := range e.Event.Data.Message.Content {
			if block.Type == "text" {
				progress.WriteString(block.Text)
			}
		}
		text := []rune(strings.TrimSpace(progress.String()))
		if len(text) > 500 {
			text = text[:500]
		}
		o.usage[id] = dshGoalUsageRecord{usage: usage, seq: e.Event.Seq, compaction: e.Event.Type == "compaction/summary", progress: string(text)}
	case "aibertai/model-settled":
		var failure dshFailedCompactionReceipt
		if json.Unmarshal(raw, &failure) == nil && failure.State == "failed_with_usage" {
			row, err := o.live.load(o.sessionID)
			if err != nil || row.Goal == nil || row.Goal.ID != o.goalID || row.Goal.DSHQueryOp != failure.OperationID || failure.SessionID != o.sessionID || !validUUID(failure.InvocationID) || failure.UsageSeq == nil || *failure.UsageSeq < 0 || failure.Seq <= *failure.UsageSeq || failure.Usage == nil || failure.Usage.Input == nil || failure.Usage.Output == nil || *failure.Usage.Input < 0 || *failure.Usage.Output < 0 {
				return errors.New("goal_usage_reconciliation_required")
			}
			if evidence, ok := o.failedUsage[failure.InvocationID]; ok && (evidence.OperationID != failure.OperationID || evidence.UsageSeq == nil || *evidence.UsageSeq != *failure.UsageSeq || evidence.Usage == nil || evidence.Usage.Input == nil || evidence.Usage.Output == nil || *evidence.Usage.Input != *failure.Usage.Input || *evidence.Usage.Output != *failure.Usage.Output) {
				return errors.New("goal_usage_reconciliation_required")
			}
			if err = o.live.settleDSHGoalInvocation(o.sessionID, o.goalID, failure.InvocationID, modelUsage{Input: failure.Usage.Input, Output: failure.Usage.Output}); err != nil {
				return err
			}
			delete(o.failedUsage, failure.InvocationID)
			return nil
		}
		var receipt struct {
			SessionID    string `json:"sessionId"`
			InvocationID string `json:"invocationId"`
			State        string `json:"state"`
			Seq          int64  `json:"seq"`
			UsageSeq     *int64 `json:"usageSeq"`
			Usage        *struct {
				Input  *int64 `json:"inputTokens"`
				Output *int64 `json:"outputTokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(raw, &receipt) != nil || receipt.SessionID != o.sessionID || !validUUID(receipt.InvocationID) {
			return errors.New("dsh_invalid_settlement")
		}
		record, ok := o.usage[receipt.InvocationID]
		valid := receipt.State == "committed" && receipt.UsageSeq != nil && receipt.Usage != nil && receipt.Seq > 0
		var durable modelUsage
		if valid {
			durable.Input, durable.Output = receipt.Usage.Input, receipt.Usage.Output
			_, valid = goalModelTokens(durable)
		}
		if valid {
			valid = *receipt.UsageSeq >= 0 && (*receipt.UsageSeq == receipt.Seq || *receipt.UsageSeq+2 == receipt.Seq)
		}
		if valid && ok {
			expected := record.seq
			if record.compaction {
				expected += 2
			}
			valid = receipt.Seq == expected && *receipt.UsageSeq == record.seq && record.usage.Input != nil && record.usage.Output != nil && *record.usage.Input == *durable.Input && *record.usage.Output == *durable.Output
		}
		if !valid {
			_ = o.live.settleDSHGoalInvocation(o.sessionID, o.goalID, receipt.InvocationID, modelUsage{})
			return errors.New("goal_usage_reconciliation_required")
		}
		if err := o.live.settleDSHGoalInvocation(o.sessionID, o.goalID, receipt.InvocationID, durable); err != nil {
			return err
		}
		if record.progress != "" {
			if err := o.live.recordDSHGoalProgress(o.sessionID, o.goalID, record.progress); err != nil {
				return err
			}
		}
		if candidate, exists := o.thoughts.candidates[receipt.InvocationID]; exists {
			delete(o.thoughts.candidates, receipt.InvocationID)
			row, err := o.live.load(o.sessionID)
			if err == nil && ok && candidate.proof.MessageSeq == record.seq && row.Goal != nil && row.Goal.ID == o.goalID && row.Goal.DSHQueryOp == candidate.proof.QueryOperation && row.Goal.DSHCommandID == candidate.commandID && o.onThought != nil {
				if err := o.onThought(candidate); err != nil {
					return err
				}
			}
		}
		delete(o.usage, receipt.InvocationID)
		delete(o.failedUsage, receipt.InvocationID)
	}
	return nil
}

func (l *acpLive) recordDSHGoalProgress(sessionID, goalID, progress string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	row, ok := l.sessions[sessionID]
	if !ok || row.Goal == nil || row.Goal.ID != goalID {
		return errors.New("goal_not_found")
	}
	goal := *row.Goal
	goal.RecentProgress = progress
	goal.Version++
	row.Goal = &goal
	if err := l.save(row); err != nil {
		return errors.New("session_write_failed")
	}
	return nil
}
