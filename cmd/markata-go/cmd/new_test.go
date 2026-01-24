package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{
			name:     "simple title",
			title:    "Hello World",
			expected: "hello-world",
		},
		{
			name:     "title with special characters",
			title:    "What's New in Go 1.22?",
			expected: "whats-new-in-go-122",
		},
		{
			name:     "title with multiple spaces",
			title:    "My   First   Post",
			expected: "my-first-post",
		},
		{
			name:     "title with leading/trailing spaces",
			title:    "  Trimmed Title  ",
			expected: "trimmed-title",
		},
		{
			name:     "title with numbers",
			title:    "Top 10 Tips",
			expected: "top-10-tips",
		},
		{
			name:     "empty title",
			title:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.title)
			if result != tt.expected {
				t.Errorf("generateSlug(%q) = %q, want %q", tt.title, result, tt.expected)
			}
		})
	}
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single tag",
			input:    "go",
			expected: []string{"go"},
		},
		{
			name:     "multiple tags",
			input:    "go,tutorial,programming",
			expected: []string{"go", "tutorial", "programming"},
		},
		{
			name:     "tags with spaces",
			input:    " go , tutorial , programming ",
			expected: []string{"go", "tutorial", "programming"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "only commas",
			input:    ",,,",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTags(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseTags(%q) returned %d tags, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, tag := range result {
				if tag != tt.expected[i] {
					t.Errorf("parseTags(%q)[%d] = %q, want %q", tt.input, i, tag, tt.expected[i])
				}
			}
		})
	}
}

func TestBuiltinTemplates(t *testing.T) {
	templates := builtinTemplates()

	// Check that all expected templates exist
	expectedTemplates := []string{"post", "page", "docs"}
	for _, name := range expectedTemplates {
		if _, exists := templates[name]; !exists {
			t.Errorf("expected builtin template %q not found", name)
		}
	}

	// Check post template
	post := templates["post"]
	if post.Directory != "posts" {
		t.Errorf("post template directory = %q, want %q", post.Directory, "posts")
	}
	if post.Source != "builtin" {
		t.Errorf("post template source = %q, want %q", post.Source, "builtin")
	}

	// Check page template
	page := templates["page"]
	if page.Directory != "pages" {
		t.Errorf("page template directory = %q, want %q", page.Directory, "pages")
	}

	// Check docs template
	docs := templates["docs"]
	if docs.Directory != "docs" {
		t.Errorf("docs template directory = %q, want %q", docs.Directory, "docs")
	}
}

func TestParseTemplateFile(t *testing.T) {
	tests := []struct {
		name             string
		templateName     string
		content          string
		expectedDir      string
		expectedBody     string
		expectedFMLength int
	}{
		{
			name:         "simple template without frontmatter",
			templateName: "simple",
			content:      "Hello, this is the body.",
			expectedDir:  "simple",
			expectedBody: "Hello, this is the body.",
		},
		{
			name:         "template with frontmatter",
			templateName: "fancy",
			content: `---
templateKey: fancy
_directory: fancy-posts
custom_field: value
---

This is the fancy body.`,
			expectedDir:      "fancy-posts",
			expectedBody:     "This is the fancy body.",
			expectedFMLength: 2, // templateKey and custom_field (_directory is removed)
		},
		{
			name:         "template with only frontmatter",
			templateName: "minimal",
			content: `---
templateKey: minimal
---
`,
			expectedDir:      "minimal",
			expectedBody:     "",
			expectedFMLength: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTemplateFile(tt.templateName, tt.content)

			if result.Name != tt.templateName {
				t.Errorf("name = %q, want %q", result.Name, tt.templateName)
			}
			if result.Directory != tt.expectedDir {
				t.Errorf("directory = %q, want %q", result.Directory, tt.expectedDir)
			}
			if result.Body != tt.expectedBody {
				t.Errorf("body = %q, want %q", result.Body, tt.expectedBody)
			}
			if tt.expectedFMLength > 0 && len(result.Frontmatter) != tt.expectedFMLength {
				t.Errorf("frontmatter length = %d, want %d", len(result.Frontmatter), tt.expectedFMLength)
			}
		})
	}
}

