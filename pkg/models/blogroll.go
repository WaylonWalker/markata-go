package models

import "time"

// BlogrollConfig configures the blogroll and RSS reader functionality.
type BlogrollConfig struct {
	// Enabled controls whether blogroll functionality is active (default: false)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// BlogrollSlug is the URL path for the blogroll page (default: "blogroll")
	// This generates the page at /{blogroll_slug}/index.html
	BlogrollSlug string `json:"blogroll_slug" yaml:"blogroll_slug" toml:"blogroll_slug"`

	// ReaderSlug is the URL path for the reader page (default: "reader")
	// This generates the page at /{reader_slug}/index.html
	ReaderSlug string `json:"reader_slug" yaml:"reader_slug" toml:"reader_slug"`

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

	// ItemsPerPage is the number of entries per page on the reader page (default: 50)
	ItemsPerPage int `json:"items_per_page" yaml:"items_per_page" toml:"items_per_page"`

	// OrphanThreshold is the minimum entries for a separate page (default: 3)
	OrphanThreshold int `json:"orphan_threshold" yaml:"orphan_threshold" toml:"orphan_threshold"`

	// PaginationType specifies the pagination strategy (manual, htmx, js)
	PaginationType PaginationType `json:"pagination_type" yaml:"pagination_type" toml:"pagination_type"`

	// FallbackImageService is an optional URL template for generating fallback images
	// for entries without images. Use {url} as placeholder for the entry URL.
	// Example: "https://shots.example.com/shot/?url={url}&width=1200"
	// If empty, no fallback images are generated (default: "")
	FallbackImageService string `json:"fallback_image_service" yaml:"fallback_image_service" toml:"fallback_image_service"`

	// Feeds is the list of RSS/Atom feeds to fetch
	Feeds []ExternalFeedConfig `json:"feeds" yaml:"feeds" toml:"feeds"`

	// Templates configures custom templates for blogroll pages
	Templates BlogrollTemplates `json:"templates" yaml:"templates" toml:"templates"`
}

// NewBlogrollConfig creates a new BlogrollConfig with default values.
func NewBlogrollConfig() BlogrollConfig {
	return BlogrollConfig{
		Enabled:              false,
		BlogrollSlug:         "blogroll",
		ReaderSlug:           "reader",
		CacheDir:             "cache/blogroll",
		CacheDuration:        "1h",
		Timeout:              30,
		ConcurrentRequests:   5,
		MaxEntriesPerFeed:    50,
		ItemsPerPage:         50,
		OrphanThreshold:      3,
		PaginationType:       PaginationManual,
		FallbackImageService: "",
		Feeds:                []ExternalFeedConfig{},
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

	// Handle is an optional explicit handle for @mentions (e.g., "daverupert")
	// If not set, a handle is auto-generated from the domain
	Handle string `json:"handle" yaml:"handle" toml:"handle"`

	// Aliases are alternative names that resolve to the canonical Handle.
	// For example, aliases = ["dave", "david"] allows @dave or @david to
	// resolve to the same person as @daverupert. Case-insensitive.
	Aliases []string `json:"aliases,omitempty" yaml:"aliases,omitempty" toml:"aliases,omitempty"`

	// MaxEntries overrides the global max_entries_per_feed for this feed
	MaxEntries *int `json:"max_entries,omitempty" yaml:"max_entries,omitempty" toml:"max_entries,omitempty"`

	// Primary marks this as the canonical/primary feed for a person (default: true for entries without PrimaryPerson)
	// When a person has multiple feeds, mark the main one as primary=true
	Primary *bool `json:"primary,omitempty" yaml:"primary,omitempty" toml:"primary,omitempty"`

	// PrimaryPerson links this feed to a primary person's handle
	// Use this to associate secondary feeds (e.g., social accounts) with a primary person
	// Example: If "daverupert" is the primary handle, set primary_person="daverupert" on secondary feeds
	PrimaryPerson string `json:"primary_person" yaml:"primary_person" toml:"primary_person"`
}

// IsActive returns whether the feed is active (defaults to true).
func (f *ExternalFeedConfig) IsActive() bool {
	if f.Active == nil {
		return true
	}
	return *f.Active
}

// GetMaxEntries returns the per-feed max_entries if set, otherwise the global default.
func (f *ExternalFeedConfig) GetMaxEntries(globalDefault int) int {
	if f.MaxEntries != nil {
		return *f.MaxEntries
	}
	return globalDefault
}

// IsPrimary returns whether this is a primary feed (defaults to true if PrimaryPerson is not set).
func (f *ExternalFeedConfig) IsPrimary() bool {
	// If explicit primary value is set, use it
	if f.Primary != nil {
		return *f.Primary
	}
	// If linked to a primary person, this is secondary (not primary)
	if f.PrimaryPerson != "" {
		return false
	}
	// Default to primary
	return true
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

	// ImageURL is the feed's logo/icon (general image, may be article image)
	ImageURL string `json:"image_url"`

	// AvatarURL is the author/site representative image (person avatar)
	// Discovered via h-card u-photo, WebFinger rel=avatar, or .well-known/avatar
	AvatarURL string `json:"avatar_url,omitempty"`

	// AvatarSource indicates where the avatar was discovered from
	// Possible values: "config", "h-card", "webfinger", "well-known", "feed", "opengraph", "favicon"
	AvatarSource string `json:"avatar_source,omitempty"`

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

// ReaderPage represents a single page of paginated reader entries.
type ReaderPage struct {
	// Number is the page number (1-indexed)
	Number int `json:"number" yaml:"number" toml:"number"`

	// Entries is the list of entries on this page
	Entries []*ExternalEntry `json:"entries" yaml:"entries" toml:"entries"`

	// HasPrev indicates if there is a previous page
	HasPrev bool `json:"has_prev" yaml:"has_prev" toml:"has_prev"`

	// HasNext indicates if there is a next page
	HasNext bool `json:"has_next" yaml:"has_next" toml:"has_next"`

	// PrevURL is the URL of the previous page
	PrevURL string `json:"prev_url" yaml:"prev_url" toml:"prev_url"`

	// NextURL is the URL of the next page
	NextURL string `json:"next_url" yaml:"next_url" toml:"next_url"`

	// TotalPages is the total number of pages
	TotalPages int `json:"total_pages" yaml:"total_pages" toml:"total_pages"`

	// TotalItems is the total number of entries
	TotalItems int `json:"total_items" yaml:"total_items" toml:"total_items"`

	// ItemsPerPage is the number of entries per page
	ItemsPerPage int `json:"items_per_page" yaml:"items_per_page" toml:"items_per_page"`

	// PageURLs contains URLs for all pages (for numbered pagination)
	PageURLs []string `json:"page_urls" yaml:"page_urls" toml:"page_urls"`

	// PaginationType is the pagination strategy used
	PaginationType PaginationType `json:"pagination_type" yaml:"pagination_type" toml:"pagination_type"`
}
