package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/coder/websocket"
)

func (l *acpLive) recordDSHGoalCommand(sessionID, goalID, commandID string) error {
	if !strings.HasPrefix(commandID, "cmd-") || len(commandID) > 256 {
		return errors.New("dsh_invalid_goal_command")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	row, ok := l.sessions[sessionID]
	if !ok || row.Goal == nil || row.Goal.ID != goalID || !row.Goal.Current {
		return errors.New("goal_not_found")
	}
	goal := *row.Goal
	goal.DSHCommandID = commandID
	goal.Version++
	row.Goal = &goal
	if err := l.save(row); err != nil {
		return errors.New("session_write_failed")
	}
	return nil
}

// runDSHGoalTurn keeps platform and SDK goal identities separate and sends
// commands only through the SDK command path. A lease serializes each session.
func (h *relaySessionHost) runDSHGoalTurn(ctx context.Context, ws *websocket.Conn, request *http.Request, session acpSession, turn *acpTurn, resume, allowWrites bool, project func(map[string]any) error) error {
	if session.Goal == nil || !session.Goal.Current || session.Goal.Status != "active" {
		return errors.New("goal_not_active")
	}
	goalID := session.Goal.ID
	h.live.mu.Lock()
	current := h.live.sessions[session.ID]
	if current.Goal == nil || current.Goal.ID != goalID || current.Goal.Status != "active" {
		h.live.mu.Unlock()
		return errors.New("goal_not_active")
	}
	started := time.Now().UTC()
	active := *current.Goal
	active.DSHActiveSince = &started
	active.Version++
	current.Goal = &active
	err := h.live.save(current)
	h.live.mu.Unlock()
	if err != nil {
		return errors.New("session_write_failed")
	}
	observer := &dshGoalObserver{live: h.live, sessionID: session.ID, goalID: goalID}
	ctx = context.WithValue(ctx, dshGoalAdmissionKey{}, func(invocationID string) error {
		return h.live.admitDSHGoalInvocation(session.ID, goalID, invocationID)
	})
	options := dshSDKOptions{Query: true, Observe: func(method string, raw json.RawMessage) error { return observer.observe(method, raw) }}
	originalPrompt := turn.Prompt
	sdkTurn := *turn
	sdkTurn.ModelInput = ""
	sdkTurn.DSHGoalID = goalID

	// Live deltas stay on sdkTurn. Only a matching committed receipt may copy
	// their parts into the durable turn and checkpoint.
	snapshot := func(update map[string]any) {
		copyTurn := sdkTurn
		copyTurn.Prompt = originalPrompt
		copyTurn.ToolRuns = slices.Clone(sdkTurn.ToolRuns)
		for i := range copyTurn.ToolRuns {
			if !validDSHGoalThoughtProof(copyTurn.ToolRuns[i].DSHThoughtProof, goalID, copyTurn.ToolRuns[i].Preamble, copyTurn.ToolRuns[i].DSHParts) {
				copyTurn.ToolRuns[i].DSHParts = nil
				copyTurn.ToolRuns[i].DSHThoughtProof = nil
			}
		}
		if !validDSHGoalThoughtProof(copyTurn.DSHThoughtProof, goalID, copyTurn.Answer, copyTurn.DSHParts) {
			copyTurn.DSHParts, copyTurn.DSHThoughtProof = nil, nil
		}
		if sdkTurn.GoalCheckpoint != nil {
			checkpoint := *sdkTurn.GoalCheckpoint
			checkpoint.Runs = slices.Clone(copyTurn.ToolRuns)
			switch update["sessionUpdate"] {
			case "agent_message_settled", "tool_call", "tool_call_update":
				checkpoint.Progress = sdkTurn.Answer
				checkpoint.DSHParts, checkpoint.DSHThoughtProof = nil, nil
				checkpoint.DSHProgressInvocation, checkpoint.DSHProgressQuery = "", ""
				if update["sessionUpdate"] == "agent_message_settled" && validUUID(sdkTurn.dshCurrentInvocation) && session.Goal != nil && validUUID(session.Goal.DSHQueryOp) {
					checkpoint.DSHProgressInvocation, checkpoint.DSHProgressQuery = sdkTurn.dshCurrentInvocation, session.Goal.DSHQueryOp
				}
			}
			if validDSHGoalThoughtProof(copyTurn.DSHThoughtProof, goalID, copyTurn.Answer, copyTurn.DSHParts) && checkpoint.Progress == copyTurn.Answer {
				checkpoint.DSHParts, checkpoint.DSHThoughtProof = copyTurn.DSHParts, copyTurn.DSHThoughtProof
			}
			copyTurn.GoalCheckpoint = &checkpoint
			sdkTurn.GoalCheckpoint = &checkpoint
		}
		*turn = copyTurn
	}
	projectUpdate := func(update map[string]any) error {
		snapshot(update)
		return project(update)
	}
	observer.onThought = func(candidate dshGoalThoughtCandidate) error {
		streamAttempt := ""
		if len(candidate.calls) == 0 {
			if sdkTurn.dshCurrentInvocation != candidate.proof.InvocationID || sdkTurn.Answer != candidate.text {
				return nil
			}
			sdkTurn.DSHParts, sdkTurn.DSHThoughtProof = candidate.parts, &candidate.proof
			streamAttempt = sdkTurn.dshStreamAttempt
		} else {
			found := -1
			for i := range sdkTurn.ToolRuns {
				run := &sdkTurn.ToolRuns[i]
				if run.DSHCallID == candidate.calls[0].id && run.dshSourceInvocation == candidate.proof.InvocationID && run.Preamble == candidate.text && run.Call != nil && run.Call.Function.Name == candidate.calls[0].name && run.Call.Function.Arguments == candidate.calls[0].arguments {
					if found >= 0 {
						return nil
					}
					found = i
				}
			}
			if found < 0 {
				return nil
			}
			sdkTurn.ToolRuns[found].DSHParts, sdkTurn.ToolRuns[found].DSHThoughtProof = candidate.parts, &candidate.proof
			streamAttempt = sdkTurn.ToolRuns[found].dshStreamAttempt
		}
		if err := projectUpdate(nil); err != nil {
			return err
		}
		if streamAttempt != "" {
			return project(map[string]any{"sessionUpdate": "agent_message_settled", "content": map[string]string{"type": "text", "text": candidate.text}, "streamAttemptId": streamAttempt, "_meta": map[string]any{"dshParts": dshRenderedParts(candidate.parts, candidate.text)}})
		}
		return nil
	}
	invoke := func(command string, restore bool) error {
		sdkTurn.Prompt = command
		queryOptions := options
		if strings.HasPrefix(command, "/goal edit ") {
			queryOptions.SkipOperation = true
			queryOptions.LeaseControl = true
			queryOptions.OnCommand = func(id string) error {
				if !strings.HasPrefix(id, "cmd-") || len(id) > 256 {
					return errors.New("dsh_invalid_goal_command")
				}
				return nil
			}
		} else {
			queryOptions.OperationID = session.Goal.DSHQueryOp
			queryOptions.OnCommand = func(id string) error { return h.live.recordDSHGoalCommand(session.ID, goalID, id) }
		}
		queryCtx := context.WithValue(ctx, dshSDKOptionsKey{}, queryOptions)
		err := runDSHTurn(queryCtx, h.live.dsh, session, &sdkTurn, restore, projectUpdate)
		sdkTurn.Prompt = originalPrompt
		snapshot(nil)
		if errors.Is(err, errDSHGoalApprovalUnavailable) {
			turn.goalOutcome = "approval_unavailable"
			if saveErr := project(nil); saveErr != nil {
				return saveErr
			}
		}
		return err
	}
	goal := session.Goal
	if goal.DSHGoalID != "" && goal.DSHObjective == "" {
		return errors.New("dsh_goal_state_unverified")
	}
	if goal.DSHGoalID != "" && goal.DSHObjective != goal.Objective {
		if err := invoke("/goal edit "+goal.Objective, true); err != nil {
			return err
		}
		current, err := h.live.load(session.ID)
		if err != nil || current.Goal == nil || current.Goal.ID != goalID || current.Goal.DSHObjective != goal.Objective {
			return errors.New("dsh_goal_edit_unconfirmed")
		}
	}
	command := "/goal " + goal.Objective
	if goal.DSHGoalID != "" {
		command, resume = "/goal resume", true
	}
	queryOperation, err := newUUID()
	if err != nil {
		return err
	}
	h.live.mu.Lock()
	current = h.live.sessions[session.ID]
	if current.Goal == nil || current.Goal.ID != goalID || current.Goal.Status != "active" {
		h.live.mu.Unlock()
		return errors.New("goal_not_active")
	}
	queryGoal := *current.Goal
	queryGoal.DSHQueryOp, queryGoal.DSHCommandID = queryOperation, ""
	queryGoal.Version++
	current.Goal = &queryGoal
	err = h.live.save(current)
	h.live.mu.Unlock()
	if err != nil {
		return errors.New("session_write_failed")
	}
	session.Goal = &queryGoal
	// Bind to the working SDK turn so approval counts and call receipts survive
	// the next durable checkpoint. Local goals can use the desktop canvas.
	ctx, err = h.bindDSHRules(ctx)
	if err != nil {
		return err
	}
	if h.local != nil {
		bridges, instructions, err := h.dshTurnHostTools(ctx, ws, request, session, &sdkTurn)
		if err != nil {
			return err
		}
		if instructions != "" {
			ctx = context.WithValue(ctx, dshExpertInstructionsKey{}, instructions)
		}
		if len(bridges) > 0 {
			ctx = context.WithValue(ctx, dshHostToolBridgesKey{}, bridges)
		}
	} else {
		if len(h.dshSelectedKnowledge(session)) > 0 {
			return errors.New("dsh_knowledge_goal_unsupported")
		}
		if session.ExpertID == designExpertID {
			return errors.New("dsh_design_expert_unsupported")
		}
		bridges, err := h.dshRemoteTurnConnectorTools(ctx, ws, request, session, &sdkTurn)
		if err != nil {
			return err
		}
		if len(bridges) > 0 {
			ctx = context.WithValue(ctx, dshHostToolBridgesKey{}, bridges)
		}
	}
	if allowWrites {
		ctx = context.WithValue(ctx, dshWorkspaceBridgeKey{}, h.dshWorkspaceBridge(ctx, ws, session, &sdkTurn))
	}
	if err := invoke(command, resume); err != nil {
		if current, loadErr := h.live.load(session.ID); loadErr == nil && current.Goal != nil && current.Goal.DSHPhase == "blocked" {
			_ = h.live.prepareDSHGoalCompletion(session.ID, goalID)
		}
		return err
	}
	if err := h.live.prepareDSHGoalCompletion(session.ID, goalID); err != nil {
		return err
	}
	current, err = h.live.load(session.ID)
	if err != nil {
		return err
	}
	if current.Goal.Status == "blocked" {
		return errors.New("dsh_goal_blocked")
	}
	usage, err := goalReportedUsage(current.Goal, turn.Usage)
	if err != nil {
		return err
	}
	// The goal ledger replaces per-run accounting; its calls carry no cache split.
	turn.Usage, turn.Tokens = usage, turnTokenFacts{}
	if err := h.live.finishDSHGoal(session.ID, goalID, *turn); err != nil {
		return err
	}
	turn.goalOutcome = "completed"
	return nil
}

// A clear is serialized after any active goal query has stopped. The durable
// SDK clear event is persisted before the platform releases the session.
func (h *relaySessionHost) clearDSHGoal(ctx context.Context, session acpSession) error {
	if session.Goal == nil || session.Goal.Status != "clearing" {
		return errors.New("goal_not_clearing")
	}
	goalID := session.Goal.ID
	if session.Goal.DSHPhase != "cleared" {
		observer := &dshGoalObserver{live: h.live, sessionID: session.ID, goalID: goalID}
		ctx = context.WithValue(ctx, dshGoalAdmissionKey{}, func(string) error { return errors.New("goal_not_active") })
		ctx = context.WithValue(ctx, dshSDKOptionsKey{}, dshSDKOptions{Query: true, SkipOperation: true, LeaseControl: true, Observe: observer.observe, OnCommand: func(id string) error {
			if !strings.HasPrefix(id, "cmd-") || len(id) > 256 {
				return errors.New("dsh_invalid_goal_command")
			}
			return nil
		}})
		turn := acpTurn{Prompt: "/goal clear"}
		if err := runDSHTurn(ctx, h.live.dsh, session, &turn, true, nil); err != nil {
			return err
		}
	}
	h.live.mu.Lock()
	defer h.live.mu.Unlock()
	row, ok := h.live.sessions[session.ID]
	if !ok || row.Goal == nil || row.Goal.ID != goalID || row.Goal.Status != "clearing" || row.Goal.DSHPhase != "cleared" {
		return errors.New("dsh_goal_clear_unconfirmed")
	}
	goal := *row.Goal
	goal.Current, goal.Status, goal.Reason = false, "cleared", ""
	if n := len(row.Turns); n > 0 && row.Turns[n-1].Operation == row.GoalRun && row.Turns[n-1].Ended.IsZero() {
		row.Turns = slices.Clone(row.Turns)
		turn := &row.Turns[n-1]
		if turn.GoalCheckpoint != nil {
			goal.ActiveSeconds += int64(turn.GoalCheckpoint.ActiveTime / time.Second)
		}
		materializeGoalCheckpoint(turn)
		turn.Ended, turn.Interrupted = time.Now().UTC(), true
		row.Updated = turn.Ended
	}
	goal.Version++
	row.Goal, row.GoalRun, row.State = &goal, "", "idle"
	if err := h.live.save(row); err != nil {
		return errors.New("session_write_failed")
	}
	return nil
}

// Reconciliation is initiated by an explicit resume request. It never sends a
// model request, and it re-arms the goal only after all usage is durable.
func (h *relaySessionHost) reconcileDSHGoal(ctx context.Context, session acpSession) error {
	if session.Goal == nil || session.Goal.Status != "reconciling" {
		return errors.New("goal_not_reconciling")
	}
	goalID := session.Goal.ID
	observer := &dshGoalObserver{live: h.live, sessionID: session.ID, goalID: goalID}
	ctx = context.WithValue(ctx, dshGoalAdmissionKey{}, func(string) error { return errors.New("goal_not_active") })
	options := dshSDKOptions{Observe: observer.observe}
	var recoveredAnswer string
	goalWriteRecovery := false
	if (session.Goal.Uncertain || session.Goal.Reason == "host_restarted") && h.live.store != nil && h.live.store.Root != nil {
		_, _, e := goalWorkspaceClaim(h.live.store.Root, session)
		goalWriteRecovery = e == nil
	}
	options.AfterReconcile = func(ctx context.Context, p *dshProcess) error {
		if !goalWriteRecovery {
			// The SDK can stop inside an otherwise valid open turn. Recovering
			// a verified message is independent of replaying any tool.
			_ = h.live.recoverDSHGoalThoughts(ctx, p, session.ID, goalID)
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		if goalWriteRecovery {
			if err := h.reconcileGoalWorkspace(ctx, session); err != nil {
				return err
			}
		} else if err := h.live.repairDSHGoalReadProjection(ctx, func(ctx context.Context, params map[string]any) (json.RawMessage, error) {
			return p.call(ctx, "aibertai/history", params, nil)
		}, session.ID, goalID); err != nil {
			return err
		}
		// Reasoning is optional presentation: an old SDK or an unverifiable
		// prefix must never prevent the existing safe reconciliation path.
		_ = h.live.recoverDSHGoalThoughts(ctx, p, session.ID, goalID)
		return ctx.Err()
	}
	if session.Goal.DSHPhase == "complete" && !goalWriteRecovery {
		options.AfterReconcile = func(ctx context.Context, p *dshProcess) error {
			var err error
			recoveredAnswer, err = h.live.readDSHGoalCompletion(ctx, p, session.ID, goalID)
			if err != nil {
				return err
			}
			_ = h.live.recoverDSHGoalThoughts(ctx, p, session.ID, goalID)
			return ctx.Err()
		}
	}
	ctx = context.WithValue(ctx, dshSDKOptionsKey{}, options)
	err := reconcileDSHSession(ctx, h.live.dsh, session)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	h.live.mu.Lock()
	row, ok := h.live.sessions[session.ID]
	if !ok || row.Goal == nil || row.Goal.ID != goalID || row.Goal.Status != "reconciling" {
		h.live.mu.Unlock()
		return errors.New("goal_reconciliation_changed")
	}
	goal := *row.Goal
	if errors.Is(err, errDSHGoalProjectionUnverified) {
		goal.Status, goal.Reason, goal.Uncertain = "paused", "resume_confirmation", true
	} else if goal.DSHPhase == "complete" && err != nil && !goal.hasPendingDSHInvocation() && !goal.Uncertain {
		goal.Status, goal.Reason = "paused", "completion_unverified"
	} else if err != nil || goal.hasPendingDSHInvocation() || goal.Uncertain {
		goal.Status, goal.Reason, goal.Uncertain = "paused", "usage_unknown", true
		if err == nil {
			err = errors.New("goal_usage_reconciliation_required")
		}
	} else if goalWriteRecovery {
		goal.Status, goal.Reason = "paused", "write_reconciled"
	} else if goal.DSHPhase == "complete" {
		goal.Status, goal.Reason = "paused", "verification_required"
		row.State = "running"
	} else if goal.Usage.Tokens >= goal.Limits.Tokens {
		goal.Status, goal.Reason = "budget_limited", "token_budget"
	} else {
		goal.Status, goal.Reason = "active", ""
	}
	goal.Version++
	row.Goal = &goal
	if saveErr := h.live.save(row); saveErr != nil {
		h.live.mu.Unlock()
		return errors.New("session_write_failed")
	}
	h.live.mu.Unlock()
	if err == nil && goal.DSHPhase == "complete" {
		if finishErr := h.finishRecoveredDSHGoal(ctx, row, recoveredAnswer); finishErr != nil {
			h.live.mu.Lock()
			current := h.live.sessions[row.ID]
			if current.Goal != nil && current.Goal.ID == goalID {
				if current.Goal.Current && current.Goal.Status == "paused" && current.Goal.Reason == "verification_required" {
					current.State = "idle"
					_ = h.live.save(current)
				}
				if !current.Goal.Current && current.Goal.Status == "completed" && current.GoalRun != "" {
					g := *current.Goal
					g.Current, g.Status, g.Reason = true, "paused", "completion_unverified"
					g.Version++
					current.Goal, current.State = &g, "idle"
					_ = h.live.save(current)
				}
			}
			h.live.mu.Unlock()
			return finishErr
		}
		return nil
	}
	return err
}

func (h *relaySessionHost) finishRecoveredDSHGoal(ctx context.Context, row acpSession, answer string) error {
	if len(row.Turns) == 0 || row.Goal == nil {
		return errors.New("dsh_completion_evidence_unavailable")
	}
	index := len(row.Turns) - 1
	turn := row.Turns[index]
	turn.Answer, turn.Failed, turn.Interrupted = answer, false, false
	turn.DSHParts, turn.DSHThoughtProof = nil, nil
	if saved := turn.GoalCheckpoint; saved != nil && saved.Progress == answer && validDSHGoalThoughtProof(saved.DSHThoughtProof, turn.DSHGoalID, saved.Progress, saved.DSHParts) {
		turn.DSHParts, turn.DSHThoughtProof = saved.DSHParts, saved.DSHThoughtProof
	}
	usage, err := goalReportedUsage(row.Goal, turn.Usage)
	if err != nil {
		return err
	}
	// The goal ledger replaces per-run accounting; its calls carry no cache split.
	turn.Usage, turn.Tokens = usage, turnTokenFacts{}
	if err := h.live.finishDSHGoal(row.ID, row.Goal.ID, turn); err != nil {
		return err
	}
	h.live.mu.Lock()
	current := h.live.sessions[row.ID]
	if current.Goal == nil || current.Goal.ID != row.Goal.ID || current.Goal.Status != "completed" || len(current.Turns) != index+1 || current.Turns[index].Operation != turn.Operation {
		h.live.mu.Unlock()
		return errors.New("goal_completion_changed")
	}
	goal := *current.Goal
	if turn.GoalCheckpoint != nil {
		goal.ActiveSeconds += int64(turn.GoalCheckpoint.ActiveTime / time.Second)
	}
	turn.GoalCheckpoint = nil
	turn.Ended = time.Now().UTC()
	if turn.Ended.Before(turn.Started) {
		turn.Ended = turn.Started
	}
	current.Turns = slices.Clone(current.Turns)
	current.Turns[index] = turn
	current.Goal, current.GoalRun, current.State, current.Updated = &goal, "", "idle", turn.Ended
	if err := h.live.save(current); err != nil {
		h.live.mu.Unlock()
		return errors.New("session_write_failed")
	}
	h.live.mu.Unlock()
	if !turn.Acked {
		if err := h.submitACPTurn(ctx, row.ID, index+1, turn, index == 0); err == nil {
			h.live.mu.Lock()
			current = h.live.sessions[row.ID]
			if len(current.Turns) > index && current.Turns[index].Operation == turn.Operation {
				current.Turns = slices.Clone(current.Turns)
				current.Turns[index].Acked = true
				_ = h.live.save(current)
			}
			h.live.mu.Unlock()
		}
	}
	return nil
}
