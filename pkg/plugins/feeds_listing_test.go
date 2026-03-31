package plugins

import (
	"os"
	"path/filepath"
	"strconv"
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

func TestFeedsListingPlugin_Write_TruncatesGeneratedFeedsOnMainPage(t *testing.T) {
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
	feedConfigs := make([]models.FeedConfig, 0, generatedFeedsPreviewLimit+2)
	for i := 0; i < generatedFeedsPreviewLimit+2; i++ {
		title := "Generated " + strconv.Itoa(i)
		slug := "generated-" + strconv.Itoa(i)
		feedConfigs = append(feedConfigs, models.FeedConfig{
			Slug:        slug,
			Title:       title,
			Description: "Generated feed",
			Formats:     models.FeedFormats{HTML: true, RSS: true},
			Posts: []*models.Post{{
				Slug:      slug + "-post",
				Href:      "/" + slug + "/",
				Title:     &title,
				Published: true,
				Date:      &now,
			}},
		})
	}
	m.Cache().Set("feed_configs", feedConfigs)

	if err := plugin.Write(m); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	mainBodyBytes, err := os.ReadFile(filepath.Join(config.OutputDir, "feeds", "index.html"))
	if err != nil {
		t.Fatalf("ReadFile(main feeds page) error = %v", err)
	}
	mainBody := string(mainBodyBytes)
	if !strings.Contains(mainBody, "Browse all 26 generated feeds") {
		t.Fatalf("main feeds page should link to all generated feeds")
	}
	if got := strings.Count(mainBody, `class="feed-row"`); got != generatedFeedsPreviewLimit {
		t.Fatalf("main feeds page should render %d preview rows, got %d", generatedFeedsPreviewLimit, got)
	}

	generatedBodyBytes, err := os.ReadFile(filepath.Join(config.OutputDir, "feeds", "generated", "index.html"))
	if err != nil {
		t.Fatalf("ReadFile(generated feeds page) error = %v", err)
	}
	generatedBody := string(generatedBodyBytes)
	if !strings.Contains(generatedBody, "Generated 25") {
		t.Fatalf("generated feeds page should include all generated feeds")
	}
}
