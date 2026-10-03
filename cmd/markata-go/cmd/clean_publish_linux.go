//go:build linux

package cmd

import "golang.org/x/sys/unix"

func exchangeOutputDirectories(stagedOutput, finalOutput string) error {
	err := unix.Renameat2(
		unix.AT_FDCWD,
		stagedOutput,
		unix.AT_FDCWD,
		finalOutput,
		unix.RENAME_EXCHANGE,
	)
	switch err {
	case nil:
		return nil
	case unix.ENOSYS, unix.EINVAL, unix.EOPNOTSUPP, unix.EXDEV:
		return errAtomicExchangeUnsupported
	default:
		return err
	}
}
