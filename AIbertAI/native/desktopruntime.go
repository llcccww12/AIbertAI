package main

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type desktopRuntimeAccess struct {
	address, origin, protocol string
	serverID, reportMachine   string
	userID, installedMachine  string
	clientVersion             string
	attachments               *os.Root
}

func desktopRelayDevice(encodedKey, origin, serverID, userID string) (relayDevice, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	defer clear(key)
	if err != nil || len(key) != 32 || !validUUID(serverID) || !validUUID(userID) {
		return relayDevice{}, errDesktopCredential
	}
	boot, err := newUUID()
	if err != nil {
		return relayDevice{}, err
	}
	mac := hmac.New(sha256.New, key)
	// Separate the desktop proof from legacy keys and keep it stable across boots.
	mac.Write([]byte("aibertai/desktop-device/v1\x00" + origin + "\x00" + serverID + "\x00" + userID))
	seed := mac.Sum(nil)
	defer clear(seed)
	private := ed25519.NewKeyFromSeed(seed)
	public := private.Public().(ed25519.PublicKey)
	sum := sha256.Sum256(public)
	var machine [16]byte
	copy(machine[:], sum[:])
	// The deployed relay uses the UUIDv4 wire format for machine identifiers.
	machine[6] = (machine[6] & 0x0f) | 0x40
	return relayDevice{machine: formatUUID(machine), boot: boot, name: "AIbertAI Desktop", public: public, private: private}, nil
}

func (a *desktopRuntimeAccess) authorize(r *http.Request) error {
	return a.authorizeProtocol(r, "monkeyai.v1")
}

func (a *desktopRuntimeAccess) authorizeProtocol(r *http.Request, protocol string) error {
	origin := r.Header.Values("Origin")
	protocols := websocketProtocols(r)
	if r.Host != a.address || (len(origin) != 0 && (len(origin) != 1 || origin[0] != a.origin)) ||
		len(protocols) != 2 || protocols[0] != protocol ||
		subtle.ConstantTimeCompare([]byte(protocols[1]), []byte(a.protocol)) != 1 {
		return errRelayAccess
	}
	return nil
}

func (h *relaySessionHost) reportMachine() string {
	if h.local != nil {
		return h.local.reportMachine
	}
	if h.access == nil && h.desktopRelay != nil {
		return h.desktopRelay.device.machine
	}
	return h.access.ready.MachineID
}

func (h *relaySessionHost) acpContext(r *http.Request) (context.Context, context.CancelFunc, error) {
	if h.local != nil {
		if err := h.local.authorize(r); err != nil {
			return nil, nil, err
		}
		ctx, cancel := context.WithCancel(r.Context())
		return ctx, cancel, nil
	}
	return h.access.leaseContext(r, "session:read")
}

func (h *relaySessionHost) acpScope(r *http.Request, scope string) error {
	if h.local != nil {
		return h.local.authorize(r)
	}
	_, err := h.access.authorize(r, scope)
	return err
}

func (h *relaySessionHost) sessionIDs() []string {
	if h.local == nil && !h.shareDesktopSessions {
		return []string{h.sessionID}
	}
	dir, err := os.Open(filepath.Join(h.profile, "runtime", "sessions"))
	if err != nil {
		return nil
	}
	defer dir.Close()
	entries, _ := dir.ReadDir(1000)
	var ids []string
	for _, entry := range entries {
		if entry.IsDir() && validUUID(entry.Name()) {
			ids = append(ids, entry.Name())
		}
	}
	return ids
}

func (h *relaySessionHost) listSessionMetadata(id string) (json.RawMessage, error) {
	if h.local == nil && !h.shareDesktopSessions {
		archive, err := readLegacySessionArchive(h.profile, id)
		return archive.Session, err
	}
	root, err := os.OpenRoot(h.profile)
	if err != nil {
		return nil, errInvalidSessionArchive
	}
	defer root.Close()
	dir, err := root.OpenRoot(filepath.Join("runtime", "sessions", id))
	if err != nil {
		return nil, errInvalidSessionArchive
	}
	defer dir.Close()
	meta, err := readLegacySessionMetadata(dir)
	if err != nil {
		return nil, err
	}
	var row legacySessionMetadata
	if json.Unmarshal(meta, &row) != nil || row.ID != id || row.SchemaVersion != legacySessionSchema || row.ReplayVersion != legacyReplayVersion {
		return nil, errInvalidSessionArchive
	}
	return meta, nil
}

