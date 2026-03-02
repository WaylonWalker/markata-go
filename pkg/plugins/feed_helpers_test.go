package plugins

import (
	"strings"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

func TestFeedHelpers_FeedPostsCallable(t *testing.T) {
	m := newFeedTestManager(t)
	callable := createFeedPostsFunc(m)
	posts, err := callable("blog", map[string]interface{}{"limit": 1})
	if err != nil {
		t.Fatalf("feed_posts returned error: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0]["slug"] != "second" {
		t.Fatalf("unexpected post slug: %v", posts[0]["slug"])
	}
}

func TestFeedHelpers_RenderFeedCallable(t *testing.T) {
	m := newFeedTestManager(t)
	engine, err := templates.NewEngine("../../templates")
	if err != nil {
		t.Fatalf("templates.NewEngine: %v", err)
	}
	m.Cache().Set("templates.engine", engine)

	render := createRenderFeedFunc(m)
	value, err := render("blog", map[string]interface{}{"limit": 2})
	if err != nil {
		t.Fatalf("render_feed returned error: %v", err)
	}
	output := value.String()
	if !strings.Contains(output, "feed h-feed") {
		t.Fatalf("rendered snippet missing expected wrapper: %s", output)
	}
}

func TestFeedHelpers_RenderFeedWithoutCachedEngine(t *testing.T) {
	m := newFeedTestManager(t)
	config := m.Config()
	if config.Extra == nil {
		config.Extra = make(map[string]interface{})
	}
	config.Extra["theme"] = ThemeDefault

	render := createRenderFeedFunc(m)
	value, err := render("blog", map[string]interface{}{"limit": 1})
	if err != nil {
		t.Fatalf("render_feed returned error: %v", err)
	}
	output := value.String()
	if !strings.Contains(output, "feed h-feed") {
		t.Fatalf("rendered snippet missing expected wrapper: %s", output)
	}
}

func TestFeedHelpers_RenderFeedPhotoCard(t *testing.T) {
	title := "Wicket's Lab"
	description := "Wicket in his lab"
	date := time.Now()
	photo := &models.Post{
		Slug:        "lab",
		Href:        "/lab/",
		Published:   true,
		Date:        &date,
		Template:    "photo",
		Title:       &title,
		Description: &description,
		Extra: map[string]interface{}{
			"image": "/lab.webp",
		},
	}
	feed := models.FeedConfig{
		Slug:    "photo-feed",
		Title:   "Photos",
		Filter:  "template == 'photo'",
		Sort:    "date",
		Reverse: true,
	}
	m := newFeedTestManagerWithCustom(t, []models.FeedConfig{feed}, []*models.Post{photo})
	engine, err := templates.NewEngine("../../templates")
	if err != nil {
		t.Fatalf("templates.NewEngine: %v", err)
	}
	m.Cache().Set("templates.engine", engine)

	render := createRenderFeedFunc(m)
	value, err := render("photo-feed", map[string]interface{}{"limit": 1})
	if err != nil {
		t.Fatalf("render_feed returned error: %v", err)
	}
	output := value.String()
	if !strings.Contains(output, "photo-figure") {
		t.Fatalf("expected photo figure class, got: %s", output)
	}
	if !strings.Contains(output, "<figure") || !strings.Contains(output, "<figcaption") {
		t.Fatalf("expected figure markup for photo card, got: %s", output)
	}
	if !strings.Contains(output, description) {
		t.Fatalf("expected photo description in figcaption, got: %s", output)
	}
}

func newFeedTestManager(t *testing.T) *lifecycle.Manager {
	date := time.Now()
	older := date.Add(-time.Hour)
	posts := []*models.Post{
		{
			Slug:      "first",
			Href:      "/first/",
			Published: true,
			Date:      &older,
		},
		{
			Slug:      "second",
			Href:      "/second/",
			Published: true,
			Date:      &date,
		},
	}
	feeds := []models.FeedConfig{
		{
			Slug:    "blog",
			Title:   "Blog",
			Filter:  "published == true",
			Sort:    "date",
			Reverse: true,
		},
	}
	return newFeedTestManagerWithCustom(t, feeds, posts)
}

func newFeedTestManagerWithCustom(t *testing.T, feeds []models.FeedConfig, posts []*models.Post) *lifecycle.Manager {
	t.Helper()
	m := lifecycle.NewManager()
	config := m.Config()
	if config.Extra == nil {
		config.Extra = make(map[string]interface{})
	}
	config.Extra["feeds"] = feeds
	m.SetPosts(posts)
	return m
}
