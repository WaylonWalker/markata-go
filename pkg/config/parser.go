package config

import (
	"encoding/json"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

// configSource defines the interface for format-specific config structs (TOML, YAML, JSON)
// to provide data for building a models.Config.
type configSource interface {
	getBaseConfig() baseConfigData
	getNavItems() []navItemData
	getFeeds() []feedConfigConverter
	getFeedDefaults() feedDefaultsConverter
	getPostFormats() postFormatsConverter
	getSEO() seoConverter
	getIndieAuth() indieAuthConverter
	getWebmention() webmentionConverter
	getComponents() componentsConverter
	getLayout() layoutConverter
	getSidebar() sidebarConverter
	getToc() tocConverter
	getHeader() headerConverter
	getBlogroll() blogrollConverter
}

// baseConfigData holds the basic config fields that are directly assignable.
type baseConfigData struct {
	OutputDir     string
	URL           string
	Title         string
	Description   string
	Author        string
	AssetsDir     string
	TemplatesDir  string
	Hooks         []string
	DisabledHooks []string
	GlobPatterns  []string
	UseGitignore  *bool
	Extensions    []string
	Concurrency   int
	Theme         models.ThemeConfig
	Footer        models.FooterConfig
}

// navItemData holds nav item fields.
type navItemData struct {
	Label    string
	URL      string
	External bool
}

// Converter interfaces for nested config types.
type feedConfigConverter interface {
	toFeedConfig() models.FeedConfig
}

type feedDefaultsConverter interface {
	toFeedDefaults() models.FeedDefaults
}

type postFormatsConverter interface {
	toPostFormatsConfig() models.PostFormatsConfig
}

type seoConverter interface {
	toSEOConfig() models.SEOConfig
}

type indieAuthConverter interface {
	toIndieAuthConfig() models.IndieAuthConfig
}

type webmentionConverter interface {
	toWebmentionConfig() models.WebmentionConfig
}

type componentsConverter interface {
	toComponentsConfig() models.ComponentsConfig
}

type layoutConverter interface {
	toLayoutConfig() models.LayoutConfig
}

type sidebarConverter interface {
	toSidebarConfig() models.SidebarConfig
}

type tocConverter interface {
	toTocConfig() models.TocConfig
}

type headerConverter interface {
	toHeaderLayoutConfig() models.HeaderLayoutConfig
}

type blogrollConverter interface {
	toBlogrollConfig() models.BlogrollConfig
}

// buildConfig constructs a models.Config from a configSource.
// This helper eliminates code duplication across TOML, YAML, and JSON config converters.
func buildConfig(src configSource) *models.Config {
	base := src.getBaseConfig()
	config := &models.Config{
		OutputDir:     base.OutputDir,
		URL:           base.URL,
		Title:         base.Title,
		Description:   base.Description,
		Author:        base.Author,
		AssetsDir:     base.AssetsDir,
		TemplatesDir:  base.TemplatesDir,
		Hooks:         base.Hooks,
		DisabledHooks: base.DisabledHooks,
		GlobConfig: models.GlobConfig{
			Patterns: base.GlobPatterns,
		},
		MarkdownConfig: models.MarkdownConfig{
			Extensions: base.Extensions,
		},
		Concurrency: base.Concurrency,
		Theme:       base.Theme,
		Footer:      base.Footer,
	}

	if base.UseGitignore != nil {
		config.GlobConfig.UseGitignore = *base.UseGitignore
	}

	// Convert nav items
	for _, nav := range src.getNavItems() {
		config.Nav = append(config.Nav, models.NavItem{
			Label:    nav.Label,
			URL:      nav.URL,
			External: nav.External,
		})
	}

	// Convert feeds
	for _, feed := range src.getFeeds() {
		config.Feeds = append(config.Feeds, feed.toFeedConfig())
	}

	// Convert feed defaults
	config.FeedDefaults = src.getFeedDefaults().toFeedDefaults()

	// Convert post formats
	config.PostFormats = src.getPostFormats().toPostFormatsConfig()

	// Convert SEO config
	config.SEO = src.getSEO().toSEOConfig()

	// Convert IndieAuth and Webmention config
	config.IndieAuth = src.getIndieAuth().toIndieAuthConfig()
	config.Webmention = src.getWebmention().toWebmentionConfig()

	// Convert Components config
	config.Components = src.getComponents().toComponentsConfig()

	// Convert Layout config
	config.Layout = src.getLayout().toLayoutConfig()

	// Convert Sidebar config
	config.Sidebar = src.getSidebar().toSidebarConfig()

	// Convert Toc config
	config.Toc = src.getToc().toTocConfig()

	// Convert Header config
	config.Header = src.getHeader().toHeaderLayoutConfig()

	// Convert Blogroll config
	config.Blogroll = src.getBlogroll().toBlogrollConfig()

	return config
}

// ParseTOML parses TOML configuration data into a Config struct.
// The TOML data is expected to have a top-level [markata-go] section.
func ParseTOML(data []byte) (*models.Config, error) {
	// Wrapper struct for the markata-go section
	var wrapper struct {
		MarkataGo tomlConfig `toml:"markata-go"`
	}

	if err := toml.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}

	return wrapper.MarkataGo.toConfig(), nil
}

// ParseYAML parses YAML configuration data into a Config struct.
// The YAML data is expected to have a top-level markata-go key.
func ParseYAML(data []byte) (*models.Config, error) {
	// Wrapper struct for the markata-go section
	var wrapper struct {
		MarkataGo yamlConfig `yaml:"markata-go"`
	}

	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}

	return wrapper.MarkataGo.toConfig(), nil
}

// ParseJSON parses JSON configuration data into a Config struct.
// The JSON data is expected to have a top-level "markata-go" key.
func ParseJSON(data []byte) (*models.Config, error) {
	// Wrapper struct for the markata-go section
	var wrapper struct {
		MarkataGo jsonConfig `json:"markata-go"`
	}

	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}

	return wrapper.MarkataGo.toConfig(), nil
}

// tomlConfig is an internal struct for parsing TOML configuration.
type tomlConfig struct {
	OutputDir     string                 `toml:"output_dir"`
	URL           string                 `toml:"url"`
	Title         string                 `toml:"title"`
	Description   string                 `toml:"description"`
	Author        string                 `toml:"author"`
	AssetsDir     string                 `toml:"assets_dir"`
	TemplatesDir  string                 `toml:"templates_dir"`
	Nav           []tomlNavItem          `toml:"nav"`
	Footer        tomlFooterConfig       `toml:"footer"`
	Hooks         []string               `toml:"hooks"`
	DisabledHooks []string               `toml:"disabled_hooks"`
	Glob          tomlGlobConfig         `toml:"glob"`
	Markdown      tomlMarkdownConfig     `toml:"markdown"`
	Feeds         []tomlFeedConfig       `toml:"feeds"`
	FeedDefaults  tomlFeedDefaults       `toml:"feed_defaults"`
	Concurrency   int                    `toml:"concurrency"`
	Theme         tomlThemeConfig        `toml:"theme"`
	PostFormats   tomlPostFormatsConfig  `toml:"post_formats"`
	SEO           tomlSEOConfig          `toml:"seo"`
	IndieAuth     tomlIndieAuthConfig    `toml:"indieauth"`
	Webmention    tomlWebmentionConfig   `toml:"webmention"`
	Components    tomlComponentsConfig   `toml:"components"`
	Layout        tomlLayoutConfig       `toml:"layout"`
	Sidebar       tomlSidebarConfig      `toml:"sidebar"`
	Toc           tomlTocConfig          `toml:"toc"`
	Header        tomlHeaderLayoutConfig `toml:"header"`
	Blogroll      tomlBlogrollConfig     `toml:"blogroll"`
	UnknownFields map[string]any         `toml:"-"`
}

type tomlNavItem struct {
	Label    string `toml:"label"`
	URL      string `toml:"url"`
	External bool   `toml:"external"`
}

type tomlFooterConfig struct {
	Text          string `toml:"text"`
	ShowCopyright *bool  `toml:"show_copyright"`
}

type tomlThemeConfig struct {
	Name         string            `toml:"name"`
	Palette      string            `toml:"palette"`
	PaletteLight string            `toml:"palette_light"`
	PaletteDark  string            `toml:"palette_dark"`
	Variables    map[string]string `toml:"variables"`
	CustomCSS    string            `toml:"custom_css"`
}

type tomlGlobConfig struct {
	Patterns     []string `toml:"patterns"`
	UseGitignore *bool    `toml:"use_gitignore"`
}

type tomlMarkdownConfig struct {
	Extensions []string `toml:"extensions"`
}

type tomlFeedConfig struct {
	Slug            string            `toml:"slug"`
	Title           string            `toml:"title"`
	Description     string            `toml:"description"`
	Filter          string            `toml:"filter"`
	Sort            string            `toml:"sort"`
	Reverse         bool              `toml:"reverse"`
	ItemsPerPage    int               `toml:"items_per_page"`
	OrphanThreshold int               `toml:"orphan_threshold"`
	PaginationType  string            `toml:"pagination_type"`
	Formats         tomlFeedFormats   `toml:"formats"`
	Templates       tomlFeedTemplates `toml:"templates"`
}

type tomlFeedFormats struct {
	HTML     *bool `toml:"html"`
	RSS      *bool `toml:"rss"`
	Atom     *bool `toml:"atom"`
	JSON     *bool `toml:"json"`
	Markdown *bool `toml:"markdown"`
	Text     *bool `toml:"text"`
	Sitemap  *bool `toml:"sitemap"`
}

type tomlFeedTemplates struct {
	HTML    string `toml:"html"`
	RSS     string `toml:"rss"`
	Atom    string `toml:"atom"`
	JSON    string `toml:"json"`
	Card    string `toml:"card"`
	Sitemap string `toml:"sitemap"`
}

type tomlFeedDefaults struct {
	ItemsPerPage    int                   `toml:"items_per_page"`
	OrphanThreshold int                   `toml:"orphan_threshold"`
	PaginationType  string                `toml:"pagination_type"`
	Formats         tomlFeedFormats       `toml:"formats"`
	Templates       tomlFeedTemplates     `toml:"templates"`
	Syndication     tomlSyndicationConfig `toml:"syndication"`
}

type tomlSyndicationConfig struct {
	MaxItems       int  `toml:"max_items"`
	IncludeContent bool `toml:"include_content"`
}

type tomlPostFormatsConfig struct {
	HTML     *bool `toml:"html"`
	Markdown bool  `toml:"markdown"`
	Text     bool  `toml:"text"`
	OG       bool  `toml:"og"`
}

type tomlSEOConfig struct {
	TwitterHandle string `toml:"twitter_handle"`
	DefaultImage  string `toml:"default_image"`
	LogoURL       string `toml:"logo_url"`
}

type tomlIndieAuthConfig struct {
	Enabled               bool   `toml:"enabled"`
	AuthorizationEndpoint string `toml:"authorization_endpoint"`
	TokenEndpoint         string `toml:"token_endpoint"`
	MeURL                 string `toml:"me_url"`
}

type tomlWebmentionConfig struct {
	Enabled  bool   `toml:"enabled"`
	Endpoint string `toml:"endpoint"`
}

func (s *tomlSEOConfig) toSEOConfig() models.SEOConfig {
	return models.SEOConfig{
		TwitterHandle: s.TwitterHandle,
		DefaultImage:  s.DefaultImage,
		LogoURL:       s.LogoURL,
	}
}

