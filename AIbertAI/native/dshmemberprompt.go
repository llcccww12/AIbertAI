package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

type dshMemberPromptRequest struct {
	SessionID      string `json:"sessionId"`
	MemberID       string `json:"memberId"`
	RequestID      string `json:"requestId"`
	Text           string `json:"text"`
	ClientTimeZone string `json:"clientTimeZone,omitempty"`
}

func (h *relaySessionHost) memberPrompt(ctx context.Context, ws *websocket.Conn, r *http.Request, raw json.RawMessage, accept func(any) error) (any, error) {
	var req dshMemberPromptRequest
	if h.local == nil || h.live == nil || h.live.dsh == nil || h.acpScope(r, "session:control") != nil || h.acpScope(r, "files:read") != nil {
		return nil, errRelayAccess
	}
	if json.Unmarshal(raw, &req) != nil || !validUUID(req.SessionID) || !validUUID(req.RequestID) || req.MemberID == "" || req.MemberID == req.SessionID || len(req.MemberID) > 256 || len(req.Text) > 16<<10 || strings.TrimSpace(req.Text) == "" || len(req.ClientTimeZone) > 128 {
		return nil, errors.New("invalid_member_prompt")
	}
	row, err := h.live.load(req.SessionID)
	if err != nil {
		return nil, err
	}
	if row.Engine != "dsh" || row.DSHSessionID != row.ID {
		return nil, errors.New("dsh_member_unavailable")
	}
	reply := map[string]any{"accepted": true, "requestId": req.RequestID}
	// A reconnect may retry a queued or already admitted cold operation while its lease is active.
	for _, t := range row.Turns {
		if t.Operation != req.RequestID && t.ClientID != req.RequestID {
			continue
		}
		if t.DSHMemberID != req.MemberID || t.Prompt != req.Text || t.ClientID != req.RequestID || t.Operation != req.RequestID {
			return nil, errors.New("member_request_conflict")
		}
		if t.Failed {
			return nil, errors.New(t.FailureCode)
		}
		return reply, nil
	}
	for _, m := range row.Queue.Messages {
		if m.ID != req.RequestID && m.OperationID != req.RequestID {
			continue
		}
		if m.DSHMemberID != req.MemberID || m.Text != req.Text || m.ID != req.RequestID || m.OperationID != req.RequestID {
			return nil, errors.New("member_request_conflict")
		}
		return reply, nil
	}
	if row.State == "running" && len(row.Turns) > 0 {
		active := row.Turns[len(row.Turns)-1]
		slot, p := h.live.dsh.LeasePool.observer(row.ID)
		if p != nil && active.Ended.IsZero() {
			// The SDK will only admit steering under this operation's unchanged billing lease.
			read, cancel := context.WithTimeout(ctx, 15*time.Second)
			args := map[string]any{"sessionId": row.ID, "operationId": active.Operation, "memberId": req.MemberID, "requestId": req.RequestID, "text": req.Text}
			if req.ClientTimeZone != "" {
				args["clientTimeZone"] = req.ClientTimeZone
			}
			result, e := slot.observerCall(read, p, "aibertai/member-prompt", args)
			cancel()
			if e == nil {
				var admitted struct{ MessageID string }
				if json.Unmarshal(result, &admitted) != nil || admitted.MessageID == "" {
					return nil, errors.New("member_admission_unverified")
				}
				return reply, nil
			}
			// If the lease closed before admission, the persistent queue can start a separate operation.
			if !strings.Contains(e.Error(), "dsh_member_lease_inactive") {
				return nil, e
			}
		}
	}
	query, _ := json.Marshal(dshMemberRequest{SessionID: row.ID})
	catalog, err := h.sessionMembers(ctx, query)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(catalog)
	var listing struct{ Members []struct{ ID, Mode string } }
	if err != nil || json.Unmarshal(encoded, &listing) != nil {
		return nil, errors.New("invalid_dsh_members_response")
	}
	for _, m := range listing.Members {
		if m.ID != req.MemberID || m.Mode != "continuable" {
			continue
		}
		args, _ := json.Marshal(map[string]any{"sessionId": row.ID, "action": "add", "id": req.RequestID, "text": req.Text, "sendIfIdle": true, "_meta": map[string]string{"operationId": req.RequestID}})
		_, e := h.handleQueue(context.WithValue(ctx, dshMemberQueueKey{}, req.MemberID), ws, r, args, func(any) error { return accept(reply) })
		return reply, e
	}
	return nil, errors.New("dsh_member_unavailable")
}

func (h *relaySessionHost) memberInterrupt(ctx context.Context, raw json.RawMessage) (any, error) {
	var req struct{ SessionID, MemberID string }
	if json.Unmarshal(raw, &req) != nil || !validUUID(req.SessionID) || req.MemberID == "" || req.MemberID == req.SessionID || len(req.MemberID) > 256 {
		return nil, errors.New("invalid_member_interrupt")
	}
	row, err := h.live.load(req.SessionID)
	if err != nil {
		return nil, err
	}
	if row.Engine != "dsh" || row.DSHSessionID != row.ID {
		return nil, errors.New("dsh_member_unavailable")
	}
	slot, p := h.live.dsh.LeasePool.observer(row.ID)
	if row.State != "running" || p == nil || len(row.Turns) == 0 {
		return map[string]bool{"accepted": false}, nil
	}
	op := row.Turns[len(row.Turns)-1]
	if !op.Ended.IsZero() || !validUUID(op.Operation) {
		return map[string]bool{"accepted": false}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return slot.observerCall(ctx, p, "aibertai/member-interrupt", map[string]string{"sessionId": row.ID, "memberId": req.MemberID, "operationId": op.Operation})
}
