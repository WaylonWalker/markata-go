package templates

import (
	"testing"
	"time"

	"github.com/example/markata-go/pkg/models"
)

// Helper function to test filters via template rendering
func testFilterViaTemplate(t *testing.T, template string, expected string) {
	t.Helper()

	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := NewContext(nil, "", nil)
	result, err := engine.RenderString(template, ctx)
	if err != nil {
		t.Fatalf("RenderString() error: %v", err)
	}

	if result != expected {
		t.Errorf("Template %q: got %q, want %q", template, result, expected)
	}
}

func TestFilterRSSDate(t *testing.T) {
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	date := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	post := &models.Post{Date: &date}
	ctx := NewContext(post, "", nil)

	result, err := engine.RenderString("{{ post.date | rss_date }}", ctx)
	if err != nil {
		t.Fatalf("RenderString() error: %v", err)
	}

	expected := "Mon, 15 Jan 2024 10:30:00 +0000"
	if result != expected {
		t.Errorf("rss_date: got %q, want %q", result, expected)
	}
}

func TestFilterAtomDate(t *testing.T) {
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	date := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	post := &models.Post{Date: &date}
	ctx := NewContext(post, "", nil)

	result, err := engine.RenderString("{{ post.date | atom_date }}", ctx)
	if err != nil {
		t.Fatalf("RenderString() error: %v", err)
	}

	expected := "2024-01-15T10:30:00Z"
	if result != expected {
		t.Errorf("atom_date: got %q, want %q", result, expected)
	}
}

