package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	desktopTerminalProtocol = "monkeyai.terminal.v1"
	maxDesktopTerminals     = 8
	maxTerminalPeers        = 4
	maxTerminalInput        = 16 << 10
	maxTerminalHistory      = 512 << 10
	maxTerminalFrames       = 2048
	maxTerminalViews        = 128
	maxTerminalCounter      = 1<<53 - 1
)

type terminalSize struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

func (s terminalSize) valid() bool {
	return s.Cols >= 2 && s.Cols <= 500 && s.Rows >= 1 && s.Rows <= 300
}

type terminalProcess interface {
	io.Reader
	io.Writer
	Resize(terminalSize) error
	Wait() int
	Stop()
	Shell() string
}

type terminalSlot struct {
	Session string
	Slot    int
}

type terminalHistoryFrame struct {
	Type            string `json:"type"`
	Seq             uint64 `json:"seq"`
	Data            string `json:"data,omitempty"`
	Cols            int    `json:"cols,omitempty"`
	Rows            int    `json:"rows,omitempty"`
	ExitCode        *int   `json:"exitCode,omitempty"`
	Reason          string `json:"reason,omitempty"`
	OutputTruncated bool   `json:"outputTruncated,omitempty"`
}

type terminalController struct {
	ID   string `json:"attachmentId"`
	Name string `json:"name"`
}

type terminalInfo struct {
	Instance   string              `json:"runtimeInstanceId"`
	Session    string              `json:"sessionId"`
	Slot       int                 `json:"slot,omitempty"`
	Revision   uint64              `json:"slotRevision"`
	State      string              `json:"state"`
	Deleting   bool                `json:"deleting"`
	ID         string              `json:"terminalId,omitempty"`
	CWD        string              `json:"cwd,omitempty"`
	Shell      string              `json:"shell,omitempty"`
	Created    string              `json:"createdAt,omitempty"`
	First      uint64              `json:"firstSeq"`
	Last       uint64              `json:"lastSeq"`
	Generation uint64              `json:"controlGeneration"`
	Controller *terminalController `json:"controller,omitempty"`
	Initial    terminalSize        `json:"initialSize"`
	ExitCode   *int                `json:"exitCode,omitempty"`
	Truncated  bool                `json:"outputTruncated,omitempty"`
	terminalSize
}

type desktopTerminal struct {
	info          terminalInfo
	openRevision  uint64
	process       terminalProcess
	done          chan struct{}
	mu            sync.Mutex
	action        sync.Mutex
	frames        []terminalHistoryFrame
	historyWeight int
	views         map[string]terminalView
	peers         map[string]*terminalPeer
}

type terminalView struct {
	generation uint64
	lastAttach time.Time
}

func (t *desktopTerminal) active() bool {
	return t.info.State == "running" || t.info.State == "closing"
}

func (t *desktopTerminal) wakePeers() {
	for _, peer := range t.peers {
		select {
		case peer.wake <- struct{}{}:
		default:
		}
	}
}

func (t *desktopTerminal) appendFrame(frame terminalHistoryFrame) {
	t.info.Last++
	frame.Seq = t.info.Last
	t.frames = append(t.frames, frame)
	t.historyWeight += len(frame.Data)
	for t.historyWeight > maxTerminalHistory || len(t.frames) > maxTerminalFrames {
		t.historyWeight -= len(t.frames[0].Data)
		t.frames[0] = terminalHistoryFrame{}
		t.frames = t.frames[1:]
		t.info.First++
		t.info.Truncated = true
	}
	t.wakePeers()
}

func (t *desktopTerminal) setController(peer *terminalPeer) {
	t.info.Generation++
	t.info.Controller = nil
	if peer != nil {
		t.info.Controller = &terminalController{ID: peer.id, Name: peer.name}
	}
	t.wakePeers()
}

func (t *desktopTerminal) terminate() {
	t.mu.Lock()
	if t.info.State != "running" {
		t.mu.Unlock()
		return
	}
	t.info.State = "closing"
	t.setController(nil)
	t.mu.Unlock()
	t.process.Stop()
}

func (t *desktopTerminal) collect(lifetime context.Context) {
	defer close(t.done)
	stop := context.AfterFunc(lifetime, t.terminate)
	defer stop()
	buffer := make([]byte, 4096)
	for {
		n, err := t.process.Read(buffer)
		if n > 0 {
			t.mu.Lock()
			t.appendFrame(terminalHistoryFrame{Type: "output", Data: base64.StdEncoding.EncodeToString(buffer[:n])})
			t.mu.Unlock()
		}
		if err != nil {
			break
		}
	}
	code := t.process.Wait()
	t.mu.Lock()
	defer t.mu.Unlock()
	reason := "exited"
	if t.info.State == "closing" {
		reason = "closed"
	}
	t.info.State, t.info.ExitCode = reason, &code
	t.setController(nil)
	t.appendFrame(terminalHistoryFrame{Type: "exit", ExitCode: &code, Reason: reason, OutputTruncated: t.info.Truncated})
}

type desktopTerminals struct {
	instance string
	mu       sync.Mutex
	slots    map[terminalSlot]*desktopTerminal
}

func newDesktopTerminals() (*desktopTerminals, error) {
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	return &desktopTerminals{instance: id, slots: make(map[terminalSlot]*desktopTerminal)}, nil
}

func (m *desktopTerminals) capability() map[string]any {
	return map[string]any{"version": 1, "profile": "xterm6-unicode15-v2", "runtimeInstanceId": m.instance,
		"maxTerminals": maxDesktopTerminals, "maxAttachmentsPerTerminal": maxTerminalPeers,
		"maxInputBytes": maxTerminalInput, "multiplePerSession": true}
}

