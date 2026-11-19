package enginehost

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"aibertai.local/backend/internal/web"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

//go:embed install.sh
var installer []byte

type manifest struct {
	Version      string            `json:"version"`
	Architecture string            `json:"architecture"`
	Files        map[string]string `json:"files"`
}

func (s *Service) manifest(version, arch string) (manifest, []byte, error) {
	var m manifest
	if s.releases == "" || !versionPattern.MatchString(version) || (arch != "amd64" && arch != "arm64") {
		return m, nil, errors.New("发行配置无效")
	}
	raw, err := os.ReadFile(filepath.Join(s.releases, version, arch, "manifest.json"))
	if err != nil {
		return m, nil, err
	}
	if len(raw) > 16<<10 || json.Unmarshal(raw, &m) != nil || m.Version != version || m.Architecture != arch || !preparedHostRelease(m.Files) {
		return m, nil, errors.New("发行清单无效")
	}
	for name := range m.Files {
		if !hashPattern.MatchString(m.Files[name]) {
			return m, nil, errors.New("发行摘要缺失")
		}
		info, err := os.Lstat(filepath.Join(s.releases, version, arch, name))
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > 256<<20 {
			return m, nil, errors.New("发行文件缺失或类型无效")
		}
	}
	return m, raw, nil
}
func (s *Service) script(w http.ResponseWriter, r *http.Request) {
	s.bootstrapScript(w, r, false)
}
func (s *Service) bootstrapScript(w http.ResponseWriter, r *http.Request, upgrade bool) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if !validServer(s.server) {
		http.Error(w, web.Message(r.Context(), "", "后台 HTTP(S) 地址无效"), 503)
		return
	}
	serverURL := r.URL.Query().Get("server_url")
	if serverURL != "" && !validServer(serverURL) {
		fail(w, r, 400, "server_url 必须为无凭据、路径或查询参数的 HTTP 或 HTTPS 根地址")
		return
	}
	serverURL = strings.TrimRight(serverURL, "/")
	secret := r.URL.Query().Get("token")
	version := r.URL.Query().Get("version")
	if upgrade && r.URL.Query().Has("token") {
		fail(w, r, 400, "升级不接收设备接入凭据")
		return
	}
	if secret != "" || r.URL.Query().Has("token") {
		var err error
		version, err = s.installationVersion(r.Context(), secret)
		if err != nil {
			http.Error(w, web.Message(r.Context(), "", "安装命令无效或已过期"), 401)
			return
		}
	} else if version == "" {
		version = s.version
	}
	if !versionPattern.MatchString(version) || version == "latest" {
		http.Error(w, web.Message(r.Context(), "", "升级版本无效"), 400)
		return
	}
	var body strings.Builder
	body.WriteString("#!/bin/bash\ninstall_engine() {\n")
	if upgrade {
		body.WriteString("bootstrap_mode=upgrade\n")
	} else {
		body.WriteString("bootstrap_mode=install\n")
	}
	fmt.Fprintf(&body, "server_url=%s\n", quote(serverURL))
	body.WriteString("case \"$(uname -m)\" in\n")
	for _, arch := range []string{"amd64", "arm64"} {
		m, raw, err := s.manifest(version, arch)
		if err != nil {
			http.Error(w, web.Message(r.Context(), "", "服务器安装资源不完整"), http.StatusServiceUnavailable)
			return
		}
		pattern := "x86_64|amd64"
		if arch == "arm64" {
			pattern = "aarch64|arm64"
		}
		args := []string{s.server, secret, version, arch, m.Files["installer"], hash(raw), "", "installer"}
		for i := range args {
			args[i] = quote(args[i])
		}
		fmt.Fprintf(&body, "%s) set -- %s ;;\n", pattern, strings.Join(args, " "))
	}
	body.WriteString("*) echo '仅支持 amd64 / arm64' >&2; return 1 ;;\nesac\n")
	body.Write(installer)
	body.WriteString("\n}\ninstall_engine\n")
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write([]byte(body.String()))
}
func (s *Service) artifact(w http.ResponseWriter, r *http.Request) {
	version, arch, name := chi.URLParam(r, "version"), chi.URLParam(r, "arch"), chi.URLParam(r, "file")
	if name != "manifest.json" && name != "installer" && name != "monkeyai-engine" {
		web.NotFound(w, r)
		return
	}
	if s.releases == "" {
		web.NotFound(w, r)
		return
	}
	if _, _, err := s.manifest(version, arch); err != nil {
		web.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, filepath.Join(s.releases, version, arch, name))
}

func preparedHostRelease(files map[string]string) bool {
	return len(files) == 2 && files["installer"] != "" && files["installer"] == files["monkeyai-engine"]
}
