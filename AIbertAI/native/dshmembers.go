package main

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type dshMemberRequest struct {
	SessionID string `json:"sessionId"`
	MemberID  string `json:"memberId,omitempty"`
	From      *int64 `json:"from,omitempty"`
}

func (h *relaySessionHost) sessionMembers(ctx context.Context, raw json.RawMessage) (any, error) {
	var req dshMemberRequest
	if json.Unmarshal(raw, &req) != nil || len(req.MemberID) > 256 || req.From != nil && (req.MemberID == "" || *req.From < 0) {
		return nil, errors.New("invalid_member_request")
	}
	row, err := h.live.load(req.SessionID)
	if err != nil {
		return nil, err
	}
	if row.Engine != "dsh" || row.ID != row.DSHSessionID {
		return nil, errors.New("dsh_members_unavailable")
	}
	cfg := h.live.dsh
	var data json.RawMessage
	slot, p := cfg.LeasePool.observer(row.ID)
	if p != nil {
		readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		data, err = slot.observerCall(readCtx, p, "aibertai/members", req)
	} else if row.State == "running" {
		// The owner publishes its observer immediately after initialization.
		// Cold recovery here would race the first prompt for the same lease.
		if req.MemberID == "" {
			return map[string]any{"members": []any{}, "live": true}, nil
		}
		return nil, errors.New("dsh_members_starting")
	} else if len(row.Turns) > 0 {
		// A closed desktop process has no warm Agent. Re-open the persisted root
		// only when the member view is requested, without issuing a model turn.
		readCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		opts := dshSDKOptions{InspectSession: func(ctx context.Context, p *dshProcess) error {
			var readErr error
			data, readErr = p.call(ctx, "aibertai/members", req, nil)
			return readErr
		}}
		err = reconcileDSHSession(context.WithValue(readCtx, dshSDKOptionsKey{}, opts), cfg, row)
	} else if req.MemberID == "" {
		return map[string]any{"members": []any{}, "live": row.State == "running"}, nil
	} else {
		return nil, errors.New("dsh_member_unavailable")
	}
	if err != nil {
		return nil, err
	}
	var result map[string]json.RawMessage
	if json.Unmarshal(data, &result) != nil || result == nil || result["members"] == nil {
		return nil, errors.New("invalid_dsh_members_response")
	}
	result["live"], _ = json.Marshal(p != nil)
	return result, nil
}
