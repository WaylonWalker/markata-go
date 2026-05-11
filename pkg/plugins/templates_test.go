package plugins

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

func TestTemplatesPlugin_GetFeedSidebarPosts_PrefersPrimaryFeed(t *testing.T) {
	p := NewTemplatesPlugin()
	m := lifecycle.NewManager()

	enabled := true
	m.Config().Extra["components"] = models.ComponentsConfig{
		FeedSidebar: models.FeedSidebarConfig{Enabled: &enabled},
	}

	title := "Post"
	post := &models.Post{Slug: "post", Title: &title, Href: "/post/"}
	secondary := models.FeedConfig{
		Slug:  "secondary",
		Title: "Secondary",
		Posts: []*models.Post{post},
	}
	primary := models.FeedConfig{
		Slug:    "primary",
		Title:   "Primary",
		Primary: true,
		Posts:   []*models.Post{post},
	}
	m.Cache().Set("feed_configs", []models.FeedConfig{secondary, primary})

	posts, feed := p.getFeedSidebarPosts(post, m.Config(), m)
	if feed == nil {
		t.Fatal("expected a sidebar feed")
	}
	if feed.Slug != "primary" {
		t.Fatalf("expected primary feed, got %q", feed.Slug)
	}
	if len(posts) != 1 || posts[0].Slug != post.Slug {
		t.Fatalf("unexpected sidebar posts: %#v", posts)
	}
}

func TestTemplatesPlugin_BuildSidebarFeedsJSON_RotationIncludesOnlyPrimary(t *testing.T) {
	p := NewTemplatesPlugin()
	m := lifecycle.NewManager()

	enabled := true
	config := m.Config()
	config.Extra["components"] = models.ComponentsConfig{
		FeedSidebar: models.FeedSidebarConfig{Enabled: &enabled},
	}

	title := "Post"
	post := &models.Post{Slug: "post", Title: &title, Href: "/post/"}
	primary := models.FeedConfig{Slug: "primary", Title: "Primary", Primary: true, Posts: []*models.Post{post}}
	secondary := models.FeedConfig{Slug: "secondary", Title: "Secondary", Posts: []*models.Post{post}}
	publicOnly := models.FeedConfig{Slug: "public", Title: "Public", Posts: []*models.Post{post}}
	private := models.FeedConfig{Slug: "private", Title: "Private", IncludePrivate: true, Posts: []*models.Post{post}}
	m.Cache().Set("feed_configs", []models.FeedConfig{secondary, primary, publicOnly, private})

	jsonText := p.buildSidebarFeedsJSON(post, config, m, &primary)
	if jsonText == "" {
		t.Fatal("expected sidebar feeds JSON")
	}

	var data sidebarFeedsDataJSON
	if err := json.Unmarshal([]byte(jsonText), &data); err != nil {
		t.Fatalf("unmarshal sidebar feeds JSON: %v", err)
	}

	if len(data.RotationFeedSlugs) != 1 || data.RotationFeedSlugs[0] != "primary" {
		t.Fatalf("unexpected rotation feeds: %#v", data.RotationFeedSlugs)
	}

	for _, feed := range data.Feeds {
		if feed.Slug == "private" {
			t.Fatal("private feed should not be included in picker feeds")
		}
	}
}

func TestTemplatesPlugin_BuildSidebarFeedsJSON_RotationIncludesAllPublicPrimaryFeeds(t *testing.T) {
	p := NewTemplatesPlugin()
	m := lifecycle.NewManager()

	enabled := true
	config := m.Config()
	config.Extra["components"] = models.ComponentsConfig{
		FeedSidebar: models.FeedSidebarConfig{Enabled: &enabled},
	}

	title := "Post"
	post := &models.Post{Slug: "post", Title: &title, Href: "/post/"}
	matchingPrimary := models.FeedConfig{Slug: "primary-a", Title: "Primary A", Primary: true, Posts: []*models.Post{post}}
	otherMatching := models.FeedConfig{Slug: "primary-b", Title: "Primary B", Primary: true, Posts: []*models.Post{post}}
	nonMatchingPrimary := models.FeedConfig{Slug: "primary-c", Title: "Primary C", Primary: true, Posts: []*models.Post{{Slug: "other", Href: "/other/"}}}
	privatePrimary := models.FeedConfig{Slug: "primary-private", Title: "Private", Primary: true, IncludePrivate: true, Posts: []*models.Post{{Slug: "hidden", Href: "/hidden/"}}}
	m.Cache().Set("feed_configs", []models.FeedConfig{matchingPrimary, otherMatching, nonMatchingPrimary, privatePrimary})

	jsonText := p.buildSidebarFeedsJSON(post, config, m, &matchingPrimary)
	if jsonText == "" {
		t.Fatal("expected sidebar feeds JSON")
	}

	var data sidebarFeedsDataJSON
	if err := json.Unmarshal([]byte(jsonText), &data); err != nil {
		t.Fatalf("unmarshal sidebar feeds JSON: %v", err)
	}

	want := []string{"primary-a", "primary-b", "primary-c"}
	if len(data.RotationFeedSlugs) != len(want) {
		t.Fatalf("rotation feeds = %#v, want %#v", data.RotationFeedSlugs, want)
	}
	for i := range want {
		if data.RotationFeedSlugs[i] != want[i] {
			t.Fatalf("rotation feeds = %#v, want %#v", data.RotationFeedSlugs, want)
		}
	}
}

