package main

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Desktop prompts belong to a running SDK operation. A renderer may disconnect
// and be replaced while the original operation is awaiting the user's answer.
type desktopDSHInteractions struct {
	mu       sync.Mutex
	viewers  []*websocket.Conn
	changed  chan struct{}
	requests map[string]*desktopDSHRequest
}

type desktopDSHRequest struct {
	viewer *websocket.Conn
	reply  chan acpPermissionReply
}

func newDesktopDSHInteractions() *desktopDSHInteractions {
	return &desktopDSHInteractions{changed: make(chan struct{}), requests: make(map[string]*desktopDSHRequest)}
}

func (d *desktopDSHInteractions) viewer(ws *websocket.Conn, connected bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	i := slices.Index(d.viewers, ws)
	if connected && i < 0 {
		d.viewers = append(d.viewers, ws)
	} else if !connected && i >= 0 {
		d.viewers = slices.Delete(d.viewers, i, i+1)
	} else {
		return
	}
	close(d.changed)
	d.changed = make(chan struct{})
}

func (d *desktopDSHInteractions) answer(ws *websocket.Conn, id string, reply acpPermissionReply) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	request := d.requests[id]
	if request == nil || request.viewer != ws || !slices.Contains(d.viewers, ws) {
		return false
	}
	select {
	case request.reply <- reply:
	default:
	}
	return true
}

func (d *desktopDSHInteractions) ask(ctx context.Context, preferred *websocket.Conn, method string, params any) (acpPermissionReply, error) {
	id, err := newUUID()
	if err != nil {
		return acpPermissionReply{}, err
	}
	request := &desktopDSHRequest{reply: make(chan acpPermissionReply, 1)}
	d.mu.Lock()
	d.requests[id] = request
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		delete(d.requests, id)
		d.mu.Unlock()
	}()

	var sent *websocket.Conn
	for ctx.Err() == nil {
		d.mu.Lock()
		var viewer *websocket.Conn
		if slices.Contains(d.viewers, preferred) {
			viewer = preferred
		} else if len(d.viewers) > 0 {
			viewer = d.viewers[0]
		}
		request.viewer = viewer
		changed := d.changed
		d.mu.Unlock()
		if viewer != nil && viewer != sent {
			write, cancel := context.WithTimeout(ctx, 3*time.Second)
			err := wsjson.Write(write, viewer, map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
			cancel()
			if err != nil {
				d.viewer(viewer, false)
				continue
			}
		}
		sent = viewer
		select {
		case reply := <-request.reply:
			return reply, nil
		case <-changed:
		case <-ctx.Done():
		}
	}
	return acpPermissionReply{}, ctx.Err()
}

func (h *relaySessionHost) askDesktopDSHQuestion(ctx context.Context, ws *websocket.Conn, q dshQuestionRequest) (any, error) {
	defer func() {
		clear, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		h.goals.notify(clear, "_monkeyai/session/request_cancelled", map[string]any{"sessionId": q.SessionID, "requestId": q.RequestID})
	}()
	reply, err := h.interactions.ask(ctx, ws, "_monkeyai/session/question", map[string]any{"sessionId": q.SessionID, "requestId": q.RequestID, "questions": q.Questions})
	if err != nil {
		return nil, err
	}
	if len(reply.Error) != 0 || !json.Valid(reply.Result) {
		return nil, errors.New("dsh_question_unavailable")
	}
	return dshQuestionAnswer(q, reply.Result)
}
