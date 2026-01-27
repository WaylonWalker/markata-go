package lsp

import (
	"context"
	"encoding/json"
	"regexp"
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

	// Check if the cursor is on a mention (@handle)
	handle, mentionRange := getMentionAtPosition(line, col, params.Position.Line)
	if handle != "" {
		return s.handleMentionHover(msg, handle, mentionRange)
	}

	// Check if we're in frontmatter
	if hover := s.getFrontmatterHover(doc.Content, lines, params.Position.Line, col); hover != nil {
		return s.sendResponse(msg.ID, hover)
	}

	// Check if we're on an admonition type
	if hover := s.getAdmonitionHover(line, params.Position.Line, col); hover != nil {
		return s.sendResponse(msg.ID, hover)
	}

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

// handleMentionHover handles hover for @mentions.
func (s *Server) handleMentionHover(msg *Message, handle string, mentionRange *Range) error {
	mention := s.index.GetByHandle(handle)
	if mention == nil {
		// Show error hover for unknown mentions
		hover := &Hover{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: "**Unknown mention**\n\n`@" + handle + "` not found in blogroll configuration.",
			},
			Range: mentionRange,
		}
		return s.sendResponse(msg.ID, hover)
	}

	// Format hover content
	var sb strings.Builder
	sb.WriteString("## @")
	sb.WriteString(mention.Handle)
	if mention.Title != "" {
		sb.WriteString(" - ")
		sb.WriteString(mention.Title)
	}
	sb.WriteString("\n\n")

	if mention.Description != "" {
		sb.WriteString(mention.Description)
		sb.WriteString("\n\n")
	}

	sb.WriteString("---\n")
	if mention.SiteURL != "" {
		sb.WriteString("*Site:* ")
		sb.WriteString(mention.SiteURL)
		sb.WriteString("\n\n")
	}
	if mention.FeedURL != "" {
		sb.WriteString("*Feed:* ")
		sb.WriteString(mention.FeedURL)
		sb.WriteString("\n\n")
	}
	if len(mention.Aliases) > 0 {
		sb.WriteString("*Aliases:* @")
		sb.WriteString(strings.Join(mention.Aliases, ", @"))
	}

	hover := &Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: sb.String(),
		},
		Range: mentionRange,
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

// admonitionLineRegex matches a line starting an admonition block.
// Matches: !!!, ???, ???+ followed by space and type name (optionally with title in quotes).
var admonitionLineRegex = regexp.MustCompile(`^(\s*)(\?{3}\+?|!!!)\s+(\w+)(?:\s+"[^"]*")?`)

// getFrontmatterHover returns hover information if the cursor is on a frontmatter field.
func (s *Server) getFrontmatterHover(_ string, lines []string, lineNum, col int) *Hover {
	// Find frontmatter boundaries
	startLine, endLine := findFrontmatterBoundaries(lines)
	if startLine == -1 || endLine == -1 {
		return nil
	}

	// Check if cursor is within frontmatter
	if lineNum <= startLine || lineNum >= endLine {
		return nil
	}

	currentLine := lines[lineNum]
	if col > len(currentLine) {
		col = len(currentLine)
	}

	// Check if this is a top-level field line (not indented)
	trimmedLine := strings.TrimLeft(currentLine, " \t")
	if strings.HasPrefix(trimmedLine, "- ") {
		// It's a list item, not a field
		return nil
	}

	// Find the colon position
	colonIdx := strings.Index(currentLine, ":")
	if colonIdx == -1 {
		return nil
	}

	// Get the field name
	leadingSpaces := len(currentLine) - len(trimmedLine)
	if leadingSpaces > 0 {
		// Indented line, not a top-level field
		return nil
	}

	fieldName := strings.TrimSpace(currentLine[:colonIdx])
	if fieldName == "" {
		return nil
	}

	// Look up the field definition
	var field *FrontmatterField
	for i := range frontmatterFields {
		if frontmatterFields[i].Name == fieldName {
			field = &frontmatterFields[i]
			break
		}
	}

	if field == nil {
		// Unknown field, show generic hover
		return &Hover{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: "**" + fieldName + "**\n\n*Custom field*",
			},
			Range: &Range{
				Start: Position{Line: lineNum, Character: 0},
				End:   Position{Line: lineNum, Character: colonIdx},
			},
		}
	}

	// Determine if hovering over field name or value
	var hoverRange *Range
	if col <= colonIdx {
		// Hovering over field name
		hoverRange = &Range{
			Start: Position{Line: lineNum, Character: 0},
			End:   Position{Line: lineNum, Character: colonIdx},
		}
	} else {
		// Hovering over field value
		valueStart := colonIdx + 1
		// Skip leading space after colon
		if valueStart < len(currentLine) && currentLine[valueStart] == ' ' {
			valueStart++
		}
		hoverRange = &Range{
			Start: Position{Line: lineNum, Character: valueStart},
			End:   Position{Line: lineNum, Character: len(currentLine)},
		}
	}

	// Build hover content
	doc := formatFieldDocumentation(field)

	return &Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: doc,
		},
		Range: hoverRange,
	}
}

// getAdmonitionHover returns hover information if the cursor is on an admonition type.
func (s *Server) getAdmonitionHover(line string, lineNum, col int) *Hover {
	// Check if this line matches an admonition pattern
	match := admonitionLineRegex.FindStringSubmatchIndex(line)
	if match == nil {
		return nil
	}

	// match groups:
	// [0:2] full match
	// [2:4] leading whitespace
	// [4:6] marker (!!!, ???, ???+)
	// [6:8] type name

	if len(match) < 8 {
		return nil
	}

	// Get the type name position
	typeStart := match[6]
	typeEnd := match[7]

	// Check if cursor is on the type name
	if col < typeStart || col > typeEnd {
		return nil
	}

	typeName := line[typeStart:typeEnd]

	// Look up the admonition type
	adType := GetAdmonitionType(typeName)
	if adType == nil {
		// Unknown type
		return &Hover{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: "**" + typeName + "**\n\n*Unknown admonition type*",
			},
			Range: &Range{
				Start: Position{Line: lineNum, Character: typeStart},
				End:   Position{Line: lineNum, Character: typeEnd},
			},
		}
	}

	// Build hover content using existing formatter
	doc := formatAdmonitionDocumentation(adType)

	return &Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: doc,
		},
		Range: &Range{
			Start: Position{Line: lineNum, Character: typeStart},
			End:   Position{Line: lineNum, Character: typeEnd},
		},
	}
}
