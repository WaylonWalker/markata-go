// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
	extensionast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// AttributeTransformer is a Goldmark AST transformer that handles
// attribute syntax for block and inline elements.
//
// This extends Goldmark's built-in attribute support (which only works for
// block-level elements like headings) to also work with additional blocks
// and inline elements.
//
// Supported syntax:
//   - {.classname} - adds a CSS class
//   - {#idname} - adds an id attribute
//   - {.class1 .class2} - multiple classes
//   - {#id .class} - id and class combined
//
// Example usage:
//
//	Paragraph text
//	{.lead}
//
//	![alt text](image.webp){.more-cinematic}
//	[link text](url){.external-link}
//	*emphasis*{.highlight}
//	`code`{.inline-code}
type AttributeTransformer struct{}

// inlineAttrPattern matches attribute syntax at the start of text: {.class}, {#id}
// This pattern is anchored to the start of the accumulated text to match attributes
// that immediately follow an inline element.
var inlineAttrPattern = regexp.MustCompile(`^\{[^}]+\}`)

// attributeOnlyPattern matches a line that only contains attribute syntax.
var attributeOnlyPattern = regexp.MustCompile(`^\s*\{[^}]+\}\s*$`)

// Transform implements parser.ASTTransformer.
// It walks the AST looking for attribute syntax and applies it to the
// appropriate block or inline element.
func (t *AttributeTransformer) Transform(node *ast.Document, reader text.Reader, _ parser.Context) {
	for n := node.FirstChild(); n != nil; n = n.NextSibling() {
		if list, ok := n.(*ast.List); ok {
			t.processListAttributes(list, reader)
		}
	}

	//nolint:errcheck // ast.Walk error is always nil when callback returns nil error
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch typed := n.(type) {
		case *ast.Paragraph:
			t.processInlineAttributes(typed, reader)
			if t.applyAttributeOnlyParagraph(typed, reader) {
				return ast.WalkContinue, nil
			}
			t.applyTrailingAttributesToContainer(typed, reader)
		case *extensionast.TableCell:
			if t.applyAttributeOnlyParagraphForTableCell(typed, reader) {
				return ast.WalkContinue, nil
			}
			t.processInlineAttributesInContainer(typed, reader)
			t.applyTrailingAttributesToContainer(typed, reader)
		case *ast.ListItem:
			if t.applyAttributeOnlyParagraphForListItem(typed, reader) {
				return ast.WalkContinue, nil
			}
			if t.applyInlineAttributesForListItem(typed, reader) {
				return ast.WalkContinue, nil
			}
			t.processInlineAttributesInContainer(typed, reader)
			t.applyTrailingAttributesToContainer(typed, reader)
		}
		return ast.WalkContinue, nil
	})
}

func (t *AttributeTransformer) processListAttributes(list *ast.List, reader text.Reader) {
	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		li, ok := item.(*ast.ListItem)
		if !ok {
			continue
		}
		_ = t.applyAttributeOnlyParagraphForListItem(li, reader)
		_ = t.applyInlineAttributesForListItem(li, reader)
	}
}

// processInlineAttributes processes a paragraph's children to find and apply
// inline attributes to inline elements.
//
// Note: The Linkify extension can split text nodes around periods (e.g., ".bordered"
// might look like a domain suffix), so this function accumulates consecutive text
// nodes before matching the attribute pattern.
func (t *AttributeTransformer) processInlineAttributes(para *ast.Paragraph, reader text.Reader) {
	t.processInlineAttributesInContainer(para, reader)
}

