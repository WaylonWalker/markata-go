package lsp

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// AdmonitionType represents a built-in admonition type with metadata.
type AdmonitionType struct {
	// Name is the admonition type name (e.g., "note", "warning")
	Name string

	// Description provides a brief explanation of when to use this type
	Description string

	// Color is the primary color associated with this type (CSS color name or hex)
	Color string

	// Icon is an optional icon identifier for this type
	Icon string
}

// builtinAdmonitionTypes defines all supported admonition types.
// These match the types defined in pkg/plugins/admonitions.go.
var builtinAdmonitionTypes = []AdmonitionType{
	{
		Name:        "note",
		Description: "Additional information or context",
		Color:       "#448aff",
		Icon:        "pencil",
	},
	{
		Name:        "info",
		Description: "General information",
		Color:       "#00b8d4",
		Icon:        "info-circle",
	},
	{
		Name:        "tip",
		Description: "Helpful suggestions or best practices",
		Color:       "#00bfa5",
		Icon:        "lightbulb",
	},
	{
		Name:        "hint",
		Description: "Subtle guidance or clues",
		Color:       "#00bfa5",
		Icon:        "question-circle",
	},
	{
		Name:        "success",
		Description: "Positive outcomes or confirmations",
		Color:       "#00c853",
		Icon:        "check-circle",
	},
	{
		Name:        "warning",
		Description: "Potential issues or things to be careful about",
		Color:       "#ff9100",
		Icon:        "exclamation-triangle",
	},
	{
		Name:        "caution",
		Description: "Proceed with care",
		Color:       "#ff9100",
		Icon:        "exclamation-circle",
	},
	{
		Name:        "important",
		Description: "Critical information that shouldn't be missed",
		Color:       "#00bfa5",
		Icon:        "exclamation",
	},
	{
		Name:        "danger",
		Description: "Actions that may cause data loss or security issues",
		Color:       "#ff5252",
		Icon:        "bolt",
	},
	{
		Name:        "error",
		Description: "Error conditions or failure states",
		Color:       "#ff5252",
		Icon:        "times-circle",
	},
	{
		Name:        "bug",
		Description: "Known issues or bugs to be aware of",
		Color:       "#f50057",
		Icon:        "bug",
	},
	{
		Name:        "example",
		Description: "Code examples or demonstrations",
		Color:       "#7c4dff",
		Icon:        "code",
	},
	{
		Name:        "quote",
		Description: "Quotations or citations",
		Color:       "#9e9e9e",
		Icon:        "quote-left",
	},
	{
		Name:        "abstract",
		Description: "Summary or overview of content",
		Color:       "#00b0ff",
		Icon:        "clipboard-list",
	},
	{
		Name:        "aside",
		Description: "Side notes or tangential information",
		Color:       "#64dd17",
		Icon:        "comment-alt",
	},
}

// admonitionTypeMap provides fast lookup by name.
// Initialized via initAdmonitionTypeMap.
var admonitionTypeMap = make(map[string]*AdmonitionType, len(builtinAdmonitionTypes))

// titleCaser provides title case conversion.
var titleCaser = cases.Title(language.English)

// initAdmonitionTypeMap initializes the type lookup map.
// Called automatically on first use.
func initAdmonitionTypeMap() {
	if len(admonitionTypeMap) > 0 {
		return
	}
	for i := range builtinAdmonitionTypes {
		admonitionTypeMap[builtinAdmonitionTypes[i].Name] = &builtinAdmonitionTypes[i]
	}
}

// GetAdmonitionType returns the admonition type by name, or nil if not found.
func GetAdmonitionType(name string) *AdmonitionType {
	initAdmonitionTypeMap()
	return admonitionTypeMap[strings.ToLower(name)]
}

// AllAdmonitionTypes returns all built-in admonition types.
func AllAdmonitionTypes() []AdmonitionType {
	return builtinAdmonitionTypes
}

// admonitionMarkerRegex matches admonition markers at the start of a line.
// Matches: !!!, ???, ???+ followed by optional space and partial type.
var admonitionMarkerRegex = regexp.MustCompile(`^(\?{3}\+?|!!!)(?:\s+(\w*))?$`)

