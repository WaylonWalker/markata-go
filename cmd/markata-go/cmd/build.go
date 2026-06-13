package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/buildstats"
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

	// buildFast skips expensive non-essential plugins (minification, CSS purging,
	// Tailwind rebuilds, and Pagefind indexing)
	// for faster development iteration.
	buildFast bool

	// buildBenchmarkJSON writes benchmark details as JSON. Use "-" for stdout.
	buildBenchmarkJSON string

	// buildBenchmarkDetailed prints per-stage benchmark detail.
	buildBenchmarkDetailed bool
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
	  --fast       Skip minification (JS/CSS), CSS purging, Tailwind rebuilds,
	               and Pagefind indexing for faster builds.
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
	buildCmd.Flags().BoolVar(&buildFast, "fast", false, "skip minification, CSS purging, tailwind rebuilds, and pagefind indexing for faster builds")
	buildCmd.Flags().StringVar(&buildBenchmarkJSON, "benchmark-json", "", "write benchmark details as JSON (use '-' for stdout)")
	buildCmd.Flags().Lookup("benchmark-json").NoOptDefVal = "-"
	buildCmd.Flags().BoolVar(&buildBenchmarkDetailed, "benchmark-detailed", false, "print per-stage benchmark resource summaries")
}

func runBuildCommand(_ *cobra.Command, _ []string) error {
	startTime := time.Now()

	verbosef("Starting build...")

	// Create the manager
	m, err := createManager(cfgFile)
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}
	configureLoggerForManager(m)

	// Pass fast mode flag to plugins via config
	if buildFast {
		applyFastMode(m)
	}

	verbosef("Configuration loaded (output: %s, patterns: %v)", m.Config().OutputDir, m.Config().GlobPatterns)

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

	if buildBenchmarkJSON == "-" {
		if err := writeBenchmarkJSON(outWriter(), result); err != nil {
			return fmt.Errorf("writing benchmark json: %w", err)
		}
	} else {
		// Print results
		printBuildResult(result)
	}

	if buildBenchmarkJSON != "" && buildBenchmarkJSON != "-" {
		if err := writeBenchmarkJSONFile(buildBenchmarkJSON, result); err != nil {
			return fmt.Errorf("writing benchmark json file: %w", err)
		}
	}

	// Print warnings
	if len(result.Warnings) > 0 && verbose {
		errln("\nWarnings:")
		for _, w := range result.Warnings {
			errlnf("  - %s", w)
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
		verbosef("Cleaning output directory: %s", outputPath)
		verbosef("Cleaning cache directory: %s", cacheDir)
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
					verbosef("Cleaning external cache: %s", dir)
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
	errln("Dry run mode - no files will be written")

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
		verbosef("Running stage: %s", stage)
		if err := m.RunTo(stage); err != nil {
			return fmt.Errorf("stage %s failed: %w", stage, err)
		}
	}

	// Print what would be written
	outlnf("Files discovered: %d", len(m.Files()))
	outlnf("Posts to process: %d", len(m.Posts()))
	outlnf("Feeds to generate: %d", len(m.Feeds()))

	if verbose {
		errln("\nFiles that would be processed:")
		for _, f := range m.Files() {
			errlnf("  - %s", f)
		}

		errln("\nPosts that would be generated:")
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
			errlnf("  - %s (%s) [%s]", title, p.Slug, status)
		}

		errln("\nFeeds that would be generated:")
		for _, f := range m.Feeds() {
			errlnf("  - %s (%d posts)", f.Name, len(f.Posts))
		}
	}

	outputDir := m.Config().OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir
	}
	outlnf("\nOutput directory: %s", filepath.Clean(outputDir))

	return nil
}

// printBuildResult prints a summary of the build result.
func printBuildResult(result *BuildResult) {
	outln("\n" + colorizeOutput("Build completed successfully!", currentLogTheme.Component))
	outlnf("  %s %d", buildLabel("Posts processed:"), result.PostsProcessed)

	// Only show feeds if any were generated
	if result.FeedsGenerated > 0 {
		outlnf("  %s %d", buildLabel("Feeds generated:"), result.FeedsGenerated)
	}

	// Show blogroll status if configured
	printBlogrollStatus(result.BlogrollStatus)
	printBuildBenchmarkSummary(result.Benchmark)

	outlnf("  %s %.2fs", buildLabel("Duration:"), result.Duration)
}

