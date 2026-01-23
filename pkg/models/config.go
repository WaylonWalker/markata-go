package models

// NavItem represents a navigation link.
type NavItem struct {
	// Label is the display text for the nav link
	Label string `json:"label" yaml:"label" toml:"label"`

	// URL is the link destination (can be relative or absolute)
	URL string `json:"url" yaml:"url" toml:"url"`

	// External indicates if the link opens in a new tab (default: false)
	External bool `json:"external,omitempty" yaml:"external,omitempty" toml:"external,omitempty"`
}

// FooterConfig configures the site footer.
type FooterConfig struct {
	// Text is the footer text (supports template variables like {{ year }})
	Text string `json:"text,omitempty" yaml:"text,omitempty" toml:"text,omitempty"`

	// ShowCopyright shows the copyright line (default: true)
	ShowCopyright *bool `json:"show_copyright,omitempty" yaml:"show_copyright,omitempty" toml:"show_copyright,omitempty"`
}

// ComponentsConfig configures the layout components system.
// This enables configuration-driven control over common UI elements.
type ComponentsConfig struct {
	// Nav configures the navigation component
	Nav NavComponentConfig `json:"nav" yaml:"nav" toml:"nav"`

	// Footer configures the footer component
	Footer FooterComponentConfig `json:"footer" yaml:"footer" toml:"footer"`

	// DocSidebar configures the document sidebar (table of contents)
	DocSidebar DocSidebarConfig `json:"doc_sidebar" yaml:"doc_sidebar" toml:"doc_sidebar"`

	// FeedSidebar configures the feed sidebar (series/collection navigation)
	FeedSidebar FeedSidebarConfig `json:"feed_sidebar" yaml:"feed_sidebar" toml:"feed_sidebar"`
}

// NavComponentConfig configures the navigation component.
type NavComponentConfig struct {
	// Enabled controls whether navigation is displayed (default: true)
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty" toml:"enabled,omitempty"`

	// Position controls where navigation appears: "header", "sidebar" (default: "header")
	Position string `json:"position,omitempty" yaml:"position,omitempty" toml:"position,omitempty"`

	// Style controls the navigation style: "horizontal", "vertical" (default: "horizontal")
	Style string `json:"style,omitempty" yaml:"style,omitempty" toml:"style,omitempty"`

	// Items are the navigation links (overrides top-level nav if set)
	Items []NavItem `json:"items,omitempty" yaml:"items,omitempty" toml:"items,omitempty"`
}

// FooterComponentConfig configures the footer component.
type FooterComponentConfig struct {
	// Enabled controls whether footer is displayed (default: true)
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty" toml:"enabled,omitempty"`

	// Text is the footer text (supports template variables)
	Text string `json:"text,omitempty" yaml:"text,omitempty" toml:"text,omitempty"`

	// ShowCopyright shows the copyright line (default: true)
	ShowCopyright *bool `json:"show_copyright,omitempty" yaml:"show_copyright,omitempty" toml:"show_copyright,omitempty"`

	// Links are additional footer links
	Links []NavItem `json:"links,omitempty" yaml:"links,omitempty" toml:"links,omitempty"`
}

// DocSidebarConfig configures the document sidebar (table of contents).
type DocSidebarConfig struct {
	// Enabled controls whether the TOC sidebar is displayed (default: false)
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty" toml:"enabled,omitempty"`

	// Position controls sidebar position: "left", "right" (default: "right")
	Position string `json:"position,omitempty" yaml:"position,omitempty" toml:"position,omitempty"`

	// Width is the sidebar width (default: "250px")
	Width string `json:"width,omitempty" yaml:"width,omitempty" toml:"width,omitempty"`

	// MinDepth is the minimum heading level to include (default: 2)
	MinDepth int `json:"min_depth,omitempty" yaml:"min_depth,omitempty" toml:"min_depth,omitempty"`

	// MaxDepth is the maximum heading level to include (default: 4)
	MaxDepth int `json:"max_depth,omitempty" yaml:"max_depth,omitempty" toml:"max_depth,omitempty"`
}

