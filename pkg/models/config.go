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

	// Search configures site-wide search functionality using Pagefind
	Search SearchConfig `json:"search" yaml:"search" toml:"search"`

	// Layout configures the layout system for page structure
	Layout LayoutConfig `json:"layout" yaml:"layout" toml:"layout"`

	// Sidebar configures the sidebar navigation component
	Sidebar SidebarConfig `json:"sidebar" yaml:"sidebar" toml:"sidebar"`

	// Toc configures the table of contents component
	Toc TocConfig `json:"toc" yaml:"toc" toml:"toc"`

	// Header configures the header component for layouts
	Header HeaderLayoutConfig `json:"header" yaml:"header" toml:"header"`

	// FooterLayout configures the footer component for layouts
	FooterLayout FooterLayoutConfig `json:"footer_layout" yaml:"footer_layout" toml:"footer_layout"`

	// ContentTemplates configures the content template system for the new command
	ContentTemplates ContentTemplatesConfig `json:"content_templates" yaml:"content_templates" toml:"content_templates"`

	// Blogroll configures the blogroll and RSS reader functionality
	Blogroll BlogrollConfig `json:"blogroll" yaml:"blogroll" toml:"blogroll"`
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

// SearchConfig configures site-wide search functionality using Pagefind.
type SearchConfig struct {
	// Enabled controls whether search is active (default: true)
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty" toml:"enabled,omitempty"`

	// Position controls where search UI appears: "navbar", "sidebar", "footer", "custom"
	Position string `json:"position,omitempty" yaml:"position,omitempty" toml:"position,omitempty"`

	// Placeholder is the search input placeholder text
	Placeholder string `json:"placeholder,omitempty" yaml:"placeholder,omitempty" toml:"placeholder,omitempty"`

	// ShowImages shows thumbnails in search results
	ShowImages *bool `json:"show_images,omitempty" yaml:"show_images,omitempty" toml:"show_images,omitempty"`

	// ExcerptLength is the character limit for result excerpts
	ExcerptLength int `json:"excerpt_length,omitempty" yaml:"excerpt_length,omitempty" toml:"excerpt_length,omitempty"`

	// Pagefind configures the Pagefind CLI options
	Pagefind PagefindConfig `json:"pagefind,omitempty" yaml:"pagefind,omitempty" toml:"pagefind,omitempty"`

	// Feeds configures feed-specific search instances
	Feeds []SearchFeedConfig `json:"feeds,omitempty" yaml:"feeds,omitempty" toml:"feeds,omitempty"`
}

// IsEnabled returns whether search is enabled.
// Defaults to true if not explicitly set.
func (s *SearchConfig) IsEnabled() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

// IsShowImages returns whether to show images in search results.
// Defaults to true if not explicitly set.
func (s *SearchConfig) IsShowImages() bool {
	if s.ShowImages == nil {
		return true
	}
	return *s.ShowImages
}

// PagefindConfig configures Pagefind CLI behavior.
type PagefindConfig struct {
	// BundleDir is the output directory for search index (default: "_pagefind")
	BundleDir string `json:"bundle_dir,omitempty" yaml:"bundle_dir,omitempty" toml:"bundle_dir,omitempty"`

	// ExcludeSelectors are CSS selectors for elements to exclude from indexing
	ExcludeSelectors []string `json:"exclude_selectors,omitempty" yaml:"exclude_selectors,omitempty" toml:"exclude_selectors,omitempty"`

	// RootSelector is the CSS selector for the searchable content container
	RootSelector string `json:"root_selector,omitempty" yaml:"root_selector,omitempty" toml:"root_selector,omitempty"`

	// AutoInstall enables automatic Pagefind binary installation (default: true)
	AutoInstall *bool `json:"auto_install,omitempty" yaml:"auto_install,omitempty" toml:"auto_install,omitempty"`

	// Version is the Pagefind version to install (default: "latest")
	Version string `json:"version,omitempty" yaml:"version,omitempty" toml:"version,omitempty"`

	// CacheDir is the directory for caching Pagefind binaries (default: XDG cache)
	CacheDir string `json:"cache_dir,omitempty" yaml:"cache_dir,omitempty" toml:"cache_dir,omitempty"`

	// Verbose enables verbose output from Pagefind (default: false)
	// When false, only errors are shown. When true or when --verbose CLI flag is used, all output is shown.
	Verbose *bool `json:"verbose,omitempty" yaml:"verbose,omitempty" toml:"verbose,omitempty"`
}

