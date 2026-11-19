package main

import (
	"context"
	"encoding/base64"
	"net/http"
	"slices"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type terminalPeer struct {
	id, name, view         string
	cancel                 context.CancelFunc
	wake                   chan struct{}
	responses              chan map[string]any
	sent, ack, lastContact atomic.Uint64
	input                  uint64
}

type terminalAttach struct {
	Type, SessionID, RuntimeInstanceID, TerminalID, ViewID, Mode string
	ViewGeneration, AfterSeq                                     uint64
}

type terminalAttached struct {
	Type           string `json:"type"`
	AttachmentID   string `json:"attachmentId"`
	ViewID         string `json:"viewId"`
	ViewGeneration uint64 `json:"viewGeneration"`
	ReplayTo       uint64 `json:"replayTo"`
	terminalInfo
}

func terminalSocketError(kind, code string) map[string]any {
	return map[string]any{"type": "error", "requestType": kind, "code": code}
}

func sendTerminalFrame(ctx context.Context, ws *websocket.Conn, value any) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return wsjson.Write(ctx, ws, value)
}

func (h *relaySessionHost) serveTerminal(w http.ResponseWriter, r *http.Request) {
	if h.terminals == nil || r.Method != http.MethodGet || r.URL.RawQuery != "" && r.URL.RawQuery != "reattach=1" {
		http.NotFound(w, r)
		return
	}
	var ctx context.Context
	var cancel context.CancelFunc
	name := "本机"
	if h.local != nil {
		if h.local.authorizeProtocol(r, desktopTerminalProtocol) != nil {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		ctx, cancel = context.WithCancel(r.Context())
	} else {
		if !slices.Equal(websocketProtocols(r), []string{desktopTerminalProtocol}) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var err error
		ctx, cancel, err = h.access.leaseContext(r, "session:control")
		if err != nil {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		name = "远程设备"
	}
	defer cancel()
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{desktopTerminalProtocol}, InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	defer ws.CloseNow()
	ws.SetReadLimit(32 << 10)
	handshake, stop := context.WithTimeout(ctx, 8*time.Second)
	var request terminalAttach
	err = wsjson.Read(handshake, ws, &request)
	stop()
	if err != nil {
		return
	}
	m := h.terminals
	if request.Type != "attach" || !validUUID(request.SessionID) || !validUUID(request.TerminalID) ||
		!validUUID(request.ViewID) || request.ViewGeneration < 1 || request.ViewGeneration > maxTerminalCounter ||
		request.AfterSeq > maxTerminalCounter || request.Mode != "control-if-free" && request.Mode != "observe" {
		_ = sendTerminalFrame(ctx, ws, terminalSocketError("attach", "invalid_terminal_attachment"))
		return
	}
	if request.RuntimeInstanceID != m.instance {
		_ = sendTerminalFrame(ctx, ws, terminalSocketError("attach", "terminal_stale_instance"))
		return
	}
	m.mu.Lock()
	var term *desktopTerminal
	for slot, found := range m.slots {
		if slot.Session == request.SessionID && found.info.ID == request.TerminalID {
			term = found
			break
		}
	}
	m.mu.Unlock()
	if term == nil {
		_ = sendTerminalFrame(ctx, ws, terminalSocketError("attach", "terminal_conflict"))
		return
	}
	id, err := newUUID()
	if err != nil {
		return
	}
	peer := &terminalPeer{id: id, name: name, view: request.ViewID, cancel: cancel,
		wake: make(chan struct{}, 1), responses: make(chan map[string]any, 32)}
	peer.sent.Store(request.AfterSeq)
	peer.ack.Store(request.AfterSeq)
	peer.lastContact.Store(uint64(time.Now().UnixNano()))
	attached, reason := term.attach(peer, request)
	if reason != "" {
		frame := terminalSocketError("attach", reason)
		if reason == "terminal_conflict" {
			frame["reason"] = "view_expired"
		}
		_ = sendTerminalFrame(ctx, ws, frame)
		return
	}
	defer term.detach(peer)
	if err := sendTerminalFrame(ctx, ws, attached); err != nil {
		return
	}
	go term.readTerminalPeer(ctx, ws, peer)
	term.writeTerminalPeer(ctx, ws, peer, attached)
}

func (t *desktopTerminal) attach(peer *terminalPeer, req terminalAttach) (terminalAttached, string) {
	t.action.Lock()
	defer t.action.Unlock()
	t.mu.Lock()
	defer t.mu.Unlock()
	if req.AfterSeq+1 < t.info.First {
		return terminalAttached{}, "terminal_history_lost"
	}
	if req.AfterSeq > t.info.Last {
		return terminalAttached{}, "invalid_terminal_sequence"
	}
	previous, known := t.views[req.ViewID]
	if known && previous.generation >= req.ViewGeneration {
		return terminalAttached{}, "terminal_conflict"
	}
	for id, old := range t.peers {
		if old.view == req.ViewID {
			delete(t.peers, id)
			old.cancel()
			if t.info.Controller != nil && t.info.Controller.ID == id {
				t.setController(nil)
			}
		}
	}
	if len(t.peers) >= maxTerminalPeers {
		return terminalAttached{}, "terminal_limit_reached"
	}
	if !known && len(t.views) >= maxTerminalViews {
		t.forgetOldView()
	}
	t.views[req.ViewID], t.peers[peer.id] = terminalView{generation: req.ViewGeneration, lastAttach: time.Now()}, peer
	if t.info.State == "running" && req.Mode == "control-if-free" && t.info.Controller == nil {
		t.setController(peer)
	}
	return terminalAttached{Type: "attached", AttachmentID: peer.id, ViewID: peer.view,
		ViewGeneration: req.ViewGeneration, ReplayTo: t.info.Last, terminalInfo: t.info}, ""
}

func (t *desktopTerminal) forgetOldView() {
	inUse := make(map[string]bool, len(t.peers))
	for _, peer := range t.peers {
		inUse[peer.view] = true
	}
	var oldest string
	for key, record := range t.views {
		if !inUse[key] && (oldest == "" || record.lastAttach.Before(t.views[oldest].lastAttach)) {
			oldest = key
		}
	}
	delete(t.views, oldest)
}

func (t *desktopTerminal) detach(peer *terminalPeer) {
	peer.cancel()
	t.action.Lock()
	defer t.action.Unlock()
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.peers, peer.id)
	if t.info.Controller != nil && t.info.Controller.ID == peer.id {
		t.setController(nil)
	}
}

func (t *desktopTerminal) readTerminalPeer(ctx context.Context, ws *websocket.Conn, peer *terminalPeer) {
	defer peer.cancel()
	for {
		var in struct {
			Type, Data, Nonce                string
			Seq, InputSeq, ControlGeneration uint64
			ExpectedControlGeneration        uint64
			terminalSize
		}
		if wsjson.Read(ctx, ws, &in) != nil {
			return
		}
		peer.lastContact.Store(uint64(time.Now().UnixNano()))
		var reply map[string]any
		switch in.Type {
		case "ack":
			if in.Seq < peer.ack.Load() || in.Seq > peer.sent.Load() {
				return
			}
			peer.ack.Store(in.Seq)
			select {
			case peer.wake <- struct{}{}:
			default:
			}
		case "heartbeat":
			if !validUUID(in.Nonce) {
				return
			}
		case "claim":
			reply = t.claimControl(peer, in.ExpectedControlGeneration)
		case "resize":
			reply = t.resizeTerminal(peer, in.ControlGeneration, in.terminalSize)
		case "input":
			reply = t.writeTerminalInput(ctx, peer, in.ControlGeneration, in.InputSeq, in.Data)
		default:
			reply = terminalSocketError(in.Type, "terminal_protocol_failed")
		}
		if reply != nil {
			select {
			case peer.responses <- reply:
			case <-ctx.Done():
				return
			default:
				return
			}
		}
	}
}

func (t *desktopTerminal) writeTerminalPeer(ctx context.Context, ws *websocket.Conn, peer *terminalPeer, attached terminalAttached) {
	pulse := time.NewTicker(15 * time.Second)
	defer pulse.Stop()
	generation := attached.Generation
	replayed := false
	for ctx.Err() == nil {
		t.mu.Lock()
		cursor, ack := peer.sent.Load(), peer.ack.Load()
		if cursor+1 < t.info.First {
			t.mu.Unlock()
			_ = sendTerminalFrame(ctx, ws, terminalSocketError("attach", "terminal_history_lost"))
			return
		}
		currentGen, controller := t.info.Generation, t.info.Controller
		var batch []terminalHistoryFrame
		if cursor < t.info.Last && cursor-ack < 32 {
			first := cursor + 1 - t.info.First
			last := min(t.info.Last+1-t.info.First, first+(32-(cursor-ack)))
			if !replayed {
				last = min(last, attached.ReplayTo+1-t.info.First)
			}
			batch = slices.Clone(t.frames[first:last])
		}
		t.mu.Unlock()
		if currentGen != generation {
			generation = currentGen
			if err := sendTerminalFrame(ctx, ws, map[string]any{"type": "control", "controlGeneration": generation, "controller": controller, "pending": false}); err != nil {
				return
			}
		}
		for _, frame := range batch {
			peer.sent.Store(frame.Seq)
			if err := sendTerminalFrame(ctx, ws, frame); err != nil {
				return
			}
		}
		if !replayed && peer.sent.Load() >= attached.ReplayTo {
			if err := sendTerminalFrame(ctx, ws, map[string]any{"type": "replay_end", "seq": attached.ReplayTo}); err != nil {
				return
			}
			replayed = true
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-peer.wake:
		case reply := <-peer.responses:
			if err := sendTerminalFrame(ctx, ws, reply); err != nil {
				return
			}
		case <-pulse.C:
			if time.Since(time.Unix(0, int64(peer.lastContact.Load()))) > 45*time.Second {
				return
			}
			nonce, err := newUUID()
			if err != nil || sendTerminalFrame(ctx, ws, map[string]any{"type": "heartbeat", "nonce": nonce}) != nil {
				return
			}
		}
	}
}

func (t *desktopTerminal) claimControl(peer *terminalPeer, expected uint64) map[string]any {
	t.action.Lock()
	defer t.action.Unlock()
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.info.State != "running" || t.info.Generation != expected || t.peers[peer.id] != peer {
		return terminalSocketError("claim", "terminal_control_changed")
	}
	if t.info.Controller == nil || t.info.Controller.ID != peer.id {
		t.setController(peer)
	}
	return nil
}

func (t *desktopTerminal) canControl(peer *terminalPeer, generation uint64) bool {
	return t.info.State == "running" && t.info.Controller != nil && t.info.Controller.ID == peer.id && t.info.Generation == generation
}

func (t *desktopTerminal) resizeTerminal(peer *terminalPeer, generation uint64, size terminalSize) map[string]any {
	if !size.valid() {
		return terminalSocketError("resize", "invalid_terminal_size")
	}
	t.action.Lock()
	defer t.action.Unlock()
	t.mu.Lock()
	if !t.canControl(peer, generation) {
		t.mu.Unlock()
		return terminalSocketError("resize", "terminal_control_changed")
	}
	if t.info.terminalSize == size {
		t.mu.Unlock()
		return nil
	}
	t.mu.Unlock()
	if err := t.process.Resize(size); err != nil {
		return terminalSocketError("resize", "terminal_unavailable")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.canControl(peer, generation) {
		t.info.terminalSize = size
		t.appendFrame(terminalHistoryFrame{Type: "size", Cols: size.Cols, Rows: size.Rows})
	}
	return nil
}

func (t *desktopTerminal) writeTerminalInput(ctx context.Context, peer *terminalPeer, generation, sequence uint64, encoded string) map[string]any {
	fail := func(code string, written int, unknown bool) map[string]any {
		reply := terminalSocketError("input", code)
		if sequence > 0 && sequence <= maxTerminalCounter {
			reply["inputSeq"] = sequence
		}
		reply["writtenBytes"], reply["outcomeUnknown"] = written, unknown
		return reply
	}
	if sequence != peer.input+1 || sequence > maxTerminalCounter {
		return fail("invalid_terminal_sequence", 0, false)
	}
	peer.input = sequence
	data, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil || len(data) == 0 || len(data) > maxTerminalInput || base64.StdEncoding.EncodeToString(data) != encoded {
		return fail("invalid_terminal_input", 0, false)
	}
	t.action.Lock()
	defer t.action.Unlock()
	t.mu.Lock()
	control := t.canControl(peer, generation)
	t.mu.Unlock()
	if !control || ctx.Err() != nil {
		return fail("terminal_control_changed", 0, false)
	}
	type result struct {
		n   int
		err error
	}
	done := make(chan result, 1)
	go func() {
		n, err := t.process.Write(data)
		done <- result{n, err}
	}()
	deadline, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	select {
	case outcome := <-done:
		if outcome.err != nil || outcome.n != len(data) {
			return fail("terminal_unavailable", outcome.n, outcome.n != 0)
		}
		return map[string]any{"type": "input_ack", "inputSeq": sequence, "writtenBytes": outcome.n}
	case <-deadline.Done():
		// End this terminal before another viewer can take control with an
		// outstanding write whose outcome is unknown.
		t.terminate()
		return fail("terminal_unavailable", 0, true)
	}
}