// processInlineAttributesInContainer processes a node's children to find and apply
// inline attributes to inline elements.
func (t *AttributeTransformer) processInlineAttributesInContainer(container ast.Node, reader text.Reader) {
	var lastInlineElement ast.Node
	var textContent strings.Builder
	var textNodes []*ast.Text
	var stringNodes []*ast.String

	for child := container.FirstChild(); child != nil; child = child.NextSibling() {
		switch node := child.(type) {
		case *ast.Image, *ast.Link, *ast.Emphasis, *ast.CodeSpan, *ast.AutoLink, *extensionast.Strikethrough:
			// If we have accumulated text, process it first
			t.applyAttributesIfMatch(container, lastInlineElement, textContent.String(), textNodes, stringNodes)
			textContent.Reset()
			textNodes = nil
			stringNodes = nil
			lastInlineElement = node

		case *ast.Text:
			if lastInlineElement != nil {
				// Accumulate text content (may be split by Linkify extension)
				content := string(node.Segment.Value(reader.Source()))
				textContent.WriteString(content)
				if node.SoftLineBreak() || node.HardLineBreak() {
					textContent.WriteString("\n")
				}
				textNodes = append(textNodes, node)
				break
			}

			// No inline element to attach attributes to; reset accumulator
			textContent.Reset()
			textNodes = nil
			stringNodes = nil

		case *ast.String:
			if lastInlineElement == nil {
				textContent.Reset()
				textNodes = nil
				stringNodes = nil
				break
			}
			textContent.WriteString(string(node.Value))
			stringNodes = append(stringNodes, node)

		default:
			// Process accumulated text before resetting
			t.applyAttributesIfMatch(container, lastInlineElement, textContent.String(), textNodes, stringNodes)
			textContent.Reset()
			textNodes = nil
			stringNodes = nil
			lastInlineElement = nil
		}
	}

	// Handle any remaining text after the last inline element
	t.applyAttributesIfMatch(container, lastInlineElement, textContent.String(), textNodes, stringNodes)
}

// applyAttributesIfMatch checks if the accumulated text matches the attribute pattern
// and applies the attributes to the inline element if so.
func (t *AttributeTransformer) applyAttributesIfMatch(container, element ast.Node, attrText string, textNodes []*ast.Text, stringNodes []*ast.String) {
	if element == nil || attrText == "" {
		return
	}

	if !inlineAttrPattern.MatchString(attrText) {
		return
	}
	attrs, remaining, ok := parseLeadingAttributes(attrText)
	if !ok {
		return
	}

	// Parse and apply attributes
	applyParsedAttributes(element, attrs)

	// Check if there's remaining text after the attribute syntax
	if remaining == "" {
		// Remove all accumulated text nodes
		for _, tn := range textNodes {
			container.RemoveChild(container, tn)
		}
		for _, sn := range stringNodes {
			container.RemoveChild(container, sn)
		}
		return
	}

	replaceTextNodes(container, remaining, textNodes, stringNodes)
}

func (t *AttributeTransformer) applyAttributeOnlyParagraph(para *ast.Paragraph, reader text.Reader) bool {
	textValue, _, _, ok := paragraphTextOnly(para, reader)
	if !ok {
		return false
	}
	if !attributeOnlyPattern.MatchString(textValue) {
		return false
	}
	attrs, ok := parseAttributeOnly(textValue)
	if !ok {
		return false
	}

	parent := para.Parent()
	if parent == nil {
		return false
	}

	var target ast.Node
	switch parent.(type) {
	case *ast.Blockquote:
		if prev := para.PreviousSibling(); prev != nil {
			target = prev
		} else {
			target = parent
		}
	case *extensionast.TableCell:
		if prev := para.PreviousSibling(); prev != nil {
			target = prev
		} else {
			target = parent
		}
	default:
		target = para.PreviousSibling()
	}

	if target == nil {
		return false
	}

	applyParsedAttributes(target, attrs)
	parent.RemoveChild(parent, para)

	return true
}

