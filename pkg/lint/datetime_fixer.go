// Package lint provides markdown linting functionality for markata-go.
package lint

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	dps "github.com/markusmobius/go-dateparser"
)

// Date format and configuration constants.
const (
	// DefaultDateFormat is the ISO 8601 date format (YYYY-MM-DD).
	DefaultDateFormat = "2006-01-02"

	// DefaultDateTimeFormat is the ISO 8601 datetime format (RFC3339 simplified).
	DefaultDateTimeFormat = "2006-01-02T15:04:05Z"

	// AmbiguousFormatMDY interprets ambiguous dates as month/day/year (US format).
	AmbiguousFormatMDY = "mdy"

	// AmbiguousFormatDMY interprets ambiguous dates as day/month/year (European format).
	AmbiguousFormatDMY = "dmy"

	// MissingDateSkip returns empty string for missing dates.
	MissingDateSkip = "skip"

	// MissingDateToday uses current date for missing dates.
	MissingDateToday = "today"

	// MissingDateError returns an error for missing dates.
	MissingDateError = "error"

	// Natural language time units.
	unitDay   = "day"
	unitWeek  = "week"
	unitMonth = "month"
	unitYear  = "year"
)

// DateTimeFixerConfig configures the datetime fixer behavior.
type DateTimeFixerConfig struct {
	// Format is the output date format. Default: "2006-01-02" (ISO 8601 date)
	Format string

	// DateTimeFormat is the output datetime format. Default: "2006-01-02T15:04:05Z" (RFC3339 simplified)
	DateTimeFormat string

	// PreserveTime indicates whether to preserve time components when present in input.
	// Default: true
	PreserveTime bool

	// AmbiguousFormat specifies how to interpret ambiguous dates like "01/02/2024".
	// "mdy" interprets as month/day/year (US format)
	// "dmy" interprets as day/month/year (European format)
	// Default: "mdy"
	AmbiguousFormat string

	// MissingDate specifies behavior when date is empty or missing.
	// "today" uses current date
	// "skip" returns empty string
	// "error" returns an error
	// Default: "skip"
	MissingDate string

	// WarnFuture logs a warning for future dates
	WarnFuture bool

	// WarnOld logs a warning for dates older than 50 years
	WarnOld bool

	// ReferenceTime is the time to use for relative date calculations.
	// If nil, time.Now() is used. Useful for testing.
	ReferenceTime *time.Time
}

// DefaultDateTimeFixerConfig returns a configuration with sensible defaults.
func DefaultDateTimeFixerConfig() DateTimeFixerConfig {
	return DateTimeFixerConfig{
		Format:          DefaultDateFormat,
		DateTimeFormat:  DefaultDateTimeFormat,
		PreserveTime:    true,
		AmbiguousFormat: AmbiguousFormatMDY,
		MissingDate:     MissingDateSkip,
		WarnFuture:      false,
		WarnOld:         false,
	}
}

// DateTimeFixer normalizes various date formats to ISO 8601.
type DateTimeFixer struct {
	config DateTimeFixerConfig

	// Compiled regex patterns for date parsing
	isoDateTimeRegex  *regexp.Regexp
	isoDateRegex      *regexp.Regexp
	slashMDYRegex     *regexp.Regexp
	slashYMDRegex     *regexp.Regexp
	writtenMonthRegex *regexp.Regexp
	writtenDayRegex   *regexp.Regexp
	rfc2822Regex      *regexp.Regexp

	// go-dateparser for comprehensive multi-locale parsing
	dateParser *dps.Parser
}

