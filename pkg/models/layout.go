package models

import "strings"

// Layout name constants
const (
	layoutBlog = "blog"
)

// LayoutConfig configures the site layout system.
// Layouts control the overall page structure including sidebars, TOC, header, and footer.
type LayoutConfig struct {
	// Name is the default layout preset name (default: "blog")
	// Options: "docs", "blog", "landing", "bare"
	Name string `json:"name,omitempty" yaml:"name,omitempty" toml:"name,omitempty"`

	// Paths maps URL path prefixes to layout names for automatic layout selection.
	// Example: {"/docs/": "docs", "/blog/": "blog", "/about/": "landing"}
	Paths map[string]string `json:"paths,omitempty" yaml:"paths,omitempty" toml:"paths,omitempty"`

	// Feeds maps feed slugs to layout names for automatic layout selection.
	// Example: {"docs": "docs", "blog": "blog"}
	Feeds map[string]string `json:"feeds,omitempty" yaml:"feeds,omitempty" toml:"feeds,omitempty"`

	// Docs configures the documentation layout
	Docs DocsLayoutConfig `json:"docs,omitempty" yaml:"docs,omitempty" toml:"docs,omitempty"`

	// Blog configures the blog layout
	Blog BlogLayoutConfig `json:"blog,omitempty" yaml:"blog,omitempty" toml:"blog,omitempty"`

	// Landing configures the landing page layout
	Landing LandingLayoutConfig `json:"landing,omitempty" yaml:"landing,omitempty" toml:"landing,omitempty"`

	// Bare configures the bare layout (content only)
	Bare BareLayoutConfig `json:"bare,omitempty" yaml:"bare,omitempty" toml:"bare,omitempty"`

	// Defaults provides global layout defaults
	Defaults LayoutDefaults `json:"defaults,omitempty" yaml:"defaults,omitempty" toml:"defaults,omitempty"`
}

// ResolveLayout determines the appropriate layout for a post based on its path and feed.
// Priority: path-based > feed-based > global default
// Returns the layout name (e.g., "docs", "blog", "landing", "bare") or empty string if no match.
func (l *LayoutConfig) ResolveLayout(postPath, feedSlug string) string {
	// 1. Check path-based layout (longest prefix wins)
	if len(l.Paths) > 0 {
		var bestMatch string
		var bestLayout string

		for pathPrefix, layout := range l.Paths {
			if strings.HasPrefix(postPath, pathPrefix) {
				if len(pathPrefix) > len(bestMatch) {
					bestMatch = pathPrefix
					bestLayout = layout
				}
			}
		}

		if bestLayout != "" {
			return bestLayout
		}
	}

	// 2. Check feed-based layout
	if feedSlug != "" && len(l.Feeds) > 0 {
		if layout, ok := l.Feeds[feedSlug]; ok {
			return layout
		}
	}

	// 3. Fall back to global default
	return l.Name
}

// LayoutToTemplate converts a layout name to a template file path.
// Layout names map to templates as follows:
//   - "docs" -> "layouts/docs.html"
//   - "blog" -> "post.html"
//   - "landing" -> "layouts/landing.html"
//   - "bare" -> "layouts/bare.html"
//   - "" (empty) -> "post.html" (default)
func LayoutToTemplate(layout string) string {
	switch layout {
	case "docs":
		return "layouts/docs.html"
	case layoutBlog, "":
		return "post.html"
	case "landing":
		return "layouts/landing.html"
	case "bare":
		return "layouts/bare.html"
	default:
		// For custom layouts, assume the layout name is the template name
		if strings.HasSuffix(layout, ".html") {
			return layout
		}
		return layout + ".html"
	}
}

// LayoutDefaults provides global layout defaults.
type LayoutDefaults struct {
	// ContentMaxWidth is the maximum width of the content area (default: "800px")
	ContentMaxWidth string `json:"content_max_width,omitempty" yaml:"content_max_width,omitempty" toml:"content_max_width,omitempty"`

	// HeaderSticky makes the header stick to the top when scrolling (default: true)
	HeaderSticky *bool `json:"header_sticky,omitempty" yaml:"header_sticky,omitempty" toml:"header_sticky,omitempty"`

	// FooterSticky makes the footer stick to the bottom (default: false)
	FooterSticky *bool `json:"footer_sticky,omitempty" yaml:"footer_sticky,omitempty" toml:"footer_sticky,omitempty"`
}

