package config

import (
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestMergeConfigs_NilBase(t *testing.T) {
	override := &models.Config{OutputDir: "override"}
	result := MergeConfigs(nil, override)

	if result != override {
		t.Error("MergeConfigs(nil, override) should return override")
	}
}

func TestMergeConfigs_NilOverride(t *testing.T) {
	base := &models.Config{OutputDir: "base"}
	result := MergeConfigs(base, nil)

	if result != base {
		t.Error("MergeConfigs(base, nil) should return base")
	}
}

func TestMergeConfigs_StringFields(t *testing.T) {
	base := &models.Config{
		OutputDir:   "base-output",
		URL:         "https://base.com",
		Title:       "Base Title",
		Description: "Base Description",
		Author:      "Base Author",
	}
	override := &models.Config{
		OutputDir:   "override-output",
		URL:         "", // Empty, should keep base
		Title:       "Override Title",
		Description: "", // Empty, should keep base
		Author:      "Override Author",
	}

	result := MergeConfigs(base, override)

	tests := []struct {
		field string
		got   string
		want  string
	}{
		{"OutputDir", result.OutputDir, "override-output"},
		{"URL", result.URL, "https://base.com"}, // Kept base
		{"Title", result.Title, "Override Title"},
		{"Description", result.Description, "Base Description"}, // Kept base
		{"Author", result.Author, "Override Author"},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.field, tt.got, tt.want)
		}
	}
}

func TestMergeConfigs_SliceFields(t *testing.T) {
	base := &models.Config{
		Hooks:         []string{"base-hook1", "base-hook2"},
		DisabledHooks: []string{"disabled1"},
	}
	override := &models.Config{
		Hooks:         []string{"override-hook"},
		DisabledHooks: nil, // Nil, should keep base
	}

	result := MergeConfigs(base, override)

	// Override hooks should replace
	if len(result.Hooks) != 1 || result.Hooks[0] != "override-hook" {
		t.Errorf("Hooks = %v, want [\"override-hook\"]", result.Hooks)
	}

	// Nil override should keep base
	if len(result.DisabledHooks) != 1 || result.DisabledHooks[0] != "disabled1" {
		t.Errorf("DisabledHooks = %v, want [\"disabled1\"]", result.DisabledHooks)
	}
}

func TestMergeConfigs_IntFields(t *testing.T) {
	base := &models.Config{Concurrency: 4}
	override := &models.Config{Concurrency: 8}

	result := MergeConfigs(base, override)
	if result.Concurrency != 8 {
		t.Errorf("Concurrency = %d, want 8", result.Concurrency)
	}

	// Zero value should keep base
	override2 := &models.Config{Concurrency: 0}
	result2 := MergeConfigs(base, override2)
	if result2.Concurrency != 4 {
		t.Errorf("Concurrency = %d, want 4 (base value)", result2.Concurrency)
	}
}

func TestMergeConfigs_GlobConfig(t *testing.T) {
	base := &models.Config{
		GlobConfig: models.GlobConfig{
			Patterns:     []string{"base/**/*.md"},
			UseGitignore: true,
		},
	}
	override := &models.Config{
		GlobConfig: models.GlobConfig{
			Patterns:     []string{"override/**/*.md"},
			UseGitignore: false,
		},
	}

	result := MergeConfigs(base, override)

	if len(result.GlobConfig.Patterns) != 1 || result.GlobConfig.Patterns[0] != "override/**/*.md" {
		t.Errorf("GlobConfig.Patterns = %v, want [\"override/**/*.md\"]", result.GlobConfig.Patterns)
	}
	if result.GlobConfig.UseGitignore != false {
		t.Error("GlobConfig.UseGitignore should be false (from override)")
	}
}

