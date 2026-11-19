package main

import (
	"aibertai.local/engine/internal/hostpaths"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"

	"aibertai.local/engine/internal/control"
)

const installedProgram = "/opt/aibertai-engine/aibertai-engine"

var preparedHostUnit = "/etc/systemd/system/" + hostpaths.Unit

var hostReleaseVersion = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

func installPreparedHost(ctx context.Context, args []string, out, stderr io.Writer) error {
	var cfg control.Config
	flags := flag.NewFlagSet("install", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&cfg.Server, "server", "", "后台地址")
	flags.StringVar(&cfg.ServerURL, "server-url", "", "宿主实际访问的后台地址")
	flags.StringVar(&cfg.Version, "release", "", "待安装版本")
	flags.StringVar(&cfg.Architecture, "architecture", "", "宿主架构")
	flags.StringVar(&cfg.DataDir, "data-dir", hostpaths.State+"/data", "持久数据目录")
	flags.BoolVar(&cfg.Insecure, "insecure", false, "使用后台指定的证书模式")
	enrollment := flags.String("enrollment", "", "短期接入凭据")
	checksum := flags.String("manifest-sha256", "", "发行清单摘要")
	if flags.Parse(args) != nil || flags.NArg() != 0 || cfg.Architecture != runtime.GOARCH ||
		cfg.Version != version || !hostReleaseVersion.MatchString(version) || version == "latest" ||
		!filepath.IsAbs(cfg.DataDir) || filepath.Clean(cfg.DataDir) != cfg.DataDir || cfg.DataDir == "/" ||
		len(*checksum) != 64 || *enrollment == "" {
		return errors.New("宿主安装参数无效")
	}
	if runtime.GOOS != "linux" || os.Geteuid() != 0 {
		return errors.New("需要在 Linux 宿主以 root 安装")
	}
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return errors.New("宿主需要 systemd")
	}
	unlock, err := lockHostInstall(filepath.Dir(control.KeyPath))
	if err != nil {
		return err
	}
	defer unlock()
	// Reuse the host key after a failed registration or an interrupted install.
	if raw, err := os.ReadFile(control.ConfigPath); err == nil {
		var current control.Config
		cfg.HostID = ""
		if json.Unmarshal(raw, &current) != nil || current.HostID == "" {
			return errors.New("现有宿主配置无效")
		}
		host := current.HostID
		current.HostID = ""
		current.DataDir, cfg.DataDir = hostpaths.SavedPath(current.DataDir), hostpaths.SavedPath(cfg.DataDir)
		if current != cfg {
			return errors.New("宿主已经登记，请使用现有的宿主配置")
		}
		cfg.HostID = host
		if _, err := os.Lstat(control.KeyPath); err != nil {
			return errors.New("已登记宿主缺少原私钥")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	key, err := control.Key(control.KeyPath)
	if err != nil {
		return err
	}
	client, err := cfg.Client(key)
	if err != nil {
		return err
	}
	program, err := verifiedHostProgram(ctx, client, cfg.Version, cfg.Architecture, *checksum)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(hostpaths.SavedPath(cfg.DataDir), 0700); err != nil {
		return err
	}
	if cfg.HostID == "" {
		if _, err := os.Lstat(preparedHostUnit); !errors.Is(err, os.ErrNotExist) {
			return errors.New("已存在同名宿主服务，请检查现有安装")
		}
		cfg.HostID, err = client.Enroll(ctx, *enrollment, cfg.Architecture, cfg.Version)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		if err := control.Write(control.ConfigPath, raw, 0600); err != nil {
			return err
		}
	}
	if err := control.Write(installedProgram, program, 0755); err != nil {
		return err
	}
	unit := []byte("[Unit]\nDescription=AIbertAI Linux Host Engine\nAfter=network-online.target docker.service\n" +
		"Wants=network-online.target\n[Service]\nExecStart=" + installedProgram + " serve\nRestart=on-failure\n[Install]\nWantedBy=multi-user.target\n")
	if err := control.Write(preparedHostUnit, unit, 0644); err != nil {
		return err
	}
	for _, args := range [][]string{{"daemon-reload"}, {"enable", "--now", hostpaths.Unit}} {
		cmd := exec.CommandContext(ctx, "systemctl", args...)
		cmd.Stdout, cmd.Stderr = out, stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("启用宿主服务失败: %w", err)
		}
	}
	fmt.Fprintln(out, "宿主已注册，等待后台接收预检和心跳")
	return nil
}

func lockHostInstall(controlDir string) (func(), error) {
	if err := os.MkdirAll(controlDir, 0700); err != nil {
		return nil, fmt.Errorf("创建宿主控制目录: %w", err)
	}
	return control.Lock(filepath.Join(controlDir, "install.lock"))
}

func verifiedHostProgram(ctx context.Context, client *control.Client, version, arch, checksum string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.URL+"/api/host/v1/releases/"+version+"/"+arch+"/manifest.json", nil)
	if err != nil {
		return nil, err
	}
	response, err := client.HTTP.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, (16<<10)+1))
	var manifest struct {
		Version      string            `json:"version"`
		Architecture string            `json:"architecture"`
		Files        map[string]string `json:"files"`
	}
	if err != nil || response.StatusCode != 200 || len(raw) > 16<<10 || control.Digest(raw) != checksum ||
		json.Unmarshal(raw, &manifest) != nil || manifest.Version != version || manifest.Architecture != arch || len(manifest.Files) != 2 {
		return nil, errors.New("宿主发行清单校验失败")
	}
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	program, err := os.ReadFile(self)
	if err != nil {
		return nil, err
	}
	digest := control.Digest(program)
	if manifest.Files["installer"] != digest || (manifest.Files["aibertai-engine"] != digest && manifest.Files["monkeyai-engine"] != digest) {
		return nil, errors.New("已下载的宿主程序与发行清单不符")
	}
	return program, nil
}
