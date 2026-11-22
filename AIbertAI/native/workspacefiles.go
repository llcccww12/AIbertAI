package main

import (
	"cmp"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const localWorkspaceFileRoute = "/_aibertai/workspace/file"
const maxWorkspaceTextPreview = 1 << 20

var errWorkspacePath = errors.New("invalid_path")

func workspaceRelative(value string) (string, error) {
	if value == "" || len(value) > 4096 || path.Clean(value) != value ||
		strings.ContainsAny(value, "\\:") || strings.ContainsFunc(value, unicode.IsControl) {
		return "", errWorkspacePath
	}
	relative := filepath.FromSlash(canonicalWorkspaceReference(value))
	if !filepath.IsLocal(relative) {
		return "", errWorkspacePath
	}
	return relative, nil
}

func readWorkspaceText(root *os.Root, requested string) (string, error) {
	relative, err := workspaceRelative(requested)
	if err != nil || relative == "." {
		return "", errWorkspacePath
	}
	file, err := openAttachmentFile(root, relative, maxToolText)
	if err != nil {
		return "", errors.New("tool_file_unavailable")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxToolText+1))
	if err != nil || len(data) > maxToolText || !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
		return "", errors.New("tool_file_unavailable")
	}
	return string(data), nil
}

func (h *relaySessionHost) writeToolWorkspaceFile(ctx context.Context, row acpSession, path, content string) (string, error) {
	workspaces := h.live.workspaces
	workspaces.writes.Lock()
	defer workspaces.writes.Unlock()
	root, err := workspaces.sessionRoot(row)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := writeWorkspaceText(ctx, root, path, content); err != nil {
		return "", err
	}
	return fmt.Sprintf("Wrote %d bytes to %s", len(content), path), nil
}

func writeWorkspaceText(ctx context.Context, root *os.Root, requested, content string) error {
	return writeWorkspaceTextIfUnchanged(ctx, root, requested, content, nil)
}

func writeWorkspaceTextIfUnchanged(ctx context.Context, root *os.Root, requested, content string, expected os.FileInfo) error {
	return writeWorkspaceTextIfUnchangedLimit(ctx, root, requested, content, expected, maxToolText)
}
func writeWorkspaceTextIfUnchangedLimit(ctx context.Context, root *os.Root, requested, content string, expected os.FileInfo, limit int) error {
	relative, err := workspaceRelative(requested)
	if err != nil || relative == "." || !utf8.ValidString(content) || len(content) > limit || strings.ContainsRune(content, 0) {
		return errWorkspacePath
	}
	parent, err := root.OpenRoot(filepath.Dir(relative))
	if err != nil {
		return errWorkspacePath
	}
	defer parent.Close()
	name := filepath.Base(relative)
	before, err := parent.Lstat(name)
	if err == nil && !before.Mode().IsRegular() || err != nil && !errors.Is(err, os.ErrNotExist) {
		return errWorkspacePath
	}
	if expected != nil && (before == nil || !sameWorkspaceVersion(expected, before)) {
		return errors.New("workspace_file_changed")
	}
	nonce, err := newUUID()
	if err != nil {
		return errors.New("workspace_write_failed")
	}
	temp := ".aibertai-" + nonce + ".tmp"
	file, err := parent.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return errors.New("workspace_write_failed")
	}
	defer parent.Remove(temp)
	_, writeErr := io.WriteString(file, content)
	if writeErr == nil && before != nil {
		writeErr = file.Chmod(before.Mode().Perm())
	}
	if err := errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		return errors.New("workspace_write_failed")
	}
	after, err := parent.Lstat(name)
	if before == nil && !errors.Is(err, os.ErrNotExist) || before != nil &&
		(err != nil || !sameWorkspaceVersion(before, after)) {
		return errors.New("workspace_file_changed")
	}
	if ctx.Err() != nil {
		return errors.New("workspace_write_cancelled")
	}
	if err := parent.Rename(temp, name); err != nil {
		return errors.New("workspace_write_failed")
	}
	return nil
}

