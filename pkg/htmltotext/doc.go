// Package htmltotext converts HTML content to plain text with proper formatting.
//
// # Overview
//
// This package provides HTML-to-plain-text conversion that:
//   - Decodes all HTML entities to their Unicode equivalents
//   - Converts hyperlinks to footnote-style references (Lynx/Pandoc convention)
//   - Strips all HTML tags while preserving meaningful whitespace
//   - Preserves block-level structure (paragraphs, headings, lists)
//
// # Link Formatting
//
// Links are converted to footnote-style references following the Lynx/Pandoc
// convention. Each unique URL gets a sequential reference number:
//
//	Input:  <a href="https://go.dev">Go</a> is great. See <a href="https://go.dev/doc">docs</a>.
//	Output: Go [1] is great. See docs [2].
//
//	        References:
//	        [1]: https://go.dev
//	        [2]: https://go.dev/doc
//
// When the link text matches the URL (bare links), no footnote is added:
//
//	Input:  Visit <a href="https://go.dev">https://go.dev</a>
//	Output: Visit https://go.dev
//
// Duplicate URLs reuse the same reference number.
//
// # Usage
//
//	text := htmltotext.Convert("<p>Hello &amp; <a href=\"https://go.dev\">Go</a></p>")
//	// Returns: "Hello & Go [1]\n\nReferences:\n[1]: https://go.dev"
package htmltotext
