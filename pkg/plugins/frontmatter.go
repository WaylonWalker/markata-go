// Package plugins provides core plugins for the markata-go static site generator.
package plugins

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrInvalidFrontmatter indicates the frontmatter could not be parsed.
var ErrInvalidFrontmatter = errors.New("invalid frontmatter")

// frontmatterDelimiter is the standard YAML frontmatter delimiter.
const frontmatterDelimiter = "---"

// ExtractFrontmatter splits content into frontmatter YAML string and body content.
// Returns:
//   - frontmatter: the raw YAML string between --- delimiters (empty if no frontmatter)
//   - body: the content after the frontmatter
//   - err: error if frontmatter is malformed
//
// Edge cases:
//   - No frontmatter (doesn't start with ---): returns empty frontmatter, full content as body
//   - Empty frontmatter (---, then ---): returns empty frontmatter, content after second ---
//   - Unclosed frontmatter: returns error
func ExtractFrontmatter(content string) (frontmatter string, body string, err error) {
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// Check if content starts with frontmatter delimiter
	if !strings.HasPrefix(content, frontmatterDelimiter) {
		// No frontmatter - entire content is body
		return "", content, nil
	}

	// Find the end of the opening delimiter line
	afterOpening := content[len(frontmatterDelimiter):]

	// The opening delimiter must be on its own line
	if len(afterOpening) > 0 && afterOpening[0] != '\n' {
		// Not a valid frontmatter start (e.g., "---something")
		return "", content, nil
	}

	// Skip the newline after opening delimiter
	if len(afterOpening) > 0 && afterOpening[0] == '\n' {
		afterOpening = afterOpening[1:]
	}

	// Handle empty frontmatter case (--- immediately follows)
	if strings.HasPrefix(afterOpening, frontmatterDelimiter) {
		// Empty frontmatter
		remaining := afterOpening[len(frontmatterDelimiter):]
		if strings.HasPrefix(remaining, "\n") {
			remaining = remaining[1:]
		}
		return "", remaining, nil
	}

	// Find the closing delimiter (must be on its own line)
	closingIdx := strings.Index(afterOpening, "\n"+frontmatterDelimiter)
	if closingIdx == -1 {
		// Check if content ends with the delimiter on its own line
		if strings.HasSuffix(afterOpening, "\n"+frontmatterDelimiter) {
			closingIdx = len(afterOpening) - len(frontmatterDelimiter) - 1
		} else {
			// Unclosed frontmatter
			return "", "", fmt.Errorf("%w: unclosed frontmatter delimiter", ErrInvalidFrontmatter)
		}
	}

	// Extract frontmatter content (everything before the closing delimiter line)
	frontmatter = afterOpening[:closingIdx]

	// Extract body (skip the newline, the closing delimiter, and optional trailing newline)
	remaining := afterOpening[closingIdx+1:] // Skip the newline before ---
	remaining = strings.TrimPrefix(remaining, frontmatterDelimiter)
	if strings.HasPrefix(remaining, "\n") {
		remaining = remaining[1:] // Skip newline after ---
	}
	body = remaining

	return frontmatter, body, nil
}

// ParseFrontmatter parses content containing optional YAML frontmatter.
// Returns the parsed metadata as a map, the body content, and any error.
//
// The frontmatter must be delimited by --- at the start of the content.
//
// Example:
//
//	---
//	title: My Post
//	date: 2024-01-15
//	tags:
//	  - go
//	  - programming
//	---
//	# Content here
//
// Edge cases handled:
//   - No frontmatter: returns empty map, full content as body
//   - Empty frontmatter: returns empty map, content after delimiters
//   - Invalid YAML: returns error with context
func ParseFrontmatter(content string) (map[string]interface{}, string, error) {
	frontmatter, body, err := ExtractFrontmatter(content)
	if err != nil {
		return nil, "", err
	}

	// No frontmatter case
	if frontmatter == "" {
		return make(map[string]interface{}), body, nil
	}

	// Parse the YAML
	metadata := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err)
	}

	// Handle nil result from empty YAML
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return metadata, body, nil
}

// GetString extracts a string value from metadata, returning empty string if not found or wrong type.
func GetString(metadata map[string]interface{}, key string) string {
	if v, ok := metadata[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetBool extracts a boolean value from metadata, returning defaultVal if not found or wrong type.
// Handles YAML boolean values: true, false, yes, no, on, off.
func GetBool(metadata map[string]interface{}, key string, defaultVal bool) bool {
	v, ok := metadata[key]
	if !ok {
		return defaultVal
	}

	switch b := v.(type) {
	case bool:
		return b
	case string:
		switch strings.ToLower(b) {
		case "true", "yes", "on":
			return true
		case "false", "no", "off":
			return false
		}
	}
	return defaultVal
}

// GetStringSlice extracts a string slice from metadata.
// Handles both []interface{} (common from YAML) and []string.
func GetStringSlice(metadata map[string]interface{}, key string) []string {
	v, ok := metadata[key]
	if !ok {
		return nil
	}

	switch s := v.(type) {
	case []string:
		return s
	case []interface{}:
		result := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return nil
}
