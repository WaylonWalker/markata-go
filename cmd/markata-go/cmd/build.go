package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/spf13/cobra"
)

// Common string constants.
const (
	defaultOutputDir = "output"
)

var (
	// buildClean removes output directory and build cache before building.
	buildClean bool

	// buildCleanAll removes output, build cache, AND external plugin caches before building.
	buildCleanAll bool

	// buildDryRun shows what would be built without building.
	buildDryRun bool

	// buildFast skips expensive non-essential plugins (minification, CSS purging)
	// for faster development iteration.
	buildFast bool
)

// buildCmd represents the build command.
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the static site",
	Long: `Build runs all lifecycle stages to generate the static site.

The build process includes:
  1. Configure - Load and validate configuration
  2. Glob - Discover content files
  3. Load - Parse markdown files and frontmatter
  4. Transform - Process Jinja templates in markdown
  5. Render - Convert markdown to HTML
  6. Collect - Build feeds and archives
  7. Write - Output files to disk

Clean modes:
  --clean      Remove output directory and build cache (.markata/).
               This is cheap to rebuild and fixes most caching issues.

  --clean-all  Everything --clean does, plus remove external plugin caches
               (blogroll feeds, embeds metadata, mentions, webmentions).
               These are expensive to re-fetch from remote servers.

Fast mode:
  --fast       Skip minification (JS/CSS) and CSS purging for faster builds.
               Useful during development iteration when you don't need
               optimized output.

Example usage:
  markata-go build              # Standard build
  markata-go build --clean      # Clean build cache + output
  markata-go build --clean-all  # Also nuke external plugin caches
  markata-go build --fast       # Skip minification for faster builds
  markata-go build --dry-run    # Show what would be built
  markata-go build -v           # Build with verbose output`,
	RunE: runBuildCommand,
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().BoolVar(&buildClean, "clean", false, "clean output directory and build cache before build")
	buildCmd.Flags().BoolVar(&buildCleanAll, "clean-all", false, "clean everything including external plugin caches (blogroll, embeds, etc.)")
	buildCmd.Flags().BoolVar(&buildDryRun, "dry-run", false, "show what would be built without building")
	buildCmd.Flags().BoolVar(&buildFast, "fast", false, "skip minification and CSS purging for faster builds")
}

func runBuildCommand(_ *cobra.Command, _ []string) error {
	startTime := time.Now()

	if verbose {
		fmt.Println("Starting build...")
	}

	// Create the manager
	m, err := createManager(cfgFile)
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	// Pass fast mode flag to plugins via config
	if buildFast {
		applyFastMode(m)
	}

	if verbose {
		fmt.Printf("Configuration loaded (output: %s, patterns: %v)\n",
			m.Config().OutputDir, m.Config().GlobPatterns)
	}

	// Clean directories if requested
	if buildCleanAll || buildClean {
		if err := cleanBuildDirs(m); err != nil {
			return err
		}
	}

	// Dry run - just show what would be processed
	if buildDryRun {
		return runDryBuild(m)
	}

	// Run the build
	result, err := runBuild(m)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Calculate duration
	duration := time.Since(startTime)
	result.Duration = duration.Seconds()

	// Print results
	printBuildResult(result)

	// Print warnings
	if len(result.Warnings) > 0 && verbose {
		fmt.Println("\nWarnings:")
		for _, w := range result.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}

	return nil
}

