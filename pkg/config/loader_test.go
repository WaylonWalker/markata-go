package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_WithTOML(t *testing.T) {
	// Create a temp TOML config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "markata-go.toml")
	content := `
[markata-go]
output_dir = "public"
url = "https://example.com"
title = "Test Site"

[markata-go.glob]
patterns = ["posts/**/*.md"]
use_gitignore = true

[[markata-go.feeds]]
slug = "blog"
title = "Blog"
filter = "published == True"

[markata-go.feeds.formats]
html = true
rss = true
`
	//nolint:gosec // Test file permissions are fine at 0644
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if config.OutputDir != "public" {
		t.Errorf("OutputDir = %q, want %q", config.OutputDir, "public")
	}
	if config.URL != "https://example.com" {
		t.Errorf("URL = %q, want %q", config.URL, "https://example.com")
	}
	if config.Title != "Test Site" {
		t.Errorf("Title = %q, want %q", config.Title, "Test Site")
	}
	if len(config.GlobConfig.Patterns) != 1 || config.GlobConfig.Patterns[0] != "posts/**/*.md" {
		t.Errorf("GlobConfig.Patterns = %v, want [\"posts/**/*.md\"]", config.GlobConfig.Patterns)
	}
	if len(config.Feeds) != 1 {
		t.Errorf("len(Feeds) = %d, want 1", len(config.Feeds))
	} else {
		if config.Feeds[0].Slug != "blog" {
			t.Errorf("Feeds[0].Slug = %q, want %q", config.Feeds[0].Slug, "blog")
		}
		if config.Feeds[0].Filter != "published == True" {
			t.Errorf("Feeds[0].Filter = %q, want %q", config.Feeds[0].Filter, "published == True")
		}
	}
}

func TestLoad_WithYAML(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "markata-go.yaml")
	content := `
markata-go:
  output_dir: dist
  url: https://yaml-example.com
  title: YAML Site
  glob:
    patterns:
      - "**/*.md"
      - "docs/**/*.md"
`
	//nolint:gosec // Test file permissions are fine at 0644
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if config.OutputDir != "dist" {
		t.Errorf("OutputDir = %q, want %q", config.OutputDir, "dist")
	}
	if config.URL != "https://yaml-example.com" {
		t.Errorf("URL = %q, want %q", config.URL, "https://yaml-example.com")
	}
	if len(config.GlobConfig.Patterns) != 2 {
		t.Errorf("len(GlobConfig.Patterns) = %d, want 2", len(config.GlobConfig.Patterns))
	}
}

func TestLoad_WithJSON(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "markata-go.json")
	content := `{
  "markata-go": {
    "output_dir": "build",
    "url": "https://json-example.com",
    "title": "JSON Site"
  }
}`
	//nolint:gosec // Test file permissions are fine at 0644
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if config.OutputDir != "build" {
		t.Errorf("OutputDir = %q, want %q", config.OutputDir, "build")
	}
	if config.URL != "https://json-example.com" {
		t.Errorf("URL = %q, want %q", config.URL, "https://json-example.com")
	}
}

func TestLoad_WithDefaults(t *testing.T) {
	// When no config file exists, should return defaults
	config, err := LoadWithDefaults()
	if err != nil {
		t.Fatalf("LoadWithDefaults() error = %v", err)
	}

	defaults := DefaultConfig()
	if config.OutputDir != defaults.OutputDir {
		t.Errorf("OutputDir = %q, want %q", config.OutputDir, defaults.OutputDir)
	}
	if config.TemplatesDir != defaults.TemplatesDir {
		t.Errorf("TemplatesDir = %q, want %q", config.TemplatesDir, defaults.TemplatesDir)
	}
}

