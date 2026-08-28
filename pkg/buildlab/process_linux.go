//go:build linux

package buildlab

import (
	"os/exec"
	"syscall"
)

func configureProcessGroup(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return nil
}

func killProcessGroup(pid int) error { return syscall.Kill(-pid, syscall.SIGKILL) }

func killProcess(pid int) error { return syscall.Kill(pid, syscall.SIGKILL) }
