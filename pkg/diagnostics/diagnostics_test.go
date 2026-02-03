package diagnostics

import (
	"testing"
)

// mockResolver implements Resolver for testing.
type mockResolver struct {
	slugs   map[string]bool
	handles map[string]bool
}

func (m *mockResolver) ResolveSlug(slug string) bool {
	if m.slugs == nil {
		return false
	}
	return m.slugs[slug]
}

func (m *mockResolver) ResolveHandle(handle string) bool {
	if m.handles == nil {
		return false
	}
	return m.handles[handle]
}

func TestCheck_DuplicateKeys(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantLen int
	}{
		{
			name: "no duplicates",
			content: `---
title: Test
date: 2024-01-01
---
Content`,
			wantLen: 0,
		},
		{
			name: "duplicate key",
			content: `---
title: First
date: 2024-01-01
title: Second
---
Content`,
			wantLen: 1,
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
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := Check("test.md", tt.content, nil)

			var dupIssues []Issue
			for _, issue := range issues {
				if issue.Code == "duplicate-key" {
					dupIssues = append(dupIssues, issue)
				}
			}

			if len(dupIssues) != tt.wantLen {
				t.Errorf("got %d duplicate-key issues, want %d", len(dupIssues), tt.wantLen)
			}
		})
	}
}

func TestCheck_InvalidDateFormats(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := Check("test.md", tt.content, nil)

			var dateIssues []Issue
			for _, issue := range issues {
				if issue.Code == "invalid-date" {
					dateIssues = append(dateIssues, issue)
				}
			}

			if len(dateIssues) != tt.wantLen {
				t.Errorf("got %d invalid-date issues, want %d", len(dateIssues), tt.wantLen)
			}
		})
	}
}

func TestCheck_MissingAltText(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := Check("test.md", tt.content, nil)

			var altIssues []Issue
			for _, issue := range issues {
				if issue.Code == "missing-alt-text" {
					altIssues = append(altIssues, issue)
				}
			}

			if len(altIssues) != tt.wantLen {
				t.Errorf("got %d missing-alt-text issues, want %d", len(altIssues), tt.wantLen)
			}
		})
	}
}

func TestCheck_ProtocollessURLs(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := Check("test.md", tt.content, nil)

			var urlIssues []Issue
			for _, issue := range issues {
				if issue.Code == "protocol-less-url" {
					urlIssues = append(urlIssues, issue)
				}
			}

			if len(urlIssues) != tt.wantLen {
				t.Errorf("got %d protocol-less-url issues, want %d", len(urlIssues), tt.wantLen)
			}
		})
	}
}

func TestCheck_H1Headings(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantLen int
	}{
		{
			name: "H2 heading OK",
			content: `---
title: Test
---
## This is fine`,
			wantLen: 0,
		},
		{
			name: "H1 heading warning",
			content: `---
title: Test
---
# This is bad`,
			wantLen: 1,
		},
		{
			name: "H1 in code block OK",
			content: `---
title: Test
---
` + "```markdown" + `
# This is in code block
` + "```",
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := Check("test.md", tt.content, nil)

			var h1Issues []Issue
			for _, issue := range issues {
				if issue.Code == "h1-in-content" {
					h1Issues = append(h1Issues, issue)
				}
			}

			if len(h1Issues) != tt.wantLen {
				t.Errorf("got %d h1-in-content issues, want %d", len(h1Issues), tt.wantLen)
			}
		})
	}
}

func TestCheck_BrokenWikilinks(t *testing.T) {
	resolver := &mockResolver{
		slugs: map[string]bool{
			"existing-post": true,
		},
	}

	tests := []struct {
		name     string
		content  string
		wantLen  int
		resolver Resolver
	}{
		{
			name: "valid wikilink",
			content: `---
title: Test
---
Check out [[existing-post]]`,
			wantLen:  0,
			resolver: resolver,
		},
		{
			name: "broken wikilink",
			content: `---
title: Test
---
Check out [[nonexistent-post]]`,
			wantLen:  1,
			resolver: resolver,
		},
		{
			name: "no resolver skips check",
			content: `---
title: Test
---
Check out [[nonexistent-post]]`,
			wantLen:  0,
			resolver: nil,
		},
		{
			name: "wikilink in code block OK",
			content: `---
title: Test
---
` + "```" + `
[[broken-in-code]]
` + "```",
			wantLen:  0,
			resolver: resolver,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := Check("test.md", tt.content, tt.resolver)

			var wikiIssues []Issue
			for _, issue := range issues {
				if issue.Code == "broken-wikilink" {
					wikiIssues = append(wikiIssues, issue)
				}
			}

			if len(wikiIssues) != tt.wantLen {
				t.Errorf("got %d broken-wikilink issues, want %d", len(wikiIssues), tt.wantLen)
			}
		})
	}
}

func TestCheck_UnknownMentions(t *testing.T) {
	resolver := &mockResolver{
		handles: map[string]bool{
			"knownuser": true,
		},
	}

	tests := []struct {
		name     string
		content  string
		wantLen  int
		resolver Resolver
	}{
		{
			name: "valid mention",
			content: `---
title: Test
---
Thanks @knownuser!`,
			wantLen:  0,
			resolver: resolver,
		},
		{
			name: "unknown mention",
			content: `---
title: Test
---
Thanks @unknownuser!`,
			wantLen:  1,
			resolver: resolver,
		},
		{
			name: "no resolver skips check",
			content: `---
title: Test
---
Thanks @unknownuser!`,
			wantLen:  0,
			resolver: nil,
		},
		{
			name: "email not a mention",
			content: `---
title: Test
---
Contact me at user@example.com`,
			wantLen:  0,
			resolver: resolver,
		},
		{
			name: "mention in code block OK",
			content: `---
title: Test
---
` + "```" + `
@unknownuser
` + "```",
			wantLen:  0,
			resolver: resolver,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := Check("test.md", tt.content, tt.resolver)

			var mentionIssues []Issue
			for _, issue := range issues {
				if issue.Code == "unknown-mention" {
					mentionIssues = append(mentionIssues, issue)
				}
			}

			if len(mentionIssues) != tt.wantLen {
				t.Errorf("got %d unknown-mention issues, want %d", len(mentionIssues), tt.wantLen)
			}
		})
	}
}

func TestCheck_AdmonitionFencedCode(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantLen int
	}{
		{
			name: "admonition with blank line before code OK",
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
			name: "admonition without blank line warning",
			content: `---
title: Test
---
!!! note
    ` + "```python" + `
    print("hello")
    ` + "```",
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := Check("test.md", tt.content, nil)

			var admonIssues []Issue
			for _, issue := range issues {
				if issue.Code == "admonition-fenced-code" {
					admonIssues = append(admonIssues, issue)
				}
			}

			if len(admonIssues) != tt.wantLen {
				t.Errorf("got %d admonition-fenced-code issues, want %d", len(admonIssues), tt.wantLen)
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
