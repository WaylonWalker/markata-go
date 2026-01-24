package config

import (
	"fmt"
	"regexp"
	"strings"
)

// ConfigError represents a detailed configuration error with file position
// and fix suggestion. The name is intentionally verbose to clearly indicate
// this is a configuration-specific error type with position tracking.
//
//nolint:revive // ConfigError name is intentional for clarity
type ConfigError struct {
	File    string   // Path to the configuration file
	Line    int      // 1-based line number
	Column  int      // 1-based column number
	Field   string   // Configuration field name (e.g., "url", "feeds[0].items_per_page")
	Value   string   // The problematic value
	Message string   // Error message describing the problem
	Fix     string   // Suggested fix
	IsWarn  bool     // If true, this is a warning rather than an error
	Context []string // Surrounding lines for context
}

// Error implements the error interface with a detailed, user-friendly message.
func (e *ConfigError) Error() string {
	var sb strings.Builder

	// Write header with file location
	prefix := "Error"
	if e.IsWarn {
		prefix = "Warning"
	}

	switch {
	case e.File != "" && e.Line > 0 && e.Column > 0:
		sb.WriteString(fmt.Sprintf("%s: configuration error in %s:%d:%d\n", prefix, e.File, e.Line, e.Column))
	case e.File != "" && e.Line > 0:
		sb.WriteString(fmt.Sprintf("%s: configuration error in %s:%d\n", prefix, e.File, e.Line))
	case e.File != "":
		sb.WriteString(fmt.Sprintf("%s: configuration error in %s\n", prefix, e.File))
	default:
		sb.WriteString(fmt.Sprintf("%s: configuration error\n", prefix))
	}

	// Write context lines if available
	if len(e.Context) > 0 {
		sb.WriteString("\n")
		for _, line := range e.Context {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	// Write the error message
	sb.WriteString("\n")
	sb.WriteString(e.Field)
	sb.WriteString(": ")
	sb.WriteString(e.Message)
	sb.WriteString("\n")

	// Write fix suggestion if available
	if e.Fix != "" {
		sb.WriteString("\nFix: ")
		sb.WriteString(e.Fix)
		sb.WriteString("\n")
	}

	return sb.String()
}

// ShortError returns a concise single-line error message.
func (e *ConfigError) ShortError() string {
	prefix := "error"
	if e.IsWarn {
		prefix = "warning"
	}

	if e.File != "" && e.Line > 0 {
		return fmt.Sprintf("%s:%d: config %s: %s: %s", e.File, e.Line, prefix, e.Field, e.Message)
	}
	return fmt.Sprintf("config %s: %s: %s", prefix, e.Field, e.Message)
}

// ConfigErrors collects multiple configuration errors for batch reporting.
//
//nolint:revive // ConfigErrors name is intentional for clarity
type ConfigErrors struct {
	Errors []*ConfigError
}

// Error implements the error interface with a summary of all errors.
func (e *ConfigErrors) Error() string {
	if len(e.Errors) == 0 {
		return ""
	}

	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("configuration validation failed with %d issues:\n\n", len(e.Errors)))

	for i, err := range e.Errors {
		if i > 0 {
			sb.WriteString("\n" + strings.Repeat("-", 60) + "\n\n")
		}
		sb.WriteString(err.Error())
	}

	return sb.String()
}

// Add adds an error to the collection.
func (e *ConfigErrors) Add(err *ConfigError) {
	e.Errors = append(e.Errors, err)
}

// HasErrors returns true if there are any actual errors (not warnings).
func (e *ConfigErrors) HasErrors() bool {
	for _, err := range e.Errors {
		if !err.IsWarn {
			return true
		}
	}
	return false
}

// HasWarnings returns true if there are any warnings.
func (e *ConfigErrors) HasWarnings() bool {
	for _, err := range e.Errors {
		if err.IsWarn {
			return true
		}
	}
	return false
}

// SplitErrorsAndWarnings separates errors and warnings.
func (e *ConfigErrors) SplitErrorsAndWarnings() (errors, warnings []*ConfigError) {
	for _, err := range e.Errors {
		if err.IsWarn {
			warnings = append(warnings, err)
		} else {
			errors = append(errors, err)
		}
	}
	return
}

// FieldPosition tracks the location of a field in a configuration file.
type FieldPosition struct {
	Line   int
	Column int
}

// PositionTracker helps find field positions in configuration files.
type PositionTracker struct {
	lines    []string
	filePath string
}

// NewPositionTracker creates a new position tracker for the given file content.
func NewPositionTracker(content []byte, filePath string) *PositionTracker {
	return &PositionTracker{
		lines:    strings.Split(string(content), "\n"),
		filePath: filePath,
	}
}

// FindFieldPosition finds the line and column of a field in the config file.
// It searches for patterns like `field = value` or `field: value`.
func (p *PositionTracker) FindFieldPosition(fieldName string) FieldPosition {
	// Handle nested fields like "feeds[0].items_per_page" or "markata-go.url"
	// Extract the leaf field name for searching
	leafField := fieldName
	if idx := strings.LastIndex(fieldName, "."); idx != -1 {
		leafField = fieldName[idx+1:]
	}

	// Remove array indices
	if idx := strings.Index(leafField, "["); idx != -1 {
		leafField = leafField[:idx]
	}

	// Build regex pattern for TOML/YAML style assignment
	pattern := regexp.MustCompile(fmt.Sprintf(`(?i)^\s*%s\s*[=:]`, regexp.QuoteMeta(leafField)))

	for i, line := range p.lines {
		if matches := pattern.FindStringIndex(line); matches != nil {
			// Find the column where the value starts (after = or :)
			col := strings.IndexAny(line, "=:") + 1
			if col > 0 {
				// Skip whitespace after = or :
				for col < len(line) && (line[col] == ' ' || line[col] == '\t') {
					col++
				}
			}
			return FieldPosition{
				Line:   i + 1, // 1-based line number
				Column: col + 1,
			}
		}
	}

	return FieldPosition{Line: 0, Column: 0}
}

// ExtractContext extracts surrounding lines for context display.
// The target line is marked with "> " prefix, others with "  ".
func (p *PositionTracker) ExtractContext(targetLine, contextLines int) []string {
	if targetLine <= 0 || len(p.lines) == 0 {
		return nil
	}

	start := targetLine - contextLines - 1
	if start < 0 {
		start = 0
	}

	end := targetLine + contextLines
	if end > len(p.lines) {
		end = len(p.lines)
	}

	context := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		lineNum := i + 1
		prefix := "  "
		if lineNum == targetLine {
			prefix = "> "
		}
		context = append(context, fmt.Sprintf("%s%4d | %s", prefix, lineNum, p.lines[i]))
	}

	return context
}