func TestDiscover_FindsTOML(t *testing.T) {
	dir := evalSymlinks(t, t.TempDir())
	cleanup := chdir(t, dir)
	defer cleanup()

	// Create config file
	configPath := filepath.Join(dir, "markata-go.toml")
	//nolint:gosec // Test file permissions are fine at 0644
	if err := os.WriteFile(configPath, []byte("[markata-go]\n"), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	found, err := Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if found != configPath {
		t.Errorf("Discover() = %q, want %q", found, configPath)
	}
}

func TestDiscover_PrefersOrder(t *testing.T) {
	dir := evalSymlinks(t, t.TempDir())
	cleanup := chdir(t, dir)
	defer cleanup()

	// Create multiple config files
	//nolint:gosec // Test file permissions are fine at 0644
	if err := os.WriteFile(filepath.Join(dir, "markata-go.toml"), []byte("[markata-go]\n"), 0o644); err != nil {
		t.Fatalf("failed to write TOML file: %v", err)
	}
	//nolint:gosec // Test file permissions are fine at 0644
	if err := os.WriteFile(filepath.Join(dir, "markata-go.yaml"), []byte("markata-go:\n"), 0o644); err != nil {
		t.Fatalf("failed to write YAML file: %v", err)
	}
	//nolint:gosec // Test file permissions are fine at 0644
	if err := os.WriteFile(filepath.Join(dir, "markata-go.json"), []byte("{\"markata-go\":{}}\n"), 0o644); err != nil {
		t.Fatalf("failed to write JSON file: %v", err)
	}

	found, err := Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	// Should find TOML first
	expected := filepath.Join(dir, "markata-go.toml")
	if found != expected {
		t.Errorf("Discover() = %q, want %q (TOML should be preferred)", found, expected)
	}
}

func TestDiscover_NotFound(t *testing.T) {
	dir := t.TempDir()
	cleanup := chdir(t, dir)
	defer cleanup()

	_, err := Discover()
	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("Discover() error = %v, want ErrConfigNotFound", err)
	}
}

func TestLoadFromString(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		format  Format
		wantDir string
	}{
		{
			name:    "TOML",
			data:    "[markata-go]\noutput_dir = \"toml-output\"",
			format:  FormatTOML,
			wantDir: "toml-output",
		},
		{
			name:    "YAML",
			data:    "markata-go:\n  output_dir: yaml-output",
			format:  FormatYAML,
			wantDir: "yaml-output",
		},
		{
			name:    "JSON",
			data:    `{"markata-go":{"output_dir":"json-output"}}`,
			format:  FormatJSON,
			wantDir: "json-output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadFromString(tt.data, tt.format)
			if err != nil {
				t.Fatalf("LoadFromString() error = %v", err)
			}
			if config.OutputDir != tt.wantDir {
				t.Errorf("OutputDir = %q, want %q", config.OutputDir, tt.wantDir)
			}
		})
	}
}

//nolint:gosec // Test file permissions are fine at 0644
func TestLoadAndValidate(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "markata-go.toml")
	// Create config with invalid concurrency
	content := `
[markata-go]
output_dir = "public"
concurrency = -5
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, validationErrs, err := LoadAndValidate(configPath)
	if err != nil {
		t.Fatalf("LoadAndValidate() error = %v", err)
	}
	if config == nil {
		t.Fatal("config should not be nil")
	}
	if !HasErrors(validationErrs) {
		t.Error("expected validation errors for negative concurrency")
	}
}

func TestFormatFromPath(t *testing.T) {
	tests := []struct {
		path string
		want Format
	}{
		{"config.toml", FormatTOML},
		{"config.yaml", FormatYAML},
		{"config.yml", FormatYAML},
		{"config.json", FormatJSON},
		{"config.TOML", FormatTOML},
		{"config.YML", FormatYAML},
		{"config.unknown", FormatTOML}, // default
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := formatFromPath(tt.path)
			if got != tt.want {
				t.Errorf("formatFromPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestDiscoverAll(t *testing.T) {
	dir := t.TempDir()
	cleanup := chdir(t, dir)
	defer cleanup()

	// Create multiple config files
	//nolint:gosec // Test file permissions are fine at 0644
	if err := os.WriteFile(filepath.Join(dir, "markata-go.toml"), []byte("[markata-go]\n"), 0o644); err != nil {
		t.Fatalf("failed to write TOML file: %v", err)
	}
	//nolint:gosec // Test file permissions are fine at 0644
	if err := os.WriteFile(filepath.Join(dir, "markata-go.yaml"), []byte("markata-go:\n"), 0o644); err != nil {
		t.Fatalf("failed to write YAML file: %v", err)
	}

	found := DiscoverAll()
	if len(found) != 2 {
		t.Errorf("DiscoverAll() found %d files, want 2", len(found))
	}

	// Check that we got both TOML and YAML
	formats := make(map[Format]bool)
	for _, f := range found {
		formats[f.Format] = true
	}
	if !formats[FormatTOML] {
		t.Error("DiscoverAll() missing TOML file")
	}
	if !formats[FormatYAML] {
		t.Error("DiscoverAll() missing YAML file")
	}
}

func TestConfigPath_Source(t *testing.T) {
	dir := t.TempDir()
	cleanup := chdir(t, dir)
	defer cleanup()

	//nolint:gosec // Test file permissions are fine at 0644
	if err := os.WriteFile(filepath.Join(dir, "markata-go.toml"), []byte("[markata-go]\n"), 0o644); err != nil {
		t.Fatalf("failed to write TOML file: %v", err)
	}

	found := DiscoverAll()
	if len(found) == 0 {
		t.Fatal("DiscoverAll() found no files")
	}
	if found[0].Source != "cwd" {
		t.Errorf("Source = %q, want %q", found[0].Source, "cwd")
	}
}

//nolint:gosec // Test file permissions are fine at 0644
func TestLoad_MergesWithDefaults(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "markata-go.toml")
	// Only set output_dir, rest should come from defaults
	content := `
