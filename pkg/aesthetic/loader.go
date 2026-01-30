package aesthetic

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

//go:embed aesthetics/*.toml
var builtinFS embed.FS

// sourceBuiltIn is the constant string for built-in aesthetic source.
const sourceBuiltIn = "built-in"

// Loader handles aesthetic discovery and loading from multiple sources.
type Loader struct {
	// Search paths in priority order (later paths override earlier ones)
	paths []string

	// Cache of loaded aesthetics
	cache map[string]*Aesthetic
}

// NewLoader creates a new Loader with default search paths.
// Search order: built-in, user config, project directory.
func NewLoader() *Loader {
	paths := []string{}

	// User config directory (~/.config/markata-go/aesthetics/)
	if configDir, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(configDir, "markata-go", "aesthetics"))
	}

	// Project directory (./aesthetics/)
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, "aesthetics"))
	}

	return &Loader{
		paths: paths,
		cache: make(map[string]*Aesthetic),
	}
}

// NewLoaderWithPaths creates a new Loader with custom search paths.
func NewLoaderWithPaths(paths []string) *Loader {
	return &Loader{
		paths: paths,
		cache: make(map[string]*Aesthetic),
	}
}

// AddPath adds a search path to the loader.
// Paths added later have higher priority.
func (l *Loader) AddPath(path string) {
	l.paths = append(l.paths, path)
}

// Load loads an aesthetic by name.
// It searches built-in aesthetics first, then search paths in order.
// Returns ErrAestheticNotFound if the aesthetic cannot be found.
func (l *Loader) Load(name string) (*Aesthetic, error) {
	// Normalize name
	normalized := normalizeAestheticName(name)

	// Check cache first
	if a, ok := l.cache[normalized]; ok {
		return a.Clone(), nil
	}

	// Try built-in aesthetics first
	if a, err := LoadBuiltin(name); err == nil {
		l.cache[normalized] = a
		return a.Clone(), nil
	}

	// Search paths in order (later paths override)
	var lastErr error
	for _, searchPath := range l.paths {
		a, err := l.loadFromPath(name, searchPath)
		if err == nil {
			l.cache[normalized] = a
			return a.Clone(), nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, NewAestheticLoadError(name, "", "aesthetic not found in any search path", ErrAestheticNotFound)
}

// loadFromPath attempts to load an aesthetic from a specific path.
func (l *Loader) loadFromPath(name, searchPath string) (*Aesthetic, error) {
	// Try exact name with .toml extension
	filePath := filepath.Join(searchPath, name+".toml")
	if _, err := os.Stat(filePath); err == nil {
		return LoadFromFile(filePath)
	}

	// Try normalized name (lowercase, hyphens)
	normalized := normalizeAestheticName(name)
	filePath = filepath.Join(searchPath, normalized+".toml")
	if _, err := os.Stat(filePath); err == nil {
		return LoadFromFile(filePath)
	}

	return nil, NewAestheticLoadError(name, searchPath, "file not found", ErrAestheticNotFound)
}

// LoadFromFile loads an aesthetic from a specific file path.
func LoadFromFile(path string) (*Aesthetic, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, NewAestheticLoadError("", path, "failed to read file", err)
	}

	return parseAesthetic(data, path, sourceFromPath(path))
}

// LoadBuiltin loads a built-in aesthetic by name.
func LoadBuiltin(name string) (*Aesthetic, error) {
	normalized := normalizeAestheticName(name)

	// Try to read from embedded filesystem
	data, err := builtinFS.ReadFile("aesthetics/" + normalized + ".toml")
	if err != nil {
		return nil, NewAestheticLoadError(name, "", "built-in aesthetic not found", ErrAestheticNotFound)
	}

	return parseAesthetic(data, "", sourceBuiltIn)
}

// rawTokens is an intermediate type for parsing TOML files with mixed types.
type rawTokens struct {
	Radius     map[string]any `toml:"radius"`
	Spacing    *SpacingTokens `toml:"spacing"`
	Border     map[string]any `toml:"border"`
	Shadow     map[string]any `toml:"shadow"`
	Typography map[string]any `toml:"typography"`
}

// rawAesthetic is an intermediate type for parsing TOML files.
type rawAesthetic struct {
	Name        string    `toml:"name"`
	Description string    `toml:"description"`
	Tokens      rawTokens `toml:"tokens"`
}

// parseAesthetic parses TOML data into an Aesthetic.
func parseAesthetic(data []byte, path, source string) (*Aesthetic, error) {
	var raw rawAesthetic
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, NewAestheticParseError(path, "failed to parse TOML", err)
	}

	// Convert raw aesthetic to typed Aesthetic
	a := &Aesthetic{
		Name:        raw.Name,
		Description: raw.Description,
		Source:      source,
		SourcePath:  path,
		Tokens: Tokens{
			Radius:     convertToStringMap(raw.Tokens.Radius),
			Spacing:    raw.Tokens.Spacing,
			Border:     convertToStringMap(raw.Tokens.Border),
			Shadow:     convertToStringMap(raw.Tokens.Shadow),
			Typography: convertToStringMap(raw.Tokens.Typography),
		},
	}

	// Initialize nil maps
	if a.Tokens.Radius == nil {
		a.Tokens.Radius = make(map[string]string)
	}
	if a.Tokens.Border == nil {
		a.Tokens.Border = make(map[string]string)
	}
	if a.Tokens.Shadow == nil {
		a.Tokens.Shadow = make(map[string]string)
	}
	if a.Tokens.Typography == nil {
		a.Tokens.Typography = make(map[string]string)
	}

	// Validate the loaded aesthetic
	if errs := a.Validate(); len(errs) > 0 {
		return nil, NewAestheticLoadError(a.Name, path, fmt.Sprintf("validation failed: %v", errs[0]), errs[0])
	}

	return a, nil
}