func printBuildBenchmarkSummary(summary buildstats.Summary) {
	if summary.Total <= 0 {
		return
	}

	outlnf("  %s %s", buildLabel("Resource profile:"), colorizeOutput("estimated wall time", currentLogTheme.Component))
	printResourceLine("CPU", summary.Resources.CPU, summary.Total)
	printResourceLine("Network wait", summary.Resources.NetworkWait, summary.Total)
	printResourceLine("Disk read", summary.Resources.DiskReadWait, summary.Total)
	printResourceLine("Disk write", summary.Resources.DiskWriteWait, summary.Total)
	printResourceLine("Idle", summary.Resources.Idle, summary.Total)

	if verbose || buildBenchmarkDetailed {
		printStageBenchmarkSummary(summary)
	}

	hotspots := topHotspots(summary.Hotspots, 5)
	if len(hotspots) == 0 {
		return
	}

	outlnf("  %s", buildLabel("Hotspots:"))
	for _, hotspot := range hotspots {
		outlnf("    %s %s", hotspotKey(hotspot.Stage, hotspot.Plugin), formatDuration(hotspot.Duration))
	}
}

func printResourceLine(label string, duration, total time.Duration) {
	outlnf("    %-12s %8s (%5.1f%%)", colorizeOutput(label, currentLogTheme.Warning), formatDuration(duration), percent(duration, total))
}

func printStageBenchmarkSummary(summary buildstats.Summary) {
	stages := nonZeroStages(summary.Stages)
	if len(stages) == 0 {
		return
	}

	outlnf("  %s %s", buildLabel("Stage resources:"), colorizeOutput("estimated wall time", currentLogTheme.Component))
	for _, stage := range stages {
		outlnf("    %s total=%-8s cpu=%-8s net=%-8s read=%-8s write=%-8s idle=%-8s",
			colorizeOutput(stage.Stage, stageThemeColor(stage.Stage)),
			formatDuration(stage.Duration),
			formatDuration(stage.Resources.CPU),
			formatDuration(stage.Resources.NetworkWait),
			formatDuration(stage.Resources.DiskReadWait),
			formatDuration(stage.Resources.DiskWriteWait),
			formatDuration(stage.Resources.Idle),
		)
	}
}

func nonZeroStages(stages []buildstats.StageTiming) []buildstats.StageTiming {
	filtered := make([]buildstats.StageTiming, 0, len(stages))
	for _, stage := range stages {
		if stage.Duration <= 0 {
			continue
		}
		filtered = append(filtered, stage)
	}
	return filtered
}

func topHotspots(hotspots []buildstats.Hotspot, limit int) []buildstats.Hotspot {
	if len(hotspots) == 0 || limit <= 0 {
		return nil
	}
	if len(hotspots) < limit {
		limit = len(hotspots)
	}
	return hotspots[:limit]
}

func formatDuration(duration time.Duration) string {
	text := duration.Round(10 * time.Millisecond).String()
	if text != "0s" && strings.HasSuffix(text, "0s") {
		return strings.TrimSuffix(text, "0s") + "s"
	}
	return text
}

func percent(duration, total time.Duration) float64 {
	if total <= 0 {
		return 0
	}
	return float64(duration) / float64(total) * 100
}

func hotspotKey(stage, plugin string) string {
	return colorizeOutput(stage+"/"+plugin, stageThemeColor(stage))
}

func buildLabel(name string) string {
	return colorizeOutput(name, currentLogTheme.Warning)
}

func stageThemeColor(stage string) string {
	if stageColor, ok := currentLogTheme.PhaseColor[stage]; ok && stageColor != "" {
		return stageColor
	}
	return currentLogTheme.Component
}

type benchmarkJSONOutput struct {
	PostsProcessed int                `json:"posts_processed"`
	FeedsGenerated int                `json:"feeds_generated"`
	Duration       float64            `json:"duration_seconds"`
	Warnings       []string           `json:"warnings,omitempty"`
	Benchmark      buildstats.Summary `json:"benchmark"`
	Blogroll       BlogrollStatus     `json:"blogroll"`
}

func writeBenchmarkJSONFile(path string, result *BuildResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeBenchmarkJSON(file, result)
}

func writeBenchmarkJSON(w io.Writer, result *BuildResult) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(benchmarkJSONOutput{
		PostsProcessed: result.PostsProcessed,
		FeedsGenerated: result.FeedsGenerated,
		Duration:       result.Duration,
		Warnings:       result.Warnings,
		Benchmark:      result.Benchmark,
		Blogroll:       result.BlogrollStatus,
	})
}

// printBlogrollStatus prints the blogroll feature status.
func printBlogrollStatus(status BlogrollStatus) {
	if !status.Configured {
		return
	}

	if status.Enabled {
		// Active blogroll - show pages and feed count
		outlnf("  %s /blogroll, /reader (%d %s)", buildLabel("Blogroll:"), status.FeedsFetched, pluralize(status.FeedsFetched, "feed", "feeds"))
	} else if status.FeedsConfigured > 0 {
		// Configured but disabled - show warning
		warnf("Blogroll: feeds configured but enabled=false")
	}
}

// pluralize returns singular or plural form based on count.
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}
