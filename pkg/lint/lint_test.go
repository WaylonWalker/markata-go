package lint

import (
	"strings"
	"testing"
)

func TestLint_DuplicateKeys(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantType string
		wantLen  int
	}{
		{
			name: "no duplicates",
			content: `---
title: Test
date: 2024-01-01
---
Content`,
			wantType: "",
			wantLen:  0,
		},
		{
			name: "duplicate key",
			content: `---
title: First
date: 2024-01-01
title: Second
---
Content`,
			wantType: "duplicate-key",
			wantLen:  1,
		},
		{
			name: "multiple duplicate keys",
			content: `---
title: First
date: 2024-01-01
title: Second
tags: [a, b]
tags: [c, d]
---
Content`,
			wantType: "duplicate-key",
			wantLen:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Lint("test.md", tt.content)

			var dupIssues []Issue
			for _, issue := range result.Issues {
				if issue.Type == "duplicate-key" {
					dupIssues = append(dupIssues, issue)
				}
			}

			if len(dupIssues) != tt.wantLen {
				t.Errorf("got %d duplicate-key issues, want %d", len(dupIssues), tt.wantLen)
			}

			if tt.wantLen > 0 && dupIssues[0].Type != tt.wantType {
				t.Errorf("got issue type %q, want %q", dupIssues[0].Type, tt.wantType)
			}
		})
	}
}

func TestLint_InvalidDateFormats(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantLen int
	}{
		{
			name: "valid ISO date",
			content: `---
date: 2024-01-15
---`,
			wantLen: 0,
		},
		{
			name: "valid RFC3339 date",
			content: `---
date: 2024-01-15T10:30:00Z
---`,
			wantLen: 0,
		},
		{
			name: "single digit month",
			content: `---
date: 2020-1-15T00:00:00
---`,
			wantLen: 1,
		},
		{
			name: "single digit day",
			content: `---
date: 2020-01-1T00:00:00
---`,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Lint("test.md", tt.content)

			var dateIssues []Issue
			for _, issue := range result.Issues {
				if issue.Type == "invalid-date" {
					dateIssues = append(dateIssues, issue)
				}
			}

			if len(dateIssues) != tt.wantLen {
				t.Errorf("got %d invalid-date issues, want %d", len(dateIssues), tt.wantLen)
			}
		})
	}
}

func TestLint_MissingAltText(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantLen int
	}{
		{
			name: "image with alt text",
			content: `---
title: Test
---
![Alt text](image.png)`,
			wantLen: 0,
		},
		{
			name: "image without alt text",
			content: `---
title: Test
---
![](image.png)`,
			wantLen: 1,
		},
		{
			name: "multiple images without alt",
			content: `---
title: Test
---
![](first.png)
![](second.png)`,
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Lint("test.md", tt.content)

			var altIssues []Issue
			for _, issue := range result.Issues {
				if issue.Type == "missing-alt-text" {
					altIssues = append(altIssues, issue)
				}
			}

			if len(altIssues) != tt.wantLen {
				t.Errorf("got %d missing-alt-text issues, want %d", len(altIssues), tt.wantLen)
			}
		})
	}
}

func TestLint_ProtocollessURLs(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantLen int
	}{
		{
			name: "proper https URL",
			content: `---
title: Test
---
[Link](https://example.com)`,
			wantLen: 0,
		},
		{
			name: "protocol-less URL",
			content: `---
title: Test
---
[Link](//example.com/path)`,
			wantLen: 1,
		},
		{
			name: "protocol-less image URL",
			content: `---
title: Test
---
![img](//images.example.com/img.png)`,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Lint("test.md", tt.content)

			var urlIssues []Issue
			for _, issue := range result.Issues {
				if issue.Type == "protocol-less-url" {
					urlIssues = append(urlIssues, issue)
				}
			}

			if len(urlIssues) != tt.wantLen {
				t.Errorf("got %d protocol-less-url issues, want %d", len(urlIssues), tt.wantLen)
			}
		})
	}
}

func TestFix_DuplicateKeys(t *testing.T) {
	content := `---
title: First Title
date: 2024-01-01
title: Second Title
---
Content`

	result := Fix("test.md", content)

	// Should only have one title in fixed content
	if strings.Count(result.Fixed, "title:") != 1 {
		t.Errorf("expected 1 title key, got %d", strings.Count(result.Fixed, "title:"))
	}

	// Should keep the last occurrence (Second Title)
	if !strings.Contains(result.Fixed, "Second Title") {
		t.Error("expected to keep last occurrence of duplicate key")
	}
}

