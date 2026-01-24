package models

import "time"

// BlogrollConfig configures the blogroll and RSS reader functionality.
type BlogrollConfig struct {
	// Enabled controls whether blogroll functionality is active (default: false)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// CacheDir is the directory for caching fetched feeds (default: "cache/blogroll")
	CacheDir string `json:"cache_dir" yaml:"cache_dir" toml:"cache_dir"`

	// CacheDuration is how long to cache fetched feeds (default: "1h")
	CacheDuration string `json:"cache_duration" yaml:"cache_duration" toml:"cache_duration"`

	// Timeout is the HTTP request timeout in seconds (default: 30)
	Timeout int `json:"timeout" yaml:"timeout" toml:"timeout"`

	// ConcurrentRequests is the max concurrent feed fetches (default: 5)
	ConcurrentRequests int `json:"concurrent_requests" yaml:"concurrent_requests" toml:"concurrent_requests"`

	// MaxEntriesPerFeed limits entries fetched per feed (default: 50)
	MaxEntriesPerFeed int `json:"max_entries_per_feed" yaml:"max_entries_per_feed" toml:"max_entries_per_feed"`

	// Feeds is the list of RSS/Atom feeds to fetch
	Feeds []ExternalFeedConfig `json:"feeds" yaml:"feeds" toml:"feeds"`

	// Templates configures custom templates for blogroll pages
	Templates BlogrollTemplates `json:"templates" yaml:"templates" toml:"templates"`
}

// NewBlogrollConfig creates a new BlogrollConfig with default values.
func NewBlogrollConfig() BlogrollConfig {
	return BlogrollConfig{
		Enabled:            false,
		CacheDir:           "cache/blogroll",
		CacheDuration:      "1h",
		Timeout:            30,
		ConcurrentRequests: 5,
		MaxEntriesPerFeed:  50,
		Feeds:              []ExternalFeedConfig{},
		Templates: BlogrollTemplates{
			Blogroll: "blogroll.html",
			Reader:   "reader.html",
		},
	}
}

// BlogrollTemplates specifies custom templates for blogroll pages.
type BlogrollTemplates struct {
	// Blogroll is the template for the /blogroll page
	Blogroll string `json:"blogroll" yaml:"blogroll" toml:"blogroll"`

	// Reader is the template for the /reader page
	Reader string `json:"reader" yaml:"reader" toml:"reader"`
}

// ExternalFeedConfig represents a configured external RSS/Atom feed.
type ExternalFeedConfig struct {
	// URL is the feed URL (required)
	URL string `json:"url" yaml:"url" toml:"url"`

	// Title is the human-readable feed title (optional, fetched if not set)
	Title string `json:"title" yaml:"title" toml:"title"`

	// Description is a short description of the feed
	Description string `json:"description" yaml:"description" toml:"description"`

	// Category groups feeds together (e.g., "technology", "design")
	Category string `json:"category" yaml:"category" toml:"category"`

	// Tags are additional labels for filtering
	Tags []string `json:"tags" yaml:"tags" toml:"tags"`

	// Active controls whether this feed is fetched (default: true)
	Active *bool `json:"active,omitempty" yaml:"active,omitempty" toml:"active,omitempty"`

	// SiteURL is the main website URL (fetched from feed if not set)
	SiteURL string `json:"site_url" yaml:"site_url" toml:"site_url"`

	// ImageURL is a logo or icon for the feed
	ImageURL string `json:"image_url" yaml:"image_url" toml:"image_url"`
}

// IsActive returns whether the feed is active (defaults to true).
func (f *ExternalFeedConfig) IsActive() bool {
	if f.Active == nil {
		return true
	}
	return *f.Active
}

// ExternalFeed represents a fetched and parsed external RSS/Atom feed.
type ExternalFeed struct {
	// Config is the original configuration for this feed
	Config ExternalFeedConfig `json:"config"`

	// Title is the feed title (from config or fetched)
	Title string `json:"title"`

	// Description is the feed description
	Description string `json:"description"`

	// SiteURL is the main website URL
	SiteURL string `json:"site_url"`

	// FeedURL is the RSS/Atom feed URL
	FeedURL string `json:"feed_url"`

	// ImageURL is the feed's logo/icon
	ImageURL string `json:"image_url"`

	// Category is the feed category
	Category string `json:"category"`

	// Tags are the feed tags
	Tags []string `json:"tags"`

	// LastFetched is when the feed was last fetched
	LastFetched *time.Time `json:"last_fetched,omitempty"`

	// LastUpdated is the feed's last build/update date
	LastUpdated *time.Time `json:"last_updated,omitempty"`

	// EntryCount is the number of entries in the feed
	EntryCount int `json:"entry_count"`

	// Entries are the feed entries/items
	Entries []*ExternalEntry `json:"entries"`

	// Error holds any error that occurred during fetching
	Error string `json:"error,omitempty"`
}

// ExternalEntry represents a single entry/item from an external feed.
type ExternalEntry struct {
	// FeedURL is the source feed URL
	FeedURL string `json:"feed_url"`

	// FeedTitle is the source feed title
	FeedTitle string `json:"feed_title"`

	// ID is the unique identifier (GUID) for the entry
	ID string `json:"id"`

	// URL is the link to the full article
	URL string `json:"url"`

	// Title is the entry title
	Title string `json:"title"`

	// Description is a short summary or excerpt
	Description string `json:"description"`

	// Content is the full entry content (HTML)
	Content string `json:"content"`

	// Author is the entry author
	Author string `json:"author,omitempty"`

	// Published is the publication date
	Published *time.Time `json:"published,omitempty"`

	// Updated is the last update date
	Updated *time.Time `json:"updated,omitempty"`

	// Categories are the entry categories/tags
	Categories []string `json:"categories,omitempty"`

	// ImageURL is the entry's featured image
	ImageURL string `json:"image_url,omitempty"`

	// ReadingTime is estimated reading time in minutes
	ReadingTime int `json:"reading_time"`
}

// BlogrollCategory groups feeds by category for display.
type BlogrollCategory struct {
	// Name is the category name
	Name string `json:"name"`

	// Slug is the URL-safe category identifier
	Slug string `json:"slug"`

	// Feeds are the feeds in this category
	Feeds []*ExternalFeed `json:"feeds"`
}
