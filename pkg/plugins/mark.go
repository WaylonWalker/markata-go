// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// KindMark is the AST node kind for mark (highlight) elements.
var KindMark = ast.NewNodeKind("Mark")

// Mark is an AST node representing highlighted text (==text==).
type Mark struct {
	ast.BaseInline
}

// Kind returns the kind of this node.
func (n *Mark) Kind() ast.NodeKind {
	return KindMark
}

// Dump dumps the node for debugging.
func (n *Mark) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// NewMark creates a new Mark node.
func NewMark() *Mark {
	return &Mark{}
}

// markParser parses ==text== syntax for highlighted text as a delimiter pair.
// Using Goldmark's delimiter processing lets the normal inline parsers build
// nested children (for example, ==**bold**==) instead of treating the marked
// source as one literal text node.
type markParser struct{}

type markDelimiterProcessor struct{}

func (p *markDelimiterProcessor) IsDelimiter(b byte) bool { return b == '=' }

func (p *markDelimiterProcessor) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

func (p *markDelimiterProcessor) OnMatch(_ int) ast.Node { return NewMark() }

var defaultMarkDelimiterProcessor = &markDelimiterProcessor{}

// newMarkParser creates a new mark parser.
func newMarkParser() parser.InlineParser {
	return &markParser{}
}

// Trigger returns the trigger bytes for this parser.
func (p *markParser) Trigger() []byte {
	return []byte{'='}
}

// Parse parses the ==text== syntax.
func (p *markParser) Parse(_ ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 2, defaultMarkDelimiterProcessor)
	if node == nil || node.OriginalLength != 2 || before == '=' {
		return nil
	}
	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

func (p *markParser) CloseBlock(_ ast.Node, _ parser.Context) {}

// markHTMLRenderer renders Mark nodes to HTML.
type markHTMLRenderer struct {
	html.Config
}

// newMarkHTMLRenderer creates a new mark HTML renderer.
func newMarkHTMLRenderer() renderer.NodeRenderer {
	return &markHTMLRenderer{
		Config: html.NewConfig(),
	}
}

// RegisterFuncs registers the render functions.
func (r *markHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindMark, r.renderMark)
}

// renderMark renders a Mark node to HTML.
//
//nolint:errcheck // WriteString errors are handled at a higher level in goldmark
func (r *markHTMLRenderer) renderMark(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		// Keep Markdown highlight pixels owned by the canonical presentation
		// contract even when a component library installs warning-style <mark>
		// defaults with a stronger cascade.
		_, _ = w.WriteString(`<mark style="background-color:var(--color-highlight)!important;color:var(--color-highlight-text)!important">`)
	} else {
		_, _ = w.WriteString("</mark>")
	}
	return ast.WalkContinue, nil
}

// MarkExtension is a goldmark extension for mark (highlight) syntax.
type MarkExtension struct{}

// Extend adds the mark parser and renderer to goldmark.
func (e *MarkExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithInlineParsers(
			util.Prioritized(newMarkParser(), 500),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(newMarkHTMLRenderer(), 500),
		),
	)
}
