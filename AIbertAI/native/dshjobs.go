package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type dshJobsRequest struct {
	SessionID string `json:"sessionId"`
	Action    string `json:"action"`
	JobID     string `json:"jobId,omitempty"`
	From      *int64 `json:"from,omitempty"`
}

func (h *relaySessionHost) sessionJobs(ctx context.Context, raw json.RawMessage) (any, error) {
	var req dshJobsRequest
	if json.Unmarshal(raw, &req) != nil || (req.Action != "list" && req.Action != "output" && req.Action != "kill") ||
		(req.Action == "list") != (req.JobID == "") || len(req.JobID) > 256 || req.From != nil && (req.Action != "output" || *req.From < 0) {
		return nil, errors.New("invalid_jobs_request")
	}
	row, err := h.live.load(req.SessionID)
	if err != nil {
		return nil, err
	}
	if row.Engine != "dsh" || row.DSHSessionID != row.ID {
		return nil, errors.New("dsh_jobs_unavailable")
	}
	if slot, p := h.live.dsh.LeasePool.observer(row.ID); p != nil {
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		return slot.observerCall(ctx, p, "aibertai/jobs", req)
	}
	if req.Action == "list" {
		return map[string]any{"jobs": []any{}, "available": false}, nil
	}
	return nil, errors.New("dsh_job_unavailable")
}

func (pool *dshLeasePool) observer(id string) (*dshLeaseSlot, *dshProcess) {
	if pool == nil {
		return nil, nil
	}
	pool.mu.Lock()
	defer pool.mu.Unlock()
	if slot := pool.slots[id]; !pool.closed && slot != nil {
		return slot, slot.observerOwner.Load()
	}
	return nil, nil
}

func (p *dshProcess) deferFrame(f dshFrame) error {
	size := len(f.ID) + len(f.Params) + len(f.Result) + len(f.Method)
	if len(p.deferred) >= 4096 || p.deferredBytes+size > 8<<20 {
		return errors.New("dsh_history_buffer_exceeded")
	}
	p.deferred = append(p.deferred, f)
	p.deferredBytes += size
	return nil
}

// Stdio reads run on one goroutine. Human requests have their own reply
// channels, so they can inspect and stop jobs while session/wait is pending.
func (p *dshProcess) deliverObserverReply(f dshFrame) bool {
	if f.Method != "" || len(f.ID) == 0 || f.ID[0] != '"' {
		return false
	}
	var id string
	if json.Unmarshal(f.ID, &id) != nil || !strings.HasPrefix(id, "employee-observer-") {
		return false
	}
	if c, ok := p.observerReplies.Load(id); ok {
		select {
		case c.(chan dshFrame) <- f:
		default:
		}
	}
	return true
}

func (slot *dshLeaseSlot) observerCall(ctx context.Context, p *dshProcess, method string, req any) (json.RawMessage, error) {
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	id = "employee-observer-" + id
	reply := make(chan dshFrame, 1)
	p.observerReplies.Store(id, reply)
	defer p.observerReplies.Delete(id)
	if err := p.write(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": req}); err != nil {
		return nil, err
	}
	// An idle SDK may emit a final notification before this response. Reserve
	// the ordinary reader only while idle and leave such frames for its next call.
	gate := slot.gate
	var idleFrames <-chan dshFrame
	for {
		select {
		case frame := <-reply:
			if frame.Error != nil {
				return nil, fmt.Errorf("%s: %s", method, frame.Error.Message)
			}
			return frame.Result, nil
		case gate <- struct{}{}:
			defer func() { <-slot.gate }()
			gate = nil
			idleFrames = p.frames
		case f, ok := <-idleFrames:
			if !ok {
				return nil, errors.New("dsh_runtime_exited")
			}
			if err := p.deferFrame(f); err != nil {
				return nil, err
			}
			if f.readErr != nil {
				return nil, f.readErr
			}
		case <-p.stop:
			return nil, errors.New("dsh_runtime_exited")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
