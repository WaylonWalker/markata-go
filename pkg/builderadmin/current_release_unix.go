//go:build !windows

package builderadmin

import (
	"fmt"
	"os"
)

func replaceCurrentRelease(currentNext, current string) error {
	if err := validatePendingCurrentRelease(currentNext); err != nil {
		return fmt.Errorf("refuse invalid release activation: %w", err)
	}
	return os.Rename(currentNext, current)
}
