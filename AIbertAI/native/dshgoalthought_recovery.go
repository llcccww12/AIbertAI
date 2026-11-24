package main

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"slices"
)

// recoverDSHGoalThoughts derives optional presentation from an immutable SDK
// history prefix. Failure leaves the checkpoint untouched; it never authorizes
// a tool, releases a workspace claim or substitutes for usage reconciliation.
func (l *acpLive) recoverDSHGoalThoughts(ctx context.Context, p *dshProcess, sessionID, goalID string) error {
	return l.recoverDSHGoalThoughtsWith(ctx, sessionID, goalID, func(ctx context.Context, method string, params any) (json.RawMessage, error) {
		return p.call(ctx, method, params, nil)
	})
}

func (l *acpLive) recoverDSHGoalThoughtsWith(ctx context.Context, sessionID, goalID string, call func(context.Context, string, any) (json.RawMessage, error)) error {
	row, err := l.load(sessionID)
	if err != nil || row.Goal == nil || row.Goal.ID != goalID || row.Goal.Status != "reconciling" || row.Goal.Uncertain || row.Goal.hasPendingDSHInvocation() || !validUUID(row.Goal.DSHQueryOp) || row.Goal.DSHCommandID == "" || len(row.Turns) == 0 {
		return nil
	}
	index := len(row.Turns) - 1
	original := row.Turns[index]
	if row.GoalRun != original.Operation || original.DSHGoalID != goalID || original.GoalCheckpoint == nil || !reflect.DeepEqual(original.ToolRuns, original.GoalCheckpoint.Runs) {
		return nil
	}
	type callEvidence struct {
		count           int
		seq, turn, step int64
		name, arguments string
	}
	wanted := map[string]workspaceToolRun{}
	for _, run := range original.ToolRuns {
		if run.DSHCallID == "" || run.Call == nil || wanted[run.DSHCallID].Call != nil {
			return nil
		}
		wanted[run.DSHCallID] = run
	}
	calls := map[string]callEvidence{}
	invocations := map[string]int{}
	messages := map[string]int{}
	tracker := dshGoalThoughtTracker{}
	retained := 0
	fetch := func(ctx context.Context, params map[string]any) (json.RawMessage, error) {
		return call(ctx, "aibertai/history", params)
	}
	err = scanDSHGoalPages(ctx, row.DSHSessionID, fetch, func(e dshConnectorHistoryEvent, raw json.RawMessage) error {
		switch e.Type {
		case "aibertai/invocation", "assistant/message":
			var d struct{ InvocationID string }
			if json.Unmarshal(e.Data, &d) != nil {
				return errDSHInvalidHistory
			}
			if validUUID(d.InvocationID) {
				if e.Type == "aibertai/invocation" {
					invocations[d.InvocationID]++
				} else {
					messages[d.InvocationID]++
				}
			}
		case "tool/call":
			var d struct {
				Turn, Step              int64
				CallID, Name, Arguments string
			}
			if json.Unmarshal(e.Data, &d) != nil {
				return errDSHInvalidHistory
			}
			if run, ok := wanted[d.CallID]; ok {
				v := calls[d.CallID]
				v.count++
				if v.count == 1 && d.Name == run.Call.Function.Name && d.Arguments == run.Call.Function.Arguments {
					v.seq, v.turn, v.step, v.name, v.arguments = *e.Seq, d.Turn, d.Step, d.Name, d.Arguments
				}
				calls[d.CallID] = v
			}
		}
		if len(invocations) > 4096 || len(messages) > 4096 {
			return errDSHHistoryTooLarge
		}
		frame, marshalErr := json.Marshal(map[string]any{"sessionId": row.DSHSessionID, "event": raw})
		if marshalErr != nil {
			return marshalErr
		}
		before := len(tracker.candidates)
		tracker.event(frame, row)
		if len(tracker.candidates) > before {
			retained += len(raw)
			if retained > maxDesktopSessionBytes || len(tracker.candidates) > 4096 {
				return errDSHHistoryTooLarge
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	projected := original
	projected.ToolRuns = slices.Clone(original.ToolRuns)
	checkpoint := *original.GoalCheckpoint
	changed := false
	for id, candidate := range tracker.candidates {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if invocations[id] != 1 || messages[id] != 1 || candidate.proof.QueryOperation != row.Goal.DSHQueryOp || candidate.commandID != row.Goal.DSHCommandID {
			continue
		}
		ledger, ok := row.Goal.Invocations[id]
		if !ok || !ledger.Settled || ledger.Input == nil || ledger.Output == nil || *ledger.Input != candidate.input || *ledger.Output != candidate.output {
			continue
		}
		raw, callErr := call(ctx, "aibertai/reconcile", map[string]string{"sessionId": row.DSHSessionID, "invocationId": id})
		if callErr != nil {
			return callErr
		}
		var receipt struct {
			State         string
			Seq, UsageSeq *int64
			Usage         *struct {
				Input  *int64 `json:"inputTokens"`
				Output *int64 `json:"outputTokens"`
			}
		}
		if json.Unmarshal(raw, &receipt) != nil || receipt.State != "committed" || receipt.Seq == nil || receipt.UsageSeq == nil || *receipt.Seq != candidate.proof.MessageSeq || *receipt.UsageSeq != candidate.proof.MessageSeq || receipt.Usage == nil || receipt.Usage.Input == nil || receipt.Usage.Output == nil || *receipt.Usage.Input != candidate.input || *receipt.Usage.Output != candidate.output {
			continue
		}
		if len(candidate.calls) == 0 {
			if checkpoint.DSHProgressInvocation == id && checkpoint.DSHProgressQuery == candidate.proof.QueryOperation && checkpoint.Progress == candidate.text {
				checkpoint.DSHParts, checkpoint.DSHThoughtProof = candidate.parts, &candidate.proof
				changed = true
			}
			continue
		}
		call := candidate.calls[0]
		evidence := calls[call.id]
		if evidence.count != 1 || evidence.seq <= candidate.proof.MessageSeq || evidence.turn != candidate.turn || evidence.step != candidate.step || evidence.name != call.name || evidence.arguments != call.arguments {
			continue
		}
		for i := range projected.ToolRuns {
			run := &projected.ToolRuns[i]
			if run.DSHCallID == call.id && run.Preamble == candidate.text {
				run.DSHParts, run.DSHThoughtProof = candidate.parts, &candidate.proof
				changed = true
			}
		}
	}
	if !changed {
		return nil
	}
	checkpoint.Runs = slices.Clone(projected.ToolRuns)
	projected.GoalCheckpoint = &checkpoint
	l.mu.Lock()
	defer l.mu.Unlock()
	current, ok := l.sessions[sessionID]
	if !ok || current.Goal == nil || current.Goal.ID != goalID || current.Goal.Version != row.Goal.Version || current.GoalRun != row.GoalRun || len(current.Turns) != len(row.Turns) {
		return nil
	}
	before, e1 := json.Marshal(original)
	after, e2 := json.Marshal(current.Turns[index])
	if e1 != nil || e2 != nil || !bytes.Equal(before, after) {
		return nil
	}
	current.Turns = slices.Clone(current.Turns)
	current.Turns[index] = projected
	return l.save(current)
}
