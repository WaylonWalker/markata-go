package diagnostics

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Severity indicates the severity of a diagnostic issue.
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

// Range represents a position range in a document.
type Range struct {
	StartLine int // 0-based line number
	StartCol  int // 0-based column (character offset)
	EndLine   int // 0-based line number
	EndCol    int // 0-based column
}

// Issue represents a diagnostic issue found in a file.
type Issue struct {
	File     string   // File path
	Range    Range    // Position in the file
	Code     string   // Issue code (e.g., "duplicate-key", "broken-wikilink")
	Severity Severity // Severity level
	Message  string   // Human-readable message
	Fixable  bool     // Whether this issue can be automatically fixed
}

// Resolver provides lookup functionality for wikilinks and mentions.
// This interface allows the diagnostics package to check if references exist
// without directly depending on the LSP index or any specific implementation.
type Resolver interface {
	// ResolveSlug returns true if a post with the given slug exists.
	ResolveSlug(slug string) bool
	// ResolveHandle returns true if a mention handle exists in the blogroll.
	ResolveHandle(handle string) bool
}

// Check runs all diagnostic checks on the content and returns any issues found.
// The resolver is optional; if nil, wikilink and mention checks are skipped.
func Check(filePath, content string, resolver Resolver) []Issue {
	var issues []Issue

	// Extract frontmatter for YAML-specific checks
	frontmatter, body, hasFrontmatter := extractFrontmatter(content)

	if hasFrontmatter {
		issues = append(issues, checkDuplicateKeys(filePath, frontmatter)...)
		issues = append(issues, checkDateFormats(filePath, frontmatter)...)
	}

	// Body checks
	issues = append(issues, checkImageLinks(filePath, body, hasFrontmatter, frontmatter)...)
	issues = append(issues, checkProtocollessURLs(filePath, content)...)
	issues = append(issues, checkH1Headings(filePath, body, hasFrontmatter, frontmatter)...)
	issues = append(issues, checkAdmonitionFencedCode(filePath, body, hasFrontmatter, frontmatter)...)

	// Reference checks (require resolver)
	if resolver != nil {
		issues = append(issues, checkWikilinks(filePath, body, hasFrontmatter, frontmatter, resolver)...)
		issues = append(issues, checkMentions(filePath, body, hasFrontmatter, frontmatter, resolver)...)
	}

	return issues
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
					File: filePath,
					Range: Range{
						StartLine: lineNum - 1, // 0-based
						StartCol:  0,
						EndLine:   lineNum - 1,
						EndCol:    len(line),
					},
					Code:     "duplicate-key",
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
								File: filePath,
								Range: Range{
									StartLine: lineNum - 1,
									StartCol:  0,
									EndLine:   lineNum - 1,
									EndCol:    len(line),
								},
								Code:     "invalid-date",
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
					File: filePath,
					Range: Range{
						StartLine: lineNum + lineOffset - 1,
						StartCol:  match[0],
						EndLine:   lineNum + lineOffset - 1,
						EndCol:    match[1],
					},
					Code:     "missing-alt-text",
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
					File: filePath,
					Range: Range{
						StartLine: lineNum - 1,
						StartCol:  match[0] + 1, // +1 to skip the non-colon char
						EndLine:   lineNum - 1,
						EndCol:    match[1],
					},
					Code:     "protocol-less-url",
					Severity: SeverityWarning,
					Message:  "protocol-less URL found (should use https://)",
					Fixable:  true,
				})
			}
		}
	}

	return issues
}

// checkH1Headings finds H1 headings in markdown content.
func checkH1Headings(filePath, body string, hasFrontmatter bool, frontmatter string) []Issue {
	var issues []Issue

	lineOffset := 0
	if hasFrontmatter {
		lineOffset = strings.Count(frontmatter, "\n") + 1
		body = strings.TrimPrefix(body, "\n")
	}

	inCodeBlock := false
	scanner := bufio.NewScanner(strings.NewReader(body))
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "```") || strings.HasPrefix(trimmedLine, "~~~") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		if strings.HasPrefix(line, "# ") || line == "#" {
			issues = append(issues, Issue{
				File: filePath,
				Range: Range{
					StartLine: lineNum + lineOffset - 1,
					StartCol:  0,
					EndLine:   lineNum + lineOffset - 1,
					EndCol:    len(line),
				},
				Code:     "h1-in-content",
				Severity: SeverityWarning,
				Message:  "H1 heading found in content. Templates already add an H1 from frontmatter title. Use H2 (##) or deeper instead.",
				Fixable:  false,
			})
		}
	}

	return issues
}

