package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/config"
	"github.com/WaylonWalker/markata-go/pkg/lint"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/spf13/cobra"
)

var (
	// lintFix automatically fixes issues when possible.
	lintFix bool

	// lintDryRun shows which files would be checked without actually linting them.
	lintDryRun bool
)

// ANSI color codes for terminal output.
const (
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorReset  = "\033[0m"
)

// lintCmd represents the lint command.
var lintCmd = &cobra.Command{
	Use:   "lint [files...]",
	Short: "Lint markdown files for common issues",
	Long: `Lint checks markdown files for common issues that can cause build failures.

The linter detects:
  - Duplicate YAML keys in frontmatter
  - Invalid date formats (non-ISO 8601)
  - Malformed image links (missing alt text)
  - Protocol-less URLs (should use https://)

When run without arguments, lints all files matching the configured glob patterns
(defaults to **/*.md). Explicit file arguments override config patterns.

Use --fix to automatically fix detected issues.
Use --dry-run to see which files would be checked without actually linting them.

Example usage:
  markata-go lint                        # Lint all configured input files
  markata-go lint --dry-run              # Show which files would be checked
  markata-go lint posts/**/*.md          # Lint specific pattern (overrides config)
  markata-go lint posts/**/*.md --fix    # Lint and auto-fix issues
  markata-go lint pages/about.md         # Lint a specific file`,
	RunE: runLintCommand,
}

func init() {
	rootCmd.AddCommand(lintCmd)

	lintCmd.Flags().BoolVar(&lintFix, "fix", false, "automatically fix issues")
	lintCmd.Flags().BoolVar(&lintDryRun, "dry-run", false, "show which files would be checked without linting")
}

// lintStats tracks linting statistics.
type lintStats struct {
	totalFiles      int
	totalIssues     int
	fixableIssues   int
	totalFixed      int
	filesWithIssues int
	hasErrors       bool
}

func runLintCommand(_ *cobra.Command, args []string) error {
	var files []string
	var err error

	if len(args) > 0 {
		// Explicit file arguments override config patterns
		files, err = expandGlobPatterns(args)
		if err != nil {
			return err
		}
	} else {
		// No arguments: use configured glob patterns
		files, err = getFilesFromConfig()
		if err != nil {
			return err
		}
	}

	if len(files) == 0 {
		return fmt.Errorf("no files to lint")
	}

	// Sort files for consistent output
	sort.Strings(files)

	// Dry run: just show which files would be checked
	if lintDryRun {
		fmt.Printf("Would lint %d file(s):\n", len(files))
		for _, f := range files {
			fmt.Printf("  %s\n", f)
		}
		return nil
	}

	stats := &lintStats{}
	for _, file := range files {
		processFile(file, stats)
	}
	processEncryptionPolicyLint(stats)

	printSummary(stats)

	// Exit with error code if there are errors (not just warnings)
	if stats.hasErrors && !lintFix {
		return fmt.Errorf("linting found errors")
	}

	return nil
}

func processEncryptionPolicyLint(stats *lintStats) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config for encryption lint: %v\n", err)
		stats.hasErrors = true
		return
	}

	if !cfg.Encryption.Enabled {
		return
	}

	results, _, _, err := evaluateEncryptionKeyPolicy(cfg, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking encryption keys: %v\n", err)
		stats.hasErrors = true
		return
	}

	issues := make([]lint.Issue, 0, len(results)+1)
	if !cfg.Encryption.EnforceStrength {
		issues = append(issues, lint.Issue{
			Line:     1,
			Severity: lint.SeverityWarning,
			Message:  "encryption.enforce_strength is false; build will not fail on weak encryption keys",
		})
	}

	for _, result := range results {
		if result.Err == nil {
			continue
		}
		issues = append(issues, lint.Issue{
			Line:     1,
			Severity: lint.SeverityError,
			Message:  fmt.Sprintf("encryption key %q failed policy (%s): %v", result.KeyName, result.EnvName, result.Err),
		})
	}

	if len(issues) == 0 {
		return
	}

	stats.filesWithIssues++
	stats.totalIssues += len(issues)
	fmt.Printf("\n[encryption-config]:\n")
	for _, issue := range issues {
		printIssue(issue)
		if issue.Severity == lint.SeverityError {
			stats.hasErrors = true
		}
	}
}

// getFilesFromConfig discovers files using the configured glob patterns.
func getFilesFromConfig() ([]string, error) {
	// Load config (will use defaults if no config file found)
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Get glob patterns from config, default to **/*.md
	patterns := cfg.GlobConfig.Patterns
	if len(patterns) == 0 {
		patterns = []string{"**/*.md"}
	}

	// Get gitignore setting
	useGitignore := cfg.GlobConfig.UseGitignore

	// Parse .gitignore if enabled
	var gitignorePatterns []string
	if useGitignore {
		gitignorePatterns = loadGitignore(".")
	}

	// Get absolute path for base directory
	absBaseDir, err := filepath.Abs(".")
	if err != nil {
		return nil, fmt.Errorf("getting absolute path: %w", err)
	}

	// Expand glob patterns
	fileSet := make(map[string]struct{})
	for _, pattern := range patterns {
		fullPattern := pattern
		if !filepath.IsAbs(pattern) {
			fullPattern = filepath.Join(absBaseDir, pattern)
		}

		matches, err := doublestar.FilepathGlob(fullPattern)
		if err != nil {
			return nil, fmt.Errorf("glob pattern %q: %w", pattern, err)
		}

		for _, match := range matches {
			// Get relative path for consistency
			relPath, err := filepath.Rel(absBaseDir, match)
			if err != nil {
				relPath = match
			}

			// Skip if ignored by gitignore
			if useGitignore && isIgnored(relPath, gitignorePatterns) {
				continue
			}

			// Skip directories
			info, err := os.Stat(match)
			if err != nil {
				continue
			}
			if info.IsDir() {
				continue
			}

			// Only include markdown files
			ext := filepath.Ext(match)
			if ext != ".md" && ext != ".markdown" {
				continue
			}

			fileSet[relPath] = struct{}{}
		}
	}

	// Convert to sorted slice
	files := make([]string, 0, len(fileSet))
	for file := range fileSet {
		files = append(files, file)
	}
	sort.Strings(files)

	return files, nil
}

