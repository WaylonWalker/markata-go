package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// TestPublishFeedsPlugin_Name tests the plugin name.
func TestPublishFeedsPlugin_Name(t *testing.T) {
	p := NewPublishFeedsPlugin()
	if got := p.Name(); got != "publish_feeds" {
		t.Errorf("Name() = %q, want %q", got, "publish_feeds")
	}
}

// TestPublishFeedsPlugin_FormatRedirectsCreateDirectories tests that .md, .txt, and .json redirects
// create slug.ext/index.html files, allowing /slug.ext URLs to serve the redirect.
// This ensures web servers serve the redirect as HTML at /slug.ext (without trailing slash).
// Fixes: https://github.com/WaylonWalker/markata-go/issues/248
func TestPublishFeedsPlugin_FormatRedirectsCreateDirectories(t *testing.T) {
	tempDir := t.TempDir()
	plugin := NewPublishFeedsPlugin()

	// Create manager with feed configs in cache
	m := lifecycle.NewManager()
	cfg := m.Config()
	cfg.OutputDir = tempDir
	cfg.Extra = map[string]interface{}{
		"url":   "https://example.com",
		"title": "Test Site",
	}

	// Create a feed config with all format types enabled
	feedConfigs := []models.FeedConfig{
		{
			Slug:        "archive",
			Title:       "Archive",
			Description: "All posts",
			Posts:       []*models.Post{},
			Pages:       []models.FeedPage{},
			Formats: models.FeedFormats{
				HTML:     true,
				RSS:      true,
				Atom:     true,
				JSON:     true,
				Markdown: true,
				Text:     true,
			},
		},
	}
	m.Cache().Set("feed_configs", feedConfigs)

	// Run write
	if err := plugin.Write(m); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Test cases for each format redirect
	tests := []struct {
		name       string
		dirPath    string
		targetFile string
	}{
		{
			name:       "markdown redirect creates directory",
			dirPath:    filepath.Join(tempDir, "archive.md"),
			targetFile: "index.md",
		},
		{
			name:       "text redirect creates directory",
			dirPath:    filepath.Join(tempDir, "archive.txt"),
			targetFile: "index.txt",
		},
		{
			name:       "json redirect creates directory",
			dirPath:    filepath.Join(tempDir, "archive.json"),
			targetFile: "feed.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check that the path is a directory, not a file
			info, err := os.Stat(tt.dirPath)
			if err != nil {
				t.Fatalf("redirect path %s not found: %v", tt.dirPath, err)
			}

			if !info.IsDir() {
				t.Errorf("expected %s to be a directory, but it's a file", tt.dirPath)
			}

			// Check that index.html exists inside the directory
			indexPath := filepath.Join(tt.dirPath, "index.html")
			indexInfo, err := os.Stat(indexPath)
			if err != nil {
				t.Fatalf("index.html not found in %s: %v", tt.dirPath, err)
			}

			if indexInfo.IsDir() {
				t.Errorf("expected %s to be a file, but it's a directory", indexPath)
			}

			// Read content and verify it's valid redirect HTML
			content, err := os.ReadFile(indexPath)
			if err != nil {
				t.Fatalf("failed to read %s: %v", indexPath, err)
			}

			contentStr := string(content)
			if !strings.Contains(contentStr, "<!DOCTYPE html>") {
				t.Error("redirect file should contain DOCTYPE")
			}
			if !strings.Contains(contentStr, "meta http-equiv=\"refresh\"") {
				t.Error("redirect file should contain meta refresh")
			}
			expectedTarget := "/archive/" + tt.targetFile
			if !strings.Contains(contentStr, expectedTarget) {
				t.Errorf("redirect file should point to %s, got: %s", expectedTarget, contentStr)
			}
		})
	}
}

