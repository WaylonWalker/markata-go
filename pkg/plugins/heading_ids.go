package plugins

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// HeadingIDTransformer regenerates auto heading IDs from the heading's
// visible text. goldmark's auto ID generator slugs the raw source line, so
// `## Recent [TIL](/til/)` becomes `recent-tiltil` because the link
// destination leaks into the ID. Headings with an explicit `{#id}` attribute
// are left untouched.
type HeadingIDTransformer struct{}

// Transform implements parser.ASTTransformer.
func (t *HeadingIDTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		idVal, ok := h.AttributeString("id")
		if !ok {
			return ast.WalkContinue, nil
		}
		id, ok := idVal.([]byte)
		if !ok {
			return ast.WalkContinue, nil
		}

		// Only rewrite IDs goldmark derived from the raw line; anything else
		// was set explicitly by the author.
		if !bytes.Equal(id, goldmarkHeadingSlug(headingRawLine(h, source))) {
			return ast.WalkContinue, nil
		}

		clean := headingPlainText(h, source)
		if bytes.Equal(goldmarkHeadingSlug(clean), id) {
			return ast.WalkContinue, nil
		}
		h.SetAttribute([]byte("id"), pc.IDs().Generate(clean, ast.KindHeading))
		return ast.WalkContinue, nil
	})
}

func headingRawLine(h *ast.Heading, source []byte) []byte {
	lines := h.Lines()
	if lines.Len() == 0 {
		return nil
	}
	seg := lines.At(lines.Len() - 1)
	return seg.Value(source)
}

// headingPlainText concatenates the text of all inline descendants.
func headingPlainText(h *ast.Heading, source []byte) []byte {
	var buf bytes.Buffer
	_ = ast.Walk(h, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Text:
			buf.Write(v.Segment.Value(source))
			if v.SoftLineBreak() || v.HardLineBreak() {
				buf.WriteByte(' ')
			}
		case *ast.String:
			buf.Write(v.Value)
		}
		return ast.WalkContinue, nil
	})
	return buf.Bytes()
}

// goldmarkHeadingSlug mirrors goldmark's default IDs.Generate for headings,
// minus collision suffixes.
func goldmarkHeadingSlug(value []byte) []byte {
	value = util.TrimLeftSpace(value)
	value = util.TrimRightSpace(value)
	result := make([]byte, 0, len(value))
	for i := 0; i < len(value); {
		v := value[i]
		l := util.UTF8Len(v)
		i += int(l)
		if l != 1 {
			continue
		}
		switch {
		case util.IsAlphaNumeric(v):
			if 'A' <= v && v <= 'Z' {
				v += 'a' - 'A'
			}
			result = append(result, v)
		case util.IsSpace(v) || v == '-' || v == '_':
			result = append(result, '-')
		}
	}
	if len(result) == 0 {
		return []byte("heading")
	}
	return result
}
