package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/spf13/cobra"
)

// Common string constants.
const (
	defaultOutputDir = "output"
)

var (
	// buildClean removes output directory before building.
	buildClean bool

	// buildDryRun shows what would be built without building.
	buildDryRun bool
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

Example usage:
  markata-go build              # Standard build
  markata-go build --clean      # Clean output directory first
  markata-go build --dry-run    # Show what would be built
  markata-go build -v           # Build with verbose output`,
	RunE: runBuildCommand,
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().BoolVar(&buildClean, "clean", false, "clean output directory before build")
	buildCmd.Flags().BoolVar(&buildDryRun, "dry-run", false, "show what would be built without building")
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

	if verbose {
		fmt.Printf("Configuration loaded (output: %s, patterns: %v)\n",
			m.Config().OutputDir, m.Config().GlobPatterns)
	}

	// Clean output directory if requested
	if buildClean {
		outputPath := m.Config().OutputDir
		if outputPath == "" {
			outputPath = defaultOutputDir
		}

		if verbose {
			fmt.Printf("Cleaning output directory: %s\n", outputPath)
		}

		if !buildDryRun {
			if err := os.RemoveAll(outputPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to clean output directory: %w", err)
			}
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
	fmt.Printf("  Feeds generated: %d\n", result.FeedsGenerated)
	fmt.Printf("  Duration: %.2fs\n", result.Duration)
}