// checkAdmonitionFencedCode detects fenced code blocks inside admonitions
// that don't have a blank line before them.
func checkAdmonitionFencedCode(filePath, body string, hasFrontmatter bool, frontmatter string) []Issue {
	var issues []Issue

	lineOffset := 0
	if hasFrontmatter {
		lineOffset = strings.Count(frontmatter, "\n") + 1
		body = strings.TrimPrefix(body, "\n")
	}

	lines := strings.Split(body, "\n")

	admonitionRegex := regexp.MustCompile(`^(\s*)!!!\s+\w+`)
	fencedCodeRegex := regexp.MustCompile(`^\s*` + "```")

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		lineNum := i + 1 + lineOffset

		if match := admonitionRegex.FindStringSubmatch(line); match != nil {
			admonitionIndent := len(match[1])

			if i+1 < len(lines) {
				nextLine := lines[i+1]
				trimmedNext := strings.TrimLeft(nextLine, " \t")
				nextIndent := len(nextLine) - len(trimmedNext)

				if nextIndent > admonitionIndent && fencedCodeRegex.MatchString(nextLine) {
					issues = append(issues, Issue{
						File: filePath,
						Range: Range{
							StartLine: lineNum - 1,
							StartCol:  0,
							EndLine:   lineNum - 1,
							EndCol:    len(line),
						},
						Code:     "admonition-fenced-code",
						Severity: SeverityWarning,
						Message:  "fenced code block immediately follows admonition without blank line - this may not render correctly due to goldmark limitation",
						Fixable:  true,
					})
				}
			}
		}
	}

	return issues
}

// wikilinkRegex matches [[slug]] and [[slug|display text]] patterns.
var wikilinkRegex = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

// checkWikilinks finds broken wikilinks in content.
func checkWikilinks(filePath, body string, hasFrontmatter bool, frontmatter string, resolver Resolver) []Issue {
	var issues []Issue

	lineOffset := 0
	if hasFrontmatter {
		lineOffset = strings.Count(frontmatter, "\n") + 2 // +2 for both --- lines
	}

	lines := strings.Split(body, "\n")
	inCodeBlock := false
	codeBlockPattern := regexp.MustCompile("^```|^~~~")

	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)
		if codeBlockPattern.MatchString(trimmed) {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		matches := wikilinkRegex.FindAllStringSubmatchIndex(line, -1)
		for _, match := range matches {
			if len(match) < 4 {
				continue
			}

			slugStart := match[2]
			slugEnd := match[3]
			slug := strings.TrimSpace(line[slugStart:slugEnd])

			if !resolver.ResolveSlug(slug) {
				issues = append(issues, Issue{
					File: filePath,
					Range: Range{
						StartLine: lineNum + lineOffset,
						StartCol:  match[0],
						EndLine:   lineNum + lineOffset,
						EndCol:    match[1],
					},
					Code:     "broken-wikilink",
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("broken wikilink: target post %q not found", slug),
					Fixable:  false,
				})
			}
		}
	}

	return issues
}

// mentionRegex matches @handle patterns.
var mentionRegex = regexp.MustCompile(`@([a-zA-Z][a-zA-Z0-9_.-]*)`)

// checkMentions finds unknown mentions in content.
func checkMentions(filePath, body string, hasFrontmatter bool, frontmatter string, resolver Resolver) []Issue {
	var issues []Issue

	lineOffset := 0
	if hasFrontmatter {
		lineOffset = strings.Count(frontmatter, "\n") + 2
	}

	lines := strings.Split(body, "\n")
	inCodeBlock := false
	codeBlockPattern := regexp.MustCompile("^```|^~~~")

	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)
		if codeBlockPattern.MatchString(trimmed) {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		matches := mentionRegex.FindAllStringSubmatchIndex(line, -1)
		for _, match := range matches {
			if len(match) < 4 {
				continue
			}

			start := match[0]
			// Validate that @ is at a valid boundary
			if start > 0 {
				prevChar := line[start-1]
				if (prevChar >= 'a' && prevChar <= 'z') ||
					(prevChar >= 'A' && prevChar <= 'Z') ||
					(prevChar >= '0' && prevChar <= '9') || prevChar == '_' || prevChar == '@' {
					continue // Skip email addresses and @@mentions
				}
			}

			handleStart := match[2]
			handleEnd := match[3]
			handle := strings.ToLower(line[handleStart:handleEnd])

			if !resolver.ResolveHandle(handle) {
				issues = append(issues, Issue{
					File: filePath,
					Range: Range{
						StartLine: lineNum + lineOffset,
						StartCol:  match[0],
						EndLine:   lineNum + lineOffset,
						EndCol:    match[1],
					},
					Code:     "unknown-mention",
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("unknown mention: @%s not found in blogroll", handle),
					Fixable:  false,
				})
			}
		}
	}

	return issues
}