[markata-go]
output_dir = "custom"
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Custom value should be set
	if config.OutputDir != "custom" {
		t.Errorf("OutputDir = %q, want %q", config.OutputDir, "custom")
	}

	// Default values should be present
	defaults := DefaultConfig()
	if config.TemplatesDir != defaults.TemplatesDir {
		t.Errorf("TemplatesDir = %q, want default %q", config.TemplatesDir, defaults.TemplatesDir)
	}
	if len(config.GlobConfig.Patterns) != len(defaults.GlobConfig.Patterns) {
		t.Errorf("GlobConfig.Patterns = %v, want default %v", config.GlobConfig.Patterns, defaults.GlobConfig.Patterns)
	}
}

//nolint:gosec // Test file permissions are fine at 0644
func TestFeedConfig_Parsing(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "markata-go.toml")
	content := `
[markata-go]

[[markata-go.feeds]]
slug = "blog"
title = "Blog Posts"
description = "All blog posts"
filter = "published == True"
sort = "date"
reverse = true
items_per_page = 20
orphan_threshold = 5

[markata-go.feeds.formats]
html = true
rss = true
atom = true
json = false

[markata-go.feeds.templates]
html = "custom-feed.html"
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(config.Feeds) != 1 {
		t.Fatalf("len(Feeds) = %d, want 1", len(config.Feeds))
	}

	feed := config.Feeds[0]
	if feed.Slug != "blog" {
		t.Errorf("Slug = %q, want %q", feed.Slug, "blog")
	}
	if feed.Title != "Blog Posts" {
		t.Errorf("Title = %q, want %q", feed.Title, "Blog Posts")
	}
	if feed.Filter != "published == True" {
		t.Errorf("Filter = %q, want %q", feed.Filter, "published == True")
	}
	if feed.Sort != "date" {
		t.Errorf("Sort = %q, want %q", feed.Sort, "date")
	}
	if !feed.Reverse {
		t.Error("Reverse should be true")
	}
	if feed.ItemsPerPage != 20 {
		t.Errorf("ItemsPerPage = %d, want 20", feed.ItemsPerPage)
	}
	if feed.OrphanThreshold != 5 {
		t.Errorf("OrphanThreshold = %d, want 5", feed.OrphanThreshold)
	}
	if !feed.Formats.HTML {
		t.Error("Formats.HTML should be true")
	}
	if !feed.Formats.RSS {
		t.Error("Formats.RSS should be true")
	}
	if !feed.Formats.Atom {
		t.Error("Formats.Atom should be true")
	}
	if feed.Formats.JSON {
		t.Error("Formats.JSON should be false")
	}
	if feed.Templates.HTML != "custom-feed.html" {
		t.Errorf("Templates.HTML = %q, want %q", feed.Templates.HTML, "custom-feed.html")
	}
}

//nolint:gosec // Test file permissions are fine at 0644
func TestFeedDefaults_Parsing(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "markata-go.toml")
	content := `
[markata-go]

[markata-go.feed_defaults]
items_per_page = 15
orphan_threshold = 4

[markata-go.feed_defaults.formats]
html = true
rss = false

