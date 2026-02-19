package plugins

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestEmbedsPlugin_Name(t *testing.T) {
	p := NewEmbedsPlugin()
	if p.Name() != "embeds" {
		t.Errorf("expected name 'embeds', got '%s'", p.Name())
	}
}

func TestEmbedsPlugin_Priority(t *testing.T) {
	p := NewEmbedsPlugin()

	if p.Priority(lifecycle.StageTransform) != lifecycle.PriorityEarly {
		t.Errorf("expected PriorityEarly for Transform stage")
	}

	if p.Priority(lifecycle.StageRender) != lifecycle.PriorityDefault {
		t.Errorf("expected PriorityDefault for Render stage")
	}
}

func TestEmbedsPlugin_InternalEmbed(t *testing.T) {
	p := NewEmbedsPlugin()

	m := lifecycle.NewManager()

	// Create posts
	targetTitle := "Target Post"
	targetDesc := "This is the target post description"
	targetDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	targetPost := &models.Post{
		Path:        "target.md",
		Slug:        "target-post",
		Href:        "/target-post/",
		Title:       &targetTitle,
		Description: &targetDesc,
		Date:        &targetDate,
	}

	sourcePost := &models.Post{
		Path:    "source.md",
		Slug:    "source-post",
		Href:    "/source-post/",
		Content: "Here is an embed: ![[target-post]]",
	}

	m.SetPosts([]*models.Post{targetPost, sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	var result *models.Post
	for _, post := range posts {
		if post.Slug == "source-post" {
			result = post
			break
		}
	}

	if result == nil {
		t.Fatal("source post not found")
	}

	// Check that the embed was replaced
	if contains := result.Content; !containsString(contains, `class="embed-card"`) {
		t.Errorf("expected embed card class in content, got: %s", contains)
	}

	if !containsString(result.Content, `href="/target-post/"`) {
		t.Errorf("expected href to target post in content")
	}

	if !containsString(result.Content, "Target Post") {
		t.Errorf("expected target post title in content")
	}

	if !containsString(result.Content, "This is the target post description") {
		t.Errorf("expected target post description in content")
	}
}

func TestEmbedsPlugin_InternalEmbed_WithDisplayText(t *testing.T) {
	p := NewEmbedsPlugin()

	m := lifecycle.NewManager()

	targetTitle := "Target Post"
	targetPost := &models.Post{
		Path:  "target.md",
		Slug:  "target-post",
		Href:  "/target-post/",
		Title: &targetTitle,
	}

	sourcePost := &models.Post{
		Path:    "source.md",
		Slug:    "source-post",
		Content: "Here is an embed: ![[target-post|Custom Title]]",
	}

	m.SetPosts([]*models.Post{targetPost, sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	var result *models.Post
	for _, post := range posts {
		if post.Slug == "source-post" {
			result = post
			break
		}
	}

	if result == nil {
		t.Fatal("source post not found")
	}

	if !containsString(result.Content, "Custom Title") {
		t.Errorf("expected custom title in content, got: %s", result.Content)
	}
}

func TestEmbedsPlugin_InternalEmbed_NotFound(t *testing.T) {
	p := NewEmbedsPlugin()

	m := lifecycle.NewManager()

	sourcePost := &models.Post{
		Path:    "source.md",
		Slug:    "source-post",
		Content: "Here is an embed: ![[nonexistent-post]]",
	}

	m.SetPosts([]*models.Post{sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	result := posts[0]

	// Should have warning comment and original syntax
	if !containsString(result.Content, "<!-- embed not found: nonexistent-post -->") {
		t.Errorf("expected not found comment, got: %s", result.Content)
	}

	if !containsString(result.Content, "![[nonexistent-post]]") {
		t.Errorf("expected original syntax preserved")
	}
}

func TestEmbedsPlugin_InternalEmbed_CannotEmbedSelf(t *testing.T) {
	p := NewEmbedsPlugin()

	m := lifecycle.NewManager()

	title := "Self Post"
	selfPost := &models.Post{
		Path:    "self.md",
		Slug:    "self-post",
		Href:    "/self-post/",
		Title:   &title,
		Content: "Trying to embed myself: ![[self-post]]",
	}

	m.SetPosts([]*models.Post{selfPost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	result := posts[0]

	if !containsString(result.Content, "<!-- cannot embed self -->") {
		t.Errorf("expected self-embed warning, got: %s", result.Content)
	}
}

func TestEmbedsPlugin_InternalEmbed_InCodeBlock(t *testing.T) {
	p := NewEmbedsPlugin()

	m := lifecycle.NewManager()

	targetTitle := "Target Post"
	targetPost := &models.Post{
		Path:  "target.md",
		Slug:  "target-post",
		Title: &targetTitle,
	}

	sourcePost := &models.Post{
		Path: "source.md",
		Slug: "source-post",
		Content: `Normal embed: ![[target-post]]

` + "```" + `
Code block embed: ![[target-post]]
` + "```",
	}

	m.SetPosts([]*models.Post{targetPost, sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	var result *models.Post
	for _, post := range posts {
		if post.Slug == "source-post" {
			result = post
			break
		}
	}

	if result == nil {
		t.Fatal("source post not found")
	}

	// First embed should be processed
	if !containsString(result.Content, `class="embed-card"`) {
		t.Errorf("expected embed card in normal text")
	}

	// Code block embed should NOT be processed
	if !containsString(result.Content, "Code block embed: ![[target-post]]") {
		t.Errorf("expected code block embed to be preserved, got: %s", result.Content)
	}
}

func TestEmbedsPlugin_ExternalEmbed(t *testing.T) {
	// Create a test server that returns HTML with OG metadata
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		//nolint:errcheck // test helper
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
	<meta property="og:title" content="OG Test Title">
	<meta property="og:description" content="OG Test Description">
	<meta property="og:image" content="https://example.com/image.jpg">
	<meta property="og:site_name" content="Test Site">
</head>
<body></body>
</html>`))
	}))
	defer server.Close()

	p := NewEmbedsPlugin()
	p.config.OEmbedEnabled = false
	// Use temp cache dir
	tmpDir := t.TempDir()
	p.config.CacheDir = filepath.Join(tmpDir, "cache")

	m := lifecycle.NewManager()

	sourcePost := &models.Post{
		Path:    "source.md",
		Slug:    "source-post",
		Content: "Here is an external embed: ![embed](" + server.URL + ")",
	}

	m.SetPosts([]*models.Post{sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	result := posts[0]

	// Check embed card was created
	if !containsString(result.Content, "embed-card-external") {
		t.Errorf("expected external embed card class, got: %s", result.Content)
	}

	if !containsString(result.Content, "OG Test Title") {
		t.Errorf("expected OG title in content")
	}

	if !containsString(result.Content, "OG Test Description") {
		t.Errorf("expected OG description in content")
	}

	if !containsString(result.Content, `src="https://example.com/image.jpg"`) {
		t.Errorf("expected OG image in content")
	}
}

func TestEmbedsPlugin_ExternalEmbed_ObsidianStyle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		//nolint:errcheck // test helper
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
	<meta property="og:title" content="OG Test Title">
	<meta property="og:description" content="OG Test Description">
	<meta property="og:image" content="https://example.com/image.jpg">
	<meta property="og:site_name" content="Test Site">
</head>
<body></body>
</html>`))
	}))
	defer server.Close()

	p := NewEmbedsPlugin()
	// Use temp cache dir
	tmpDir := t.TempDir()
	p.config.CacheDir = filepath.Join(tmpDir, "cache")

	m := lifecycle.NewManager()

	sourcePost := &models.Post{
		Path:    "source.md",
		Slug:    "source-post",
		Content: "Here is an external embed: ![[" + server.URL + "]]",
	}

	m.SetPosts([]*models.Post{sourcePost})

	if err := p.Transform(m); err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	result := posts[0]

	if !containsString(result.Content, "embed-card-external") {
		t.Errorf("expected external embed card class, got: %s", result.Content)
	}

	if !containsString(result.Content, "OG Test Title") {
		t.Errorf("expected OG title in content")
	}
}

func TestEmbedsPlugin_ExternalEmbed_ObsidianStyle_WithTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		//nolint:errcheck // test helper
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
	<meta property="og:title" content="OG Test Title">
</head>
<body></body>
</html>`))
	}))
	defer server.Close()

	p := NewEmbedsPlugin()
	// Use temp cache dir
	tmpDir := t.TempDir()
	p.config.CacheDir = filepath.Join(tmpDir, "cache")

	m := lifecycle.NewManager()

	sourcePost := &models.Post{
		Path:    "source.md",
		Slug:    "source-post",
		Content: "Here is an external embed: ![[" + server.URL + "|Custom Title]]",
	}

	m.SetPosts([]*models.Post{sourcePost})

	if err := p.Transform(m); err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	result := posts[0]

	if !containsString(result.Content, "Custom Title") {
		t.Errorf("expected custom title in content")
	}
}

func TestEmbedsPlugin_ExternalEmbed_BracketSyntax(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		//nolint:errcheck // test helper
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<meta property="og:title" content="Bracket Title">
	<meta property="og:description" content="Bracket Description">
	<meta property="og:image" content="https://example.com/bracket.jpg">
</head>
<body></body>
</html>`))
	}))
	defer server.Close()

	p := NewEmbedsPlugin()
	tmpDir := t.TempDir()
	p.config.CacheDir = filepath.Join(tmpDir, "cache")

	m := lifecycle.NewManager()
	m.SetPosts([]*models.Post{{
		Path:    "source.md",
		Slug:    "source-post",
		Content: "Here is an external embed: [!embed](" + server.URL + ")",
	}})

	if err := p.Transform(m); err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	result := m.Posts()[0]
	if !containsString(result.Content, "embed-card-external") {
		t.Errorf("expected external embed card class, got: %s", result.Content)
	}
	if !containsString(result.Content, "Bracket Title") {
		t.Errorf("expected title in content")
	}
	if !containsString(result.Content, "Bracket Description") {
		t.Errorf("expected description in content")
	}
}

