package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type acpLive struct {
	dsh             *dshRuntimeConfig
	client          *client
	auth            authorization
	source          *desktopSourceAuth
	key             string
	protocol        string
	dir, cwd        string
	cloudWorkspace  string
	store           *desktopSessionStore
	modelBudget     int
	modelLocked     bool
	workspaces      *desktopWorkspaces
	connectorPrefs  connectorPreferences
	settings        desktopSettings
	mu              sync.Mutex
	operations      map[string]string
	sessions        map[string]acpSession
	activeTurns     map[string]context.CancelFunc
	dshStoppedTurns map[string]string
	queueWorkers    map[string]chan struct{}
	compacting      map[string]bool
	restarting      bool
}

func (l *acpLive) authorization(ctx context.Context) (authorization, error) {
	if l.source != nil {
		return l.source.access(ctx)
	}
	return l.auth, nil
}

type acpSession struct {
	desktopSessionPresentation
	ContextSummary     *desktopContextSummary `json:"contextSummary,omitempty"`
	Goal               *desktopGoal           `json:"goal,omitempty"`
	LastGoalRequest    string                 `json:"lastGoalRequest,omitempty"`
	GoalRun            string                 `json:"goalRun,omitempty"`
	ID                 string                 `json:"sessionId"`
	Engine             string                 `json:"engine,omitempty"`
	DSHSessionID       string                 `json:"dshSessionId,omitempty"`
	AgentTeamEnabled   *bool                  `json:"agentTeamEnabled,omitempty"`
	AgentTeamAvailable *bool                  `json:"agentTeamAvailable,omitempty"`
	CWD                string                 `json:"cwd"`
	Workspace          string                 `json:"workspace,omitempty"`
	// SharedDefault marks a session opened in the shared default workspace
	// (DSH default-workspace). It is still a granted "selected" folder.
	SharedDefault        bool                  `json:"sharedDefaultWorkspace,omitempty"`
	ProjectID            string                `json:"projectId,omitempty"`
	ExpertID             string                `json:"expertId,omitempty"`
	ExpertName           string                `json:"expertName,omitempty"`
	Model                string                `json:"model,omitempty"`
	Mode                 string                `json:"mode,omitempty"`
	MediaImageModelID    string                `json:"mediaImageModelId,omitempty"`
	MediaVideoModelID    string                `json:"mediaVideoModelId,omitempty"`
	Thinking             *modelThinking        `json:"thinking,omitempty"`
	Created              time.Time             `json:"createdAt"`
	Updated              time.Time             `json:"updatedAt"`
	State                string                `json:"state"`
	Operation            string                `json:"operationId,omitempty"`
	Turns                []acpTurn             `json:"turns,omitempty"`
	ConnectorPreferences *connectorPreferences `json:"connectorPreferences,omitempty"`
	ResourceRevision     int                   `json:"resourceRevision,omitempty"`
	Queue                desktopSessionQueue   `json:"queue,omitzero"`
	archivedTurns        int
}

// The verification host keeps new sessions in a separate directory. It never
// changes the legacy runtime's on-disk log or session identifiers.
func (l *acpLive) save(session acpSession) error {
	if l.store != nil {
		previous := l.sessions[session.ID].archivedTurns
		archived, err := saveDesktopSession(l.store.Root, session, previous)
		if err != nil {
			return err
		}
		session.archivedTurns = archived
		l.sessions[session.ID] = session
		if wake := l.queueWorkers[session.ID]; wake != nil {
			select {
			case wake <- struct{}{}:
			default:
			}
		}
		return nil
	}
	file, err := os.CreateTemp(l.dir, "session-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	writeErr := json.NewEncoder(file).Encode(session)
	if err := errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), filepath.Join(l.dir, session.ID+".json")); err != nil {
		return err
	}
	l.sessions[session.ID] = session
	return nil
}

func (l *acpLive) create(operation, cwd string) (string, error) {
	return l.createInWorkspace(operation, cwd, "", modelID, expertSelection{}, nil, "", desktopMediaModels{})
}

func (l *acpLive) createInWorkspace(operation, cwd, projectID, selected string, expert expertSelection, preference *modelThinking, mode string, media desktopMediaModels) (string, error) {
	return l.createSession(operation, cwd, projectID, selected, expert, preference, mode, media, l.dsh != nil)
}

func (l *acpLive) createSession(operation, cwd, projectID, selected string, expert expertSelection, preference *modelThinking, mode string, media desktopMediaModels, useDSH bool) (string, error) {
	return l.createSessionWithTeam(operation, cwd, projectID, selected, expert, preference, mode, media, useDSH, nil, nil)
}

