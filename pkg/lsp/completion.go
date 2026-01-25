package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// CompletionParams contains the parameters for textDocument/completion.
type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      *CompletionContext     `json:"context,omitempty"`
}

// CompletionContext contains additional information about the context.
type CompletionContext struct {
	TriggerKind      int     `json:"triggerKind"`
	TriggerCharacter *string `json:"triggerCharacter,omitempty"`
}

// CompletionList represents a collection of completion items.
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// CompletionItem represents a completion suggestion.
type CompletionItem struct {
	Label            string         `json:"label"`
	Kind             int            `json:"kind,omitempty"`
	Detail           string         `json:"detail,omitempty"`
	Documentation    *MarkupContent `json:"documentation,omitempty"`
	InsertText       string         `json:"insertText,omitempty"`
	InsertTextFormat int            `json:"insertTextFormat,omitempty"`
	TextEdit         *TextEdit      `json:"textEdit,omitempty"`
	FilterText       string         `json:"filterText,omitempty"`
	SortText         string         `json:"sortText,omitempty"`
	Data             interface{}    `json:"data,omitempty"`
	AdditionalEdits  []TextEdit     `json:"additionalTextEdits,omitempty"`
	CommitCharacters []string       `json:"commitCharacters,omitempty"`
}

// CompletionItemKind defines types of completion items.
const (
	CompletionItemKindText          = 1
	CompletionItemKindMethod        = 2
	CompletionItemKindFunction      = 3
	CompletionItemKindConstructor   = 4
	CompletionItemKindField         = 5
	CompletionItemKindVariable      = 6
	CompletionItemKindClass         = 7
	CompletionItemKindInterface     = 8
	CompletionItemKindModule        = 9
	CompletionItemKindProperty      = 10
	CompletionItemKindUnit          = 11
	CompletionItemKindValue         = 12
	CompletionItemKindEnum          = 13
	CompletionItemKindKeyword       = 14
	CompletionItemKindSnippet       = 15
	CompletionItemKindColor         = 16
	CompletionItemKindFile          = 17
	CompletionItemKindReference     = 18
	CompletionItemKindFolder        = 19
	CompletionItemKindEnumMember    = 20
	CompletionItemKindConstant      = 21
	CompletionItemKindStruct        = 22
	CompletionItemKindEvent         = 23
	CompletionItemKindOperator      = 24
	CompletionItemKindTypeParameter = 25
)

// InsertTextFormat constants.
const (
	InsertTextFormatPlainText = 1
	InsertTextFormatSnippet   = 2
)

// handleCompletion handles textDocument/completion requests.
func (s *Server) handleCompletion(_ context.Context, msg *Message) error {
	var params CompletionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.sendError(msg.ID, InvalidParams, "invalid completion params")
	}

	// Get the document
	s.docMu.RLock()
	doc, ok := s.documents[params.TextDocument.URI]
	s.docMu.RUnlock()

	if !ok {
		return s.sendResponse(msg.ID, &CompletionList{Items: []CompletionItem{}})
	}

	// Get the line at the cursor position
	lines := strings.Split(doc.Content, "\n")
	if params.Position.Line >= len(lines) {
		return s.sendResponse(msg.ID, &CompletionList{Items: []CompletionItem{}})
	}

	line := lines[params.Position.Line]
	col := params.Position.Character
	if col > len(line) {
		col = len(line)
	}

	// Check if we're inside a wikilink
	prefix, startCol, inWikilink := getWikilinkContext(line, col)
	if !inWikilink {
		return s.sendResponse(msg.ID, &CompletionList{Items: []CompletionItem{}})
	}

	// Get matching posts
	var posts []*PostInfo
	if prefix == "" {
		posts = s.index.AllPosts()
	} else {
		posts = s.index.SearchPosts(prefix)
	}

	// Sort by title for consistent ordering
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Title < posts[j].Title
	})

	// Build completion items
	items := make([]CompletionItem, 0, len(posts))
	for i, post := range posts {
		// Create completion item
		item := CompletionItem{
			Label:  post.Slug,
			Kind:   CompletionItemKindReference,
			Detail: post.Title,
			Documentation: &MarkupContent{
				Kind:  "markdown",
				Value: formatPostDocumentation(post),
			},
			InsertText:       post.Slug,
			InsertTextFormat: InsertTextFormatPlainText,
			FilterText:       post.Slug + " " + post.Title,
			SortText:         fmt.Sprintf("%05d", i), // Preserve sort order
		}

		// If we have a prefix, use TextEdit to replace it
		if prefix != "" {
			item.TextEdit = &TextEdit{
				Range: Range{
					Start: Position{Line: params.Position.Line, Character: startCol},
					End:   Position{Line: params.Position.Line, Character: col},
				},
				NewText: post.Slug,
			}
		}

		items = append(items, item)
	}

	result := &CompletionList{
		IsIncomplete: false,
		Items:        items,
	}

	return s.sendResponse(msg.ID, result)
}

// wikilinkStartRegex matches [[ at the beginning of a wikilink.
var wikilinkStartRegex = regexp.MustCompile(`\[\[([^\]|]*)$`)

// getWikilinkContext checks if the cursor is inside a wikilink and returns the prefix.
// Returns (prefix, startColumn, isInWikilink).
func getWikilinkContext(line string, col int) (prefix string, startCol int, inWikilink bool) {
	if col > len(line) {
		col = len(line)
	}

	// Look for [[ before the cursor
	textBeforeCursor := line[:col]

	// Check if we're inside a wikilink (after [[ but not after ]])
	match := wikilinkStartRegex.FindStringSubmatchIndex(textBeforeCursor)
	if match == nil {
		return "", 0, false
	}

	// match[0:2] is the full match position
	// match[2:4] is the captured slug prefix position
	startCol = match[2] // Start of the slug prefix
	prefix = textBeforeCursor[startCol:]

	// Check if we're in the display text part (after |)
	if strings.Contains(prefix, "|") {
		return "", 0, false
	}

	return prefix, startCol, true
}

// formatPostDocumentation formats post info for display in completion documentation.
func formatPostDocumentation(post *PostInfo) string {
	var sb strings.Builder

	sb.WriteString("**")
	sb.WriteString(post.Title)
	sb.WriteString("**\n\n")

	if post.Description != "" {
		sb.WriteString(post.Description)
		sb.WriteString("\n\n")
	}

	sb.WriteString("*Path: ")
	sb.WriteString(post.Path)
	sb.WriteString("*")

	return sb.String()
}
