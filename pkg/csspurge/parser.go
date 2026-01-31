package csspurge

import (
	"regexp"
	"strings"
)

// CSSRule represents a CSS rule with its selector and content.
type CSSRule struct {
	// Selector is the CSS selector (e.g., ".class", "#id", "div")
	Selector string
	// Content is the full rule including braces (e.g., ".class { color: red; }")
	Content string
	// IsAtRule indicates if this is an @-rule (@media, @keyframes, etc.)
	IsAtRule bool
	// AtRuleType is the type of @-rule (e.g., "media", "keyframes", "font-face")
	AtRuleType string
	// NestedRules contains rules nested inside @-rules like @media
	NestedRules []CSSRule
}

// Regular expressions for CSS parsing
var (
	// Match @-rules: @media, @keyframes, @font-face, @import, @supports, etc.
	atRuleRegex = regexp.MustCompile(`@([\w-]+)\s*([^{;]*?)(\{|;)`)

	// Match a CSS rule: selector { properties }
	// This is a simplified regex - handles most common cases
	ruleRegex = regexp.MustCompile(`([^{}@]+?)\s*\{([^{}]*)\}`)

	// Match selectors within a selector list (comma-separated)
	selectorSplitRegex = regexp.MustCompile(`\s*,\s*`)

	// Extract class names from selector: .class-name
	classRegex = regexp.MustCompile(`\.(-?[_a-zA-Z][_a-zA-Z0-9-]*)`)

	// Extract ID from selector: #id-name
	idRegex = regexp.MustCompile(`#(-?[_a-zA-Z][_a-zA-Z0-9-]*)`)

	// Extract element names from selector (start of selector or after space/combinator)
	elementRegex = regexp.MustCompile(`(?:^|[\s>+~])([a-zA-Z][a-zA-Z0-9]*)`)

	// Extract attribute selectors: [attr], [attr=value], etc.
	attrRegex = regexp.MustCompile(`\[([a-zA-Z][a-zA-Z0-9-]*)`)
)

// ParseCSS parses CSS content into a slice of CSSRule structs.
// It handles nested @-rules like @media queries.
func ParseCSS(content string) []CSSRule {
	var rules []CSSRule

	// Remove CSS comments
	content = removeComments(content)

	// Process the CSS content
	rules = parseRules(content, rules)

	return rules
}

// removeComments strips CSS comments from content.
func removeComments(css string) string {
	var result strings.Builder
	i := 0
	for i < len(css) {
		// Check for comment start
		if i+1 < len(css) && css[i] == '/' && css[i+1] == '*' {
			// Find comment end
			end := strings.Index(css[i+2:], "*/")
			if end == -1 {
				// Unclosed comment - skip rest of content
				break
			}
			i += end + 4 // Skip past */
			continue
		}
		result.WriteByte(css[i])
		i++
	}
	return result.String()
}

// parseRules parses CSS rules from content, handling @-rules recursively.
func parseRules(content string, rules []CSSRule) []CSSRule {
	pos := 0
	for pos < len(content) {
		// Skip whitespace
		for pos < len(content) && isWhitespace(content[pos]) {
			pos++
		}
		if pos >= len(content) {
			break
		}

		// Check for @-rule
		if content[pos] == '@' {
			rule, newPos := parseAtRule(content, pos)
			if rule != nil {
				rules = append(rules, *rule)
			}
			if newPos > pos {
				pos = newPos
			} else {
				pos++
			}
			continue
		}

		// Try to parse a regular rule
		rule, newPos := parseRegularRule(content, pos)
		if rule != nil {
			rules = append(rules, *rule)
		}
		if newPos > pos {
			pos = newPos
		} else {
			pos++
		}
	}

	return rules
}