func (l *acpLive) createSessionWithTeam(operation, cwd, projectID, selected string, expert expertSelection, preference *modelThinking, mode string, media desktopMediaModels, useDSH bool, teamEnabled, teamAvailable *bool) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !validUUID(operation) {
		return "", errors.New("invalid_session_parameters")
	}
	kind := ""
	shared := false
	if l.cloudWorkspace != "" && cwd == "" && projectID == "" {
		cwd = l.cloudWorkspace
	}
	if l.workspaces == nil {
		if cwd != "" && cwd != l.cwd || projectID != "" {
			return "", errors.New("invalid_session_parameters")
		}
		cwd = l.cwd
	} else {
		var err error
		cwd, projectID, err = l.workspaces.choose(cwd, projectID)
		if err != nil {
			return "", err
		}
		kind = "selected"
		if cwd == "" {
			kind = "default"
			// DSH opens one shared default workspace. If it cannot be
			// prepared, fall back to a private per-session folder.
			if path, err := l.workspaces.sharedDefault(); err == nil {
				cwd, kind, projectID = path, "selected", l.workspaces.defaultProjectID()
			}
		}
		shared = kind == "selected" && projectID == "" && l.workspaces.shared != "" && cwd == l.workspaces.shared
	}
	if id := l.operations[operation]; id != "" {
		old := l.sessions[id]
		if old.Workspace != kind || old.ProjectID != projectID || kind != "default" && old.CWD != cwd {
			return "", errors.New("session_operation_conflict")
		}
		return id, nil
	}
	if l.store == nil && len(l.sessions) >= 16 {
		return "", errors.New("session_limit_reached")
	}
	id, err := newUUID()
	if err != nil {
		return "", err
	}
	if kind == "default" {
		if err := l.workspaces.allocate(id); err != nil {
			return "", errProjectFolder
		}
		cwd = l.workspaces.defaultPath(id)
	}
	now := time.Now().UTC()
	engine, dshID := "", ""
	if useDSH {
		engine, dshID = "dsh", id
	}
	prefs := l.connectorPrefs
	if preference == nil {
		if fallback, ok := l.settings.ModelThinking[selected]; ok {
			preference = &fallback
		}
	}
	if mode == "" {
		mode = modeWorkspaceWrite
	}
	if err := l.save(acpSession{ID: id, Engine: engine, DSHSessionID: dshID, AgentTeamEnabled: teamEnabled, AgentTeamAvailable: teamAvailable, CWD: cwd, Workspace: kind, SharedDefault: shared, ProjectID: projectID,
		Model: selected, Thinking: preference, Mode: mode, MediaImageModelID: media.Image, MediaVideoModelID: media.Video, ExpertID: expert.ID, ExpertName: expert.Name, Created: now, Updated: now, State: "idle", Operation: operation, ConnectorPreferences: &prefs}); err != nil {
		return "", err
	}
	l.operations[operation] = id
	return id, nil
}