func (i *tomlIndieAuthConfig) toIndieAuthConfig() models.IndieAuthConfig {
	return models.IndieAuthConfig{
		Enabled:               i.Enabled,
		AuthorizationEndpoint: i.AuthorizationEndpoint,
		TokenEndpoint:         i.TokenEndpoint,
		MeURL:                 i.MeURL,
	}
}

func (w *tomlWebmentionConfig) toWebmentionConfig() models.WebmentionConfig {
	return models.WebmentionConfig{
		Enabled:  w.Enabled,
		Endpoint: w.Endpoint,
	}
}

type tomlComponentsConfig struct {
	Nav        tomlNavComponentConfig    `toml:"nav"`
	Footer     tomlFooterComponentConfig `toml:"footer"`
	DocSidebar tomlDocSidebarConfig      `toml:"doc_sidebar"`
}

type tomlNavComponentConfig struct {
	Enabled  *bool         `toml:"enabled"`
	Position string        `toml:"position"`
	Style    string        `toml:"style"`
	Items    []tomlNavItem `toml:"items"`
}

type tomlFooterComponentConfig struct {
	Enabled       *bool         `toml:"enabled"`
	Text          string        `toml:"text"`
	ShowCopyright *bool         `toml:"show_copyright"`
	Links         []tomlNavItem `toml:"links"`
}

type tomlDocSidebarConfig struct {
	Enabled  *bool  `toml:"enabled"`
	Position string `toml:"position"`
	Width    string `toml:"width"`
	MinDepth int    `toml:"min_depth"`
	MaxDepth int    `toml:"max_depth"`
}

// Layout-related TOML structs

type tomlLayoutConfig struct {
	Name     string                  `toml:"name"`
	Paths    map[string]string       `toml:"paths"`
	Feeds    map[string]string       `toml:"feeds"`
	Docs     tomlDocsLayoutConfig    `toml:"docs"`
	Blog     tomlBlogLayoutConfig    `toml:"blog"`
	Landing  tomlLandingLayoutConfig `toml:"landing"`
	Bare     tomlBareLayoutConfig    `toml:"bare"`
	Defaults tomlLayoutDefaults      `toml:"defaults"`
}

type tomlLayoutDefaults struct {
	ContentMaxWidth string `toml:"content_max_width"`
	HeaderSticky    *bool  `toml:"header_sticky"`
	FooterSticky    *bool  `toml:"footer_sticky"`
}

type tomlDocsLayoutConfig struct {
	SidebarPosition    string `toml:"sidebar_position"`
	SidebarWidth       string `toml:"sidebar_width"`
	SidebarCollapsible *bool  `toml:"sidebar_collapsible"`
	SidebarDefaultOpen *bool  `toml:"sidebar_default_open"`
	TocPosition        string `toml:"toc_position"`
	TocWidth           string `toml:"toc_width"`
	TocCollapsible     *bool  `toml:"toc_collapsible"`
	TocDefaultOpen     *bool  `toml:"toc_default_open"`
	ContentMaxWidth    string `toml:"content_max_width"`
	HeaderStyle        string `toml:"header_style"`
	FooterStyle        string `toml:"footer_style"`
}

type tomlBlogLayoutConfig struct {
	ContentMaxWidth string `toml:"content_max_width"`
	ShowToc         *bool  `toml:"show_toc"`
	TocPosition     string `toml:"toc_position"`
	TocWidth        string `toml:"toc_width"`
	HeaderStyle     string `toml:"header_style"`
	FooterStyle     string `toml:"footer_style"`
	ShowAuthor      *bool  `toml:"show_author"`
	ShowDate        *bool  `toml:"show_date"`
	ShowTags        *bool  `toml:"show_tags"`
	ShowReadingTime *bool  `toml:"show_reading_time"`
	ShowPrevNext    *bool  `toml:"show_prev_next"`
}

type tomlLandingLayoutConfig struct {
	ContentMaxWidth string `toml:"content_max_width"`
	HeaderStyle     string `toml:"header_style"`
	HeaderSticky    *bool  `toml:"header_sticky"`
	FooterStyle     string `toml:"footer_style"`
	HeroEnabled     *bool  `toml:"hero_enabled"`
}

type tomlBareLayoutConfig struct {
	ContentMaxWidth string `toml:"content_max_width"`
}

func (l *tomlLayoutConfig) toLayoutConfig() models.LayoutConfig {
	return models.LayoutConfig{
		Name:  l.Name,
		Paths: l.Paths,
		Feeds: l.Feeds,
		Docs: models.DocsLayoutConfig{
			SidebarPosition:    l.Docs.SidebarPosition,
			SidebarWidth:       l.Docs.SidebarWidth,
			SidebarCollapsible: l.Docs.SidebarCollapsible,
			SidebarDefaultOpen: l.Docs.SidebarDefaultOpen,
			TocPosition:        l.Docs.TocPosition,
			TocWidth:           l.Docs.TocWidth,
			TocCollapsible:     l.Docs.TocCollapsible,
			TocDefaultOpen:     l.Docs.TocDefaultOpen,
			ContentMaxWidth:    l.Docs.ContentMaxWidth,
			HeaderStyle:        l.Docs.HeaderStyle,
			FooterStyle:        l.Docs.FooterStyle,
		},
		Blog: models.BlogLayoutConfig{
			ContentMaxWidth: l.Blog.ContentMaxWidth,
			ShowToc:         l.Blog.ShowToc,
			TocPosition:     l.Blog.TocPosition,
			TocWidth:        l.Blog.TocWidth,
			HeaderStyle:     l.Blog.HeaderStyle,
			FooterStyle:     l.Blog.FooterStyle,
			ShowAuthor:      l.Blog.ShowAuthor,
			ShowDate:        l.Blog.ShowDate,
			ShowTags:        l.Blog.ShowTags,
			ShowReadingTime: l.Blog.ShowReadingTime,
			ShowPrevNext:    l.Blog.ShowPrevNext,
		},
		Landing: models.LandingLayoutConfig{
			ContentMaxWidth: l.Landing.ContentMaxWidth,
			HeaderStyle:     l.Landing.HeaderStyle,
			HeaderSticky:    l.Landing.HeaderSticky,
			FooterStyle:     l.Landing.FooterStyle,
			HeroEnabled:     l.Landing.HeroEnabled,
		},
		Bare: models.BareLayoutConfig{
			ContentMaxWidth: l.Bare.ContentMaxWidth,
		},
		Defaults: models.LayoutDefaults{
			ContentMaxWidth: l.Defaults.ContentMaxWidth,
			HeaderSticky:    l.Defaults.HeaderSticky,
			FooterSticky:    l.Defaults.FooterSticky,
		},
	}
}

// Sidebar-related TOML structs

type tomlSidebarConfig struct {
	Enabled      *bool                             `toml:"enabled"`
	Position     string                            `toml:"position"`
	Width        string                            `toml:"width"`
	Collapsible  *bool                             `toml:"collapsible"`
	DefaultOpen  *bool                             `toml:"default_open"`
	Nav          []tomlSidebarNavItem              `toml:"nav"`
	Title        string                            `toml:"title"`
	Paths        map[string]*tomlPathSidebarConfig `toml:"paths"`
	MultiFeed    *bool                             `toml:"multi_feed"`
	Feeds        []string                          `toml:"feeds"`
	FeedSections []tomlMultiFeedSection            `toml:"feed_sections"`
	AutoGenerate *tomlSidebarAutoGenerate          `toml:"auto_generate"`
}

type tomlSidebarNavItem struct {
	Title    string               `toml:"title"`
	Href     string               `toml:"href"`
	Children []tomlSidebarNavItem `toml:"children"`
}

type tomlPathSidebarConfig struct {
	Title        string                   `toml:"title"`
	AutoGenerate *tomlSidebarAutoGenerate `toml:"auto_generate"`
	Items        []tomlSidebarNavItem     `toml:"items"`
	Feed         string                   `toml:"feed"`
	Position     string                   `toml:"position"`
	Collapsible  *bool                    `toml:"collapsible"`
}

type tomlSidebarAutoGenerate struct {
	Directory string   `toml:"directory"`
	OrderBy   string   `toml:"order_by"`
	Reverse   *bool    `toml:"reverse"`
	MaxDepth  int      `toml:"max_depth"`
	Exclude   []string `toml:"exclude"`
}

type tomlMultiFeedSection struct {
	Feed      string `toml:"feed"`
	Title     string `toml:"title"`
	Collapsed *bool  `toml:"collapsed"`
	MaxItems  int    `toml:"max_items"`
}

func convertTomlSidebarNavItems(items []tomlSidebarNavItem) []models.SidebarNavItem {
	result := make([]models.SidebarNavItem, len(items))
	for i, item := range items {
		result[i] = models.SidebarNavItem{
			Title:    item.Title,
			Href:     item.Href,
			Children: convertTomlSidebarNavItems(item.Children),
		}
	}
	return result
}

func (s *tomlSidebarConfig) toSidebarConfig() models.SidebarConfig {
	config := models.SidebarConfig{
		Enabled:     s.Enabled,
		Position:    s.Position,
		Width:       s.Width,
		Collapsible: s.Collapsible,
		DefaultOpen: s.DefaultOpen,
		Nav:         convertTomlSidebarNavItems(s.Nav),
		Title:       s.Title,
		MultiFeed:   s.MultiFeed,
		Feeds:       s.Feeds,
	}

	// Convert paths
	if len(s.Paths) > 0 {
		config.Paths = make(map[string]*models.PathSidebarConfig)
		for path, pathConfig := range s.Paths {
			var autoGen *models.SidebarAutoGenerate
			if pathConfig.AutoGenerate != nil {
				autoGen = &models.SidebarAutoGenerate{
					Directory: pathConfig.AutoGenerate.Directory,
					OrderBy:   pathConfig.AutoGenerate.OrderBy,
					Reverse:   pathConfig.AutoGenerate.Reverse,
					MaxDepth:  pathConfig.AutoGenerate.MaxDepth,
					Exclude:   pathConfig.AutoGenerate.Exclude,
				}
			}
			config.Paths[path] = &models.PathSidebarConfig{
				Title:        pathConfig.Title,
				AutoGenerate: autoGen,
				Items:        convertTomlSidebarNavItems(pathConfig.Items),
				Feed:         pathConfig.Feed,
				Position:     pathConfig.Position,
				Collapsible:  pathConfig.Collapsible,
			}
		}
	}

	// Convert feed sections
	if len(s.FeedSections) > 0 {
		config.FeedSections = make([]models.MultiFeedSection, len(s.FeedSections))
		for i, section := range s.FeedSections {
			config.FeedSections[i] = models.MultiFeedSection{
				Feed:      section.Feed,
				Title:     section.Title,
				Collapsed: section.Collapsed,
				MaxItems:  section.MaxItems,
			}
		}
	}

	// Convert auto-generate
	if s.AutoGenerate != nil {
		config.AutoGenerate = &models.SidebarAutoGenerate{
			Directory: s.AutoGenerate.Directory,
			OrderBy:   s.AutoGenerate.OrderBy,
			Reverse:   s.AutoGenerate.Reverse,
			MaxDepth:  s.AutoGenerate.MaxDepth,
			Exclude:   s.AutoGenerate.Exclude,
		}
	}

	return config
}

