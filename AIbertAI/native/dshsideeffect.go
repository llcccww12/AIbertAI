package main

import (
	"context"
	"errors"
	"os"
)

// dshSideEffect is a persisted exclusive claim, not proof of a completed call.
// Every side-effecting bridge must claim before execution and persist its exact
// result before replying. An abandoned claim blocks the entire session.
type dshSideEffect struct {
	root                     *os.Root
	session, operation, call string
}

// beginDSHSideEffect binds a claim to the live running operation while holding
// its session lock. validate runs while that lock is held and must not reacquire it.
// dshConnectorDispatch also uses a durable O_EXCL sentinel,
// so separate processes cannot acquire a competing claim for the session.
func (h *relaySessionHost) beginDSHSideEffect(ctx context.Context, session, operation, call string, validate func(acpSession) bool) (*dshSideEffect, error) {
	if h.live == nil || h.live.store == nil || !validUUID(session) || !validUUID(operation) || call == "" || validate == nil {
		return nil, errors.New("dsh_side_effect_unavailable")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	h.live.mu.Lock()
	defer h.live.mu.Unlock()
	row, ok := h.live.sessions[session]
	if !ok || row.ID != session || row.Engine != "dsh" || row.State != "running" || len(row.Turns) == 0 || row.Turns[len(row.Turns)-1].Operation != operation || !validate(row) {
		return nil, errors.New("dsh_side_effect_session_changed")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := dshConnectorDispatch(h.live.store.Root, session, operation, call); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(errors.New("dsh_connector_outcome_unknown"), err)
	}
	return &dshSideEffect{root: h.live.store.Root, session: session, operation: operation, call: call}, nil
}

// complete persists the exact result before it may be returned to the SDK.
// The claim remains pending until a verified durable SDK tool/result is bound.
func (effect *dshSideEffect) complete(output string) error {
	if effect == nil {
		return errors.New("dsh_connector_outcome_unknown")
	}
	if err := dshConnectorRecordResult(effect.root, effect.session, effect.operation, effect.call, output); err != nil {
		return errors.Join(errors.New("dsh_connector_outcome_unknown"), err)
	}
	return nil
}
