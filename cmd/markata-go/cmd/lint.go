package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/WaylonWalker/markata-go/pkg/lint"
	"github.com/spf13/cobra"
)

var (
	// lintFix automatically fixes issues when possible.
	lintFix bool
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

Use --fix to automatically fix detected issues.

Example usage:
  markata-go lint posts/**/*.md          # Lint all posts
  markata-go lint posts/**/*.md --fix    # Lint and auto-fix issues
  markata-go lint pages/about.md         # Lint a specific file`,
	Args: cobra.MinimumNArgs(1),
	RunE: runLintCommand,
}

func init() {
	rootCmd.AddCommand(lintCmd)

	lintCmd.Flags().BoolVar(&lintFix, "fix", false, "automatically fix issues")
}

// lintStats tracks linting statistics.
type lintStats struct {
	totalFiles      int
	totalIssues     int
	totalFixed      int
	filesWithIssues int
	hasErrors       bool
}

func runLintCommand(_ *cobra.Command, args []string) error {
	files, err := expandGlobPatterns(args)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no files to lint")
	}

	// Sort files for consistent output
	sort.Strings(files)

	stats := &lintStats{}
	for _, file := range files {
		processFile(file, stats)
	}

	printSummary(stats)

	// Exit with error code if there are errors (not just warnings)
	if stats.hasErrors && !lintFix {
		return fmt.Errorf("linting found errors")
	}

	return nil
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
