package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

var desktopSessionRecordMu sync.RWMutex

const maxDesktopSessionBytes = 8 << 20
const desktopSessionManifestSuffix = ".state.json"

type desktopSessionStore struct {
	*os.Root
	lock *os.File
}

func (s *desktopSessionStore) Close() error {
	return errors.Join(s.lock.Close(), s.Root.Close())
}

type desktopSessionManifest struct {
	acpSession
	Archived int `json:"archivedTurnCount"`
}

func openEmployeeSessions(profile, serverID, machineID, verificationDirectory string) (*acpLive, error) {
	base := filepath.Join(profile, "source-runtime", serverID, machineID)
	cwd := filepath.Join(base, "workspaces")
	live, err := openDesktopSessions(profile, serverID, machineID, cwd)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(base)
	if err == nil {
		live.workspaces, err = openDesktopWorkspaces(root, cwd, verificationDirectory)
	}
	if err == nil {
		err = live.workspaces.importLegacyProjects(profile)
	}
	if err == nil {
		err = live.migrateDefaultWorkspaces()
	}
	if err == nil {
		err = live.workspaces.recoverPatch(live.sessions)
	}
	if err != nil {
		if root != nil {
			_ = root.Close()
		}
		_ = live.store.Close()
		return nil, err
	}
	return live, nil
}

func (l *acpLive) migrateDefaultWorkspaces() error {
	for _, row := range l.sessions {
		if row.Workspace == "selected" || row.Workspace == "default" && row.CWD == l.workspaces.defaultPath(row.ID) {
			continue
		}
		if err := l.workspaces.allocate(row.ID); err != nil {
			return err
		}
		row.Workspace, row.CWD = "default", l.workspaces.defaultPath(row.ID)
		if err := l.save(row); err != nil {
			return err
		}
	}
	return nil
}

// fileSharedDefaultSessions moves conversations already opened in the shared
// default workspace under its 默认工作区 project, so the sidebar lists them
// there like new ones.
func (l *acpLive) fileSharedDefaultSessions(projectID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, row := range l.sessions {
		if !row.SharedDefault || row.ProjectID != "" || row.Workspace != "selected" || row.CWD != l.workspaces.shared {
			continue
		}
		row.SharedDefault, row.ProjectID = false, projectID
		if err := l.save(row); err != nil {
			return err
		}
	}
	return nil
}

func openDesktopSessions(profile, serverID, machineID, cwd string) (*acpLive, error) {
	if !filepath.IsAbs(profile) || !filepath.IsAbs(cwd) || !validUUID(serverID) || !validUUID(machineID) {
		return nil, errors.New("invalid_desktop_session_store")
	}
	profileRoot, err := os.OpenRoot(profile)
	if err != nil {
		return nil, err
	}
	defer profileRoot.Close()
	relative := filepath.Join("source-runtime", serverID, machineID, "sessions")
	if err := profileRoot.MkdirAll(relative, 0700); err != nil {
		return nil, err
	}
	root, err := profileRoot.OpenRoot(relative)
	if err != nil {
		return nil, err
	}
	lock, err := lockDesktopSessionStore(root)
	if err != nil {
		root.Close()
		return nil, err
	}
	store := &desktopSessionStore{root, lock}
	live := &acpLive{store: store, cwd: cwd, operations: map[string]string{}, sessions: map[string]acpSession{}}
	if err := live.loadConnectorPreferences(); err != nil {
		store.Close()
		return nil, err
	}
	if err := live.loadDesktopSettings(); err != nil {
		store.Close()
		return nil, err
	}
	if err := live.restoreDesktopSessions(); err != nil {
		store.Close()
		return nil, err
	}
	return live, nil
}