// GetLine returns the content of a specific line (1-based).
func (p *PositionTracker) GetLine(lineNum int) string {
	if lineNum <= 0 || lineNum > len(p.lines) {
		return ""
	}
	return p.lines[lineNum-1]
}

// FilePath returns the file path associated with this tracker.
func (p *PositionTracker) FilePath() string {
	return p.filePath
}

// fixSuggestions contains common fix suggestions for configuration errors.
var fixSuggestions = map[string]func(field, value string) string{
	"url_no_scheme": func(_, value string) string {
		// Remove any existing protocol prefix that might be malformed
		cleanValue := strings.TrimPrefix(value, "//")
		return fmt.Sprintf("url = \"https://%s\"", cleanValue)
	},
	"url_invalid_scheme": func(_, value string) string {
		// Replace the scheme with https
		if idx := strings.Index(value, "://"); idx != -1 {
			return fmt.Sprintf("url = \"https://%s\"", value[idx+3:])
		}
		return fmt.Sprintf("url = \"https://%s\"", value)
	},
	"url_no_host": func(_, _ string) string {
		return "url = \"https://example.com\""
	},
	"negative_value": func(field, _ string) string {
		return fmt.Sprintf("%s = 0", field)
	},
	"empty_patterns": func(_, _ string) string {
		return "patterns = [\"**/*.md\"]"
	},
	"no_formats": func(_, _ string) string {
		return "[formats]\nhtml = true"
	},
}

// GetFixSuggestion returns a fix suggestion for a known error type.
func GetFixSuggestion(errorType, field, value string) string {
	if fn, ok := fixSuggestions[errorType]; ok {
		return fn(field, value)
	}
	return ""
}

// NewConfigError creates a new ConfigError with position tracking.
func NewConfigError(tracker *PositionTracker, field, value, message string, isWarn bool) *ConfigError {
	err := &ConfigError{
		Field:   field,
		Value:   value,
		Message: message,
		IsWarn:  isWarn,
	}

	if tracker != nil {
		err.File = tracker.FilePath()
		pos := tracker.FindFieldPosition(field)
		err.Line = pos.Line
		err.Column = pos.Column
		if pos.Line > 0 {
			err.Context = tracker.ExtractContext(pos.Line, 2)
		}
	}

	return err
}

// NewConfigErrorWithFix creates a ConfigError with a fix suggestion.
func NewConfigErrorWithFix(tracker *PositionTracker, field, value, message, fix string, isWarn bool) *ConfigError {
	err := NewConfigError(tracker, field, value, message, isWarn)
	err.Fix = fix
	return err
}