func (m *desktopTerminals) list() []terminalInfo {
	items := make([]terminalInfo, 0, maxDesktopTerminals)
	for _, term := range m.slots {
		term.mu.Lock()
		if term.active() {
			items = append(items, term.info)
		}
		term.mu.Unlock()
	}
	return items
}

func (m *desktopTerminals) closeSession(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for slot, term := range m.slots {
		if slot.Session == id {
			delete(m.slots, slot)
			go term.terminate()
		}
	}
}

func (m *desktopTerminals) shutdown() {
	m.mu.Lock()
	active := make([]*desktopTerminal, 0, maxDesktopTerminals)
	for _, term := range m.slots {
		active = append(active, term)
		go term.terminate()
	}
	m.mu.Unlock()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for _, term := range active {
		select {
		case <-term.done:
		case <-deadline.C:
			return
		}
	}
}

type terminalRequest struct {
	SessionID         string
	RuntimeInstanceID string
	TerminalID        string
	Slot              int
	ExpectedRevision  *uint64
	terminalSize
}

func (h *relaySessionHost) terminalRPC(method string, raw json.RawMessage) (any, error) {
	m := h.terminals
	if method == "_monkeyai/terminal/list" {
		m.mu.Lock()
		items := m.list()
		m.mu.Unlock()
		return map[string]any{"items": items, "maxTerminals": maxDesktopTerminals, "activeCount": len(items), "runtimeInstanceId": m.instance}, nil
	}
	var req terminalRequest
	if json.Unmarshal(raw, &req) != nil || !validUUID(req.SessionID) || req.Slot < 0 || req.Slot >= maxDesktopTerminals {
		return nil, errors.New("invalid_request")
	}
	if method != "_monkeyai/terminal/get" && req.RuntimeInstanceID != m.instance {
		return nil, errors.New("terminal_stale_instance")
	}
	slot := terminalSlot{Session: req.SessionID, Slot: req.Slot}
	m.mu.Lock()
	defer m.mu.Unlock()
	current := m.slots[slot]
	if method == "_monkeyai/terminal/close" {
		// Older clients address the close request by terminal ID without slot.
		for key, term := range m.slots {
			if key.Session == req.SessionID && term.info.ID == req.TerminalID {
				go term.terminate()
				return map[string]any{}, nil
			}
		}
		return nil, errors.New("terminal_conflict")
	}
	info := terminalInfo{Instance: m.instance, Session: req.SessionID, Slot: req.Slot, State: "empty", First: 1}
	if current != nil {
		current.mu.Lock()
		info = current.info
		current.mu.Unlock()
	}
	if method == "_monkeyai/terminal/get" {
		return info, nil
	}
	if req.ExpectedRevision == nil || !validUUID(req.TerminalID) || !req.terminalSize.valid() {
		return nil, errors.New("invalid_terminal_attachment")
	}
	if current != nil && info.ID == req.TerminalID && current.openRevision == *req.ExpectedRevision {
		if info.Initial != req.terminalSize {
			return nil, errors.New("terminal_conflict")
		}
		return info, nil
	}
	if *req.ExpectedRevision != info.Revision || info.State == "running" || info.State == "closing" {
		return nil, errors.New("terminal_conflict")
	}
	if len(m.list()) >= maxDesktopTerminals {
		return nil, errors.New("terminal_limit_reached")
	}
	for _, term := range m.slots {
		if term.info.ID == req.TerminalID {
			return nil, errors.New("terminal_conflict")
		}
	}
	root, _, err := h.workspaceRoot(req.SessionID)
	if err != nil {
		return nil, errors.New("workspace_missing")
	}
	defer root.Close()
	rootInfo, err := root.Stat(".")
	infoAtPath, pathErr := os.Stat(root.Name())
	if err != nil || pathErr != nil || !os.SameFile(rootInfo, infoAtPath) || h.terminalLifetime.Err() != nil {
		return nil, errors.New("workspace_missing")
	}
	process, err := startTerminalProcess(root.Name(), req.terminalSize)
	if err != nil {
		return nil, errors.New("terminal_unavailable")
	}
	info = terminalInfo{Instance: m.instance, Session: req.SessionID, Slot: req.Slot, Revision: *req.ExpectedRevision + 1,
		State: "running", ID: req.TerminalID, CWD: root.Name(), Created: time.Now().UTC().Format(time.RFC3339Nano),
		Shell: process.Shell(), Initial: req.terminalSize, terminalSize: req.terminalSize, First: 1}
	term := &desktopTerminal{info: info, openRevision: *req.ExpectedRevision, process: process, done: make(chan struct{}),
		views: make(map[string]terminalView), peers: make(map[string]*terminalPeer)}
	m.slots[slot] = term
	go term.collect(h.terminalLifetime)
	return info, nil
}

func terminalEnvironment() []string {
	env := make([]string, 0, len(os.Environ()))
	for _, assignment := range os.Environ() {
		name, _, found := strings.Cut(assignment, "=")
		name = strings.ToUpper(name)
		if !found || name == "TERM" || name == "COLORTERM" || name == "NODE_OPTIONS" ||
			strings.HasPrefix(name, "MONKEYAI_") || strings.HasPrefix(name, "AIBERTAI_") || strings.HasPrefix(name, "ELECTRON_") {
			continue
		}
		env = append(env, assignment)
	}
	return append(env, "TERM=xterm-256color", "COLORTERM=truecolor")
}