// TestPublishFeedsPlugin_RootFeedNoRedirects tests that root feeds (empty slug) don't create redirects.
// Root feeds write to the output root, so there's no slug to redirect from.
func TestPublishFeedsPlugin_RootFeedNoRedirects(t *testing.T) {
	tempDir := t.TempDir()
	plugin := NewPublishFeedsPlugin()

	// Create manager with root feed config
	m := lifecycle.NewManager()
	cfg := m.Config()
	cfg.OutputDir = tempDir
	cfg.Extra = map[string]interface{}{
		"url":   "https://example.com",
		"title": "Test Site",
	}

	// Create a root feed config (empty slug) with formats enabled
	feedConfigs := []models.FeedConfig{
		{
			Slug:        "", // Root feed
			Title:       "Home",
			Description: "Home page",
			Posts:       []*models.Post{},
			Pages:       []models.FeedPage{},
			Formats: models.FeedFormats{
				Markdown: true,
				Text:     true,
				JSON:     true,
			},
		},
	}
	m.Cache().Set("feed_configs", feedConfigs)

	// Run write
	if err := plugin.Write(m); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify that no redirect directories were created at root level
	// (They would be named ".md", ".txt", ".json" which doesn't make sense)
	redirectPaths := []string{
		filepath.Join(tempDir, ".md"),
		filepath.Join(tempDir, ".txt"),
		filepath.Join(tempDir, ".json"),
	}

	for _, path := range redirectPaths {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("root feed should not create redirect at %s", path)
		}
	}

	// Verify the actual format files were created at root
	expectedFiles := []string{
		filepath.Join(tempDir, "index.md"),
		filepath.Join(tempDir, "index.txt"),
		filepath.Join(tempDir, "feed.json"),
	}

	for _, path := range expectedFiles {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s not found: %v", path, err)
		}
	}
}

// TestPublishFeedsPlugin_writeFeedFormatRedirect tests the redirect file generation directly.
func TestPublishFeedsPlugin_writeFeedFormatRedirect(t *testing.T) {
	tempDir := t.TempDir()
	plugin := NewPublishFeedsPlugin()

	tests := []struct {
		name       string
		slug       string
		ext        string
		targetFile string
		wantURL    string
	}{
		{
			name:       "markdown redirect",
			slug:       "archive",
			ext:        "md",
			targetFile: "index.md",
			wantURL:    "/archive/index.md",
		},
		{
			name:       "text redirect",
			slug:       "blog",
			ext:        "txt",
			targetFile: "index.txt",
			wantURL:    "/blog/index.txt",
		},
		{
			name:       "json redirect",
			slug:       "posts",
			ext:        "json",
			targetFile: "feed.json",
			wantURL:    "/posts/feed.json",
		},
		{
			name:       "nested slug redirect",
			slug:       "tags/python",
			ext:        "md",
			targetFile: "index.md",
			wantURL:    "/tags/python/index.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh subdirectory for each test
			testDir := filepath.Join(tempDir, tt.name)
			if err := os.MkdirAll(testDir, 0o755); err != nil {
				t.Fatalf("failed to create test dir: %v", err)
			}

			// Write redirect
			if err := plugin.writeFeedFormatRedirect(tt.slug, tt.ext, tt.targetFile, testDir); err != nil {
				t.Fatalf("writeFeedFormatRedirect() error = %v", err)
			}

			// Verify redirect directory was created
			redirectDir := filepath.Join(testDir, tt.slug+"."+tt.ext)
			info, err := os.Stat(redirectDir)
			if err != nil {
				t.Fatalf("redirect directory not found: %v", err)
			}
			if !info.IsDir() {
				t.Error("expected redirect path to be a directory")
			}

			// Verify index.html exists
			indexPath := filepath.Join(redirectDir, "index.html")
			content, err := os.ReadFile(indexPath)
			if err != nil {
				t.Fatalf("failed to read index.html: %v", err)
			}

			contentStr := string(content)

			// Verify HTML structure
			if !strings.Contains(contentStr, "<!DOCTYPE html>") {
				t.Error("missing DOCTYPE")
			}
			if !strings.Contains(contentStr, "meta http-equiv=\"refresh\"") {
				t.Error("missing meta refresh")
			}
			if !strings.Contains(contentStr, tt.wantURL) {
				t.Errorf("redirect should point to %s, got: %s", tt.wantURL, contentStr)
			}
			if !strings.Contains(contentStr, `rel="canonical"`) {
				t.Error("missing canonical link")
			}
		})
	}
}
