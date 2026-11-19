package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const relayMaxUpload = 32 << 20
const relayUploadChunk = 1 << 20
const relayUploadTTL = 15 * time.Minute

var errInvalidUpload = errors.New("invalid_upload_request")
var errUploadOperationMissing = errors.New("upload_operation_not_found")

type relayFileRef struct {
	ServerID  string `json:"serverId"`
	UserID    string `json:"userId"`
	MachineID string `json:"machineId"`
	SessionID string `json:"sessionId"`
	Path      string `json:"path"`
}

type relayUploadParams struct {
	SessionID string `json:"sessionId"`
	UploadID  string `json:"uploadId"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	SHA       string `json:"sha256"`
	Purpose   string `json:"purpose,omitempty"`
}

type relayUpload struct {
	relayUploadParams
	Auth      relayAuthorization `json:"-"`
	Offset    int64              `json:"offset"`
	Expires   time.Time          `json:"expiresAt"`
	State     string             `json:"state"`
	Target    map[string]string  `json:"target"`
	Ref       *relayFileRef      `json:"fileRef,omitempty"`
	operation string
}

type relayUploads struct {
	mu            sync.Mutex
	root          *os.Root
	persist       bool
	private       ed25519.PrivateKey
	slots         map[string]*relayUpload
	openHistory   func(relayFileRef) (*os.File, string, error)
	openWorkspace func(relayFileRef) (*os.File, string, error)
}

func openRelayUploads(root *os.Root, key ed25519.PrivateKey, persist bool) *relayUploads {
	return &relayUploads{root: root, private: key, persist: persist, slots: make(map[string]*relayUpload)}
}

func (u *relayUploads) create(p relayUploadParams, auth relayAuthorization) (relayUpload, error) {
	hash, decodeErr := hex.DecodeString(p.SHA)
	if !validUUID(p.SessionID) || !validUUID(p.UploadID) || p.Size < 0 || p.Size > relayMaxUpload || p.Purpose != "" || decodeErr != nil || len(hash) != sha256.Size || p.SHA != strings.ToLower(p.SHA) ||
		p.Name == "" || len([]rune(p.Name)) > 255 || p.Name == "." || p.Name == ".." || strings.ContainsAny(p.Name, "/\\:") || strings.ContainsFunc(p.Name, unicode.IsControl) {
		return relayUpload{}, errInvalidUpload
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	for id, slot := range u.slots {
		if time.Now().Before(slot.Expires) {
			continue
		}
		if slot.State != "completed" {
			if err := u.root.Remove(uploadFilename(id, false)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return relayUpload{}, err
			}
		}
		delete(u.slots, id)
	}
	if prior := u.slots[p.UploadID]; prior != nil {
		if prior.relayUploadParams == p && prior.Auth.same(auth) && time.Now().Before(prior.Expires) {
			return *prior, nil
		}
		return relayUpload{}, errInvalidUpload
	}
	if len(u.slots) >= 16 {
		return relayUpload{}, errors.New("upload_slot_limit_reached")
	}
	if _, err := u.root.Lstat(uploadFilename(p.UploadID, true)); !errors.Is(err, os.ErrNotExist) {
		return relayUpload{}, errInvalidUpload
	}
	file, err := u.root.OpenFile(uploadFilename(p.UploadID, false), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return relayUpload{}, err
	}
	if err := file.Close(); err != nil {
		_ = u.root.Remove(uploadFilename(p.UploadID, false))
		return relayUpload{}, err
	}
	slot := &relayUpload{relayUploadParams: p, Auth: auth, Expires: time.Now().UTC().Add(relayUploadTTL), State: "uploading",
		Target: map[string]string{"serverId": auth.ServerID, "userId": auth.OwnerUserID, "machineId": auth.MachineID}}
	u.slots[p.UploadID] = slot
	return *slot, nil
}

func uploadFilename(id string, completed bool) string {
	suffix := ".partial"
	if completed {
		suffix = ".ready"
	}
	return id + suffix
}

func (u *relayUploads) openCompleted(slot *relayUpload) (*os.File, error) {
	f, err := u.root.Open(uploadFilename(slot.UploadID, true))
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != slot.Size {
		_ = f.Close()
		return nil, errInvalidUpload
	}
	return f, nil
}

// The caller holds u.mu for the duration of a slot operation.
func (u *relayUploads) slot(id, session string, auth relayAuthorization) (*relayUpload, error) {
	slot := u.slots[id]
	if slot == nil || slot.SessionID != session || !slot.Auth.same(auth) || !time.Now().Before(slot.Expires) {
		return nil, errInvalidUpload
	}
	return slot, nil
}

func (u *relayUploads) operation(method, id, session, op string, auth relayAuthorization) (relayUpload, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	slot, err := u.slot(id, session, auth)
	if err != nil {
		return relayUpload{}, err
	}
	switch method {
	case "_monkeyai/uploads/status":
		return *slot, nil
	case "_monkeyai/uploads/cancel":
		if slot.State == "completed" {
			// The employee client releases its upload slot after the message
			// consumes it. Keep the published attachment available for history.
			return *slot, nil
		}
		if err := u.root.Remove(uploadFilename(id, false)); err != nil {
			return relayUpload{}, err
		}
		delete(u.slots, id)
		return *slot, nil
	case "_monkeyai/uploads/complete":
		if !validUUID(op) || slot.Offset != slot.Size {
			return relayUpload{}, errInvalidUpload
		}
		if slot.State == "completed" {
			if op != slot.operation {
				return relayUpload{}, errors.New("upload_operation_conflict")
			}
			return *slot, nil
		}
		for _, prior := range u.slots {
			if prior.operation == op && prior.Auth.same(auth) {
				return relayUpload{}, errors.New("upload_operation_conflict")
			}
		}
		file, err := u.root.Open(uploadFilename(id, false))
		if err != nil {
			return relayUpload{}, err
		}
		digest := sha256.New()
		length, err := io.Copy(digest, io.LimitReader(file, slot.Size+1))
		closeErr := file.Close()
		if err != nil || closeErr != nil || length != slot.Size || hex.EncodeToString(digest.Sum(nil)) != slot.SHA {
			return relayUpload{}, errors.New("upload_hash_mismatch")
		}
		completed := *slot
		completed.State, completed.operation = "completed", op
		completed.Ref = &relayFileRef{ServerID: auth.ServerID, UserID: auth.OwnerUserID, MachineID: auth.MachineID, SessionID: slot.SessionID,
			Path: ".monkeyai/uploads/upload-" + id + "/" + slot.Name}
		if u.persist {
			data, err := json.Marshal(completed)
			if err != nil {
				return relayUpload{}, err
			}
			if err := writeSourceState(u.root, id+".json", data); err != nil {
				return relayUpload{}, err
			}
		}
		if err := u.root.Rename(uploadFilename(id, false), uploadFilename(id, true)); err != nil {
			return relayUpload{}, err
		}
		*slot = completed
		return *slot, nil
	}
	return relayUpload{}, errInvalidUpload
}

func (u *relayUploads) query(operation string, auth relayAuthorization) (any, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if validUUID(operation) {
		for _, slot := range u.slots {
			if slot.operation == operation && slot.State == "completed" && slot.Auth.same(auth) && time.Now().Before(slot.Expires) {
				return map[string]any{"version": 1, "operationId": operation, "method": "_monkeyai/uploads/complete",
					"target": slot.Target, "state": "completed", "sessionId": slot.SessionID, "result": *slot}, nil
			}
		}
	}
	return nil, errUploadOperationMissing
}

func (u *relayUploads) transfer(w http.ResponseWriter, r *http.Request, id string, auth relayAuthorization) {
	if len(r.Header.Values("Upload-Session-ID")) != 1 {
		http.Error(w, "invalid_upload_request", http.StatusBadRequest)
		return
	}
	var chunk []byte
	var offset int64
	if r.Method == http.MethodPut {
		var err error
		offset, err = strconv.ParseInt(r.Header.Get("Upload-Offset"), 10, 64)
		if err != nil || offset < 0 || len(r.Header.Values("Upload-Offset")) != 1 || r.Header.Get("Content-Type") != "application/octet-stream" {
			http.Error(w, "invalid_upload_request", 400)
			return
		}
		chunk, err = io.ReadAll(http.MaxBytesReader(w, r.Body, relayUploadChunk))
		if err != nil || len(chunk) == 0 {
			http.Error(w, "invalid_upload_chunk", 400)
			return
		}
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	slot, err := u.slot(id, r.Header.Get("Upload-Session-ID"), auth)
	if err != nil {
		http.Error(w, "upload_unavailable", 404)
		return
	}
	if r.Method == http.MethodPut {
		if slot.State != "uploading" || offset != slot.Offset || int64(len(chunk)) > slot.Size-slot.Offset {
			u.headers(w, slot)
			w.WriteHeader(http.StatusConflict)
			return
		}
		file, err := u.root.OpenFile(uploadFilename(id, false), os.O_WRONLY, 0600)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		n, err := file.WriteAt(chunk, offset)
		if err == nil && n == len(chunk) {
			err = file.Sync()
		}
		if err != nil || n != len(chunk) {
			_ = file.Truncate(offset)
			_ = file.Close()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = file.Close()
		slot.Offset += int64(n)
	}
	u.headers(w, slot)
	w.WriteHeader(http.StatusNoContent)
}

func (u *relayUploads) headers(w http.ResponseWriter, slot *relayUpload) {
	w.Header().Set("Upload-Offset", strconv.FormatInt(slot.Offset, 10))
	w.Header().Set("Upload-Length", strconv.FormatInt(slot.Size, 10))
	w.Header().Set("Upload-State", slot.State)
}

type relayFileGrant struct {
	Type      string             `json:"type"`
	Version   int                `json:"version"`
	KeyID     string             `json:"key_id"`
	Auth      relayAuthorization `json:"authorization"`
	SessionID string             `json:"session_id"`
	Path      string             `json:"path"`
	Purpose   string             `json:"purpose"`
	Issued    int64              `json:"issued_at"`
	Expires   int64              `json:"expires_at"`
}

func (u *relayUploads) link(issuer, purpose string, ref relayFileRef, access relayFileAccess) (any, error) {
	if purpose == "" {
		purpose = "resource"
	}
	if purpose != "resource" && purpose != "download" {
		return nil, errInvalidUpload
	}
	f, _, err := u.openReferencedFile(ref, access.Auth)
	if err != nil {
		return nil, err
	}
	_ = f.Close()
	seconds := int64(min(30*time.Second, time.Until(access.Deadline)) / time.Second)
	if seconds < 1 {
		return nil, errRelayAccess
	}
	digest := sha256.Sum256(u.private.Public().(ed25519.PublicKey))
	expires := access.ServerTime.Add(time.Duration(seconds) * time.Second)
	grant := relayFileGrant{Type: "monkeyai.file-grant.v1", Version: 1, KeyID: base64.RawURLEncoding.EncodeToString(digest[:]), Auth: access.Auth,
		SessionID: ref.SessionID, Path: ref.Path, Purpose: purpose, Issued: access.ServerTime.UnixMilli(), Expires: expires.UnixMilli()}
	payload, err := json.Marshal(grant)
	if err != nil {
		return nil, err
	}
	encode := base64.RawURLEncoding.EncodeToString
	token := "fg1." + encode(payload) + "." + encode(ed25519.Sign(u.private, append([]byte("monkeyai.file-grant.v1\n"), payload...)))
	parts := strings.Split(ref.Path, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return map[string]any{"fileRef": ref, "expires_in": seconds, "expires_at": expires, "url": issuer + "/api/v1/endpoints/" + ref.MachineID + "/proxy/files/" + token + "/" + strings.Join(parts, "/")}, nil
}

// The caller holds u.mu when resolving a reference.
func (u *relayUploads) completed(ref relayFileRef, auth relayAuthorization) (*relayUpload, error) {
	if ref.ServerID != auth.ServerID || ref.UserID != auth.OwnerUserID || ref.MachineID != auth.MachineID || !validUUID(ref.SessionID) {
		return nil, errInvalidUpload
	}
	if u.persist {
		return u.persistedCompleted(ref)
	}
	for _, slot := range u.slots {
		if slot.Ref != nil && *slot.Ref == ref && slot.Auth.same(auth) && time.Now().Before(slot.Expires) {
			return slot, nil
		}
	}
	return nil, errInvalidUpload
}

func (u *relayUploads) promptFile(ref relayFileRef, session, name string, remaining int64) (*os.File, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	var row *relayUpload
	if u.persist {
		row, _ = u.persistedCompleted(ref)
	} else {
		for _, slot := range u.slots {
			if slot.Ref != nil && *slot.Ref == ref {
				row = slot
				break
			}
		}
	}
	if row == nil || row.SessionID != session || row.Name != name || row.Size > remaining {
		return nil, errInvalidUpload
	}
	return u.openCompleted(row)
}

func (u *relayUploads) persistedCompleted(ref relayFileRef) (*relayUpload, error) {
	path, _ := strings.CutPrefix(ref.Path, ".monkeyai/uploads/upload-")
	id, name, ok := strings.Cut(path, "/")
	if !ok || !validUUID(id) || name == "" || name == path || strings.ContainsAny(name, "/\\") ||
		ref.Path != ".monkeyai/uploads/upload-"+id+"/"+name {
		return nil, errInvalidUpload
	}
	meta := id + ".json"
	info, err := u.root.Lstat(meta)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 4096 {
		return nil, errInvalidUpload
	}
	f, err := u.root.Open(meta)
	if err != nil {
		return nil, errInvalidUpload
	}
	defer f.Close()
	decoder := json.NewDecoder(io.LimitReader(f, 4097))
	var row relayUpload
	if decoder.Decode(&row) != nil || decoder.Decode(new(any)) != io.EOF || row.State != "completed" ||
		row.UploadID != id || row.SessionID != ref.SessionID || row.Name != name || row.Size < 0 ||
		row.Size > relayMaxUpload || row.Offset != row.Size || row.Ref == nil || *row.Ref != ref {
		return nil, errInvalidUpload
	}
	if info, err := u.root.Lstat(uploadFilename(id, true)); err != nil || !info.Mode().IsRegular() || info.Size() != row.Size {
		return nil, errInvalidUpload
	}
	return &row, nil
}

func (u *relayUploads) download(w http.ResponseWriter, r *http.Request, access relayFileAccess) {
	token, path, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/files/"), "/")
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "fg1" || len(token) > 12<<10 {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	decode := base64.RawURLEncoding.Strict().DecodeString
	payload, payloadErr := decode(parts[1])
	sig, sigErr := decode(parts[2])
	var claim relayFileGrant
	if payloadErr != nil || sigErr != nil || !ed25519.Verify(u.private.Public().(ed25519.PublicKey), append([]byte("monkeyai.file-grant.v1\n"), payload...), sig) ||
		json.Unmarshal(payload, &claim) != nil || claim.Type != "monkeyai.file-grant.v1" || claim.Version != 1 || claim.Path != path ||
		!claim.Auth.same(access.Auth) || claim.Expires <= access.ServerTime.UnixMilli() || (claim.Purpose != "download" && claim.Purpose != "resource") {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	auth := access.Auth
	ref := relayFileRef{ServerID: auth.ServerID, UserID: auth.OwnerUserID, MachineID: auth.MachineID, SessionID: claim.SessionID, Path: path}
	file, name, err := u.openReferencedFile(ref, auth)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	if rangeValue := r.Header.Get("Range"); rangeValue != "" && !transferRange.MatchString(rangeValue) {
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, name, time.Time{}, file)
}

func (u *relayUploads) openReferencedFile(ref relayFileRef, auth relayAuthorization) (*os.File, string, error) {
	if ref.ServerID != auth.ServerID || ref.UserID != auth.OwnerUserID || ref.MachineID != auth.MachineID || !validUUID(ref.SessionID) {
		return nil, "", errInvalidUpload
	}
	if u.openHistory != nil && strings.HasPrefix(ref.Path, ".monkeyai/history-files/") {
		return u.openHistory(ref)
	}
	if u.openWorkspace != nil && !strings.HasPrefix(ref.Path, ".monkeyai/uploads/upload-") {
		return u.openWorkspace(ref)
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	slot, err := u.completed(ref, auth)
	if err != nil {
		return nil, "", err
	}
	file, err := u.openCompleted(slot)
	return file, slot.Name, err
}
