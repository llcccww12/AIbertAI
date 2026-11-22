package main

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const dshLeaseShutdownTimeout = 3 * time.Second

// A slot serializes all users of one SDK process. It remains in the map so a
// waiting caller cannot race with process eviction and create a second child.
type dshLeaseSlot struct {
	observerOwner   atomic.Pointer[dshProcess]
	gate            chan struct{}
	process         *dshProcess
	local           *dshLocalGateway
	cleanup         func()
	stop            context.CancelFunc
	cwd, model      string
	standardMode    bool
	disabledPlugins []string
}

type dshLeasePool struct {
	mu     sync.Mutex
	slots  map[string]*dshLeaseSlot
	closed bool
}

func (pool *dshLeasePool) acquire(ctx context.Context, id string) (*dshLeaseSlot, error) {
	pool.mu.Lock()
	if pool.closed {
		pool.mu.Unlock()
		return nil, errors.New("dsh_lease_pool_closed")
	}
	if pool.slots == nil {
		pool.slots = make(map[string]*dshLeaseSlot)
	}
	slot := pool.slots[id]
	if slot == nil {
		slot = &dshLeaseSlot{gate: make(chan struct{}, 1)}
		pool.slots[id] = slot
	}
	pool.mu.Unlock()
	select {
	case slot.gate <- struct{}{}:
		pool.mu.Lock()
		closed := pool.closed
		pool.mu.Unlock()
		if closed {
			<-slot.gate
			return nil, errors.New("dsh_lease_pool_closed")
		}
		return slot, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (slot *dshLeaseSlot) discard() {
	slot.observerOwner.Store(nil)
	if slot.process != nil {
		// A bounded graceful shutdown permits the SDK to cancel owned jobs.
		ctx, cancel := context.WithTimeout(context.Background(), dshLeaseShutdownTimeout)
		_, _ = slot.process.call(ctx, "shutdown", nil, nil)
		cancel()
		slot.process.close()
	}
	if slot.stop != nil {
		slot.stop()
	}
	if slot.cleanup != nil {
		slot.cleanup()
	}
	slot.process, slot.local, slot.cleanup, slot.stop = nil, nil, nil, nil
	slot.cwd, slot.model = "", ""
	slot.disabledPlugins = nil
}

func (pool *dshLeasePool) close(ctx context.Context) error {
	pool.mu.Lock()
	pool.closed = true
	slots := make([]*dshLeaseSlot, 0, len(pool.slots))
	for _, slot := range pool.slots {
		slots = append(slots, slot)
	}
	pool.mu.Unlock()
	var result error
	for _, slot := range slots {
		select {
		case slot.gate <- struct{}{}:
			slot.discard()
			<-slot.gate
		case <-ctx.Done():
			result = errors.Join(result, ctx.Err())
		}
	}
	return result
}

func dshLeaseRequestMatches(raw json.RawMessage, sessionID, operationID string) bool {
	var request struct {
		SessionID   string `json:"sessionId"`
		OperationID string `json:"operationId"`
	}
	return json.Unmarshal(raw, &request) == nil && request.SessionID == sessionID && request.OperationID == operationID
}

func dshLeaseEventMatches(raw json.RawMessage, sessionID, operationID string) bool {
	var event struct {
		SessionID   string `json:"sessionId"`
		OperationID string `json:"aibertaiOperationId"`
	}
	return json.Unmarshal(raw, &event) == nil && event.SessionID == sessionID && event.OperationID == operationID
}

func (pool *dshLeasePool) isClosed() bool {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	return pool.closed
}

// Caller serializes session admission while evicting the idle SDK owner.
func (pool *dshLeasePool) evictIfIdle(sessionID string) error {
	pool.mu.Lock()
	slot := pool.slots[sessionID]
	pool.mu.Unlock()
	if slot == nil {
		return nil
	}
	select {
	case slot.gate <- struct{}{}:
		defer func() { <-slot.gate }()
		slot.discard()
		return nil
	default:
		return errors.New("session_busy")
	}
}
