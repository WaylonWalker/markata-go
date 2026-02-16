package plugins

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestImageOptimization_Render_WrapsLocalImages(t *testing.T) {
	plugin := NewImageOptimizationPlugin()
	plugin.config = defaultImageOptimizationConfig()
	plugin.availableFormats = []string{"avif", "webp"}
	plugin.config.Widths = []int{480, 960}
	plugin.config.Sizes = "(max-width: 960px) 100vw, 960px"

	post := &models.Post{
		ArticleHTML: `<p><img src="/images/cat.jpg" alt="Cat"></p><p><img src="https://example.com/dog.jpg" alt="Dog"></p>`,
		Slug:        "post",
	}

	if !isLocalImageSrc("/images/cat.jpg") {
		t.Fatalf("expected local image src to be recognized")
	}
	if !isOptimizableImageSrc("/images/cat.jpg") {
		t.Fatalf("expected optimizable image src to be recognized")
	}
	if sources := buildPictureSources("/images/cat.jpg", plugin.availableFormats, plugin.config.Widths, plugin.config.Sizes); len(sources) == 0 {
		t.Fatalf("expected picture sources to be generated")
	}

	if err := plugin.processPost(post); err != nil {
		t.Fatalf("processPost error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, "<picture>") {
		t.Fatalf("expected picture wrapper, got: %s", post.ArticleHTML)
	}
	if !strings.Contains(post.ArticleHTML, `srcset="/images/cat-480w.avif 480w, /images/cat-960w.avif 960w"`) {
		t.Fatalf("expected AVIF source, got: %s", post.ArticleHTML)
	}
	if !strings.Contains(post.ArticleHTML, `srcset="/images/cat-480w.webp 480w, /images/cat-960w.webp 960w"`) {
		t.Fatalf("expected WebP source, got: %s", post.ArticleHTML)
	}
	if strings.Contains(post.ArticleHTML, "example.com/dog") && strings.Contains(post.ArticleHTML, "<picture>") {
		if strings.Contains(post.ArticleHTML, "dog.webp") || strings.Contains(post.ArticleHTML, "dog.avif") {
			t.Fatalf("expected external image to remain unchanged")
		}
	}

	if post.Extra == nil {
		t.Fatalf("expected post.Extra to be set")
	}
	if _, ok := post.Extra["image_optimization"]; !ok {
		t.Fatalf("expected image_optimization targets in post.Extra")
	}
}

func TestImageOptimization_Render_SkipsExistingPicture(t *testing.T) {
	plugin := NewImageOptimizationPlugin()
	plugin.config = defaultImageOptimizationConfig()
	plugin.availableFormats = []string{"avif"}

	post := &models.Post{
		ArticleHTML: `<picture><img src="/images/cat.jpg" alt="Cat"></picture>`,
		Slug:        "post",
	}

	if err := plugin.processPost(post); err != nil {
		t.Fatalf("processPost error: %v", err)
	}

	if strings.Count(post.ArticleHTML, "<picture>") != 1 {
		t.Fatalf("expected existing picture untouched, got: %s", post.ArticleHTML)
	}
	if post.Extra != nil {
		if _, ok := post.Extra["image_optimization"]; ok {
			t.Fatalf("did not expect image_optimization targets")
		}
	}
}

func TestImageOptimization_ResolveOutputPath_Relative(t *testing.T) {
	target := imageOptimizationTarget{
		Src:      "images/cat.jpg",
		PostSlug: "post",
	}
	path, err := resolveImageOutputPath("output", target)
	if err != nil {
		t.Fatalf("resolveImageOutputPath error: %v", err)
	}
	normalized := filepath.ToSlash(path)
	if !strings.HasSuffix(normalized, "output/post/images/cat.jpg") {
		t.Fatalf("unexpected output path: %s", path)
	}
}
