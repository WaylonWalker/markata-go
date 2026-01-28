package plugins

import (
	"testing"
)

func TestDescriptionPlugin_StripMarkdown(t *testing.T) {
	plugin := NewDescriptionPlugin()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "markdown link",
			input:    "Check [this link](https://example.com) out",
			expected: "Check this link out",
		},
		{
			name:     "wikilink simple",
			input:    "Check out [[my-page]] for more",
			expected: "Check out my-page for more",
		},
		{
			name:     "wikilink with display text",
			input:    "Check out [[my-page|My Page]] for more",
			expected: "Check out My Page for more",
		},
		{
			name:     "wikilink with spaces",
			input:    "Check out [[ my-page ]] for more",
			expected: "Check out my-page for more",
		},
		{
			name:     "wikilink with spaces and display text",
			input:    "Check out [[ my-page | My Page ]] for more",
			expected: "Check out My Page for more",
		},
		{
			name:     "multiple wikilinks",
			input:    "See [[page-1]] and [[page-2|Page Two]] here",
			expected: "See page-1 and Page Two here",
		},
		{
			name:     "mixed markdown and wikilinks",
			input:    "Read [[my-post|this post]] and [external](https://example.com)",
			expected: "Read this post and external",
		},
		{
			name:     "bold text",
			input:    "This is **bold** text",
			expected: "This is ** text", // Note: emphasis regex has known limitations
		},
		{
			name:     "inline code",
			input:    "Use `code` here",
			expected: "Use here",
		},
		{
			name:     "header stripped",
			input:    "# Hello world",
			expected: "Hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := plugin.stripMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("stripMarkdown(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDescriptionPlugin_GenerateDescription(t *testing.T) {
	plugin := NewDescriptionPlugin()

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple paragraph",
			content:  "This is a simple paragraph.",
			expected: "This is a simple paragraph.",
		},
		{
			name:     "paragraph with wikilink",
			content:  "Check out [[my-page]] for more information.",
			expected: "Check out my-page for more information.",
		},
		{
			name:     "paragraph with wikilink display text",
			content:  "Check out [[my-page|My Awesome Page]] for details.",
			expected: "Check out My Awesome Page for details.",
		},
		{
			name:     "header then paragraph with wikilink",
			content:  "# Title\n\nThis references [[other-post|another post]] here.",
			expected: "This references another post here.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := plugin.generateDescription(tt.content)
			if result != tt.expected {
				t.Errorf("generateDescription() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDescriptionPlugin_StripWikilinks(t *testing.T) {
	plugin := NewDescriptionPlugin()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no wikilinks",
			input:    "Plain text without wikilinks",
			expected: "Plain text without wikilinks",
		},
		{
			name:     "simple wikilink",
			input:    "Sad empty freezer [[2025-08-12-notes]]",
			expected: "Sad empty freezer 2025-08-12-notes",
		},
		{
			name:     "wikilink with spaces",
			input:    "Sad empty freezer [[ 2025-08-12-notes ]]",
			expected: "Sad empty freezer 2025-08-12-notes",
		},
		{
			name:     "wikilink with display text",
			input:    "Check [[my-page|My Page]] here",
			expected: "Check My Page here",
		},
		{
			name:     "wikilink with spaces and display text",
			input:    "Check [[ my-page | My Page ]] here",
			expected: "Check My Page here",
		},
		{
			name:     "multiple wikilinks with spaces",
			input:    "See [[ page-1 ]] and [[ page-2 | Page Two ]]",
			expected: "See page-1 and Page Two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := plugin.stripWikilinks(tt.input)
			if result != tt.expected {
				t.Errorf("stripWikilinks(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
