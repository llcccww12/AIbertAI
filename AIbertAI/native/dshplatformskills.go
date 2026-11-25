package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// DSH accepts only lowercase hyphenated skill names (skill/src/index.ts).
var dshSkillName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

const dshPlatformSkillStamp = ".aibertai-skill"

// One sync at a time: concurrent session starts share the directory.
var dshPlatformSkillSync sync.Mutex

// dshPlatformSkillName maps a platform skill name to a DSH skill name. Names
// DSH cannot accept (for example Chinese names) fall back to the skill ID.
func dshPlatformSkillName(name, id string) string {
	candidate := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), "_", "-"))
	if dshSkillName.MatchString(candidate) && len(candidate) <= 64 {
		return candidate
	}
	return "skill-" + strings.ToLower(strings.ReplaceAll(id, "-", ""))[:12]
}

// dshSkillDocument replaces any frontmatter with the name and description DSH
// requires, keeping the instruction body unchanged.
func dshSkillDocument(name, description, content string) []byte {
	body := strings.ReplaceAll(content, "\r\n", "\n")
	if rest, ok := strings.CutPrefix(body, "---\n"); ok {
		if end := strings.Index(rest, "\n---"); end >= 0 {
			body = strings.TrimLeft(rest[end+4:], "\n")
		}
	}
	if strings.TrimSpace(description) == "" {
		description = name
	}
	quoted := func(s string) string { data, _ := json.Marshal(s); return string(data) }
	return []byte("---\nname: " + quoted(name) + "\ndescription: " + quoted(strings.TrimSpace(description)) + "\n---\n\n" + body)
}

// syncDSHPlatformSkills mirrors the account's platform skills into dir as
// <name>/SKILL.md bundles for DSH's filesystem skill provider, which watches
// the directory. Unchanged packages are kept; skills no longer granted are
// removed. A skill that fails to install is left out and reported.
func (c *client) syncDSHPlatformSkills(ctx context.Context, auth authorization, dir string) error {
	dshPlatformSkillSync.Lock()
	defer dshPlatformSkillSync.Unlock()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	catalog, err := c.desktopSkillCatalog(ctx, auth)
	if err != nil {
		return err
	}
	keep := map[string]bool{}
	var failed error
	for _, item := range catalog {
		id, _ := item["id"].(string)
		sha, _ := item["package_sha256"].(string)
		route, _ := item["download_path"].(string)
		platformName, _ := item["name"].(string)
		description, _ := item["description"].(string)
		name := dshPlatformSkillName(platformName, id)
		if keep[name] || !skillPackageSHA.MatchString(sha) || !strings.HasPrefix(route, "/api/v1/skills/"+id+"/package?") {
			continue
		}
		keep[name] = true
		target := filepath.Join(dir, name)
		stamp := id + "\n" + sha + "\n" + platformName + "\n" + description
		if current, err := os.ReadFile(filepath.Join(target, dshPlatformSkillStamp)); err == nil && string(current) == stamp {
			continue
		}
		if err := c.installDSHPlatformSkill(ctx, auth, dir, target, name, description, route, sha, stamp); err != nil {
			failed = errors.Join(failed, err)
			delete(keep, name)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !keep[entry.Name()] {
			_ = os.RemoveAll(filepath.Join(dir, entry.Name()))
		}
	}
	return failed
}

func (c *client) installDSHPlatformSkill(ctx context.Context, auth authorization, dir, target, name, description, route, sha, stamp string) error {
	const maxPackage = 16 << 20
	resp, err := c.request(ctx, http.MethodGet, route, auth.Access, "", "", "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxPackage+1))
	sum := sha256.Sum256(data)
	if err != nil || len(data) > maxPackage || hex.EncodeToString(sum[:]) != sha {
		return errSkillPackageInvalid
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return errSkillPackageInvalid
	}
	// Stage beside the watched directory so DSH never sees a partial bundle.
	stage, err := os.MkdirTemp(filepath.Dir(dir), ".platform-skill-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if err := extractSkillPackage(ctx, archive, stage, "SKILL.md"); err != nil {
		return err
	}
	content, err := os.ReadFile(filepath.Join(stage, "SKILL.md"))
	if err != nil || len(content) > 128<<10 {
		return errSkillPackageInvalid
	}
	if err := os.WriteFile(filepath.Join(stage, "SKILL.md"), dshSkillDocument(name, description, string(content)), 0600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(stage, dshPlatformSkillStamp), []byte(stamp), 0600); err != nil {
		return err
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	return os.Rename(stage, target)
}

// dshSkillPreferencePrefix keys a platform skill in resource preferences; the
// desktop's "default enable" switch and per-conversation picks use it.
const dshSkillPreferencePrefix = "skill/server/"

func dshSessionSkillsDir(cache, sessionID string) string {
	return filepath.Join(cache+"-sessions", sessionID)
}

// materializeDSHSessionSkills gives one conversation only the skills it has
// enabled: copies of the account cache's bundles in its own watched folder.
func materializeDSHSessionSkills(cache, sessionDir string, enabled func(id string) bool) error {
	dshPlatformSkillSync.Lock()
	defer dshPlatformSkillSync.Unlock()
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return err
	}
	entries, err := os.ReadDir(cache)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	keep := map[string]bool{}
	var failed error
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		stamp, err := os.ReadFile(filepath.Join(cache, entry.Name(), dshPlatformSkillStamp))
		id, _, _ := strings.Cut(string(stamp), "\n")
		if err != nil || !validUUID(id) || !enabled(id) {
			continue
		}
		keep[entry.Name()] = true
		target := filepath.Join(sessionDir, entry.Name())
		if current, err := os.ReadFile(filepath.Join(target, dshPlatformSkillStamp)); err == nil && bytes.Equal(current, stamp) {
			continue
		}
		if err := copyDSHSkillBundle(filepath.Join(cache, entry.Name()), target); err != nil {
			failed = errors.Join(failed, err)
		}
	}
	current, err := os.ReadDir(sessionDir)
	if err != nil {
		return errors.Join(failed, err)
	}
	for _, entry := range current {
		if !keep[entry.Name()] {
			_ = os.RemoveAll(filepath.Join(sessionDir, entry.Name()))
		}
	}
	return failed
}

func copyDSHSkillBundle(source, target string) error {
	// Stage beside the watched folder so DSH never sees a partial bundle.
	stage, err := os.MkdirTemp(filepath.Dir(filepath.Dir(target)), ".session-skill-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if err := os.CopyFS(stage, os.DirFS(source)); err != nil {
		return err
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	return os.Rename(stage, target)
}
