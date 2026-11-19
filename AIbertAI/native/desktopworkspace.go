package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
)

const desktopWorkspaceState = "workspace-state.json"

var errProjectFolder = errors.New("project_folder_invalid")

type employeeProject struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CWD       string    `json:"cwd"`
	Expanded  bool      `json:"expanded"`
	Sort      string    `json:"sort,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitzero"`
}

type workspaceState struct {
	Directories    map[string]string `json:"directories"`
	Projects       []employeeProject `json:"projects"`
	LegacyImported bool              `json:"legacyImported,omitempty"`
}

type desktopWorkspaces struct {
	root   *os.Root
	base   string
	legacy string
	mu     sync.Mutex
	writes sync.Mutex
	state  workspaceState
}

func openDesktopWorkspaces(root *os.Root, base, verificationDirectory string) (*desktopWorkspaces, error) {
	if err := root.MkdirAll("workspaces", 0700); err != nil {
		return nil, err
	}
	w := &desktopWorkspaces{root: root, base: base, legacy: verificationDirectory,
		state: workspaceState{Directories: make(map[string]string), Projects: []employeeProject{}}}
	info, err := root.Lstat(desktopWorkspaceState)
	if errors.Is(err, os.ErrNotExist) {
		return w, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > 4<<20 {
		return nil, errors.New("workspace_state_unavailable")
	}
	f, err := root.Open(desktopWorkspaceState)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	dec := json.NewDecoder(io.LimitReader(f, (4<<20)+1))
	if err != nil || !os.SameFile(info, opened) || dec.Decode(&w.state) != nil ||
		dec.Decode(new(any)) != io.EOF || w.state.Directories == nil || w.state.Projects == nil {
		return nil, errors.New("workspace_state_unavailable")
	}
	return w, nil
}

func (w *desktopWorkspaces) persist(state workspaceState) error {
	data, err := json.Marshal(state)
	if err != nil || len(data) > 4<<20 {
		return errors.New("workspace_state_unavailable")
	}
	if err := writeSourceState(w.root, desktopWorkspaceState, data); err != nil {
		return errors.New("workspace_state_unavailable")
	}
	w.state = state
	return nil
}

func (w *desktopWorkspaces) importLegacyProjects(profile string) error {
	if w.state.LegacyImported {
		return nil
	}
	root, err := os.OpenRoot(profile)
	if err != nil {
		return err
	}
	defer root.Close()
	const fileName = "projects.json"
	info, err := root.Lstat(fileName)
	if errors.Is(err, os.ErrNotExist) {
		// A later cutover may still have bookmarks to import.
		return nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > 4<<20 {
		return errors.New("legacy_projects_unavailable")
	}
	f, err := root.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	opened, err := f.Stat()
	var list []struct {
		employeeProject
		Expanded *bool `json:"expanded"`
	}
	decoder := json.NewDecoder(io.LimitReader(f, (4<<20)+1))
	if err != nil || !os.SameFile(info, opened) || decoder.Decode(&list) != nil ||
		decoder.Decode(new(any)) != io.EOF || list == nil {
		return errors.New("legacy_projects_unavailable")
	}
	state := workspaceState{Directories: make(map[string]string), Projects: slices.Clone(w.state.Projects), LegacyImported: true}
	for path, target := range w.state.Directories {
		state.Directories[path] = target
	}
	for _, item := range list {
		row := item.employeeProject
		if !validUUID(row.ID) || !validProjectName(row.Name) || !filepath.IsAbs(row.CWD) || filepath.Clean(row.CWD) != row.CWD || len(row.CWD) > 4096 {
			return errors.New("legacy_projects_unavailable")
		}
		if slices.ContainsFunc(state.Projects, func(p employeeProject) bool { return p.ID == row.ID || p.CWD == row.CWD }) {
			continue
		}
		row.Expanded = item.Expanded == nil || *item.Expanded
		state.Projects = append(state.Projects, row)
		// The old project registry used canonical paths when recording a
		// selection. Preserve that selection without granting symlink targets.
		if state.Directories[row.CWD] == "" {
			state.Directories[row.CWD] = row.CWD
		}
	}
	return w.persist(state)
}

func (w *desktopWorkspaces) authorizeDirectory(raw json.RawMessage) (any, error) {
	var p struct{ CWD string }
	if json.Unmarshal(raw, &p) != nil || !filepath.IsAbs(p.CWD) || filepath.Clean(p.CWD) != p.CWD || len(p.CWD) > 4096 {
		return nil, errProjectFolder
	}
	resolved, err := filepath.EvalSymlinks(p.CWD)
	if err != nil {
		return nil, errProjectFolder
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return nil, errProjectFolder
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.state.Directories[p.CWD] == resolved {
		return map[string]string{"cwd": p.CWD}, nil
	}
	directories := make(map[string]string, len(w.state.Directories)+1)
	for path, target := range w.state.Directories {
		directories[path] = target
	}
	directories[p.CWD] = resolved
	err = w.persist(workspaceState{Directories: directories, Projects: w.state.Projects, LegacyImported: w.state.LegacyImported})
	return map[string]string{"cwd": p.CWD}, err
}

// A directory grant follows the originally selected mount path. A later symlink
// replacement requires a fresh selection before the runtime can open it.
func (w *desktopWorkspaces) approvedRoot(cwd string) (*os.Root, error) {
	w.mu.Lock()
	resolved := w.state.Directories[cwd]
	w.mu.Unlock()
	if resolved == "" {
		return nil, errProjectFolder
	}
	return openWorkspaceDirectory(cwd, resolved)
}

func openWorkspaceDirectory(path, expected string) (*os.Root, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != expected {
		return nil, errProjectFolder
	}
	before, err := os.Stat(expected)
	if err != nil || !before.IsDir() {
		return nil, errProjectFolder
	}
	root, err := os.OpenRoot(expected)
	if err != nil {
		return nil, errProjectFolder
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(before, opened) {
		_ = root.Close()
		return nil, errProjectFolder
	}
	return root, nil
}

func (w *desktopWorkspaces) defaultPath(id string) string {
	return filepath.Join(w.base, id)
}

func (w *desktopWorkspaces) choose(cwd, projectID string) (string, string, error) {
	if projectID == "" && (cwd == "" || cwd == w.base || w.legacy != "" && cwd == w.legacy) {
		return "", "", nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, row := range w.state.Projects {
		if projectID != "" && row.ID == projectID && (cwd == "" || cwd == row.CWD) ||
			projectID == "" && row.CWD == cwd {
			return row.CWD, row.ID, nil
		}
	}
	if projectID == "" && w.state.Directories[cwd] != "" {
		return cwd, "", nil
	}
	return "", "", errProjectFolder
}

func (w *desktopWorkspaces) allocate(id string) error {
	return w.root.MkdirAll(filepath.Join("workspaces", id), 0700)
}

func (w *desktopWorkspaces) sessionRoot(row acpSession) (*os.Root, error) {
	if row.Workspace == "default" && row.CWD == w.defaultPath(row.ID) {
		name := filepath.Join("workspaces", row.ID)
		for _, path := range []string{"workspaces", name} {
			info, err := w.root.Lstat(path)
			if err != nil || !info.IsDir() {
				return nil, errProjectFolder
			}
		}
		return w.root.OpenRoot(name)
	}
	if row.Workspace == "selected" {
		return w.approvedRoot(row.CWD)
	}
	return nil, errProjectFolder
}

func validProjectName(name string) bool {
	return strings.TrimSpace(name) == name && len([]rune(name)) >= 1 &&
		len([]rune(name)) <= 80 && !strings.ContainsFunc(name, unicode.IsControl)
}

func (w *desktopWorkspaces) projects(method string, raw json.RawMessage) (any, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	current := w.state.Projects
	if method == "_monkeyai/projects/list" {
		return slices.Clone(current), nil
	}
	var p struct {
		ID       string
		Name     *string
		CWD      *string
		Expanded *bool
		Sort     *string
		BeforeID *string
	}
	if json.Unmarshal(raw, &p) != nil {
		return nil, errors.New("invalid_request")
	}
	items := slices.Clone(current)
	idx := slices.IndexFunc(items, func(row employeeProject) bool { return row.ID == p.ID })
	if method == "_monkeyai/projects/create" {
		id, err := newUUID()
		if err != nil {
			return nil, err
		}
		if p.Name == nil || p.CWD == nil {
			return nil, errors.New("invalid_request")
		}
		items = append([]employeeProject{{ID: id, Expanded: true, CreatedAt: time.Now().UTC()}}, items...)
		idx = 0
	}
	if idx < 0 {
		return nil, errors.New("project_not_found")
	}
	row := items[idx]
	switch method {
	case "_monkeyai/projects/create", "_monkeyai/projects/update":
		if p.Name != nil {
			if !validProjectName(*p.Name) {
				return nil, errors.New("project_name_invalid")
			}
			row.Name = *p.Name
		}
		if p.CWD != nil && *p.CWD != row.CWD {
			allowed := w.state.Directories[*p.CWD]
			if allowed == "" {
				return nil, errProjectFolder
			}
			if slices.ContainsFunc(items, func(item employeeProject) bool {
				return item.ID != row.ID && w.state.Directories[item.CWD] == allowed
			}) {
				return nil, errors.New("project_folder_exists")
			}
			row.CWD = *p.CWD
		}
		if p.Expanded != nil {
			row.Expanded = *p.Expanded
		}
		if p.Sort != nil {
			if len(*p.Sort) > 32 || strings.ContainsFunc(*p.Sort, unicode.IsControl) {
				return nil, errors.New("invalid_request")
			}
			row.Sort = *p.Sort
		}
		items[idx] = row
	case "_monkeyai/projects/remove":
		items = slices.Delete(items, idx, idx+1)
	case "_monkeyai/projects/move":
		if p.BeforeID != nil && *p.BeforeID == p.ID {
			return items, nil
		}
		items = slices.Delete(items, idx, idx+1)
		at := len(items)
		if p.BeforeID != nil {
			at = slices.IndexFunc(items, func(item employeeProject) bool { return item.ID == *p.BeforeID })
			if at < 0 {
				return nil, errors.New("project_not_found")
			}
		}
		items = slices.Insert(items, at, row)
	}
	err := w.persist(workspaceState{Directories: w.state.Directories, Projects: items, LegacyImported: w.state.LegacyImported})
	if method == "_monkeyai/projects/create" || method == "_monkeyai/projects/update" {
		return row, err
	}
	return slices.Clone(items), err
}
