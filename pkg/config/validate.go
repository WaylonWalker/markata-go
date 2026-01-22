package config

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/example/markata-go/pkg/models"
)

// ValidationError represents a configuration validation error.
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
	for i, feed := range config.Feeds {
		feedWithDefaults := feed
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

// isWarning returns true if the error is a ValidationError with IsWarn=true.
func isWarning(err error) bool {
	var ve ValidationError
	if errors.As(err, &ve) {
		return ve.IsWarn
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
func SplitErrorsAndWarnings(errs []error) (errors, warnings []error) {
	for _, err := range errs {
		if isWarning(err) {
			warnings = append(warnings, err)
		} else {
			errors = append(errors, err)
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