func readDesktopSessionRecord(root *os.Root, name string, destination any) error {
	desktopSessionRecordMu.RLock()
	defer desktopSessionRecordMu.RUnlock()
	info, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > maxDesktopSessionBytes {
		return errors.New("invalid_desktop_session_snapshot")
	}
	file, err := root.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return errors.New("desktop_session_snapshot_changed")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxDesktopSessionBytes+1))
	if err != nil || len(data) > maxDesktopSessionBytes {
		return errors.New("invalid_desktop_session_snapshot")
	}
	var marker struct {
		Format string `json:"_aibertaiChunkRecord"`
	}
	if json.Unmarshal(data, &marker) != nil {
		return errors.New("invalid_desktop_session_snapshot")
	}
	if marker.Format != "" {
		var record desktopChunkRecord
		if json.Unmarshal(data, &record) != nil {
			return errors.New("invalid_desktop_session_snapshot")
		}
		return readDesktopLargeRecord(root, name, record, destination)
	}
	return json.Unmarshal(data, destination)
}

func desktopTurnRecordName(id string, index int) string {
	return filepath.Join(id+".turns", strconv.Itoa(index+1)+".json")
}

func readDesktopSession(root *os.Root, id string) (acpSession, error) {
	if !validUUID(id) {
		return acpSession{}, errors.New("invalid_desktop_session_id")
	}
	var manifest desktopSessionManifest
	if err := readDesktopSessionRecord(root, id+desktopSessionManifestSuffix, &manifest); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return acpSession{}, err
		}
		manifest = desktopSessionManifest{}
		if err := readDesktopSessionRecord(root, id+".json", &manifest.acpSession); err != nil {
			return acpSession{}, err
		}
	} else if manifest.Archived < 0 || manifest.Archived > 0 && len(manifest.Turns) == 0 {
		return acpSession{}, errors.New("invalid_desktop_session_manifest")
	}
	session := manifest.acpSession
	if session.ID != id || !validUUID(session.Operation) || !validUUID(session.Model) || !filepath.IsAbs(session.CWD) ||
		(session.Workspace != "" && session.Workspace != "default" && session.Workspace != "selected") ||
		(session.ProjectID != "" && (!validUUID(session.ProjectID) || session.Workspace != "selected")) ||
		(session.SharedDefault && (session.Workspace != "selected" || session.ProjectID != "")) ||
		(session.State != "idle" && session.State != "error" && session.State != "running") ||
		session.Created.IsZero() || session.Updated.Before(session.Created) ||
		(session.State == "running" && len(session.Turns) == 0) {
		return acpSession{}, errors.New("invalid_desktop_session_snapshot")
	}
	if manifest.Archived > 0 {
		var turns []acpTurn
		for index := range manifest.Archived {
			var turn acpTurn
			if err := readDesktopSessionRecord(root, desktopTurnRecordName(id, index), &turn); err != nil {
				return acpSession{}, fmt.Errorf("读取会话回合 %d: %w", index+1, err)
			}
			turns = append(turns, turn)
		}
		session.Turns = append(turns, session.Turns...)
		session.archivedTurns = manifest.Archived
	}
	if summary := session.ContextSummary; summary != nil && (summary.Through < 1 || summary.Through > len(session.Turns) ||
		len(summary.Text) == 0 || len(summary.Text) > 8<<10) {
		return acpSession{}, errors.New("invalid_desktop_context_summary")
	}
	operations := make(map[string]struct{}, len(session.Turns))
	for i, turn := range session.Turns {
		_, duplicate := operations[turn.Operation]
		unfinishedGoal := i == len(session.Turns)-1 && session.GoalRun == turn.Operation && turn.GoalCheckpoint != nil &&
			session.Goal != nil && session.Goal.Current
		if !validUUID(turn.Operation) || turn.Started.IsZero() ||
			(turn.Ended.IsZero() && (i < len(session.Turns)-1 || session.State != "running" && !unfinishedGoal)) ||
			(!turn.Ended.IsZero() && turn.Ended.Before(turn.Started)) ||
			(!turn.Acked && i < manifest.Archived) ||
			(turn.Acked && turn.Ended.IsZero()) || duplicate {
			return acpSession{}, errors.New("invalid_desktop_turn_snapshot")
		}
		operations[turn.Operation] = struct{}{}
	}
	return session, nil
}

