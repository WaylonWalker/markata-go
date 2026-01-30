package aesthetic

import "errors"

// Common errors returned by the aesthetic package.
var (
	// ErrAestheticNotFound is returned when an aesthetic cannot be found.
	ErrAestheticNotFound = errors.New("aesthetic not found")

	// ErrInvalidAesthetic is returned when an aesthetic file is malformed.
	ErrInvalidAesthetic = errors.New("invalid aesthetic")
)

// AestheticLoadError provides context for aesthetic loading failures.
type AestheticLoadError struct {
	Name    string // Aesthetic name that failed to load
	Path    string // Path attempted (if applicable)
	Message string // Human-readable error message
	Err     error  // Underlying error
}

func (e *AestheticLoadError) Error() string {
	if e.Path != "" {
		return "failed to load aesthetic " + e.Name + " from " + e.Path + ": " + e.Message
	}
	return "failed to load aesthetic " + e.Name + ": " + e.Message
}

func (e *AestheticLoadError) Unwrap() error {
	return e.Err
}

// NewAestheticLoadError creates a new AestheticLoadError.
func NewAestheticLoadError(name, path, message string, err error) *AestheticLoadError {
	return &AestheticLoadError{
		Name:    name,
		Path:    path,
		Message: message,
		Err:     err,
	}
}

// AestheticParseError provides context for aesthetic parsing failures.
type AestheticParseError struct {
	Path    string // File path that failed to parse
	Message string // Human-readable error message
	Err     error  // Underlying error
}

func (e *AestheticParseError) Error() string {
	if e.Path != "" {
		return "failed to parse aesthetic at " + e.Path + ": " + e.Message
	}
	return "failed to parse aesthetic: " + e.Message
}

func (e *AestheticParseError) Unwrap() error {
	return e.Err
}

// NewAestheticParseError creates a new AestheticParseError.
func NewAestheticParseError(path, message string, err error) *AestheticParseError {
	return &AestheticParseError{
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
