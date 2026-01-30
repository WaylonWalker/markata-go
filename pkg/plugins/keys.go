// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// KindKeys is the AST node kind for keyboard key elements.
var KindKeys = ast.NewNodeKind("Keys")

// Keys is an AST node representing keyboard keys (++Ctrl+Alt+Del++).
type Keys struct {
	ast.BaseInline
	// KeySequence holds the individual keys (e.g., ["Ctrl", "Alt", "Del"])
	KeySequence []string
}

// Kind returns the kind of this node.
func (n *Keys) Kind() ast.NodeKind {
	return KindKeys
}

// Dump dumps the node for debugging.
func (n *Keys) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{
		"Keys": strings.Join(n.KeySequence, "+"),
	}, nil)
}

// NewKeys creates a new Keys node.
func NewKeys(keys []string) *Keys {
	return &Keys{
		KeySequence: keys,
	}
}

// keysParser parses ++Key+Key++ syntax for keyboard keys.
type keysParser struct{}

// newKeysParser creates a new keys parser.
func newKeysParser() parser.InlineParser {
	return &keysParser{}
}

// Trigger returns the trigger bytes for this parser.
func (p *keysParser) Trigger() []byte {
	return []byte{'+'}
}

// Parse parses the ++Key+Key++ syntax.
func (p *keysParser) Parse(_ ast.Node, block text.Reader, _ parser.Context) ast.Node {
	line, _ := block.PeekLine()

	// Must start with ++
	if len(line) < 4 || line[0] != '+' || line[1] != '+' {
		return nil
	}

	// Find the closing ++
	// Start after the opening ++
	content := line[2:]
	endIdx := -1

	for i := 0; i < len(content)-1; i++ {
		if content[i] == '+' && content[i+1] == '+' {
			// Make sure this isn't a key separator (single +)
			// Check that we have content before this
			if i > 0 {
				endIdx = i
				break
			}
		}
	}

	if endIdx == -1 {
		return nil
	}

	// Extract the key sequence
	keyStr := string(content[:endIdx])
	if keyStr == "" {
		return nil
	}

	// Split by + to get individual keys
	keys := splitKeys(keyStr)
	if len(keys) == 0 {
		return nil
	}

	// Total length consumed: ++ + content + ++
	totalLen := 2 + endIdx + 2
	block.Advance(totalLen)

	return NewKeys(keys)
}

// splitKeys splits a key string by + while handling edge cases.
// e.g., "Ctrl+Alt+Del" -> ["Ctrl", "Alt", "Del"]
// e.g., "Ctrl++" -> ["Ctrl", "+"] (plus key)
func splitKeys(s string) []string {
	var keys []string
	var current strings.Builder

	i := 0
	for i < len(s) {
		if s[i] == '+' {
			// Check if this is a plus key (at the end or followed by another +)
			if current.Len() == 0 {
				// Empty before +, this might be a + key at the start
				// But that's unusual, skip
				i++
				continue
			}

			// We have content, this + is a separator
			keys = append(keys, current.String())
			current.Reset()
			i++
		} else {
			current.WriteByte(s[i])
			i++
		}
	}

	// Don't forget the last key
	if current.Len() > 0 {
		keys = append(keys, current.String())
	}

	return keys
}

// keysHTMLRenderer renders Keys nodes to HTML.
type keysHTMLRenderer struct {
	html.Config
}

// newKeysHTMLRenderer creates a new keys HTML renderer.
func newKeysHTMLRenderer() renderer.NodeRenderer {
	return &keysHTMLRenderer{
		Config: html.NewConfig(),
	}
}

// RegisterFuncs registers the render functions.
func (r *keysHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindKeys, r.renderKeys)
}

// keyClassMap maps common key names to CSS class suffixes.
var keyClassMap = map[string]string{
	"ctrl":        "ctrl",
	"control":     "ctrl",
	"alt":         "alt",
	"shift":       "shift",
	"meta":        "meta",
	"cmd":         "meta",
	"command":     "meta",
	"win":         "win",
	"windows":     "win",
	"super":       "win",
	"enter":       "enter",
	"return":      "enter",
	"tab":         "tab",
	"space":       "space",
	"backspace":   "backspace",
	"delete":      "delete",
	"del":         "delete",
	"escape":      "escape",
	"esc":         "escape",
	"up":          "up",
	"down":        "down",
	"left":        "left",
	"right":       "right",
	"home":        "home",
	"end":         "end",
	"pageup":      "page-up",
	"pagedown":    "page-down",
	"insert":      "insert",
	"caps":        "caps-lock",
	"capslock":    "caps-lock",
	"num":         "num-lock",
	"numlock":     "num-lock",
	"scroll":      "scroll-lock",
	"print":       "print-screen",
	"printscreen": "print-screen",
	"pause":       "pause",
	"break":       "break",
	"f1":          "f1",
	"f2":          "f2",
	"f3":          "f3",
	"f4":          "f4",
	"f5":          "f5",
	"f6":          "f6",
	"f7":          "f7",
	"f8":          "f8",
	"f9":          "f9",
	"f10":         "f10",
	"f11":         "f11",
	"f12":         "f12",
}

// renderKeys renders a Keys node to HTML.
//
//nolint:errcheck // WriteString errors are handled at a higher level in goldmark
func (r *keysHTMLRenderer) renderKeys(w util.BufWriter, _ []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	keys, ok := n.(*Keys)
	if !ok {
		return ast.WalkContinue, nil
	}

	_, _ = w.WriteString(`<span class="keys">`)

	for i, key := range keys.KeySequence {
		if i > 0 {
			_, _ = w.WriteString(`<span class="key-separator">+</span>`)
		}

		// Determine CSS class for this key
		class := "kbd"
		if keyClass, ok := keyClassMap[strings.ToLower(key)]; ok {
			class = "kbd key-" + keyClass
		}

		_, _ = w.WriteString(`<kbd class="`)
		_, _ = w.WriteString(class)
		_, _ = w.WriteString(`">`)
		// HTML escape the key name
		for _, c := range key {
			switch c {
			case '<':
				_, _ = w.WriteString("&lt;")
			case '>':
				_, _ = w.WriteString("&gt;")
			case '&':
				_, _ = w.WriteString("&amp;")
			case '"':
				_, _ = w.WriteString("&quot;")
			default:
				_, _ = w.WriteRune(c)
			}
		}
		_, _ = w.WriteString(`</kbd>`)
	}

	_, _ = w.WriteString(`</span>`)

	return ast.WalkContinue, nil
}

// KeysExtension is a goldmark extension for keyboard keys syntax.
type KeysExtension struct{}

// Extend adds the keys parser and renderer to goldmark.
func (e *KeysExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithInlineParsers(
			// Higher priority than mark to handle ++ before =
			util.Prioritized(newKeysParser(), 400),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(newKeysHTMLRenderer(), 500),
		),
	)
}
