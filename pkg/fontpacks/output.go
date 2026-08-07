package fontpacks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Asset struct {
	Source, Tier, File, URL string
	Bytes                   int64
	SHA256                  string
	UnicodeRange            []string
	Style                   string
	Weight                  []float64
}
type Resolved struct {
	Name   string
	Pack   FontPack
	Packs  map[string]FontPack
	CSS    string
	Assets []Asset
	Bytes  int64
}

// ResolveMany resolves a site pack plus any per-page overrides into one
// reusable stylesheet and one deduplicated asset set. Font binaries remain
// site-level; only the semantic variables vary per page.
func (c *Catalog) ResolveMany(names []string, catalogRoot, renderedHTML string) (*Resolved, error) {
	if len(names) == 0 {
		names = []string{"system"}
	}
	result := &Resolved{Packs: map[string]FontPack{}}
	seen := map[string]bool{}
	for _, name := range names {
		resolved, err := c.Resolve(name, catalogRoot, "", renderedHTML)
		if err != nil {
			return nil, err
		}
		result.Packs[resolved.Name] = resolved.Pack
		if result.Name == "" {
			result.Name, result.Pack = resolved.Name, resolved.Pack
		}
		for _, asset := range resolved.Assets {
			key := asset.Source + "\x00" + asset.Tier
			if seen[key] {
				continue
			}
			seen[key] = true
			result.Assets = append(result.Assets, asset)
			result.Bytes += asset.Bytes
		}
	}
	result.CSS = c.cssForPacks(result.Packs, result.Assets)
	return result, nil
}

// Resolve builds deterministic CSS and the asset copy plan. catalogRoot is the
// directory containing family directories with manifest.yaml files.
func (c *Catalog) Resolve(name, catalogRoot, outputDir, renderedHTML string) (*Resolved, error) {
	resolvedName, pack, err := c.ResolvePack(name)
	if err != nil {
		return nil, err
	}
	r := &Resolved{Name: resolvedName, Pack: pack}
	required := c.RequiredTiers(pack, renderedHTML)
	for _, source := range SortedKeys(required) {
		manifestPath := filepath.Join(catalogRoot, source, "manifest.yaml")
		var manifest Manifest
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("font source %q requires %s: %w", source, manifestPath, err)
		}
		if err := yamlUnmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("parse font manifest %q: %w", source, err)
		}
		for _, tier := range SortedKeys(required[source]) {
			entry, ok := manifest.Tiers[tier]
			if !ok {
				return nil, fmt.Errorf("font source %q has no required tier %q", source, tier)
			}
			path := filepath.Join(catalogRoot, source, entry.File)
			info, err := os.Stat(path)
			if err != nil {
				return nil, fmt.Errorf("font tier %q for %s is missing: %w", tier, source, err)
			}
			if entry.Profile != "" && entry.Profile != tier {
				return nil, fmt.Errorf("font tier %q for %s declares profile %q", tier, source, entry.Profile)
			}
			if entry.SHA256 != "" {
				hash, _, hashErr := AssetHash(path)
				if hashErr != nil || !strings.HasPrefix(entry.SHA256, hash) {
					return nil, fmt.Errorf("font tier %q for %s has checksum %q, want %s", tier, source, hash, entry.SHA256)
				}
			}
			face := manifest.Faces["normal"]
			asset := Asset{Source: source, Tier: tier, File: entry.File, URL: "/" + filepath.ToSlash(filepath.Join("assets/fonts", entry.File)), Bytes: info.Size(), SHA256: entry.SHA256, UnicodeRange: entry.UnicodeRange, Style: face.Style, Weight: face.Weight}
			r.Assets = append(r.Assets, asset)
			r.Bytes += info.Size()
		}
	}
	r.CSS = c.css(pack, r.Assets)
	_ = outputDir // retained in the API for callers that resolve before writing
	return r, nil
}

func yamlUnmarshal(data []byte, v any) error { return yaml.Unmarshal(data, v) }

