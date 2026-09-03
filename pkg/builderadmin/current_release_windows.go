//go:build windows

package builderadmin

import (
	"fmt"
	"os"
)

func replaceCurrentRelease(currentNext, current string) error {
	// Windows cannot rename a symlink over an existing symlink. Remove only
	// the link itself, then install the already-created replacement link. Keep
	// the old target so a failed replacement does not strand the site without a
	// current release.
	oldTarget, readErr := os.Readlink(current)
	hadCurrent := readErr == nil
	if readErr != nil && !os.IsNotExist(readErr) {
		return fmt.Errorf("read current release link %q: %w", current, readErr)
	}
	if err := os.Remove(current); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove current release link %q: %w", current, err)
	}
	if err := os.Rename(currentNext, current); err != nil {
		if hadCurrent {
			if restoreErr := os.Symlink(oldTarget, current); restoreErr != nil {
				return fmt.Errorf("replace current release link %q: %w (restore previous link: %v)", current, err, restoreErr)
			}
		}
		return fmt.Errorf("rename current release link %q to %q: %w", currentNext, current, err)
	}
	return nil
}
