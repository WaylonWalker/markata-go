// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// InlineAttributeTransformer is a Goldmark AST transformer that handles
// inline attribute syntax for images and links.
//
// This extends Goldmark's built-in attribute support (which only works for
// block-level elements like headings) to also work with inline elements.
//
// Supported syntax:
//   - {.classname} - adds a CSS class
//   - {#idname} - adds an id attribute
//   - {.class1 .class2} - multiple classes
//   - {#id .class} - id and class combined
//
// Example usage:
//
//	![alt text](image.webp){.more-cinematic}
//	[link text](url){.external-link}
type InlineAttributeTransformer struct{}

// inlineAttrPattern matches attribute syntax at the start of text: {.class}, {#id}
// This pattern is anchored to the start of the accumulated text to match attributes
// that immediately follow an inline element.
var inlineAttrPattern = regexp.MustCompile(`^\{([^}]+)\}`)

// Transform implements parser.ASTTransformer.
// It walks the AST looking for inline elements (images, links) followed by
// text nodes containing attribute syntax, and applies those attributes to
// the preceding element.
func (t *InlineAttributeTransformer) Transform(node *ast.Document, reader text.Reader, _ parser.Context) {
	//nolint:errcheck // ast.Walk error is always nil when callback returns nil error
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		// Look for paragraphs containing inline elements followed by attribute syntax
		if para, ok := n.(*ast.Paragraph); ok {
			t.processInlineAttributes(para, reader)
		}
		return ast.WalkContinue, nil
	})
}

// processInlineAttributes processes a paragraph's children to find and apply
// inline attributes to images and links.
//
// Note: The Linkify extension can split text nodes around periods (e.g., ".bordered"
// might look like a domain suffix), so this function accumulates consecutive text
// nodes before matching the attribute pattern.
func (t *InlineAttributeTransformer) processInlineAttributes(para *ast.Paragraph, reader text.Reader) {
	var lastInlineElement ast.Node
	var textContent strings.Builder
	var textNodes []*ast.Text

	for child := para.FirstChild(); child != nil; child = child.NextSibling() {
		switch node := child.(type) {
		case *ast.Image, *ast.Link:
			// If we have accumulated text, process it first
			t.applyAttributesIfMatch(para, lastInlineElement, textContent.String(), textNodes)
			textContent.Reset()
			textNodes = nil
			lastInlineElement = node

		case *ast.Text:
			// Accumulate text content (may be split by Linkify extension)
			content := string(node.Segment.Value(reader.Source()))
			textContent.WriteString(content)
			textNodes = append(textNodes, node)

		default:
			// Process accumulated text before resetting
			t.applyAttributesIfMatch(para, lastInlineElement, textContent.String(), textNodes)
			textContent.Reset()
			textNodes = nil
			lastInlineElement = nil
		}
	}

	// Handle any remaining text after the last inline element
	t.applyAttributesIfMatch(para, lastInlineElement, textContent.String(), textNodes)
}

// applyAttributesIfMatch checks if the accumulated text matches the attribute pattern
// and applies the attributes to the inline element if so.
func (t *InlineAttributeTransformer) applyAttributesIfMatch(para *ast.Paragraph, element ast.Node, attrText string, textNodes []*ast.Text) {
	if element == nil || attrText == "" {
		return
	}

	matches := inlineAttrPattern.FindStringSubmatch(attrText)
	if len(matches) < 2 {
		return
	}

	// Parse and apply attributes
	attrs := parseInlineAttributes(matches[1])
	for k, v := range attrs {
		element.SetAttributeString(k, []byte(v))
	}

	// Check if there's remaining text after the attribute syntax
	remaining := strings.TrimPrefix(attrText, matches[0])
	if remaining == "" {
		// Remove all accumulated text nodes
		for _, tn := range textNodes {
			para.RemoveChild(para, tn)
		}
	}
	// Note: If there's remaining text, we leave the text nodes as-is
	// This could be improved to modify the text content, but for now
	// we handle the common case where attributes are at the end
}

// parseInlineAttributes parses the content inside braces and returns a map
// of attribute names to values.
//
// Supported formats:
//   - .classname -> class="classname"
//   - #idname -> id="idname"
//   - .class1 .class2 -> class="class1 class2"
//   - key=value -> key="value"
//   - key="quoted value" -> key="quoted value"
func parseInlineAttributes(attrStr string) map[string]string {
	attrs := make(map[string]string)
	parts := strings.Fields(attrStr)
	var classes []string

	for _, part := range parts {
		switch {
		case strings.HasPrefix(part, "."):
			// CSS class
			classes = append(classes, strings.TrimPrefix(part, "."))
		case strings.HasPrefix(part, "#"):
			// ID attribute
			attrs["id"] = strings.TrimPrefix(part, "#")
		case strings.Contains(part, "="):
			// Key=value attribute
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				// Remove quotes if present
				val := strings.Trim(kv[1], `"'`)
				attrs[kv[0]] = val
			}
		}
	}

	if len(classes) > 0 {
		attrs["class"] = strings.Join(classes, " ")
	}

	return attrs
}
