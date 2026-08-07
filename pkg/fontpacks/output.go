package fontpacks

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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

const managedFontsManifest = ".markata-fonts.json"

// ResolveMany resolves a site pack plus any per-page overrides into one
// reusable stylesheet and one deduplicated asset set. Font binaries remain
// site-level; only the semantic variables vary per page.
func (c *Catalog) ResolveMany(names []string, catalogRoot, renderedHTML string) (*Resolved, error) {
	return c.ResolveManyFS(names, os.DirFS(catalogRoot), ".", renderedHTML)
}

// ResolveManyFS resolves multiple packs from a portable filesystem source.
func (c *Catalog) ResolveManyFS(names []string, assetFS fs.FS, assetRoot, renderedHTML string) (*Resolved, error) {
	if len(names) == 0 {
		names = []string{"system"}
	}
	result := &Resolved{Packs: map[string]FontPack{}}
	seen := map[string]bool{}
	for _, name := range names {
		resolved, err := c.ResolveFS(name, assetFS, assetRoot, renderedHTML)
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
	return c.ResolveFS(name, os.DirFS(catalogRoot), ".", renderedHTML)
}

// ResolveFS resolves manifests and assets from an fs.FS. The filesystem is
// rooted at the catalog's asset directory, so all catalog paths are portable.
func (c *Catalog) ResolveFS(name string, assetFS fs.FS, assetRoot, renderedHTML string) (*Resolved, error) {
	resolvedName, pack, err := c.ResolvePack(name)
	if err != nil {
		return nil, err
	}
	r := &Resolved{Name: resolvedName, Pack: pack}
	required := c.RequiredTiers(pack, renderedHTML)
	for _, source := range SortedKeys(required) {
		manifestPath := filepath.ToSlash(filepath.Join(assetRoot, source, "manifest.yaml"))
		var manifest Manifest
		data, err := fs.ReadFile(assetFS, manifestPath)
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
			path := filepath.ToSlash(filepath.Join(assetRoot, source, entry.File))
			info, err := fs.Stat(assetFS, path)
			if err != nil {
				return nil, fmt.Errorf("font tier %q for %s is missing: %w", tier, source, err)
			}
			if entry.Profile != "" && entry.Profile != tier {
				return nil, fmt.Errorf("font tier %q for %s declares profile %q", tier, source, entry.Profile)
			}
			if entry.SHA256 != "" {
				hash, _, hashErr := AssetSHA256FS(assetFS, path)
				if hashErr != nil || entry.SHA256 != hash {
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
	return r.CopyFS(os.DirFS(catalogRoot), ".", outputDir)
}

// CopyFS copies selected assets from a portable filesystem source and records
// which files Markata owns. Previous Markata-managed files are removed before
// the new selection is written; unrelated user files are preserved.
func (r *Resolved) CopyFS(assetFS fs.FS, assetRoot, outputDir string) error {
	if err := os.MkdirAll(filepath.Join(outputDir, "assets/fonts"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "css"), 0o755); err != nil {
		return err
	}
	for _, a := range r.Assets {
		name := filepath.Base(a.File)
		src := filepath.ToSlash(filepath.Join(assetRoot, a.Source, a.File))
		data, err := fs.ReadFile(assetFS, src)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(outputDir, "assets/fonts", name), data, 0o644); err != nil {
			return err
		}
	}
	if err := updateManagedFonts(outputDir, r.Assets); err != nil {
		return err
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

type managedFonts struct {
	Files []string `json:"files"`
}

func updateManagedFonts(outputDir string, assets []Asset) error {
	dir := filepath.Join(outputDir, "assets/fonts")
	manifestPath := filepath.Join(dir, managedFontsManifest)
	var previous managedFonts
	if data, err := os.ReadFile(manifestPath); err == nil {
		if err := json.Unmarshal(data, &previous); err != nil {
			return fmt.Errorf("parse generated font manifest: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	wanted := make(map[string]bool, len(assets))
	for _, asset := range assets {
		wanted[filepath.Base(asset.File)] = true
	}
	for _, name := range previous.Files {
		name = filepath.Base(name)
		if !wanted[name] && name != managedFontsManifest {
			if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove stale generated font %q: %w", name, err)
			}
		}
	}
	files := make([]string, 0, len(wanted))
	for name := range wanted {
		files = append(files, name)
	}
	sort.Strings(files)
	data, err := json.MarshalIndent(managedFonts{Files: files}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(manifestPath, data, 0o644)
}

// CleanManagedFonts removes only files named by the previous Markata manifest.
func CleanManagedFonts(outputDir string) error {
	return updateManagedFonts(outputDir, nil)
}

// ManagedFontFiles returns the authoritative list of Markata-generated font
// files in an output directory.
func ManagedFontFiles(outputDir string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(outputDir, "assets/fonts", managedFontsManifest))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var manifest managedFonts
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	sort.Strings(manifest.Files)
	return manifest.Files, nil
}

// SystemResolved is useful to callers that have no catalog files installed.
func SystemResolved() *Resolved {
	return &Resolved{Name: "system", Pack: FontPack{Performance: Performance{Class: "zero-download"}}, CSS: ":root {\n  --font-display: system-ui, sans-serif;\n  --font-heading: system-ui, sans-serif;\n  --font-body: system-ui, sans-serif;\n  --font-code: ui-monospace, monospace;\n}\n"}
}