// NewDateTimeFixer creates a new DateTimeFixer with the given configuration.
func NewDateTimeFixer(config DateTimeFixerConfig) *DateTimeFixer {
	// Apply defaults for empty config values
	if config.Format == "" {
		config.Format = DefaultDateFormat
	}
	if config.DateTimeFormat == "" {
		config.DateTimeFormat = DefaultDateTimeFormat
	}
	// PreserveTime defaults to true (zero value is false, so we need explicit handling)
	// We use a different approach: check if this is a zero config
	// For backward compatibility, if Format is set but PreserveTime was not explicitly set,
	// we default to true. The only way to disable is to explicitly set PreserveTime = false.
	// Since Go doesn't distinguish between "not set" and "set to false", we always default
	// to preserving time unless explicitly disabled in calling code.
	if config.AmbiguousFormat == "" {
		config.AmbiguousFormat = AmbiguousFormatMDY
	}
	if config.MissingDate == "" {
		config.MissingDate = MissingDateSkip
	}

	// Configure go-dateparser with appropriate parser types
	dateParser := &dps.Parser{
		ParserTypes: []dps.ParserType{
			dps.AbsoluteTime,
			dps.NoSpacesTime,
			dps.Timestamp,
			dps.RelativeTime,
			dps.CustomFormat,
		},
	}

	return &DateTimeFixer{
		config:     config,
		dateParser: dateParser,
		// ISO 8601 with time: 2024-01-15T10:30:00Z or 2024-01-15T10:30:00+05:00
		isoDateTimeRegex: regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})T(\d{2}):(\d{2}):(\d{2})(.*)$`),
		// ISO 8601 date only: 2024-01-15
		isoDateRegex: regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})$`),
		// Slash format MM/DD/YYYY or DD/MM/YYYY or YYYY/MM/DD
		slashMDYRegex: regexp.MustCompile(`^(\d{1,2})/(\d{1,2})/(\d{4})$`),
		slashYMDRegex: regexp.MustCompile(`^(\d{4})/(\d{1,2})/(\d{1,2})$`),
		// Written month format: January 15, 2024 or Jan 15, 2024
		writtenMonthRegex: regexp.MustCompile(`^([A-Za-z]+)\s+(\d{1,2}),?\s+(\d{4})$`),
		// Written day first: 15 January 2024 or 15 Jan 2024
		writtenDayRegex: regexp.MustCompile(`^(\d{1,2})\s+([A-Za-z]+),?\s+(\d{4})$`),
		// RFC 2822: Mon, 15 Jan 2024 or 15 Jan 2024
		rfc2822Regex: regexp.MustCompile(`^(?:[A-Za-z]{3},?\s+)?(\d{1,2})\s+([A-Za-z]{3})\s+(\d{4})(?:\s+\d{2}:\d{2}(?::\d{2})?(?:\s*[+-]\d{4})?)?$`),
	}
}

// Fix attempts to parse the input date string and return it in ISO 8601 format.
// Returns the normalized date string and any error encountered.
func (f *DateTimeFixer) Fix(dateStr string) (string, error) {
	dateStr = strings.TrimSpace(dateStr)

	// Handle empty input
	if dateStr == "" {
		return f.handleMissingDate()
	}

	// Try natural language first (these never have time components)
	if result, ok := f.parseNaturalLanguage(dateStr); ok {
		return result, nil
	}

	// Try RFC3339 first (handles timezone offsets properly)
	if t, hasTime, ok := f.parseRFC3339(dateStr); ok {
		return f.formatResult(t, hasTime), nil
	}

	// Try ISO datetime without timezone (our custom parser)
	if t, hasTime, ok := f.parseISODateTime(dateStr); ok {
		return f.formatResult(t, hasTime), nil
	}

	// Date-only parsers (fast parsers for common formats)
	dateOnlyParsers := []func(string) (time.Time, bool){
		f.parseISODate,
		f.parseSlashYMD,
		f.parseSlashMDY,
		f.parseWrittenMonth,
		f.parseWrittenDay,
		f.parseRFC2822,
	}

	for _, parser := range dateOnlyParsers {
		if t, ok := parser(dateStr); ok {
			return f.formatDate(t), nil
		}
	}

	// Fallback to go-dateparser for complex/multi-locale date formats
	if t, ok := f.parseWithDateParser(dateStr); ok {
		return f.formatDate(t), nil
	}

	return "", fmt.Errorf("unable to parse date: %q", dateStr)
}

// handleMissingDate handles the case when the input date is empty.
func (f *DateTimeFixer) handleMissingDate() (string, error) {
	switch f.config.MissingDate {
	case MissingDateToday:
		return f.formatDate(f.now()), nil
	case MissingDateSkip:
		return "", nil
	case MissingDateError:
		return "", fmt.Errorf("missing date value")
	default:
		return "", nil
	}
}

// now returns the current time or the reference time if set.
func (f *DateTimeFixer) now() time.Time {
	if f.config.ReferenceTime != nil {
		return *f.config.ReferenceTime
	}
	return time.Now()
}

// formatDate formats a time.Time to the configured output format (date only).
func (f *DateTimeFixer) formatDate(t time.Time) string {
	return t.Format(f.config.Format)
}

// formatResult formats a time.Time based on whether time components should be preserved.
func (f *DateTimeFixer) formatResult(t time.Time, hasTime bool) string {
	if hasTime && f.config.PreserveTime {
		return t.UTC().Format(f.config.DateTimeFormat)
	}
	return t.Format(f.config.Format)
}

