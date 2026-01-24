package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/config"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/plugins"
	"github.com/spf13/cobra"
)

var (
	// benchmarkSizes is the list of post counts to benchmark.
	benchmarkSizes []int

	// benchmarkKeep keeps the generated test files after benchmark.
	benchmarkKeep bool
)

// BenchmarkResult holds the metrics for a single benchmark run.
type BenchmarkResult struct {
	PostCount    int
	TotalTime    time.Duration
	PostsPerSec  float64
	MemoryMB     float64
	StageTimes   map[string]time.Duration
	Success      bool
	ErrorMessage string
}

// benchmarkCmd represents the benchmark command.
var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run performance benchmarks",
	Long: `Run performance benchmarks to measure build performance at various site sizes.

The benchmark generates test sites with configurable post counts,
measures build time, memory usage, and throughput.

Example usage:
  markata-go benchmark                      # Run with default sizes (10, 100, 1000)
  markata-go benchmark --sizes 10,50,100    # Custom sizes
  markata-go benchmark --sizes 10 --keep    # Keep test files after benchmark
  markata-go benchmark -v                   # Verbose output with stage timings`,
	RunE: runBenchmarkCommand,
}

func init() {
	rootCmd.AddCommand(benchmarkCmd)

	benchmarkCmd.Flags().IntSliceVar(&benchmarkSizes, "sizes", []int{10, 100, 1000}, "comma-separated list of post counts to benchmark")
	benchmarkCmd.Flags().BoolVar(&benchmarkKeep, "keep", false, "keep generated test files after benchmark")
}

func runBenchmarkCommand(_ *cobra.Command, _ []string) error {
	fmt.Println("Markata-go Performance Benchmark")
	fmt.Println("=================================")
	fmt.Println()

	results := make([]*BenchmarkResult, 0, len(benchmarkSizes))

	for _, size := range benchmarkSizes {
		if verbose {
			fmt.Printf("\n--- Benchmarking %d posts ---\n", size)
		} else {
			fmt.Printf("Benchmarking %d posts... ", size)
		}

		result := runSingleBenchmark(size)
		results = append(results, result)

		if result.Success {
			if verbose {
				fmt.Printf("Completed in %.2fs (%.1f posts/sec)\n", result.TotalTime.Seconds(), result.PostsPerSec)
			} else {
				fmt.Printf("%.2fs (%.1f posts/sec)\n", result.TotalTime.Seconds(), result.PostsPerSec)
			}
		} else {
			fmt.Printf("FAILED: %s\n", result.ErrorMessage)
		}
	}

	// Print summary table
	fmt.Println()
	printBenchmarkSummary(results)

	return nil
}

