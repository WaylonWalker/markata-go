package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/fontpacks"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
)

// FontpackPlugin installs one site-wide typography stylesheet. It never calls
// a subsetter: bundled tiers are immutable catalog artifacts.
type FontpackPlugin struct {
	name, catalogRoot string
	catalog           *fontpacks.Catalog
}

func NewFontpackPlugin() *FontpackPlugin { return &FontpackPlugin{} }
func (p *FontpackPlugin) Name() string   { return "fontpack" }
func (p *FontpackPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageWrite {
		return lifecycle.PriorityFirst
	}
	return lifecycle.PriorityDefault
}

func (p *FontpackPlugin) Configure(m *lifecycle.Manager) error {
	if m.Config().Extra == nil {
		m.Config().Extra = make(map[string]any)
	}
	name := "system"
	if v, ok := m.Config().Extra["fontpack"].(string); ok && v != "" {
		name = v
	}
	p.name = name
	path := "markata-fontpacks.yaml"
	if v, ok := m.Config().Extra["fontpacks_file"].(string); ok && v != "" {
		path = v
	}
	c, err := fontpacks.Load(path)
	if err != nil {
		if name == "system" || name == "default" || name == "fastest" {
			c = nil
		} else {
			return fmt.Errorf("load font catalog for pack %q: %w", name, err)
		}
	} else {
		p.catalog = c
	}
	m.Config().Extra["fontpack_css"] = true
	return nil
}

func (p *FontpackPlugin) Write(m *lifecycle.Manager) error {
	rendered := strings.Builder{}
	names := []string{p.name}
	pageNames := make(map[string]string)
	for _, post := range m.Posts() {
		rendered.WriteString(post.ArticleHTML)
		rendered.WriteByte('\n')
		name := p.name
		if value, ok := post.Extra["fontpack"].(string); ok && value != "" {
			name = value
		}
		if p.catalog != nil {
			resolvedName, _, err := p.catalog.ResolvePack(name)
			if err != nil {
				return fmt.Errorf("post %q fontpack %q: %w", post.Path, name, err)
			}
			pageNames[post.Path] = resolvedName
			if post.Extra == nil {
				post.Extra = make(map[string]interface{})
			}
			post.Extra["_resolved_fontpack"] = resolvedName
			if !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
	}
	output := m.Config().OutputDir
	if p.catalog == nil {
		if p.name != "system" && p.name != "default" && p.name != "fastest" {
			return fmt.Errorf("font pack %q requires markata-fontpacks.yaml", p.name)
		}
		return writeSystemCSS(output)
	}
	_, _, err := p.catalog.ResolvePack(p.name)
	if err != nil {
		return err
	}
	root := p.catalog.Catalog.BundledAssetRoot
	if root == "" {
		root = "internal/fontcatalog"
	}
	if !filepath.IsAbs(root) {
		root = filepath.Join(filepath.Dir("markata-fontpacks.yaml"), root)
	}
	resolved, err := p.catalog.ResolveMany(names, root, rendered.String())
	if err != nil {
		return err
	}
	if err := resolved.Copy(root, output); err != nil {
		return err
	}
	for _, post := range m.Posts() {
		if name := pageNames[post.Path]; name != "" {
			post.HTML = markPostFontpack(post.HTML, name)
		}
	}
	return nil
}

func markPostFontpack(content, name string) string {
	if content == "" {
		return content
	}
	content = strings.Replace(content, "<html>", `<html data-fontpack="`+name+`">`, 1)
	content = strings.Replace(content, "<html ", `<html data-fontpack="`+name+`" `, 1)
	if !strings.Contains(content, `href="/css/fonts.css"`) {
		content = strings.Replace(content, "</head>", `  <link rel="stylesheet" href="/css/fonts.css">`+"\n</head>", 1)
	}
	return content
}

func writeSystemCSS(output string) error {
	css := ":root {\n  --font-display: system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif;\n  --font-heading: system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif;\n  --font-subheading: system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif;\n  --font-body: system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif;\n  --font-lead: system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif;\n  --font-quote: ui-serif, Georgia, serif;\n  --font-caption: system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif;\n  --font-code: ui-monospace, \"SFMono-Regular\", \"Cascadia Mono\", \"Segoe UI Mono\", Menlo, Consolas, monospace;\n  --font-ui: system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif;\n  --font-metadata: system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif;\n  --font-label: system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif;\n  --font-numbers: ui-monospace, \"SFMono-Regular\", Menlo, monospace;\n}\n"
	if err := os.MkdirAll(filepath.Join(output, "css"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(output, "css/fonts.css"), []byte(css), 0o644)
}

var _ lifecycle.ConfigurePlugin = (*FontpackPlugin)(nil)
var _ lifecycle.WritePlugin = (*FontpackPlugin)(nil)
var _ lifecycle.PriorityPlugin = (*FontpackPlugin)(nil)