// parseNaturalLanguage handles natural language date expressions.
func (f *DateTimeFixer) parseNaturalLanguage(dateStr string) (string, bool) {
	lower := strings.ToLower(dateStr)
	now := f.now()

	// Simple natural language mappings
	naturalDates := map[string]time.Time{
		"today":     now,
		"now":       now,
		"yesterday": now.AddDate(0, 0, -1),
		"tomorrow":  now.AddDate(0, 0, 1),
	}

	if t, ok := naturalDates[lower]; ok {
		return f.formatDate(t), true
	}

	// Handle "last week", "last month", "last year"
	if strings.HasPrefix(lower, "last ") {
		suffix := strings.TrimPrefix(lower, "last ")
		var t time.Time
		switch suffix {
		case unitWeek:
			t = now.AddDate(0, 0, -7)
		case unitMonth:
			t = now.AddDate(0, -1, 0)
		case unitYear:
			t = now.AddDate(-1, 0, 0)
		default:
			return "", false
		}
		return f.formatDate(t), true
	}

	// Handle "next week", "next month", "next year"
	if strings.HasPrefix(lower, "next ") {
		suffix := strings.TrimPrefix(lower, "next ")
		var t time.Time
		switch suffix {
		case unitWeek:
			t = now.AddDate(0, 0, 7)
		case unitMonth:
			t = now.AddDate(0, 1, 0)
		case unitYear:
			t = now.AddDate(1, 0, 0)
		default:
			return "", false
		}
		return f.formatDate(t), true
	}

	// Handle "N days ago", "N weeks ago", "N months ago", "N years ago"
	agoPattern := regexp.MustCompile(`^(\d+)\s+(day|week|month|year)s?\s+ago$`)
	if matches := agoPattern.FindStringSubmatch(lower); matches != nil {
		n, err := strconv.Atoi(matches[1])
		if err != nil {
			return "", false
		}
		unit := matches[2]
		var t time.Time
		switch unit {
		case unitDay:
			t = now.AddDate(0, 0, -n)
		case unitWeek:
			t = now.AddDate(0, 0, -n*7)
		case unitMonth:
			t = now.AddDate(0, -n, 0)
		case unitYear:
			t = now.AddDate(-n, 0, 0)
		default:
			return "", false
		}
		return f.formatDate(t), true
	}

	return "", false
}

// parseWithDateParser uses go-dateparser as a fallback for complex/multi-locale date parsing.
// This provides support for 200+ locales and comprehensive natural language date formats
// that aren't covered by our fast built-in parsers.
func (f *DateTimeFixer) parseWithDateParser(dateStr string) (time.Time, bool) {
	// Skip dates that look like they might be ISO format with invalid values
	// These should fail rather than be "fixed" by go-dateparser's lenient parsing
	if f.looksLikeInvalidISODate(dateStr) {
		return time.Time{}, false
	}

	// Configure go-dateparser with date order based on ambiguous format setting
	dateOrder := dps.MDY
	if f.config.AmbiguousFormat == AmbiguousFormatDMY {
		dateOrder = dps.DMY
	}

	// Build configuration for this parse
	cfg := &dps.Configuration{
		DateOrder:           dateOrder,
		PreferredDayOfMonth: dps.First,
		StrictParsing:       false,
		Languages:           []string{"en", "fr", "es", "de", "it", "pt", "nl", "ru", "zh", "ja"},
	}

	// Set reference time for relative date calculations
	if f.config.ReferenceTime != nil {
		cfg.CurrentTime = *f.config.ReferenceTime
	}

	result, err := f.dateParser.Parse(cfg, dateStr)
	if err != nil {
		return time.Time{}, false
	}

	return result.Time, true
}

