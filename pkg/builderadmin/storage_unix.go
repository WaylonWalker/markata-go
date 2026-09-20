//go:build linux || darwin || freebsd || openbsd || netbsd

package builderadmin

import "syscall"

func diskFreeBytes(path string) (uint64, bool, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, true, err
	}
	return stat.Bavail * uint64(stat.Bsize), true, nil
}