func TestFix_DateFormats(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "single digit month",
			content: "date: 2020-1-15",
			want:    "date: 2020-01-15",
		},
		{
			name:    "single digit day",
			content: "date: 2020-01-1",
			want:    "date: 2020-01-01",
		},
		{
			name:    "single digit month and day with time",
			content: "date: 2020-1-1T00:00:00",
			want:    "date: 2020-01-01", // DateTimeFixer normalizes to date-only format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Fix("test.md", tt.content)
			if result.Fixed != tt.want {
				t.Errorf("got %q, want %q", result.Fixed, tt.want)
			}
		})
	}
}

func TestFix_ImageLinks(t *testing.T) {
	content := `![](image.png)`
	result := Fix("test.md", content)

	want := `![image](image.png)`
	if result.Fixed != want {
		t.Errorf("got %q, want %q", result.Fixed, want)
	}
}

func TestFix_ProtocollessURLs(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "in parentheses",
			content: "(//example.com)",
			want:    "(https://example.com)",
		},
		{
			name:    "in quotes",
			content: `"//images.example.com/img.png"`,
			want:    `"https://images.example.com/img.png"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Fix("test.md", tt.content)
			if result.Fixed != tt.want {
				t.Errorf("got %q, want %q", result.Fixed, tt.want)
			}
		})
	}
}

func TestResult_HasErrors(t *testing.T) {
	tests := []struct {
		name   string
		issues []Issue
		want   bool
	}{
		{
			name:   "no issues",
			issues: nil,
			want:   false,
		},
		{
			name:   "only warnings",
			issues: []Issue{{Severity: SeverityWarning}},
			want:   false,
		},
		{
			name:   "has error",
			issues: []Issue{{Severity: SeverityError}},
			want:   true,
		},
		{
			name:   "mixed",
			issues: []Issue{{Severity: SeverityWarning}, {Severity: SeverityError}},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{Issues: tt.issues}
			if got := r.HasErrors(); got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		severity Severity
		want     string
	}{
		{SeverityError, "error"},
		{SeverityWarning, "warning"},
		{SeverityInfo, "info"},
		{Severity(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.severity.String(); got != tt.want {
				t.Errorf("Severity.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLint_AdmonitionFencedCode(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantLen int
	}{
		{
			name: "admonition with blank line before code - OK",
			content: `---
title: Test
---
!!! note

    ` + "```python" + `
    print("hello")
    ` + "```",
			wantLen: 0,
		},
		{
			name: "admonition without blank line before code - warning",
			content: `---
title: Test
---
!!! note
    ` + "```python" + `
    print("hello")
    ` + "```",
			wantLen: 1,
		},
		{
			name: "nested admonition without blank line - warning",
			content: `---
title: Test
---
!!! vsplit
    !!! vsplit
        ` + "```python" + `
        print("hello")
        ` + "```",
			wantLen: 1, // Only the inner admonition triggers warning
		},
		{
			name: "regular code block not in admonition - OK",
			content: `---
title: Test
---
Some text
` + "```python" + `
print("hello")
` + "```",
			wantLen: 0,
		},
		{
			name: "admonition with text content - OK",
			content: `---
title: Test
---
!!! note
    This is some text content.
    More text here.`,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Lint("test.md", tt.content)

			var admonIssues []Issue
			for _, issue := range result.Issues {
				if issue.Type == "admonition-fenced-code" {
					admonIssues = append(admonIssues, issue)
				}
			}

			if len(admonIssues) != tt.wantLen {
				t.Errorf("got %d admonition-fenced-code issues, want %d", len(admonIssues), tt.wantLen)
				for _, issue := range admonIssues {
					t.Logf("  issue: %s at line %d", issue.Message, issue.Line)
				}
			}
		})
	}
}

func TestFix_AdmonitionFencedCode(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "adds blank line after admonition",
			content: `!!! note
    ` + "```python" + `
    print("hello")
    ` + "```",
			want: `!!! note

    ` + "```python" + `
    print("hello")
    ` + "```",
		},
		{
			name: "does not modify with existing blank line",
			content: `!!! note

    ` + "```python" + `
    print("hello")
    ` + "```",
			want: `!!! note

    ` + "```python" + `
    print("hello")
    ` + "```",
		},
		{
			name: "does not modify non-admonition content",
			content: `Some text
` + "```python" + `
print("hello")
` + "```",
			want: `Some text
` + "```python" + `
print("hello")
` + "```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Fix("test.md", tt.content)
			if result.Fixed != tt.want {
				t.Errorf("got:\n%s\n\nwant:\n%s", result.Fixed, tt.want)
			}
		})
	}
}
