package plugins

import (
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

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
		{Slug: pinsFeedSlug, Draft: true},
		{Slug: "/pins/", Skip: true},
		{Slug: "other"},
	}
	if postOwnsFeedSlug(posts, pinsFeedSlug) {
		t.Fatal("draft or skipped posts should not reserve the feed output path")
	}
}
