package models

import "fmt"

// FrontmatterParseError indicates an error parsing frontmatter from a post.
type FrontmatterParseError struct {
	Path    string
	Line    int
	Message string
	Err     error
}

func (e *FrontmatterParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("frontmatter parse error in %s at line %d: %s", e.Path, e.Line, e.Message)
	}
	return fmt.Sprintf("frontmatter parse error in %s: %s", e.Path, e.Message)
}

func (e *FrontmatterParseError) Unwrap() error {
	return e.Err
}

// NewFrontmatterParseError creates a new FrontmatterParseError.
func NewFrontmatterParseError(path, message string, err error) *FrontmatterParseError {
	return &FrontmatterParseError{
		Path:    path,
		Message: message,
		Err:     err,
	}
}

// FilterExpressionError indicates an error in a filter expression.
type FilterExpressionError struct {
	Expression string
	Position   int
	Message    string
	Err        error
}

func (e *FilterExpressionError) Error() string {
	if e.Position >= 0 {
		return fmt.Sprintf("filter expression error at position %d in '%s': %s", e.Position, e.Expression, e.Message)
	}
	return fmt.Sprintf("filter expression error in '%s': %s", e.Expression, e.Message)
}

func (e *FilterExpressionError) Unwrap() error {
	return e.Err
}

// NewFilterExpressionError creates a new FilterExpressionError.
func NewFilterExpressionError(expression, message string, err error) *FilterExpressionError {
	return &FilterExpressionError{
		Expression: expression,
		Position:   -1,
		Message:    message,
		Err:        err,
	}
}

// TemplateNotFoundError indicates a template file was not found.
type TemplateNotFoundError struct {
	Name       string
	SearchPath string
}

func (e *TemplateNotFoundError) Error() string {
	if e.SearchPath != "" {
		return fmt.Sprintf("template '%s' not found in %s", e.Name, e.SearchPath)
	}
	return fmt.Sprintf("template '%s' not found", e.Name)
}

// NewTemplateNotFoundError creates a new TemplateNotFoundError.
func NewTemplateNotFoundError(name, searchPath string) *TemplateNotFoundError {
	return &TemplateNotFoundError{
		Name:       name,
		SearchPath: searchPath,
	}
}

// TemplateSyntaxError indicates a syntax error in a template.
type TemplateSyntaxError struct {
	Name    string
	Line    int
	Column  int
	Message string
	Err     error
}

func (e *TemplateSyntaxError) Error() string {
	if e.Line > 0 && e.Column > 0 {
		return fmt.Sprintf("template syntax error in '%s' at line %d, column %d: %s", e.Name, e.Line, e.Column, e.Message)
	}
	if e.Line > 0 {
		return fmt.Sprintf("template syntax error in '%s' at line %d: %s", e.Name, e.Line, e.Message)
	}
	return fmt.Sprintf("template syntax error in '%s': %s", e.Name, e.Message)
}

func (e *TemplateSyntaxError) Unwrap() error {
	return e.Err
}

// NewTemplateSyntaxError creates a new TemplateSyntaxError.
func NewTemplateSyntaxError(name, message string, err error) *TemplateSyntaxError {
	return &TemplateSyntaxError{
		Name:    name,
		Message: message,
		Err:     err,
	}
}

// ConfigValidationError indicates a configuration validation error.
type ConfigValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ConfigValidationError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("config validation error for '%s' (value: %v): %s", e.Field, e.Value, e.Message)
	}
	return fmt.Sprintf("config validation error for '%s': %s", e.Field, e.Message)
}

// NewConfigValidationError creates a new ConfigValidationError.
func NewConfigValidationError(field string, value interface{}, message string) *ConfigValidationError {
	return &ConfigValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// PluginNotFoundError indicates a plugin was not found.
type PluginNotFoundError struct {
	Name      string
	Available []string
}

func (e *PluginNotFoundError) Error() string {
	if len(e.Available) > 0 {
		return fmt.Sprintf("plugin '%s' not found, available plugins: %v", e.Name, e.Available)
	}
	return fmt.Sprintf("plugin '%s' not found", e.Name)
}

// NewPluginNotFoundError creates a new PluginNotFoundError.
func NewPluginNotFoundError(name string, available []string) *PluginNotFoundError {
	return &PluginNotFoundError{
		Name:      name,
		Available: available,
	}
}

// PostProcessingError indicates an error during post processing.
type PostProcessingError struct {
	Path    string
	Stage   string
	Message string
	Err     error
}

func (e *PostProcessingError) Error() string {
	if e.Stage != "" {
		return fmt.Sprintf("error processing post '%s' during %s: %s", e.Path, e.Stage, e.Message)
	}
	return fmt.Sprintf("error processing post '%s': %s", e.Path, e.Message)
}

func (e *PostProcessingError) Unwrap() error {
	return e.Err
}

// NewPostProcessingError creates a new PostProcessingError.
func NewPostProcessingError(path, stage, message string, err error) *PostProcessingError {
	return &PostProcessingError{
		Path:    path,
		Stage:   stage,
		Message: message,
		Err:     err,
	}
}

// MermaidRenderError indicates an error rendering a Mermaid diagram.
type MermaidRenderError struct {
	Path       string
	DiagramID  string
	Mode       string
	Message    string
	Err        error
	Suggestion string
}

func (e *MermaidRenderError) Error() string {
	msg := fmt.Sprintf("mermaid render error in %s (%s mode): %s", e.Path, e.Mode, e.Message)
	if e.Suggestion != "" {
		msg += fmt.Sprintf("\n\nSuggestion:\n%s", e.Suggestion)
	}
	return msg
}

func (e *MermaidRenderError) Unwrap() error {
	return e.Err
}

// NewMermaidRenderError creates a new MermaidRenderError.
func NewMermaidRenderError(path, mode, message string, err error) *MermaidRenderError {
	return &MermaidRenderError{
		Path:    path,
		Mode:    mode,
		Message: message,
		Err:     err,
	}
}
