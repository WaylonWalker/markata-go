//go:build windows

package builderadmin

import (
	"os"
	"os/exec"
)

// Windows test and development runs use a single process, so an open lock file is sufficient.
func tryLockFile(_ *os.File) error { return nil }

func unlockFile(_ *os.File) error { return nil }

func processExitCode(exitErr *exec.ExitError) (int, bool) {
	return exitErr.ExitCode(), true
}
