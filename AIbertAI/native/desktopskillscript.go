package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const skillScriptTool = "run_skill_script"

var skillPackageSHA = regexp.MustCompile(`^[0-9a-f]{64}$`)
var errSkillPackageInvalid = errors.New("skill_package_invalid")
var skillInterpreters = map[string]string{".py": "python3", ".sh": "bash", ".js": "node", ".mjs": "node", ".cjs": "node"}

type skillScriptCall struct {
	Path string   `json:"path"`
	Args []string `json:"arguments"`
}

func validSkillPackagePath(name string) bool {
	return name != "" && !strings.ContainsAny(name, "\\\x00:") && !path.IsAbs(name) &&
		path.Clean(name) == name && name != "." && name != ".." && !strings.HasPrefix(name, "../")
}

func validSkillScript(call skillScriptCall) bool {
	if !validSkillPackagePath(call.Path) || len(call.Path) > 512 || skillInterpreters[path.Ext(call.Path)] == "" || len(call.Args) > 64 {
		return false
	}
	for _, arg := range call.Args {
		if strings.ContainsRune(arg, 0) {
			return false
		}
	}
	return true
}

func (c *client) executeSkillScript(ctx context.Context, auth authorization, skill desktopSkillPackage, call skillScriptCall, cwd string) (string, error) {
	if !validSkillScript(call) || !validUUID(skill.ID) || !skillPackageSHA.MatchString(skill.SHA) {
		return "", errSkillPackageInvalid
	}
	interpreter, err := exec.LookPath(skillInterpreters[path.Ext(call.Path)])
	if err != nil || !filepath.IsAbs(interpreter) {
		return "", errors.New("skill_script_interpreter_unavailable")
	}
	temp, err := os.MkdirTemp("", "aibertai-skill-")
	if err != nil {
		return "", errSkillPackageInvalid
	}
	defer os.RemoveAll(temp)
	archive, err := os.Create(filepath.Join(temp, "package.zip"))
	if err != nil {
		return "", errSkillPackageInvalid
	}
	defer archive.Close()
	resp, err := c.request(ctx, http.MethodGet, "/api/v1/skills/"+skill.ID+"/package?sha256="+skill.SHA,
		auth.Access, "", "", "", nil)
	if err != nil {
		return "", errSkillCommandUnavailable
	}
	defer resp.Body.Close()
	const maxPackage = 500 << 20
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(archive, hash), io.LimitReader(resp.Body, maxPackage+1))
	if err != nil || size > maxPackage || hex.EncodeToString(hash.Sum(nil)) != skill.SHA {
		return "", errSkillPackageInvalid
	}
	pkg, err := zip.NewReader(archive, size)
	if err != nil {
		return "", errSkillPackageInvalid
	}
	dir := filepath.Join(temp, "files")
	if err := os.Mkdir(dir, 0700); err != nil {
		return "", errSkillPackageInvalid
	}
	if err := extractSkillPackage(ctx, pkg, dir, call.Path); err != nil {
		return "", err
	}
	env, err := localMCPEnvironment(nil)
	if err != nil {
		return "", err
	}
	env = append(env, "AIBERTAI_SKILL_DIR="+dir, "PYTHONDONTWRITEBYTECODE=1")
	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, interpreter, append([]string{filepath.Join(dir, filepath.FromSlash(call.Path))}, call.Args...)...)
	cmd.Dir, cmd.Env, cmd.WaitDelay = cwd, env, 2*time.Second
	var output skillScriptOutput
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Run(); err != nil {
		if runCtx.Err() != nil {
			return "", errors.New("skill_script_timed_out_or_cancelled")
		}
		return "", errors.New("skill_script_failed")
	}
	return "Script completed (exit 0).\n" + strings.ToValidUTF8(output.buffer.String(), "�"), nil
}

func extractSkillPackage(ctx context.Context, pkg *zip.Reader, dir, script string) error {
	var prefix string
	found := false
	for _, f := range pkg.File {
		if f.Mode().IsRegular() && path.Base(f.Name) == "SKILL.md" {
			if parent := strings.TrimSuffix(f.Name, "SKILL.md"); !found || len(parent) < len(prefix) {
				prefix, found = parent, true
			}
		}
	}
	if !found {
		return errSkillPackageInvalid
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return errSkillPackageInvalid
	}
	defer root.Close()
	var expanded int64
	const maxExpanded = 1 << 30
	hasScript := false
	for _, f := range pkg.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := strings.TrimSuffix(f.Name, "/")
		if !validSkillPackagePath(name) || (!f.FileInfo().IsDir() && !f.Mode().IsRegular()) {
			return errSkillPackageInvalid
		}
		if f.FileInfo().IsDir() {
			continue
		}
		if !strings.HasPrefix(name, prefix) || f.UncompressedSize64 > uint64(maxExpanded-expanded) {
			return errSkillPackageInvalid
		}
		name = strings.TrimPrefix(name, prefix)
		if !validSkillPackagePath(name) || copySkillFile(root, filepath.FromSlash(name), f) != nil {
			return errSkillPackageInvalid
		}
		expanded += int64(f.UncompressedSize64)
		hasScript = hasScript || name == script
	}
	if !hasScript {
		return errors.New("skill_script_not_found")
	}
	return nil
}

func copySkillFile(root *os.Root, name string, file *zip.File) error {
	if err := root.MkdirAll(filepath.Dir(name), 0700); err != nil {
		return err
	}
	dst, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer dst.Close()
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	n, err := io.Copy(dst, io.LimitReader(src, int64(file.UncompressedSize64)+1))
	if err != nil || uint64(n) != file.UncompressedSize64 {
		return errSkillPackageInvalid
	}
	return nil
}

type skillScriptOutput struct{ buffer bytes.Buffer }

func (out *skillScriptOutput) Write(p []byte) (int, error) {
	const limit = 16 << 10
	if remaining := limit - out.buffer.Len(); remaining > 0 {
		out.buffer.Write(p[:min(len(p), remaining)])
	}
	return len(p), nil
}
