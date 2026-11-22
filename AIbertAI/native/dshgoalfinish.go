package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Called only after the SDK reports that goal closing activity has ended.
// Until then the active status allows the SDK's final model call to finish.
func (l *acpLive) prepareDSHGoalCompletion(sessionID, goalID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	row, ok := l.sessions[sessionID]
	if !ok || row.Engine != "dsh" || row.Goal == nil || row.Goal.ID != goalID {
		return errors.New("goal_not_found")
	}
	goal := *row.Goal
	if !goal.Current || (goal.Status != "active" && !(goal.Status == "budget_limited" && goal.DSHPhase == "complete")) || goal.Uncertain || goal.hasPendingDSHInvocation() {
		return errors.New("goal_completion_unavailable")
	}
	switch goal.DSHPhase {
	case "complete":
		goal.Status, goal.Reason = "paused", "verification_required"
	case "blocked":
		goal.Status, goal.Reason = "blocked", "dsh_goal_blocked"
	default:
		return errors.New("goal_completion_unavailable")
	}
	goal.Version++
	row.Goal = &goal
	if err := l.save(row); err != nil {
		return errors.New("session_write_failed")
	}
	return nil
}

// finishDSHGoal verifies observable final output before publishing completion.
// A successful write/edit is checked against the final workspace file; this
// proves the recorded artifact exists, not that it satisfies every requirement.
func (l *acpLive) finishDSHGoal(sessionID, goalID string, turn acpTurn) error {
	l.mu.Lock()
	row, ok := l.sessions[sessionID]
	if !ok || row.Engine != "dsh" || row.Goal == nil || row.Goal.ID != goalID {
		l.mu.Unlock()
		return errors.New("goal_not_found")
	}
	goal := *row.Goal
	if !goal.Current && goal.Status == "completed" && goal.DSHPhase == "complete" {
		l.mu.Unlock()
		return nil
	}
	if !goal.Current || goal.Status != "paused" || goal.Reason != "verification_required" || goal.DSHPhase != "complete" || goal.Uncertain || goal.hasPendingDSHInvocation() {
		l.mu.Unlock()
		return errors.New("goal_completion_unavailable")
	}
	if turn.Operation == "" || row.GoalRun != turn.Operation || len(row.Turns) == 0 || row.Turns[len(row.Turns)-1].Operation != turn.Operation {
		l.mu.Unlock()
		return errors.New("goal_completion_mismatch")
	}
	if dshPendingWrite(turn) || l.store != nil && dshUnsettledWorkspaceWrite(l.store.Root, sessionID, turn) {
		goal.Reason, goal.Uncertain = "resume_confirmation", true
		goal.Version++
		row.Goal = &goal
		err := l.save(row)
		l.mu.Unlock()
		if err != nil {
			return errors.New("session_write_failed")
		}
		return errors.New("dsh_write_outcome_unknown")
	}
	l.mu.Unlock()
	if strings.TrimSpace(turn.Answer) == "" || turn.Failed || turn.Interrupted {
		return errors.New("goal_output_missing")
	}
	type fileResult struct {
		name        string
		content     *string
		replacement *string
		limit       int
	}
	files := map[string]fileResult{}
	for _, run := range turn.ToolRuns {
		if run.Pending {
			return errors.New("goal_tool_result_unknown")
		}
		if run.Call == nil || run.Report.Failed != 0 || (run.Call.Function.Name != "write" && run.Call.Function.Name != "edit") {
			continue
		}
		var args struct {
			Path    string  `json:"file_path"`
			Content *string `json:"content"`
			New     *string `json:"new_string"`
		}
		if json.Unmarshal([]byte(run.Call.Function.Arguments), &args) != nil || args.Path == "" {
			return errors.New("goal_artifact_unverifiable")
		}
		path := args.Path
		if filepath.IsAbs(path) {
			var err error
			path, err = filepath.Rel(row.CWD, path)
			if err != nil {
				return errors.New("goal_artifact_unverifiable")
			}
		}
		path = filepath.ToSlash(path)
		if _, err := workspaceRelative(path); err != nil || path == "." {
			return errors.New("goal_artifact_unverifiable")
		}
		if run.Call.Function.Name == "write" && args.Content == nil || run.Call.Function.Name == "edit" && args.New == nil {
			return errors.New("goal_artifact_unverifiable")
		}
		limit := maxToolText
		if l.store != nil && l.store.Root != nil {
			claim, err := readConnectorClaim(l.store.Root, connectorClaimPath(sessionID, turn.Operation, run.DSHCallID), sessionID)
			if err == nil && claim.State == "reconciled" && claim.SDKSeq != nil && claim.WorkspaceBoundary != nil {
				w := claim.WorkspaceBoundary
				canonical, canonicalErr := strictDSHJSON([]byte(run.Call.Function.Arguments))
				sum := sha256.Sum256(canonical)
				if canonicalErr == nil && claim.OperationID == turn.Operation && claim.CallID == run.DSHCallID && w.GoalID == goal.ID && w.SDKGoalID == goal.DSHGoalID && validUUID(w.HistoryOperation) && w.Name == run.Call.Function.Name && w.ArgumentsSHA == fmt.Sprintf("%x", sum) {
					limit = maxDSHWorkspaceText
				}
			}
		}
		files[path] = fileResult{name: path, content: args.Content, replacement: args.New, limit: limit}
	}
	if len(files) != 0 {
		root, err := os.OpenRoot(row.CWD)
		if err != nil {
			return errors.New("goal_artifact_unverifiable")
		}
		defer root.Close()
		for _, result := range files {
			f, err := openAttachmentFile(root, filepath.FromSlash(result.name), int64(result.limit))
			if err != nil {
				return errors.New("goal_artifact_unverifiable")
			}
			data, readErr := io.ReadAll(io.LimitReader(f, int64(result.limit)+1))
			if errors.Join(readErr, f.Close()) != nil || len(data) > result.limit {
				return errors.New("goal_artifact_unverifiable")
			}
			if result.content != nil && string(data) != *result.content || result.replacement != nil && *result.replacement != "" && !strings.Contains(string(data), *result.replacement) {
				return errors.New("goal_artifact_unverifiable")
			}
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	current := l.sessions[sessionID]
	if current.Goal == nil || current.Goal.ID != goalID || current.Goal.Version != goal.Version || !current.Goal.Current || current.Goal.Status != "paused" || current.Goal.Reason != "verification_required" {
		return errors.New("goal_completion_changed")
	}
	finished := *current.Goal
	finished.Current, finished.Status, finished.Reason = false, "completed", ""
	finished.Version++
	current.Goal = &finished
	if err := l.save(current); err != nil {
		return errors.New("session_write_failed")
	}
	return nil
}