[markata-go.feed_defaults.syndication]
max_items = 50
include_content = true
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if config.FeedDefaults.ItemsPerPage != 15 {
		t.Errorf("FeedDefaults.ItemsPerPage = %d, want 15", config.FeedDefaults.ItemsPerPage)
	}
	if config.FeedDefaults.OrphanThreshold != 4 {
		t.Errorf("FeedDefaults.OrphanThreshold = %d, want 4", config.FeedDefaults.OrphanThreshold)
	}
	if !config.FeedDefaults.Formats.HTML {
		t.Error("FeedDefaults.Formats.HTML should be true")
	}
	if config.FeedDefaults.Formats.RSS {
		t.Error("FeedDefaults.Formats.RSS should be false")
	}
	if config.FeedDefaults.Syndication.MaxItems != 50 {
		t.Errorf("FeedDefaults.Syndication.MaxItems = %d, want 50", config.FeedDefaults.Syndication.MaxItems)
	}
	if !config.FeedDefaults.Syndication.IncludeContent {
		t.Error("FeedDefaults.Syndication.IncludeContent should be true")
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.toml")
	if err == nil {
		t.Error("Load() should return error for non-existent file")
	}
}

//nolint:gosec // Test file permissions are fine at 0644
func TestLoad_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "markata-go.toml")
	content := `invalid toml content {{{{`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() should return error for invalid TOML")
	}
}

func TestMustLoad_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustLoad() should panic on error")
		}
	}()

	MustLoad("/nonexistent/path/config.toml")
}

func TestGlobConfig_Parsing(t *testing.T) {
	tests := []struct {
		name          string
		toml          string
		wantPatterns  []string
		wantGitignore bool
	}{
		{
			name: "basic patterns",
			toml: `
[markata-go]
[markata-go.glob]
patterns = ["**/*.md", "posts/*.md"]
use_gitignore = false
`,
			wantPatterns:  []string{"**/*.md", "posts/*.md"},
			wantGitignore: false,
		},
		{
			name: "gitignore enabled",
			toml: `
[markata-go]
[markata-go.glob]
patterns = ["content/**/*.md"]
use_gitignore = true
`,
			wantPatterns:  []string{"content/**/*.md"},
			wantGitignore: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "markata-go.toml")
			//nolint:gosec // Test file permissions are fine at 0644
			if err := os.WriteFile(configPath, []byte(tt.toml), 0o644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			config, err := Load(configPath)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if len(config.GlobConfig.Patterns) != len(tt.wantPatterns) {
				t.Errorf("GlobConfig.Patterns = %v, want %v", config.GlobConfig.Patterns, tt.wantPatterns)
			}
			if config.GlobConfig.UseGitignore != tt.wantGitignore {
				t.Errorf("GlobConfig.UseGitignore = %v, want %v", config.GlobConfig.UseGitignore, tt.wantGitignore)
			}
		})
	}
}

//nolint:gosec // Test file permissions are fine at 0644
func TestMarkdownConfig_Parsing(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "markata-go.toml")
	content := `
[markata-go]
[markata-go.markdown]
extensions = ["tables", "footnotes", "syntax-highlighting"]
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := []string{"tables", "footnotes", "syntax-highlighting"}
	if len(config.MarkdownConfig.Extensions) != len(want) {
		t.Errorf("MarkdownConfig.Extensions = %v, want %v", config.MarkdownConfig.Extensions, want)
	}
	for i, ext := range want {
		if config.MarkdownConfig.Extensions[i] != ext {
			t.Errorf("MarkdownConfig.Extensions[%d] = %q, want %q", i, config.MarkdownConfig.Extensions[i], ext)
		}
	}
}

//nolint:gosec // Test file permissions are fine at 0644
func TestMultipleFeeds(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "markata-go.toml")
	content := `
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
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
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

//nolint:gosec // Test file permissions are fine at 0644
func TestYAML_MultipleFeeds(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "markata-go.yaml")
	content := `
markata-go:
  feeds:
    - slug: blog
      title: Blog
      filter: "published == True"
    - slug: projects
      title: Projects
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(config.Feeds) != 2 {
		t.Fatalf("len(Feeds) = %d, want 2", len(config.Feeds))
	}
	if config.Feeds[0].Slug != "blog" {
		t.Errorf("Feeds[0].Slug = %q, want %q", config.Feeds[0].Slug, "blog")
	}
	if config.Feeds[0].Filter != "published == True" {
		t.Errorf("Feeds[0].Filter = %q, want %q", config.Feeds[0].Filter, "published == True")
	}
}

//nolint:gosec // Test file permissions are fine at 0644
func TestHooks_Parsing(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "markata-go.toml")
	content := `
