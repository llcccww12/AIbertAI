package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

const workspacePatchJournalFile = "workspace-patch-pending.json"

var errPatchConflict = errors.New("workspace_patch_conflict")

type workspacePatchState struct {
	Exists bool
	Mode   uint32
	Data   []byte
}

func (s workspacePatchState) same(other workspacePatchState) bool {
	return s.Exists == other.Exists && s.Mode == other.Mode && bytes.Equal(s.Data, other.Data)
}

type workspacePatchChange struct {
	Path          string
	Before, After workspacePatchState
}

type workspacePatchPlan struct {
	Operation, SessionID string
	Changes              []workspacePatchChange
	Directories          []string
	FilesStarted         int
	DirectoriesStarted   int
	Committed            bool
}

func patchFileState(root *os.Root, name string) (workspacePatchState, os.FileInfo, error) {
	var zero workspacePatchState
	info, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return zero, nil, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxWorkspaceTextPreview {
		return zero, nil, errPatchConflict
	}
	file, err := openAttachmentFile(root, name, maxWorkspaceTextPreview)
	if err != nil {
		return zero, nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !sameWorkspaceVersion(info, opened) {
		return zero, nil, errPatchConflict
	}
	data, err := io.ReadAll(io.LimitReader(file, maxWorkspaceTextPreview+1))
	after, statErr := file.Stat()
	if err != nil || statErr != nil || len(data) > maxWorkspaceTextPreview || !sameWorkspaceVersion(opened, after) {
		return zero, nil, errPatchConflict
	}
	return workspacePatchState{true, uint32(after.Mode().Perm()), data}, after, nil
}

func prepareWorkspacePatch(root *os.Root, session acpSession, operation string, files []workspacePatchFile) (workspacePatchPlan, error) {
	plan := workspacePatchPlan{Operation: operation, SessionID: session.ID}
	touched, parents := map[string]bool{}, map[string]bool{}
	backupBytes := 0
	capture := func(requested string) (string, workspacePatchState, error) {
		name, err := resolveWorkspaceToolPath(session.CWD, requested)
		if err != nil || name == "." || touched[strings.ToLower(name)] {
			return "", workspacePatchState{}, errPatchConflict
		}
		touched[strings.ToLower(name)] = true
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			info, err := root.Lstat(filepath.FromSlash(parent))
			if errors.Is(err, os.ErrNotExist) {
				parents[parent] = true
				continue
			}
			if err != nil || !info.IsDir() {
				return "", workspacePatchState{}, errPatchConflict
			}
		}
		state, _, err := patchFileState(root, filepath.FromSlash(name))
		backupBytes += len(state.Data)
		if backupBytes > maxDesktopSessionBytes {
			return "", workspacePatchState{}, errors.New("workspace_patch_backup_too_large")
		}
		return name, state, err
	}
	for _, file := range files {
		name, before, err := capture(file.Path)
		if err != nil {
			return plan, err
		}
		var after workspacePatchState
		switch file.Kind {
		case "add":
			if before.Exists {
				return plan, errPatchConflict
			}
			mode := uint32(0600)
			if runtime.GOOS == "windows" {
				mode = 0666
			}
			after = workspacePatchState{true, mode, nil}
			if len(file.Added) > 0 {
				after.Data = []byte(strings.Join(file.Added, "\n") + "\n")
			}
		case "delete", "update":
			if !before.Exists {
				return plan, errPatchConflict
			}
			if file.Kind == "update" {
				after = before
				if len(file.Hunks) > 0 {
					data, err := applyWorkspacePatchHunks(string(before.Data), file.Hunks)
					if err != nil {
						return plan, err
					}
					after.Data = []byte(data)
				}
			}
		default:
			return plan, errWorkspacePatch
		}
		if file.Move != "" {
			destination, target, err := capture(file.Move)
			if err != nil || target.Exists {
				return plan, errPatchConflict
			}
			plan.Changes = append(plan.Changes, workspacePatchChange{destination, target, after})
			after = workspacePatchState{}
		}
		plan.Changes = append(plan.Changes, workspacePatchChange{name, before, after})
	}
	for directory := range parents {
		if touched[strings.ToLower(directory)] {
			return plan, errPatchConflict
		}
		plan.Directories = append(plan.Directories, directory)
	}
	slices.Sort(plan.Directories)
	return plan, nil
}

