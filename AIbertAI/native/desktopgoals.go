package main

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type desktopGoal struct {
	ID             string                       `json:"goal_id"`
	Objective      string                       `json:"objective"`
	Status         string                       `json:"status"`
	Reason         string                       `json:"reason_code,omitempty"`
	ExecutionID    string                       `json:"current_execution_id,omitempty"`
	Generation     int64                        `json:"generation"`
	Version        int64                        `json:"state_version"`
	Current        bool                         `json:"is_current"`
	ActiveSeconds  int64                        `json:"active_seconds"`
	DSHActiveSince *time.Time                   `json:"dsh_active_since,omitempty"`
	Uncertain      bool                         `json:"usage_uncertain"`
	Usage          desktopGoalUsage             `json:"usage"`
	Limits         desktopGoalLimit             `json:"effective_limits"`
	Invocations    map[string]dshGoalInvocation `json:"dsh_invocations,omitempty"`
	DSHGoalID      string                       `json:"dsh_goal_id,omitempty"`
	DSHCommandID   string                       `json:"dsh_command_id,omitempty"`
	DSHQueryOp     string                       `json:"dsh_query_operation,omitempty"`
	DSHObjective   string                       `json:"dsh_objective,omitempty"`
	DSHRevision    int64                        `json:"dsh_revision,omitempty"`
	DSHEventSeq    int64                        `json:"dsh_event_seq,omitempty"`
	DSHReadProjOp  string                       `json:"dsh_read_projection_operation,omitempty"`
	DSHReadProjSeq *int64                       `json:"dsh_read_projection_seq,omitempty"`
	DSHPhase       string                       `json:"dsh_phase,omitempty"`
	RoundsStarted  int                          `json:"rounds_started,omitempty"`
	RecentProgress string                       `json:"recent_progress,omitempty"`
	// Observed marks a goal the model created on its own in DSH. DSH owns it:
	// the record exists so the employee can see the goal, and every platform
	// rule that treats a goal as owning the session must ignore it.
	Observed bool `json:"dsh_observed,omitempty"`
}

// An admitted DSH invocation remains pending until its committed usage has
// been persisted. A pending invocation must never be admitted again.
type dshGoalInvocation struct {
	Settled bool   `json:"settled"`
	Tokens  int64  `json:"tokens,omitempty"`
	Input   *int64 `json:"input_tokens,omitempty"`
	Output  *int64 `json:"output_tokens,omitempty"`
}

func (g *desktopGoal) hasPendingDSHInvocation() bool {
	for _, call := range g.Invocations {
		if !call.Settled {
			return true
		}
	}
	return false
}

// A goal owns the session only while it can advance on its own. A paused,
// blocked or finished goal keeps its record so it can be resumed, but it must
// not keep the user out of the session: GoalRun alone blocked every prompt.
func desktopGoalAdvancing(goal *desktopGoal) bool {
	return goal != nil && !goal.Observed && goal.Current && (goal.Status == "active" || goal.Status == "reconciling" || goal.Status == "clearing")
}

// platformGoal reports the goal this app created and drives. A goal the model
// created in DSH is display-only, so every rule about who owns the session
// must read through this accessor instead of row.Goal.
func (s acpSession) platformGoal() *desktopGoal {
	if s.Goal == nil || s.Goal.Observed {
		return nil
	}
	return s.Goal
}

type desktopGoalUsage struct {
	Tokens int64 `json:"charged_tokens"`
}

type desktopGoalLimit struct {
	Tokens int64 `json:"token_budget"`
}