// runSingleBenchmark runs a benchmark for a specific post count.
func runSingleBenchmark(postCount int) *BenchmarkResult {
	result := &BenchmarkResult{
		PostCount:  postCount,
		StageTimes: make(map[string]time.Duration),
	}

	// Create temp directory for test site
	tempDir, err := os.MkdirTemp("", "markata-benchmark-*")
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create temp dir: %v", err)
		return result
	}

	if !benchmarkKeep {
		defer os.RemoveAll(tempDir)
	} else if verbose {
		fmt.Printf("Test site directory: %s\n", tempDir)
	}

	// Generate test posts
	if verbose {
		fmt.Printf("Generating %d test posts...\n", postCount)
	}

	postsDir := filepath.Join(tempDir, "posts")
	if err := os.MkdirAll(postsDir, 0o755); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create posts dir: %v", err)
		return result
	}

	genStart := time.Now()
	if err := generateTestPosts(postsDir, postCount); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to generate posts: %v", err)
		return result
	}

	if verbose {
		fmt.Printf("Generated %d posts in %.2fs\n", postCount, time.Since(genStart).Seconds())
	}

	// Create minimal config
	configPath := filepath.Join(tempDir, "markata-go.toml")
	configContent := fmt.Sprintf(`
url = "https://benchmark.example.com"
title = "Benchmark Site"
output_dir = "%s/output"

[glob]
patterns = ["posts/**/*.md"]
`, tempDir)

	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to write config: %v", err)
		return result
	}

	// Force garbage collection before benchmark and record baseline
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Track peak memory during build
	peakAlloc := memBefore.Alloc

	// Run build
	buildStart := time.Now()

	m, err := createBenchmarkManager(configPath, tempDir)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create manager: %v", err)
		return result
	}

	// Run each stage and time it
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
		if err := m.RunTo(stage); err != nil {
			result.ErrorMessage = fmt.Sprintf("stage %s failed: %v", stage, err)
			return result
		}
		result.StageTimes[string(stage)] = time.Since(stageStart)

		// Check peak memory after each stage
		var memStage runtime.MemStats
		runtime.ReadMemStats(&memStage)
		if memStage.Alloc > peakAlloc {
			peakAlloc = memStage.Alloc
		}

		if verbose {
			fmt.Printf("  [%s] %.3fs\n", stage, result.StageTimes[string(stage)].Seconds())
		}
	}

	result.TotalTime = time.Since(buildStart)

	// Use peak allocation relative to baseline for memory measurement
	result.MemoryMB = float64(peakAlloc) / (1024 * 1024)

	// Calculate throughput
	if result.TotalTime.Seconds() > 0 {
		result.PostsPerSec = float64(postCount) / result.TotalTime.Seconds()
	}

	result.Success = true
	return result
}

// createBenchmarkManager creates a manager for benchmarking without relying on global state.
func createBenchmarkManager(cfgPath, workDir string) (*lifecycle.Manager, error) {
	// Change to work directory temporarily
	originalDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting current dir: %w", err)
	}

	if err := os.Chdir(workDir); err != nil {
		return nil, fmt.Errorf("changing to work dir: %w", err)
	}
	defer func() {
		_ = os.Chdir(originalDir) //nolint:errcheck // best-effort restore of original directory
	}()

	// Load config
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
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
	lcConfig.Extra["theme"] = map[string]interface{}{
		"name":       cfg.Theme.Name,
		"palette":    cfg.Theme.Palette,
		"variables":  cfg.Theme.Variables,
		"custom_css": cfg.Theme.CustomCSS,
	}

	// Pass layout configuration for automatic layout selection
	lcConfig.Extra["layout"] = &cfg.Layout

	m.SetConfig(lcConfig)

	if cfg.Concurrency > 0 {
		m.SetConcurrency(cfg.Concurrency)
	}

	// Register default plugins
	m.RegisterPlugins(plugins.DefaultPlugins()...)

	return m, nil
}

// generateTestPosts creates test markdown files with realistic content.
func generateTestPosts(dir string, count int) error {
	//nolint:gosec // Using math/rand is fine for benchmark content generation
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < count; i++ {
		title := generateTitle(rng, i)
		slug := generateSlug(title)
		filename := filepath.Join(dir, fmt.Sprintf("%s.md", slug))

		content := generatePostMarkdown(rng, title, i)

		if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
			return fmt.Errorf("writing post %d: %w", i, err)
		}
	}

	return nil
}

// generateTitle creates a realistic post title.
func generateTitle(rng *rand.Rand, index int) string {
	adjectives := []string{
		"Amazing", "Ultimate", "Complete", "Essential", "Practical",
		"Modern", "Advanced", "Simple", "Quick", "Deep",
	}
	topics := []string{
		"Guide to Go", "Tutorial on Testing", "Introduction to APIs",
		"Overview of Databases", "Walkthrough of Docker", "Exploration of Kubernetes",
		"Analysis of Performance", "Review of Best Practices", "Comparison of Frameworks",
		"Journey into Microservices",
	}

	adj := adjectives[rng.Intn(len(adjectives))]
	topic := topics[rng.Intn(len(topics))]

	return fmt.Sprintf("%s %s Part %d", adj, topic, index+1)
}

