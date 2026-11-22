package main

import (
	"encoding/json"
	"errors"
	"strings"
)

// recordDSHGoalEvent projects only durable SDK events for this platform goal.
// Completion remains pending verification until the requested deliverable has
// been checked by the caller.
func (l *acpLive) recordDSHGoalEvent(sessionID, goalID string, raw json.RawMessage) error {
	var e struct {
		SessionID string `json:"sessionId"`
		Event     struct {
			Type string `json:"type"`
			Seq  int64  `json:"seq"`
			Data struct {
				Kind      string `json:"kind"`
				Version   int    `json:"version"`
				Operation string `json:"operation"`
				Rounds    int    `json:"roundsStarted"`
				Goal      struct {
					ID            string `json:"id"`
					Revision      int64  `json:"revision"`
					Objective     string `json:"objective"`
					Phase         string `json:"phase"`
					BlockedReason *struct {
						Code string `json:"code"`
					} `json:"blockedReason"`
				} `json:"goal"`
				Cleared struct {
					ID       string `json:"id"`
					Revision int64  `json:"revision"`
				} `json:"cleared"`
				Source struct {
					Kind     string `json:"kind"`
					GoalID   string `json:"goalId"`
					Revision int64  `json:"revision"`
					Round    int    `json:"round"`
				} `json:"source"`
			} `json:"data"`
		} `json:"event"`
	}
	if json.Unmarshal(raw, &e) != nil || e.SessionID != sessionID || e.Event.Seq < 0 {
		return errors.New("dsh_invalid_goal_event")
	}
	if e.Event.Type != "goal/change" && e.Event.Type != "user/message" {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	row, ok := l.sessions[sessionID]
	if !ok || row.Engine != "dsh" || row.Goal == nil || row.Goal.ID != goalID {
		return errors.New("goal_not_found")
	}
	goal := *row.Goal
	if goal.DSHGoalID != "" && e.Event.Seq <= goal.DSHEventSeq {
		return nil
	}
	d := e.Event.Data
	switch e.Event.Type {
	case "goal/change":
		if d.Kind != "goal/change" || d.Version != 1 {
			return errors.New("dsh_invalid_goal_event")
		}
		id, revision := d.Goal.ID, d.Goal.Revision
		if d.Operation == "clear" {
			id, revision = d.Cleared.ID, d.Cleared.Revision
		}
		if !strings.HasPrefix(id, "goal-") || !validUUID(strings.TrimPrefix(id, "goal-")) || revision < 1 || d.Rounds < 0 {
			return errors.New("dsh_invalid_goal_event")
		}
		if goal.DSHGoalID == "" {
			if d.Operation != "create" || d.Goal.Objective != goal.Objective || d.Goal.Phase != "active" {
				return errors.New("dsh_goal_mismatch")
			}
			goal.DSHGoalID = id
		} else if goal.DSHGoalID != id || revision < goal.DSHRevision {
			return errors.New("dsh_goal_mismatch")
		}
		switch d.Operation {
		case "create", "edit", "resume":
			if d.Goal.Phase != "active" && d.Operation != "edit" {
				return errors.New("dsh_invalid_goal_event")
			}
			if (d.Operation == "create" || d.Operation == "edit") && d.Goal.Objective != goal.Objective {
				return errors.New("dsh_goal_mismatch")
			}
		case "pause":
			if d.Goal.Phase != "paused" {
				return errors.New("dsh_invalid_goal_event")
			}
			if goal.Current && goal.Status == "active" {
				goal.Status, goal.Reason = "paused", "user_paused"
			}
		case "block":
			if d.Goal.Phase != "blocked" {
				return errors.New("dsh_invalid_goal_event")
			}
		case "complete":
			if d.Goal.Phase != "complete" {
				return errors.New("dsh_invalid_goal_event")
			}
		case "clear":
			if goal.Status == "clearing" {
				goal.DSHPhase = "cleared"
			} else {
				goal.Current, goal.Status, goal.Reason = false, "cleared", ""
			}
		default:
			return errors.New("dsh_invalid_goal_event")
		}
		goal.DSHRevision = revision
		if d.Operation != "clear" {
			goal.DSHPhase = d.Goal.Phase
			goal.DSHObjective = d.Goal.Objective
		}
		if d.Rounds > goal.RoundsStarted {
			goal.RoundsStarted = d.Rounds
		}
	case "user/message":
		if d.Source.Kind != "goal" {
			return nil
		}
		if d.Source.GoalID != goal.DSHGoalID || d.Source.Revision != goal.DSHRevision || d.Source.Round < 1 || d.Source.Round < goal.RoundsStarted {
			return errors.New("dsh_goal_mismatch")
		}
		goal.RoundsStarted = d.Source.Round
	}
	goal.DSHEventSeq = e.Event.Seq
	goal.Version++
	row.Goal = &goal
	if err := l.save(row); err != nil {
		return errors.New("session_write_failed")
	}
	return nil
}
