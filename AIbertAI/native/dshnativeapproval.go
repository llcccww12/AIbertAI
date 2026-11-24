package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/coder/websocket"
)

// DSH accepts read-only, workspace-write and danger-full-access, and requires a
// justification whenever an escalation is requested. An ordinary call omits both.
func dshSandboxEscalationValid(mode, justification any) bool {
	if mode == nil {
		return justification == nil
	}
	value, ok := mode.(string)
	if !ok {
		return false
	}
	switch value {
	case "read-only", "workspace-write", "danger-full-access":
	default:
		return false
	}
	text, ok := justification.(string)
	return ok && strings.TrimSpace(text) != ""
}

const dshApprovalNotExecuted = "Approval unavailable; the tool was not executed."

type dshNativeApprovalKey struct{}
type dshNativeApproval func(json.RawMessage, *acpTurn) (string, error)

// A native approval has no arguments of its own. It is usable only when one
// pending SDK call and its persisted platform projection agree exactly.
func (h *relaySessionHost) dshNativeApproval(ctx context.Context, ws *websocket.Conn, request *http.Request, session acpSession) dshNativeApproval {
	seen := map[string]bool{}
	return func(raw json.RawMessage, turn *acpTurn) (string, error) {
		var req struct{ SessionID, OperationID, ToolName, CallID, Reason string }
		if _, err := strictDSHJSON(raw); err != nil || json.Unmarshal(raw, &req) != nil || req.SessionID != session.DSHSessionID || (req.ToolName != "bash" && req.ToolName != dshPluginManagerTool) || req.CallID == "" || len(req.CallID) > 1024 || seen[req.CallID] || turn == nil {
			return "rejected", nil
		}
		seen[req.CallID] = true
		var run *workspaceToolRun
		for i := range turn.ToolRuns {
			if turn.ToolRuns[i].DSHCallID == req.CallID {
				if run != nil {
					return "rejected", nil
				}
				run = &turn.ToolRuns[i]
			}
		}
		if run == nil || !run.Pending || run.Call == nil || run.Call.Function.Name != req.ToolName || run.Call.ID == "" {
			return "rejected", nil
		}
		if len(run.Call.Function.Arguments) > 64<<10 {
			return "rejected", nil
		}
		canonical, err := strictDSHJSON([]byte(run.Call.Function.Arguments))
		if err != nil || len(canonical) == 0 || len(canonical) > 64<<10 || canonical[0] != '{' {
			return "rejected", nil
		}
		var input map[string]any
		dec := json.NewDecoder(bytes.NewReader(canonical))
		dec.UseNumber()
		if dec.Decode(&input) != nil {
			return "rejected", nil
		}
		manager := req.ToolName == dshPluginManagerTool
		title, kind := req.Reason, "execute"
		if manager {
			title, kind = dshPluginApprovalTitle(input), "other"
			if title == "" || !validUUID(turn.Operation) || req.OperationID != turn.Operation || !h.dshPluginManagerBound(ctx, session) {
				return "rejected", nil
			}
		} else if !dshSandboxEscalationValid(input["sandbox_permissions"], input["justification"]) {
			return "rejected", nil
		} else if title == "" {
			title = "执行命令"
		}
		if h.live == nil || h.live.workspaces == nil || h.acpScope(request.WithContext(ctx), "files:write") != nil {
			return "rejected", nil
		}
		valid := func(row acpSession) bool {
			if manager && (row.ID != session.ID || row.ExpertID != session.ExpertID || row.ResourceRevision != session.ResourceRevision) {
				return false
			}
			if row.DSHSessionID != session.DSHSessionID || row.CWD != session.CWD || row.Workspace != session.Workspace || row.ProjectID != session.ProjectID || len(row.Turns) == 0 {
				return false
			}
			var found *workspaceToolRun
			for i := range row.Turns[len(row.Turns)-1].ToolRuns {
				candidate := &row.Turns[len(row.Turns)-1].ToolRuns[i]
				if candidate.DSHCallID == req.CallID {
					if found != nil {
						return false
					}
					found = candidate
				}
			}
			if found == nil || !found.Pending || found.Call == nil || found.Call.ID != run.Call.ID || found.Call.Function.Name != req.ToolName {
				return false
			}
			stored, err := strictDSHJSON([]byte(found.Call.Function.Arguments))
			return err == nil && bytes.Equal(stored, canonical)
		}
		current, err := h.live.load(session.ID)
		if err != nil || !valid(current) {
			return "rejected", nil
		}
		_, background := ctx.Value(goalRuntimeContext{}).(*goalRuntime)
		viewerDone, _ := ctx.Value(acpViewerDoneKey{}).(<-chan struct{})
		trusted := !manager && h.local != nil && !background && viewerDone != nil && current.permissionMode() == "bypassPermissions"
		if !trusted {
			approvalCtx := context.WithValue(ctx, dshNativeApprovalWaitKey{}, true)
			counts, err := h.requestDSHToolApproval(approvalCtx, ws, session.ID, run.Call.ID, title, req.ToolName, kind, input, false)
			turn.Permissions.Asked += counts.Asked
			turn.Permissions.Allowed += counts.Allowed
			turn.Permissions.Denied += counts.Denied
			if err != nil {
				if err.Error() == "tool_permission_denied" {
					return "rejected", nil
				}
				// No grant has been sent to the SDK, so execution did not start.
				run.Pending, run.Report.Failed, run.Output = false, 1, dshApprovalNotExecuted
				if background && ctx.Err() == nil {
					return "rejected", errDSHGoalApprovalUnavailable
				}
				return "rejected", err
			}
		}
		if h.acpScope(request.WithContext(ctx), "files:write") != nil || manager && !h.dshPluginManagerBound(ctx, session) {
			return "rejected", nil
		}
		effect, err := h.beginDSHSideEffect(ctx, session.ID, turn.Operation, req.CallID, func(row acpSession) bool {
			if trusted {
				select {
				case <-viewerDone:
					return false
				default:
				}
				if row.permissionMode() != "bypassPermissions" {
					return false
				}
			}
			return valid(row)
		})
		if err != nil {
			return "rejected", err
		}
		claim, err := connectorPending(effect.root, effect.session, effect.operation, effect.call)
		if err != nil || claim.State != "dispatch_claimed" {
			return "rejected", errors.New("dsh_connector_outcome_unknown")
		}
		claim.Kind = "native"
		if writeConnectorClaim(effect.root, claim) != nil {
			return "rejected", errors.New("dsh_connector_outcome_unknown")
		}
		return "allowed-once", nil
	}
}

