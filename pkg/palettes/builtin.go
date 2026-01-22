package palettes

import (
	"embed"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

//go:embed palettes/*.toml
var builtinFS embed.FS

// builtinPalettes holds the pre-loaded built-in palettes.
var builtinPalettes map[string]*Palette

// builtinInfos holds info about built-in palettes for discovery.
var builtinInfos []PaletteInfo

// builtinOnce ensures built-in palettes are loaded only once.
var builtinOnce sync.Once

// initBuiltinPalettes initializes the built-in palettes from embedded files.
func initBuiltinPalettes() {
	builtinPalettes = make(map[string]*Palette)

	entries, err := builtinFS.ReadDir("palettes")
	if err != nil {
		return // No built-in palettes available
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		filePath := filepath.Join("palettes", entry.Name())
		data, err := builtinFS.ReadFile(filePath)
		if err != nil {
			continue
		}

		var pf paletteFile
		if err := toml.Unmarshal(data, &pf); err != nil {
			continue
		}

		p := &Palette{
			Name:        pf.Palette.Name,
			Variant:     Variant(pf.Palette.Variant),
			Author:      pf.Palette.Author,
			License:     pf.Palette.License,
			Homepage:    pf.Palette.Homepage,
			Description: pf.Palette.Description,
			Colors:      pf.Palette.Colors,
			Semantic:    pf.Palette.Semantic,
			Components:  pf.Palette.Components,
			Source:      sourceBuiltIn,
			SourcePath:  filePath,
		}

		// Initialize empty maps if nil
		if p.Colors == nil {
			p.Colors = make(map[string]string)
		}
		if p.Semantic == nil {
			p.Semantic = make(map[string]string)
		}
		if p.Components == nil {
			p.Components = make(map[string]string)
		}

		// Skip invalid palettes
		if errs := p.Validate(); len(errs) > 0 {
			continue
		}

		// Store with normalized name as key
		normalizedName := normalizeFileName(p.Name)
		builtinPalettes[normalizedName] = p
		builtinPalettes[p.Name] = p // Also store with original name

		builtinInfos = append(builtinInfos, PaletteInfo{
			Name:        p.Name,
			Variant:     p.Variant,
			Description: p.Description,
			Author:      p.Author,
			Source:      sourceBuiltIn,
			Path:        filePath,
		})
	}
}

// ensureBuiltinLoaded ensures built-in palettes are loaded.
func ensureBuiltinLoaded() {
	builtinOnce.Do(initBuiltinPalettes)
}

// LoadBuiltin loads a built-in palette by name.
// Returns ErrPaletteNotFound if the palette doesn't exist.
func LoadBuiltin(name string) (*Palette, error) {
	ensureBuiltinLoaded()

	// Try exact name first
	if p, ok := builtinPalettes[name]; ok {
		return p.Clone(), nil
	}

	// Try normalized name
	normalized := normalizeFileName(name)
	if p, ok := builtinPalettes[normalized]; ok {
		return p.Clone(), nil
	}

	return nil, NewPaletteLoadError(name, "", "built-in palette not found", ErrPaletteNotFound)
}

// DiscoverBuiltin returns info about all built-in palettes.
func DiscoverBuiltin() []PaletteInfo {
	ensureBuiltinLoaded()

	result := make([]PaletteInfo, len(builtinInfos))
	copy(result, builtinInfos)
	return result
}

// BuiltinNames returns the names of all built-in palettes.
func BuiltinNames() []string {
	ensureBuiltinLoaded()

	seen := make(map[string]bool)
	var names []string

	for _, info := range builtinInfos {
		if !seen[info.Name] {
			names = append(names, info.Name)
			seen[info.Name] = true
		}
	}

	return names
}

// HasBuiltin checks if a palette name exists in built-in palettes.
func HasBuiltin(name string) bool {
	ensureBuiltinLoaded()

	_, ok := builtinPalettes[name]
	if ok {
		return true
	}
	_, ok = builtinPalettes[normalizeFileName(name)]
	return ok
}