// IsHeaderSticky returns whether the header should be sticky.
func (d *LayoutDefaults) IsHeaderSticky() bool {
	if d.HeaderSticky == nil {
		return true
	}
	return *d.HeaderSticky
}

// IsFooterSticky returns whether the footer should be sticky.
func (d *LayoutDefaults) IsFooterSticky() bool {
	if d.FooterSticky == nil {
		return false
	}
	return *d.FooterSticky
}

// DocsLayoutConfig configures the documentation layout.
// This is a 3-panel layout with sidebar navigation, content, and table of contents.
type DocsLayoutConfig struct {
	// SidebarPosition controls sidebar placement: "left" or "right" (default: "left")
	SidebarPosition string `json:"sidebar_position,omitempty" yaml:"sidebar_position,omitempty" toml:"sidebar_position,omitempty"`

	// SidebarWidth is the width of the sidebar (default: "280px")
	SidebarWidth string `json:"sidebar_width,omitempty" yaml:"sidebar_width,omitempty" toml:"sidebar_width,omitempty"`

	// SidebarCollapsible allows the sidebar to be collapsed (default: true)
	SidebarCollapsible *bool `json:"sidebar_collapsible,omitempty" yaml:"sidebar_collapsible,omitempty" toml:"sidebar_collapsible,omitempty"`

	// SidebarDefaultOpen controls if sidebar is open by default on desktop (default: true)
	SidebarDefaultOpen *bool `json:"sidebar_default_open,omitempty" yaml:"sidebar_default_open,omitempty" toml:"sidebar_default_open,omitempty"`

	// TocPosition controls TOC placement: "left" or "right" (default: "right")
	TocPosition string `json:"toc_position,omitempty" yaml:"toc_position,omitempty" toml:"toc_position,omitempty"`

	// TocWidth is the width of the table of contents (default: "220px")
	TocWidth string `json:"toc_width,omitempty" yaml:"toc_width,omitempty" toml:"toc_width,omitempty"`

	// TocCollapsible allows the TOC to be collapsed (default: true)
	TocCollapsible *bool `json:"toc_collapsible,omitempty" yaml:"toc_collapsible,omitempty" toml:"toc_collapsible,omitempty"`

	// TocDefaultOpen controls if TOC is open by default on desktop (default: true)
	TocDefaultOpen *bool `json:"toc_default_open,omitempty" yaml:"toc_default_open,omitempty" toml:"toc_default_open,omitempty"`

	// ContentMaxWidth is the maximum width of the content area (default: "800px")
	ContentMaxWidth string `json:"content_max_width,omitempty" yaml:"content_max_width,omitempty" toml:"content_max_width,omitempty"`

	// HeaderStyle controls the header appearance: "full", "minimal", "transparent", "none" (default: "minimal")
	HeaderStyle string `json:"header_style,omitempty" yaml:"header_style,omitempty" toml:"header_style,omitempty"`

	// FooterStyle controls the footer appearance: "full", "minimal", "none" (default: "minimal")
	FooterStyle string `json:"footer_style,omitempty" yaml:"footer_style,omitempty" toml:"footer_style,omitempty"`
}

// IsSidebarCollapsible returns whether the sidebar can be collapsed.
func (d *DocsLayoutConfig) IsSidebarCollapsible() bool {
	if d.SidebarCollapsible == nil {
		return true
	}
	return *d.SidebarCollapsible
}

// IsSidebarDefaultOpen returns whether the sidebar is open by default.
func (d *DocsLayoutConfig) IsSidebarDefaultOpen() bool {
	if d.SidebarDefaultOpen == nil {
		return true
	}
	return *d.SidebarDefaultOpen
}

// IsTocCollapsible returns whether the TOC can be collapsed.
func (d *DocsLayoutConfig) IsTocCollapsible() bool {
	if d.TocCollapsible == nil {
		return true
	}
	return *d.TocCollapsible
}

// IsTocDefaultOpen returns whether the TOC is open by default.
func (d *DocsLayoutConfig) IsTocDefaultOpen() bool {
	if d.TocDefaultOpen == nil {
		return true
	}
	return *d.TocDefaultOpen
}

