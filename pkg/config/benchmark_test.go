package config

import (
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkLoad_TOML measures TOML config loading performance.
func BenchmarkLoad_TOML(b *testing.B) {
	dir := b.TempDir()
	configPath := filepath.Join(dir, "markata-go.toml")
	content := generateTOMLConfig()
	//nolint:gosec // G306: test files don't need restrictive permissions
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Load(configPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoad_YAML measures YAML config loading performance.
func BenchmarkLoad_YAML(b *testing.B) {
	dir := b.TempDir()
	configPath := filepath.Join(dir, "markata-go.yaml")
	content := generateYAMLConfig()
	//nolint:gosec // G306: test files don't need restrictive permissions
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Load(configPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoad_JSON measures JSON config loading performance.
func BenchmarkLoad_JSON(b *testing.B) {
	dir := b.TempDir()
	configPath := filepath.Join(dir, "markata-go.json")
	content := generateJSONConfig()
	//nolint:gosec // G306: test files don't need restrictive permissions
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Load(configPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoadAndValidate measures config loading with validation.
func BenchmarkLoadAndValidate(b *testing.B) {
	dir := b.TempDir()
	configPath := filepath.Join(dir, "markata-go.toml")
	content := generateTOMLConfig()
	//nolint:gosec // G306: test files don't need restrictive permissions
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := LoadAndValidate(configPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoadWithDefaults measures loading config with all defaults.
func BenchmarkLoadWithDefaults(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := LoadWithDefaults()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDefaultConfig measures creating the default config.
func BenchmarkDefaultConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultConfig()
	}
}

// BenchmarkMergeConfigs measures config merging performance.
func BenchmarkMergeConfigs(b *testing.B) {
	base := DefaultConfig()
	override := DefaultConfig()
	override.OutputDir = "public"
	override.Title = "My Site"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MergeConfigs(base, override)
	}
}

// BenchmarkValidateConfig measures config validation performance.
func BenchmarkValidateConfig(b *testing.B) {
	config := DefaultConfig()
	config.OutputDir = "public"
	config.URL = "https://example.com"
	config.Title = "My Site"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateConfig(config)
	}
}

// BenchmarkParseTOML measures raw TOML parsing performance.
func BenchmarkParseTOML(b *testing.B) {
	content := []byte(generateTOMLConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseTOML(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseYAML measures raw YAML parsing performance.
func BenchmarkParseYAML(b *testing.B) {
	content := []byte(generateYAMLConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseYAML(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseJSON measures raw JSON parsing performance.
func BenchmarkParseJSON(b *testing.B) {
	content := []byte(generateJSONConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseJSON(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoad_ComplexConfig measures loading a complex config with many feeds.
func BenchmarkLoad_ComplexConfig(b *testing.B) {
	dir := b.TempDir()
	configPath := filepath.Join(dir, "markata-go.toml")
	content := generateComplexTOMLConfig()
	//nolint:gosec // G306: test files don't need restrictive permissions
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Load(configPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDiscover measures config file discovery performance.
func BenchmarkDiscover(b *testing.B) {
	dir := b.TempDir()
	//nolint:gosec // G306: test files don't need restrictive permissions
	if err := os.WriteFile(filepath.Join(dir, "markata-go.toml"), []byte("[markata-go]\n"), 0o644); err != nil {
		b.Fatal(err)
	}

	// Change to the directory for discovery
	oldDir, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			b.Logf("failed to change back to original directory: %v", err)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Discover()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper functions to generate test configs

func generateTOMLConfig() string {
	return `[markata-go]
output_dir = "public"
url = "https://example.com"
title = "My Site"
description = "A great site built with markata-go"
author = "Test Author"
assets_dir = "assets"
templates_dir = "templates"
concurrency = 4

[markata-go.glob]
patterns = ["posts/**/*.md", "pages/*.md", "docs/**/*.md"]
use_gitignore = true

[markata-go.markdown]
extensions = ["tables", "footnotes", "syntax-highlighting"]

[markata-go.feed_defaults]
items_per_page = 10
orphan_threshold = 3

[markata-go.feed_defaults.formats]
html = true
rss = true

[[markata-go.feeds]]
slug = "blog"
title = "Blog"
filter = "published == True"
sort = "date"
reverse = true

[markata-go.feeds.formats]
html = true
rss = true
atom = true
`
}

func generateYAMLConfig() string {
	return `markata-go:
  output_dir: public
  url: https://example.com
  title: My Site
  description: A great site built with markata-go
  author: Test Author
  assets_dir: assets
  templates_dir: templates
  concurrency: 4

  glob:
    patterns:
      - "posts/**/*.md"
      - "pages/*.md"
      - "docs/**/*.md"
    use_gitignore: true

  markdown:
    extensions:
      - tables
      - footnotes
      - syntax-highlighting

  feed_defaults:
    items_per_page: 10
    orphan_threshold: 3
    formats:
      html: true
      rss: true

  feeds:
    - slug: blog
      title: Blog
      filter: "published == True"
      sort: date
      reverse: true
      formats:
        html: true
        rss: true
        atom: true
`
}

func generateJSONConfig() string {
	return `{
  "markata-go": {
    "output_dir": "public",
    "url": "https://example.com",
    "title": "My Site",
    "description": "A great site built with markata-go",
    "author": "Test Author",
    "assets_dir": "assets",
    "templates_dir": "templates",
    "concurrency": 4,
    "glob": {
      "patterns": ["posts/**/*.md", "pages/*.md", "docs/**/*.md"],
      "use_gitignore": true
    },
    "markdown": {
      "extensions": ["tables", "footnotes", "syntax-highlighting"]
    },
    "feed_defaults": {
      "items_per_page": 10,
      "orphan_threshold": 3,
      "formats": {
        "html": true,
        "rss": true
      }
    },
    "feeds": [
      {
        "slug": "blog",
        "title": "Blog",
        "filter": "published == True",
        "sort": "date",
        "reverse": true,
        "formats": {
          "html": true,
          "rss": true,
          "atom": true
        }
      }
    ]
  }
}`
}

func generateComplexTOMLConfig() string {
	return `[markata-go]
output_dir = "public"
url = "https://example.com"
title = "My Complex Site"
description = "A complex site with many feeds and settings"
author = "Test Author"
assets_dir = "assets"
templates_dir = "templates"
concurrency = 8
hooks = ["default", "sitemap", "robots", "seo"]
disabled_hooks = []

[markata-go.glob]
patterns = ["posts/**/*.md", "pages/*.md", "docs/**/*.md", "notes/**/*.md"]
use_gitignore = true

[markata-go.markdown]
extensions = ["tables", "footnotes", "syntax-highlighting", "autolinks", "strikethrough"]

[markata-go.feed_defaults]
items_per_page = 15
orphan_threshold = 4

[markata-go.feed_defaults.formats]
html = true
rss = true
atom = true
json = false

[markata-go.feed_defaults.syndication]
max_items = 50
include_content = true

[[markata-go.feeds]]
slug = "blog"
title = "Blog"
description = "All blog posts"
filter = "published == True and draft != True"
sort = "date"
reverse = true
items_per_page = 10

[markata-go.feeds.formats]
html = true
rss = true
atom = true

[[markata-go.feeds]]
slug = "projects"
title = "Projects"
description = "Project showcase"
filter = "tags contains project"
sort = "date"
reverse = true

[[markata-go.feeds]]
slug = "notes"
title = "Notes"
description = "Quick notes and thoughts"
filter = "path startswith notes/"
sort = "date"
reverse = true
items_per_page = 20

[[markata-go.feeds]]
slug = "docs"
title = "Documentation"
description = "Documentation pages"
filter = "path startswith docs/"
sort = "title"
reverse = false

[[markata-go.feeds]]
slug = "archive"
title = "Archive"
description = "All posts archive"
filter = "published == True"
sort = "date"
reverse = true
items_per_page = 50
`
}
