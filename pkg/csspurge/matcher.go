package csspurge

import (
	"path/filepath"
	"strings"
)

// PurgeOptions configures CSS purging behavior.
type PurgeOptions struct {
	// Preserve is a list of glob patterns for selectors to keep.
	// Example: ["js-*", "htmx-*", "active", "hidden"]
	Preserve []string

	// Verbose enables detailed logging.
	Verbose bool
}

// PurgeStats tracks statistics about CSS purging.
type PurgeStats struct {
	// TotalRules is the number of rules before purging.
	TotalRules int
	// KeptRules is the number of rules kept after purging.
	KeptRules int
	// RemovedRules is the number of rules removed.
	RemovedRules int
	// OriginalSize is the original CSS size in bytes.
	OriginalSize int
	// PurgedSize is the purged CSS size in bytes.
	PurgedSize int
}

// SavingsPercent returns the percentage of size reduction.
func (s *PurgeStats) SavingsPercent() float64 {
	if s.OriginalSize == 0 {
		return 0
	}
	return float64(s.OriginalSize-s.PurgedSize) / float64(s.OriginalSize) * 100
}

// PurgeCSS removes unused CSS rules based on used selectors.
// It returns the purged CSS content and statistics.
func PurgeCSS(css string, used *UsedSelectors, opts PurgeOptions) (string, PurgeStats) {
	stats := PurgeStats{
		OriginalSize: len(css),
	}

	rules := ParseCSS(css)
	stats.TotalRules = countRules(rules)

	var result strings.Builder
	for _, rule := range rules {
		kept := processRule(rule, used, opts, &result)
		if kept {
			stats.KeptRules++
		}
	}

	purged := result.String()
	stats.PurgedSize = len(purged)
	stats.RemovedRules = stats.TotalRules - stats.KeptRules

	return purged, stats
}

// countRules counts the total number of rules including nested ones.
func countRules(rules []CSSRule) int {
	count := 0
	for _, rule := range rules {
		count++
		count += countRules(rule.NestedRules)
	}
	return count
}

// processRule processes a single CSS rule and writes it if used.
// Returns true if the rule was kept.
func processRule(rule CSSRule, used *UsedSelectors, opts PurgeOptions, out *strings.Builder) bool {
	// Always keep certain @-rules
	if rule.IsAtRule {
		switch rule.AtRuleType {
		case "charset", "import", "font-face", "keyframes", "-webkit-keyframes", "-moz-keyframes":
			// Always preserve these
			out.WriteString(rule.Content)
			out.WriteString("\n")
			return true

		case atRuleMedia, "supports", "layer":
			// Process nested rules
			var nestedOut strings.Builder
			keptNested := 0
			for _, nested := range rule.NestedRules {
				if processRule(nested, used, opts, &nestedOut) {
					keptNested++
				}
			}

			// Only keep @media if it has used rules
			if keptNested > 0 {
				// Reconstruct the @-rule with filtered content
				// Extract the at-rule header (everything before the {)
				idx := strings.Index(rule.Content, "{")
				if idx != -1 {
					header := rule.Content[:idx+1]
					out.WriteString(header)
					out.WriteString("\n")
					out.WriteString(nestedOut.String())
					out.WriteString("}\n")
					return true
				}
			}
			return false

		default:
			// Unknown @-rule - preserve to be safe
			out.WriteString(rule.Content)
			out.WriteString("\n")
			return true
		}
	}

	// Check if regular rule is used
	if isSelectorUsed(rule.Selector, used, opts.Preserve) {
		out.WriteString(rule.Content)
		out.WriteString("\n")
		return true
	}

	return false
}

// isSelectorUsed checks if a CSS selector matches any used elements.
// For comma-separated selectors, returns true if ANY selector matches.
func isSelectorUsed(selector string, used *UsedSelectors, preserve []string) bool {
	selectors := ExtractSelectorsFromRule(selector)

	for _, sel := range selectors {
		if isSingleSelectorUsed(sel, used, preserve) {
			return true
		}
	}

	return false
}

