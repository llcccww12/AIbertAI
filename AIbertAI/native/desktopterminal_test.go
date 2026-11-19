package main

import (
	"context"
	"encoding/base64"
	"testing"
)

func TestTerminalViewGenerationAndInputAuthority(t *testing.T) {
	terminal := desktopTerminal{info: terminalInfo{State: "running", First: 1},
		views: make(map[string]terminalView), peers: make(map[string]*terminalPeer)}
	makePeer := func(id, view string) (*terminalPeer, context.Context) {
		ctx, cancel := context.WithCancel(t.Context())
		t.Cleanup(cancel)
		return &terminalPeer{id: id, view: view, name: "test", cancel: cancel, wake: make(chan struct{}, 1)}, ctx
	}
	first, old := makePeer("first", "view-1")
	attached, fault := terminal.attach(first, terminalAttach{ViewID: first.view, ViewGeneration: 1, Mode: "control-if-free"})
	if fault != "" || attached.Controller == nil || attached.Controller.ID != first.id {
		t.Fatalf("initial terminal controller: %+v, %s", attached, fault)
	}
	second, _ := makePeer("second", first.view)
	if _, fault := terminal.attach(second, terminalAttach{ViewID: second.view, ViewGeneration: 1}); fault != "terminal_conflict" {
		t.Fatalf("accepted a replayed attachment generation: %s", fault)
	}
	replacement, fault := terminal.attach(second, terminalAttach{ViewID: second.view, ViewGeneration: 2, Mode: "control-if-free"})
	if fault != "" || old.Err() == nil || replacement.Controller == nil || replacement.Controller.ID != second.id {
		t.Fatalf("view replacement did not transfer authority: %+v, %s", replacement, fault)
	}
	reply := terminal.writeTerminalInput(t.Context(), first, attached.Generation, 1, base64.StdEncoding.EncodeToString([]byte("x")))
	if reply["code"] != "terminal_control_changed" {
		t.Fatalf("replaced viewer could still write: %v", reply)
	}
	terminal.detach(first)
	if terminal.info.Controller == nil || terminal.info.Controller.ID != second.id {
		t.Fatal("disconnecting the old viewer cleared the new viewer")
	}
}

func TestTerminalHistoryKeepsContiguousReplayWindow(t *testing.T) {
	terminal := desktopTerminal{info: terminalInfo{State: "running", First: 1},
		views: make(map[string]terminalView), peers: make(map[string]*terminalPeer)}
	for range maxTerminalFrames + 3 {
		terminal.appendFrame(terminalHistoryFrame{Type: "size", Cols: 100, Rows: 30})
	}
	if !terminal.info.Truncated || terminal.info.First != 4 || len(terminal.frames) != maxTerminalFrames {
		t.Fatal("terminal did not retain exactly the expected replay window")
	}
	for index, frame := range terminal.frames {
		if frame.Seq != terminal.info.First+uint64(index) {
			t.Fatal("terminal replay has a sequence gap")
		}
	}
	view := &terminalPeer{}
	if _, fault := terminal.attach(view, terminalAttach{ViewGeneration: 1, AfterSeq: 2}); fault != "terminal_history_lost" {
		t.Fatalf("expired output cursor was accepted: %q", fault)
	}
}
