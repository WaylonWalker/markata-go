package plugins

import (
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestImageZoomPlugin_Name(t *testing.T) {
	p := NewImageZoomPlugin()
	if got := p.Name(); got != "image_zoom" {
		t.Errorf("Name() = %q, want %q", got, "image_zoom")
	}
}

func TestImageZoomPlugin_DefaultConfig(t *testing.T) {
	p := NewImageZoomPlugin()
	cfg := p.Config()

	if cfg.Enabled != false {
		t.Errorf("default Enabled = %v, want false", cfg.Enabled)
	}
	if cfg.Library != "glightbox" {
		t.Errorf("default Library = %q, want %q", cfg.Library, "glightbox")
	}
	if cfg.Selector != ".glightbox" {
		t.Errorf("default Selector = %q, want %q", cfg.Selector, ".glightbox")
	}
	if cfg.CDN != true {
		t.Errorf("default CDN = %v, want true", cfg.CDN)
	}
	if cfg.AutoAllImages != false {
		t.Errorf("default AutoAllImages = %v, want false", cfg.AutoAllImages)
	}
	if cfg.OpenEffect != "zoom" {
		t.Errorf("default OpenEffect = %q, want %q", cfg.OpenEffect, "zoom")
	}
}

func TestImageZoomPlugin_ProcessPostDisabled(t *testing.T) {
	p := NewImageZoomPlugin()
	// Plugin is disabled by default

	post := &models.Post{
		ArticleHTML: `<p><img src="test.jpg" alt="Test image"></p>`,
	}

	// Process should be a no-op when disabled
	err := p.processPost(post)
	if err != nil {
		t.Errorf("processPost() error = %v", err)
	}

	// HTML should be unchanged
	expected := `<p><img src="test.jpg" alt="Test image"></p>`
	if post.ArticleHTML != expected {
		t.Errorf("ArticleHTML = %q, want %q", post.ArticleHTML, expected)
	}
}

func TestImageZoomPlugin_ProcessPostWithDataZoomable(t *testing.T) {
	p := NewImageZoomPlugin()
	p.SetConfig(models.ImageZoomConfig{
		Enabled:  true,
		Library:  "glightbox",
		Selector: ".glightbox",
	})

	post := &models.Post{
		ArticleHTML: `<p><img src="test.jpg" alt="Test image {data-zoomable}"></p>`,
	}

	err := p.processPost(post)
	if err != nil {
		t.Errorf("processPost() error = %v", err)
	}

	// Check that the image was wrapped in an anchor
	if !containsSubstring(post.ArticleHTML, `<a href="test.jpg" class="glightbox-link">`) {
		t.Errorf("ArticleHTML should contain glightbox anchor, got: %s", post.ArticleHTML)
	}

	// Check that data-glightbox attribute was added
	if !containsSubstring(post.ArticleHTML, `data-glightbox=`) {
		t.Errorf("ArticleHTML should contain data-glightbox attribute, got: %s", post.ArticleHTML)
	}

	// Check that the marker was removed from alt text
	if containsSubstring(post.ArticleHTML, `{data-zoomable}`) {
		t.Errorf("ArticleHTML should not contain {data-zoomable} marker, got: %s", post.ArticleHTML)
	}

	// Check that needs_image_zoom flag was set
	if post.Extra == nil || post.Extra["needs_image_zoom"] != true {
		t.Errorf("post.Extra[needs_image_zoom] should be true")
	}
}

func TestImageZoomPlugin_ProcessPostWithZoomableClass(t *testing.T) {
	p := NewImageZoomPlugin()
	p.SetConfig(models.ImageZoomConfig{
		Enabled:  true,
		Library:  "glightbox",
		Selector: ".glightbox",
	})

	post := &models.Post{
		ArticleHTML: `<p><img src="photo.png" alt="Photo {.zoomable}"></p>`,
	}

	err := p.processPost(post)
	if err != nil {
		t.Errorf("processPost() error = %v", err)
	}

	// Check that the image was processed
	if !containsSubstring(post.ArticleHTML, `<a href="photo.png" class="glightbox-link">`) {
		t.Errorf("ArticleHTML should contain glightbox anchor, got: %s", post.ArticleHTML)
	}

	// Check that the marker was removed
	if containsSubstring(post.ArticleHTML, `{.zoomable}`) {
		t.Errorf("ArticleHTML should not contain {.zoomable} marker, got: %s", post.ArticleHTML)
	}
}

func TestImageZoomPlugin_ProcessPostAutoAllImages(t *testing.T) {
	p := NewImageZoomPlugin()
	p.SetConfig(models.ImageZoomConfig{
		Enabled:       true,
		Library:       "glightbox",
		Selector:      ".glightbox",
		AutoAllImages: true,
	})

	post := &models.Post{
		ArticleHTML: `<p><img src="test.jpg" alt="Regular image"></p>`,
	}

	err := p.processPost(post)
	if err != nil {
		t.Errorf("processPost() error = %v", err)
	}

	// All images should be zoomable when AutoAllImages is true
	if !containsSubstring(post.ArticleHTML, `<a href="test.jpg" class="glightbox-link">`) {
		t.Errorf("ArticleHTML should contain glightbox anchor with AutoAllImages, got: %s", post.ArticleHTML)
	}
}

func TestImageZoomPlugin_ProcessPostFrontmatterOverride(t *testing.T) {
	p := NewImageZoomPlugin()
	p.SetConfig(models.ImageZoomConfig{
		Enabled:       true,
		Library:       "glightbox",
		Selector:      ".glightbox",
		AutoAllImages: false, // Default off
	})

	post := &models.Post{
		ArticleHTML: `<p><img src="test.jpg" alt="Regular image"></p>`,
		Extra: map[string]interface{}{
			"image_zoom": true, // Frontmatter enables for this post
		},
	}

	err := p.processPost(post)
	if err != nil {
		t.Errorf("processPost() error = %v", err)
	}

	// Image should be zoomable due to frontmatter override
	if !containsSubstring(post.ArticleHTML, `<a href="test.jpg" class="glightbox-link">`) {
		t.Errorf("ArticleHTML should contain glightbox anchor with frontmatter override, got: %s", post.ArticleHTML)
	}
}

func TestImageZoomPlugin_ProcessPostSkipsSkippedPosts(t *testing.T) {
	p := NewImageZoomPlugin()
	p.SetConfig(models.ImageZoomConfig{
		Enabled:       true,
		AutoAllImages: true,
	})

	post := &models.Post{
		Skip:        true,
		ArticleHTML: `<p><img src="test.jpg" alt="Test"></p>`,
	}

	originalHTML := post.ArticleHTML

	err := p.processPost(post)
	if err != nil {
		t.Errorf("processPost() error = %v", err)
	}

	// HTML should be unchanged for skipped posts
	if post.ArticleHTML != originalHTML {
		t.Errorf("ArticleHTML should be unchanged for skipped posts")
	}
}

func TestImageZoomPlugin_ProcessPostPreservesExistingGlightbox(t *testing.T) {
	p := NewImageZoomPlugin()
	p.SetConfig(models.ImageZoomConfig{
		Enabled:       true,
		AutoAllImages: true,
	})

	// Image already has data-glightbox
	post := &models.Post{
		ArticleHTML: `<p><img src="test.jpg" alt="Test" data-glightbox="gallery1"></p>`,
	}

	err := p.processPost(post)
	if err != nil {
		t.Errorf("processPost() error = %v", err)
	}

	// Should not double-wrap
	if containsSubstring(post.ArticleHTML, `<a href="test.jpg" class="glightbox-link">`) {
		t.Errorf("ArticleHTML should not wrap already-glightbox images")
	}

	// But should still mark as needs_image_zoom
	if post.Extra == nil || post.Extra["needs_image_zoom"] != true {
		t.Errorf("post.Extra[needs_image_zoom] should be true even for existing glightbox")
	}
}

func TestImageZoomPlugin_MultipleImages(t *testing.T) {
	p := NewImageZoomPlugin()
	p.SetConfig(models.ImageZoomConfig{
		Enabled:  true,
		Library:  "glightbox",
		Selector: ".glightbox",
	})

	post := &models.Post{
		ArticleHTML: `<p><img src="a.jpg" alt="Image A {data-zoomable}"></p>
<p><img src="b.jpg" alt="Image B"></p>
<p><img src="c.jpg" alt="Image C {data-zoomable}"></p>`,
	}

	err := p.processPost(post)
	if err != nil {
		t.Errorf("processPost() error = %v", err)
	}

	// First and third images should be zoomable
	if !containsSubstring(post.ArticleHTML, `<a href="a.jpg" class="glightbox-link">`) {
		t.Errorf("First image should be zoomable")
	}
	if !containsSubstring(post.ArticleHTML, `<a href="c.jpg" class="glightbox-link">`) {
		t.Errorf("Third image should be zoomable")
	}

	// Second image should NOT be zoomable (no marker, AutoAllImages is false)
	// It should still be a plain img tag
	if containsSubstring(post.ArticleHTML, `<a href="b.jpg"`) {
		t.Errorf("Second image should not be zoomable without marker")
	}
}

// containsSubstring is a helper to check if a string contains a substring.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
