//go:build !windows

package builderadmin

import "os"

func replaceCurrentRelease(currentNext, current string) error {
	return os.Rename(currentNext, current)
}
