package main

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Stop during a host tool must still answer the SDK and send the cancel, or
// DSH never learns about it and the turn waits out the cancel timer.
func TestDSHStopDuringHostToolSendsCancel(t *testing.T) {
	frames := make(chan dshFrame, 4)
	frames <- dshFrame{JSONRPC: "2.0", ID: json.RawMessage(`7`), Method: "aibertai/host-tool", Params: json.RawMessage(`{"name":"canvas_commit_html"}`)}
	frames <- dshFrame{JSONRPC: "2.0", ID: json.RawMessage(`-1`), Result: json.RawMessage(`{"accepted":true}`)}
	frames <- dshFrame{JSONRPC: "2.0", ID: json.RawMessage(`1`), Result: json.RawMessage(`{"stopReason":"cancelled"}`)}
	in := &dshTestWriteCloser{}
	p := &dshProcess{in: in, frames: frames, cancelSession: "session-1"}
	ctx, cancel := context.WithCancel(t.Context())
	p.hostTool = func(json.RawMessage) (any, error) {
		cancel()
		return nil, errors.New("design_tool_unavailable")
	}
	result, err := p.call(ctx, "session/wait", nil, nil)
	if err != nil || !strings.Contains(string(result), "cancelled") || !p.cancelAccepted {
		t.Fatalf("stop was not confirmed: result=%s err=%v", result, err)
	}
	out := in.String()
	if !strings.Contains(out, `"id":7`) || !strings.Contains(out, "aibertai/session-cancel") {
		t.Fatalf("DSH was not told: %s", out)
	}
}

// A child that keeps DSH's output pipe open must not block closing it.
func TestDSHCloseDoesNotWaitForChildren(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	cmd := exec.Command("sh", "-c", "(sleep 10) & sleep 10")
	p, err := startDSH(t.Context(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	p.close()
	if elapsed := time.Since(started); elapsed > 4*time.Second {
		t.Fatalf("close waited %v", elapsed)
	}
}
