//go:build darwin || linux

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

type unixTerminal struct {
	*os.File
	command *exec.Cmd
	stop    sync.Once
}

func startTerminalProcess(cwd string, size terminalSize) (terminalProcess, error) {
	shell := os.Getenv("SHELL")
	if !filepath.IsAbs(shell) {
		shell = "/bin/sh"
	}
	command := exec.Command(shell, "-i")
	command.Dir = cwd
	command.Env = terminalEnvironment()
	terminal, err := pty.StartWithSize(command, &pty.Winsize{Cols: uint16(size.Cols), Rows: uint16(size.Rows)})
	if err != nil {
		return nil, err
	}
	return &unixTerminal{File: terminal, command: command}, nil
}

func (t *unixTerminal) Shell() string { return filepath.Base(t.command.Path) }

func (t *unixTerminal) Resize(size terminalSize) error {
	return pty.Setsize(t.File, &pty.Winsize{Cols: uint16(size.Cols), Rows: uint16(size.Rows)})
}

func (t *unixTerminal) Stop() {
	t.stop.Do(func() {
		pid := t.command.Process.Pid
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		_ = t.command.Process.Kill()
		_ = t.File.Close()
	})
}

func (t *unixTerminal) Wait() int {
	_ = t.command.Wait()
	_ = t.File.Close()
	if t.command.ProcessState == nil {
		return -1
	}
	return t.command.ProcessState.ExitCode()
}
