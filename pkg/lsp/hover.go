package lsp

import (
	"context"
	"encoding/json"
	"strings"
)

// handleHover handles textDocument/hover requests.
func (s *Server) handleHover(_ context.Context, msg *Message) error {
	var params HoverParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, InvalidParams, "invalid hover params")
	}

	// Get the document
	s.docMu.RLock()
	doc, ok := s.documents[params.TextDocument.URI]
	s.docMu.RUnlock()

	if !ok {
		return s.sendResponse(msg.ID, nil)
	}

	// Get the line at the cursor position
	lines := strings.Split(doc.Content, "\n")
	if params.Position.Line >= len(lines) {
		return s.sendResponse(msg.ID, nil)
	}

	line := lines[params.Position.Line]
	col := params.Position.Character

	// Check if the cursor is on a wikilink
	slug, wikilinkRange := getWikilinkAtPosition(line, col, params.Position.Line)
	if slug == "" {
		return s.sendResponse(msg.ID, nil)
	}

	// Look up the post
	post := s.index.GetBySlug(slug)
	if post == nil {
		// Show error hover for broken links
		hover := &Hover{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: "**Broken link**\n\nTarget post `" + slug + "` not found.",
			},
			Range: wikilinkRange,
		}
		return s.sendResponse(msg.ID, hover)
	}

	// Format hover content
	var sb strings.Builder
	sb.WriteString("## ")
	sb.WriteString(post.Title)
	sb.WriteString("\n\n")

	if post.Description != "" {
		sb.WriteString(post.Description)
		sb.WriteString("\n\n")
	}

	sb.WriteString("---\n")
	sb.WriteString("*Slug:* `")
	sb.WriteString(post.Slug)
	sb.WriteString("`\n\n")
	sb.WriteString("*Path:* `")
	sb.WriteString(post.Path)
	sb.WriteString("`")

	hover := &Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: sb.String(),
		},
		Range: wikilinkRange,
	}

	return s.sendResponse(msg.ID, hover)
}

// getWikilinkAtPosition returns the wikilink slug at the given position.
// Returns empty string if the position is not on a wikilink.
func getWikilinkAtPosition(line string, col, lineNum int) (string, *Range) {
	// Find all wikilinks on this line
	matches := wikilinkRegex.FindAllStringSubmatchIndex(line, -1)

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		// match[0:2] is the full match position
		// match[2:4] is the slug group position
		start := match[0]
		end := match[1]

		// Check if cursor is within this wikilink
		if col >= start && col <= end {
			slug := strings.TrimSpace(line[match[2]:match[3]])
			wikilinkRange := &Range{
				Start: Position{Line: lineNum, Character: start},
				End:   Position{Line: lineNum, Character: end},
			}
			return slug, wikilinkRange
		}
	}

	return "", nil
}