func TestEmbedsPlugin_ExternalEmbed_OptionsInMarkdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		//nolint:errcheck // test helper
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<meta property="og:title" content="Options Title">
	<meta property="og:description" content="Options Description">
	<meta property="og:image" content="https://example.com/options.jpg">
</head>
<body></body>
</html>`))
	}))
	defer server.Close()

	p := NewEmbedsPlugin()
	p.config.OEmbedEnabled = false
	tmpDir := t.TempDir()
	p.config.CacheDir = filepath.Join(tmpDir, "cache")

	m := lifecycle.NewManager()
	m.SetPosts([]*models.Post{{
		Path:    "source.md",
		Slug:    "source-post",
		Content: "![embed](" + server.URL + "|no_title)",
	}})

	if err := p.Transform(m); err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	result := m.Posts()[0]
	if !containsString(result.Content, "embed-card-external") {
		t.Errorf("expected external embed card class, got: %s", result.Content)
	}
	if containsString(result.Content, "Options Title") {
		t.Errorf("expected title to be suppressed")
	}
	if !containsString(result.Content, `src="https://example.com/options.jpg"`) {
		t.Errorf("expected image to still render with no_title option")
	}
}

func TestEmbedsPlugin_ExternalEmbed_ObsidianStyle_WithClasses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		//nolint:errcheck // test helper
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<meta property="og:title" content="Classy Title">
</head>
<body></body>
</html>`))
	}))
	defer server.Close()

	p := NewEmbedsPlugin()
	tmpDir := t.TempDir()
	p.config.CacheDir = filepath.Join(tmpDir, "cache")

	m := lifecycle.NewManager()
	m.SetPosts([]*models.Post{{
		Path:    "source.md",
		Slug:    "source-post",
		Content: "![[" + server.URL + "|Custom Title|center full_width]]",
	}})

	if err := p.Transform(m); err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	result := m.Posts()[0]
	if !containsString(result.Content, `class="embed-card embed-card-external embed-card-center embed-card-full-width center full_width"`) {
		t.Errorf("expected classes to be applied, got: %s", result.Content)
	}
}

