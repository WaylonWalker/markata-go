package lsp

import (
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestGetWikilinkContext(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		col        int
		wantPrefix string
		wantStart  int
		wantInLink bool
	}{
		{
			name:       "start of wikilink",
			line:       "See [[",
			col:        6,
			wantPrefix: "",
			wantStart:  6,
			wantInLink: true,
		},
		{
			name:       "partial slug",
			line:       "See [[my-po",
			col:        11,
			wantPrefix: "my-po",
			wantStart:  6,
			wantInLink: true,
		},
		{
			name:       "middle of slug",
			line:       "See [[my-post]]",
			col:        9,
			wantPrefix: "my-",
			wantStart:  6,
			wantInLink: true,
		},
		{
			name:       "not in wikilink",
			line:       "See my-post",
			col:        8,
			wantPrefix: "",
			wantStart:  0,
			wantInLink: false,
		},
		{
			name:       "after closing brackets",
			line:       "See [[my-post]] and more",
			col:        20,
			wantPrefix: "",
			wantStart:  0,
			wantInLink: false,
		},
		{
			name:       "in display text",
			line:       "See [[my-post|Display",
			col:        20,
			wantPrefix: "",
			wantStart:  0,
			wantInLink: false,
		},
		{
			name:       "empty line",
			line:       "",
			col:        0,
			wantPrefix: "",
			wantStart:  0,
			wantInLink: false,
		},
		{
			name:       "single bracket",
			line:       "See [incomplete",
			col:        10,
			wantPrefix: "",
			wantStart:  0,
			wantInLink: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, startCol, inLink := getWikilinkContext(tt.line, tt.col)
			if prefix != tt.wantPrefix {
				t.Errorf("prefix = %q, want %q", prefix, tt.wantPrefix)
			}
			if startCol != tt.wantStart {
				t.Errorf("startCol = %d, want %d", startCol, tt.wantStart)
			}
			if inLink != tt.wantInLink {
				t.Errorf("inLink = %v, want %v", inLink, tt.wantInLink)
			}
		})
	}
}

func TestGetWikilinkAtPosition(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		col      int
		lineNum  int
		wantSlug string
		wantNil  bool
	}{
		{
			name:     "cursor on wikilink",
			line:     "See [[my-post]] here",
			col:      10,
			lineNum:  5,
			wantSlug: "my-post",
			wantNil:  false,
		},
		{
			name:     "cursor at start of wikilink",
			line:     "See [[my-post]] here",
			col:      4,
			lineNum:  0,
			wantSlug: "my-post",
			wantNil:  false,
		},
		{
			name:     "cursor at end of wikilink",
			line:     "See [[my-post]] here",
			col:      15,
			lineNum:  0,
			wantSlug: "my-post",
			wantNil:  false,
		},
		{
			name:     "cursor not on wikilink",
			line:     "See [[my-post]] here",
			col:      18,
			lineNum:  0,
			wantSlug: "",
			wantNil:  true,
		},
		{
			name:     "wikilink with display text",
			line:     "See [[my-post|My Post Title]]",
			col:      10,
			lineNum:  0,
			wantSlug: "my-post",
			wantNil:  false,
		},
		{
			name:     "no wikilinks",
			line:     "Just regular text",
			col:      5,
			lineNum:  0,
			wantSlug: "",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug, rng := getWikilinkAtPosition(tt.line, tt.col, tt.lineNum)
			if slug != tt.wantSlug {
				t.Errorf("slug = %q, want %q", slug, tt.wantSlug)
			}
			if (rng == nil) != tt.wantNil {
				t.Errorf("range nil = %v, want nil = %v", rng == nil, tt.wantNil)
			}
			if rng != nil && rng.Start.Line != tt.lineNum {
				t.Errorf("range line = %d, want %d", rng.Start.Line, tt.lineNum)
			}
		})
	}
}

