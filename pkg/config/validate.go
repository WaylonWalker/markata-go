package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

// ValidationError represents a configuration validation error.
// This type is kept for backward compatibility.
type ValidationError struct {
	Field   string
	Message string
	IsWarn  bool // If true, this is a warning rather than an error
}

func (e ValidationError) Error() string {
	prefix := "error"
	if e.IsWarn {
		prefix = "warning"
	}
	return fmt.Sprintf("config %s: %s: %s", prefix, e.Field, e.Message)
}

// ValidateConfig validates a configuration and returns any errors or warnings.
// Errors are returned first, followed by warnings.
// The returned slice will be nil if no issues are found.
func ValidateConfig(config *models.Config) []error {
	if config == nil {
		return []error{errors.New("config is nil")}
	}

	var errs []error

	// Validate URL format if provided
	if config.URL != "" {
		if err := validateURL(config.URL); err != nil {
			errs = append(errs, ValidationError{
				Field:   "url",
				Message: err.Error(),
			})
		}
	}

	// Validate concurrency
	if config.Concurrency < 0 {
		errs = append(errs, ValidationError{
			Field:   "concurrency",
			Message: "must be >= 0 (0 means auto-detect)",
		})
	}

	// Warn on empty glob patterns
	if len(config.GlobConfig.Patterns) == 0 {
		errs = append(errs, ValidationError{
			Field:   "glob.patterns",
			Message: "no glob patterns specified, no files will be processed",
			IsWarn:  true,
		})
	}

	// Validate feed configurations
	// Apply feed defaults before validation so we can check effective values
	for i := range config.Feeds {
		feedWithDefaults := config.Feeds[i]
		feedWithDefaults.ApplyDefaults(config.FeedDefaults)
		feedErrs := validateFeedConfig(i, &feedWithDefaults)
		errs = append(errs, feedErrs...)
	}

	// Validate feed defaults
	if config.FeedDefaults.ItemsPerPage < 0 {
		errs = append(errs, ValidationError{
			Field:   "feed_defaults.items_per_page",
			Message: "must be >= 0",
		})
	}
	if config.FeedDefaults.OrphanThreshold < 0 {
		errs = append(errs, ValidationError{
			Field:   "feed_defaults.orphan_threshold",
			Message: "must be >= 0",
		})
	}

	// Sort errors first, then warnings
	sortErrors(errs)

	return errs
}

// ValidateConfigWithPositions validates a configuration with file position tracking.
// It returns a ConfigErrors collection with detailed error information including
// file paths, line numbers, and fix suggestions.
func ValidateConfigWithPositions(config *models.Config, tracker *PositionTracker) *ConfigErrors {
	configErrors := &ConfigErrors{}

	if config == nil {
		configErrors.Add(&ConfigError{
			Field:   "config",
			Message: "configuration is nil",
		})
		return configErrors
	}

	// Validate URL format if provided
	if config.URL != "" {
		validateURLWithDetails(config.URL, tracker, configErrors)
	}

	// Validate concurrency
	if config.Concurrency < 0 {
		configErrors.Add(NewConfigErrorWithFix(
			tracker,
			"concurrency",
			fmt.Sprintf("%d", config.Concurrency),
			"must be >= 0 (0 means auto-detect)",
			GetFixSuggestion("negative_value", "concurrency", ""),
			false,
		))
	}

	// Warn on empty glob patterns
	if len(config.GlobConfig.Patterns) == 0 {
		configErrors.Add(NewConfigErrorWithFix(
			tracker,
			"glob.patterns",
			"[]",
			"no glob patterns specified, no files will be processed",
			GetFixSuggestion("empty_patterns", "", ""),
			true,
		))
	}

	// Validate feed configurations
	for i := range config.Feeds {
		feedWithDefaults := config.Feeds[i]
		feedWithDefaults.ApplyDefaults(config.FeedDefaults)
		validateFeedConfigWithPositions(i, &feedWithDefaults, tracker, configErrors)
	}

	// Validate feed defaults
	if config.FeedDefaults.ItemsPerPage < 0 {
		configErrors.Add(NewConfigErrorWithFix(
			tracker,
			"feed_defaults.items_per_page",
			fmt.Sprintf("%d", config.FeedDefaults.ItemsPerPage),
			"must be >= 0",
			GetFixSuggestion("negative_value", "items_per_page", ""),
			false,
		))
	}
	if config.FeedDefaults.OrphanThreshold < 0 {
		configErrors.Add(NewConfigErrorWithFix(
			tracker,
			"feed_defaults.orphan_threshold",
			fmt.Sprintf("%d", config.FeedDefaults.OrphanThreshold),
			"must be >= 0",
			GetFixSuggestion("negative_value", "orphan_threshold", ""),
			false,
		))
	}

	return configErrors
}

