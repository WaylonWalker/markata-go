package palettes

import "strings"

// Variant represents the light/dark mode of a palette.
type Variant string

const (
	VariantLight Variant = "light"
	VariantDark  Variant = "dark"
)

// Palette represents a complete color palette with metadata and color definitions.
type Palette struct {
	// Metadata
	Name        string  `json:"name" toml:"name"`
	Variant     Variant `json:"variant" toml:"variant"`
	Author      string  `json:"author,omitempty" toml:"author,omitempty"`
	License     string  `json:"license,omitempty" toml:"license,omitempty"`
	Homepage    string  `json:"homepage,omitempty" toml:"homepage,omitempty"`
	Description string  `json:"description,omitempty" toml:"description,omitempty"`

	// Color definitions
	Colors     map[string]string `json:"colors" toml:"colors"`         // Raw colors (hex values)
	Semantic   map[string]string `json:"semantic" toml:"semantic"`     // Semantic mappings (references)
	Components map[string]string `json:"components" toml:"components"` // Component colors (references)

	// Source information (set after loading)
	Source     string `json:"-" toml:"-"` // "built-in", "user", "project"
	SourcePath string `json:"-" toml:"-"` // File path if loaded from file

	// Resolved colors cache (populated on first Resolve call)
	resolved map[string]string
}

// paletteFile represents the TOML file structure for a palette.
type paletteFile struct {
	Palette paletteSection `toml:"palette"`
}

type paletteSection struct {
	Name        string            `toml:"name"`
	Variant     string            `toml:"variant"`
	Author      string            `toml:"author,omitempty"`
	License     string            `toml:"license,omitempty"`
	Homepage    string            `toml:"homepage,omitempty"`
	Description string            `toml:"description,omitempty"`
	Colors      map[string]string `toml:"colors"`
	Semantic    map[string]string `toml:"semantic"`
	Components  map[string]string `toml:"components,omitempty"`
}

// NewPalette creates a new empty palette with the given name and variant.
func NewPalette(name string, variant Variant) *Palette {
	return &Palette{
		Name:       name,
		Variant:    variant,
		Colors:     make(map[string]string),
		Semantic:   make(map[string]string),
		Components: make(map[string]string),
		resolved:   make(map[string]string),
	}
}

// Resolve resolves a color name to its hex value.
// It checks in order: resolved cache, raw colors, semantic colors, component colors.
// Returns empty string if not found.
func (p *Palette) Resolve(name string) string {
	if p.resolved == nil {
		p.resolved = make(map[string]string)
	}

	// Check cache first
	if hex, ok := p.resolved[name]; ok {
		return hex
	}

	// Resolve the color
	hex, err := p.resolveColor(name, make(map[string]bool))
	if err != nil {
		return ""
	}

	// Cache and return
	p.resolved[name] = hex
	return hex
}

// resolveColor recursively resolves a color reference.
// visited tracks colors being resolved to detect circular references.
func (p *Palette) resolveColor(name string, visited map[string]bool) (string, error) {
	// Check for circular reference
	if visited[name] {
		return "", NewColorResolutionError(name, p.Name, "circular reference", ErrCircularReference)
	}
	visited[name] = true

	// Check if it's a hex color directly
	if isHexColor(name) {
		return normalizeHexColor(name), nil
	}

	// Check raw colors first (these are always hex values)
	if hex, ok := p.Colors[name]; ok {
		if isHexColor(hex) {
			return normalizeHexColor(hex), nil
		}
		// Raw color shouldn't reference other colors
		return "", NewColorResolutionError(name, p.Name, "raw color is not a hex value", ErrInvalidHexColor)
	}

	// Check semantic colors (can reference raw colors only)
	if ref, ok := p.Semantic[name]; ok {
		return p.resolveColor(ref, visited)
	}

	// Check component colors (can reference raw or semantic colors)
	if ref, ok := p.Components[name]; ok {
		return p.resolveColor(ref, visited)
	}

	return "", NewColorResolutionError(name, p.Name, "color not found", ErrUnknownColor)
}

