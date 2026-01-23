package models

// LayoutConfig configures the site layout system.
// Layouts control the overall page structure including sidebars, TOC, header, and footer.
type LayoutConfig struct {
	// Name is the default layout preset name (default: "blog")
	// Options: "docs", "blog", "landing", "bare"
	Name string `json:"name,omitempty" yaml:"name,omitempty" toml:"name,omitempty"`

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
		Name: "blog",
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