// TOC-related TOML structs

type tomlTocConfig struct {
	Enabled     *bool  `toml:"enabled"`
	Position    string `toml:"position"`
	Width       string `toml:"width"`
	MinDepth    int    `toml:"min_depth"`
	MaxDepth    int    `toml:"max_depth"`
	Title       string `toml:"title"`
	Collapsible *bool  `toml:"collapsible"`
	DefaultOpen *bool  `toml:"default_open"`
	ScrollSpy   *bool  `toml:"scroll_spy"`
}

func (t *tomlTocConfig) toTocConfig() models.TocConfig {
	return models.TocConfig{
		Enabled:     t.Enabled,
		Position:    t.Position,
		Width:       t.Width,
		MinDepth:    t.MinDepth,
		MaxDepth:    t.MaxDepth,
		Title:       t.Title,
		Collapsible: t.Collapsible,
		DefaultOpen: t.DefaultOpen,
		ScrollSpy:   t.ScrollSpy,
	}
}

// Header layout TOML structs

type tomlHeaderLayoutConfig struct {
	Style           string `toml:"style"`
	Sticky          *bool  `toml:"sticky"`
	ShowLogo        *bool  `toml:"show_logo"`
	ShowTitle       *bool  `toml:"show_title"`
	ShowNav         *bool  `toml:"show_nav"`
	ShowSearch      *bool  `toml:"show_search"`
	ShowThemeToggle *bool  `toml:"show_theme_toggle"`
}

func (h *tomlHeaderLayoutConfig) toHeaderLayoutConfig() models.HeaderLayoutConfig {
	return models.HeaderLayoutConfig{
		Style:           h.Style,
		Sticky:          h.Sticky,
		ShowLogo:        h.ShowLogo,
		ShowTitle:       h.ShowTitle,
		ShowNav:         h.ShowNav,
		ShowSearch:      h.ShowSearch,
		ShowThemeToggle: h.ShowThemeToggle,
	}
}

// Blogroll-related TOML structs

type tomlBlogrollConfig struct {
	Enabled              bool                     `toml:"enabled"`
	BlogrollSlug         string                   `toml:"blogroll_slug"`
	ReaderSlug           string                   `toml:"reader_slug"`
	CacheDir             string                   `toml:"cache_dir"`
	CacheDuration        string                   `toml:"cache_duration"`
	Timeout              int                      `toml:"timeout"`
	ConcurrentRequests   int                      `toml:"concurrent_requests"`
	MaxEntriesPerFeed    int                      `toml:"max_entries_per_feed"`
	FallbackImageService string                   `toml:"fallback_image_service"`
	Feeds                []tomlExternalFeedConfig `toml:"feeds"`
	Templates            tomlBlogrollTemplates    `toml:"templates"`
}

type tomlExternalFeedConfig struct {
	URL           string   `toml:"url"`
	Title         string   `toml:"title"`
	Description   string   `toml:"description"`
	Category      string   `toml:"category"`
	Tags          []string `toml:"tags"`
	Active        *bool    `toml:"active"`
	SiteURL       string   `toml:"site_url"`
	ImageURL      string   `toml:"image_url"`
	Handle        string   `toml:"handle"`
	Aliases       []string `toml:"aliases,omitempty"`
	MaxEntries    *int     `toml:"max_entries,omitempty"`
	Primary       *bool    `toml:"primary,omitempty"`
	PrimaryPerson string   `toml:"primary_person"`
}

type tomlBlogrollTemplates struct {
	Blogroll string `toml:"blogroll"`
	Reader   string `toml:"reader"`
}

//nolint:dupl // Intentional duplication - each format has its own conversion method
func (b *tomlBlogrollConfig) toBlogrollConfig() models.BlogrollConfig {
	config := models.BlogrollConfig{
		Enabled:              b.Enabled,
		BlogrollSlug:         b.BlogrollSlug,
		ReaderSlug:           b.ReaderSlug,
		CacheDir:             b.CacheDir,
		CacheDuration:        b.CacheDuration,
		Timeout:              b.Timeout,
		ConcurrentRequests:   b.ConcurrentRequests,
		MaxEntriesPerFeed:    b.MaxEntriesPerFeed,
		FallbackImageService: b.FallbackImageService,
		Templates: models.BlogrollTemplates{
			Blogroll: b.Templates.Blogroll,
			Reader:   b.Templates.Reader,
		},
	}

	for i := range b.Feeds {
		fc := &b.Feeds[i]
		config.Feeds = append(config.Feeds, models.ExternalFeedConfig{
			URL:           fc.URL,
			Title:         fc.Title,
			Description:   fc.Description,
			Category:      fc.Category,
			Tags:          fc.Tags,
			Active:        fc.Active,
			SiteURL:       fc.SiteURL,
			ImageURL:      fc.ImageURL,
			Handle:        fc.Handle,
			Aliases:       fc.Aliases,
			MaxEntries:    fc.MaxEntries,
			Primary:       fc.Primary,
			PrimaryPerson: fc.PrimaryPerson,
		})
	}

	return config
}

func (c *tomlComponentsConfig) toComponentsConfig() models.ComponentsConfig {
	config := models.ComponentsConfig{
		Nav: models.NavComponentConfig{
			Enabled:  c.Nav.Enabled,
			Position: c.Nav.Position,
			Style:    c.Nav.Style,
		},
		Footer: models.FooterComponentConfig{
			Enabled:       c.Footer.Enabled,
			Text:          c.Footer.Text,
			ShowCopyright: c.Footer.ShowCopyright,
		},
		DocSidebar: models.DocSidebarConfig{
			Enabled:  c.DocSidebar.Enabled,
			Position: c.DocSidebar.Position,
			Width:    c.DocSidebar.Width,
			MinDepth: c.DocSidebar.MinDepth,
			MaxDepth: c.DocSidebar.MaxDepth,
		},
	}

	// Convert nav items
	for _, item := range c.Nav.Items {
		config.Nav.Items = append(config.Nav.Items, models.NavItem{
			Label:    item.Label,
			URL:      item.URL,
			External: item.External,
		})
	}

	// Convert footer links
	for _, link := range c.Footer.Links {
		config.Footer.Links = append(config.Footer.Links, models.NavItem{
			Label:    link.Label,
			URL:      link.URL,
			External: link.External,
		})
	}

	return config
}

func (p *tomlPostFormatsConfig) toPostFormatsConfig() models.PostFormatsConfig {
	return models.PostFormatsConfig{
		HTML:     p.HTML,
		Markdown: p.Markdown,
		Text:     p.Text,
		OG:       p.OG,
	}
}

// configSource interface implementation for tomlConfig.
func (c *tomlConfig) getBaseConfig() baseConfigData {
	return baseConfigData{
		OutputDir:     c.OutputDir,
		URL:           c.URL,
		Title:         c.Title,
		Description:   c.Description,
		Author:        c.Author,
		AssetsDir:     c.AssetsDir,
		TemplatesDir:  c.TemplatesDir,
		Hooks:         c.Hooks,
		DisabledHooks: c.DisabledHooks,
		GlobPatterns:  c.Glob.Patterns,
		UseGitignore:  c.Glob.UseGitignore,
		Extensions:    c.Markdown.Extensions,
		Concurrency:   c.Concurrency,
		Theme:         c.Theme.toThemeConfig(),
		Footer:        c.Footer.toFooterConfig(),
	}
}

func (c *tomlConfig) getNavItems() []navItemData {
	items := make([]navItemData, len(c.Nav))
	for i, nav := range c.Nav {
		items[i] = navItemData(nav)
	}
	return items
}

func (c *tomlConfig) getFeeds() []feedConfigConverter {
	feeds := make([]feedConfigConverter, len(c.Feeds))
	for i := range c.Feeds {
		feeds[i] = &c.Feeds[i]
	}
	return feeds
}

func (c *tomlConfig) getFeedDefaults() feedDefaultsConverter { return &c.FeedDefaults }
func (c *tomlConfig) getPostFormats() postFormatsConverter   { return &c.PostFormats }
func (c *tomlConfig) getSEO() seoConverter                   { return &c.SEO }
func (c *tomlConfig) getIndieAuth() indieAuthConverter       { return &c.IndieAuth }
func (c *tomlConfig) getWebmention() webmentionConverter     { return &c.Webmention }
func (c *tomlConfig) getComponents() componentsConverter     { return &c.Components }
func (c *tomlConfig) getLayout() layoutConverter             { return &c.Layout }
func (c *tomlConfig) getSidebar() sidebarConverter           { return &c.Sidebar }
func (c *tomlConfig) getToc() tocConverter                   { return &c.Toc }
func (c *tomlConfig) getHeader() headerConverter             { return &c.Header }
func (c *tomlConfig) getBlogroll() blogrollConverter         { return &c.Blogroll }

func (c *tomlConfig) toConfig() *models.Config {
	return buildConfig(c)
}

func (f *tomlFooterConfig) toFooterConfig() models.FooterConfig {
	return models.FooterConfig{
		Text:          f.Text,
		ShowCopyright: f.ShowCopyright,
	}
}

func (t *tomlThemeConfig) toThemeConfig() models.ThemeConfig {
	variables := t.Variables
	if variables == nil {
		variables = make(map[string]string)
	}
	return models.ThemeConfig{
		Name:         t.Name,
		Palette:      t.Palette,
		PaletteLight: t.PaletteLight,
		PaletteDark:  t.PaletteDark,
		Variables:    variables,
		CustomCSS:    t.CustomCSS,
	}
}

func (f *tomlFeedConfig) toFeedConfig() models.FeedConfig {
	return models.FeedConfig{
		Slug:            f.Slug,
		Title:           f.Title,
		Description:     f.Description,
		Filter:          f.Filter,
		Sort:            f.Sort,
		Reverse:         f.Reverse,
		ItemsPerPage:    f.ItemsPerPage,
		OrphanThreshold: f.OrphanThreshold,
		PaginationType:  models.PaginationType(f.PaginationType),
		Formats:         f.Formats.toFeedFormats(),
		Templates:       f.Templates.toFeedTemplates(),
	}
}

func (f *tomlFeedFormats) toFeedFormats() models.FeedFormats {
	formats := models.FeedFormats{}
	if f.HTML != nil {
		formats.HTML = *f.HTML
	}
	if f.RSS != nil {
		formats.RSS = *f.RSS
	}
	if f.Atom != nil {
		formats.Atom = *f.Atom
	}
	if f.JSON != nil {
		formats.JSON = *f.JSON
	}
	if f.Markdown != nil {
		formats.Markdown = *f.Markdown
	}
	if f.Text != nil {
		formats.Text = *f.Text
	}
	if f.Sitemap != nil {
		formats.Sitemap = *f.Sitemap
	}
	return formats
}

func (t *tomlFeedTemplates) toFeedTemplates() models.FeedTemplates {
	return models.FeedTemplates{
		HTML:    t.HTML,
		RSS:     t.RSS,
		Atom:    t.Atom,
		JSON:    t.JSON,
		Card:    t.Card,
		Sitemap: t.Sitemap,
	}
}