// loadGitignore reads .gitignore patterns from the specified directory.
func loadGitignore(baseDir string) []string {
	gitignorePath := filepath.Join(baseDir, ".gitignore")
	file, err := os.Open(gitignorePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}

	return patterns
}

// isIgnored checks if a path matches any gitignore pattern.
func isIgnored(path string, gitignorePatterns []string) bool {
	if len(gitignorePatterns) == 0 {
		return false
	}

	normalizedPath := filepath.ToSlash(path)

	for _, pattern := range gitignorePatterns {
		// Skip negation patterns
		if strings.HasPrefix(pattern, "!") {
			continue
		}

		normalizedPattern := filepath.ToSlash(pattern)
		normalizedPattern = strings.TrimSuffix(normalizedPattern, "/")

		// Try direct match
		matched, err := doublestar.Match(normalizedPattern, normalizedPath)
		if err == nil && matched {
			return true
		}

		// Try as prefix (for directories)
		if strings.HasPrefix(normalizedPath, normalizedPattern+"/") {
			return true
		}

		// Match against filename
		filename := filepath.Base(normalizedPath)
		matched, err = doublestar.Match(normalizedPattern, filename)
		if err == nil && matched {
			return true
		}

		// Try with **/ prefix
		if !strings.HasPrefix(normalizedPattern, "**/") && !strings.HasPrefix(normalizedPattern, "/") {
			matched, err = doublestar.Match("**/"+normalizedPattern, normalizedPath)
			if err == nil && matched {
				return true
			}
		}
	}

	return false
}

// expandGlobPatterns expands glob patterns from args into a list of files.
func expandGlobPatterns(args []string) ([]string, error) {
	var files []string
	for _, pattern := range args {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			// Try as a literal file path
			if _, err := os.Stat(pattern); err == nil {
				files = append(files, pattern)
			} else {
				fmt.Fprintf(os.Stderr, "Warning: no files match pattern %q\n", pattern)
			}
		} else {
			files = append(files, matches...)
		}
	}
	return files, nil
}

// processFile lints a single file and updates stats.
func processFile(file string, stats *lintStats) {
	// Skip non-markdown files
	ext := filepath.Ext(file)
	if ext != ".md" && ext != ".markdown" {
		return
	}

	stats.totalFiles++

	content, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
		return
	}

	var result *lint.Result
	if lintFix {
		result = lint.Fix(file, string(content))
	} else {
		result = lint.Lint(file, string(content))
	}

	if len(result.Issues) == 0 {
		return
	}

	stats.filesWithIssues++
	stats.totalIssues += len(result.Issues)

	// Count fixable issues
	for _, issue := range result.Issues {
		if issue.Fixable {
			stats.fixableIssues++
		}
	}

	fmt.Printf("\n%s:\n", file)
	for _, issue := range result.Issues {
		printIssue(issue)
		if issue.Severity == lint.SeverityError {
			stats.hasErrors = true
		}
	}

	// Write fixed content if --fix was used
	if lintFix && result.Fixed != result.Content {
		if err := os.WriteFile(file, []byte(result.Fixed), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", file, err)
		} else {
			stats.totalFixed++
			fmt.Printf("  → Fixed %d issue(s)\n", len(result.Issues))
		}
	}
}

// printIssue prints a single lint issue with colors.
func printIssue(issue lint.Issue) {
	severityColor, resetColor := getSeverityColors(issue.Severity)

	location := fmt.Sprintf("line %d", issue.Line)
	if issue.Column > 0 {
		location += fmt.Sprintf(", col %d", issue.Column)
	}

	fmt.Printf("  %s%s%s [%s]: %s\n",
		severityColor, issue.Severity.String(), resetColor,
		location, issue.Message)
}

// getSeverityColors returns ANSI color codes for a severity level.
func getSeverityColors(severity lint.Severity) (color, reset string) {
	if !isTerminal() {
		return "", ""
	}

	switch severity {
	case lint.SeverityError:
		return colorRed, colorReset
	case lint.SeverityWarning:
		return colorYellow, colorReset
	case lint.SeverityInfo:
		return colorBlue, colorReset
	default:
		return "", ""
	}
}

// printSummary prints the linting summary.
func printSummary(stats *lintStats) {
	fmt.Println()
	if stats.totalIssues == 0 {
		fmt.Printf("✓ %d file(s) linted, no issues found\n", stats.totalFiles)
	} else {
		fmt.Printf("✗ %d file(s) linted, %d issue(s) in %d file(s)\n",
			stats.totalFiles, stats.totalIssues, stats.filesWithIssues)

		if lintFix {
			fmt.Printf("  → Fixed %d file(s)\n", stats.totalFixed)
		} else if stats.fixableIssues > 0 {
			// Show fixable count and suggest fix command
			nonFixable := stats.totalIssues - stats.fixableIssues
			fmt.Printf("ℹ  %d issue(s) can be automatically fixed", stats.fixableIssues)
			if nonFixable > 0 {
				fmt.Printf(", %d cannot", nonFixable)
			}
			fmt.Println()
			fmt.Println()
			fmt.Println("Run 'markata-go lint --fix' to automatically fix fixable issues")
		}
	}
}

// isTerminal returns true if stdout appears to be a terminal.
func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
