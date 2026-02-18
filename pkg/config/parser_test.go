package config

import (
	"testing"
)

func TestParseTOML(t *testing.T) {
	data := []byte(`
[markata-go]
output_dir = "public"
url = "https://example.com"
title = "Test Site"

[markata-go.components.share]
enabled = false
title = "Share this"
position = "bottom"
platforms = ["twitter", "copy"]

[markata-go.components.share.custom]
mastodon = { name = "Mastodon", icon = "mastodon.svg", url = "https://mastodon.social/share?text={{title}}&url={{url}}" }

[markata-go.glob]
patterns = ["**/*.md"]
use_gitignore = true

[[markata-go.feeds]]
slug = "blog"
title = "Blog"
filter = "published == True"
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	if config.OutputDir != "public" {
		t.Errorf("OutputDir = %q, want %q", config.OutputDir, "public")
	}
	if config.URL != "https://example.com" {
		t.Errorf("URL = %q, want %q", config.URL, "https://example.com")
	}
	if len(config.Feeds) != 1 {
		t.Errorf("len(Feeds) = %d, want 1", len(config.Feeds))
	}
	if config.Components.Share.Enabled == nil || *config.Components.Share.Enabled != false {
		t.Errorf("Components.Share.Enabled = %v, want false", config.Components.Share.Enabled)
	}
	if config.Components.Share.Title != "Share this" {
		t.Errorf("Components.Share.Title = %q, want %q", config.Components.Share.Title, "Share this")
	}
	if config.Components.Share.Position != "bottom" {
		t.Errorf("Components.Share.Position = %q, want %q", config.Components.Share.Position, "bottom")
	}
	if len(config.Components.Share.Platforms) != 2 || config.Components.Share.Platforms[0] != "twitter" || config.Components.Share.Platforms[1] != "copy" {
		t.Errorf("Components.Share.Platforms = %v, want [twitter copy]", config.Components.Share.Platforms)
	}
	if config.Components.Share.Custom["mastodon"].Name != "Mastodon" {
		t.Errorf("Components.Share.Custom[mastodon].Name = %q, want %q", config.Components.Share.Custom["mastodon"].Name, "Mastodon")
	}
}

func TestParseTOML_WellKnown(t *testing.T) {
	data := []byte(`
[markata-go]
title = "Test Site"

[markata-go.well_known]
enabled = false
auto_generate = ["time"]
ssh_fingerprint = "SHA256:abc"
keybase_username = "alice"
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	if config.WellKnown.Enabled == nil || *config.WellKnown.Enabled != false {
		t.Fatalf("WellKnown.Enabled = %v, want false", config.WellKnown.Enabled)
	}
	if len(config.WellKnown.AutoGenerate) != 1 || config.WellKnown.AutoGenerate[0] != "time" {
		t.Fatalf("WellKnown.AutoGenerate = %v, want [time]", config.WellKnown.AutoGenerate)
	}
	if config.WellKnown.SSHFingerprint != "SHA256:abc" {
		t.Fatalf("WellKnown.SSHFingerprint = %q, want %q", config.WellKnown.SSHFingerprint, "SHA256:abc")
	}
	if config.WellKnown.KeybaseUsername != "alice" {
		t.Fatalf("WellKnown.KeybaseUsername = %q, want %q", config.WellKnown.KeybaseUsername, "alice")
	}
}

func TestParseTOML_WebSub(t *testing.T) {
	data := []byte(`
[markata-go]
title = "Test Site"

[markata-go.websub]
enabled = true
hubs = ["https://hub.example.com/"]
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	if config.WebSub.Enabled == nil || *config.WebSub.Enabled != true {
		t.Fatalf("WebSub.Enabled = %v, want true", config.WebSub.Enabled)
	}
	if len(config.WebSub.Hubs) != 1 || config.WebSub.Hubs[0] != "https://hub.example.com/" {
		t.Fatalf("WebSub.Hubs = %v, want hub list", config.WebSub.Hubs)
	}
}

func TestParseTOML_InvalidSyntax(t *testing.T) {
	data := []byte(`invalid toml {{{{ syntax`)

	_, err := ParseTOML(data)
	if err == nil {
		t.Error("ParseTOML() should return error for invalid TOML")
	}
}

func TestParseTOML_EmptySection(t *testing.T) {
	data := []byte(`
[markata-go]
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	// Should return empty config
	if config.OutputDir != "" {
		t.Errorf("OutputDir = %q, want empty", config.OutputDir)
	}
}

func TestParseYAML(t *testing.T) {
	data := []byte(`
markata-go:
  output_dir: public
  url: https://example.com
  title: Test Site
  components:
    share:
      enabled: false
      title: Share this
      position: bottom
      platforms:
        - twitter
        - copy
      custom:
        mastodon:
          name: Mastodon
          icon: mastodon.svg
          url: "https://mastodon.social/share?text={{title}}&url={{url}}"
  glob:
    patterns:
      - "**/*.md"
    use_gitignore: true
  feeds:
    - slug: blog
      title: Blog
      filter: "published == True"
`)

	config, err := ParseYAML(data)
	if err != nil {
		t.Fatalf("ParseYAML() error = %v", err)
	}

	if config.OutputDir != "public" {
		t.Errorf("OutputDir = %q, want %q", config.OutputDir, "public")
	}
	if config.URL != "https://example.com" {
		t.Errorf("URL = %q, want %q", config.URL, "https://example.com")
	}
	if len(config.Feeds) != 1 {
		t.Errorf("len(Feeds) = %d, want 1", len(config.Feeds))
	}
	if config.Components.Share.Enabled == nil || *config.Components.Share.Enabled != false {
		t.Errorf("Components.Share.Enabled = %v, want false", config.Components.Share.Enabled)
	}
	if config.Components.Share.Title != "Share this" {
		t.Errorf("Components.Share.Title = %q, want %q", config.Components.Share.Title, "Share this")
	}
	if config.Components.Share.Position != "bottom" {
		t.Errorf("Components.Share.Position = %q, want %q", config.Components.Share.Position, "bottom")
	}
	if len(config.Components.Share.Platforms) != 2 || config.Components.Share.Platforms[0] != "twitter" || config.Components.Share.Platforms[1] != "copy" {
		t.Errorf("Components.Share.Platforms = %v, want [twitter copy]", config.Components.Share.Platforms)
	}
	if config.Components.Share.Custom["mastodon"].Icon != "mastodon.svg" {
		t.Errorf("Components.Share.Custom[mastodon].Icon = %q, want %q", config.Components.Share.Custom["mastodon"].Icon, "mastodon.svg")
	}
}

func TestParseYAML_InvalidSyntax(t *testing.T) {
	data := []byte(`
markata-go:
  invalid:
    - unclosed list
   bad indent
`)

	_, err := ParseYAML(data)
	if err == nil {
		t.Error("ParseYAML() should return error for invalid YAML")
	}
}

func TestParseYAML_EmptySection(t *testing.T) {
	data := []byte(`
markata-go:
`)

	config, err := ParseYAML(data)
	if err != nil {
		t.Fatalf("ParseYAML() error = %v", err)
	}

	if config.OutputDir != "" {
		t.Errorf("OutputDir = %q, want empty", config.OutputDir)
	}
}

func TestParseJSON(t *testing.T) {
	data := []byte(`{
  "markata-go": {
    "output_dir": "public",
    "url": "https://example.com",
    "title": "Test Site",
    "components": {
      "share": {
        "enabled": false,
        "title": "Share this",
        "position": "bottom",
        "platforms": ["twitter", "copy"],
        "custom": {
          "mastodon": {
            "name": "Mastodon",
            "icon": "mastodon.svg",
            "url": "https://mastodon.social/share?text={{title}}&url={{url}}"
          }
        }
      }
    },
    "glob": {
      "patterns": ["**/*.md"],
      "use_gitignore": true
    },
    "feeds": [
      {
        "slug": "blog",
        "title": "Blog",
        "filter": "published == True"
      }
    ]
  }
}`)

	config, err := ParseJSON(data)
	if err != nil {
		t.Fatalf("ParseJSON() error = %v", err)
	}

	if config.OutputDir != "public" {
		t.Errorf("OutputDir = %q, want %q", config.OutputDir, "public")
	}
	if config.URL != "https://example.com" {
		t.Errorf("URL = %q, want %q", config.URL, "https://example.com")
	}
	if len(config.Feeds) != 1 {
		t.Errorf("len(Feeds) = %d, want 1", len(config.Feeds))
	}
	if config.Components.Share.Enabled == nil || *config.Components.Share.Enabled != false {
		t.Errorf("Components.Share.Enabled = %v, want false", config.Components.Share.Enabled)
	}
	if config.Components.Share.Title != "Share this" {
		t.Errorf("Components.Share.Title = %q, want %q", config.Components.Share.Title, "Share this")
	}
	if config.Components.Share.Position != "bottom" {
		t.Errorf("Components.Share.Position = %q, want %q", config.Components.Share.Position, "bottom")
	}
	if len(config.Components.Share.Platforms) != 2 || config.Components.Share.Platforms[0] != "twitter" || config.Components.Share.Platforms[1] != "copy" {
		t.Errorf("Components.Share.Platforms = %v, want [twitter copy]", config.Components.Share.Platforms)
	}
	if config.Components.Share.Custom["mastodon"].URL == "" {
		t.Error("Components.Share.Custom[mastodon].URL should be set")
	}
}

func TestParseJSON_InvalidSyntax(t *testing.T) {
	data := []byte(`{invalid json}`)

	_, err := ParseJSON(data)
	if err == nil {
		t.Error("ParseJSON() should return error for invalid JSON")
	}
}

func TestParseJSON_EmptySection(t *testing.T) {
	data := []byte(`{"markata-go": {}}`)

	config, err := ParseJSON(data)
	if err != nil {
		t.Fatalf("ParseJSON() error = %v", err)
	}

	if config.OutputDir != "" {
		t.Errorf("OutputDir = %q, want empty", config.OutputDir)
	}
}

func TestParseTOML_UseGitignoreFalse(t *testing.T) {
	data := []byte(`
[markata-go]
[markata-go.glob]
use_gitignore = false
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	if config.GlobConfig.UseGitignore != false {
		t.Error("UseGitignore should be false")
	}
}

func TestParseTOML_FeedFormats(t *testing.T) {
	data := []byte(`
[markata-go]
[[markata-go.feeds]]
slug = "blog"
[markata-go.feeds.formats]
html = true
rss = true
atom = false
json = false
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	if len(config.Feeds) != 1 {
		t.Fatalf("len(Feeds) = %d, want 1", len(config.Feeds))
	}

	formats := config.Feeds[0].Formats
	if !formats.HTML {
		t.Error("HTML should be true")
	}
	if !formats.RSS {
		t.Error("RSS should be true")
	}
	if formats.Atom {
		t.Error("Atom should be false")
	}
	if formats.JSON {
		t.Error("JSON should be false")
	}
}

func TestParseTOML_FeedDefaults(t *testing.T) {
	data := []byte(`
[markata-go]
[markata-go.feed_defaults]
items_per_page = 15
orphan_threshold = 4
[markata-go.feed_defaults.formats]
html = true
rss = true
[markata-go.feed_defaults.syndication]
max_items = 30
include_content = true
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	if config.FeedDefaults.ItemsPerPage != 15 {
		t.Errorf("ItemsPerPage = %d, want 15", config.FeedDefaults.ItemsPerPage)
	}
	if config.FeedDefaults.OrphanThreshold != 4 {
		t.Errorf("OrphanThreshold = %d, want 4", config.FeedDefaults.OrphanThreshold)
	}
	if !config.FeedDefaults.Formats.HTML {
		t.Error("Formats.HTML should be true")
	}
	if config.FeedDefaults.Syndication.MaxItems != 30 {
		t.Errorf("Syndication.MaxItems = %d, want 30", config.FeedDefaults.Syndication.MaxItems)
	}
}

func TestParseTOML_MarkdownConfig(t *testing.T) {
	data := []byte(`
[markata-go]
[markata-go.markdown]
extensions = ["tables", "footnotes", "syntax-highlighting"]
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	expected := []string{"tables", "footnotes", "syntax-highlighting"}
	if len(config.MarkdownConfig.Extensions) != len(expected) {
		t.Fatalf("len(Extensions) = %d, want %d", len(config.MarkdownConfig.Extensions), len(expected))
	}
	for i, ext := range expected {
		if config.MarkdownConfig.Extensions[i] != ext {
			t.Errorf("Extensions[%d] = %q, want %q", i, config.MarkdownConfig.Extensions[i], ext)
		}
	}
}

func TestParseTOML_Hooks(t *testing.T) {
	data := []byte(`
[markata-go]
hooks = ["markdown", "template", "sitemap"]
disabled_hooks = ["seo", "analytics"]
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	if len(config.Hooks) != 3 {
		t.Errorf("len(Hooks) = %d, want 3", len(config.Hooks))
	}
	if len(config.DisabledHooks) != 2 {
		t.Errorf("len(DisabledHooks) = %d, want 2", len(config.DisabledHooks))
	}
}

func TestParseTOML_MultipleFeeds(t *testing.T) {
	data := []byte(`
[markata-go]
[[markata-go.feeds]]
slug = "blog"
title = "Blog"

[[markata-go.feeds]]
slug = "projects"
title = "Projects"

[[markata-go.feeds]]
slug = "notes"
title = "Notes"
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	if len(config.Feeds) != 3 {
		t.Fatalf("len(Feeds) = %d, want 3", len(config.Feeds))
	}

	expectedSlugs := []string{"blog", "projects", "notes"}
	for i, slug := range expectedSlugs {
		if config.Feeds[i].Slug != slug {
			t.Errorf("Feeds[%d].Slug = %q, want %q", i, config.Feeds[i].Slug, slug)
		}
	}
}

func TestParseTOML_FeedTemplates(t *testing.T) {
	data := []byte(`
[markata-go]
[[markata-go.feeds]]
slug = "blog"
[markata-go.feeds.templates]
html = "custom-feed.html"
rss = "custom-rss.xml"
atom = "custom-atom.xml"
card = "custom-card.html"
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	if len(config.Feeds) != 1 {
		t.Fatalf("len(Feeds) = %d, want 1", len(config.Feeds))
	}

	templates := config.Feeds[0].Templates
	if templates.HTML != "custom-feed.html" {
		t.Errorf("Templates.HTML = %q, want %q", templates.HTML, "custom-feed.html")
	}
	if templates.RSS != "custom-rss.xml" {
		t.Errorf("Templates.RSS = %q, want %q", templates.RSS, "custom-rss.xml")
	}
	if templates.Atom != "custom-atom.xml" {
		t.Errorf("Templates.Atom = %q, want %q", templates.Atom, "custom-atom.xml")
	}
	if templates.Card != "custom-card.html" {
		t.Errorf("Templates.Card = %q, want %q", templates.Card, "custom-card.html")
	}
}

func TestParseYAML_NestedConfig(t *testing.T) {
	data := []byte(`
markata-go:
  output_dir: public
  glob:
    patterns:
      - "posts/**/*.md"
      - "pages/*.md"
    use_gitignore: true
  feed_defaults:
    items_per_page: 20
    formats:
      html: true
      rss: true
    syndication:
      max_items: 50
      include_content: true
`)

	config, err := ParseYAML(data)
	if err != nil {
		t.Fatalf("ParseYAML() error = %v", err)
	}

	if len(config.GlobConfig.Patterns) != 2 {
		t.Errorf("len(Patterns) = %d, want 2", len(config.GlobConfig.Patterns))
	}
	if config.FeedDefaults.ItemsPerPage != 20 {
		t.Errorf("ItemsPerPage = %d, want 20", config.FeedDefaults.ItemsPerPage)
	}
	if config.FeedDefaults.Syndication.MaxItems != 50 {
		t.Errorf("Syndication.MaxItems = %d, want 50", config.FeedDefaults.Syndication.MaxItems)
	}
}

func TestParseJSON_NestedConfig(t *testing.T) {
	data := []byte(`{
  "markata-go": {
    "output_dir": "public",
    "glob": {
      "patterns": ["posts/**/*.md", "pages/*.md"],
      "use_gitignore": true
    },
    "feed_defaults": {
      "items_per_page": 20,
      "formats": {
        "html": true,
        "rss": true
      },
      "syndication": {
        "max_items": 50,
        "include_content": true
      }
    }
  }
}`)

	config, err := ParseJSON(data)
	if err != nil {
		t.Fatalf("ParseJSON() error = %v", err)
	}

	if len(config.GlobConfig.Patterns) != 2 {
		t.Errorf("len(Patterns) = %d, want 2", len(config.GlobConfig.Patterns))
	}
	if config.FeedDefaults.ItemsPerPage != 20 {
		t.Errorf("ItemsPerPage = %d, want 20", config.FeedDefaults.ItemsPerPage)
	}
	if config.FeedDefaults.Syndication.MaxItems != 50 {
		t.Errorf("Syndication.MaxItems = %d, want 50", config.FeedDefaults.Syndication.MaxItems)
	}
}

func TestParseTOML_AllFields(t *testing.T) {
	data := []byte(`
[markata-go]
output_dir = "public"
url = "https://example.com"
title = "My Site"
description = "A great site"
author = "John Doe"
assets_dir = "assets"
templates_dir = "themes/default"
hooks = ["markdown", "template"]
disabled_hooks = ["seo"]
concurrency = 4

[markata-go.glob]
patterns = ["**/*.md"]
use_gitignore = true

[markata-go.markdown]
extensions = ["tables"]

[[markata-go.feeds]]
slug = "blog"
title = "Blog"
description = "All posts"
filter = "published == True"
sort = "date"
reverse = true
items_per_page = 10
orphan_threshold = 3

[markata-go.feeds.formats]
html = true
rss = true

[markata-go.feed_defaults]
items_per_page = 15
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	// Verify all top-level fields
	tests := []struct {
		field string
		got   interface{}
		want  interface{}
	}{
		{"OutputDir", config.OutputDir, "public"},
		{"URL", config.URL, "https://example.com"},
		{"Title", config.Title, "My Site"},
		{"Description", config.Description, "A great site"},
		{"Author", config.Author, "John Doe"},
		{"AssetsDir", config.AssetsDir, "assets"},
		{"TemplatesDir", config.TemplatesDir, "themes/default"},
		{"Concurrency", config.Concurrency, 4},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.field, tt.got, tt.want)
		}
	}
}

func TestParseTOML_BlogrollExternalFeedFields(t *testing.T) {
	maxEntries := 5
	primary := true

	data := []byte(`
[markata-go]

[markata-go.blogroll]
enabled = true

[[markata-go.blogroll.feeds]]
url = "https://example.com/feed.xml"
title = "Example Blog"
handle = "exampleblog"
aliases = ["example", "ex"]
max_entries = 5
primary = true
primary_person = "mainauthor"
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	if len(config.Blogroll.Feeds) != 1 {
		t.Fatalf("len(Blogroll.Feeds) = %d, want 1", len(config.Blogroll.Feeds))
	}

	feed := config.Blogroll.Feeds[0]

	// Verify all external feed fields are parsed correctly
	tests := []struct {
		field string
		got   interface{}
		want  interface{}
	}{
		{"URL", feed.URL, "https://example.com/feed.xml"},
		{"Title", feed.Title, "Example Blog"},
		{"Handle", feed.Handle, "exampleblog"},
		{"PrimaryPerson", feed.PrimaryPerson, "mainauthor"},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.field, tt.got, tt.want)
		}
	}

	// Check slice fields
	if len(feed.Aliases) != 2 {
		t.Errorf("len(Aliases) = %d, want 2", len(feed.Aliases))
	} else if feed.Aliases[0] != "example" || feed.Aliases[1] != "ex" {
		t.Errorf("Aliases = %v, want [example ex]", feed.Aliases)
	}

	// Check pointer fields
	if feed.MaxEntries == nil || *feed.MaxEntries != maxEntries {
		t.Errorf("MaxEntries = %v, want %d", feed.MaxEntries, maxEntries)
	}
	if feed.Primary == nil || *feed.Primary != primary {
		t.Errorf("Primary = %v, want %v", feed.Primary, primary)
	}
}

func TestParseYAML_BlogrollExternalFeedFields(t *testing.T) {
	maxEntries := 10
	primary := false

	data := []byte(`
markata-go:
  blogroll:
    enabled: true
    feeds:
      - url: "https://blog.example.org/rss"
        title: "Another Blog"
        handle: "anotherblog"
        aliases:
          - another
          - blog
        max_entries: 10
        primary: false
        primary_person: "someauthor"
`)

	config, err := ParseYAML(data)
	if err != nil {
		t.Fatalf("ParseYAML() error = %v", err)
	}

	if len(config.Blogroll.Feeds) != 1 {
		t.Fatalf("len(Blogroll.Feeds) = %d, want 1", len(config.Blogroll.Feeds))
	}

	feed := config.Blogroll.Feeds[0]

	if feed.URL != "https://blog.example.org/rss" {
		t.Errorf("URL = %q, want %q", feed.URL, "https://blog.example.org/rss")
	}
	if feed.Handle != "anotherblog" {
		t.Errorf("Handle = %q, want %q", feed.Handle, "anotherblog")
	}
	if feed.PrimaryPerson != "someauthor" {
		t.Errorf("PrimaryPerson = %q, want %q", feed.PrimaryPerson, "someauthor")
	}
	if len(feed.Aliases) != 2 {
		t.Errorf("len(Aliases) = %d, want 2", len(feed.Aliases))
	}
	if feed.MaxEntries == nil || *feed.MaxEntries != maxEntries {
		t.Errorf("MaxEntries = %v, want %d", feed.MaxEntries, maxEntries)
	}
	if feed.Primary == nil || *feed.Primary != primary {
		t.Errorf("Primary = %v, want %v", feed.Primary, primary)
	}
}

func TestParseJSON_BlogrollExternalFeedFields(t *testing.T) {
	maxEntries := 3

	data := []byte(`{
  "markata-go": {
    "blogroll": {
      "enabled": true,
      "feeds": [
        {
          "url": "https://json.example.com/feed",
          "title": "JSON Blog",
          "handle": "jsonblog",
          "aliases": ["json", "jb"],
          "max_entries": 3,
          "primary": true,
          "primary_person": "jsonauthor"
        }
      ]
    }
  }
}`)

	config, err := ParseJSON(data)
	if err != nil {
		t.Fatalf("ParseJSON() error = %v", err)
	}

	if len(config.Blogroll.Feeds) != 1 {
		t.Fatalf("len(Blogroll.Feeds) = %d, want 1", len(config.Blogroll.Feeds))
	}

	feed := config.Blogroll.Feeds[0]

	if feed.URL != "https://json.example.com/feed" {
		t.Errorf("URL = %q, want %q", feed.URL, "https://json.example.com/feed")
	}
	if feed.Handle != "jsonblog" {
		t.Errorf("Handle = %q, want %q", feed.Handle, "jsonblog")
	}
	if feed.PrimaryPerson != "jsonauthor" {
		t.Errorf("PrimaryPerson = %q, want %q", feed.PrimaryPerson, "jsonauthor")
	}
	if len(feed.Aliases) != 2 || feed.Aliases[0] != "json" || feed.Aliases[1] != "jb" {
		t.Errorf("Aliases = %v, want [json jb]", feed.Aliases)
	}
	if feed.MaxEntries == nil || *feed.MaxEntries != maxEntries {
		t.Errorf("MaxEntries = %v, want %d", feed.MaxEntries, maxEntries)
	}
	if feed.Primary == nil || !*feed.Primary {
		t.Errorf("Primary = %v, want true", feed.Primary)
	}
}

// TestParseTOML_BlogrollFallbackImageService tests parsing of fallback_image_service from TOML
func TestParseTOML_BlogrollFallbackImageService(t *testing.T) {
	data := []byte(`
[markata-go]
title = "Test Site"

[markata-go.blogroll]
enabled = true
fallback_image_service = "https://shots.waylonwalker.com/shot/?url={url}&height=160&width=240"

[[markata-go.blogroll.feeds]]
url = "https://simonwillison.net/atom/everything/"
title = "Simon Willison"
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	if !config.Blogroll.Enabled {
		t.Error("Blogroll.Enabled should be true")
	}

	expectedURL := "https://shots.waylonwalker.com/shot/?url={url}&height=160&width=240"
	if config.Blogroll.FallbackImageService != expectedURL {
		t.Errorf("Blogroll.FallbackImageService = %q, want %q",
			config.Blogroll.FallbackImageService, expectedURL)
	}

	if len(config.Blogroll.Feeds) != 1 {
		t.Errorf("len(Blogroll.Feeds) = %d, want 1", len(config.Blogroll.Feeds))
	}
}

// TestParseYAML_BlogrollFallbackImageService tests parsing of fallback_image_service from YAML
func TestParseYAML_BlogrollFallbackImageService(t *testing.T) {
	data := []byte(`
markata-go:
  title: "Test Site"
  blogroll:
    enabled: true
    fallback_image_service: "https://shots.waylonwalker.com/shot/?url={url}&height=160&width=240"
    feeds:
      - url: "https://simonwillison.net/atom/everything/"
        title: "Simon Willison"
`)

	config, err := ParseYAML(data)
	if err != nil {
		t.Fatalf("ParseYAML() error = %v", err)
	}

	if !config.Blogroll.Enabled {
		t.Error("Blogroll.Enabled should be true")
	}

	expectedURL := "https://shots.waylonwalker.com/shot/?url={url}&height=160&width=240"
	if config.Blogroll.FallbackImageService != expectedURL {
		t.Errorf("Blogroll.FallbackImageService = %q, want %q",
			config.Blogroll.FallbackImageService, expectedURL)
	}

	if len(config.Blogroll.Feeds) != 1 {
		t.Errorf("len(Blogroll.Feeds) = %d, want 1", len(config.Blogroll.Feeds))
	}
}

// TestParseJSON_BlogrollFallbackImageService tests parsing of fallback_image_service from JSON
func TestParseJSON_BlogrollFallbackImageService(t *testing.T) {
	data := []byte(`{
  "markata-go": {
    "title": "Test Site",
    "blogroll": {
      "enabled": true,
      "fallback_image_service": "https://shots.waylonwalker.com/shot/?url={url}&height=160&width=240",
      "feeds": [
        {
          "url": "https://simonwillison.net/atom/everything/",
          "title": "Simon Willison"
        }
      ]
    }
  }
}`)

	config, err := ParseJSON(data)
	if err != nil {
		t.Fatalf("ParseJSON() error = %v", err)
	}

	if !config.Blogroll.Enabled {
		t.Error("Blogroll.Enabled should be true")
	}

	expectedURL := "https://shots.waylonwalker.com/shot/?url={url}&height=160&width=240"
	if config.Blogroll.FallbackImageService != expectedURL {
		t.Errorf("Blogroll.FallbackImageService = %q, want %q",
			config.Blogroll.FallbackImageService, expectedURL)
	}

	if len(config.Blogroll.Feeds) != 1 {
		t.Errorf("len(Blogroll.Feeds) = %d, want 1", len(config.Blogroll.Feeds))
	}
}

// TestParseTOML_Shortcuts tests parsing of shortcuts from TOML config
func TestParseTOML_Shortcuts(t *testing.T) {
	data := []byte(`
[markata-go]
title = "Test Site"

[markata-go.shortcuts.navigation]
"g t" = "/tags/"
"g a" = "/about/"
"g h" = "/"
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	if config.Shortcuts.Navigation == nil {
		t.Fatal("Shortcuts.Navigation should not be nil")
	}

	if len(config.Shortcuts.Navigation) != 3 {
		t.Errorf("len(Shortcuts.Navigation) = %d, want 3", len(config.Shortcuts.Navigation))
	}

	tests := []struct {
		key  string
		want string
	}{
		{"g t", "/tags/"},
		{"g a", "/about/"},
		{"g h", "/"},
	}

	for _, tt := range tests {
		if got := config.Shortcuts.Navigation[tt.key]; got != tt.want {
			t.Errorf("Shortcuts.Navigation[%q] = %q, want %q", tt.key, got, tt.want)
		}
	}
}

// TestParseTOML_Shortcuts_Empty tests parsing of empty shortcuts from TOML config
func TestParseTOML_Shortcuts_Empty(t *testing.T) {
	data := []byte(`
[markata-go]
title = "Test Site"
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	// Empty shortcuts should still work (navigation may be nil or empty)
	if len(config.Shortcuts.Navigation) != 0 {
		t.Errorf("Shortcuts.Navigation should be nil or empty, got %v", config.Shortcuts.Navigation)
	}
}

// TestParseYAML_Shortcuts tests parsing of shortcuts from YAML config
func TestParseYAML_Shortcuts(t *testing.T) {
	data := []byte(`
markata-go:
  title: Test Site
  shortcuts:
    navigation:
      "g t": "/tags/"
      "g a": "/about/"
`)

	config, err := ParseYAML(data)
	if err != nil {
		t.Fatalf("ParseYAML() error = %v", err)
	}

	if config.Shortcuts.Navigation == nil {
		t.Fatal("Shortcuts.Navigation should not be nil")
	}

	if len(config.Shortcuts.Navigation) != 2 {
		t.Errorf("len(Shortcuts.Navigation) = %d, want 2", len(config.Shortcuts.Navigation))
	}

	if got := config.Shortcuts.Navigation["g t"]; got != "/tags/" {
		t.Errorf("Shortcuts.Navigation[\"g t\"] = %q, want %q", got, "/tags/")
	}

	if got := config.Shortcuts.Navigation["g a"]; got != "/about/" {
		t.Errorf("Shortcuts.Navigation[\"g a\"] = %q, want %q", got, "/about/")
	}
}

// TestParseJSON_Shortcuts tests parsing of shortcuts from JSON config
func TestParseJSON_Shortcuts(t *testing.T) {
	data := []byte(`{
  "markata-go": {
    "title": "Test Site",
    "shortcuts": {
      "navigation": {
        "g t": "/tags/",
        "g a": "/about/"
      }
    }
  }
}`)

	config, err := ParseJSON(data)
	if err != nil {
		t.Fatalf("ParseJSON() error = %v", err)
	}

	if config.Shortcuts.Navigation == nil {
		t.Fatal("Shortcuts.Navigation should not be nil")
	}

	if len(config.Shortcuts.Navigation) != 2 {
		t.Errorf("len(Shortcuts.Navigation) = %d, want 2", len(config.Shortcuts.Navigation))
	}

	if got := config.Shortcuts.Navigation["g t"]; got != "/tags/" {
		t.Errorf("Shortcuts.Navigation[\"g t\"] = %q, want %q", got, "/tags/")
	}

	if got := config.Shortcuts.Navigation["g a"]; got != "/about/" {
		t.Errorf("Shortcuts.Navigation[\"g a\"] = %q, want %q", got, "/about/")
	}
}

// TestParseTOML_MentionsFromPostsDefault tests that default from_posts source
// is applied when mentions section exists but no from_posts are configured.
func TestParseTOML_MentionsFromPostsDefault(t *testing.T) {
	data := []byte(`
[markata-go]
output_dir = "public"

[markata-go.mentions]
css_class = "mention"
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	if len(config.Mentions.FromPosts) == 0 {
		t.Fatal("Mentions.FromPosts should have default entries when not configured")
	}

	if config.Mentions.FromPosts[0].Filter != "template == 'contact'" {
		t.Errorf("default FromPosts[0].Filter = %q, want %q",
			config.Mentions.FromPosts[0].Filter, "template == 'contact'")
	}

	if config.Mentions.FromPosts[0].HandleField != "handle" {
		t.Errorf("default FromPosts[0].HandleField = %q, want %q",
			config.Mentions.FromPosts[0].HandleField, "handle")
	}
}

// TestParseTOML_MentionsFromPostsExplicit tests that explicit from_posts
// overrides the defaults and includes AvatarField.
func TestParseTOML_MentionsFromPostsExplicit(t *testing.T) {
	data := []byte(`
[markata-go]
output_dir = "public"

[markata-go.mentions]
css_class = "mention"

[[markata-go.mentions.from_posts]]
filter = "'team' in tags"
handle_field = "github_handle"
aliases_field = "nicknames"
avatar_field = "photo"
`)

	config, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	if len(config.Mentions.FromPosts) != 1 {
		t.Fatalf("len(Mentions.FromPosts) = %d, want 1", len(config.Mentions.FromPosts))
	}

	src := config.Mentions.FromPosts[0]
	if src.Filter != "'team' in tags" {
		t.Errorf("FromPosts[0].Filter = %q, want %q", src.Filter, "'team' in tags")
	}
	if src.HandleField != "github_handle" {
		t.Errorf("FromPosts[0].HandleField = %q, want %q", src.HandleField, "github_handle")
	}
	if src.AliasesField != "nicknames" {
		t.Errorf("FromPosts[0].AliasesField = %q, want %q", src.AliasesField, "nicknames")
	}
	if src.AvatarField != "photo" {
		t.Errorf("FromPosts[0].AvatarField = %q, want %q", src.AvatarField, "photo")
	}
}

// TestParseYAML_MentionsFromPostsDefault tests default from_posts in YAML config.
func TestParseYAML_MentionsFromPostsDefault(t *testing.T) {
	data := []byte(`
markata-go:
  output_dir: public
  mentions:
    css_class: mention
`)

	config, err := ParseYAML(data)
	if err != nil {
		t.Fatalf("ParseYAML() error = %v", err)
	}

	if len(config.Mentions.FromPosts) == 0 {
		t.Fatal("Mentions.FromPosts should have default entries when not configured")
	}

	if config.Mentions.FromPosts[0].Filter != "template == 'contact'" {
		t.Errorf("default FromPosts[0].Filter = %q, want %q",
			config.Mentions.FromPosts[0].Filter, "template == 'contact'")
	}
}

// TestParseJSON_MentionsFromPostsDefault tests default from_posts in JSON config.
func TestParseJSON_MentionsFromPostsDefault(t *testing.T) {
	data := []byte(`{
  "markata-go": {
    "output_dir": "public",
    "mentions": {
      "css_class": "mention"
    }
  }
}`)

	config, err := ParseJSON(data)
	if err != nil {
		t.Fatalf("ParseJSON() error = %v", err)
	}

	if len(config.Mentions.FromPosts) == 0 {
		t.Fatal("Mentions.FromPosts should have default entries when not configured")
	}

	if config.Mentions.FromPosts[0].Filter != "template == 'contact'" {
		t.Errorf("default FromPosts[0].Filter = %q, want %q",
			config.Mentions.FromPosts[0].Filter, "template == 'contact'")
	}
}
