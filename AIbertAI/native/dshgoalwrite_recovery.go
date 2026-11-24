package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"slices"
)

type goalWorkspaceClaimEntry struct {
	claim dshConnectorClaim
	index int
}

// goalWorkspaceClaims selects every proven write in the unfinished Goal when
// no pending marker exists; a live marker always identifies exactly one claim.
func goalWorkspaceClaims(root *os.Root, row acpSession) ([]goalWorkspaceClaimEntry, bool, error) {
	fail := errors.New("dsh_goal_projection_unverified")
	if root == nil || row.Goal == nil || !row.Goal.Current || row.GoalRun == "" || len(row.Turns) == 0 {
		return nil, false, fail
	}
	turn := row.Turns[len(row.Turns)-1]
	if turn.Operation != row.GoalRun || turn.GoalCheckpoint == nil || turn.GoalCheckpoint.Pending != nil || !reflect.DeepEqual(turn.GoalCheckpoint.Runs, turn.ToolRuns) {
		return nil, false, fail
	}
	pending, err := readConnectorClaim(root, "dsh-connector-dispatch/"+row.ID+"/.pending", row.ID)
	marker := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, false, fail
	}
	if !marker && dshConnectorAssertSettled(root, row.ID) != nil {
		return nil, false, fail
	}
	var selected []goalWorkspaceClaimEntry
	seen := map[string]bool{}
	seenCurrent := false
	lastCallSeq := int64(-1)
	for i, run := range turn.ToolRuns {
		if run.DSHCallID == "" || run.Call == nil || seen[run.DSHCallID] {
			return nil, false, fail
		}
		seen[run.DSHCallID] = true
		if run.Pending && run.Call.Function.Name != "write" && run.Call.Function.Name != "edit" {
			return nil, false, fail
		}
		if marker && run.DSHCallID != pending.CallID {
			if run.Pending {
				return nil, false, fail
			}
			continue
		}
		if !marker && (run.Call.Function.Name != "write" && run.Call.Function.Name != "edit" || run.Report.Failed != 0) {
			if run.Pending {
				return nil, false, fail
			}
			continue
		}
		candidate, e := readConnectorClaim(root, connectorClaimPath(row.ID, turn.Operation, run.DSHCallID), row.ID)
		if e != nil || candidate.WorkspaceBoundary == nil {
			return nil, false, fail
		}
		w := candidate.WorkspaceBoundary
		if candidate.OperationID != turn.Operation || candidate.CallID != run.DSHCallID || candidate.Kind != "workspace" || candidate.Output == nil || (candidate.State != "result_ready" && candidate.State != "reconciled") || w.GoalID != row.Goal.ID || w.SDKGoalID != row.Goal.DSHGoalID || !validUUID(w.HistoryOperation) || w.Name != run.Call.Function.Name {
			return nil, false, fail
		}
		if marker && w.HistoryOperation != row.Goal.DSHQueryOp {
			return nil, false, fail
		}
		if !marker && (candidate.State != "reconciled" || candidate.SDKSeq == nil || run.Pending || w.CallSeq <= lastCallSeq) {
			return nil, false, fail
		}
		lastCallSeq = w.CallSeq
		seenCurrent = seenCurrent || w.HistoryOperation == row.Goal.DSHQueryOp
		selected = append(selected, goalWorkspaceClaimEntry{candidate, i})
	}
	if len(selected) == 0 || marker && (len(selected) != 1 || pending.OperationID != selected[0].claim.OperationID || pending.CallID != selected[0].claim.CallID) || !marker && !seenCurrent {
		return nil, false, fail
	}
	return selected, marker, nil
}

func goalWorkspaceClaim(root *os.Root, row acpSession) (dshConnectorClaim, int, error) {
	claims, _, err := goalWorkspaceClaims(root, row)
	if err != nil {
		return dshConnectorClaim{}, 0, err
	}
	return claims[0].claim, claims[0].index, nil
}

