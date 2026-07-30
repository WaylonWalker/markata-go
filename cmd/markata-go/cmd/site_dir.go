package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const siteDirEnv = "MARKATA_GO_SITE_DIR"

var (
	// siteDir selects the site directory for this command invocation.
	siteDir string

	activeSiteDir    string
	activeContentDir string
	siteDirSelected  bool
)

// activateSiteDir selects the command's site directory before site files are
// loaded. It changes the process working directory because several commands
// intentionally resolve caches and other operational files from the site root.
func activateSiteDir() error {
	requested, selected := requestedSiteDir()
	if !selected {
		activeSiteDir = ""
		activeContentDir = ""
		siteDirSelected = false
		return nil
	}

	absPath, err := filepath.Abs(requested)
	if err != nil {
		return fmt.Errorf("resolve site directory %q: %w", requested, err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("site directory %q: %w", requested, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("site directory %q is not a directory", requested)
	}
	if err := os.Chdir(absPath); err != nil {
		return fmt.Errorf("change to site directory %q: %w", absPath, err)
	}

	activeSiteDir = absPath
	activeContentDir = ""
	siteDirSelected = true
	return nil
}

// setSourcePathBaseDir records the content root used by the active manager.
// Post paths are relative to this directory, which can differ from the selected
// site when an explicitly configured file lives in a subdirectory.
func setSourcePathBaseDir(contentDir string) {
	if !siteDirSelected {
		return
	}
	absPath, err := filepath.Abs(contentDir)
	if err == nil {
		activeContentDir = absPath
	}
}

func requestedSiteDir() (string, bool) {
	if strings.TrimSpace(siteDir) != "" {
		return siteDir, true
	}
	if envDir := strings.TrimSpace(os.Getenv(siteDirEnv)); envDir != "" {
		return envDir, true
	}
	return "", false
}

// sourcePathForOutput returns an absolute source path only when a site was
// explicitly selected. Existing current-directory output remains relative.
func sourcePathForOutput(path string) string {
	if !siteDirSelected || path == "" || filepath.IsAbs(path) {
		return path
	}
	baseDir := activeContentDir
	if baseDir == "" {
		baseDir = activeSiteDir
	}
	return filepath.Join(baseDir, path)
}
