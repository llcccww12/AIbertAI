package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestDSHPlatformSkillName(t *testing.T) {
	const id = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	for name, want := range map[string]string{"pdf": "pdf", "Call_Summary": "call-summary", "周报助手": "skill-c1bd985c9f71", "a--b": "skill-c1bd985c9f71"} {
		if got := dshPlatformSkillName(name, id); got != want {
			t.Errorf("%q: got %q, want %q", name, got, want)
		}
	}
}

func TestDSHSkillDocumentReplacesFrontmatter(t *testing.T) {
	got := string(dshSkillDocument("weekly", "写周报", "---\nname: 周报\nfoo: bar\n---\n\nBody line\n"))
	if got != "---\nname: \"weekly\"\ndescription: \"写周报\"\n---\n\nBody line\n" {
		t.Fatalf("unexpected document:\n%s", got)
	}
	if got := string(dshSkillDocument("plain", "", "Just do it")); !strings.HasSuffix(got, "---\n\nJust do it") || !strings.Contains(got, `description: "plain"`) {
		t.Fatalf("plain body not wrapped:\n%s", got)
	}
}

func TestSyncDSHPlatformSkills(t *testing.T) {
	const keepID, goneID = "c1bd985c-9f71-4b85-937a-c41b1b104d38", "d2bd985c-9f71-4b85-937a-c41b1b104d38"
	pkg := func(files map[string]string) ([]byte, string) {
		var b bytes.Buffer
		z := zip.NewWriter(&b)
		for name, body := range files {
			w, _ := z.Create(name)
			_, _ = w.Write([]byte(body))
		}
		_ = z.Close()
		sum := sha256.Sum256(b.Bytes())
		return b.Bytes(), hex.EncodeToString(sum[:])
	}
	keep, keepSHA := pkg(map[string]string{"weekly/SKILL.md": "Write the weekly report.", "weekly/scripts/run.sh": "echo hi"})
	_, goneSHA := pkg(map[string]string{"SKILL.md": "old"})
	var revoked atomic.Bool
	var downloads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/skills":
			items := fmt.Sprintf(`{"id":%q,"name":"weekly","description":"写周报","package_sha256":%q,"download_path":"/api/v1/skills/%s/package?sha256=%s"}`, keepID, keepSHA, keepID, keepSHA)
			if !revoked.Load() {
				items += fmt.Sprintf(`,{"id":%q,"name":"周报","package_sha256":%q,"download_path":"/api/v1/skills/%s/package?sha256=%s"}`, goneID, goneSHA, goneID, goneSHA)
			}
			fmt.Fprintf(w, `{"skills":[%s]}`, items)
		case "/api/v1/skills/" + keepID + "/package":
			downloads.Add(1)
			_, _ = w.Write(keep)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	c := &client{server: server.URL, http: server.Client()}
	dir := filepath.Join(t.TempDir(), "platform-skills")
	// The second package is missing upstream: it is reported, the first still installs.
	if err := c.syncDSHPlatformSkills(context.Background(), authorization{Access: "account"}, dir); err == nil {
		t.Fatal("failed download was not reported")
	}
	doc, err := os.ReadFile(filepath.Join(dir, "weekly", "SKILL.md"))
	if err != nil || !strings.HasPrefix(string(doc), "---\nname: \"weekly\"\ndescription: \"写周报\"\n---\n") || !strings.HasSuffix(string(doc), "Write the weekly report.") {
		t.Fatalf("skill not installed for DSH: %q %v", doc, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "weekly", "scripts", "run.sh")); err != nil {
		t.Fatal("package files not installed")
	}
	revoked.Store(true)
	if err := c.syncDSHPlatformSkills(context.Background(), authorization{Access: "account"}, dir); err != nil {
		t.Fatal(err)
	}
	if downloads.Load() != 1 {
		t.Fatalf("unchanged package downloaded again: %d", downloads.Load())
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "weekly" {
		t.Fatalf("stale skills left behind: %v", entries)
	}
	staged, _ := filepath.Glob(filepath.Join(filepath.Dir(dir), ".platform-skill-*"))
	if len(staged) != 0 {
		t.Fatalf("staging left behind: %v", staged)
	}
}

func TestDSHPatchAddsPlatformSkillsDir(t *testing.T) {
	g := &dshLocalGateway{model: "m", protocol: "openai_chat_completions", platformSkillsDir: "/profile/platform-skills"}
	patch, err := g.Patch("http://127.0.0.1:1234/v1", selectedModelConfig{ID: "m", Protocol: "openai_chat_completions"})
	if err != nil || !strings.Contains(string(patch), `{"config":{"customSkillDirs":["/profile/platform-skills"]},"id":"skill-filesystem"}`) {
		t.Fatalf("platform skills dir missing from patch: %s %v", patch, err)
	}
	patch, _ = g.Patch("http://127.0.0.1:1234/v1", selectedModelConfig{ID: "m", Protocol: "openai_chat_completions"}, "skill-filesystem")
	if strings.Contains(string(patch), "customSkillDirs") {
		t.Fatal("disabled skill provider still configured")
	}
}

func TestMaterializeDSHSessionSkillsKeepsOnlyEnabled(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, "platform-skills")
	a, b := "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"
	for id, name := range map[string]string{a: "alpha", b: "beta"} {
		if err := os.MkdirAll(filepath.Join(cache, name, "scripts"), 0700); err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(cache, name, "SKILL.md"), []byte(name), 0600)
		_ = os.WriteFile(filepath.Join(cache, name, "scripts", "run.sh"), []byte("echo"), 0600)
		_ = os.WriteFile(filepath.Join(cache, name, dshPlatformSkillStamp), []byte(id+"\nsha"), 0600)
	}
	dir := dshSessionSkillsDir(cache, "s1")
	enabled := map[string]bool{a: true}
	if err := materializeDSHSessionSkills(cache, dir, func(id string) bool { return enabled[id] }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "alpha", "scripts", "run.sh")); err != nil {
		t.Fatal("enabled skill not copied with its files")
	}
	if _, err := os.Stat(filepath.Join(dir, "beta")); !os.IsNotExist(err) {
		t.Fatal("disabled skill exposed")
	}
	enabled = map[string]bool{b: true}
	if err := materializeDSHSessionSkills(cache, dir, func(id string) bool { return enabled[id] }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "alpha")); !os.IsNotExist(err) {
		t.Fatal("switched-off skill kept")
	}
	if _, err := os.Stat(filepath.Join(dir, "beta", "SKILL.md")); err != nil {
		t.Fatal("newly enabled skill missing")
	}
}

