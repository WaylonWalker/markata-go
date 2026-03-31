package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestFeedsListingPlugin_Write(t *testing.T) {
	plugin := NewFeedsListingPlugin()
	m := lifecycle.NewManager()
	config := m.Config()
	config.OutputDir = t.TempDir()
	feedsPage := models.NewFeedsPageConfig()
	defaults := models.NewFeedDefaults()
	config.Extra = map[string]interface{}{
		"title":         "Test Site",
		"description":   "A test site",
		"url":           "https://example.com",
		"feeds_page":    feedsPage,
		"feed_defaults": defaults,
	}

	now := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	title := "Post"
	m.Cache().Set("feed_configs", []models.FeedConfig{
		{
			Slug:        "blog",
			Title:       "Blog",
			Description: "All posts",
			Formats:     models.FeedFormats{HTML: true, RSS: true, Atom: true, JSON: true, Sitemap: true},
			Posts: []*models.Post{{
				Slug:      "post",
				Href:      "/post/",
				Title:     &title,
				Published: true,
				Date:      &now,
			}},
		},
		{
			Slug:           "private-feed",
			Title:          "Private",
			IncludePrivate: true,
			Formats:        models.FeedFormats{RSS: true},
		},
	})

	if err := plugin.Write(m); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(config.OutputDir, "feeds", "index.html"))
	if err != nil {
		t.Fatalf("ReadFile(feeds page) error = %v", err)
	}
	body := string(content)
	if !strings.Contains(body, "Blog") {
		t.Fatalf("feeds page should contain public feed title")
	}
	if !strings.Contains(body, "/blog/archive/rss.xml") {
		t.Fatalf("feeds page should link to archive rss variant")
	}
	if strings.Contains(body, "Private") {
		t.Fatalf("feeds page should not contain private feeds")
	}
}