func TestEmbedsPlugin_ExternalEmbed_DefaultModeFromConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		//nolint:errcheck // test helper
		w.Write([]byte(`{"type":"link","version":"1.0","title":"Rich Title","provider_name":"Test Provider","html":"<iframe src=\"https://example.com/embed\"></iframe>"}`))
	}))
	defer server.Close()

	provider := oembedProvider{
		Name:           "test provider",
		Endpoint:       server.URL,
		URLPrefixes:    []string{"https://example.com/"},
		SupportsFormat: false,
	}

	config := models.NewEmbedsConfig()
	config.DefaultEmbedMode = "rich"
	config.CacheDir = t.TempDir()
	config.OEmbedProviders = map[string]models.OEmbedProviderConfig{
		"test provider": {Enabled: true},
	}

	client := server.Client()
	p := NewEmbedsPlugin()
	p.SetConfig(config)
	p.oembed = newOEmbedResolverWithProviders(config, client, []oembedProvider{provider})

	m := lifecycle.NewManager()
	m.SetPosts([]*models.Post{{
		Path:    "source.md",
		Slug:    "source-post",
		Content: "![embed](https://example.com/rich-post)",
	}})

	if err := p.Transform(m); err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	result := m.Posts()[0]
	if !containsString(result.Content, "embed-card-rich") {
		t.Errorf("expected rich embed rendering, got: %s", result.Content)
	}
}