// BlogLayoutConfig configures the blog layout.
// This is a single-column layout optimized for reading long-form content.
type BlogLayoutConfig struct {
	// ContentMaxWidth is the maximum width of the content area (default: "720px")
	ContentMaxWidth string `json:"content_max_width,omitempty" yaml:"content_max_width,omitempty" toml:"content_max_width,omitempty"`

	// ShowToc enables table of contents for blog posts (default: false)
	ShowToc *bool `json:"show_toc,omitempty" yaml:"show_toc,omitempty" toml:"show_toc,omitempty"`

	// TocPosition controls TOC placement: "left" or "right" (default: "right")
	TocPosition string `json:"toc_position,omitempty" yaml:"toc_position,omitempty" toml:"toc_position,omitempty"`

	// TocWidth is the width of the table of contents (default: "200px")
	TocWidth string `json:"toc_width,omitempty" yaml:"toc_width,omitempty" toml:"toc_width,omitempty"`

	// HeaderStyle controls the header appearance: "full", "minimal", "transparent", "none" (default: "full")
	HeaderStyle string `json:"header_style,omitempty" yaml:"header_style,omitempty" toml:"header_style,omitempty"`

	// FooterStyle controls the footer appearance: "full", "minimal", "none" (default: "full")
	FooterStyle string `json:"footer_style,omitempty" yaml:"footer_style,omitempty" toml:"footer_style,omitempty"`

	// ShowAuthor displays the post author (default: true)
	ShowAuthor *bool `json:"show_author,omitempty" yaml:"show_author,omitempty" toml:"show_author,omitempty"`

	// ShowDate displays the post date (default: true)
	ShowDate *bool `json:"show_date,omitempty" yaml:"show_date,omitempty" toml:"show_date,omitempty"`

	// ShowTags displays the post tags (default: true)
	ShowTags *bool `json:"show_tags,omitempty" yaml:"show_tags,omitempty" toml:"show_tags,omitempty"`

	// ShowReadingTime displays estimated reading time (default: true)
	ShowReadingTime *bool `json:"show_reading_time,omitempty" yaml:"show_reading_time,omitempty" toml:"show_reading_time,omitempty"`

	// ShowPrevNext displays previous/next post navigation (default: true)
	ShowPrevNext *bool `json:"show_prev_next,omitempty" yaml:"show_prev_next,omitempty" toml:"show_prev_next,omitempty"`
}

// IsShowToc returns whether to show the table of contents.
func (b *BlogLayoutConfig) IsShowToc() bool {
	if b.ShowToc == nil {
		return false
	}
	return *b.ShowToc
}

// IsShowAuthor returns whether to show the author.
func (b *BlogLayoutConfig) IsShowAuthor() bool {
	if b.ShowAuthor == nil {
		return true
	}
	return *b.ShowAuthor
}

// IsShowDate returns whether to show the date.
func (b *BlogLayoutConfig) IsShowDate() bool {
	if b.ShowDate == nil {
		return true
	}
	return *b.ShowDate
}

// IsShowTags returns whether to show tags.
func (b *BlogLayoutConfig) IsShowTags() bool {
	if b.ShowTags == nil {
		return true
	}
	return *b.ShowTags
}

// IsShowReadingTime returns whether to show reading time.
func (b *BlogLayoutConfig) IsShowReadingTime() bool {
	if b.ShowReadingTime == nil {
		return true
	}
	return *b.ShowReadingTime
}

// IsShowPrevNext returns whether to show previous/next navigation.
func (b *BlogLayoutConfig) IsShowPrevNext() bool {
	if b.ShowPrevNext == nil {
		return true
	}
	return *b.ShowPrevNext
}

// LandingLayoutConfig configures the landing page layout.
// This is a full-width layout for marketing pages and home pages.
type LandingLayoutConfig struct {
	// ContentMaxWidth is the maximum width of the content area (default: "100%")
	ContentMaxWidth string `json:"content_max_width,omitempty" yaml:"content_max_width,omitempty" toml:"content_max_width,omitempty"`

	// HeaderStyle controls the header appearance: "full", "minimal", "transparent", "none" (default: "transparent")
	HeaderStyle string `json:"header_style,omitempty" yaml:"header_style,omitempty" toml:"header_style,omitempty"`

	// HeaderSticky makes the header stick when scrolling (default: true)
	HeaderSticky *bool `json:"header_sticky,omitempty" yaml:"header_sticky,omitempty" toml:"header_sticky,omitempty"`

	// FooterStyle controls the footer appearance: "full", "minimal", "none" (default: "full")
	FooterStyle string `json:"footer_style,omitempty" yaml:"footer_style,omitempty" toml:"footer_style,omitempty"`

	// HeroEnabled enables the hero section (default: true)
	HeroEnabled *bool `json:"hero_enabled,omitempty" yaml:"hero_enabled,omitempty" toml:"hero_enabled,omitempty"`
}