// generatePostMarkdown creates a complete post with frontmatter and body.
func generatePostMarkdown(rng *rand.Rand, title string, index int) string {
	tags := generateTags(rng)
	date := generateDate(rng, index)
	body := generateBody(rng)

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: %q\n", title))
	sb.WriteString(fmt.Sprintf("date: %s\n", date.Format("2006-01-02")))
	sb.WriteString("published: true\n")
	sb.WriteString("draft: false\n")
	sb.WriteString(fmt.Sprintf("tags: [%s]\n", formatTags(tags)))
	sb.WriteString(fmt.Sprintf("description: \"A benchmark test post about %s\"\n", title))
	sb.WriteString("---\n\n")
	sb.WriteString(body)

	return sb.String()
}

// generateTags creates a random set of tags.
func generateTags(rng *rand.Rand) []string {
	allTags := []string{
		"go", "golang", "programming", "tutorial", "guide",
		"testing", "api", "web", "backend", "devops",
		"docker", "kubernetes", "cloud", "performance", "best-practices",
	}

	numTags := 2 + rng.Intn(4) // 2-5 tags
	tags := make([]string, numTags)
	used := make(map[int]bool)

	for i := 0; i < numTags; i++ {
		for {
			idx := rng.Intn(len(allTags))
			if !used[idx] {
				used[idx] = true
				tags[i] = allTags[idx]
				break
			}
		}
	}

	return tags
}

func formatTags(tags []string) string {
	quoted := make([]string, len(tags))
	for i, tag := range tags {
		quoted[i] = fmt.Sprintf("%q", tag)
	}
	return strings.Join(quoted, ", ")
}

// generateDate creates a random date within the past year.
func generateDate(rng *rand.Rand, index int) time.Time {
	now := time.Now()
	daysAgo := index + rng.Intn(30) // Spread posts across time
	return now.AddDate(0, 0, -daysAgo)
}

// generateBody creates realistic markdown content (~500 words).
func generateBody(rng *rand.Rand) string {
	var sb strings.Builder

	// Introduction
	sb.WriteString("## Introduction\n\n")
	sb.WriteString(generateParagraph(rng))
	sb.WriteString("\n\n")

	// Main content with subheadings
	sections := []string{"Getting Started", "Core Concepts", "Implementation", "Best Practices"}
	for _, section := range sections {
		sb.WriteString(fmt.Sprintf("## %s\n\n", section))
		sb.WriteString(generateParagraph(rng))
		sb.WriteString("\n\n")

		// Sometimes add a code block
		if rng.Float32() < 0.5 {
			sb.WriteString("```go\n")
			sb.WriteString(generateCodeBlock(rng))
			sb.WriteString("```\n\n")
		}

		// Sometimes add a list
		if rng.Float32() < 0.5 {
			sb.WriteString(generateList(rng))
			sb.WriteString("\n")
		}
	}

	// Conclusion
	sb.WriteString("## Conclusion\n\n")
	sb.WriteString(generateParagraph(rng))
	sb.WriteString("\n")

	return sb.String()
}

// generateParagraph creates a paragraph of lorem ipsum text.
func generateParagraph(rng *rand.Rand) string {
	sentences := []string{
		"This is an important aspect of modern software development.",
		"Understanding these concepts is essential for building scalable applications.",
		"Many developers struggle with this topic initially.",
		"With practice and dedication, mastery is achievable.",
		"The key is to start with fundamentals and build from there.",
		"Best practices have evolved significantly over the years.",
		"Performance considerations should not be overlooked.",
		"Testing is crucial for maintaining code quality.",
		"Documentation helps teams collaborate effectively.",
		"Continuous improvement leads to better outcomes.",
		"The ecosystem provides many tools to help with this.",
		"Community support is invaluable when learning new technologies.",
		"Real-world experience often differs from tutorials.",
		"Edge cases require careful consideration and handling.",
		"Security should be a primary concern from the start.",
	}

	numSentences := 4 + rng.Intn(4) // 4-7 sentences
	var result []string
	for i := 0; i < numSentences; i++ {
		result = append(result, sentences[rng.Intn(len(sentences))])
	}

	return strings.Join(result, " ")
}