// parseAtRule parses an @-rule starting at pos.
func parseAtRule(content string, pos int) (*CSSRule, int) {
	// Find the @-rule type
	start := pos
	pos++ // Skip @

	// Read the at-rule name
	nameStart := pos
	for pos < len(content) && !isWhitespace(content[pos]) && content[pos] != '{' && content[pos] != ';' && content[pos] != '(' {
		pos++
	}
	atType := strings.ToLower(content[nameStart:pos])

	// Skip whitespace and any condition/query
	for pos < len(content) && content[pos] != '{' && content[pos] != ';' {
		pos++
	}

	if pos >= len(content) {
		return nil, pos
	}

	// Handle simple @-rules that end with semicolon (@import, @charset)
	if content[pos] == ';' {
		return &CSSRule{
			Selector:   "",
			Content:    strings.TrimSpace(content[start : pos+1]),
			IsAtRule:   true,
			AtRuleType: atType,
		}, pos + 1
	}

	// Handle block @-rules (@media, @keyframes, @font-face, @supports)
	if content[pos] == '{' {
		// Find matching closing brace
		braceCount := 1
		blockStart := pos + 1
		pos++
		for pos < len(content) && braceCount > 0 {
			if content[pos] == '{' {
				braceCount++
			} else if content[pos] == '}' {
				braceCount--
			}
			pos++
		}

		blockContent := content[blockStart : pos-1]
		fullContent := content[start:pos]

		rule := &CSSRule{
			Selector:   "",
			Content:    fullContent,
			IsAtRule:   true,
			AtRuleType: atType,
		}

		// Parse nested rules for @media and @supports
		if atType == "media" || atType == "supports" || atType == "layer" {
			rule.NestedRules = ParseCSS(blockContent)
		}

		return rule, pos
	}

	return nil, pos
}

// parseRegularRule parses a regular CSS rule (selector { properties }).
func parseRegularRule(content string, pos int) (*CSSRule, int) {
	// Find the opening brace
	selectorStart := pos
	for pos < len(content) && content[pos] != '{' && content[pos] != '@' {
		pos++
	}

	if pos >= len(content) || content[pos] == '@' {
		return nil, selectorStart
	}

	selector := strings.TrimSpace(content[selectorStart:pos])
	if selector == "" {
		return nil, pos
	}

	// Find the closing brace
	braceCount := 1
	pos++ // Skip opening brace
	propsStart := pos
	for pos < len(content) && braceCount > 0 {
		if content[pos] == '{' {
			braceCount++
		} else if content[pos] == '}' {
			braceCount--
		}
		pos++
	}

	props := content[propsStart : pos-1]

	return &CSSRule{
		Selector: selector,
		Content:  selector + " {" + props + "}",
		IsAtRule: false,
	}, pos
}

// isWhitespace returns true if c is a whitespace character.
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// ExtractSelectorsFromRule extracts individual selectors from a CSS rule.
// A rule like "h1, h2, .title { ... }" returns ["h1", "h2", ".title"].
func ExtractSelectorsFromRule(selector string) []string {
	parts := selectorSplitRegex.Split(selector, -1)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// ExtractClassesFromSelector extracts class names from a CSS selector.
func ExtractClassesFromSelector(selector string) []string {
	matches := classRegex.FindAllStringSubmatch(selector, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			result = append(result, match[1])
		}
	}
	return result
}

// ExtractIDsFromSelector extracts ID names from a CSS selector.
func ExtractIDsFromSelector(selector string) []string {
	matches := idRegex.FindAllStringSubmatch(selector, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			result = append(result, match[1])
		}
	}
	return result
}

// ExtractElementsFromSelector extracts element/tag names from a CSS selector.
func ExtractElementsFromSelector(selector string) []string {
	matches := elementRegex.FindAllStringSubmatch(selector, -1)
	result := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			elem := strings.ToLower(match[1])
			if !seen[elem] {
				result = append(result, elem)
				seen[elem] = true
			}
		}
	}
	return result
}

// ExtractAttributesFromSelector extracts attribute names from a CSS selector.
func ExtractAttributesFromSelector(selector string) []string {
	matches := attrRegex.FindAllStringSubmatch(selector, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			result = append(result, strings.ToLower(match[1]))
		}
	}
	return result
}