// IsHeaderSticky returns whether the header should be sticky.
func (l *LandingLayoutConfig) IsHeaderSticky() bool {
	if l.HeaderSticky == nil {
		return true
	}
	return *l.HeaderSticky
}

// IsHeroEnabled returns whether the hero section is enabled.
func (l *LandingLayoutConfig) IsHeroEnabled() bool {
	if l.HeroEnabled == nil {
		return true
	}
	return *l.HeroEnabled
}

// BareLayoutConfig configures the bare layout.
// This is a minimal layout with no chrome - just the content.
type BareLayoutConfig struct {
	// ContentMaxWidth is the maximum width of the content area (default: "100%")
	ContentMaxWidth string `json:"content_max_width,omitempty" yaml:"content_max_width,omitempty" toml:"content_max_width,omitempty"`
}

// SidebarNavItem represents a navigation item in the sidebar.
type SidebarNavItem struct {
	// Title is the display text for the navigation item
	Title string `json:"title" yaml:"title" toml:"title"`

	// Href is the link destination (can be relative or absolute)
	Href string `json:"href,omitempty" yaml:"href,omitempty" toml:"href,omitempty"`

	// Children are nested navigation items
	Children []SidebarNavItem `json:"children,omitempty" yaml:"children,omitempty" toml:"children,omitempty"`
}

// SidebarConfig configures the sidebar navigation component.
type SidebarConfig struct {
	// Enabled controls whether the sidebar is displayed (default: true for docs layout)
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty" toml:"enabled,omitempty"`

	// Position controls sidebar placement: "left" or "right" (default: "left")
	Position string `json:"position,omitempty" yaml:"position,omitempty" toml:"position,omitempty"`

	// Width is the sidebar width (default: "280px")
	Width string `json:"width,omitempty" yaml:"width,omitempty" toml:"width,omitempty"`

	// Collapsible allows the sidebar to be collapsed (default: true)
	Collapsible *bool `json:"collapsible,omitempty" yaml:"collapsible,omitempty" toml:"collapsible,omitempty"`

	// DefaultOpen controls if sidebar is open by default (default: true)
	DefaultOpen *bool `json:"default_open,omitempty" yaml:"default_open,omitempty" toml:"default_open,omitempty"`

	// Nav is the navigation structure (auto-generated from feeds if not specified)
	Nav []SidebarNavItem `json:"nav,omitempty" yaml:"nav,omitempty" toml:"nav,omitempty"`

	// Title is the optional sidebar title/header
	Title string `json:"title,omitempty" yaml:"title,omitempty" toml:"title,omitempty"`

	// Paths maps URL path prefixes to path-specific sidebar configs
	// Keys should be paths like "/docs/", "/blog/", "/guides/"
	Paths map[string]*PathSidebarConfig `json:"paths,omitempty" yaml:"paths,omitempty" toml:"paths,omitempty"`

	// MultiFeed enables multi-feed mode with collapsible sections
	MultiFeed *bool `json:"multi_feed,omitempty" yaml:"multi_feed,omitempty" toml:"multi_feed,omitempty"`

	// Feeds is the list of feed slugs to show in multi-feed mode
	Feeds []string `json:"feeds,omitempty" yaml:"feeds,omitempty" toml:"feeds,omitempty"`

	// FeedSections provides detailed config for multi-feed sections
	FeedSections []MultiFeedSection `json:"feed_sections,omitempty" yaml:"feed_sections,omitempty" toml:"feed_sections,omitempty"`

	// AutoGenerate configures default auto-generation settings
	AutoGenerate *SidebarAutoGenerate `json:"auto_generate,omitempty" yaml:"auto_generate,omitempty" toml:"auto_generate,omitempty"`
}

// IsEnabled returns whether the sidebar is enabled.
func (s *SidebarConfig) IsEnabled() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

// IsCollapsible returns whether the sidebar can be collapsed.
func (s *SidebarConfig) IsCollapsible() bool {
	if s.Collapsible == nil {
		return true
	}
	return *s.Collapsible
}

