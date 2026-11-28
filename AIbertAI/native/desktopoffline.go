package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type offlineDesktopIdentity struct {
	serverID, machineID string
}

// Existing session directories are bound to the account, server origin and
// local data key by the derived machine identity. An ambiguous match is unsafe.
func findOfflineDesktopIdentity(profile, key, origin, user string) offlineDesktopIdentity {
	dir := filepath.Join(profile, "source-runtime")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return offlineDesktopIdentity{}
	}
	var found offlineDesktopIdentity
	for _, entry := range entries {
		if !entry.IsDir() || !validUUID(entry.Name()) {
			continue
		}
		device, err := desktopRelayDevice(key, origin, entry.Name(), user)
		if err != nil {
			return offlineDesktopIdentity{}
		}
		machineDir := filepath.Join(dir, entry.Name(), device.machine)
		machine, err := os.Lstat(machineDir)
		if err != nil || !machine.IsDir() {
			continue
		}
		info, err := os.Lstat(filepath.Join(machineDir, "sessions"))
		if err != nil || !info.IsDir() {
			continue
		}
		if found.serverID != "" {
			return offlineDesktopIdentity{}
		}
		found = offlineDesktopIdentity{entry.Name(), device.machine}
	}
	return found
}

func desktopNetworkUnavailable(err error) bool {
	var transport *url.Error
	var response *backendHTTPError
	var socket *net.OpError
	switch websocket.CloseStatus(err) {
	case websocket.StatusAbnormalClosure, websocket.StatusInternalError, websocket.StatusServiceRestart, websocket.StatusTryAgainLater:
		return true
	}
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, errDesktopModelUnavailable) ||
		errors.As(err, &transport) || errors.As(err, &socket) ||
		errors.As(err, &response) && response.Status >= 500
}

func serveOfflineDesktop(ctx context.Context, profile, workspace, token, origin, userID string, identity offlineDesktopIdentity, source *desktopSourceAuth,
	check func(context.Context) error, send func(any) error) error {
	live, err := openEmployeeSessions(profile, identity.serverID, identity.machineID, workspace)
	if err != nil {
		return err
	}
	defer live.store.Close()
	defer live.workspaces.root.Close()
	live.source = source
	live.workspaces.shared = dshDefaultWorkspaceDirectory()
	attachments, err := openDesktopAttachments(profile)
	if err != nil {
		return err
	}
	defer attachments.Close()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()

	lifetime, stop := context.WithCancel(ctx)
	defer stop()
	host := &relaySessionHost{profile: profile, live: live, offline: true, goals: newGoalRuntime(),
		local: &desktopRuntimeAccess{address: listener.Addr().String(), origin: origin, protocol: "monkeyai.token." + token,
			serverID: identity.serverID, reportMachine: identity.machineID, userID: userID, attachments: attachments}}
	// The online host opens the same session store. Wait for offline requests
	// (including upgraded ACP sockets) to finish before releasing that store.
	var requests sync.WaitGroup
	var requestMu sync.Mutex
	closing := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMu.Lock()
		if closing {
			requestMu.Unlock()
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		requests.Add(1)
		requestMu.Unlock()
		defer requests.Done()
		host.ServeHTTP(w, r)
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, MaxHeaderBytes: 32 << 10,
		BaseContext: func(net.Listener) context.Context { return lifetime }}
	if err := send(map[string]any{"event": "ready", "port": listener.Addr().(*net.TCPAddr).Port, "version": version, "offline": true}); err != nil {
		return err
	}
	monitorDone := make(chan struct{})
	go func() {
		defer close(monitorDone)
		defer server.Close()
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-lifetime.Done():
				return
			case <-ticker.C:
				probe, cancel := context.WithTimeout(lifetime, 12*time.Second)
				err := check(probe)
				cancel()
				if err == nil {
					stop()
					return
				}
			}
		}
	}()
	err = server.Serve(listener)
	stop()
	<-monitorDone
	requestMu.Lock()
	closing = true
	requestMu.Unlock()
	requests.Wait()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func offlineDesktopMethod(method string) bool {
	switch method {
	case "session/list", "session/load", "_monkeyai/projects/list", "_monkeyai/session/messages", "_monkeyai/session/state/get",
		"_monkeyai/session/subscribe", "_monkeyai/session/outline", "_monkeyai/session/tool_output",
		"_monkeyai/drafts", "_monkeyai/draft/set", "_monkeyai/runtime/status",
		"_monkeyai/workspace/list", "_monkeyai/workspace/read", "_monkeyai/workspace/stat", "_monkeyai/workspace/resolve",
		"_monkeyai/session/files/preview", "_monkeyai/workspace/link", "_aibertai/session/attachment/read",
		"_monkeyai/auth/credentials":
		return true
	default:
		return false
	}
}