// validateURLWithDetails validates a URL and adds detailed errors to configErrors.
func validateURLWithDetails(rawURL string, tracker *PositionTracker, configErrors *ConfigErrors) {
	u, err := url.Parse(rawURL)
	if err != nil {
		configErrors.Add(NewConfigErrorWithFix(
			tracker,
			"url",
			rawURL,
			fmt.Sprintf("invalid URL format: %v", err),
			GetFixSuggestion("url_no_scheme", "url", rawURL),
			false,
		))
		return
	}

	if u.Scheme == "" {
		configErrors.Add(NewConfigErrorWithFix(
			tracker,
			"url",
			rawURL,
			"URL must include a scheme (e.g., https://)",
			GetFixSuggestion("url_no_scheme", "url", rawURL),
			false,
		))
		return
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		configErrors.Add(NewConfigErrorWithFix(
			tracker,
			"url",
			rawURL,
			fmt.Sprintf("URL scheme must be http or https, got %q", u.Scheme),
			GetFixSuggestion("url_invalid_scheme", "url", rawURL),
			false,
		))
		return
	}

	if u.Host == "" {
		configErrors.Add(NewConfigErrorWithFix(
			tracker,
			"url",
			rawURL,
			"URL must include a host",
			GetFixSuggestion("url_no_host", "url", rawURL),
			false,
		))
	}
}

// validateFeedConfigWithPositions validates a feed configuration with position tracking.
func validateFeedConfigWithPositions(index int, feed *models.FeedConfig, tracker *PositionTracker, configErrors *ConfigErrors) {
	prefix := fmt.Sprintf("feeds[%d]", index)

	// Validate items_per_page
	if feed.ItemsPerPage < 0 {
		configErrors.Add(NewConfigErrorWithFix(
			tracker,
			prefix+".items_per_page",
			fmt.Sprintf("%d", feed.ItemsPerPage),
			"must be >= 0",
			GetFixSuggestion("negative_value", "items_per_page", ""),
			false,
		))
	}

	// Validate orphan_threshold
	if feed.OrphanThreshold < 0 {
		configErrors.Add(NewConfigErrorWithFix(
			tracker,
			prefix+".orphan_threshold",
			fmt.Sprintf("%d", feed.OrphanThreshold),
			"must be >= 0",
			GetFixSuggestion("negative_value", "orphan_threshold", ""),
			false,
		))
	}

	// Warn if no output formats are enabled
	if !hasAnyFormat(feed.Formats) {
		configErrors.Add(NewConfigErrorWithFix(
			tracker,
			prefix+".formats",
			"{}",
			"no output formats enabled, feed will not produce any output",
			GetFixSuggestion("no_formats", "", ""),
			true,
		))
	}
}

// validateURL validates a URL string.
func validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if u.Scheme == "" {
		return errors.New("URL must include a scheme (e.g., https://)")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got %q", u.Scheme)
	}

	if u.Host == "" {
		return errors.New("URL must include a host")
	}

	return nil
}

// validateFeedConfig validates a single feed configuration.
func validateFeedConfig(index int, feed *models.FeedConfig) []error {
	var errs []error
	prefix := fmt.Sprintf("feeds[%d]", index)

	// Note: Empty slug is allowed - it represents the home page feed (index.html)
	// No validation needed for slug field

	// Validate items_per_page
	if feed.ItemsPerPage < 0 {
		errs = append(errs, ValidationError{
			Field:   prefix + ".items_per_page",
			Message: "must be >= 0",
		})
	}

	// Validate orphan_threshold
	if feed.OrphanThreshold < 0 {
		errs = append(errs, ValidationError{
			Field:   prefix + ".orphan_threshold",
			Message: "must be >= 0",
		})
	}

	// Warn if no output formats are enabled
	if !hasAnyFormat(feed.Formats) {
		errs = append(errs, ValidationError{
			Field:   prefix + ".formats",
			Message: "no output formats enabled, feed will not produce any output",
			IsWarn:  true,
		})
	}

	return errs
}

