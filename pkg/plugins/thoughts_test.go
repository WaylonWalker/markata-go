package plugins

import (
	"fmt"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestThoughtsPlugin_Name(t *testing.T) {
	plugin := NewThoughtsPlugin()
	expected := "thoughts"
	if got := plugin.Name(); got != expected {
		t.Errorf("Name() = %q, want %q", got, expected)
	}
}

func TestThoughtsPlugin_Configure(t *testing.T) {
	plugin := NewThoughtsPlugin()
	manager := lifecycle.NewManager()

	// Test with empty config
	if err := plugin.Configure(manager); err != nil {
		t.Errorf("Configure() with empty config returned error: %v", err)
	}

	// Test default values
	if !plugin.enabled {
		t.Error("Plugin should be enabled by default")
	}
	if plugin.thoughtsDir != "thoughts" {
		t.Errorf("Default thoughts_dir = %q, want %q", plugin.thoughtsDir, "thoughts")
	}
	if plugin.maxItems != 200 {
		t.Errorf("Default max_items = %d, want %d", plugin.maxItems, 200)
	}
}

func TestThoughtsPlugin_ConfigureWithCustomSettings(t *testing.T) {
	plugin := NewThoughtsPlugin()
	manager := lifecycle.NewManager()

	// Add custom configuration
	manager.Config().Extra = map[string]interface{}{
		"thoughts": map[string]interface{}{
			"enabled":      false,
			"thoughts_dir": "micro",
			"max_items":    100,
			"cache_dir":    "cache/micro",
		},
	}

	if err := plugin.Configure(manager); err != nil {
		t.Errorf("Configure() with custom config returned error: %v", err)
	}

	if plugin.enabled {
		t.Error("Plugin should be disabled when configured so")
	}
	if plugin.thoughtsDir != "micro" {
		t.Errorf("Custom thoughts_dir = %q, want %q", plugin.thoughtsDir, "micro")
	}
	if plugin.maxItems != 100 {
		t.Errorf("Custom max_items = %d, want %d", plugin.maxItems, 100)
	}
	if plugin.cacheDir != "cache/micro" {
		t.Errorf("Custom cache_dir = %q, want %q", plugin.cacheDir, "cache/micro")
	}
}

func TestThoughtsPlugin_ConfigureWithSources(t *testing.T) {
	plugin := NewThoughtsPlugin()
	manager := lifecycle.NewManager()

	// Add sources configuration
	manager.Config().Extra = map[string]interface{}{
		"thoughts": map[string]interface{}{
			"sources": map[string]interface{}{
				"mastodon": map[string]interface{}{
					"type":      "mastodon",
					"url":       "https://mastodon.social/@user.rss",
					"handle":    "user",
					"active":    true,
					"max_items": 25,
				},
				"twitter": map[string]interface{}{
					"type":      "twitter",
					"handle":    "user",
					"active":    false,
					"max_items": 50,
				},
			},
		},
	}

	if err := plugin.Configure(manager); err != nil {
		t.Errorf("Configure() with sources returned error: %v", err)
	}

	// Check mastodon source
	mastodon, ok := plugin.sources["mastodon"]
	if !ok {
		t.Fatal("Mastodon source not found")
	}
	if mastodon.Type != "mastodon" {
		t.Errorf("Mastodon type = %q, want %q", mastodon.Type, "mastodon")
	}
	if mastodon.URL != "https://mastodon.social/@user.rss" {
		t.Errorf("Mastodon URL = %q, want %q", mastodon.URL, "https://mastodon.social/@user.rss")
	}
	if mastodon.Handle != "user" {
		t.Errorf("Mastodon handle = %q, want %q", mastodon.Handle, "user")
	}
	if !mastodon.Active {
		t.Error("Mastodon source should be active")
	}
	if mastodon.MaxItems != 25 {
		t.Errorf("Mastodon max_items = %d, want %d", mastodon.MaxItems, 25)
	}

	// Check twitter source
	twitter, ok := plugin.sources["twitter"]
	if !ok {
		t.Fatal("Twitter source not found")
	}
	if twitter.Type != "twitter" {
		t.Errorf("Twitter type = %q, want %q", twitter.Type, "twitter")
	}
	if twitter.Active {
		t.Error("Twitter source should be inactive")
	}
}

func TestThoughtsPlugin_ConfigureWithSyndication(t *testing.T) {
	plugin := NewThoughtsPlugin()
	manager := lifecycle.NewManager()

	// Add syndication configuration
	manager.Config().Extra = map[string]interface{}{
		"thoughts": map[string]interface{}{
			"syndication": map[string]interface{}{
				"enabled": true,
				"mastodon": map[string]interface{}{
					"access_token":    "token123",
					"instance_url":    "https://mastodon.social",
					"character_limit": 500,
				},
			},
		},
	}

	if err := plugin.Configure(manager); err != nil {
		t.Errorf("Configure() with syndication returned error: %v", err)
	}

	if !plugin.syndication.Enabled {
		t.Error("Syndication should be enabled")
	}
	if plugin.syndication.Mastodon == nil {
		t.Fatal("Mastodon syndication config not loaded")
	}
	if plugin.syndication.Mastodon.AccessToken != "token123" {
		t.Errorf("Mastodon access_token = %q, want %q", plugin.syndication.Mastodon.AccessToken, "token123")
	}
	if plugin.syndication.Mastodon.InstanceURL != "https://mastodon.social" {
		t.Errorf("Mastodon instance_url = %q, want %q", plugin.syndication.Mastodon.InstanceURL, "https://mastodon.social")
	}
	if plugin.syndication.Mastodon.CharacterLimit != 500 {
		t.Errorf("Mastodon character_limit = %d, want %d", plugin.syndication.Mastodon.CharacterLimit, 500)
	}
}

func TestConvertExternalEntryToPost(t *testing.T) {
	plugin := NewThoughtsPlugin()

	// Create test external entry
	published := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	entry := &models.ExternalEntry{
		ID:          "123",
		Title:       "Test Thought",
		Description: "A test thought from external source",
		Content:     "This is the full content of the test thought.",
		Author:      "testuser",
		Published:   &published,
		Categories:  []string{"test", "thought"},
		ImageURL:    "https://example.com/image.jpg",
	}

	// Create test source
	source := &ThoughtSource{
		Type:   "mastodon",
		Handle: "testuser",
	}

	// Convert to post
	post := plugin.convertExternalEntryToPost(entry, source, "testsource")

	if post == nil {
		t.Fatal("convertExternalEntryToPost returned nil")
	}

	// Check basic fields
	if post.Title == nil || *post.Title != "Test Thought" {
		t.Errorf("Title = %v, want %q", post.Title, "Test Thought")
	}
	if post.Template != "thought.html" {
		t.Errorf("Template = %q, want %q", post.Template, "thought.html")
	}
	if !post.Published {
		t.Error("Post should be published")
	}
	if post.Date == nil || !post.Date.Equal(published) {
		t.Errorf("Date = %v, want %v", post.Date, published)
	}

	// Check metadata
	if post.Get("thought_source") != "testsource" {
		t.Errorf("thought_source = %v, want %q", post.Get("thought_source"), "testsource")
	}
	if post.Get("thought_type") != "mastodon" {
		t.Errorf("thought_type = %v, want %q", post.Get("thought_type"), "mastodon")
	}
	if post.Get("source_handle") != "testuser" {
		t.Errorf("source_handle = %v, want %q", post.Get("source_handle"), "testuser")
	}
	if post.Get("external_id") != "123" {
		t.Errorf("external_id = %v, want %q", post.Get("external_id"), "123")
	}

	// Check tags
	expectedTags := []string{"mastodon", "test", "thought"}
	if len(post.Tags) != len(expectedTags) {
		t.Errorf("Tags length = %d, want %d", len(post.Tags), len(expectedTags))
	} else {
		for i, tag := range expectedTags {
			if i >= len(post.Tags) || post.Tags[i] != tag {
				t.Errorf("Tag[%d] = %q, want %q", i, post.Tags[i], tag)
			}
		}
	}

	// Check image URL
	if post.Get("image_url") != "https://example.com/image.jpg" {
		t.Errorf("image_url = %v, want %q", post.Get("image_url"), "https://example.com/image.jpg")
	}
}

func TestGenerateThoughtSlug(t *testing.T) {
	plugin := NewThoughtsPlugin()

	// Use a fixed timestamp for predictable results
	fixedTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	fixedUnix := fixedTime.Unix()

	tests := []struct {
		name       string
		entry      *models.ExternalEntry
		sourceName string
		want       string
	}{
		{
			name: "basic title",
			entry: &models.ExternalEntry{
				ID:        "123",
				Title:     "Test Title",
				Published: &fixedTime,
			},
			sourceName: "mastodon",
			want:       "test-title-mastodon-" + fmt.Sprintf("%d", fixedUnix),
		},
		{
			name: "title with special chars",
			entry: &models.ExternalEntry{
				ID:        "456",
				Title:     "Hello, World! This is a test",
				Published: &fixedTime,
			},
			sourceName: "twitter",
			want:       "hello-world-this-is-a-test-twitter-" + fmt.Sprintf("%d", fixedUnix),
		},
		{
			name: "long title gets truncated",
			entry: &models.ExternalEntry{
				ID:        "789",
				Title:     "This is a very long title that should be truncated to fifty characters exactly",
				Published: &fixedTime,
			},
			sourceName: "blog",
			want:       "this-is-a-very-long-title-that-should-be-truncated-blog-" + fmt.Sprintf("%d", fixedUnix),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := plugin.generateThoughtSlug(tt.entry, tt.sourceName)
			if got != tt.want {
				t.Errorf("generateThoughtSlug() = %q, want %q", got, tt.want)
				t.Logf("Title length: %d, title: %q", len(tt.entry.Title), tt.entry.Title)
			}
		})
	}
}

func TestNormalizeThoughtContent(t *testing.T) {
	plugin := NewThoughtsPlugin()

	tests := []struct {
		name       string
		content    string
		sourceType string
		want       string
	}{
		{
			name:       "plain text unchanged",
			content:    "This is plain text",
			sourceType: "mastodon",
			want:       "This is plain text",
		},
		{
			name:       "HTML tags stripped for mastodon",
			content:    "<p>This is <br/> HTML content</p>",
			sourceType: "mastodon",
			want:       "This is \n HTML content",
		},
		{
			name:       "long content truncated",
			content:    string(make([]byte, 600)), // 600 chars
			sourceType: "twitter",
			want:       string(make([]byte, 497)) + "...",
		},
		{
			name:       "whitespace trimmed",
			content:    "   \n\t  This content has whitespace  \n\t  ",
			sourceType: "rss",
			want:       "This content has whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := plugin.normalizeThoughtContent(tt.content, tt.sourceType)
			if got != tt.want {
				t.Errorf("normalizeThoughtContent() = %q, want %q", got, tt.want)
			}
		})
	}
}
