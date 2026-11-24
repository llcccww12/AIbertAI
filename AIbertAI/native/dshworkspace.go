package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/coder/websocket"
)

type dshWorkspaceBridgeKey struct{}
type dshWorkspaceAuthorizeKey struct{}
type dshMutation struct {
	SessionID string          `json:"sessionId"`
	CallID    string          `json:"callId"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type dshTargetState struct {
	Root   os.FileInfo
	Parent os.FileInfo
	File   os.FileInfo
}

func dshWorkspaceTarget(root *os.Root, relative string) (dshTargetState, error) {
	var state dshTargetState
	var err error
	if _, err = workspaceRelative(relative); err != nil || relative == "." {
		return state, errWorkspacePath
	}
	state.Root, err = root.Stat(".")
	if err != nil {
		return state, errWorkspacePath
	}
	parent := filepath.Dir(filepath.FromSlash(relative))
	parts := strings.Split(parent, string(filepath.Separator))
	current := "."
	for _, part := range parts {
		if part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := root.Lstat(current)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return state, errWorkspacePath
		}
	}
	p, err := root.OpenRoot(parent)
	if err != nil {
		return state, errWorkspacePath
	}
	defer p.Close()
	state.Parent, err = p.Stat(".")
	if err != nil {
		return state, errWorkspacePath
	}
	state.File, err = p.Lstat(filepath.Base(relative))
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil || !state.File.Mode().IsRegular() {
		return state, errWorkspacePath
	}
	return state, nil
}

// Reports whether relative names a file whose parent folders are partly
// missing, while every existing ancestor is a real directory (no symlinks).
func dshMissingWriteParent(root *os.Root, relative string) bool {
	if _, err := workspaceRelative(relative); err != nil || relative == "." {
		return false
	}
	current := "."
	for _, part := range strings.Split(filepath.Dir(filepath.FromSlash(relative)), string(filepath.Separator)) {
		if part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := root.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return true
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return false
		}
	}
	return false
}

// Counts added and removed lines like a diff summary: lines kept in order
// (longest common subsequence) are unchanged. Very large files fall back to a
// multiset comparison so the count stays cheap.
func workspaceLineChanges(before, after string) (added, deleted int) {
	split := func(text string) []string {
		if text == "" {
			return nil
		}
		return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	}
	a, b := split(before), split(after)
	for len(a) > 0 && len(b) > 0 && a[0] == b[0] {
		a, b = a[1:], b[1:]
	}
	for len(a) > 0 && len(b) > 0 && a[len(a)-1] == b[len(b)-1] {
		a, b = a[:len(a)-1], b[:len(b)-1]
	}
	if len(a) == 0 || len(b) == 0 || len(a)*len(b) > 4_000_000 {
		left := make(map[string]int, len(a))
		for _, line := range a {
			left[line]++
		}
		common := 0
		for _, line := range b {
			if left[line] > 0 {
				left[line]--
				common++
			}
		}
		return len(b) - common, len(a) - common
	}
	row := make([]int, len(b)+1)
	for i := range a {
		diagonal := 0
		for j := range b {
			above := row[j+1]
			if a[i] == b[j] {
				row[j+1] = diagonal + 1
			} else if row[j] > row[j+1] {
				row[j+1] = row[j]
			}
			diagonal = above
		}
	}
	return len(b) - row[len(b)], len(a) - row[len(b)]
}
func sameDSHTarget(a, b dshTargetState) bool {
	return os.SameFile(a.Root, b.Root) && os.SameFile(a.Parent, b.Parent) && (a.File == nil && b.File == nil || a.File != nil && b.File != nil && sameWorkspaceVersion(a.File, b.File))
}

func validDSHWorkspaceText(text *string, limit int) bool {
	return text != nil && len(*text) <= limit && utf8.ValidString(*text) && !strings.ContainsRune(*text, 0)
}

var errDSHGoalApprovalUnavailable = errors.New("dsh_goal_approval_unavailable")

// The SDK request is matched to an already projected, pending tool call. The
// path and immutable arguments shown to the user are the ones passed to the
// existing rooted workspace writer after approval.
func (h *relaySessionHost) dshWorkspaceBridge(ctx context.Context, ws *websocket.Conn, session acpSession, turn *acpTurn) func(json.RawMessage) (any, error) {
	seen := map[string]bool{}
	return func(raw json.RawMessage) (any, error) {
		var req dshMutation
		if json.Unmarshal(raw, &req) != nil || req.SessionID != session.DSHSessionID || req.CallID == "" || seen[req.CallID] || req.Name != "write" && req.Name != "edit" {
			return nil, errors.New("dsh_workspace_request_invalid")
		}
		seen[req.CallID] = true
		var run *workspaceToolRun
		for i := range turn.ToolRuns {
			if turn.ToolRuns[i].DSHCallID == req.CallID {
				run = &turn.ToolRuns[i]
				break
			}
		}
		if run == nil || !run.Pending || run.Call == nil || run.Call.Function.Name != req.Name {
			return nil, errors.New("dsh_workspace_request_invalid")
		}
		var original, requested any
		if json.Unmarshal([]byte(run.Call.Function.Arguments), &original) != nil || json.Unmarshal(req.Arguments, &requested) != nil {
			return nil, errors.New("dsh_workspace_request_invalid")
		}
		a, _ := json.Marshal(original)
		b, _ := json.Marshal(requested)
		if !bytes.Equal(a, b) {
			return nil, errors.New("dsh_workspace_request_invalid")
		}
		limit := maxToolText
		_, background := ctx.Value(goalRuntimeContext{}).(*goalRuntime)
		boundary, _ := ctx.Value(dshBoundaryStateKey{}).(*dshBoundaryState)
		if boundary != nil && boundary.capable && (background || session.GoalRun == "") {
			limit = maxDSHWorkspaceText
		}
		var args struct {
			Path          string  `json:"file_path"`
			Content       *string `json:"content"`
			Old           *string `json:"old_string"`
			New           *string `json:"new_string"`
			All           bool    `json:"replace_all"`
			Permissions   *string `json:"sandbox_permissions"`
			Justification *string `json:"justification"`
		}
		if decodeWorkspaceTool(req.Arguments, &args) != nil || req.Name == "write" && (!validDSHWorkspaceText(args.Content, limit) || args.Old != nil || args.New != nil) || req.Name == "edit" && (args.Content != nil || !validDSHWorkspaceText(args.Old, limit) || !validDSHWorkspaceText(args.New, limit) || *args.Old == "") {
			return nil, errors.New("dsh_workspace_request_invalid")
		}
		var permission, justification any
		if args.Permissions != nil {
			permission = *args.Permissions
		}
		if args.Justification != nil {
			justification = *args.Justification
		}
		if !dshSandboxEscalationValid(permission, justification) {
			return nil, errors.New("dsh_workspace_request_invalid")
		}
		relative := args.Path
		if filepath.IsAbs(relative) {
			var err error
			relative, err = filepath.Rel(session.CWD, relative)
			if err != nil {
				return nil, errWorkspacePath
			}
			relative = filepath.ToSlash(relative)
		}
		if _, err := workspaceRelative(relative); err != nil || relative == "." {
			return nil, errWorkspacePath
		}
		current, err := h.live.load(session.ID)
		if err != nil || current.Engine != "dsh" || current.DSHSessionID != session.DSHSessionID || current.CWD != session.CWD || current.Workspace != session.Workspace || current.ProjectID != session.ProjectID || len(current.Turns) == 0 || current.Turns[len(current.Turns)-1].Operation != turn.Operation {
			return nil, errors.New("dsh_workspace_session_changed")
		}
		root, err := h.live.workspaces.sessionRoot(session)
		if err != nil {
			return nil, err
		}
		before, err := dshWorkspaceTarget(root, relative)
		// Like DSH's write, a new file may sit in folders that do not exist yet.
		// They are created only after the permission check, under the write lock.
		createParents := false
		if err != nil && req.Name == "write" && dshMissingWriteParent(root, relative) {
			before, err, createParents = dshTargetState{}, nil, true
		}
		root.Close()
		if err != nil {
			return nil, err
		}
		if req.Name == "edit" && before.File == nil {
			return nil, errors.New("workspace_file_changed")
		}
		if before.File != nil && before.File.Size() > int64(limit) {
			return nil, errors.New("tool_file_unavailable")
		}
		input := map[string]any{"path": relative}
		if req.Name == "write" {
			input["content"] = *args.Content
		} else {
			input["old_string"] = *args.Old
			input["new_string"] = *args.New
			input["replace_all"] = args.All
		}
		title := "修改工作区文件"
		if args.Permissions != nil {
			input["sandbox_permissions"] = *args.Permissions
			input["justification"] = *args.Justification
			title = *args.Justification
		}
		// As DSH's file sandbox: read-only refuses a write unless the model asks
		// to escalate, workspace-write changes the workspace without asking,
		// and any escalation is the user's call (完全权限 approves it).
		mode := current.permissionMode()
		if mode == modeReadOnly && args.Permissions == nil {
			return nil, errWorkspaceReadOnly
		}
		var counts reportedPermissions
		if mode != modeWorkspaceWrite || args.Permissions != nil {
			approvalCtx := context.WithValue(ctx, dshNativeApprovalWaitKey{}, true)
			counts, err = h.requestDSHToolApproval(approvalCtx, ws, session.ID, run.Call.ID, title, req.Name, "edit", input, true)
		}
		turn.Permissions.Asked += counts.Asked
		turn.Permissions.Allowed += counts.Allowed
		turn.Permissions.Denied += counts.Denied
		if err != nil {
			if _, background := ctx.Value(goalRuntimeContext{}).(*goalRuntime); background && ctx.Err() == nil && (err.Error() == "tool_approval_unavailable" || err.Error() == "tool_permission_cancelled") {
				// No approval was obtained, so the rooted write has not started.
				run.Pending, run.Report.Failed, run.Output = false, 1, "Approval unavailable; no workspace change was made."
				return nil, errDSHGoalApprovalUnavailable
			}
			return nil, err
		}
		authorize, ok := ctx.Value(dshWorkspaceAuthorizeKey{}).(func(context.Context) error)
		checkAuthority := func() error {
			if !ok || authorize(ctx) != nil || ctx.Err() != nil {
				return errRelayAccess
			}
			if _, background := ctx.Value(goalRuntimeContext{}).(*goalRuntime); !background {
				if done, exists := ctx.Value(acpViewerDoneKey{}).(<-chan struct{}); exists {
					select {
					case <-done:
						return errors.New("tool_approval_unavailable")
					default:
					}
				}
			}
			return nil
		}
		if err = checkAuthority(); err != nil {
			return nil, err
		}
		current, err = h.live.load(session.ID)
		if err != nil || current.Engine != "dsh" || current.DSHSessionID != session.DSHSessionID || current.CWD != session.CWD || current.Workspace != session.Workspace || current.ProjectID != session.ProjectID || len(current.Turns) == 0 || current.Turns[len(current.Turns)-1].Operation != turn.Operation {
			return nil, errors.New("dsh_workspace_session_changed")
		}
		h.live.workspaces.writes.Lock()
		defer h.live.workspaces.writes.Unlock()
		root, err = h.live.workspaces.sessionRoot(session)
		if err != nil {
			return nil, err
		}
		defer root.Close()
		if createParents {
			if !dshMissingWriteParent(root, relative) {
				return nil, errors.New("workspace_file_changed")
			}
			if err := root.MkdirAll(filepath.Dir(filepath.FromSlash(relative)), 0o755); err != nil {
				return nil, errWorkspacePath
			}
		}
		after, err := dshWorkspaceTarget(root, relative)
		if err != nil || !createParents && !sameDSHTarget(before, after) || createParents && after.File != nil {
			return nil, errors.New("workspace_file_changed")
		}
		var previous *string
		if before.File != nil {
			f, err := openAttachmentFile(root, filepath.FromSlash(relative), int64(limit))
			if err != nil {
				return nil, err
			}
			data, readErr := io.ReadAll(io.LimitReader(f, int64(limit)+1))
			if err = errors.Join(readErr, f.Close()); err != nil {
				return nil, err
			}
			text := string(data)
			if !validDSHWorkspaceText(&text, limit) {
				return nil, errors.New("tool_file_unavailable")
			}
			previous = &text
		}
		var content string
		if req.Name == "write" {
			content = *args.Content
		} else {
			content, _, err = replaceWorkspaceTextLimit(*previous, []workspaceReplacement{{Old: *args.Old, New: args.New, All: args.All}}, limit)
			if err != nil {
				return nil, err
			}
		}
		value := dshWorkspaceReceiptValue{Path: filepath.Join(session.CWD, filepath.FromSlash(relative)), Before: previous, After: content}
		if req.Name == "write" {
			value.Operation = "create"
			if previous != nil {
				value.Operation = "update"
			}
		}
		if _, err = prepareDSHWorkspaceReceipt(value); err != nil {
			return nil, err
		}
		if err = checkAuthority(); err != nil {
			return nil, err
		}
		var witness *dshWorkspaceWitness
		if boundary != nil && boundary.capable && (background || session.GoalRun == "") {
			historyOperation := turn.Operation
			var goalID, sdkGoalID string
			if background {
				row, loadErr := h.live.load(session.ID)
				if loadErr != nil || session.Goal == nil || row.Goal == nil || !row.Goal.Current || row.Goal.Status != "active" || row.Goal.ID != session.Goal.ID || row.GoalRun != turn.Operation || row.Goal.DSHQueryOp != session.Goal.DSHQueryOp || !validUUID(row.Goal.DSHQueryOp) || !strings.HasPrefix(row.Goal.DSHGoalID, "goal-") || !validUUID(strings.TrimPrefix(row.Goal.DSHGoalID, "goal-")) {
					return nil, errors.New("dsh_workspace_session_changed")
				}
				historyOperation, goalID, sdkGoalID = row.Goal.DSHQueryOp, row.Goal.ID, row.Goal.DSHGoalID
			}
			verified, auditErr := dshAuditWorkspacePrefixForGoal(ctx, session.DSHSessionID, historyOperation, req.CallID, req.Name, req.Arguments, boundary.history, sdkGoalID)
			if auditErr != nil {
				return nil, errors.New("dsh_workspace_history_unverified")
			}
			verified.Operation = turn.Operation
			if background {
				verified.HistoryOperation, verified.GoalID, verified.SDKGoalID = historyOperation, goalID, sdkGoalID
			}
			witness = &verified
			if err = checkAuthority(); err != nil {
				return nil, err
			}
		}
		effect, err := h.beginDSHSideEffect(ctx, session.ID, turn.Operation, req.CallID, func(row acpSession) bool {
			if session.GoalRun != "" && (row.GoalRun != session.GoalRun || row.Goal == nil || session.Goal == nil || !row.Goal.Current || row.Goal.ID != session.Goal.ID || row.Goal.Status != "active" || row.Goal.DSHQueryOp != session.Goal.DSHQueryOp || (witness != nil && witness.GoalID != "" && (row.Goal.ID != witness.GoalID || row.Goal.DSHGoalID != witness.SDKGoalID))) {
				return false
			}
			if row.DSHSessionID != session.DSHSessionID || row.CWD != session.CWD || row.Workspace != session.Workspace || row.ProjectID != session.ProjectID {
				return false
			}
			for _, candidate := range row.Turns[len(row.Turns)-1].ToolRuns {
				if candidate.DSHCallID == req.CallID && candidate.Pending && candidate.Call != nil && candidate.Call.Function.Name == req.Name {
					expected, e1 := strictDSHJSON([]byte(candidate.Call.Function.Arguments))
					actual, e2 := strictDSHJSON(req.Arguments)
					return e1 == nil && e2 == nil && bytes.Equal(expected, actual)
				}
			}
			return false
		})
		if err != nil {
			return nil, err
		}
		if witness != nil {
			claim, claimErr := connectorPending(effect.root, effect.session, effect.operation, effect.call)
			if claimErr != nil || claim.State != "dispatch_claimed" {
				return nil, errors.New("dsh_connector_dispatch_uncertain")
			}
			claim.WorkspaceBoundary = witness
			if writeConnectorClaim(effect.root, claim) != nil {
				return nil, errors.New("dsh_connector_dispatch_uncertain")
			}
		}
		if err = writeWorkspaceTextIfUnchangedLimit(ctx, root, relative, content, before.File, limit); err != nil {
			return nil, errors.Join(errors.New("dsh_connector_outcome_unknown"), err)
		}
		run.FileOperation = "updated"
		if before.File == nil {
			run.FileOperation = "created"
		}
		old := ""
		if previous != nil {
			old = *previous
		}
		run.FileAdded, run.FileDeleted = workspaceLineChanges(old, content)
		if err = effect.completeWorkspace(value); err != nil {
			return nil, err
		}
		return map[string]any{"output": fmt.Sprintf("Updated workspace file %s", relative), "value": value}, nil
	}
}
