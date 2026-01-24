package lint

import (
	"testing"
	"time"
)

func TestNewDateTimeFixer(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		fixer := NewDateTimeFixer(DateTimeFixerConfig{})

		// Should apply defaults
		if fixer.config.Format != "2006-01-02" {
			t.Errorf("expected default format, got %q", fixer.config.Format)
		}
		if fixer.config.AmbiguousFormat != "mdy" {
			t.Errorf("expected default ambiguous format, got %q", fixer.config.AmbiguousFormat)
		}
		if fixer.config.MissingDate != "skip" {
			t.Errorf("expected default missing date handling, got %q", fixer.config.MissingDate)
		}
	})

	t.Run("custom config", func(t *testing.T) {
		fixer := NewDateTimeFixer(DateTimeFixerConfig{
			Format:          "01/02/2006",
			AmbiguousFormat: "dmy",
			MissingDate:     "today",
		})

		if fixer.config.Format != "01/02/2006" {
			t.Errorf("expected custom format, got %q", fixer.config.Format)
		}
		if fixer.config.AmbiguousFormat != "dmy" {
			t.Errorf("expected custom ambiguous format, got %q", fixer.config.AmbiguousFormat)
		}
	})
}

func TestDateTimeFixer_Fix_ISO8601(t *testing.T) {
	fixer := NewDateTimeFixer(DefaultDateTimeFixerConfig())

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "ISO date",
			input:   "2024-01-15",
			want:    "2024-01-15",
			wantErr: false,
		},
		{
			name:    "ISO date single digit month",
			input:   "2024-1-15",
			want:    "2024-01-15",
			wantErr: false,
		},
		{
			name:    "ISO date single digit day",
			input:   "2024-01-5",
			want:    "2024-01-05",
			wantErr: false,
		},
		{
			name:    "ISO date single digit month and day",
			input:   "2024-1-5",
			want:    "2024-01-05",
			wantErr: false,
		},
		{
			name:    "ISO datetime with Z preserves time",
			input:   "2024-01-15T10:30:00Z",
			want:    "2024-01-15T10:30:00Z",
			wantErr: false,
		},
		{
			name:    "ISO datetime with timezone offset preserves time",
			input:   "2024-01-15T10:30:00+05:00",
			want:    "2024-01-15T05:30:00Z", // Converted to UTC
			wantErr: false,
		},
		{
			name:    "ISO datetime without timezone preserves time",
			input:   "2024-01-15T10:30:00",
			want:    "2024-01-15T10:30:00Z",
			wantErr: false,
		},
		{
			name:    "ISO datetime midnight becomes date only",
			input:   "2024-01-15T00:00:00Z",
			want:    "2024-01-15",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fixer.Fix(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Fix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Fix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDateTimeFixer_Fix_SlashFormats(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		ambiguousFormat string
		want            string
		wantErr         bool
	}{
		{
			name:            "YYYY/MM/DD format",
			input:           "2024/01/15",
			ambiguousFormat: "mdy",
			want:            "2024-01-15",
			wantErr:         false,
		},
		{
			name:            "MM/DD/YYYY US format",
			input:           "01/15/2024",
			ambiguousFormat: "mdy",
			want:            "2024-01-15",
			wantErr:         false,
		},
		{
			name:            "DD/MM/YYYY European format",
			input:           "15/01/2024",
			ambiguousFormat: "dmy",
			want:            "2024-01-15",
			wantErr:         false,
		},
		{
			name:            "ambiguous date MDY interpretation",
			input:           "03/04/2024",
			ambiguousFormat: "mdy",
			want:            "2024-03-04", // March 4
			wantErr:         false,
		},
		{
			name:            "ambiguous date DMY interpretation",
			input:           "03/04/2024",
			ambiguousFormat: "dmy",
			want:            "2024-04-03", // April 3
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixer := NewDateTimeFixer(DateTimeFixerConfig{
				AmbiguousFormat: tt.ambiguousFormat,
			})
			got, err := fixer.Fix(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Fix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Fix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDateTimeFixer_Fix_WrittenFormats(t *testing.T) {
	fixer := NewDateTimeFixer(DefaultDateTimeFixerConfig())

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "full month name, month first",
			input:   "January 15, 2024",
			want:    "2024-01-15",
			wantErr: false,
		},
		{
			name:    "abbreviated month name, month first",
			input:   "Jan 15, 2024",
			want:    "2024-01-15",
			wantErr: false,
		},
		{
			name:    "full month name without comma",
			input:   "January 15 2024",
			want:    "2024-01-15",
			wantErr: false,
		},
		{
			name:    "day first, full month",
			input:   "15 January 2024",
			want:    "2024-01-15",
			wantErr: false,
		},
		{
			name:    "day first, abbreviated month",
			input:   "15 Jan 2024",
			want:    "2024-01-15",
			wantErr: false,
		},
		{
			name:    "December date",
			input:   "December 25, 2024",
			want:    "2024-12-25",
			wantErr: false,
		},
		{
			name:    "Sept abbreviation",
			input:   "Sept 15, 2024",
			want:    "2024-09-15",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fixer.Fix(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Fix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Fix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDateTimeFixer_Fix_RFC2822(t *testing.T) {
	fixer := NewDateTimeFixer(DefaultDateTimeFixerConfig())

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "RFC 2822 with day name",
			input:   "Mon, 15 Jan 2024",
			want:    "2024-01-15",
			wantErr: false,
		},
		{
			name:    "RFC 2822 without day name",
			input:   "15 Jan 2024",
			want:    "2024-01-15",
			wantErr: false,
		},
		{
			name:    "RFC 2822 with time",
			input:   "Mon, 15 Jan 2024 10:30:00",
			want:    "2024-01-15",
			wantErr: false,
		},
		{
			name:    "RFC 2822 with timezone",
			input:   "Mon, 15 Jan 2024 10:30:00 +0000",
			want:    "2024-01-15",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fixer.Fix(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Fix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Fix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDateTimeFixer_Fix_NaturalLanguage(t *testing.T) {
	// Use a fixed reference time for reproducible tests
	refTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	fixer := NewDateTimeFixer(DateTimeFixerConfig{
		ReferenceTime: &refTime,
	})

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "today",
			input:   "today",
			want:    "2024-06-15",
			wantErr: false,
		},
		{
			name:    "Today (capitalized)",
			input:   "Today",
			want:    "2024-06-15",
			wantErr: false,
		},
		{
			name:    "now",
			input:   "now",
			want:    "2024-06-15",
			wantErr: false,
		},
		{
			name:    "yesterday",
			input:   "yesterday",
			want:    "2024-06-14",
			wantErr: false,
		},
		{
			name:    "tomorrow",
			input:   "tomorrow",
			want:    "2024-06-16",
			wantErr: false,
		},
		{
			name:    "last week",
			input:   "last week",
			want:    "2024-06-08",
			wantErr: false,
		},
		{
			name:    "last month",
			input:   "last month",
			want:    "2024-05-15",
			wantErr: false,
		},
		{
			name:    "last year",
			input:   "last year",
			want:    "2023-06-15",
			wantErr: false,
		},
		{
			name:    "next week",
			input:   "next week",
			want:    "2024-06-22",
			wantErr: false,
		},
		{
			name:    "next month",
			input:   "next month",
			want:    "2024-07-15",
			wantErr: false,
		},
		{
			name:    "next year",
			input:   "next year",
			want:    "2025-06-15",
			wantErr: false,
		},
		{
			name:    "3 days ago",
			input:   "3 days ago",
			want:    "2024-06-12",
			wantErr: false,
		},
		{
			name:    "2 weeks ago",
			input:   "2 weeks ago",
			want:    "2024-06-01",
			wantErr: false,
		},
		{
			name:    "1 month ago",
			input:   "1 month ago",
			want:    "2024-05-15",
			wantErr: false,
		},
		{
			name:    "5 years ago",
			input:   "5 years ago",
			want:    "2019-06-15",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fixer.Fix(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Fix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Fix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDateTimeFixer_Fix_MissingDate(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		missingDate string
		want        string
		wantErr     bool
	}{
		{
			name:        "skip empty date",
			missingDate: "skip",
			want:        "",
			wantErr:     false,
		},
		{
			name:        "use today for empty date",
			missingDate: "today",
			want:        "2024-06-15",
			wantErr:     false,
		},
		{
			name:        "error on empty date",
			missingDate: "error",
			want:        "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixer := NewDateTimeFixer(DateTimeFixerConfig{
				MissingDate:   tt.missingDate,
				ReferenceTime: &refTime,
			})
			got, err := fixer.Fix("")
			if (err != nil) != tt.wantErr {
				t.Errorf("Fix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Fix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDateTimeFixer_Fix_InvalidDates(t *testing.T) {
	fixer := NewDateTimeFixer(DefaultDateTimeFixerConfig())

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "invalid month",
			input:   "2024-13-15",
			wantErr: true,
		},
		{
			name:    "invalid day",
			input:   "2024-01-32",
			wantErr: true,
		},
		{
			name:    "february 30",
			input:   "2024-02-30",
			wantErr: true,
		},
		{
			name:    "february 29 non-leap year",
			input:   "2023-02-29",
			wantErr: true,
		},
		{
			name:    "random text",
			input:   "not a date",
			wantErr: true,
		},
		{
			name:    "partial date",
			input:   "2024-01",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fixer.Fix(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Fix() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDateTimeFixer_Fix_LeapYears(t *testing.T) {
	fixer := NewDateTimeFixer(DefaultDateTimeFixerConfig())

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "leap year february 29",
			input:   "2024-02-29",
			want:    "2024-02-29",
			wantErr: false,
		},
		{
			name:    "century non-leap year",
			input:   "1900-02-29",
			wantErr: true,
		},
		{
			name:    "400-year leap year",
			input:   "2000-02-29",
			want:    "2000-02-29",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fixer.Fix(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Fix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Fix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDateTimeFixer_Fix_CustomFormat(t *testing.T) {
	fixer := NewDateTimeFixer(DateTimeFixerConfig{
		Format: "01/02/2006",
	})

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "output in custom format",
			input: "2024-01-15",
			want:  "01/15/2024",
		},
		{
			name:  "natural language with custom format",
			input: "January 15, 2024",
			want:  "01/15/2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fixer.Fix(tt.input)
			if err != nil {
				t.Errorf("Fix() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Fix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDateTimeFixer_Fix_WhitespaceHandling(t *testing.T) {
	fixer := NewDateTimeFixer(DefaultDateTimeFixerConfig())

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "leading whitespace",
			input: "  2024-01-15",
			want:  "2024-01-15",
		},
		{
			name:  "trailing whitespace",
			input: "2024-01-15  ",
			want:  "2024-01-15",
		},
		{
			name:  "both whitespace",
			input: "  2024-01-15  ",
			want:  "2024-01-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fixer.Fix(tt.input)
			if err != nil {
				t.Errorf("Fix() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Fix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDateTimeFixer_FixDateInContent(t *testing.T) {
	fixer := NewDateTimeFixer(DefaultDateTimeFixerConfig())

	tests := []struct {
		name        string
		content     string
		want        string
		wantChanges int
	}{
		{
			name: "fix slash date",
			content: `---
title: Test Post
date: 2024/01/15
---
Content here`,
			want: `---
title: Test Post
date: 2024-01-15
---
Content here`,
			wantChanges: 1,
		},
		{
			name: "fix written date",
			content: `---
title: Test Post
date: January 15, 2024
---
Content here`,
			want: `---
title: Test Post
date: 2024-01-15
---
Content here`,
			wantChanges: 1,
		},
		{
			name: "fix multiple date fields",
			content: `---
title: Test Post
date: 2024/01/15
modified: Jan 20, 2024
---
Content here`,
			want: `---
title: Test Post
date: 2024-01-15
modified: 2024-01-20
---
Content here`,
			wantChanges: 2,
		},
		{
			name: "no changes needed",
			content: `---
title: Test Post
date: 2024-01-15
---
Content here`,
			want: `---
title: Test Post
date: 2024-01-15
---
Content here`,
			wantChanges: 0,
		},
		{
			name: "quoted date",
			content: `---
title: Test Post
date: "January 15, 2024"
---
Content here`,
			want: `---
title: Test Post
date: 2024-01-15
---
Content here`,
			wantChanges: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changes := fixer.FixDateInContent(tt.content)
			if got != tt.want {
				t.Errorf("FixDateInContent() content =\n%q\nwant\n%q", got, tt.want)
			}
			if len(changes) != tt.wantChanges {
				t.Errorf("FixDateInContent() changes = %d, want %d", len(changes), tt.wantChanges)
			}
		})
	}
}

func TestIsValidDate(t *testing.T) {
	tests := []struct {
		name  string
		year  int
		month int
		day   int
		want  bool
	}{
		{"valid date", 2024, 1, 15, true},
		{"valid leap day", 2024, 2, 29, true},
		{"invalid leap day", 2023, 2, 29, false},
		{"invalid month 0", 2024, 0, 15, false},
		{"invalid month 13", 2024, 13, 15, false},
		{"invalid day 0", 2024, 1, 0, false},
		{"invalid day 32", 2024, 1, 32, false},
		{"april 31", 2024, 4, 31, false},
		{"june 30", 2024, 6, 30, true},
		{"june 31", 2024, 6, 31, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidDate(tt.year, tt.month, tt.day); got != tt.want {
				t.Errorf("isValidDate(%d, %d, %d) = %v, want %v", tt.year, tt.month, tt.day, got, tt.want)
			}
		})
	}
}

func TestIsLeapYear(t *testing.T) {
	tests := []struct {
		year int
		want bool
	}{
		{2024, true},  // Divisible by 4
		{2023, false}, // Not divisible by 4
		{2000, true},  // Divisible by 400
		{1900, false}, // Divisible by 100 but not 400
		{2100, false}, // Divisible by 100 but not 400
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.year)), func(t *testing.T) {
			if got := isLeapYear(tt.year); got != tt.want {
				t.Errorf("isLeapYear(%d) = %v, want %v", tt.year, got, tt.want)
			}
		})
	}
}

func TestParseMonthName(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{"january", 1},
		{"January", 1},
		{"JANUARY", 1},
		{"jan", 1},
		{"Jan", 1},
		{"february", 2},
		{"feb", 2},
		{"march", 3},
		{"mar", 3},
		{"april", 4},
		{"apr", 4},
		{"may", 5},
		{"june", 6},
		{"jun", 6},
		{"july", 7},
		{"jul", 7},
		{"august", 8},
		{"aug", 8},
		{"september", 9},
		{"sep", 9},
		{"sept", 9},
		{"october", 10},
		{"oct", 10},
		{"november", 11},
		{"nov", 11},
		{"december", 12},
		{"dec", 12},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseMonthName(tt.name); got != tt.want {
				t.Errorf("parseMonthName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestDefaultDateTimeFixerConfig(t *testing.T) {
	config := DefaultDateTimeFixerConfig()

	if config.Format != "2006-01-02" {
		t.Errorf("Format = %q, want %q", config.Format, "2006-01-02")
	}
	if config.DateTimeFormat != "2006-01-02T15:04:05Z" {
		t.Errorf("DateTimeFormat = %q, want %q", config.DateTimeFormat, "2006-01-02T15:04:05Z")
	}
	if config.PreserveTime != true {
		t.Errorf("PreserveTime = %v, want %v", config.PreserveTime, true)
	}
	if config.AmbiguousFormat != "mdy" {
		t.Errorf("AmbiguousFormat = %q, want %q", config.AmbiguousFormat, "mdy")
	}
	if config.MissingDate != "skip" {
		t.Errorf("MissingDate = %q, want %q", config.MissingDate, "skip")
	}
	if config.WarnFuture != false {
		t.Errorf("WarnFuture = %v, want %v", config.WarnFuture, false)
	}
	if config.WarnOld != false {
		t.Errorf("WarnOld = %v, want %v", config.WarnOld, false)
	}
}

func TestDateTimeFixer_Fix_PreserveTime(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		preserveTime bool
		want         string
	}{
		{
			name:         "preserve time enabled - keeps time",
			input:        "2024-01-15T14:30:00Z",
			preserveTime: true,
			want:         "2024-01-15T14:30:00Z",
		},
		{
			name:         "preserve time disabled - strips time",
			input:        "2024-01-15T14:30:00Z",
			preserveTime: false,
			want:         "2024-01-15",
		},
		{
			name:         "preserve time enabled - date only stays date only",
			input:        "2024-01-15",
			preserveTime: true,
			want:         "2024-01-15",
		},
		{
			name:         "preserve time enabled - midnight becomes date only",
			input:        "2024-01-15T00:00:00Z",
			preserveTime: true,
			want:         "2024-01-15",
		},
		{
			name:         "preserve time enabled - RFC3339 with offset",
			input:        "2024-01-15T14:30:00+05:00",
			preserveTime: true,
			want:         "2024-01-15T09:30:00Z", // Converted to UTC
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixer := NewDateTimeFixer(DateTimeFixerConfig{
				PreserveTime: tt.preserveTime,
			})
			got, err := fixer.Fix(tt.input)
			if err != nil {
				t.Errorf("Fix() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Fix() = %q, want %q", got, tt.want)
			}
		})
	}
}
