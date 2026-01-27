package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

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
					Markdown: "blog.md",
					OG:       "blog-og.html",
				},
				"docs": {
					HTML:     "docs.html",
					Text:     "docs.txt",
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
		{"post.html", "markdown", "post.md"},
		{"post.html", "md", "post.md"},
		{"post.html", "og", "post-og.html"},
		{"blog.html", "txt", "blog.txt"},
		{"layouts/docs.html", "txt", "layouts/docs.txt"},
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
				"/secret/index.txt",
				"/secret/index.md",
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
				"/secret/index.txt",
				"/secret/index.md",
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
				"/real-private/index.txt",
				"/real-private/index.md",
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
				"/private1/index.txt",
				"/private1/index.md",
				"/private1.og/",
				"/private2/",
				"/private2/index.txt",
				"/private2/index.md",
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
