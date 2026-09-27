package plugins

import (
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestSubscriptionFeedsPlugin_Collect_AuthoredHomepageKeepsRootSyndicationWithoutFeedHTML(t *testing.T) {
	config := lifecycle.NewConfig()
	config.Extra = map[string]interface{}{
		"title": "Test Site",
	}

	m := lifecycle.NewManager()
	m.SetConfig(config)
	m.SetPosts([]*models.Post{
		{Path: "index.md", Slug: "", Published: true},
		{Path: "post.md", Slug: "post", Published: true},
	})

	if err := NewSubscriptionFeedsPlugin().Collect(m); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	feedConfigs, ok := m.Cache().Get("feed_configs")
	if !ok {
		t.Fatal("feed_configs not found in cache")
	}
	feeds, ok := feedConfigs.([]models.FeedConfig)
	if !ok {
		t.Fatalf("feed_configs type = %T, want []models.FeedConfig", feedConfigs)
	}

	root := GetFeedBySlug("", feeds)
	if root == nil {
		t.Fatal("implicit root feed not found")
	}
	if root.Formats.HTML {
		t.Error("implicit root feed HTML should be disabled when an authored homepage owns /index.html")
	}
	if !root.Formats.RSS || !root.Formats.Atom {
		t.Fatalf("root syndication should remain enabled: formats = %#v", root.Formats)
	}
}

func TestSubscriptionFeedsPlugin_Collect_DraftHomepageDoesNotSuppressDefaultHomepage(t *testing.T) {
	config := lifecycle.NewConfig()
	config.Extra = map[string]interface{}{
		"title": "Test Site",
	}

	m := lifecycle.NewManager()
	m.SetConfig(config)
	m.SetPosts([]*models.Post{
		{Path: "index.md", Slug: "", Draft: true},
	})

	if err := NewSubscriptionFeedsPlugin().Collect(m); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	cached, ok := m.Cache().Get("feed_configs")
	if !ok {
		t.Fatal("feed_configs not found in cache")
	}
	feeds, ok := cached.([]models.FeedConfig)
	if !ok {
		t.Fatalf("feed_configs type = %T, want []models.FeedConfig", cached)
	}

	root := GetFeedBySlug("", feeds)
	if root == nil {
		t.Fatal("implicit root feed not found")
	}
	if !root.Formats.HTML {
		t.Error("draft homepage should not suppress the generated default homepage")
	}
}
