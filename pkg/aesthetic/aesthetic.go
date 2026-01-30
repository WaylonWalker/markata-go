package aesthetic

// SpacingTokens holds spacing-related design tokens.
type SpacingTokens struct {
	Scale float64 `json:"scale" yaml:"scale" toml:"scale"`
}

// Tokens holds all design token categories for an aesthetic.
type Tokens struct {
	Radius     map[string]string `json:"radius,omitempty" yaml:"radius,omitempty" toml:"radius,omitempty"`
	Spacing    *SpacingTokens    `json:"spacing,omitempty" yaml:"spacing,omitempty" toml:"spacing,omitempty"`
	Border     map[string]string `json:"border,omitempty" yaml:"border,omitempty" toml:"border,omitempty"`
	Shadow     map[string]string `json:"shadow,omitempty" yaml:"shadow,omitempty" toml:"shadow,omitempty"`
	Typography map[string]string `json:"typography,omitempty" yaml:"typography,omitempty" toml:"typography,omitempty"`
}

// Aesthetic represents a complete design token set with metadata.
type Aesthetic struct {
	// Metadata
	Name        string `json:"name" yaml:"name" toml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty" toml:"description,omitempty"`

	// Tokens organized by category
	Tokens Tokens `json:"tokens" yaml:"tokens" toml:"tokens"`

	// Legacy flat maps for CSS generation compatibility
	// These are populated from Tokens when GenerateCSS is called
	Typography map[string]string `json:"-" yaml:"-" toml:"-"`
	Spacing    map[string]string `json:"-" yaml:"-" toml:"-"`
	Borders    map[string]string `json:"-" yaml:"-" toml:"-"`
	Shadows    map[string]string `json:"-" yaml:"-" toml:"-"`

	// Source information (set after loading)
	Source     string `json:"-" yaml:"-" toml:"-"` // "built-in", "user", "project"
	SourcePath string `json:"-" yaml:"-" toml:"-"` // File path if loaded from file
}

// NewAesthetic creates a new empty aesthetic with initialized maps.
func NewAesthetic(name string) *Aesthetic {
	return &Aesthetic{
		Name: name,
		Tokens: Tokens{
			Radius:     make(map[string]string),
			Spacing:    &SpacingTokens{Scale: 1.0},
			Border:     make(map[string]string),
			Shadow:     make(map[string]string),
			Typography: make(map[string]string),
		},
	}
}

// Get retrieves a token value by type and key.
// Returns empty string if not found.
func (a *Aesthetic) Get(tokenType, tokenKey string) string {
	switch tokenType {
	case "radius":
		if a.Tokens.Radius == nil {
			return ""
		}
		return a.Tokens.Radius[tokenKey]
	case "border":
		if a.Tokens.Border == nil {
			return ""
		}
		return a.Tokens.Border[tokenKey]
	case "shadow":
		if a.Tokens.Shadow == nil {
			return ""
		}
		return a.Tokens.Shadow[tokenKey]
	case "typography":
		if a.Tokens.Typography == nil {
			return ""
		}
		return a.Tokens.Typography[tokenKey]
	default:
		return ""
	}
}

// GetWithDefault retrieves a token value, returning defaultVal if not found.
func (a *Aesthetic) GetWithDefault(tokenType, tokenKey, defaultVal string) string {
	val := a.Get(tokenType, tokenKey)
	if val == "" {
		return defaultVal
	}
	return val
}

// GetSpacingScale returns the spacing scale multiplier.
// Returns 1.0 if no spacing is configured.
func (a *Aesthetic) GetSpacingScale() float64 {
	if a.Tokens.Spacing == nil {
		return 1.0
	}
	if a.Tokens.Spacing.Scale == 0 {
		return 1.0
	}
	return a.Tokens.Spacing.Scale
}

// Clone creates a deep copy of the aesthetic.
func (a *Aesthetic) Clone() *Aesthetic {
	clone := &Aesthetic{
		Name:        a.Name,
		Description: a.Description,
		Source:      a.Source,
		SourcePath:  a.SourcePath,
		Tokens: Tokens{
			Radius:     make(map[string]string),
			Border:     make(map[string]string),
			Shadow:     make(map[string]string),
			Typography: make(map[string]string),
		},
	}

	// Copy spacing
	if a.Tokens.Spacing != nil {
		clone.Tokens.Spacing = &SpacingTokens{
			Scale: a.Tokens.Spacing.Scale,
		}
	}

	// Copy maps
	for k, v := range a.Tokens.Radius {
		clone.Tokens.Radius[k] = v
	}
	for k, v := range a.Tokens.Border {
		clone.Tokens.Border[k] = v
	}
	for k, v := range a.Tokens.Shadow {
		clone.Tokens.Shadow[k] = v
	}
	for k, v := range a.Tokens.Typography {
		clone.Tokens.Typography[k] = v
	}

	// Copy legacy flat maps if populated
	if a.Typography != nil {
		clone.Typography = make(map[string]string)
		for k, v := range a.Typography {
			clone.Typography[k] = v
		}
	}
	if a.Spacing != nil {
		clone.Spacing = make(map[string]string)
		for k, v := range a.Spacing {
			clone.Spacing[k] = v
		}
	}
	if a.Borders != nil {
		clone.Borders = make(map[string]string)
		for k, v := range a.Borders {
			clone.Borders[k] = v
		}
	}
	if a.Shadows != nil {
		clone.Shadows = make(map[string]string)
		for k, v := range a.Shadows {
			clone.Shadows[k] = v
		}
	}

	return clone
}

// Validate checks if the aesthetic has required fields.
// Returns a slice of validation errors (empty if valid).
func (a *Aesthetic) Validate() []error {
	var errs []error

	if a.Name == "" {
		errs = append(errs, NewValidationError("name", "aesthetic name is required"))
	}

	return errs
}

// Merge combines this aesthetic with an override aesthetic.
// The override values take precedence. Returns a new aesthetic.
func (a *Aesthetic) Merge(override *Aesthetic) *Aesthetic {
	merged := a.Clone()

	// Override name if provided
	if override.Name != "" {
		merged.Name = override.Name
	}
	if override.Description != "" {
		merged.Description = override.Description
	}

	// Merge token maps
	if override.Tokens.Radius != nil {
		for k, v := range override.Tokens.Radius {
			merged.Tokens.Radius[k] = v
		}
	}
	if override.Tokens.Border != nil {
		for k, v := range override.Tokens.Border {
			merged.Tokens.Border[k] = v
		}
	}
	if override.Tokens.Shadow != nil {
		for k, v := range override.Tokens.Shadow {
			merged.Tokens.Shadow[k] = v
		}
	}
	if override.Tokens.Typography != nil {
		for k, v := range override.Tokens.Typography {
			merged.Tokens.Typography[k] = v
		}
	}

	// Merge spacing
	if override.Tokens.Spacing != nil {
		merged.Tokens.Spacing = &SpacingTokens{
			Scale: override.Tokens.Spacing.Scale,
		}
	}

	return merged
}

// Info contains summary information about an aesthetic for listing.
type Info struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source"` // "built-in", "user", "project"
	Path        string `json:"path,omitempty"`
}
