// Package lint provides markdown linting functionality for markata-go.
// It detects common issues in markdown files that can cause build failures.
//
// # Supported Checks
//
// The linter detects the following issues:
//   - Duplicate YAML keys in frontmatter
//   - Invalid date formats (non-ISO 8601)
//   - Malformed image links (missing alt text)
//   - Protocol-less URLs (//example.com instead of https://example.com)
//
// All issues can be auto-fixed using the Fix function.
package lint

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Issue represents a linting issue found in a file.
type Issue struct {
	File       string   // File path
	Line       int      // Line number (1-indexed)
	Column     int      // Column number (1-indexed, 0 if not applicable)
	Type       string   // Issue type (e.g., "duplicate-key", "invalid-date")
	Severity   Severity // Severity level
	Message    string   // Human-readable message
	Fixable    bool     // Whether this issue can be automatically fixed
	FixApplied bool     // Whether fix was applied
}

// Severity indicates the severity of a linting issue.
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
	SeverityInfo
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	default:
		return "unknown"
	}
}

// Result contains the linting results for a file.
type Result struct {
	File    string  // File path
	Issues  []Issue // Issues found
	Content string  // Original content
	Fixed   string  // Fixed content (same as Content if no fixes)
}

// HasErrors returns true if any issues are errors.
func (r *Result) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}

// FixableCount returns the number of fixable issues.
func (r *Result) FixableCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.Fixable {
			count++
		}
	}
	return count
}

// Lint analyzes content and returns any issues found.
func Lint(filePath, content string) *Result {
	result := &Result{
		File:    filePath,
		Content: content,
		Fixed:   content,
	}

	// Extract frontmatter for YAML-specific checks
	frontmatter, body, hasFrontmatter := extractFrontmatter(content)

	if hasFrontmatter {
		// Check for duplicate YAML keys
		result.Issues = append(result.Issues, checkDuplicateKeys(filePath, frontmatter)...)

		// Check for invalid date formats
		result.Issues = append(result.Issues, checkDateFormats(filePath, frontmatter)...)
	}

	// Check for malformed image links (in body)
	result.Issues = append(result.Issues, checkImageLinks(filePath, body, hasFrontmatter, frontmatter)...)

	// Check for protocol-less URLs (in entire content)
	result.Issues = append(result.Issues, checkProtocollessURLs(filePath, content)...)

	return result
}

// Fix applies automatic fixes to the content and returns the fixed content.
func Fix(filePath, content string) *Result {
	result := Lint(filePath, content)
	fixed := content

	// Apply fixes in order
	fixed = fixDuplicateKeys(fixed)
	fixed = fixDateFormats(fixed)
	fixed = fixImageLinks(fixed)
	fixed = fixProtocollessURLs(fixed)

	result.Fixed = fixed

	// Mark issues as fixed
	for i := range result.Issues {
		result.Issues[i].FixApplied = true
	}

	return result
}

// extractFrontmatter extracts frontmatter from content.
func extractFrontmatter(content string) (frontmatter, body string, hasFrontmatter bool) {
	if !strings.HasPrefix(content, "---") {
		return "", content, false
	}

	parts := strings.SplitN(content[3:], "---", 2)
	if len(parts) < 2 {
		return "", content, false
	}

	return parts[0], parts[1], true
}

