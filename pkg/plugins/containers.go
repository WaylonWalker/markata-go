// Package plugins provides lifecycle plugins for markata-go.
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

// KindContainer is the AST node kind for container elements.
var KindContainer = ast.NewNodeKind("Container")

// Container is an AST node representing a container block (::: class).
type Container struct {
	ast.BaseBlock
	// Classes holds the CSS classes for this container
	Classes []string
	// ContainerID holds the optional ID for this container
	ContainerID string
	// ExtraAttrs holds additional attributes
	ExtraAttrs map[string]string
}

// Kind returns the kind of this node.
func (n *Container) Kind() ast.NodeKind {
	return KindContainer
}

// Dump dumps the node for debugging.
func (n *Container) Dump(source []byte, level int) {
	attrs := map[string]string{
		"Classes": strings.Join(n.Classes, " "),
	}
	if n.ContainerID != "" {
		attrs["ID"] = n.ContainerID
	}
	ast.DumpHelper(n, source, level, attrs, nil)
}

// NewContainer creates a new Container node.
func NewContainer(classes []string, id string, attrs map[string]string) *Container {
	return &Container{
		Classes:     classes,
		ContainerID: id,
		ExtraAttrs:  attrs,
	}
}

// containerRegex matches container opening syntax:
// ::: class1 class2 {#id .extra-class key=value}
// Group 1: classes (space-separated)
// Group 2: optional attribute block
var containerRegex = regexp.MustCompile(`^:::\s*(\S.*?)?\s*(?:\{([^}]*)\})?\s*$`)

// containerParser parses ::: container syntax.
type containerParser struct{}

// newContainerParser creates a new container parser.
func newContainerParser() parser.BlockParser {
	return &containerParser{}
}

// Trigger returns the trigger bytes for this parser.
func (p *containerParser) Trigger() []byte {
	return []byte{':'}
}

// Open is called when a new block starts.
func (p *containerParser) Open(_ ast.Node, reader text.Reader, _ parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	lineStr := string(line)

	// Check for opening :::
	if !strings.HasPrefix(lineStr, ":::") {
		return nil, parser.NoChildren
	}

	// Check for closing ::: (just ":::" with optional whitespace)
	trimmed := strings.TrimSpace(lineStr)
	if trimmed == ":::" {
		// This is a closing tag, not an opening
		return nil, parser.NoChildren
	}

	// Parse the opening line
	matches := containerRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}

	// Extract classes
	var classes []string
	if matches[1] != "" {
		// Split by whitespace
		classes = append(classes, strings.Fields(matches[1])...)
	}

	// Extract attributes from {#id .class key=value} block
	var id string
	attrs := make(map[string]string)

	if matches[2] != "" {
		attrStr := matches[2]
		for _, part := range strings.Fields(attrStr) {
			if strings.HasPrefix(part, "#") {
				id = part[1:]
			} else if strings.HasPrefix(part, ".") {
				classes = append(classes, part[1:])
			} else if idx := strings.Index(part, "="); idx > 0 {
				key := part[:idx]
				value := strings.Trim(part[idx+1:], `"'`)
				attrs[key] = value
			}
		}
	}

	// Create the container node
	node := NewContainer(classes, id, attrs)

	// Advance past the opening line
	reader.Advance(segment.Len() - 1)
	// Find the newline
	for {
		line, _ := reader.PeekLine()
		if len(line) == 0 {
			break
		}
		if line[0] == '\n' {
			reader.Advance(1)
			break
		}
		reader.Advance(1)
	}

	return node, parser.HasChildren
}

// Continue is called for each subsequent line.
func (p *containerParser) Continue(_ ast.Node, reader text.Reader, _ parser.Context) parser.State {
	line, segment := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))

	// Check for closing :::
	if lineStr == ":::" {
		reader.Advance(segment.Len())
		return parser.Close
	}

	return parser.Continue | parser.HasChildren
}

// Close is called when the block ends.
func (p *containerParser) Close(_ ast.Node, _ text.Reader, _ parser.Context) {
	// Nothing special to do
}

// CanInterruptParagraph returns true if this parser can interrupt a paragraph.
func (p *containerParser) CanInterruptParagraph() bool {
	return true
}

// CanAcceptIndentedLine returns true if this parser can accept an indented line.
func (p *containerParser) CanAcceptIndentedLine() bool {
	return false
}

// containerHTMLRenderer renders Container nodes to HTML.
type containerHTMLRenderer struct {
	html.Config
}

// newContainerHTMLRenderer creates a new container HTML renderer.
func newContainerHTMLRenderer() renderer.NodeRenderer {
	return &containerHTMLRenderer{
		Config: html.NewConfig(),
	}
}

// RegisterFuncs registers the render functions.
func (r *containerHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindContainer, r.renderContainer)
}

// renderContainer renders a Container node to HTML.
//
//nolint:errcheck // WriteString errors are handled at a higher level in goldmark
func (r *containerHTMLRenderer) renderContainer(w util.BufWriter, _ []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	container, ok := n.(*Container)
	if !ok {
		return ast.WalkContinue, nil
	}

	if entering {
		_, _ = w.WriteString("<div")

		// Add ID if present
		if container.ContainerID != "" {
			_, _ = w.WriteString(` id="`)
			_, _ = w.WriteString(escapeHTMLAttr(container.ContainerID))
			_, _ = w.WriteString(`"`)
		}

		// Add classes if present
		if len(container.Classes) > 0 {
			_, _ = w.WriteString(` class="`)
			for i, class := range container.Classes {
				if i > 0 {
					_, _ = w.WriteString(" ")
				}
				_, _ = w.WriteString(escapeHTMLAttr(class))
			}
			_, _ = w.WriteString(`"`)
		}

		// Add other attributes
		for key, value := range container.ExtraAttrs {
			_, _ = w.WriteString(` `)
			_, _ = w.WriteString(escapeHTMLAttr(key))
			_, _ = w.WriteString(`="`)
			_, _ = w.WriteString(escapeHTMLAttr(value))
			_, _ = w.WriteString(`"`)
		}

		_, _ = w.WriteString(">\n")
	} else {
		_, _ = w.WriteString("</div>\n")
	}

	return ast.WalkContinue, nil
}

// escapeHTMLAttr escapes a string for use in an HTML attribute.
func escapeHTMLAttr(s string) string {
	var b strings.Builder
	for _, c := range s {
		switch c {
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '&':
			b.WriteString("&amp;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&#39;")
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}

// ContainerExtension is a goldmark extension for container syntax.
type ContainerExtension struct{}

// Extend adds the container parser and renderer to goldmark.
func (e *ContainerExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithBlockParsers(
			// Lower priority than admonitions so !!! takes precedence
			util.Prioritized(newContainerParser(), 800),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(newContainerHTMLRenderer(), 500),
		),
	)
}