func sameWorkspaceVersion(a, b os.FileInfo) bool {
	return os.SameFile(a, b) && a.ModTime().Equal(b.ModTime()) && a.Size() == b.Size()
}

func (h *relaySessionHost) openWorkspaceFile(ref relayFileRef) (*os.File, string, error) {
	relative, err := workspaceRelative(ref.Path)
	if err != nil || relative == "." {
		return nil, "", errWorkspacePath
	}
	root, _, err := h.workspaceRoot(ref.SessionID)
	if err != nil {
		return nil, "", err
	}
	defer root.Close()
	file, err := openAttachmentFile(root, relative, 1<<40)
	return file, filepath.Base(relative), err
}

func (h *relaySessionHost) workspaceRoot(id string) (*os.Root, acpSession, error) {
	row, err := h.live.load(id)
	if err == nil {
		root, err := h.live.workspaces.sessionRoot(row)
		return root, row, err
	}
	metadata, err := h.listSessionMetadata(id)
	if err != nil {
		return nil, row, err
	}
	var legacy struct {
		CWD, ProjectID     string
		IsDefaultWorkspace bool
	}
	if json.Unmarshal(metadata, &legacy) != nil || !filepath.IsAbs(legacy.CWD) {
		return nil, row, errProjectFolder
	}
	row.ID, row.CWD, row.ProjectID = id, legacy.CWD, legacy.ProjectID
	if !legacy.IsDefaultWorkspace {
		row.Workspace = "selected"
		root, err := prepareWorkspaceBrand(h.live.workspaces.approvedRoot(row.CWD))
		return root, row, err
	}
	folder := filepath.Base(row.CWD)
	key, ok := strings.CutPrefix(folder, "chat-")
	if !ok || !validUUID(key) {
		return nil, row, errProjectFolder
	}
	profile, err := os.OpenRoot(h.profile)
	if err != nil {
		return nil, row, err
	}
	defer profile.Close()
	relative := filepath.Join("workspaces", folder)
	for _, name := range []string{"workspaces", relative} {
		info, err := profile.Lstat(name)
		if err != nil || !info.IsDir() {
			return nil, row, errProjectFolder
		}
	}
	root, err := prepareWorkspaceBrand(profile.OpenRoot(relative))
	row.Workspace, row.CWD = "default", filepath.Join(h.profile, relative)
	return root, row, err
}

type workspaceRequest struct {
	TextPage  *workspaceTextCursor
	SessionID string
	Path      string
	Purpose   string
	Offset    int
	TTL       int64
	Paths     []string
}

