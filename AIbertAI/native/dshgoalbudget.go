package main

import (
	"errors"
	"maps"
)

// admitDSHGoalInvocation is called before forwarding an invocation to the
// enterprise gateway. Its durable reservation prevents replay after a crash.
func (l *acpLive) admitDSHGoalInvocation(sessionID, goalID, invocationID string) error {
	if !validUUID(invocationID) {
		return errors.New("goal_invalid_invocation")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	row, ok := l.sessions[sessionID]
	if !ok || row.Engine != "dsh" || row.Goal == nil || row.Goal.ID != goalID {
		return errors.New("goal_invalid_binding")
	}
	goal := *row.Goal
	if _, exists := goal.Invocations[invocationID]; exists {
		return errors.New("goal_invocation_already_admitted")
	}
	if goal.hasPendingDSHInvocation() || goal.Uncertain {
		return errors.New("goal_usage_reconciliation_required")
	}
	if !goal.Current || goal.Status != "active" {
		return errors.New("goal_not_active")
	}
	if goal.Usage.Tokens >= goal.Limits.Tokens {
		return errors.New("goal_token_budget")
	}
	goal.Invocations = maps.Clone(goal.Invocations)
	if goal.Invocations == nil {
		goal.Invocations = make(map[string]dshGoalInvocation)
	}
	goal.Invocations[invocationID] = dshGoalInvocation{}
	goal.Version++
	row.Goal = &goal
	if err := l.save(row); err != nil {
		return errors.New("session_write_failed")
	}
	return nil
}

// settleDSHGoalInvocation commits usage exactly once, before the gateway
// permits another invocation. Unknown usage keeps the reservation unresolved.
func (l *acpLive) settleDSHGoalInvocation(sessionID, goalID, invocationID string, usage modelUsage) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	row, ok := l.sessions[sessionID]
	if !ok || row.Engine != "dsh" || row.Goal == nil || row.Goal.ID != goalID {
		return errors.New("goal_not_found")
	}
	goal := *row.Goal
	call, exists := goal.Invocations[invocationID]
	if !exists {
		return errors.New("goal_invocation_not_admitted")
	}
	tokens, known := goalModelTokens(usage)
	if call.Settled {
		if (call.Input == nil) != (call.Output == nil) {
			return errors.New("goal_invocation_usage_conflict")
		}
		if known && call.Tokens == tokens && (call.Input == nil || *call.Input == *usage.Input) && (call.Output == nil || *call.Output == *usage.Output) {
			return nil
		}
		return errors.New("goal_invocation_usage_conflict")
	}
	goal.Invocations = maps.Clone(goal.Invocations)
	if !known {
		goal.Uncertain = true
		if goal.Current && goal.Status == "active" {
			goal.Status, goal.Reason = "paused", "usage_unknown"
		}
	} else {
		call.Settled, call.Tokens = true, tokens
		input, output := *usage.Input, *usage.Output
		call.Input, call.Output = &input, &output
		goal.Invocations[invocationID] = call
		goal.Usage.Tokens = min(goal.Usage.Tokens+tokens, maxGoalBudget)
		if goal.Reason == "usage_unknown" && !goal.hasPendingDSHInvocation() {
			goal.Uncertain = false
		}
		if goal.Current && goal.Status == "active" && goal.Usage.Tokens >= goal.Limits.Tokens {
			goal.Status, goal.Reason = "budget_limited", "token_budget"
		}
	}
	goal.Version++
	row.Goal = &goal
	if err := l.save(row); err != nil {
		return errors.New("session_write_failed")
	}
	if !known {
		return errors.New("goal_usage_reconciliation_required")
	}
	return nil
}

// goalReportedUsage reconstructs exact input/output totals from the durable
// per-invocation ledger, including auxiliary compaction requests.
func goalReportedUsage(goal *desktopGoal, fallback modelUsage) (modelUsage, error) {
	if goal == nil || goal.hasPendingDSHInvocation() || goal.Uncertain {
		return modelUsage{}, errors.New("goal_usage_reconciliation_required")
	}
	var input, output int64
	complete := len(goal.Invocations) > 0
	for _, call := range goal.Invocations {
		if !call.Settled {
			return modelUsage{}, errors.New("goal_usage_reconciliation_required")
		}
		if (call.Input == nil) != (call.Output == nil) {
			return modelUsage{}, errors.New("goal_usage_reconciliation_required")
		}
		if call.Input == nil {
			complete = false
			continue
		}
		if *call.Input < 0 || *call.Output < 0 || *call.Input > maxGoalBudget || *call.Output > maxGoalBudget || call.Tokens != min(*call.Input+*call.Output, maxGoalBudget) {
			return modelUsage{}, errors.New("goal_usage_reconciliation_required")
		}
		input = min(input+*call.Input, maxGoalBudget)
		output = min(output+*call.Output, maxGoalBudget)
	}
	if complete && min(input+output, maxGoalBudget) == goal.Usage.Tokens {
		return modelUsage{Input: &input, Output: &output}, nil
	}
	// Older sessions recorded only totals. Their already-persisted aggregate is
	// usable only if it accounts for the complete goal ledger.
	if tokens, known := goalModelTokens(fallback); known && tokens == goal.Usage.Tokens {
		return fallback, nil
	}
	return modelUsage{}, errors.New("goal_usage_reconciliation_required")
}
