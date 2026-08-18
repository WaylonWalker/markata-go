package contentindex

import "fmt"

var (
	ErrMissingSchema      = fmt.Errorf("content index is missing schema identity")
	ErrUnsupportedSchema  = fmt.Errorf("unsupported content index schema")
	ErrMissingVersion     = fmt.Errorf("content index is missing schema_version")
	ErrUnsupportedVersion = fmt.Errorf("unsupported content index schema version")
)

type ParseError struct {
	Field string
	Err   error
}

func (e *ParseError) Error() string { return fmt.Sprintf("content index %s: %v", e.Field, e.Err) }
func (e *ParseError) Unwrap() error { return e.Err }