// cleanBuildDirs removes build artifacts before a fresh build.
// --clean removes output dir and .markata/ build cache (Tier 1).
// --clean-all additionally removes external plugin caches (Tier 2).
func cleanBuildDirs(m *lifecycle.Manager) error {
	outputPath := m.Config().OutputDir
	if outputPath == "" {
		outputPath = defaultOutputDir
	}

	// Build cache directory (.markata/) is a sibling of the output directory.
	// Without cleaning it, the glob cache retains stale file lists.
	cacheDir := filepath.Join(outputPath, "..", ".markata")
	if extra := m.Config().Extra; extra != nil {
		if dir, ok := extra["cache_dir"].(string); ok && dir != "" {
			cacheDir = dir
		}
	}

	if verbose {
		fmt.Printf("Cleaning output directory: %s\n", outputPath)
		fmt.Printf("Cleaning cache directory: %s\n", cacheDir)
	}

	if !buildDryRun {
		if err := os.RemoveAll(outputPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to clean output directory: %w", err)
		}
		if err := os.RemoveAll(cacheDir); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to clean cache directory: %w", err)
		}
	}

	// --clean-all: also remove external plugin caches (Tier 2)
	if buildCleanAll {
		if modelsConfig, ok := m.Config().Extra["models_config"].(*models.Config); ok {
			for _, dir := range modelsConfig.ExternalCacheDirs() {
				if verbose {
					fmt.Printf("Cleaning external cache: %s\n", dir)
				}
				if !buildDryRun {
					if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
						return fmt.Errorf("failed to clean external cache %s: %w", dir, err)
					}
				}
			}
		}
	}

	return nil
}

// runDryBuild performs a dry run build showing what would be processed.
func runDryBuild(m *lifecycle.Manager) error {
	fmt.Println("Dry run mode - no files will be written")

	// Run stages up to Write (but not Write)
	stages := []lifecycle.Stage{
		lifecycle.StageConfigure,
		lifecycle.StageValidate,
		lifecycle.StageGlob,
		lifecycle.StageLoad,
		lifecycle.StageTransform,
		lifecycle.StageRender,
		lifecycle.StageCollect,
	}

	for _, stage := range stages {
		if verbose {
			fmt.Printf("Running stage: %s\n", stage)
		}
		if err := m.RunTo(stage); err != nil {
			return fmt.Errorf("stage %s failed: %w", stage, err)
		}
	}

	// Print what would be written
	fmt.Printf("Files discovered: %d\n", len(m.Files()))
	fmt.Printf("Posts to process: %d\n", len(m.Posts()))
	fmt.Printf("Feeds to generate: %d\n", len(m.Feeds()))

	if verbose {
		fmt.Println("\nFiles that would be processed:")
		for _, f := range m.Files() {
			fmt.Printf("  - %s\n", f)
		}

		fmt.Println("\nPosts that would be generated:")
		for _, p := range m.Posts() {
			title := p.Slug
			if p.Title != nil {
				title = *p.Title
			}
			status := "published"
			if p.Draft {
				status = "draft"
			} else if !p.Published {
				status = "unpublished"
			}
			fmt.Printf("  - %s (%s) [%s]\n", title, p.Slug, status)
		}

		fmt.Println("\nFeeds that would be generated:")
		for _, f := range m.Feeds() {
			fmt.Printf("  - %s (%d posts)\n", f.Name, len(f.Posts))
		}
	}

	outputDir := m.Config().OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir
	}
	fmt.Printf("\nOutput directory: %s\n", filepath.Clean(outputDir))

	return nil
}

// printBuildResult prints a summary of the build result.
func printBuildResult(result *BuildResult) {
	fmt.Println("\nBuild completed successfully!")
	fmt.Printf("  Posts processed: %d\n", result.PostsProcessed)

	// Only show feeds if any were generated
	if result.FeedsGenerated > 0 {
		fmt.Printf("  Feeds generated: %d\n", result.FeedsGenerated)
	}

	// Show blogroll status if configured
	printBlogrollStatus(result.BlogrollStatus)

	fmt.Printf("  Duration: %.2fs\n", result.Duration)
}

// printBlogrollStatus prints the blogroll feature status.
func printBlogrollStatus(status BlogrollStatus) {
	if !status.Configured {
		return
	}

	if status.Enabled {
		// Active blogroll - show pages and feed count
		fmt.Printf("  Blogroll: /blogroll, /reader (%d %s)\n",
			status.FeedsFetched, pluralize(status.FeedsFetched, "feed", "feeds"))
	} else if status.FeedsConfigured > 0 {
		// Configured but disabled - show warning
		fmt.Printf("  \u26a0 Blogroll: feeds configured but enabled=false\n")
	}
}

// pluralize returns singular or plural form based on count.
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}
