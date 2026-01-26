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

// TestPublishHTMLPlugin_FormatRedirectsCreateDirectories tests that .md and .txt formats
// use reversed redirects where content is at /slug.ext and redirect is at /slug/index.ext.
// This allows standard web txt files (robots.txt, llms.txt, humans.txt) to be served
// at their canonical URLs.
// Fixes: https://github.com/WaylonWalker/markata-go/issues/395
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

	// Test cases for each format with REVERSED redirect structure
	tests := []struct {
		name         string
		ext          string
		contentPath  string // Primary content file (canonical)
		redirectPath string // Redirect file (backwards compat)
	}{
		{
			name:         "markdown uses reversed redirect",
			ext:          "md",
			contentPath:  filepath.Join(tempDir, "test-post.md"),
			redirectPath: filepath.Join(tempDir, "test-post", "index.md"),
		},
		{
			name:         "text uses reversed redirect",
			ext:          "txt",
			contentPath:  filepath.Join(tempDir, "test-post.txt"),
			redirectPath: filepath.Join(tempDir, "test-post", "index.txt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check that primary content file exists at /slug.ext
			info, err := os.Stat(tt.contentPath)
			if err != nil {
				t.Fatalf("primary content file %s not found: %v", tt.contentPath, err)
			}
			if info.IsDir() {
				t.Errorf("expected %s to be a file, but it's a directory", tt.contentPath)
			}

			// Read content and verify it's actual content (not redirect HTML)
			content, err := os.ReadFile(tt.contentPath)
			if err != nil {
				t.Fatalf("failed to read %s: %v", tt.contentPath, err)
			}
			contentStr := string(content)
			if strings.Contains(contentStr, "<!DOCTYPE html>") {
				t.Error("primary content file should NOT be HTML redirect")
			}
			if !strings.Contains(contentStr, "Test") {
				t.Error("primary content file should contain post content")
			}

			// Check that redirect file exists at /slug/index.ext
			redirectInfo, err := os.Stat(tt.redirectPath)
			if err != nil {
				t.Fatalf("redirect file %s not found: %v", tt.redirectPath, err)
			}
			if redirectInfo.IsDir() {
				t.Errorf("expected %s to be a file, but it's a directory", tt.redirectPath)
			}

			// Read redirect and verify it points to the canonical URL
			redirectContent, err := os.ReadFile(tt.redirectPath)
			if err != nil {
				t.Fatalf("failed to read %s: %v", tt.redirectPath, err)
			}
			redirectStr := string(redirectContent)
			if !strings.Contains(redirectStr, "<!DOCTYPE html>") {
				t.Error("redirect file should contain DOCTYPE")
			}
			if !strings.Contains(redirectStr, "meta http-equiv=\"refresh\"") {
				t.Error("redirect file should contain meta refresh")
			}
			// Redirect should point TO the canonical /slug.ext location
			expectedTarget := "/test-post." + tt.ext
			if !strings.Contains(redirectStr, expectedTarget) {
				t.Errorf("redirect file should point to %s, got: %s", expectedTarget, redirectStr)
			}
		})
	}
}

// TestPublishHTMLPlugin_StandardWebTxtFiles tests that standard web txt files
// (robots.txt, llms.txt, humans.txt) are generated at their expected canonical URLs.
// This is the primary use case for the reversed redirect feature.
// Fixes: https://github.com/WaylonWalker/markata-go/issues/395
func TestPublishHTMLPlugin_StandardWebTxtFiles(t *testing.T) {
	tempDir := t.TempDir()
	plugin := NewPublishHTMLPlugin()

	// Create config with Text format enabled
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

	// Standard web txt files that should be at root level
	standardFiles := []struct {
		slug    string
		title   string
		content string
	}{
		{"robots", "Robots", "User-agent: *\nAllow: /"},
		{"llms", "LLMs", "# LLMs.txt\n\nThis site allows AI training."},
		{"humans", "Humans", "/* TEAM */\nDeveloper: Test"},
	}

	for _, sf := range standardFiles {
		title := sf.title
		post := &models.Post{
			Path:        sf.slug + ".md",
			Slug:        sf.slug,
			Title:       &title,
			Content:     sf.content,
			HTML:        "<html><body>Test</body></html>",
			Published:   true,
			Draft:       false,
			Skip:        false,
			ArticleHTML: "<p>Test</p>",
		}

		if err := plugin.writePost(post, config); err != nil {
			t.Fatalf("writePost() error for %s: %v", sf.slug, err)
		}
	}

	// Verify standard web files are at canonical URLs
	for _, sf := range standardFiles {
		t.Run(sf.slug+".txt at root", func(t *testing.T) {
			// Check /robots.txt, /llms.txt, /humans.txt exist at root
			canonicalPath := filepath.Join(tempDir, sf.slug+".txt")
			content, err := os.ReadFile(canonicalPath)
			if err != nil {
				t.Fatalf("standard web file %s.txt not found at root: %v", sf.slug, err)
			}

			// Should contain actual content, not redirect HTML
			contentStr := string(content)
			if strings.Contains(contentStr, "<!DOCTYPE html>") {
				t.Errorf("%s.txt should be content, not HTML redirect", sf.slug)
			}
		})
	}
}

// TestPublishHTMLPlugin_PostFormatsDefaultEnabled tests that Markdown and Text
// formats are enabled by default.
// Fixes: https://github.com/WaylonWalker/markata-go/issues/395
func TestPublishHTMLPlugin_PostFormatsDefaultEnabled(t *testing.T) {
	defaults := models.NewPostFormatsConfig()

	if !defaults.IsHTMLEnabled() {
		t.Error("HTML should be enabled by default")
	}

	if !defaults.Markdown {
		t.Error("Markdown should be enabled by default")
	}

	if !defaults.Text {
		t.Error("Text should be enabled by default")
	}

	if defaults.OG {
		t.Error("OG should NOT be enabled by default")
	}
}