func (l *acpLive) load(id string) (acpSession, error) {
	var session acpSession
	if !validUUID(id) {
		return session, errors.New("session_not_found")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	cached, ok := l.sessions[id]
	if !ok {
		return session, errors.New("session_not_found")
	}
	if l.store != nil {
		return cached, nil
	}
	file, err := os.Open(filepath.Join(l.dir, id+".json"))
	if err != nil {
		return session, err
	}
	defer file.Close()
	if json.NewDecoder(io.LimitReader(file, 8<<20)).Decode(&session) != nil || session.ID != id || session.CWD != l.cwd {
		return acpSession{}, errors.New("invalid_session_snapshot")
	}
	return session, nil
}

func (h *relaySessionHost) currentSessions(cwd string) []map[string]any {
	items := []map[string]any{}
	for _, id := range h.sessionIDs() {
		meta, err := h.listSessionMetadata(id)
		if err != nil {
			continue
		}
		var row struct {
			CWD, Title, UpdatedAt, ProjectID string
			IsDefaultWorkspace               bool
		}
		if json.Unmarshal(meta, &row) == nil && (cwd == "" || cwd == row.CWD) {
			items = append(items, map[string]any{"sessionId": id, "cwd": row.CWD, "title": row.Title, "updatedAt": row.UpdatedAt,
				"_meta": map[string]any{"projectId": row.ProjectID, "isDefaultWorkspace": row.IsDefaultWorkspace}})
		}
	}
	if h.live != nil {
		h.live.mu.Lock()
		defer h.live.mu.Unlock()
		for _, row := range h.live.sessions {
			if cwd == "" || cwd == row.CWD {
				meta := row.workspaceMetadata()
				meta["dshWorkspaceRecoveryAvailable"] = h.live.dsh != nil && h.live.store != nil && dshWorkspaceRecoveryAvailable(row)
				if dshWorkspaceRecoveryAvailable(row) {
					_, _, reviewErr := h.dshManualReviewLocked(row.ID, false)
					meta["dshWorkspaceManualReviewAvailable"] = reviewErr == nil
				}
				meta["dshCanvasRecoveryAvailable"] = h.live.dsh != nil && h.live.store != nil && dshCanvasRecoveryAvailable(row)
				meta["dshPromptRecoveryAvailable"] = h.live.dsh != nil && h.live.store != nil && dshPromptRecoveryAvailable(row)
				items = append(items, map[string]any{"sessionId": row.ID, "cwd": row.CWD, "title": meta["title"], "updatedAt": row.Updated,
					"_meta": meta})
			}
		}
	}
	return items
}

type acpPermissionContextKey struct{}
type acpViewerDoneKey struct{}

type acpPermissionReply struct {
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   json.RawMessage `json:"error"`
}

type acpPermissionWaiter struct {
	sessionID string
	reply     chan acpPermissionReply
}

func (w *acpPermissionWaiter) resolve(reply acpPermissionReply) {
	select {
	case w.reply <- reply:
	default:
	}
}

func (h *relaySessionHost) acp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || r.URL.RawQuery != "" || !slices.Contains(websocketProtocols(r), "monkeyai.v1") {
		http.Error(w, "invalid ACP upgrade", http.StatusBadRequest)
		return
	}
	ctx, cancel, err := h.acpContext(r)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	defer cancel()
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"monkeyai.v1"}, InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer ws.CloseNow()
	ws.SetReadLimit(64 << 10)
	viewer := h.goals != nil && h.local != nil && r.Header.Get("Origin") == h.local.origin
	if viewer {
		defer h.goals.removeViewer(ws)
		if h.interactions != nil {
			defer h.interactions.viewer(ws, false)
		}
	}
	initialized := false
	permissions := new(sync.Map)
	ctx = context.WithValue(ctx, acpPermissionContextKey{}, permissions)
	viewerDone := make(chan struct{})
	defer close(viewerDone)
	ctx = context.WithValue(ctx, acpViewerDoneKey{}, (<-chan struct{})(viewerDone))
	for {
		var in acpPermissionReply
		if wsjson.Read(ctx, ws, &in) != nil {
			return
		}
		if in.JSONRPC != "2.0" || string(in.ID) == "null" {
			continue
		}
		if len(in.ID) == 0 {
			// The browser ACP client sends session/cancel as a notification.
			if initialized && in.Method == "session/cancel" && h.local != nil && h.live != nil && h.acpScope(r, "session:control") == nil {
				var input struct{ SessionID string }
				if json.Unmarshal(in.Params, &input) == nil && validUUID(input.SessionID) {
					h.live.cancelTurn(input.SessionID)
				}
			}
			continue
		}
		if in.Method == "" {
			var permissionID string
			if json.Unmarshal(in.ID, &permissionID) == nil {
				if h.interactions != nil && h.interactions.answer(ws, permissionID, in) {
					continue
				}
				if wait, ok := permissions.LoadAndDelete(permissionID); ok {
					wait.(*acpPermissionWaiter).resolve(in)
				}
			}
			if initialized && viewer {
				h.goals.respond(ws, in.ID, in.Result, in.Error)
			}
			continue
		}
		if h.trace && len(in.Method) <= 128 {
			fmt.Fprintf(os.Stderr, "staff preview ACP method %q\n", in.Method)
		}
		var result any
		var fault error
		code := -32602
		if !initialized {
			var p struct{ ProtocolVersion int }
			if in.Method != "initialize" || json.Unmarshal(in.Params, &p) != nil || p.ProtocolVersion != 1 {
				fault = errors.New("initialize_required")
			} else {
				initialized = true
				meta := map[string]any{"sessionReads": map[string]any{"version": 1, "maxBytes": 5 << 20, "queue": true, "state": 1, "pageFragments": 1}}
				if h.live != nil && (h.local != nil) && h.acpScope(r, "session:control") == nil {
					meta["goalMode"] = map[string]int{"version": 1}
					if h.live.dsh != nil {
						meta["agentTeam"] = 1
						meta["standardPreset"] = 1
					}
				}
				if h.terminals != nil && h.acpScope(r, "session:control") == nil {
					meta["terminal"] = h.terminals.capability()
				}
				if h.local == nil && h.live != nil && h.acpScope(r, "session:control") == nil {
					meta["sessionCancel"] = 1
				}
				if h.uploads != nil {
					meta["uploads"] = map[string]int{"version": 1, "maxFileBytes": relayMaxUpload, "maxChunkBytes": relayUploadChunk, "slotTtlSeconds": int(relayUploadTTL / time.Second)}
					meta["fileLinks"] = map[string]any{"version": 1, "scope": "file", "purposes": []string{"resource", "download"}}
					meta["operations"] = map[string]any{"version": 1, "methods": []string{"_monkeyai/uploads/complete", "session/new", "_monkeyai/session/queue"}, "queueActions": []string{"add"}}
					meta["configReads"] = map[string]int{"version": 1, "maxBytes": 1 << 20}
				}
				result = map[string]any{"protocolVersion": 1, "agentInfo": map[string]string{"name": "aibertai-native", "version": version},
					"agentCapabilities": map[string]any{"loadSession": true, "sessionCapabilities": map[string]any{"list": map[string]any{}},
						"_meta": meta}}
			}
		} else {
			var p struct {
				CWD, Cursor, SessionID string
				MCPServers             []json.RawMessage
				AdditionalDirectories  []string
				Prompt                 []struct{ Type, Text string }
				Meta                   struct {
					OperationID, ProjectID, Model, Profile, Mode string
					MediaModels                                  desktopMediaModels
					Thinking                                     *modelThinking
					AgentTeamEnabled                             *bool
				} `json:"_meta"`
			}
			if len(in.Params) != 0 && json.Unmarshal(in.Params, &p) != nil {
				fault = errors.New("invalid_session_parameters")
			} else if len(p.MCPServers) != 0 || len(p.AdditionalDirectories) != 0 {
				fault = errors.New("workspace_scope_fixed")
			} else {
				switch in.Method {
				case "_monkeyai/terminal/get", "_monkeyai/terminal/list", "_monkeyai/terminal/open", "_monkeyai/terminal/close":
					if h.terminals == nil {
						code, fault = -32601, errors.New("terminal_unavailable")
					} else if err := h.acpScope(r, "session:control"); err != nil {
						code, fault = -32001, errRelayAccess
					} else {
						result, fault = h.terminalRPC(in.Method, in.Params)
					}
				case "session/list":
					if p.Cursor != "" {
						fault = errors.New("invalid_session_cursor")
					} else {
						result = map[string]any{"sessions": h.currentSessions(p.CWD)}
					}
				case "session/load":
					result, fault = h.loadACP(ctx, ws, p.SessionID, p.CWD)
				case "session/delete", "_monkeyai/sessions/delete":
					if h.local == nil || h.live == nil || h.live.store == nil || h.acpScope(r, "session:control") != nil {
						code, fault = -32001, errRelayAccess
					} else {
						result, fault = h.deleteDesktopSessions(ctx, ws, in.Method, in.Params)
					}
				case "_aibertai/session/jobs", "_aibertai/session/members", "_aibertai/session/member-interrupt":
					if h.local == nil || h.live == nil || h.live.dsh == nil || h.acpScope(r, "session:control") != nil {
						code, fault = -32001, errRelayAccess
					} else if in.Method == "_aibertai/session/member-interrupt" {
						result, fault = h.memberInterrupt(ctx, in.Params)
					} else if in.Method == "_aibertai/session/members" {
						result, fault = h.sessionMembers(ctx, in.Params)
					} else {
						result, fault = h.sessionJobs(ctx, in.Params)
					}
				case "_aibertai/session/agent-team":
					if h.local == nil || h.live == nil || h.live.dsh == nil || h.acpScope(r, "session:control") != nil {
						code, fault = -32001, errRelayAccess
					} else {
						result, fault = h.sessionAgentTeam(ctx, in.Params)
					}
				case "_aibertai/session/review-workspace":
					if h.live == nil || h.acpScope(r, "session:control") != nil {
						code, fault = -32001, errRelayAccess
					} else {
						var input struct {
							SessionID   string `json:"sessionId"`
							Action      string `json:"action"`
							Fingerprint string `json:"fingerprint"`
							Decision    string `json:"decision"`
						}
						if json.Unmarshal(in.Params, &input) != nil || input.SessionID != p.SessionID {
							fault = errors.New("invalid_session_parameters")
						} else if input.Action == "inspect" {
							result, fault = h.inspectDSHManualReview(input.SessionID)
						} else if input.Action == "confirm" {
							go func(id json.RawMessage, input struct {
								SessionID   string `json:"sessionId"`
								Action      string `json:"action"`
								Fingerprint string `json:"fingerprint"`
								Decision    string `json:"decision"`
							}) {
								err := h.confirmDSHManualReview(ctx, input.SessionID, input.Fingerprint, input.Decision)
								reply := map[string]any{"jsonrpc": "2.0", "id": id}
								if err != nil {
									reply["error"] = map[string]any{"code": -32602, "message": err.Error()}
								} else {
									if row, loadErr := h.live.load(input.SessionID); loadErr == nil && len(row.Turns) > 0 {
										_ = notifyACPReply(ctx, ws, row, len(row.Turns)-1, true)
										h.notifyDSHHistory(ctx, row)
									}
									reply["result"] = map[string]any{"reviewed": true}
								}
								_ = wsjson.Write(ctx, ws, reply)
							}(in.ID, input)
							continue
						} else {
							fault = errors.New("invalid_session_parameters")
						}
					}
				case "_aibertai/session/reconcile-workspace", "_aibertai/session/reconcile-prompt", "_aibertai/session/reconcile-canvas":
					if h.live == nil || h.acpScope(r, "session:control") != nil {
						code, fault = -32001, errRelayAccess
					} else {
						go func(id json.RawMessage, sessionID, method string) {
							var err error
							if method == "_aibertai/session/reconcile-workspace" {
								err = h.recoverDSHWorkspaceSession(ctx, sessionID)
							} else if method == "_aibertai/session/reconcile-canvas" {
								err = h.recoverDSHCanvasSession(ctx, sessionID)
							} else {
								err = h.recoverDSHPromptSession(ctx, sessionID, false)
							}
							reply := map[string]any{"jsonrpc": "2.0", "id": id}
							if err != nil {
								reply["error"] = map[string]any{"code": -32602, "message": err.Error()}
							} else {
								if row, loadErr := h.live.load(sessionID); loadErr == nil && len(row.Turns) > 0 {
									_ = notifyACPReply(ctx, ws, row, len(row.Turns)-1, true)
									_ = notifyACPTurnState(ctx, ws, row, row.Turns[len(row.Turns)-1].Operation, true)
									if row.Turns[len(row.Turns)-1].DSHModelFailureSettled {
										// Receipt verification can finish after its requesting window closes.
										push, cancel := context.WithTimeout(context.WithoutCancel(ctx), 4*time.Second)
										h.notifyDSHTurnCheckpoint(push, row, len(row.Turns)-1, row.Turns[len(row.Turns)-1].Operation)
										cancel()
									} else {
										h.notifyDSHHistory(ctx, row)
									}
								}
								reply["result"] = map[string]any{"reconciled": true}
							}
							_ = wsjson.Write(ctx, ws, reply)
						}(in.ID, p.SessionID, in.Method)
						continue
					}
				case "_monkeyai/session/queue", "_aibertai/session/member-prompt":
					// A queued turn may run for minutes. Leave the ACP reader free for
					// history, cancellation and other control requests on this connection.
					go func(method string, id, params json.RawMessage) {
						replied := false
						accept := func(result any) error {
							replied = true
							return wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
						}
						handle := h.handleQueue
						if method == "_aibertai/session/member-prompt" {
							handle = h.memberPrompt
						}
						result, err := handle(ctx, ws, r, params, accept)
						if replied {
							return
						}
						if err == nil {
							_ = accept(result)
							return
						}
						_ = wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": id,
							"error": map[string]any{"code": -32602, "message": err.Error()}})
					}(in.Method, in.ID, in.Params)
					continue
				case "_aibertai/runtime/prepare-restart":
					if h.local == nil || h.live == nil || len(r.Header.Values("Origin")) != 0 {
						code, fault = -32001, errRelayAccess
					} else {
						result, fault = h.live.prepareDesktopRestart()
					}
				case "_monkeyai/session/outline", "_monkeyai/session/compact":
					scope := "session:read"
					if in.Method != "_monkeyai/session/outline" {
						scope = "session:control"
					}
					cloud := h.live != nil && h.live.cloudWorkspace != ""
					if h.live == nil || h.local == nil && !cloud || h.acpScope(r, scope) != nil {
						code, fault = -32001, errRelayAccess
					} else {
						switch in.Method {
						case "_monkeyai/session/outline":
							result, fault = h.desktopOutline(p.SessionID)
						case "_monkeyai/session/compact":
							row, loadErr := h.live.load(p.SessionID)
							if loadErr != nil {
								fault = loadErr
								break
							}
							if row.Engine != "dsh" {
								result, fault = h.compactDesktopSession(ctx, p.SessionID)
								break
							}
							if h.local == nil || !validUUID(p.Meta.OperationID) || h.acpScope(r, "files:read") != nil {
								fault = errors.New("dsh_compaction_entry_unavailable")
								break
							}
							go func(requestID json.RawMessage, id, operation string) {
								runCtx, release := h.dshPromptContext(ctx, id)
								defer release()
								value, err := h.promptACP(runCtx, ws, r, id, operation, "", "/compact", nil, nil)
								reply := map[string]any{"jsonrpc": "2.0", "id": requestID}
								if err != nil {
									reply["error"] = map[string]any{"code": -32602, "message": err.Error()}
								} else if result, ok := value.(map[string]string); ok && result["stopReason"] == "cancelled" {
									reply["result"] = map[string]any{"status": "cancelled"}
								} else if current, loadErr := h.live.load(id); loadErr == nil && len(current.Turns) > 0 && current.Turns[len(current.Turns)-1].Operation == operation && current.Turns[len(current.Turns)-1].DSHCompactID != "" {
									turn := current.Turns[len(current.Turns)-1]
									skipped := turn.Answer == "No compactable history yet."
									reply["result"] = map[string]any{"status": "completed", "skipped": skipped, "text": turn.Answer, "usage": turn.Usage}
								} else {
									reply["error"] = map[string]any{"code": -32602, "message": "dsh_compaction_unverified"}
								}
								_ = wsjson.Write(ctx, ws, reply)
							}(in.ID, p.SessionID, p.Meta.OperationID)
							continue
						}
					}
				case "_aibertai/connectors/local/test", "_aibertai/connectors/local/save":
					if h.local == nil || h.acpScope(r, "resources:manage") != nil {
						code, fault = -32001, errRelayAccess
					} else if in.Method == "_aibertai/connectors/local/save" {
						result, fault = h.saveLocalConnector(ctx, in.Params)
					} else {
						result, fault = h.testSavedLocalConnector(ctx, in.Params)
					}
				case "session/cancel":
					if h.live == nil || h.acpScope(r, "session:control") != nil {
						code, fault = -32001, errRelayAccess
					} else if !validUUID(p.SessionID) {
						fault = errors.New("invalid_session_parameters")
					} else {
						result = map[string]bool{"stopped": h.live.cancelTurn(p.SessionID)}
					}
				case "_monkeyai/session/state/get", "_monkeyai/session/subscribe", "_monkeyai/session/generate-title",
					"_monkeyai/session/task", "_monkeyai/session/interaction/get", "_monkeyai/session/interaction/answer":
					scope := "session:read"
					if in.Method == "_monkeyai/session/interaction/answer" {
						scope = "session:control"
					}
					if h.live == nil || h.acpScope(r, scope) != nil {
						code, fault = -32001, errRelayAccess
					} else {
						result, fault = h.sessionDetail(r, in.Method, in.Params)
					}
				case "_aibertai/workspace/authorize":
					if h.local == nil || h.live == nil || h.live.workspaces == nil || len(r.Header.Values("Origin")) != 0 {
						code, fault = -32601, errors.New("method_not_found")
					} else {
						result, fault = h.live.workspaces.authorizeDirectory(in.Params)
					}
				case "_aibertai/session/attachment/read":
					if h.local == nil || len(r.Header.Values("Origin")) != 0 {
						code, fault = -32601, errors.New("method_not_found")
					} else {
						result, fault = h.savedDesktopAttachment(in.Params)
					}
				case "_aibertai/design/media-model":
					if h.local == nil || h.live == nil || len(r.Header.Values("Origin")) != 0 || h.acpScope(r, "session:control") != nil {
						fault = errRelayAccess
					} else {
						result, fault = h.desktopMediaGateway(ctx, in.Params)
					}
				case "_monkeyai/devices/list", "_monkeyai/devices/connect",
					"_monkeyai/device/remote/status", "_monkeyai/device/remote/set", "_monkeyai/device/remote/power":
					if h.local == nil || len(r.Header.Values("Origin")) != 0 {
						code, fault = -32601, errors.New("method_not_found")
					} else {
						result, fault = h.localQuery(ctx, in.Method, in.Params)
					}
				case "_monkeyai/config/models/get", "_monkeyai/config/models/save", "_monkeyai/config/models/delete", "_monkeyai/config/models/create-many", "_monkeyai/config/models/test", "_monkeyai/config/models/discover",
					"_monkeyai/resources/get", "_monkeyai/resources/save", "_monkeyai/resources/delete",
					"_monkeyai/resources/experts/details", "_monkeyai/resources/experts/tags", "_monkeyai/resources/connectors/tags", "_monkeyai/resources/skills/tags",
					"_monkeyai/resources/skills/content", "_monkeyai/resources/skills/manifest",
					"_monkeyai/resources/connectors/test", "_monkeyai/resources/connectors/authorize",
					"_monkeyai/resources/connectors/authorization", "_monkeyai/resources/connectors/credentials",
					"_monkeyai/resources/connectors/credentials/get", "_monkeyai/resources/connectors/credentials/create",
					"_monkeyai/resources/connectors/credentials/update", "_monkeyai/resources/connectors/credentials/revoke":
					if h.local == nil {
						code, fault = -32601, errors.New("method_not_found")
					} else {
						result, fault = h.localQuery(ctx, in.Method, in.Params)
					}
				case "_monkeyai/config/get", "_monkeyai/config/sync", "_monkeyai/resources/preferences/set", "_monkeyai/session/messages",
					"_monkeyai/session/tool_output", "_monkeyai/runtime/status",
					"_monkeyai/preferences/model-thinking/set", "_monkeyai/resources/recipients", "_monkeyai/resources/share", "_monkeyai/users/search", "_monkeyai/groups/search",
					"_monkeyai/drafts", "_monkeyai/draft/set",
					"_monkeyai/environments/list", "_monkeyai/environments/get", "_monkeyai/environments/operations", "_monkeyai/environments/change",
					"_monkeyai/config/goals/set", "_monkeyai/config/display/set", "_monkeyai/billing/account", "_monkeyai/billing/entries":
					if h.local == nil {
						code, fault = -32601, errors.New("method_not_found")
					} else {
						result, fault = h.localQuery(ctx, in.Method, in.Params)
					}
				case "_monkeyai/session/expert/set":
					if h.live == nil || h.live.store == nil {
						code, fault = -32601, errors.New("method_not_found")
					} else if err := h.acpScope(r, "session:control"); err != nil {
						code, fault = -32001, errRelayAccess
					} else {
						result, fault = h.setDesktopSessionExpert(ctx, in.Params)
					}
				case "session/set_mode":
					result, fault = h.setACPPermissionMode(r, in.Params)
				case "_monkeyai/session/goal":
					if h.local == nil || h.live == nil || h.acpScope(r, "session:control") != nil {
						fault = errRelayAccess
					} else {
						result, fault = h.live.saveGoalAction(in.Params, h.goals != nil)
						if fault == nil && h.goals != nil {
							h.goals.schedule()
						}
					}
				case "_monkeyai/session/rename", "_monkeyai/session/flags":
					if h.local == nil || h.live == nil || h.acpScope(r, "session:control") != nil {
						fault = errRelayAccess
					} else {
						result, fault = h.updateDesktopSessionPresentation(in.Method, in.Params)
					}
				case "_monkeyai/session/media-models/set":
					if h.local == nil || h.acpScope(r, "session:control") != nil {
						code, fault = -32001, errRelayAccess
					} else {
						result, fault = h.setDesktopSessionMedia(ctx, in.Params)
					}
				case "_monkeyai/session/model/set", "_monkeyai/session/thinking/set":
					if h.live == nil || h.local == nil && h.live.cloudWorkspace == "" {
						code, fault = -32601, errors.New("method_not_found")
					} else if err := h.acpScope(r, "session:control"); err != nil {
						code, fault = -32001, errRelayAccess
					} else {
						result, fault = h.setDesktopSessionModel(ctx, in.Method, in.Params)
					}
				case "_monkeyai/session/resources", "_monkeyai/session/resources/set":
					if h.live == nil || h.live.workspaces == nil {
						code, fault = -32601, errors.New("method_not_found")
					} else if err := h.acpScope(r, "resources:manage"); err != nil {
						code, fault = -32001, errRelayAccess
					} else {
						result, fault = h.sessionConnectorResources(in.Params, in.Method == "_monkeyai/session/resources/set")
					}
				case "_monkeyai/projects/list", "_monkeyai/projects/create", "_monkeyai/projects/update", "_monkeyai/projects/move", "_monkeyai/projects/remove":
					result = []any{}
					if h.live != nil && h.live.workspaces != nil {
						if in.Method != "_monkeyai/projects/list" && h.local == nil {
							code, fault = -32001, errRelayAccess
						} else {
							result, fault = h.live.workspaces.projects(in.Method, in.Params)
						}
					}
				case "_monkeyai/workspace/list", "_monkeyai/workspace/read", "_monkeyai/workspace/stat", "_monkeyai/workspace/resolve", "_monkeyai/workspace/changes", "_monkeyai/workspace/diff", "_monkeyai/session/workspace":
					if h.live == nil || h.live.workspaces == nil {
						code, fault = -32601, errors.New("method_not_found")
					} else if err := h.acpScope(r, "files:read"); err != nil {
						code, fault = -32001, errRelayAccess
					} else {
						result, fault = h.workspaceQuery(ctx, in.Method, in.Params)
					}
				case "_monkeyai/session/files/preview":
					if h.live == nil || h.live.workspaces == nil || h.acpScope(r, "files:read") != nil {
						code, fault = -32001, errRelayAccess
					} else {
						result, fault = h.sessionFilePreview(ctx, in.Params)
					}
				case "_monkeyai/workspace/link":
					if h.local != nil && h.live != nil && h.live.workspaces != nil {
						result, fault = h.workspaceQuery(ctx, in.Method, in.Params)
					} else if h.uploads != nil {
						result, fault = h.acpFile(r, in.Method, in.Params)
					} else {
						code, fault = -32601, errors.New("file_transfer_unavailable")
					}
				case "_monkeyai/uploads/create", "_monkeyai/uploads/status", "_monkeyai/uploads/cancel", "_monkeyai/uploads/complete", "_monkeyai/files/link", "_monkeyai/operations/get":
					if h.uploads == nil {
						code, fault = -32601, errors.New("file_transfer_unavailable")
					} else {
						result, fault = h.acpFile(r, in.Method, in.Params)
						if errors.Is(fault, errUploadOperationMissing) {
							code = -32002
						}
					}
				case "session/new", "session/prompt":
					if h.live == nil {
						code, fault = -32601, errors.New("live_session_unavailable")
					} else if err := h.acpScope(r, "session:control"); err != nil {
						code, fault = -32001, errRelayAccess
					} else if in.Method == "session/new" {
						var id string
						if h.local != nil {
							if p.Meta.Mode != "" && !validACPMode(p.Meta.Mode) {
								fault = errors.New("invalid_session_mode")
							} else {
								fault = h.validateDesktopMedia(ctx, p.Meta.MediaModels)
							}
						}
						if p.Meta.Thinking != nil && !p.Meta.Thinking.valid() {
							fault = errors.New("invalid_model_thinking")
						}
						if fault == nil && h.local != nil && p.Meta.OperationID == "" {
							p.Meta.OperationID, fault = newUUID()
						}
						if fault == nil {
							selected := modelID
							if (h.local != nil || h.live.cloudWorkspace != "") && p.Meta.Model != "" {
								selected, fault = h.desktopSelectedModel(ctx, p.Meta.Model)
							}
							if fault == nil {
								var expert desktopExpert
								if p.Meta.Profile != "" {
									expert, fault = h.resolveDesktopExpert(ctx, p.Meta.Profile)
								}
								if fault == nil {
									dsh := h.local != nil && h.live.dsh != nil
									available := dsh && expert.allowsAgentTeam()
									if p.Meta.AgentTeamEnabled != nil && *p.Meta.AgentTeamEnabled && !available {
										fault = errors.New("agent_team_unavailable")
									} else {
										id, fault = h.live.createSessionWithTeam(p.Meta.OperationID, p.CWD, p.Meta.ProjectID, selected, expert.expertSelection, p.Meta.Thinking, p.Meta.Mode, p.Meta.MediaModels, dsh, p.Meta.AgentTeamEnabled, &available)
									}
								}
							}
						}
						if fault == nil {
							var row acpSession
							row, fault = h.live.load(id)
							result = map[string]any{"sessionId": id, "_meta": row.workspaceMetadata(), "modes": map[string]string{"currentModeId": row.permissionMode()}}
						}
					} else if len(p.Prompt) != 1 || p.Prompt[0].Type != "text" || strings.TrimSpace(p.Prompt[0].Text) == "" || len(p.Prompt[0].Text) > 16<<10 {
						fault = errors.New("invalid_prompt")
					} else {
						if err := h.acpScope(r, "files:read"); err != nil {
							code, fault = -32001, errRelayAccess
						} else {
							go func(requestID json.RawMessage, id, operation, prompt string) {
								runCtx, release := h.dshPromptContext(ctx, id)
								defer release()
								value, err := h.promptACP(runCtx, ws, r, id, operation, "", prompt, nil, nil)
								reply := map[string]any{"jsonrpc": "2.0", "id": requestID}
								if err != nil {
									reply["error"] = map[string]any{"code": -32602, "message": err.Error()}
								} else {
									reply["result"] = value
								}
								if wsjson.Write(ctx, ws, reply) != nil {
									row, loadErr := h.live.load(id)
									if loadErr != nil || row.Engine != "dsh" || runCtx.Err() != nil {
										return
									}
								}
								h.live.mu.Lock()
								start := h.live.claimQueue(h.live.sessions[id])
								h.live.mu.Unlock()
								if start {
									_, _ = h.runSessionQueue(ctx, ws, r, id, nil, nil)
								}
							}(in.ID, p.SessionID, p.Meta.OperationID, p.Prompt[0].Text)
							continue
						}
					}
				default:
					if h.local != nil && (strings.HasPrefix(in.Method, "_monkeyai/kb/") || in.Method == "_monkeyai/resources/connectors/tools") {
						var input desktopKnowledgeInput
						input.Method = in.Method
						if json.Unmarshal(in.Params, &input.Params) != nil {
							fault = errors.New("invalid_request")
						} else if auth, authErr := h.live.authorization(ctx); authErr != nil {
							fault = errDesktopCredential
						} else {
							result, fault = h.live.client.employeeKnowledge(ctx, auth, input)
						}
					} else {
						code, fault = -32601, errors.New("method_not_found")
					}
				}
			}
		}
		reply := map[string]any{"jsonrpc": "2.0", "id": in.ID}
		if fault != nil {
			reply["error"] = map[string]any{"code": code, "message": fault.Error()}
		} else {
			reply["result"] = result
		}
		if wsjson.Write(ctx, ws, reply) != nil {
			return
		}
		if viewer && initialized && in.Method == "initialize" {
			h.goals.addViewer(ws)
			if h.interactions != nil {
				h.interactions.viewer(ws, true)
			}
		}
	}
}