// isSingleSelectorUsed checks if a single CSS selector is used.
func isSingleSelectorUsed(selector string, used *UsedSelectors, preserve []string) bool {
	// Check if selector matches a preserve pattern
	if matchesPreservePattern(selector, preserve) {
		return true
	}

	// Universal selector is always used
	if strings.Contains(selector, "*") {
		return true
	}

	// Extract components from selector
	classes := ExtractClassesFromSelector(selector)
	ids := ExtractIDsFromSelector(selector)
	elements := ExtractElementsFromSelector(selector)
	attrs := ExtractAttributesFromSelector(selector)

	// For pure element selectors (no class/id), check if element is used
	if len(classes) == 0 && len(ids) == 0 && len(attrs) == 0 && len(elements) > 0 {
		return anyElementUsed(elements, used)
	}

	// Check classes, IDs, and attributes
	if !allClassesUsed(classes, used, preserve) {
		return false
	}
	if !allIDsUsed(ids, used, preserve) {
		return false
	}
	if !allAttributesUsed(attrs, used) {
		return false
	}

	// If we have classes or IDs that passed, the selector is used
	if len(classes) > 0 || len(ids) > 0 || len(attrs) > 0 {
		return true
	}

	// Fall back to element check
	return anyElementUsed(elements, used)
}

// anyElementUsed checks if any element is in the used set.
func anyElementUsed(elements []string, used *UsedSelectors) bool {
	for _, elem := range elements {
		if used.Elements[elem] {
			return true
		}
	}
	return false
}

// allClassesUsed checks if all classes (not matching preserve patterns) are used.
func allClassesUsed(classes []string, used *UsedSelectors, preserve []string) bool {
	for _, class := range classes {
		if matchesPreservePatterns(class, preserve) {
			continue
		}
		if !used.Classes[class] {
			return false
		}
	}
	return true
}

// allIDsUsed checks if all IDs (not matching preserve patterns) are used.
func allIDsUsed(ids []string, used *UsedSelectors, preserve []string) bool {
	for _, id := range ids {
		if matchesPreservePatterns(id, preserve) {
			continue
		}
		if !used.IDs[id] {
			return false
		}
	}
	return true
}

// allAttributesUsed checks if all attributes are used.
func allAttributesUsed(attrs []string, used *UsedSelectors) bool {
	for _, attr := range attrs {
		if !used.Attributes[attr] {
			return false
		}
	}
	return true
}

// matchesPreservePattern checks if a selector matches any preserve pattern.
func matchesPreservePattern(selector string, patterns []string) bool {
	// Extract classes and IDs from selector to check against patterns
	classes := ExtractClassesFromSelector(selector)
	ids := ExtractIDsFromSelector(selector)

	for _, pattern := range patterns {
		// Match the entire selector directly
		if matched, _ := filepath.Match(pattern, selector); matched {
			return true
		}

		// Match against classes
		for _, class := range classes {
			if matched, _ := filepath.Match(pattern, class); matched {
				return true
			}
		}

		// Match against IDs
		for _, id := range ids {
			if matched, _ := filepath.Match(pattern, id); matched {
				return true
			}
		}
	}

	return false
}

// matchesPreservePatterns checks if a class or ID matches any preserve pattern.
func matchesPreservePatterns(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, value); matched {
			return true
		}
	}
	return false
}

// DefaultPreservePatterns returns the default patterns for selectors to preserve.
// These are commonly used by JavaScript frameworks and should not be purged.
func DefaultPreservePatterns() []string {
	return []string{
		// JavaScript-added classes
		"js-*",

		// HTMX framework classes
		"htmx-*",

		// Alpine.js framework (x-show, x-bind, x-data, x-cloak, etc.)
		"x-*",

		// Pagefind search UI classes
		"pagefind-*",

		// GLightbox image viewer
		"glightbox*",
		"gslide*",
		"goverlay*",

		// Common state classes (often added by JS)
		"active",
		"inactive",
		"hidden",
		"visible",
		"show",
		"hide",
		"open",
		"closed",
		"loading",
		"loaded",
		"error",
		"success",
		"disabled",
		"enabled",
		"selected",
		"focused",
		"expanded",
		"collapsed",

		// Theme/mode classes
		"dark",
		"light",
		"dark-mode",
		"light-mode",

		// Animation classes
		"fade-*",
		"slide-*",
		"animate-*",

		// Transition classes
		"transition-*",
		"entering",
		"leaving",

		// Accessibility
		"sr-only",
		"visually-hidden",
	}
}
