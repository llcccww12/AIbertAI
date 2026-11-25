package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestDSHRealResourceInventory(t *testing.T) {
	root, digest := os.Getenv("DSH_TEST_RESOURCE_ROOT"), os.Getenv("DSH_TEST_RESOURCE_SHA256")
	if root == "" || digest == "" {
		t.Skip("isolated DSH resource not configured")
	}
	started := time.Now()
	node, entry, err := verifyDSHResources(root, digest, runtime.GOOS+"-"+runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}
	nodeName := "node"
	if runtime.GOOS == "windows" {
		nodeName = "node.exe"
	}
	if node != filepath.Join(root, "bin", nodeName) || entry != filepath.Join(root, filepath.FromSlash(dshEntry)) {
		t.Fatalf("unexpected resource paths: %s %s", node, entry)
	}
	t.Logf("verified real resource inventory in %s", time.Since(started))
}

func TestDSHResourcePatchPinMatchesSource(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "dsh-integration", "patches", "experimental-invocation-id.patch"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != dshPatchSHA256 {
		t.Fatal("native resource pin does not match SDK patch")
	}
}

func TestDSHResourcesRequireTrustedInventory(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"bin/node", dshEntry} {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	hash := func(b []byte) string { v := sha256.Sum256(b); return hex.EncodeToString(v[:]) }
	m := dshManifest{Version: 1, SourceCommit: dshSourceCommit, SourceArchive: dshSourceArchiveSHA256, Patch: dshPatchSHA256, Lockfile: hash([]byte("lock")), NodeVersion: "v25.9.0", Target: "darwin-arm64", Node: "bin/node", Entry: dshEntry, Files: []dshManifestFile{{Path: "bin/node", Type: "file", SHA256: hash([]byte("bin/node"))}, {Path: dshEntry, Type: "file", SHA256: hash([]byte(dshEntry))}}}
	data, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	trusted := hash(data)
	if _, _, err := verifyDSHResources(root, trusted, m.Target); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 4)
	for i := 0; i < cap(results); i++ {
		go func() {
			_, _, err := verifyDSHResources(root, trusted, m.Target)
			results <- err
		}()
	}
	for i := 0; i < cap(results); i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	live := &acpLive{dir: root, cwd: root, operations: map[string]string{}, sessions: map[string]acpSession{}}
	profile := t.TempDir()
	if err := configureDSHResources(live, profile, "server", "machine", root, trusted, m.Target); err != nil {
		t.Fatal(err)
	}
	id, err := live.createInWorkspace("11111111-1111-4111-8111-111111111111", root, "", "22222222-2222-4222-8222-222222222222", expertSelection{}, nil, "", desktopMediaModels{})
	if err != nil {
		t.Fatal(err)
	}
	if live.sessions[id].Engine != "dsh" || live.sessions[id].DSHSessionID != id {
		t.Fatal("new session not assigned DSH")
	}
	// Platform session JSON is intentionally outside the resource inventory.
	if err = os.Remove(filepath.Join(root, id+".json")); err != nil {
		t.Fatal(err)
	}

	if _, _, err := verifyDSHResources(root, hash([]byte("other")), m.Target); err == nil {
		t.Fatal("accepted untrusted manifest")
	}
	if _, _, err := verifyDSHResources(root, trusted, "linux-amd64"); err == nil {
		t.Fatal("accepted wrong target")
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "node"), []byte("changed"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := verifyDSHResources(root, trusted, m.Target); err == nil {
		t.Fatal("accepted modified runtime")
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "node"), []byte("bin/node"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "extra"), []byte("extra"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := verifyDSHResources(root, trusted, m.Target); err == nil {
		t.Fatal("accepted unlisted file")
	}
}
func TestDSHResourcesRejectEscapingSymlink(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "secret"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	hash := func(b []byte) string { v := sha256.Sum256(b); return hex.EncodeToString(v[:]) }
	for _, name := range []string{"bin/node", dshEntry} {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("../secret", filepath.Join(root, "escape")); err != nil {
		t.Skip(err)
	}
	m := dshManifest{Version: 1, SourceCommit: dshSourceCommit, SourceArchive: dshSourceArchiveSHA256, Patch: dshPatchSHA256, Lockfile: dshPatchSHA256, NodeVersion: "v25.9.0", Target: "darwin-arm64", Node: "bin/node", Entry: dshEntry, Files: []dshManifestFile{{Path: "bin/node", Type: "file", SHA256: hash([]byte("bin/node"))}, {Path: dshEntry, Type: "file", SHA256: hash([]byte(dshEntry))}, {Path: "escape", Type: "symlink", Target: "../secret"}}}
	data, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := verifyDSHResources(root, hash(data), m.Target); err == nil {
		t.Fatal("accepted escaping symlink")
	}
}

func TestDSHResourceStampSkipsRehashUntilFilesChange(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string, mtime time.Time) {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mtime, mtime); err != nil {
			t.Fatal(err)
		}
	}
	at := time.Unix(1700000000, 0)
	write("bin/node", "bin/node", at)
	write(dshEntry, dshEntry, at)
	hash := func(b []byte) string { v := sha256.Sum256(b); return hex.EncodeToString(v[:]) }
	m := dshManifest{Version: 1, SourceCommit: dshSourceCommit, SourceArchive: dshSourceArchiveSHA256, Patch: dshPatchSHA256, Lockfile: hash([]byte("lock")), NodeVersion: "v25.9.0", Target: "darwin-arm64", Node: "bin/node", Entry: dshEntry, Files: []dshManifestFile{{Path: "bin/node", Type: "file", SHA256: hash([]byte("bin/node"))}, {Path: dshEntry, Type: "file", SHA256: hash([]byte(dshEntry))}}}
	data, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	trusted := hash(data)
	stamp := filepath.Join(t.TempDir(), "dsh-resources-verified")
	if _, _, err := verifyDSHResourcesWithStamp(root, trusted, m.Target, stamp); err != nil {
		t.Fatal(err)
	}
	if readDSHVerifiedStamp(stamp) == "" {
		t.Fatal("full verification left no stamp")
	}
	// Same size and mtime: the stamp is trusted and the content is not re-read.
	write("bin/node", "bin/nodX", at)
	if _, _, err := verifyDSHResourcesWithStamp(root, trusted, m.Target, stamp); err != nil {
		t.Fatal(err)
	}
	// A changed mtime forces a full hash, which catches the modification.
	write("bin/node", "bin/nodX", at.Add(time.Second))
	if _, _, err := verifyDSHResourcesWithStamp(root, trusted, m.Target, stamp); err == nil {
		t.Fatal("accepted modified runtime after mtime change")
	}
	write("bin/node", "bin/node", at.Add(2*time.Second))
	if _, _, err := verifyDSHResourcesWithStamp(root, trusted, m.Target, stamp); err != nil {
		t.Fatal(err)
	}
	if _, _, err := verifyDSHResourcesWithStamp(root, hash([]byte("other")), m.Target, stamp); err == nil {
		t.Fatal("stamp bypassed the trusted manifest digest")
	}
}