func (row acpSession) workspaceMetadata() map[string]any {
	title, source := row.sessionTitle()
	metadata := map[string]any{"cwd": row.CWD, "model": row.Model, "thinking": row.Thinking, "mode": row.permissionMode(), "mediaImageModelId": row.MediaImageModelID, "mediaVideoModelId": row.MediaVideoModelID, "expertId": row.ExpertID, "configurationRevision": row.ResourceRevision, "isDefaultWorkspace": row.Workspace != "selected" || row.SharedDefault, "projectId": row.ProjectID, "status": row.State, "goalModeAvailable": row.Workspace != "", "goal": goalView(row),
		"title": title, "titleSource": source, "pinned": row.Pinned, "archivedAt": row.ArchivedAt, "readAt": row.ReadAt}
	if row.Engine == "dsh" {
		row.addDSHTodoMetadata(metadata)
		if n := len(row.Turns); n > 0 {
			last := row.Turns[n-1]
			if last.DSHStopObserved && last.Failed && !last.DSHReconciled {
				metadata["dshStopAccountingPending"] = true
			}
		}
		row.addAgentTeamMetadata(metadata)
		for i := len(row.Turns) - 1; i >= 0; i-- {
			turn := row.Turns[i]
			if turn.Model != row.Model {
				break
			}
			if turn.DSHContext.valid() {
				metadata["context"] = turn.DSHContext
				break
			}
		}
	}
	if reason := row.lastTurnFailure(); reason != "" {
		metadata["lastError"] = reason
	}
	return metadata
}