func TestTemplatesPlugin_BuildSidebarFeedsJSON_RotationFollowsConfigOrder(t *testing.T) {
	p := NewTemplatesPlugin()
	m := lifecycle.NewManager()

	enabled := true
	config := m.Config()
	config.Extra["components"] = models.ComponentsConfig{
		FeedSidebar: models.FeedSidebarConfig{Enabled: &enabled},
	}

	title := "Post"
	post := &models.Post{Slug: "post", Title: &title, Href: "/post/"}
	archive := models.FeedConfig{Slug: "archive", Title: "Archive", Primary: true, Posts: []*models.Post{post}}
	blog := models.FeedConfig{Slug: "blog", Title: "Blog", Primary: true, Posts: []*models.Post{post}}
	pings := models.FeedConfig{Slug: "pings", Title: "Pings", Primary: true, Posts: []*models.Post{post}}
	m.Cache().Set("feed_configs", []models.FeedConfig{archive, blog, pings})

	jsonText := p.buildSidebarFeedsJSON(post, config, m, &pings)
	if jsonText == "" {
		t.Fatal("expected sidebar feeds JSON")
	}

	var data sidebarFeedsDataJSON
	if err := json.Unmarshal([]byte(jsonText), &data); err != nil {
		t.Fatalf("unmarshal sidebar feeds JSON: %v", err)
	}

	want := []string{"archive", "blog", "pings"}
	if len(data.RotationFeedSlugs) != len(want) {
		t.Fatalf("rotation feeds = %#v, want %#v", data.RotationFeedSlugs, want)
	}
	for i := range want {
		if data.RotationFeedSlugs[i] != want[i] {
			t.Fatalf("rotation feeds = %#v, want %#v", data.RotationFeedSlugs, want)
		}
	}
}

func TestAppendFeedParamToHref_PreservesSidebarFlow(t *testing.T) {
	tests := []struct {
		name string
		href string
		feed string
		want string
	}{
		{name: "plain path", href: "/nope/", feed: "snippets/home-posts", want: "/nope/?feed=snippets%2Fhome-posts"},
		{name: "existing query", href: "/nope/?foo=bar", feed: "snippets/home-posts", want: "/nope/?feed=snippets%2Fhome-posts&foo=bar"},
		{name: "fragment", href: "/nope/#section", feed: "snippets/home-posts", want: "/nope/?feed=snippets%2Fhome-posts#section"},
		{name: "empty feed", href: "/nope/", feed: "", want: "/nope/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appendFeedParamToHref(tt.href, tt.feed); got != tt.want {
				t.Fatalf("appendFeedParamToHref(%q, %q) = %q, want %q", tt.href, tt.feed, got, tt.want)
			}
		})
	}
}

func TestTemplatesPlugin_BuildSidebarFeedEntry_AppendsFeedParamToLinks(t *testing.T) {
	p := NewTemplatesPlugin()
	title := "Current"
	prevTitle := "Prev"
	current := &models.Post{Slug: "current", Title: &title, Href: "/current/"}
	prev := &models.Post{Slug: "prev", Title: &prevTitle, Href: "/prev/"}
	feed := &models.FeedConfig{
		Slug:  "snippets/home-posts",
		Title: "Home Posts",
		Posts: []*models.Post{prev, current},
	}

	entry := p.buildSidebarFeedEntry(current, feed, feed.Posts, "primary", models.NewFeedDefaults().Syndication, models.NewPostFormatsConfig())
	if len(entry.Posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(entry.Posts))
	}
	for _, post := range entry.Posts {
		if !strings.Contains(post.Href, "feed=snippets%2Fhome-posts") {
			t.Fatalf("expected sidebar href to preserve feed, got %q", post.Href)
		}
	}
	if entry.Prev == nil || !strings.Contains(entry.Prev.Href, "feed=snippets%2Fhome-posts") {
		t.Fatalf("expected prev href to preserve feed, got %#v", entry.Prev)
	}
}

