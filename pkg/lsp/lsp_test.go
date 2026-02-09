package lsp

import (
	"io"
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

func TestIndexAliases(t *testing.T) {
	logger := log.New(os.Stderr, "[test] ", 0)
	idx := NewIndex(logger)

	// Test indexing content with aliases
	content := `---
title: "What I'm Doing Now"
slug: now
aliases:
  - doing
  - upto
---

This is my now page.
`

	err := idx.indexContent("now.md", content)
	if err != nil {
		t.Fatalf("indexContent failed: %v", err)
	}

	// Test GetBySlug returns post for slug
	post := idx.GetBySlug("now")
	if post == nil {
		t.Fatal("GetBySlug('now') returned nil")
	}
	if post.Title != "What I'm Doing Now" {
		t.Errorf("Title = %q, want %q", post.Title, "What I'm Doing Now")
	}

	// Test Aliases field is populated
	if len(post.Aliases) != 2 {
		t.Errorf("got %d aliases, want 2", len(post.Aliases))
	}

	// Test GetBySlug returns post for alias
	postByAlias := idx.GetBySlug("doing")
	if postByAlias == nil {
		t.Fatal("GetBySlug('doing') returned nil for alias")
	}
	if postByAlias != post {
		t.Error("GetBySlug('doing') should return same post as GetBySlug('now')")
	}

	// Test SearchPostsWithMatch finds post by alias prefix
	results := idx.SearchPostsWithMatch("do")
	if len(results) != 1 {
		t.Errorf("SearchPostsWithMatch('do') returned %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0].MatchedBy != MatchTypeAlias {
		t.Errorf("SearchPostsWithMatch('do').MatchedBy = %q, want %q", results[0].MatchedBy, MatchTypeAlias)
	}

	// Test SearchPostsWithMatch prefers slug over alias
	resultsNow := idx.SearchPostsWithMatch("now")
	if len(resultsNow) != 1 {
		t.Errorf("SearchPostsWithMatch('now') returned %d results, want 1", len(resultsNow))
	}
	if len(resultsNow) > 0 && resultsNow[0].MatchedBy != MatchTypeSlug {
		t.Errorf("SearchPostsWithMatch('now').MatchedBy = %q, want %q", resultsNow[0].MatchedBy, MatchTypeSlug)
	}

	// Test AllPosts deduplicates entries
	allPosts := idx.AllPosts()
	if len(allPosts) != 1 {
		t.Errorf("AllPosts returned %d posts, want 1 (should deduplicate aliases)", len(allPosts))
	}
}

func TestIndexAliasSlugPrecedence(t *testing.T) {
	logger := log.New(os.Stderr, "[test] ", 0)
	idx := NewIndex(logger)

	// Index two posts where one has an alias that matches the other's slug
	content1 := `---
title: "Post About Doing Things"
slug: doing
---

Main content.
`
	content2 := `---
title: "What I'm Doing Now"
slug: now
aliases:
  - doing
  - upto
---

My now page.
`

	// Index the slug first
	if err := idx.indexContent("doing.md", content1); err != nil {
		t.Fatalf("indexContent for doing.md failed: %v", err)
	}
	// Index the post with alias second
	if err := idx.indexContent("now.md", content2); err != nil {
		t.Fatalf("indexContent for now.md failed: %v", err)
	}

	// Slug should take precedence
	postDoing := idx.GetBySlug("doing")
	if postDoing == nil {
		t.Fatal("GetBySlug('doing') returned nil")
	}
	if postDoing.Title != "Post About Doing Things" {
		t.Errorf("GetBySlug('doing').Title = %q, want %q (slug should take precedence over alias)",
			postDoing.Title, "Post About Doing Things")
	}

	// Other alias should still work
	postUpto := idx.GetBySlug("upto")
	if postUpto == nil {
		t.Fatal("GetBySlug('upto') returned nil")
	}
	if postUpto.Title != "What I'm Doing Now" {
		t.Errorf("GetBySlug('upto').Title = %q, want %q", postUpto.Title, "What I'm Doing Now")
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

func TestGetMentionContext(t *testing.T) {
	tests := []struct {
		name          string
		line          string
		col           int
		wantPrefix    string
		wantStart     int
		wantInMention bool
	}{
		{
			name:          "start of mention",
			line:          "Hello @",
			col:           7,
			wantPrefix:    "",
			wantStart:     0,
			wantInMention: false, // @ alone without a letter isn't a valid mention start
		},
		{
			name:          "partial handle",
			line:          "Hello @dave",
			col:           11,
			wantPrefix:    "dave",
			wantStart:     7,
			wantInMention: true,
		},
		{
			name:          "middle of handle",
			line:          "Hello @daverupert!",
			col:           12,
			wantPrefix:    "daver",
			wantStart:     7,
			wantInMention: true,
		},
		{
			name:          "handle with dots",
			line:          "See @simon.willison.net",
			col:           23,
			wantPrefix:    "simon.willison.net",
			wantStart:     5,
			wantInMention: true,
		},
		{
			name:          "not in mention - no @",
			line:          "Hello world",
			col:           5,
			wantPrefix:    "",
			wantStart:     0,
			wantInMention: false,
		},
		{
			name:          "email address - not a mention",
			line:          "Email me at test@example.com",
			col:           22,
			wantPrefix:    "",
			wantStart:     0,
			wantInMention: false,
		},
		{
			name:          "double @ - not a mention",
			line:          "Something @@handle",
			col:           15,
			wantPrefix:    "",
			wantStart:     0,
			wantInMention: false,
		},
		{
			name:          "start_of_line_mention",
			line:          "@daverupert is cool",
			col:           11,
			wantPrefix:    "daverupert",
			wantStart:     1,
			wantInMention: true,
		},
		{
			name:          "after space mention",
			line:          "Thanks @jane",
			col:           12,
			wantPrefix:    "jane",
			wantStart:     8,
			wantInMention: true,
		},
		{
			name:          "empty line",
			line:          "",
			col:           0,
			wantPrefix:    "",
			wantStart:     0,
			wantInMention: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, startCol, inMention := getMentionContext(tt.line, tt.col)
			if prefix != tt.wantPrefix {
				t.Errorf("prefix = %q, want %q", prefix, tt.wantPrefix)
			}
			if startCol != tt.wantStart {
				t.Errorf("startCol = %d, want %d", startCol, tt.wantStart)
			}
			if inMention != tt.wantInMention {
				t.Errorf("inMention = %v, want %v", inMention, tt.wantInMention)
			}
		})
	}
}

func TestIndexBlogrollMentions(t *testing.T) {
	logger := log.New(os.Stderr, "[test] ", 0)
	idx := NewIndex(logger)

	// Create a temp directory with a config file
	tmpDir := t.TempDir()
	configContent := `
[blogroll]
enabled = true

[[blogroll.feeds]]
url = "https://daverupert.com/feed.xml"
title = "Dave Rupert"
handle = "daverupert"
aliases = ["dave", "rupert"]

[[blogroll.feeds]]
url = "https://simonwillison.net/atom/everything/"
title = "Simon Willison"
site_url = "https://simonwillison.net"
`
	configPath := filepath.Join(tmpDir, "markata-go.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Build index which should load mentions
	if err := idx.Build(tmpDir); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Check that mentions were indexed
	mentions := idx.AllMentions()
	if len(mentions) != 2 {
		t.Errorf("got %d mentions, want 2", len(mentions))
	}

	// Test searching
	results := idx.SearchMentions("dave")
	if len(results) != 1 {
		t.Errorf("SearchMentions(dave) returned %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0].Handle != "daverupert" {
		t.Errorf("SearchMentions(dave) handle = %q, want %q", results[0].Handle, "daverupert")
	}

	// Test alias searching
	results = idx.SearchMentions("rupert")
	if len(results) != 1 {
		t.Errorf("SearchMentions(rupert) returned %d results, want 1", len(results))
	}

	// Test auto-generated handle from site_url
	results = idx.SearchMentions("simon")
	if len(results) != 1 {
		t.Errorf("SearchMentions(simon) returned %d results, want 1", len(results))
	}
}

func TestFromPostsMentions(t *testing.T) {
	// Create temp directory with config and contact posts
	tmpDir := t.TempDir()
	logger := log.New(io.Discard, "", 0)
	idx := NewIndex(logger)

	// Create config with from_posts mentions
	configContent := `[markata-go.mentions]
css_class = "mention"

[[markata-go.mentions.from_posts]]
filter = "'contact' in tags"
handle_field = "slug"
`
	configPath := filepath.Join(tmpDir, "markata-go.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create contact posts
	aliceContent := `---
title: Alice Smith
slug: alice-smith
tags:
  - contact
---
Alice is a developer.
`
	alicePath := filepath.Join(tmpDir, "alice-smith.md")
	if err := os.WriteFile(alicePath, []byte(aliceContent), 0o600); err != nil {
		t.Fatalf("Failed to write alice.md: %v", err)
	}

	bobContent := `---
title: Bob Jones
slug: bob-jones
tags:
  - contact
  - team
---
Bob is a designer.
`
	bobPath := filepath.Join(tmpDir, "bob-jones.md")
	if err := os.WriteFile(bobPath, []byte(bobContent), 0o600); err != nil {
		t.Fatalf("Failed to write bob.md: %v", err)
	}

	// Create a non-contact post (should not be indexed as mention)
	blogContent := `---
title: My Blog Post
slug: my-blog-post
tags:
  - blog
---
This is a blog post.
`
	blogPath := filepath.Join(tmpDir, "my-blog-post.md")
	if err := os.WriteFile(blogPath, []byte(blogContent), 0o600); err != nil {
		t.Fatalf("Failed to write blog.md: %v", err)
	}

	// Build index
	if err := idx.Build(tmpDir); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Check that contact mentions were indexed
	mentions := idx.AllMentions()
	if len(mentions) != 2 {
		t.Errorf("got %d mentions, want 2 (alice-smith, bob-jones)", len(mentions))
	}

	// Test specific handle lookup
	alice := idx.GetByHandle("alice-smith")
	if alice == nil {
		t.Error("GetByHandle(alice-smith) returned nil")
	} else {
		if alice.Handle != "alice-smith" {
			t.Errorf("alice.Handle = %q, want %q", alice.Handle, "alice-smith")
		}
		if alice.Title != "Alice Smith" {
			t.Errorf("alice.Title = %q, want %q", alice.Title, "Alice Smith")
		}
		if !alice.IsInternal {
			t.Error("alice.IsInternal should be true for from_posts mention")
		}
		if alice.Slug != "alice-smith" {
			t.Errorf("alice.Slug = %q, want %q", alice.Slug, "alice-smith")
		}
	}

	bob := idx.GetByHandle("bob-jones")
	if bob == nil {
		t.Error("GetByHandle(bob-jones) returned nil")
	}

	// Test that non-contact post is NOT a mention
	blogMention := idx.GetByHandle("my-blog-post")
	if blogMention != nil {
		t.Error("GetByHandle(my-blog-post) should return nil for non-contact post")
	}

	// Test prefix search
	results := idx.SearchMentions("alice")
	if len(results) != 1 {
		t.Errorf("SearchMentions(alice) returned %d results, want 1", len(results))
	}
}

func TestGetMentionAtPosition(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		col        int
		lineNum    int
		wantHandle string
		wantNil    bool
	}{
		{
			name:       "cursor on mention",
			line:       "See @daverupert here",
			col:        10,
			lineNum:    5,
			wantHandle: "daverupert",
			wantNil:    false,
		},
		{
			name:       "cursor at start of mention",
			line:       "See @daverupert here",
			col:        4,
			lineNum:    0,
			wantHandle: "daverupert",
			wantNil:    false,
		},
		{
			name:       "cursor at end of mention",
			line:       "See @daverupert here",
			col:        15,
			lineNum:    0,
			wantHandle: "daverupert",
			wantNil:    false,
		},
		{
			name:       "cursor not on mention",
			line:       "See @daverupert here",
			col:        18,
			lineNum:    0,
			wantHandle: "",
			wantNil:    true,
		},
		{
			name:       "mention with dots",
			line:       "See @simon.willison.net here",
			col:        15,
			lineNum:    0,
			wantHandle: "simon.willison.net",
			wantNil:    false,
		},
		{
			name:       "no mentions",
			line:       "Just regular text",
			col:        5,
			lineNum:    0,
			wantHandle: "",
			wantNil:    true,
		},
		{
			name:       "email not a mention",
			line:       "Email test@example.com here",
			col:        15,
			lineNum:    0,
			wantHandle: "",
			wantNil:    true,
		},
		{
			name:       "start of line mention",
			line:       "@daverupert said something",
			col:        5,
			lineNum:    0,
			wantHandle: "daverupert",
			wantNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handle, rng := getMentionAtPosition(tt.line, tt.col, tt.lineNum)
			if handle != tt.wantHandle {
				t.Errorf("handle = %q, want %q", handle, tt.wantHandle)
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

func TestGetByHandle(t *testing.T) {
	logger := log.New(os.Stderr, "[test] ", 0)
	idx := NewIndex(logger)

	// Create a temp directory with a config file
	tmpDir := t.TempDir()
	configContent := `
[blogroll]
enabled = true

[[blogroll.feeds]]
url = "https://daverupert.com/feed.xml"
title = "Dave Rupert"
site_url = "https://daverupert.com"
handle = "daverupert"
aliases = ["dave", "rupert"]
`
	configPath := filepath.Join(tmpDir, "markata-go.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Build index
	if err := idx.Build(tmpDir); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Test GetByHandle with primary handle
	mention := idx.GetByHandle("daverupert")
	if mention == nil {
		t.Fatal("GetByHandle(daverupert) returned nil")
	}
	if mention.Title != "Dave Rupert" {
		t.Errorf("Title = %q, want %q", mention.Title, "Dave Rupert")
	}

	// Test GetByHandle with alias
	mention = idx.GetByHandle("dave")
	if mention == nil {
		t.Fatal("GetByHandle(dave) returned nil")
	}
	if mention.Handle != "daverupert" {
		t.Errorf("Handle = %q, want %q", mention.Handle, "daverupert")
	}

	// Test GetByHandle with another alias
	mention = idx.GetByHandle("rupert")
	if mention == nil {
		t.Fatal("GetByHandle(rupert) returned nil")
	}

	// Test GetByHandle case insensitivity
	mention = idx.GetByHandle("DAVERUPERT")
	if mention == nil {
		t.Fatal("GetByHandle(DAVERUPERT) returned nil - should be case insensitive")
	}

	// Test GetByHandle with unknown handle
	mention = idx.GetByHandle("unknown")
	if mention != nil {
		t.Errorf("GetByHandle(unknown) returned non-nil: %v", mention)
	}

	// Test GetByHandle with domain alias
	mention = idx.GetByHandle("daverupert.com")
	if mention == nil {
		t.Fatal("GetByHandle(daverupert.com) returned nil - domain alias should work")
	}
}

func TestMentionDiagnostics(t *testing.T) {
	logger := log.New(os.Stderr, "[test] ", 0)
	idx := NewIndex(logger)

	// Create a temp directory with a config file
	tmpDir := t.TempDir()
	configContent := `
[blogroll]
enabled = true

[[blogroll.feeds]]
url = "https://daverupert.com/feed.xml"
title = "Dave Rupert"
handle = "daverupert"
`
	configPath := filepath.Join(tmpDir, "markata-go.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Build index
	if err := idx.Build(tmpDir); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Create a server with the index
	server := &Server{
		index:  idx,
		logger: logger,
	}

	tests := []struct {
		name         string
		content      string
		wantWarnings int
		wantCodes    []string
	}{
		{
			name:         "valid mention",
			content:      "Hello @daverupert!",
			wantWarnings: 0,
			wantCodes:    nil,
		},
		{
			name:         "unknown mention",
			content:      "Hello @unknown!",
			wantWarnings: 1,
			wantCodes:    []string{"unknown-mention"},
		},
		{
			name:         "multiple mentions mixed",
			content:      "Hello @daverupert and @unknown!",
			wantWarnings: 1,
			wantCodes:    []string{"unknown-mention"},
		},
		{
			name:         "mention in code block ignored",
			content:      "```\n@unknown\n```",
			wantWarnings: 0,
			wantCodes:    nil,
		},
		{
			name:         "email not flagged",
			content:      "Email test@example.com for help",
			wantWarnings: 0,
			wantCodes:    nil,
		},
		{
			name:         "broken wikilink",
			content:      "See [[nonexistent]]",
			wantWarnings: 1,
			wantCodes:    []string{"broken-wikilink"},
		},
		{
			name:         "both broken wikilink and unknown mention",
			content:      "See [[nonexistent]] and @unknown",
			wantWarnings: 2,
			wantCodes:    []string{"broken-wikilink", "unknown-mention"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnostics := server.computeDiagnostics("test.md", tt.content)
			if len(diagnostics) != tt.wantWarnings {
				t.Errorf("got %d diagnostics, want %d", len(diagnostics), tt.wantWarnings)
				for i, d := range diagnostics {
					t.Logf("  diagnostic %d: %s (%s)", i, d.Message, d.Code)
				}
			}

			// Check diagnostic codes
			for i, wantCode := range tt.wantCodes {
				if i < len(diagnostics) && diagnostics[i].Code != wantCode {
					t.Errorf("diagnostic %d: code = %q, want %q", i, diagnostics[i].Code, wantCode)
				}
			}
		})
	}
}
