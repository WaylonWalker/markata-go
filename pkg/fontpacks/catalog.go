// Package fontpacks implements Markata's catalog-driven typography system.
//
// Normal builds only read YAML manifests and copy already-built WOFF2 files;
// font processing tools are intentionally outside this package's build path.
package fontpacks

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

var ErrCatalogNotFound = errors.New("font pack catalog not found")

type Catalog struct {
	Schema         string                   `yaml:"schema"`
	Version        int                      `yaml:"version"`
	Catalog        CatalogPaths             `yaml:"catalog"`
	SystemStacks   map[string]SystemStack   `yaml:"system_stacks"`
	SubsetProfiles map[string]SubsetProfile `yaml:"subset_profiles"`
	FontSources    map[string]FontSource    `yaml:"font_sources"`
	FontPacks      map[string]FontPack      `yaml:"fontpacks"`
	Aliases        map[string]string        `yaml:"aliases"`
}

// CatalogSource keeps a catalog together with the filesystem that owns the
// paths referenced by its manifests. Relative asset paths therefore remain
// relative to the catalog, whether the source is embedded or user-provided.
type CatalogSource struct {
	Catalog *Catalog
	FS      fs.FS
	Root    string
	LockFS  fs.FS
	Lock    string
	Builtin bool
}

type CatalogPaths struct {
	ContentVersion   string   `yaml:"content_version"`
	Lockfile         string   `yaml:"lockfile"`
	BundledAssetRoot string   `yaml:"bundled_asset_root"`
	ProjectAssetRoot string   `yaml:"project_asset_root"`
	OutputAssetRoot  string   `yaml:"output_asset_root"`
	ResolutionOrder  []string `yaml:"resolution_order"`
}

type SystemStack struct {
	CSS string `yaml:"css"`
}
type SubsetProfile struct {
	Unicode     []string `yaml:"unicode"`
	Passthrough bool     `yaml:"passthrough"`
}
type FontSource struct {
	Provider string `yaml:"provider"`
	Family   string `yaml:"family"`
}
type FontPack struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Performance Performance     `yaml:"performance"`
	Roles       map[string]Role `yaml:"roles"`
}
type Performance struct {
	Class             string `yaml:"class"`
	ExpectedFontBytes int64  `yaml:"expected_font_bytes"`
}
type Role struct {
	Stack       string  `yaml:"stack"`
	Source      string  `yaml:"source"`
	Tier        string  `yaml:"tier"`
	Fallback    string  `yaml:"fallback"`
	Weight      float64 `yaml:"weight"`
	Style       string  `yaml:"style"`
	Size        string  `yaml:"size"`
	OpticalSize float64 `yaml:"optical_size"`
}

// Manifest describes one vendored family and its stable tiers.
type Manifest struct {
	Schema  string          `yaml:"schema"`
	ID      string          `yaml:"id"`
	Family  string          `yaml:"family"`
	Scope   string          `yaml:"scope"`
	Source  ManifestSource  `yaml:"source"`
	License License         `yaml:"license"`
	Faces   map[string]Face `yaml:"faces"`
	Tiers   map[string]Tier `yaml:"tiers"`
}
type ManifestSource struct {
	Provider   string            `yaml:"provider"`
	Repository string            `yaml:"repository"`
	Revision   string            `yaml:"revision"`
	Directory  string            `yaml:"directory"`
	Files      map[string]string `yaml:"files"`
}
type License struct {
	ID        string `yaml:"id"`
	File      string `yaml:"file"`
	Copyright string `yaml:"copyright"`
	SHA256    string `yaml:"sha256"`
}