func TestMergeConfigs_FeedDefaults(t *testing.T) {
	base := &models.Config{
		FeedDefaults: models.FeedDefaults{
			ItemsPerPage:    10,
			OrphanThreshold: 3,
			Formats: models.FeedFormats{
				HTML: true,
				RSS:  true,
			},
		},
	}
	override := &models.Config{
		FeedDefaults: models.FeedDefaults{
			ItemsPerPage:    20,
			OrphanThreshold: 0, // Zero, should keep base
			Formats: models.FeedFormats{
				Atom: true, // This format is active, so override replaces base
			},
		},
	}

	result := MergeConfigs(base, override)

	if result.FeedDefaults.ItemsPerPage != 20 {
		t.Errorf("FeedDefaults.ItemsPerPage = %d, want 20", result.FeedDefaults.ItemsPerPage)
	}
	if result.FeedDefaults.OrphanThreshold != 3 {
		t.Errorf("FeedDefaults.OrphanThreshold = %d, want 3 (base value)", result.FeedDefaults.OrphanThreshold)
	}
	// Since override has Atom: true, the override formats replace the base
	// HTML and RSS from base are not preserved
	if result.FeedDefaults.Formats.HTML {
		t.Error("FeedDefaults.Formats.HTML should be false (override replaces)")
	}
	if result.FeedDefaults.Formats.RSS {
		t.Error("FeedDefaults.Formats.RSS should be false (override replaces)")
	}
	if !result.FeedDefaults.Formats.Atom {
		t.Error("FeedDefaults.Formats.Atom should be true")
	}
}

func TestMergeConfigs_Feeds(t *testing.T) {
	base := &models.Config{
		Feeds: []models.FeedConfig{
			{Slug: "base-feed", Title: "Base Feed"},
		},
	}
	override := &models.Config{
		Feeds: []models.FeedConfig{
			{Slug: "override-feed1", Title: "Override Feed 1"},
			{Slug: "override-feed2", Title: "Override Feed 2"},
		},
	}

	result := MergeConfigs(base, override)

	// Feeds should be replaced, not merged
	if len(result.Feeds) != 2 {
		t.Fatalf("len(Feeds) = %d, want 2", len(result.Feeds))
	}
	if result.Feeds[0].Slug != "override-feed1" {
		t.Errorf("Feeds[0].Slug = %q, want %q", result.Feeds[0].Slug, "override-feed1")
	}
}

func TestMergeSlice_Replace(t *testing.T) {
	base := []string{"a", "b", "c"}
	override := []string{"x", "y"}

	result := MergeSlice(base, override, false)

	if len(result) != 2 || result[0] != "x" || result[1] != "y" {
		t.Errorf("MergeSlice replace = %v, want [\"x\", \"y\"]", result)
	}
}

func TestMergeSlice_Append(t *testing.T) {
	base := []string{"a", "b", "c"}
	override := []string{"x", "y"}

	result := MergeSlice(base, override, true)

	if len(result) != 5 {
		t.Fatalf("MergeSlice append len = %d, want 5", len(result))
	}
	expected := []string{"a", "b", "c", "x", "y"}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("MergeSlice append[%d] = %q, want %q", i, result[i], v)
		}
	}
}

func TestMergeSlice_EmptyOverride(t *testing.T) {
	base := []string{"a", "b"}
	var override []string

	result := MergeSlice(base, override, false)
	if len(result) != 2 || result[0] != "a" {
		t.Errorf("MergeSlice with empty override = %v, want base", result)
	}
}

func TestAppendHooks(t *testing.T) {
	config := &models.Config{Hooks: []string{"default"}}
	AppendHooks(config, "markdown", "template")

	if len(config.Hooks) != 3 {
		t.Fatalf("len(Hooks) = %d, want 3", len(config.Hooks))
	}
	expected := []string{"default", "markdown", "template"}
	for i, v := range expected {
		if config.Hooks[i] != v {
			t.Errorf("Hooks[%d] = %q, want %q", i, config.Hooks[i], v)
		}
	}
}

func TestAppendDisabledHooks(t *testing.T) {
	config := &models.Config{DisabledHooks: []string{"seo"}}
	AppendDisabledHooks(config, "analytics", "social")

	if len(config.DisabledHooks) != 3 {
		t.Fatalf("len(DisabledHooks) = %d, want 3", len(config.DisabledHooks))
	}
}