func (d *tomlFeedDefaults) toFeedDefaults() models.FeedDefaults {
	return models.FeedDefaults{
		ItemsPerPage:    d.ItemsPerPage,
		OrphanThreshold: d.OrphanThreshold,
		PaginationType:  models.PaginationType(d.PaginationType),
		Formats:         d.Formats.toFeedFormats(),
		Templates:       d.Templates.toFeedTemplates(),
		Syndication: models.SyndicationConfig{
			MaxItems:       d.Syndication.MaxItems,
			IncludeContent: d.Syndication.IncludeContent,
		},
	}
}

// yamlConfig is an internal struct for parsing YAML configuration.
type yamlConfig struct {
	OutputDir     string                 `yaml:"output_dir"`
	URL           string                 `yaml:"url"`
	Title         string                 `yaml:"title"`
	Description   string                 `yaml:"description"`
	Author        string                 `yaml:"author"`
	AssetsDir     string                 `yaml:"assets_dir"`
	TemplatesDir  string                 `yaml:"templates_dir"`
	Nav           []yamlNavItem          `yaml:"nav"`
	Footer        yamlFooterConfig       `yaml:"footer"`
	Hooks         []string               `yaml:"hooks"`
	DisabledHooks []string               `yaml:"disabled_hooks"`
	Glob          yamlGlobConfig         `yaml:"glob"`
	Markdown      yamlMarkdownConfig     `yaml:"markdown"`
	Feeds         []yamlFeedConfig       `yaml:"feeds"`
	FeedDefaults  yamlFeedDefaults       `yaml:"feed_defaults"`
	Concurrency   int                    `yaml:"concurrency"`
	Theme         yamlThemeConfig        `yaml:"theme"`
	PostFormats   yamlPostFormatsConfig  `yaml:"post_formats"`
	IndieAuth     yamlIndieAuthConfig    `yaml:"indieauth"`
	Webmention    yamlWebmentionConfig   `yaml:"webmention"`
	SEO           yamlSEOConfig          `yaml:"seo"`
	Components    yamlComponentsConfig   `yaml:"components"`
	Layout        yamlLayoutConfig       `yaml:"layout"`
	Sidebar       yamlSidebarConfig      `yaml:"sidebar"`
	Toc           yamlTocConfig          `yaml:"toc"`
	Header        yamlHeaderLayoutConfig `yaml:"header"`
	Blogroll      yamlBlogrollConfig     `yaml:"blogroll"`
}

type yamlNavItem struct {
	Label    string `yaml:"label"`
	URL      string `yaml:"url"`
	External bool   `yaml:"external"`
}

type yamlFooterConfig struct {
	Text          string `yaml:"text"`
	ShowCopyright *bool  `yaml:"show_copyright"`
}

type yamlGlobConfig struct {
	Patterns     []string `yaml:"patterns"`
	UseGitignore *bool    `yaml:"use_gitignore"`
}

type yamlMarkdownConfig struct {
	Extensions []string `yaml:"extensions"`
}

type yamlFeedConfig struct {
	Slug            string            `yaml:"slug"`
	Title           string            `yaml:"title"`
	Description     string            `yaml:"description"`
	Filter          string            `yaml:"filter"`
	Sort            string            `yaml:"sort"`
	Reverse         bool              `yaml:"reverse"`
	ItemsPerPage    int               `yaml:"items_per_page"`
	OrphanThreshold int               `yaml:"orphan_threshold"`
	PaginationType  string            `yaml:"pagination_type"`
	Formats         yamlFeedFormats   `yaml:"formats"`
	Templates       yamlFeedTemplates `yaml:"templates"`
}

type yamlFeedFormats struct {
	HTML     *bool `yaml:"html"`
	RSS      *bool `yaml:"rss"`
	Atom     *bool `yaml:"atom"`
	JSON     *bool `yaml:"json"`
	Markdown *bool `yaml:"markdown"`
	Text     *bool `yaml:"text"`
	Sitemap  *bool `yaml:"sitemap"`
}

type yamlFeedTemplates struct {
	HTML    string `yaml:"html"`
	RSS     string `yaml:"rss"`
	Atom    string `yaml:"atom"`
	JSON    string `yaml:"json"`
	Card    string `yaml:"card"`
	Sitemap string `yaml:"sitemap"`
}

type yamlFeedDefaults struct {
	ItemsPerPage    int                   `yaml:"items_per_page"`
	OrphanThreshold int                   `yaml:"orphan_threshold"`
	PaginationType  string                `yaml:"pagination_type"`
	Formats         yamlFeedFormats       `yaml:"formats"`
	Templates       yamlFeedTemplates     `yaml:"templates"`
	Syndication     yamlSyndicationConfig `yaml:"syndication"`
}

type yamlSyndicationConfig struct {
	MaxItems       int  `yaml:"max_items"`
	IncludeContent bool `yaml:"include_content"`
}

type yamlPostFormatsConfig struct {
	HTML     *bool `yaml:"html"`
	Markdown bool  `yaml:"markdown"`
	Text     bool  `yaml:"text"`
	OG       bool  `yaml:"og"`
}

type yamlSEOConfig struct {
	TwitterHandle string `yaml:"twitter_handle"`
	DefaultImage  string `yaml:"default_image"`
	LogoURL       string `yaml:"logo_url"`
}

type yamlIndieAuthConfig struct {
	Enabled               bool   `yaml:"enabled"`
	AuthorizationEndpoint string `yaml:"authorization_endpoint"`
	TokenEndpoint         string `yaml:"token_endpoint"`
	MeURL                 string `yaml:"me_url"`
}

type yamlWebmentionConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Endpoint string `yaml:"endpoint"`
}

type yamlThemeConfig struct {
	Name         string            `yaml:"name"`
	Palette      string            `yaml:"palette"`
	PaletteLight string            `yaml:"palette_light"`
	PaletteDark  string            `yaml:"palette_dark"`
	Variables    map[string]string `yaml:"variables"`
	CustomCSS    string            `yaml:"custom_css"`
}

func (t *yamlThemeConfig) toThemeConfig() models.ThemeConfig {
	variables := t.Variables
	if variables == nil {
		variables = make(map[string]string)
	}
	return models.ThemeConfig{
		Name:         t.Name,
		Palette:      t.Palette,
		PaletteLight: t.PaletteLight,
		PaletteDark:  t.PaletteDark,
		Variables:    variables,
		CustomCSS:    t.CustomCSS,
	}
}

func (s *yamlSEOConfig) toSEOConfig() models.SEOConfig {
	return models.SEOConfig{
		TwitterHandle: s.TwitterHandle,
		DefaultImage:  s.DefaultImage,
		LogoURL:       s.LogoURL,
	}
}

func (i *yamlIndieAuthConfig) toIndieAuthConfig() models.IndieAuthConfig {
	return models.IndieAuthConfig{
		Enabled:               i.Enabled,
		AuthorizationEndpoint: i.AuthorizationEndpoint,
		TokenEndpoint:         i.TokenEndpoint,
		MeURL:                 i.MeURL,
	}
}

func (w *yamlWebmentionConfig) toWebmentionConfig() models.WebmentionConfig {
	return models.WebmentionConfig{
		Enabled:  w.Enabled,
		Endpoint: w.Endpoint,
	}
}

type yamlComponentsConfig struct {
	Nav        yamlNavComponentConfig    `yaml:"nav"`
	Footer     yamlFooterComponentConfig `yaml:"footer"`
	DocSidebar yamlDocSidebarConfig      `yaml:"doc_sidebar"`
}

type yamlNavComponentConfig struct {
	Enabled  *bool         `yaml:"enabled"`
	Position string        `yaml:"position"`
	Style    string        `yaml:"style"`
	Items    []yamlNavItem `yaml:"items"`
}

type yamlFooterComponentConfig struct {
	Enabled       *bool         `yaml:"enabled"`
	Text          string        `yaml:"text"`
	ShowCopyright *bool         `yaml:"show_copyright"`
	Links         []yamlNavItem `yaml:"links"`
}

type yamlDocSidebarConfig struct {
	Enabled  *bool  `yaml:"enabled"`
	Position string `yaml:"position"`
	Width    string `yaml:"width"`
	MinDepth int    `yaml:"min_depth"`
	MaxDepth int    `yaml:"max_depth"`
}

// Layout-related YAML structs

type yamlLayoutConfig struct {
	Name     string                  `yaml:"name"`
	Paths    map[string]string       `yaml:"paths"`
	Feeds    map[string]string       `yaml:"feeds"`
	Docs     yamlDocsLayoutConfig    `yaml:"docs"`
	Blog     yamlBlogLayoutConfig    `yaml:"blog"`
	Landing  yamlLandingLayoutConfig `yaml:"landing"`
	Bare     yamlBareLayoutConfig    `yaml:"bare"`
	Defaults yamlLayoutDefaults      `yaml:"defaults"`
}

type yamlLayoutDefaults struct {
	ContentMaxWidth string `yaml:"content_max_width"`
	HeaderSticky    *bool  `yaml:"header_sticky"`
	FooterSticky    *bool  `yaml:"footer_sticky"`
}

type yamlDocsLayoutConfig struct {
	SidebarPosition    string `yaml:"sidebar_position"`
	SidebarWidth       string `yaml:"sidebar_width"`
	SidebarCollapsible *bool  `yaml:"sidebar_collapsible"`
	SidebarDefaultOpen *bool  `yaml:"sidebar_default_open"`
	TocPosition        string `yaml:"toc_position"`
	TocWidth           string `yaml:"toc_width"`
	TocCollapsible     *bool  `yaml:"toc_collapsible"`
	TocDefaultOpen     *bool  `yaml:"toc_default_open"`
	ContentMaxWidth    string `yaml:"content_max_width"`
	HeaderStyle        string `yaml:"header_style"`
	FooterStyle        string `yaml:"footer_style"`
}

type yamlBlogLayoutConfig struct {
	ContentMaxWidth string `yaml:"content_max_width"`
	ShowToc         *bool  `yaml:"show_toc"`
	TocPosition     string `yaml:"toc_position"`
	TocWidth        string `yaml:"toc_width"`
	HeaderStyle     string `yaml:"header_style"`
	FooterStyle     string `yaml:"footer_style"`
	ShowAuthor      *bool  `yaml:"show_author"`
	ShowDate        *bool  `yaml:"show_date"`
	ShowTags        *bool  `yaml:"show_tags"`
	ShowReadingTime *bool  `yaml:"show_reading_time"`
	ShowPrevNext    *bool  `yaml:"show_prev_next"`
}

type yamlLandingLayoutConfig struct {
	ContentMaxWidth string `yaml:"content_max_width"`
	HeaderStyle     string `yaml:"header_style"`
	HeaderSticky    *bool  `yaml:"header_sticky"`
	FooterStyle     string `yaml:"footer_style"`
	HeroEnabled     *bool  `yaml:"hero_enabled"`
}

type yamlBareLayoutConfig struct {
	ContentMaxWidth string `yaml:"content_max_width"`
}

