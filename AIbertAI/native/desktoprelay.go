package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

const remotePreferenceFile = "remote-access.json"

type remotePreference struct {
	Enabled       bool  `json:"enabled"`
	PendingRevoke bool  `json:"pendingRevoke"`
	Registration  int64 `json:"registration"`
}

type desktopRelay struct {
	local     *relaySessionHost
	device    relayDevice
	user      account
	serverID  string
	supported bool
	uploads   *relayUploads
	wake      chan struct{}
	mu        sync.Mutex
	pref      remotePreference
	stateErr  error
	active    context.CancelFunc
	restore   bool
	suspended bool
	online    bool
	reason    string
}

func newDesktopRelay(host *relaySessionHost, device relayDevice, user account, ready relayReady, auth authorization) *desktopRelay {
	scopes := strings.Fields(auth.Scope)
	h := &desktopRelay{
		local: host, device: device, user: user, serverID: ready.ServerID, wake: make(chan struct{}, 1),
		supported: slices.Contains(scopes, "device:publish") && slices.Contains(scopes, "device:manage"),
		pref:      remotePreference{Registration: ready.RegistrationVersion},
	}
	root := host.live.store
	info, err := root.Lstat(remotePreferenceFile)
	if err == nil {
		err = h.readPreference(root.Root, info)
	} else if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	if err == nil {
		const dir = "remote-uploads"
		err = root.MkdirAll(dir, 0700)
		if err == nil {
			var uploads *os.Root
			uploads, err = root.OpenRoot(dir)
			if err == nil {
				h.uploads = openRelayUploads(uploads, device.private, true)
				h.uploads.openHistory = host.openHistoryFile
				h.uploads.openWorkspace = host.openWorkspaceFile
			}
		}
	}
	h.stateErr = err
	return h
}

func (h *desktopRelay) readPreference(root *os.Root, info os.FileInfo) error {
	if !info.Mode().IsRegular() || info.Size() > 2048 {
		return errors.New("invalid_remote_preference")
	}
	f, err := root.Open(remotePreferenceFile)
	if err != nil {
		return err
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 2049))
	var pref remotePreference
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) || d.Decode(&pref) != nil || d.Decode(new(any)) != io.EOF ||
		pref.Registration <= 0 || pref.Enabled && pref.PendingRevoke {
		return errors.New("invalid_remote_preference")
	}
	h.pref = pref
	return nil
}

func (h *desktopRelay) save(pref remotePreference) error {
	data, err := json.Marshal(pref)
	if err != nil {
		return err
	}
	if err := writeSourceState(h.local.live.store.Root, remotePreferenceFile, data); err != nil {
		return errors.New("device_state_unavailable")
	}
	h.pref = pref
	return nil
}

func (h *desktopRelay) signal() {
	select {
	case h.wake <- struct{}{}:
	default:
	}
}

func (h *desktopRelay) status() map[string]any {
	phase, reason := "disabled", h.reason
	switch {
	case h.stateErr != nil:
		reason = "device_state_unavailable"
	case !h.supported:
		reason = "permission_denied"
	case h.pref.PendingRevoke:
		phase = "revocation_pending"
	case h.online:
		phase = "online"
	case h.pref.Enabled && h.suspended:
		phase = "idle"
	case h.pref.Enabled:
		phase = "connecting"
	}
	return map[string]any{"supported": h.supported && h.stateErr == nil, "phase": phase, "reason": reason,
		"device": map[string]any{"enabled": h.pref.Enabled && h.stateErr == nil, "machineId": h.device.machine}}
}

