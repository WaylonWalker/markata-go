package models

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
