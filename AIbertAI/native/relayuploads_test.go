package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestRelayUploadRejectsWrongDigestAndDifferentConnection(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	store := openRelayUploads(root, nil, false)
	const session = "064983cd-5c67-4af6-870b-6ce7c56a792b"
	const id = "7ad85231-aca4-432a-8299-ae29691cc356"
	const operation = "ded938e0-cd27-42b0-abb8-85bc046d2c86"
	auth := relayAuthorization{ConnectionID: "first", Epoch: 1}
	digest := sha256.Sum256([]byte("right"))
	if _, err := store.create(relayUploadParams{SessionID: session, UploadID: id, Name: "文件.txt", Size: 5, SHA: hex.EncodeToString(digest[:])}, auth); err != nil {
		t.Fatal(err)
	}
	put := func(a relayAuthorization) int {
		req := httptest.NewRequest(http.MethodPut, "/uploads/"+id, strings.NewReader("wrong"))
		req.Header.Set("Upload-Session-ID", session)
		req.Header.Set("Upload-Offset", "0")
		req.Header.Set("Content-Type", "application/octet-stream")
		w := httptest.NewRecorder()
		store.transfer(w, req, id, a)
		return w.Code
	}
	if s := put(relayAuthorization{ConnectionID: "another", Epoch: 1}); s != http.StatusNotFound {
		t.Fatalf("unrelated authorization had access to the upload: %d", s)
	}
	if s := put(auth); s != http.StatusNoContent {
		t.Fatalf("expected written chunk, got %d", s)
	}
	if _, err := store.operation("_monkeyai/uploads/complete", id, session, operation, auth); err == nil || err.Error() != "upload_hash_mismatch" {
		t.Fatalf("accepted bytes with the wrong digest: %v", err)
	}
	if _, err := store.query(operation, auth); err != errUploadOperationMissing {
		t.Fatalf("failed upload appeared as completed: %v", err)
	}
}

func TestCompletedEmployeeUploadSurvivesNewLeaseAndStore(t *testing.T) {
	const session = "064983cd-5c67-4af6-870b-6ce7c56a792b"
	const id = "7ad85231-aca4-432a-8299-ae29691cc356"
	const operation = "ded938e0-cd27-42b0-abb8-85bc046d2c86"
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	auth := relayAuthorization{ServerID: "server", OwnerUserID: "employee", MachineID: "host", ConnectionID: "first", Epoch: 1}
	store := openRelayUploads(root, nil, true)
	digest := sha256.Sum256([]byte("data"))
	p := relayUploadParams{SessionID: session, UploadID: id, Name: "附件.txt", Size: 4, SHA: hex.EncodeToString(digest[:])}
	if _, err := store.create(p, auth); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPut, "/uploads/"+id, strings.NewReader("data"))
	req.Header.Set("Upload-Session-ID", session)
	req.Header.Set("Upload-Offset", "0")
	req.Header.Set("Content-Type", "application/octet-stream")
	w := httptest.NewRecorder()
	store.transfer(w, req, id, auth)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected written chunk, got %d", w.Code)
	}
	result, err := store.operation("_monkeyai/uploads/complete", id, session, operation, auth)
	if err != nil {
		t.Fatal(err)
	}
	store = openRelayUploads(root, nil, true)
	auth.ConnectionID, auth.Epoch = "new-connection", 2
	if _, err := store.completed(*result.Ref, auth); err != nil {
		t.Fatal(err)
	}
	wrongSession := *result.Ref
	wrongSession.SessionID = id
	if _, err := store.completed(wrongSession, auth); err == nil {
		t.Fatal("attachment was exposed in another session")
	}
	auth.OwnerUserID = "other-employee"
	if _, err := store.completed(*result.Ref, auth); err == nil {
		t.Fatal("attachment was exposed to another employee")
	}
}
