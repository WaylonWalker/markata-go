package config

import (
	"strings"
	"testing"
)

func TestConfigError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ConfigError
		contains []string
	}{
		{
			name: "full error with all fields",
			err: &ConfigError{
				File:    "markata-go.toml",
				Line:    5,
				Column:  10,
				Field:   "url",
				Value:   "example.com",
				Message: "URL must include a scheme",
				Fix:     `url = "https://example.com"`,
				Context: []string{
					"     4 | title = \"My Site\"",
					">    5 | url = \"example.com\"",
					"     6 | output_dir = \"public\"",
				},
			},
			contains: []string{
				"markata-go.toml:5:10",
				"url",
				"URL must include a scheme",
				"Fix:",
				"https://example.com",
			},
		},
		{
			name: "error without column",
			err: &ConfigError{
				File:    "config.yaml",
				Line:    3,
				Field:   "concurrency",
				Message: "must be >= 0",
			},
			contains: []string{
				"config.yaml:3",
				"concurrency",
				"must be >= 0",
			},
		},
		{
			name: "error without file info",
			err: &ConfigError{
				Field:   "url",
				Message: "invalid format",
			},
			contains: []string{
				"configuration error",
				"url",
				"invalid format",
			},
		},
		{
			name: "warning",
			err: &ConfigError{
				File:    "markata-go.toml",
				Line:    10,
				Field:   "glob.patterns",
				Message: "empty patterns",
				IsWarn:  true,
			},
			contains: []string{
				"Warning:",
				"glob.patterns",
				"empty patterns",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("Error() missing %q\ngot:\n%s", want, got)
				}
			}
		})
	}
}