// FindFamily resolves a family directory using the caller's precedence order.
// Passing project, user, then builtin roots implements the catalog policy while
// keeping the package independent of a particular installation layout.
func FindFamily(id string, roots ...string) (string, error) {
	for _, root := range roots {
		if root == "" {
			continue
		}
		path := filepath.Join(root, id)
		if _, err := os.Stat(filepath.Join(path, "manifest.yaml")); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("font family %q not found in catalog precedence roots", id)
}

type Face struct {
	Style      string               `yaml:"style"`
	Variable   bool                 `yaml:"variable"`
	Weight     []float64            `yaml:"weight"`
	Axes       map[string][]float64 `yaml:"axes"`
	SourceFile string               `yaml:"source_file"`
}
type Tier struct {
	File         string   `yaml:"file"`
	Profile      string   `yaml:"profile"`
	SHA256       string   `yaml:"sha256"`
	Bytes        int64    `yaml:"bytes"`
	UnicodeRange []string `yaml:"unicode_range"`
}

// Load reads and validates a markata-fontpacks.yaml file.
func Load(path string) (*Catalog, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrCatalogNotFound, path)
		}
		return nil, err
	}
	var c Catalog
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse font catalog: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// LoadSource loads a filesystem catalog and resolves its asset root relative
// to the catalog file rather than the process working directory.
func LoadSource(path string) (*CatalogSource, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	c, err := Load(abs)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(abs)
	root := c.Catalog.BundledAssetRoot
	if root == "" {
		root = "."
	}
	assetRoot := root
	if !filepath.IsAbs(assetRoot) {
		assetRoot = filepath.Join(dir, assetRoot)
	}
	assetRoot = filepath.Clean(assetRoot)
	lock := c.Catalog.Lockfile
	if lock == "" {
		lock = "markata-fonts.lock.yaml"
	}
	if !filepath.IsAbs(lock) {
		lock = filepath.Join(dir, lock)
	}
	return &CatalogSource{Catalog: c, FS: os.DirFS(assetRoot), Root: ".", LockFS: os.DirFS(filepath.Dir(lock)), Lock: filepath.Base(lock)}, nil
}

func (c *Catalog) Validate() error {
	if c == nil {
		return errors.New("font catalog is nil")
	}
	if len(c.SystemStacks) == 0 {
		return errors.New("font catalog declares no system_stacks")
	}
	if len(c.FontPacks) == 0 {
		return errors.New("font catalog declares no fontpacks")
	}
	for name, p := range c.FontPacks {
		if p.Performance.Class == "" {
			return fmt.Errorf("font pack %q has no performance class", name)
		}
		for role, r := range p.Roles {
			if r.Stack == "" && r.Source == "" {
				return fmt.Errorf("font pack %q role %q has neither stack nor source", name, role)
			}
			if r.Stack != "" && c.SystemStacks[r.Stack].CSS == "" {
				return fmt.Errorf("font pack %q role %q references missing system stack %q", name, role, r.Stack)
			}
			if r.Source != "" && c.FontSources[r.Source].Family == "" {
				return fmt.Errorf("font pack %q role %q references missing source %q", name, role, r.Source)
			}
		}
	}
	return nil
}

func (c *Catalog) ResolvePack(name string) (string, FontPack, error) {
	if name == "" {
		name = "system"
	}
	seen := map[string]bool{}
	for {
		if seen[name] {
			return "", FontPack{}, fmt.Errorf("font pack alias cycle at %q", name)
		}
		seen[name] = true
		if p, ok := c.FontPacks[name]; ok {
			return name, p, nil
		}
		next, ok := c.Aliases[name]
		if !ok {
			return "", FontPack{}, fmt.Errorf("unknown font pack %q", name)
		}
		name = next
	}
}

// RequiredTiers returns stable tiers needed by the pack and final rendered text.
func (c *Catalog) RequiredTiers(pack FontPack, renderedHTML string) map[string]map[string]bool {
	result := map[string]map[string]bool{}
	for _, r := range pack.Roles {
		if r.Source != "" {
			if result[r.Source] == nil {
				result[r.Source] = map[string]bool{}
			}
			result[r.Source][r.Tier] = true
		}
	}
	text := VisibleText(renderedHTML)
	for source, tiers := range result {
		for _, r := range text {
			if r == utf8.RuneError {
				continue
			}
			if inProfile(c.SubsetProfiles, "latin-ext", r) {
				tiers["latin-ext"] = true
				continue
			}
			if !inAnyProfile(c.SubsetProfiles, tiers, r) {
				tiers["full"] = true
			}
		}
		// Full coverage supersedes every smaller tier for that family. Keeping
		// both would let the unrestricted full face win for ordinary glyphs and
		// would waste bytes without improving coverage.
		if tiers["full"] {
			result[source] = map[string]bool{"full": true}
		}
	}
	return result
}