func (t *AttributeTransformer) applyAttributeOnlyParagraphForListItem(item *ast.ListItem, reader text.Reader) bool {
	last := item.LastChild()
	container, ok := listItemTextContainer(last)
	if !ok {
		return false
	}
	textValue, textNodes, stringNodes, ok := blockTextOnly(container, reader)
	if !ok {
		return false
	}
	lines := strings.Split(strings.TrimRight(textValue, "\n\r"), "\n")
	if len(lines) == 0 {
		return false
	}
	lastLine := strings.TrimSpace(lines[len(lines)-1])
	attrs, ok := parseAttributeOnly(lastLine)
	if !ok {
		attrs, lastLine, ok = parseTrailingAttributeSuffix(lastLine)
		if !ok {
			return false
		}
		lines[len(lines)-1] = lastLine
	} else {
		lines = lines[:len(lines)-1]
	}

	applyParsedAttributes(item, attrs)
	remaining := strings.TrimRight(strings.Join(lines, "\n"), " \t\n\r")
	if remaining == "" {
		item.RemoveChild(item, container)
		return true
	}
	replaceTextNodes(container, remaining, textNodes, stringNodes)
	return true
}

func (t *AttributeTransformer) applyInlineAttributesForListItem(item *ast.ListItem, reader text.Reader) bool {
	first := item.FirstChild()
	container, ok := listItemTextContainer(first)
	if !ok {
		return false
	}
	if para, ok := container.(*ast.Paragraph); ok {
		textValue, textNodes, stringNodes, ok := paragraphTextOnly(para, reader)
		if !ok {
			return false
		}
		attrs, remaining, ok := parseTrailingAttributeSuffix(textValue)
		if !ok {
			return false
		}
		applyParsedAttributes(item, attrs)
		replaceTextNodes(container, remaining, textNodes, stringNodes)
		return true
	}

	textValue, textNodes, stringNodes, ok := blockTextOnly(container, reader)
	if !ok {
		return false
	}
	attrs, remaining, ok := parseTrailingAttributeSuffix(textValue)
	if !ok {
		return false
	}
	applyParsedAttributes(item, attrs)
	replaceTextNodes(container, remaining, textNodes, stringNodes)
	return true
}

func parseTrailingAttributeSuffix(line string) (parser.Attributes, string, bool) {
	trimmed := strings.TrimRight(line, " \t\n\r")
	end := strings.LastIndex(trimmed, "}")
	if end == -1 || end != len(trimmed)-1 {
		return nil, "", false
	}
	start := strings.LastIndex(trimmed, "{")
	if start == -1 || start > end {
		return nil, "", false
	}
	block := trimmed[start : end+1]
	attrs, ok := parser.ParseAttributes(text.NewReader([]byte(block)))
	if !ok {
		return nil, "", false
	}
	remaining := strings.TrimRight(trimmed[:start], " \t")
	return attrs, remaining, true
}

func (t *AttributeTransformer) applyAttributeOnlyParagraphForTableCell(cell *extensionast.TableCell, reader text.Reader) bool {
	last := cell.LastChild()
	para, ok := last.(*ast.Paragraph)
	if !ok {
		return false
	}
	textValue, _, _, ok := paragraphTextOnly(para, reader)
	if !ok {
		return false
	}
	if !attributeOnlyPattern.MatchString(textValue) {
		return false
	}
	attrs, ok := parseAttributeOnly(textValue)
	if !ok {
		return false
	}

	applyParsedAttributes(cell, attrs)
	cell.RemoveChild(cell, para)
	return true
}