func saveDesktopSession(root *os.Root, session acpSession, previous int) (int, error) {
	committed := max(0, len(session.Turns)-1)
	if previous > committed {
		return 0, errors.New("cannot_discard_desktop_session_turns")
	}
	// Keep the unreported tail mutable until the background reporter has
	// acknowledged each turn in order.
	for index := previous; index < committed; index++ {
		if !session.Turns[index].Acked {
			committed = index
			break
		}
	}
	// The manifest is the commit point. Older acknowledged turns are immutable.
	// A record written before a failed manifest rename is ignored on restart.
	if committed > previous {
		if err := root.MkdirAll(session.ID+".turns", 0700); err != nil {
			return 0, err
		}
		for index := previous; index < committed; index++ {
			turn := session.Turns[index]
			if !turn.Acked || turn.Ended.IsZero() {
				return 0, errors.New("cannot_archive_unacknowledged_turn")
			}
			if err := saveDesktopSessionRecord(root, desktopTurnRecordName(session.ID, index), turn); err != nil {
				return 0, err
			}
		}
	}
	session.Turns = session.Turns[committed:]
	err := saveDesktopSessionRecord(root, session.ID+desktopSessionManifestSuffix, desktopSessionManifest{session, committed})
	return committed, err
}

func saveDesktopSessionRecord(root *os.Root, name string, value any) error {
	desktopSessionRecordMu.Lock()
	defer desktopSessionRecordMu.Unlock()
	data, err := json.Marshal(value)
	if err != nil {
		return errors.New("desktop_session_snapshot_too_large")
	}
	if len(data) > maxDesktopSessionBytes {
		return saveDesktopLargeRecord(root, name, data)
	}
	if err := writeSourceState(root, name, data); err != nil {
		return err
	}
	if info, err := root.Lstat(name + ".chunks"); err == nil && info.IsDir() {
		if err := syncDesktopDir(root, filepath.Dir(name)); err != nil {
			return err
		}
		pruneDesktopChunks(root, name, nil)
	}
	return nil
}

func writeSourceState(root *os.Root, name string, data []byte) error {
	return persistSourceState(root, name, data, false)
}

func persistSourceState(root *os.Root, name string, data []byte, exclusive bool) error {
	temp, err := newUUID()
	if err != nil {
		return err
	}
	temp += ".tmp"
	file, err := root.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer root.Remove(temp)
	_, writeErr := file.Write(data)
	if err := errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		return err
	}
	if exclusive {
		return root.Link(temp, name)
	}
	return root.Rename(temp, name)
}

