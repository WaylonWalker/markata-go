package plugins

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Supported admonition types
var admonitionTypes = map[string]bool{
	"note":       true,
	"info":       true,
	"tip":        true,
	"hint":       true,
	"success":    true,
	"warning":    true,
	"caution":    true,
	"important":  true,
	"danger":     true,
	"error":      true,
	"bug":        true,
	"example":    true,
	"quote":      true,
	"abstract":   true,
	"aside":      true,
	"seealso":    true,
	"reminder":   true,
	"attention":  true,
	"todo":       true,
	"settings":   true,
	"vsplit":     true,
	"chat":       true,
	"chat-reply": true,
}

// admonitionRegex matches admonition syntax:
// - !!! type "title" (standard with quoted title)
// - !!! type title text (standard with unquoted title)
// - !!! aside left "title" (aside positioned left)
// - !!! aside right "title" (aside positioned right, default)
// - !!! aside inline "title" (Material for MkDocs compat: left)
// - !!! aside inline end "title" (Material for MkDocs compat: right)
// - ??? type "title" (collapsible, collapsed by default)
// - ???+ type "title" (collapsible, expanded by default)
// Group 1: marker (!!!, ???, ???+)
// Group 2: type (allows hyphens for types like chat-reply)
// Group 3: optional modifiers (for aside: left, right, inline, inline end)
// Group 4: optional quoted title
// Group 5: optional unquoted title (everything after type/modifiers if no quotes)
var admonitionRegex = regexp.MustCompile(`^(\?\?\?\+?|!!!)\s+([\w-]+)(?:\s+(left|right|inline(?:\s+end)?))?\s*(?:"([^"]*)"|(.*))?$`)

// KindAdmonition is the AST node kind for admonitions.
var KindAdmonition = ast.NewNodeKind("Admonition")

// Admonition is an AST node representing an admonition block.
type Admonition struct {
	ast.BaseBlock
	AdmonitionType  string
	AdmonitionTitle string
	Collapsible     bool   // true if ??? or ???+ syntax
	DefaultOpen     bool   // true if ???+ (expanded by default)
	Position        string // "left", "right", or "" (for aside type; "" means right/default)
}

// Kind returns the kind of this node.
func (n *Admonition) Kind() ast.NodeKind {
	return KindAdmonition
}

// Dump dumps the node for debugging.
func (n *Admonition) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{
		"Type":        n.AdmonitionType,
		"Title":       n.AdmonitionTitle,
		"Collapsible": boolToString(n.Collapsible),
		"DefaultOpen": boolToString(n.DefaultOpen),
		"Position":    n.Position,
	}, nil)
}

func boolToString(b bool) string {
	if b {
		return BoolTrue
	}
	return "false"
}

// NewAdmonition creates a new Admonition node.
func NewAdmonition(adType, title string, collapsible, defaultOpen bool, position string) *Admonition {
	return &Admonition{
		AdmonitionType:  adType,
		AdmonitionTitle: title,
		Collapsible:     collapsible,
		DefaultOpen:     defaultOpen,
		Position:        position,
	}
}

// AdmonitionParser is a block parser for admonitions.
type AdmonitionParser struct{}

// NewAdmonitionParser creates a new AdmonitionParser.
func NewAdmonitionParser() *AdmonitionParser {
	return &AdmonitionParser{}
}

// Trigger returns the characters that trigger this parser.
func (p *AdmonitionParser) Trigger() []byte {
	return []byte{'!', '?'}
}

// Open parses the opening line of an admonition block.
func (p *AdmonitionParser) Open(_ ast.Node, reader text.Reader, _ parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))

	matches := admonitionRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}

	marker := matches[1]
	adType := strings.ToLower(matches[2])
	modifier := strings.ToLower(matches[3])
	quotedTitle := matches[4]
	unquotedTitle := strings.TrimSpace(matches[5])

	if !admonitionTypes[adType] {
		return nil, parser.NoChildren
	}

	// Determine collapsible state from marker
	collapsible := strings.HasPrefix(marker, "???")
	defaultOpen := marker == "???+"

	// Parse position modifiers for aside type
	// Supports: left, right, inline (=left), inline end (=right)
	position := ""
	if adType == AdmonitionTypeAside && modifier != "" {
		switch modifier {
		case PositionLeft, "inline":
			position = PositionLeft
		case "right", "inline end":
			position = "right"
		}
	}

	// Use quoted title if present, otherwise use unquoted title
	title := quotedTitle
	if title == "" && unquotedTitle != "" {
		title = unquotedTitle
	}

	// Set default title if not provided
	if title == "" {
		if adType == AdmonitionTypeAside {
			// Aside has no default title per spec
			title = ""
		} else {
			// Use capitalized type as default title
			title = strings.ToUpper(adType[:1]) + adType[1:]
		}
	}

	reader.Advance(len(line))

	return NewAdmonition(adType, title, collapsible, defaultOpen, position), parser.HasChildren
}