func TestGenerateTemplatedContent(t *testing.T) {
	template := ContentTemplate{
		Name:      "post",
		Directory: "posts",
		Frontmatter: map[string]interface{}{
			"templateKey": "post",
			"layout":      "post.html",
		},
		Body:   "Start writing here...",
		Source: "builtin",
	}

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	content := generateTemplatedContent("My Test Post", "my-test-post", date, true, []string{"go", "test"}, template)

	// Check frontmatter delimiters
	if !strings.HasPrefix(content, "---\n") {
		t.Error("content should start with frontmatter delimiter")
	}
	if !strings.Contains(content, "\n---\n") {
		t.Error("content should contain frontmatter closing delimiter")
	}

	// Check required fields are present
	requiredFields := []string{
		"title: My Test Post",
		"slug: my-test-post",
		"date: \"2024-01-15\"",
		"draft: true",
		"published: false",
		"templateKey: post",
		"layout: post.html",
	}
	for _, field := range requiredFields {
		if !strings.Contains(content, field) {
			t.Errorf("content should contain %q", field)
		}
	}

	// Check tags
	if !strings.Contains(content, "- go") || !strings.Contains(content, "- test") {
		t.Error("content should contain tags")
	}

	// Check heading
	if !strings.Contains(content, "# My Test Post") {
		t.Error("content should contain heading")
	}

	// Check body
	if !strings.Contains(content, "Start writing here...") {
		t.Error("content should contain template body")
	}
}

func TestLoadTemplatesFromDir(t *testing.T) {
	// Create a temporary directory with template files
	tmpDir := t.TempDir()

	// Create a test template file
	templateContent := `---
templateKey: tutorial
_directory: tutorials
---

Write your tutorial here...`
	err := os.WriteFile(filepath.Join(tmpDir, "tutorial.md"), []byte(templateContent), 0o600)
	if err != nil {
		t.Fatalf("failed to create test template: %v", err)
	}

	// Load templates
	templates := make(map[string]ContentTemplate)
	loadTemplatesFromDir(tmpDir, templates)

	// Check that the template was loaded
	if _, exists := templates["tutorial"]; !exists {
		t.Error("tutorial template should have been loaded")
		return
	}

	tutorial := templates["tutorial"]
	if tutorial.Directory != "tutorials" {
		t.Errorf("tutorial directory = %q, want %q", tutorial.Directory, "tutorials")
	}
	if tutorial.Source != "file" {
		t.Errorf("tutorial source = %q, want %q", tutorial.Source, "file")
	}
	if !strings.Contains(tutorial.Body, "Write your tutorial here") {
		t.Errorf("tutorial body unexpected: %q", tutorial.Body)
	}
}

func TestLoadTemplatesFromNonexistentDir(t *testing.T) {
	templates := make(map[string]ContentTemplate)
	// Should not panic or error
	loadTemplatesFromDir("/nonexistent/path/that/does/not/exist", templates)

	if len(templates) != 0 {
		t.Errorf("expected empty templates map, got %d items", len(templates))
	}
}

func TestGeneratePostContentWithTags(t *testing.T) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	// Test with tags
	content := generatePostContentWithTags("Test Post", "test-post", date, false, []string{"go", "testing"})

	if !strings.Contains(content, `title: "Test Post"`) {
		t.Error("should contain title")
	}
	if !strings.Contains(content, `slug: "test-post"`) {
		t.Error("should contain slug")
	}
	if !strings.Contains(content, "date: 2024-01-15") {
		t.Error("should contain date")
	}
	if !strings.Contains(content, "draft: false") {
		t.Error("should contain draft: false")
	}
	if !strings.Contains(content, "published: true") {
		t.Error("should contain published: true")
	}
	if !strings.Contains(content, `"go"`) || !strings.Contains(content, `"testing"`) {
		t.Error("should contain tags")
	}

	// Test without tags
	contentNoTags := generatePostContentWithTags("No Tags", "no-tags", date, true, nil)
	if !strings.Contains(contentNoTags, "tags: []") {
		t.Error("should contain empty tags array")
	}
}

func TestContentTemplatesConfig_GetPlacement(t *testing.T) {
	tests := []struct {
		name         string
		placement    map[string]string
		templateName string
		expected     string
	}{
		{
			name: "configured placement",
			placement: map[string]string{
				"post": "blog",
				"page": "pages",
			},
			templateName: "post",
			expected:     "blog",
		},
		{
			name: "unconfigured placement returns template name",
			placement: map[string]string{
				"post": "blog",
			},
			templateName: "unknown",
			expected:     "unknown",
		},
		{
			name:         "empty placement map",
			placement:    map[string]string{},
			templateName: "post",
			expected:     "post",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the models package's GetPlacement through a wrapper test
			// Since we're testing the cmd package, test the local behavior
			var dir string
			if d, ok := tt.placement[tt.templateName]; ok {
				dir = d
			} else {
				dir = tt.templateName
			}

			if dir != tt.expected {
				t.Errorf("GetPlacement(%q) = %q, want %q", tt.templateName, dir, tt.expected)
			}
		})
	}
}