func TestSessionSkillSelection(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	skill := "skill/server/22222222-2222-4222-8222-222222222222"
	h := &relaySessionHost{live: &acpLive{dir: t.TempDir(), sessions: map[string]acpSession{id: {ID: id, State: "idle"}},
		connectorPrefs: connectorPreferences{Enabled: map[string]bool{"skill/server/33333333-3333-4333-8333-333333333333": true}}}}
	change := func(c map[string]any) error {
		raw, _ := json.Marshal(map[string]any{"sessionId": id, "change": c})
		_, err := h.sessionConnectorResources(raw, true)
		return err
	}
	if err := change(map[string]any{"kind": "skill_selection", "ids": []string{skill}}); err != nil {
		t.Fatal(err)
	}
	enabled := h.live.sessions[id].ConnectorPreferences.Enabled
	if !enabled[skill] || len(enabled) != 1 {
		t.Fatalf("selection = %v", enabled)
	}
	if err := change(map[string]any{"kind": "resource", "id": skill, "enabled": false}); err != nil || h.live.sessions[id].ConnectorPreferences.Enabled[skill] {
		t.Fatal("per-conversation switch not saved", err)
	}
	if change(map[string]any{"kind": "skill_selection", "ids": []string{"mcp/x"}}) == nil {
		t.Fatal("non-skill id accepted")
	}
	rule := "rule/server/44444444-4444-4444-8444-444444444444"
	if err := change(map[string]any{"kind": "resource", "id": rule, "enabled": true}); err != nil || !h.live.sessions[id].ConnectorPreferences.Enabled[rule] {
		t.Fatal("per-conversation rule not saved", err)
	}
	if change(map[string]any{"kind": "resource", "id": "rule/server/not-a-uuid", "enabled": true}) == nil {
		t.Fatal("invalid rule id accepted")
	}
}