// FeedSidebarConfig configures the feed sidebar (series/collection navigation).
type FeedSidebarConfig struct {
	// Enabled controls whether the feed sidebar is displayed (default: false)
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty" toml:"enabled,omitempty"`

	// Position controls sidebar position: "left", "right" (default: "left")
	Position string `json:"position,omitempty" yaml:"position,omitempty" toml:"position,omitempty"`

	// Width is the sidebar width (default: "250px")
	Width string `json:"width,omitempty" yaml:"width,omitempty" toml:"width,omitempty"`

	// Title is the sidebar title (default: uses feed title or "In this series")
	Title string `json:"title,omitempty" yaml:"title,omitempty" toml:"title,omitempty"`

	// Feeds is the list of feed slugs to show navigation for
	Feeds []string `json:"feeds,omitempty" yaml:"feeds,omitempty" toml:"feeds,omitempty"`
}

// NewComponentsConfig creates a new ComponentsConfig with default values.
func NewComponentsConfig() ComponentsConfig {
	navEnabled := true
	footerEnabled := true
	docSidebarEnabled := false
	feedSidebarEnabled := false
	showCopyright := true

	return ComponentsConfig{
		Nav: NavComponentConfig{
			Enabled:  &navEnabled,
			Position: "header",
			Style:    "horizontal",
		},
		Footer: FooterComponentConfig{
			Enabled:       &footerEnabled,
			ShowCopyright: &showCopyright,
		},
		DocSidebar: DocSidebarConfig{
			Enabled:  &docSidebarEnabled,
			Position: "right",
			Width:    "250px",
			MinDepth: 2,
			MaxDepth: 4,
		},
		FeedSidebar: FeedSidebarConfig{
			Enabled:  &feedSidebarEnabled,
			Position: "left",
			Width:    "250px",
		},
	}
}

// IsNavEnabled returns whether navigation is enabled.
func (c *ComponentsConfig) IsNavEnabled() bool {
	if c.Nav.Enabled == nil {
		return true
	}
	return *c.Nav.Enabled
}

// IsFooterEnabled returns whether footer is enabled.
func (c *ComponentsConfig) IsFooterEnabled() bool {
	if c.Footer.Enabled == nil {
		return true
	}
	return *c.Footer.Enabled
}

// IsDocSidebarEnabled returns whether the document sidebar is enabled.
func (c *ComponentsConfig) IsDocSidebarEnabled() bool {
	if c.DocSidebar.Enabled == nil {
		return false
	}
	return *c.DocSidebar.Enabled
}

// IsFeedSidebarEnabled returns whether the feed sidebar is enabled.
func (c *ComponentsConfig) IsFeedSidebarEnabled() bool {
	if c.FeedSidebar.Enabled == nil {
		return false
	}
	return *c.FeedSidebar.Enabled
}