func (h *relaySessionHost) acpFile(r *http.Request, method string, raw json.RawMessage) (any, error) {
	scope := "files:write"
	if method == "_monkeyai/files/link" || method == "_monkeyai/workspace/link" {
		scope = "files:read"
	}
	access, err := h.access.authorizeAccess(r, scope)
	if err != nil {
		return nil, errRelayAccess
	}
	var params struct {
		relayUploadParams
		Path        string                       `json:"path"`
		OperationID string                       `json:"operationId"`
		Ref         relayFileRef                 `json:"fileRef"`
		Meta        struct{ OperationID string } `json:"_meta"`
	}
	if json.Unmarshal(raw, &params) != nil {
		return nil, errInvalidUpload
	}
	if method == "_monkeyai/files/link" {
		return h.uploads.link(h.access.issuer, params.Purpose, params.Ref, access)
	}
	if method == "_monkeyai/workspace/link" {
		auth := access.Auth
		return h.uploads.link(h.access.issuer, params.Purpose, relayFileRef{ServerID: auth.ServerID, UserID: auth.OwnerUserID,
			MachineID: auth.MachineID, SessionID: params.SessionID, Path: params.Path}, access)
	}
	if method == "_monkeyai/operations/get" {
		if result, err := h.uploads.query(params.OperationID, access.Auth); err == nil {
			return result, nil
		}
		return h.sessionOperation(params.OperationID, access.Auth)
	}
	if method == "_monkeyai/uploads/create" {
		if _, err := h.live.load(params.SessionID); err != nil {
			return nil, errInvalidUpload
		}
		return h.uploads.create(params.relayUploadParams, access.Auth)
	}
	return h.uploads.operation(method, params.UploadID, params.SessionID, params.Meta.OperationID, access.Auth)
}

