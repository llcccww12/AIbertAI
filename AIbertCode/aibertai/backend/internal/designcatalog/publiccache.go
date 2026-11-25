package designcatalog

import (
	"cmp"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"aibertai.local/backend/internal/web"
	"github.com/go-chi/chi/v5"
)

const publicCachePath = "/api/v1/design-resources/public-cache"

var publicCacheAsset = regexp.MustCompile(`^[a-f0-9]{64}\.(jpg|png|webp|avif|gif|mp4|webm|zip)$`)

type publicCatalog struct {
	Revision string                                `json:"revision"`
	Listing  map[string]map[string]json.RawMessage `json:"listing"`
	Details  map[string]map[string]json.RawMessage `json:"details"`
	Buckets  map[string][]string                   `json:"mode_buckets"`
}

func registerPublicCache(router chi.Router, dir string) {
	router.Route("/public-cache", func(r chi.Router) {
		r.Get("/catalog/{kind}", func(w http.ResponseWriter, r *http.Request) {
			readPublicCatalog(w, r, dir)
		})
		r.Get("/catalog/{kind}/{id}", func(w http.ResponseWriter, r *http.Request) {
			readPublicCatalog(w, r, dir)
		})
		assets := http.StripPrefix(publicCachePath+"/assets", http.FileServer(http.Dir(filepath.Join(dir, "assets"))))
		serveAsset := func(w http.ResponseWriter, r *http.Request) {
			if !publicCacheAsset.MatchString(chi.URLParam(r, "name")) {
				web.NotFound(w, r)
				return
			}
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			w.Header().Set("ETag", strconv.Quote(chi.URLParam(r, "name")))
			assets.ServeHTTP(w, r)
		}
		r.Get("/assets/{name}", serveAsset)
		r.Head("/assets/{name}", serveAsset)
	})
}

func readPublicCatalog(w http.ResponseWriter, r *http.Request, dir string) {
	kind := chi.URLParam(r, "kind")
	if kind != "templates" && kind != "design-systems" {
		web.NotFound(w, r)
		return
	}
	raw, err := os.ReadFile(filepath.Join(dir, "catalog.json"))
	var catalog publicCatalog
	if err != nil || json.Unmarshal(raw, &catalog) != nil || catalog.Revision == "" {
		web.NotFound(w, r)
		return
	}
	w.Header().Set("X-Design-Cache", "local")
	if id := chi.URLParam(r, "id"); id != "" {
		detail := catalog.Details[kind][id]
		if detail == nil {
			web.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		if digest := q.Get("content_digest"); digest != "" {
			var row struct {
				Digest string `json:"content_digest"`
			}
			if json.Unmarshal(detail, &row) != nil || strings.TrimPrefix(digest, "sha256:") != strings.TrimPrefix(row.Digest, "sha256:") {
				writeAPIError(w, r, fault(http.StatusConflict, "resource_version_mismatch", "设计资源内容已变化"))
				return
			}
		}
		apiJSON(w, cachedResource(detail))
		return
	}
	if len(r.URL.RawQuery) > 4096 {
		writeAPIError(w, r, fault(http.StatusBadRequest, "invalid_argument", "查询编码无效"))
		return
	}
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	size, _ := strconv.Atoi(q.Get("page_size"))
	page = min(1000, max(page, 1))
	size = min(100, max(size, 1))
	if q.Get("page_size") == "" {
		size = 20
	}
	mode, category, templateKind := q.Get("mode"), q.Get("category"), q.Get("template_kind")
	search := strings.ToLower(strings.TrimSpace(q.Get("q")))
	rows := catalog.Listing[kind]
	var ids []string
	if kind == "templates" && search == "" {
		ids = catalog.Buckets[mode]
		if bucket, ok := catalog.Buckets[mode+"/"+category]; ok {
			ids = bucket
		}
	} else {
		for id := range rows {
			ids = append(ids, id)
		}
		slices.Sort(ids)
	}
	start := (page - 1) * size
	total := 0
	items := make([]map[string]any, 0, min(size, len(ids)))
	for _, id := range ids {
		var item struct {
			Category, Mode, Name, Title, Description string
			TemplateKind                             string            `json:"template_kind"`
			LocalizedName                            map[string]string `json:"localizedName"`
		}
		row := rows[id]
		if json.Unmarshal(row, &item) != nil || category != "" && !strings.EqualFold(item.Category, category) ||
			mode != "" && item.Mode != mode || templateKind != "" && templateKind != item.TemplateKind {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(strings.Join([]string{id, cmp.Or(item.Name, item.Title), item.Description, item.LocalizedName["zh"]}, " ")), search) {
			continue
		}
		resource := cachedResource(row)
		// A bundle template whose package was not cached cannot be used, so
		// do not offer it to clients.
		if item.TemplateKind == "bundle" && resource["downloadable"] == false {
			continue
		}
		if total >= start && len(items) < size {
			items = append(items, resource)
		}
		total++
	}
	apiJSON(w, map[string]any{"asset_snapshot_id": catalog.Revision, "items": items, "total": total,
		"page": page, "page_size": size, "total_pages": (total + size - 1) / size})
}

// The cached public catalog can retain source URLs for auditing. Never send
// those URLs to installed clients, even when a preview was unavailable at sync.
func cachedResource(raw json.RawMessage) map[string]any {
	var item map[string]any
	if json.Unmarshal(raw, &item) != nil {
		return nil
	}
	local := func(value any) bool {
		text, ok := value.(string)
		name, inside := strings.CutPrefix(text, publicCachePath+"/assets/")
		return ok && inside && publicCacheAsset.MatchString(name)
	}
	for _, key := range []string{"preview_url", "download_url", "external_source"} {
		if !local(item[key]) {
			delete(item, key)
		}
	}
	for _, key := range []string{"preview", "card_preview", "download"} {
		if metadata, ok := item[key].(map[string]any); ok {
			for _, field := range []string{"url", "poster_url", "video_url"} {
				if !local(metadata[field]) {
					delete(metadata, field)
				}
			}
		}
	}
	download, _ := item["download"].(map[string]any)
	if !local(item["download_url"]) && !local(download["url"]) {
		item["downloadable"] = false
	}
	return item
}
