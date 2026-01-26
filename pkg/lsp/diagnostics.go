package lsp

import (
	"regexp"
	"strings"
)

// publishDiagnostics publishes diagnostics for a document.
func (s *Server) publishDiagnostics(uri, content string) error {
	diagnostics := s.computeDiagnostics(uri, content)

	params := PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	}

	return s.sendNotification("textDocument/publishDiagnostics", params)
}

// computeDiagnostics computes diagnostics for a document.
func (s *Server) computeDiagnostics(_, content string) []Diagnostic {
	var diagnostics []Diagnostic

	// Find all wikilinks and check if they resolve
	lines := strings.Split(content, "\n")

	// Skip fenced code blocks
	inCodeBlock := false
	codeBlockPattern := regexp.MustCompile("^```|^~~~")

	for lineNum, line := range lines {
		// Track code block state
		if codeBlockPattern.MatchString(strings.TrimSpace(line)) {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		// Find wikilinks on this line
		matches := wikilinkRegex.FindAllStringSubmatchIndex(line, -1)
		for _, match := range matches {
			if len(match) < 4 {
				continue
			}

			// Extract the slug from group 1
			slugStart := match[2]
			slugEnd := match[3]
			slug := strings.TrimSpace(line[slugStart:slugEnd])

			// Check if the target exists
			if s.index.GetBySlug(slug) == nil {
				diag := Diagnostic{
					Range: Range{
						Start: Position{Line: lineNum, Character: match[0]},
						End:   Position{Line: lineNum, Character: match[1]},
					},
					Severity: DiagnosticSeverityWarning,
					Source:   "markata-go",
					Message:  "Broken wikilink: target post \"" + slug + "\" not found",
					Code:     "broken-wikilink",
				}
				diagnostics = append(diagnostics, diag)
			}
		}
	}

	return diagnostics
}

// clearDiagnostics clears diagnostics for a document.
func (s *Server) clearDiagnostics(uri string) error {
	params := PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: []Diagnostic{},
	}

	return s.sendNotification("textDocument/publishDiagnostics", params)
}
