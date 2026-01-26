package lsp

import (
	"context"
	"encoding/json"
	"strings"
)

// handleDefinition handles textDocument/definition requests.
func (s *Server) handleDefinition(_ context.Context, msg *Message) error {
	var params DefinitionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, InvalidParams, "invalid definition params")
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

	// Check if the cursor is on a mention (@handle)
	handle, _ := getMentionAtPosition(line, col, params.Position.Line)
	if handle != "" {
		mention := s.index.GetByHandle(handle)
		if mention != nil && mention.SiteURL != "" {
			// Return a location with the site URL
			// LSP clients may open this as an external link
			location := Location{
				URI: mention.SiteURL,
				Range: Range{
					Start: Position{Line: 0, Character: 0},
					End:   Position{Line: 0, Character: 0},
				},
			}
			return s.sendResponse(msg.ID, location)
		}
		return s.sendResponse(msg.ID, nil)
	}

	// Check if the cursor is on a wikilink
	slug, _ := getWikilinkAtPosition(line, col, params.Position.Line)
	if slug == "" {
		return s.sendResponse(msg.ID, nil)
	}

	// Look up the post
	post := s.index.GetBySlug(slug)
	if post == nil {
		return s.sendResponse(msg.ID, nil)
	}

	// Return the location of the target file
	location := Location{
		URI: post.URI,
		Range: Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: 0, Character: 0},
		},
	}

	return s.sendResponse(msg.ID, location)
}