func (l *yamlLayoutConfig) toLayoutConfig() models.LayoutConfig {
	return models.LayoutConfig{
		Name:  l.Name,
		Paths: l.Paths,
		Feeds: l.Feeds,
		Docs: models.DocsLayoutConfig{
			SidebarPosition:    l.Docs.SidebarPosition,
			SidebarWidth:       l.Docs.SidebarWidth,
			SidebarCollapsible: l.Docs.SidebarCollapsible,
			SidebarDefaultOpen: l.Docs.SidebarDefaultOpen,
			TocPosition:        l.Docs.TocPosition,
			TocWidth:           l.Docs.TocWidth,
			TocCollapsible:     l.Docs.TocCollapsible,
			TocDefaultOpen:     l.Docs.TocDefaultOpen,
			ContentMaxWidth:    l.Docs.ContentMaxWidth,
			HeaderStyle:        l.Docs.HeaderStyle,
			FooterStyle:        l.Docs.FooterStyle,
		},
		Blog: models.BlogLayoutConfig{
			ContentMaxWidth: l.Blog.ContentMaxWidth,
			ShowToc:         l.Blog.ShowToc,
			TocPosition:     l.Blog.TocPosition,
			TocWidth:        l.Blog.TocWidth,
			HeaderStyle:     l.Blog.HeaderStyle,
			FooterStyle:     l.Blog.FooterStyle,
			ShowAuthor:      l.Blog.ShowAuthor,
			ShowDate:        l.Blog.ShowDate,
			ShowTags:        l.Blog.ShowTags,
			ShowReadingTime: l.Blog.ShowReadingTime,
			ShowPrevNext:    l.Blog.ShowPrevNext,
		},
		Landing: models.LandingLayoutConfig{
			ContentMaxWidth: l.Landing.ContentMaxWidth,
			HeaderStyle:     l.Landing.HeaderStyle,
			HeaderSticky:    l.Landing.HeaderSticky,
			FooterStyle:     l.Landing.FooterStyle,
			HeroEnabled:     l.Landing.HeroEnabled,
		},
		Bare: models.BareLayoutConfig{
			ContentMaxWidth: l.Bare.ContentMaxWidth,
		},
		Defaults: models.LayoutDefaults{
			ContentMaxWidth: l.Defaults.ContentMaxWidth,
			HeaderSticky:    l.Defaults.HeaderSticky,
			FooterSticky:    l.Defaults.FooterSticky,
		},
	}
}

// Sidebar-related YAML structs

type yamlSidebarConfig struct {
	Enabled      *bool                             `yaml:"enabled"`
	Position     string                            `yaml:"position"`
	Width        string                            `yaml:"width"`
	Collapsible  *bool                             `yaml:"collapsible"`
	DefaultOpen  *bool                             `yaml:"default_open"`
	Nav          []yamlSidebarNavItem              `yaml:"nav"`
	Title        string                            `yaml:"title"`
	Paths        map[string]*yamlPathSidebarConfig `yaml:"paths"`
	MultiFeed    *bool                             `yaml:"multi_feed"`
	Feeds        []string                          `yaml:"feeds"`
	FeedSections []yamlMultiFeedSection            `yaml:"feed_sections"`
	AutoGenerate *yamlSidebarAutoGenerate          `yaml:"auto_generate"`
}

type yamlSidebarNavItem struct {
	Title    string               `yaml:"title"`
	Href     string               `yaml:"href"`
	Children []yamlSidebarNavItem `yaml:"children"`
}

type yamlPathSidebarConfig struct {
	Title        string                   `yaml:"title"`
	AutoGenerate *yamlSidebarAutoGenerate `yaml:"auto_generate"`
	Items        []yamlSidebarNavItem     `yaml:"items"`
	Feed         string                   `yaml:"feed"`
	Position     string                   `yaml:"position"`
	Collapsible  *bool                    `yaml:"collapsible"`
}

type yamlSidebarAutoGenerate struct {
	Directory string   `yaml:"directory"`
	OrderBy   string   `yaml:"order_by"`
	Reverse   *bool    `yaml:"reverse"`
	MaxDepth  int      `yaml:"max_depth"`
	Exclude   []string `yaml:"exclude"`
}

type yamlMultiFeedSection struct {
	Feed      string `yaml:"feed"`
	Title     string `yaml:"title"`
	Collapsed *bool  `yaml:"collapsed"`
	MaxItems  int    `yaml:"max_items"`
}

func convertYamlSidebarNavItems(items []yamlSidebarNavItem) []models.SidebarNavItem {
	result := make([]models.SidebarNavItem, len(items))
	for i, item := range items {
		result[i] = models.SidebarNavItem{
			Title:    item.Title,
			Href:     item.Href,
			Children: convertYamlSidebarNavItems(item.Children),
		}
	}
	return result
}

func (s *yamlSidebarConfig) toSidebarConfig() models.SidebarConfig {
	config := models.SidebarConfig{
		Enabled:     s.Enabled,
		Position:    s.Position,
		Width:       s.Width,
		Collapsible: s.Collapsible,
		DefaultOpen: s.DefaultOpen,
		Nav:         convertYamlSidebarNavItems(s.Nav),
		Title:       s.Title,
		MultiFeed:   s.MultiFeed,
		Feeds:       s.Feeds,
	}

	// Convert paths
	if len(s.Paths) > 0 {
		config.Paths = make(map[string]*models.PathSidebarConfig)
		for path, pathConfig := range s.Paths {
			var autoGen *models.SidebarAutoGenerate
			if pathConfig.AutoGenerate != nil {
				autoGen = &models.SidebarAutoGenerate{
					Directory: pathConfig.AutoGenerate.Directory,
					OrderBy:   pathConfig.AutoGenerate.OrderBy,
					Reverse:   pathConfig.AutoGenerate.Reverse,
					MaxDepth:  pathConfig.AutoGenerate.MaxDepth,
					Exclude:   pathConfig.AutoGenerate.Exclude,
				}
			}
			config.Paths[path] = &models.PathSidebarConfig{
				Title:        pathConfig.Title,
				AutoGenerate: autoGen,
				Items:        convertYamlSidebarNavItems(pathConfig.Items),
				Feed:         pathConfig.Feed,
				Position:     pathConfig.Position,
				Collapsible:  pathConfig.Collapsible,
			}
		}
	}

	// Convert feed sections
	if len(s.FeedSections) > 0 {
		config.FeedSections = make([]models.MultiFeedSection, len(s.FeedSections))
		for i, section := range s.FeedSections {
			config.FeedSections[i] = models.MultiFeedSection{
				Feed:      section.Feed,
				Title:     section.Title,
				Collapsed: section.Collapsed,
				MaxItems:  section.MaxItems,
			}
		}
	}

	// Convert auto-generate
	if s.AutoGenerate != nil {
		config.AutoGenerate = &models.SidebarAutoGenerate{
			Directory: s.AutoGenerate.Directory,
			OrderBy:   s.AutoGenerate.OrderBy,
			Reverse:   s.AutoGenerate.Reverse,
			MaxDepth:  s.AutoGenerate.MaxDepth,
			Exclude:   s.AutoGenerate.Exclude,
		}
	}

	return config
}

// TOC-related YAML structs

type yamlTocConfig struct {
	Enabled     *bool  `yaml:"enabled"`
	Position    string `yaml:"position"`
	Width       string `yaml:"width"`
	MinDepth    int    `yaml:"min_depth"`
	MaxDepth    int    `yaml:"max_depth"`
	Title       string `yaml:"title"`
	Collapsible *bool  `yaml:"collapsible"`
	DefaultOpen *bool  `yaml:"default_open"`
	ScrollSpy   *bool  `yaml:"scroll_spy"`
}

func (t *yamlTocConfig) toTocConfig() models.TocConfig {
	return models.TocConfig{
		Enabled:     t.Enabled,
		Position:    t.Position,
		Width:       t.Width,
		MinDepth:    t.MinDepth,
		MaxDepth:    t.MaxDepth,
		Title:       t.Title,
		Collapsible: t.Collapsible,
		DefaultOpen: t.DefaultOpen,
		ScrollSpy:   t.ScrollSpy,
	}
}

// Header layout YAML structs

type yamlHeaderLayoutConfig struct {
	Style           string `yaml:"style"`
	Sticky          *bool  `yaml:"sticky"`
	ShowLogo        *bool  `yaml:"show_logo"`
	ShowTitle       *bool  `yaml:"show_title"`
	ShowNav         *bool  `yaml:"show_nav"`
	ShowSearch      *bool  `yaml:"show_search"`
	ShowThemeToggle *bool  `yaml:"show_theme_toggle"`
}

func (h *yamlHeaderLayoutConfig) toHeaderLayoutConfig() models.HeaderLayoutConfig {
	return models.HeaderLayoutConfig{
		Style:           h.Style,
		Sticky:          h.Sticky,
		ShowLogo:        h.ShowLogo,
		ShowTitle:       h.ShowTitle,
		ShowNav:         h.ShowNav,
		ShowSearch:      h.ShowSearch,
		ShowThemeToggle: h.ShowThemeToggle,
	}
}

// Blogroll-related YAML structs

type yamlBlogrollConfig struct {
	Enabled              bool                     `yaml:"enabled"`
	BlogrollSlug         string                   `yaml:"blogroll_slug"`
	ReaderSlug           string                   `yaml:"reader_slug"`
	CacheDir             string                   `yaml:"cache_dir"`
	CacheDuration        string                   `yaml:"cache_duration"`
	Timeout              int                      `yaml:"timeout"`
	ConcurrentRequests   int                      `yaml:"concurrent_requests"`
	MaxEntriesPerFeed    int                      `yaml:"max_entries_per_feed"`
	FallbackImageService string                   `yaml:"fallback_image_service"`
	Feeds                []yamlExternalFeedConfig `yaml:"feeds"`
	Templates            yamlBlogrollTemplates    `yaml:"templates"`
}

type yamlExternalFeedConfig struct {
	URL           string   `yaml:"url"`
	Title         string   `yaml:"title"`
	Description   string   `yaml:"description"`
	Category      string   `yaml:"category"`
	Tags          []string `yaml:"tags"`
	Active        *bool    `yaml:"active"`
	SiteURL       string   `yaml:"site_url"`
	ImageURL      string   `yaml:"image_url"`
	Handle        string   `yaml:"handle"`
	Aliases       []string `yaml:"aliases,omitempty"`
	MaxEntries    *int     `yaml:"max_entries,omitempty"`
	Primary       *bool    `yaml:"primary,omitempty"`
	PrimaryPerson string   `yaml:"primary_person"`
}

type yamlBlogrollTemplates struct {
	Blogroll string `yaml:"blogroll"`
	Reader   string `yaml:"reader"`
}

//nolint:dupl // Intentional duplication - each format has its own conversion method
func (b *yamlBlogrollConfig) toBlogrollConfig() models.BlogrollConfig {
	config := models.BlogrollConfig{
		Enabled:              b.Enabled,
		BlogrollSlug:         b.BlogrollSlug,
		ReaderSlug:           b.ReaderSlug,
		CacheDir:             b.CacheDir,
		CacheDuration:        b.CacheDuration,
		Timeout:              b.Timeout,
		ConcurrentRequests:   b.ConcurrentRequests,
		MaxEntriesPerFeed:    b.MaxEntriesPerFeed,
		FallbackImageService: b.FallbackImageService,
		Templates: models.BlogrollTemplates{
			Blogroll: b.Templates.Blogroll,
			Reader:   b.Templates.Reader,
		},
	}

	for i := range b.Feeds {
		fc := &b.Feeds[i]
		config.Feeds = append(config.Feeds, models.ExternalFeedConfig{
			URL:           fc.URL,
			Title:         fc.Title,
			Description:   fc.Description,
			Category:      fc.Category,
			Tags:          fc.Tags,
			Active:        fc.Active,
			SiteURL:       fc.SiteURL,
			ImageURL:      fc.ImageURL,
			Handle:        fc.Handle,
			Aliases:       fc.Aliases,
			MaxEntries:    fc.MaxEntries,
			Primary:       fc.Primary,
			PrimaryPerson: fc.PrimaryPerson,
		})
	}

	return config
}