// AdmonitionContext contains information about the admonition context at a position.
type AdmonitionContext struct {
	// Marker is the admonition marker (!!!, ???, ???+)
	Marker string

	// TypePrefix is the partial type that has been typed (may be empty)
	TypePrefix string

	// MarkerStart is the column where the marker starts
	MarkerStart int

	// TypeStart is the column where the type starts (after marker + space)
	TypeStart int
}

// getAdmonitionContext checks if the cursor is in an admonition context and returns details.
// An admonition context is when the cursor is after an admonition marker (!!!, ???, ???+)
// at the start of a line.
// Returns (context, isInAdmonitionContext).
func getAdmonitionContext(line string, col int) (*AdmonitionContext, bool) {
	if col > len(line) {
		col = len(line)
	}

	textBeforeCursor := line[:col]

	// Check if the line starts with an admonition marker
	trimmed := strings.TrimLeft(textBeforeCursor, " \t")
	leadingSpaces := len(textBeforeCursor) - len(trimmed)

	match := admonitionMarkerRegex.FindStringSubmatch(trimmed)
	if match == nil {
		return nil, false
	}

	marker := match[1]
	typePrefix := ""
	if len(match) > 2 {
		typePrefix = match[2]
	}

	// Calculate positions
	markerStart := leadingSpaces
	typeStart := markerStart + len(marker) + 1 // +1 for the space after marker

	return &AdmonitionContext{
		Marker:      marker,
		TypePrefix:  typePrefix,
		MarkerStart: markerStart,
		TypeStart:   typeStart,
	}, true
}

// getAdmonitionCompletions returns completion items for admonition types.
func getAdmonitionCompletions(ctx *AdmonitionContext, params CompletionParams) []CompletionItem {
	prefix := strings.ToLower(ctx.TypePrefix)

	// Filter and sort types
	var matchingTypes []AdmonitionType
	for _, adType := range builtinAdmonitionTypes {
		if prefix == "" || strings.HasPrefix(adType.Name, prefix) {
			matchingTypes = append(matchingTypes, adType)
		}
	}

	// Sort alphabetically
	sort.Slice(matchingTypes, func(i, j int) bool {
		return matchingTypes[i].Name < matchingTypes[j].Name
	})

	// Pre-allocate items slice
	items := make([]CompletionItem, 0, len(matchingTypes))

	for i, adType := range matchingTypes {
		// Build documentation with color info
		titleName := titleCaser.String(adType.Name)
		docValue := fmt.Sprintf("**%s**\n\n%s\n\n*Color: %s*",
			titleName,
			adType.Description,
			adType.Color,
		)

		// Create snippet with placeholder for title
		// e.g., "note \"${1:Title}\""
		snippetText := fmt.Sprintf("%s \"${1:%s}\"", adType.Name, titleName)

		item := CompletionItem{
			Label:  adType.Name,
			Kind:   CompletionItemKindKeyword,
			Detail: adType.Description,
			Documentation: &MarkupContent{
				Kind:  "markdown",
				Value: docValue,
			},
			InsertText:       snippetText,
			InsertTextFormat: InsertTextFormatSnippet,
			FilterText:       adType.Name,
			SortText:         fmt.Sprintf("%05d", i), // Preserve alphabetical order
		}

		// If there's a prefix, use TextEdit to replace it
		if ctx.TypePrefix != "" {
			item.TextEdit = &TextEdit{
				Range: Range{
					Start: Position{Line: params.Position.Line, Character: ctx.TypeStart},
					End:   Position{Line: params.Position.Line, Character: params.Position.Character},
				},
				NewText: snippetText,
			}
		}

		items = append(items, item)
	}

	return items
}

// formatAdmonitionDocumentation formats admonition type info for display.
func formatAdmonitionDocumentation(adType *AdmonitionType) string {
	var sb strings.Builder

	sb.WriteString("**")
	sb.WriteString(titleCaser.String(adType.Name))
	sb.WriteString("**\n\n")

	sb.WriteString(adType.Description)
	sb.WriteString("\n\n")

	sb.WriteString("*Color: ")
	sb.WriteString(adType.Color)
	sb.WriteString("*\n\n")

	sb.WriteString("**Usage:**\n```markdown\n")
	sb.WriteString(fmt.Sprintf("!!! %s \"Optional Title\"\n    Content goes here.\n```", adType.Name))

	return sb.String()
}
