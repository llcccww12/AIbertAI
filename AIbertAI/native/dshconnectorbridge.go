package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/coder/websocket"
)

// dshConnectorRequest must match the already persisted SDK tool call exactly.
type dshConnectorRequest struct {
	SessionID string          `json:"sessionId"`
	CallID    string          `json:"callId"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// dshConnectorDispatch claims an invocation before executing a side effect.
// The per-session sentinel blocks subsequent connector side effects until a
// durable SDK result is reconciled. The call record survives reconciliation
// so a duplicate call cannot repeat the action.
func dshConnectorDispatch(root *os.Root, session, operation, call string) error {
	if root == nil || !validUUID(session) || !validUUID(operation) || call == "" || len(call) > 1024 {
		return errors.New("dsh_connector_dispatch_unavailable")
	}
	const base = "dsh-connector-dispatch"
	if err := root.MkdirAll(base, 0700); err != nil {
		return errors.New("dsh_connector_dispatch_unavailable")
	}
	// Persist each newly created directory entry before dispatch can occur.
	parent, err := root.Open(".")
	if err != nil {
		return errors.New("dsh_connector_dispatch_unavailable")
	}
	err = errors.Join(syncDirFile(parent), parent.Close())
	if err != nil {
		return errors.New("dsh_connector_dispatch_unavailable")
	}
	dir := base + "/" + session
	if err = root.MkdirAll(dir, 0700); err != nil {
		return errors.New("dsh_connector_dispatch_unavailable")
	}
	parent, err = root.Open(base)
	if err != nil {
		return errors.New("dsh_connector_dispatch_unavailable")
	}
	err = errors.Join(syncDirFile(parent), parent.Close())
	if err != nil {
		return errors.New("dsh_connector_dispatch_unavailable")
	}
	claims, err := dshConnectorClaims(root, session)
	if err != nil {
		return errors.New("dsh_connector_dispatch_uncertain")
	}
	for _, claim := range claims {
		// A claim from an operation that already ended stays on disk as evidence
		// of an unknown outcome, but it may only refuse that same call. Blocking
		// every later call would turn one failed tool into a dead session.
		if claim.OperationID != operation && claim.CallID != call {
			continue
		}
		if claim.State != "reconciled" && claim.State != "user_reviewed" || (claim.OperationID == operation || claim.State == "user_reviewed") && claim.CallID == call {
			return errors.New("dsh_connector_dispatch_uncertain")
		}
	}
	claim := dshConnectorClaim{SessionID: session, OperationID: operation, CallID: call, State: "dispatch_claimed"}
	record, err := json.Marshal(claim)
	if err != nil {
		return errors.New("dsh_connector_dispatch_unavailable")
	}
	// The per-session exclusive sentinel serializes claims across processes.
	pending := dir + "/.pending"
	// A sentinel left behind by an ended operation must not lock the session
	// either; its claim file keeps the unknown outcome on record.
	if stale, e := readConnectorClaim(root, pending, session); e == nil && stale.OperationID != operation {
		if e = root.Remove(pending); e != nil {
			return errors.New("dsh_connector_dispatch_uncertain")
		}
		if e = syncConnectorDir(root, dir); e != nil {
			return errors.New("dsh_connector_dispatch_uncertain")
		}
	}
	f, err := root.OpenFile(pending, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("dsh_connector_dispatch_uncertain")
	}
	if _, err = f.Write(record); err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return errors.New("dsh_connector_dispatch_uncertain")
	}
	if err = syncConnectorDir(root, dir); err != nil {
		return errors.New("dsh_connector_dispatch_uncertain")
	}
	name := connectorClaimPath(session, operation, call)
	f, err = root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("dsh_connector_dispatch_uncertain")
	}
	if _, err = f.Write(record); err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return errors.New("dsh_connector_dispatch_uncertain")
	}
	if err = syncConnectorDir(root, dir); err != nil {
		return errors.New("dsh_connector_dispatch_uncertain")
	}

	return nil
}

// dshConnectorBridge accepts only tools supplied by trusted, turn-scoped
// discovery. reauthorize must revalidate the exact selected tool after the
// user approves, including enabled preferences, credential and expert filters.
// Local servers must have been separately approved before startup; the caller
// owns their shutdown. No SDK-supplied identifier selects an executable or key.
func (h *relaySessionHost) dshConnectorBridge(ctx context.Context, ws *websocket.Conn, request *http.Request, session acpSession, turn *acpTurn, tools map[string]connectorTool, local map[string]*desktopMCP, reauthorize func(context.Context, connectorTool) error) func(json.RawMessage) (any, error) {
	return func(raw json.RawMessage) (any, error) {
		var req dshConnectorRequest
		if json.Unmarshal(raw, &req) != nil || req.SessionID != session.DSHSessionID || req.CallID == "" || len(req.CallID) > 1024 || len(req.Arguments) > 64<<10 {
			return nil, errors.New("dsh_connector_request_invalid")
		}
		tool, ok := tools[req.Name]
		if !ok || tool.alias() != req.Name || reauthorize == nil {
			return nil, errors.New("dsh_connector_request_invalid")
		}
		var input map[string]any
		decoder := json.NewDecoder(bytes.NewReader(req.Arguments))
		decoder.UseNumber()
		if decoder.Decode(&input) != nil || input == nil || decoder.Decode(new(any)) != io.EOF {
			return nil, errors.New("dsh_connector_request_invalid")
		}
		find := func(row acpSession) (int, *workspaceToolRun) {
			if row.Engine != "dsh" || row.State != "running" || row.DSHSessionID != session.DSHSessionID || row.CWD != session.CWD || row.Workspace != session.Workspace || row.ProjectID != session.ProjectID || len(row.Turns) == 0 {
				return 0, nil
			}
			if session.GoalRun != "" && (row.GoalRun != session.GoalRun || row.Goal == nil || session.Goal == nil || !row.Goal.Current || row.Goal.ID != session.Goal.ID || row.Goal.Status != "active") {
				return 0, nil
			}
			current := row.Turns[len(row.Turns)-1]
			if current.Operation != turn.Operation || row.ResourceRevision != session.ResourceRevision || !reflect.DeepEqual(row.ConnectorPreferences, session.ConnectorPreferences) {
				return 0, nil
			}
			for i := range current.ToolRuns {
				run := &current.ToolRuns[i]
				if run.DSHCallID == req.CallID && run.Pending && run.Call != nil && run.Call.Function.Name == req.Name {
					var expected any
					decoder := json.NewDecoder(strings.NewReader(run.Call.Function.Arguments))
					decoder.UseNumber()
					if decoder.Decode(&expected) != nil || decoder.Decode(new(any)) != io.EOF {
						return 0, nil
					}
					a, _ := json.Marshal(expected)
					b, _ := json.Marshal(input)
					if bytes.Equal(a, b) {
						return i, run
					}
					return 0, nil
				}
			}
			return 0, nil
		}
		if h.live == nil || h.live.store == nil || h.acpScope(request.WithContext(ctx), "resources:manage") != nil {
			return nil, errRelayAccess
		}
		row, err := h.live.load(session.ID)
		if err != nil {
			return nil, err
		}
		step, run := find(row)
		if run == nil {
			return nil, errors.New("dsh_connector_request_invalid")
		}
		approval := map[string]any{"connector": tool.ConnectorName, "tool": tool.Name, "arguments": input}
		var target dshConnectorTarget
		if !tool.Local {
			if h.live.client == nil {
				return nil, errors.New("connector_unavailable")
			}
			target, err = newDSHConnectorTarget(h.live.client.server, tool, session.ResourceRevision, step)
			if err != nil {
				return nil, err
			}
			approval["connectorId"], approval["endpointDigest"] = target.ConnectorID, target.EndpointDigest
		}
		permissions, err := h.requestDSHToolApproval(ctx, ws, session.ID, run.Call.ID, tool.ConnectorName+" / "+tool.Name, req.Name, "other", approval, true)
		turn.Permissions.Asked += permissions.Asked
		turn.Permissions.Allowed += permissions.Allowed
		turn.Permissions.Denied += permissions.Denied
		if err != nil {
			return nil, err
		}
		if h.acpScope(request.WithContext(ctx), "resources:manage") != nil {
			return nil, errRelayAccess
		}
		row, err = h.live.load(session.ID)
		if err != nil {
			return nil, err
		}
		approvedStep := step
		step, run = find(row)
		if run == nil || step != approvedStep {
			return nil, errors.New("dsh_connector_session_changed")
		}
		if err = reauthorize(ctx, tool); err != nil {
			return nil, err
		}
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		var server *desktopMCP
		if tool.Local {
			server = local[tool.ConnectorID]
			if server == nil {
				return nil, errors.New("local_mcp_unavailable")
			}
		} else if h.live.client == nil {
			return nil, errors.New("connector_unavailable")
		}
		effect, err := h.beginDSHSideEffect(ctx, session.ID, turn.Operation, req.CallID, func(current acpSession) bool {
			step, run = find(current)
			return run != nil && step == approvedStep
		})
		if err != nil {
			return nil, err
		}
		if !tool.Local {
			currentTarget, targetErr := newDSHConnectorTarget(h.live.client.server, tool, session.ResourceRevision, step)
			if targetErr != nil || currentTarget != target {
				return nil, errors.New("dsh_connector_outcome_unknown")
			}
			if targetErr = effect.bindConnectorTarget(target); targetErr != nil {
				return nil, targetErr
			}
		}
		var output string
		if tool.Local {
			output, err = server.invoke(tool, req.Arguments)
		} else {
			var auth authorization
			auth, err = h.live.authorization(ctx)
			if err == nil {
				output, err = h.live.client.callConnector(ctx, auth, session.ID, turn.Operation, step, tool, req.Arguments)
			}
		}
		if err != nil {
			return nil, errors.Join(errors.New("dsh_connector_outcome_unknown"), err)
		}
		if err = effect.complete(output); err != nil {
			return nil, err
		}
		return map[string]any{"output": output}, nil
	}
}
