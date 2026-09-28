package builderadmin

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// stageWorkspaceRelease copies a completed build workspace into a temporary
// directory on the release filesystem, then atomically renames that directory
// into the final retained release path. This is the safe promotion primitive
// used when build work lives on a different filesystem from SiteDir.
//
// The source workspace is intentionally left intact. The caller owns workspace
// cleanup so a failed promotion can retain the build output for diagnosis.
func stageWorkspaceRelease(workspace, releasesDir, releaseID string) (string, error) {
	if workspace == "" || releasesDir == "" || releaseID == "" {
		return "", fmt.Errorf("workspace, releases directory, and release ID are required")
	}
	if info, err := os.Stat(workspace); err != nil {
		return "", fmt.Errorf("stat build workspace: %w", err)
	} else if !info.IsDir() {
		return "", fmt.Errorf("build workspace is not a directory: %s", workspace)
	}
	if err := os.MkdirAll(releasesDir, 0o755); err != nil {
		return "", fmt.Errorf("create releases directory: %w", err)
	}

	finalPath := filepath.Join(releasesDir, releaseID)
	if _, err := os.Lstat(finalPath); err == nil {
		return "", fmt.Errorf("release already exists: %s", finalPath)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat release destination: %w", err)
	}

	stagingPath, err := os.MkdirTemp(releasesDir, ".staging-"+releaseID+"-")
	if err != nil {
		return "", fmt.Errorf("create release staging directory: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(stagingPath)
		}
	}()

	if err := copyWorkspaceTree(workspace, stagingPath); err != nil {
		return "", fmt.Errorf("stage build workspace: %w", err)
	}
	if err := os.Rename(stagingPath, finalPath); err != nil {
		return "", fmt.Errorf("commit staged release: %w", err)
	}
	committed = true
	return finalPath, nil
}

func copyWorkspaceTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		target := filepath.Join(destination, relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}

		if entry.Type()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			return os.Symlink(linkTarget, target)
		}
		if entry.IsDir() {
			if err := os.MkdirAll(target, info.Mode().Perm()); err != nil {
				return err
			}
			return os.Chtimes(target, info.ModTime(), info.ModTime())
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported workspace entry %s (%s)", path, entry.Type())
		}
		return copyWorkspaceFile(path, target, info.Mode().Perm(), info.ModTime())
	})
}

func copyWorkspaceFile(source, destination string, mode fs.FileMode, modTime time.Time) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	copied := false
	defer func() {
		_ = output.Close()
		if !copied {
			_ = os.Remove(destination)
		}
	}()
	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	if err := output.Sync(); err != nil {
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	if err := os.Chtimes(destination, modTime, modTime); err != nil {
		return err
	}
	copied = true
	return nil
}