func (c *yamlComponentsConfig) toComponentsConfig() models.ComponentsConfig {
	config := models.ComponentsConfig{
		Nav: models.NavComponentConfig{
			Enabled:  c.Nav.Enabled,
			Position: c.Nav.Position,
			Style:    c.Nav.Style,
		},
		Footer: models.FooterComponentConfig{
			Enabled:       c.Footer.Enabled,
			Text:          c.Footer.Text,
			ShowCopyright: c.Footer.ShowCopyright,
		},
		DocSidebar: models.DocSidebarConfig{
			Enabled:  c.DocSidebar.Enabled,
			Position: c.DocSidebar.Position,
			Width:    c.DocSidebar.Width,
			MinDepth: c.DocSidebar.MinDepth,
			MaxDepth: c.DocSidebar.MaxDepth,
		},
	}

	// Convert nav items
	for _, item := range c.Nav.Items {
		config.Nav.Items = append(config.Nav.Items, models.NavItem{
			Label:    item.Label,
			URL:      item.URL,
			External: item.External,
		})
	}

	// Convert footer links
	for _, link := range c.Footer.Links {
		config.Footer.Links = append(config.Footer.Links, models.NavItem{
			Label:    link.Label,
			URL:      link.URL,
			External: link.External,
		})
	}

	return config
}

func (p *yamlPostFormatsConfig) toPostFormatsConfig() models.PostFormatsConfig {
	return models.PostFormatsConfig{
		HTML:     p.HTML,
		Markdown: p.Markdown,
		Text:     p.Text,
		OG:       p.OG,
	}
}

// configSource interface implementation for yamlConfig.
func (c *yamlConfig) getBaseConfig() baseConfigData {
	return baseConfigData{
		OutputDir:     c.OutputDir,
		URL:           c.URL,
		Title:         c.Title,
		Description:   c.Description,
		Author:        c.Author,
		AssetsDir:     c.AssetsDir,
		TemplatesDir:  c.TemplatesDir,
		Hooks:         c.Hooks,
		DisabledHooks: c.DisabledHooks,
		GlobPatterns:  c.Glob.Patterns,
		UseGitignore:  c.Glob.UseGitignore,
		Extensions:    c.Markdown.Extensions,
		Concurrency:   c.Concurrency,
		Theme:         c.Theme.toThemeConfig(),
		Footer:        c.Footer.toFooterConfig(),
	}
}

func (c *yamlConfig) getNavItems() []navItemData {
	items := make([]navItemData, len(c.Nav))
	for i, nav := range c.Nav {
		items[i] = navItemData(nav)
	}
	return items
}

func (c *yamlConfig) getFeeds() []feedConfigConverter {
	feeds := make([]feedConfigConverter, len(c.Feeds))
	for i := range c.Feeds {
		feeds[i] = &c.Feeds[i]
	}
	return feeds
}

func (c *yamlConfig) getFeedDefaults() feedDefaultsConverter { return &c.FeedDefaults }
func (c *yamlConfig) getPostFormats() postFormatsConverter   { return &c.PostFormats }
func (c *yamlConfig) getSEO() seoConverter                   { return &c.SEO }
func (c *yamlConfig) getIndieAuth() indieAuthConverter       { return &c.IndieAuth }
func (c *yamlConfig) getWebmention() webmentionConverter     { return &c.Webmention }
func (c *yamlConfig) getComponents() componentsConverter     { return &c.Components }
func (c *yamlConfig) getLayout() layoutConverter             { return &c.Layout }
func (c *yamlConfig) getSidebar() sidebarConverter           { return &c.Sidebar }
func (c *yamlConfig) getToc() tocConverter                   { return &c.Toc }
func (c *yamlConfig) getHeader() headerConverter             { return &c.Header }
func (c *yamlConfig) getBlogroll() blogrollConverter         { return &c.Blogroll }

func (c *yamlConfig) toConfig() *models.Config {
	return buildConfig(c)
}

func (f *yamlFooterConfig) toFooterConfig() models.FooterConfig {
	return models.FooterConfig{
		Text:          f.Text,
		ShowCopyright: f.ShowCopyright,
	}
}

func (f *yamlFeedConfig) toFeedConfig() models.FeedConfig {
	return models.FeedConfig{
		Slug:            f.Slug,
		Title:           f.Title,
		Description:     f.Description,
		Filter:          f.Filter,
		Sort:            f.Sort,
		Reverse:         f.Reverse,
		ItemsPerPage:    f.ItemsPerPage,
		OrphanThreshold: f.OrphanThreshold,
		PaginationType:  models.PaginationType(f.PaginationType),
		Formats:         f.Formats.toFeedFormats(),
		Templates:       f.Templates.toFeedTemplates(),
	}
}

func (f *yamlFeedFormats) toFeedFormats() models.FeedFormats {
	formats := models.FeedFormats{}
	if f.HTML != nil {
		formats.HTML = *f.HTML
	}
	if f.RSS != nil {
		formats.RSS = *f.RSS
	}
	if f.Atom != nil {
		formats.Atom = *f.Atom
	}
	if f.JSON != nil {
		formats.JSON = *f.JSON
	}
	if f.Markdown != nil {
		formats.Markdown = *f.Markdown
	}
	if f.Text != nil {
		formats.Text = *f.Text
	}
	if f.Sitemap != nil {
		formats.Sitemap = *f.Sitemap
	}
	return formats
}

func (t *yamlFeedTemplates) toFeedTemplates() models.FeedTemplates {
	return models.FeedTemplates{
		HTML:    t.HTML,
		RSS:     t.RSS,
		Atom:    t.Atom,
		JSON:    t.JSON,
		Card:    t.Card,
		Sitemap: t.Sitemap,
	}
}

func (d *yamlFeedDefaults) toFeedDefaults() models.FeedDefaults {
	return models.FeedDefaults{
		ItemsPerPage:    d.ItemsPerPage,
		OrphanThreshold: d.OrphanThreshold,
		PaginationType:  models.PaginationType(d.PaginationType),
		Formats:         d.Formats.toFeedFormats(),
		Templates:       d.Templates.toFeedTemplates(),
		Syndication: models.SyndicationConfig{
			MaxItems:       d.Syndication.MaxItems,
			IncludeContent: d.Syndication.IncludeContent,
		},
	}
}

// jsonConfig is an internal struct for parsing JSON configuration.
type jsonConfig struct {
	OutputDir     string                 `json:"output_dir"`
	URL           string                 `json:"url"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	Author        string                 `json:"author"`
	AssetsDir     string                 `json:"assets_dir"`
	TemplatesDir  string                 `json:"templates_dir"`
	Nav           []jsonNavItem          `json:"nav"`
	Footer        jsonFooterConfig       `json:"footer"`
	Hooks         []string               `json:"hooks"`
	DisabledHooks []string               `json:"disabled_hooks"`
	Glob          jsonGlobConfig         `json:"glob"`
	Markdown      jsonMarkdownConfig     `json:"markdown"`
	Feeds         []jsonFeedConfig       `json:"feeds"`
	FeedDefaults  jsonFeedDefaults       `json:"feed_defaults"`
	Concurrency   int                    `json:"concurrency"`
	Theme         jsonThemeConfig        `json:"theme"`
	PostFormats   jsonPostFormatsConfig  `json:"post_formats"`
	IndieAuth     jsonIndieAuthConfig    `json:"indieauth"`
	Webmention    jsonWebmentionConfig   `json:"webmention"`
	SEO           jsonSEOConfig          `json:"seo"`
	Components    jsonComponentsConfig   `json:"components"`
	Layout        jsonLayoutConfig       `json:"layout"`
	Sidebar       jsonSidebarConfig      `json:"sidebar"`
	Toc           jsonTocConfig          `json:"toc"`
	Header        jsonHeaderLayoutConfig `json:"header"`
	Blogroll      jsonBlogrollConfig     `json:"blogroll"`
}

type jsonNavItem struct {
	Label    string `json:"label"`
	URL      string `json:"url"`
	External bool   `json:"external"`
}

type jsonFooterConfig struct {
	Text          string `json:"text"`
	ShowCopyright *bool  `json:"show_copyright"`
}

type jsonGlobConfig struct {
	Patterns     []string `json:"patterns"`
	UseGitignore *bool    `json:"use_gitignore"`
}

type jsonMarkdownConfig struct {
	Extensions []string `json:"extensions"`
}

type jsonFeedConfig struct {
	Slug            string            `json:"slug"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	Filter          string            `json:"filter"`
	Sort            string            `json:"sort"`
	Reverse         bool              `json:"reverse"`
	ItemsPerPage    int               `json:"items_per_page"`
	OrphanThreshold int               `json:"orphan_threshold"`
	PaginationType  string            `json:"pagination_type"`
	Formats         jsonFeedFormats   `json:"formats"`
	Templates       jsonFeedTemplates `json:"templates"`
}

type jsonFeedFormats struct {
	HTML     *bool `json:"html"`
	RSS      *bool `json:"rss"`
	Atom     *bool `json:"atom"`
	JSON     *bool `json:"json"`
	Markdown *bool `json:"markdown"`
	Text     *bool `json:"text"`
	Sitemap  *bool `json:"sitemap"`
}

type jsonFeedTemplates struct {
	HTML    string `json:"html"`
	RSS     string `json:"rss"`
	Atom    string `json:"atom"`
	JSON    string `json:"json"`
	Card    string `json:"card"`
	Sitemap string `json:"sitemap"`
}

type jsonFeedDefaults struct {
	ItemsPerPage    int                   `json:"items_per_page"`
	OrphanThreshold int                   `json:"orphan_threshold"`
	PaginationType  string                `json:"pagination_type"`
	Formats         jsonFeedFormats       `json:"formats"`
	Templates       jsonFeedTemplates     `json:"templates"`
	Syndication     jsonSyndicationConfig `json:"syndication"`
}

type jsonSyndicationConfig struct {
	MaxItems       int  `json:"max_items"`
	IncludeContent bool `json:"include_content"`
}

type jsonPostFormatsConfig struct {
	HTML     *bool `json:"html"`
	Markdown bool  `json:"markdown"`
	Text     bool  `json:"text"`
	OG       bool  `json:"og"`
}

type jsonSEOConfig struct {
	TwitterHandle string `json:"twitter_handle"`
	DefaultImage  string `json:"default_image"`
	LogoURL       string `json:"logo_url"`
}