func TestFindWikilinks(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []WikilinkInfo
	}{
		{
			name:    "single wikilink",
			content: "Link to [[my-post]]",
			want: []WikilinkInfo{
				{Target: "my-post", Line: 0, StartChar: 8, EndChar: 19},
			},
		},
		{
			name:    "multiple wikilinks",
			content: "Link [[one]] and [[two]]",
			want: []WikilinkInfo{
				{Target: "one", Line: 0, StartChar: 5, EndChar: 12},
				{Target: "two", Line: 0, StartChar: 17, EndChar: 24},
			},
		},
		{
			name:    "wikilink with display text",
			content: "See [[slug|Display Text]]",
			want: []WikilinkInfo{
				{Target: "slug", DisplayText: "Display Text", Line: 0, StartChar: 4, EndChar: 25},
			},
		},
		{
			name:    "multiline content",
			content: "Line 1 [[post1]]\nLine 2 [[post2]]",
			want: []WikilinkInfo{
				{Target: "post1", Line: 0, StartChar: 7, EndChar: 16},
				{Target: "post2", Line: 1, StartChar: 7, EndChar: 16},
			},
		},
		{
			name:    "no wikilinks",
			content: "Just regular text with [brackets] but not wikilinks",
			want:    []WikilinkInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findWikilinks(tt.content)
			if len(got) != len(tt.want) {
				t.Errorf("got %d wikilinks, want %d", len(got), len(tt.want))
				return
			}
			for i, w := range got {
				if w.Target != tt.want[i].Target {
					t.Errorf("wikilink %d: target = %q, want %q", i, w.Target, tt.want[i].Target)
				}
				if w.DisplayText != tt.want[i].DisplayText {
					t.Errorf("wikilink %d: displayText = %q, want %q", i, w.DisplayText, tt.want[i].DisplayText)
				}
				if w.Line != tt.want[i].Line {
					t.Errorf("wikilink %d: line = %d, want %d", i, w.Line, tt.want[i].Line)
				}
			}
		})
	}
}

func TestNormalizeSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"My Post", "my-post"},
		{"my-post", "my-post"},
		{"MY-POST", "my-post"},
		{"my_post", "my_post"},
		{"My   Post", "my-post"},
		{"Post!!!", "post"},
		{"  trimmed  ", "trimmed"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeSlug(tt.input)
			if got != tt.want {
				t.Errorf("normalizeSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractExcerpt(t *testing.T) {
	tests := []struct {
		name    string
		content string
		maxLen  int
		want    string
	}{
		{
			name:    "short content",
			content: "Hello world",
			maxLen:  100,
			want:    "Hello world",
		},
		{
			name:    "content with header",
			content: "# Title\n\nThis is the body.",
			maxLen:  100,
			want:    "This is the body.",
		},
		{
			name:    "long content truncated",
			content: "This is a very long paragraph that should be truncated.",
			maxLen:  20,
			want:    "This is a very lo...",
		},
		{
			name:    "empty content",
			content: "",
			maxLen:  100,
			want:    "",
		},
		{
			name:    "only headers",
			content: "# Title\n## Subtitle",
			maxLen:  100,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractExcerpt(tt.content, tt.maxLen)
			if got != tt.want {
				t.Errorf("extractExcerpt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIndex(t *testing.T) {
	logger := log.New(os.Stderr, "[test] ", 0)
	idx := NewIndex(logger)

	// Test indexing content
	content := `---
title: Test Post
description: A test post
slug: test-post
---

This is the body with a [[wikilink]].
`

	err := idx.indexContent("test.md", content)
	if err != nil {
		t.Fatalf("indexContent failed: %v", err)
	}

	// Test GetBySlug
	post := idx.GetBySlug("test-post")
	if post == nil {
		t.Fatal("GetBySlug returned nil")
	}
	if post.Title != "Test Post" {
		t.Errorf("Title = %q, want %q", post.Title, "Test Post")
	}
	if post.Description != "A test post" {
		t.Errorf("Description = %q, want %q", post.Description, "A test post")
	}

	// Test wikilinks extraction
	if len(post.Wikilinks) != 1 {
		t.Errorf("got %d wikilinks, want 1", len(post.Wikilinks))
	}
	if len(post.Wikilinks) > 0 && post.Wikilinks[0].Target != "wikilink" {
		t.Errorf("wikilink target = %q, want %q", post.Wikilinks[0].Target, "wikilink")
	}

	// Test SearchPosts
	results := idx.SearchPosts("test")
	if len(results) != 1 {
		t.Errorf("SearchPosts returned %d results, want 1", len(results))
	}

	// Test case-insensitive GetBySlug
	post2 := idx.GetBySlug("Test-Post")
	if post2 == nil {
		t.Error("Case-insensitive GetBySlug returned nil")
	}
}

func TestURIConversion(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"temp file", filepath.Join(t.TempDir(), "test.md")},
		{"nested file", filepath.Join(t.TempDir(), "docs", "file.md")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := pathToURI(tt.path)
			path := uriToPath(uri)
			if path != tt.path {
				t.Errorf("round-trip: got %q, want %q (uri: %q)", path, tt.path, uri)
			}
		})
	}
}
