package config

import (
	"encoding/json"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

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
	OutputDir     string                `toml:"output_dir"`
	URL           string                `toml:"url"`
	Title         string                `toml:"title"`
	Description   string                `toml:"description"`
	Author        string                `toml:"author"`
	AssetsDir     string                `toml:"assets_dir"`
	TemplatesDir  string                `toml:"templates_dir"`
	Nav           []tomlNavItem         `toml:"nav"`
	Footer        tomlFooterConfig      `toml:"footer"`
	Hooks         []string              `toml:"hooks"`
	DisabledHooks []string              `toml:"disabled_hooks"`
	Glob          tomlGlobConfig        `toml:"glob"`
	Markdown      tomlMarkdownConfig    `toml:"markdown"`
	Feeds         []tomlFeedConfig      `toml:"feeds"`
	FeedDefaults  tomlFeedDefaults      `toml:"feed_defaults"`
	Concurrency   int                   `toml:"concurrency"`
	Theme         tomlThemeConfig       `toml:"theme"`
	PostFormats   tomlPostFormatsConfig `toml:"post_formats"`
	SEO           tomlSEOConfig         `toml:"seo"`
	IndieAuth     tomlIndieAuthConfig   `toml:"indieauth"`
	Webmention    tomlWebmentionConfig  `toml:"webmention"`
	Components    tomlComponentsConfig  `toml:"components"`
	UnknownFields map[string]any        `toml:"-"`
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
	Name      string            `toml:"name"`
	Palette   string            `toml:"palette"`
	Variables map[string]string `toml:"variables"`
	CustomCSS string            `toml:"custom_css"`
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
}

type tomlFeedTemplates struct {
	HTML string `toml:"html"`
	RSS  string `toml:"rss"`
	Atom string `toml:"atom"`
	JSON string `toml:"json"`
	Card string `toml:"card"`
}

