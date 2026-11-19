package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
)

var errRelayAccess = errors.New("relay_access_denied")

type relayAuthorization struct {
	ServerID      string   `json:"server_id"`
	OwnerUserID   string   `json:"owner_user_id"`
	MachineID     string   `json:"machine_id"`
	ConnectionID  string   `json:"connection_id"`
	ClientGrantID string   `json:"client_grant_id"`
	GrantVersion  int64    `json:"grant_version"`
	AccessVersion int64    `json:"access_version"`
	Epoch         int64    `json:"lease_epoch"`
	Scopes        []string `json:"scopes"`
}

func (a relayAuthorization) same(b relayAuthorization) bool {
	return a.ServerID == b.ServerID && a.OwnerUserID == b.OwnerUserID && a.MachineID == b.MachineID &&
		a.ConnectionID == b.ConnectionID && a.ClientGrantID == b.ClientGrantID && a.GrantVersion == b.GrantVersion &&
		a.AccessVersion == b.AccessVersion && a.Epoch == b.Epoch && slices.Equal(a.Scopes, b.Scopes)
}

type relaySignedAccess struct {
	Type          string             `json:"type"`
	KeyID         string             `json:"kid"`
	Issuer        string             `json:"issuer"`
	Audience      string             `json:"audience"`
	Expires       time.Time          `json:"expires_at"`
	ServerTime    time.Time          `json:"server_time"`
	Authorization relayAuthorization `json:"authorization"`
	Nonce         string             `json:"nonce"`
	Boot          string             `json:"boot_id"`
	Sequence      int64              `json:"lease_seq"`
	TTL           int64              `json:"ttl_ms"`
	Remaining     int64              `json:"ticket_remaining_ms"`
	State         string             `json:"verification_state"`
}

type relayAccess struct {
	issuer, owner, boot string
	ready               relayReady
	keys                map[string]ed25519.PublicKey
	control             *yamux.Session
	mu                  sync.Mutex
	sequence            int64
}

func (h *relayAccess) verify(wire, kind string) (relaySignedAccess, error) {
	var value relaySignedAccess
	if len(wire) > 16<<10 {
		return value, errRelayAccess
	}
	encoded, signature, ok := strings.Cut(wire, ".")
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if !ok || err != nil || json.Unmarshal(data, &value) != nil {
		return value, errRelayAccess
	}
	sig, err := base64.RawURLEncoding.DecodeString(signature)
	key := h.keys[value.KeyID]
	if err != nil || len(key) != ed25519.PublicKeySize || !ed25519.Verify(key, []byte("monkeyai.access.signed.v1."+encoded), sig) ||
		value.Type != kind || value.Issuer != h.issuer || value.Audience != "monkeyai-runtime" {
		return value, errRelayAccess
	}
	return value, nil
}

type relayFileAccess struct {
	Auth                 relayAuthorization
	Deadline, ServerTime time.Time
}

func (h *relayAccess) authorize(r *http.Request, scope string) (time.Time, error) {
	access, err := h.authorizeAccess(r, scope)
	return access.Deadline, err
}

func (h *relayAccess) authorizeAccess(r *http.Request, scope string) (relayFileAccess, error) {
	values := r.Header.Values("X-MonkeyAI-Access")
	if r.Host != "device.internal" || len(values) != 1 {
		return relayFileAccess{}, errRelayAccess
	}
	assertion, err := h.verify(values[0], "monkeyai.access.assertion.v1")
	a := assertion.Authorization
	if err != nil || a.ServerID != h.ready.ServerID || a.OwnerUserID != h.owner || a.MachineID != h.ready.MachineID ||
		a.GrantVersion != h.ready.GrantVersion || a.Epoch != h.ready.Epoch || !validUUID(a.ConnectionID) ||
		!validUUID(a.ClientGrantID) || a.AccessVersion < 1 || !slices.Contains(a.Scopes, scope) || !assertion.Expires.After(time.Now()) {
		return relayFileAccess{}, errRelayAccess
	}
	return h.freshLease(r.Context(), a, assertion.Expires)
}

func (h *relayAccess) leaseContext(r *http.Request, scope string) (context.Context, context.CancelFunc, error) {
	lease, err := h.authorizeAccess(r, scope)
	if err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithCancel(r.Context())
	expiry := time.AfterFunc(time.Until(lease.Deadline), cancel)
	go func() {
		defer expiry.Stop()
		for ctx.Err() == nil {
			delay := max(time.Second, min(15*time.Second, time.Until(lease.Deadline)/2))
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			renewal, stop := context.WithDeadline(ctx, lease.Deadline)
			next, err := h.authorizeAccess(r.WithContext(renewal), scope)
			stop()
			if err != nil || ctx.Err() != nil || !expiry.Reset(time.Until(next.Deadline)) {
				cancel()
				return
			}
			lease = next
		}
	}()
	return ctx, cancel, nil
}

func (h *relayAccess) freshLease(ctx context.Context, auth relayAuthorization, expiry time.Time) (relayFileAccess, error) {
	// The relay requires strictly ordered lease sequences within each connection.
	h.mu.Lock()
	defer h.mu.Unlock()
	if ctx.Err() != nil || !expiry.After(time.Now()) {
		return relayFileAccess{}, errRelayAccess
	}
	started := time.Now()
	nonce, err := randomState()
	if err != nil {
		return relayFileAccess{}, err
	}
	h.sequence++
	data, err := json.Marshal(map[string]any{"type": "access_lease_request", "authorization": auth,
		"nonce": nonce, "lease_seq": h.sequence, "boot_id": h.boot, "require_fresh": true})
	if err != nil {
		return relayFileAccess{}, err
	}
	conn, err := h.control.OpenStream()
	if err != nil {
		return relayFileAccess{}, err
	}
	defer conn.Close()
	deadline := started.Add(8 * time.Second)
	if requestDeadline, ok := ctx.Deadline(); ok {
		if requestDeadline.Before(deadline) {
			deadline = requestDeadline
		}
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return relayFileAccess{}, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "http://relay.internal/_control/access-lease", bytes.NewReader(data))
	if err != nil {
		return relayFileAccess{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Close = true
	if err := req.Write(conn); err != nil {
		return relayFileAccess{}, err
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return relayFileAccess{}, err
	}
	defer resp.Body.Close()
	var reply struct{ Signed string }
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 32<<10)).Decode(&reply) != nil {
		return relayFileAccess{}, errRelayAccess
	}
	lease, err := h.verify(reply.Signed, "monkeyai.access.lease.v1")
	if err != nil || lease.Nonce != nonce || lease.Boot != h.boot || lease.Sequence != h.sequence || !auth.same(lease.Authorization) ||
		lease.State != "fresh" || lease.ServerTime.IsZero() || lease.TTL <= 0 || lease.TTL > 60000 || lease.Remaining < lease.TTL {
		return relayFileAccess{}, errRelayAccess
	}
	deadline = expiry
	if bounded := started.Add(time.Duration(lease.TTL) * time.Millisecond); bounded.Before(deadline) {
		deadline = bounded
	}
	if !deadline.After(time.Now()) {
		return relayFileAccess{}, errRelayAccess
	}
	return relayFileAccess{Auth: auth, Deadline: deadline, ServerTime: lease.ServerTime}, nil
}