func (c *Catalog) css(pack FontPack, assets []Asset) string {
	var b strings.Builder
	b.WriteString("/* Markata font pack: ")
	b.WriteString(pack.Name)
	b.WriteString(" */\n")
	for _, a := range assets {
		family := ""
		if src, ok := c.FontSources[a.Source]; ok {
			family = src.Family
		}
		b.WriteString("@font-face {\n  font-family: ")
		b.WriteString(cssQuote(family))
		b.WriteString(";\n  src: url('")
		b.WriteString(a.URL)
		b.WriteString("') format('woff2');\n  font-display: swap;\n")
		if a.Style != "" {
			b.WriteString("  font-style: ")
			b.WriteString(a.Style)
			b.WriteString(";\n")
		}
		if len(a.Weight) == 2 {
			b.WriteString(fmt.Sprintf("  font-weight: %g %g;\n", a.Weight[0], a.Weight[1]))
		} else if len(a.Weight) == 1 {
			b.WriteString(fmt.Sprintf("  font-weight: %g;\n", a.Weight[0]))
		}
		if len(a.UnicodeRange) > 0 {
			b.WriteString("  unicode-range: ")
			b.WriteString(strings.Join(a.UnicodeRange, ", "))
			b.WriteString(";\n")
		}
		b.WriteString("}\n")
	}
	b.WriteString(rolesCSS(c, pack))
	return b.String()
}

func (c *Catalog) cssForPacks(packs map[string]FontPack, assets []Asset) string {
	var b strings.Builder
	b.WriteString("/* Markata font packs */\n")
	for _, a := range assets {
		family := ""
		if src, ok := c.FontSources[a.Source]; ok {
			family = src.Family
		}
		b.WriteString("@font-face {\n  font-family: ")
		b.WriteString(cssQuote(family))
		b.WriteString(";\n  src: url('")
		b.WriteString(a.URL)
		b.WriteString("') format('woff2');\n  font-display: swap;\n")
		if a.Style != "" {
			b.WriteString("  font-style: ")
			b.WriteString(a.Style)
			b.WriteString(";\n")
		}
		if len(a.Weight) == 2 {
			b.WriteString(fmt.Sprintf("  font-weight: %g %g;\n", a.Weight[0], a.Weight[1]))
		} else if len(a.Weight) == 1 {
			b.WriteString(fmt.Sprintf("  font-weight: %g;\n", a.Weight[0]))
		}
		if len(a.UnicodeRange) > 0 {
			b.WriteString("  unicode-range: ")
			b.WriteString(strings.Join(a.UnicodeRange, ", "))
			b.WriteString(";\n")
		}
		b.WriteString("}\n")
	}
	for _, name := range SortedKeys(packs) {
		b.WriteString("[data-fontpack=\"")
		b.WriteString(name)
		b.WriteString("\"] {\n")
		b.WriteString(roleDeclarations(c, packs[name]))
		b.WriteString("}\n")
	}
	return b.String()
}

func rolesCSS(c *Catalog, pack FontPack) string {
	var b strings.Builder
	b.WriteString(":root {\n")
	b.WriteString(roleDeclarations(c, pack))
	b.WriteString("}\n")
	return b.String()
}

func roleDeclarations(c *Catalog, pack FontPack) string {
	var b strings.Builder
	for _, role := range SortedKeys(pack.Roles) {
		r := pack.Roles[role]
		value := ""
		if r.Stack != "" {
			value = c.SystemStacks[r.Stack].CSS
		} else if src, ok := c.FontSources[r.Source]; ok {
			value = cssQuote(src.Family) + ", " + fallback(c, r.Fallback)
		}
		b.WriteString("  --font-")
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(value)
		b.WriteString(";\n")
	}
	return b.String()
}
func fallback(c *Catalog, name string) string {
	if s, ok := c.SystemStacks[name]; ok {
		return s.CSS
	}
	switch name {
	case "handwriting":
		return "cursive"
	case "serif":
		return "ui-serif, Georgia, serif"
	case "mono":
		return "ui-monospace, monospace"
	}
	if s, ok := c.SystemStacks["sans"]; ok {
		return s.CSS
	}
	return "system-ui, sans-serif"
}
func cssQuote(s string) string { return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"` }

// Copy writes only selected assets and the shared font stylesheet.
func (r *Resolved) Copy(catalogRoot, outputDir string) error {
	if err := os.MkdirAll(filepath.Join(outputDir, "assets/fonts"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "css"), 0o755); err != nil {
		return err
	}
	for _, a := range r.Assets {
		if err := copyFile(filepath.Join(catalogRoot, a.Source, a.File), filepath.Join(outputDir, "assets/fonts", a.File)); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(outputDir, "css", "fonts.css"), []byte(r.CSS), 0o644)
}
func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

// SystemResolved is useful to callers that have no catalog files installed.
func SystemResolved() *Resolved {
	return &Resolved{Name: "system", Pack: FontPack{Performance: Performance{Class: "zero-download"}}, CSS: ":root {\n  --font-display: system-ui, sans-serif;\n  --font-heading: system-ui, sans-serif;\n  --font-body: system-ui, sans-serif;\n  --font-code: ui-monospace, monospace;\n}\n"}
}