func TestConfigError_ShortError(t *testing.T) {
	tests := []struct {
		name string
		err  *ConfigError
		want string
	}{
		{
			name: "with file and line",
			err: &ConfigError{
				File:    "markata-go.toml",
				Line:    5,
				Field:   "url",
				Message: "must include scheme",
			},
			want: "markata-go.toml:5: config error: url: must include scheme",
		},
		{
			name: "without file",
			err: &ConfigError{
				Field:   "url",
				Message: "must include scheme",
			},
			want: "config error: url: must include scheme",
		},
		{
			name: "warning",
			err: &ConfigError{
				File:    "config.toml",
				Line:    10,
				Field:   "patterns",
				Message: "empty",
				IsWarn:  true,
			},
			want: "config.toml:10: config warning: patterns: empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.ShortError(); got != tt.want {
				t.Errorf("ShortError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfigErrors_Error(t *testing.T) {
	t.Run("single error", func(t *testing.T) {
		errs := &ConfigErrors{
			Errors: []*ConfigError{
				{Field: "url", Message: "invalid"},
			},
		}
		got := errs.Error()
		if !strings.Contains(got, "url") || !strings.Contains(got, "invalid") {
			t.Errorf("Error() = %q, should contain error details", got)
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		errs := &ConfigErrors{
			Errors: []*ConfigError{
				{Field: "url", Message: "invalid"},
				{Field: "concurrency", Message: "negative"},
			},
		}
		got := errs.Error()
		if !strings.Contains(got, "2 issues") {
			t.Errorf("Error() should mention issue count, got: %s", got)
		}
		if !strings.Contains(got, "url") || !strings.Contains(got, "concurrency") {
			t.Errorf("Error() should contain all errors, got: %s", got)
		}
	})

	t.Run("empty errors", func(t *testing.T) {
		errs := &ConfigErrors{}
		if got := errs.Error(); got != "" {
			t.Errorf("Error() = %q, want empty string", got)
		}
	})
}

func TestConfigErrors_HasErrors(t *testing.T) {
	tests := []struct {
		name   string
		errors []*ConfigError
		want   bool
	}{
		{
			name:   "no errors",
			errors: nil,
			want:   false,
		},
		{
			name: "only warnings",
			errors: []*ConfigError{
				{Field: "f", Message: "m", IsWarn: true},
			},
			want: false,
		},
		{
			name: "has errors",
			errors: []*ConfigError{
				{Field: "f", Message: "m", IsWarn: false},
			},
			want: true,
		},
		{
			name: "mixed",
			errors: []*ConfigError{
				{Field: "f1", Message: "m1", IsWarn: true},
				{Field: "f2", Message: "m2", IsWarn: false},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := &ConfigErrors{Errors: tt.errors}
			if got := errs.HasErrors(); got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigErrors_HasWarnings(t *testing.T) {
	tests := []struct {
		name   string
		errors []*ConfigError
		want   bool
	}{
		{
			name:   "no errors",
			errors: nil,
			want:   false,
		},
		{
			name: "only errors",
			errors: []*ConfigError{
				{Field: "f", Message: "m", IsWarn: false},
			},
			want: false,
		},
		{
			name: "has warnings",
			errors: []*ConfigError{
				{Field: "f", Message: "m", IsWarn: true},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := &ConfigErrors{Errors: tt.errors}
			if got := errs.HasWarnings(); got != tt.want {
				t.Errorf("HasWarnings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigErrors_SplitErrorsAndWarnings(t *testing.T) {
	errs := &ConfigErrors{
		Errors: []*ConfigError{
			{Field: "f1", Message: "error1", IsWarn: false},
			{Field: "f2", Message: "warning1", IsWarn: true},
			{Field: "f3", Message: "error2", IsWarn: false},
			{Field: "f4", Message: "warning2", IsWarn: true},
		},
	}

	errors, warnings := errs.SplitErrorsAndWarnings()

	if len(errors) != 2 {
		t.Errorf("len(errors) = %d, want 2", len(errors))
	}
	if len(warnings) != 2 {
		t.Errorf("len(warnings) = %d, want 2", len(warnings))
	}

	for _, err := range errors {
		if err.IsWarn {
			t.Errorf("found warning in errors: %v", err)
		}
	}
	for _, warn := range warnings {
		if !warn.IsWarn {
			t.Errorf("found error in warnings: %v", warn)
		}
	}
}

func TestPositionTracker_FindFieldPosition(t *testing.T) {
	tomlContent := `[markata-go]
title = "My Site"
url = "example.com"
concurrency = -1

[markata-go.glob]
patterns = []
`

	tracker := NewPositionTracker([]byte(tomlContent), "markata-go.toml")

	tests := []struct {
		field    string
		wantLine int
	}{
		{"title", 2},
		{"url", 3},
		{"concurrency", 4},
		{"patterns", 7},
		{"nonexistent", 0},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			pos := tracker.FindFieldPosition(tt.field)
			if pos.Line != tt.wantLine {
				t.Errorf("FindFieldPosition(%q).Line = %d, want %d", tt.field, pos.Line, tt.wantLine)
			}
		})
	}
}

func TestPositionTracker_ExtractContext(t *testing.T) {
	content := `line 1
line 2
line 3
line 4
line 5`

	tracker := NewPositionTracker([]byte(content), "test.txt")

	tests := []struct {
		name         string
		targetLine   int
		contextLines int
		wantLen      int
		wantMarked   string
	}{
		{
			name:         "middle line",
			targetLine:   3,
			contextLines: 1,
			wantLen:      3,
			wantMarked:   "> ",
		},
		{
			name:         "first line",
			targetLine:   1,
			contextLines: 2,
			wantLen:      3, // lines 1, 2, 3
			wantMarked:   "> ",
		},
		{
			name:         "last line",
			targetLine:   5,
			contextLines: 2,
			wantLen:      3, // lines 3, 4, 5
			wantMarked:   "> ",
		},
		{
			name:         "invalid line",
			targetLine:   0,
			contextLines: 2,
			wantLen:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := tracker.ExtractContext(tt.targetLine, tt.contextLines)
			if len(context) != tt.wantLen {
				t.Errorf("len(context) = %d, want %d", len(context), tt.wantLen)
			}

			if tt.wantLen > 0 && tt.wantMarked != "" {
				// Find the marked line
				found := false
				for _, line := range context {
					if strings.HasPrefix(line, tt.wantMarked) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("no line marked with %q in context: %v", tt.wantMarked, context)
				}
			}
		})
	}
}

func TestGetFixSuggestion(t *testing.T) {
	tests := []struct {
		errorType string
		field     string
		value     string
		contains  string
	}{
		{
			errorType: "url_no_scheme",
			value:     "example.com",
			contains:  "https://example.com",
		},
		{
			errorType: "url_invalid_scheme",
			value:     "ftp://example.com",
			contains:  "https://example.com",
		},
		{
			errorType: "negative_value",
			field:     "concurrency",
			contains:  "concurrency = 0",
		},
		{
			errorType: "empty_patterns",
			contains:  "**/*.md",
		},
		{
			errorType: "unknown_error",
			contains:  "", // Should return empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.errorType, func(t *testing.T) {
			got := GetFixSuggestion(tt.errorType, tt.field, tt.value)
			if tt.contains != "" && !strings.Contains(got, tt.contains) {
				t.Errorf("GetFixSuggestion(%q, %q, %q) = %q, should contain %q",
					tt.errorType, tt.field, tt.value, got, tt.contains)
			}
			if tt.contains == "" && got != "" {
				t.Errorf("GetFixSuggestion(%q, %q, %q) = %q, want empty",
					tt.errorType, tt.field, tt.value, got)
			}
		})
	}
}

func TestNewConfigError(t *testing.T) {
	content := `[markata-go]
url = "example.com"
concurrency = 4
`
	tracker := NewPositionTracker([]byte(content), "markata-go.toml")

	err := NewConfigError(tracker, "url", "example.com", "must include scheme", false)

	if err.File != "markata-go.toml" {
		t.Errorf("File = %q, want %q", err.File, "markata-go.toml")
	}
	if err.Line != 2 {
		t.Errorf("Line = %d, want 2", err.Line)
	}
	if err.Field != "url" {
		t.Errorf("Field = %q, want %q", err.Field, "url")
	}
	if err.Message != "must include scheme" {
		t.Errorf("Message = %q, want %q", err.Message, "must include scheme")
	}
	if len(err.Context) == 0 {
		t.Error("Context should not be empty")
	}
}

func TestNewConfigErrorWithFix(t *testing.T) {
	tracker := NewPositionTracker([]byte("url = bad"), "test.toml")

	err := NewConfigErrorWithFix(tracker, "url", "bad", "invalid", "url = good", false)

	if err.Fix != "url = good" {
		t.Errorf("Fix = %q, want %q", err.Fix, "url = good")
	}
}

func TestFormatConfigError(t *testing.T) {
	err := &ConfigError{
		File:    "markata-go.toml",
		Line:    3,
		Column:  7,
		Field:   "url",
		Value:   "example.com",
		Message: "URL must include a scheme (e.g., https://)",
		Fix:     `url = "https://example.com"`,
		Context: []string{
			"     2 | title = \"My Site\"",
			">    3 | url = \"example.com\"",
			"     4 | output_dir = \"public\"",
		},
	}

	formatted := FormatConfigError(err)

	expectedContains := []string{
		"markata-go.toml:3:7",
		"url = \"example.com\"",
		"URL must include a scheme",
		"Fix:",
		"https://example.com",
	}

	for _, want := range expectedContains {
		if !strings.Contains(formatted, want) {
			t.Errorf("FormatConfigError() missing %q\ngot:\n%s", want, formatted)
		}
	}
}

func TestFormatConfigErrors(t *testing.T) {
	t.Run("single error", func(t *testing.T) {
		errs := &ConfigErrors{
			Errors: []*ConfigError{
				{Field: "url", Message: "invalid"},
			},
		}
		formatted := FormatConfigErrors(errs)
		if !strings.Contains(formatted, "url") {
			t.Errorf("FormatConfigErrors() should contain error, got: %s", formatted)
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		errs := &ConfigErrors{
			Errors: []*ConfigError{
				{Field: "url", Message: "invalid"},
				{Field: "concurrency", Message: "negative"},
			},
		}
		formatted := FormatConfigErrors(errs)
		if !strings.Contains(formatted, "2 issues") {
			t.Errorf("FormatConfigErrors() should mention issue count, got: %s", formatted)
		}
		if !strings.Contains(formatted, "url") || !strings.Contains(formatted, "concurrency") {
			t.Errorf("FormatConfigErrors() should contain all errors, got: %s", formatted)
		}
	})

	t.Run("nil errors", func(t *testing.T) {
		formatted := FormatConfigErrors(nil)
		if formatted != "" {
			t.Errorf("FormatConfigErrors(nil) = %q, want empty", formatted)
		}
	})
}
