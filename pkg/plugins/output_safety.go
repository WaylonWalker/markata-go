package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validateOutputRoot verifies that an output directory is a real directory.
// When the directory does not exist yet, only its nearest existing ancestor
// is inspected. Symlinks above that boundary are normal on some platforms
// (for example, macOS maps /var to /private/var) and do not redirect paths
// below the configured root.
func validateOutputRoot(outputRoot string) error {
	root, err := filepath.Abs(outputRoot)
	if err != nil {
		return err
	}
	root = filepath.Clean(root)

	current := root
	for {
		info, err := os.Lstat(current)
		switch {
		case err == nil:
			if info.Mode()&os.ModeSymlink != 0 {
				if current == root {
					return fmt.Errorf("output root is a symlink: %s", current)
				}
				resolved, statErr := os.Stat(current)
				if statErr != nil {
					return fmt.Errorf("inspect output root ancestor %s: %w", current, statErr)
				}
				if !resolved.IsDir() {
					return fmt.Errorf("output root ancestor is not a directory: %s", current)
				}
			}
			if info.Mode()&os.ModeSymlink == 0 && !info.IsDir() {
				return fmt.Errorf("output root is not a directory: %s", current)
			}
			return nil
		case !os.IsNotExist(err):
			return fmt.Errorf("inspect output root %s: %w", current, err)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

// openOutputRoot creates the configured output directory when needed and
// returns a descriptor-backed root for race-resistant output operations.
func openOutputRoot(outputRoot string) (*os.Root, error) {
	rootPath, err := filepath.Abs(outputRoot)
	if err != nil {
		return nil, err
	}
	rootPath = filepath.Clean(rootPath)
	if err := validateOutputRoot(rootPath); err != nil {
		return nil, err
	}
	expected, err := os.Lstat(rootPath)
	if os.IsNotExist(err) {
		expected = nil
	} else if err != nil {
		return nil, fmt.Errorf("inspect output root %s: %w", rootPath, err)
	}
	if err := ensureOutputRoot(rootPath); err != nil {
		return nil, fmt.Errorf("create output root: %w", err)
	}
	if err := validateOutputRoot(rootPath); err != nil {
		return nil, err
	}
	return openCheckedOutputRoot(rootPath, expected)
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
	return openCheckedOutputRoot(rootPath, nil)
}

// ensureOutputRoot creates a missing root through the nearest existing
// descriptor-backed ancestor. This avoids using an ambient MkdirAll call for
// the portion of the path that may be controlled by another process.
func ensureOutputRoot(rootPath string) error {
	if info, err := os.Lstat(rootPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output root is a symlink: %s", rootPath)
		}
		if !info.IsDir() {
			return fmt.Errorf("output root is not a directory: %s", rootPath)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect output root %s: %w", rootPath, err)
	}

	ancestor, err := nearestExistingOutputAncestor(rootPath)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(ancestor, rootPath)
	if err != nil || relative == "." || filepath.IsAbs(relative) {
		return fmt.Errorf("resolve output root relative path: %s", rootPath)
	}
	ancestorRoot, err := os.OpenRoot(ancestor)
	if err != nil {
		return fmt.Errorf("open output root ancestor: %w", err)
	}
	defer ancestorRoot.Close()
	return ancestorRoot.MkdirAll(relative, 0o755)
}

func nearestExistingOutputAncestor(path string) (string, error) {
	current := filepath.Clean(path)
	for {
		if _, err := os.Lstat(current); err == nil {
			return current, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect output root ancestor %s: %w", current, err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return current, nil
		}
		current = parent
	}
}

func openCheckedOutputRoot(rootPath string, expected os.FileInfo) (*os.Root, error) {
	checked, err := os.Lstat(rootPath)
	if err != nil {
		return nil, fmt.Errorf("inspect output root %s: %w", rootPath, err)
	}
	if checked.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("output root is a symlink: %s", rootPath)
	}
	if !checked.IsDir() {
		return nil, fmt.Errorf("output root is not a directory: %s", rootPath)
	}
	if expected != nil && !os.SameFile(expected, checked) {
		return nil, fmt.Errorf("output root changed while opening: %s", rootPath)
	}

	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, fmt.Errorf("open output root: %w", err)
	}
	opened, err := root.Stat(".")
	if err != nil {
		_ = root.Close()
		return nil, fmt.Errorf("inspect opened output root: %w", err)
	}
	if !os.SameFile(checked, opened) {
		_ = root.Close()
		return nil, fmt.Errorf("output root changed while opening: %s", rootPath)
	}
	current, err := os.Lstat(rootPath)
	if err != nil {
		_ = root.Close()
		return nil, fmt.Errorf("recheck output root %s: %w", rootPath, err)
	}
	if current.Mode()&os.ModeSymlink != 0 || !current.IsDir() || !os.SameFile(checked, current) {
		_ = root.Close()
		return nil, fmt.Errorf("output root changed while opening: %s", rootPath)
	}
	return root, nil
}

// validateOutputPath verifies lexical containment below outputRoot and
// rejects symlinks or non-directory components in every existing path
// component from outputRoot through the destination, including the root and
// final destination. Ancestors above outputRoot are intentionally ignored.
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
	if err := validateOutputRoot(root); err != nil {
		return err
	}

	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		switch {
		case err == nil:
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("output path traverses a symlink: %s", current)
			}
			if current != absolute && !info.IsDir() {
				return fmt.Errorf("output path component is not a directory: %s", current)
			}
		case os.IsNotExist(err):
			// Missing components cannot redirect the path. The descriptor-backed
			// root creates them below the already-validated output root.
			return nil
		default:
			return fmt.Errorf("inspect output path %s: %w", current, err)
		}
	}
	return nil
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
