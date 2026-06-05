package plugins

import (
	"strings"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestGenerateAtom_ArchiveMetadataAndFallbacks(t *testing.T) {
	config := lifecycle.NewConfig()
	config.Extra = map[string]interface{}{
		"url":           "https://example.com",
		"title":         "Example Site",
		"description":   "Site description",
		"language":      "en",
		"author_url":    "https://example.com/about/",
		"models_config": &models.Config{},
	}

	feed := &lifecycle.Feed{
		Title:       "Blog Archive",
		Description: "Blog posts",
		Path:        "blog/archive",
		Posts: []*models.Post{{
			Slug: "one",
			Href: "/one/",
		}},
	}

	atom, err := GenerateAtom(feed, config)
	if err != nil {
		t.Fatalf("GenerateAtom() error = %v", err)
	}

	checks := []string{
		`xml:lang="en"`,
		`xmlns:fh="http://purl.org/syndication/history/1.0"`,
		`<subtitle>Blog posts</subtitle>`,
		`<name>Example Site</name>`,
		`<uri>https://example.com/about/</uri>`,
		`<fh:complete></fh:complete>`,
		`href="https://example.com/blog/atom.xml" rel="current" type="application/atom+xml"`,
		`<updated>1970-01-01T00:00:00Z</updated>`,
	}
	for _, want := range checks {
		if !strings.Contains(atom, want) {
			t.Fatalf("expected atom feed to contain %q\n%s", want, atom)
		}
	}
}

func TestGenerateRSS_UsesFeedSpecificMetadata(t *testing.T) {
	date := time.Date(2024, 2, 2, 12, 0, 0, 0, time.UTC)
	config := lifecycle.NewConfig()
	config.Extra = map[string]interface{}{
		"url":             "https://example.com",
		"title":           "Example Site",
		"description":     "Site description",
		"language":        "en-us",
		"managing_editor": "editor@example.com (Editor)",
		"webmaster":       "webmaster@example.com (Webmaster)",
		"copyright":       "Copyright 2026 Example",
		"models_config": &models.Config{
			Authors: models.AuthorsConfig{Authors: map[string]models.Author{
				"waylon": {ID: "waylon", Name: "Waylon", Email: testStringPtr("waylon@example.com")},
			}},
		},
	}

	feed := &lifecycle.Feed{
		Title:       "Blog",
		Description: "Blog posts",
		Path:        "blog",
		Posts: []*models.Post{{
			Slug:      "one",
			Href:      "/one/",
			Title:     testStringPtr("One"),
			Date:      &date,
			Tags:      []string{"go", "feeds"},
			Authors:   []string{"waylon"},
			Published: true,
		}},
	}

	rss, err := GenerateRSS(feed, config)
	if err != nil {
		t.Fatalf("GenerateRSS() error = %v", err)
	}

	checks := []string{
		`<link>https://example.com/blog/</link>`,
		`<description>Blog posts</description>`,
		`<language>en-us</language>`,
		`<managingEditor>editor@example.com (Editor)</managingEditor>`,
		`<webMaster>webmaster@example.com (Webmaster)</webMaster>`,
		`<copyright>Copyright 2026 Example</copyright>`,
		`<generator>markata-go</generator>`,
		`<docs>https://www.rssboard.org/rss-specification</docs>`,
		`<author>waylon@example.com</author>`,
		`<category>go</category>`,
		`<category>feeds</category>`,
	}
	for _, want := range checks {
		if !strings.Contains(rss, want) {
			t.Fatalf("expected rss feed to contain %q\n%s", want, rss)
		}
	}
}

func TestGenerateJSONFeed_UsesFeedMetadata(t *testing.T) {
	config := lifecycle.NewConfig()
	config.Extra = map[string]interface{}{
		"url":         "https://example.com",
		"title":       "Example Site",
		"description": "Site description",
		"author":      "Waylon",
		"author_url":  "https://example.com/about/",
		"language":    "en",
		"models_config": &models.Config{
			SEO: models.SEOConfig{LogoURL: "https://example.com/logo.png"},
		},
	}

	feed := &lifecycle.Feed{
		Title:       "Blog Archive",
		Description: "Blog posts",
		Path:        "blog/archive",
		Posts: []*models.Post{{
			Slug:      "one",
			Href:      "/one/",
			Published: true,
		}},
	}

	jsonFeed, err := GenerateJSONFeed(feed, config)
	if err != nil {
		t.Fatalf("GenerateJSONFeed() error = %v", err)
	}

	checks := []string{
		`"home_page_url": "https://example.com/blog/"`,
		`"description": "Blog posts"`,
		`"language": "en"`,
		`"icon": "https://example.com/logo.png"`,
		`"name": "Waylon"`,
		`"url": "https://example.com/about/"`,
	}
	for _, want := range checks {
		if !strings.Contains(jsonFeed, want) {
			t.Fatalf("expected json feed to contain %q\n%s", want, jsonFeed)
		}
	}
}

func TestGenerateJSONFeed_IncludePrivateUsesEncryptedHTML(t *testing.T) {
	config := lifecycle.NewConfig()
	config.Extra = map[string]interface{}{
		"url":   "https://example.com",
		"title": "Example Site",
	}

	date := time.Date(2024, 2, 2, 12, 0, 0, 0, time.UTC)
	feed := &lifecycle.Feed{
		Title:          "Blog Archive",
		Path:           "blog/archive",
		IncludePrivate: true,
		Posts: []*models.Post{{
			Slug:        "one",
			Href:        "/one/",
			Title:       testStringPtr("One"),
			Published:   true,
			Private:     true,
			Date:        &date,
			Description: testStringPtr("secret summary"),
			Content:     "secret body",
			ArticleHTML: `<div class="encrypted-content">locked</div>`,
		}},
	}

	jsonFeed, err := GenerateJSONFeed(feed, config)
	if err != nil {
		t.Fatalf("GenerateJSONFeed() error = %v", err)
	}

	if !strings.Contains(jsonFeed, `"content_html": "\u003cdiv class=\"encrypted-content\"\u003elocked\u003c/div\u003e"`) {
		t.Fatalf("expected json feed to include encrypted HTML\n%s", jsonFeed)
	}
	if strings.Contains(jsonFeed, "secret summary") || strings.Contains(jsonFeed, "secret body") || strings.Contains(jsonFeed, `"content_text"`) {
		t.Fatalf("json feed should not expose private plaintext\n%s", jsonFeed)
	}
}

func testStringPtr(s string) *string {
	return &s
}
