package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

// Reject before reserving a new turn or sending anything to the SDK. A 404 is
// an older server without this optional capability; its existing behavior is
// preserved. Once the SDK has accepted a prompt, the gateway remains the
// authoritative dispatch guard even if readiness subsequently changes.
func (h *relaySessionHost) preflightOwnedModel(ctx context.Context, id, op string) error {
	if h.local == nil || h.live == nil || h.live.client == nil {
		return nil
	}
	live := h.live
	live.mu.Lock()
	session, ok := live.sessions[id]
	if ok {
		for _, turn := range session.Turns {
			if turn.Operation == op {
				ok = false
				break
			}
		}
	}
	live.mu.Unlock()
	if !ok || session.Engine != "dsh" {
		return nil
	}
	auth, err := live.authorization(ctx)
	if err != nil {
		return nil
	}
	selected, err := live.client.modelConfig(ctx, auth, session.Model)
	if err != nil || selected.OwnershipType != "user" {
		return nil
	}
	resp, err := live.client.request(ctx, http.MethodGet, "/api/v1/billing/owned-model-invocation-readiness", auth.Access, "", "", "", nil, http.StatusNotFound)
	if err != nil {
		return errors.New("owned_model_index_unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	var body struct {
		Ready *bool `json:"ready"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || body.Ready == nil || !*body.Ready {
		return errors.New("owned_model_index_unavailable")
	}
	return nil
}