func (h *relaySessionHost) reconcileGoalWorkspace(ctx context.Context, snapshot acpSession) error {
	fail := errDSHGoalProjectionUnverified
	allowed := func() bool { return ctx.Err() == nil && h.local != nil }
	if !allowed() {
		return errRelayAccess
	}
	root := h.live.store.Root
	h.live.mu.Lock()
	cached, ok := h.live.sessions[snapshot.ID]
	oldBytes, err := json.Marshal(cached)
	h.live.mu.Unlock()
	var row acpSession
	if !ok || err != nil || json.Unmarshal(oldBytes, &row) != nil || row.Goal == nil || row.Goal.ID != snapshot.Goal.ID || row.Goal.Status != "reconciling" || row.Goal.hasPendingDSHInvocation() {
		return fail
	}
	claims, marker, err := goalWorkspaceClaims(root, row)
	if err != nil {
		return fail
	}
	turn := row.Turns[len(row.Turns)-1]
	state, _ := ctx.Value(dshBoundaryStateKey{}).(*dshBoundaryState)
	if state == nil || !state.capable || state.proof == nil || !marker && state.history == nil {
		return fail
	}
	projected := turn
	projected.ToolRuns = slices.Clone(turn.ToolRuns)
	var releaseClaim dshConnectorClaim
	var releaseSeq int64
	for _, item := range claims {
		claim, index := item.claim, item.index
		run := turn.ToolRuns[index]
		w := claim.WorkspaceBoundary
		if !marker {
			// Reconstruct the exact write-time prefix, including its Goal query
			// binding, even if the Goal has since issued another query.
			prefix := func(ctx context.Context, params map[string]any) (json.RawMessage, error) {
				copy := make(map[string]any, len(params)+1)
				for k, v := range params {
					copy[k] = v
				}
				copy["throughSeq"] = w.CallSeq
				return state.history(ctx, copy)
			}
			audited, e := dshAuditWorkspacePrefixForGoal(ctx, row.DSHSessionID, w.HistoryOperation, w.Call, w.Name, json.RawMessage(run.Call.Function.Arguments), prefix, w.SDKGoalID)
			if e != nil || audited.CallSeq != w.CallSeq || audited.Turn != w.Turn || audited.Step != w.Step || audited.ArgumentsSHA != w.ArgumentsSHA {
				return fail
			}
		}
		pages := make([]json.RawMessage, 0, 3)
		for _, part := range []string{"call", "result", "proof"} {
			raw, e := state.proof(ctx, map[string]any{"sessionId": row.DSHSessionID, "operationId": claim.OperationID, "callId": claim.CallID, "callSeq": w.CallSeq, "part": part})
			if e != nil {
				return fail
			}
			pages = append(pages, raw)
		}
		seq, e := dshVerifyWorkspaceBoundaryPages(*w, json.RawMessage(run.Call.Function.Arguments), *claim.Output, pages)
		if e != nil || !marker && (claim.SDKSeq == nil || *claim.SDKSeq != seq) {
			return fail
		}
		var page struct {
			Events []json.RawMessage `json:"events"`
		}
		if json.Unmarshal(pages[1], &page) != nil || len(page.Events) != 1 {
			return fail
		}
		raw, e := json.Marshal(map[string]any{"sessionId": row.DSHSessionID, "event": page.Events[0]})
		if e != nil {
			return fail
		}
		if run.Pending {
			update, e := projectDSHTool(&projected, raw)
			if e != nil || update == nil || projected.ToolRuns[index].Pending || projected.ToolRuns[index].Report.Failed != 0 {
				return fail
			}
		} else {
			check := acpTurn{Operation: turn.Operation, ToolRuns: []workspaceToolRun{run}}
			check.ToolRuns[0].Pending = true
			check.ToolRuns[0].Output = ""
			update, e := projectDSHTool(&check, raw)
			if e != nil || update == nil || check.ToolRuns[0].Output != run.Output || check.ToolRuns[0].Report.Failed != run.Report.Failed {
				return fail
			}
		}
		if marker {
			releaseClaim, releaseSeq = claim, seq
		}
	}

	checkpoint := *turn.GoalCheckpoint
	checkpoint.Runs = slices.Clone(projected.ToolRuns)
	projected.GoalCheckpoint = &checkpoint
	if !allowed() {
		return errRelayAccess
	}
	h.live.mu.Lock()
	current, ok := h.live.sessions[row.ID]
	currentBytes, _ := json.Marshal(current)
	if !ok || !bytes.Equal(oldBytes, currentBytes) || ctx.Err() != nil {
		h.live.mu.Unlock()
		return fail
	}
	current.Turns = slices.Clone(current.Turns)
	current.Turns[len(current.Turns)-1] = projected
	goal := *current.Goal
	goal.Uncertain = false
	goal.Version++
	current.Goal = &goal
	err = h.live.save(current)
	h.live.mu.Unlock()
	if err != nil {
		return errors.New("session_write_failed")
	}
	if !allowed() {
		return errRelayAccess
	}
	if marker {
		if err = dshReleaseSideEffect(root, releaseClaim, releaseSeq); err != nil {
			return fail
		}
	}

	if err := dshConnectorAssertSettled(root, row.ID); err != nil {
		return fail
	}
	return nil
}