// generateCodeBlock creates a simple Go code snippet.
func generateCodeBlock(rng *rand.Rand) string {
	snippets := []string{
		`func main() {
    fmt.Println("Hello, World!")
}
`,
		`type Config struct {
    Name    string
    Value   int
    Enabled bool
}
`,
		`for i := 0; i < 10; i++ {
    process(items[i])
}
`,
		`if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
`,
		`results := make([]Result, 0, len(items))
for _, item := range items {
    results = append(results, process(item))
}
`,
	}

	return snippets[rng.Intn(len(snippets))]
}

// generateList creates a markdown list.
func generateList(rng *rand.Rand) string {
	items := []string{
		"First, set up your development environment",
		"Install the required dependencies",
		"Configure the application settings",
		"Run the test suite to verify everything works",
		"Deploy to a staging environment",
		"Monitor for any issues or errors",
		"Document your changes thoroughly",
		"Review and refactor as needed",
	}

	numItems := 3 + rng.Intn(3) // 3-5 items
	var sb strings.Builder

	for i := 0; i < numItems; i++ {
		sb.WriteString(fmt.Sprintf("- %s\n", items[rng.Intn(len(items))]))
	}

	return sb.String()
}

// printBenchmarkSummary outputs the results in a markdown table format.
func printBenchmarkSummary(results []*BenchmarkResult) {
	fmt.Println("## Benchmark Results")
	fmt.Println()
	fmt.Println("| Posts | Build Time | Posts/sec | Memory |")
	fmt.Println("|-------|------------|-----------|--------|")

	for _, r := range results {
		if r.Success {
			memStr := formatMemory(r.MemoryMB)
			fmt.Printf("| %d | %.2fs | %.1f | %s |\n",
				r.PostCount,
				r.TotalTime.Seconds(),
				r.PostsPerSec,
				memStr,
			)
		} else {
			fmt.Printf("| %d | FAILED | - | - |\n", r.PostCount)
		}
	}

	fmt.Println()

	// Print stage breakdown if verbose
	if verbose {
		fmt.Println("### Stage Breakdown")
		fmt.Println()
		fmt.Println("| Stage | " + stageHeaders(results) + " |")
		fmt.Println("|-------|" + stageDividers(results) + "|")

		stages := []string{"configure", "validate", "glob", "load", "transform", "render", "collect", "write", "cleanup"}
		for _, stage := range stages {
			fmt.Printf("| %s |", stage)
			for _, r := range results {
				if r.Success {
					if t, ok := r.StageTimes[stage]; ok {
						fmt.Printf(" %.3fs |", t.Seconds())
					} else {
						fmt.Printf(" - |")
					}
				} else {
					fmt.Printf(" - |")
				}
			}
			fmt.Println()
		}
		fmt.Println()
	}

	// Notes
	fmt.Println("**Notes:**")
	fmt.Println("- Memory shows peak heap allocation during build")
	fmt.Println("- Results may vary based on system load and hardware")
	if benchmarkKeep {
		fmt.Println("- Test files were kept in temp directories")
	}
}

func stageHeaders(results []*BenchmarkResult) string {
	headers := make([]string, 0, len(results))
	for _, r := range results {
		headers = append(headers, fmt.Sprintf(" %d posts", r.PostCount))
	}
	return strings.Join(headers, " |")
}

func stageDividers(results []*BenchmarkResult) string {
	dividers := make([]string, len(results))
	for i := range dividers {
		dividers[i] = "--------"
	}
	return strings.Join(dividers, "|")
}

func formatMemory(mb float64) string {
	if mb < 0 {
		return "N/A"
	}
	if mb >= 1024 {
		return fmt.Sprintf("%.1fGB", mb/1024)
	}
	return fmt.Sprintf("%.1fMB", mb)
}
