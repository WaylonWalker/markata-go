package models

import (
	"time"
)

// MentionMetadata holds fetched metadata for a mention domain.
// This is cached between builds to avoid repeated network requests.
type MentionMetadata struct {
	// Domain is the normalized domain name (e.g., "example.com")
	Domain string `json:"domain"`

	// Name is the site name from og:site_name or title tag
	Name string `json:"name"`

	// Bio is the site description from og:description or meta description
	Bio string `json:"bio"`

	// Avatar is the avatar/image URL from og:image or icon
	Avatar string `json:"avatar"`

	// URL is the original URL that was fetched
	URL string `json:"url"`

	// LastFetched is when the metadata was last fetched
	LastFetched time.Time `json:"last_fetched"`

	// Error contains any error that occurred during fetching (empty if successful)
	Error string `json:"error,omitempty"`
}

// IsExpired checks if the cached metadata is older than the given duration.
func (m *MentionMetadata) IsExpired(maxAge time.Duration) bool {
	return time.Since(m.LastFetched) > maxAge
}

// IsValid returns true if the metadata was successfully fetched (no error).
func (m *MentionMetadata) IsValid() bool {
	return m.Error == ""
}

// MentionsConfig configures the @mentions resolution plugin.
type MentionsConfig struct {
	// Enabled controls whether mentions processing is active (default: true)
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty" toml:"enabled,omitempty"`

	// CSSClass is the CSS class applied to mention links (default: "mention")
	CSSClass string `json:"css_class,omitempty" yaml:"css_class,omitempty" toml:"css_class,omitempty"`

	// FromPosts configures mention sources from internal posts
	FromPosts []MentionPostSource `json:"from_posts,omitempty" yaml:"from_posts,omitempty" toml:"from_posts,omitempty"`

	// CacheDir is the directory for caching fetched mention metadata (default: "cache/mentions")
	CacheDir string `json:"cache_dir,omitempty" yaml:"cache_dir,omitempty" toml:"cache_dir,omitempty"`

	// CacheDuration is how long to cache fetched metadata (default: "24h")
	// Mentions change less frequently than RSS feeds, so use longer cache
	CacheDuration string `json:"cache_duration,omitempty" yaml:"cache_duration,omitempty" toml:"cache_duration,omitempty"`

	// Timeout is the HTTP request timeout in seconds for metadata fetching (default: 30)
	Timeout int `json:"timeout,omitempty" yaml:"timeout,omitempty" toml:"timeout,omitempty"`

	// ConcurrentRequests is the max concurrent metadata fetches (default: 3)
	// Lower than blogroll to be more respectful to external sites
	ConcurrentRequests int `json:"concurrent_requests,omitempty" yaml:"concurrent_requests,omitempty" toml:"concurrent_requests,omitempty"`
}

// MentionPostSource configures a source of @mentions from internal posts.
// This allows resolving @handles from posts like contact pages or team member pages.
type MentionPostSource struct {
	// Filter is a filter expression to select which posts to extract handles from
	// Example: "'contact' in tags" or "template == 'team-member.html'"
	Filter string `json:"filter" yaml:"filter" toml:"filter"`

	// HandleField is the frontmatter field containing the handle (default: uses slug)
	// Example: "handle" for frontmatter like `handle: alice`
	HandleField string `json:"handle_field,omitempty" yaml:"handle_field,omitempty" toml:"handle_field,omitempty"`

	// AliasesField is the frontmatter field containing handle aliases (defaults to "aliases")
	// Example: "aliases" for frontmatter like `aliases: [alices, asmith]`
	AliasesField string `json:"aliases_field,omitempty" yaml:"aliases_field,omitempty" toml:"aliases_field,omitempty"`

	// AvatarField is the frontmatter field containing the avatar/image URL (optional).
	// If not set, looks for "avatar", "image", and "icon" fields in order.
	// This is used for mention hovercard display.
	AvatarField string `json:"avatar_field,omitempty" yaml:"avatar_field,omitempty" toml:"avatar_field,omitempty"`
}

// NewMentionsConfig creates a new MentionsConfig with default values.
func NewMentionsConfig() MentionsConfig {
	enabled := true
	return MentionsConfig{
		Enabled:  &enabled,
		CSSClass: "mention",
		FromPosts: []MentionPostSource{
			{
				Filter:       "template == 'contact'",
				HandleField:  "handle",
				AliasesField: "aliases",
			},
			{
				Filter:       "template == 'author'",
				HandleField:  "handle",
				AliasesField: "aliases",
			},
		},
		CacheDir:           "cache/mentions",
		CacheDuration:      "24h",
		Timeout:            30,
		ConcurrentRequests: 3,
	}
}

// IsEnabled returns whether mentions processing is enabled.
// Defaults to true if not explicitly set.
func (m *MentionsConfig) IsEnabled() bool {
	if m.Enabled == nil {
		return true
	}
	return *m.Enabled
}

// GetCSSClass returns the CSS class for mention links.
// Defaults to "mention" if not set.
func (m *MentionsConfig) GetCSSClass() string {
	if m.CSSClass == "" {
		return "mention"
	}
	return m.CSSClass
}

// GetCacheDir returns the cache directory for mention metadata.
// Defaults to "cache/mentions" if not set.
func (m *MentionsConfig) GetCacheDir() string {
	if m.CacheDir == "" {
		return "cache/mentions"
	}
	return m.CacheDir
}

// GetCacheDuration returns the cache duration for mention metadata.
// Defaults to "24h" if not set.
func (m *MentionsConfig) GetCacheDuration() string {
	if m.CacheDuration == "" {
		return "24h"
	}
	return m.CacheDuration
}

// GetTimeout returns the HTTP timeout for metadata fetching.
// Defaults to 30 seconds if not set.
func (m *MentionsConfig) GetTimeout() int {
	if m.Timeout == 0 {
		return 30
	}
	return m.Timeout
}

// GetConcurrentRequests returns the max concurrent metadata fetches.
// Defaults to 3 if not set.
func (m *MentionsConfig) GetConcurrentRequests() int {
	if m.ConcurrentRequests == 0 {
		return 3
	}
	return m.ConcurrentRequests
}