// IsDefaultOpen returns whether the sidebar is open by default.
func (s *SidebarConfig) IsDefaultOpen() bool {
	if s.DefaultOpen == nil {
		return true
	}
	return *s.DefaultOpen
}

// IsMultiFeed returns whether multi-feed mode is enabled.
func (s *SidebarConfig) IsMultiFeed() bool {
	if s.MultiFeed == nil {
		return false
	}
	return *s.MultiFeed
}

// ResolveForPath finds the best matching sidebar configuration for a given path.
// It checks path-specific sidebars and returns the most specific match (longest prefix wins).
// Returns the matching PathSidebarConfig and true if found, or nil and false otherwise.
func (s *SidebarConfig) ResolveForPath(path string) (*PathSidebarConfig, bool) {
	if len(s.Paths) == 0 {
		return nil, false
	}

	var bestMatch string
	var bestConfig *PathSidebarConfig

	for pathPrefix, config := range s.Paths {
		if strings.HasPrefix(path, pathPrefix) {
			if len(pathPrefix) > len(bestMatch) {
				bestMatch = pathPrefix
				bestConfig = config
			}
		}
	}

	return bestConfig, bestConfig != nil
}

// GetEffectiveConfig returns an effective sidebar configuration for a path,
// merging path-specific settings with the default sidebar config.
func (s *SidebarConfig) GetEffectiveConfig(path string) *SidebarConfig {
	pathConfig, found := s.ResolveForPath(path)
	if !found {
		return s
	}

	// Create effective config by merging defaults with path-specific settings
	effective := &SidebarConfig{
		Enabled:     s.Enabled,
		Position:    s.Position,
		Width:       s.Width,
		Collapsible: s.Collapsible,
		DefaultOpen: s.DefaultOpen,
		Title:       pathConfig.Title,
		Nav:         pathConfig.Items,
	}

	// Override with path-specific values if set
	if pathConfig.Position != "" {
		effective.Position = pathConfig.Position
	}
	if pathConfig.Collapsible != nil {
		effective.Collapsible = pathConfig.Collapsible
	}

	return effective
}

// SidebarAutoGenerate configures automatic sidebar generation from a directory or feed.
type SidebarAutoGenerate struct {
	// Directory is the source directory for auto-generation (relative to content root)
	Directory string `json:"directory,omitempty" yaml:"directory,omitempty" toml:"directory,omitempty"`

	// OrderBy specifies how to order items: "title", "date", "nav_order", "filename" (default: "filename")
	OrderBy string `json:"order_by,omitempty" yaml:"order_by,omitempty" toml:"order_by,omitempty"`

	// Reverse reverses the sort order (default: false)
	Reverse *bool `json:"reverse,omitempty" yaml:"reverse,omitempty" toml:"reverse,omitempty"`

	// MaxDepth limits how deep to recurse into subdirectories (0 = unlimited, default: 0)
	MaxDepth int `json:"max_depth,omitempty" yaml:"max_depth,omitempty" toml:"max_depth,omitempty"`

	// Exclude is a list of glob patterns to exclude from auto-generation
	Exclude []string `json:"exclude,omitempty" yaml:"exclude,omitempty" toml:"exclude,omitempty"`
}

// IsReverse returns whether to reverse the sort order.
func (a *SidebarAutoGenerate) IsReverse() bool {
	if a.Reverse == nil {
		return false
	}
	return *a.Reverse
}

// PathSidebarConfig configures a sidebar for a specific URL path prefix.
type PathSidebarConfig struct {
	// Title is the optional sidebar title/header for this path
	Title string `json:"title,omitempty" yaml:"title,omitempty" toml:"title,omitempty"`

	// AutoGenerate configures auto-generation from directory structure
	AutoGenerate *SidebarAutoGenerate `json:"auto_generate,omitempty" yaml:"auto_generate,omitempty" toml:"auto_generate,omitempty"`

	// Items is the manual navigation structure for this path
	Items []SidebarNavItem `json:"items,omitempty" yaml:"items,omitempty" toml:"items,omitempty"`

	// Feed links this sidebar to a specific feed slug for auto-generation
	Feed string `json:"feed,omitempty" yaml:"feed,omitempty" toml:"feed,omitempty"`

	// Position overrides the default sidebar position for this path
	Position string `json:"position,omitempty" yaml:"position,omitempty" toml:"position,omitempty"`

	// Collapsible overrides the default collapsible setting for this path
	Collapsible *bool `json:"collapsible,omitempty" yaml:"collapsible,omitempty" toml:"collapsible,omitempty"`
}

