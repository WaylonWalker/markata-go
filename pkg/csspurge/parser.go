package csspurge

import (
	"regexp"
	"strings"
)

// CSS at-rule type constants.
const (
	atRuleMedia = "media"
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

// Regular expressions for CSS parsing.
var (
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

	// Extract @-rule type (e.g., "media" from "@media")
	atRuleTypeRegex = regexp.MustCompile(`@([a-zA-Z-]+)`)
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

// parseAtRule parses an @-rule starting at the given position.
func parseAtRule(content string, startPos int) (rule *CSSRule, newPos int) {
	pos := startPos

	// Find the end of the @-rule header
	bracePos := strings.Index(content[pos:], "{")
	semicolonPos := strings.Index(content[pos:], ";")

	// If no { or ; found, invalid rule
	if bracePos == -1 && semicolonPos == -1 {
		return nil, startPos + 1
	}

	// Check if this is a simple @-rule (ends with ;) or a block rule (has { ... })
	if semicolonPos != -1 && (bracePos == -1 || semicolonPos < bracePos) {
		// Simple rule like @import or @charset
		endPos := pos + semicolonPos + 1
		rule = &CSSRule{
			IsAtRule: true,
			Content:  strings.TrimSpace(content[pos:endPos]),
		}
		// Determine rule type
		rule.AtRuleType = getAtRuleType(rule.Content)
		return rule, endPos
	}

	// Block rule with {...}
	bracePos = pos + bracePos
	endPos := findMatchingBrace(content, bracePos)
	if endPos == -1 {
		return nil, startPos + 1
	}
	endPos++

	fullContent := strings.TrimSpace(content[pos:endPos])
	if fullContent == "" {
		return nil, startPos + 1
	}

	// Parse the @-rule
	rule = &CSSRule{
		IsAtRule: true,
		Content:  fullContent,
	}

	// Extract the rule type and nested content
	rule.AtRuleType = getAtRuleType(fullContent)

	// For @media and other nested rules, parse nested content
	if rule.AtRuleType == atRuleMedia || rule.AtRuleType == "supports" || rule.AtRuleType == "layer" {
		// Extract content inside braces
		innerContent := extractInnerContent(fullContent)
		if innerContent != "" {
			rule.NestedRules = parseRules(innerContent, nil)
		}
	}

	return rule, endPos
}

// parseRegularRule parses a regular CSS rule.
func parseRegularRule(content string, startPos int) (rule *CSSRule, newPos int) {
	// Find the opening brace
	bracePos := strings.Index(content[startPos:], "{")
	if bracePos == -1 {
		return nil, startPos + 1
	}
	bracePos += startPos

	// Find the matching closing brace
	endPos := findMatchingBrace(content, bracePos)
	if endPos == -1 {
		return nil, startPos + 1
	}
	endPos++

	// Extract selector and content
	selector := strings.TrimSpace(content[startPos:bracePos])
	if selector == "" {
		return nil, endPos
	}

	fullContent := strings.TrimSpace(content[startPos:endPos])

	return &CSSRule{
		Selector: selector,
		Content:  fullContent,
	}, endPos
}

// getAtRuleType extracts the type of @-rule.
func getAtRuleType(content string) string {
	matches := atRuleTypeRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.ToLower(matches[1])
	}
	return ""
}

// findMatchingBrace finds the matching closing brace for the opening brace at position start.
func findMatchingBrace(content string, start int) int {
	count := 0
	for i := start; i < len(content); i++ {
		switch content[i] {
		case '{':
			count++
		case '}':
			count--
			if count == 0 {
				return i
			}
		}
	}
	return -1
}

// extractInnerContent extracts the content inside braces.
func extractInnerContent(content string) string {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(content[start+1 : end])
}

// isWhitespace returns true for CSS whitespace characters.
func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t'
}

// ExtractSelectorsFromRule extracts individual selectors from a selector list.
func ExtractSelectorsFromRule(selector string) []string {
	selectors := selectorSplitRegex.Split(selector, -1)
	result := make([]string, 0, len(selectors))
	for _, sel := range selectors {
		trimmed := strings.TrimSpace(sel)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ExtractClassesFromSelector extracts class names from a selector.
func ExtractClassesFromSelector(selector string) []string {
	matches := classRegex.FindAllStringSubmatch(selector, -1)
	classes := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			classes = append(classes, match[1])
		}
	}
	return classes
}

// ExtractIDsFromSelector extracts IDs from a selector.
func ExtractIDsFromSelector(selector string) []string {
	matches := idRegex.FindAllStringSubmatch(selector, -1)
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			ids = append(ids, match[1])
		}
	}
	return ids
}

// ExtractElementsFromSelector extracts unique element names from a selector.
// Element names are normalized to lowercase and deduplicated.
func ExtractElementsFromSelector(selector string) []string {
	matches := elementRegex.FindAllStringSubmatch(selector, -1)
	elements := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			elem := strings.ToLower(match[1])
			if !seen[elem] {
				elements = append(elements, elem)
				seen[elem] = true
			}
		}
	}
	return elements
}

// ExtractAttributesFromSelector extracts attribute names from a selector.
// Attribute names are normalized to lowercase for case-insensitive matching.
func ExtractAttributesFromSelector(selector string) []string {
	matches := attrRegex.FindAllStringSubmatch(selector, -1)
	attrs := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			attrs = append(attrs, strings.ToLower(match[1]))
		}
	}
	return attrs
}
