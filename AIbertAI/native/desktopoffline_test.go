package main

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOfflineDesktopUsesOnlyTheMatchingAccountStore(t *testing.T) {
	profile := t.TempDir()
	const server = "11111111-1111-4111-8111-111111111111"
	const user = "22222222-2222-4222-8222-222222222222"
	const origin = "https://employee.example.test"
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	device, err := desktopRelayDevice(key, origin, server, user)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(profile, "source-runtime", server, device.machine, "sessions"), 0700); err != nil {
		t.Fatal(err)
	}
	want := offlineDesktopIdentity{server, device.machine}
	if got := findOfflineDesktopIdentity(profile, key, origin, user); got != want {
		t.Fatalf("offline store = %+v, want %+v", got, want)
	}
	if got := findOfflineDesktopIdentity(profile, key, "https://another.example.test", user); got.serverID != "" {
		t.Fatalf("the previous server's files became visible: %+v", got)
	}
	for _, method := range []string{"session/new", "session/prompt", "_monkeyai/session/queue", "_monkeyai/session/goal", "_monkeyai/workspace/write"} {
		if offlineDesktopMethod(method) {
			t.Fatalf("offline connection admits %s", method)
		}
	}
}

func TestReauthenticationReloadsRejectedGrant(t *testing.T) {
	const user = "11111111-1111-4111-8111-111111111111"
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	store, err := openSourceGrantStore(t.TempDir(), key, "https://employee.example.test", user)
	if err != nil {
		t.Fatal(err)
	}
	defer store.root.Close()
	initial := authorization{Access: "old", Refresh: "old-refresh", Grant: user, TTL: 3600, Scope: sourceGrantScopes}
	previous, err := store.replace(initial)
	if err != nil {
		t.Fatal(err)
	}
	source := &desktopSourceAuth{store: store, grant: previous, rejected: true}
	if err := source.reload(); !errors.Is(err, errSourceGrantRejected) {
		t.Fatalf("unchanged revoked credential became usable: %v", err)
	}
	initial.Access, initial.Refresh = "new", "new-refresh"
	if _, err := store.replace(initial); err != nil {
		t.Fatal(err)
	}
	if err := source.reload(); err != nil {
		t.Fatal(err)
	}
	if auth, err := source.access(t.Context()); err != nil || auth != initial || time.Until(source.grant.Expires) <= 0 {
		t.Fatalf("running source did not use the new credential: %v", err)
	}
}
