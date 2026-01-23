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

func runLintCommand(_ *cobra.Command, args []string) error {
	// Expand glob patterns
	var files []string
	for _, pattern := range args {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("invalid pattern %q: %w", pattern, err)
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

	if len(files) == 0 {
		return fmt.Errorf("no files to lint")
	}

	// Sort files for consistent output
	sort.Strings(files)

	var (
		totalFiles      int
		totalIssues     int
		totalFixed      int
		filesWithIssues int
		hasErrors       bool
	)

	for _, file := range files {
		// Skip non-markdown files
		ext := filepath.Ext(file)
		if ext != ".md" && ext != ".markdown" {
			continue
		}

		totalFiles++

		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
			continue
		}

		var result *lint.Result
		if lintFix {
			result = lint.Fix(file, string(content))
		} else {
			result = lint.Lint(file, string(content))
		}

		if len(result.Issues) > 0 {
			filesWithIssues++
			totalIssues += len(result.Issues)

			fmt.Printf("\n%s:\n", file)
			for _, issue := range result.Issues {
				severityColor := ""
				resetColor := ""

				// Use ANSI colors if terminal supports it
				if isTerminal() {
					switch issue.Severity {
					case lint.SeverityError:
						severityColor = "\033[31m" // Red
					case lint.SeverityWarning:
						severityColor = "\033[33m" // Yellow
					case lint.SeverityInfo:
						severityColor = "\033[34m" // Blue
					}
					resetColor = "\033[0m"
				}

				location := fmt.Sprintf("line %d", issue.Line)
				if issue.Column > 0 {
					location += fmt.Sprintf(", col %d", issue.Column)
				}

				fmt.Printf("  %s%s%s [%s]: %s\n",
					severityColor, issue.Severity.String(), resetColor,
					location, issue.Message)

				if issue.Severity == lint.SeverityError {
					hasErrors = true
				}
			}

			// Write fixed content if --fix was used
			if lintFix && result.Fixed != result.Content {
				if err := os.WriteFile(file, []byte(result.Fixed), 0644); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", file, err)
				} else {
					totalFixed++
					fmt.Printf("  → Fixed %d issue(s)\n", len(result.Issues))
				}
			}
		}
	}

	// Print summary
	fmt.Println()
	if totalIssues == 0 {
		fmt.Printf("✓ %d file(s) linted, no issues found\n", totalFiles)
	} else {
		fmt.Printf("✗ %d file(s) linted, %d issue(s) in %d file(s)\n",
			totalFiles, totalIssues, filesWithIssues)
		if lintFix {
			fmt.Printf("  → Fixed %d file(s)\n", totalFixed)
		}
	}

	// Exit with error code if there are errors (not just warnings)
	if hasErrors && !lintFix {
		return fmt.Errorf("linting found errors")
	}

	return nil
}

// isTerminal returns true if stdout appears to be a terminal.
func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