func (t *AttributeTransformer) applyTrailingAttributesToContainer(container ast.Node, reader text.Reader) {
	textValue, textNodes, stringNodes, ok := containerTextOnly(container, reader)
	if !ok {
		return
	}
	if para, ok := container.(*ast.Paragraph); ok {
		if parent, ok := para.Parent().(*ast.ListItem); ok {
			textValue, textNodes, stringNodes, ok := paragraphTextOnly(para, reader)
			if ok {
				if attrs, remaining, ok := parseTrailingAttributeSuffix(textValue); ok {
					applyParsedAttributes(parent, attrs)
					replaceTextNodes(container, remaining, textNodes, stringNodes)
					return
				}
			}
		}
		if parent, ok := para.Parent().(*ast.ListItem); ok {
			if attrs, remaining, ok := parseTrailingAttributeBlock(textValue); ok {
				applyParsedAttributes(parent, attrs)
				replaceTextNodes(container, remaining, textNodes, stringNodes)
				return
			}

			if attrs, remaining, ok := parseTrailingAttributes(textValue); ok {
				applyParsedAttributes(parent, attrs)
				replaceTextNodes(container, remaining, textNodes, stringNodes)
				return
			}

			if attrs, remaining, ok := parseTrailingAttributeLine(textValue); ok {
				applyParsedAttributes(parent, attrs)
				replaceTextNodes(container, remaining, textNodes, stringNodes)
				return
			}

			if attrs, ok := parseAttributeOnly(textValue); ok {
				applyParsedAttributes(parent, attrs)
				replaceTextNodes(container, "", textNodes, stringNodes)
				return
			}
		}
	}

	attrs, remaining, ok := parseTrailingAttributes(textValue)
	if !ok {
		return
	}

	applyParsedAttributes(container, attrs)
	if remaining == "" {
		for _, tn := range textNodes {
			container.RemoveChild(container, tn)
		}
		for _, sn := range stringNodes {
			container.RemoveChild(container, sn)
		}
		return
	}
	replaceTextNodes(container, remaining, textNodes, stringNodes)
}

