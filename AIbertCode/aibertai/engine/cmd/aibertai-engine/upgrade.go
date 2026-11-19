package main

import (
	"aibertai.local/engine/internal/hostpaths"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"aibertai.local/engine/internal/control"
)

type hostUpgrade struct {
	config, unit, program, journal string
	restart                        func(context.Context) error
}

type hostUpgradeBackup struct {
	Config, Unit []byte
	HadProgram   bool
}

func (u hostUpgrade) recover(ctx context.Context) error {
	raw, err := os.ReadFile(u.journal)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	var backup hostUpgradeBackup
	if err != nil || json.Unmarshal(raw, &backup) != nil || len(backup.Config) == 0 || len(backup.Unit) == 0 {
		return errors.New("宿主升级恢复记录不可读")
	}
	if backup.HadProgram {
		data, err := os.ReadFile(u.journal + ".program")
		if err != nil {
			return err
		}
		if err := control.Write(u.program, data, 0755); err != nil {
			return err
		}
	} else if err := os.Remove(u.program); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := control.Write(u.config, backup.Config, 0600); err != nil {
		return err
	}
	if err := control.Write(u.unit, backup.Unit, 0644); err != nil {
		return err
	}
	if err := u.restart(ctx); err != nil {
		return err
	}
	return u.finish()
}

func (u hostUpgrade) finish() error {
	if err := os.Remove(u.journal); err != nil {
		return err
	}
	_ = os.Remove(u.journal + ".program")
	return nil
}

func (u hostUpgrade) apply(ctx context.Context, config, unit, program []byte, confirm func(context.Context) error) (failure error) {
	oldConfig, err := os.ReadFile(u.config)
	if err != nil {
		return err
	}
	oldUnit, err := os.ReadFile(u.unit)
	if err != nil {
		return err
	}
	oldProgram, err := os.ReadFile(u.program)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	hadProgram := err == nil
	if bytes.Equal(oldConfig, config) && bytes.Equal(oldUnit, unit) && bytes.Equal(oldProgram, program) {
		return nil
	}
	if hadProgram {
		if err := control.Write(u.journal+".program", oldProgram, 0600); err != nil {
			return err
		}
	}
	backup, err := json.Marshal(hostUpgradeBackup{oldConfig, oldUnit, hadProgram})
	if err != nil {
		return err
	}
	if err := control.Write(u.journal, backup, 0600); err != nil {
		return err
	}
	defer func() {
		if failure != nil {
			recovery, cancel := context.WithTimeout(context.WithoutCancel(ctx), 45*time.Second)
			defer cancel()
			failure = errors.Join(failure, u.recover(recovery))
		}
	}()
	if err := control.Write(u.program, program, 0755); err != nil {
		return err
	}
	if err := control.Write(u.config, config, 0600); err != nil {
		return err
	}
	if err := control.Write(u.unit, unit, 0644); err != nil {
		return err
	}
	if err := u.restart(ctx); err != nil {
		return err
	}
	if err := confirm(ctx); err != nil {
		return err
	}
	return u.finish()
}

func upgradePreparedHost(ctx context.Context, args []string, out, stderr io.Writer) error {
	flags := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	flags.SetOutput(stderr)
	server := flags.String("server", "", "已有后台标识")
	release := flags.String("release", "", "目标版本")
	arch := flags.String("architecture", "", "目标架构")
	checksum := flags.String("manifest-sha256", "", "发行清单摘要")
	if flags.Parse(args) != nil || flags.NArg() != 0 || *server == "" || *arch != runtime.GOARCH ||
		*release != version || !hostReleaseVersion.MatchString(version) || version == "latest" || len(*checksum) != 64 {
		return errors.New("宿主升级参数无效")
	}
	if runtime.GOOS != "linux" || os.Geteuid() != 0 {
		return errors.New("需要在 Linux 宿主以 root 升级")
	}
	unlock, err := control.Lock(filepath.Join(filepath.Dir(control.KeyPath), "install.lock"))
	if err != nil {
		return err
	}
	defer unlock()
	if _, err := os.Lstat(control.UpgradePath); !errors.Is(err, os.ErrNotExist) {
		return errors.New("宿主存在待恢复的旧升级记录")
	}
	u := hostUpgrade{config: control.ConfigPath, unit: preparedHostUnit, program: installedProgram, journal: control.SourceUpgradePath}
	u.restart = func(ctx context.Context) error {
		for _, args := range [][]string{{"daemon-reload"}, {"restart", hostpaths.Unit}} {
			cmd := exec.CommandContext(ctx, "systemctl", args...)
			cmd.Stdout, cmd.Stderr = out, stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("重启宿主服务失败: %w", err)
			}
		}
		return nil
	}
	if err := u.recover(ctx); err != nil {
		return err
	}
	raw, err := os.ReadFile(u.config)
	var cfg control.Config
	if err != nil || json.Unmarshal(raw, &cfg) != nil || cfg.HostID == "" || cfg.Server != *server || cfg.Architecture != *arch {
		return errors.New("升级目标与现有宿主绑定不一致")
	}
	if _, err := os.Lstat(control.KeyPath); err != nil {
		return errors.New("缺少宿主原有私钥")
	}
	key, err := control.Key(control.KeyPath)
	if err != nil {
		return err
	}
	client, err := cfg.Client(key)
	if err != nil {
		return err
	}
	program, err := verifiedHostProgram(ctx, client, version, *arch, *checksum)
	if err != nil {
		return err
	}
	unit, err := os.ReadFile(u.unit)
	if err != nil {
		return err
	}
	lines := strings.Split(string(unit), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "ExecStart=") {
			if found {
				return errors.New("宿主服务存在多个启动命令")
			}
			lines[i], found = "ExecStart="+installedProgram+" serve", true
		}
	}
	if !found {
		return errors.New("宿主服务缺少启动命令")
	}
	if cfg.Version != version {
		var config map[string]json.RawMessage
		if err := json.Unmarshal(raw, &config); err != nil {
			return err
		}
		config["version"], _ = json.Marshal(version)
		raw, err = json.Marshal(config)
		if err != nil {
			return err
		}
	}
	start := time.Now()
	err = u.apply(ctx, raw, []byte(strings.Join(lines, "\n")), program, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
		defer cancel()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			raw, err := os.ReadFile(control.HealthPath)
			var health control.Health
			if err == nil && json.Unmarshal(raw, &health) == nil && health.Version == version && health.HostID == cfg.HostID && health.At.After(start) && health.PID > 0 {
				pid, err := exec.CommandContext(ctx, "systemctl", "show", hostpaths.Unit, "--property=MainPID", "--value").Output()
				if err == nil && strings.TrimSpace(string(pid)) == strconv.Itoa(health.PID) {
					return nil
				}
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("未收到升级后宿主的健康心跳: %w", ctx.Err())
			case <-ticker.C:
			}
		}
	})
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "AIbertAI 宿主升级已确认")
	return nil
}
