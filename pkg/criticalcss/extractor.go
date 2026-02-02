package criticalcss

import (
	"regexp"
	"strings"
)

// defaultCriticalSelectors defines selectors known to appear above the fold in typical layouts.
// These are based on the default markata-go theme structure.
var defaultCriticalSelectors = []string{
	// Reset and base
	"*",
	"html",
	"body",

	// Structure
	"main",
	"header",
	"footer",
	"nav",
	"article",
	"section",
	"aside",
	"div",

	// Typography
	"h1",
	"h2",
	"h3",
	"h4",
	"h5",
	"h6",
	"p",
	"a",
	"span",
	"strong",
	"em",
	"blockquote",

	// Lists
	"ul",
	"ol",
	"li",

	// Media
	"img",
	"figure",
	"figcaption",
	"video",

	// Forms
	"input",
	"button",
	"select",
	"textarea",

	// Tables
	"table",
	"thead",
	"tbody",
	"tr",
	"th",
	"td",

	// Misc
	"hr",
	"pre",
	"code",
	"mark",

	// Site structure classes (from main.css)
	".site-header",
	".site-title",
	".site-nav",
	".site-footer",
	".container",
	".content-width",
	".page-wrapper",
	".main-content",
	".header-controls",

	// Post/content classes
	".post",
	".post-header",
	".post-meta",
	".post-content",
	".post-footer",
	".feed",
	".feed-header",

	// Card components
	".card",
	".card-description",
	".card-meta",

	// Navigation
	".nav",
	".breadcrumbs",
	".tags",
	".tag",

	// Typography classes
	".heading-anchor",
	".toc",
	".toc-title",

	// Common utilities
	".sr-only",
}

// Extractor handles CSS parsing and critical CSS extraction.
type Extractor struct {
	// CriticalSelectors is the list of selectors considered critical
	CriticalSelectors []string

	// ExcludeSelectors is the list of selectors to exclude from critical CSS
	ExcludeSelectors []string

	// MinifyOutput controls whether to minify the output CSS
	MinifyOutput bool
}

// NewExtractor creates a new Extractor with default settings.
func NewExtractor() *Extractor {
	return &Extractor{
		CriticalSelectors: defaultCriticalSelectors,
		ExcludeSelectors:  []string{},
		MinifyOutput:      true,
	}
}

// WithSelectors adds additional selectors to the critical selectors list.
func (e *Extractor) WithSelectors(selectors []string) *Extractor {
	e.CriticalSelectors = append(e.CriticalSelectors, selectors...)
	return e
}

// WithExcludeSelectors sets selectors to exclude from critical CSS.
func (e *Extractor) WithExcludeSelectors(selectors []string) *Extractor {
	e.ExcludeSelectors = selectors
	return e
}

// WithMinify sets whether to minify the output.
func (e *Extractor) WithMinify(minify bool) *Extractor {
	e.MinifyOutput = minify
	return e
}

// Result holds the extracted CSS parts.
type Result struct {
	// Critical is the CSS that should be inlined
	Critical string

	// NonCritical is the CSS that can be async loaded
	NonCritical string

	// CriticalSize is the size of critical CSS in bytes
	CriticalSize int

	// TotalSize is the total size of all CSS in bytes
	TotalSize int
}

// Extract separates CSS into critical and non-critical parts.
// It parses the CSS and identifies rules matching critical selectors.
func (e *Extractor) Extract(css string) (*Result, error) {
	// Build selector sets for fast lookup
	criticalSet := e.buildSelectorSet(e.CriticalSelectors)
	excludeSet := e.buildSelectorSet(e.ExcludeSelectors)

	// Parse CSS into rules
	rules := e.parseRules(css)

	var criticalRules []string
	var nonCriticalRules []string

	for _, rule := range rules {
		if e.isAtRule(rule) {
			// Handle @rules specially
			switch {
			case e.isAlwaysCriticalAtRule(rule):
				criticalRules = append(criticalRules, rule)
			case e.isAtRuleWithSelectors(rule):
				// For @media, @supports, etc., check the selectors inside
				criticalPart, nonCriticalPart := e.splitAtRule(rule, criticalSet, excludeSet)
				if criticalPart != "" {
					criticalRules = append(criticalRules, criticalPart)
				}
				if nonCriticalPart != "" {
					nonCriticalRules = append(nonCriticalRules, nonCriticalPart)
				}
			default:
				// @keyframes, @font-face, etc. - include in critical if referenced
				criticalRules = append(criticalRules, rule)
			}
		} else {
			// Regular rule
			if e.isCriticalRule(rule, criticalSet, excludeSet) {
				criticalRules = append(criticalRules, rule)
			} else {
				nonCriticalRules = append(nonCriticalRules, rule)
			}
		}
	}

	// Build output
	critical := strings.Join(criticalRules, "\n")
	nonCritical := strings.Join(nonCriticalRules, "\n")

	if e.MinifyOutput {
		critical = e.minify(critical)
		nonCritical = e.minify(nonCritical)
	}

	return &Result{
		Critical:     critical,
		NonCritical:  nonCritical,
		CriticalSize: len(critical),
		TotalSize:    len(css),
	}, nil
}