type jsonIndieAuthConfig struct {
	Enabled               bool   `json:"enabled"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	MeURL                 string `json:"me_url"`
}

type jsonWebmentionConfig struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
}

type jsonThemeConfig struct {
	Name         string            `json:"name"`
	Palette      string            `json:"palette"`
	PaletteLight string            `json:"palette_light"`
	PaletteDark  string            `json:"palette_dark"`
	Variables    map[string]string `json:"variables"`
	CustomCSS    string            `json:"custom_css"`
}

func (t *jsonThemeConfig) toThemeConfig() models.ThemeConfig {
	variables := t.Variables
	if variables == nil {
		variables = make(map[string]string)
	}
	return models.ThemeConfig{
		Name:         t.Name,
		Palette:      t.Palette,
		PaletteLight: t.PaletteLight,
		PaletteDark:  t.PaletteDark,
		Variables:    variables,
		CustomCSS:    t.CustomCSS,
	}
}

func (s *jsonSEOConfig) toSEOConfig() models.SEOConfig {
	return models.SEOConfig{
		TwitterHandle: s.TwitterHandle,
		DefaultImage:  s.DefaultImage,
		LogoURL:       s.LogoURL,
	}
}

func (i *jsonIndieAuthConfig) toIndieAuthConfig() models.IndieAuthConfig {
	return models.IndieAuthConfig{
		Enabled:               i.Enabled,
		AuthorizationEndpoint: i.AuthorizationEndpoint,
		TokenEndpoint:         i.TokenEndpoint,
		MeURL:                 i.MeURL,
	}
}

func (w *jsonWebmentionConfig) toWebmentionConfig() models.WebmentionConfig {
	return models.WebmentionConfig{
		Enabled:  w.Enabled,
		Endpoint: w.Endpoint,
	}
}

type jsonComponentsConfig struct {
	Nav        jsonNavComponentConfig    `json:"nav"`
	Footer     jsonFooterComponentConfig `json:"footer"`
	DocSidebar jsonDocSidebarConfig      `json:"doc_sidebar"`
}

type jsonNavComponentConfig struct {
	Enabled  *bool         `json:"enabled"`
	Position string        `json:"position"`
	Style    string        `json:"style"`
	Items    []jsonNavItem `json:"items"`
}

type jsonFooterComponentConfig struct {
	Enabled       *bool         `json:"enabled"`
	Text          string        `json:"text"`
	ShowCopyright *bool         `json:"show_copyright"`
	Links         []jsonNavItem `json:"links"`
}

type jsonDocSidebarConfig struct {
	Enabled  *bool  `json:"enabled"`
	Position string `json:"position"`
	Width    string `json:"width"`
	MinDepth int    `json:"min_depth"`
	MaxDepth int    `json:"max_depth"`
}

// Layout-related JSON structs

type jsonLayoutConfig struct {
	Name     string                  `json:"name"`
	Paths    map[string]string       `json:"paths"`
	Feeds    map[string]string       `json:"feeds"`
	Docs     jsonDocsLayoutConfig    `json:"docs"`
	Blog     jsonBlogLayoutConfig    `json:"blog"`
	Landing  jsonLandingLayoutConfig `json:"landing"`
	Bare     jsonBareLayoutConfig    `json:"bare"`
	Defaults jsonLayoutDefaults      `json:"defaults"`
}

type jsonLayoutDefaults struct {
	ContentMaxWidth string `json:"content_max_width"`
	HeaderSticky    *bool  `json:"header_sticky"`
	FooterSticky    *bool  `json:"footer_sticky"`
}

type jsonDocsLayoutConfig struct {
	SidebarPosition    string `json:"sidebar_position"`
	SidebarWidth       string `json:"sidebar_width"`
	SidebarCollapsible *bool  `json:"sidebar_collapsible"`
	SidebarDefaultOpen *bool  `json:"sidebar_default_open"`
	TocPosition        string `json:"toc_position"`
	TocWidth           string `json:"toc_width"`
	TocCollapsible     *bool  `json:"toc_collapsible"`
	TocDefaultOpen     *bool  `json:"toc_default_open"`
	ContentMaxWidth    string `json:"content_max_width"`
	HeaderStyle        string `json:"header_style"`
	FooterStyle        string `json:"footer_style"`
}

type jsonBlogLayoutConfig struct {
	ContentMaxWidth string `json:"content_max_width"`
	ShowToc         *bool  `json:"show_toc"`
	TocPosition     string `json:"toc_position"`
	TocWidth        string `json:"toc_width"`
	HeaderStyle     string `json:"header_style"`
	FooterStyle     string `json:"footer_style"`
	ShowAuthor      *bool  `json:"show_author"`
	ShowDate        *bool  `json:"show_date"`
	ShowTags        *bool  `json:"show_tags"`
	ShowReadingTime *bool  `json:"show_reading_time"`
	ShowPrevNext    *bool  `json:"show_prev_next"`
}

type jsonLandingLayoutConfig struct {
	ContentMaxWidth string `json:"content_max_width"`
	HeaderStyle     string `json:"header_style"`
	HeaderSticky    *bool  `json:"header_sticky"`
	FooterStyle     string `json:"footer_style"`
	HeroEnabled     *bool  `json:"hero_enabled"`
}

type jsonBareLayoutConfig struct {
	ContentMaxWidth string `json:"content_max_width"`
}

func (l *jsonLayoutConfig) toLayoutConfig() models.LayoutConfig {
	return models.LayoutConfig{
		Name:  l.Name,
		Paths: l.Paths,
		Feeds: l.Feeds,
		Docs: models.DocsLayoutConfig{
			SidebarPosition:    l.Docs.SidebarPosition,
			SidebarWidth:       l.Docs.SidebarWidth,
			SidebarCollapsible: l.Docs.SidebarCollapsible,
			SidebarDefaultOpen: l.Docs.SidebarDefaultOpen,
			TocPosition:        l.Docs.TocPosition,
			TocWidth:           l.Docs.TocWidth,
			TocCollapsible:     l.Docs.TocCollapsible,
			TocDefaultOpen:     l.Docs.TocDefaultOpen,
			ContentMaxWidth:    l.Docs.ContentMaxWidth,
			HeaderStyle:        l.Docs.HeaderStyle,
			FooterStyle:        l.Docs.FooterStyle,
		},
		Blog: models.BlogLayoutConfig{
			ContentMaxWidth: l.Blog.ContentMaxWidth,
			ShowToc:         l.Blog.ShowToc,
			TocPosition:     l.Blog.TocPosition,
			TocWidth:        l.Blog.TocWidth,
			HeaderStyle:     l.Blog.HeaderStyle,
			FooterStyle:     l.Blog.FooterStyle,
			ShowAuthor:      l.Blog.ShowAuthor,
			ShowDate:        l.Blog.ShowDate,
			ShowTags:        l.Blog.ShowTags,
			ShowReadingTime: l.Blog.ShowReadingTime,
			ShowPrevNext:    l.Blog.ShowPrevNext,
		},
		Landing: models.LandingLayoutConfig{
			ContentMaxWidth: l.Landing.ContentMaxWidth,
			HeaderStyle:     l.Landing.HeaderStyle,
			HeaderSticky:    l.Landing.HeaderSticky,
			FooterStyle:     l.Landing.FooterStyle,
			HeroEnabled:     l.Landing.HeroEnabled,
		},
		Bare: models.BareLayoutConfig{
			ContentMaxWidth: l.Bare.ContentMaxWidth,
		},
		Defaults: models.LayoutDefaults{
			ContentMaxWidth: l.Defaults.ContentMaxWidth,
			HeaderSticky:    l.Defaults.HeaderSticky,
			FooterSticky:    l.Defaults.FooterSticky,
		},
	}
}

// Sidebar-related JSON structs

type jsonSidebarConfig struct {
	Enabled      *bool                             `json:"enabled"`
	Position     string                            `json:"position"`
	Width        string                            `json:"width"`
	Collapsible  *bool                             `json:"collapsible"`
	DefaultOpen  *bool                             `json:"default_open"`
	Nav          []jsonSidebarNavItem              `json:"nav"`
	Title        string                            `json:"title"`
	Paths        map[string]*jsonPathSidebarConfig `json:"paths"`
	MultiFeed    *bool                             `json:"multi_feed"`
	Feeds        []string                          `json:"feeds"`
	FeedSections []jsonMultiFeedSection            `json:"feed_sections"`
	AutoGenerate *jsonSidebarAutoGenerate          `json:"auto_generate"`
}

type jsonSidebarNavItem struct {
	Title    string               `json:"title"`
	Href     string               `json:"href"`
	Children []jsonSidebarNavItem `json:"children"`
}

type jsonPathSidebarConfig struct {
	Title        string                   `json:"title"`
	AutoGenerate *jsonSidebarAutoGenerate `json:"auto_generate"`
	Items        []jsonSidebarNavItem     `json:"items"`
	Feed         string                   `json:"feed"`
	Position     string                   `json:"position"`
	Collapsible  *bool                    `json:"collapsible"`
}

type jsonSidebarAutoGenerate struct {
	Directory string   `json:"directory"`
	OrderBy   string   `json:"order_by"`
	Reverse   *bool    `json:"reverse"`
	MaxDepth  int      `json:"max_depth"`
	Exclude   []string `json:"exclude"`
}

type jsonMultiFeedSection struct {
	Feed      string `json:"feed"`
	Title     string `json:"title"`
	Collapsed *bool  `json:"collapsed"`
	MaxItems  int    `json:"max_items"`
}

func convertJSONSidebarNavItems(items []jsonSidebarNavItem) []models.SidebarNavItem {
	result := make([]models.SidebarNavItem, len(items))
	for i, item := range items {
		result[i] = models.SidebarNavItem{
			Title:    item.Title,
			Href:     item.Href,
			Children: convertJSONSidebarNavItems(item.Children),
		}
	}
	return result
}

func (s *jsonSidebarConfig) toSidebarConfig() models.SidebarConfig {
	config := models.SidebarConfig{
		Enabled:     s.Enabled,
		Position:    s.Position,
		Width:       s.Width,
		Collapsible: s.Collapsible,
		DefaultOpen: s.DefaultOpen,
		Nav:         convertJSONSidebarNavItems(s.Nav),
		Title:       s.Title,
		MultiFeed:   s.MultiFeed,
		Feeds:       s.Feeds,
	}

	// Convert paths
	if len(s.Paths) > 0 {
		config.Paths = make(map[string]*models.PathSidebarConfig)
		for path, pathConfig := range s.Paths {
			var autoGen *models.SidebarAutoGenerate
			if pathConfig.AutoGenerate != nil {
				autoGen = &models.SidebarAutoGenerate{
					Directory: pathConfig.AutoGenerate.Directory,
					OrderBy:   pathConfig.AutoGenerate.OrderBy,
					Reverse:   pathConfig.AutoGenerate.Reverse,
					MaxDepth:  pathConfig.AutoGenerate.MaxDepth,
					Exclude:   pathConfig.AutoGenerate.Exclude,
				}
			}
			config.Paths[path] = &models.PathSidebarConfig{
				Title:        pathConfig.Title,
				AutoGenerate: autoGen,
				Items:        convertJSONSidebarNavItems(pathConfig.Items),
				Feed:         pathConfig.Feed,
				Position:     pathConfig.Position,
				Collapsible:  pathConfig.Collapsible,
			}
		}
	}

	// Convert feed sections
	if len(s.FeedSections) > 0 {
		config.FeedSections = make([]models.MultiFeedSection, len(s.FeedSections))
		for i, section := range s.FeedSections {
			config.FeedSections[i] = models.MultiFeedSection{
				Feed:      section.Feed,
				Title:     section.Title,
				Collapsed: section.Collapsed,
				MaxItems:  section.MaxItems,
			}
		}
	}

	// Convert auto-generate
	if s.AutoGenerate != nil {
		config.AutoGenerate = &models.SidebarAutoGenerate{
			Directory: s.AutoGenerate.Directory,
			OrderBy:   s.AutoGenerate.OrderBy,
			Reverse:   s.AutoGenerate.Reverse,
			MaxDepth:  s.AutoGenerate.MaxDepth,
			Exclude:   s.AutoGenerate.Exclude,
		}
	}

	return config
}

// TOC-related JSON structs

type jsonTocConfig struct {
	Enabled     *bool  `json:"enabled"`
	Position    string `json:"position"`
	Width       string `json:"width"`
	MinDepth    int    `json:"min_depth"`
	MaxDepth    int    `json:"max_depth"`
	Title       string `json:"title"`
	Collapsible *bool  `json:"collapsible"`
	DefaultOpen *bool  `json:"default_open"`
	ScrollSpy   *bool  `json:"scroll_spy"`
}

func (t *jsonTocConfig) toTocConfig() models.TocConfig {
	return models.TocConfig{
		Enabled:     t.Enabled,
		Position:    t.Position,
		Width:       t.Width,
		MinDepth:    t.MinDepth,
		MaxDepth:    t.MaxDepth,
		Title:       t.Title,
		Collapsible: t.Collapsible,
		DefaultOpen: t.DefaultOpen,
		ScrollSpy:   t.ScrollSpy,
	}
}

// Header layout JSON structs

type jsonHeaderLayoutConfig struct {
	Style           string `json:"style"`
	Sticky          *bool  `json:"sticky"`
	ShowLogo        *bool  `json:"show_logo"`
	ShowTitle       *bool  `json:"show_title"`
	ShowNav         *bool  `json:"show_nav"`
	ShowSearch      *bool  `json:"show_search"`
	ShowThemeToggle *bool  `json:"show_theme_toggle"`
}

func (h *jsonHeaderLayoutConfig) toHeaderLayoutConfig() models.HeaderLayoutConfig {
	return models.HeaderLayoutConfig{
		Style:           h.Style,
		Sticky:          h.Sticky,
		ShowLogo:        h.ShowLogo,
		ShowTitle:       h.ShowTitle,
		ShowNav:         h.ShowNav,
		ShowSearch:      h.ShowSearch,
		ShowThemeToggle: h.ShowThemeToggle,
	}
}

// Blogroll-related JSON structs

type jsonBlogrollConfig struct {
	Enabled              bool                     `json:"enabled"`
	BlogrollSlug         string                   `json:"blogroll_slug"`
	ReaderSlug           string                   `json:"reader_slug"`
	CacheDir             string                   `json:"cache_dir"`
	CacheDuration        string                   `json:"cache_duration"`
	Timeout              int                      `json:"timeout"`
	ConcurrentRequests   int                      `json:"concurrent_requests"`
	MaxEntriesPerFeed    int                      `json:"max_entries_per_feed"`
	FallbackImageService string                   `json:"fallback_image_service"`
	Feeds                []jsonExternalFeedConfig `json:"feeds"`
	Templates            jsonBlogrollTemplates    `json:"templates"`
}

type jsonExternalFeedConfig struct {
	URL           string   `json:"url"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Category      string   `json:"category"`
	Tags          []string `json:"tags"`
	Active        *bool    `json:"active"`
	SiteURL       string   `json:"site_url"`
	ImageURL      string   `json:"image_url"`
	Handle        string   `json:"handle"`
	Aliases       []string `json:"aliases,omitempty"`
	MaxEntries    *int     `json:"max_entries,omitempty"`
	Primary       *bool    `json:"primary,omitempty"`
	PrimaryPerson string   `json:"primary_person"`
}

