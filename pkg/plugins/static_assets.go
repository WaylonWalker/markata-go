package plugins

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/example/markata-go/pkg/lifecycle"
)

// StaticAssetsPlugin copies static assets from themes and project directories to output.
// It handles:
// 1. Theme static files (themes/[theme]/static/*)
// 2. Project static files (static/*)
// Project files take precedence over theme files (local override).
type StaticAssetsPlugin struct{}

// NewStaticAssetsPlugin creates a new StaticAssetsPlugin.
func NewStaticAssetsPlugin() *StaticAssetsPlugin {
	return &StaticAssetsPlugin{}
}

// Name returns the unique name of the plugin.
func (p *StaticAssetsPlugin) Name() string {
	return "static_assets"
}

// Write copies static assets to the output directory.
func (p *StaticAssetsPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	outputDir := config.OutputDir

	// Get theme name from config (default to "default")
	themeName := "default"
	if extra := config.Extra; extra != nil {
		if theme, ok := extra["theme"].(map[string]interface{}); ok {
			if name, ok := theme["name"].(string); ok && name != "" {
				themeName = name
			}
		}
		// Also check for simple theme string
		if name, ok := extra["theme"].(string); ok && name != "" {
			themeName = name
		}
	}

	// Copy theme static files first (lower priority)
	themeStaticDir := filepath.Join("themes", themeName, "static")
	if _, err := os.Stat(themeStaticDir); err == nil {
		if err := p.copyDir(themeStaticDir, outputDir); err != nil {
			return fmt.Errorf("copying theme static files: %w", err)
		}
	}

	// Copy project static files (higher priority, can override theme files)
	projectStaticDir := "static"
	if _, err := os.Stat(projectStaticDir); err == nil {
		if err := p.copyDir(projectStaticDir, outputDir); err != nil {
			return fmt.Errorf("copying project static files: %w", err)
		}
	}

	return nil
}

// copyDir recursively copies a directory to the destination.
func (p *StaticAssetsPlugin) copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path from source
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("calculating relative path: %w", err)
		}

		// Calculate destination path
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(dstPath, 0755)
		}

		// Copy file
		return p.copyFile(path, dstPath)
	})
}

// copyFile copies a single file from src to dst.
func (p *StaticAssetsPlugin) copyFile(src, dst string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copying file content: %w", err)
	}

	// Preserve file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return nil // Non-critical, continue without preserving permissions
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// Priority returns the plugin priority for the write stage.
// Static assets should be written early so that other plugins can reference them.
func (p *StaticAssetsPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageWrite {
		return lifecycle.PriorityEarly
	}
	return lifecycle.PriorityDefault
}

// Ensure StaticAssetsPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin         = (*StaticAssetsPlugin)(nil)
	_ lifecycle.WritePlugin    = (*StaticAssetsPlugin)(nil)
	_ lifecycle.PriorityPlugin = (*StaticAssetsPlugin)(nil)
)
