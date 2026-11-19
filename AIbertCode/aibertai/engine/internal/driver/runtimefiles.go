package driver

import (
	"context"
	"errors"
	"path/filepath"

	"aibertai.local/engine/internal/lifecycle"
	"aibertai.local/engine/internal/standalone"
)

func runtimeInstalled(spec lifecycle.Spec, files standalone.Files) bool {
	pkg := spec.RuntimePackage
	return pkg != nil && pkg.SourceRuntime() && files.Version == pkg.Version &&
		filepath.Base(files.Binary) == "aibertai-native"
}

func (d *Driver) runtimeFiles(ctx context.Context, spec lifecycle.Spec) (standalone.Files, error) {
	if d.runtimeSource == nil || spec.RuntimePackage == nil || !spec.RuntimePackage.SourceRuntime() {
		return standalone.Files{}, errors.New("请先迁移到 AIbertAI 源码运行时")
	}
	return d.runtimeSource(ctx, spec)
}
func (d *Driver) UpdateRuntime(ctx context.Context, spec lifecycle.Spec) error {
	if spec.DesiredState != lifecycle.DesiredStopped || spec.RuntimePackage == nil || !spec.PersistentContainer || d.runtimeUpdater == nil {
		return errors.New("runtime 更新需要已停止的持久容器")
	}
	container, err := d.load(ctx, spec, "")
	if err != nil {
		return err
	}
	if container.State.Running || container.State.Paused || container.State.Dead || container.State.Status != "created" && container.State.Status != "exited" {
		return errors.New("容器未确认停止，拒绝更新 runtime")
	}
	return d.runtimeUpdater(ctx, spec)
}

func (d *Driver) RuntimeVersion(ctx context.Context, spec lifecycle.Spec) (string, error) {
	files, err := d.runtimeFiles(ctx, spec)
	return files.Version, err
}
