package plugins

import (
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestMentionsPlugin_Name(t *testing.T) {
	p := NewMentionsPlugin()
	if got := p.Name(); got != "mentions" {
		t.Errorf("Name() = %q, want %q", got, "mentions")
	}
}

func TestMentionsPlugin_ProcessMentions(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		handleMap map[string]*mentionEntry
		want      string
	}{
		{
			name:    "simple mention",
			content: "I was reading @daverupert's post",
			handleMap: map[string]*mentionEntry{
				"daverupert": {Handle: "daverupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
			},
			want: `I was reading <a href="https://daverupert.com" class="mention">@daverupert</a>'s post`,
		},
		{
			name:    "mention at start of line",
			content: "@daverupert wrote about CSS",
			handleMap: map[string]*mentionEntry{
				"daverupert": {Handle: "daverupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
			},
			want: `<a href="https://daverupert.com" class="mention">@daverupert</a> wrote about CSS`,
		},
		{
			name:    "mention at end of line",
			content: "Great article by @daverupert",
			handleMap: map[string]*mentionEntry{
				"daverupert": {Handle: "daverupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
			},
			want: `Great article by <a href="https://daverupert.com" class="mention">@daverupert</a>`,
		},
		{
			name:    "multiple mentions",
			content: "Both @alice and @bob are great",
			handleMap: map[string]*mentionEntry{
				"alice": {Handle: "alice", SiteURL: "https://alice.dev", Title: "Alice"},
				"bob":   {Handle: "bob", SiteURL: "https://bob.io", Title: "Bob"},
			},
			want: `Both <a href="https://alice.dev" class="mention">@alice</a> and <a href="https://bob.io" class="mention">@bob</a> are great`,
		},
		{
			name:    "unknown mention preserved",
			content: "I follow @unknown",
			handleMap: map[string]*mentionEntry{
				"known": {Handle: "known", SiteURL: "https://known.com", Title: "Known"},
			},
			want: "I follow @unknown",
		},
		{
			name:    "mention in code block preserved",
			content: "Check this:\n```\n@daverupert\n```\nAnd @daverupert outside",
			handleMap: map[string]*mentionEntry{
				"daverupert": {Handle: "daverupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
			},
			want: "Check this:\n```\n@daverupert\n```\nAnd <a href=\"https://daverupert.com\" class=\"mention\">@daverupert</a> outside",
		},
		{
			name:    "email address not replaced",
			content: "Contact me at test@example.com",
			handleMap: map[string]*mentionEntry{
				"example": {Handle: "example", SiteURL: "https://example.com", Title: "Example"},
			},
			want: "Contact me at test@example.com",
		},
		{
			name:    "case insensitive handle matching",
			content: "Follow @DaveRupert",
			handleMap: map[string]*mentionEntry{
				"daverupert": {Handle: "daverupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
			},
			want: `Follow <a href="https://daverupert.com" class="mention">@daverupert</a>`,
		},
		{
			name:    "handle with hyphen",
			content: "Check out @dave-rupert",
			handleMap: map[string]*mentionEntry{
				"dave-rupert": {Handle: "dave-rupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
			},
			want: `Check out <a href="https://daverupert.com" class="mention">@dave-rupert</a>`,
		},
		{
			name:    "handle with underscore",
			content: "Check out @dave_rupert",
			handleMap: map[string]*mentionEntry{
				"dave_rupert": {Handle: "dave_rupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
			},
			want: `Check out <a href="https://daverupert.com" class="mention">@dave_rupert</a>`,
		},
		{
			name:      "empty handleMap",
			content:   "Hello @world",
			handleMap: map[string]*mentionEntry{},
			want:      "Hello @world",
		},
		{
			name:    "special characters escaped in URL",
			content: "Check @test",
			handleMap: map[string]*mentionEntry{
				"test": {Handle: "test", SiteURL: "https://test.com/path?a=1&b=2", Title: "Test"},
			},
			want: `Check <a href="https://test.com/path?a=1&amp;b=2" class="mention">@test</a>`,
		},
		{
			name:    "domain-style handle",
			content: "I was reading @simonwillison.net's post",
			handleMap: map[string]*mentionEntry{
				"simonwillison.net": {Handle: "simonwillison.net", SiteURL: "https://simonwillison.net", Title: "Simon Willison"},
			},
			want: `I was reading <a href="https://simonwillison.net" class="mention">@simonwillison.net</a>'s post`,
		},
		{
			name:    "domain-style handle at start",
			content: "@simonwillison.net wrote about LLMs",
			handleMap: map[string]*mentionEntry{
				"simonwillison.net": {Handle: "simonwillison.net", SiteURL: "https://simonwillison.net", Title: "Simon Willison"},
			},
			want: `<a href="https://simonwillison.net" class="mention">@simonwillison.net</a> wrote about LLMs`,
		},
		{
			name:    "domain-style handle at end",
			content: "Great article by @simonwillison.net",
			handleMap: map[string]*mentionEntry{
				"simonwillison.net": {Handle: "simonwillison.net", SiteURL: "https://simonwillison.net", Title: "Simon Willison"},
			},
			want: `Great article by <a href="https://simonwillison.net" class="mention">@simonwillison.net</a>`,
		},
		{
			name:    "handle with multiple dots",
			content: "Follow @blog.example.co.uk",
			handleMap: map[string]*mentionEntry{
				"blog.example.co.uk": {Handle: "blog.example.co.uk", SiteURL: "https://blog.example.co.uk", Title: "Example Blog"},
			},
			want: `Follow <a href="https://blog.example.co.uk" class="mention">@blog.example.co.uk</a>`,
		},
		{
			name:    "mixed handles simple and domain-style",
			content: "Both @daverupert and @simonwillison.net are great",
			handleMap: map[string]*mentionEntry{
				"daverupert":        {Handle: "daverupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
				"simonwillison.net": {Handle: "simonwillison.net", SiteURL: "https://simonwillison.net", Title: "Simon Willison"},
			},
			want: `Both <a href="https://daverupert.com" class="mention">@daverupert</a> and <a href="https://simonwillison.net" class="mention">@simonwillison.net</a> are great`,
		},
	}

	p := NewMentionsPlugin()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.processMentions(tt.content, tt.handleMap)
			if got != tt.want {
				t.Errorf("processMentions() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractHandleFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://daverupert.com", "daverupert"},
		{"https://www.example.com", "example"},
		{"https://blog.jane.dev", "jane"},
		{"https://blog.example.org/feed", "example"},
		{"http://test-site.io", "test-site"},
		{"https://mysite123.com", "mysite123"},
		{"https://www.blog.example.com", "example"}, // www. is stripped first, then blog.
		{"", ""},
		{"not-a-url", ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := extractHandleFromURL(tt.url)
			if got != tt.want {
				t.Errorf("extractHandleFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestExtractSiteURL(t *testing.T) {
	tests := []struct {
		feedURL string
		want    string
	}{
		{"https://daverupert.com/feed.xml", "https://daverupert.com"},
		{"https://example.com/blog/rss", "https://example.com"},
		{"http://test.org/atom.xml", "http://test.org"},
		{"", ""},
		{"not-a-url", ""},
	}

	for _, tt := range tests {
		t.Run(tt.feedURL, func(t *testing.T) {
			got := extractSiteURL(tt.feedURL)
			if got != tt.want {
				t.Errorf("extractSiteURL(%q) = %q, want %q", tt.feedURL, got, tt.want)
			}
		})
	}
}

func TestMentionsPlugin_BuildHandleMap(t *testing.T) {
	p := NewMentionsPlugin()
	m := lifecycle.NewManager()

	// Set up blogroll config
	boolTrue := true
	config := m.Config()
	config.Extra = map[string]interface{}{
		"blogroll": models.BlogrollConfig{
			Enabled: true,
			Feeds: []models.ExternalFeedConfig{
				{
					URL:     "https://daverupert.com/feed.xml",
					Title:   "Dave Rupert",
					Handle:  "daverupert",
					SiteURL: "https://daverupert.com",
					Active:  &boolTrue,
				},
				{
					URL:     "https://example.com/rss",
					Title:   "Example Blog",
					SiteURL: "https://example.com",
					Active:  &boolTrue,
					// No explicit handle - should auto-generate from domain
				},
				{
					URL:    "https://blog.test.org/feed",
					Title:  "Test Blog",
					Active: &boolTrue,
					// No SiteURL - should extract from feed URL
				},
			},
		},
	}

	handleMap := p.buildHandleMap(m)

	// Check explicit handle
	if entry, ok := handleMap["daverupert"]; !ok {
		t.Error("expected 'daverupert' in handleMap")
	} else if entry.SiteURL != "https://daverupert.com" {
		t.Errorf("daverupert.SiteURL = %q, want %q", entry.SiteURL, "https://daverupert.com")
	}

	// Check auto-generated handle from domain
	if entry, ok := handleMap["example"]; !ok {
		t.Error("expected 'example' in handleMap (auto-generated from domain)")
	} else if entry.SiteURL != "https://example.com" {
		t.Errorf("example.SiteURL = %q, want %q", entry.SiteURL, "https://example.com")
	}

	// Check auto-generated handle from feed URL domain
	if entry, ok := handleMap["test"]; !ok {
		t.Error("expected 'test' in handleMap (auto-generated from feed URL domain)")
	} else if entry.SiteURL != "https://blog.test.org" {
		t.Errorf("test.SiteURL = %q, want %q", entry.SiteURL, "https://blog.test.org")
	}
}

func TestMentionsPlugin_Transform(t *testing.T) {
	p := NewMentionsPlugin()
	m := lifecycle.NewManager()

	// Set up blogroll config
	boolTrue := true
	config := m.Config()
	config.Extra = map[string]interface{}{
		"blogroll": models.BlogrollConfig{
			Enabled: true,
			Feeds: []models.ExternalFeedConfig{
				{
					URL:     "https://daverupert.com/feed.xml",
					Title:   "Dave Rupert",
					Handle:  "daverupert",
					SiteURL: "https://daverupert.com",
					Active:  &boolTrue,
				},
			},
		},
	}

	// Add a test post
	post := &models.Post{
		Path:    "test.md",
		Content: "I was reading @daverupert's latest post about CSS",
	}
	m.AddPost(post)

	// Run transform
	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	// Check the result
	posts := m.Posts()
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}

	want := `I was reading <a href="https://daverupert.com" class="mention">@daverupert</a>'s latest post about CSS`
	if posts[0].Content != want {
		t.Errorf("Content = %q, want %q", posts[0].Content, want)
	}
}

func TestMentionsPlugin_Interfaces(_ *testing.T) {
	p := NewMentionsPlugin()

	// Verify interface implementations
	var _ lifecycle.Plugin = p
	var _ lifecycle.ConfigurePlugin = p
	var _ lifecycle.TransformPlugin = p
	var _ lifecycle.PriorityPlugin = p
}

func TestMentionsPlugin_AliasResolution(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		handleMap map[string]*mentionEntry
		want      string
	}{
		{
			name:    "alias resolves to canonical handle",
			content: "I was reading @dave's post",
			handleMap: map[string]*mentionEntry{
				"daverupert": {Handle: "daverupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
				"dave":       {Handle: "daverupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
			},
			want: `I was reading <a href="https://daverupert.com" class="mention">@daverupert</a>'s post`,
		},
		{
			name:    "multiple aliases for same person",
			content: "@dave and @david are the same as @daverupert",
			handleMap: map[string]*mentionEntry{
				"daverupert": {Handle: "daverupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
				"dave":       {Handle: "daverupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
				"david":      {Handle: "daverupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
			},
			want: `<a href="https://daverupert.com" class="mention">@daverupert</a> and <a href="https://daverupert.com" class="mention">@daverupert</a> are the same as <a href="https://daverupert.com" class="mention">@daverupert</a>`,
		},
		{
			name:    "alias case insensitive",
			content: "Follow @DAVE",
			handleMap: map[string]*mentionEntry{
				"daverupert": {Handle: "daverupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
				"dave":       {Handle: "daverupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
			},
			want: `Follow <a href="https://daverupert.com" class="mention">@daverupert</a>`,
		},
		{
			name:    "canonical handle still works",
			content: "Check out @daverupert",
			handleMap: map[string]*mentionEntry{
				"daverupert": {Handle: "daverupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
				"dave":       {Handle: "daverupert", SiteURL: "https://daverupert.com", Title: "Dave Rupert"},
			},
			want: `Check out <a href="https://daverupert.com" class="mention">@daverupert</a>`,
		},
	}

	p := NewMentionsPlugin()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.processMentions(tt.content, tt.handleMap)
			if got != tt.want {
				t.Errorf("processMentions() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMentionsPlugin_BuildHandleMapWithAliases(t *testing.T) {
	p := NewMentionsPlugin()
	m := lifecycle.NewManager()

	// Set up blogroll config with aliases
	boolTrue := true
	config := m.Config()
	config.Extra = map[string]interface{}{
		"blogroll": models.BlogrollConfig{
			Enabled: true,
			Feeds: []models.ExternalFeedConfig{
				{
					URL:     "https://daverupert.com/feed.xml",
					Title:   "Dave Rupert",
					Handle:  "daverupert",
					SiteURL: "https://daverupert.com",
					Active:  &boolTrue,
					Aliases: []string{"dave", "david", "rupert"},
				},
				{
					URL:     "https://example.com/rss",
					Title:   "Example Blog",
					Handle:  "example",
					SiteURL: "https://example.com",
					Active:  &boolTrue,
					Aliases: []string{"ex"},
				},
			},
		},
	}

	handleMap := p.buildHandleMap(m)

	// Check canonical handle
	if entry, ok := handleMap["daverupert"]; !ok {
		t.Error("expected 'daverupert' in handleMap")
	} else if entry.Handle != "daverupert" {
		t.Errorf("daverupert.Handle = %q, want %q", entry.Handle, "daverupert")
	}

	// Check aliases resolve to canonical handle
	for _, alias := range []string{"dave", "david", "rupert"} {
		if entry, ok := handleMap[alias]; !ok {
			t.Errorf("expected alias %q in handleMap", alias)
		} else if entry.Handle != "daverupert" {
			t.Errorf("alias %q resolves to Handle = %q, want %q", alias, entry.Handle, "daverupert")
		}
	}

	// Check second feed's alias
	if entry, ok := handleMap["ex"]; !ok {
		t.Error("expected alias 'ex' in handleMap")
	} else if entry.Handle != "example" {
		t.Errorf("alias 'ex' resolves to Handle = %q, want %q", entry.Handle, "example")
	}
}

func TestMentionsPlugin_DuplicateAliasFirstWins(t *testing.T) {
	p := NewMentionsPlugin()
	m := lifecycle.NewManager()

	// Set up blogroll config where two feeds have the same alias
	boolTrue := true
	config := m.Config()
	config.Extra = map[string]interface{}{
		"blogroll": models.BlogrollConfig{
			Enabled: true,
			Feeds: []models.ExternalFeedConfig{
				{
					URL:     "https://alice.com/feed.xml",
					Title:   "Alice",
					Handle:  "alice",
					SiteURL: "https://alice.com",
					Active:  &boolTrue,
					Aliases: []string{"al"}, // First feed with alias "al"
				},
				{
					URL:     "https://albert.com/rss",
					Title:   "Albert",
					Handle:  "albert",
					SiteURL: "https://albert.com",
					Active:  &boolTrue,
					Aliases: []string{"al"}, // Duplicate alias - should be ignored
				},
			},
		},
	}

	handleMap := p.buildHandleMap(m)

	// The alias "al" should resolve to alice (first entry wins)
	if entry, ok := handleMap["al"]; !ok {
		t.Error("expected alias 'al' in handleMap")
	} else if entry.Handle != "alice" {
		t.Errorf("alias 'al' resolves to Handle = %q, want %q (first entry wins)", entry.Handle, "alice")
	}
}

func TestMentionsPlugin_FromPosts(t *testing.T) {
	p := NewMentionsPlugin()
	m := lifecycle.NewManager()

	// Configure mentions with from_posts source
	config := m.Config()
	config.Extra = map[string]interface{}{
		"mentions": models.MentionsConfig{
			FromPosts: []models.MentionPostSource{
				{
					Filter:       "'contact' in tags",
					HandleField:  "handle",
					AliasesField: "aliases",
				},
			},
		},
	}

	// Add contact posts
	aliceTitle := "Alice Smith"
	post1 := &models.Post{
		Path:  "pages/contact/alice-smith.md",
		Slug:  "contact/alice-smith",
		Href:  "/contact/alice-smith/",
		Title: &aliceTitle,
		Tags:  []string{"contact", "team"},
		Extra: map[string]interface{}{
			"handle":  "alice",
			"aliases": []interface{}{"alices", "asmith"},
		},
	}

	bobTitle := "Bob Jones"
	post2 := &models.Post{
		Path:  "pages/contact/bob-jones.md",
		Slug:  "contact/bob-jones",
		Href:  "/contact/bob-jones/",
		Title: &bobTitle,
		Tags:  []string{"contact"},
		Extra: map[string]interface{}{
			"handle": "bob",
		},
	}

	// Add a non-contact post that shouldn't match
	otherTitle := "Other Post"
	post3 := &models.Post{
		Path:  "posts/other.md",
		Slug:  "other",
		Href:  "/other/",
		Title: &otherTitle,
		Tags:  []string{"blog"},
		Extra: map[string]interface{}{
			"handle": "other",
		},
	}

	m.AddPost(post1)
	m.AddPost(post2)
	m.AddPost(post3)

	handleMap := p.buildHandleMap(m)

	// Check alice's handle
	if entry, ok := handleMap["alice"]; !ok {
		t.Error("expected 'alice' in handleMap")
	} else {
		if entry.SiteURL != "/contact/alice-smith/" {
			t.Errorf("alice.SiteURL = %q, want %q", entry.SiteURL, "/contact/alice-smith/")
		}
		if entry.Title != "Alice Smith" {
			t.Errorf("alice.Title = %q, want %q", entry.Title, "Alice Smith")
		}
	}

	// Check alice's aliases
	if _, ok := handleMap["alices"]; !ok {
		t.Error("expected alias 'alices' in handleMap")
	}
	if _, ok := handleMap["asmith"]; !ok {
		t.Error("expected alias 'asmith' in handleMap")
	}

	// Check bob's handle
	if entry, ok := handleMap["bob"]; !ok {
		t.Error("expected 'bob' in handleMap")
	} else if entry.SiteURL != "/contact/bob-jones/" {
		t.Errorf("bob.SiteURL = %q, want %q", entry.SiteURL, "/contact/bob-jones/")
	}

	// Check that 'other' is NOT in the map (didn't match filter)
	if _, ok := handleMap["other"]; ok {
		t.Error("'other' should not be in handleMap (didn't match filter)")
	}
}

func TestMentionsPlugin_FromPosts_FallbackToSlug(t *testing.T) {
	p := NewMentionsPlugin()
	m := lifecycle.NewManager()

	// Configure mentions with from_posts but no handle_field
	config := m.Config()
	config.Extra = map[string]interface{}{
		"mentions": models.MentionsConfig{
			FromPosts: []models.MentionPostSource{
				{
					Filter: "'team' in tags",
					// No HandleField - should fall back to slug
				},
			},
		},
	}

	// Add a team post without explicit handle
	charlieTitle := "Charlie Brown"
	post := &models.Post{
		Path:  "team/charlie.md",
		Slug:  "charlie",
		Href:  "/team/charlie/",
		Title: &charlieTitle,
		Tags:  []string{"team"},
	}

	m.AddPost(post)

	handleMap := p.buildHandleMap(m)

	// Check that slug is used as handle
	entry, ok := handleMap["charlie"]
	if !ok {
		t.Error("expected 'charlie' (from slug) in handleMap")
	} else if entry.SiteURL != "/team/charlie/" {
		t.Errorf("charlie.SiteURL = %q, want %q", entry.SiteURL, "/team/charlie/")
	}
}

func TestMentionsPlugin_Transform_FromPosts(t *testing.T) {
	p := NewMentionsPlugin()
	m := lifecycle.NewManager()

	// Configure mentions with from_posts
	config := m.Config()
	config.Extra = map[string]interface{}{
		"mentions": models.MentionsConfig{
			CSSClass: "mention",
			FromPosts: []models.MentionPostSource{
				{
					Filter:      "'contact' in tags",
					HandleField: "handle",
				},
			},
		},
	}

	// Add a contact post
	aliceTitle := "Alice Smith"
	contactPost := &models.Post{
		Path:      "pages/contact/alice.md",
		Slug:      "contact/alice",
		Href:      "/contact/alice/",
		Title:     &aliceTitle,
		Tags:      []string{"contact"},
		Published: true,
		Extra: map[string]interface{}{
			"handle": "alice",
		},
	}

	// Add a blog post that mentions alice
	blogTitle := "Working with Alice"
	blogPost := &models.Post{
		Path:      "posts/working-with-alice.md",
		Slug:      "working-with-alice",
		Href:      "/working-with-alice/",
		Title:     &blogTitle,
		Tags:      []string{"blog"},
		Published: true,
		Content:   "I recently collaborated with @alice on a project.",
	}

	m.AddPost(contactPost)
	m.AddPost(blogPost)

	// Run transform
	err := p.Transform(m)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	// Check the blog post content was transformed
	posts := m.Posts()
	var transformedPost *models.Post
	for _, post := range posts {
		if post.Slug == "working-with-alice" {
			transformedPost = post
			break
		}
	}

	if transformedPost == nil {
		t.Fatal("could not find transformed blog post")
	}

	want := `I recently collaborated with <a href="/contact/alice/" class="mention">@alice</a> on a project.`
	if transformedPost.Content != want {
		t.Errorf("Content = %q, want %q", transformedPost.Content, want)
	}
}

func TestMentionsPlugin_CombinedSources(t *testing.T) {
	p := NewMentionsPlugin()
	m := lifecycle.NewManager()

	// Configure both blogroll and from_posts
	boolTrue := true
	config := m.Config()
	config.Extra = map[string]interface{}{
		"blogroll": models.BlogrollConfig{
			Enabled: true,
			Feeds: []models.ExternalFeedConfig{
				{
					URL:     "https://external.example.com/feed.xml",
					Title:   "External Blog",
					Handle:  "external",
					SiteURL: "https://external.example.com",
					Active:  &boolTrue,
				},
			},
		},
		"mentions": models.MentionsConfig{
			FromPosts: []models.MentionPostSource{
				{
					Filter:      "'contact' in tags",
					HandleField: "handle",
				},
			},
		},
	}

	// Add an internal contact post
	aliceTitle := "Alice Smith"
	contactPost := &models.Post{
		Path:  "contact/alice.md",
		Slug:  "contact/alice",
		Href:  "/contact/alice/",
		Title: &aliceTitle,
		Tags:  []string{"contact"},
		Extra: map[string]interface{}{
			"handle": "alice",
		},
	}

	m.AddPost(contactPost)

	handleMap := p.buildHandleMap(m)

	// Check external handle from blogroll
	if entry, ok := handleMap["external"]; !ok {
		t.Error("expected 'external' from blogroll in handleMap")
	} else if entry.SiteURL != "https://external.example.com" {
		t.Errorf("external.SiteURL = %q, want %q", entry.SiteURL, "https://external.example.com")
	}

	// Check internal handle from from_posts
	if entry, ok := handleMap["alice"]; !ok {
		t.Error("expected 'alice' from from_posts in handleMap")
	} else if entry.SiteURL != "/contact/alice/" {
		t.Errorf("alice.SiteURL = %q, want %q", entry.SiteURL, "/contact/alice/")
	}
}