func (h *desktopRelay) command(method string, raw json.RawMessage) (any, error) {
	var p struct {
		Enabled *bool
		Refresh bool
		State   string
	}
	if len(raw) > 0 && json.Unmarshal(raw, &p) != nil {
		return nil, errors.New("invalid_request")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	switch method {
	case "_monkeyai/device/remote/status":
		if p.Refresh && !h.online {
			h.signal()
		}
	case "_monkeyai/device/remote/power":
		if p.State != "suspend" && p.State != "resume" {
			return nil, errors.New("invalid_request")
		}
		h.suspended = p.State == "suspend"
		if h.suspended && h.active != nil {
			h.active()
			h.online = false
		}
		h.signal()
	case "_monkeyai/device/remote/set":
		if p.Enabled == nil {
			return nil, errors.New("invalid_request")
		}
		if h.stateErr != nil {
			return nil, errors.New("device_state_unavailable")
		}
		if !h.supported {
			return nil, errors.New("permission_denied")
		}
		if *p.Enabled && h.pref.PendingRevoke {
			return nil, errors.New("device_revocation_pending")
		}
		if h.pref.Enabled != *p.Enabled {
			pref := h.pref
			pref.Enabled = *p.Enabled
			pref.PendingRevoke = !pref.Enabled
			if !pref.Enabled && h.active != nil {
				h.active()
				h.online = false
			}
			if err := h.save(pref); err != nil {
				h.stateErr = err
				h.signal()
				return nil, err
			}
			h.restore, h.reason = pref.Enabled, ""
			h.signal()
		}
	}
	return h.status(), nil
}

func (h *desktopRelay) supervise(ctx context.Context, notify func()) {
	for ctx.Err() == nil {
		h.mu.Lock()
		pref := h.pref
		runnable := h.supported && h.stateErr == nil && !h.suspended && (pref.Enabled || pref.PendingRevoke)
		if !runnable {
			h.mu.Unlock()
			select {
			case <-ctx.Done():
				return
			case <-h.wake:
				notify()
			}
			continue
		}
		active, cancel := context.WithCancel(ctx)
		h.active = cancel
		device := h.device
		device.restoreRevoked, h.restore = h.restore, false
		h.mu.Unlock()

		var err error
		if pref.PendingRevoke {
			err = h.revoke(active, pref.Registration)
		} else {
			err = h.serve(active, device, notify)
		}
		h.mu.Lock()
		h.active, h.online = nil, false
		if active.Err() == nil {
			switch {
			case pref.PendingRevoke && (err == nil || err.Error() == "device_not_found"):
				h.stateErr = h.save(remotePreference{Registration: pref.Registration})
				h.reason = ""
			case !pref.PendingRevoke && err != nil && err.Error() == "device_revoked":
				h.stateErr = h.save(remotePreference{Registration: h.pref.Registration})
				h.reason = "device_revoked"
			case err != nil && knowledgeErrorCode.MatchString(err.Error()):
				h.reason = err.Error()
			case err != nil:
				h.reason = "authorization_state_unavailable"
			}
		}
		cancel()
		h.mu.Unlock()
		if ctx.Err() != nil {
			return
		}
		notify()
		select {
		case <-ctx.Done():
			return
		case <-h.wake:
		case <-time.After(3 * time.Second):
		}
	}
}

func (h *desktopRelay) revoke(ctx context.Context, registration int64) error {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	live := h.local.live
	auth, err := live.authorization(ctx)
	if err != nil {
		return err
	}
	_, err = live.client.knowledgeChange(ctx, auth, knowledgeOperation{
		Verb: http.MethodDelete, Path: "/api/v1/endpoints/" + h.device.machine, ETag: strconv.FormatInt(registration, 10),
	})
	return err
}

func (h *desktopRelay) serve(ctx context.Context, device relayDevice, notify func()) error {
	live := h.local.live
	c := live.client
	startup, endStartup := context.WithTimeout(ctx, 25*time.Second)
	defer endStartup()
	auth, err := live.authorization(startup)
	if err != nil {
		return err
	}
	serverID, keys, err := c.relaySigningKeys(startup)
	if err != nil {
		return err
	}
	if serverID != h.serverID {
		return errors.New("device_identity_mismatch")
	}
	controlWS, ready, err := c.relayHandshake(startup, auth, h.user, serverID, device, "publish", "control", 0)
	if err != nil {
		return err
	}
	defer controlWS.CloseNow()
	h.mu.Lock()
	// A concurrent disable keeps the newly observed registration for revocation.
	pref := h.pref
	pref.Registration = ready.RegistrationVersion
	if err = h.save(pref); err != nil {
		h.stateErr = err
	}
	h.mu.Unlock()
	if err != nil {
		return err
	}
	control, err := relayMux(ctx, controlWS)
	if err != nil {
		return err
	}
	defer control.Close()
	device.restoreRevoked = false
	dataWS, attached, err := c.relayHandshake(startup, auth, h.user, serverID, device, "attach", "data", ready.Epoch)
	if err != nil {
		return err
	}
	defer dataWS.CloseNow()
	if attached.RegistrationVersion != ready.RegistrationVersion || attached.GrantVersion != ready.GrantVersion {
		return errors.New("device_registration_changed")
	}
	data, err := relayMux(ctx, dataWS)
	if err != nil {
		return err
	}
	defer data.Close()
	var modelKey invocationKey
	if live.cloudWorkspace != "" {
		// The backend issues cloud model keys only after the device publishes
		// and binds this grant to its active execution endpoint.
		modelKey, err = c.createModelKey(startup, auth)
		if err != nil {
			return err
		}
		defer func() {
			cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
			defer stop()
			if current, err := live.authorization(cleanup); err == nil {
				_ = c.json(cleanup, "DELETE", "/api/v1/api-keys/"+modelKey.ID, current.Access, "", nil, nil)
			}
		}()
	}
	endStartup()
	remote := &relaySessionHost{
		access:  &relayAccess{issuer: c.server, owner: h.user.ID, boot: device.boot, ready: ready, keys: keys, control: control},
		profile: h.local.profile, root: h.local.root, approvedFile: h.local.approvedFile,
		live: live, uploads: h.uploads, shareDesktopSessions: true,
		terminals: h.local.terminals, terminalLifetime: ctx,
		cloudModelKey: modelKey.Value,
	}
	server := &http.Server{Handler: remote, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 20 * time.Second, MaxHeaderBytes: 32 << 10, BaseContext: func(net.Listener) context.Context { return ctx }}
	defer server.Close()
	h.mu.Lock()
	if ctx.Err() != nil {
		h.mu.Unlock()
		return ctx.Err()
	}
	h.online, h.reason = true, ""
	h.mu.Unlock()
	go server.Serve(dataRelayListener{mux: control})
	go server.Serve(dataRelayListener{mux: data})
	notify()
	select {
	case <-ctx.Done():
	case <-control.CloseChan():
	case <-data.CloseChan():
	}
	return nil
}