func (h *relaySessionHost) workspaceQuery(ctx context.Context, method string, raw json.RawMessage) (any, error) {
	var p workspaceRequest
	if json.Unmarshal(raw, &p) != nil || !validUUID(p.SessionID) || p.Offset < 0 || p.Offset > 1000000 {
		return nil, errors.New("invalid_request")
	}
	if method == "_monkeyai/workspace/read" && strings.HasPrefix(p.Path, "file:") && h.local != nil {
		return h.previewSessionAttachment(p.SessionID, p.Path)
	}
	root, row, err := h.workspaceRoot(p.SessionID)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	if method == "_monkeyai/workspace/changes" || method == "_monkeyai/workspace/diff" {
		return workspaceGit(ctx, root, method, p.Path)
	}
	if method == "_monkeyai/session/workspace" {
		meta := row.workspaceMetadata()
		if design := workspaceDesign(root, row.ID); design != nil {
			meta["capabilities"] = map[string]json.RawMessage{"design": design}
		}
		return meta, nil
	}
	if method == "_monkeyai/workspace/resolve" {
		return h.resolveWorkspacePaths(root, row, p.Paths)
	}
	if p.Path == "" && method == "_monkeyai/workspace/list" {
		p.Path = "."
	}
	relative, err := workspaceRelative(p.Path)
	if err != nil {
		return nil, err
	}
	if method == "_monkeyai/workspace/list" {
		return listWorkspaceDirectory(root, relative, filepath.ToSlash(relative), p.Offset)
	}
	if relative == "." {
		return nil, errWorkspacePath
	}
	info, err := root.Lstat(relative)
	if err != nil || !info.Mode().IsRegular() {
		if method == "_monkeyai/workspace/read" {
			return map[string]any{"path": p.Path, "kind": "unavailable", "reason": "missing"}, nil
		}
		return nil, errWorkspacePath
	}
	switch method {
	case "_monkeyai/workspace/stat":
		return workspaceFileStat{Exists: true, Size: info.Size(), MtimeMS: info.ModTime().UnixMilli()}, nil
	case "_monkeyai/workspace/read":
		if p.TextPage != nil && (info.Size() > maxWorkspaceTextPreview || p.TextPage.Offset != 0) {
			return previewWorkspaceTextPage(root, relative, p.Path, *p.TextPage)
		}
		return previewWorkspaceFile(root, relative, p.Path, info.Size())
	case "_monkeyai/workspace/link":
		if p.Purpose != "" && p.Purpose != "resource" && p.Purpose != "download" || p.TTL < 0 || p.TTL > int64(24*time.Hour/time.Millisecond) {
			return nil, errors.New("invalid_request")
		}
		if p.TTL == 0 {
			p.TTL = int64(12 * time.Hour / time.Millisecond)
		}
		return h.issueLocalWorkspaceLink(p)
	}
	return nil, errors.New("method_not_found")
}

func (h *relaySessionHost) resolveWorkspacePaths(root *os.Root, row acpSession, paths []string) (any, error) {
	if len(paths) == 0 || len(paths) > 100 {
		return nil, errors.New("invalid_paths")
	}
	items := make([]map[string]any, 0, len(paths))
	for _, input := range paths {
		item := map[string]any{"input": input, "status": "invalid"}
		items = append(items, item)
		value := input
		if filepath.IsAbs(input) {
			rel, err := filepath.Rel(row.CWD, input)
			if err != nil || !filepath.IsLocal(rel) {
				item["status"] = "outside"
				continue
			}
			value = filepath.ToSlash(rel)
		}
		relative, err := workspaceRelative(value)
		if err != nil {
			continue
		}
		value = filepath.ToSlash(relative)
		info, err := root.Lstat(relative)
		if err != nil {
			item["status"] = "missing"
			continue
		}
		switch {
		case info.IsDir():
			item["kind"] = "directory"
		case info.Mode().IsRegular():
			item["kind"] = "file"
			if h.access != nil {
				item["fileRef"] = relayFileRef{ServerID: h.access.ready.ServerID, UserID: h.access.owner,
					MachineID: h.access.ready.MachineID, SessionID: row.ID, Path: value}
			}
		default:
			item["status"] = "unsupported"
			continue
		}
		item["status"], item["path"] = "resolved", value
	}
	return map[string]any{"sessionId": row.ID, "items": items}, nil
}

func listWorkspaceDirectory(root *os.Root, relative, requested string, offset int) (any, error) {
	dir, err := root.Open(relative)
	if err != nil {
		return nil, errWorkspacePath
	}
	defer dir.Close()
	info, err := dir.Stat()
	if err != nil || !info.IsDir() {
		return nil, errWorkspacePath
	}
	entries, err := dir.ReadDir(100001)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, errWorkspacePath
	}
	if len(entries) > 100000 {
		return nil, errors.New("workspace_listing_too_large")
	}
	slices.SortFunc(entries, func(a, b os.DirEntry) int {
		if a.IsDir() != b.IsDir() {
			if a.IsDir() {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.Name(), b.Name())
	})
	items := make([]map[string]any, 0, 200)
	start := min(offset, len(entries))
	stop := min(start+200, len(entries))
	for _, item := range entries[start:stop] {
		relativePath := path.Join(requested, item.Name())
		if _, err := workspaceRelative(relativePath); err != nil {
			continue
		}
		kind := "file"
		switch {
		case item.IsDir():
			kind = "directory"
		case item.Type()&os.ModeSymlink != 0:
			kind = "symlink"
		case !item.Type().IsRegular():
			continue
		}
		items = append(items, map[string]any{"kind": kind, "name": item.Name(), "path": relativePath})
	}
	var next any
	if stop < len(entries) {
		next = stop
	}
	return map[string]any{"path": requested, "entries": items, "nextOffset": next}, nil
}

