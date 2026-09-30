package plugins

import (
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestSubscriptionFeedsPlugin_Collect_AddsPinsFeed(t *testing.T) {
	config := lifecycle.NewConfig()
	config.Extra = map[string]interface{}{"title": "Pins Site"}

	m := lifecycle.NewManager()
	m.SetConfig(config)

	if err := NewSubscriptionFeedsPlugin().Collect(m); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	for _, feed := range getFeedConfigs(m.Config()) {
		if feed.Slug != pinsFeedSlug {
			continue
		}
		if !feed.Formats.HTML || feed.Templates.HTML != "pins.html" {
			t.Fatalf("pins feed = %#v, want HTML pins template", feed)
		}
		if feed.Filter != "published == true and link" {
			t.Fatalf("pins filter = %q", feed.Filter)
		}
		return
	}
	t.Fatal("implicit pins feed was not injected")
}

func TestSubscriptionFeedsPlugin_Collect_PreservesExistingPinsPost(t *testing.T) {
	config := lifecycle.NewConfig()
	config.Extra = map[string]interface{}{"title": "Existing Site"}

	m := lifecycle.NewManager()
	m.SetConfig(config)
	m.SetPosts([]*models.Post{{
		Path:      "pages/pins.md",
		Slug:      pinsFeedSlug,
		Published: true,
	}})

	if err := NewSubscriptionFeedsPlugin().Collect(m); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	for _, feed := range getFeedConfigs(m.Config()) {
		if feed.Slug == pinsFeedSlug {
			t.Fatalf("implicit pins feed was injected over existing /pins/ post: %#v", feed)
		}
	}
}

func TestPostOwnsFeedSlug_IgnoresNonOutputPosts(t *testing.T) {
	posts := []*models.Post{
		{Slug: pinsFeedSlug, Published: true, Draft: true},
		{Slug: "/pins/", Published: true, Skip: true},
		{Slug: pinsFeedSlug, Published: false},
		{Slug: "other", Published: true},
	}
	if postOwnsFeedSlug(posts, pinsFeedSlug) {
		t.Fatal("draft, skipped, or unpublished posts should not reserve the feed output path")
	}
}