// IsCollapsible returns whether the path sidebar is collapsible.
func (p *PathSidebarConfig) IsCollapsible() bool {
	if p.Collapsible == nil {
		return true
	}
	return *p.Collapsible
}

// MultiFeedSection represents a section in a multi-feed sidebar.
type MultiFeedSection struct {
	// Feed is the feed slug to include in this section
	Feed string `json:"feed" yaml:"feed" toml:"feed"`

	// Title overrides the feed title for this section
	Title string `json:"title,omitempty" yaml:"title,omitempty" toml:"title,omitempty"`

	// Collapsed starts this section collapsed (default: false for first, true for others)
	Collapsed *bool `json:"collapsed,omitempty" yaml:"collapsed,omitempty" toml:"collapsed,omitempty"`

	// MaxItems limits the number of items shown (0 = unlimited, default: 0)
	MaxItems int `json:"max_items,omitempty" yaml:"max_items,omitempty" toml:"max_items,omitempty"`
}

// IsCollapsed returns whether the section should start collapsed.
func (m *MultiFeedSection) IsCollapsed() bool {
	if m.Collapsed == nil {
		return false
	}
	return *m.Collapsed
}

// TocConfig configures the table of contents component.
type TocConfig struct {
	// Enabled controls whether the TOC is displayed (default: true for docs layout)
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty" toml:"enabled,omitempty"`

	// Position controls TOC placement: "left" or "right" (default: "right")
	Position string `json:"position,omitempty" yaml:"position,omitempty" toml:"position,omitempty"`

	// Width is the TOC width (default: "220px")
	Width string `json:"width,omitempty" yaml:"width,omitempty" toml:"width,omitempty"`

	// MinDepth is the minimum heading level to include (default: 2)
	MinDepth int `json:"min_depth,omitempty" yaml:"min_depth,omitempty" toml:"min_depth,omitempty"`

	// MaxDepth is the maximum heading level to include (default: 4)
	MaxDepth int `json:"max_depth,omitempty" yaml:"max_depth,omitempty" toml:"max_depth,omitempty"`

	// Title is the TOC section title (default: "On this page")
	Title string `json:"title,omitempty" yaml:"title,omitempty" toml:"title,omitempty"`

	// Collapsible allows the TOC to be collapsed (default: true)
	Collapsible *bool `json:"collapsible,omitempty" yaml:"collapsible,omitempty" toml:"collapsible,omitempty"`

	// DefaultOpen controls if TOC is open by default (default: true)
	DefaultOpen *bool `json:"default_open,omitempty" yaml:"default_open,omitempty" toml:"default_open,omitempty"`

	// ScrollSpy enables highlighting the current section (default: true)
	ScrollSpy *bool `json:"scroll_spy,omitempty" yaml:"scroll_spy,omitempty" toml:"scroll_spy,omitempty"`
}

// IsEnabled returns whether the TOC is enabled.
func (t *TocConfig) IsEnabled() bool {
	if t.Enabled == nil {
		return true
	}
	return *t.Enabled
}

// IsCollapsible returns whether the TOC can be collapsed.
func (t *TocConfig) IsCollapsible() bool {
	if t.Collapsible == nil {
		return true
	}
	return *t.Collapsible
}

// IsDefaultOpen returns whether the TOC is open by default.
func (t *TocConfig) IsDefaultOpen() bool {
	if t.DefaultOpen == nil {
		return true
	}
	return *t.DefaultOpen
}

// IsScrollSpy returns whether scroll spy is enabled.
func (t *TocConfig) IsScrollSpy() bool {
	if t.ScrollSpy == nil {
		return true
	}
	return *t.ScrollSpy
}

