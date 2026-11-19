package main

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

type desktopGoal struct {
	ID            string           `json:"goal_id"`
	Objective     string           `json:"objective"`
	Status        string           `json:"status"`
	Reason        string           `json:"reason_code,omitempty"`
	Generation    int64            `json:"generation"`
	Version       int64            `json:"state_version"`
	Current       bool             `json:"is_current"`
	ActiveSeconds int64            `json:"active_seconds"`
	Uncertain     bool             `json:"usage_uncertain"`
	Usage         desktopGoalUsage `json:"usage"`
	Limits        desktopGoalLimit `json:"effective_limits"`
}

type desktopGoalUsage struct {
	Tokens int64 `json:"charged_tokens"`
}

type desktopGoalLimit struct {
	Tokens int64 `json:"token_budget"`
}

func (l *acpLive) desktopGoalAction(raw json.RawMessage) (any, error) {
	var p struct {
		SessionID, Action, RequestID, Objective, ExpectedGoalID string
		ExpectedGeneration                                      int64
	}
	if json.Unmarshal(raw, &p) != nil || !validUUID(p.SessionID) || p.Action != "get" && !validUUID(p.RequestID) {
		return nil, errors.New("invalid_session_parameters")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	row, ok := l.sessions[p.SessionID]
	if !ok {
		return nil, errors.New("session_not_found")
	}
	goal := row.Goal
	if p.Action == "get" || p.RequestID == row.LastGoalRequest {
		return map[string]any{"goal": goal}, nil
	}
	if p.Action != "create" && (goal == nil || !goal.Current || p.ExpectedGoalID != goal.ID || p.ExpectedGeneration != goal.Generation) {
		return nil, errors.New("stale_goal")
	}
	if row.State == "running" && p.Action != "pause" && p.Action != "clear" {
		return nil, errors.New("goal_session_busy")
	}
	if p.Action == "create" || p.Action == "edit" {
		p.Objective = strings.TrimSpace(p.Objective)
		if p.Objective == "" || utf8.RuneCountInString(p.Objective) > 4000 {
			return nil, errors.New("invalid_goal_objective")
		}
	}
	var next desktopGoal
	if goal != nil {
		next = *goal
	}
	switch p.Action {
	case "create":
		if next.Current {
			return nil, errors.New("goal_already_exists")
		}
		next = desktopGoal{ID: p.RequestID, Objective: p.Objective, Status: "active", Current: true, Generation: 1,
			Version: next.Version, Limits: desktopGoalLimit{Tokens: l.settings.goalBudget()}}
	case "edit":
		next.Objective = p.Objective
		next.Generation++
	case "pause":
		next.Status, next.Reason = "paused", "user_paused"
	case "resume":
		if next.Status == "active" || next.Usage.Tokens >= next.Limits.Tokens {
			return nil, errors.New("goal_cannot_resume")
		}
		next.Status, next.Reason = "active", ""
	case "clear":
		next.Current, next.Status = false, "cleared"
	default:
		return nil, errors.New("invalid_goal_action")
	}
	next.Version++
	row.Goal, row.LastGoalRequest = &next, p.RequestID
	if err := l.save(row); err != nil {
		return nil, errors.New("session_write_failed")
	}
	return map[string]any{"goal": row.Goal}, nil
}

func goalModelTokens(usage modelUsage) (int64, bool) {
	if usage.Input == nil || usage.Output == nil || *usage.Input < 0 || *usage.Output < 0 ||
		*usage.Input > maxGoalBudget || *usage.Output > maxGoalBudget {
		return 0, false
	}
	return min(*usage.Input+*usage.Output, maxGoalBudget), true
}

type desktopGoalRun struct {
	remaining int64
	active    func() bool
	outcome   string
}

func (g *desktopGoalRun) stop(usage modelUsage) bool {
	switch tokens, known := goalModelTokens(usage); {
	case !g.active():
		g.outcome = "user_paused"
	case !known:
		g.outcome = "usage_unknown"
	case tokens >= g.remaining:
		g.outcome = "token_budget"
	}
	return g.outcome != ""
}

func settleDesktopGoal(row *acpSession, original *desktopGoal, turn acpTurn) {
	if original == nil || original.Status != "active" || row.Goal == nil || row.Goal.ID != original.ID {
		return
	}
	next := *row.Goal
	tokens, known := goalModelTokens(turn.Usage)
	next.Usage.Tokens = min(next.Usage.Tokens+tokens, maxGoalBudget)
	next.Uncertain = next.Uncertain || !known
	next.ActiveSeconds += int64(turn.Ended.Sub(turn.Started) / time.Second)
	if next.Current && next.Status == "active" {
		switch {
		case turn.Failed:
			next.Status, next.Reason = "blocked", "execution_failure"
		case turn.goalOutcome == "completed":
			next.Current, next.Status, next.Reason = false, "completed", ""
		case turn.goalOutcome == "token_budget":
			next.Status, next.Reason = "budget_limited", "token_budget"
		default:
			next.Status, next.Reason = "paused", turn.goalOutcome
		}
	}
	next.Version++
	row.Goal = &next
}
