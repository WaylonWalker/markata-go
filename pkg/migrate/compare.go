package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// CompareResult contains the results of comparing two site directories.
type CompareResult struct {
	// OldDir is the path to the old site directory
	OldDir string

	// NewDir is the path to the new site directory
	NewDir string

	// OldFiles is the list of files in the old directory
	OldFiles []string

	// NewFiles is the list of files in the new directory
	NewFiles []string

	// MissingInNew is the list of files in old but not in new
	MissingInNew []string

	// NewOnly is the list of files in new but not in old
	NewOnly []string

	// Common is the list of files present in both directories
	Common []string
}

// CompareOptions configures the directory comparison.
type CompareOptions struct {
	// Extensions to compare (e.g., []string{".html"})
	// If empty, all files are compared
	Extensions []string

	// IgnorePatterns are glob patterns to ignore
	IgnorePatterns []string
}

// DefaultCompareOptions returns sensible defaults for comparing SSG output.
func DefaultCompareOptions() CompareOptions {
	return CompareOptions{
		Extensions: []string{".html"},
		IgnorePatterns: []string{
			"**/assets/**",
			"**/_pagefind/**",
			"**/static/**",
		},
	}
}

// Compare compares two directories and returns the differences.
func Compare(oldDir, newDir string, opts CompareOptions) (*CompareResult, error) {
	result := &CompareResult{
		OldDir: oldDir,
		NewDir: newDir,
	}

	// Validate directories exist
	if err := validateDirectory(oldDir); err != nil {
		return nil, fmt.Errorf("old directory: %w", err)
	}
	if err := validateDirectory(newDir); err != nil {
		return nil, fmt.Errorf("new directory: %w", err)
	}

	// List files in both directories
	oldFiles, err := listFiles(oldDir, opts)
	if err != nil {
		return nil, fmt.Errorf("listing old directory: %w", err)
	}
	result.OldFiles = oldFiles

	newFiles, err := listFiles(newDir, opts)
	if err != nil {
		return nil, fmt.Errorf("listing new directory: %w", err)
	}
	result.NewFiles = newFiles

	// Create sets for comparison
	oldSet := make(map[string]struct{}, len(oldFiles))
	for _, f := range oldFiles {
		oldSet[f] = struct{}{}
	}

	newSet := make(map[string]struct{}, len(newFiles))
	for _, f := range newFiles {
		newSet[f] = struct{}{}
	}

	// Find missing in new (present in old, not in new)
	for _, f := range oldFiles {
		if _, found := newSet[f]; !found {
			result.MissingInNew = append(result.MissingInNew, f)
		} else {
			result.Common = append(result.Common, f)
		}
	}

	// Find new only (present in new, not in old)
	for _, f := range newFiles {
		if _, found := oldSet[f]; !found {
			result.NewOnly = append(result.NewOnly, f)
		}
	}

	// Sort results for consistent output
	sort.Strings(result.MissingInNew)
	sort.Strings(result.NewOnly)
	sort.Strings(result.Common)

	return result, nil
}

// validateDirectory checks if a path exists and is a directory.
func validateDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory not found: %s", path)
		}
		return fmt.Errorf("cannot access: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}
	return nil
}

// listFiles returns all files in a directory matching the options.
// Paths are returned relative to the directory root, with forward slashes.
func listFiles(dir string, opts CompareOptions) ([]string, error) {
	var files []string

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(absDir, path)
		if err != nil {
			return err
		}

		// Normalize to forward slashes for consistency
		relPath = filepath.ToSlash(relPath)

		// Add leading slash for URL-like paths
		relPath = "/" + relPath

		// Check extension filter
		if len(opts.Extensions) > 0 {
			ext := strings.ToLower(filepath.Ext(path))
			matched := false
			for _, allowedExt := range opts.Extensions {
				if strings.EqualFold(ext, allowedExt) {
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
		}

		// Check ignore patterns
		for _, pattern := range opts.IgnorePatterns {
			// Match against the relative path without leading slash
			pathToMatch := strings.TrimPrefix(relPath, "/")
			matched, err := doublestar.Match(pattern, pathToMatch)
			if err != nil {
				continue // Skip invalid patterns
			}
			if matched {
				return nil
			}
		}

		files = append(files, relPath)
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

// HasDifferences returns true if there are any differences between directories.
func (r *CompareResult) HasDifferences() bool {
	return len(r.MissingInNew) > 0 || len(r.NewOnly) > 0
}

// ExitCode returns the appropriate exit code for the comparison result.
// 0 = directories match, 1 = differences found.
func (r *CompareResult) ExitCode() int {
	if r.HasDifferences() {
		return 1
	}
	return 0
}

// Report generates a human-readable comparison report.
func (r *CompareResult) Report() string {
	var sb strings.Builder

	sb.WriteString("Migration Comparison Report\n")
	sb.WriteString("===========================\n\n")

	fmt.Fprintf(&sb, "Old site: %s/ (%d files)\n", r.OldDir, len(r.OldFiles))
	fmt.Fprintf(&sb, "New site: %s/ (%d files)\n\n", r.NewDir, len(r.NewFiles))

	if !r.HasDifferences() {
		sb.WriteString("No differences found - sites match!\n")
		return sb.String()
	}

	// Missing in new
	if len(r.MissingInNew) > 0 {
		fmt.Fprintf(&sb, "Missing in new site (%d files):\n", len(r.MissingInNew))
		for _, f := range r.MissingInNew {
			fmt.Fprintf(&sb, "  - %s\n", f)
		}
		sb.WriteString("\n")
	}

	// New files
	if len(r.NewOnly) > 0 {
		fmt.Fprintf(&sb, "New files not in old (%d files):\n", len(r.NewOnly))
		for _, f := range r.NewOnly {
			fmt.Fprintf(&sb, "  + %s\n", f)
		}
		sb.WriteString("\n")
	}

	// Summary
	sb.WriteString("Summary:\n")
	fmt.Fprintf(&sb, "  Common files:     %d\n", len(r.Common))
	fmt.Fprintf(&sb, "  Missing in new:   %d\n", len(r.MissingInNew))
	fmt.Fprintf(&sb, "  New files:        %d\n", len(r.NewOnly))

	return sb.String()
}

// JSONReport returns a JSON-friendly structure for programmatic use.
func (r *CompareResult) JSONReport() map[string]interface{} {
	return map[string]interface{}{
		"old_dir":         r.OldDir,
		"new_dir":         r.NewDir,
		"old_file_count":  len(r.OldFiles),
		"new_file_count":  len(r.NewFiles),
		"common_count":    len(r.Common),
		"missing_count":   len(r.MissingInNew),
		"new_count":       len(r.NewOnly),
		"has_differences": r.HasDifferences(),
		"missing_in_new":  r.MissingInNew,
		"new_only":        r.NewOnly,
		"common":          r.Common,
		"exit_code":       r.ExitCode(),
	}
}
