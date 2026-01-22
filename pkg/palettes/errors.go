package palettes

import "errors"

// Common errors returned by the palettes package.
var (
	// ErrPaletteNotFound is returned when a palette cannot be found.
	ErrPaletteNotFound = errors.New("palette not found")

	// ErrInvalidPalette is returned when a palette file is malformed.
	ErrInvalidPalette = errors.New("invalid palette")

	// ErrCircularReference is returned when color references form a cycle.
	ErrCircularReference = errors.New("circular color reference detected")

	// ErrUnknownColor is returned when a color reference cannot be resolved.
	ErrUnknownColor = errors.New("unknown color reference")

	// ErrInvalidHexColor is returned when a hex color value is malformed.
	ErrInvalidHexColor = errors.New("invalid hex color")
)

// PaletteLoadError provides context for palette loading failures.
type PaletteLoadError struct {
	Name    string // Palette name that failed to load
	Path    string // Path attempted (if applicable)
	Message string // Human-readable error message
	Err     error  // Underlying error
}

func (e *PaletteLoadError) Error() string {
	if e.Path != "" {
		return "failed to load palette " + e.Name + " from " + e.Path + ": " + e.Message
	}
	return "failed to load palette " + e.Name + ": " + e.Message
}

func (e *PaletteLoadError) Unwrap() error {
	return e.Err
}

// NewPaletteLoadError creates a new PaletteLoadError.
func NewPaletteLoadError(name, path, message string, err error) *PaletteLoadError {
	return &PaletteLoadError{
		Name:    name,
		Path:    path,
		Message: message,
		Err:     err,
	}
}

// ColorResolutionError provides context for color resolution failures.
type ColorResolutionError struct {
	Color   string // Color name that failed to resolve
	Palette string // Palette name
	Message string // Human-readable error message
	Err     error  // Underlying error
}

func (e *ColorResolutionError) Error() string {
	return "failed to resolve color " + e.Color + " in palette " + e.Palette + ": " + e.Message
}

func (e *ColorResolutionError) Unwrap() error {
	return e.Err
}

// NewColorResolutionError creates a new ColorResolutionError.
func NewColorResolutionError(color, palette, message string, err error) *ColorResolutionError {
	return &ColorResolutionError{
		Color:   color,
		Palette: palette,
		Message: message,
		Err:     err,
	}
}

// ValidationError represents a palette validation failure.
type ValidationError struct {
	Field   string // Field that failed validation
	Message string // Human-readable error message
}

func (e *ValidationError) Error() string {
	return "validation error for " + e.Field + ": " + e.Message
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}