// Config represents the site configuration for markata-go.
type Config struct {
	// OutputDir is the directory where generated files are written (default: "output")
	OutputDir string `json:"output_dir" yaml:"output_dir" toml:"output_dir"`

	// URL is the base URL of the site
	URL string `json:"url" yaml:"url" toml:"url"`

	// Title is the site title
	Title string `json:"title" yaml:"title" toml:"title"`

	// Description is the site description
	Description string `json:"description" yaml:"description" toml:"description"`

	// Author is the site author
	Author string `json:"author" yaml:"author" toml:"author"`

	// AssetsDir is the directory containing static assets (default: "static")
	AssetsDir string `json:"assets_dir" yaml:"assets_dir" toml:"assets_dir"`

	// TemplatesDir is the directory containing templates (default: "templates")
	TemplatesDir string `json:"templates_dir" yaml:"templates_dir" toml:"templates_dir"`

	// Nav is the list of navigation links
	Nav []NavItem `json:"nav" yaml:"nav" toml:"nav"`

	// Footer configures the site footer
	Footer FooterConfig `json:"footer" yaml:"footer" toml:"footer"`

	// Hooks is the list of hooks to run (default: ["default"])
	Hooks []string `json:"hooks" yaml:"hooks" toml:"hooks"`

	// DisabledHooks is the list of hooks to disable
	DisabledHooks []string `json:"disabled_hooks" yaml:"disabled_hooks" toml:"disabled_hooks"`

	// GlobConfig configures file globbing behavior
	GlobConfig GlobConfig `json:"glob" yaml:"glob" toml:"glob"`

	// MarkdownConfig configures markdown processing
	MarkdownConfig MarkdownConfig `json:"markdown" yaml:"markdown" toml:"markdown"`

	// Feeds is the list of feed configurations
	Feeds []FeedConfig `json:"feeds" yaml:"feeds" toml:"feeds"`

	// FeedDefaults provides default values for feed configurations
	FeedDefaults FeedDefaults `json:"feed_defaults" yaml:"feed_defaults" toml:"feed_defaults"`

	// Concurrency is the number of concurrent workers (default: 0 = auto)
	Concurrency int `json:"concurrency" yaml:"concurrency" toml:"concurrency"`

	// Theme configures the site theme
	Theme ThemeConfig `json:"theme" yaml:"theme" toml:"theme"`

	// PostFormats configures output formats for individual posts
	PostFormats PostFormatsConfig `json:"post_formats" yaml:"post_formats" toml:"post_formats"`

	// SEO configures SEO metadata generation
	SEO SEOConfig `json:"seo" yaml:"seo" toml:"seo"`

	// IndieAuth configures IndieAuth link tags for identity and authentication
	IndieAuth IndieAuthConfig `json:"indieauth" yaml:"indieauth" toml:"indieauth"`

	// Webmention configures Webmention endpoint for receiving mentions
	Webmention WebmentionConfig `json:"webmention" yaml:"webmention" toml:"webmention"`

	// Components configures layout components (nav, footer, sidebar)
	Components ComponentsConfig `json:"components" yaml:"components" toml:"components"`

	// Head configures elements added to the HTML <head> section
	Head HeadConfig `json:"head" yaml:"head" toml:"head"`
}

// HeadConfig configures elements added to the HTML <head> section.
type HeadConfig struct {
	// Text is raw HTML/text to include in the head (use with caution)
	Text string `json:"text,omitempty" yaml:"text,omitempty" toml:"text,omitempty"`

	// Meta is a list of meta tags to include
	Meta []MetaTag `json:"meta,omitempty" yaml:"meta,omitempty" toml:"meta,omitempty"`

	// Link is a list of link tags to include
	Link []LinkTag `json:"link,omitempty" yaml:"link,omitempty" toml:"link,omitempty"`

	// Script is a list of script tags to include
	Script []ScriptTag `json:"script,omitempty" yaml:"script,omitempty" toml:"script,omitempty"`

	// AlternateFeeds configures which feeds get <link rel="alternate"> tags
	// If empty, defaults to RSS and Atom feeds
	AlternateFeeds []AlternateFeed `json:"alternate_feeds,omitempty" yaml:"alternate_feeds,omitempty" toml:"alternate_feeds,omitempty"`
}

// MetaTag represents a <meta> tag configuration.
type MetaTag struct {
	Name     string `json:"name,omitempty" yaml:"name,omitempty" toml:"name,omitempty"`
	Property string `json:"property,omitempty" yaml:"property,omitempty" toml:"property,omitempty"`
	Content  string `json:"content" yaml:"content" toml:"content"`
}

// LinkTag represents a <link> tag configuration.
type LinkTag struct {
	Rel         string `json:"rel" yaml:"rel" toml:"rel"`
	Href        string `json:"href" yaml:"href" toml:"href"`
	Crossorigin bool   `json:"crossorigin,omitempty" yaml:"crossorigin,omitempty" toml:"crossorigin,omitempty"`
}

// ScriptTag represents a <script> tag configuration.
type ScriptTag struct {
	Src string `json:"src" yaml:"src" toml:"src"`
}