func TestTemplatesPlugin_BuildSidebarFeedEntry_IncludesEnabledVariants(t *testing.T) {
	p := NewTemplatesPlugin()
	title := "Current"
	current := &models.Post{Slug: "current", Title: &title, Href: "/current/"}
	feed := &models.FeedConfig{
		Slug:  "til-feed",
		Title: "Today I Learned",
		Formats: models.FeedFormats{
			HTML:       true,
			SimpleHTML: true,
			RSS:        true,
			Atom:       true,
			JSON:       true,
			Markdown:   true,
		},
		Posts: []*models.Post{current},
	}

	entry := p.buildSidebarFeedEntry(current, feed, feed.Posts, "primary", models.NewFeedDefaults().Syndication, models.NewPostFormatsConfig())
	got := make([]string, 0, len(entry.Variants))
	for _, variant := range entry.Variants {
		got = append(got, variant.Key)
	}

	want := []string{"md", "json", "archive-rss", "rss", "atom", "html", "simple"}
	if len(got) != len(want) {
		t.Fatalf("variant count = %d, want %d (%#v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("variant order = %#v, want %#v", got, want)
		}
	}
}

func TestTemplatesPlugin_BuildSidebarFeedEntry_UsesCanonicalVariantURLs(t *testing.T) {
	p := NewTemplatesPlugin()
	title := "Current"
	current := &models.Post{Slug: "current", Title: &title, Href: "/current/"}
	feed := &models.FeedConfig{
		Slug:  "til-feed",
		Title: "Today I Learned",
		Formats: models.FeedFormats{
			HTML:       true,
			SimpleHTML: true,
			RSS:        true,
			Atom:       true,
			JSON:       true,
			Markdown:   true,
			Text:       true,
		},
		Posts: []*models.Post{current},
	}

	entry := p.buildSidebarFeedEntry(current, feed, feed.Posts, "primary", models.NewFeedDefaults().Syndication, models.NewPostFormatsConfig())
	for _, variant := range entry.Variants {
		switch variant.Key {
		case "md":
			if variant.Href != "/til-feed.md" {
				t.Fatalf("md href = %q, want %q", variant.Href, "/til-feed.md")
			}
		case "txt":
			if variant.Href != "/til-feed.txt" {
				t.Fatalf("txt href = %q, want %q", variant.Href, "/til-feed.txt")
			}
		}
	}
}

func TestTemplatesPlugin_BuildSidebarFeedEntry_UsesCanonicalVariantURLsForRootFeed(t *testing.T) {
	p := NewTemplatesPlugin()
	title := "Current"
	current := &models.Post{Slug: "current", Title: &title, Href: "/current/"}
	feed := &models.FeedConfig{
		Title: "Posts",
		Formats: models.FeedFormats{
			Markdown: true,
			Text:     true,
		},
		Posts: []*models.Post{current},
	}

	entry := p.buildSidebarFeedEntry(current, feed, feed.Posts, "primary", models.NewFeedDefaults().Syndication, models.NewPostFormatsConfig())
	for _, variant := range entry.Variants {
		switch variant.Key {
		case "md":
			if variant.Href != "/index.md" {
				t.Fatalf("md href = %q, want %q", variant.Href, "/index.md")
			}
		case "txt":
			if variant.Href != "/index.txt" {
				t.Fatalf("txt href = %q, want %q", variant.Href, "/index.txt")
			}
		}
	}
}

func TestTemplatesPlugin_BuildSidebarFeedEntry_HidesDisabledPostFormats(t *testing.T) {
	p := NewTemplatesPlugin()
	title := "Current"
	current := &models.Post{Slug: "current", Title: &title, Href: "/current/"}
	feed := &models.FeedConfig{
		Slug:  "til-feed",
		Title: "Today I Learned",
		Formats: models.FeedFormats{
			HTML:       true,
			SimpleHTML: true,
			RSS:        true,
			Atom:       true,
			JSON:       true,
			Markdown:   true,
			Text:       true,
		},
		Posts: []*models.Post{current},
	}
	postFormats := models.NewPostFormatsConfig()
	postFormats.Markdown = false
	postFormats.Text = false

	entry := p.buildSidebarFeedEntry(current, feed, feed.Posts, "primary", models.NewFeedDefaults().Syndication, postFormats)
	for _, variant := range entry.Variants {
		if variant.Key == "md" || variant.Key == "txt" {
			t.Fatalf("unexpected variant %q in %#v", variant.Key, entry.Variants)
		}
	}
}

func TestTemplatesPlugin_Name(t *testing.T) {
	p := NewTemplatesPlugin()
	if got := p.Name(); got != "templates" {
		t.Errorf("Name() = %q, want %q", got, "templates")
	}
}

func TestTemplatesPlugin_Configure(t *testing.T) {
	p := NewTemplatesPlugin()
	m := lifecycle.NewManager()

	// Set templates directory in config
	config := m.Config()
	config.Extra["templates_dir"] = "templates"

	err := p.Configure(m)
	if err != nil {
		t.Errorf("Configure() error = %v", err)
	}

	if p.engine == nil {
		t.Error("Configure() did not initialize engine")
	}

	// Check that engine is stored in cache
	cached, ok := m.Cache().Get("templates.engine")
	if !ok {
		t.Error("Configure() did not cache engine")
	}
	if cached != p.engine {
		t.Error("Configure() cached wrong engine")
	}
}