// hasAnyFormat returns true if any feed format is enabled.
func hasAnyFormat(formats models.FeedFormats) bool {
	return formats.HTML || formats.RSS || formats.Atom ||
		formats.JSON || formats.Markdown || formats.Text
}

// sortErrors sorts validation errors so that errors come before warnings.
func sortErrors(errs []error) {
	if len(errs) <= 1 {
		return
	}

	// Simple bubble sort - stable and fine for small slices
	for i := 0; i < len(errs)-1; i++ {
		for j := 0; j < len(errs)-i-1; j++ {
			// If current is warning and next is error, swap
			currWarn := isWarning(errs[j])
			nextWarn := isWarning(errs[j+1])
			if currWarn && !nextWarn {
				errs[j], errs[j+1] = errs[j+1], errs[j]
			}
		}
	}
}

// isWarning returns true if the error is a ValidationError with IsWarn=true
// or a ConfigError with IsWarn=true.
func isWarning(err error) bool {
	var ve ValidationError
	if errors.As(err, &ve) {
		return ve.IsWarn
	}
	var ce *ConfigError
	if errors.As(err, &ce) {
		return ce.IsWarn
	}
	return false
}

// HasErrors returns true if the error slice contains any actual errors (not warnings).
func HasErrors(errs []error) bool {
	for _, err := range errs {
		if !isWarning(err) {
			return true
		}
	}
	return false
}

// HasWarnings returns true if the error slice contains any warnings.
func HasWarnings(errs []error) bool {
	for _, err := range errs {
		if isWarning(err) {
			return true
		}
	}
	return false
}

// SplitErrorsAndWarnings separates errors and warnings into separate slices.
func SplitErrorsAndWarnings(errs []error) (errsOut, warnsOut []error) {
	for _, err := range errs {
		if isWarning(err) {
			warnsOut = append(warnsOut, err)
		} else {
			errsOut = append(errsOut, err)
		}
	}
	return
}

// ValidateAndWarn validates config and logs warnings but only returns actual errors.
// This is useful when you want to proceed with warnings but stop on errors.
func ValidateAndWarn(config *models.Config, warnFunc func(error)) []error {
	allErrs := ValidateConfig(config)
	actualErrors, warnings := SplitErrorsAndWarnings(allErrs)

	for _, w := range warnings {
		if warnFunc != nil {
			warnFunc(w)
		}
	}

	return actualErrors
}

// FormatConfigError formats a ConfigError for CLI display with colors and context.
func FormatConfigError(err *ConfigError) string {
	var sb strings.Builder

	// Header with location
	switch {
	case err.File != "" && err.Line > 0 && err.Column > 0:
		sb.WriteString(fmt.Sprintf("Error: configuration error in %s:%d:%d\n", err.File, err.Line, err.Column))
	case err.File != "" && err.Line > 0:
		sb.WriteString(fmt.Sprintf("Error: configuration error in %s:%d\n", err.File, err.Line))
	case err.File != "":
		sb.WriteString(fmt.Sprintf("Error: configuration error in %s\n", err.File))
	default:
		sb.WriteString("Error: configuration error\n")
	}

	// Context with arrow pointing to problematic line
	if len(err.Context) > 0 {
		sb.WriteString("\n")
		for _, line := range err.Context {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	// Error details
	sb.WriteString("\n")
	sb.WriteString(err.Field)
	sb.WriteString(": ")
	sb.WriteString(err.Message)
	sb.WriteString("\n")

	// Fix suggestion
	if err.Fix != "" {
		sb.WriteString("\nFix: ")
		sb.WriteString(err.Fix)
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatConfigErrors formats multiple ConfigErrors for CLI display.
func FormatConfigErrors(configErrors *ConfigErrors) string {
	if configErrors == nil || len(configErrors.Errors) == 0 {
		return ""
	}

	if len(configErrors.Errors) == 1 {
		return FormatConfigError(configErrors.Errors[0])
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Error: configuration validation failed with %d issues\n", len(configErrors.Errors)))

	for i, err := range configErrors.Errors {
		if i > 0 {
			sb.WriteString("\n")
			sb.WriteString(strings.Repeat("-", 60))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
		sb.WriteString(FormatConfigError(err))
	}

	return sb.String()
}