func (l *acpLive) restoreDesktopSessions() error {
	entries, err := l.store.Open(".")
	if err != nil {
		return err
	}
	defer entries.Close()
	for {
		batch, readErr := entries.ReadDir(128)
		for _, item := range batch {
			id, ok := strings.CutSuffix(item.Name(), desktopSessionManifestSuffix)
			if !ok {
				id, ok = strings.CutSuffix(item.Name(), ".json")
			}
			if !ok || !validUUID(id) || l.sessions[id].ID != "" {
				continue
			}
			session, err := readDesktopSession(l.store.Root, id)
			if err != nil {
				// One unreadable record must not keep every other conversation
				// shut. Keep the file for repair, move it out of the way so the
				// next start does not trip over it again, and carry on.
				if renameErr := l.store.Rename(item.Name(), item.Name()+".corrupt-"+strconv.FormatInt(time.Now().Unix(), 10)); renameErr != nil {
					fmt.Fprintln(os.Stderr, "员工端会话记录无法读取，且未能移开", item.Name(), err, renameErr)
				} else {
					fmt.Fprintln(os.Stderr, "已跳过无法读取的员工端会话记录，文件保留为 .corrupt 备份", item.Name(), err)
				}
				continue
			}
			if _, duplicate := l.operations[session.Operation]; duplicate {
				return errors.New("duplicate_session_creation_operation")
			}
			l.operations[session.Operation], l.sessions[id] = id, session
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				return readErr
			}
			break
		}
	}
	// An interrupted process may have completed an action before recording
	// its result. Close the interrupted turn without replaying the action.
	for _, session := range l.sessions {
		if session.Engine == "dsh" && session.Goal != nil && session.Goal.Status == "completed" && session.GoalRun != "" {
			if n := len(session.Turns); n > 0 && session.Turns[n-1].Operation == session.GoalRun && session.Turns[n-1].Ended.IsZero() {
				goal := *session.Goal
				goal.Current, goal.Status, goal.Reason = true, "paused", "completion_unverified"
				goal.Version++
				session.Goal, session.State = &goal, "idle"
				if err := l.save(session); err != nil {
					return err
				}
				continue
			}
		}
		// A goal the model opened lives in DSH's own session state, which a
		// platform restart does not restore. Do not show it as still running.
		if session.Engine == "dsh" && session.Goal != nil && session.Goal.Observed && session.Goal.Current && session.Goal.Status == "active" {
			goal := *session.Goal
			goal.Status, goal.Reason = "paused", "host_restarted"
			goal.Version++
			session.Goal = &goal
			if err := l.save(session); err != nil {
				return err
			}
		}
		activeGoal := session.platformGoal() != nil && session.Goal.Current && session.Goal.Status == "active"
		if session.Engine == "dsh" && session.platformGoal() != nil && session.Goal.Current && (activeGoal || session.State == "running" || session.Goal.Status == "reconciling") {
			// DSH restores durable goal state disarmed. Do not re-arm it merely
			// because the platform host restarted.
			goal := *session.Goal
			if activeGoal {
				goal.Status, goal.Reason = "paused", "host_restarted"
			}
			if goal.Status == "reconciling" {
				goal.Status, goal.Reason, goal.Uncertain = "paused", "usage_unknown", true
			}
			if goal.hasPendingDSHInvocation() {
				goal.Reason, goal.Uncertain = "usage_unknown", true
			}
			if n := len(session.Turns); n > 0 {
				for _, run := range session.Turns[n-1].ToolRuns {
					if run.Pending {
						goal.Reason, goal.Uncertain = "resume_confirmation", true
						break
					}
				}
				if saved := session.Turns[n-1].GoalCheckpoint; saved != nil && saved.Pending != nil {
					goal.Reason, goal.Uncertain = "resume_confirmation", true
				}
			}
			goal.Version++
			session.Goal, session.State = &goal, "idle"
			if err := l.save(session); err != nil {
				return err
			}
			continue
		}
		if session.GoalRun != "" && session.Goal != nil && session.Goal.Current {
			if n := len(session.Turns); n > 0 && session.Turns[n-1].Operation == session.GoalRun {
				saved := session.Turns[n-1].GoalCheckpoint
				if saved != nil && (saved.Pending != nil || !activeGoal) {
					goal := *session.Goal
					if saved.Pending != nil {
						goal.Status, goal.Reason, goal.Uncertain = "paused", "resume_confirmation", true
						goal.Version++
					}
					session.Goal, session.State = &goal, "idle"
					if err := l.save(session); err != nil {
						return err
					}
				}
			}
			continue
		}
		if session.State != "running" && !activeGoal {
			continue
		}
		if session.State == "running" {
			session.GoalRun = ""
			session.Turns = slices.Clone(session.Turns)
			last := &session.Turns[len(session.Turns)-1]
			if last.Ended.IsZero() {
				materializeGoalCheckpoint(last)
				last.Failed, last.Ended = true, time.Now().UTC()
				if session.Engine == "dsh" && dshPendingWrite(*last) {
					last.FailureCode = "dsh_write_outcome_unknown"
				} else if session.Engine == "dsh" && dshPendingCanvasWrite(*last) {
					last.FailureCode = "dsh_canvas_outcome_unknown"
				} else if session.Engine == "dsh" && last.FailureCode == "" {
					last.FailureCode = "dsh_runtime_exited"
				}
				if last.Ended.Before(last.Started) {
					last.Ended = last.Started
				}
			}
			session.State, session.Updated = completedState(*last), last.Ended
		}
		if activeGoal {
			goal := *session.Goal
			goal.Status, goal.Reason, goal.Uncertain = "paused", "interrupted_unknown", true
			goal.Version++
			session.Goal = &goal
		}
		if err := l.save(session); err != nil {
			return err
		}
	}
	return nil
}

