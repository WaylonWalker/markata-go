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