func TestTemplatesPlugin_Render(t *testing.T) {
	// Create a temporary directory with a template
	tmpDir, err := os.MkdirTemp("", "templates-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create post.html template
	templateContent := `<!DOCTYPE html>
<html>
<head><title>{{ post.title }}</title></head>
<body>{{ body | safe }}</body>
</html>`
	//nolint:gosec // test file
	err = os.WriteFile(filepath.Join(tmpDir, "post.html"), []byte(templateContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write template: %v", err)
	}

	p := NewTemplatesPlugin()
	m := lifecycle.NewManager()

	// Configure with temp directory
	config := m.Config()
	config.Extra["templates_dir"] = tmpDir

	err = p.Configure(m)
	if err != nil {
		t.Fatalf("Configure() error = %v", err)
	}

	// Create a test post
	title := "Test Post"
	post := &models.Post{
		Title:       &title,
		Template:    "post.html",
		ArticleHTML: "<p>Hello World</p>",
	}
	m.AddPost(post)

	// Render
	err = p.Render(m)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Check post.HTML is set
	if post.HTML == "" {
		t.Error("Render() did not set post.HTML")
	}

	// Check that template was applied
	if post.HTML == post.ArticleHTML {
		t.Error("Render() did not wrap content in template")
	}

	// Check that title is in output
	if !contains(post.HTML, "Test Post") {
		t.Error("Render() output does not contain title")
	}

	// Check that body is in output
	if !contains(post.HTML, "<p>Hello World</p>") {
		t.Error("Render() output does not contain body")
	}
}

func TestTemplatesPlugin_Render_NoTemplate(t *testing.T) {
	p := NewTemplatesPlugin()
	m := lifecycle.NewManager()

	// Configure without templates directory (templates won't exist on filesystem)
	// But embedded templates will be used as fallback
	config := m.Config()
	config.Extra["templates_dir"] = "/nonexistent"

	err := p.Configure(m)
	if err != nil {
		t.Fatalf("Configure() error = %v", err)
	}

	// Create a test post
	title := "Test Post"
	post := &models.Post{
		Title:       &title,
		Template:    "post.html",
		ArticleHTML: "<p>Hello World</p>",
	}
	m.AddPost(post)

	// Render
	err = p.Render(m)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// With embedded templates, post should be rendered with full HTML template
	// Check that the content is wrapped in a proper HTML document
	if post.HTML == post.ArticleHTML {
		t.Errorf("Render() with embedded templates: HTML should be wrapped in template, got raw ArticleHTML")
	}

	// Check that the HTML contains expected elements from the embedded template
	if !strings.Contains(post.HTML, "<!DOCTYPE html>") {
		t.Errorf("Render() with embedded templates: HTML should contain DOCTYPE")
	}
	if !strings.Contains(post.HTML, "<p>Hello World</p>") {
		t.Errorf("Render() with embedded templates: HTML should contain ArticleHTML content")
	}
	if !strings.Contains(post.HTML, "Test Post") {
		t.Errorf("Render() with embedded templates: HTML should contain post title")
	}
	if !strings.Contains(post.HTML, "css/main.css") {
		t.Errorf("Render() with embedded templates: HTML should include CSS links")
	}
}

func TestTemplatesPlugin_Render_PostGraphScriptOnlyWhenGraphRenders(t *testing.T) {
	tests := []struct {
		name            string
		inlinks         int
		outlinks        int
		wantGraphScript bool
	}{
		{name: "no graph for sparse post", inlinks: 1, outlinks: 1, wantGraphScript: false},
		{name: "graph for connected post", inlinks: 2, outlinks: 1, wantGraphScript: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewTemplatesPlugin()
			m := lifecycle.NewManager()

			config := m.Config()
			config.Extra["templates_dir"] = "/nonexistent"

			if err := p.Configure(m); err != nil {
				t.Fatalf("Configure() error = %v", err)
			}

			title := "Test Post"
			post := &models.Post{
				Title:       &title,
				Href:        "/test-post/",
				Template:    "post.html",
				ArticleHTML: "<p>Hello World</p>",
			}
			for i := 0; i < tt.inlinks; i++ {
				post.Inlinks = append(post.Inlinks, &models.Link{SourceURL: "https://example.com/source/"})
			}
			for i := 0; i < tt.outlinks; i++ {
				post.Outlinks = append(post.Outlinks, &models.Link{TargetURL: "https://example.com/target/"})
			}
			m.AddPost(post)

			if err := p.Render(m); err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			hasGraphScript := strings.Contains(post.HTML, "post-graph.js")
			if hasGraphScript != tt.wantGraphScript {
				t.Fatalf("post-graph.js present = %v, want %v; HTML=%q", hasGraphScript, tt.wantGraphScript, post.HTML)
			}
		})
	}
}

func TestTemplatesPlugin_Render_UsesResolvedPostFormatsInTemplateContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templates-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	templateContent := `{% if config.post_formats.markdown %}md-on{% else %}md-off{% endif %} {% if config.post_formats.ansi %}ansi-on{% else %}ansi-off{% endif %}`
	templatePath := filepath.Join(tmpDir, "post.html")

	err = os.WriteFile(templatePath, []byte(templateContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write template: %v", err)
	}

	p := NewTemplatesPlugin()
	m := lifecycle.NewManager()
	config := m.Config()
	config.Extra["templates_dir"] = tmpDir
	config.Extra["url"] = "https://example.com"
	htmlEnabled := true
	config.Extra["post_formats"] = models.PostFormatsConfig{
		HTML:     &htmlEnabled,
		Markdown: true,
		Text:     true,
		ANSI:     false,
	}

	err = p.Configure(m)
	if err != nil {
		t.Fatalf("Configure() error = %v", err)
	}
	if !p.engine.TemplateExists("post.html") {
		t.Fatalf("expected post.html template to be available from %s", templatePath)
	}

	title := "Test Post"
	post := &models.Post{
		Title:       &title,
		Template:    "post.html",
		ArticleHTML: "<p>Hello World</p>",
		Extra: map[string]interface{}{
			"post_formats": map[string]interface{}{
				"markdown": false,
				"ansi":     true,
			},
		},
	}
	m.AddPost(post)

	err = p.Render(m)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(post.HTML, "md-off ansi-on") {
		t.Fatalf("expected template context to use resolved per-post post_formats, got %q", post.HTML)
	}
}

func TestTemplatesPlugin_Render_PostCopyShowsOnlyAvailableRoutes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templates-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	componentDir := filepath.Join(tmpDir, "components")
	if err := os.MkdirAll(componentDir, 0o755); err != nil {
		t.Fatalf("Failed to create component dir: %v", err)
	}
	componentSource, err := os.ReadFile(filepath.Join("..", "..", "templates", "components", "post_copy.html"))
	if err != nil {
		t.Fatalf("Failed to read post_copy component: %v", err)
	}
	if err := os.WriteFile(filepath.Join(componentDir, "post_copy.html"), componentSource, 0o600); err != nil {
		t.Fatalf("Failed to write post_copy component: %v", err)
	}

	templateContent := `{% include "components/post_copy.html" %}`
	templatePath := filepath.Join(tmpDir, "post.html")
	if err := os.WriteFile(templatePath, []byte(templateContent), 0o600); err != nil {
		t.Fatalf("Failed to write template: %v", err)
	}

	engine, err := templates.NewEngine(tmpDir)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	config := &lifecycle.Config{Extra: map[string]interface{}{}}
	htmlEnabled := true
	config.Extra["post_formats"] = models.PostFormatsConfig{
		HTML:     &htmlEnabled,
		Markdown: true,
		Text:     false,
		ANSI:     false,
	}

	title := "Test Post"
	post := &models.Post{
		Title:       &title,
		Slug:        "test-post",
		Href:        "/test-post/",
		Template:    "post.html",
		Content:     "hello",
		ArticleHTML: "<p>Hello World</p>",
	}
	payloads := buildPostCopyPayloads(post, config, "https://example.com")
	ctx := templates.NewContext(post, post.ArticleHTML, models.NewConfig())
	ctx.Set("post_copy_payloads", payloads)
	ctx.Set("post_copy_payloads_json", payloads.JSON())

	rendered, err := engine.Render("post.html", ctx)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(rendered, `data-copy-kind="url"`) {
		t.Fatal("expected URL copy route in rendered post copy menu")
	}
	if !strings.Contains(rendered, `data-copy-kind="markdown-url"`) {
		t.Fatal("expected markdown route in rendered post copy menu")
	}
	if strings.Contains(rendered, `data-copy-kind="text-url"`) {
		t.Fatal("did not expect text route in rendered post copy menu")
	}
	if strings.Contains(rendered, `data-copy-kind="ansi-curl"`) {
		t.Fatal("did not expect ansi route in rendered post copy menu")
	}
}

