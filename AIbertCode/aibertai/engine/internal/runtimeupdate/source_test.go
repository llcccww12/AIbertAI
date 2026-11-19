package runtimeupdate

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aibertai.local/engine/internal/control"
)

func TestLegacyRuntimePackageIsNeverExecuted(t *testing.T) {
	release, _ := packageFixture("26.1105.2")
	release.EngineRevision = strings.Repeat("b", 40)
	release.Files["monkeyai-runtime"] = release.Files["aibertai-native"]
	delete(release.Files, "aibertai-native")

	root := t.TempDir()
	if err := os.Chmod(root, 0755); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(root, directory)
	if err := os.Mkdir(folder, 0755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(release)
	if err := os.WriteFile(filepath.Join(folder, "current.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	manager := New(root, nil)
	current, err := manager.Current()
	if err != nil || current.Binary != "" {
		t.Fatalf("legacy release was selected: %+v, %v", current, err)
	}
	if err := manager.Update(t.Context(), release); err == nil || !strings.Contains(err.Error(), "AIbertAI") {
		t.Fatalf("legacy release was accepted: %v", err)
	}
}

func TestAIbertNativeReleaseInstallsWithoutLegacyEngine(t *testing.T) {
	release, data := packageFixture("26.1119.1")
	const binary = "aibertai-native"
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests++
		if req.URL.Path != "/api/host/v1/runtime-releases/"+release.Version+"/"+release.Architecture+"/"+binary {
			t.Errorf("unexpected runtime file: %s", req.URL.Path)
			http.NotFound(w, req)
			return
		}
		_, _ = w.Write(data[binary])
	}))
	defer server.Close()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	client, err := control.New(server.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Chmod(root, 0755); err != nil {
		t.Fatal(err)
	}
	m := New(root, client)
	for range 2 {
		if err := m.Update(t.Context(), release); err != nil {
			t.Fatal(err)
		}
	}
	files, err := m.Current()
	if err != nil || filepath.Base(files.Binary) != binary || requests != 1 {
		t.Fatalf("expected one checked source artifact with no external engine: %+v, downloads=%d, %v", files, requests, err)
	}
}
