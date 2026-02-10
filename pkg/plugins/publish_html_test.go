package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// createTestManager creates a minimal lifecycle manager for testing.
func createTestManager(t *testing.T, config *lifecycle.Config) *lifecycle.Manager {
	t.Helper()
	m := lifecycle.NewManager()
	// Set the config in the manager
	m.SetConfig(config)
	return m
}

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

			// Create manager for testing
			m := createTestManager(t, config)

			// Write post
			err := plugin.writePost(tt.post, config, nil, m)
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

	// Create manager for testing
	m := createTestManager(t, config)

	// Write post (which includes OG format)
	if err := plugin.writePost(post, config, nil, m); err != nil {
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

	// Create manager for testing
	m := createTestManager(t, config)

	if err := plugin.writePost(shadowPost, config, nil, m); err != nil {
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
// use the directory-based structure with redirects at /slug.ext/index.html pointing
// to content at /slug/index.ext.
// This ensures web servers serve the redirect as HTML at /slug.ext (without trailing slash).
// Fixes: https://github.com/WaylonWalker/markata-go/issues/395
// Fixes: https://github.com/WaylonWalker/markata-go/issues/437
// Fixes: https://github.com/WaylonWalker/markata-go/issues/465
func TestPublishHTMLPlugin_FormatRedirectsCreateDirectories(t *testing.T) {
	// Test each format independently to avoid redirect file overwrites
	// (when both formats are enabled, they share the same slug directory)
	tests := []struct {
		name          string
		enabledFormat string
		ext           string
	}{
		{
			name:          "markdown uses directory-based redirect",
			enabledFormat: "markdown",
			ext:           "md",
		},
		{
			name:          "text uses directory-based redirect",
			enabledFormat: "text",
			ext:           "txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			plugin := NewPublishHTMLPlugin()

			// Create config with only one format enabled at a time
			// HTML is DISABLED to test redirect behavior
			// (when HTML is enabled, /slug/index.html has main content instead of redirect)
			htmlEnabled := false
			postFormats := models.PostFormatsConfig{
				HTML: &htmlEnabled,
			}
			if tt.enabledFormat == "markdown" {
				postFormats.Markdown = true
			} else if tt.enabledFormat == "text" {
				postFormats.Text = true
			}

			config := &lifecycle.Config{
				OutputDir: tempDir,
				Extra: map[string]interface{}{
					"post_formats": postFormats,
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

			// Create manager for testing
			m := createTestManager(t, config)

			// Write post (which includes format redirects)
			if err := plugin.writePost(post, config, nil, m); err != nil {
				t.Fatalf("writePost() error = %v", err)
			}

			// Content should be at /slug.ext
			contentPath := filepath.Join(tempDir, "test-post."+tt.ext)
			// Redirect should be at /slug/index.ext/index.html
			redirectPath := filepath.Join(tempDir, "test-post", "index."+tt.ext, "index.html")
			// When HTML is disabled, /slug/index.html should also be a redirect
			slugRedirectPath := filepath.Join(tempDir, "test-post", "index.html")

			// Check that primary content file exists at /slug.ext
			info, err := os.Stat(contentPath)
			if err != nil {
				t.Fatalf("primary content file %s not found: %v", contentPath, err)
			}
			if info.IsDir() {
				t.Errorf("expected %s to be a file, but it's a directory", contentPath)
			}

			// Read content and verify it's actual content (not redirect HTML)
			content, err := os.ReadFile(contentPath)
			if err != nil {
				t.Fatalf("failed to read %s: %v", contentPath, err)
			}
			contentStr := string(content)
			if strings.Contains(contentStr, "<!DOCTYPE html>") {
				t.Error("primary content file should NOT be HTML redirect")
			}
			if !strings.Contains(contentStr, "Test") {
				t.Error("primary content file should contain post content")
			}

			// Check that redirect file exists at /slug/index.ext/index.html (always HTML!)
			redirectInfo, err := os.Stat(redirectPath)
			if err != nil {
				t.Fatalf("redirect file %s not found: %v", redirectPath, err)
			}
			if redirectInfo.IsDir() {
				t.Errorf("expected %s to be a file, but it's a directory", redirectPath)
			}

			// Read redirect and verify it points to the content URL
			redirectContent, err := os.ReadFile(redirectPath)
			if err != nil {
				t.Fatalf("failed to read %s: %v", redirectPath, err)
			}
			redirectStr := string(redirectContent)
			if !strings.Contains(redirectStr, "<!DOCTYPE html>") {
				t.Error("redirect file should contain DOCTYPE")
			}
			if !strings.Contains(redirectStr, "meta http-equiv=\"refresh\"") {
				t.Error("redirect file should contain meta refresh")
			}
			// Redirect should point TO the content at /slug.ext
			expectedTarget := "/test-post." + tt.ext
			if !strings.Contains(redirectStr, expectedTarget) {
				t.Errorf("redirect file should point to %s, got: %s", expectedTarget, redirectStr)
			}

			// When HTML is disabled, /slug/index.html should also be a redirect
			slugRedirectContent, err := os.ReadFile(slugRedirectPath)
			if err != nil {
				t.Fatalf("slug redirect file %s not found: %v", slugRedirectPath, err)
			}
			slugRedirectStr := string(slugRedirectContent)
			if !strings.Contains(slugRedirectStr, "meta http-equiv=\"refresh\"") {
				t.Error("/slug/index.html should be a redirect when HTML is disabled")
			}
		})
	}
}

// TestPublishHTMLPlugin_StandardWebTxtFiles tests that standard web txt files
// (robots.txt, llms.txt, humans.txt) are accessible via redirects from their expected URLs.
// Content is at /slug/index.txt, redirect at /slug.txt/index.html points there.
// Fixes: https://github.com/WaylonWalker/markata-go/issues/395
// Fixes: https://github.com/WaylonWalker/markata-go/issues/465
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

	// Create manager for testing
	m := createTestManager(t, config)

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

		if err := plugin.writePost(post, config, nil, m); err != nil {
			t.Fatalf("writePost() error for %s: %v", sf.slug, err)
		}
	}

	// Verify standard web files have content and redirects
	for _, sf := range standardFiles {
		t.Run(sf.slug+".txt structure", func(t *testing.T) {
			// Check content at /slug.txt (root level for special files)
			contentPath := filepath.Join(tempDir, sf.slug+".txt")
			content, err := os.ReadFile(contentPath)
			if err != nil {
				t.Fatalf("content file %s.txt not found: %v", sf.slug, err)
			}

			// Should contain actual content, not redirect HTML
			contentStr := string(content)
			if strings.Contains(contentStr, "<!DOCTYPE html>") {
				t.Errorf("%s.txt should be content, not HTML redirect", sf.slug)
			}

			// Check HTML redirect at /slug/index.txt/index.html
			redirectPath := filepath.Join(tempDir, sf.slug, "index.txt", "index.html")
			redirectContent, err := os.ReadFile(redirectPath)
			if err != nil {
				t.Fatalf("redirect file %s/index.txt/index.html not found: %v", sf.slug, err)
			}

			// Should be an HTML redirect pointing to the content
			redirectStr := string(redirectContent)
			if !strings.Contains(redirectStr, "http-equiv=\"refresh\"") {
				t.Errorf("%s/index.txt/index.html should be an HTML redirect", sf.slug)
			}
			expectedTarget := fmt.Sprintf("/%s.txt", sf.slug)
			if !strings.Contains(redirectStr, expectedTarget) {
				t.Errorf("%s/index.txt/index.html should redirect to %s, got: %s", sf.slug, expectedTarget, redirectStr)
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

	if !defaults.OG {
		t.Error("OG should be enabled by default")
	}
}

// TestPublishHTMLPlugin_TxtTemplateRendering tests that txt output uses templates
// when a template engine is available.
// Fixes: https://github.com/WaylonWalker/markata-go/issues/397
func TestPublishHTMLPlugin_TxtTemplateRendering(t *testing.T) {
	tempDir := t.TempDir()
	plugin := NewPublishHTMLPlugin()

	// Create config with Text format enabled
	htmlEnabled := true
	config := &lifecycle.Config{
		OutputDir: tempDir,
		Extra: map[string]interface{}{
			"post_formats": models.PostFormatsConfig{
				HTML: &htmlEnabled,
				Text: true,
			},
		},
	}

	// Create test post
	title := "My Test Title"
	description := "A test description"
	post := &models.Post{
		Path:        "test.md",
		Slug:        "test-post",
		Title:       &title,
		Description: &description,
		Content:     "This is the post content.",
		HTML:        "<html><body>Test</body></html>",
		Published:   true,
		Draft:       false,
		Skip:        false,
		ArticleHTML: "<p>Test</p>",
	}

	// Create manager for testing
	m := createTestManager(t, config)

	// Write post without template engine (should use fallback)
	if err := plugin.writePost(post, config, nil, m); err != nil {
		t.Fatalf("writePost() error = %v", err)
	}

	// Read txt content from /slug.txt
	txtPath := filepath.Join(tempDir, "test-post.txt")
	content, err := os.ReadFile(txtPath)
	if err != nil {
		t.Fatalf("failed to read txt file: %v", err)
	}

	contentStr := string(content)

	// Verify title is present with underline
	if !strings.Contains(contentStr, "My Test Title") {
		t.Error("txt file should contain the title")
	}
	if !strings.Contains(contentStr, "=============") {
		t.Error("txt file should contain title underline")
	}

	// Verify description is present
	if !strings.Contains(contentStr, "A test description") {
		t.Error("txt file should contain the description")
	}

	// Verify content is present
	if !strings.Contains(contentStr, "This is the post content.") {
		t.Error("txt file should contain the post content")
	}
}

// TestPublishHTMLPlugin_FormatExtRedirectsWithHTMLEnabled tests that redirect pages
// are created at /slug.ext/index.html (e.g., /test.txt/index.html) even when HTML
// format is enabled. This allows users to navigate to /test.txt/ and be redirected
// to /test/index.txt.
// Fixes: https://github.com/WaylonWalker/markata-go/issues/465
func TestPublishHTMLPlugin_FormatExtRedirectsWithHTMLEnabled(t *testing.T) {
	tempDir := t.TempDir()
	plugin := NewPublishHTMLPlugin()

	// Create config with HTML AND other formats enabled
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
		Slug:        "test",
		Title:       &title,
		Content:     "Test content",
		HTML:        "<html><body>Test HTML content</body></html>",
		Published:   true,
		Draft:       false,
		Skip:        false,
		ArticleHTML: "<p>Test content</p>",
	}

	// Create manager for testing
	m := createTestManager(t, config)

	// Write post
	if err := plugin.writePost(post, config, nil, m); err != nil {
		t.Fatalf("writePost() error = %v", err)
	}

	// Expected file structure:
	// /test.txt      -> text content
	// /test.md       -> markdown content
	// /test/index.html    -> HTML content (NOT a redirect)
	// /test/index.txt/index.html -> redirect to /test.txt
	// /test/index.md/index.html  -> redirect to /test.md

	// 1. Verify /test/index.html exists and contains HTML content (NOT a redirect)
	htmlPath := filepath.Join(tempDir, "test", "index.html")
	htmlContent, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("failed to read HTML file %s: %v", htmlPath, err)
	}
	htmlStr := string(htmlContent)
	if strings.Contains(htmlStr, "http-equiv=\"refresh\"") {
		t.Error("/test/index.html should contain HTML content, not a redirect")
	}
	if !strings.Contains(htmlStr, "Test HTML content") {
		t.Error("/test/index.html should contain the post's HTML content")
	}

	// 2. Verify /test.txt exists with text content
	txtContentPath := filepath.Join(tempDir, "test.txt")
	txtContent, err := os.ReadFile(txtContentPath)
	if err != nil {
		t.Fatalf("failed to read txt file %s: %v", txtContentPath, err)
	}
	if strings.Contains(string(txtContent), "<!DOCTYPE html>") {
		t.Error("/test.txt should contain text content, not HTML")
	}

	// 3. Verify /test.md exists with markdown content
	mdContentPath := filepath.Join(tempDir, "test.md")
	mdContent, err := os.ReadFile(mdContentPath)
	if err != nil {
		t.Fatalf("failed to read md file %s: %v", mdContentPath, err)
	}
	if strings.Contains(string(mdContent), "<!DOCTYPE html>") {
		t.Error("/test.md should contain markdown content, not HTML")
	}

	// 4. Verify /test/index.txt/index.html exists and is a redirect to /test.txt
	txtIndexRedirectPath := filepath.Join(tempDir, "test", "index.txt", "index.html")
	txtIndexRedirectContent, err := os.ReadFile(txtIndexRedirectPath)
	if err != nil {
		t.Fatalf("failed to read txt index redirect %s: %v", txtIndexRedirectPath, err)
	}
	if !strings.Contains(string(txtIndexRedirectContent), "/test.txt") {
		t.Error("/test/index.txt/index.html should redirect to /test.txt")
	}

	// 5. Verify /test/index.md/index.html exists and is a redirect to /test.md
	mdIndexRedirectPath := filepath.Join(tempDir, "test", "index.md", "index.html")
	mdIndexRedirectContent, err := os.ReadFile(mdIndexRedirectPath)
	if err != nil {
		t.Fatalf("failed to read md index redirect %s: %v", mdIndexRedirectPath, err)
	}
	if !strings.Contains(string(mdIndexRedirectContent), "/test.md") {
		t.Error("/test/index.md/index.html should redirect to /test.md")
	}
}

// TestPublishHTMLPlugin_RawTxtForSpecialFiles tests that special files
// (robots.txt, llms.txt, etc.) can use a raw.txt template for content-only output.
// Fixes: https://github.com/WaylonWalker/markata-go/issues/397
func TestPublishHTMLPlugin_RawTxtForSpecialFiles(t *testing.T) {
	tempDir := t.TempDir()
	plugin := NewPublishHTMLPlugin()

	// Create config with Text format enabled
	htmlEnabled := true
	config := &lifecycle.Config{
		OutputDir: tempDir,
		Extra: map[string]interface{}{
			"post_formats": models.PostFormatsConfig{
				HTML: &htmlEnabled,
				Text: true,
			},
		},
	}

	// Create robots post - should just output raw content
	robotsContent := "User-agent: *\nAllow: /"
	robotsPost := &models.Post{
		Path:        "robots.md",
		Slug:        "robots",
		Content:     robotsContent,
		HTML:        "<html><body>Test</body></html>",
		Published:   true,
		Draft:       false,
		Skip:        false,
		ArticleHTML: "<p>Test</p>",
	}

	// Create manager for testing
	m := createTestManager(t, config)

	// Write post without template engine (should use fallback)
	if err := plugin.writePost(robotsPost, config, nil, m); err != nil {
		t.Fatalf("writePost() error = %v", err)
	}

	// Read robots content from /robots.txt (root level for special files)
	robotsPath := filepath.Join(tempDir, "robots.txt")
	content, err := os.ReadFile(robotsPath)
	if err != nil {
		t.Fatalf("failed to read robots.txt: %v", err)
	}

	contentStr := string(content)

	// Verify content is present (without title/description header since no title set)
	if !strings.Contains(contentStr, "User-agent: *") {
		t.Error("robots.txt should contain the user-agent directive")
	}
	if !strings.Contains(contentStr, "Allow: /") {
		t.Error("robots.txt should contain the allow directive")
	}

	// Verify HTML redirect exists at /robots/index.txt/index.html
	redirectPath := filepath.Join(tempDir, "robots", "index.txt", "index.html")
	redirectContent, err := os.ReadFile(redirectPath)
	if err != nil {
		t.Fatalf("failed to read robots/index.txt/index.html: %v", err)
	}

	expectedTarget := "/robots.txt"
	if !strings.Contains(string(redirectContent), expectedTarget) {
		t.Errorf("robots/index.txt/index.html should redirect to %s", expectedTarget)
	}
	if !strings.Contains(string(redirectContent), "http-equiv=\"refresh\"") {
		t.Error("robots/index.txt/index.html should be an HTML redirect")
	}
}
