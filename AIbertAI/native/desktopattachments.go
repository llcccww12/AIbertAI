package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const desktopAttachmentDir = "source-runtime/attachments"
const maxAttachmentText = 128 << 10

func openDesktopAttachments(profile string) (*os.Root, error) {
	root, err := os.OpenRoot(profile)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	dir := filepath.FromSlash(desktopAttachmentDir)
	if err := root.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return root.OpenRoot(dir)
}

func readDesktopAttachment(root *os.Root, file acpAttachment) (string, error) {
	f, err := openDesktopAttachment(root, file, maxAttachmentText)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxAttachmentText+1))
	if err != nil || len(data) > maxAttachmentText || !isAttachmentText(data) {
		return "", errors.New("attachment_text_unavailable")
	}
	return string(data), nil
}

func openDesktopAttachment(root *os.Root, file acpAttachment, maximum int64) (*os.File, error) {
	if root == nil || file.UploadID != "" || file.Ref != (relayFileRef{}) ||
		file.Name == "" || !filepath.IsLocal(file.Name) || strings.ContainsAny(file.Name, "/\\\x00\r\n") {
		return nil, errInvalidUpload
	}
	path, err := desktopFileURIPath(file.URI)
	if err != nil {
		return nil, err
	}
	relative, err := filepath.Rel(root.Name(), path)
	if err != nil || !filepath.IsLocal(relative) {
		return nil, errInvalidUpload
	}
	id := filepath.Dir(relative)
	if !validUUID(id) || filepath.Base(relative) != file.Name {
		return nil, errInvalidUpload
	}
	info, err := root.Lstat(id)
	if err != nil || !info.IsDir() {
		return nil, errInvalidUpload
	}
	dir, err := root.OpenRoot(id)
	if err != nil {
		return nil, errInvalidUpload
	}
	defer dir.Close()
	return openAttachmentFile(dir, file.Name, maximum)
}

func desktopFileURIPath(uriText string) (string, error) {
	uri, err := url.Parse(uriText)
	if err != nil || uri.Scheme != "file" || uri.Host != "" || uri.User != nil || uri.RawQuery != "" ||
		uri.ForceQuery || uri.Fragment != "" || uri.Opaque != "" {
		return "", errInvalidUpload
	}
	path := uri.Path
	if runtime.GOOS == "windows" && len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	path = filepath.FromSlash(path)
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return "", errInvalidUpload
	}
	return path, nil
}

func (h *relaySessionHost) savedDesktopAttachment(raw json.RawMessage) (any, error) {
	var p struct{ SessionID, URI string }
	if json.Unmarshal(raw, &p) != nil || !validUUID(p.SessionID) || p.URI == "" {
		return nil, errInvalidUpload
	}
	session, err := h.live.load(p.SessionID)
	if err != nil {
		archive, err := readLegacySessionArchive(h.profile, p.SessionID)
		if err != nil {
			return nil, err
		}
		name := legacyAttachmentName(archive, p.URI)
		file, err := openLegacyAttachment(h.profile, archive, p.URI, name)
		if err != nil {
			return nil, err
		}
		return encodeSavedAttachment(file, name)
	}
	for _, turn := range session.Turns {
		for _, file := range turn.Attachments {
			if file.URI == p.URI {
				original, err := openDesktopAttachment(h.local.attachments, file, relayMaxUpload)
				if err != nil {
					return nil, err
				}
				return encodeSavedAttachment(original, file.Name)
			}
		}
	}
	return nil, errors.New("attachment_not_in_session")
}

func encodeSavedAttachment(file *os.File, name string) (map[string]string, error) {
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, relayMaxUpload+1))
	if err != nil || len(data) > relayMaxUpload {
		return nil, errInvalidUpload
	}
	if len(data) <= maxAttachmentText && isAttachmentText(data) {
		return map[string]string{"name": name, "content": string(data)}, nil
	}
	return map[string]string{"name": name, "base64": base64.StdEncoding.EncodeToString(data)}, nil
}