func TestEmbedsPlugin_ExternalEmbed_ProviderModeOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		//nolint:errcheck // test helper
		w.Write([]byte(`{"type":"link","version":"1.0","title":"Performance Title","provider_name":"Test Provider","thumbnail_url":"https://example.com/image.jpg","url":"https://example.com/image.jpg"}`))
	}))
	defer server.Close()

	provider := oembedProvider{
		Name:           "test",
		Endpoint:       server.URL,
		URLPrefixes:    []string{"https://example.com/"},
		SupportsFormat: false,
	}

	config := models.NewEmbedsConfig()
	config.OEmbedEnabled = true
	config.CacheDir = t.TempDir()
	config.OEmbedProviders = map[string]models.OEmbedProviderConfig{
		"test provider": {Enabled: true, Mode: "performance"},
	}

	client := server.Client()
	p := NewEmbedsPlugin()
	p.SetConfig(config)
	p.oembed = newOEmbedResolverWithProviders(config, client, []oembedProvider{provider})

	m := lifecycle.NewManager()
	m.SetPosts([]*models.Post{{
		Path:    "source.md",
		Slug:    "source-post",
		Content: "![embed](https://example.com/perf-post)",
	}})

	if err := p.Transform(m); err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	result := m.Posts()[0]
	if !containsString(result.Content, "embed-card-img") {
		t.Errorf("expected performance/image-only rendering, got: %s", result.Content)
	}
	if containsString(result.Content, "embed-card-content") {
		t.Errorf("expected card content to be omitted in performance mode")
	}
}

