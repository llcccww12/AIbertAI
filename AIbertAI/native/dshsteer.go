package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/coder/websocket"
)

const queueSteering = "dsh-steering"

// The caller holds live.mu. Keep the queue item reserved until the SDK
// confirms admission, so a finished turn cannot dispatch it a second time.
func (h *relaySessionHost) steerQueuedMessage(ctx context.Context, ws *websocket.Conn, r *http.Request, row acpSession, index int, result func(acpSession) map[string]any, accept func(any) error) (any, error) {
	live := h.live
	if index < 0 {
		live.mu.Unlock()
		return result(row), nil
	}
	item := row.Queue.Messages[index]
	if h.local == nil || live.dsh == nil || row.Engine != "dsh" || row.State != "running" ||
		len(row.Turns) == 0 || item.editID != "" || len(item.CanvasNodes) != 0 {
		live.mu.Unlock()
		return nil, errors.New("dsh_steer_unavailable")
	}
	slot, process := live.dsh.LeasePool.observer(row.ID)
	if process == nil {
		live.mu.Unlock()
		return nil, errors.New("dsh_steer_unavailable")
	}
	active := row.Turns[len(row.Turns)-1]
	row.Queue.Messages = slices.Clone(row.Queue.Messages)
	row.Queue.Messages[index].editID = queueSteering
	live.sessions[row.ID] = row
	live.mu.Unlock()

	// A window may close while the SDK is admitting the message.
	steerCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	err := h.sendDSHSteer(steerCtx, slot, process, row, active, item)

	live.mu.Lock()
	row = live.sessions[row.ID]
	index = slices.IndexFunc(row.Queue.Messages, func(message desktopQueuedMessage) bool { return message.ID == item.ID })
	if index >= 0 {
		row.Queue.Messages = slices.Clone(row.Queue.Messages)
		row.Queue.Messages[index].editID = ""
		live.sessions[row.ID] = row
		if err == nil {
			row.Queue.Messages = slices.Delete(row.Queue.Messages, index, index+1)
			row.advanceQueue()
			if saveErr := live.save(row); saveErr != nil {
				live.mu.Unlock()
				return nil, errors.New("session_write_failed")
			}
		}
	}
	start := live.claimQueue(row)
	live.mu.Unlock()
	if start {
		// If the SDK finished before admission, preserve the queued message
		// and let it proceed normally without ever cancelling the old turn.
		reply := result(row)
		return h.runSessionQueue(ctx, ws, r, row.ID, reply, func(current acpSession) error { return accept(result(current)) })
	}
	if err != nil {
		return nil, err
	}
	return result(row), nil
}

func (h *relaySessionHost) sendDSHSteer(ctx context.Context, slot *dshLeaseSlot, process *dshProcess, row acpSession, active acpTurn, item desktopQueuedMessage) error {
	input := dshImageInput{blocks: []map[string]string{{"type": "text", "text": item.Text}}}
	if len(item.Attachments) != 0 {
		auth, err := h.live.authorization(ctx)
		if err != nil {
			return err
		}
		selected, err := h.live.client.modelConfig(ctx, auth, active.Model)
		if err != nil {
			return err
		}
		input, _, err = h.prepareDSHImageInput(ctx, row, item.Text, item.Attachments, selected.SupportsVision)
		if err != nil {
			return err
		}
	}
	raw, err := slot.observerCall(ctx, process, "aibertai/steer", map[string]any{
		"sessionId": row.ID, "operationId": active.Operation, "requestId": item.OperationID, "contentBlocks": input.blocks,
	})
	if err != nil {
		return err
	}
	var receipt struct{ MessageID string }
	if json.Unmarshal(raw, &receipt) != nil || receipt.MessageID == "" {
		return errors.New("dsh_steer_admission_unverified")
	}
	return nil
}

type dshSteerMessage struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	At   int64  `json:"at"`
}

func readDSHSteeredMessage(raw json.RawMessage, activeOperation string) (dshSteerMessage, bool) {
	var frame struct {
		Event struct {
			Type      string
			Time      int64
			SurfaceOp dshHistorySurfaceOp
			Data      struct {
				ID     string
				Source struct {
					Kind, AibertaiOperationID string
				}
				Content []struct{ Type, Text string }
			}
		}
	}
	if json.Unmarshal(raw, &frame) != nil {
		return dshSteerMessage{}, false
	}
	event := frame.Event
	data := event.Data
	if event.Type != "user/message" || event.SurfaceOp.Op != "append" || data.Source.Kind != "user" ||
		data.ID == "" || !validUUID(data.Source.AibertaiOperationID) || data.Source.AibertaiOperationID == activeOperation ||
		len(data.Content) == 0 || data.Content[0].Type != "text" {
		return dshSteerMessage{}, false
	}
	return dshSteerMessage{ID: data.ID, Text: data.Content[0].Text, At: event.Time}, true
}