func completedState(turn acpTurn) string {
	if turn.DSHUserStopped {
		return "idle"
	}
	if turn.Failed {
		return "error"
	}
	return "idle"
}

func (l *acpLive) usesAsyncTurnReports(session acpSession) bool {
	return l.store != nil && (l.reportWake != nil || session.Engine == "dsh" && l.dsh != nil && l.dsh.ContinueAfterFailedTurn)
}

func (l *acpLive) wakeDesktopReports() {
	select {
	case l.reportWake <- struct{}{}:
	default:
	}
}

func pendingTurnReport(row acpSession) int {
	for index := row.archivedTurns; index < len(row.Turns); index++ {
		if !row.Turns[index].Acked {
			if row.Turns[index].Ended.IsZero() {
				return -1
			}
			return index
		}
	}
	return -1
}

func (h *relaySessionHost) reportTurnBacklog(ctx context.Context, id string) error {
	live := h.live
	live.mu.Lock()
	if live.restarting || live.reportingTurns[id] {
		live.mu.Unlock()
		return nil
	}
	if live.reportingTurns == nil {
		live.reportingTurns = make(map[string]bool)
	}
	live.reportingTurns[id] = true
	live.mu.Unlock()
	defer func() {
		live.mu.Lock()
		delete(live.reportingTurns, id)
		live.mu.Unlock()
	}()
	for ctx.Err() == nil {
		live.mu.Lock()
		row := live.sessions[id]
		index := pendingTurnReport(row)
		if live.restarting || index < 0 {
			live.mu.Unlock()
			return nil
		}
		turn := row.Turns[index]
		live.mu.Unlock()

		if err := h.submitACPTurn(ctx, id, index+1, turn, index == 0); err != nil {
			return err
		}

		live.mu.Lock()
		row = live.sessions[id]
		if index >= len(row.Turns) || row.Turns[index].Operation != turn.Operation {
			live.mu.Unlock()
			return nil
		}
		row.Turns = slices.Clone(row.Turns)
		row.Turns[index].Acked = true
		err := live.save(row)
		live.mu.Unlock()
		if err != nil {
			return err
		}
	}
	return ctx.Err()
}

func (h *relaySessionHost) recoverDesktopReports(ctx context.Context) {
	h.live.mu.Lock()
	var pending []string
	for _, session := range h.live.sessions {
		if h.live.usesAsyncTurnReports(session) {
			if pendingTurnReport(session) >= 0 {
				pending = append(pending, session.ID)
			}
			continue
		}
		last := len(session.Turns) - 1
		if last >= 0 && session.State != "running" && !session.Turns[last].Acked {
			pending = append(pending, session.ID)
		}
	}
	h.live.mu.Unlock()
	for _, id := range pending {
		if ctx.Err() != nil {
			return
		}
		if err := h.retryPendingReport(ctx, id); err != nil && !errors.Is(err, errSessionReportPending) {
			fmt.Fprintln(os.Stderr, "员工端有待补传的会话回合", id, err)
		}
	}
}

func (h *relaySessionHost) superviseDesktopReports(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-h.live.reportWake:
		}
		attempt, cancel := context.WithTimeout(ctx, 10*time.Second)
		var err error
		if h.live.source != nil {
			_, err = h.live.source.access(attempt)
		}
		if err == nil {
			h.recoverDesktopReports(attempt)
		}
		cancel()
	}
}