// convertToStringMap converts a map[string]any to map[string]string.
// Values are converted using fmt.Sprintf("%v", v).
func convertToStringMap(m map[string]any) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}

// BuiltinNames returns the names of all built-in aesthetics.
func BuiltinNames() []string {
	entries, err := builtinFS.ReadDir("aesthetics")
	if err != nil {
		return nil
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".toml") {
			names = append(names, strings.TrimSuffix(name, ".toml"))
		}
	}

	sort.Strings(names)
	return names
}

// HasBuiltin checks if a built-in aesthetic with the given name exists.
func HasBuiltin(name string) bool {
	normalized := normalizeAestheticName(name)
	_, err := builtinFS.ReadFile("aesthetics/" + normalized + ".toml")
	return err == nil
}

// DiscoverBuiltin returns info for all built-in aesthetics.
func DiscoverBuiltin() []AestheticInfo {
	names := BuiltinNames()
	infos := make([]AestheticInfo, 0, len(names))

	for _, name := range names {
		a, err := LoadBuiltin(name)
		if err != nil {
			continue
		}
		infos = append(infos, AestheticInfo{
			Name:        a.Name,
			Description: a.Description,
			Source:      sourceBuiltIn,
		})
	}

	return infos
}

// Discover finds all available aesthetics across all sources.
// Returns aesthetic info sorted by source priority.
func (l *Loader) Discover() ([]AestheticInfo, error) {
	infos := make(map[string]AestheticInfo)

	// Discover built-in aesthetics first
	builtinInfos := DiscoverBuiltin()
	for _, info := range builtinInfos {
		infos[normalizeAestheticName(info.Name)] = info
	}

	// Discover from search paths (later paths override)
	for _, searchPath := range l.paths {
		pathInfos, err := discoverFromPath(searchPath)
		if err != nil {
			continue // Skip paths that don't exist or can't be read
		}
		for _, info := range pathInfos {
			infos[normalizeAestheticName(info.Name)] = info // Override with higher priority
		}
	}

	// Convert map to sorted slice
	result := make([]AestheticInfo, 0, len(infos))
	for _, info := range infos {
		result = append(result, info)
	}

	// Sort by name
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// discoverFromPath discovers aesthetics in a specific directory.
func discoverFromPath(searchPath string) ([]AestheticInfo, error) {
	entries, err := os.ReadDir(searchPath)
	if err != nil {
		return nil, err
	}

	infos := make([]AestheticInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		filePath := filepath.Join(searchPath, entry.Name())
		a, err := LoadFromFile(filePath)
		if err != nil {
			continue // Skip invalid aesthetics
		}

		infos = append(infos, AestheticInfo{
			Name:        a.Name,
			Description: a.Description,
			Source:      a.Source,
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

// normalizeAestheticName converts an aesthetic name to a normalized form.
// e.g., "Brutal Design" -> "brutal-design"
func normalizeAestheticName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	return name
}

// ListAesthetics returns info for all available aesthetics (built-in and discovered).
func ListAesthetics() []AestheticInfo {
	loader := NewLoader()
	infos, err := loader.Discover()
	if err != nil {
		return nil
	}
	return infos
}

// DefaultLoader is the default aesthetic loader instance.
var DefaultLoader = NewLoader()

// Load loads an aesthetic using the default loader.
func Load(name string) (*Aesthetic, error) {
	return DefaultLoader.Load(name)
}

// Discover discovers all aesthetics using the default loader.
func Discover() ([]AestheticInfo, error) {
	return DefaultLoader.Discover()
}

// ClearCache clears the aesthetic cache.
func (l *Loader) ClearCache() {
	l.cache = make(map[string]*Aesthetic)
}