// RequiredTiersForManifest makes optional coverage decisions against the
// tiers that a particular family actually ships. A family without latin-ext
// must use full coverage rather than failing the entire pack resolution.
func (c *Catalog) RequiredTiersForManifest(pack FontPack, renderedHTML string, manifests map[string]Manifest) map[string]map[string]bool {
	requested := c.RequiredTiers(pack, renderedHTML)
	result := make(map[string]map[string]bool, len(requested))
	for source, tiers := range requested {
		manifest := manifests[source]
		available := make(map[string]bool, len(manifest.Tiers))
		for tier := range manifest.Tiers {
			available[tier] = true
		}
		selected := make(map[string]bool)
		for tier := range tiers {
			if available[tier] {
				selected[tier] = true
				continue
			}
			if available["full"] {
				selected["full"] = true
				continue
			}
			selected[tier] = true // retain the useful missing-tier diagnostic
		}
		if selected["full"] {
			selected = map[string]bool{"full": true}
		}
		result[source] = selected
	}
	return result
}

func inAnyProfile(profiles map[string]SubsetProfile, tiers map[string]bool, r rune) bool {
	for tier := range tiers {
		if inProfile(profiles, tier, r) {
			return true
		}
	}
	return false
}
func inProfile(profiles map[string]SubsetProfile, name string, r rune) bool {
	p, ok := profiles[name]
	if !ok || p.Passthrough {
		return false
	}
	for _, s := range p.Unicode {
		if unicodeRangeContains(s, r) {
			return true
		}
	}
	return false
}

var unicodeRangeRE = regexp.MustCompile(`(?i)^U\+([0-9A-F]+)(?:-([0-9A-F]+))?$`)

func unicodeRangeContains(s string, r rune) bool {
	m := unicodeRangeRE.FindStringSubmatch(strings.TrimSpace(s))
	if len(m) == 0 {
		return false
	}
	a, err := strconv.ParseInt(m[1], 16, 32)
	if err != nil {
		return false
	}
	b := a
	if m[2] != "" {
		b, err = strconv.ParseInt(m[2], 16, 32)
		if err != nil {
			return false
		}
	}
	return int64(r) >= a && int64(r) <= b
}

// VisibleText extracts text nodes while excluding markup, attributes, scripts,
// and styles. It is deliberately site-level and never creates a page subset.
func VisibleText(source string) string {
	doc, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return regexp.MustCompile(`<[^>]*>`).ReplaceAllString(source, " ")
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(doc)
	return b.String()
}

// AssetSHA256 returns the authoritative full SHA-256 hash for an asset.
func AssetSHA256(path string) (hash string, bytes int64, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), int64(len(b)), nil
}

// AssetSHA256FS is the filesystem equivalent of AssetSHA256.
func AssetSHA256FS(source fs.FS, path string) (hash string, bytes int64, err error) {
	b, err := fs.ReadFile(source, path)
	if err != nil {
		return "", 0, err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), int64(len(b)), nil
}

// ShortHash returns a display/cache identifier from a full hash.
func ShortHash(full string) string {
	if len(full) < 12 {
		return full
	}
	return full[:12]
}

// AssetHash is kept as a compatibility alias and now returns the full hash.
func AssetHash(path string) (hash string, bytes int64, err error) {
	return AssetSHA256(path)
}

// SortedKeys makes generated CSS and reports deterministic.
func SortedKeys[T any](m map[string]T) []string {
	r := make([]string, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	sort.Strings(r)
	return r
}
