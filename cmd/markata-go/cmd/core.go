package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/config"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/plugins"
)

// createManager creates and configures a lifecycle manager with all plugins.
func createManager(cfgPath string) (*lifecycle.Manager, error) {
	// Load config, with optional merge configs
	var cfg *models.Config
	var err error

	if len(mergeConfigFiles) > 0 {
		// Use merge mode: base config + overrides
		// If cfgPath is empty, discover the base config
		basePath := cfgPath
		if basePath == "" {
			discovered, discoverErr := config.Discover()
			if discoverErr == nil {
				basePath = discovered
			}
		}

		cfg, err = config.LoadWithMerge(basePath, mergeConfigFiles...)
		if err != nil {
			return nil, fmt.Errorf("loading merged config: %w", err)
		}
	} else {
		// Standard load (single config or auto-discover)
		cfg, err = config.Load(cfgPath)
		if err != nil {
			return nil, fmt.Errorf("loading config: %w", err)
		}
	}

	// Apply output directory override from CLI flag
	if outputDir != "" {
		cfg.OutputDir = outputDir
	}

	// Validate config
	validationErrs := config.ValidateConfig(cfg)
	actualErrors, warnings := config.SplitErrorsAndWarnings(validationErrs)

	// Print warnings
	for _, w := range warnings {
		if verbose || isLicenseWarning(w) {
			fmt.Printf("Warning: %v\n", w)
		}
	}

	// Return errors
	if len(actualErrors) > 0 {
		return nil, fmt.Errorf("config validation failed: %w", actualErrors[0])
	}

	// Create manager
	m := lifecycle.NewManager()

	// Convert models.Config to lifecycle.Config
	lcConfig := &lifecycle.Config{
		ContentDir:   ".",
		OutputDir:    cfg.OutputDir,
		GlobPatterns: cfg.GlobConfig.Patterns,
		Extra:        make(map[string]interface{}),
	}

	// Copy config values to Extra for plugins to access
	lcConfig.Extra["url"] = cfg.URL
	lcConfig.Extra["title"] = cfg.Title
	lcConfig.Extra["description"] = cfg.Description
	lcConfig.Extra["author"] = cfg.Author
	lcConfig.Extra["templates_dir"] = cfg.TemplatesDir
	lcConfig.Extra["assets_dir"] = cfg.AssetsDir
	lcConfig.Extra["feeds"] = cfg.Feeds
	lcConfig.Extra["feed_defaults"] = cfg.FeedDefaults
	lcConfig.Extra["use_gitignore"] = cfg.GlobConfig.UseGitignore
	lcConfig.Extra["nav"] = cfg.Nav
	lcConfig.Extra["footer"] = cfg.Footer
	lcConfig.Extra["post_formats"] = cfg.PostFormats
	lcConfig.Extra["websub"] = cfg.WebSub
	lcConfig.Extra["well_known"] = cfg.WellKnown
	lcConfig.Extra["seo"] = cfg.SEO
	lcConfig.Extra["search"] = cfg.Search
	lcConfig.Extra["components"] = cfg.Components
	lcConfig.Extra["header"] = cfg.Header
	lcConfig.Extra["head"] = cfg.Head
	lcConfig.Extra["toc"] = cfg.Toc
	lcConfig.Extra["sidebar"] = cfg.Sidebar

	// Pass theme configuration to plugins
	lcConfig.Extra["theme"] = cfg.Theme

	// Pass layout configuration for automatic layout selection
	lcConfig.Extra["layout"] = &cfg.Layout

	// Pass blogroll configuration
	lcConfig.Extra["blogroll"] = cfg.Blogroll

	// Pass mentions configuration
	lcConfig.Extra["mentions"] = cfg.Mentions

	// Pass assets configuration for self-hosting CDN assets
	lcConfig.Extra["assets"] = cfg.Assets

	// Pass sidebar, toc, and header configurations
	lcConfig.Extra["sidebar"] = cfg.Sidebar
	lcConfig.Extra["toc"] = cfg.Toc
	lcConfig.Extra["header"] = cfg.Header

	// Pass search configuration with verbose flag override from CLI
	searchConfig := cfg.Search
	if verbose {
		// CLI --verbose flag overrides config setting
		v := true
		searchConfig.Pagefind.Verbose = &v
	}
	lcConfig.Extra["search"] = searchConfig

	// Copy arbitrary plugin configs from cfg.Extra (e.g., image_zoom, wikilinks)
	if cfg.Extra != nil {
		for key, value := range cfg.Extra {
			lcConfig.Extra[key] = value
		}
	}

	// Store full models.Config for components that need direct access (e.g., 404 page handler)
	lcConfig.Extra["models_config"] = cfg

	m.SetConfig(lcConfig)

	// Set concurrency if specified
	if cfg.Concurrency > 0 {
		m.SetConcurrency(cfg.Concurrency)
	}

	// Register default plugins
	registerDefaultPlugins(m)

	return m, nil
}

