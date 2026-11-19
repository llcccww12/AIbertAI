package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCloudMigrationPreservesStateAndRefusesDifferentKey(t *testing.T) {
	root := t.TempDir()
	oldState := filepath.Join(root, "legacy-auth")
	if err := os.WriteFile(oldState, []byte("original state"), 0600); err != nil {
		t.Fatal(err)
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(t.TempDir(), "private.seed")
	if err := os.WriteFile(keyPath, private.Seed(), 0600); err != nil {
		t.Fatal(err)
	}
	identity := cloudIdentity{
		Server: "https://private.example", ServerID: "6d2cf4aa-c422-425d-bc5d-b33cffbfab3b",
		UserID: "82b00383-feb2-4c60-8790-299dbc95de13", MachineID: "f401a973-26b2-4aa6-811d-7d0971e5c509", Name: "Existing Cloud",
	}
	const environment = "30d79c77-b463-4ef8-8f49-1c1e769f5237"
	const version = "26.1119.2"
	var out bytes.Buffer
	stage := func(p ed25519.PublicKey, account cloudIdentity) error {
		out.Reset()
		return stageCloudMigration(root, keyPath, base64.RawURLEncoding.EncodeToString(p), environment, 2, version, account, &out)
	}
	if err := stage(make([]byte, ed25519.PublicKeySize), identity); err == nil || out.Len() != 0 {
		t.Fatal("staged a different key")
	}
	if _, err := os.Stat(filepath.Join(root, cloudIdentityFile)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("wrote an identity before checking the original public key")
	}
	for range 2 {
		if err := stage(public, identity); err != nil {
			t.Fatal(err)
		}
	}
	before, err := os.ReadFile(oldState)
	if err != nil || string(before) != "original state" {
		t.Fatalf("legacy state changed: %v", err)
	}
	other := identity
	other.MachineID = "8eb98fb2-a815-4c19-bc0a-b23634b52225"
	if err := stage(public, other); err == nil {
		t.Fatal("replaced an already staged identity")
	}
}