func TestAppendGlobPatterns(t *testing.T) {
	config := &models.Config{
		GlobConfig: models.GlobConfig{Patterns: []string{"**/*.md"}},
	}
	AppendGlobPatterns(config, "posts/**/*.md", "pages/*.md")

	if len(config.GlobConfig.Patterns) != 3 {
		t.Fatalf("len(Patterns) = %d, want 3", len(config.GlobConfig.Patterns))
	}
}

func TestAppendFeeds(t *testing.T) {
	config := &models.Config{
		Feeds: []models.FeedConfig{{Slug: "blog"}},
	}
	AppendFeeds(config, models.FeedConfig{Slug: "projects"}, models.FeedConfig{Slug: "notes"})

	if len(config.Feeds) != 3 {
		t.Fatalf("len(Feeds) = %d, want 3", len(config.Feeds))
	}
	expected := []string{"blog", "projects", "notes"}
	for i, slug := range expected {
		if config.Feeds[i].Slug != slug {
			t.Errorf("Feeds[%d].Slug = %q, want %q", i, config.Feeds[i].Slug, slug)
		}
	}
}

func TestMergeGlobConfig_EmptyPatterns(t *testing.T) {
	base := models.GlobConfig{
		Patterns:     []string{"base/**/*.md"},
		UseGitignore: true,
	}
	override := models.GlobConfig{
		Patterns:     nil, // Empty, should keep base
		UseGitignore: false,
	}

	result := mergeGlobConfig(base, override)

	if len(result.Patterns) != 1 || result.Patterns[0] != "base/**/*.md" {
		t.Errorf("Patterns = %v, want base patterns", result.Patterns)
	}
}

func TestMergeMarkdownConfig(t *testing.T) {
	base := models.MarkdownConfig{Extensions: []string{"tables"}}
	override := models.MarkdownConfig{Extensions: []string{"footnotes", "syntax"}}

	result := mergeMarkdownConfig(base, override)

	if len(result.Extensions) != 2 {
		t.Fatalf("len(Extensions) = %d, want 2", len(result.Extensions))
	}
	if result.Extensions[0] != "footnotes" {
		t.Errorf("Extensions[0] = %q, want %q", result.Extensions[0], "footnotes")
	}
}

func TestMergeFeedFormats(t *testing.T) {
	base := models.FeedFormats{HTML: true, RSS: true}
	override := models.FeedFormats{Atom: true, JSON: true}

	result := mergeFeedFormats(base, override)

	// Override has active formats, so it replaces base entirely
	if result.HTML || result.RSS {
		t.Errorf("FeedFormats = %+v, HTML and RSS should be false", result)
	}
	if !result.Atom || !result.JSON {
		t.Errorf("FeedFormats = %+v, Atom and JSON should be true", result)
	}
}

func TestMergeFeedTemplates(t *testing.T) {
	base := models.FeedTemplates{HTML: "base.html", RSS: "base.xml"}
	override := models.FeedTemplates{HTML: "custom.html", Atom: "atom.xml"}

	result := mergeFeedTemplates(base, override)

	if result.HTML != "custom.html" {
		t.Errorf("HTML = %q, want %q", result.HTML, "custom.html")
	}
	if result.RSS != "base.xml" {
		t.Errorf("RSS = %q, want %q (from base)", result.RSS, "base.xml")
	}
	if result.Atom != "atom.xml" {
		t.Errorf("Atom = %q, want %q", result.Atom, "atom.xml")
	}
}

func TestMergeSyndicationConfig(t *testing.T) {
	base := models.SyndicationConfig{MaxItems: 20, IncludeContent: false}
	override := models.SyndicationConfig{MaxItems: 50, IncludeContent: true}

	result := mergeSyndicationConfig(base, override)

	if result.MaxItems != 50 {
		t.Errorf("MaxItems = %d, want 50", result.MaxItems)
	}
	if !result.IncludeContent {
		t.Error("IncludeContent should be true")
	}
}

