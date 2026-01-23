package plugins

import (
	"os"
	"path/filepath"
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