// HeaderLayoutConfig configures the header component for layouts.
type HeaderLayoutConfig struct {
	// Style controls the header appearance: "full", "minimal", "transparent", "none" (default: "full")
	Style string `json:"style,omitempty" yaml:"style,omitempty" toml:"style,omitempty"`

	// Sticky makes the header stick to the top when scrolling (default: true)
	Sticky *bool `json:"sticky,omitempty" yaml:"sticky,omitempty" toml:"sticky,omitempty"`

	// ShowLogo displays the site logo (default: true)
	ShowLogo *bool `json:"show_logo,omitempty" yaml:"show_logo,omitempty" toml:"show_logo,omitempty"`

	// ShowTitle displays the site title (default: true)
	ShowTitle *bool `json:"show_title,omitempty" yaml:"show_title,omitempty" toml:"show_title,omitempty"`

	// ShowNav displays the navigation links (default: true)
	ShowNav *bool `json:"show_nav,omitempty" yaml:"show_nav,omitempty" toml:"show_nav,omitempty"`

	// ShowSearch displays the search box (default: true)
	ShowSearch *bool `json:"show_search,omitempty" yaml:"show_search,omitempty" toml:"show_search,omitempty"`

	// ShowThemeToggle displays the theme toggle button (default: true)
	ShowThemeToggle *bool `json:"show_theme_toggle,omitempty" yaml:"show_theme_toggle,omitempty" toml:"show_theme_toggle,omitempty"`
}

// IsSticky returns whether the header should be sticky.
func (h *HeaderLayoutConfig) IsSticky() bool {
	if h.Sticky == nil {
		return true
	}
	return *h.Sticky
}

// IsShowLogo returns whether to show the logo.
func (h *HeaderLayoutConfig) IsShowLogo() bool {
	if h.ShowLogo == nil {
		return true
	}
	return *h.ShowLogo
}

// IsShowTitle returns whether to show the title.
func (h *HeaderLayoutConfig) IsShowTitle() bool {
	if h.ShowTitle == nil {
		return true
	}
	return *h.ShowTitle
}

// IsShowNav returns whether to show navigation.
func (h *HeaderLayoutConfig) IsShowNav() bool {
	if h.ShowNav == nil {
		return true
	}
	return *h.ShowNav
}

// IsShowSearch returns whether to show the search box.
func (h *HeaderLayoutConfig) IsShowSearch() bool {
	if h.ShowSearch == nil {
		return true
	}
	return *h.ShowSearch
}

// IsShowThemeToggle returns whether to show the theme toggle.
func (h *HeaderLayoutConfig) IsShowThemeToggle() bool {
	if h.ShowThemeToggle == nil {
		return true
	}
	return *h.ShowThemeToggle
}

// FooterLayoutConfig configures the footer component for layouts.
type FooterLayoutConfig struct {
	// Style controls the footer appearance: "full", "minimal", "none" (default: "full")
	Style string `json:"style,omitempty" yaml:"style,omitempty" toml:"style,omitempty"`

	// Sticky makes the footer stick to the bottom (default: false)
	Sticky *bool `json:"sticky,omitempty" yaml:"sticky,omitempty" toml:"sticky,omitempty"`

	// ShowCopyright displays the copyright notice (default: true)
	ShowCopyright *bool `json:"show_copyright,omitempty" yaml:"show_copyright,omitempty" toml:"show_copyright,omitempty"`

	// CopyrightText is the copyright text (default: auto-generated)
	CopyrightText string `json:"copyright_text,omitempty" yaml:"copyright_text,omitempty" toml:"copyright_text,omitempty"`

	// ShowSocialLinks displays social media links (default: true)
	ShowSocialLinks *bool `json:"show_social_links,omitempty" yaml:"show_social_links,omitempty" toml:"show_social_links,omitempty"`

	// ShowNavLinks displays navigation links in footer (default: true)
	ShowNavLinks *bool `json:"show_nav_links,omitempty" yaml:"show_nav_links,omitempty" toml:"show_nav_links,omitempty"`
}

// IsSticky returns whether the footer should be sticky.
func (f *FooterLayoutConfig) IsSticky() bool {
	if f.Sticky == nil {
		return false
	}
	return *f.Sticky
}

// IsShowCopyright returns whether to show the copyright.
func (f *FooterLayoutConfig) IsShowCopyright() bool {
	if f.ShowCopyright == nil {
		return true
	}
	return *f.ShowCopyright
}

// IsShowSocialLinks returns whether to show social links.
func (f *FooterLayoutConfig) IsShowSocialLinks() bool {
	if f.ShowSocialLinks == nil {
		return true
	}
	return *f.ShowSocialLinks
}

