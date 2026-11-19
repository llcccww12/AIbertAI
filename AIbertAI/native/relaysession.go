package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/hashicorp/yamux"
)

type relaySessionHost struct {
	access               *relayAccess
	local                *desktopRuntimeAccess
	profile, sessionID   string
	root                 *os.Root
	approvedFile         string
	live                 *acpLive
	uploads              *relayUploads
	desktopRelay         *desktopRelay
	shareDesktopSessions bool
	terminals            *desktopTerminals
	terminalLifetime     context.Context
	trace                bool
	designBridge         *desktopDesignBridge
}

type archivedHistory struct {
	Session     map[string]json.RawMessage `json:"session"`
	Messages    []json.RawMessage          `json:"messages"`
	ToolCalls   map[string]json.RawMessage `json:"toolCalls"`
	EventSeq    int64                      `json:"eventSeq"`
	OlderCursor *string                    `json:"olderCursor"`
	NewerCursor *string                    `json:"newerCursor"`
}

func legacyHistoryPage(archive legacySessionArchive) (archivedHistory, error) {
	return legacyHistoryWindow(archive, nil)
}

func legacyHistoryWindow(archive legacySessionArchive, query *historyQuery) (archivedHistory, error) {
	var page archivedHistory
	if json.Unmarshal(archive.Session, &page.Session) != nil || json.Unmarshal(page.Session["view_seq"], &page.EventSeq) != nil || page.EventSeq < 0 ||
		(string(page.Session["status"]) != `"idle"` && string(page.Session["status"]) != `"error"`) ||
		!bytes.Equal(page.Session["replay_applied_seq"], page.Session["last_event_seq"]) {
		return page, errInvalidSessionArchive
	}
	replay := archive.Replay
	if query != nil {
		var generation string
		if json.Unmarshal(page.Session["replay_generation"], &generation) != nil || generation == "" {
			return page, errInvalidSessionArchive
		}
		start, end, older, newer, err := query.window(generation, len(replay))
		if err != nil {
			return page, err
		}
		replay, page.OlderCursor, page.NewerCursor = replay[start:end], older, newer
	}
	for _, key := range []string{"schema_version", "replay_version", "replay_generation", "replay_applied_seq", "replay_bytes", "last_event_seq", "events_offset", "message_count", "view_seq", "engine_session_id"} {
		delete(page.Session, key)
	}
	page.Messages = make([]json.RawMessage, 0, len(replay))
	page.ToolCalls = map[string]json.RawMessage{}
	for _, record := range replay {
		var item struct {
			Message   json.RawMessage            `json:"message"`
			ToolCalls map[string]json.RawMessage `json:"toolCalls"`
		}
		if json.Unmarshal(record, &item) != nil {
			return page, errInvalidSessionArchive
		}
		page.Messages = append(page.Messages, item.Message)
		for key, call := range item.ToolCalls {
			page.ToolCalls[key] = call
		}
	}
	return page, nil
}

func (h *relaySessionHost) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/acp" {
		h.acp(w, r)
		return
	}
	if r.URL.Path == "/terminal" {
		h.serveTerminal(w, r)
		return
	}
	if h.local != nil {
		if r.URL.Path == localWorkspaceFileRoute && h.live != nil && h.live.workspaces != nil {
			h.localWorkspaceFile(w, r)
			return
		}
		http.NotFound(w, r)
		return
	}
	if h.uploads != nil && (strings.HasPrefix(r.URL.Path, "/uploads/") || strings.HasPrefix(r.URL.Path, "/files/")) {
		h.transferFile(w, r)
		return
	}
	var scope string
	switch r.URL.Path {
	case "/read/session":
		scope = "session:read"
	case "/read/config":
		scope = "resources:manage"
	case "/cap/workspace/mcp":
		scope = "files:read"
	default:
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost || r.URL.RawQuery != "" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	access, err := h.access.authorizeAccess(r, scope)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	response := http.NewResponseController(w)
	if response.SetWriteDeadline(access.Deadline) != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if scope == "session:read" {
		h.history(w, r, access.Auth)
		return
	}
	if scope == "resources:manage" {
		h.readConfig(w, r, access.Auth)
		return
	}
	h.tool(w, r)
}

func (h *relaySessionHost) transferFile(w http.ResponseWriter, r *http.Request) {
	id, upload := strings.CutPrefix(r.URL.Path, "/uploads/")
	scope := "files:read"
	if upload {
		scope = "files:write"
	}
	if r.URL.RawQuery != "" || upload && (!validUUID(id) || r.Method != http.MethodPut && r.Method != http.MethodHead) ||
		!upload && r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	access, err := h.access.authorizeAccess(r, scope)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if http.NewResponseController(w).SetWriteDeadline(access.Deadline) != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if upload {
		h.uploads.transfer(w, r, id, access.Auth)
		return
	}
	h.uploads.download(w, r, access)
}

