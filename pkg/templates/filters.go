package templates

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/flosch/pongo2/v6"
)

var registerOnce sync.Once

// registerFilters registers all custom template filters with pongo2.
// This is called once when the first Engine is created.
func registerFilters() {
	registerOnce.Do(func() {
		// Date formatting filters
		pongo2.RegisterFilter("rss_date", filterRSSDate)
		pongo2.RegisterFilter("atom_date", filterAtomDate)
		pongo2.RegisterFilter("date_format", filterDateFormat)
		// Override the built-in date filter to handle *time.Time and string parsing
		pongo2.ReplaceFilter("date", filterDate)

		// String manipulation filters
		pongo2.RegisterFilter("slugify", filterSlugify)
		pongo2.RegisterFilter("truncate", filterTruncate)
		pongo2.RegisterFilter("truncatewords", filterTruncateWords)

		// Default/fallback filter
		pongo2.RegisterFilter("default_if_none", filterDefaultIfNone)

		// Collection filters
		pongo2.RegisterFilter("length", filterLength)
		pongo2.RegisterFilter("first", filterFirst)
		pongo2.RegisterFilter("last", filterLast)
		pongo2.RegisterFilter("join", filterJoin)
		pongo2.RegisterFilter("reverse", filterReverse)
		pongo2.RegisterFilter("sort", filterSort)

		// HTML/text filters
		pongo2.RegisterFilter("striptags", filterStripTags)
		pongo2.RegisterFilter("linebreaks", filterLinebreaks)
		pongo2.RegisterFilter("linebreaksbr", filterLinebreaksBR)

		// URL filters
		pongo2.RegisterFilter("urlencode", filterURLEncode)
		pongo2.RegisterFilter("absolute_url", filterAbsoluteURL)
	})
}

// filterRSSDate formats a date for RSS feeds.
// Format: "Mon, 02 Jan 2006 15:04:05 -0700"
func filterRSSDate(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	t, err := toTime(in)
	if err != nil {
		return pongo2.AsValue(""), nil
	}
	return pongo2.AsValue(t.Format(time.RFC1123Z)), nil
}

// filterAtomDate formats a date for Atom feeds.
// Format: RFC3339 (e.g., "2006-01-02T15:04:05Z07:00")
func filterAtomDate(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	t, err := toTime(in)
	if err != nil {
		return pongo2.AsValue(""), nil
	}
	return pongo2.AsValue(t.Format(time.RFC3339)), nil
}

// filterDate is a replacement for pongo2's built-in date filter that handles
// *time.Time pointers and string parsing in addition to time.Time values.
// Uses Go's time formatting (e.g., "2006-01-02", "January 2, 2006").
func filterDate(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	t, err := toTime(in)
	if err != nil {
		return pongo2.AsValue(""), nil
	}

	format := param.String()
	if format == "" {
		format = "2006-01-02"
	}

	return pongo2.AsValue(t.Format(format)), nil
}

// filterDateFormat formats a date using a custom format string.
// Uses Go's time formatting (e.g., "2006-01-02", "January 2, 2006").
func filterDateFormat(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	t, err := toTime(in)
	if err != nil {
		return pongo2.AsValue(""), nil
	}

	format := param.String()
	if format == "" {
		format = "2006-01-02"
	}

	return pongo2.AsValue(t.Format(format)), nil
}

// filterSlugify converts a string to a URL-safe slug.
// Converts to lowercase, replaces spaces with hyphens, removes special characters.
func filterSlugify(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	s := in.String()

	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")

	// Remove non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			result.WriteRune(r)
		}
	}
	s = result.String()

	// Collapse multiple hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	// Trim leading/trailing hyphens
	s = strings.Trim(s, "-")

	return pongo2.AsValue(s), nil
}

// filterTruncate truncates a string to a specified length with an ellipsis.
// Usage: {{ text|truncate:100 }} or {{ text|truncate:"50" }}
func filterTruncate(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	s := in.String()
	length := param.Integer()

	if length <= 0 {
		length = 80 // default
	}

	if len(s) <= length {
		return in, nil
	}

	// Truncate and add ellipsis
	truncated := s[:length]
	// Try to break at a word boundary
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > length/2 {
		truncated = truncated[:lastSpace]
	}

	return pongo2.AsValue(truncated + "..."), nil
}

// filterTruncateWords truncates a string to a specified number of words.
// Usage: {{ text|truncatewords:20 }}
func filterTruncateWords(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	s := in.String()
	wordCount := param.Integer()

	if wordCount <= 0 {
		wordCount = 20 // default
	}

	words := strings.Fields(s)
	if len(words) <= wordCount {
		return in, nil
	}

	return pongo2.AsValue(strings.Join(words[:wordCount], " ") + "..."), nil
}

