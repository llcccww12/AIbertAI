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
	viewers map[*websocket.Conn]*goalViewer
	waiting map[string]goalApproval
}

type goalApproval struct {
	viewer  *websocket.Conn
	session string
	answer  chan string
}

// A nil check identifies a trusted local viewer. Remote viewers must recheck
// both the session binding and the requested scope on each delivery.
type goalViewer struct {
	check func(context.Context, string, string) bool
}

func newGoalRuntime() *goalRuntime {
	return &goalRuntime{
		wake: make(chan struct{}, 1), viewers: make(map[*websocket.Conn]*goalViewer),
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
	g.viewers[ws] = &goalViewer{}
}

func (g *goalRuntime) addScopedViewer(ws *websocket.Conn, check func(context.Context, string, string) bool) {
	if check == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.viewers[ws] = &goalViewer{check: check}
}

func goalViewerAllowed(ctx context.Context, viewer *goalViewer, session, scope string) bool {
	return viewer.check == nil || session != "" && viewer.check(ctx, session, scope)
}

func goalPayloadSession(payload any) string {
	values, ok := payload.(map[string]any)
	if !ok {
		return ""
	}
	session, _ := values["sessionId"].(string)
	return session
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
	pending, ok := g.waiting[key]
	viewer, registered := g.viewers[ws]
	g.mu.Unlock()
	if !ok || !registered || pending.viewer != ws {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	allowed := goalViewerAllowed(ctx, viewer, pending.session, "session:control")
	cancel()
	g.mu.Lock()
	defer g.mu.Unlock()
	if current, exists := g.waiting[key]; exists && current.answer == pending.answer {
		if g.viewers[ws] != viewer {
			allowed = false
		}
		if !allowed {
			choice = ""
		}
		current.answer <- choice
		delete(g.waiting, key)
	}
}

func (g *goalRuntime) notify(ctx context.Context, method string, payload any) {
	g.mu.Lock()
	viewers := make(map[*websocket.Conn]*goalViewer, len(g.viewers))
	for ws, viewer := range g.viewers {
		viewers[ws] = viewer
	}
	g.mu.Unlock()
	for ws, viewer := range viewers {
		write, cancel := context.WithTimeout(ctx, time.Second)
		if !goalViewerAllowed(write, viewer, goalPayloadSession(payload), "session:read") {
			cancel()
			continue
		}
		g.mu.Lock()
		registered := g.viewers[ws] == viewer
		g.mu.Unlock()
		if !registered {
			cancel()
			continue
		}
		err := wsjson.Write(write, ws, map[string]any{"jsonrpc": "2.0", "method": method, "params": payload})
		cancel()
		if err != nil {
			g.removeViewer(ws)
		}
	}
}

func (g *goalRuntime) permission(ctx context.Context, id string, request any) (string, error) {
	values, _ := request.(map[string]any)
	session := goalPayloadSession(values["params"])
	g.mu.Lock()
	viewers := make(map[*websocket.Conn]*goalViewer, len(g.viewers))
	for ws, viewer := range g.viewers {
		viewers[ws] = viewer
	}
	g.mu.Unlock()
	var selected *websocket.Conn
	var selectedViewer *goalViewer
	for ws, viewer := range viewers {
		if goalViewerAllowed(ctx, viewer, session, "session:control") {
			selected = ws
			selectedViewer = viewer
			break
		}
	}
	g.mu.Lock()
	viewer := selected
	if selectedViewer == nil || g.viewers[viewer] != selectedViewer {
		viewer = nil
	}
	if viewer == nil {
		g.mu.Unlock()
		return "", errors.New("tool_approval_unavailable")
	}
	pending := goalApproval{viewer: viewer, session: session, answer: make(chan string, 1)}
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
	// One session may wait for user input while other goals keep advancing.
	var workers sync.WaitGroup
	defer workers.Wait()
	started := make(map[string]bool)
	finished := make(chan string)
	g.schedule()
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-finished:
			delete(started, id)
			continue
		case <-ticker.C:
		case <-g.wake:
		}
		if ctx.Err() != nil {
			return
		}
		h.live.mu.Lock()
		var selected acpSession
		for _, row := range h.live.sessions {
			if started[row.ID] || row.GoalRun == "" || row.Goal == nil || !row.Goal.Current || (row.Goal.Status != "active" && !(row.Engine == "dsh" && (row.Goal.Status == "clearing" || row.Goal.Status == "reconciling"))) ||
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
		if selected.Engine != "dsh" {
			h.runDesktopGoal(ctx, selected)
			continue
		}
		started[selected.ID] = true
		workers.Go(func() {
			h.runDesktopGoal(ctx, selected)
			select {
			case finished <- selected.ID:
			case <-ctx.Done():
			}
		})
		g.schedule()
	}
}

func (h *relaySessionHost) runDesktopGoal(ctx context.Context, selected acpSession) {
	request := &http.Request{Host: h.local.address, Header: http.Header{}}
	request.Header.Set("Sec-WebSocket-Protocol", "monkeyai.v1, "+h.local.protocol)
	run := context.WithValue(ctx, goalRuntimeContext{}, h.goals)
	if selected.Engine == "dsh" && selected.Goal.Status == "reconciling" {
		if err := h.reconcileDSHGoal(run, selected); err != nil && ctx.Err() == nil {
			slog.Debug("持续目标对账失败", "session_id", selected.ID, "reason", safeDesktopTurnFailure(err))
		}
		if row, err := h.live.load(selected.ID); err == nil && ctx.Err() == nil {
			h.goals.notify(ctx, "_monkeyai/session/state", map[string]any{
				"sessionId": row.ID, "eventSeq": row.historySequence(),
				"state": map[string]any{"goal": goalView(row)}, "finish": false,
			})
		}
		return
	}
	if selected.Engine == "dsh" && selected.Goal.Status == "clearing" {
		if err := h.clearDSHGoal(run, selected); err != nil && ctx.Err() == nil {
			h.live.mu.Lock()
			row := h.live.sessions[selected.ID]
			if row.Goal != nil && row.Goal.ID == selected.Goal.ID && row.Goal.Status == "clearing" {
				goal := *row.Goal
				goal.Status, goal.Reason = "paused", "dsh_goal_clear_failed"
				goal.Version++
				row.Goal = &goal
				_ = h.live.save(row)
			}
			h.live.mu.Unlock()
			slog.Debug("持续目标清除失败", "session_id", selected.ID, "reason", safeDesktopTurnFailure(err))
		}
		return
	}
	if _, err := h.promptACP(run, nil, request, selected.ID, selected.GoalRun, selected.GoalRun, selected.Goal.Objective, nil, nil); err != nil && ctx.Err() == nil {
		slog.Debug("持续目标执行结束", "session_id", selected.ID, "reason", safeDesktopTurnFailure(err))
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
	goal.Invocations = nil
	if row.GoalRun != "" && row.State == "running" {
		goal.ExecutionID = row.GoalRun
	}
	if n := len(row.Turns); n != 0 && row.Turns[n-1].Operation == row.GoalRun {
		if saved := row.Turns[n-1].GoalCheckpoint; saved != nil {
			if row.Engine != "dsh" {
				tokens, known := goalModelTokens(saved.Usage)
				goal.Usage.Tokens = min(goal.Usage.Tokens+tokens, maxGoalBudget)
				goal.Uncertain = goal.Uncertain || saved.Rounds != 0 && !known
			}
			goal.ActiveSeconds += int64(saved.ActiveTime / time.Second)
		}
	}
	if row.Engine == "dsh" && row.State == "running" && row.GoalRun != "" && goal.Current && goal.DSHActiveSince != nil {
		if elapsed := time.Since(*goal.DSHActiveSince); elapsed > 0 {
			goal.ActiveSeconds += int64(elapsed / time.Second)
		}
	}
	return &goal
}

func materializeGoalCheckpoint(turn *acpTurn) {
	if saved := turn.GoalCheckpoint; saved != nil {
		turn.DSHParts, turn.DSHThoughtProof = nil, nil
		if validDSHGoalThoughtProof(saved.DSHThoughtProof, turn.DSHGoalID, saved.Progress, saved.DSHParts) {
			turn.DSHParts, turn.DSHThoughtProof = saved.DSHParts, saved.DSHThoughtProof
		}
		turn.Answer, turn.ToolRuns, turn.Usage = saved.Progress, append([]workspaceToolRun(nil), saved.Runs...), saved.Usage
		for i := range turn.ToolRuns {
			if !validDSHGoalThoughtProof(turn.ToolRuns[i].DSHThoughtProof, turn.DSHGoalID, turn.ToolRuns[i].Preamble, turn.ToolRuns[i].DSHParts) {
				turn.ToolRuns[i].DSHParts = nil
			}
		}
		turn.Permissions = saved.Permissions
		turn.GoalCheckpoint = nil
	}
}
