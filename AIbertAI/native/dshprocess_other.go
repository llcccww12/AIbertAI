//go:build !windows

package main

import (
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
)

// DSH 会起 shell、语言服务和模型自己拉起的进程。只杀 DSH 本身，这些子进程会
// 留在后台继续占着端口和文件句柄，退出应用也清不掉。让它自成进程组，关闭时
// 整组一起收掉。
func dshPrepareProcess(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func dshProcessTree(process *os.Process) func() {
	if process == nil {
		return func() {}
	}
	pid := process.Pid
	return func() {
		// 负号表示整个进程组。DSH 自己被 Kill 之后组还在，留下的子进程一起收掉。
		if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			slog.Warn("结束 DSH 进程组失败", "module", "dsh", "operation", "dshProcessTree", "pid", pid, "error", err)
		}
	}
}