func (h *relaySessionHost) localQuery(ctx context.Context, method string, raw json.RawMessage) (any, error) {
	switch method {
	case "_monkeyai/config/get", "_monkeyai/config/sync":
		return h.employeeConfig(ctx)
	case "_monkeyai/resources/preferences/set":
		if err := h.live.saveConnectorPreferences(raw); err != nil {
			return nil, err
		}
		return h.employeeConfig(ctx)
	case "_monkeyai/config/goals/set", "_monkeyai/config/display/set":
		if err := h.live.saveDesktopSettings(method, raw); err != nil {
			return nil, err
		}
		return h.employeeConfig(ctx)
	case "_monkeyai/billing/account", "_monkeyai/billing/entries":
		return h.desktopBilling(ctx, method, raw)
	case "_monkeyai/profile/avatar":
		return h.desktopAvatar(ctx, raw)
	case "_monkeyai/environments/list", "_monkeyai/environments/get", "_monkeyai/environments/operations", "_monkeyai/environments/change":
		return h.desktopEnvironment(ctx, method, raw)
	case "_monkeyai/preferences/model-thinking/set":
		if err := h.live.saveDesktopSettings(method, raw); err != nil {
			return nil, err
		}
		return raw, nil
	case "_monkeyai/resources/recipients", "_monkeyai/resources/share", "_monkeyai/users/search", "_monkeyai/groups/search":
		return h.desktopSharing(ctx, method, raw)
	case "_monkeyai/drafts", "_monkeyai/draft/set":
		return h.live.desktopDrafts(method, raw)
	case "_monkeyai/config/models/get", "_monkeyai/config/models/save", "_monkeyai/config/models/delete", "_monkeyai/config/models/create-many", "_monkeyai/config/models/test", "_monkeyai/config/models/discover":
		return h.desktopModelSettings(ctx, method, raw)
	case "_monkeyai/resources/get", "_monkeyai/resources/save", "_monkeyai/resources/delete",
		"_monkeyai/resources/experts/details", "_monkeyai/resources/experts/tags", "_monkeyai/resources/connectors/tags", "_monkeyai/resources/skills/tags",
		"_monkeyai/resources/skills/content", "_monkeyai/resources/skills/manifest", "_monkeyai/resources/skills/create",
		"_monkeyai/resources/connectors/test", "_monkeyai/resources/connectors/authorize",
		"_monkeyai/resources/connectors/authorization", "_monkeyai/resources/connectors/credentials",
		"_monkeyai/resources/connectors/credentials/get", "_monkeyai/resources/connectors/credentials/create",
		"_monkeyai/resources/connectors/credentials/update", "_monkeyai/resources/connectors/credentials/revoke":
		return h.desktopResource(ctx, method, raw)
	case "_monkeyai/devices/list":
		return h.desktopDevices(ctx)
	case "_monkeyai/devices/connect":
		return h.desktopConnect(ctx, raw)
	case "_monkeyai/device/remote/status", "_monkeyai/device/remote/set", "_monkeyai/device/remote/power":
		return h.desktopRelay.command(method, raw)
	case "_monkeyai/runtime/status":
		return map[string]any{"runtimeVersion": version, "legacyResourceImport": h.legacyImport}, nil
	}
	var p historyQuery
	if json.Unmarshal(raw, &p) != nil {
		return nil, errors.New("invalid_session_history")
	}
	if method == "_monkeyai/session/tool_output" {
		return h.historyToolOutput(p)
	}
	if !p.validate() {
		return nil, errors.New("invalid_session_history")
	}
	if session, err := h.live.load(p.SessionID); err == nil {
		page, err := session.historyWindow(p, nil)
		if err == nil && h.live.workspaces != nil {
			if root, rootErr := h.live.workspaces.sessionRoot(session); rootErr == nil {
				if design := workspaceDesign(root, session.ID); design != nil {
					meta := page["session"].(map[string]any)
					meta["capabilities"] = map[string]json.RawMessage{"design": design}
					if session.ExpertID == "" && h.local != nil && h.designBridge != nil {
						meta["expertId"] = designExpertID
					}
				}
				root.Close()
			}
		}
		return page, err
	}
	archive, err := readLegacySessionArchive(h.profile, p.SessionID)
	if err != nil {
		return nil, errors.New("session_not_found")
	}
	page, err := legacyHistoryWindow(archive, &p)
	if err != nil {
		return nil, err
	}
	decorateLegacyAttachments(&page, archive, p.SessionID, nil)
	return page, nil
}

