package htmltotext

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

// Pre-compiled regex patterns for HTML parsing.
var (
	// Matches any HTML tag (opening, closing, or self-closing).
	htmlTagRe = regexp.MustCompile(`<[^>]*>`)

	// Matches opening anchor tags and captures the href attribute.
	anchorOpenRe = regexp.MustCompile(`(?i)<a\s[^>]*href\s*=\s*["']([^"']*)["'][^>]*>`)

	// Matches closing anchor tags.
	anchorCloseRe = regexp.MustCompile(`(?i)</a\s*>`)

	// Matches <br> and <br/> tags (with optional whitespace).
	brTagRe = regexp.MustCompile(`(?i)<br\s*/?\s*>`)

	// Matches block-level closing tags that should produce line breaks.
	blockCloseRe = regexp.MustCompile(`(?i)</(?:p|div|section|article|header|footer|nav|aside|blockquote|li|dd|dt|figcaption|figure|main)\s*>`)

	// Matches heading closing tags (produce double line breaks).
	headingCloseRe = regexp.MustCompile(`(?i)</h[1-6]\s*>`)

	// Matches <hr> / <hr/> tags.
	hrTagRe = regexp.MustCompile(`(?i)<hr\s*/?\s*>`)

	// Matches <li> opening tags to insert list bullet.
	liOpenRe = regexp.MustCompile(`(?i)<li[^>]*>`)

	// Collapses 3+ consecutive newlines to 2.
	multiNewlineRe = regexp.MustCompile(`\n{3,}`)

	// Collapses multiple spaces (not newlines) to a single space.
	multiSpaceRe = regexp.MustCompile(`[^\S\n]+`)
)

// Convert transforms HTML content into plain text with footnote-style link
// references. It decodes HTML entities, strips tags while preserving block
// structure, and appends a references section for any hyperlinks found.
//
// Links where the visible text matches the URL are rendered inline without
// a footnote reference. Duplicate URLs share the same reference number.
func Convert(htmlContent string) string {
	if htmlContent == "" {
		return ""
	}

	// Phase 1: Extract links and replace anchor tags with placeholders
	var links []linkRef
	urlToRef := make(map[string]int) // url -> 1-based reference number
	nextRef := 1

	// Process anchor tags: extract href and link text, replace with placeholders.
	// We process the HTML string by finding anchor open/close pairs.
	result := processAnchors(htmlContent, &links, urlToRef, &nextRef)

	// Phase 2: Convert block-level tags to newlines for structure
	result = hrTagRe.ReplaceAllString(result, "\n\n---\n\n")
	result = brTagRe.ReplaceAllString(result, "\n")
	result = liOpenRe.ReplaceAllString(result, "\n- ")
	result = headingCloseRe.ReplaceAllString(result, "\n\n")
	result = blockCloseRe.ReplaceAllString(result, "\n\n")

	// Phase 3: Strip remaining HTML tags
	result = htmlTagRe.ReplaceAllString(result, "")

	// Phase 4: Decode HTML entities
	result = html.UnescapeString(result)

	// Phase 5: Clean up whitespace
	// Collapse multiple spaces (not newlines) to single space
	result = multiSpaceRe.ReplaceAllString(result, " ")
	// Clean up spaces around newlines
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	result = strings.Join(lines, "\n")
	// Collapse 3+ newlines to 2
	result = multiNewlineRe.ReplaceAllString(result, "\n\n")
	result = strings.TrimSpace(result)

	// Phase 6: Append references section if there are any links
	if len(links) > 0 {
		var refs strings.Builder
		refs.WriteString("\n\nReferences:\n")
		// Build deduplicated reference list in order of first appearance
		seen := make(map[int]bool)
		for _, link := range links {
			refNum := urlToRef[link.url]
			if !seen[refNum] {
				seen[refNum] = true
				refs.WriteString("[")
				refs.WriteString(strconv.Itoa(refNum))
				refs.WriteString("]: ")
				refs.WriteString(link.url)
				refs.WriteString("\n")
			}
		}
		result += refs.String()
		result = strings.TrimRight(result, "\n")
	}

	return result
}

// processAnchors finds all <a href="...">text</a> pairs in the HTML and
// replaces them with either "text [N]" (footnote reference) or just "text"
// (when text matches URL). It populates the links slice and urlToRef map.
func processAnchors(
	htmlContent string,
	links *[]linkRef,
	urlToRef map[string]int,
	nextRef *int,
) string {
	type anchorSpan struct {
		start    int // start of <a ...>
		end      int // end of </a> (after >)
		href     string
		textHTML string // HTML between <a> and </a>
	}

	// Find all anchor open tags
	openMatches := anchorOpenRe.FindAllStringIndex(htmlContent, -1)
	if len(openMatches) == 0 {
		return htmlContent
	}

	spans := make([]anchorSpan, 0, len(openMatches))

	lastEnd := 0
	// For each open tag, find the matching close tag
	for _, openIdx := range openMatches {
		// Skip if this open tag starts before the previous span ended
		// (handles nested anchors which would create invalid spans)
		if openIdx[0] < lastEnd {
			continue
		}

		openTag := htmlContent[openIdx[0]:openIdx[1]]
		hrefMatch := anchorOpenRe.FindStringSubmatch(openTag)
		if len(hrefMatch) < 2 {
			continue
		}
		href := hrefMatch[1]

		// Find the next </a> after this open tag
		closeIdx := anchorCloseRe.FindStringIndex(htmlContent[openIdx[1]:])
		if closeIdx == nil {
			continue
		}

		textStart := openIdx[1]
		textEnd := openIdx[1] + closeIdx[0]
		fullEnd := openIdx[1] + closeIdx[1]

		spans = append(spans, anchorSpan{
			start:    openIdx[0],
			end:      fullEnd,
			href:     href,
			textHTML: htmlContent[textStart:textEnd],
		})
		lastEnd = fullEnd
	}

	if len(spans) == 0 {
		return htmlContent
	}

	// First pass (left-to-right): assign reference numbers and compute replacements
	type spanReplacement struct {
		start       int
		end         int
		replacement string
	}
	replacements := make([]spanReplacement, len(spans))

	for i, span := range spans {
		// Strip any nested HTML tags from link text
		linkText := htmlTagRe.ReplaceAllString(span.textHTML, "")
		linkText = html.UnescapeString(linkText)
		linkText = strings.TrimSpace(linkText)

		var replacement string
		// If link text matches the URL, just show the URL inline (no footnote)
		if linkText == span.href || linkText == strings.TrimSuffix(span.href, "/") ||
			strings.TrimSuffix(linkText, "/") == strings.TrimSuffix(span.href, "/") {
			replacement = linkText
		} else {
			// Assign or reuse reference number
			refNum, exists := urlToRef[span.href]
			if !exists {
				refNum = *nextRef
				urlToRef[span.href] = refNum
				*nextRef++
			}
			*links = append(*links, linkRef{url: span.href, text: linkText})
			replacement = linkText + " [" + strconv.Itoa(refNum) + "]"
		}

		replacements[i] = spanReplacement{
			start:       span.start,
			end:         span.end,
			replacement: replacement,
		}
	}

	// Second pass (right-to-left): apply replacements to preserve indices
	result := htmlContent
	for i := len(replacements) - 1; i >= 0; i-- {
		r := replacements[i]
		result = result[:r.start] + r.replacement + result[r.end:]
	}

	return result
}

// linkRef holds a link's URL and visible text for footnote generation.
type linkRef struct {
	url  string
	text string
}
