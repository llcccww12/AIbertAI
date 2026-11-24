package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
)

var errDSHGoalProjectionUnverified = errors.New("dsh_goal_projection_unverified")

// repairDSHGoalReadProjection handles the conservative read-only subset. If a
// history cannot be verified, it stops reconciliation without advancing the
// read projection watermark. No tool is dispatched here.
func (l *acpLive) repairDSHGoalReadProjection(ctx context.Context, fetchHistory dshConnectorHistoryFetch, sessionID, goalID string) error {
	unverified := errDSHGoalProjectionUnverified
	l.mu.Lock()
	cached, ok := l.sessions[sessionID]
	var snapshot []byte
	var err error
	if ok {
		snapshot, err = json.Marshal(cached)
	}
	l.mu.Unlock()
	if len(snapshot) > maxDesktopSessionBytes {
		return errDSHHistoryTooLarge
	}
	var row acpSession
	if !ok || err != nil || json.Unmarshal(snapshot, &row) != nil || row.Goal == nil || row.Goal.ID != goalID || row.Goal.Status != "reconciling" {
		return errors.New("goal_reconciliation_changed")
	}
	if row.Goal.Uncertain || row.Goal.hasPendingDSHInvocation() || row.Goal.DSHPhase != "active" && row.Goal.DSHPhase != "paused" || row.GoalRun == "" {
		return nil
	}
	if len(row.Turns) == 0 || !validUUID(row.Goal.DSHQueryOp) || row.Goal.DSHCommandID == "" {
		return unverified
	}
	index := len(row.Turns) - 1
	original := row.Turns[index]
	if row.GoalRun != original.Operation || original.GoalCheckpoint == nil {
		return unverified
	}
	if dshPendingWrite(original) || l.store != nil && l.store.Root != nil && dshConnectorAssertSettled(l.store.Root, sessionID) != nil {
		return unverified
	}
	plan, err := collectDSHGoalReads(ctx, row, original, fetchHistory)
	if err != nil {
		return err
	}
	if !plan.needsRepair {
		return nil
	}
	projected := original
	projected.ToolRuns = slices.Clone(original.ToolRuns)
	existing := make(map[string]workspaceToolRun, len(original.ToolRuns))
	for _, run := range original.ToolRuns {
		existing[run.DSHCallID] = run
	}
	verified := map[string]bool{}
	added := false
	projectedThrough := int64(-1)
	for _, record := range plan.records {
		if len(record.call) == 0 || len(record.result) == 0 || record.resultSeq <= record.callSeq {
			return unverified
		}
		call, err := json.Marshal(map[string]any{"sessionId": row.DSHSessionID, "event": record.call})
		if err != nil {
			return err
		}
		result, err := json.Marshal(map[string]any{"sessionId": row.DSHSessionID, "event": record.result})
		if err != nil {
			return err
		}
		check := acpTurn{Operation: original.Operation}
		if update, err := projectDSHTool(&check, call); err != nil || update == nil {
			return unverified
		}
		if update, err := projectDSHTool(&check, result); err != nil || update == nil {
			return unverified
		}
		got := check.ToolRuns[0]
		if got.Pending || got.Call == nil || got.Report.Failed != 0 {
			return unverified
		}
		verified[record.id] = true
		if old, ok := existing[record.id]; ok {
			if added || old.Call == nil || old.Call.Function.Name != got.Call.Function.Name || old.Call.Function.Arguments != got.Call.Function.Arguments || old.Pending || old.Output != got.Output || old.Report.Failed != got.Report.Failed {
				return unverified
			}
		} else {
			got.Call.ID = fmt.Sprintf("%s-tool-%d", original.Operation, len(projected.ToolRuns)+1)
			projected.ToolRuns = append(projected.ToolRuns, got)
			added = true
		}
		if record.resultSeq > projectedThrough {
			projectedThrough = record.resultSeq
		}
	}
	if projectedThrough < 0 {
		return unverified
	}
	if len(original.GoalCheckpoint.Runs) > len(original.ToolRuns) {
		return unverified
	}
	for i, run := range original.ToolRuns {
		if i < len(original.GoalCheckpoint.Runs) {
			if !reflect.DeepEqual(run, original.GoalCheckpoint.Runs[i]) {
				return unverified
			}
		} else if !verified[run.DSHCallID] {
			return unverified
		}
	}
	if row.Goal.DSHReadProjSeq != nil && row.Goal.DSHReadProjOp == row.Goal.DSHQueryOp && projectedThrough <= *row.Goal.DSHReadProjSeq {
		return unverified
	}
	checkpoint := *original.GoalCheckpoint
	checkpoint.Runs = slices.Clone(projected.ToolRuns)
	projected.GoalCheckpoint = &checkpoint
	l.mu.Lock()
	defer l.mu.Unlock()
	current, ok := l.sessions[sessionID]
	if !ok || current.Goal == nil || current.Goal.ID != goalID || current.Goal.Status != "reconciling" || current.Goal.Version != row.Goal.Version || current.GoalRun != row.GoalRun || len(current.Turns) != len(row.Turns) || current.Turns[index].Operation != original.Operation {
		return errors.New("goal_reconciliation_changed")
	}
	before, err := json.Marshal(original)
	if err != nil {
		return err
	}
	after, err := json.Marshal(current.Turns[index])
	if err != nil || !bytes.Equal(before, after) {
		return errors.New("goal_reconciliation_changed")
	}
	goal := *current.Goal
	goal.DSHReadProjOp, goal.DSHReadProjSeq = goal.DSHQueryOp, &projectedThrough
	goal.Version++
	current.Goal = &goal
	current.Turns = slices.Clone(current.Turns)
	current.Turns[index] = projected
	if err := l.save(current); err != nil {
		return errors.New("session_write_failed")
	}
	return nil
}