// IsAutoInstallEnabled returns whether automatic Pagefind installation is enabled.
// Defaults to true if not explicitly set.
func (p *PagefindConfig) IsAutoInstallEnabled() bool {
	if p.AutoInstall == nil {
		return true
	}
	return *p.AutoInstall
}

// IsVerbose returns whether verbose output is enabled.
// Defaults to false if not explicitly set.
func (p *PagefindConfig) IsVerbose() bool {
	if p.Verbose == nil {
		return false
	}
	return *p.Verbose
}

// SearchFeedConfig configures a feed-specific search instance.
type SearchFeedConfig struct {
	// Name is the search instance identifier
	Name string `json:"name" yaml:"name" toml:"name"`

	// Filter is the filter expression for posts in this search
	Filter string `json:"filter" yaml:"filter" toml:"filter"`

	// Position controls where this search UI appears
	Position string `json:"position,omitempty" yaml:"position,omitempty" toml:"position,omitempty"`

	// Placeholder is the search input placeholder text
	Placeholder string `json:"placeholder,omitempty" yaml:"placeholder,omitempty" toml:"placeholder,omitempty"`
}

// NewSearchConfig creates a new SearchConfig with default values.
func NewSearchConfig() SearchConfig {
	enabled := true
	showImages := true
	autoInstall := true
	return SearchConfig{
		Enabled:       &enabled,
		Position:      "navbar",
		Placeholder:   "Search...",
		ShowImages:    &showImages,
		ExcerptLength: 200,
		Pagefind: PagefindConfig{
			BundleDir:        "_pagefind",
			ExcludeSelectors: []string{},
			RootSelector:     "",
			AutoInstall:      &autoInstall,
			Version:          "latest",
			CacheDir:         "",
		},
		Feeds: []SearchFeedConfig{},
	}
}

// FontConfig configures typography settings for the site.
type FontConfig struct {
	// Family is the primary font family for body text (default: "system-ui, -apple-system, sans-serif")
	Family string `json:"family,omitempty" yaml:"family,omitempty" toml:"family,omitempty"`

	// HeadingFamily is the font family for headings (default: inherits from Family)
	HeadingFamily string `json:"heading_family,omitempty" yaml:"heading_family,omitempty" toml:"heading_family,omitempty"`

	// CodeFamily is the font family for code blocks and inline code (default: "ui-monospace, monospace")
	CodeFamily string `json:"code_family,omitempty" yaml:"code_family,omitempty" toml:"code_family,omitempty"`

	// Size is the base font size (default: "16px")
	Size string `json:"size,omitempty" yaml:"size,omitempty" toml:"size,omitempty"`

	// LineHeight is the base line height (default: "1.6")
	LineHeight string `json:"line_height,omitempty" yaml:"line_height,omitempty" toml:"line_height,omitempty"`

	// GoogleFonts is a list of Google Fonts to load (e.g., ["Inter", "Fira Code"])
	GoogleFonts []string `json:"google_fonts,omitempty" yaml:"google_fonts,omitempty" toml:"google_fonts,omitempty"`

	// CustomURLs is a list of custom font CSS URLs to load
	CustomURLs []string `json:"custom_urls,omitempty" yaml:"custom_urls,omitempty" toml:"custom_urls,omitempty"`
}

// NewFontConfig creates a new FontConfig with default values.
func NewFontConfig() FontConfig {
	return FontConfig{
		Family:        "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
		HeadingFamily: "",
		CodeFamily:    "ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, monospace",
		Size:          "16px",
		LineHeight:    "1.6",
		GoogleFonts:   []string{},
		CustomURLs:    []string{},
	}
}

// GetHeadingFamily returns the heading font family, falling back to Family if not set.
func (f *FontConfig) GetHeadingFamily() string {
	if f.HeadingFamily != "" {
		return f.HeadingFamily
	}
	return f.Family
}

