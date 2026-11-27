package main

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
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

// 只杀 DSH 本身会把它的子进程留成孤儿，继续占着端口和句柄。关闭后整组必须消失。
func TestDSHCloseEndsProcessGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses process groups")
	}
	cmd := exec.Command("sh", "-c", "sleep 30 & sleep 30")
	p, err := startDSH(t.Context(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	pid := p.cmd.Process.Pid
	p.close()
	deadline := time.Now().Add(3 * time.Second)
	for {
		err := syscall.Kill(-pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("关闭后进程组 %d 仍然存在: %v", pid, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