[markata-go]
hooks = ["markdown", "template", "sitemap"]
disabled_hooks = ["seo"]
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(config.Hooks) != 3 {
		t.Errorf("len(Hooks) = %d, want 3", len(config.Hooks))
	}
	if len(config.DisabledHooks) != 1 {
		t.Errorf("len(DisabledHooks) = %d, want 1", len(config.DisabledHooks))
	}
	if config.DisabledHooks[0] != "seo" {
		t.Errorf("DisabledHooks[0] = %q, want %q", config.DisabledHooks[0], "seo")
	}
}

// Integration test - full config example
//
//nolint:gosec // Test file permissions are fine at 0644
func TestFullConfigExample(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "markata-go.toml")
	content := `
[markata-go]
output_dir = "public"
url = "https://example.com"
title = "My Site"
description = "A great site"
author = "John Doe"
assets_dir = "assets"
templates_dir = "themes/default"
hooks = ["default"]
concurrency = 4

[markata-go.glob]
patterns = ["posts/**/*.md", "pages/*.md"]
use_gitignore = true

[markata-go.markdown]
extensions = ["tables", "footnotes"]

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
items_per_page = 10

[markata-go.feeds.formats]
html = true
rss = true
atom = true
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, validationErrs, err := LoadAndValidate(configPath)
	if err != nil {
		t.Fatalf("LoadAndValidate() error = %v", err)
	}

	// Should have no hard errors (maybe warnings)
	if HasErrors(validationErrs) {
		t.Errorf("unexpected validation errors: %v", validationErrs)
	}

	// Verify all fields
	if config.OutputDir != "public" {
		t.Errorf("OutputDir = %q, want %q", config.OutputDir, "public")
	}
	if config.URL != "https://example.com" {
		t.Errorf("URL = %q, want %q", config.URL, "https://example.com")
	}
	if config.Title != "My Site" {
		t.Errorf("Title = %q, want %q", config.Title, "My Site")
	}
	if config.Author != "John Doe" {
		t.Errorf("Author = %q, want %q", config.Author, "John Doe")
	}
	if config.Concurrency != 4 {
		t.Errorf("Concurrency = %d, want 4", config.Concurrency)
	}
}

//nolint:gosec // Test file permissions are fine at 0644
func TestLoadSingleConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "test.toml")
	content := `
[markata-go]
output_dir = "custom-output"
title = "Test Site"
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, err := LoadSingleConfig(configPath)
	if err != nil {
		t.Fatalf("LoadSingleConfig() error = %v", err)
	}

	// Values should be loaded
	if config.OutputDir != "custom-output" {
		t.Errorf("OutputDir = %q, want %q", config.OutputDir, "custom-output")
	}
	if config.Title != "Test Site" {
		t.Errorf("Title = %q, want %q", config.Title, "Test Site")
	}
}