func TestTemplatesPlugin_Render_SkippedPost(t *testing.T) {
	p := NewTemplatesPlugin()
	m := lifecycle.NewManager()

	config := m.Config()
	config.Extra["templates_dir"] = ""

	err := p.Configure(m)
	if err != nil {
		t.Fatalf("Configure() error = %v", err)
	}

	// Create a skipped post
	post := &models.Post{
		Skip:        true,
		ArticleHTML: "<p>Should not change</p>",
	}
	m.AddPost(post)

	// Render
	err = p.Render(m)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// HTML should not be set for skipped posts
	if post.HTML != "" {
		t.Error("Render() set HTML for skipped post")
	}
}

func TestTemplatesPlugin_Priority(t *testing.T) {
	p := NewTemplatesPlugin()

	// Should run late in render stage
	renderPriority := p.Priority(lifecycle.StageRender)
	if renderPriority != lifecycle.PriorityLate {
		t.Errorf("Priority(StageRender) = %d, want %d", renderPriority, lifecycle.PriorityLate)
	}

	// Default priority for other stages
	otherPriority := p.Priority(lifecycle.StageTransform)
	if otherPriority != lifecycle.PriorityDefault {
		t.Errorf("Priority(StageTransform) = %d, want %d", otherPriority, lifecycle.PriorityDefault)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestTemplatesPlugin_ResolveTemplate(t *testing.T) {
	tests := []struct {
		name         string
		layoutConfig *models.LayoutConfig
		post         *models.Post
		want         string
	}{
		{
			name:         "explicit template in frontmatter takes priority",
			layoutConfig: &models.LayoutConfig{Name: "docs"},
			post: &models.Post{
				Template: "custom.html",
				Href:     "/docs/getting-started/",
			},
			want: "custom.html",
		},
		{
			name: "path-based layout selection",
			layoutConfig: &models.LayoutConfig{
				Name:  "blog",
				Paths: map[string]string{"/docs/": "docs"},
			},
			post: &models.Post{
				Href: "/docs/getting-started/",
			},
			want: "layouts/docs.html",
		},
		{
			name: "path-based layout with longest prefix wins",
			layoutConfig: &models.LayoutConfig{
				Name: "blog",
				Paths: map[string]string{
					"/docs/":     "docs",
					"/docs/api/": "bare",
				},
			},
			post: &models.Post{
				Href: "/docs/api/endpoint/",
			},
			want: "layouts/bare.html",
		},
		{
			name: "feed-based layout selection",
			layoutConfig: &models.LayoutConfig{
				Name:  "blog",
				Feeds: map[string]string{"documentation": "docs"},
			},
			post: &models.Post{
				Href:         "/some/path/",
				PrevNextFeed: "documentation",
			},
			want: "layouts/docs.html",
		},
		{
			name: "path takes priority over feed",
			layoutConfig: &models.LayoutConfig{
				Name:  "blog",
				Paths: map[string]string{"/blog/": "blog"},
				Feeds: map[string]string{"posts": "docs"},
			},
			post: &models.Post{
				Href:         "/blog/my-post/",
				PrevNextFeed: "posts",
			},
			want: "post.html", // blog layout -> post.html
		},
		{
			name: "global default layout",
			layoutConfig: &models.LayoutConfig{
				Name:  "docs",
				Paths: map[string]string{"/blog/": "blog"},
			},
			post: &models.Post{
				Href: "/unmatched/path/",
			},
			want: "layouts/docs.html",
		},
		{
			name: "landing layout",
			layoutConfig: &models.LayoutConfig{
				Name:  "blog",
				Paths: map[string]string{"/": "landing"},
			},
			post: &models.Post{
				Href: "/",
			},
			want: "layouts/landing.html",
		},
		{
			name: "feed from Extra field",
			layoutConfig: &models.LayoutConfig{
				Name:  "blog",
				Feeds: map[string]string{"guides": "docs"},
			},
			post: &models.Post{
				Href:  "/guides/intro/",
				Extra: map[string]interface{}{"feed": "guides"},
			},
			want: "layouts/docs.html",
		},
		{
			name:         "nil layout config falls back to post.html",
			layoutConfig: nil,
			post: &models.Post{
				Href: "/any/path/",
			},
			want: "post.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &TemplatesPlugin{
				layoutConfig: tt.layoutConfig,
			}
			got := p.resolveTemplate(tt.post)
			if got != tt.want {
				t.Errorf("resolveTemplate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLayoutConfig_ResolveLayout(t *testing.T) {
	tests := []struct {
		name     string
		config   *models.LayoutConfig
		postPath string
		feedSlug string
		want     string
	}{
		{
			name: "path match",
			config: &models.LayoutConfig{
				Name:  "blog",
				Paths: map[string]string{"/docs/": "docs"},
			},
			postPath: "/docs/intro/",
			feedSlug: "",
			want:     "docs",
		},
		{
			name: "feed match",
			config: &models.LayoutConfig{
				Name:  "blog",
				Feeds: map[string]string{"tutorials": "docs"},
			},
			postPath: "/random/path/",
			feedSlug: "tutorials",
			want:     "docs",
		},
		{
			name: "default fallback",
			config: &models.LayoutConfig{
				Name:  "landing",
				Paths: map[string]string{"/docs/": "docs"},
			},
			postPath: "/about/",
			feedSlug: "",
			want:     "landing",
		},
		{
			name: "empty config returns default",
			config: &models.LayoutConfig{
				Name: "bare",
			},
			postPath: "/any/",
			feedSlug: "",
			want:     "bare",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ResolveLayout(tt.postPath, tt.feedSlug)
			if got != tt.want {
				t.Errorf("ResolveLayout(%q, %q) = %q, want %q", tt.postPath, tt.feedSlug, got, tt.want)
			}
		})
	}
}

func TestLayoutToTemplate(t *testing.T) {
	tests := []struct {
		layout string
		want   string
	}{
		{"docs", "layouts/docs.html"},
		{"blog", "post.html"},
		{"landing", "layouts/landing.html"},
		{"bare", "layouts/bare.html"},
		{"", "post.html"},
		{"custom", "custom.html"},
		{"already.html", "already.html"},
	}

	for _, tt := range tests {
		t.Run(tt.layout, func(t *testing.T) {
			got := models.LayoutToTemplate(tt.layout)
			if got != tt.want {
				t.Errorf("LayoutToTemplate(%q) = %q, want %q", tt.layout, got, tt.want)
			}
		})
	}
}

func TestTemplatesPlugin_ResolveTemplateForFormat(t *testing.T) {
	tests := []struct {
		name   string
		post   *models.Post
		config *lifecycle.Config
		format string
		want   string
	}{
		{
			name: "per-format override takes priority",
			post: &models.Post{
				Template: "blog.html",
				Templates: map[string]string{
					"txt": "raw.txt",
				},
			},
			config: nil,
			format: "txt",
			want:   "raw.txt",
		},
		{
			name: "per-format override for markdown",
			post: &models.Post{
				Template: "blog.html",
				Templates: map[string]string{
					"markdown": "custom.md",
				},
			},
			config: nil,
			format: "markdown",
			want:   "custom.md",
		},
		{
			name: "fallback to adapted template when no per-format override",
			post: &models.Post{
				Template: "blog.html",
			},
			config: nil,
			format: "txt",
			want:   "blog.txt",
		},
		{
			name: "html format uses template directly",
			post: &models.Post{
				Template: "custom.html",
			},
			config: nil,
			format: "html",
			want:   "custom.html",
		},
		{
			name: "html format appends extension for extensionless template name",
			post: &models.Post{
				Template: "home",
			},
			config: nil,
			format: "html",
			want:   "home.html",
		},
		{
			name: "og format adapts template",
			post: &models.Post{
				Template: "post.html",
			},
			config: nil,
			format: "og",
			want:   "post-og.html",
		},
		{
			name:   "no template falls back to hardcoded default for html",
			post:   &models.Post{},
			config: nil,
			format: "html",
			want:   "post.html",
		},
		{
			name:   "no template falls back to hardcoded default for txt",
			post:   &models.Post{},
			config: nil,
			format: "txt",
			want:   "default.txt",
		},
		{
			name:   "no template falls back to hardcoded default for markdown",
			post:   &models.Post{},
			config: nil,
			format: "markdown",
			want:   "raw.txt",
		},
		{
			name:   "no template falls back to hardcoded default for og",
			post:   &models.Post{},
			config: nil,
			format: "og",
			want:   "og-card.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &TemplatesPlugin{
				config: tt.config,
			}
			got := p.resolveTemplateForFormat(tt.post, tt.format)
			if got != tt.want {
				t.Errorf("resolveTemplateForFormat(%v, %q) = %q, want %q", tt.post.Template, tt.format, got, tt.want)
			}
		})
	}
}

func TestTemplatesPlugin_ResolveTemplateForFormat_WithPresets(t *testing.T) {
	// Test with template presets in config
	config := &lifecycle.Config{
		Extra: map[string]interface{}{
			"template_presets": map[string]models.TemplatePreset{
				"blog": {
					HTML:     "blog.html",
					Text:     "blog.txt",
					ANSI:     "blog.ansi",
					Markdown: "blog.md",
					OG:       "blog-og.html",
				},
				"docs": {
					HTML:     "docs.html",
					Text:     "docs.txt",
					ANSI:     "docs.ansi",
					Markdown: "docs.md",
					OG:       "docs-og.html",
				},
			},
		},
	}

	tests := []struct {
		name   string
		post   *models.Post
		format string
		want   string
	}{
		{
			name: "preset resolves html template",
			post: &models.Post{
				Template: "blog",
			},
			format: "html",
			want:   "blog.html",
		},
		{
			name: "preset resolves txt template",
			post: &models.Post{
				Template: "blog",
			},
			format: "txt",
			want:   "blog.txt",
		},
		{
			name: "preset resolves markdown template",
			post: &models.Post{
				Template: "docs",
			},
			format: "markdown",
			want:   "docs.md",
		},
		{
			name: "preset resolves ansi template",
			post: &models.Post{
				Template: "blog",
			},
			format: "ansi",
			want:   "blog.ansi",
		},
		{
			name: "preset resolves og template",
			post: &models.Post{
				Template: "docs",
			},
			format: "og",
			want:   "docs-og.html",
		},
		{
			name: "per-format override takes priority over preset",
			post: &models.Post{
				Template: "blog",
				Templates: map[string]string{
					"txt": "custom.txt",
				},
			},
			format: "txt",
			want:   "custom.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &TemplatesPlugin{
				config: config,
			}
			got := p.resolveTemplateForFormat(tt.post, tt.format)
			if got != tt.want {
				t.Errorf("resolveTemplateForFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAdaptTemplateForFormat(t *testing.T) {
	tests := []struct {
		template string
		format   string
		want     string
	}{
		{"post.html", "html", "post.html"},
		{"post.html", "txt", "post.txt"},
		{"post.html", "text", "post.txt"},
		{"post.html", "ansi", "post.ansi"},
		{"post.html", "markdown", "post.md"},
		{"post.html", "md", "post.md"},
		{"post.html", "og", "post-og.html"},
		{"blog.html", "txt", "blog.txt"},
		{"blog.html", "ansi", "blog.ansi"},
		{"layouts/docs.html", "txt", "layouts/docs.txt"},
		{"layouts/docs.html", "ansi", "layouts/docs.ansi"},
	}

	for _, tt := range tests {
		t.Run(tt.template+"_"+tt.format, func(t *testing.T) {
			got := adaptTemplateForFormat(tt.template, tt.format)
			if got != tt.want {
				t.Errorf("adaptTemplateForFormat(%q, %q) = %q, want %q", tt.template, tt.format, got, tt.want)
			}
		})
	}
}

func TestGetHardcodedDefault(t *testing.T) {
	tests := []struct {
		format string
		want   string
	}{
		{"html", "post.html"},
		{"txt", "default.txt"},
		{"text", "default.txt"},
		{"ansi", "default.ansi"},
		{"markdown", "raw.txt"},
		{"md", "raw.txt"},
		{"og", "og-card.html"},
		{"unknown", "post.html"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			got := getHardcodedDefault(tt.format)
			if got != tt.want {
				t.Errorf("getHardcodedDefault(%q) = %q, want %q", tt.format, got, tt.want)
			}
		})
	}
}

func TestCollectPrivatePaths(t *testing.T) {
	tests := []struct {
		name     string
		posts    []*models.Post
		expected []string
	}{
		{
			name:     "empty posts",
			posts:    []*models.Post{},
			expected: nil,
		},
		{
			name: "no private posts",
			posts: []*models.Post{
				{Slug: "public", Href: "/public/", Private: false},
			},
			expected: nil,
		},
		{
			name: "private post includes all variants",
			posts: []*models.Post{
				{Slug: "secret", Href: "/secret/", Private: true},
			},
			expected: []string{
				"/secret/",
				"/secret.txt",
				"/secret.ansi",
				"/secret.md",
				"/secret.og/",
			},
		},
		{
			name: "excludes robots post",
			posts: []*models.Post{
				{Slug: "robots", Href: "/robots/", Private: true},
				{Slug: "secret", Href: "/secret/", Private: true},
			},
			expected: []string{
				"/secret/",
				"/secret.txt",
				"/secret.ansi",
				"/secret.md",
				"/secret.og/",
			},
		},
		{
			name: "excludes drafts and skipped",
			posts: []*models.Post{
				{Slug: "draft-post", Href: "/draft-post/", Private: true, Draft: true},
				{Slug: "skipped-post", Href: "/skipped-post/", Private: true, Skip: true},
				{Slug: "real-private", Href: "/real-private/", Private: true},
			},
			expected: []string{
				"/real-private/",
				"/real-private.txt",
				"/real-private.ansi",
				"/real-private.md",
				"/real-private.og/",
			},
		},
		{
			name: "multiple private posts",
			posts: []*models.Post{
				{Slug: "private1", Href: "/private1/", Private: true},
				{Slug: "public", Href: "/public/", Private: false},
				{Slug: "private2", Href: "/private2/", Private: true},
			},
			expected: []string{
				"/private1/",
				"/private1.txt",
				"/private1.ansi",
				"/private1.md",
				"/private1.og/",
				"/private2/",
				"/private2.txt",
				"/private2.ansi",
				"/private2.md",
				"/private2.og/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectPrivatePaths(tt.posts)
			if len(got) != len(tt.expected) {
				t.Errorf("collectPrivatePaths() returned %d paths, want %d\ngot: %v\nwant: %v",
					len(got), len(tt.expected), got, tt.expected)
				return
			}
			for i, path := range got {
				if path != tt.expected[i] {
					t.Errorf("collectPrivatePaths()[%d] = %q, want %q", i, path, tt.expected[i])
				}
			}
		})
	}
}
