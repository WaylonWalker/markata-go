package builderadmin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const defaultReleaseContentIndex = "content-index.json"

// validateReleaseOutput checks the minimum serving contract required before a
// retained release may become current. The homepage is mandatory. A content
// index is optional here, but when present it must be a non-empty regular JSON
// file so rollback cannot activate an obviously corrupt generated artifact.
func validateReleaseOutput(releaseDir string) error {
	if err := validateReleaseRegularFile(filepath.Join(releaseDir, "index.html"), "homepage"); err != nil {
		return err
	}

	contentIndex := filepath.Join(releaseDir, defaultReleaseContentIndex)
	info, err := os.Lstat(contentIndex)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect release content index: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("release content index is not a regular file: %s", contentIndex)
	}
	if info.Size() == 0 {
		return fmt.Errorf("release content index is empty: %s", contentIndex)
	}
	data, err := os.ReadFile(contentIndex)
	if err != nil {
		return fmt.Errorf("read release content index: %w", err)
	}
	if !json.Valid(data) {
		return fmt.Errorf("release content index is not valid JSON: %s", contentIndex)
	}
	return nil
}

func validateReleaseRegularFile(path, label string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("release %s is missing: %w", label, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("release %s is not a regular file: %s", label, path)
	}
	if info.Size() == 0 {
		return fmt.Errorf("release %s is empty: %s", label, path)
	}
	return nil
}

func validatePendingCurrentRelease(currentNext string) error {
	target, err := os.Readlink(currentNext)
	if err != nil {
		return fmt.Errorf("read pending current release link %q: %w", currentNext, err)
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(currentNext), target)
	}
	if err := validateReleaseOutput(filepath.Clean(target)); err != nil {
		return fmt.Errorf("validate pending current release: %w", err)
	}
	return nil
}
