package lint

import (
	"testing"
)

func TestLint_H1Headings(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantLen int
	}{
		{
			name: "no H1 heading",
			content: `---
title: Test Post
---
## This is H2

Some content here.

### This is H3`,
			wantLen: 0,
		},
		{
			name: "single H1 heading",
			content: `---
title: Test Post
---
# This is H1

Some content here.`,
			wantLen: 1,
		},
		{
			name: "multiple H1 headings",
			content: `---
title: Test Post
---
# First H1

Some content.

# Second H1

More content.

# Third H1`,
			wantLen: 3,
		},
		{
			name: "H1 in code block should not trigger",
			content: `---
title: Test Post
---
## Introduction

Here's some markdown syntax:

` + "```markdown" + `
# This is a heading example
` + "```" + `

## Conclusion`,
			wantLen: 0,
		},
		{
			name: "H1 in triple tilde code block should not trigger",
			content: `---
title: Test Post
---
## Introduction

~~~markdown
# This is a heading example
~~~

## Conclusion`,
			wantLen: 0,
		},
		{
			name: "H1 outside code block triggers but inside does not",
			content: `---
title: Test Post
---
# Real H1 before code

` + "```markdown" + `
# This is in code block
` + "```" + `

# Real H1 after code`,
			wantLen: 2,
		},
		{
			name: "H2 and H3 do not trigger",
			content: `---
title: Test Post
---
## H2 heading

### H3 heading

#### H4 heading

##### H5 heading

###### H6 heading`,
			wantLen: 0,
		},
		{
			name: "H1 without frontmatter",
			content: `# This is H1

Some content without frontmatter.`,
			wantLen: 1,
		},
		{
			name: "bare hash without space is not H1",
			content: `---
title: Test
---
#notaheading
#also-not-a-heading`,
			wantLen: 0,
		},
		{
			name: "lone hash is H1",
			content: `---
title: Test
---
#

Some content.`,
			wantLen: 1,
		},
		{
			name: "H1 with special characters in title",
			content: `---
title: Test
---
# Hello World! How's it going?

Content here.`,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Lint("test.md", tt.content)

			var h1Issues []Issue
			for _, issue := range result.Issues {
				if issue.Type == "h1-in-content" {
					h1Issues = append(h1Issues, issue)
				}
			}

			if len(h1Issues) != tt.wantLen {
				t.Errorf("got %d h1-in-content issues, want %d", len(h1Issues), tt.wantLen)
				for _, issue := range h1Issues {
					t.Logf("  - Line %d: %s", issue.Line, issue.Message)
				}
			}

			// Verify severity is warning
			for _, issue := range h1Issues {
				if issue.Severity != SeverityWarning {
					t.Errorf("expected severity %v, got %v", SeverityWarning, issue.Severity)
				}
				if issue.Fixable {
					t.Error("H1 issues should not be auto-fixable")
				}
			}
		})
	}
}

func TestLint_H1Headings_LineNumbers(t *testing.T) {
	content := `---
title: Test
date: 2024-01-01
---
## Introduction

# First H1

Some content.

# Second H1`

	result := Lint("test.md", content)

	var h1Issues []Issue
	for _, issue := range result.Issues {
		if issue.Type == "h1-in-content" {
			h1Issues = append(h1Issues, issue)
		}
	}

	if len(h1Issues) != 2 {
		t.Fatalf("expected 2 H1 issues, got %d", len(h1Issues))
	}

	// First H1 is on line 7 (4 frontmatter lines + 2 body lines + 1)
	// Frontmatter: line 1 (---), 2 (title), 3 (date), 4 (---)
	// Body: line 5 (## Introduction), 6 (empty), 7 (# First H1)
	expectedLine1 := 7
	if h1Issues[0].Line != expectedLine1 {
		t.Errorf("first H1 line: got %d, want %d", h1Issues[0].Line, expectedLine1)
	}

	// Second H1 is on line 11
	expectedLine2 := 11
	if h1Issues[1].Line != expectedLine2 {
		t.Errorf("second H1 line: got %d, want %d", h1Issues[1].Line, expectedLine2)
	}
}
