package aesthetic

import "errors"

// Common errors returned by the aesthetic package.
var (
	// ErrAestheticNotFound is returned when an aesthetic cannot be found.
	ErrAestheticNotFound = errors.New("aesthetic not found")

	// ErrInvalidAesthetic is returned when an aesthetic file is malformed.
	ErrInvalidAesthetic = errors.New("invalid aesthetic")
)

// LoadError provides context for aesthetic loading failures.
type LoadError struct {
	Name    string // Aesthetic name that failed to load
	Path    string // Path attempted (if applicable)
	Message string // Human-readable error message
	Err     error  // Underlying error
}

func (e *LoadError) Error() string {
	if e.Path != "" {
		return "failed to load aesthetic " + e.Name + " from " + e.Path + ": " + e.Message
	}
	return "failed to load aesthetic " + e.Name + ": " + e.Message
}

func (e *LoadError) Unwrap() error {
	return e.Err
}

// NewLoadError creates a new LoadError.
func NewLoadError(name, path, message string, err error) *LoadError {
	return &LoadError{
		Name:    name,
		Path:    path,
		Message: message,
		Err:     err,
	}
}

// ParseError provides context for aesthetic parsing failures.
type ParseError struct {
	Path    string // File path that failed to parse
	Message string // Human-readable error message
	Err     error  // Underlying error
}

func (e *ParseError) Error() string {
	if e.Path != "" {
		return "failed to parse aesthetic at " + e.Path + ": " + e.Message
	}
	return "failed to parse aesthetic: " + e.Message
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// NewParseError creates a new ParseError.
func NewParseError(path, message string, err error) *ParseError {
	return &ParseError{
		Path:    path,
		Message: message,
		Err:     err,
	}
}

// ValidationError represents an aesthetic validation failure.
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
