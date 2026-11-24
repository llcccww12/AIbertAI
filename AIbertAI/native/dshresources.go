package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// Set only by the signed packaging build using -X, never from a runtime env var.
// An empty value leaves legacy releases on their existing engine.
var dshManifestSHA256 string
var dshDevelopmentResources string

const dshSourceCommit = "3f321d32bfe382212a8ed8785c1d47a5a300aa23"
const dshSourceArchiveSHA256 = "a46ccc2890645d0a7f3b032eeadb988769893392da5dcd48a1bc4ce0e49ba3fc"
const dshPatchSHA256 = "9739f29e682dfb9eb1789f4c9ece7bb2a7c7f98ec77d2ae01f9bc6326db6a15d"
const dshEntry = "app/apps/cli/lib/bin.js"

type dshManifestFile struct {
	Path   string `json:"path"`
	Type   string `json:"type"`
	SHA256 string `json:"sha256,omitempty"`
	Target string `json:"target,omitempty"`
}
type dshManifest struct {
	Version       int               `json:"version"`
	SourceCommit  string            `json:"source_commit"`
	SourceArchive string            `json:"source_archive_sha256"`
	Patch         string            `json:"patch_sha256"`
	Lockfile      string            `json:"lockfile_sha256"`
	NodeVersion   string            `json:"node_version"`
	Target        string            `json:"target"`
	Node          string            `json:"node"`
	Entry         string            `json:"entry"`
	Files         []dshManifestFile `json:"files"`
}

var dshDigestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var dshNodeVersionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)
var dshResourceHashSlots = make(chan struct{}, 8)

