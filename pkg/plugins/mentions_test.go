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
