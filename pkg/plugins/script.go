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

var (
	KindSuperscript = ast.NewNodeKind("Superscript")
	KindSubscript   = ast.NewNodeKind("Subscript")
)

type scriptNode struct {
	ast.BaseInline
	kind ast.NodeKind
}

func (n *scriptNode) Kind() ast.NodeKind { return n.kind }

func (n *scriptNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type scriptParser struct {
	trigger byte
	kind    ast.NodeKind
}

func (p *scriptParser) Trigger() []byte { return []byte{p.trigger} }

func (p *scriptParser) Parse(_ ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	processor := &scriptDelimiterProcessor{trigger: p.trigger, kind: p.kind}
	node := parser.ScanDelimiter(line, before, 1, processor)
	if node == nil || node.OriginalLength != 1 || before == rune(p.trigger) {
		return nil
	}
	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

func (p *scriptParser) CloseBlock(_ ast.Node, _ parser.Context) {}

type scriptDelimiterProcessor struct {
	trigger byte
	kind    ast.NodeKind
}

func (p *scriptDelimiterProcessor) IsDelimiter(b byte) bool { return b == p.trigger }

func (p *scriptDelimiterProcessor) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

func (p *scriptDelimiterProcessor) OnMatch(_ int) ast.Node {
	return &scriptNode{kind: p.kind}
}

type scriptHTMLRenderer struct{ html.Config }

func newScriptHTMLRenderer() renderer.NodeRenderer {
	return &scriptHTMLRenderer{Config: html.NewConfig()}
}

func (r *scriptHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindSuperscript, r.renderSuperscript)
	reg.Register(KindSubscript, r.renderSubscript)
}

func (r *scriptHTMLRenderer) renderSuperscript(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
	return renderScriptTag(w, "sup", entering), nil
}

func (r *scriptHTMLRenderer) renderSubscript(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
	return renderScriptTag(w, "sub", entering), nil
}

func renderScriptTag(w util.BufWriter, tag string, entering bool) ast.WalkStatus {
	if entering {
		_, _ = w.WriteString("<" + tag + ">") //nolint:errcheck // Goldmark owns the writer lifecycle
	} else {
		_, _ = w.WriteString("</" + tag + ">") //nolint:errcheck // Goldmark owns the writer lifecycle
	}
	return ast.WalkContinue
}

// ScriptExtension adds Markdown-native ^superscript^ and ~subscript~ while
// leaving GFM's ~~strikethrough~~ delimiter intact.
type ScriptExtension struct{}

func (e *ScriptExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(&scriptParser{trigger: '^', kind: KindSuperscript}, 490),
		util.Prioritized(&scriptParser{trigger: '~', kind: KindSubscript}, 490),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(newScriptHTMLRenderer(), 490),
	))
}