func previewWorkspaceFile(root *os.Root, relative, requested string, size int64) (any, error) {
	unavailable := func(reason string) (any, error) {
		return map[string]any{"path": requested, "kind": "unavailable", "reason": reason}, nil
	}
	if size > maxWorkspaceTextPreview {
		return unavailable("too_large")
	}
	file, err := openAttachmentFile(root, relative, maxWorkspaceTextPreview)
	if err != nil {
		return unavailable("missing")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxWorkspaceTextPreview+1))
	if err != nil || len(data) > maxWorkspaceTextPreview {
		return unavailable("too_large")
	}
	switch mime := http.DetectContentType(data); mime {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/bmp":
		return map[string]any{"path": requested, "kind": "image", "imageUrl": "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)}, nil
	}
	if !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
		return unavailable("binary")
	}
	kind := "text"
	switch strings.ToLower(filepath.Ext(relative)) {
	case ".md", ".markdown":
		kind = "markdown"
	case ".html", ".htm":
		kind = "html"
	}
	return map[string]any{"path": requested, "kind": kind, "content": string(data)}, nil
}

type localWorkspaceGrant struct {
	Session, Path, Purpose string
	Expires                int64
}

func (h *relaySessionHost) workspaceGrantMAC(payload string) []byte {
	mac := hmac.New(sha256.New, []byte(h.local.protocol))
	mac.Write([]byte("aibertai/workspace-file/v1\x00"))
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}

func (h *relaySessionHost) issueLocalWorkspaceLink(p workspaceRequest) (any, error) {
	expires := time.Now().Add(time.Duration(p.TTL) * time.Millisecond)
	raw, err := json.Marshal(localWorkspaceGrant{Session: p.SessionID, Path: p.Path, Purpose: p.Purpose, Expires: expires.UnixMilli()})
	if err != nil {
		return nil, err
	}
	encode := base64.RawURLEncoding.EncodeToString
	payload := encode(raw)
	token := payload + "." + encode(h.workspaceGrantMAC(payload))
	return map[string]any{"url": "http://" + h.local.address + localWorkspaceFileRoute + "?grant=" + token, "expires_at": expires}, nil
}

func (h *relaySessionHost) localWorkspaceFile(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Values("Origin")
	if r.Host != h.local.address || len(origin) > 1 || len(origin) == 1 && origin[0] != h.local.origin ||
		r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	token := r.URL.Query().Get("grant")
	payload, signature, ok := strings.Cut(token, ".")
	if !ok || len(token) > 32768 {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	decode := base64.RawURLEncoding.Strict().DecodeString
	sig, err := decode(signature)
	data, decodeErr := decode(payload)
	var grant localWorkspaceGrant
	if err != nil || decodeErr != nil || !hmac.Equal(sig, h.workspaceGrantMAC(payload)) ||
		json.Unmarshal(data, &grant) != nil || grant.Expires <= time.Now().UnixMilli() || !validUUID(grant.Session) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	f, name, err := h.openWorkspaceFile(relayFileRef{SessionID: grant.Session, Path: grant.Path})
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/octet-stream")
	if grant.Purpose == "download" {
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	}
	if len(origin) == 1 {
		w.Header().Set("Access-Control-Allow-Origin", origin[0])
		w.Header().Set("Vary", "Origin")
	}
	// Use positioned reads so the response does not depend on the platform's
	// file-to-socket fast path. Range requests still share ServeContent's checks.
	http.ServeContent(w, r, name, time.Time{}, io.NewSectionReader(f, 0, info.Size()))
}