// looksLikeInvalidISODate checks if a string looks like an ISO date format but
// contains invalid date values or is incomplete. This prevents go-dateparser from
// "fixing" dates like 2024-02-30 to 2024-03-02 or completing partial dates.
func (f *DateTimeFixer) looksLikeInvalidISODate(dateStr string) bool {
	// Check for partial ISO date pattern (YYYY-MM without day)
	partialISOPattern := regexp.MustCompile(`^(\d{4})-(\d{1,2})$`)
	if partialISOPattern.MatchString(dateStr) {
		return true
	}

	// Check for YYYY-MM-DD pattern
	isoPattern := regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})`)
	matches := isoPattern.FindStringSubmatch(dateStr)
	if matches == nil {
		return false
	}

	year, err1 := strconv.Atoi(matches[1])
	month, err2 := strconv.Atoi(matches[2])
	day, err3 := strconv.Atoi(matches[3])
	if err1 != nil || err2 != nil || err3 != nil {
		return false
	}

	// If it looks like ISO format but has invalid values, reject it
	return !isValidDate(year, month, day)
}

// parseISODateTime parses ISO 8601 datetime format (2024-01-15T10:30:00Z).
// Returns the parsed time, whether time components were present, and success flag.
func (f *DateTimeFixer) parseISODateTime(dateStr string) (time.Time, bool, bool) {
	matches := f.isoDateTimeRegex.FindStringSubmatch(dateStr)
	if matches == nil {
		return time.Time{}, false, false
	}

	year, month, day, ok := parseYearMonthDay(matches[1], matches[2], matches[3])
	if !ok {
		return time.Time{}, false, false
	}

	hour, err1 := strconv.Atoi(matches[4])
	minute, err2 := strconv.Atoi(matches[5])
	second, err3 := strconv.Atoi(matches[6])
	if err1 != nil || err2 != nil || err3 != nil {
		return time.Time{}, false, false
	}

	if !isValidDate(year, month, day) {
		return time.Time{}, false, false
	}

	// Check if time components are non-zero (i.e., meaningful time was present)
	hasTime := hour != 0 || minute != 0 || second != 0

	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC), hasTime, true
}

// parseISODate parses ISO 8601 date format (2024-01-15).
func (f *DateTimeFixer) parseISODate(dateStr string) (time.Time, bool) {
	matches := f.isoDateRegex.FindStringSubmatch(dateStr)
	if matches == nil {
		return time.Time{}, false
	}

	year, month, day, ok := parseYearMonthDay(matches[1], matches[2], matches[3])
	if !ok {
		return time.Time{}, false
	}

	if !isValidDate(year, month, day) {
		return time.Time{}, false
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), true
}

// parseRFC3339 parses RFC 3339 format using Go's built-in parser.
// Returns the parsed time, whether time components were present, and success flag.
func (f *DateTimeFixer) parseRFC3339(dateStr string) (time.Time, bool, bool) {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return time.Time{}, false, false
	}
	// RFC3339 always has time components; check if they're non-zero
	hasTime := t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0
	return t, hasTime, true
}

// parseSlashYMD parses YYYY/MM/DD format.
func (f *DateTimeFixer) parseSlashYMD(dateStr string) (time.Time, bool) {
	matches := f.slashYMDRegex.FindStringSubmatch(dateStr)
	if matches == nil {
		return time.Time{}, false
	}

	year, month, day, ok := parseYearMonthDay(matches[1], matches[2], matches[3])
	if !ok {
		return time.Time{}, false
	}

	if !isValidDate(year, month, day) {
		return time.Time{}, false
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), true
}

// parseSlashMDY parses MM/DD/YYYY or DD/MM/YYYY format based on config.
func (f *DateTimeFixer) parseSlashMDY(dateStr string) (time.Time, bool) {
	matches := f.slashMDYRegex.FindStringSubmatch(dateStr)
	if matches == nil {
		return time.Time{}, false
	}

	first, err1 := strconv.Atoi(matches[1])
	second, err2 := strconv.Atoi(matches[2])
	year, err3 := strconv.Atoi(matches[3])
	if err1 != nil || err2 != nil || err3 != nil {
		return time.Time{}, false
	}

	var month, day int
	if f.config.AmbiguousFormat == AmbiguousFormatDMY {
		day = first
		month = second
	} else {
		// Default to MDY (US format)
		month = first
		day = second
	}

	if !isValidDate(year, month, day) {
		return time.Time{}, false
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), true
}

// parseWrittenMonth parses "January 15, 2024" or "Jan 15, 2024" format.
func (f *DateTimeFixer) parseWrittenMonth(dateStr string) (time.Time, bool) {
	matches := f.writtenMonthRegex.FindStringSubmatch(dateStr)
	if matches == nil {
		return time.Time{}, false
	}

	monthName := matches[1]
	day, err1 := strconv.Atoi(matches[2])
	year, err2 := strconv.Atoi(matches[3])
	if err1 != nil || err2 != nil {
		return time.Time{}, false
	}

	month := parseMonthName(monthName)
	if month == 0 {
		return time.Time{}, false
	}

	if !isValidDate(year, month, day) {
		return time.Time{}, false
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), true
}

// parseWrittenDay parses "15 January 2024" or "15 Jan 2024" format.
func (f *DateTimeFixer) parseWrittenDay(dateStr string) (time.Time, bool) {
	matches := f.writtenDayRegex.FindStringSubmatch(dateStr)
	if matches == nil {
		return time.Time{}, false
	}

	day, err1 := strconv.Atoi(matches[1])
	monthName := matches[2]
	year, err2 := strconv.Atoi(matches[3])
	if err1 != nil || err2 != nil {
		return time.Time{}, false
	}

	month := parseMonthName(monthName)
	if month == 0 {
		return time.Time{}, false
	}

	if !isValidDate(year, month, day) {
		return time.Time{}, false
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), true
}

// parseRFC2822 parses RFC 2822 format (Mon, 15 Jan 2024 or 15 Jan 2024).
func (f *DateTimeFixer) parseRFC2822(dateStr string) (time.Time, bool) {
	matches := f.rfc2822Regex.FindStringSubmatch(dateStr)
	if matches == nil {
		return time.Time{}, false
	}

	day, err1 := strconv.Atoi(matches[1])
	monthName := matches[2]
	year, err2 := strconv.Atoi(matches[3])
	if err1 != nil || err2 != nil {
		return time.Time{}, false
	}

	month := parseMonthName(monthName)
	if month == 0 {
		return time.Time{}, false
	}

	if !isValidDate(year, month, day) {
		return time.Time{}, false
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), true
}

// parseYearMonthDay parses year, month, and day strings into integers.
func parseYearMonthDay(yearStr, monthStr, dayStr string) (year, month, day int, ok bool) {
	var err error
	year, err = strconv.Atoi(yearStr)
	if err != nil {
		return 0, 0, 0, false
	}
	month, err = strconv.Atoi(monthStr)
	if err != nil {
		return 0, 0, 0, false
	}
	day, err = strconv.Atoi(dayStr)
	if err != nil {
		return 0, 0, 0, false
	}
	return year, month, day, true
}

// parseMonthName converts a month name to its numeric value.
func parseMonthName(name string) int {
	months := map[string]int{
		"january":   1,
		"jan":       1,
		"february":  2,
		"feb":       2,
		"march":     3,
		"mar":       3,
		"april":     4,
		"apr":       4,
		"may":       5,
		"june":      6,
		"jun":       6,
		"july":      7,
		"jul":       7,
		"august":    8,
		"aug":       8,
		"september": 9,
		"sep":       9,
		"sept":      9,
		"october":   10,
		"oct":       10,
		"november":  11,
		"nov":       11,
		"december":  12,
		"dec":       12,
	}

	return months[strings.ToLower(name)]
}

// isValidDate checks if the given year, month, day form a valid date.
func isValidDate(year, month, day int) bool {
	if year < 1 || year > 9999 {
		return false
	}
	if month < 1 || month > 12 {
		return false
	}
	if day < 1 || day > 31 {
		return false
	}

	// Check days in month
	daysInMonth := []int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

	// Handle leap year for February
	if month == 2 && isLeapYear(year) {
		daysInMonth[2] = 29
	}

	return day <= daysInMonth[month]
}

// isLeapYear checks if a year is a leap year.
func isLeapYear(year int) bool {
	return (year%4 == 0 && year%100 != 0) || year%400 == 0
}

// FixDateInContent finds and fixes date values in frontmatter content.
// It returns the fixed content and a list of changes made.
func (f *DateTimeFixer) FixDateInContent(content string) (string, []DateFixChange) {
	var changes []DateFixChange

	// Regex to find date fields in frontmatter
	dateKeyRegex := regexp.MustCompile(`(?m)^(date|published_date|created|modified|updated)\s*:\s*["']?([^"'\n]+)["']?\s*$`)

	fixed := dateKeyRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := dateKeyRegex.FindStringSubmatch(match)
		if parts == nil || len(parts) < 3 {
			return match
		}

		key := parts[1]
		value := strings.TrimSpace(parts[2])

		newValue, err := f.Fix(value)
		if err != nil || newValue == "" || newValue == value {
			return match
		}

		changes = append(changes, DateFixChange{
			Key:      key,
			OldValue: value,
			NewValue: newValue,
		})

		return fmt.Sprintf("%s: %s", key, newValue)
	})

	return fixed, changes
}

// DateFixChange records a date fix that was applied.
type DateFixChange struct {
	Key      string // The frontmatter key (e.g., "date", "published_date")
	OldValue string // The original date value
	NewValue string // The normalized date value
}
