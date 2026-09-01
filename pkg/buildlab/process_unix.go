//go:build darwin || dragonfly || freebsd || netbsd || openbsd || solaris

package buildlab

import (
	"os/exec"
	"syscall"
)

type unixProcessTree struct {
	pgid int
}

func newProcessTree(cmd *exec.Cmd) (processTree, error) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	return &unixProcessTree{}, nil
}

func (p *unixProcessTree) Start(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	p.pgid = cmd.Process.Pid
	return nil
}

func (p *unixProcessTree) Kill() error {
	if p.pgid == 0 {
		return nil
	}
	err := syscall.Kill(-p.pgid, syscall.SIGKILL)
	if err == syscall.ESRCH {
		return nil
	}
	return err
}

func (p *unixProcessTree) Close() error {
	err := p.Kill()
	p.pgid = 0
	return err
}
