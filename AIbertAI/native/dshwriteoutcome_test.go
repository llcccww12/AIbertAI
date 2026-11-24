package main

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// A failed tool must not freeze the conversation: DSH reports the outcome as an
// isError result and the next message is accepted. Only session-level unknowns
// still require reconciliation.
func TestDSHToolOutcomeNeverBlocksTheNextPrompt(t *testing.T) {
	const id = "d1bd985c-9f71-4b85-937a-c41b1b104d38"
	const op = "d2bd985c-9f71-4b85-937a-c41b1b104d38"
	now := time.Now().UTC()
	for _, code := range []string{"dsh_write_outcome_unknown", "dsh_canvas_outcome_unknown", "dsh_connector_outcome_unknown"} {
		t.Run(code, func(t *testing.T) {
			last := acpTurn{Failed: true, FailureCode: code, Ended: now, Acked: true}
			live := &acpLive{dir: t.TempDir(), sessions: map[string]acpSession{id: {ID: id, Engine: "dsh", State: "error", Turns: []acpTurn{last}}}}
			_, index, replay, err := live.beginTurn(id, op, "", "next", nil, nil)
			if err != nil || replay || index != 1 {
				t.Fatalf("tool outcome blocked the session: index=%d replay=%v err=%v", index, replay, err)
			}
		})
	}
}

func TestDSHWorkspaceToolReasonIsModelFacing(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want string
	}{
		{errors.New("dsh_connector_outcome_unknown"), "write_outcome_unknown"},
		{errors.New("dsh_connector_dispatch_uncertain"), "write_not_dispatched"},
		{errors.New("dsh_workspace_changed"), "write_conflict"},
		{errors.New("something_else"), "workspace_mutation_denied"},
	} {
		if got := dshWorkspaceToolReason(tc.err); !strings.HasPrefix(got, tc.want+":") {
			t.Errorf("reason for %q = %q, want prefix %q", tc.err, got, tc.want)
		}
	}
}