// ExtractMultiple extracts critical CSS from multiple CSS sources.
func (e *Extractor) ExtractMultiple(cssFiles map[string]string) (*Result, error) {
	// Combine all CSS content
	var combined strings.Builder
	for _, content := range cssFiles {
		combined.WriteString(content)
		combined.WriteString("\n")
	}

	return e.Extract(combined.String())
}

// buildSelectorSet creates a map for fast selector lookup.
func (e *Extractor) buildSelectorSet(selectors []string) map[string]bool {
	set := make(map[string]bool, len(selectors))
	for _, s := range selectors {
		// Normalize selector
		normalized := strings.TrimSpace(strings.ToLower(s))
		set[normalized] = true
	}
	return set
}

// parseRules splits CSS into individual rules.
// This is a simplified parser that handles common CSS constructs.
func (e *Extractor) parseRules(css string) []string {
	var rules []string
	var current strings.Builder
	depth := 0
	inString := false
	stringChar := rune(0)

	for i := 0; i < len(css); i++ {
		c := rune(css[i])

		// Handle strings
		if (c == '"' || c == '\'') && !inString {
			inString = true
			stringChar = c
			current.WriteRune(c)
			continue
		}
		if inString && c == stringChar {
			// Check for escape
			if i > 0 && css[i-1] != '\\' {
				inString = false
			}
			current.WriteRune(c)
			continue
		}
		if inString {
			current.WriteRune(c)
			continue
		}

		// Handle braces
		if c == '{' {
			depth++
			current.WriteRune(c)
			continue
		}
		if c == '}' {
			depth--
			current.WriteRune(c)
			if depth == 0 {
				rule := strings.TrimSpace(current.String())
				if rule != "" {
					rules = append(rules, rule)
				}
				current.Reset()
			}
			continue
		}

		current.WriteRune(c)
	}

	// Handle any remaining content (shouldn't happen in valid CSS)
	if remaining := strings.TrimSpace(current.String()); remaining != "" {
		rules = append(rules, remaining)
	}

	return rules
}

// isAtRule checks if a rule starts with @.
func (e *Extractor) isAtRule(rule string) bool {
	return strings.HasPrefix(strings.TrimSpace(rule), "@")
}

// isAlwaysCriticalAtRule checks for @rules that should always be critical.
func (e *Extractor) isAlwaysCriticalAtRule(rule string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(rule))
	return strings.HasPrefix(trimmed, "@charset") ||
		strings.HasPrefix(trimmed, "@import") ||
		strings.HasPrefix(trimmed, "@namespace")
}

// isAtRuleWithSelectors checks for @rules that contain selectors.
func (e *Extractor) isAtRuleWithSelectors(rule string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(rule))
	return strings.HasPrefix(trimmed, "@media") ||
		strings.HasPrefix(trimmed, "@supports") ||
		strings.HasPrefix(trimmed, "@document") ||
		strings.HasPrefix(trimmed, "@layer")
}

// splitAtRule splits an @rule into critical and non-critical parts.
func (e *Extractor) splitAtRule(rule string, criticalSet, excludeSet map[string]bool) (critical, nonCritical string) {
	// Find the opening brace
	braceIdx := strings.Index(rule, "{")
	if braceIdx == -1 {
		return rule, ""
	}

	// Extract the at-rule header (e.g., "@media (max-width: 768px)")
	header := rule[:braceIdx+1]

	// Extract the content between the first { and last }
	lastBrace := strings.LastIndex(rule, "}")
	if lastBrace == -1 || lastBrace <= braceIdx {
		return rule, ""
	}
	content := rule[braceIdx+1 : lastBrace]

	// Parse the inner rules
	innerRules := e.parseRules(content)

	var criticalInner []string
	var nonCriticalInner []string

	for _, innerRule := range innerRules {
		if e.isCriticalRule(innerRule, criticalSet, excludeSet) {
			criticalInner = append(criticalInner, innerRule)
		} else {
			nonCriticalInner = append(nonCriticalInner, innerRule)
		}
	}

	// Rebuild at-rules with their respective contents
	if len(criticalInner) > 0 {
		critical = header + "\n" + strings.Join(criticalInner, "\n") + "\n}"
	}
	if len(nonCriticalInner) > 0 {
		nonCritical = header + "\n" + strings.Join(nonCriticalInner, "\n") + "\n}"
	}

	return critical, nonCritical
}

