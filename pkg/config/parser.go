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
	OutputDir     string             `toml:"output_dir"`
	URL           string             `toml:"url"`
	Title         string             `toml:"title"`
	Description   string             `toml:"description"`
	Author        string             `toml:"author"`
	AssetsDir     string             `toml:"assets_dir"`
	TemplatesDir  string             `toml:"templates_dir"`
	Hooks         []string           `toml:"hooks"`
	DisabledHooks []string           `toml:"disabled_hooks"`
	Glob          tomlGlobConfig     `toml:"glob"`
	Markdown      tomlMarkdownConfig `toml:"markdown"`
	Feeds         []tomlFeedConfig   `toml:"feeds"`
	FeedDefaults  tomlFeedDefaults   `toml:"feed_defaults"`
	Concurrency   int                `toml:"concurrency"`
	Theme         tomlThemeConfig    `toml:"theme"`
	UnknownFields map[string]any     `toml:"-"`
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
	}

	if c.Glob.UseGitignore != nil {
		config.GlobConfig.UseGitignore = *c.Glob.UseGitignore
	}

	// Convert feeds
	for i := range c.Feeds {
		config.Feeds = append(config.Feeds, c.Feeds[i].toFeedConfig())
	}

	// Convert feed defaults
	config.FeedDefaults = c.FeedDefaults.toFeedDefaults()

	return config
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
	OutputDir     string             `yaml:"output_dir"`
	URL           string             `yaml:"url"`
	Title         string             `yaml:"title"`
	Description   string             `yaml:"description"`
	Author        string             `yaml:"author"`
	AssetsDir     string             `yaml:"assets_dir"`
	TemplatesDir  string             `yaml:"templates_dir"`
	Hooks         []string           `yaml:"hooks"`
	DisabledHooks []string           `yaml:"disabled_hooks"`
	Glob          yamlGlobConfig     `yaml:"glob"`
	Markdown      yamlMarkdownConfig `yaml:"markdown"`
	Feeds         []yamlFeedConfig   `yaml:"feeds"`
	FeedDefaults  yamlFeedDefaults   `yaml:"feed_defaults"`
	Concurrency   int                `yaml:"concurrency"`
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

//nolint:dupl // Intentional duplication - each format has its own conversion method
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
	}

	if c.Glob.UseGitignore != nil {
		config.GlobConfig.UseGitignore = *c.Glob.UseGitignore
	}

	// Convert feeds
	for i := range c.Feeds {
		config.Feeds = append(config.Feeds, c.Feeds[i].toFeedConfig())
	}

	// Convert feed defaults
	config.FeedDefaults = c.FeedDefaults.toFeedDefaults()

	return config
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
	OutputDir     string             `json:"output_dir"`
	URL           string             `json:"url"`
	Title         string             `json:"title"`
	Description   string             `json:"description"`
	Author        string             `json:"author"`
	AssetsDir     string             `json:"assets_dir"`
	TemplatesDir  string             `json:"templates_dir"`
	Hooks         []string           `json:"hooks"`
	DisabledHooks []string           `json:"disabled_hooks"`
	Glob          jsonGlobConfig     `json:"glob"`
	Markdown      jsonMarkdownConfig `json:"markdown"`
	Feeds         []jsonFeedConfig   `json:"feeds"`
	FeedDefaults  jsonFeedDefaults   `json:"feed_defaults"`
	Concurrency   int                `json:"concurrency"`
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

//nolint:dupl // Intentional duplication - each format has its own conversion method
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
	}

	if c.Glob.UseGitignore != nil {
		config.GlobConfig.UseGitignore = *c.Glob.UseGitignore
	}

	// Convert feeds
	for i := range c.Feeds {
		config.Feeds = append(config.Feeds, c.Feeds[i].toFeedConfig())
	}

	// Convert feed defaults
	config.FeedDefaults = c.FeedDefaults.toFeedDefaults()

	return config
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