func (p workspacePatchPlan) valid() bool {
	if !validUUID(p.Operation) || !validUUID(p.SessionID) || len(p.Changes) == 0 || p.FilesStarted < 0 ||
		p.FilesStarted > len(p.Changes) || p.DirectoriesStarted < 0 || p.DirectoriesStarted > len(p.Directories) {
		return false
	}
	for _, change := range p.Changes {
		if _, err := workspaceRelative(change.Path); err != nil || change.Path == "." ||
			change.Before.Mode > 0777 || change.After.Mode > 0777 {
			return false
		}
	}
	for _, dir := range p.Directories {
		if _, err := workspaceRelative(dir); err != nil || dir == "." {
			return false
		}
	}
	return true
}

func (w *desktopWorkspaces) savePatchPlan(plan workspacePatchPlan) error {
	if !plan.valid() {
		return errWorkspacePatch
	}
	data, err := json.Marshal(plan)
	if err != nil || len(data) > maxDesktopSessionBytes {
		return errors.New("workspace_patch_backup_too_large")
	}
	return writeSourceState(w.root, workspacePatchJournalFile, data)
}

func workspacePatchTemp(operation string, index int, undo bool) string {
	return fmt.Sprintf(".aibertai-patch-%s-%d-%t.tmp", operation, index, undo)
}

func setWorkspacePatchFile(root *os.Root, change workspacePatchChange, temp string, undo bool) error {
	for directory := path.Dir(change.Path); directory != "."; directory = path.Dir(directory) {
		info, err := root.Lstat(filepath.FromSlash(directory))
		if err != nil || !info.IsDir() {
			return errPatchConflict
		}
	}
	path := filepath.FromSlash(change.Path)
	parent, err := root.OpenRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer parent.Close()
	name := filepath.Base(path)
	before, after := change.Before, change.After
	if undo {
		before, after = after, before
	}
	current, expected, err := patchFileState(parent, name)
	if err != nil {
		return err
	}
	if undo && current.same(after) {
		return nil
	}
	if !current.same(before) {
		return errPatchConflict
	}
	if current.same(after) {
		return nil
	}
	if after.Exists {
		file, err := parent.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer parent.Remove(temp)
		_, writeErr := file.Write(after.Data)
		if err := errors.Join(writeErr, file.Chmod(os.FileMode(after.Mode)), file.Sync(), file.Close()); err != nil {
			return err
		}
	}
	info, err := parent.Lstat(name)
	if expected == nil {
		if !errors.Is(err, os.ErrNotExist) {
			return errPatchConflict
		}
		return parent.Link(temp, name)
	}
	if err != nil || !sameWorkspaceVersion(expected, info) {
		return errPatchConflict
	}
	if after.Exists {
		return parent.Rename(temp, name)
	}
	return parent.Remove(name)
}