func historyRelayRef(auth relayAuthorization, sessionID, path string) relayFileRef {
	return relayFileRef{ServerID: auth.ServerID, UserID: auth.OwnerUserID, MachineID: auth.MachineID, SessionID: sessionID, Path: path}
}

func decorateLegacyAttachments(page *archivedHistory, archive legacySessionArchive, sessionID string, auth *relayAuthorization) {
	for i, rawMessage := range page.Messages {
		var message map[string]json.RawMessage
		var parts []map[string]any
		if json.Unmarshal(rawMessage, &message) != nil || string(message["role"]) != `"user"` ||
			json.Unmarshal(message["parts"], &parts) != nil {
			continue
		}
		changed := false
		for _, part := range parts {
			uri, ok := part["uri"].(string)
			if part["type"] != "file" || !ok {
				continue
			}
			name, _ := part["name"].(string)
			_, err := legacyAttachmentPath(archive, uri, name)
			if auth != nil {
				delete(part, "uri")
				delete(part, "employeeAttachment")
				if refPath := historyFilePath(uri, name); err == nil && refPath != "" {
					part["fileRef"] = historyRelayRef(*auth, sessionID, refPath)
				}
				changed = true
			} else if err == nil {
				part["employeeAttachment"] = true
				changed = true
			}
		}
		if changed {
			message["parts"], _ = json.Marshal(parts)
			page.Messages[i], _ = json.Marshal(message)
		}
	}
}

