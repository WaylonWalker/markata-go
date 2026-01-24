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
			want: "docs.html",
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
			want: "bare.html",
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
			want: "docs.html",
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
			want: "docs.html",
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
			want: "landing.html",
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
			want: "docs.html",
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
		{"docs", "docs.html"},
		{"blog", "post.html"},
		{"landing", "landing.html"},
		{"bare", "bare.html"},
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