func (l *acpLive) saveGoalAction(raw json.RawMessage, background bool) (any, error) {
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
		return map[string]any{"goal": goalView(row), "background": background}, nil
	}
	if row.Engine == "dsh" && !background && (goal == nil || !goal.Observed) {
		return nil, errors.New("dsh_goal_control_unavailable")
	}
	// The model's own DSH goal belongs to DSH, which has no platform-side pause
	// or edit. Only dismissing the record is honest here; offering pause would
	// show "paused" while the SDK kept advancing the goal.
	observed := goal != nil && goal.Observed
	if observed && p.Action != "clear" {
		return nil, errors.New("dsh_goal_control_unavailable")
	}
	if p.Action != "create" {
		switch {
		case goal == nil:
			return nil, errors.New("stale_goal")
		case observed:
			// Stopping a goal DSH owns is idempotent: the button may fire again
			// after the record already moved on, and a second delete must not
			// raise a red "the goal changed" warning.
			if p.ExpectedGoalID != "" && p.ExpectedGoalID != goal.ID {
				return nil, errors.New("stale_goal")
			}
		case !goal.Current || p.ExpectedGoalID != goal.ID || p.ExpectedGeneration != goal.Generation:
			return nil, errors.New("stale_goal")
		}
	}
	if row.Engine == "dsh" && goal != nil && p.Action == "edit" && (goal.Status == "reconciling" || goal.Status == "clearing" || goal.DSHPhase == "complete") {
		return nil, errors.New("goal_session_busy")
	}
	if row.State == "running" && p.Action != "pause" && p.Action != "clear" {
		return nil, errors.New("goal_session_busy")
	}
	if p.Action == "create" || p.Action == "edit" {
		p.Objective = strings.TrimSpace(p.Objective)
		if p.Objective == "" || utf8.RuneCountInString(p.Objective) > 4000 {
			return nil, errors.New("invalid_goal_objective")
		}
		if row.Engine == "dsh" && p.Action == "create" && dshReservedGoalObjective(p.Objective) {
			return nil, errors.New("invalid_goal_objective")
		}
	}
	var next desktopGoal
	if goal != nil {
		next = *goal
	}
	switch p.Action {
	case "create":
		if next.Current || next.hasPendingDSHInvocation() || row.Engine == "dsh" && next.Uncertain {
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
		if next.Status == "active" || next.Status == "reconciling" || next.Status == "clearing" || goalView(row).Usage.Tokens >= next.Limits.Tokens && next.DSHPhase != "complete" {
			return nil, errors.New("goal_cannot_resume")
		}
		workspaceRecovery := false
		if row.Engine == "dsh" && (next.Uncertain || next.Reason == "host_restarted") && l.store != nil && l.store.Root != nil {
			_, _, err := goalWorkspaceClaim(l.store.Root, row)
			workspaceRecovery = err == nil
		}
		if row.Engine == "dsh" && next.Reason == "host_restarted" && !workspaceRecovery && len(row.Turns) > 0 {
			for _, run := range row.Turns[len(row.Turns)-1].ToolRuns {
				if run.Call != nil && run.Report.Failed == 0 && (run.Call.Function.Name == "write" || run.Call.Function.Name == "edit") {
					return nil, errors.New("goal_cannot_resume")
				}
			}
		}
		if workspaceRecovery {
			next.Status = "reconciling"
		} else if row.Engine == "dsh" && next.DSHPhase == "complete" {
			if next.DSHCommandID == "" || next.Reason == "resume_confirmation" {
				return nil, errors.New("goal_cannot_resume")
			}
			next.Status = "reconciling"
		} else if row.Engine == "dsh" && (next.hasPendingDSHInvocation() && next.Reason == "usage_unknown" || next.Reason == "host_restarted" && validUUID(next.DSHQueryOp)) {
			next.Status = "reconciling"
		} else {
			if next.hasPendingDSHInvocation() || row.Engine == "dsh" && next.Uncertain {
				return nil, errors.New("goal_cannot_resume")
			}
			next.Status, next.Reason = "active", ""
		}
	case "clear":
		if row.Engine == "dsh" && background && (next.DSHGoalID != "" || row.State == "running") {
			next.Status, next.Reason = "clearing", ""
			if row.GoalRun == "" {
				row.GoalRun = p.RequestID
			}
			break
		}
		next.Current, next.Status = false, "cleared"
		if row.GoalRun != "" && row.State != "running" {
			if n := len(row.Turns); n != 0 && row.Turns[n-1].Operation == row.GoalRun {
				row.Turns = append([]acpTurn(nil), row.Turns...)
				turn := &row.Turns[n-1]
				active := turn.GoalCheckpoint
				materializeGoalCheckpoint(turn)
				turn.Ended, turn.Interrupted = time.Now().UTC(), true
				row.Updated = turn.Ended
				if row.Engine != "dsh" {
					tokens, known := goalModelTokens(turn.Usage)
					next.Usage.Tokens = min(next.Usage.Tokens+tokens, maxGoalBudget)
					next.Uncertain = next.Uncertain || !known
				}
				if active != nil {
					next.ActiveSeconds += int64(active.ActiveTime / time.Second)
				}
			}
			row.GoalRun = ""
		}
	default:
		return nil, errors.New("invalid_goal_action")
	}
	if background && (p.Action == "create" || p.Action == "resume") && row.GoalRun == "" {
		row.GoalRun = p.RequestID
	}
	next.Version++
	row.Goal, row.LastGoalRequest = &next, p.RequestID
	if err := l.save(row); err != nil {
		return nil, errors.New("session_write_failed")
	}
	if (background || observed) && (p.Action == "pause" || p.Action == "clear") {
		if cancel := l.activeTurns[row.ID]; cancel != nil {
			cancel()
		}
	}
	return map[string]any{"goal": goalView(row), "background": background}, nil
}