// isCriticalRule determines if a CSS rule is critical.
func (e *Extractor) isCriticalRule(rule string, criticalSet, excludeSet map[string]bool) bool {
	// Extract selector(s) from the rule
	braceIdx := strings.Index(rule, "{")
	if braceIdx == -1 {
		return false
	}

	selectorPart := strings.TrimSpace(rule[:braceIdx])

	// Check if excluded
	for selector := range excludeSet {
		if e.selectorMatches(selectorPart, selector) {
			return false
		}
	}

	// Check if critical
	for selector := range criticalSet {
		if e.selectorMatches(selectorPart, selector) {
			return true
		}
	}

	return false
}

// selectorMatches checks if a CSS rule's selector matches a critical selector.
func (e *Extractor) selectorMatches(ruleSelector, criticalSelector string) bool {
	// Normalize both
	ruleSelector = strings.ToLower(strings.TrimSpace(ruleSelector))
	criticalSelector = strings.ToLower(strings.TrimSpace(criticalSelector))

	// Handle multiple selectors (comma-separated)
	selectors := strings.Split(ruleSelector, ",")
	for _, sel := range selectors {
		sel = strings.TrimSpace(sel)

		// Direct match
		if sel == criticalSelector {
			return true
		}

		// Check if the rule selector starts with or contains the critical selector
		// This handles cases like "body.dark" matching "body" or ".card:hover" matching ".card"
		if strings.HasPrefix(sel, criticalSelector) {
			// Make sure it's not just a partial class match (e.g., ".card" shouldn't match ".cards")
			if len(sel) == len(criticalSelector) {
				return true
			}
			nextChar := sel[len(criticalSelector)]
			// Valid separator characters after a selector
			if nextChar == ' ' || nextChar == '.' || nextChar == '#' || nextChar == ':' ||
				nextChar == '[' || nextChar == '>' || nextChar == '+' || nextChar == '~' {
				return true
			}
		}

		// Check if any part of a compound selector matches
		parts := e.splitSelector(sel)
		for _, part := range parts {
			if part == criticalSelector {
				return true
			}
			// Handle pseudo-classes and pseudo-elements
			basePart := strings.Split(part, ":")[0]
			basePart = strings.Split(basePart, "[")[0]
			if basePart == criticalSelector {
				return true
			}
		}
	}

	return false
}

// splitSelector splits a compound selector into parts.
func (e *Extractor) splitSelector(selector string) []string {
	// Split on spaces and combinators
	re := regexp.MustCompile(`[\s>+~]+`)
	parts := re.Split(selector, -1)

	var result []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// minify performs basic CSS minification.
func (e *Extractor) minify(css string) string {
	// Remove comments
	css = e.removeComments(css)

	// Remove excessive whitespace
	css = regexp.MustCompile(`\s+`).ReplaceAllString(css, " ")

	// Remove whitespace around certain characters
	css = regexp.MustCompile(`\s*{\s*`).ReplaceAllString(css, "{")
	css = regexp.MustCompile(`\s*}\s*`).ReplaceAllString(css, "}")
	css = regexp.MustCompile(`\s*;\s*`).ReplaceAllString(css, ";")
	css = regexp.MustCompile(`\s*:\s*`).ReplaceAllString(css, ":")
	css = regexp.MustCompile(`\s*,\s*`).ReplaceAllString(css, ",")

	// Remove trailing semicolons before closing braces
	css = strings.ReplaceAll(css, ";}", "}")

	return strings.TrimSpace(css)
}

// removeComments removes CSS comments.
func (e *Extractor) removeComments(css string) string {
	var result strings.Builder
	i := 0
	for i < len(css) {
		// Check for comment start
		if i+1 < len(css) && css[i] == '/' && css[i+1] == '*' {
			// Find comment end
			end := strings.Index(css[i+2:], "*/")
			if end != -1 {
				i = i + 2 + end + 2
				continue
			}
			// Unclosed comment, skip to end
			break
		}
		result.WriteByte(css[i])
		i++
	}
	return result.String()
}