// The manifest digest comes from the signed native binary, not the resource
// directory. This verifies package contents; installers must protect the files
// against modification after verification and before/during execution.
func verifyDSHResources(root, trustedDigest, target string) (string, string, error) {
	bad := func() (string, string, error) { return "", "", errors.New("dsh_resources_invalid") }
	if !filepath.IsAbs(root) || !dshDigestPattern.MatchString(trustedDigest) {
		return bad()
	}
	switch target {
	case "darwin-arm64", "windows-amd64", "linux-amd64", "linux-arm64":
	default:
		return bad()
	}

	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return bad()
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return bad()
	}
	manifestPath := filepath.Join(root, "manifest.json")
	info, err = os.Lstat(manifestPath)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 128<<20 {
		return bad()
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return bad()
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != trustedDigest {
		return bad()
	}
	var m dshManifest
	if json.Unmarshal(data, &m) != nil || m.Version != 1 || m.SourceCommit != dshSourceCommit || m.SourceArchive != dshSourceArchiveSHA256 || m.Patch != dshPatchSHA256 || !dshDigestPattern.MatchString(m.Lockfile) || !dshNodeVersionPattern.MatchString(m.NodeVersion) || m.Target != target || m.Entry != dshEntry {
		return bad()
	}
	node := "bin/node"
	if strings.HasPrefix(target, "windows-") {
		node = "bin/node.exe"
	}
	if m.Node != node {
		return bad()
	}
	entries := make(map[string]dshManifestFile, len(m.Files))
	for _, v := range m.Files {
		if !fs.ValidPath(v.Path) || v.Path == "." || strings.Contains(v.Path, `\`) || v.Path == "manifest.json" {
			return bad()
		}
		if _, ok := entries[v.Path]; ok {
			return bad()
		}
		if v.Type == "file" {
			if !dshDigestPattern.MatchString(v.SHA256) || v.Target != "" {
				return bad()
			}
		} else if v.Type == "symlink" {
			if v.Target == "" || filepath.IsAbs(v.Target) || strings.Contains(v.Target, `\`) || v.SHA256 != "" {
				return bad()
			}
		} else {
			return bad()
		}
		entries[v.Path] = v
	}
	// Hash the inventory concurrently, with a fixed bound on open files. The
	// full resource can contain many small files, so serial reads add a large
	// delay each time a short-lived SDK process starts.
	type fileToVerify struct{ path, digest string }
	files := make(chan fileToVerify, 32)
	failed := make(chan struct{}, 1)
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for item := range files {
				dshResourceHashSlots <- struct{}{}
				file, err := os.Open(item.path)
				if err == nil {
					h := sha256.New()
					_, readErr := io.Copy(h, file)
					err = errors.Join(readErr, file.Close())
					if err == nil && hex.EncodeToString(h.Sum(nil)) != item.digest {
						err = errors.New("digest mismatch")
					}
				}
				<-dshResourceHashSlots
				if err != nil {
					select {
					case failed <- struct{}{}:
					default:
					}
				}
			}
		}()
	}
	seen := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." || rel == "manifest.json" {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		v, ok := entries[rel]
		if !ok {
			return errors.New("unexpected resource")
		}
		seen++
		if d.Type()&os.ModeSymlink != 0 {
			if v.Type != "symlink" {
				return errors.New("wrong type")
			}
			link, err := os.Readlink(path)
			if err != nil || link != v.Target {
				return errors.New("wrong symlink")
			}
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			within, err := filepath.Rel(root, resolved)
			if err != nil || !filepath.IsLocal(within) {
				return errors.New("symlink escapes resources")
			}
			return nil
		}
		if !d.Type().IsRegular() || v.Type != "file" || d.Name() == ".env" || d.Name() == ".npmrc" {
			return errors.New("invalid resource")
		}
		files <- fileToVerify{path: path, digest: v.SHA256}
		return nil
	})
	close(files)
	workers.Wait()
	if err != nil || seen != len(entries) || len(failed) != 0 {
		return bad()
	}
	for _, p := range []string{node, dshEntry} {
		if _, ok := entries[p]; !ok {
			return bad()
		}
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil || !info.Mode().IsRegular() {
			return bad()
		}
	}
	return filepath.Join(root, filepath.FromSlash(node)), filepath.Join(root, filepath.FromSlash(dshEntry)), nil
}

func configureInstalledDSH(live *acpLive, profile, serverID, machineID string) error {
	if dshManifestSHA256 == "" {
		return nil
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("dsh resources: %w", err)
	}
	resourceRoot := filepath.Join(filepath.Dir(filepath.Dir(executable)), "dsh")
	if dshDevelopmentResources == "enabled" {
		resourceRoot = os.Getenv("AIBERTAI_DSH_RESOURCES")
	}
	return configureDSHResources(live, profile, serverID, machineID, resourceRoot, dshManifestSHA256, runtime.GOOS+"-"+runtime.GOARCH)
}

func configureDSHResources(live *acpLive, profile, serverID, machineID, resourceRoot, trustedDigest, target string) error {
	node, entry, err := verifyDSHResources(resourceRoot, trustedDigest, target)
	if err != nil {
		return fmt.Errorf("dsh resources: %w", err)
	}
	base := filepath.Join(profile, "source-runtime", serverID, machineID, "dsh")
	if err = os.MkdirAll(filepath.Join(base, "home"), 0700); err != nil {
		return fmt.Errorf("dsh home: %w", err)
	}
	gateway := &dshGateway{client: live.client, authorization: live.authorization}
	config, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, StandardSkillsDir: filepath.Join(resourceRoot, "app", "packages", "preset", "agent-preset", "skills"), Home: filepath.Join(base, "home"), Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(base, "gateway"), gateway, func(ctx context.Context, session acpSession) (selectedModelConfig, error) {
		auth, err := live.authorization(ctx)
		if err != nil {
			return selectedModelConfig{}, err
		}
		return live.client.modelConfig(ctx, auth, session.Model)
	})
	if err != nil {
		return err
	}
	prepare := config.Prepare
	config.Prepare = func(ctx context.Context, session acpSession) (*exec.Cmd, string, string, *dshLocalGateway, func(), error) {
		if _, _, err := verifyDSHResources(resourceRoot, trustedDigest, target); err != nil {
			return nil, "", "", nil, nil, err
		}
		return prepare(ctx, session)
	}
	config.VerifiedResourceRoot = filepath.Dir(filepath.Dir(node))
	config.LeasePool = &dshLeasePool{}
	live.dsh = config
	return nil
}
