package plugins

import (
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestAutoTitlePlugin_Name(t *testing.T) {
	plugin := NewAutoTitlePlugin()
	if plugin.Name() != "auto_title" {
		t.Errorf("expected name 'auto_title', got %q", plugin.Name())
	}
}

func TestAutoTitlePlugin_Priority(t *testing.T) {
	plugin := NewAutoTitlePlugin()

	// Should return PriorityFirst for Transform stage
	if got := plugin.Priority(lifecycle.StageTransform); got != lifecycle.PriorityFirst {
		t.Errorf("expected PriorityFirst for Transform, got %d", got)
	}

	// Should return PriorityDefault for other stages
	if got := plugin.Priority(lifecycle.StageRender); got != lifecycle.PriorityDefault {
		t.Errorf("expected PriorityDefault for Render, got %d", got)
	}
}

func TestAutoTitlePlugin_generateTitle(t *testing.T) {
	plugin := NewAutoTitlePlugin()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "hyphenated filename",
			path:     "posts/my-first-post.md",
			expected: "My First Post",
		},
		{
			name:     "underscored filename",
			path:     "posts/python_tips.md",
			expected: "Python Tips",
		},
		{
			name:     "mixed separators",
			path:     "posts/python_tips-and-tricks.md",
			expected: "Python Tips And Tricks",
		},
		{
			name:     "date prefixed",
			path:     "posts/2024-01-15-new-feature.md",
			expected: "2024 01 15 New Feature",
		},
		{
			name:     "single word",
			path:     "readme.md",
			expected: "Readme",
		},
		{
			name:     "uppercase characters",
			path:     "posts/README.md",
			expected: "Readme",
		},
		{
			name:     "nested path",
			path:     "blog/tutorials/getting-started.md",
			expected: "Getting Started",
		},
		{
			name:     "special characters preserved",
			path:     "posts/c++_tutorial.md",
			expected: "C++ Tutorial",
		},
		{
			name:     "numbers in filename",
			path:     "posts/api-v2-reference.md",
			expected: "Api V2 Reference",
		},
		{
			name:     "empty stem after extension removal",
			path:     ".md",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := plugin.generateTitle(tt.path)
			if got != tt.expected {
				t.Errorf("generateTitle(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestAutoTitlePlugin_Transform(t *testing.T) {
	plugin := NewAutoTitlePlugin()

	tests := []struct {
		name          string
		posts         []*models.Post
		expectedTitle map[string]string // path -> expected title
	}{
		{
			name: "generates title for post without title",
			posts: []*models.Post{
				{Path: "posts/my-first-post.md", Content: "Hello"},
			},
			expectedTitle: map[string]string{
				"posts/my-first-post.md": "My First Post",
			},
		},
		{
			name: "skips post with existing title",
			posts: func() []*models.Post {
				title := "My Custom Title"
				return []*models.Post{
					{Path: "posts/my-post.md", Title: &title, Content: "Hello"},
				}
			}(),
			expectedTitle: map[string]string{
				"posts/my-post.md": "My Custom Title",
			},
		},
		{
			name: "skips post marked as skip",
			posts: []*models.Post{
				{Path: "posts/skipped-post.md", Skip: true, Content: "Hello"},
			},
			expectedTitle: map[string]string{
				"posts/skipped-post.md": "", // Should remain nil
			},
		},
		{
			name: "generates title for post with empty title",
			posts: func() []*models.Post {
				emptyTitle := ""
				return []*models.Post{
					{Path: "posts/empty-title.md", Title: &emptyTitle, Content: "Hello"},
				}
			}(),
			expectedTitle: map[string]string{
				"posts/empty-title.md": "Empty Title",
			},
		},
		{
			name: "multiple posts mixed",
			posts: func() []*models.Post {
				customTitle := "Custom"
				return []*models.Post{
					{Path: "posts/auto-gen.md", Content: "Hello"},
					{Path: "posts/has-title.md", Title: &customTitle, Content: "World"},
					{Path: "posts/another-auto.md", Content: "!"},
				}
			}(),
			expectedTitle: map[string]string{
				"posts/auto-gen.md":     "Auto Gen",
				"posts/has-title.md":    "Custom",
				"posts/another-auto.md": "Another Auto",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a manager with the posts
			m := lifecycle.NewManager()
			for _, post := range tt.posts {
				m.AddPost(post)
			}

			// Run the transform
			err := plugin.Transform(m)
			if err != nil {
				t.Fatalf("Transform() error = %v", err)
			}

			// Check titles
			for _, post := range m.Posts() {
				expected := tt.expectedTitle[post.Path]
				var got string
				if post.Title != nil {
					got = *post.Title
				}
				if got != expected {
					t.Errorf("post %q: title = %q, want %q", post.Path, got, expected)
				}
			}
		})
	}
}

func TestToTitleCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", "Hello World"},
		{"HELLO WORLD", "Hello World"},
		{"hElLo WoRlD", "Hello World"},
		{"hello", "Hello"},
		{"", ""},
		{"multiple   spaces", "Multiple Spaces"},
		{"c++ tutorial", "C++ Tutorial"},
		{"api v2 reference", "Api V2 Reference"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toTitleCase(tt.input)
			if got != tt.expected {
				t.Errorf("toTitleCase(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
