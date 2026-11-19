package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"aibertai.local/engine/internal/control"
	"aibertai.local/engine/internal/hostpaths"
)

const hostBrandReceipt = "aibertai-engine host layout v1\n"

type hostBrandPaths struct {
	config, state, unit string
}

func migrateHostBrand(ctx context.Context, rollback bool, out, stderr io.Writer) error {
	if runtime.GOOS != "linux" || os.Geteuid() != 0 {
		return errors.New("需要在 Linux 宿主以 root 迁移")
	}
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return errors.New("宿主需要 systemd")
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	current, currentErr := os.Stat(executable)
	installed, installedErr := os.Stat(installedProgram)
	if currentErr != nil || installedErr != nil || !os.SameFile(current, installed) {
		return errors.New("请通过已安装的新版宿主程序运行迁移")
	}
	unlock, err := control.Lock("/run/lock/aibertai-engine-brand.lock")
	if err != nil {
		return err
	}
	defer unlock()
	oldPaths := hostBrandPaths{hostpaths.LegacyConfigRoot, hostpaths.LegacyStateRoot, "/etc/systemd/system/" + hostpaths.LegacyUnitName}
	newPaths := hostBrandPaths{hostpaths.ConfigRoot, hostpaths.StateRoot, "/etc/systemd/system/" + hostpaths.UnitName}
	return moveHostBrand(oldPaths, newPaths, "/etc/aibertai-engine-brand-v1", rollback, func(args ...string) error {
		cmd := exec.CommandContext(ctx, "systemctl", args...)
		cmd.Stdout, cmd.Stderr = out, stderr
		return cmd.Run()
	})
}

func moveHostBrand(oldPaths, newPaths hostBrandPaths, receipt string, rollback bool, systemctl func(...string) error) error {
	from, to := oldPaths, newPaths
	if rollback {
		from, to = to, from
	}
	config, err := brandPath(from.config, to.config, true, false)
	if err != nil {
		return err
	}
	state, err := brandPath(from.state, to.state, true, true)
	if err != nil {
		return err
	}
	unit, err := brandPath(from.unit, to.unit, false, false)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(config, "config.json"))
	var cfg control.Config
	if err != nil || json.Unmarshal(raw, &cfg) != nil || cfg.HostID == "" {
		return errors.New("缺少已登记的宿主配置")
	}
	if info, err := os.Lstat(filepath.Join(state, "control", "identity.key")); err != nil || !info.Mode().IsRegular() {
		return errors.New("缺少已有宿主私钥")
	}
	unitData, err := os.ReadFile(unit)
	if err != nil || !strings.Contains(string(unitData), "\nExecStart="+installedProgram+" serve\n") {
		return errors.New("请先升级到支持路径迁移的 AIbertAI 宿主程序")
	}
	saved, err := os.ReadFile(receipt)
	if errors.Is(err, os.ErrNotExist) {
		if rollback {
			return errors.New("当前宿主没有可回滚的路径迁移")
		}
		if config != from.config || state != from.state || unit != from.unit {
			return errors.New("当前宿主并非旧布局")
		}
	} else if err != nil || string(saved) != hostBrandReceipt {
		return errors.New("宿主路径迁移记录无效")
	}

	unlock, err := control.Lock(filepath.Join(state, "control", "install.lock"))
	if err != nil {
		return err
	}
	defer unlock()
	for _, file := range []string{"upgrade.json", "source-upgrade.json"} {
		if _, err := os.Lstat(filepath.Join(state, "control", file)); !errors.Is(err, os.ErrNotExist) {
			return errors.New("宿主有待恢复的升级，请先恢复再迁移路径")
		}
	}
	if raw, err := os.ReadFile(filepath.Join(state, "control", "remote-upgrade.json")); err == nil {
		var upgrade struct{ Result control.UpgradeResult }
		if json.Unmarshal(raw, &upgrade) != nil || upgrade.Result.Status == "running" {
			return errors.New("宿主远程升级尚未结束")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if saved == nil {
		if err := control.Write(receipt, []byte(hostBrandReceipt), 0600); err != nil {
			return err
		}
	}
	if config != to.config || state != to.state || unit != to.unit {
		if err := systemctl("disable", "--now", filepath.Base(unit)); err != nil {
			return err
		}
	}
	if state == from.state {
		if err := removeBrandAlias(to.state, from.state); err != nil {
			return err
		}
		if err := os.Rename(from.state, to.state); err != nil {
			return err
		}
	}
	// Docker retains the original absolute host bind sources. Keep an alias to
	// the same data so restarting an existing container preserves its rootfs.
	if alias, err := os.Readlink(from.state); errors.Is(err, os.ErrNotExist) {
		if err := os.Symlink(to.state, from.state); err != nil {
			return err
		}
	} else if err != nil || alias != to.state {
		return errors.New("宿主数据旧路径不能作为兼容链接")
	}
	if config == from.config {
		if err := os.Rename(from.config, to.config); err != nil {
			return err
		}
	}
	if unit == from.unit {
		if err := os.Rename(from.unit, to.unit); err != nil {
			return err
		}
	}
	if err := systemctl("daemon-reload"); err != nil {
		return err
	}
	return systemctl("enable", "--now", filepath.Base(to.unit))
}

func brandPath(from, to string, dir, allowAlias bool) (string, error) {
	exists := func(path, other string) (bool, error) {
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if allowAlias && info.Mode()&os.ModeSymlink != 0 {
			if target, err := os.Readlink(path); err == nil && target == other {
				return false, nil
			}
		}
		if dir && info.IsDir() || !dir && info.Mode().IsRegular() {
			return true, nil
		}
		return false, fmt.Errorf("宿主路径类型不符: %s", path)
	}
	previous, err := exists(from, to)
	if err != nil {
		return "", err
	}
	current, err := exists(to, from)
	if err != nil {
		return "", err
	}
	if previous == current {
		return "", fmt.Errorf("宿主路径冲突或缺失: %s, %s", from, to)
	}
	if previous {
		return from, nil
	}
	return to, nil
}

func removeBrandAlias(path, target string) error {
	alias, err := os.Readlink(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || alias != target {
		return fmt.Errorf("迁移目标路径已被占用: %s", path)
	}
	return os.Remove(path)
}
