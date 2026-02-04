package plugins

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/WaylonWalker/markata-go/pkg/assets"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// CDNAssetsPlugin handles downloading and self-hosting external CDN assets.
// When configured for self-hosting, it:
// 1. Downloads CDN assets during Configure stage
// 2. Copies them to output during Write stage
// 3. Provides URL mappings for templates to use local paths
type CDNAssetsPlugin struct{}

// NewCDNAssetsPlugin creates a new CDNAssetsPlugin.
func NewCDNAssetsPlugin() *CDNAssetsPlugin {
	return &CDNAssetsPlugin{}
}

// Name returns the unique name of the plugin.
func (p *CDNAssetsPlugin) Name() string {
	return "cdn_assets"
}

// Configure downloads CDN assets when self-hosting is enabled.
func (p *CDNAssetsPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	assetsConfig := p.getAssetsConfig(config)

	// Skip if not self-hosting
	if !assetsConfig.IsSelfHosted() {
		return nil
	}

	log.Printf("[cdn_assets] Self-hosting enabled (mode: %s)", assetsConfig.Mode)

	// Create downloader
	cacheDir := assetsConfig.GetCacheDir()
	verifyIntegrity := assetsConfig.IsVerifyIntegrityEnabled()
	downloader := assets.NewDownloader(cacheDir, verifyIntegrity)

	// Download all assets
	ctx := context.Background()
	results := downloader.DownloadAll(ctx, 4)

	// Check for errors and log them (but don't fail - we can still use CDN fallback)
	var successCount, cachedCount, errorCount int
	for _, result := range results {
		if result.Error != nil {
			log.Printf("[cdn_assets] Warning: failed to download %s: %v", result.Asset.Name, result.Error)
			errorCount++
		} else if result.Cached {
			cachedCount++
		} else {
			successCount++
		}
	}
	log.Printf("[cdn_assets] Download complete: %d downloaded, %d cached, %d errors",
		successCount, cachedCount, errorCount)

	// Store URL mappings in config.Extra for templates
	urlMappings := p.buildURLMappings(assetsConfig.GetOutputDir())
	if config.Extra == nil {
		config.Extra = make(map[string]interface{})
	}
	config.Extra["asset_urls"] = urlMappings

	return nil
}

// Write copies cached assets to the output directory.
func (p *CDNAssetsPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	assetsConfig := p.getAssetsConfig(config)

	// Skip if not self-hosting
	if !assetsConfig.IsSelfHosted() {
		return nil
	}

	// Create downloader to access cache
	cacheDir := assetsConfig.GetCacheDir()
	downloader := assets.NewDownloader(cacheDir, false)

	// Determine output directory for vendor assets
	vendorOutputDir := filepath.Join(config.OutputDir, assetsConfig.GetOutputDir())

	// Copy all cached assets to output
	if err := downloader.CopyAllToOutput(vendorOutputDir); err != nil {
		return fmt.Errorf("copying assets to output: %w", err)
	}

	log.Printf("[cdn_assets] Copied assets to %s", vendorOutputDir)

	return nil
}

// getAssetsConfig extracts AssetsConfig from lifecycle.Config.Extra.
func (p *CDNAssetsPlugin) getAssetsConfig(config *lifecycle.Config) *models.AssetsConfig {
	if config.Extra == nil {
		defaultConfig := models.NewAssetsConfig()
		return &defaultConfig
	}

	if assetsConfig, ok := config.Extra["assets"].(models.AssetsConfig); ok {
		return &assetsConfig
	}

	// Return default config if not found
	defaultConfig := models.NewAssetsConfig()
	return &defaultConfig
}

// buildURLMappings creates a map of asset names to their local URLs.
// This allows templates to conditionally use local or CDN URLs.
func (p *CDNAssetsPlugin) buildURLMappings(outputDir string) map[string]string {
	mappings := make(map[string]string)

	for _, asset := range assets.Registry() {
		// Build local URL path (e.g., "/assets/vendor/htmx/htmx.min.js")
		localURL := "/" + filepath.ToSlash(filepath.Join(outputDir, asset.LocalPath))
		mappings[asset.Name] = localURL
	}

	return mappings
}

// Priority returns the plugin priority for each stage.
// Configure: Early - download assets before other plugins need them
// Write: Early - copy assets before HTML is generated so paths resolve
func (p *CDNAssetsPlugin) Priority(stage lifecycle.Stage) int {
	switch stage {
	case lifecycle.StageConfigure:
		return lifecycle.PriorityEarly
	case lifecycle.StageWrite:
		return lifecycle.PriorityEarly
	default:
		return lifecycle.PriorityDefault
	}
}

// Ensure CDNAssetsPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*CDNAssetsPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*CDNAssetsPlugin)(nil)
	_ lifecycle.WritePlugin     = (*CDNAssetsPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*CDNAssetsPlugin)(nil)
)
