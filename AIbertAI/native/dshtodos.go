package main

import (
	"context"
	"encoding/json"
	"slices"
	"time"
)

// DSH's todos projection is the latest complete todo/write in the current
// native turn. Its event sequence orders a live push against a history reload.
type dshTodoItem struct {
	Content  string `json:"content"`
	Status   string `json:"status"`
	Priority string `json:"priority"` // required by the ACP plan entry schema
}
type dshTodoSnapshot struct {
	Seq   int64         `json:"seq"`
	Items []dshTodoItem `json:"items"`
}

func (t *acpTurn) applyDSHTodo(e dshConnectorHistoryEvent) bool {
	if e.Seq == nil || *e.Seq < 0 || t.DSHTodos != nil && t.DSHTodos.Seq >= *e.Seq {
		return false
	}
	items := []dshTodoItem{}
	switch e.Type {
	case "turn/start":
	case "todo/write":
		var data struct{ Todos []dshTodoItem }
		if json.Unmarshal(e.Data, &data) != nil || data.Todos == nil {
			return false
		}
		for i, item := range data.Todos {
			if item.Content == "" || (item.Status != "pending" && item.Status != "in_progress" && item.Status != "completed") {
				return false
			}
			data.Todos[i].Priority = "medium"
		}
		items = data.Todos
	default:
		return false
	}
	t.DSHTodos = &dshTodoSnapshot{Seq: *e.Seq, Items: items}
	return true
}

func (s *dshTodoSnapshot) update() map[string]any {
	return map[string]any{"sessionUpdate": "plan", "entries": s.Items, "_meta": map[string]any{"dshTodoSeq": s.Seq}}
}

func (row acpSession) addDSHTodoMetadata(meta map[string]any) {
	if row.Engine != "dsh" {
		return
	}
	for i := len(row.Turns) - 1; i >= 0; i-- {
		if snapshot := row.Turns[i].DSHTodos; snapshot != nil {
			meta["plan"], meta["dshTodoSeq"] = snapshot.Items, snapshot.Seq
			return
		}
	}
}

// Pre-upgrade rows saved tool calls but not todo/write. A historical call
// only triggers a read; the employee sees the SDK's journaled status values.
func (h *relaySessionHost) restoreLegacyDSHTodos(ctx context.Context, id string) {
	row, err := h.live.load(id)
	if err != nil || row.Engine != "dsh" || row.ID != row.DSHSessionID || row.State == "running" || len(row.Turns) == 0 {
		return
	}
	sawTool := false
	for _, t := range row.Turns {
		if t.DSHTodos != nil {
			return
		}
		sawTool = sawTool || slices.ContainsFunc(t.ToolRuns, func(r workspaceToolRun) bool { return r.DSHCallID != "" && r.Report.Name == "todo_write" })
	}
	if !sawTool {
		return
	}
	var replay acpTurn
	read := func(fetch dshConnectorHistoryFetch) error {
		_, err := dshRepairHistory(ctx, row.ID, -1, fetch, func(_ int64, raw json.RawMessage) error {
			var e struct{ Type string }
			if json.Unmarshal(raw, &e) == nil && (e.Type == "turn/start" || e.Type == "todo/write") {
				var event dshConnectorHistoryEvent
				if err := json.Unmarshal(raw, &event); err != nil {
					return err
				}
				replay.applyDSHTodo(event)
			}
			return nil
		})
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	cfg := h.live.dsh
	if slot, p := cfg.LeasePool.observer(row.ID); p != nil {
		err = read(func(ctx context.Context, params map[string]any) (json.RawMessage, error) {
			return slot.observerCall(ctx, p, "aibertai/history", params)
		})
	} else {
		opts := dshSDKOptions{InspectSession: func(ctx context.Context, p *dshProcess) error { return read(p.historyCall) }}
		err = reconcileDSHSession(context.WithValue(ctx, dshSDKOptionsKey{}, opts), cfg, row)
	}
	if err != nil || replay.DSHTodos == nil {
		return
	}
	h.live.mu.Lock()
	defer h.live.mu.Unlock()
	current, ok := h.live.sessions[row.ID]
	index := len(row.Turns) - 1
	if !ok || current.State == "running" || len(current.Turns) != len(row.Turns) || current.Turns[index].Operation != row.Turns[index].Operation || current.Turns[index].DSHTodos != nil {
		return
	}
	current.Turns = slices.Clone(current.Turns)
	current.Turns[index].DSHTodos = replay.DSHTodos
	_ = h.live.save(current)
}
