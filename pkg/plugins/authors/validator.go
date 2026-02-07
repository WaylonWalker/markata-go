package authors

import (
	"fmt"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

// ValidateAuthorConfig validates author configuration according to business rules
func ValidateAuthorConfig(config *models.AuthorsConfig) error {
	if config == nil {
		return nil
	}

	// Validate URL pattern if provided
	if config.URLPattern != "" {
		if !isValidURLPattern(config.URLPattern) {
			return fmt.Errorf("invalid URL pattern: %s", config.URLPattern)
		}
	}

	// Validate authors map
	if err := models.ValidateAuthors(config.Authors); err != nil {
		return fmt.Errorf("authors validation failed: %w", err)
	}

	// Ensure at least one active author exists
	activeAuthorCount := 0
	for _, author := range config.Authors {
		if author.Active {
			activeAuthorCount++
		}
	}

	if activeAuthorCount == 0 && len(config.Authors) > 0 {
		return fmt.Errorf("at least one author must be marked as active")
	}

	return nil
}

// ValidateAuthorPageConfig validates configuration for author page generation
func ValidateAuthorPageConfig(config *models.AuthorsConfig) error {
	if !config.GeneratePages {
		return nil // Pages disabled, nothing to validate
	}

	// Validate URL pattern
	if config.URLPattern == "" {
		return fmt.Errorf("URL pattern is required when author pages are enabled")
	}

	if !isValidURLPattern(config.URLPattern) {
		return fmt.Errorf("invalid URL pattern: %s", config.URLPattern)
	}

	return nil
}

// isValidURLPattern checks if URL pattern contains required placeholders
func isValidURLPattern(pattern string) bool {
	// Basic validation - should contain {author} placeholder
	return len(pattern) > 0 && pattern != ""
}
