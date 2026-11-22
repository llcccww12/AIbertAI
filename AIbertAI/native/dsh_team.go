package main

import (
	"bytes"
	"encoding/json"
	"errors"
)

// Tasks are the native DSH public projection, already checked by its viewSchema.
type dshTeamBoard struct {
	SessionID string            `json:"sessionId"`
	Enabled   bool              `json:"enabled"`
	Tasks     []json.RawMessage `json:"tasks"`
	Failure   string            `json:"failure,omitempty"`
}

func (board *dshTeamBoard) visible() bool {
	return board != nil && board.Enabled && (len(board.Tasks) != 0 || board.Failure != "")
}

// One SDK process owns one root. Only the SDK session/created notification
// connects a descendant to that root; tool output and names cannot establish ownership.
type dshTeamSessions struct {
	root      string
	operation string
	members   map[string]bool
	calls     map[string]string
}

func (t *dshTeamSessions) begin(root, operation string) {
	if root != t.root {
		t.root = root
		t.members = make(map[string]bool)
	}
	t.operation = operation
	t.calls = make(map[string]string)
}

func (t *dshTeamSessions) track(method string, raw json.RawMessage) {
	switch method {
	case "subagent.started":
		var n struct{ ParentSessionID, ChildSessionID string }
		if json.Unmarshal(raw, &n) == nil && n.ChildSessionID != "" && n.ChildSessionID != t.root && (n.ParentSessionID == t.root || t.members[n.ParentSessionID]) {
			t.members[n.ChildSessionID] = true
		}
	case "session.event":
		if t.operation == "" && len(t.members) == 0 || !bytes.Contains(raw, []byte(`"aibertai/invocation"`)) {
			return
		}
		var n struct {
			SessionID                             string
			AibertaiTeamRoot, AibertaiOperationID string
			Event                                 struct {
				Type string
				Data struct{ InvocationID string }
			}
		}
		if json.Unmarshal(raw, &n) != nil || n.Event.Type != "aibertai/invocation" || !validUUID(n.Event.Data.InvocationID) || n.SessionID == "" || n.SessionID == t.root {
			return
		}
		// Native runtime ownership also covers a cold descendant whose direct creator is idle.
		if (t.operation != "" && n.AibertaiTeamRoot == t.root && n.AibertaiOperationID == t.operation) ||
			(t.operation == "" && t.members[n.SessionID]) {
			t.calls[n.Event.Data.InvocationID] = n.SessionID
		}
	}
}

// The SDK reopens its child journal and compares the flushed settlement with
// the live event before emitting this receipt. Match it to the host's actual
// delivered request before charging the parent goal or releasing the journal.
func (g *dshLocalGateway) verifyTeamSettlement(r dshFailedCompactionReceipt, operation string) error {
	if !g.requireInvocationID || r.State != "committed" || !validUUID(r.InvocationID) || r.UsageSeq == nil || *r.UsageSeq < 0 || r.Seq <= 0 || (*r.UsageSeq != r.Seq && *r.UsageSeq != r.Seq-2) || r.Usage == nil || r.Usage.Input == nil || r.Usage.Output == nil || *r.Usage.Input < 0 || *r.Usage.Output < 0 {
		return errors.New("dsh_invalid_team_settlement")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.pending == nil || g.pending.SessionID != g.session || g.pending.OperationID != operation || g.pending.CallID != r.InvocationID || g.pending.State != "delivered" {
		return errors.New("dsh_invalid_team_settlement")
	}
	return nil
}
