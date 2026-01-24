package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// TestPublishHTMLPlugin_ShadowPages tests that unpublished posts are rendered as shadow pages.
func TestPublishHTMLPlugin_ShadowPages(t *testing.T) {
	tests := []struct {
		name          string
		post          *models.Post
		wantRendered  bool
		wantInSitemap bool
	}{
		{
			name: "published post is rendered",
			post: &models.Post{
				Path:        "published.md",
				Slug:        "published-post",
				HTML:        "<p>Published content</p>",
				Published:   true,
				Draft:       false,
				Skip:        false,
				ArticleHTML: "<p>Published content</p>",
			},
			wantRendered:  true,
			wantInSitemap: true,
		},
		{
			name: "unpublished post is rendered as shadow page",
			post: &models.Post{
				Path:        "unpublished.md",
				Slug:        "shadow-post",
				HTML:        "<p>Shadow content</p>",
				Published:   false,
				Draft:       false,
				Skip:        false,
				ArticleHTML: "<p>Shadow content</p>",
			},
			wantRendered:  true,
			wantInSitemap: false,
		},
		{
			name: "draft post is not rendered",
			post: &models.Post{
				Path:        "draft.md",
				Slug:        "draft-post",
				HTML:        "<p>Draft content</p>",
				Published:   false,
				Draft:       true,
				Skip:        false,
				ArticleHTML: "<p>Draft content</p>",
			},
			wantRendered:  false,
			wantInSitemap: false,
		},
		{
			name: "skipped post is not rendered",
			post: &models.Post{
				Path:        "skipped.md",
				Slug:        "skipped-post",
				HTML:        "<p>Skipped content</p>",
				Published:   true,
				Draft:       false,
				Skip:        true,
				ArticleHTML: "<p>Skipped content</p>",
			},
			wantRendered:  false,
			wantInSitemap: false,
		},
		{
			name: "published draft is not rendered",
			post: &models.Post{
				Path:        "published-draft.md",
				Slug:        "published-draft",
				HTML:        "<p>Published draft content</p>",
				Published:   true,
				Draft:       true,
				Skip:        false,
				ArticleHTML: "<p>Published draft content</p>",
			},
			wantRendered:  false,
			wantInSitemap: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir := t.TempDir()

			// Create plugin
			plugin := NewPublishHTMLPlugin()

			// Create config
			config := &lifecycle.Config{
				OutputDir: tempDir,
				Extra:     make(map[string]interface{}),
			}

			// Write post
			err := plugin.writePost(tt.post, config)
			if err != nil {
				t.Fatalf("writePost() error = %v", err)
			}

			// Check if file was created
			outputPath := filepath.Join(tempDir, tt.post.Slug, "index.html")
			_, statErr := os.Stat(outputPath)
			wasRendered := statErr == nil

			if wasRendered != tt.wantRendered {
				t.Errorf("post rendered = %v, want %v", wasRendered, tt.wantRendered)
			}
		})
	}
}

// TestPublishHTMLPlugin_OGCardCanonicalURL tests that OG cards include canonical URL and robots meta.
func TestPublishHTMLPlugin_OGCardCanonicalURL(t *testing.T) {
	tempDir := t.TempDir()
	plugin := NewPublishHTMLPlugin()

	// Create config with OG format enabled and a site URL
	config := &lifecycle.Config{
		OutputDir: tempDir,
		Extra: map[string]interface{}{
			"url":          "https://example.com",
			"title":        "Test Site",
			"post_formats": models.PostFormatsConfig{OG: true},
		},
	}

	// Create test post
	title := "Test Post Title"
	post := &models.Post{
		Path:        "test.md",
		Slug:        "test-post",
		Title:       &title,
		HTML:        "<html><body>Test content</body></html>",
		Published:   true,
		Draft:       false,
		Skip:        false,
		ArticleHTML: "<p>Test content</p>",
	}

	// Write post (which includes OG format)
	if err := plugin.writePost(post, config); err != nil {
		t.Fatalf("writePost() error = %v", err)
	}

	// Read OG card content
	ogPath := filepath.Join(tempDir, "test-post", "og", "index.html")
	content, err := os.ReadFile(ogPath)
	if err != nil {
		t.Fatalf("failed to read OG card: %v", err)
	}

	ogHTML := string(content)

	// Verify canonical URL is present
	expectedCanonical := `<link rel="canonical" href="https://example.com/test-post/">`
	if !strings.Contains(ogHTML, expectedCanonical) {
		t.Errorf("OG card should contain canonical URL.\nExpected: %s\nGot: %s", expectedCanonical, ogHTML)
	}

	// Verify robots meta is present
	expectedRobots := `<meta name="robots" content="noindex, nofollow">`
	if !strings.Contains(ogHTML, expectedRobots) {
		t.Errorf("OG card should contain robots meta.\nExpected: %s\nGot: %s", expectedRobots, ogHTML)
	}
}