// IsShowNavLinks returns whether to show nav links in footer.
func (f *FooterLayoutConfig) IsShowNavLinks() bool {
	if f.ShowNavLinks == nil {
		return true
	}
	return *f.ShowNavLinks
}

// NewLayoutConfig creates a new LayoutConfig with default values.
func NewLayoutConfig() LayoutConfig {
	sidebarCollapsible := true
	sidebarDefaultOpen := true
	tocCollapsible := true
	tocDefaultOpen := true
	headerSticky := true
	footerSticky := false
	heroEnabled := true
	showToc := false
	showAuthor := true
	showDate := true
	showTags := true
	showReadingTime := true
	showPrevNext := true

	return LayoutConfig{
		Name:  layoutBlog,
		Paths: make(map[string]string),
		Feeds: make(map[string]string),
		Docs: DocsLayoutConfig{
			SidebarPosition:    "left",
			SidebarWidth:       "280px",
			SidebarCollapsible: &sidebarCollapsible,
			SidebarDefaultOpen: &sidebarDefaultOpen,
			TocPosition:        "right",
			TocWidth:           "220px",
			TocCollapsible:     &tocCollapsible,
			TocDefaultOpen:     &tocDefaultOpen,
			ContentMaxWidth:    "800px",
			HeaderStyle:        "minimal",
			FooterStyle:        "minimal",
		},
		Blog: BlogLayoutConfig{
			ContentMaxWidth: "720px",
			ShowToc:         &showToc,
			TocPosition:     "right",
			TocWidth:        "200px",
			HeaderStyle:     "full",
			FooterStyle:     "full",
			ShowAuthor:      &showAuthor,
			ShowDate:        &showDate,
			ShowTags:        &showTags,
			ShowReadingTime: &showReadingTime,
			ShowPrevNext:    &showPrevNext,
		},
		Landing: LandingLayoutConfig{
			ContentMaxWidth: "100%",
			HeaderStyle:     "transparent",
			HeaderSticky:    &headerSticky,
			FooterStyle:     "full",
			HeroEnabled:     &heroEnabled,
		},
		Bare: BareLayoutConfig{
			ContentMaxWidth: "100%",
		},
		Defaults: LayoutDefaults{
			ContentMaxWidth: "800px",
			HeaderSticky:    &headerSticky,
			FooterSticky:    &footerSticky,
		},
	}
}

// NewSidebarConfig creates a new SidebarConfig with default values.
func NewSidebarConfig() SidebarConfig {
	enabled := true
	collapsible := true
	defaultOpen := true

	return SidebarConfig{
		Enabled:     &enabled,
		Position:    "left",
		Width:       "280px",
		Collapsible: &collapsible,
		DefaultOpen: &defaultOpen,
		Nav:         []SidebarNavItem{},
	}
}

// NewTocConfig creates a new TocConfig with default values.
func NewTocConfig() TocConfig {
	enabled := true
	collapsible := true
	defaultOpen := true
	scrollSpy := true

	return TocConfig{
		Enabled:     &enabled,
		Position:    "right",
		Width:       "220px",
		MinDepth:    2,
		MaxDepth:    4,
		Title:       "On this page",
		Collapsible: &collapsible,
		DefaultOpen: &defaultOpen,
		ScrollSpy:   &scrollSpy,
	}
}

// NewHeaderLayoutConfig creates a new HeaderLayoutConfig with default values.
func NewHeaderLayoutConfig() HeaderLayoutConfig {
	sticky := true
	showLogo := true
	showTitle := true
	showNav := true
	showSearch := true
	showThemeToggle := true

	return HeaderLayoutConfig{
		Style:           "full",
		Sticky:          &sticky,
		ShowLogo:        &showLogo,
		ShowTitle:       &showTitle,
		ShowNav:         &showNav,
		ShowSearch:      &showSearch,
		ShowThemeToggle: &showThemeToggle,
	}
}

// NewFooterLayoutConfig creates a new FooterLayoutConfig with default values.
func NewFooterLayoutConfig() FooterLayoutConfig {
	sticky := false
	showCopyright := true
	showSocialLinks := true
	showNavLinks := true

	return FooterLayoutConfig{
		Style:           "full",
		Sticky:          &sticky,
		ShowCopyright:   &showCopyright,
		ShowSocialLinks: &showSocialLinks,
		ShowNavLinks:    &showNavLinks,
	}
}