func TestEmbedsPlugin_ExternalEmbed_Caching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/html")
		//nolint:errcheck // test helper
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<meta property="og:title" content="Cached Title">
</head>
</html>`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	// First request
	p1 := NewEmbedsPlugin()
	p1.config.OEmbedEnabled = false
	p1.config.CacheDir = filepath.Join(tmpDir, "cache")

	m1 := lifecycle.NewManager()
	m1.SetPosts([]*models.Post{{
		Path:    "source.md",
		Slug:    "source",
		Content: "![embed](" + server.URL + ")",
	}})

	if err := p1.Transform(m1); err != nil {
		t.Fatalf("First transform failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", callCount)
	}

	// Second request should use cache
	p2 := NewEmbedsPlugin()
	p2.config.OEmbedEnabled = false
	p2.config.CacheDir = filepath.Join(tmpDir, "cache")

	m2 := lifecycle.NewManager()
	m2.SetPosts([]*models.Post{{
		Path:    "source2.md",
		Slug:    "source2",
		Content: "![embed](" + server.URL + ")",
	}})

	if err := p2.Transform(m2); err != nil {
		t.Fatalf("Second transform failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected still 1 HTTP call (cached), got %d", callCount)
	}

	// Verify cache file exists
	cacheDir := filepath.Join(tmpDir, "cache")
	files, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("Failed to read cache dir: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 cache file, got %d", len(files))
	}
}

func TestEmbedsPlugin_ExternalEmbed_FetchDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("HTTP request made when fetch_external is disabled")
	}))
	defer server.Close()

	p := NewEmbedsPlugin()
	p.config.FetchExternal = false
	p.config.OEmbedEnabled = false
	p.config.FallbackTitle = "Fallback"

	m := lifecycle.NewManager()
	m.SetPosts([]*models.Post{{
		Path:    "source.md",
		Slug:    "source",
		Content: "![embed](" + server.URL + ")",
	}})

	if err := p.Transform(m); err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	result := posts[0]

	if !containsString(result.Content, "Fallback") {
		t.Errorf("expected fallback title, got: %s", result.Content)
	}
}

func TestEmbedsPlugin_ExternalEmbed_InCodeBlock(t *testing.T) {
	p := NewEmbedsPlugin()
	p.config.FetchExternal = false
	p.config.OEmbedEnabled = false

	m := lifecycle.NewManager()

	sourcePost := &models.Post{
		Path: "source.md",
		Slug: "source-post",
		Content: `Normal embed: ![embed](https://example.com)

` + "```" + `
Code block embed: ![embed](https://example.com/code)
` + "```",
	}

	m.SetPosts([]*models.Post{sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	result := posts[0]

	// Normal embed should be processed
	if !containsString(result.Content, `class="embed-card embed-card-external"`) {
		t.Errorf("expected external embed card in normal text")
	}

	// Code block embed should NOT be processed
	if !containsString(result.Content, "![embed](https://example.com/code)") {
		t.Errorf("expected code block embed syntax to be preserved")
	}
}

func TestEmbedsPlugin_Configure(t *testing.T) {
	p := NewEmbedsPlugin()

	m := lifecycle.NewManager()
	config := m.Config()
	config.Extra = map[string]interface{}{
		"embeds": map[string]interface{}{
			"enabled":             false,
			"internal_card_class": "custom-internal",
			"external_card_class": "custom-external",
			"fetch_external":      false,
			"oembed_enabled":      false,
			"resolution_strategy": "og_first",
			"cache_dir":           "custom-cache",
			"cache_ttl":           3600,
			"timeout":             30,
			"fallback_title":      "Custom Fallback",
			"show_image":          false,
			"providers": map[string]interface{}{
				"youtube": map[string]interface{}{
					"enabled": false,
				},
				"vimeo": true,
			},
		},
	}

	err := p.Configure(m)
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}

	if p.config.Enabled {
		t.Error("expected enabled to be false")
	}
	if p.config.InternalCardClass != "custom-internal" {
		t.Errorf("expected internal_card_class 'custom-internal', got '%s'", p.config.InternalCardClass)
	}
	if p.config.ExternalCardClass != "custom-external" {
		t.Errorf("expected external_card_class 'custom-external', got '%s'", p.config.ExternalCardClass)
	}
	if p.config.FetchExternal {
		t.Error("expected fetch_external to be false")
	}
	if p.config.OEmbedEnabled {
		t.Error("expected oembed_enabled to be false")
	}
	if p.config.ResolutionStrategy != "og_first" {
		t.Errorf("expected resolution_strategy 'og_first', got '%s'", p.config.ResolutionStrategy)
	}
	if p.config.CacheDir != "custom-cache" {
		t.Errorf("expected cache_dir 'custom-cache', got '%s'", p.config.CacheDir)
	}
	if p.config.CacheTTL != 3600 {
		t.Errorf("expected cache_ttl 3600, got %d", p.config.CacheTTL)
	}
	if p.config.Timeout != 30 {
		t.Errorf("expected timeout 30, got %d", p.config.Timeout)
	}
	if p.config.FallbackTitle != "Custom Fallback" {
		t.Errorf("expected fallback_title 'Custom Fallback', got '%s'", p.config.FallbackTitle)
	}
	if p.config.ShowImage {
		t.Error("expected show_image to be false")
	}
	if p.config.OEmbedProviders == nil {
		t.Fatal("expected oembed providers to be configured")
	}
	if p.config.OEmbedProviders["youtube"].Enabled {
		t.Error("expected youtube provider to be disabled")
	}
	if !p.config.OEmbedProviders["vimeo"].Enabled {
		t.Error("expected vimeo provider to be enabled")
	}
}

func TestEmbedsPlugin_Disabled(t *testing.T) {
	p := NewEmbedsPlugin()
	p.config.Enabled = false

	m := lifecycle.NewManager()

	sourcePost := &models.Post{
		Path:    "source.md",
		Slug:    "source",
		Content: "![[target-post]] and ![embed](https://example.com)",
	}

	m.SetPosts([]*models.Post{sourcePost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	result := posts[0]

	// Content should be unchanged
	if result.Content != "![[target-post]] and ![embed](https://example.com)" {
		t.Errorf("expected content to be unchanged when disabled, got: %s", result.Content)
	}
}

func TestEmbedsPlugin_SkippedPosts(t *testing.T) {
	p := NewEmbedsPlugin()

	m := lifecycle.NewManager()

	targetTitle := "Target"
	targetPost := &models.Post{
		Path:  "target.md",
		Slug:  "target-post",
		Title: &targetTitle,
	}

	skippedPost := &models.Post{
		Path:    "skipped.md",
		Slug:    "skipped",
		Skip:    true,
		Content: "![[target-post]]",
	}

	m.SetPosts([]*models.Post{targetPost, skippedPost})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	var result *models.Post
	for _, post := range posts {
		if post.Slug == "skipped" {
			result = post
			break
		}
	}

	// Content should be unchanged for skipped posts
	if result.Content != "![[target-post]]" {
		t.Errorf("expected skipped post content unchanged, got: %s", result.Content)
	}
}

func TestEmbedsPlugin_Interfaces(_ *testing.T) {
	p := NewEmbedsPlugin()

	// Verify interface implementations
	var _ lifecycle.Plugin = p
	var _ lifecycle.ConfigurePlugin = p
	var _ lifecycle.TransformPlugin = p
	var _ lifecycle.PriorityPlugin = p
}

func TestEmbedsPlugin_AttachmentEmbed(t *testing.T) {
	m := lifecycle.NewManager()

	m.SetPosts([]*models.Post{
		{
			Path:    "test.md",
			Slug:    "test",
			Content: "Here is my photo: ![[photo.jpg]]",
		},
	})

	p := NewEmbedsPlugin()
	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	if posts[0].Content != "Here is my photo: ![photo.jpg](/static/photo.jpg)" {
		t.Errorf("expected attachment embed converted, got: %s", posts[0].Content)
	}
}

func TestEmbedsPlugin_AttachmentEmbed_WithAltText(t *testing.T) {
	m := lifecycle.NewManager()

	m.SetPosts([]*models.Post{
		{
			Path:    "test.md",
			Slug:    "test",
			Content: "Here is my photo: ![[photo.jpg|Custom Alt Text]]",
		},
	})

	p := NewEmbedsPlugin()
	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	if posts[0].Content != "Here is my photo: ![Custom Alt Text](/static/photo.jpg)" {
		t.Errorf("expected attachment embed with alt text, got: %s", posts[0].Content)
	}
}

func TestEmbedsPlugin_AttachmentEmbed_NotInternalEmbed(t *testing.T) {
	targetTitle := "Target Post"
	targetPost := &models.Post{
		Path:        "target.md",
		Slug:        "target-post",
		Title:       &targetTitle,
		Description: strPtr("A target post"),
	}

	m := lifecycle.NewManager()
	m.SetPosts([]*models.Post{
		{
			Path:    "test.md",
			Slug:    "test",
			Content: "Link to ![[target-post]]",
		},
		targetPost,
	})

	p := NewEmbedsPlugin()
	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	for _, post := range posts {
		if post.Slug == "test" {
			if !containsString(post.Content, "embed-card") {
				t.Errorf("expected internal embed card, got: %s", post.Content)
			}
		}
	}
}

func TestEmbedsPlugin_AttachmentEmbed_InCodeBlock(t *testing.T) {
	m := lifecycle.NewManager()

	m.SetPosts([]*models.Post{
		{
			Path:    "test.md",
			Slug:    "test",
			Content: "Text\n\n```\n![[photo.jpg]]\n```\n\nMore text",
		},
	})

	p := NewEmbedsPlugin()
	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	if !containsString(posts[0].Content, "![[photo.jpg]]") {
		t.Errorf("expected attachment embed NOT processed in code block, got: %s", posts[0].Content)
	}
}

func TestEmbedsPlugin_AttachmentEmbed_CustomPrefix(t *testing.T) {
	m := lifecycle.NewManager()

	m.SetPosts([]*models.Post{
		{
			Path:    "test.md",
			Slug:    "test",
			Content: "Photo: ![[image.png]]",
		},
	})

	p := NewEmbedsPlugin()
	p.SetConfig(models.EmbedsConfig{
		Enabled:           true,
		AttachmentsPrefix: "/attachments/",
	})

	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	posts := m.Posts()
	if !containsString(posts[0].Content, "/attachments/image.png") {
		t.Errorf("expected custom prefix, got: %s", posts[0].Content)
	}
}

func TestEmbedsPlugin_AttachmentEmbed_OtherExtensions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"png", "![[image.png]]", "![image.png](/static/image.png)"},
		{"gif", "![[animation.gif]]", "![animation.gif](/static/animation.gif)"},
		{"svg", "![[diagram.svg]]", "![diagram.svg](/static/diagram.svg)"},
		{"pdf", "![[document.pdf]]", "![document.pdf](/static/document.pdf)"},
		{"jpeg", "![[photo.jpeg]]", "![photo.jpeg](/static/photo.jpeg)"},
		{"webp", "![[image.webp]]", "![image.webp](/static/image.webp)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := lifecycle.NewManager()
			m.SetPosts([]*models.Post{
				{
					Path:    "test.md",
					Slug:    "test",
					Content: tt.input,
				},
			})

			p := NewEmbedsPlugin()
			err := p.Transform(m)
			if err != nil {
				t.Fatalf("Transform failed: %v", err)
			}

			posts := m.Posts()
			if posts[0].Content != tt.expected {
				t.Errorf("expected %s, got: %s", tt.expected, posts[0].Content)
			}
		})
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