func dshReservedGoalObjective(objective string) bool {
	if strings.TrimFunc(objective, func(r rune) bool { return unicode.IsSpace(r) || r == '\ufeff' }) != objective {
		return true
	}
	lower := strings.ToLower(objective)
	if lower == "pause" || lower == "resume" || lower == "clear" || lower == "edit" {
		return true
	}
	if strings.HasPrefix(lower, "edit") && len(lower) > len("edit") {
		r, _ := utf8.DecodeRuneInString(lower[len("edit"):])
		return unicode.IsSpace(r) || r == '\ufeff'
	}
	return false
}

func goalModelTokens(usage modelUsage) (int64, bool) {
	if usage.Input == nil || usage.Output == nil || *usage.Input < 0 || *usage.Output < 0 ||
		*usage.Input > maxGoalBudget || *usage.Output > maxGoalBudget {
		return 0, false
	}
	return min(*usage.Input+*usage.Output, maxGoalBudget), true
}

// A checkpoint belongs to one unfinished turn. Keep the Responses replay items
// separate from the Chat Completions wire representation.
type goalReplayMessage struct {
	toolConversationMessage
	Items []json.RawMessage `json:"response_items,omitempty"`
}

type goalCheckpoint struct {
	Permissions reportedPermissions `json:"permissions,omitzero"`
	Tail        []goalReplayMessage `json:"tail,omitempty"`
	Runs        []workspaceToolRun  `json:"runs,omitempty"`
	Usage       modelUsage          `json:"usage"`
	ActiveTime  time.Duration       `json:"active_time_ns,omitempty"`
	Progress    string              `json:"progress,omitempty"`
	Rounds      int                 `json:"rounds,omitempty"`
	Pending     *goalReplayMessage  `json:"pending,omitempty"`
	Unknown     []modelToolUse      `json:"unknown,omitempty"`

	DSHParts              []dshMessagePart     `json:"dshParts,omitempty"`
	DSHThoughtProof       *dshGoalThoughtProof `json:"dshThoughtProof,omitempty"`
	DSHProgressInvocation string               `json:"dshProgressInvocation,omitempty"`
	DSHProgressQuery      string               `json:"dshProgressQuery,omitempty"`
}

const goalContinuePrompt = "Continue the current goal from the recorded progress and tool results. Do not repeat work already completed. Take the next useful step and keep the same completion rule."
const maxBackgroundGoalRounds = 64
const maxBackgroundGoalTools = 24

type desktopGoalRun struct {
	checkpoint *goalCheckpoint
	save       func(goalCheckpoint) error
	remaining  int64
	active     func() bool
	outcome    string
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
	if original == nil || original.Observed || original.Status != "active" || row.Goal == nil || row.Goal.ID != original.ID {
		return
	}
	next := *row.Goal
	if row.Engine != "dsh" {
		tokens, known := goalModelTokens(turn.Usage)
		next.Usage.Tokens = min(next.Usage.Tokens+tokens, maxGoalBudget)
		next.Uncertain = next.Uncertain || !known
	}
	activeTime := turn.Ended.Sub(turn.Started)
	if turn.GoalCheckpoint != nil {
		activeTime = turn.GoalCheckpoint.ActiveTime
	}
	next.ActiveSeconds += int64(activeTime / time.Second)
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
