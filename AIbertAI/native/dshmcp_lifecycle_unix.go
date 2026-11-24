//go:build darwin || linux

package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// dshMCPContainment must be implemented by a trusted OS sandbox which both
// prevents process-group escape and can prove the entire sandbox is empty.
// There is deliberately no production implementation until that is verified.
type dshMCPContainment interface {
	configure(*exec.Cmd) error
	verifyStopped() error
}

type dshMCPProcess struct {
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      io.ReadCloser
	root        *os.Root
	record      dshMCPStartupRecord
	containment dshMCPContainment
	closeOnce   sync.Once
	closeErr    error
}
type dshMCPStartupRecord struct {
	Version     int    `json:"version"`
	SessionID   string `json:"sessionId"`
	OperationID string `json:"operationId"`
	ConnectorID string `json:"connectorId"`
	AttemptID   string `json:"attemptId"`
	State       string `json:"state"`
	PID         int    `json:"pid,omitempty"`
}

func dshMCPStartupDir(session, connector string) string {
	return filepath.ToSlash(filepath.Join("dsh-mcp-startup", session, connector))
}
func dshMCPRecordPath(r dshMCPStartupRecord) string {
	return dshMCPStartupDir(r.SessionID, r.ConnectorID) + "/.pending"
}
func dshMCPReadPending(root *os.Root, session, connector string) (dshMCPStartupRecord, error) {
	var r dshMCPStartupRecord
	if root == nil || !validUUID(session) || !validUUID(connector) {
		return r, errors.New("dsh_mcp_startup_invalid")
	}
	name := dshMCPStartupDir(session, connector) + "/.pending"
	info, err := root.Lstat(name)
	if err != nil {
		return r, err
	}
	if !info.Mode().IsRegular() || info.Size() > 4096 {
		return r, errors.New("dsh_mcp_startup_invalid")
	}
	f, err := root.Open(name)
	if err != nil {
		return r, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return r, errors.New("dsh_mcp_startup_invalid")
	}
	dec := json.NewDecoder(io.LimitReader(f, 4097))
	dec.DisallowUnknownFields()
	if dec.Decode(&r) != nil || dec.Decode(new(any)) != io.EOF || r.Version != 1 || r.SessionID != session || r.ConnectorID != connector || !validUUID(r.OperationID) || !validUUID(r.AttemptID) || (r.State != "claimed" && r.State != "running" && r.State != "closed") || (r.State == "claimed" && r.PID != 0) || (r.State != "claimed" && r.PID <= 0) {
		return dshMCPStartupRecord{}, errors.New("dsh_mcp_startup_invalid")
	}
	return r, nil
}
func dshMCPWriteRecord(root *os.Root, r dshMCPStartupRecord) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if err = writeSourceState(root, dshMCPRecordPath(r), data); err != nil {
		return err
	}
	return syncConnectorDir(root, dshMCPStartupDir(r.SessionID, r.ConnectorID))
}

