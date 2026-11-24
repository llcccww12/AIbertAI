package main

import (
	"encoding/json"
	"slices"
)

// Only the SDK transaction identity and status belong in the employee transcript.
type dshCompactionNotice struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	AfterTools int    `json:"afterTools,omitempty"`
}

func (turn *acpTurn) applyDSHCompaction(event dshConnectorHistoryEvent, owner int64) *dshCompactionNotice {
	var data struct {
		CompactionID    string
		SourceCommandID string
		Turn            *int64
		Error           *string
	}
	if json.Unmarshal(event.Data, &data) != nil || !validUUID(data.CompactionID) || data.SourceCommandID != "" || data.Turn == nil || owner > 0 && *data.Turn != owner {
		return nil
	}
	status := "in_progress"
	switch event.Type {
	case "compaction/start":
	case "compaction/end":
		status = "completed"
		if data.Error != nil {
			status = "failed"
		}
	default:
		return nil
	}
	index := slices.IndexFunc(turn.DSHCompactions, func(n dshCompactionNotice) bool { return n.ID == data.CompactionID })
	if index < 0 {
		notice := dshCompactionNotice{ID: data.CompactionID, Status: status, AfterTools: len(turn.ToolRuns)}
		turn.DSHCompactions = append(slices.Clone(turn.DSHCompactions), notice)
		return &notice
	}
	if turn.DSHCompactions[index].Status != "in_progress" || status == "in_progress" {
		return nil
	}
	turn.DSHCompactions = slices.Clone(turn.DSHCompactions)
	turn.DSHCompactions[index].Status = status
	return &turn.DSHCompactions[index]
}

func (n dshCompactionNotice) update() map[string]any {
	return map[string]any{"sessionUpdate": "compaction_update", "compactionId": n.ID, "status": n.Status, "_meta": map[string]any{"kind": "auto", "dsh": true}}
}

func (n dshCompactionNotice) part(turn acpTurn) map[string]string {
	part := map[string]string{"type": "compaction", "kind": "auto", "compactionId": n.ID}
	if n.Status != "completed" {
		part["status"] = n.Status
		if n.Status == "in_progress" && (!turn.Ended.IsZero() || turn.Interrupted || turn.Failed) {
			part["status"] = "cancelled"
		}
	}
	return part
}