// TestPublishHTMLPlugin_ShadowPagesDocumentation tests the expected behavior is documented.
func TestPublishHTMLPlugin_ShadowPagesDocumentation(t *testing.T) {
	// This test documents the shadow pages behavior:
	// - published: true → rendered + in feeds + in sitemap
	// - published: false → rendered (shadow page) + NOT in feeds + NOT in sitemap
	// - draft: true → NOT rendered (regardless of published status)
	// - skip: true → NOT rendered (regardless of published status)

	tempDir := t.TempDir()
	plugin := NewPublishHTMLPlugin()
	config := &lifecycle.Config{
		OutputDir: tempDir,
		Extra:     make(map[string]interface{}),
	}

	// Shadow page scenario
	shadowPost := &models.Post{
		Path:        "shadow.md",
		Slug:        "shadow-page",
		HTML:        "<html><body>Shadow content accessible via direct URL</body></html>",
		Published:   false, // Not in feeds
		Draft:       false, // Not a draft, so it will be rendered
		Skip:        false,
		ArticleHTML: "<p>Shadow content accessible via direct URL</p>",
	}

	if err := plugin.writePost(shadowPost, config); err != nil {
		t.Fatalf("writePost() error = %v", err)
	}

	// Verify shadow page was created
	outputPath := filepath.Join(tempDir, "shadow-page", "index.html")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Shadow page should be rendered even though published=false")
	}

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read shadow page: %v", err)
	}

	if len(content) == 0 {
		t.Error("Shadow page content should not be empty")
	}
}

// TestPublishHTMLPlugin_FormatRedirectsCreateDirectories tests that .md and .txt redirects
// create slug.ext/index.html files, allowing /slug.ext URLs to serve the redirect.
// This ensures web servers serve the redirect as HTML at /slug.ext (without trailing slash).
// Fixes: https://github.com/WaylonWalker/markata-go/issues/84
// Related: https://github.com/WaylonWalker/markata-go/issues/160
func TestPublishHTMLPlugin_FormatRedirectsCreateDirectories(t *testing.T) {
	tempDir := t.TempDir()
	plugin := NewPublishHTMLPlugin()

	// Create config with Markdown and Text formats enabled
	htmlEnabled := true
	config := &lifecycle.Config{
		OutputDir: tempDir,
		Extra: map[string]interface{}{
			"post_formats": models.PostFormatsConfig{
				HTML:     &htmlEnabled,
				Markdown: true,
				Text:     true,
			},
		},
	}

	// Create test post
	title := "Test Post"
	post := &models.Post{
		Path:        "test.md",
		Slug:        "test-post",
		Title:       &title,
		Content:     "Test content",
		HTML:        "<html><body>Test</body></html>",
		Published:   true,
		Draft:       false,
		Skip:        false,
		ArticleHTML: "<p>Test</p>",
	}

	// Write post (which includes format redirects)
	if err := plugin.writePost(post, config); err != nil {
		t.Fatalf("writePost() error = %v", err)
	}

	// Test cases for each format redirect
	tests := []struct {
		name    string
		dirPath string
	}{
		{
			name:    "markdown redirect creates directory",
			dirPath: filepath.Join(tempDir, "test-post.md"),
		},
		{
			name:    "text redirect creates directory",
			dirPath: filepath.Join(tempDir, "test-post.txt"),
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
			if !strings.Contains(contentStr, "/test-post/index.") {
				t.Error("redirect file should point to /test-post/index.*")
			}
		})
	}
}