func TestMergeConfigs_DoesNotMutateInputs(t *testing.T) {
	base := &models.Config{
		OutputDir: "base",
		Hooks:     []string{"hook1"},
	}
	override := &models.Config{
		OutputDir: "override",
		Hooks:     []string{"hook2"},
	}

	// Save original values
	baseDir := base.OutputDir
	overrideDir := override.OutputDir

	result := MergeConfigs(base, override)

	// Modify result
	result.OutputDir = "modified"
	result.Hooks[0] = "modified-hook"

	// Originals should be unchanged
	if base.OutputDir != baseDir {
		t.Error("base was mutated")
	}
	if override.OutputDir != overrideDir {
		t.Error("override was mutated")
	}
}

func TestMergeConfigs_FullExample(t *testing.T) {
	base := DefaultConfig()
	override := &models.Config{
		OutputDir: "public",
		URL:       "https://example.com",
		Title:     "My Site",
		GlobConfig: models.GlobConfig{
			Patterns: []string{"posts/**/*.md"},
		},
		FeedDefaults: models.FeedDefaults{
			ItemsPerPage: 20,
		},
		Feeds: []models.FeedConfig{
			{Slug: "blog", Title: "Blog"},
		},
	}

	result := MergeConfigs(base, override)

	// Overridden values
	if result.OutputDir != "public" {
		t.Errorf("OutputDir = %q, want %q", result.OutputDir, "public")
	}
	if result.URL != "https://example.com" {
		t.Errorf("URL = %q, want %q", result.URL, "https://example.com")
	}

	// Default values should be preserved
	if result.TemplatesDir != base.TemplatesDir {
		t.Errorf("TemplatesDir = %q, want default %q", result.TemplatesDir, base.TemplatesDir)
	}
	if result.AssetsDir != base.AssetsDir {
		t.Errorf("AssetsDir = %q, want default %q", result.AssetsDir, base.AssetsDir)
	}
}

func TestMergePostFormatsConfig_OverrideMarkdownAndOG(t *testing.T) {
	htmlEnabled := true
	base := models.PostFormatsConfig{
		HTML:     &htmlEnabled,
		Markdown: false,
		OG:       false,
	}
	override := models.PostFormatsConfig{
		HTML:     nil, // Not set, should keep base
		Markdown: true,
		OG:       true,
	}

	result := mergePostFormatsConfig(base, override)

	if result.HTML == nil || !*result.HTML {
		t.Error("HTML should be true (from base)")
	}
	if !result.Markdown {
		t.Error("Markdown should be true (from override)")
	}
	if !result.OG {
		t.Error("OG should be true (from override)")
	}
}

func TestMergePostFormatsConfig_OverrideHTML(t *testing.T) {
	htmlEnabled := true
	htmlDisabled := false
	base := models.PostFormatsConfig{
		HTML:     &htmlEnabled,
		Markdown: false,
		OG:       false,
	}
	override := models.PostFormatsConfig{
		HTML:     &htmlDisabled, // Explicitly set to false
		Markdown: false,
		OG:       false,
	}

	result := mergePostFormatsConfig(base, override)

	if result.HTML == nil || *result.HTML {
		t.Error("HTML should be false (from override)")
	}
}

func TestMergePostFormatsConfig_PreserveBase(t *testing.T) {
	htmlEnabled := true
	base := models.PostFormatsConfig{
		HTML:     &htmlEnabled,
		Markdown: true,
		OG:       true,
	}
	override := models.PostFormatsConfig{
		HTML:     nil,
		Markdown: false, // Won't override since false is zero value
		OG:       false,
	}

	result := mergePostFormatsConfig(base, override)

	if result.HTML == nil || !*result.HTML {
		t.Error("HTML should be true (from base)")
	}
	// Note: Markdown and OG from base are preserved since override is false
	if !result.Markdown {
		t.Error("Markdown should be true (preserved from base)")
	}
	if !result.OG {
		t.Error("OG should be true (preserved from base)")
	}
}