func removeWorkspacePatchTemp(root *os.Root, change workspacePatchChange, name string) error {
	err := root.Remove(filepath.Join(filepath.Dir(filepath.FromSlash(change.Path)), name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (w *desktopWorkspaces) finishPatch(root *os.Root, plan workspacePatchPlan) error {
	var result error
	for index := plan.FilesStarted - 1; index >= 0; index-- {
		change := plan.Changes[index]
		temp := workspacePatchTemp(plan.Operation, index, false)
		undo := workspacePatchTemp(plan.Operation, index, true)
		result = errors.Join(result, removeWorkspacePatchTemp(root, change, temp), removeWorkspacePatchTemp(root, change, undo))
		if !plan.Committed {
			err := setWorkspacePatchFile(root, change, undo, true)
			if err != nil {
				result = errors.Join(result, fmt.Errorf("%s: %w", change.Path, err))
			}
		}
	}
	if !plan.Committed {
		for index := plan.DirectoriesStarted - 1; index >= 0; index-- {
			err := root.Remove(filepath.FromSlash(plan.Directories[index]))
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				result = errors.Join(result, err)
			}
		}
	}
	if result != nil {
		return result
	}
	return w.root.Remove(workspacePatchJournalFile)
}

func (w *desktopWorkspaces) applyPatch(ctx context.Context, root *os.Root, plan workspacePatchPlan) error {
	if _, err := w.root.Lstat(workspacePatchJournalFile); !errors.Is(err, os.ErrNotExist) {
		return errors.New("workspace_patch_recovery_required")
	}
	for _, change := range plan.Changes {
		current, _, err := patchFileState(root, filepath.FromSlash(change.Path))
		if err != nil || !current.same(change.Before) {
			return errPatchConflict
		}
	}
	for _, dir := range plan.Directories {
		if _, err := root.Lstat(filepath.FromSlash(dir)); !errors.Is(err, os.ErrNotExist) {
			return errPatchConflict
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := w.savePatchPlan(plan); err != nil {
		return err
	}
	rollback := func(err error) error {
		return errors.Join(err, w.finishPatch(root, plan))
	}
	for index, dir := range plan.Directories {
		next := plan
		next.DirectoriesStarted = index + 1
		if err := w.savePatchPlan(next); err != nil {
			return rollback(err)
		}
		plan = next
		if err := root.Mkdir(filepath.FromSlash(dir), 0755); err != nil {
			plan.DirectoriesStarted = index
			_ = w.savePatchPlan(plan)
			return rollback(err)
		}
	}
	for index, change := range plan.Changes {
		if err := ctx.Err(); err != nil {
			return rollback(err)
		}
		next := plan
		next.FilesStarted = index + 1
		if err := w.savePatchPlan(next); err != nil {
			return rollback(err)
		}
		plan = next
		if err := setWorkspacePatchFile(root, change, workspacePatchTemp(plan.Operation, index, false), false); err != nil {
			plan.FilesStarted = index
			_ = w.savePatchPlan(plan)
			return rollback(err)
		}
	}
	committed := plan
	committed.Committed = true
	if err := w.savePatchPlan(committed); err != nil {
		return rollback(err)
	}
	// A durable commit must not be rolled back if removing temporary state fails.
	if err := w.finishPatch(root, committed); err != nil {
		fmt.Fprintln(os.Stderr, "已提交的工作区补丁等待清理", committed.Operation, err)
	}
	return nil
}

func (h *relaySessionHost) prepareToolWorkspacePatch(row acpSession, operation string, files []workspacePatchFile) (workspacePatchPlan, error) {
	root, err := h.live.workspaces.sessionRoot(row)
	if err != nil {
		return workspacePatchPlan{}, err
	}
	defer root.Close()
	return prepareWorkspacePatch(root, row, operation, files)
}

func (h *relaySessionHost) applyToolWorkspacePatch(ctx context.Context, row acpSession, plan workspacePatchPlan) (string, error) {
	workspace := h.live.workspaces
	workspace.writes.Lock()
	defer workspace.writes.Unlock()
	root, err := workspace.sessionRoot(row)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := workspace.applyPatch(ctx, root, plan); err != nil {
		return "", err
	}
	var result strings.Builder
	for _, change := range plan.Changes {
		if change.After.Exists {
			fmt.Fprintf(&result, "Updated %s\n", change.Path)
		} else {
			fmt.Fprintf(&result, "Removed %s\n", change.Path)
		}
	}
	return result.String(), nil
}

func (w *desktopWorkspaces) recoverPatch(sessions map[string]acpSession) error {
	var plan workspacePatchPlan
	err := readDesktopSessionRecord(w.root, workspacePatchJournalFile, &plan)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	session, ok := sessions[plan.SessionID]
	if err != nil || !ok || !plan.valid() {
		return errors.New("workspace_patch_recovery_invalid")
	}
	root, err := w.sessionRoot(session)
	if err != nil {
		return err
	}
	defer root.Close()
	return w.finishPatch(root, plan)
}