// GetGoogleFontsURL returns the Google Fonts CSS URL for the configured fonts.
// Returns empty string if no Google Fonts are configured.
func (f *FontConfig) GetGoogleFontsURL() string {
	if len(f.GoogleFonts) == 0 {
		return ""
	}
	// Build Google Fonts URL with all fonts
	// Format: https://fonts.googleapis.com/css2?family=Font+Name:wght@400;700&family=Other+Font
	families := make([]string, len(f.GoogleFonts))
	for i, font := range f.GoogleFonts {
		// Replace spaces with + for URL encoding
		encoded := ""
		for _, c := range font {
			if c == ' ' {
				encoded += "+"
			} else {
				encoded += string(c)
			}
		}
		families[i] = "family=" + encoded + ":wght@400;500;600;700"
	}
	return "https://fonts.googleapis.com/css2?" + joinStrings(families, "&") + "&display=swap"
}

// joinStrings joins strings with a separator (helper to avoid importing strings package).
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// ThemeConfig configures the site theme.
type ThemeConfig struct {
	// Name is the theme name (default: "default")
	Name string `json:"name" yaml:"name" toml:"name"`

	// Palette is the base color palette to use (default: "default-light")
	// When set to a base name like "everforest", the system will auto-detect
	// light/dark variants (e.g., "everforest-light" and "everforest-dark")
	Palette string `json:"palette" yaml:"palette" toml:"palette"`

	// PaletteLight is the palette to use for light mode (optional)
	// If not set, auto-detected from base Palette name
	PaletteLight string `json:"palette_light,omitempty" yaml:"palette_light,omitempty" toml:"palette_light,omitempty"`

	// PaletteDark is the palette to use for dark mode (optional)
	// If not set, auto-detected from base Palette name
	PaletteDark string `json:"palette_dark,omitempty" yaml:"palette_dark,omitempty" toml:"palette_dark,omitempty"`

	// Variables allows overriding specific CSS variables
	Variables map[string]string `json:"variables" yaml:"variables" toml:"variables"`

	// CustomCSS is a path to a custom CSS file to include
	CustomCSS string `json:"custom_css" yaml:"custom_css" toml:"custom_css"`

	// Background configures multi-layered background decorations
	Background BackgroundConfig `json:"background,omitempty" yaml:"background,omitempty" toml:"background,omitempty"`

	// Font configures typography settings
	Font FontConfig `json:"font,omitempty" yaml:"font,omitempty" toml:"font,omitempty"`
}

// BackgroundConfig configures multi-layered background decorations for pages.
// Background elements are rendered as fixed-position layers behind the main content.
type BackgroundConfig struct {
	// Enabled controls whether background decorations are active (default: false)
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty" toml:"enabled,omitempty"`

	// Backgrounds is the list of background elements to render
	Backgrounds []BackgroundElement `json:"backgrounds,omitempty" yaml:"backgrounds,omitempty" toml:"backgrounds,omitempty"`

	// Scripts is a list of script URLs to include for background functionality
	// Example: ["/static/js/snow-fall.js"]
	Scripts []string `json:"scripts,omitempty" yaml:"scripts,omitempty" toml:"scripts,omitempty"`

	// CSS is custom CSS for styling background elements
	CSS string `json:"css,omitempty" yaml:"css,omitempty" toml:"css,omitempty"`

	// ArticleBg is the background color for article/content areas (default: uses --color-background)
	// This helps ensure content is readable over decorative backgrounds.
	// Example: "rgba(255, 255, 255, 0.95)" or "#ffffff"
	ArticleBg string `json:"article_bg,omitempty" yaml:"article_bg,omitempty" toml:"article_bg,omitempty"`

	// ArticleBlur is the backdrop blur amount for article areas (default: "0px")
	// Example: "8px" or "12px" for a frosted glass effect
	ArticleBlur string `json:"article_blur,omitempty" yaml:"article_blur,omitempty" toml:"article_blur,omitempty"`

	// ArticleShadow is the box-shadow for article areas
	// Example: "0 4px 20px rgba(0, 0, 0, 0.3)"
	ArticleShadow string `json:"article_shadow,omitempty" yaml:"article_shadow,omitempty" toml:"article_shadow,omitempty"`

	// ArticleBorder is the border style for article areas
	// Example: "1px solid rgba(255, 255, 255, 0.1)"
	ArticleBorder string `json:"article_border,omitempty" yaml:"article_border,omitempty" toml:"article_border,omitempty"`

	// ArticleRadius is the border-radius for article areas (default: uses --radius-lg)
	// Example: "12px" or "1rem"
	ArticleRadius string `json:"article_radius,omitempty" yaml:"article_radius,omitempty" toml:"article_radius,omitempty"`
}

