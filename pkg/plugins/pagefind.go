// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// defaultBundleDir is the default directory name for Pagefind search index files.
const defaultBundleDir = "_pagefind"

// PagefindPlugin runs Pagefind to generate a search index after all HTML files are written.
// Pagefind is a static site search tool that creates an optimized search index
// that can be queried entirely client-side.
//
// This plugin runs in the Cleanup stage with PriorityLast to ensure all HTML
// files have been written before indexing begins.
//
// For more information about Pagefind, see: https://pagefind.app/
type PagefindPlugin struct{}

// NewPagefindPlugin creates a new PagefindPlugin.
func NewPagefindPlugin() *PagefindPlugin {
	return &PagefindPlugin{}
}

// Name returns the unique name of the plugin.
func (p *PagefindPlugin) Name() string {
	return pagefindBinaryName
}

// Cleanup runs Pagefind to index the generated site.
// This runs after all HTML files have been written in the Write stage.
func (p *PagefindPlugin) Cleanup(m *lifecycle.Manager) error {
	config := m.Config()
	searchConfig := getSearchConfig(config)

	// Skip if search is disabled
	if !searchConfig.IsEnabled() {
		return nil
	}

	verbose := searchConfig.Pagefind.IsVerbose()

	// Try to find or install Pagefind
	pagefindPath, err := p.findOrInstallPagefind(searchConfig, verbose)
	if err != nil {
		// Log warning but don't fail the build
		fmt.Printf("[pagefind] WARNING: %v\n", err)
		fmt.Printf("[pagefind] The site will work fine, just without search functionality\n")
		return nil
	}

	if pagefindPath == "" {
		// No Pagefind available - already warned
		return nil
	}

	return p.runPagefind(pagefindPath, config, searchConfig, verbose)
}

// findOrInstallPagefind locates or automatically installs the Pagefind binary.
// It first checks the system PATH, then attempts auto-install if enabled.
func (p *PagefindPlugin) findOrInstallPagefind(searchConfig models.SearchConfig, verbose bool) (string, error) {
	// First, check if pagefind is in PATH
	pagefindPath, err := exec.LookPath("pagefind")
	if err == nil {
		return pagefindPath, nil
	}

	// Check if auto-install is enabled
	if !searchConfig.Pagefind.IsAutoInstallEnabled() {
		fmt.Printf("[pagefind] WARNING: pagefind not found in PATH, skipping search index generation\n")
		fmt.Printf("[pagefind] Install with: npm install -g pagefind  OR  cargo install pagefind\n")
		fmt.Printf("[pagefind] Or enable auto_install in config: [search.pagefind] auto_install = true\n")
		return "", nil
	}

	// Check if we have a cached version for the requested version
	installer := NewPagefindInstallerWithConfig(PagefindInstallerConfig{
		Version:  searchConfig.Pagefind.Version,
		CacheDir: searchConfig.Pagefind.CacheDir,
	})
	installer.Verbose = verbose

	// Attempt to install
	if verbose {
		fmt.Printf("[pagefind] Pagefind not found in PATH, attempting auto-install...\n")
	}
	installedPath, err := installer.Install()
	if err != nil {
		return "", fmt.Errorf("auto-install failed: %w", err)
	}

	return installedPath, nil
}

// runPagefind executes the Pagefind CLI to generate the search index.
func (p *PagefindPlugin) runPagefind(pagefindPath string, config *lifecycle.Config, searchConfig models.SearchConfig, verbose bool) error {
	outputDir := config.OutputDir

	// Verify output directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return fmt.Errorf("output directory does not exist: %s", outputDir)
	}

	// Build command arguments
	args := []string{
		"--site", outputDir,
	}

	// Configure output subdirectory (where search index files go)
	// Note: Using --output-subdir instead of deprecated --bundle-dir (deprecated in Pagefind 1.0)
	bundleDir := searchConfig.Pagefind.BundleDir
	if bundleDir == "" {
		bundleDir = defaultBundleDir
	}
	args = append(args, "--output-subdir", bundleDir)

	// Configure root selector if specified
	if searchConfig.Pagefind.RootSelector != "" {
		args = append(args, "--root-selector", searchConfig.Pagefind.RootSelector)
	}

	// Configure exclude selectors
	for _, selector := range searchConfig.Pagefind.ExcludeSelectors {
		args = append(args, "--exclude-selectors", selector)
	}

	// Run pagefind
	if verbose {
		fmt.Printf("[pagefind] Generating search index in %s/%s\n", outputDir, bundleDir)
	}
	cmd := exec.Command(pagefindPath, args...)
	cmd.Dir = "." // Run from project root

	// Capture output - show all in verbose mode, only errors when quiet
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// Capture stderr to show errors only
		var stderrBuf bytes.Buffer
		cmd.Stdout = io.Discard
		cmd.Stderr = &stderrBuf

		if err := cmd.Run(); err != nil {
			// Show captured stderr on error
			if stderrBuf.Len() > 0 {
				fmt.Fprintf(os.Stderr, "%s", stderrBuf.String())
			}
			return fmt.Errorf("pagefind indexing failed: %w", err)
		}

		// Verify the index was created
		indexPath := filepath.Join(outputDir, bundleDir, "pagefind.js")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			return fmt.Errorf("pagefind did not create expected index file: %s", indexPath)
		}

		return nil
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pagefind indexing failed: %w", err)
	}

	// Verify the index was created
	indexPath := filepath.Join(outputDir, bundleDir, "pagefind.js")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return fmt.Errorf("pagefind did not create expected index file: %s", indexPath)
	}

	fmt.Printf("[pagefind] Search index generated successfully\n")
	return nil
}

// Priority returns the plugin priority for the cleanup stage.
// Pagefind runs last in cleanup to ensure all HTML files are written first.
func (p *PagefindPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageCleanup {
		return lifecycle.PriorityLast
	}
	return lifecycle.PriorityDefault
}

// getSearchConfig extracts SearchConfig from lifecycle.Config.Extra.
func getSearchConfig(config *lifecycle.Config) models.SearchConfig {
	if config.Extra != nil {
		if sc, ok := config.Extra["search"].(models.SearchConfig); ok {
			return sc
		}
	}
	return models.NewSearchConfig()
}

// Ensure PagefindPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin         = (*PagefindPlugin)(nil)
	_ lifecycle.CleanupPlugin  = (*PagefindPlugin)(nil)
	_ lifecycle.PriorityPlugin = (*PagefindPlugin)(nil)
)