// Desktop DSH work follows the host lifetime rather than one viewer connection.
// The existing active-turn cancel function still stops the individual session.
func (h *relaySessionHost) dshPromptContext(ctx context.Context, id string) (context.Context, func()) {
	if h.local == nil || h.live == nil || h.terminalLifetime == nil {
		return ctx, func() {}
	}
	row, err := h.live.load(id)
	if err != nil || row.Engine != "dsh" {
		return ctx, func() {}
	}
	detached, cancel := context.WithCancel(context.WithoutCancel(ctx))
	stop := context.AfterFunc(h.terminalLifetime, cancel)
	return detached, func() { stop(); cancel() }
}

func websocketProtocols(r *http.Request) []string {
	var offered []string
	for _, line := range r.Header.Values("Sec-WebSocket-Protocol") {
		for _, entry := range strings.Split(line, ",") {
			offered = append(offered, strings.TrimSpace(entry))
		}
	}
	return offered
}

func sendACPText(ctx context.Context, ws *websocket.Conn, id, update, text string) error {
	return wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "method": "session/update", "params": map[string]any{
		"sessionId": id, "update": map[string]any{"sessionUpdate": update, "content": map[string]string{"type": "text", "text": text}},
	}})
}

func (h *relaySessionHost) loadACP(ctx context.Context, ws *websocket.Conn, id, cwd string) (any, error) {
	if id == h.sessionID || h.local != nil || h.shareDesktopSessions {
		archive, err := readLegacySessionArchive(h.profile, id)
		var meta struct{ CWD string }
		if err == nil && json.Unmarshal(archive.Session, &meta) == nil && (meta.CWD == cwd || h.local != nil && cwd == "") {
			// The client's follow-up history request receives messages, attachments,
			// and tool results as a single coherent snapshot.
			return map[string]any{}, nil
		}
	}
	if id != h.sessionID && h.live != nil {
		session, err := h.live.load(id)
		if err == nil && (session.CWD == cwd || h.local != nil && cwd == "") {
			for _, turn := range session.Turns {
				if sendACPText(ctx, ws, id, "user_message_chunk", turn.Prompt) != nil ||
					turn.Answer != "" && sendACPText(ctx, ws, id, "agent_message_chunk", turn.Answer) != nil {
					return nil, errors.New("session_replay_failed")
				}
			}
			return map[string]any{}, nil
		}
	}
	return nil, errors.New("session_not_found")
}