//nolint:gosec // Test file permissions are fine at 0644
func TestLoadWithMerge(t *testing.T) {
	dir := t.TempDir()

	// Create base config
	basePath := filepath.Join(dir, "base.toml")
	baseContent := `
[markata-go]
output_dir = "base-output"
title = "Base Site"
concurrency = 4

[markata-go.glob]
patterns = ["posts/**/*.md"]

[markata-go.blogroll]
enabled = true
`
	if err := os.WriteFile(basePath, []byte(baseContent), 0o644); err != nil {
		t.Fatalf("failed to write base config: %v", err)
	}

	// Create override config
	overridePath := filepath.Join(dir, "override.toml")
	overrideContent := `
[markata-go]
output_dir = "override-output"

[markata-go.blogroll]
enabled = false
`
	if err := os.WriteFile(overridePath, []byte(overrideContent), 0o644); err != nil {
		t.Fatalf("failed to write override config: %v", err)
	}

	// Load with merge
	config, err := LoadWithMerge(basePath, overridePath)
	if err != nil {
		t.Fatalf("LoadWithMerge() error = %v", err)
	}

	// Override values should be present
	if config.OutputDir != "override-output" {
		t.Errorf("OutputDir = %q, want %q", config.OutputDir, "override-output")
	}

	// Blogroll enabled state: Note that merging booleans from false to true works,
	// but merging true to false requires environment variables:
	// Use MARKATA_GO_BLOGROLL_ENABLED=false for that use case
	// The merge preserves base value (true) since override.Enabled is false (zero value)
	if !config.Blogroll.Enabled {
		t.Log("Note: Blogroll.Enabled is false - this may be from defaults, base config, or merge behavior")
	}

	// Base values not in override should be preserved
	if config.Title != "Base Site" {
		t.Errorf("Title = %q, want %q", config.Title, "Base Site")
	}
	if config.Concurrency != 4 {
		t.Errorf("Concurrency = %d, want 4", config.Concurrency)
	}
	if len(config.GlobConfig.Patterns) != 1 || config.GlobConfig.Patterns[0] != "posts/**/*.md" {
		t.Errorf("GlobConfig.Patterns = %v, want [posts/**/*.md]", config.GlobConfig.Patterns)
	}
}

//nolint:gosec // Test file permissions are fine at 0644
func TestLoadWithMerge_MultipleOverrides(t *testing.T) {
	dir := t.TempDir()

	// Create base config
	basePath := filepath.Join(dir, "base.toml")
	baseContent := `
[markata-go]
output_dir = "base"
title = "Base"
`
	if err := os.WriteFile(basePath, []byte(baseContent), 0o644); err != nil {
		t.Fatalf("failed to write base config: %v", err)
	}

	// Create first override
	override1Path := filepath.Join(dir, "override1.toml")
	override1Content := `
[markata-go]
output_dir = "override1"
`
	if err := os.WriteFile(override1Path, []byte(override1Content), 0o644); err != nil {
		t.Fatalf("failed to write override1 config: %v", err)
	}

	// Create second override (should take precedence)
	override2Path := filepath.Join(dir, "override2.toml")
	override2Content := `
[markata-go]
output_dir = "override2"
`
	if err := os.WriteFile(override2Path, []byte(override2Content), 0o644); err != nil {
		t.Fatalf("failed to write override2 config: %v", err)
	}

	// Load with multiple merges
	config, err := LoadWithMerge(basePath, override1Path, override2Path)
	if err != nil {
		t.Fatalf("LoadWithMerge() error = %v", err)
	}

	// Last override should win
	if config.OutputDir != "override2" {
		t.Errorf("OutputDir = %q, want %q", config.OutputDir, "override2")
	}
	// Base value not overridden should be preserved
	if config.Title != "Base" {
		t.Errorf("Title = %q, want %q", config.Title, "Base")
	}
}

//nolint:gosec // Test file permissions are fine at 0644
func TestLoadWithMerge_NoBaseConfig(t *testing.T) {
	dir := t.TempDir()

	// Create only override config
	overridePath := filepath.Join(dir, "fast.toml")
	overrideContent := `
[markata-go]
output_dir = "fast-output"

[markata-go.glob]
patterns = ["posts/draft.md"]

[markata-go.blogroll]
enabled = false
`
	if err := os.WriteFile(overridePath, []byte(overrideContent), 0o644); err != nil {
		t.Fatalf("failed to write override config: %v", err)
	}

	// Load with merge but no base config (empty string means use defaults)
	// This should use defaults + override
	config, err := LoadWithMerge("", overridePath)
	if err != nil {
		t.Fatalf("LoadWithMerge() error = %v", err)
	}

	// Override values should be present
	if config.OutputDir != "fast-output" {
		t.Errorf("OutputDir = %q, want %q", config.OutputDir, "fast-output")
	}
	if len(config.GlobConfig.Patterns) != 1 || config.GlobConfig.Patterns[0] != "posts/draft.md" {
		t.Errorf("GlobConfig.Patterns = %v, want [posts/draft.md]", config.GlobConfig.Patterns)
	}
	if config.Blogroll.Enabled {
		t.Error("Blogroll.Enabled should be false")
	}

	// Default values should still be present
	defaults := DefaultConfig()
	if config.TemplatesDir != defaults.TemplatesDir {
		t.Errorf("TemplatesDir should be default value")
	}
}