// ResolveAll resolves all colors and returns a map of name -> hex.
// Returns an error if any color cannot be resolved.
func (p *Palette) ResolveAll() (map[string]string, error) {
	result := make(map[string]string)

	// Resolve all raw colors
	for name, hex := range p.Colors {
		if !isHexColor(hex) {
			return nil, NewColorResolutionError(name, p.Name, "raw color is not a hex value", ErrInvalidHexColor)
		}
		result[name] = normalizeHexColor(hex)
	}

	// Resolve all semantic colors
	for name := range p.Semantic {
		hex, err := p.resolveColor(name, make(map[string]bool))
		if err != nil {
			return nil, err
		}
		result[name] = hex
	}

	// Resolve all component colors
	for name := range p.Components {
		hex, err := p.resolveColor(name, make(map[string]bool))
		if err != nil {
			return nil, err
		}
		result[name] = hex
	}

	return result, nil
}

// Validate validates the palette structure and color references.
func (p *Palette) Validate() []error {
	var errs []error

	// Check required fields
	if p.Name == "" {
		errs = append(errs, NewValidationError("name", "palette name is required"))
	}

	if p.Variant != VariantLight && p.Variant != VariantDark {
		errs = append(errs, NewValidationError("variant", "variant must be 'light' or 'dark'"))
	}

	// Validate all raw colors are valid hex
	for name, hex := range p.Colors {
		if !isHexColor(hex) {
			errs = append(errs, NewValidationError("colors."+name, "invalid hex color: "+hex))
		}
	}

	// Validate semantic colors reference only raw colors
	for name, ref := range p.Semantic {
		if isHexColor(ref) {
			continue // Direct hex is allowed
		}
		if _, ok := p.Colors[ref]; !ok {
			// Check if it references another semantic (not allowed)
			if _, isSemantic := p.Semantic[ref]; isSemantic {
				errs = append(errs, NewValidationError("semantic."+name,
					"semantic colors cannot reference other semantic colors: "+ref))
			} else {
				errs = append(errs, NewValidationError("semantic."+name,
					"references unknown raw color: "+ref))
			}
		}
	}

	// Validate component colors reference raw or semantic colors
	for name, ref := range p.Components {
		if isHexColor(ref) {
			continue // Direct hex is allowed
		}
		if _, ok := p.Colors[ref]; ok {
			continue // Raw color reference is allowed
		}
		if _, ok := p.Semantic[ref]; ok {
			continue // Semantic color reference is allowed
		}
		// Check if it references another component (not allowed)
		if _, isComponent := p.Components[ref]; isComponent {
			errs = append(errs, NewValidationError("components."+name,
				"component colors cannot reference other component colors: "+ref))
		} else {
			errs = append(errs, NewValidationError("components."+name,
				"references unknown color: "+ref))
		}
	}

	return errs
}

// Clone creates a deep copy of the palette.
func (p *Palette) Clone() *Palette {
	clone := &Palette{
		Name:        p.Name,
		Variant:     p.Variant,
		Author:      p.Author,
		License:     p.License,
		Homepage:    p.Homepage,
		Description: p.Description,
		Source:      p.Source,
		SourcePath:  p.SourcePath,
		Colors:      make(map[string]string),
		Semantic:    make(map[string]string),
		Components:  make(map[string]string),
		resolved:    nil, // Don't copy cache
	}

	for k, v := range p.Colors {
		clone.Colors[k] = v
	}
	for k, v := range p.Semantic {
		clone.Semantic[k] = v
	}
	for k, v := range p.Components {
		clone.Components[k] = v
	}

	return clone
}

// isHexColor checks if a string is a valid hex color.
func isHexColor(s string) bool {
	return hexColorRegex.MatchString(s)
}

// normalizeHexColor ensures a hex color has # prefix and is lowercase.
func normalizeHexColor(hex string) string {
	hex = strings.ToLower(hex)
	if !strings.HasPrefix(hex, "#") {
		hex = "#" + hex
	}

	// Expand short form (#RGB -> #RRGGBB)
	if len(hex) == 4 {
		hex = "#" + string(hex[1]) + string(hex[1]) +
			string(hex[2]) + string(hex[2]) +
			string(hex[3]) + string(hex[3])
	}

	return hex
}

// PaletteInfo contains summary information about a palette for listing.
type PaletteInfo struct {
	Name        string  `json:"name"`
	Variant     Variant `json:"variant"`
	Description string  `json:"description,omitempty"`
	Author      string  `json:"author,omitempty"`
	Source      string  `json:"source"` // "built-in", "user", "project"
	Path        string  `json:"path,omitempty"`
}