// AlternateFeed configures a <link rel="alternate"> tag for feed discovery.
type AlternateFeed struct {
	// Type is the feed type: "rss", "atom", or "json"
	Type string `json:"type" yaml:"type" toml:"type"`

	// Title is the human-readable feed title (e.g., "RSS Feed")
	Title string `json:"title" yaml:"title" toml:"title"`

	// Href is the URL path to the feed (e.g., "/rss.xml")
	Href string `json:"href" yaml:"href" toml:"href"`
}

// GetMIMEType returns the MIME type for this feed type.
func (f *AlternateFeed) GetMIMEType() string {
	switch f.Type {
	case "rss":
		return "application/rss+xml"
	case "atom":
		return "application/atom+xml"
	case "json":
		return "application/feed+json"
	default:
		return "application/xml"
	}
}

// ThemeConfig configures the site theme.
type ThemeConfig struct {
	// Name is the theme name (default: "default")
	Name string `json:"name" yaml:"name" toml:"name"`

	// Palette is the color palette to use (default: "default-light")
	Palette string `json:"palette" yaml:"palette" toml:"palette"`

	// Variables allows overriding specific CSS variables
	Variables map[string]string `json:"variables" yaml:"variables" toml:"variables"`

	// CustomCSS is a path to a custom CSS file to include
	CustomCSS string `json:"custom_css" yaml:"custom_css" toml:"custom_css"`
}

// GlobConfig configures file globbing behavior.
type GlobConfig struct {
	// Patterns is the list of glob patterns to match source files
	Patterns []string `json:"patterns" yaml:"patterns" toml:"patterns"`

	// UseGitignore determines whether to respect .gitignore files
	UseGitignore bool `json:"use_gitignore" yaml:"use_gitignore" toml:"use_gitignore"`
}

// MarkdownConfig configures markdown processing.
type MarkdownConfig struct {
	// Extensions is the list of markdown extensions to enable
	Extensions []string `json:"extensions" yaml:"extensions" toml:"extensions"`

	// Highlight configures syntax highlighting for code blocks
	Highlight HighlightConfig `json:"highlight" yaml:"highlight" toml:"highlight"`
}

// HighlightConfig configures syntax highlighting for code blocks.
type HighlightConfig struct {
	// Enabled controls whether syntax highlighting is active (default: true)
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty" toml:"enabled,omitempty"`

	// Theme is the Chroma theme to use for syntax highlighting.
	// If empty, the theme is automatically derived from the site's color palette.
	// See https://xyproto.github.io/splash/docs/ for available themes.
	Theme string `json:"theme,omitempty" yaml:"theme,omitempty" toml:"theme,omitempty"`

	// LineNumbers enables line numbers in code blocks (default: false)
	LineNumbers bool `json:"line_numbers" yaml:"line_numbers" toml:"line_numbers"`
}

// NewHighlightConfig creates a new HighlightConfig with default values.
func NewHighlightConfig() HighlightConfig {
	enabled := true
	return HighlightConfig{
		Enabled:     &enabled,
		Theme:       "", // Empty means auto-detect from palette
		LineNumbers: false,
	}
}

// IsEnabled returns whether syntax highlighting is enabled.
// Defaults to true if not explicitly set.
func (h *HighlightConfig) IsEnabled() bool {
	if h.Enabled == nil {
		return true
	}
	return *h.Enabled
}

// CSVFenceConfig configures the csv_fence plugin.
type CSVFenceConfig struct {
	// Enabled controls whether CSV blocks are converted to tables (default: true)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// TableClass is the CSS class for generated tables (default: "csv-table")
	TableClass string `json:"table_class" yaml:"table_class" toml:"table_class"`

	// HasHeader indicates whether the first row is a header (default: true)
	HasHeader bool `json:"has_header" yaml:"has_header" toml:"has_header"`

	// Delimiter is the CSV field delimiter (default: ",")
	Delimiter string `json:"delimiter" yaml:"delimiter" toml:"delimiter"`
}

// NewCSVFenceConfig creates a new CSVFenceConfig with default values.
func NewCSVFenceConfig() CSVFenceConfig {
	return CSVFenceConfig{
		Enabled:    true,
		TableClass: "csv-table",
		HasHeader:  true,
		Delimiter:  ",",
	}
}