type tomlFeedDefaults struct {
	ItemsPerPage    int                   `toml:"items_per_page"`
	OrphanThreshold int                   `toml:"orphan_threshold"`
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

type tomlComponentsConfig struct {
	Nav        tomlNavComponentConfig        `toml:"nav"`
	Footer     tomlFooterComponentConfig     `toml:"footer"`
	DocSidebar tomlDocSidebarComponentConfig `toml:"doc_sidebar"`
}

type tomlNavComponentConfig struct {
	Enabled  *bool  `toml:"enabled"`
	Position string `toml:"position"`
	Style    string `toml:"style"`
}

type tomlFooterComponentConfig struct {
	Enabled *bool  `toml:"enabled"`
	Content string `toml:"content"`
}

type tomlDocSidebarComponentConfig struct {
	Enabled  *bool  `toml:"enabled"`
	Position string `toml:"position"`
	MinDepth int    `toml:"min_depth"`
	MaxDepth int    `toml:"max_depth"`
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

func (c *tomlComponentsConfig) toComponentsConfig() models.ComponentsConfig {
	return models.ComponentsConfig{
		Nav: models.NavComponentConfig{
			Enabled:  c.Nav.Enabled,
			Position: c.Nav.Position,
			Style:    c.Nav.Style,
		},
		Footer: models.FooterComponentConfig{
			Enabled: c.Footer.Enabled,
			Content: c.Footer.Content,
		},
		DocSidebar: models.DocSidebarComponentConfig{
			Enabled:  c.DocSidebar.Enabled,
			Position: c.DocSidebar.Position,
			MinDepth: c.DocSidebar.MinDepth,
			MaxDepth: c.DocSidebar.MaxDepth,
		},
	}
}

func (p *tomlPostFormatsConfig) toPostFormatsConfig() models.PostFormatsConfig {
	return models.PostFormatsConfig{
		HTML:     p.HTML,
		Markdown: p.Markdown,
		OG:       p.OG,
	}
}

func (c *tomlConfig) toConfig() *models.Config {
	config := &models.Config{
		OutputDir:     c.OutputDir,
		URL:           c.URL,
		Title:         c.Title,
		Description:   c.Description,
		Author:        c.Author,
		AssetsDir:     c.AssetsDir,
		TemplatesDir:  c.TemplatesDir,
		Hooks:         c.Hooks,
		DisabledHooks: c.DisabledHooks,
		GlobConfig: models.GlobConfig{
			Patterns: c.Glob.Patterns,
		},
		MarkdownConfig: models.MarkdownConfig{
			Extensions: c.Markdown.Extensions,
		},
		Concurrency: c.Concurrency,
		Theme:       c.Theme.toThemeConfig(),
		Footer:      c.Footer.toFooterConfig(),
	}

	if c.Glob.UseGitignore != nil {
		config.GlobConfig.UseGitignore = *c.Glob.UseGitignore
	}

	// Convert nav items
	for _, nav := range c.Nav {
		config.Nav = append(config.Nav, models.NavItem{
			Label:    nav.Label,
			URL:      nav.URL,
			External: nav.External,
		})
	}

	// Convert feeds
	for i := range c.Feeds {
		config.Feeds = append(config.Feeds, c.Feeds[i].toFeedConfig())
	}

	// Convert feed defaults
	config.FeedDefaults = c.FeedDefaults.toFeedDefaults()

	// Convert post formats
	config.PostFormats = c.PostFormats.toPostFormatsConfig()

	// Convert SEO config
	config.SEO = c.SEO.toSEOConfig()

	// Convert IndieAuth and Webmention config
	config.IndieAuth = c.IndieAuth.toIndieAuthConfig()
	config.Webmention = c.Webmention.toWebmentionConfig()

	// Convert Components config
	config.Components = c.Components.toComponentsConfig()

	return config
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
		Name:      t.Name,
		Palette:   t.Palette,
		Variables: variables,
		CustomCSS: t.CustomCSS,
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
	return formats
}

func (t *tomlFeedTemplates) toFeedTemplates() models.FeedTemplates {
	return models.FeedTemplates{
		HTML: t.HTML,
		RSS:  t.RSS,
		Atom: t.Atom,
		JSON: t.JSON,
		Card: t.Card,
	}
}

func (d *tomlFeedDefaults) toFeedDefaults() models.FeedDefaults {
	return models.FeedDefaults{
		ItemsPerPage:    d.ItemsPerPage,
		OrphanThreshold: d.OrphanThreshold,
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
	OutputDir     string                `yaml:"output_dir"`
	URL           string                `yaml:"url"`
	Title         string                `yaml:"title"`
	Description   string                `yaml:"description"`
	Author        string                `yaml:"author"`
	AssetsDir     string                `yaml:"assets_dir"`
	TemplatesDir  string                `yaml:"templates_dir"`
	Nav           []yamlNavItem         `yaml:"nav"`
	Footer        yamlFooterConfig      `yaml:"footer"`
	Hooks         []string              `yaml:"hooks"`
	DisabledHooks []string              `yaml:"disabled_hooks"`
	Glob          yamlGlobConfig        `yaml:"glob"`
	Markdown      yamlMarkdownConfig    `yaml:"markdown"`
	Feeds         []yamlFeedConfig      `yaml:"feeds"`
	FeedDefaults  yamlFeedDefaults      `yaml:"feed_defaults"`
	Concurrency   int                   `yaml:"concurrency"`
	PostFormats   yamlPostFormatsConfig `yaml:"post_formats"`
	IndieAuth     yamlIndieAuthConfig   `yaml:"indieauth"`
	Webmention    yamlWebmentionConfig  `yaml:"webmention"`
	Components    yamlComponentsConfig  `yaml:"components"`
	SEO           yamlSEOConfig         `yaml:"seo"`
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
}

type yamlFeedTemplates struct {
	HTML string `yaml:"html"`
	RSS  string `yaml:"rss"`
	Atom string `yaml:"atom"`
	JSON string `yaml:"json"`
	Card string `yaml:"card"`
}

type yamlFeedDefaults struct {
	ItemsPerPage    int                   `yaml:"items_per_page"`
	OrphanThreshold int                   `yaml:"orphan_threshold"`
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

type yamlComponentsConfig struct {
	Nav        yamlNavComponentConfig        `yaml:"nav"`
	Footer     yamlFooterComponentConfig     `yaml:"footer"`
	DocSidebar yamlDocSidebarComponentConfig `yaml:"doc_sidebar"`
}

type yamlNavComponentConfig struct {
	Enabled  *bool  `yaml:"enabled"`
	Position string `yaml:"position"`
	Style    string `yaml:"style"`
}

type yamlFooterComponentConfig struct {
	Enabled *bool  `yaml:"enabled"`
	Content string `yaml:"content"`
}

type yamlDocSidebarComponentConfig struct {
	Enabled  *bool  `yaml:"enabled"`
	Position string `yaml:"position"`
	MinDepth int    `yaml:"min_depth"`
	MaxDepth int    `yaml:"max_depth"`
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

func (c *yamlComponentsConfig) toComponentsConfig() models.ComponentsConfig {
	return models.ComponentsConfig{
		Nav: models.NavComponentConfig{
			Enabled:  c.Nav.Enabled,
			Position: c.Nav.Position,
			Style:    c.Nav.Style,
		},
		Footer: models.FooterComponentConfig{
			Enabled: c.Footer.Enabled,
			Content: c.Footer.Content,
		},
		DocSidebar: models.DocSidebarComponentConfig{
			Enabled:  c.DocSidebar.Enabled,
			Position: c.DocSidebar.Position,
			MinDepth: c.DocSidebar.MinDepth,
			MaxDepth: c.DocSidebar.MaxDepth,
		},
	}
}

func (p *yamlPostFormatsConfig) toPostFormatsConfig() models.PostFormatsConfig {
	return models.PostFormatsConfig{
		HTML:     p.HTML,
		Markdown: p.Markdown,
		OG:       p.OG,
	}
}

func (c *yamlConfig) toConfig() *models.Config {
	config := &models.Config{
		OutputDir:     c.OutputDir,
		URL:           c.URL,
		Title:         c.Title,
		Description:   c.Description,
		Author:        c.Author,
		AssetsDir:     c.AssetsDir,
		TemplatesDir:  c.TemplatesDir,
		Hooks:         c.Hooks,
		DisabledHooks: c.DisabledHooks,
		GlobConfig: models.GlobConfig{
			Patterns: c.Glob.Patterns,
		},
		MarkdownConfig: models.MarkdownConfig{
			Extensions: c.Markdown.Extensions,
		},
		Concurrency: c.Concurrency,
		Footer:      c.Footer.toFooterConfig(),
	}

	if c.Glob.UseGitignore != nil {
		config.GlobConfig.UseGitignore = *c.Glob.UseGitignore
	}

	// Convert nav items
	for _, nav := range c.Nav {
		config.Nav = append(config.Nav, models.NavItem{
			Label:    nav.Label,
			URL:      nav.URL,
			External: nav.External,
		})
	}

	// Convert feeds
	for i := range c.Feeds {
		config.Feeds = append(config.Feeds, c.Feeds[i].toFeedConfig())
	}

	// Convert feed defaults
	config.FeedDefaults = c.FeedDefaults.toFeedDefaults()

	// Convert post formats
	config.PostFormats = c.PostFormats.toPostFormatsConfig()

	// Convert SEO config
	config.SEO = c.SEO.toSEOConfig()

	// Convert IndieAuth and Webmention config
	config.IndieAuth = c.IndieAuth.toIndieAuthConfig()
	config.Webmention = c.Webmention.toWebmentionConfig()

	// Convert Components config
	config.Components = c.Components.toComponentsConfig()

	return config
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
	return formats
}

func (t *yamlFeedTemplates) toFeedTemplates() models.FeedTemplates {
	return models.FeedTemplates{
		HTML: t.HTML,
		RSS:  t.RSS,
		Atom: t.Atom,
		JSON: t.JSON,
		Card: t.Card,
	}
}

func (d *yamlFeedDefaults) toFeedDefaults() models.FeedDefaults {
	return models.FeedDefaults{
		ItemsPerPage:    d.ItemsPerPage,
		OrphanThreshold: d.OrphanThreshold,
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
	OutputDir     string                `json:"output_dir"`
	URL           string                `json:"url"`
	Title         string                `json:"title"`
	Description   string                `json:"description"`
	Author        string                `json:"author"`
	AssetsDir     string                `json:"assets_dir"`
	TemplatesDir  string                `json:"templates_dir"`
	Nav           []jsonNavItem         `json:"nav"`
	Footer        jsonFooterConfig      `json:"footer"`
	Hooks         []string              `json:"hooks"`
	DisabledHooks []string              `json:"disabled_hooks"`
	Glob          jsonGlobConfig        `json:"glob"`
	Markdown      jsonMarkdownConfig    `json:"markdown"`
	Feeds         []jsonFeedConfig      `json:"feeds"`
	FeedDefaults  jsonFeedDefaults      `json:"feed_defaults"`
	Concurrency   int                   `json:"concurrency"`
	PostFormats   jsonPostFormatsConfig `json:"post_formats"`
	IndieAuth     jsonIndieAuthConfig   `json:"indieauth"`
	Webmention    jsonWebmentionConfig  `json:"webmention"`
	SEO           jsonSEOConfig         `json:"seo"`
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
}

type jsonFeedTemplates struct {
	HTML string `json:"html"`
	RSS  string `json:"rss"`
	Atom string `json:"atom"`
	JSON string `json:"json"`
	Card string `json:"card"`
}

type jsonFeedDefaults struct {
	ItemsPerPage    int                   `json:"items_per_page"`
	OrphanThreshold int                   `json:"orphan_threshold"`
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

func (p *jsonPostFormatsConfig) toPostFormatsConfig() models.PostFormatsConfig {
	return models.PostFormatsConfig{
		HTML:     p.HTML,
		Markdown: p.Markdown,
		OG:       p.OG,
	}
}

func (c *jsonConfig) toConfig() *models.Config {
	config := &models.Config{
		OutputDir:     c.OutputDir,
		URL:           c.URL,
		Title:         c.Title,
		Description:   c.Description,
		Author:        c.Author,
		AssetsDir:     c.AssetsDir,
		TemplatesDir:  c.TemplatesDir,
		Hooks:         c.Hooks,
		DisabledHooks: c.DisabledHooks,
		GlobConfig: models.GlobConfig{
			Patterns: c.Glob.Patterns,
		},
		MarkdownConfig: models.MarkdownConfig{
			Extensions: c.Markdown.Extensions,
		},
		Concurrency: c.Concurrency,
		Footer:      c.Footer.toFooterConfig(),
	}

	if c.Glob.UseGitignore != nil {
		config.GlobConfig.UseGitignore = *c.Glob.UseGitignore
	}

	// Convert nav items
	for _, nav := range c.Nav {
		config.Nav = append(config.Nav, models.NavItem{
			Label:    nav.Label,
			URL:      nav.URL,
			External: nav.External,
		})
	}

	// Convert feeds
	for i := range c.Feeds {
		config.Feeds = append(config.Feeds, c.Feeds[i].toFeedConfig())
	}

	// Convert feed defaults
	config.FeedDefaults = c.FeedDefaults.toFeedDefaults()

	// Convert post formats
	config.PostFormats = c.PostFormats.toPostFormatsConfig()

	// Convert SEO config
	config.SEO = c.SEO.toSEOConfig()

	// Convert IndieAuth and Webmention config
	config.IndieAuth = c.IndieAuth.toIndieAuthConfig()
	config.Webmention = c.Webmention.toWebmentionConfig()

	return config
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
	return formats
}

func (t *jsonFeedTemplates) toFeedTemplates() models.FeedTemplates {
	return models.FeedTemplates{
		HTML: t.HTML,
		RSS:  t.RSS,
		Atom: t.Atom,
		JSON: t.JSON,
		Card: t.Card,
	}
}

func (d *jsonFeedDefaults) toFeedDefaults() models.FeedDefaults {
	return models.FeedDefaults{
		ItemsPerPage:    d.ItemsPerPage,
		OrphanThreshold: d.OrphanThreshold,
		Formats:         d.Formats.toFeedFormats(),
		Templates:       d.Templates.toFeedTemplates(),
		Syndication: models.SyndicationConfig{
			MaxItems:       d.Syndication.MaxItems,
			IncludeContent: d.Syndication.IncludeContent,
		},
	}
}
