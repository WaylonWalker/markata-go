package plugins

import (
	"strings"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestGenerateAtom_UpdatedDeterministic(t *testing.T) {
	config := lifecycle.NewConfig()
	config.Extra = map[string]interface{}{"url": "https://example.com"}

	old := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 2, 2, 12, 0, 0, 0, time.UTC)

	feed := &lifecycle.Feed{
		Name:  "test",
		Title: "Test Feed",
		Path:  "test",
		Posts: []*models.Post{
			{Slug: "old", Href: "/old/", Date: &old},
			{Slug: "new", Href: "/new/", Date: &newer},
		},
	}

	atom, err := GenerateAtom(feed, config)
	if err != nil {
		t.Fatalf("GenerateAtom error: %v", err)
	}

	if !strings.Contains(atom, "<updated>2024-02-02T12:00:00Z</updated>") {
		t.Fatalf("expected updated to use latest post date, got:\n%s", atom)
	}
}

func TestGenerateRSS_LastBuildDateDeterministic(t *testing.T) {
	config := lifecycle.NewConfig()
	config.Extra = map[string]interface{}{"url": "https://example.com"}

	old := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 2, 2, 12, 0, 0, 0, time.UTC)

	feed := &lifecycle.Feed{
		Name:  "test",
		Title: "Test Feed",
		Path:  "test",
		Posts: []*models.Post{
			{Slug: "old", Href: "/old/", Date: &old},
			{Slug: "new", Href: "/new/", Date: &newer},
		},
	}

	rss, err := GenerateRSS(feed, config)
	if err != nil {
		t.Fatalf("GenerateRSS error: %v", err)
	}

	if !strings.Contains(rss, "<lastBuildDate>Fri, 02 Feb 2024 12:00:00 +0000</lastBuildDate>") {
		t.Fatalf("expected lastBuildDate to use latest post date, got:\n%s", rss)
	}
}

func TestGenerateAtom_UpdatedOmittedWhenNoDates(t *testing.T) {
	config := lifecycle.NewConfig()
	config.Extra = map[string]interface{}{"url": "https://example.com"}

	feed := &lifecycle.Feed{
		Name:  "test",
		Title: "Test Feed",
		Path:  "test",
		Posts: []*models.Post{
			{Slug: "one", Href: "/one/"},
			{Slug: "two", Href: "/two/"},
		},
	}

	atom, err := GenerateAtom(feed, config)
	if err != nil {
		t.Fatalf("GenerateAtom error: %v", err)
	}

	if strings.Contains(atom, "<updated>") {
		t.Fatalf("expected updated to be omitted when no dates, got:\n%s", atom)
	}
}

func TestGenerateRSS_LastBuildDateOmittedWhenNoDates(t *testing.T) {
	config := lifecycle.NewConfig()
	config.Extra = map[string]interface{}{"url": "https://example.com"}

	feed := &lifecycle.Feed{
		Name:  "test",
		Title: "Test Feed",
		Path:  "test",
		Posts: []*models.Post{
			{Slug: "one", Href: "/one/"},
			{Slug: "two", Href: "/two/"},
		},
	}

	rss, err := GenerateRSS(feed, config)
	if err != nil {
		t.Fatalf("GenerateRSS error: %v", err)
	}

	if strings.Contains(rss, "<lastBuildDate>") {
		t.Fatalf("expected lastBuildDate to be omitted when no dates, got:\n%s", rss)
	}
}