// MermaidConfig configures the mermaid plugin.
type MermaidConfig struct {
	// Enabled controls whether mermaid processing is active (default: true)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// CDNURL is the URL for the Mermaid.js library
	CDNURL string `json:"cdn_url" yaml:"cdn_url" toml:"cdn_url"`

	// Theme is the Mermaid theme to use (default, dark, forest, neutral)
	Theme string `json:"theme" yaml:"theme" toml:"theme"`
}

// NewMermaidConfig creates a new MermaidConfig with default values.
func NewMermaidConfig() MermaidConfig {
	return MermaidConfig{
		Enabled: true,
		CDNURL:  "https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.esm.min.mjs",
		Theme:   "default",
	}
}

// MDVideoConfig configures the md_video plugin.
type MDVideoConfig struct {
	// Enabled controls whether video conversion is active (default: true)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// VideoExtensions is the list of file extensions to treat as videos
	VideoExtensions []string `json:"video_extensions" yaml:"video_extensions" toml:"video_extensions"`

	// VideoClass is the CSS class added to video elements (default: "md-video")
	VideoClass string `json:"video_class" yaml:"video_class" toml:"video_class"`

	// Controls shows video controls (default: true)
	Controls bool `json:"controls" yaml:"controls" toml:"controls"`

	// Autoplay starts video automatically (default: true for GIF-like behavior)
	Autoplay bool `json:"autoplay" yaml:"autoplay" toml:"autoplay"`

	// Loop repeats the video (default: true for GIF-like behavior)
	Loop bool `json:"loop" yaml:"loop" toml:"loop"`

	// Muted mutes the video (default: true, required for autoplay in most browsers)
	Muted bool `json:"muted" yaml:"muted" toml:"muted"`

	// Playsinline enables inline playback on mobile (default: true)
	Playsinline bool `json:"playsinline" yaml:"playsinline" toml:"playsinline"`

	// Preload hints how much to preload: "none", "metadata", "auto" (default: "metadata")
	Preload string `json:"preload" yaml:"preload" toml:"preload"`
}

// NewMDVideoConfig creates a new MDVideoConfig with sensible defaults.
// Default behavior is GIF-like: autoplay, loop, muted, with controls available.
func NewMDVideoConfig() MDVideoConfig {
	return MDVideoConfig{
		Enabled:         true,
		VideoExtensions: []string{".mp4", ".webm", ".ogg", ".ogv", ".mov", ".m4v"},
		VideoClass:      "md-video",
		Controls:        true,
		Autoplay:        true,
		Loop:            true,
		Muted:           true,
		Playsinline:     true,
		Preload:         "metadata",
	}
}

// ChartJSConfig configures the chartjs plugin.
type ChartJSConfig struct {
	// Enabled controls whether Chart.js processing is active (default: true)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// CDNURL is the URL for the Chart.js library
	CDNURL string `json:"cdn_url" yaml:"cdn_url" toml:"cdn_url"`

	// ContainerClass is the CSS class for the chart container div (default: "chartjs-container")
	ContainerClass string `json:"container_class" yaml:"container_class" toml:"container_class"`
}

// NewChartJSConfig creates a new ChartJSConfig with default values.
func NewChartJSConfig() ChartJSConfig {
	return ChartJSConfig{
		Enabled:        true,
		CDNURL:         "https://cdn.jsdelivr.net/npm/chart.js",
		ContainerClass: "chartjs-container",
	}
}

