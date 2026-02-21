package palettes

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// Loader handles palette discovery and loading from multiple sources.
type Loader struct {
	// Search paths in priority order (later paths override earlier ones)
	paths []string

	// Cache of loaded palettes
	cache map[string]*Palette

	// dynamic palettes (added programmatically)
	dynamic map[string]*Palette
}

// NewLoader creates a new Loader with default search paths.
// Search order: built-in, user config, project directory.
func NewLoader() *Loader {
	paths := []string{}

	// User config directory (~/.config/markata-go/palettes/)
	if configDir, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(configDir, "markata-go", "palettes"))
	}

	// Project directory (./palettes/)
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, "palettes"))
	}

	return &Loader{
		paths:   paths,
		cache:   make(map[string]*Palette),
		dynamic: make(map[string]*Palette),
	}
}

// NewLoaderWithPaths creates a new Loader with custom search paths.
func NewLoaderWithPaths(paths []string) *Loader {
	return &Loader{
		paths:   paths,
		cache:   make(map[string]*Palette),
		dynamic: make(map[string]*Palette),
	}
}

// AddPalette dynamically adds a palette to the loader.
func (l *Loader) AddPalette(name string, p *Palette) {
	if l.dynamic == nil {
		l.dynamic = make(map[string]*Palette)
	}
	l.dynamic[name] = p
}

// AddPath adds a search path to the loader.
// Paths added later have higher priority.
func (l *Loader) AddPath(path string) {
	l.paths = append(l.paths, path)
}

// Load loads a palette by name.
// It searches in priority order: project directory, user config, then built-in.
// This allows vendored palettes to override built-in ones.
// Returns ErrPaletteNotFound if the palette cannot be found.
func (l *Loader) Load(name string) (*Palette, error) {
	// Check cache first
	if p, ok := l.cache[name]; ok {
		return p.Clone(), nil
	}

	// Check dynamic palettes
	if p, ok := l.dynamic[name]; ok {
		return p.Clone(), nil
	}

	// Search paths in reverse order (project > user > built-in)
	// Later paths in the slice have higher priority
	for i := len(l.paths) - 1; i >= 0; i-- {
		p, err := l.loadFromPath(name, l.paths[i])
		if err == nil {
			l.cache[name] = p
			return p.Clone(), nil
		}
	}

	// Fall back to built-in palettes
	if p, err := LoadBuiltin(name); err == nil {
		l.cache[name] = p
		return p.Clone(), nil
	}

	return nil, NewPaletteLoadError(name, "", "palette not found in any search path", ErrPaletteNotFound)
}

// loadFromPath attempts to load a palette from a specific path.
func (l *Loader) loadFromPath(name, searchPath string) (*Palette, error) {
	// Try exact name with .toml extension
	filePath := filepath.Join(searchPath, name+".toml")
	if _, err := os.Stat(filePath); err == nil {
		return LoadFromFile(filePath)
	}

	// Try normalized name (lowercase, hyphens)
	normalized := normalizeFileName(name)
	filePath = filepath.Join(searchPath, normalized+".toml")
	if _, err := os.Stat(filePath); err == nil {
		return LoadFromFile(filePath)
	}

	return nil, NewPaletteLoadError(name, searchPath, "file not found", ErrPaletteNotFound)
}

// LoadFromFile loads a palette from a specific file path.
func LoadFromFile(path string) (*Palette, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, NewPaletteLoadError("", path, "failed to read file", err)
	}

	var pf paletteFile
	if err := toml.Unmarshal(data, &pf); err != nil {
		return nil, NewPaletteLoadError("", path, "failed to parse TOML", err)
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
		Source:      sourceFromPath(path),
		SourcePath:  path,
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

	// Validate the loaded palette
	if errs := p.Validate(); len(errs) > 0 {
		return nil, NewPaletteLoadError(p.Name, path, fmt.Sprintf("validation failed: %v", errs[0]), errs[0])
	}

	return p, nil
}

// Discover finds all available palettes across all sources.
// Returns palette info sorted by source priority (built-in first, then user, then project).
func (l *Loader) Discover() ([]PaletteInfo, error) {
	infos := make(map[string]PaletteInfo)

	// Discover dynamic palettes
	for name, p := range l.dynamic {
		infos[name] = PaletteInfo{
			Name:        p.Name,
			Variant:     p.Variant,
			Description: p.Description,
			Author:      p.Author,
			Source:      p.Source,
		}
	}

	// Discover built-in palettes first
	builtinInfos := DiscoverBuiltin()
	for _, info := range builtinInfos {
		infos[info.Name] = info
	}

	// Discover from search paths (later paths override)
	for _, searchPath := range l.paths {
		pathInfos, err := discoverFromPath(searchPath)
		if err != nil {
			continue // Skip paths that don't exist or can't be read
		}
		for _, info := range pathInfos {
			infos[info.Name] = info // Override with higher priority
		}
	}

	// Convert map to sorted slice
	result := make([]PaletteInfo, 0, len(infos))
	for _, info := range infos {
		result = append(result, info)
	}

	// Sort by name
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// DiscoverByVariant finds all palettes with a specific variant.
func (l *Loader) DiscoverByVariant(variant Variant) ([]PaletteInfo, error) {
	all, err := l.Discover()
	if err != nil {
		return nil, err
	}

	var result []PaletteInfo
	for _, info := range all {
		if info.Variant == variant {
			result = append(result, info)
		}
	}

	return result, nil
}

// discoverFromPath discovers palettes in a specific directory.
func discoverFromPath(searchPath string) ([]PaletteInfo, error) {
	entries, err := os.ReadDir(searchPath)
	if err != nil {
		return nil, err
	}

	infos := make([]PaletteInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		filePath := filepath.Join(searchPath, entry.Name())
		p, err := LoadFromFile(filePath)
		if err != nil {
			continue // Skip invalid palettes
		}

		infos = append(infos, PaletteInfo{
			Name:        p.Name,
			Variant:     p.Variant,
			Description: p.Description,
			Author:      p.Author,
			Source:      p.Source,
			Path:        filePath,
		})
	}

	return infos, nil
}

// sourceFromPath determines the source type based on file path.
func sourceFromPath(path string) string {
	// Check if it's in user config directory
	if configDir, err := os.UserConfigDir(); err == nil {
		if strings.HasPrefix(path, filepath.Join(configDir, "markata-go")) {
			return "user"
		}
	}

	// Check if it's in the current project
	if cwd, err := os.Getwd(); err == nil {
		if strings.HasPrefix(path, cwd) {
			return "project"
		}
	}

	return sourceBuiltIn
}

// sourceBuiltIn is the constant string for built-in palette source.
const sourceBuiltIn = "built-in"

// normalizeFileName converts a palette name to a normalized file name.
// e.g., "Catppuccin Mocha" -> "catppuccin-mocha"
func normalizeFileName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	return name
}

// ClearCache clears the palette cache.
func (l *Loader) ClearCache() {
	l.cache = make(map[string]*Palette)
}

// DefaultLoader is the default palette loader instance.
var DefaultLoader = NewLoader()

// Load loads a palette using the default loader.
func Load(name string) (*Palette, error) {
	return DefaultLoader.Load(name)
}

// Discover discovers all palettes using the default loader.
func Discover() ([]PaletteInfo, error) {
	return DefaultLoader.Discover()
}