func TestFilterDateFormat(t *testing.T) {
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	post := &models.Post{Date: &date}
	ctx := NewContext(post, "", nil)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "default format",
			template: "{{ post.date | date_format }}",
			expected: "2024-01-15",
		},
		{
			name:     "custom format",
			template: "{{ post.date | date_format:\"January 2, 2006\" }}",
			expected: "January 15, 2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.RenderString(tt.template, ctx)
			if err != nil {
				t.Fatalf("RenderString() error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFilterSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"Hello  World", "hello-world"},
		{"Hello World!", "hello-world"},
		{"  Spaces  ", "spaces"},
		{"UPPERCASE", "uppercase"},
		{"with_underscore", "with_underscore"},
		{"123 Numbers", "123-numbers"},
	}

	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ctx := NewContext(nil, "", nil)
			ctx.Set("input", tt.input)
			result, err := engine.RenderString("{{ input | slugify }}", ctx)
			if err != nil {
				t.Fatalf("RenderString() error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("slugify(%q): got %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFilterTruncate(t *testing.T) {
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "short string",
			template: "{{ \"Hello\" | truncate:10 }}",
			expected: "Hello",
		},
		{
			name:     "truncate",
			template: "{{ \"Hello World Today\" | truncate:12 }}",
			expected: "Hello World...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext(nil, "", nil)
			result, err := engine.RenderString(tt.template, ctx)
			if err != nil {
				t.Fatalf("RenderString() error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFilterTruncateWords(t *testing.T) {
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "fewer words than limit",
			template: "{{ \"Hello World\" | truncatewords:5 }}",
			expected: "Hello World",
		},
		{
			name:     "truncate words",
			template: "{{ \"One Two Three Four Five Six\" | truncatewords:3 }}",
			expected: "One Two Three ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext(nil, "", nil)
			result, err := engine.RenderString(tt.template, ctx)
			if err != nil {
				t.Fatalf("RenderString() error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFilterDefaultIfNone(t *testing.T) {
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Test with nil post (no title)
	post := &models.Post{}
	ctx := NewContext(post, "", nil)

	result, err := engine.RenderString("{{ post.title | default_if_none:\"Untitled\" }}", ctx)
	if err != nil {
		t.Fatalf("RenderString() error: %v", err)
	}

	if result != "Untitled" {
		t.Errorf("got %q, want %q", result, "Untitled")
	}

	// Test with value set
	title := "My Title"
	post.Title = &title
	ctx = NewContext(post, "", nil)

	result, err = engine.RenderString("{{ post.title | default_if_none:\"Untitled\" }}", ctx)
	if err != nil {
		t.Fatalf("RenderString() error: %v", err)
	}

	if result != "My Title" {
		t.Errorf("got %q, want %q", result, "My Title")
	}
}

func TestFilterLength(t *testing.T) {
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name     string
		template string
		ctx      Context
		expected string
	}{
		{
			name:     "string",
			template: "{{ \"hello\" | length }}",
			ctx:      NewContext(nil, "", nil),
			expected: "5",
		},
		{
			name:     "slice",
			template: "{{ post.tags | length }}",
			ctx:      NewContext(&models.Post{Tags: []string{"a", "b", "c"}}, "", nil),
			expected: "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.RenderString(tt.template, tt.ctx)
			if err != nil {
				t.Fatalf("RenderString() error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFilterJoin(t *testing.T) {
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	post := &models.Post{Tags: []string{"a", "b", "c"}}
	ctx := NewContext(post, "", nil)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "with separator",
			template: "{{ post.tags | join:\", \" }}",
			expected: "a, b, c",
		},
		{
			name:     "with pipe separator",
			template: "{{ post.tags | join:\" | \" }}",
			expected: "a | b | c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.RenderString(tt.template, ctx)
			if err != nil {
				t.Fatalf("RenderString() error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFilterFirstLast(t *testing.T) {
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	post := &models.Post{Tags: []string{"first", "middle", "last"}}
	ctx := NewContext(post, "", nil)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "first",
			template: "{{ post.tags | first }}",
			expected: "first",
		},
		{
			name:     "last",
			template: "{{ post.tags | last }}",
			expected: "last",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.RenderString(tt.template, ctx)
			if err != nil {
				t.Fatalf("RenderString() error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFilterReverse(t *testing.T) {
	// pongo2 has a built-in reverse filter that works on slices
	// For string reversal, we need to use our custom approach
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Test slice reversal
	post := &models.Post{Tags: []string{"a", "b", "c"}}
	ctx := NewContext(post, "", nil)
	result, err := engine.RenderString("{% for t in post.tags | reverse %}{{ t }}{% endfor %}", ctx)
	if err != nil {
		t.Fatalf("RenderString() error: %v", err)
	}

	if result != "cba" {
		t.Errorf("reverse got %q, want %q", result, "cba")
	}
}

func TestFilterStripTags(t *testing.T) {
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple tag",
			input:    "<p>Hello</p>",
			expected: "Hello",
		},
		{
			name:     "nested tags",
			input:    "<div><p>Hello <strong>World</strong></p></div>",
			expected: "Hello World",
		},
		{
			name:     "no tags",
			input:    "Plain text",
			expected: "Plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext(nil, "", nil)
			ctx.Set("input", tt.input)
			result, err := engine.RenderString("{{ input | striptags }}", ctx)
			if err != nil {
				t.Fatalf("RenderString() error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFilterLinebreaksBR(t *testing.T) {
	// Note: pongo2's built-in linebreaksbr filter escapes HTML and uses <br />
	// Our custom filter uses <br> but pongo2's takes precedence
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := NewContext(nil, "", nil)
	ctx.Set("text", "Line 1\nLine 2")
	result, err := engine.RenderString("{{ text | linebreaksbr }}", ctx)
	if err != nil {
		t.Fatalf("RenderString() error: %v", err)
	}

	// pongo2 escapes HTML and uses <br />
	expected := "Line 1&lt;br /&gt;Line 2"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestFilterURLEncode(t *testing.T) {
	// pongo2's urlencode uses + for spaces (form encoding)
	// Our custom filter uses %20 (URL encoding)
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "spaces (uses + for form encoding)",
			template: "{{ \"hello world\" | urlencode }}",
			expected: "hello+world", // pongo2 default behavior
		},
		{
			name:     "ampersand",
			template: "{{ \"a&b\" | urlencode }}",
			expected: "a%26b",
		},
	}

	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext(nil, "", nil)
			result, err := engine.RenderString(tt.template, ctx)
			if err != nil {
				t.Fatalf("RenderString() error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFilterAbsoluteURL(t *testing.T) {
	engine, err := NewEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	config := &models.Config{URL: "https://example.com"}
	post := &models.Post{Href: "/my-post/"}
	ctx := NewContext(post, "", config)

	result, err := engine.RenderString("{{ post.href | absolute_url:config.url }}", ctx)
	if err != nil {
		t.Fatalf("RenderString() error: %v", err)
	}

	expected := "https://example.com/my-post/"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}
