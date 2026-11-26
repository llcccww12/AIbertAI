package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewWorkspaceFileReturnsSVGAsImage(t *testing.T) {
	dir := t.TempDir()
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"/>`
	if err := os.WriteFile(filepath.Join(dir, "logo.svg"), []byte(svg), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	got, err := previewWorkspaceFile(root, "logo.svg", "logo.svg", int64(len(svg)))
	if err != nil {
		t.Fatal(err)
	}
	result := got.(map[string]any)
	if result["kind"] != "image" || !strings.HasPrefix(result["imageUrl"].(string), "data:image/svg+xml;base64,") {
		t.Fatalf("svg preview = %v", result)
	}
}

func TestPreviewWorkspaceFileOffersFontsAsDataURL(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.woff2"), []byte("wOF2\x00\x01"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	got, err := previewWorkspaceFile(root, "a.woff2", "a.woff2", 6)
	if err != nil {
		t.Fatal(err)
	}
	result := got.(map[string]any)
	if result["kind"] != "unavailable" || result["reason"] != "binary" || !strings.HasPrefix(result["fontUrl"].(string), "data:font/woff2;base64,") {
		t.Fatalf("font preview = %v", result)
	}
}