func legacyAttachmentPath(archive legacySessionArchive, uri, name string) (string, error) {
	var meta struct {
		CWD                string
		IsDefaultWorkspace bool
	}
	if json.Unmarshal(archive.Session, &meta) != nil || !meta.IsDefaultWorkspace || !filepath.IsAbs(meta.CWD) {
		return "", errInvalidUpload
	}
	folder := filepath.Base(meta.CWD)
	id, ok := strings.CutPrefix(folder, "chat-")
	if !ok || !validUUID(id) {
		return "", errInvalidUpload
	}
	path, err := desktopFileURIPath(uri)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(meta.CWD, path)
	if err != nil || !filepath.IsLocal(relative) {
		return "", errInvalidUpload
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) != 4 || parts[0] != ".monkeyai" || parts[1] != "uploads" ||
		!strings.HasPrefix(parts[2], "attachment-") {
		return "", errInvalidUpload
	}
	if name == "" || !filepath.IsLocal(name) || filepath.Base(name) != name || strings.ContainsAny(name, "/\\\x00\r\n") {
		return "", errInvalidUpload
	}
	return filepath.Join("workspaces", folder, relative), nil
}

func openLegacyAttachment(profile string, archive legacySessionArchive, uri, name string) (*os.File, error) {
	relative, err := legacyAttachmentPath(archive, uri, name)
	if err != nil {
		return nil, err
	}
	profileRoot, err := os.OpenRoot(profile)
	if err != nil {
		return nil, err
	}
	defer profileRoot.Close()
	parts := strings.Split(filepath.ToSlash(relative), "/")
	workspace := filepath.Join(parts[0], parts[1])
	info, err := profileRoot.Lstat(workspace)
	if err != nil || !info.IsDir() {
		return nil, errInvalidUpload
	}
	root, err := profileRoot.OpenRoot(workspace)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	relative = filepath.Join(parts[2:]...)
	return openAttachmentFile(root, relative, relayMaxUpload)
}

func openAttachmentFile(root *os.Root, relative string, limit int64) (*os.File, error) {
	info, err := root.Lstat(relative)
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errInvalidUpload
	}
	file, err := root.Open(relative)
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err != nil || !os.SameFile(opened, info) || opened.Size() != info.Size() {
		_ = file.Close()
		return nil, errInvalidUpload
	}
	return file, nil
}

func historyFilePath(uri, name string) string {
	if name == "" || len(name) > 255 || !filepath.IsLocal(name) || filepath.Base(name) != name ||
		strings.ContainsAny(name, "/\\\x00\r\n") {
		return ""
	}
	return fmt.Sprintf(".monkeyai/history-files/%x/%s", sha256.Sum256([]byte(uri)), name)
}

func (h *relaySessionHost) openHistoryFile(ref relayFileRef) (*os.File, string, error) {
	if !strings.HasPrefix(ref.Path, ".monkeyai/history-files/") {
		return nil, "", errInvalidUpload
	}
	session, err := h.live.load(ref.SessionID)
	if err == nil {
		for _, turn := range session.Turns {
			for _, file := range turn.Attachments {
				if file.URI == "" || historyFilePath(file.URI, file.Name) != ref.Path {
					continue
				}
				f, err := openDesktopAttachment(h.local.attachments, file, relayMaxUpload)
				return f, file.Name, err
			}
		}
		return nil, "", errInvalidUpload
	}
	archive, err := readLegacySessionArchive(h.profile, ref.SessionID)
	if err != nil {
		return nil, "", err
	}
	for _, part := range legacyAttachmentParts(archive) {
		if historyFilePath(part.URI, part.Name) == ref.Path {
			f, err := openLegacyAttachment(h.profile, archive, part.URI, part.Name)
			return f, part.Name, err
		}
	}
	return nil, "", errInvalidUpload
}

type legacyAttachmentPart struct{ Type, URI, Name string }

func legacyAttachmentParts(archive legacySessionArchive) []legacyAttachmentPart {
	var parts []legacyAttachmentPart
	for _, row := range archive.Replay {
		var item struct {
			Message struct {
				Role  string
				Parts []legacyAttachmentPart
			}
		}
		if json.Unmarshal(row, &item) != nil || item.Message.Role != "user" {
			continue
		}
		for _, part := range item.Message.Parts {
			if part.Type == "file" && part.URI != "" {
				parts = append(parts, part)
			}
		}
	}
	return parts
}

func legacyAttachmentName(archive legacySessionArchive, uri string) string {
	for _, part := range legacyAttachmentParts(archive) {
		if part.URI == uri {
			return part.Name
		}
	}
	return ""
}
