package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validateOutputRoot verifies that an output directory and its existing
// ancestors are real directories. It rejects symlinks before any writer can
// create or remove files below the configured output root.
func validateOutputRoot(outputRoot string) error {
	root, err := filepath.Abs(outputRoot)
	if err != nil {
		return err
	}
	root = filepath.Clean(root)

	info, err := os.Lstat(root)
	switch {
	case err == nil:
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output root is a symlink: %s", root)
		}
		if !info.IsDir() {
			return fmt.Errorf("output root is not a directory: %s", root)
		}
	case os.IsNotExist(err):
		// The root will be created by the caller. Existing ancestors are
		// still checked below so a symlink cannot redirect that creation.
	default:
		return fmt.Errorf("inspect output root %s: %w", root, err)
	}

	return validateExistingOutputPath(root)
}

// openOutputRoot creates the configured output directory when needed and
// returns a descriptor-backed root for race-resistant output operations.
func openOutputRoot(outputRoot string) (*os.Root, error) {
	if err := validateOutputRoot(outputRoot); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create output root: %w", err)
	}
	if err := validateOutputRoot(outputRoot); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(outputRoot)
	if err != nil {
		return nil, fmt.Errorf("open output root: %w", err)
	}
	if err := validateOutputRoot(outputRoot); err != nil {
		_ = root.Close()
		return nil, err
	}
	return root, nil
}

// openExistingOutputRoot opens an output directory without creating it. It
// returns nil when the directory no longer exists, which lets cleanup remain a
// no-op for already-removed output trees.
func openExistingOutputRoot(outputRoot string) (*os.Root, error) {
	rootPath, err := filepath.Abs(outputRoot)
	if err != nil {
		return nil, err
	}
	if err := validateOutputRoot(rootPath); err != nil {
		return nil, err
	}
	if _, err := os.Lstat(rootPath); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, fmt.Errorf("open output root: %w", err)
	}
	if err := validateOutputRoot(rootPath); err != nil {
		_ = root.Close()
		return nil, err
	}
	return root, nil
}

// validateOutputPath verifies lexical containment below outputRoot and
// rejects symlinks or non-directory components in every existing path
// component, including outputRoot itself.
func validateOutputPath(outputRoot, path string) error {
	root, err := filepath.Abs(outputRoot)
	if err != nil {
		return err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	root = filepath.Clean(root)
	absolute = filepath.Clean(absolute)
	relative, err := filepath.Rel(root, absolute)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return fmt.Errorf("path escapes output root: %s", path)
	}
	return validateExistingOutputPath(absolute)
}

// outputRelativePath validates path and returns its name relative to the
// descriptor-backed output root.
func outputRelativePath(outputRoot, path string) (string, error) {
	if err := validateOutputPath(outputRoot, path); err != nil {
		return "", err
	}
	root, err := filepath.Abs(outputRoot)
	if err != nil {
		return "", err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(absolute))
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", fmt.Errorf("path escapes output root: %s", path)
	}
	return relative, nil
}

// validateExistingOutputPath walks upward through all existing components so
// it also detects a symlink in an ancestor of a not-yet-created path.
func validateExistingOutputPath(path string) error {
	cleanPath := filepath.Clean(path)
	current := cleanPath
	for {
		info, err := os.Lstat(current)
		switch {
		case err == nil:
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("output path traverses a symlink: %s", current)
			}
			if current != cleanPath && !info.IsDir() {
				return fmt.Errorf("output path component is not a directory: %s", current)
			}
		case os.IsNotExist(err):
			// Continue to an existing ancestor. A missing component cannot
			// itself redirect the path.
		default:
			return fmt.Errorf("inspect output path %s: %w", current, err)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}