// OneLineLinkConfig configures the one_line_link plugin.
type OneLineLinkConfig struct {
	// Enabled controls whether link expansion is active (default: true)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// CardClass is the CSS class for the link card (default: "link-card")
	CardClass string `json:"card_class" yaml:"card_class" toml:"card_class"`

	// FetchMetadata enables fetching title/description from URLs (default: false for performance)
	FetchMetadata bool `json:"fetch_metadata" yaml:"fetch_metadata" toml:"fetch_metadata"`

	// FallbackTitle is used when metadata fetch fails (default: "Link")
	FallbackTitle string `json:"fallback_title" yaml:"fallback_title" toml:"fallback_title"`

	// Timeout is the HTTP request timeout in seconds (default: 5)
	Timeout int `json:"timeout" yaml:"timeout" toml:"timeout"`

	// ExcludePatterns is a list of regex patterns for URLs to exclude from expansion
	ExcludePatterns []string `json:"exclude_patterns" yaml:"exclude_patterns" toml:"exclude_patterns"`
}

// NewOneLineLinkConfig creates a new OneLineLinkConfig with default values.
func NewOneLineLinkConfig() OneLineLinkConfig {
	return OneLineLinkConfig{
		Enabled:         true,
		CardClass:       "link-card",
		FetchMetadata:   false, // Disabled by default for build performance
		FallbackTitle:   "Link",
		Timeout:         5,
		ExcludePatterns: []string{},
	}
}

// WikilinkHoverConfig configures the wikilink_hover plugin.
type WikilinkHoverConfig struct {
	// Enabled controls whether hover previews are added (default: true)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// PreviewLength is the maximum characters for the preview text (default: 200)
	PreviewLength int `json:"preview_length" yaml:"preview_length" toml:"preview_length"`

	// IncludeImage adds data-preview-image attribute if post has a featured image (default: true)
	IncludeImage bool `json:"include_image" yaml:"include_image" toml:"include_image"`

	// ScreenshotService is an optional URL prefix for screenshot generation
	// If set, adds data-preview-screenshot attribute with the URL
	ScreenshotService string `json:"screenshot_service" yaml:"screenshot_service" toml:"screenshot_service"`
}

// NewWikilinkHoverConfig creates a new WikilinkHoverConfig with default values.
func NewWikilinkHoverConfig() WikilinkHoverConfig {
	return WikilinkHoverConfig{
		Enabled:           true,
		PreviewLength:     200,
		IncludeImage:      true,
		ScreenshotService: "",
	}
}

// SEOConfig configures SEO metadata for the site.
type SEOConfig struct {
	// TwitterHandle is the Twitter/X username (without @) for twitter:site meta tag
	TwitterHandle string `json:"twitter_handle" yaml:"twitter_handle" toml:"twitter_handle"`

	// DefaultImage is the default Open Graph image URL for pages without a specific image
	DefaultImage string `json:"default_image" yaml:"default_image" toml:"default_image"`

	// LogoURL is the site logo URL for Schema.org structured data
	LogoURL string `json:"logo_url" yaml:"logo_url" toml:"logo_url"`

	// StructuredData configures JSON-LD Schema.org generation
	StructuredData StructuredDataConfig `json:"structured_data" yaml:"structured_data" toml:"structured_data"`
}

// StructuredDataConfig configures JSON-LD Schema.org structured data generation.
type StructuredDataConfig struct {
	// Enabled controls whether structured data is generated (default: true)
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty" toml:"enabled,omitempty"`

	// Publisher is the site publisher information for Schema.org
	Publisher *EntityConfig `json:"publisher,omitempty" yaml:"publisher,omitempty" toml:"publisher,omitempty"`

	// DefaultAuthor is the default author for posts without explicit author
	DefaultAuthor *EntityConfig `json:"default_author,omitempty" yaml:"default_author,omitempty" toml:"default_author,omitempty"`
}

// IsEnabled returns whether structured data generation is enabled.
// Defaults to true if not explicitly set.
func (s *StructuredDataConfig) IsEnabled() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

// EntityConfig represents a Schema.org Person or Organization entity.
type EntityConfig struct {
	// Type is "Person" or "Organization" (default: "Organization")
	Type string `json:"type" yaml:"type" toml:"type"`

	// Name is the entity name
	Name string `json:"name" yaml:"name" toml:"name"`

	// URL is the entity's web page
	URL string `json:"url,omitempty" yaml:"url,omitempty" toml:"url,omitempty"`

	// Logo is the logo URL (for Organizations only)
	Logo string `json:"logo,omitempty" yaml:"logo,omitempty" toml:"logo,omitempty"`
}