// Continue checks if the admonition block continues.
// Admonition content is indented with at least 4 spaces.
// Blank lines are allowed within admonitions; the block closes when a
// non-blank line with fewer than 4 spaces of indentation is encountered.
func (p *AdmonitionParser) Continue(_ ast.Node, reader text.Reader, _ parser.Context) parser.State {
	line, _ := reader.PeekLine()

	// Blank lines are allowed inside admonitions (paragraph separators).
	// We let them through without advancing the reader; goldmark handles
	// them as part of the block's child content.
	if util.IsBlank(line) {
		return parser.Continue | parser.HasChildren
	}

	// Non-blank line: check for proper indentation (at least 4 spaces).
	indent, _ := util.IndentWidth(line, reader.LineOffset())
	if indent < 4 {
		// Not indented enough - close the admonition.
		return parser.Close
	}

	pos, padding := util.IndentPosition(line, reader.LineOffset(), 4)
	if pos < 0 {
		return parser.Close
	}

	// Advance past the 4-space indent so child parsers see unindented content.
	reader.AdvanceAndSetPadding(pos, padding)
	return parser.Continue | parser.HasChildren
}

// Close is called when the admonition block is closed.
func (p *AdmonitionParser) Close(_ ast.Node, _ text.Reader, _ parser.Context) {
	// Nothing to do
}

// CanInterruptParagraph returns true if the parser can interrupt a paragraph.
func (p *AdmonitionParser) CanInterruptParagraph() bool {
	return true
}

// CanAcceptIndentedLine returns true if the parser can accept indented lines.
func (p *AdmonitionParser) CanAcceptIndentedLine() bool {
	return false
}

// AdmonitionRenderer renders Admonition nodes to HTML.
type AdmonitionRenderer struct {
	html.Config
}

// NewAdmonitionRenderer creates a new AdmonitionRenderer.
func NewAdmonitionRenderer() *AdmonitionRenderer {
	return &AdmonitionRenderer{}
}

// RegisterFuncs registers the render functions for Admonition nodes.
func (r *AdmonitionRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindAdmonition, r.renderAdmonition)
}

// renderAdmonition renders an Admonition node to HTML.
func (r *AdmonitionRenderer) renderAdmonition(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	ad := node.(*Admonition) //nolint:errcheck // type assertion is safe here

	if entering {
		r.renderEntering(w, ad)
	} else {
		r.renderExiting(w, ad)
	}

	return ast.WalkContinue, nil
}

// renderEntering renders the opening tags for an admonition.
//
//nolint:errcheck // WriteString errors are handled at a higher level in goldmark
func (r *AdmonitionRenderer) renderEntering(w util.BufWriter, ad *Admonition) {
	switch {
	case ad.Collapsible:
		// Collapsible admonition uses <details>/<summary>
		_, _ = w.WriteString("<details class=\"admonition ")
		_, _ = w.WriteString(ad.AdmonitionType)
		_, _ = w.WriteString("\"")
		if ad.DefaultOpen {
			_, _ = w.WriteString(" open")
		}
		_, _ = w.WriteString(">\n")
		_, _ = w.WriteString("<summary class=\"admonition-title\">")
		_, _ = w.WriteString(ad.AdmonitionTitle)
		_, _ = w.WriteString("</summary>\n")
	case ad.AdmonitionType == AdmonitionTypeAside:
		// Aside uses <aside> element with position classes
		// Default position is right (margin note style, per Tufte convention)
		_, _ = w.WriteString("<aside class=\"admonition aside")
		if ad.Position == PositionLeft {
			_, _ = w.WriteString(" aside-left")
		} else {
			// Default to right (including when Position is "" or "right")
			_, _ = w.WriteString(" aside-right")
		}
		_, _ = w.WriteString("\">\n")
		if ad.AdmonitionTitle != "" {
			_, _ = w.WriteString("<p class=\"admonition-title\">")
			_, _ = w.WriteString(ad.AdmonitionTitle)
			_, _ = w.WriteString("</p>\n")
		}
	default:
		// Standard admonition uses <div>
		_, _ = w.WriteString("<div class=\"admonition ")
		_, _ = w.WriteString(ad.AdmonitionType)
		_, _ = w.WriteString("\">\n")
		_, _ = w.WriteString("<p class=\"admonition-title\">")
		_, _ = w.WriteString(ad.AdmonitionTitle)
		_, _ = w.WriteString("</p>\n")
	}
}

// renderExiting renders the closing tags for an admonition.
//
//nolint:errcheck // WriteString errors are handled at a higher level in goldmark
func (r *AdmonitionRenderer) renderExiting(w util.BufWriter, ad *Admonition) {
	switch {
	case ad.Collapsible:
		_, _ = w.WriteString("</details>\n")
	case ad.AdmonitionType == AdmonitionTypeAside:
		_, _ = w.WriteString("</aside>\n")
	default:
		_, _ = w.WriteString("</div>\n")
	}
}

// AdmonitionExtension is a goldmark extension for admonitions.
type AdmonitionExtension struct{}

// Extend adds the admonition parser and renderer to goldmark.
func (e *AdmonitionExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithBlockParsers(
			util.Prioritized(NewAdmonitionParser(), 100),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(NewAdmonitionRenderer(), 100),
		),
	)
}