// filterDefaultIfNone returns a default value if the input is nil or empty.
// Usage: {{ value|default_if_none:"fallback" }}
func filterDefaultIfNone(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if in.IsNil() || (in.String() == "" && !in.IsBool()) {
		return param, nil
	}
	return in, nil
}

// filterLength returns the length of a string, slice, or map.
func filterLength(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return pongo2.AsValue(in.Len()), nil
}

// filterFirst returns the first element of a slice.
func filterFirst(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if in.Len() == 0 {
		return pongo2.AsValue(nil), nil
	}
	return in.Index(0), nil
}

// filterLast returns the last element of a slice.
func filterLast(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	length := in.Len()
	if length == 0 {
		return pongo2.AsValue(nil), nil
	}
	return in.Index(length - 1), nil
}

// filterJoin joins slice elements with a separator.
// Usage: {{ list|join:", " }}
func filterJoin(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if !in.CanSlice() {
		return pongo2.AsValue(in.String()), nil
	}

	separator := param.String()
	if separator == "" {
		separator = ", "
	}

	var items []string
	for i := 0; i < in.Len(); i++ {
		items = append(items, in.Index(i).String())
	}

	return pongo2.AsValue(strings.Join(items, separator)), nil
}

// filterReverse reverses a slice or string.
func filterReverse(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	// For strings, reverse the characters
	if !in.CanSlice() {
		s := in.String()
		runes := []rune(s)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return pongo2.AsValue(string(runes)), nil
	}

	// For slices, create a reversed copy
	length := in.Len()
	reversed := make([]interface{}, length)
	for i := 0; i < length; i++ {
		reversed[length-1-i] = in.Index(i).Interface()
	}
	return pongo2.AsValue(reversed), nil
}

// filterSort sorts a slice of comparable values.
func filterSort(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if !in.CanSlice() || in.Len() == 0 {
		return in, nil
	}

	// Convert to string slice and sort
	var items []string
	for i := 0; i < in.Len(); i++ {
		items = append(items, in.Index(i).String())
	}

	// Simple string sort
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i] > items[j] {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	// Convert back to interface slice
	result := make([]interface{}, len(items))
	for i, s := range items {
		result[i] = s
	}

	return pongo2.AsValue(result), nil
}

// filterStripTags removes HTML tags from a string.
func filterStripTags(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	s := in.String()
	// Simple regex to remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	return pongo2.AsValue(re.ReplaceAllString(s, "")), nil
}

// filterLinebreaks converts newlines to <p> and <br> tags.
func filterLinebreaks(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	s := in.String()
	// Split by double newlines for paragraphs
	paragraphs := regexp.MustCompile(`\n\n+`).Split(s, -1)
	var result []string
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p != "" {
			// Convert single newlines to <br>
			p = strings.ReplaceAll(p, "\n", "<br>")
			result = append(result, "<p>"+p+"</p>")
		}
	}
	return pongo2.AsValue(strings.Join(result, "\n")), nil
}

// filterLinebreaksBR converts newlines to <br> tags.
func filterLinebreaksBR(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	s := in.String()
	return pongo2.AsValue(strings.ReplaceAll(s, "\n", "<br>")), nil
}

// filterURLEncode URL-encodes a string.
func filterURLEncode(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	s := in.String()
	// Simple URL encoding for common characters
	s = strings.ReplaceAll(s, " ", "%20")
	s = strings.ReplaceAll(s, "&", "%26")
	s = strings.ReplaceAll(s, "=", "%3D")
	s = strings.ReplaceAll(s, "?", "%3F")
	s = strings.ReplaceAll(s, "#", "%23")
	return pongo2.AsValue(s), nil
}

// filterAbsoluteURL converts a relative URL to an absolute URL.
// Requires the site URL to be passed as the parameter.
// Usage: {{ post.href|absolute_url:config.url }}
func filterAbsoluteURL(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	path := in.String()
	baseURL := param.String()

	if baseURL == "" {
		return in, nil
	}

	// Remove trailing slash from base URL
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return pongo2.AsValue(baseURL + path), nil
}

// toTime attempts to convert a pongo2 value to a time.Time.
func toTime(in *pongo2.Value) (time.Time, error) {
	if in.IsNil() {
		return time.Time{}, fmt.Errorf("nil value")
	}

	// Check if it's already a time.Time
	if t, ok := in.Interface().(time.Time); ok {
		return t, nil
	}

	// Check if it's a *time.Time
	if t, ok := in.Interface().(*time.Time); ok {
		if t != nil {
			return *t, nil
		}
		return time.Time{}, fmt.Errorf("nil time pointer")
	}

	// Try to parse as string
	s := in.String()
	if s == "" {
		return time.Time{}, fmt.Errorf("empty string")
	}

	// Try common formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"January 2, 2006",
		"Jan 2, 2006",
		"02 Jan 2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}
