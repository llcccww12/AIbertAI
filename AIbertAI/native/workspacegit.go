package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const maxWorkspaceGitOutput = 1 << 20

type workspaceGitBuffer struct {
	bytes.Buffer
	truncated bool
}

func (b *workspaceGitBuffer) Write(p []byte) (int, error) {
	room := max(0, maxWorkspaceGitOutput-b.Len())
	if len(p) > room {
		b.truncated = true
	}
	_, _ = b.Buffer.Write(p[:min(len(p), room)])
	return len(p), nil
}

var workspaceGitFilter = regexp.MustCompile(`^filter\.([a-zA-Z0-9_-]+)\.(?:clean|smudge|process|required)$`)

func workspaceGit(ctx context.Context, root *os.Root, method, path string) (any, error) {
	empty := map[string]any{"git": false, "files": []any{}}
	if info, err := root.Lstat(".git"); err != nil || !info.IsDir() {
		if method == "_monkeyai/workspace/changes" {
			return empty, nil
		}
		return nil, errors.New("workspace_git_unavailable")
	}
	resolved, err := os.Stat(root.Name())
	opened, openErr := root.Stat(".")
	if err != nil || openErr != nil || !os.SameFile(resolved, opened) {
		return nil, errors.New("workspace_missing")
	}
	git, err := exec.LookPath("git")
	if err != nil || !filepath.IsAbs(git) {
		return nil, errors.New("workspace_git_unavailable")
	}
	prefix := []string{"--no-pager", "--literal-pathspecs", "-c", "core.fsmonitor=false", "-c", "core.hooksPath=" + os.DevNull}
	env, err := localMCPEnvironment(nil)
	if err != nil {
		return nil, err
	}
	env = append(env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0", "GIT_NO_LAZY_FETCH=1", "GIT_ALLOW_PROTOCOL=",
		"GIT_DIR="+filepath.Join(root.Name(), ".git"), "GIT_WORK_TREE="+root.Name())
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	run := func(args ...string) (*workspaceGitBuffer, error) {
		cmd := exec.CommandContext(ctx, git, append(append([]string(nil), prefix...), args...)...)
		cmd.Dir, cmd.Env, cmd.WaitDelay = root.Name(), env, time.Second
		var output workspaceGitBuffer
		cmd.Stdout = &output
		err := cmd.Run()
		return &output, err
	}
	// A repository can define filters that run programs even for a read-only diff.
	// Clear those definitions before inspecting or comparing its working tree.
	keys, err := run("config", "--null", "--includes", "--name-only", "--get-regexp", "^filter\\.")
	var exit *exec.ExitError
	if keys.truncated || err != nil && (!errors.As(err, &exit) || exit.ExitCode() != 1) {
		return nil, errors.New("workspace_git_config_unavailable")
	}
	for _, key := range strings.Split(keys.String(), "\x00") {
		if key == "" {
			continue
		}
		filter := workspaceGitFilter.FindStringSubmatch(key)
		if filter == nil {
			return nil, errors.New("workspace_git_filter_unavailable")
		}
		value := ""
		if strings.HasSuffix(key, ".required") {
			value = "false"
		}
		prefix = append(prefix, "-c", key+"="+value)
	}
	args := []string{"status", "--porcelain=v1", "-z", "--untracked-files=all", "--no-renames", "--ignore-submodules=all"}
	if method == "_monkeyai/workspace/diff" {
		if _, err := workspaceRelative(path); err != nil || path == "." || path == ".git" || strings.HasPrefix(path, ".git/") {
			return nil, errWorkspacePath
		}
		args = append(args, "--", path)
	}
	status, err := run(args...)
	if err != nil || status.truncated {
		return nil, errors.New("workspace_git_status_failed")
	}
	if method == "_monkeyai/workspace/diff" {
		if strings.HasPrefix(status.String(), "?? ") {
			return map[string]any{"diff": "", "untracked": true, "truncated": false}, nil
		}
		args := []string{"diff", "--no-ext-diff", "--no-textconv", "--no-color", "--ignore-submodules=all", "--no-renames"}
		if _, err := run("rev-parse", "--verify", "-q", "HEAD"); err == nil {
			args = append(args, "HEAD")
		} else {
			args = append(args, "--cached")
		}
		diff, err := run(append(args, "--", path)...)
		if err != nil {
			return nil, errors.New("workspace_git_diff_failed")
		}
		return map[string]any{"diff": diff.String(), "untracked": false, "truncated": diff.truncated}, nil
	}
	files := []map[string]any{}
	for _, entry := range strings.Split(status.String(), "\x00") {
		if len(entry) < 4 {
			continue
		}
		path, state := entry[3:], entry[:2]
		if _, err := workspaceRelative(path); err != nil {
			continue
		}
		operation := "modified"
		if strings.ContainsRune(state, 'D') {
			operation = "deleted"
		} else if state == "??" || strings.ContainsRune(state, 'A') {
			operation = "added"
		}
		files = append(files, map[string]any{"path": path, "operation": operation, "staged": state[0] != ' ' && state[0] != '?'})
	}
	return map[string]any{"git": true, "files": files}, nil
}