// checkDuplicateKeys finds duplicate YAML keys in frontmatter.
func checkDuplicateKeys(filePath, frontmatter string) []Issue {
	var issues []Issue
	seen := make(map[string]int) // key -> first line number
	scanner := bufio.NewScanner(strings.NewReader(frontmatter))
	lineNum := 1 // Start at 1 (after the opening ---)

	// Regex to match top-level YAML keys (not indented)
	keyRegex := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)\s*:`)

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		if match := keyRegex.FindStringSubmatch(line); match != nil {
			key := match[1]
			if firstLine, exists := seen[key]; exists {
				issues = append(issues, Issue{
					File:     filePath,
					Line:     lineNum,
					Type:     "duplicate-key",
					Severity: SeverityError,
					Message:  fmt.Sprintf("duplicate key '%s' (first occurrence at line %d)", key, firstLine),
					Fixable:  true,
				})
			} else {
				seen[key] = lineNum
			}
		}
	}

	return issues
}

// checkDateFormats validates date formats in frontmatter.
func checkDateFormats(filePath, frontmatter string) []Issue {
	var issues []Issue
	scanner := bufio.NewScanner(strings.NewReader(frontmatter))
	lineNum := 1

	// Regex to match date-like fields
	dateKeyRegex := regexp.MustCompile(`^(date|published_date|created|modified|updated)\s*:\s*(.+)$`)

	// Common invalid date patterns
	invalidDatePatterns := []struct {
		pattern *regexp.Regexp
		desc    string
	}{
		{regexp.MustCompile(`\d{4}-\d{1,2}-\d{1,2}T\d{2}:\d{2}:\d{2}`), "single-digit month/day"},
		{regexp.MustCompile(`\d{4}/\d{2}/\d{2}`), "slash separator"},
	}

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		if match := dateKeyRegex.FindStringSubmatch(line); match != nil {
			key := match[1]
			value := strings.TrimSpace(match[2])
			value = strings.Trim(value, "\"'")

			// Try to parse as valid ISO 8601
			_, err := time.Parse(time.RFC3339, value)
			if err != nil {
				// Try other valid formats
				_, err2 := time.Parse("2006-01-02", value)
				if err2 != nil {
					// Check for specific invalid patterns
					for _, p := range invalidDatePatterns {
						if p.pattern.MatchString(value) {
							issues = append(issues, Issue{
								File:     filePath,
								Line:     lineNum,
								Type:     "invalid-date",
								Severity: SeverityWarning,
								Message:  fmt.Sprintf("invalid date format for '%s': %s (%s)", key, value, p.desc),
								Fixable:  true,
							})
							break
						}
					}
				}
			}
		}
	}

	return issues
}

// checkImageLinks finds malformed image links.
func checkImageLinks(filePath, body string, hasFrontmatter bool, frontmatter string) []Issue {
	var issues []Issue

	// Regex for image links without alt text: ![](url)
	noAltRegex := regexp.MustCompile(`!\[\]\(([^)]+)\)`)

	// Calculate line offset for body
	lineOffset := 0
	if hasFrontmatter {
		lineOffset = strings.Count(frontmatter, "\n") + 2 // +2 for both --- lines
	}

	scanner := bufio.NewScanner(strings.NewReader(body))
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		if matches := noAltRegex.FindAllStringSubmatchIndex(line, -1); matches != nil {
			for _, match := range matches {
				issues = append(issues, Issue{
					File:     filePath,
					Line:     lineNum + lineOffset,
					Column:   match[0] + 1,
					Type:     "missing-alt-text",
					Severity: SeverityWarning,
					Message:  "image link missing alt text",
					Fixable:  true,
				})
			}
		}
	}

	return issues
}

// checkProtocollessURLs finds protocol-less URLs.
func checkProtocollessURLs(filePath, content string) []Issue {
	var issues []Issue

	// Regex for protocol-less URLs: //example.com
	protocollessRegex := regexp.MustCompile(`[^:]//[a-zA-Z0-9][a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		if matches := protocollessRegex.FindAllStringIndex(line, -1); matches != nil {
			for _, match := range matches {
				issues = append(issues, Issue{
					File:     filePath,
					Line:     lineNum,
					Column:   match[0] + 2, // +2 to skip the non-colon char and point to //
					Type:     "protocol-less-url",
					Severity: SeverityWarning,
					Message:  "protocol-less URL found (should use https://)",
					Fixable:  true,
				})
			}
		}
	}

	return issues
}

// fixDuplicateKeys removes duplicate YAML keys, keeping the last occurrence.
func fixDuplicateKeys(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}

	parts := strings.SplitN(content[3:], "---", 2)
	if len(parts) < 2 {
		return content
	}

	frontmatter := parts[0]
	body := parts[1]

	// Track keys and their lines
	type keyLine struct {
		key  string
		line string
	}
	var lines []keyLine
	seen := make(map[string]int) // key -> index in lines

	scanner := bufio.NewScanner(strings.NewReader(frontmatter))
	keyRegex := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)\s*:`)
	var currentKey string
	var currentLines []string

	flushCurrent := func() {
		if currentKey != "" && len(currentLines) > 0 {
			combined := strings.Join(currentLines, "\n")
			if idx, exists := seen[currentKey]; exists {
				// Replace previous occurrence
				lines[idx] = keyLine{key: currentKey, line: combined}
			} else {
				seen[currentKey] = len(lines)
				lines = append(lines, keyLine{key: currentKey, line: combined})
			}
		}
		currentKey = ""
		currentLines = nil
	}

	for scanner.Scan() {
		line := scanner.Text()

		if match := keyRegex.FindStringSubmatch(line); match != nil {
			// New key found, flush previous
			flushCurrent()
			currentKey = match[1]
			currentLines = []string{line}
		} else if currentKey != "" && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") || line == "") {
			// Continuation of current key (indented or empty)
			currentLines = append(currentLines, line)
		} else if currentKey != "" {
			// Non-indented line that's not a key - might be a YAML array/list item
			currentLines = append(currentLines, line)
		}
	}
	flushCurrent()

	// Rebuild frontmatter
	fixedLines := make([]string, 0, len(lines))
	for _, kl := range lines {
		fixedLines = append(fixedLines, kl.line)
	}

	return "---\n" + strings.Join(fixedLines, "\n") + "\n---" + body
}

// fixDateFormats normalizes date formats to ISO 8601 using DateTimeFixer.
func fixDateFormats(content string) string {
	fixer := NewDateTimeFixer(DefaultDateTimeFixerConfig())
	fixed, _ := fixer.FixDateInContent(content)
	return fixed
}

// fixImageLinks adds placeholder alt text to images without it.
func fixImageLinks(content string) string {
	// Replace ![](url) with ![image](url)
	noAltRegex := regexp.MustCompile(`!\[\]\(([^)]+)\)`)
	return noAltRegex.ReplaceAllString(content, "![image]($1)")
}

// fixProtocollessURLs adds https:// to protocol-less URLs.
func fixProtocollessURLs(content string) string {
	// Replace (//example.com with (https://example.com
	// Be careful not to replace // in code or comments
	protocollessRegex := regexp.MustCompile(`(\(|"|\s)//([a-zA-Z0-9][a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)
	return protocollessRegex.ReplaceAllString(content, "${1}https://$2")
}