// NewStructuredDataConfig creates a new StructuredDataConfig with default values.
func NewStructuredDataConfig() StructuredDataConfig {
	enabled := true
	return StructuredDataConfig{
		Enabled: &enabled,
	}
}

// NewSEOConfig creates a new SEOConfig with default values.
func NewSEOConfig() SEOConfig {
	return SEOConfig{
		TwitterHandle:  "",
		DefaultImage:   "",
		LogoURL:        "",
		StructuredData: NewStructuredDataConfig(),
	}
}

// IndieAuthConfig configures IndieAuth link tags for identity and authentication.
// IndieAuth is a decentralized authentication protocol built on OAuth 2.0.
// See https://indieauth.spec.indieweb.org/ for the specification.
type IndieAuthConfig struct {
	// Enabled controls whether IndieAuth link tags are included (default: false)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// AuthorizationEndpoint is the URL of your authorization endpoint
	// Example: "https://indieauth.com/auth"
	AuthorizationEndpoint string `json:"authorization_endpoint" yaml:"authorization_endpoint" toml:"authorization_endpoint"`

	// TokenEndpoint is the URL of your token endpoint
	// Example: "https://tokens.indieauth.com/token"
	TokenEndpoint string `json:"token_endpoint" yaml:"token_endpoint" toml:"token_endpoint"`

	// MeURL is your profile URL for rel="me" links (optional)
	// This links your site to other profiles (GitHub, Twitter, etc.)
	MeURL string `json:"me_url" yaml:"me_url" toml:"me_url"`
}

// NewIndieAuthConfig creates a new IndieAuthConfig with default values.
func NewIndieAuthConfig() IndieAuthConfig {
	return IndieAuthConfig{
		Enabled:               false,
		AuthorizationEndpoint: "",
		TokenEndpoint:         "",
		MeURL:                 "",
	}
}

// WebmentionConfig configures Webmention endpoint for receiving mentions.
// Webmention is a simple protocol for notifying URLs when you link to them.
// See https://www.w3.org/TR/webmention/ for the specification.
type WebmentionConfig struct {
	// Enabled controls whether Webmention link tag is included (default: false)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// Endpoint is the URL of your Webmention endpoint
	// Example: "https://webmention.io/example.com/webmention"
	Endpoint string `json:"endpoint" yaml:"endpoint" toml:"endpoint"`
}

// NewWebmentionConfig creates a new WebmentionConfig with default values.
func NewWebmentionConfig() WebmentionConfig {
	return WebmentionConfig{
		Enabled:  false,
		Endpoint: "",
	}
}

// PostFormatsConfig configures the output formats for individual posts.
// This controls what file formats are generated for each post.
type PostFormatsConfig struct {
	// HTML enables standard HTML output (default: true)
	// Generates: /slug/index.html
	HTML *bool `json:"html,omitempty" yaml:"html,omitempty" toml:"html,omitempty"`

	// Markdown enables raw markdown output (default: false)
	// Generates: /slug/index.md (source with frontmatter)
	Markdown bool `json:"markdown" yaml:"markdown" toml:"markdown"`

	// OG enables OpenGraph card HTML output for social image generation (default: false)
	// Generates: /slug/og/index.html (1200x630 optimized for screenshots)
	OG bool `json:"og" yaml:"og" toml:"og"`
}

// NewPostFormatsConfig creates a new PostFormatsConfig with default values.
// By default, only HTML output is enabled.
func NewPostFormatsConfig() PostFormatsConfig {
	enabled := true
	return PostFormatsConfig{
		HTML:     &enabled,
		Markdown: false,
		OG:       false,
	}
}

// IsHTMLEnabled returns whether HTML output is enabled.
// Defaults to true if not explicitly set.
func (p *PostFormatsConfig) IsHTMLEnabled() bool {
	if p.HTML == nil {
		return true
	}
	return *p.HTML
}

