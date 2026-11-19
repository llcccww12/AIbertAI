package enginehost

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyRelease(t *testing.T) {
	root := t.TempDir()
	write := func(path string, data []byte) {
		t.Helper()
		path = filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("latest.json", []byte(`{"version":"v1"}`))
	for _, arch := range []string{"amd64", "arm64"} {
		prefix := filepath.Join("v1", arch)
		data := []byte(arch)
		digest := fmt.Sprintf("%x", sha256.Sum256(data))
		raw, err := json.Marshal(manifest{Version: "v1", Architecture: arch,
			Files: map[string]string{"installer": digest, "aibertai-engine": digest}})
		if err != nil {
			t.Fatal(err)
		}
		write(filepath.Join(prefix, "manifest.json"), raw)
		write(filepath.Join(prefix, "installer"), data)
		write(filepath.Join(prefix, "aibertai-engine"), data)
	}
	service := NewService(nil).WithInstallation("", root, "")
	if err := service.VerifyRelease(); err != nil {
		t.Fatal(err)
	}
	if err := NewService(nil).WithInstallation("", root, "v2").VerifyRelease(); err == nil {
		t.Fatal("ignored explicit version")
	}
	write("v1/arm64/installer", []byte("amd64"))
	if err := service.VerifyRelease(); err == nil {
		t.Fatal("accepted altered ARM64 installer")
	}
}
