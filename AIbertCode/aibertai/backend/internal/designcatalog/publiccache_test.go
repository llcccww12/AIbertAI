package designcatalog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestPublicCacheListsAndChecksDetailDigest(t *testing.T) {
	dir := t.TempDir()
	snapshot := `{
		"revision":"asset-snapshot:fixture",
		"listing":{"templates":{"image-one":{"id":"image-one","mode":"image","category":"logo"},"video-one":{"id":"video-one","mode":"video"}}},
		"mode_buckets":{"":["image-one","video-one"],"image":["image-one"],"video":["video-one"]},
		"details":{"templates":{"image-one":{"id":"image-one","content_digest":"test"}}}
	}`
	if err := os.WriteFile(filepath.Join(dir, "catalog.json"), []byte(snapshot), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MONKEYAI_DESIGN_PUBLIC_CACHE_DIR", dir)
	router := chi.NewRouter()
	(&Service{}).RegisterCatalog(router, func(next http.Handler) http.Handler { return next })
	read := func(path string, status int) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, publicCachePath+path, nil))
		if w.Code != status {
			t.Fatalf("%s: status %d, want %d (%s)", path, w.Code, status, w.Body)
		}
		return w
	}
	w := read("/catalog/templates?mode=image&category=logo&page_size=1", http.StatusOK)
	var payload struct {
		Data struct {
			Total int
			Items []struct{ ID string }
		}
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil || w.Header().Get("X-Design-Cache") != "local" ||
		payload.Data.Total != 1 || len(payload.Data.Items) != 1 || payload.Data.Items[0].ID != "image-one" {
		t.Fatalf("unexpected cache response: %s, %v", w.Body, err)
	}
	w = read("/catalog/templates?page=2&page_size=1", http.StatusOK)
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil ||
		payload.Data.Total != 2 || len(payload.Data.Items) != 1 || payload.Data.Items[0].ID != "video-one" {
		t.Fatalf("unexpected second cache page: %s, %v", w.Body, err)
	}
	w = read("/catalog/templates?page=3&page_size=1", http.StatusOK)
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil ||
		payload.Data.Total != 2 || payload.Data.Items == nil || len(payload.Data.Items) != 0 {
		t.Fatalf("unexpected empty cache page: %s, %v", w.Body, err)
	}
	read("/catalog/templates/image-one?content_digest=sha256:test", http.StatusOK)
	read("/catalog/templates/image-one?content_digest=sha256:different", http.StatusConflict)
	read("/catalog/styles", http.StatusNotFound)
	read("/assets/catalog.json", http.StatusNotFound)
}

func TestPublicCacheHidesBundlesWithoutPackage(t *testing.T) {
	dir := t.TempDir()
	snapshot := `{
		"revision":"asset-snapshot:fixture",
		"listing":{"templates":{
			"deck-ok":{"id":"deck-ok","mode":"deck","template_kind":"bundle","downloadable":true,"download_url":"/api/v1/design-resources/public-cache/assets/` + strings.Repeat("a", 64) + `.zip"},
			"deck-missing":{"id":"deck-missing","mode":"deck","template_kind":"bundle","downloadable":false}}},
		"mode_buckets":{"deck":["deck-missing","deck-ok"]},
		"details":{"templates":{}}
	}`
	if err := os.WriteFile(filepath.Join(dir, "catalog.json"), []byte(snapshot), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MONKEYAI_DESIGN_PUBLIC_CACHE_DIR", dir)
	router := chi.NewRouter()
	(&Service{}).RegisterCatalog(router, func(next http.Handler) http.Handler { return next })
	for _, query := range []string{"mode=deck", "q=deck"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, publicCachePath+"/catalog/templates?"+query, nil))
		var payload struct {
			Data struct {
				Total int
				Items []struct{ ID string }
			}
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil || payload.Data.Total != 1 ||
			len(payload.Data.Items) != 1 || payload.Data.Items[0].ID != "deck-ok" {
			t.Fatalf("%s: %s, %v", query, w.Body, err)
		}
	}
}