func writeRelayResult(w http.ResponseWriter, auth relayAuthorization, result any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"target": map[string]string{"serverId": auth.ServerID,
		"userId": auth.OwnerUserID, "machineId": auth.MachineID}, "result": result})
}

func (h *relaySessionHost) history(w http.ResponseWriter, r *http.Request, auth relayAuthorization) {
	var req struct {
		Method string       `json:"method"`
		Params historyQuery `json:"params"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || !validUUID(req.Params.SessionID) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.Method == "_monkeyai/session/queue" && req.Params.Action == "get" {
		result, err := h.readQueue(req.Params.SessionID)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		writeRelayResult(w, auth, result)
		return
	}
	if req.Method == "_monkeyai/session/tool_output" {
		result, err := h.historyToolOutput(req.Params)
		if err != nil {
			http.NotFound(w, r)
		} else {
			writeRelayResult(w, auth, result)
		}
		return
	}
	if req.Method != "_monkeyai/session/messages" || !req.Params.validate() {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.Params.SessionID != h.sessionID {
		if h.live != nil {
			if session, err := h.live.load(req.Params.SessionID); err == nil {
				page, err := session.historyWindow(req.Params, &auth)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				writeRelayResult(w, auth, page)
				return
			}
		}
		if !h.shareDesktopSessions {
			http.NotFound(w, r)
			return
		}
	}
	archive, err := readLegacySessionArchive(h.profile, req.Params.SessionID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	page, err := legacyHistoryWindow(archive, &req.Params)
	if err != nil {
		status := http.StatusConflict
		if errors.Is(err, errHistoryCursor) {
			status = http.StatusBadRequest
		}
		w.WriteHeader(status)
		return
	}
	if h.shareDesktopSessions {
		decorateLegacyAttachments(&page, archive, req.Params.SessionID, &auth)
	}
	writeRelayResult(w, auth, page)
}

func (h *relaySessionHost) tool(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
		Params  struct {
			Name      string `json:"name"`
			Arguments struct {
				Path string `json:"path"`
			} `json:"arguments"`
		} `json:"params"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.JSONRPC != "2.0" || len(req.ID) == 0 ||
		req.Method != "tools/call" || req.Params.Name != workspaceReadTool {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	result := map[string]any{"jsonrpc": "2.0", "id": req.ID}
	text, err := readApprovedToolFile(h.root, h.approvedFile, req.Params.Arguments.Path)
	if err != nil {
		result["error"] = map[string]any{"code": -32602, "message": err.Error()}
	} else {
		result["result"] = map[string]any{"content": []map[string]string{{"type": "text", "text": text}}}
	}
	_ = json.NewEncoder(w).Encode(result)
}

type dataRelayListener struct{ mux *yamux.Session }

func (l dataRelayListener) Accept() (net.Conn, error) { return l.mux.AcceptStream() }
func (l dataRelayListener) Close() error              { return l.mux.Close() }
func (l dataRelayListener) Addr() net.Addr            { return &net.TCPAddr{} }

func relayMux(ctx context.Context, ws *websocket.Conn) (*yamux.Session, error) {
	ws.SetReadLimit(1 << 20)
	config := yamux.DefaultConfig()
	config.AcceptBacklog = 16
	config.ConnectionWriteTimeout = 10 * time.Second
	config.StreamOpenTimeout = 8 * time.Second
	config.LogOutput = io.Discard
	return yamux.Client(websocket.NetConn(ctx, ws, websocket.MessageBinary), config)
}

func (c *client) relaySigningKeys(ctx context.Context) (string, map[string]ed25519.PublicKey, error) {
	var metadata struct {
		Issuer  string `json:"issuer"`
		Runtime struct {
			ServerID  string `json:"server_id"`
			BridgeURL string `json:"bridge_url"`
			Keys      []struct {
				ID        string `json:"kid"`
				Algorithm string `json:"algorithm"`
				Key       string `json:"public_key"`
			} `json:"access_signing_keys"`
		} `json:"monkeyai_runtime"`
	}
	if err := c.json(ctx, "GET", "/.well-known/oauth-authorization-server", "", "", nil, &metadata); err != nil {
		return "", nil, err
	}
	expectedBridge := "ws" + c.server[4:] + "/api/v1/endpoint-bridge"
	if metadata.Issuer != c.server || !validUUID(metadata.Runtime.ServerID) || metadata.Runtime.BridgeURL != expectedBridge {
		return "", nil, errors.New("invalid_relay_metadata")
	}
	keys := make(map[string]ed25519.PublicKey)
	for _, item := range metadata.Runtime.Keys {
		public, err := base64.RawURLEncoding.DecodeString(item.Key)
		if item.ID == "" || item.Algorithm != "Ed25519" || err != nil || len(public) != ed25519.PublicKeySize || keys[item.ID] != nil {
			return "", nil, errors.New("invalid_relay_signing_keys")
		}
		keys[item.ID] = public
	}
	if len(keys) == 0 {
		return "", nil, errors.New("invalid_relay_signing_keys")
	}
	return metadata.Runtime.ServerID, keys, nil
}