func TestMergeConfigs_PostFormats(t *testing.T) {
	htmlEnabled := true
	base := &models.Config{
		PostFormats: models.PostFormatsConfig{
			HTML:     &htmlEnabled,
			Markdown: false,
			OG:       false,
		},
	}
	override := &models.Config{
		PostFormats: models.PostFormatsConfig{
			HTML:     nil,
			Markdown: true,
			OG:       true,
		},
	}

	result := MergeConfigs(base, override)

	if result.PostFormats.HTML == nil || !*result.PostFormats.HTML {
		t.Error("PostFormats.HTML should be true (from base)")
	}
	if !result.PostFormats.Markdown {
		t.Error("PostFormats.Markdown should be true (from override)")
	}
	if !result.PostFormats.OG {
		t.Error("PostFormats.OG should be true (from override)")
	}
}

func TestMergeBlogrollConfig_Enabled(t *testing.T) {
	base := models.BlogrollConfig{
		Enabled:  false,
		CacheDir: "cache/blogroll",
	}
	override := models.BlogrollConfig{
		Enabled: true,
	}

	result := mergeBlogrollConfig(base, override)

	if !result.Enabled {
		t.Error("Enabled should be true (from override)")
	}
	if result.CacheDir != "cache/blogroll" {
		t.Errorf("CacheDir = %q, want %q (from base)", result.CacheDir, "cache/blogroll")
	}
}

func TestMergeBlogrollConfig_Feeds(t *testing.T) {
	base := models.BlogrollConfig{
		Enabled: true,
		Feeds: []models.ExternalFeedConfig{
			{URL: "https://base.com/feed.xml", Title: "Base Feed"},
		},
	}
	override := models.BlogrollConfig{
		Feeds: []models.ExternalFeedConfig{
			{URL: "https://override.com/feed.xml", Title: "Override Feed"},
		},
	}

	result := mergeBlogrollConfig(base, override)

	if len(result.Feeds) != 1 {
		t.Fatalf("Feeds length = %d, want 1", len(result.Feeds))
	}
	if result.Feeds[0].URL != "https://override.com/feed.xml" {
		t.Errorf("Feeds[0].URL = %q, want override URL", result.Feeds[0].URL)
	}
}

func TestMergeBlogrollConfig_Templates(t *testing.T) {
	base := models.BlogrollConfig{
		Templates: models.BlogrollTemplates{
			Blogroll: "blogroll.html",
			Reader:   "reader.html",
		},
	}
	override := models.BlogrollConfig{
		Templates: models.BlogrollTemplates{
			Blogroll: "custom-blogroll.html",
			// Reader not set, should keep base
		},
	}

	result := mergeBlogrollConfig(base, override)

	if result.Templates.Blogroll != "custom-blogroll.html" {
		t.Errorf("Templates.Blogroll = %q, want custom-blogroll.html", result.Templates.Blogroll)
	}
	if result.Templates.Reader != "reader.html" {
		t.Errorf("Templates.Reader = %q, want reader.html (from base)", result.Templates.Reader)
	}
}

func TestMergeConfigs_Blogroll(t *testing.T) {
	base := &models.Config{
		Blogroll: models.BlogrollConfig{
			Enabled:           false,
			CacheDir:          "cache/blogroll",
			CacheDuration:     "1h",
			MaxEntriesPerFeed: 50,
		},
	}
	override := &models.Config{
		Blogroll: models.BlogrollConfig{
			Enabled: true,
			Feeds: []models.ExternalFeedConfig{
				{URL: "https://example.com/feed.xml", Title: "Example"},
			},
		},
	}

	result := MergeConfigs(base, override)

	if !result.Blogroll.Enabled {
		t.Error("Blogroll.Enabled should be true")
	}
	if result.Blogroll.CacheDir != "cache/blogroll" {
		t.Errorf("Blogroll.CacheDir = %q, want cache/blogroll", result.Blogroll.CacheDir)
	}
	if len(result.Blogroll.Feeds) != 1 {
		t.Fatalf("Blogroll.Feeds length = %d, want 1", len(result.Blogroll.Feeds))
	}
	if result.Blogroll.Feeds[0].Title != "Example" {
		t.Errorf("Blogroll.Feeds[0].Title = %q, want Example", result.Blogroll.Feeds[0].Title)
	}
}