type jsonBlogrollTemplates struct {
	Blogroll string `json:"blogroll"`
	Reader   string `json:"reader"`
}

//nolint:dupl // Intentional duplication - each format has its own conversion method
func (b *jsonBlogrollConfig) toBlogrollConfig() models.BlogrollConfig {
	config := models.BlogrollConfig{
		Enabled:              b.Enabled,
		BlogrollSlug:         b.BlogrollSlug,
		ReaderSlug:           b.ReaderSlug,
		CacheDir:             b.CacheDir,
		CacheDuration:        b.CacheDuration,
		Timeout:              b.Timeout,
		ConcurrentRequests:   b.ConcurrentRequests,
		MaxEntriesPerFeed:    b.MaxEntriesPerFeed,
		FallbackImageService: b.FallbackImageService,
		Templates: models.BlogrollTemplates{
			Blogroll: b.Templates.Blogroll,
			Reader:   b.Templates.Reader,
		},
	}

	for i := range b.Feeds {
		fc := &b.Feeds[i]
		config.Feeds = append(config.Feeds, models.ExternalFeedConfig{
			URL:           fc.URL,
			Title:         fc.Title,
			Description:   fc.Description,
			Category:      fc.Category,
			Tags:          fc.Tags,
			Active:        fc.Active,
			SiteURL:       fc.SiteURL,
			ImageURL:      fc.ImageURL,
			Handle:        fc.Handle,
			Aliases:       fc.Aliases,
			MaxEntries:    fc.MaxEntries,
			Primary:       fc.Primary,
			PrimaryPerson: fc.PrimaryPerson,
		})
	}

	return config
}

func (c *jsonComponentsConfig) toComponentsConfig() models.ComponentsConfig {
	config := models.ComponentsConfig{
		Nav: models.NavComponentConfig{
			Enabled:  c.Nav.Enabled,
			Position: c.Nav.Position,
			Style:    c.Nav.Style,
		},
		Footer: models.FooterComponentConfig{
			Enabled:       c.Footer.Enabled,
			Text:          c.Footer.Text,
			ShowCopyright: c.Footer.ShowCopyright,
		},
		DocSidebar: models.DocSidebarConfig{
			Enabled:  c.DocSidebar.Enabled,
			Position: c.DocSidebar.Position,
			Width:    c.DocSidebar.Width,
			MinDepth: c.DocSidebar.MinDepth,
			MaxDepth: c.DocSidebar.MaxDepth,
		},
	}

	// Convert nav items
	for _, item := range c.Nav.Items {
		config.Nav.Items = append(config.Nav.Items, models.NavItem{
			Label:    item.Label,
			URL:      item.URL,
			External: item.External,
		})
	}

	// Convert footer links
	for _, link := range c.Footer.Links {
		config.Footer.Links = append(config.Footer.Links, models.NavItem{
			Label:    link.Label,
			URL:      link.URL,
			External: link.External,
		})
	}

	return config
}

func (p *jsonPostFormatsConfig) toPostFormatsConfig() models.PostFormatsConfig {
	return models.PostFormatsConfig{
		HTML:     p.HTML,
		Markdown: p.Markdown,
		Text:     p.Text,
		OG:       p.OG,
	}
}

// configSource interface implementation for jsonConfig.
func (c *jsonConfig) getBaseConfig() baseConfigData {
	return baseConfigData{
		OutputDir:     c.OutputDir,
		URL:           c.URL,
		Title:         c.Title,
		Description:   c.Description,
		Author:        c.Author,
		AssetsDir:     c.AssetsDir,
		TemplatesDir:  c.TemplatesDir,
		Hooks:         c.Hooks,
		DisabledHooks: c.DisabledHooks,
		GlobPatterns:  c.Glob.Patterns,
		UseGitignore:  c.Glob.UseGitignore,
		Extensions:    c.Markdown.Extensions,
		Concurrency:   c.Concurrency,
		Theme:         c.Theme.toThemeConfig(),
		Footer:        c.Footer.toFooterConfig(),
	}
}

func (c *jsonConfig) getNavItems() []navItemData {
	items := make([]navItemData, len(c.Nav))
	for i, nav := range c.Nav {
		items[i] = navItemData(nav)
	}
	return items
}

func (c *jsonConfig) getFeeds() []feedConfigConverter {
	feeds := make([]feedConfigConverter, len(c.Feeds))
	for i := range c.Feeds {
		feeds[i] = &c.Feeds[i]
	}
	return feeds
}

func (c *jsonConfig) getFeedDefaults() feedDefaultsConverter { return &c.FeedDefaults }
func (c *jsonConfig) getPostFormats() postFormatsConverter   { return &c.PostFormats }
func (c *jsonConfig) getSEO() seoConverter                   { return &c.SEO }
func (c *jsonConfig) getIndieAuth() indieAuthConverter       { return &c.IndieAuth }
func (c *jsonConfig) getWebmention() webmentionConverter     { return &c.Webmention }
func (c *jsonConfig) getComponents() componentsConverter     { return &c.Components }
func (c *jsonConfig) getLayout() layoutConverter             { return &c.Layout }
func (c *jsonConfig) getSidebar() sidebarConverter           { return &c.Sidebar }
func (c *jsonConfig) getToc() tocConverter                   { return &c.Toc }
func (c *jsonConfig) getHeader() headerConverter             { return &c.Header }
func (c *jsonConfig) getBlogroll() blogrollConverter         { return &c.Blogroll }

func (c *jsonConfig) toConfig() *models.Config {
	return buildConfig(c)
}

func (f *jsonFooterConfig) toFooterConfig() models.FooterConfig {
	return models.FooterConfig{
		Text:          f.Text,
		ShowCopyright: f.ShowCopyright,
	}
}

func (f *jsonFeedConfig) toFeedConfig() models.FeedConfig {
	return models.FeedConfig{
		Slug:            f.Slug,
		Title:           f.Title,
		Description:     f.Description,
		Filter:          f.Filter,
		Sort:            f.Sort,
		Reverse:         f.Reverse,
		ItemsPerPage:    f.ItemsPerPage,
		OrphanThreshold: f.OrphanThreshold,
		PaginationType:  models.PaginationType(f.PaginationType),
		Formats:         f.Formats.toFeedFormats(),
		Templates:       f.Templates.toFeedTemplates(),
	}
}

func (f *jsonFeedFormats) toFeedFormats() models.FeedFormats {
	formats := models.FeedFormats{}
	if f.HTML != nil {
		formats.HTML = *f.HTML
	}
	if f.RSS != nil {
		formats.RSS = *f.RSS
	}
	if f.Atom != nil {
		formats.Atom = *f.Atom
	}
	if f.JSON != nil {
		formats.JSON = *f.JSON
	}
	if f.Markdown != nil {
		formats.Markdown = *f.Markdown
	}
	if f.Text != nil {
		formats.Text = *f.Text
	}
	if f.Sitemap != nil {
		formats.Sitemap = *f.Sitemap
	}
	return formats
}

func (t *jsonFeedTemplates) toFeedTemplates() models.FeedTemplates {
	return models.FeedTemplates{
		HTML:    t.HTML,
		RSS:     t.RSS,
		Atom:    t.Atom,
		JSON:    t.JSON,
		Card:    t.Card,
		Sitemap: t.Sitemap,
	}
}

func (d *jsonFeedDefaults) toFeedDefaults() models.FeedDefaults {
	return models.FeedDefaults{
		ItemsPerPage:    d.ItemsPerPage,
		OrphanThreshold: d.OrphanThreshold,
		PaginationType:  models.PaginationType(d.PaginationType),
		Formats:         d.Formats.toFeedFormats(),
		Templates:       d.Templates.toFeedTemplates(),
		Syndication: models.SyndicationConfig{
			MaxItems:       d.Syndication.MaxItems,
			IncludeContent: d.Syndication.IncludeContent,
		},
	}
}
