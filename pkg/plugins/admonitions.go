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
	"note":      true,
	"warning":   true,
	"tip":       true,
	"important": true,
	"danger":    true,
	"caution":   true,
}

// admonitionRegex matches admonition syntax: !!! type "title"
// Group 1: type (note, warning, tip, important, danger, caution)
// Group 2: optional quoted title
var admonitionRegex = regexp.MustCompile(`^!!!\s+(\w+)(?:\s+"([^"]*)")?$`)

// KindAdmonition is the AST node kind for admonitions.
var KindAdmonition = ast.NewNodeKind("Admonition")

// Admonition is an AST node representing an admonition block.
type Admonition struct {
	ast.BaseBlock
	AdmonitionType  string
	AdmonitionTitle string
}

// Kind returns the kind of this node.
func (n *Admonition) Kind() ast.NodeKind {
	return KindAdmonition
}

// Dump dumps the node for debugging.
func (n *Admonition) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{
		"Type":  n.AdmonitionType,
		"Title": n.AdmonitionTitle,
	}, nil)
}

// NewAdmonition creates a new Admonition node.
func NewAdmonition(adType, title string) *Admonition {
	return &Admonition{
		AdmonitionType:  adType,
		AdmonitionTitle: title,
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
	return []byte{'!'}
}

// Open parses the opening line of an admonition block.
func (p *AdmonitionParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))

	matches := admonitionRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}

	adType := strings.ToLower(matches[1])
	if !admonitionTypes[adType] {
		return nil, parser.NoChildren
	}

	title := matches[2]
	if title == "" {
		// Use capitalized type as default title
		title = strings.ToUpper(adType[:1]) + adType[1:]
	}

	reader.Advance(len(line))

	return NewAdmonition(adType, title), parser.HasChildren
}

// Continue checks if the admonition block continues.
// Admonition content is indented with at least 4 spaces.
func (p *AdmonitionParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()

	// Check if this is a blank line
	if util.IsBlank(line) {
		// Check if the next non-blank line continues the admonition
		// For now, blank lines end the admonition
		return parser.Close
	}

	// Check for proper indentation (at least 4 spaces)
	indent := 0
	for i := 0; i < len(line) && line[i] == ' '; i++ {
		indent++
	}

	if indent < 4 {
		// Not indented enough - close the admonition
		return parser.Close
	}

	// Remove the indentation and advance
	reader.Advance(segment.Stop - segment.Start)
	return parser.Continue | parser.HasChildren
}

// Close is called when the admonition block is closed.
func (p *AdmonitionParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
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
func (r *AdmonitionRenderer) renderAdmonition(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	ad := node.(*Admonition)

	if entering {
		_, _ = w.WriteString(`<div class="admonition `)
		_, _ = w.WriteString(ad.AdmonitionType)
		_, _ = w.WriteString("\">\n")
		_, _ = w.WriteString(`<p class="admonition-title">`)
		_, _ = w.WriteString(ad.AdmonitionTitle)
		_, _ = w.WriteString("</p>\n")
	} else {
		_, _ = w.WriteString("</div>\n")
	}

	return ast.WalkContinue, nil
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