// startDSHMCPProcess is usable only after the host has revalidated the approved
// command and session. It never accepts an executable or environment from DSH.
func startDSHMCPProcess(ctx context.Context, root *os.Root, session, operation, connector string, entry legacyMCP, executable, cwd string, containment dshMCPContainment) (*dshMCPProcess, error) {
	if root == nil || containment == nil || !validUUID(session) || !validUUID(operation) || !validUUID(connector) || !filepath.IsAbs(executable) || !filepath.IsAbs(cwd) {
		return nil, errors.New("dsh_mcp_startup_invalid")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	env, err := localMCPEnvironment(entry.Env)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(executable, entry.Args...)
	cmd.Dir, cmd.Env, cmd.WaitDelay = cwd, env, 2*time.Second
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := containment.configure(cmd); err != nil {
		return nil, errors.New("dsh_mcp_containment_unavailable")
	}
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setpgid {
		return nil, errors.New("dsh_mcp_containment_unavailable")
	}
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		in.Close()
		return nil, err
	}
	cleanup := func() { in.Close(); out.Close() }
	attempt, err := newUUID()
	if err != nil {
		cleanup()
		return nil, err
	}
	dir := dshMCPStartupDir(session, connector)
	for _, p := range []string{"dsh-mcp-startup", "dsh-mcp-startup/" + session, dir} {
		if err = root.MkdirAll(p, 0700); err != nil {
			cleanup()
			return nil, err
		}
		parent := filepath.ToSlash(filepath.Dir(p))
		if err = syncConnectorDir(root, parent); err != nil {
			cleanup()
			return nil, err
		}
	}
	r := dshMCPStartupRecord{Version: 1, SessionID: session, OperationID: operation, ConnectorID: connector, AttemptID: attempt, State: "claimed"}
	data, err := json.Marshal(r)
	if err != nil {
		cleanup()
		return nil, err
	}
	marker, err := root.OpenFile(dshMCPRecordPath(r), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		cleanup()
		return nil, errors.New("dsh_mcp_reconciliation_required")
	}
	if _, err = marker.Write(data); err == nil {
		err = marker.Sync()
	}
	err = errors.Join(err, marker.Close())
	if err == nil {
		err = syncConnectorDir(root, dir)
	}
	if err != nil {
		cleanup()
		return nil, errors.New("dsh_mcp_reconciliation_required")
	}
	if err = ctx.Err(); err != nil {
		cleanup()
		return nil, errors.Join(errors.New("dsh_mcp_reconciliation_required"), err)
	}
	if err = cmd.Start(); err != nil {
		cleanup()
		return nil, errors.Join(errors.New("dsh_mcp_reconciliation_required"), err)
	}
	p := &dshMCPProcess{cmd: cmd, stdin: in, stdout: out, root: root, record: r, containment: containment}
	p.record.State, p.record.PID = "running", cmd.Process.Pid
	if err = dshMCPWriteRecord(root, p.record); err != nil {
		_ = p.close()
		return nil, errors.New("dsh_mcp_reconciliation_required")
	}
	context.AfterFunc(ctx, func() { _ = p.close() })
	return p, nil
}

// close is a host-only receipt: it does not claim an SDK tool/result exists.
// A failed kill, wait, group verification or durable write leaves .pending.
func (p *dshMCPProcess) close() error {
	p.closeOnce.Do(func() {
		pid := p.cmd.Process.Pid
		if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			p.closeErr = errors.New("dsh_mcp_reconciliation_required")
			return
		}
		_ = p.stdin.Close()
		_ = p.stdout.Close()
		err := p.cmd.Wait()
		var exit *exec.ExitError
		if err != nil && !errors.As(err, &exit) {
			p.closeErr = errors.New("dsh_mcp_reconciliation_required")
			return
		}
		deadline := time.Now().Add(2 * time.Second)
		for {
			err = syscall.Kill(-pid, 0)
			if errors.Is(err, syscall.ESRCH) {
				break
			}
			if err != nil || time.Now().After(deadline) {
				p.closeErr = errors.New("dsh_mcp_reconciliation_required")
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		// Group absence alone does not cover children that escaped the group.
		if err = p.containment.verifyStopped(); err != nil {
			p.closeErr = errors.New("dsh_mcp_reconciliation_required")
			return
		}
		p.record.State = "closed"
		if err = dshMCPWriteRecord(p.root, p.record); err != nil {
			p.closeErr = errors.New("dsh_mcp_reconciliation_required")
			return
		}
		dir := dshMCPStartupDir(p.record.SessionID, p.record.ConnectorID)
		if err = p.root.Link(dshMCPRecordPath(p.record), dir+"/"+p.record.AttemptID+".json"); err != nil {
			p.closeErr = errors.New("dsh_mcp_reconciliation_required")
			return
		}
		if err = syncConnectorDir(p.root, dir); err != nil {
			p.closeErr = errors.New("dsh_mcp_reconciliation_required")
			return
		}
		if err = p.root.Remove(dshMCPRecordPath(p.record)); err != nil {
			p.closeErr = errors.New("dsh_mcp_reconciliation_required")
			return
		}
		if err = syncConnectorDir(p.root, dir); err != nil {
			p.closeErr = errors.New("dsh_mcp_reconciliation_required")
		}
	})
	return p.closeErr
}