func parseTrailingAttributeBlock(textValue string) (parser.Attributes, string, bool) {
	trimmed := strings.TrimRight(textValue, " \t\n\r")
	if trimmed == "" {
		return nil, "", false
	}
	start := strings.LastIndex(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start == -1 || end == -1 || end < start {
		return nil, "", false
	}
	block := trimmed[start : end+1]
	attrs, ok := parser.ParseAttributes(text.NewReader([]byte(block)))
	if !ok {
		return nil, "", false
	}
	remaining := strings.TrimRight(trimmed[:start], " \t\n\r")
	return attrs, remaining, true
}

func parseTrailingAttributeLine(textValue string) (parser.Attributes, string, bool) {
	lastNewline := strings.LastIndex(textValue, "\n")
	if lastNewline == -1 {
		return nil, "", false
	}
	lastLine := strings.TrimSpace(textValue[lastNewline+1:])
	if !attributeOnlyPattern.MatchString(lastLine) {
		return nil, "", false
	}
	attrs, ok := parseAttributeOnly(lastLine)
	if !ok {
		return nil, "", false
	}
	remaining := strings.TrimRight(textValue[:lastNewline], " \t\n\r")
	return attrs, remaining, true
}

func parseLeadingAttributes(attrText string) (parser.Attributes, string, bool) {
	if !strings.HasPrefix(attrText, "{") {
		return nil, "", false
	}
	end := strings.Index(attrText, "}")
	if end == -1 {
		return nil, "", false
	}
	block := attrText[:end+1]
	attrs, ok := parser.ParseAttributes(text.NewReader([]byte(block)))
	if !ok {
		return nil, "", false
	}
	remaining := attrText[end+1:]
	return attrs, remaining, true
}

func parseAttributeOnly(attrText string) (parser.Attributes, bool) {
	trimmed := strings.TrimSpace(attrText)
	attrs, remaining, ok := parseLeadingAttributes(trimmed)
	if !ok {
		return nil, false
	}
	if strings.TrimSpace(remaining) != "" {
		return nil, false
	}
	return attrs, true
}

func parseTrailingAttributes(textValue string) (attrs parser.Attributes, remaining string, ok bool) {
	trimmed := strings.TrimRight(textValue, " \t\n\r")
	end := strings.LastIndex(trimmed, "}")
	if end == -1 {
		return nil, "", false
	}
	start := strings.LastIndex(trimmed[:end], "{")
	if start == -1 {
		return nil, "", false
	}
	if start > 0 {
		prev := trimmed[start-1]
		if prev != ' ' && prev != '\t' && prev != '\n' {
			return nil, "", false
		}
	}
	block := trimmed[start : end+1]
	attrs, ok = parser.ParseAttributes(text.NewReader([]byte(block)))
	if !ok {
		return nil, "", false
	}
	prefix := trimmed[:start]
	remaining = strings.TrimRight(prefix, " \t")
	prefixNoSpaces := strings.TrimRight(prefix, " \t")
	attrLineOnly := strings.HasSuffix(prefixNoSpaces, "\n") || strings.HasSuffix(prefixNoSpaces, "\r\n")
	if attrLineOnly {
		remaining = strings.TrimRight(remaining, "\n\r")
	}
	return attrs, remaining, true
}

func applyParsedAttributes(target ast.Node, attrs parser.Attributes) {
	for _, attr := range attrs {
		name := string(attr.Name)
		value := normalizeAttributeValue(attr.Value)
		if name == "class" {
			if existing, ok := target.AttributeString("class"); ok {
				existingStr := attributeValueToString(existing)
				if existingStr != "" {
					value = existingStr + " " + value
				}
			}
		}
		target.SetAttributeString(name, value)
	}
}

func normalizeAttributeValue(value interface{}) string {
	switch typed := value.(type) {
	case []byte:
		return string(typed)
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return ""
	}
}

func attributeValueToString(value interface{}) string {
	switch typed := value.(type) {
	case []byte:
		return string(typed)
	case string:
		return typed
	default:
		return ""
	}
}

func paragraphTextOnly(para *ast.Paragraph, reader text.Reader) (string, []*ast.Text, []*ast.String, bool) {
	var textNodes []*ast.Text
	var stringNodes []*ast.String
	var textContent strings.Builder

	for child := para.FirstChild(); child != nil; child = child.NextSibling() {
		switch node := child.(type) {
		case *ast.Text:
			textContent.WriteString(string(node.Segment.Value(reader.Source())))
			if node.SoftLineBreak() || node.HardLineBreak() {
				textContent.WriteString("\n")
			}
			textNodes = append(textNodes, node)
		case *ast.String:
			textContent.WriteString(string(node.Value))
			stringNodes = append(stringNodes, node)
		default:
			return "", nil, nil, false
		}
	}

	return textContent.String(), textNodes, stringNodes, true
}

func blockTextOnly(container ast.Node, reader text.Reader) (string, []*ast.Text, []*ast.String, bool) {
	return containerTextOnly(container, reader)
}

func containerTextOnly(container ast.Node, reader text.Reader) (string, []*ast.Text, []*ast.String, bool) {
	var textNodes []*ast.Text
	var stringNodes []*ast.String
	var textContent strings.Builder

	for child := container.FirstChild(); child != nil; child = child.NextSibling() {
		switch node := child.(type) {
		case *ast.Text:
			textContent.WriteString(string(node.Segment.Value(reader.Source())))
			if node.SoftLineBreak() || node.HardLineBreak() {
				textContent.WriteString("\n")
			}
			textNodes = append(textNodes, node)
		case *ast.String:
			textContent.WriteString(string(node.Value))
			stringNodes = append(stringNodes, node)
		default:
			return "", nil, nil, false
		}
	}

	return textContent.String(), textNodes, stringNodes, true
}

func listItemTextContainer(node ast.Node) (ast.Node, bool) {
	if node == nil {
		return nil, false
	}
	if _, ok := node.(*ast.Paragraph); ok {
		return node, true
	}
	if _, ok := node.(*ast.TextBlock); ok {
		return node, true
	}
	return nil, false
}

func replaceTextNodes(container ast.Node, remaining string, textNodes []*ast.Text, stringNodes []*ast.String) {
	for _, tn := range textNodes {
		container.RemoveChild(container, tn)
	}
	for _, sn := range stringNodes {
		container.RemoveChild(container, sn)
	}
	container.AppendChild(container, ast.NewString([]byte(remaining)))
}