// QRCodeConfig configures the qrcode plugin.
type QRCodeConfig struct {
	// Enabled controls whether QR codes are generated (default: true)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// Format is the output format: "svg" or "png" (default: "svg")
	Format string `json:"format" yaml:"format" toml:"format"`

	// Size is the QR code size in pixels (default: 200)
	Size int `json:"size" yaml:"size" toml:"size"`

	// OutputDir is the subdirectory in output for QR code files (default: "qrcodes")
	OutputDir string `json:"output_dir" yaml:"output_dir" toml:"output_dir"`

	// ErrorCorrection is the QR error correction level: L, M, Q, H (default: "M")
	ErrorCorrection string `json:"error_correction" yaml:"error_correction" toml:"error_correction"`

	// Foreground is the QR code foreground color in hex (default: "#000000")
	Foreground string `json:"foreground" yaml:"foreground" toml:"foreground"`

	// Background is the QR code background color in hex (default: "#ffffff")
	Background string `json:"background" yaml:"background" toml:"background"`
}

// NewQRCodeConfig creates a new QRCodeConfig with default values.
func NewQRCodeConfig() QRCodeConfig {
	return QRCodeConfig{
		Enabled:         true,
		Format:          "svg",
		Size:            200,
		OutputDir:       "qrcodes",
		ErrorCorrection: "M",
		Foreground:      "#000000",
		Background:      "#ffffff",
	}
}

// YouTubeConfig configures the youtube plugin.
type YouTubeConfig struct {
	// Enabled controls whether YouTube URL conversion is active (default: true)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// PrivacyEnhanced uses youtube-nocookie.com for enhanced privacy (default: true)
	PrivacyEnhanced bool `json:"privacy_enhanced" yaml:"privacy_enhanced" toml:"privacy_enhanced"`

	// ContainerClass is the CSS class for the embed container (default: "youtube-embed")
	ContainerClass string `json:"container_class" yaml:"container_class" toml:"container_class"`

	// LazyLoad enables lazy loading of iframe (default: true)
	LazyLoad bool `json:"lazy_load" yaml:"lazy_load" toml:"lazy_load"`
}

// NewYouTubeConfig creates a new YouTubeConfig with default values.
func NewYouTubeConfig() YouTubeConfig {
	return YouTubeConfig{
		Enabled:         true,
		PrivacyEnhanced: true,
		ContainerClass:  "youtube-embed",
		LazyLoad:        true,
	}
}

// NewConfig creates a new Config with default values.
func NewConfig() *Config {
	return &Config{
		OutputDir:     "output",
		AssetsDir:     "static",
		TemplatesDir:  "templates",
		Hooks:         []string{"default"},
		DisabledHooks: []string{},
		GlobConfig: GlobConfig{
			Patterns:     []string{},
			UseGitignore: true,
		},
		MarkdownConfig: MarkdownConfig{
			Extensions: []string{},
			Highlight:  NewHighlightConfig(),
		},
		Feeds:        []FeedConfig{},
		FeedDefaults: NewFeedDefaults(),
		Concurrency:  0,
		Theme: ThemeConfig{
			Name:      "default",
			Palette:   "default-light",
			Variables: make(map[string]string),
		},
		PostFormats: NewPostFormatsConfig(),
		SEO:         NewSEOConfig(),
		IndieAuth:   NewIndieAuthConfig(),
		Webmention:  NewWebmentionConfig(),
		Components:  NewComponentsConfig(),
	}
}

// NewThemeConfig creates a new ThemeConfig with default values.
func NewThemeConfig() ThemeConfig {
	return ThemeConfig{
		Name:      "default",
		Palette:   "default-light",
		Variables: make(map[string]string),
	}
}

// IsHookEnabled checks if a hook is enabled (in Hooks and not in DisabledHooks).
func (c *Config) IsHookEnabled(name string) bool {
	// Check if disabled
	for _, h := range c.DisabledHooks {
		if h == name {
			return false
		}
	}

	// Check if enabled
	for _, h := range c.Hooks {
		if h == name || h == "default" {
			return true
		}
	}

	return false
}
