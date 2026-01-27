package plugins

import (
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestSitemapPlugin_Name(t *testing.T) {
	plugin := NewSitemapPlugin()
	if plugin.Name() != "sitemap" {
		t.Errorf("Name() = %q, want %q", plugin.Name(), "sitemap")
	}
}

func TestSitemapPlugin_Priority(t *testing.T) {
	plugin := NewSitemapPlugin()

	if got := plugin.Priority(lifecycle.StageWrite); got != lifecycle.PriorityLate {
		t.Errorf("Priority(StageWrite) = %d, want %d", got, lifecycle.PriorityLate)
	}

	if got := plugin.Priority(lifecycle.StageRender); got != lifecycle.PriorityDefault {
		t.Errorf("Priority(StageRender) = %d, want %d", got, lifecycle.PriorityDefault)
	}
}

func TestSitemapPlugin_ExcludesPrivatePosts(t *testing.T) {
	plugin := NewSitemapPlugin()
	m := lifecycle.NewManager()

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	m.SetPosts([]*models.Post{
		{
			Slug:      "public-post",
			Href:      "/public-post/",
			Published: true,
			Draft:     false,
			Skip:      false,
			Private:   false,
			Date:      &date,
		},
		{
			Slug:      "private-post",
			Href:      "/private-post/",
			Published: true,
			Draft:     false,
			Skip:      false,
			Private:   true,
			Date:      &date,
		},
		{
			Slug:      "another-public",
			Href:      "/another-public/",
			Published: true,
			Draft:     false,
			Skip:      false,
			Private:   false,
			Date:      &date,
		},
	})

	siteURL := "https://example.com"
	sitemap := plugin.buildSitemap(m, siteURL)

	// Count post URLs (excluding home page)
	postURLCount := 0
	for _, url := range sitemap.URLs {
		if url.Loc == siteURL+"/" {
			continue // Skip home page
		}
		postURLCount++
		// Verify private post URL is not present
		if url.Loc == siteURL+"/private-post/" {
			t.Error("private post should not appear in sitemap")
		}
	}

	// Should have 2 public posts (no feeds configured)
	if postURLCount != 2 {
		t.Errorf("expected 2 post URLs in sitemap, got %d", postURLCount)
	}
}

func TestSitemapPlugin_ExcludesDraftAndSkipPosts(t *testing.T) {
	plugin := NewSitemapPlugin()
	m := lifecycle.NewManager()

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	m.SetPosts([]*models.Post{
		{
			Slug:      "published",
			Href:      "/published/",
			Published: true,
			Draft:     false,
			Skip:      false,
			Private:   false,
			Date:      &date,
		},
		{
			Slug:      "draft",
			Href:      "/draft/",
			Published: true,
			Draft:     true,
			Skip:      false,
			Private:   false,
			Date:      &date,
		},
		{
			Slug:      "skipped",
			Href:      "/skipped/",
			Published: true,
			Draft:     false,
			Skip:      true,
			Private:   false,
			Date:      &date,
		},
		{
			Slug:      "unpublished",
			Href:      "/unpublished/",
			Published: false,
			Draft:     false,
			Skip:      false,
			Private:   false,
			Date:      &date,
		},
	})

	siteURL := "https://example.com"
	sitemap := plugin.buildSitemap(m, siteURL)

	// Count post URLs (excluding home page)
	postURLCount := 0
	for _, url := range sitemap.URLs {
		if url.Loc == siteURL+"/" {
			continue
		}
		postURLCount++
	}

	// Should only have 1 published, non-draft, non-skip, non-private post
	if postURLCount != 1 {
		t.Errorf("expected 1 post URL in sitemap, got %d", postURLCount)
	}
}

// Ensure SitemapPlugin implements the required interfaces.
func TestSitemapPlugin_ImplementsInterfaces(_ *testing.T) {
	var _ lifecycle.Plugin = (*SitemapPlugin)(nil)
	var _ lifecycle.WritePlugin = (*SitemapPlugin)(nil)
	var _ lifecycle.PriorityPlugin = (*SitemapPlugin)(nil)
}
