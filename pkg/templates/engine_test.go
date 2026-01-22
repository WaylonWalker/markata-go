package templates

import (
	"testing"
	"time"

	"github.com/example/markata-go/pkg/models"
)

func TestNewEngine(t *testing.T) {
	tests := []struct {
		name         string
		templatesDir string
		wantErr      bool
	}{
		{
			name:         "empty directory",
			templatesDir: "",
			wantErr:      false,
		},
		{
			name:         "non-existent directory",
			templatesDir: "/nonexistent/path",
			wantErr:      false, // Should not error, just won't find templates
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngine(tt.templatesDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEngine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && engine == nil {
				t.Error("NewEngine() returned nil engine")
			}
		})
	}
}

func TestEngine_RenderString(t *testing.T) {
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	title := "Hello World"
	post := &models.Post{
		Title:     &title,
		Slug:      "hello-world",
		Tags:      []string{"go", "programming"},
		Published: true,
	}

	config := &models.Config{
		Title:       "My Site",
		URL:         "https://example.com",
		Description: "A test site",
	}

	tests := []struct {
		name     string
		template string
		post     *models.Post
		config   *models.Config
		want     string
		wantErr  bool
	}{
		{
			name:     "simple variable",
			template: "{{ post.title }}",
			post:     post,
			want:     "Hello World",
		},
		{
			name:     "title shortcut",
			template: "{{ title }}",
			post:     post,
			want:     "Hello World",
		},
		{
			name:     "nested config variable",
			template: "{{ config.title }}",
			post:     post,
			config:   config,
			want:     "My Site",
		},
		{
			name:     "site_title shortcut",
			template: "{{ site_title }}",
			post:     post,
			config:   config,
			want:     "My Site",
		},
		{
			name:     "for loop with tags",
			template: "{% for tag in post.tags %}{{ tag }},{% endfor %}",
			post:     post,
			want:     "go,programming,",
		},
		{
			name:     "if condition true",
			template: "{% if post.published %}yes{% endif %}",
			post:     post,
			want:     "yes",
		},
		{
			name:     "if condition false",
			template: "{% if post.draft %}yes{% else %}no{% endif %}",
			post:     post,
			want:     "no",
		},
		{
			name:     "filter upper",
			template: "{{ post.slug | upper }}",
			post:     post,
			want:     "HELLO-WORLD",
		},
		{
			name:     "filter lower",
			template: "{{ post.title | lower }}",
			post:     post,
			want:     "hello world",
		},
		{
			name:     "filter default with value",
			template: "{{ post.title | default_if_none:\"Untitled\" }}",
			post:     post,
			want:     "Hello World",
		},
		{
			name:     "filter default with nil",
			template: "{{ post.description | default_if_none:\"No description\" }}",
			post:     post,
			want:     "No description",
		},
		{
			name:     "filter truncate",
			template: "{{ \"This is a long string that should be truncated\" | truncate:20 }}",
			post:     post,
			want:     "This is a long...",
		},
		{
			name:     "filter slugify",
			template: "{{ \"Hello World!\" | slugify }}",
			post:     post,
			want:     "hello-world",
		},
		{
			name:     "filter length string",
			template: "{{ \"hello\" | length }}",
			post:     post,
			want:     "5",
		},
		{
			name:     "filter length slice",
			template: "{{ post.tags | length }}",
			post:     post,
			want:     "2",
		},
		{
			name:     "filter join",
			template: "{{ post.tags | join:\", \" }}",
			post:     post,
			want:     "go, programming",
		},
		{
			name:     "invalid template syntax",
			template: "{{ post.title",
			post:     post,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext(tt.post, "", tt.config)
			got, err := engine.RenderString(tt.template, ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("RenderString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEngine_RenderString_DateFilters(t *testing.T) {
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	date := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	post := &models.Post{
		Date: &date,
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "rss_date filter",
			template: "{{ post.date | rss_date }}",
			want:     "Mon, 15 Jan 2024 10:30:00 +0000",
		},
		{
			name:     "atom_date filter",
			template: "{{ post.date | atom_date }}",
			want:     "2024-01-15T10:30:00Z",
		},
		{
			name:     "custom date format",
			template: "{{ post.date | date_format:\"2006-01-02\" }}",
			want:     "2024-01-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext(post, "", nil)
			got, err := engine.RenderString(tt.template, ctx)
			if err != nil {
				t.Errorf("RenderString() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("RenderString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContext_ToPongo2(t *testing.T) {
	title := "Test Post"
	post := &models.Post{
		Title:     &title,
		Slug:      "test-post",
		Tags:      []string{"test"},
		Published: true,
		Extra:     map[string]interface{}{"custom_field": "custom_value"},
	}

	config := &models.Config{
		Title: "Test Site",
		URL:   "https://test.com",
	}

	ctx := NewContext(post, "<p>Body HTML</p>", config)
	ctx.Set("extra_key", "extra_value")

	p2ctx := ctx.ToPongo2()

	// Check post is converted to map
	postMap, ok := p2ctx["post"].(map[string]interface{})
	if !ok {
		t.Error("post not set in context as map")
	}
	if p2ctx["body"] != "<p>Body HTML</p>" {
		t.Error("body not set correctly")
	}

	// Check shortcuts (now strings, not pointers)
	if p2ctx["title"] != "Test Post" {
		t.Errorf("title shortcut not set correctly, got %v", p2ctx["title"])
	}
	if p2ctx["slug"] != "test-post" {
		t.Error("slug shortcut not set correctly")
	}

	// Also check via post map
	if postMap["title"] != "Test Post" {
		t.Error("post.title not set correctly in map")
	}

	// Check config shortcuts
	if p2ctx["site_title"] != "Test Site" {
		t.Error("site_title shortcut not set correctly")
	}
	if p2ctx["site_url"] != "https://test.com" {
		t.Error("site_url shortcut not set correctly")
	}

	// Check extra fields from post
	if p2ctx["custom_field"] != "custom_value" {
		t.Error("custom field from post.Extra not set")
	}

	// Check extra context values
	if p2ctx["extra_key"] != "extra_value" {
		t.Error("extra context value not set")
	}
}

func TestContext_Clone(t *testing.T) {
	title := "Original"
	post := &models.Post{Title: &title}
	config := &models.Config{Title: "Site"}

	original := NewContext(post, "body", config)
	original.Set("key", "value")

	clone := original.Clone()

	// Modify original
	original.Set("key", "modified")
	original.Set("new_key", "new_value")

	// Clone should not be affected
	if clone.Get("key") != "value" {
		t.Error("Clone was affected by modification to original")
	}
	if clone.Get("new_key") != nil {
		t.Error("Clone has key that was added after cloning")
	}
}

func TestContext_Merge(t *testing.T) {
	title1 := "Post 1"
	title2 := "Post 2"
	post1 := &models.Post{Title: &title1}
	post2 := &models.Post{Title: &title2}
	config := &models.Config{Title: "Site"}

	ctx1 := NewContext(post1, "body1", nil)
	ctx1.Set("key1", "value1")

	ctx2 := NewContext(post2, "body2", config)
	ctx2.Set("key2", "value2")

	ctx1.Merge(ctx2)

	// Post should be overwritten
	if ctx1.Post != post2 {
		t.Error("Post not merged")
	}

	// Body should be overwritten
	if ctx1.Body != "body2" {
		t.Error("Body not merged")
	}

	// Config should be set
	if ctx1.Config != config {
		t.Error("Config not merged")
	}

	// Both extra keys should exist
	if ctx1.Get("key1") != "value1" {
		t.Error("Original extra key lost")
	}
	if ctx1.Get("key2") != "value2" {
		t.Error("Merged extra key not added")
	}
}
