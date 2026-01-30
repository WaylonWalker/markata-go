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

// markParser parses ==text== syntax for highlighted text.
// This uses a simple greedy approach: find == opening, then find == closing.
type markParser struct{}

// newMarkParser creates a new mark parser.
func newMarkParser() parser.InlineParser {
	return &markParser{}
}

// Trigger returns the trigger bytes for this parser.
func (p *markParser) Trigger() []byte {
	return []byte{'='}
}

// Parse parses the ==text== syntax.
func (p *markParser) Parse(_ ast.Node, block text.Reader, _ parser.Context) ast.Node {
	line, segment := block.PeekLine()

	// Must start with ==
	if len(line) < 4 || line[0] != '=' || line[1] != '=' {
		return nil
	}

	// Don't match === (could be other syntax or just decoration)
	if len(line) > 2 && line[2] == '=' {
		return nil
	}

	// Find the closing ==
	// Start searching after the opening ==
	content := line[2:]
	closeIdx := -1

	for i := 0; i < len(content)-1; i++ {
		if content[i] == '=' && content[i+1] == '=' {
			// Found closing ==, but make sure we have content
			if i > 0 {
				closeIdx = i
				break
			}
		}
	}

	if closeIdx == -1 {
		return nil
	}

	// Extract the content between == and ==
	markedContent := content[:closeIdx]
	if len(markedContent) == 0 {
		return nil
	}

	// Create the mark node
	mark := NewMark()

	// Add the content as a text child
	// The content segment starts at: segment.Start + 2 (after opening ==)
	// and ends at: segment.Start + 2 + closeIdx
	contentSegment := text.NewSegment(segment.Start+2, segment.Start+2+closeIdx)
	textNode := ast.NewTextSegment(contentSegment)
	mark.AppendChild(mark, textNode)

	// Advance past the entire ==content== (opening + content + closing)
	totalLen := 2 + closeIdx + 2
	block.Advance(totalLen)

	return mark
}

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
		_, _ = w.WriteString("<mark>")
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
