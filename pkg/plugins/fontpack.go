package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
	"github.com/WaylonWalker/markata-go/pkg/fontpacks"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

const (
	fontpackCacheFile    = ".markata-fontpack-cache"
	fontpackCacheVersion = "1"
)

// FontpackPlugin installs one site-wide typography stylesheet. It never calls
// a subsetter: bundled tiers are immutable catalog artifacts.
type FontpackPlugin struct {
	name   string
	source *fontpacks.CatalogSource
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
	name := configuredFontpackName(m.Config().Extra)
	p.name = name
	path := ""
	if v, ok := m.Config().Extra["fontpacks_file"].(string); ok && v != "" {
		path = v
	}
	var err error
	if path != "" {
		p.source, err = fontpacks.LoadSource(path)
	} else {
		p.source, err = fontpacks.BuiltinSource()
	}
	if err != nil {
		return fmt.Errorf("load font catalog for pack %q: %w", name, err)
	}
	m.Config().Extra["fontpack_css"] = true
	return nil
}

func configuredFontpackName(extra map[string]any) string {
	canonicalize := func(name string) string {
		if name == "brush-poster" {
			return "brush"
		}
		return name
	}
	if configured, ok := extra["models_config"].(*models.Config); ok && configured.Theme.Fontpack != "" {
		return canonicalize(configured.Theme.Fontpack)
	}
	if value, ok := extra["fontpack"].(string); ok && value != "" {
		return canonicalize(value)
	}
	return "system"
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
			if value == "brush-poster" {
				value = "brush"
			}
			name = value
		}
		if p.source != nil {
			resolvedName, _, err := p.source.Catalog.ResolvePack(name)
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
	_, _, err := p.source.Catalog.ResolvePack(p.name)
	if err != nil {
		return err
	}
	cacheKey := fontpackCacheKey(rendered.String(), names, p.source.Catalog)
	if !p.source.Builtin || !fontpackOutputCached(output, cacheKey) {
		resolved, err := p.source.Catalog.ResolveManyFSWithOptions(names, p.source.FS, p.source.Root, rendered.String(), fontpackResolveOptions(p.source))
		if err != nil {
			return err
		}
		if err := resolved.CopyFS(p.source.FS, p.source.Root, output); err != nil {
			return err
		}
		if p.source.Builtin {
			if err := os.WriteFile(filepath.Join(output, fontpackCacheFile), []byte(cacheKey), 0o600); err != nil {
				return err
			}
		} else if err := os.Remove(filepath.Join(output, fontpackCacheFile)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	for _, post := range m.Posts() {
		if name := pageNames[post.Path]; name != "" {
			post.HTML = markPostFontpack(post.HTML, name)
		}
	}
	return nil
}

func fontpackCacheKey(rendered string, names []string, catalog *fontpacks.Catalog) string {
	catalogData, err := json.Marshal(catalog)
	if err != nil {
		return ""
	}
	return buildcache.ContentHash(fontpackCacheVersion + "\x00" + string(catalogData) + "\x00" + rendered + "\x00" + strings.Join(names, "\x00"))
}

func fontpackOutputCached(output, key string) bool {
	data, err := os.ReadFile(filepath.Join(output, fontpackCacheFile))
	if err != nil || strings.TrimSpace(string(data)) != key {
		return false
	}
	if info, err := os.Stat(filepath.Join(output, "css", "fonts.css")); err != nil || !info.Mode().IsRegular() {
		return false
	}
	files, err := fontpacks.ManagedFontFiles(output)
	if err != nil {
		return false
	}
	for _, file := range files {
		info, err := os.Stat(filepath.Join(output, "assets", "fonts", file))
		if err != nil || !info.Mode().IsRegular() {
			return false
		}
	}
	return true
}

func fontpackResolveOptions(source *fontpacks.CatalogSource) fontpacks.ResolveOptions {
	// Bundled assets are immutable and already validated when the release is
	// built. Re-hashing every WOFF2 file on every site build dominates warm
	// builds, while custom catalogs must retain runtime checksum validation.
	return fontpacks.ResolveOptions{ValidateChecksums: !source.Builtin}
}

func markPostFontpack(content, name string) string {
	if content == "" {
		return content
	}
	content = markHTMLFontpack(content, name)
	if !strings.Contains(content, `href="/css/fonts.css"`) {
		content = strings.Replace(content, "</head>", `  <link rel="stylesheet" href="/css/fonts.css">`+"\n</head>", 1)
	}
	return content
}

func markHTMLFontpack(content, name string) string {
	quoted := `data-fontpack="` + name + `"`
	if strings.Contains(strings.ToLower(content), "<html") {
		for _, tag := range []string{"<html>", "<html ", "<HTML>", "<HTML "} {
			start := strings.Index(content, tag)
			if start < 0 {
				continue
			}
			end := strings.Index(content[start:], ">")
			if end < 0 {
				return content
			}
			end += start
			open := content[start : end+1]
			if strings.Contains(strings.ToLower(open), "data-fontpack=") {
				parts := strings.Fields(open[:len(open)-1])
				for i, part := range parts {
					if strings.HasPrefix(strings.ToLower(part), "data-fontpack=") {
						parts[i] = quoted
					}
				}
				open = strings.Join(parts, " ") + ">"
			} else {
				open = strings.TrimSuffix(open, ">") + " " + quoted + ">"
			}
			return content[:start] + open + content[end+1:]
		}
	}
	return content
}

var _ lifecycle.ConfigurePlugin = (*FontpackPlugin)(nil)
var _ lifecycle.WritePlugin = (*FontpackPlugin)(nil)
var _ lifecycle.PriorityPlugin = (*FontpackPlugin)(nil)