// BackgroundElement represents a single background decoration layer.
type BackgroundElement struct {
	// HTML is the HTML content for this background layer
	// Example: '<snow-fall count="200"></snow-fall>'
	HTML string `json:"html" yaml:"html" toml:"html"`

	// ZIndex controls the stacking order of this layer (default: -1)
	// Negative values place the layer behind content, positive values in front
	ZIndex int `json:"z_index,omitempty" yaml:"z_index,omitempty" toml:"z_index,omitempty"`
}

// NewBackgroundConfig creates a new BackgroundConfig with default values.
func NewBackgroundConfig() BackgroundConfig {
	enabled := false
	return BackgroundConfig{
		Enabled:       &enabled,
		Backgrounds:   []BackgroundElement{},
		Scripts:       []string{},
		CSS:           "",
		ArticleBg:     "",
		ArticleBlur:   "",
		ArticleShadow: "",
		ArticleBorder: "",
		ArticleRadius: "",
	}
}

// IsEnabled returns whether background decorations are enabled.
// Defaults to false if not explicitly set.
func (b *BackgroundConfig) IsEnabled() bool {
	if b.Enabled == nil {
		return false
	}
	return *b.Enabled
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

// WebMentionsConfig configures the webmentions plugin for sending outgoing mentions.
// This is separate from WebmentionConfig which handles receiving mentions.
type WebMentionsConfig struct {
	// Enabled controls whether the webmentions plugin is active (default: false)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// Outgoing enables sending webmentions for external links (default: true when enabled)
	Outgoing bool `json:"outgoing" yaml:"outgoing" toml:"outgoing"`

	// UserAgent is the User-Agent string for HTTP requests
	UserAgent string `json:"user_agent" yaml:"user_agent" toml:"user_agent"`

	// Timeout is the HTTP request timeout (e.g., "30s")
	Timeout string `json:"timeout" yaml:"timeout" toml:"timeout"`

	// CacheDir is the directory for caching sent webmentions (default: ".cache/webmentions")
	CacheDir string `json:"cache_dir" yaml:"cache_dir" toml:"cache_dir"`

	// ConcurrentRequests is the max number of concurrent webmention requests (default: 5)
	ConcurrentRequests int `json:"concurrent_requests" yaml:"concurrent_requests" toml:"concurrent_requests"`

	// Bridges configures social media bridging for incoming webmentions
	Bridges BridgesConfig `json:"bridges" yaml:"bridges" toml:"bridges"`

	// WebmentionIOToken is the API token for webmention.io (for receiving mentions)
	WebmentionIOToken string `json:"webmention_io_token" yaml:"webmention_io_token" toml:"webmention_io_token"`
}

// BridgesConfig configures social media bridging services.
type BridgesConfig struct {
	// Enabled controls whether bridging detection is active (default: false)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// BridgyFediverse enables Bridgy Fed integration (default: true when bridges enabled)
	BridgyFediverse bool `json:"bridgy_fediverse" yaml:"bridgy_fediverse" toml:"bridgy_fediverse"`

	// Platform-specific controls
	Bluesky  bool `json:"bluesky" yaml:"bluesky" toml:"bluesky"`
	Twitter  bool `json:"twitter" yaml:"twitter" toml:"twitter"`
	Mastodon bool `json:"mastodon" yaml:"mastodon" toml:"mastodon"`
	GitHub   bool `json:"github" yaml:"github" toml:"github"`
	Flickr   bool `json:"flickr" yaml:"flickr" toml:"flickr"`

	// Filters configures filtering of bridged mentions
	Filters BridgeFiltersConfig `json:"filters" yaml:"filters" toml:"filters"`
}

// BridgeFiltersConfig configures filtering for bridged webmentions.
type BridgeFiltersConfig struct {
	// Platforms limits which platforms to accept (empty = all enabled)
	Platforms []string `json:"platforms" yaml:"platforms" toml:"platforms"`

	// InteractionTypes limits which interaction types to accept (empty = all)
	// Valid values: "like", "repost", "reply", "bookmark", "mention"
	InteractionTypes []string `json:"interaction_types" yaml:"interaction_types" toml:"interaction_types"`

	// MinContentLength filters out mentions with content shorter than this
	MinContentLength int `json:"min_content_length" yaml:"min_content_length" toml:"min_content_length"`

	// BlockedDomains is a list of domains to reject mentions from
	BlockedDomains []string `json:"blocked_domains" yaml:"blocked_domains" toml:"blocked_domains"`
}

// NewBridgesConfig creates a new BridgesConfig with default values.
func NewBridgesConfig() BridgesConfig {
	return BridgesConfig{
		Enabled:         false,
		BridgyFediverse: true,
		Bluesky:         true,
		Twitter:         true,
		Mastodon:        true,
		GitHub:          true,
		Flickr:          false,
		Filters:         BridgeFiltersConfig{},
	}
}

// NewWebMentionsConfig creates a new WebMentionsConfig with default values.
func NewWebMentionsConfig() WebMentionsConfig {
	return WebMentionsConfig{
		Enabled:            false,
		Outgoing:           true,
		UserAgent:          "markata-go/1.0 (WebMention Sender; +https://github.com/WaylonWalker/markata-go)",
		Timeout:            "30s",
		CacheDir:           ".cache/webmentions",
		ConcurrentRequests: 5,
		Bridges:            NewBridgesConfig(),
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

	// Text enables plain text output (default: false)
	// Generates: /slug/index.txt (content only, no formatting)
	Text bool `json:"text" yaml:"text" toml:"text"`

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
		Text:     false,
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

// ImageZoomConfig configures the image_zoom plugin for lightbox functionality.
type ImageZoomConfig struct {
	// Enabled controls whether image zoom is active (default: false)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// Library is the lightbox library to use (default: "glightbox")
	// Supported: "glightbox"
	Library string `json:"library" yaml:"library" toml:"library"`

	// Selector is the CSS selector for images to make zoomable (default: ".glightbox")
	Selector string `json:"selector" yaml:"selector" toml:"selector"`

	// CDN uses CDN for library files instead of local (default: true)
	CDN bool `json:"cdn" yaml:"cdn" toml:"cdn"`

	// AutoAllImages enables zoom on all images without explicit marking (default: false)
	AutoAllImages bool `json:"auto_all_images" yaml:"auto_all_images" toml:"auto_all_images"`

	// OpenEffect is the effect when opening the lightbox (default: "zoom")
	// Options: "zoom", "fade", "none"
	OpenEffect string `json:"open_effect" yaml:"open_effect" toml:"open_effect"`

	// CloseEffect is the effect when closing the lightbox (default: "zoom")
	// Options: "zoom", "fade", "none"
	CloseEffect string `json:"close_effect" yaml:"close_effect" toml:"close_effect"`

	// SlideEffect is the effect when sliding between images (default: "slide")
	// Options: "slide", "fade", "zoom", "none"
	SlideEffect string `json:"slide_effect" yaml:"slide_effect" toml:"slide_effect"`

	// TouchNavigation enables touch/swipe navigation (default: true)
	TouchNavigation bool `json:"touch_navigation" yaml:"touch_navigation" toml:"touch_navigation"`

	// Loop enables looping through images in a gallery (default: false)
	Loop bool `json:"loop" yaml:"loop" toml:"loop"`

	// Draggable enables dragging images to navigate (default: true)
	Draggable bool `json:"draggable" yaml:"draggable" toml:"draggable"`
}

// NewImageZoomConfig creates a new ImageZoomConfig with default values.
func NewImageZoomConfig() ImageZoomConfig {
	return ImageZoomConfig{
		Enabled:         false, // Disabled by default
		Library:         "glightbox",
		Selector:        ".glightbox",
		CDN:             true,
		AutoAllImages:   false,
		OpenEffect:      "zoom",
		CloseEffect:     "zoom",
		SlideEffect:     "slide",
		TouchNavigation: true,
		Loop:            false,
		Draggable:       true,
	}
}

// EmbedsConfig configures the embeds plugin for embedding internal and external content.
type EmbedsConfig struct {
	// Enabled controls whether embed processing is active (default: true)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// InternalCardClass is the CSS class for internal embed cards (default: "embed-card")
	InternalCardClass string `json:"internal_card_class" yaml:"internal_card_class" toml:"internal_card_class"`

	// ExternalCardClass is the CSS class for external embed cards (default: "embed-card embed-card-external")
	ExternalCardClass string `json:"external_card_class" yaml:"external_card_class" toml:"external_card_class"`

	// FetchExternal enables fetching OG metadata for external embeds (default: true)
	FetchExternal bool `json:"fetch_external" yaml:"fetch_external" toml:"fetch_external"`

	// CacheDir is the directory for caching external embed metadata (default: ".cache/embeds")
	CacheDir string `json:"cache_dir" yaml:"cache_dir" toml:"cache_dir"`

	// Timeout is the HTTP request timeout in seconds for external fetches (default: 10)
	Timeout int `json:"timeout" yaml:"timeout" toml:"timeout"`

	// FallbackTitle is used when OG title cannot be fetched (default: "External Link")
	FallbackTitle string `json:"fallback_title" yaml:"fallback_title" toml:"fallback_title"`

	// ShowImage controls whether to display OG images in external embeds (default: true)
	ShowImage bool `json:"show_image" yaml:"show_image" toml:"show_image"`
}

// NewEmbedsConfig creates a new EmbedsConfig with default values.
func NewEmbedsConfig() EmbedsConfig {
	return EmbedsConfig{
		Enabled:           true,
		InternalCardClass: "embed-card",
		ExternalCardClass: "embed-card embed-card-external",
		FetchExternal:     true,
		CacheDir:          ".cache/embeds",
		Timeout:           10,
		FallbackTitle:     "External Link",
		ShowImage:         true,
	}
}

// ContentTemplateConfig defines a single content template.
type ContentTemplateConfig struct {
	// Name is the template identifier (e.g., "post", "page", "docs")
	Name string `json:"name" yaml:"name" toml:"name"`

	// Directory is the output directory for this content type
	Directory string `json:"directory" yaml:"directory" toml:"directory"`

	// Frontmatter contains default frontmatter fields for this template
	Frontmatter map[string]interface{} `json:"frontmatter,omitempty" yaml:"frontmatter,omitempty" toml:"frontmatter,omitempty"`

	// Body is the default body content (markdown) for this template
	Body string `json:"body,omitempty" yaml:"body,omitempty" toml:"body,omitempty"`
}

// ContentTemplatesConfig configures the content template system for the new command.
type ContentTemplatesConfig struct {
	// Directory is where user-defined templates are stored (default: "content-templates")
	Directory string `json:"directory" yaml:"directory" toml:"directory"`

	// Placement maps template names to output directories
	Placement map[string]string `json:"placement" yaml:"placement" toml:"placement"`

	// Templates is a list of custom template configurations
	Templates []ContentTemplateConfig `json:"templates,omitempty" yaml:"templates,omitempty" toml:"templates,omitempty"`
}

// NewContentTemplatesConfig creates a new ContentTemplatesConfig with default values.
func NewContentTemplatesConfig() ContentTemplatesConfig {
	return ContentTemplatesConfig{
		Directory: "content-templates",
		Placement: map[string]string{
			"post": "posts",
			"page": "pages",
			"docs": "docs",
		},
		Templates: []ContentTemplateConfig{},
	}
}

// GetPlacement returns the output directory for a template name.
// Returns the template name itself if no explicit placement is configured.
func (c *ContentTemplatesConfig) GetPlacement(templateName string) string {
	if dir, ok := c.Placement[templateName]; ok {
		return dir
	}
	return templateName
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
			Font:      NewFontConfig(),
		},
		PostFormats:      NewPostFormatsConfig(),
		SEO:              NewSEOConfig(),
		IndieAuth:        NewIndieAuthConfig(),
		Webmention:       NewWebmentionConfig(),
		Components:       NewComponentsConfig(),
		Search:           NewSearchConfig(),
		Layout:           NewLayoutConfig(),
		Sidebar:          NewSidebarConfig(),
		Toc:              NewTocConfig(),
		Header:           NewHeaderLayoutConfig(),
		FooterLayout:     NewFooterLayoutConfig(),
		ContentTemplates: NewContentTemplatesConfig(),
		Blogroll:         NewBlogrollConfig(),
	}
}

// NewThemeConfig creates a new ThemeConfig with default values.
func NewThemeConfig() ThemeConfig {
	return ThemeConfig{
		Name:      "default",
		Palette:   "default-light",
		Variables: make(map[string]string),
		Font:      NewFontConfig(),
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
