//go:build !windows

package builderadmin

import (
	"os"
	"os/exec"
	"syscall"
)

func tryLockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

func processExitCode(exitErr *exec.ExitError) (int, bool) {
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	return status.ExitStatus(), ok
}

func unlockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