// Verify that the live call is also the unique durable SDK call with this ID.
// A reused ID cannot be bound to a one-shot approval safely.
func dshNativeApprovalHistory(ctx context.Context, session string, raw json.RawMessage, turn *acpTurn, fetch dshConnectorHistoryFetch) error {
	var req struct{ SessionID, ToolName, CallID string }
	if json.Unmarshal(raw, &req) != nil || req.SessionID != session || (req.ToolName != "bash" && req.ToolName != dshPluginManagerTool) || req.CallID == "" || turn == nil {
		return errors.New("dsh_native_approval_unverified")
	}
	var run *workspaceToolRun
	for i := range turn.ToolRuns {
		if turn.ToolRuns[i].DSHCallID == req.CallID {
			if run != nil {
				return errors.New("dsh_native_approval_unverified")
			}
			run = &turn.ToolRuns[i]
		}
	}
	if run == nil || !run.Pending || run.Call == nil || run.Call.Function.Name != req.ToolName {
		return errors.New("dsh_native_approval_unverified")
	}
	args, err := strictDSHJSON([]byte(run.Call.Function.Arguments))
	if err != nil {
		return err
	}
	count := 0
	_, err = dshRepairHistory(ctx, session, -1, fetch, func(_ int64, raw json.RawMessage) error {
		var event struct {
			Type string
			Data struct{ CallID, Name, Arguments string }
		}
		if json.Unmarshal(raw, &event) != nil {
			return errors.New("dsh_native_approval_unverified")
		}
		if event.Type == "tool/call" && event.Data.CallID == req.CallID {
			count++
			found, e := strictDSHJSON([]byte(event.Data.Arguments))
			if e != nil || event.Data.Name != req.ToolName || !bytes.Equal(found, args) {
				return errors.New("dsh_native_approval_unverified")
			}
		}
		return nil
	})
	if err != nil || count != 1 {
		return errors.New("dsh_native_approval_unverified")
	}
	return nil
}
