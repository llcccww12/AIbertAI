package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func (h *relaySessionHost) deleteDesktopSessions(ctx context.Context, ws *websocket.Conn, method string, raw json.RawMessage) (any, error) {
	var args struct {
		SessionID       string
		SessionIDs      []string
		DeleteWorkspace bool
		Meta            struct{ DeleteWorkspace bool } `json:"_meta"`
	}
	if json.Unmarshal(raw, &args) != nil {
		return nil, errors.New("invalid_session_parameters")
	}
	single := method == "session/delete"
	if single {
		args.SessionIDs = []string{args.SessionID}
		args.DeleteWorkspace = args.Meta.DeleteWorkspace
	}
	ids := make([]string, 0, min(200, len(args.SessionIDs)))
	seen := make(map[string]bool)
	for _, id := range args.SessionIDs {
		if !validUUID(id) || len(ids) == 200 && !seen[id] {
			return nil, errors.New("invalid_session_parameters")
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	if len(ids) == 0 {
		return nil, errors.New("invalid_session_parameters")
	}
	deleted := make([]string, 0, len(ids))
	failed := []map[string]string{}
	for _, id := range ids {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		found, err := h.trashDesktopSession(id, args.DeleteWorkspace)
		if err != nil {
			if single {
				return nil, err
			}
			failed = append(failed, map[string]string{"sessionId": id, "reason": err.Error()})
			continue
		}
		if found {
			if h.terminals != nil {
				h.terminals.closeSession(id)
			}
			_ = wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "method": "_monkeyai/session/deleted",
				"params": map[string]string{"sessionId": id}})
		}
		deleted = append(deleted, id)
	}
	if single {
		return map[string]any{}, nil
	}
	return map[string]any{"deleted": deleted, "failed": failed}, nil
}

type desktopTrashMove struct {
	root              *os.Root
	source, directory string
}

func (m desktopTrashMove) target() string {
	return filepath.Join(m.directory, filepath.Base(m.source))
}

func (m desktopTrashMove) move() (bool, error) {
	if _, err := m.root.Lstat(m.source); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if err := m.root.MkdirAll(m.directory, 0700); err != nil {
		return false, err
	}
	if _, err := m.root.Lstat(m.target()); !errors.Is(err, os.ErrNotExist) {
		return false, errors.New("session_trash_conflict")
	}
	err := m.root.Rename(m.source, m.target())
	return err == nil, err
}

// The pinned SDK stores a session's immutable generations and lock together
// beneath a project directory. Discover the project as its JSONL backend does:
// a workspace can move, and the SDK owns the encoded project directory name.
func desktopDSHTrashMoves(root *os.Root, id string) ([]desktopTrashMove, error) {
	projects, err := fs.ReadDir(root.FS(), "dsh/home/sessions")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	moves := []desktopTrashMove{
		{root, filepath.Join("dsh", "gateway", id), filepath.Join("deleted-dsh", "gateway")},
		{root, filepath.Join("dsh", "home", "storages", "session_projcache", "sessions", id+".json"), filepath.Join("deleted-dsh", "projection-cache")},
	}
	for _, project := range projects {
		if project.IsDir() {
			moves = append(moves, desktopTrashMove{root, filepath.Join("dsh", "home", "sessions", project.Name(), id), filepath.Join("deleted-dsh", "sessions", project.Name())})
		}
	}
	return moves, nil
}

func (h *relaySessionHost) trashDesktopSession(id string, deleteWorkspace bool) (bool, error) {
	l := h.live
	l.mu.Lock()
	defer l.mu.Unlock()
	row, exists := l.sessions[id]
	if exists && (row.State == "running" || l.compacting[id]) {
		return false, errors.New("session_busy")
	}
	if exists && l.dsh != nil && l.dsh.LeasePool != nil && row.DSHSessionID != "" {
		if err := l.dsh.LeasePool.evictIfIdle(row.DSHSessionID); err != nil {
			return false, err
		}
	}
	profile, err := os.OpenRoot(h.profile)
	if err != nil {
		return false, errors.New("session_delete_failed")
	}
	defer profile.Close()
	moves := []desktopTrashMove{{profile, filepath.Join("runtime", "sessions", id), filepath.Join("runtime", "deleted-sessions")}}
	if exists {
		if deleteWorkspace && l.workspaces != nil && row.Workspace == "default" && row.CWD == l.workspaces.defaultPath(id) {
			moves = append(moves, desktopTrashMove{l.workspaces.root, filepath.Join("workspaces", id), "deleted-workspaces"})
		}
		for _, suffix := range []string{".json", desktopSessionManifestSuffix, desktopSessionManifestSuffix + ".chunks", ".turns"} {
			moves = append(moves, desktopTrashMove{l.store.Root, id + suffix, ".deleted"})
		}
		moves = append(moves, desktopTrashMove{l.store.Root, filepath.Join("dsh-connector-dispatch", id), filepath.Join(".deleted", "dsh-connector-dispatch")})
		if l.workspaces != nil && row.Engine == "dsh" && row.DSHSessionID == id {
			owned, err := desktopDSHTrashMoves(l.workspaces.root, id)
			if err != nil {
				return false, errors.New("session_delete_failed")
			}
			moves = append(moves, owned...)
		}
	}
	var completed []desktopTrashMove
	for _, move := range moves {
		moved, err := move.move()
		if err != nil {
			for _, previous := range slices.Backward(completed) {
				if previous.root.Rename(previous.target(), previous.source) != nil {
					return false, errors.New("session_delete_recovery_required")
				}
			}
			return false, errors.New("session_delete_failed")
		}
		if moved {
			completed = append(completed, move)
		}
	}
	if exists {
		delete(l.sessions, id)
		delete(l.operations, row.Operation)
	}
	return exists || len(completed) != 0, nil
}