func licenseWarningMessage(cfg *models.Config) string {
	if cfg == nil || !cfg.NeedsLicenseWarning() {
		return ""
	}
	return fmt.Sprintf("License not configured. Set license = %q (recommended) or license = false.", models.DefaultLicenseKey)
}

func isLicenseWarning(err error) bool {
	var vErr config.ValidationError
	if !errors.As(err, &vErr) {
		return false
	}
	return vErr.IsWarn && vErr.Field == "license"
}

// registerDefaultPlugins registers all default plugins to the manager.
func registerDefaultPlugins(m *lifecycle.Manager) {
	// Use the centralized DefaultPlugins() to ensure all plugins are registered
	m.RegisterPlugins(plugins.DefaultPlugins()...)
}

// BuildResult holds the result of a build operation.
type BuildResult struct {
	PostsProcessed int
	FeedsGenerated int
	FilesWritten   int
	Warnings       []string
	Duration       float64

	// BlogrollStatus holds blogroll feature status
	BlogrollStatus BlogrollStatus
}

// BlogrollStatus holds information about the blogroll feature.
type BlogrollStatus struct {
	// Configured indicates if blogroll section exists in config
	Configured bool
	// Enabled indicates if blogroll is enabled
	Enabled bool
	// FeedsConfigured is the number of feeds configured
	FeedsConfigured int
	// FeedsFetched is the number of feeds successfully fetched
	FeedsFetched int
}

// runBuild executes a full build and returns the result.
func runBuild(m *lifecycle.Manager) (*BuildResult, error) {
	// Run all lifecycle stages with verbose output if enabled
	stages := []lifecycle.Stage{
		lifecycle.StageConfigure,
		lifecycle.StageValidate,
		lifecycle.StageGlob,
		lifecycle.StageLoad,
		lifecycle.StageTransform,
		lifecycle.StageRender,
		lifecycle.StageCollect,
		lifecycle.StageWrite,
		lifecycle.StageCleanup,
	}

	for _, stage := range stages {
		stageStart := time.Now()
		if verbose {
			fmt.Printf("  [%s] running...\n", stage)
		}
		if err := m.RunTo(stage); err != nil {
			return nil, fmt.Errorf("stage %s: %w", stage, err)
		}
		if verbose {
			fmt.Printf("  [%s] done in %s\n", stage, time.Since(stageStart).Truncate(100*time.Microsecond))
			switch stage {
			case lifecycle.StageGlob:
				fmt.Printf("  [%s] discovered %d files\n", stage, len(m.Files()))
			case lifecycle.StageLoad:
				fmt.Printf("  [%s] loaded %d posts\n", stage, len(m.Posts()))
			case lifecycle.StageCollect:
				fmt.Printf("  [%s] collected %d feeds\n", stage, len(m.Feeds()))
			case lifecycle.StageConfigure, lifecycle.StageValidate, lifecycle.StageTransform,
				lifecycle.StageRender, lifecycle.StageWrite, lifecycle.StageCleanup:
				// No extra logging for these stages
			}
		}
	}

	// Collect results
	result := &BuildResult{
		PostsProcessed: len(m.Posts()),
		FeedsGenerated: len(m.Feeds()),
	}

	// Collect blogroll status
	result.BlogrollStatus = getBlogrollStatus(m)

	// Collect warnings
	for _, w := range m.Warnings() {
		result.Warnings = append(result.Warnings, w.Error())
	}

	return result, nil
}

// getBlogrollStatus extracts blogroll feature status from the manager.
func getBlogrollStatus(m *lifecycle.Manager) BlogrollStatus {
	status := BlogrollStatus{}

	cfg := m.Config()
	if cfg == nil || cfg.Extra == nil {
		return status
	}

	blogrollVal, ok := cfg.Extra["blogroll"]
	if !ok {
		return status
	}

	blogrollConfig, ok := blogrollVal.(models.BlogrollConfig)
	if !ok {
		return status
	}

	status.Configured = true
	status.Enabled = blogrollConfig.Enabled
	status.FeedsConfigured = len(blogrollConfig.Feeds)

	// Get fetched feeds count from cache
	if feedsVal, ok := m.Cache().Get("blogroll_feeds"); ok {
		if feeds, ok := feedsVal.([]*models.ExternalFeed); ok {
			status.FeedsFetched = len(feeds)
		}
	}

	return status
}
