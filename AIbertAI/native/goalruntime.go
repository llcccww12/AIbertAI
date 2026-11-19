package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type goalRuntimeContext struct{}

type goalRuntime struct {
	wake    chan struct{}
	mu      sync.Mutex
	viewers map[*websocket.Conn]struct{}
	waiting map[string]goalApproval
}

type goalApproval struct {
	viewer *websocket.Conn
	answer chan string
}

func newGoalRuntime() *goalRuntime {
	return &goalRuntime{
		wake: make(chan struct{}, 1), viewers: make(map[*websocket.Conn]struct{}),
		waiting: make(map[string]goalApproval),
	}
}

func (g *goalRuntime) schedule() {
	select {
	case g.wake <- struct{}{}:
	default:
	}
}

func (g *goalRuntime) addViewer(ws *websocket.Conn) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.viewers[ws] = struct{}{}
}

func (g *goalRuntime) removeViewer(ws *websocket.Conn) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.viewers, ws)
	for id, waiting := range g.waiting {
		if waiting.viewer == ws {
			waiting.answer <- ""
			delete(g.waiting, id)
		}
	}
}

func (g *goalRuntime) respond(ws *websocket.Conn, id, result, failure json.RawMessage) {
	var key string
	if json.Unmarshal(id, &key) != nil {
		return
	}
	var reply struct {
		Outcome struct{ Outcome, OptionID string }
	}
	choice := ""
	if len(failure) == 0 && json.Unmarshal(result, &reply) == nil && reply.Outcome.Outcome == "selected" {
		choice = reply.Outcome.OptionID
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if pending, ok := g.waiting[key]; ok && pending.viewer == ws {
		pending.answer <- choice
		delete(g.waiting, key)
	}
}

func (g *goalRuntime) notify(ctx context.Context, method string, payload any) {
	g.mu.Lock()
	viewers := make([]*websocket.Conn, 0, len(g.viewers))
	for ws := range g.viewers {
		viewers = append(viewers, ws)
	}
	g.mu.Unlock()
	for _, ws := range viewers {
		write, cancel := context.WithTimeout(ctx, time.Second)
		err := wsjson.Write(write, ws, map[string]any{"jsonrpc": "2.0", "method": method, "params": payload})
		cancel()
		if err != nil {
			g.removeViewer(ws)
		}
	}
}

func (g *goalRuntime) permission(ctx context.Context, id string, request any) (string, error) {
	g.mu.Lock()
	var viewer *websocket.Conn
	for ws := range g.viewers {
		viewer = ws
		break
	}
	if viewer == nil {
		g.mu.Unlock()
		return "", errors.New("tool_approval_unavailable")
	}
	pending := goalApproval{viewer: viewer, answer: make(chan string, 1)}
	g.waiting[id] = pending
	g.mu.Unlock()
	defer func() {
		g.mu.Lock()
		delete(g.waiting, id)
		g.mu.Unlock()
	}()
	if err := wsjson.Write(ctx, viewer, request); err != nil {
		return "", errors.New("tool_approval_unavailable")
	}
	select {
	case choice := <-pending.answer:
		return choice, nil
	case <-ctx.Done():
		return "", errors.New("tool_approval_unavailable")
	}
}

func writeTurnNotification(ctx context.Context, ws *websocket.Conn, method string, payload any) error {
	if goals, ok := ctx.Value(goalRuntimeContext{}).(*goalRuntime); ok {
		goals.notify(ctx, method, payload)
		return nil
	}
	return wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "method": method, "params": payload})
}

func (h *relaySessionHost) superviseGoals(ctx context.Context) {
	g := h.goals
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	g.schedule()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-g.wake:
		}
		if ctx.Err() != nil {
			return
		}
		h.live.mu.Lock()
		var selected acpSession
		for _, row := range h.live.sessions {
			if row.GoalRun == "" || row.Goal == nil || !row.Goal.Current || row.Goal.Status != "active" ||
				h.live.restarting || h.live.activeTurns[row.ID] != nil || h.live.compacting[row.ID] {
				continue
			}
			if selected.ID == "" || row.Updated.Before(selected.Updated) {
				selected = row
			}
		}
		h.live.mu.Unlock()
		if selected.ID == "" {
			continue
		}
		request := &http.Request{Host: h.local.address, Header: http.Header{}}
		request.Header.Set("Sec-WebSocket-Protocol", "monkeyai.v1, "+h.local.protocol)
		run := context.WithValue(ctx, goalRuntimeContext{}, g)
		if _, err := h.promptACP(run, nil, request, selected.ID, selected.GoalRun, selected.GoalRun, selected.Goal.Objective, nil); err != nil && ctx.Err() == nil {
			slog.Debug("持续目标执行结束", "session_id", selected.ID, "reason", safeDesktopTurnFailure(err))
		}
	}
}

func (l *acpLive) checkpointGoal(id, operation string, checkpoint goalCheckpoint) (acpSession, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	row, ok := l.sessions[id]
	if index := len(row.Turns) - 1; ok && index >= 0 && row.Goal != nil && row.GoalRun == operation && row.Turns[index].Operation == operation {
		row.Turns = append([]acpTurn(nil), row.Turns...)
		row.Turns[index].GoalCheckpoint = &checkpoint
		goal := *row.Goal
		goal.Version++
		row.Goal = &goal
		return row, l.save(row)
	}
	return row, errors.New("stale_goal")
}

func goalView(row acpSession) *desktopGoal {
	if row.Goal == nil {
		return nil
	}
	goal := *row.Goal
	if row.GoalRun != "" && row.State == "running" {
		goal.ExecutionID = row.GoalRun
	}
	if n := len(row.Turns); n != 0 && row.Turns[n-1].Operation == row.GoalRun {
		if saved := row.Turns[n-1].GoalCheckpoint; saved != nil {
			tokens, known := goalModelTokens(saved.Usage)
			goal.Usage.Tokens = min(goal.Usage.Tokens+tokens, maxGoalBudget)
			goal.ActiveSeconds += int64(saved.ActiveTime / time.Second)
			goal.Uncertain = goal.Uncertain || saved.Rounds != 0 && !known
		}
	}
	return &goal
}

func materializeGoalCheckpoint(turn *acpTurn) {
	if saved := turn.GoalCheckpoint; saved != nil {
		turn.Answer, turn.ToolRuns, turn.Usage = saved.Progress, saved.Runs, saved.Usage
		turn.Permissions = saved.Permissions
		turn.GoalCheckpoint = nil
	}
}
