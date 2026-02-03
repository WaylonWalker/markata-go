// Package diagnostics provides shared diagnostic checks for markdown files.
//
// This package contains diagnostic rules that are used by both the LSP server
// and the CLI lint command to ensure consistent issue reporting across tools.
//
// # Diagnostic Types
//
// The package detects the following issues:
//   - broken-wikilink: Wikilinks pointing to non-existent posts
//   - unknown-mention: Mentions (@handle) not found in blogroll
//   - h1-in-content: H1 headings in content (templates add H1 from title)
//   - duplicate-key: Duplicate YAML keys in frontmatter
//   - invalid-date: Invalid date formats (non-ISO 8601)
//   - missing-alt-text: Images without alt text
//   - protocol-less-url: URLs without protocol (//example.com)
//   - admonition-fenced-code: Fenced code blocks in admonitions without blank line
//
// # Usage
//
// The main entry point is the Check function which runs all diagnostic checks:
//
//	issues := diagnostics.Check(filePath, content, nil)
//
// For wikilink and mention checking, provide a Resolver:
//
//	issues := diagnostics.Check(filePath, content, resolver)
package diagnostics