func desktopRuntime(ctx context.Context, profile, file string, in io.Reader, out io.Writer) (err error) {
	reader := bufio.NewReaderSize(in, 64<<10)
	line, readErr := reader.ReadSlice('\n')
	var init struct {
		Type, Token, WebOrigin, ServerURL, UserID, DataKey, MachineID string
		ClientVersion                                                 string
		DesignBridge                                                  bool
	}
	if readErr != nil || json.Unmarshal(line, &init) != nil || init.Type != "init" || !validUUID(init.UserID) ||
		(backendOrigin != "" && init.ServerURL != backendOrigin) || (modelID != "" && !validUUID(modelID)) ||
		!filepath.IsAbs(profile) || file != "" && !filepath.IsAbs(file) {
		return errors.New("invalid_desktop_runtime_configuration")
	}
	clear(line)
	token, tokenErr := base64.RawURLEncoding.DecodeString(init.Token)
	clear(token)
	origin, originErr := url.Parse(init.WebOrigin)
	if tokenErr != nil || len(token) != 32 || originErr != nil || origin.Hostname() == "" ||
		(origin.Scheme != "http" && origin.Scheme != "https" && init.WebOrigin != "app://monkeyai") ||
		origin.User != nil || init.WebOrigin != origin.Scheme+"://"+origin.Host {
		return errors.New("invalid_desktop_runtime_transport")
	}
	lifetime, cancel := context.WithCancel(ctx)
	defer cancel()
	events := json.NewEncoder(out)
	var eventMu sync.Mutex
	sendEvent := func(value any) error {
		eventMu.Lock()
		defer eventMu.Unlock()
		return events.Encode(value)
	}
	var designBridge *desktopDesignBridge
	if init.DesignBridge {
		designBridge = &desktopDesignBridge{send: sendEvent, pending: make(map[string]chan designToolResult)}
	}
	go func() {
		if designBridge != nil {
			designBridge.read(reader)
		} else {
			_, _ = io.Copy(io.Discard, reader)
		}
		cancel()
	}()
	startup, endStartup := context.WithTimeout(lifetime, 5*time.Minute)
	defer endStartup()
	var root *os.Root
	var name, verificationDirectory string
	if file != "" {
		verificationDirectory, name = filepath.Dir(file), filepath.Base(file)
		root, err = os.OpenRoot(verificationDirectory)
		if err != nil {
			return errors.New("approved_workspace_unavailable")
		}
		defer root.Close()
		if _, err := readApprovedToolFile(root, name, name); err != nil {
			return err
		}
	}
	c, err := newBackendClient(init.ServerURL, true)
	if err != nil {
		return err
	}
	store, err := openSourceGrantStore(profile, init.DataKey, init.ServerURL, init.UserID)
	if err != nil {
		return err
	}
	defer store.root.Close()
	grant, err := store.load()
	missingGrant := errors.Is(err, os.ErrNotExist)
	source := &desktopSourceAuth{client: c, store: store, grant: grant}
	var auth authorization
	if err == nil {
		auth, err = source.access(startup)
	}
	if missingGrant || errors.Is(err, errSourceGrantRejected) {
		// Use the browser flow for account migration and definitively rejected
		// grants. Replace the stored grant only after the same account consents.
		authorization, signedIn, loginErr := c.loginWithScopes(startup, sourceGrantScopes, func(link string) error {
			return sendEvent(map[string]string{"event": "source_authorize", "url": link})
		})
		if loginErr == nil && signedIn.ID != init.UserID {
			loginErr = errDesktopCredential
		}
		if loginErr == nil {
			source.grant, loginErr = store.replace(authorization)
		}
		if loginErr != nil {
			if authorization.Refresh != "" {
				cleanup, stopCleanup := context.WithTimeout(context.Background(), 3*time.Second)
				_ = c.revoke(cleanup, authorization)
				stopCleanup()
			}
			return fmt.Errorf("desktop authorization: %w", loginErr)
		}
		auth, err = authorization, nil
	}
	if err != nil {
		return fmt.Errorf("desktop authorization: %w", err)
	}
	var user account
	if err := c.json(startup, http.MethodGet, "/api/v1/me", auth.Access, "", nil, &user); err != nil {
		return err
	}
	if user.ID != init.UserID {
		return errDesktopCredential
	}
	model, err := c.selectedModelConfig(startup, auth)
	if err != nil {
		return err
	}
	modelLocked := modelID != ""
	modelID = model.ID
	serverID, _, err := c.relaySigningKeys(startup)
	if err != nil {
		return fmt.Errorf("desktop server identity: %w", err)
	}
	device, err := desktopRelayDevice(init.DataKey, init.ServerURL, serverID, init.UserID)
	init.DataKey = ""
	if err != nil {
		return err
	}
	bridge, ready, err := c.relayHandshake(startup, auth, user, serverID, device, "bind_identity", "control", 0)
	if err != nil {
		return fmt.Errorf("desktop device identity: %w", err)
	}
	_ = bridge.CloseNow()
	reportDevice := func(ctx context.Context, auth authorization) error { return c.reportDevice(ctx, auth, device) }
	if err := reportDevice(startup, auth); err != nil {
		fmt.Fprintf(os.Stderr, "device report failed: %v\n", err)
	}
	key, err := c.createModelKey(startup, auth)
	if validUUID(key.ID) {
		defer func() {
			cleanup, cancelCleanup := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancelCleanup()
			current, readErr := source.access(cleanup)
			if readErr != nil {
				err = errors.Join(err, readErr)
				return
			}
			err = errors.Join(err, c.json(cleanup, http.MethodDelete, "/api/v1/api-keys/"+key.ID, current.Access, "", nil, nil))
		}()
	}
	if err != nil {
		return fmt.Errorf("desktop model authorization: %w", err)
	}
	live, err := openEmployeeSessions(profile, serverID, device.machine, verificationDirectory)
	if err != nil {
		return err
	}
	// New conversations without a project share one visible folder, as in DSH.
	live.workspaces.shared = dshDefaultWorkspaceDirectory()
	// Like DSH's first-use workspace, it is listed as the 默认工作区 project.
	if project, err := live.workspaces.seedDefaultProject(); err != nil {
		fmt.Fprintf(os.Stderr, "default workspace project unavailable: %v\n", err)
	} else if project != "" {
		if err := live.fileSharedDefaultSessions(project); err != nil {
			fmt.Fprintf(os.Stderr, "default workspace sessions not filed: %v\n", err)
		}
	}
	live.client, live.source, live.key, live.protocol = c, source, key.Value, model.Protocol
	live.modelBudget = model.inputBudget()
	live.modelLocked = modelLocked
	live.deviceReport, live.deviceReportedAt = reportDevice, time.Now()
	if err := configureInstalledDSH(live, profile, serverID, device.machine); err != nil {
		live.store.Close()
		live.workspaces.root.Close()
		return err
	}
	defer live.store.Close()
	defer live.workspaces.root.Close()
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
	host := &relaySessionHost{profile: profile, root: root, approvedFile: name,
		local: &desktopRuntimeAccess{address: listener.Addr().String(), origin: init.WebOrigin,
			protocol: "monkeyai.token." + init.Token, serverID: serverID, reportMachine: ready.MachineID,
			userID: init.UserID, installedMachine: init.MachineID, clientVersion: init.ClientVersion, attachments: attachments},
		live: live, designBridge: designBridge, goals: newGoalRuntime()}
	if live.dsh != nil {
		host.interactions = newDesktopDSHInteractions()
		defer func() {
			shutdown, stop := context.WithTimeout(context.Background(), 10*time.Second)
			defer stop()
			_ = live.dsh.LeasePool.close(shutdown)
		}()
	}
	host.terminals, err = newDesktopTerminals()
	if err != nil {
		return err
	}
	host.terminalLifetime = lifetime
	defer host.terminals.shutdown()
	host.desktopRelay = newDesktopRelay(host, device, user, ready, auth)
	if host.desktopRelay.uploads != nil {
		defer host.desktopRelay.uploads.root.Close()
	}
	if _, err := host.employeeConfig(startup); err != nil {
		return err
	}
	importCtx, stopImport := context.WithTimeout(startup, 15*time.Second)
	host.legacyImport = host.importLegacyResources(importCtx, auth, store.cipher)
	stopImport()
	if len(host.legacyImport.Pending) != 0 {
		slog.Warn("旧本机资源还有待迁移项", "status", host.legacyImport.Pending)
	}
	recovery, stopRecovery := context.WithTimeout(startup, 10*time.Second)
	host.recoverDesktopReports(recovery)
	stopRecovery()
	endStartup()
	server := &http.Server{Handler: host, ReadHeaderTimeout: 5 * time.Second, MaxHeaderBytes: 32 << 10,
		BaseContext: func(net.Listener) context.Context { return lifetime }}
	go func() {
		<-lifetime.Done()
		_ = server.Close()
	}()
	readyEvent := map[string]any{"event": "ready", "port": listener.Addr().(*net.TCPAddr).Port, "version": version}
	if live.dsh != nil {
		readyEvent["dshResources"] = live.dsh.VerifiedResourceRoot
	}
	if err := sendEvent(readyEvent); err != nil {
		return err
	}
	remoteDone := make(chan struct{})
	reportDone := make(chan struct{})
	goalDone := make(chan struct{})
	go func() {
		defer close(goalDone)
		host.superviseGoals(lifetime)
	}()
	go func() {
		defer close(reportDone)
		host.superviseDesktopReports(lifetime, 30*time.Second)
	}()
	go func() {
		defer close(remoteDone)
		host.desktopRelay.supervise(lifetime, func() {
			_ = sendEvent(map[string]string{"event": "remote_changed"})
		})
	}()
	defer func() {
		cancel()
		<-remoteDone
		<-reportDone
		<-goalDone
	}()
	err = server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
