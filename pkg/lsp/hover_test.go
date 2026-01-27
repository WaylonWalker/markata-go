package lsp

import (
	"strings"
	"testing"
)

func TestGetFrontmatterHover_FieldName(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		lineNum      int
		col          int
		wantHover    bool
		wantContains []string
	}{
		{
			name:      "hover on title field name",
			content:   "---\ntitle: My Post\n---\n\nContent",
			lineNum:   1,
			col:       3, // On "title"
			wantHover: true,
			wantContains: []string{
				"**title**",
				"required",
				"Type: string",
			},
		},
		{
			name:      "hover on published field name",
			content:   "---\npublished: true\n---\n\nContent",
			lineNum:   1,
			col:       5, // On "published"
			wantHover: true,
			wantContains: []string{
				"**published**",
				"Type: boolean",
				"Allowed values: true, false",
			},
		},
		{
			name:      "hover on date field name",
			content:   "---\ndate: 2024-01-15\n---\n\nContent",
			lineNum:   1,
			col:       2, // On "date"
			wantHover: true,
			wantContains: []string{
				"**date**",
				"required",
				"Type: date",
			},
		},
		{
			name:      "hover on unknown custom field",
			content:   "---\ncustom_field: value\n---\n\nContent",
			lineNum:   1,
			col:       6, // On "custom_field"
			wantHover: true,
			wantContains: []string{
				"**custom_field**",
				"Custom field",
			},
		},
		{
			name:      "hover outside frontmatter",
			content:   "---\ntitle: Test\n---\n\nContent here",
			lineNum:   4,
			col:       3,
			wantHover: false,
		},
		{
			name:      "hover on closing delimiter",
			content:   "---\ntitle: Test\n---\n\nContent",
			lineNum:   2,
			col:       1,
			wantHover: false,
		},
		{
			name:      "hover on opening delimiter",
			content:   "---\ntitle: Test\n---\n\nContent",
			lineNum:   0,
			col:       1,
			wantHover: false,
		},
		{
			name:      "hover on list item (not a field)",
			content:   "---\ntags:\n  - tag1\n  - tag2\n---",
			lineNum:   2,
			col:       5,
			wantHover: false, // List items shouldn't show field hover
		},
		{
			name:      "hover on indented line (nested value)",
			content:   "---\ntags:\n  - nested\n---",
			lineNum:   2,
			col:       4,
			wantHover: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{}
			lines := strings.Split(tt.content, "\n")

			hover := s.getFrontmatterHover(tt.content, lines, tt.lineNum, tt.col)

			if tt.wantHover {
				if hover == nil {
					t.Fatal("expected hover, got nil")
				}
				for _, want := range tt.wantContains {
					if !strings.Contains(hover.Contents.Value, want) {
						t.Errorf("hover content missing %q\nGot: %s", want, hover.Contents.Value)
					}
				}
			} else if hover != nil {
				t.Errorf("expected no hover, got: %s", hover.Contents.Value)
			}
		})
	}
}

func TestGetFrontmatterHover_FieldValue(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		lineNum      int
		col          int
		wantHover    bool
		wantContains []string
	}{
		{
			name:      "hover on title field value",
			content:   "---\ntitle: My Post\n---\n\nContent",
			lineNum:   1,
			col:       10, // On "My Post"
			wantHover: true,
			wantContains: []string{
				"**title**",
				"Type: string",
			},
		},
		{
			name:      "hover on boolean field value",
			content:   "---\npublished: true\n---\n\nContent",
			lineNum:   1,
			col:       14, // On "true"
			wantHover: true,
			wantContains: []string{
				"**published**",
				"Allowed values: true, false",
			},
		},
		{
			name:      "hover on empty field value",
			content:   "---\ndescription: \n---\n\nContent",
			lineNum:   1,
			col:       14, // After colon with space
			wantHover: true,
			wantContains: []string{
				"**description**",
				"Type: string",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{}
			lines := strings.Split(tt.content, "\n")

			hover := s.getFrontmatterHover(tt.content, lines, tt.lineNum, tt.col)

			if tt.wantHover {
				if hover == nil {
					t.Fatal("expected hover, got nil")
				}
				for _, want := range tt.wantContains {
					if !strings.Contains(hover.Contents.Value, want) {
						t.Errorf("hover content missing %q\nGot: %s", want, hover.Contents.Value)
					}
				}
			} else if hover != nil {
				t.Errorf("expected no hover, got: %s", hover.Contents.Value)
			}
		})
	}
}

func TestGetFrontmatterHover_Range(t *testing.T) {
	s := &Server{}
	content := "---\ntitle: My Post\n---"
	lines := strings.Split(content, "\n")

	// Hover on field name
	hover := s.getFrontmatterHover(content, lines, 1, 3)
	if hover == nil {
		t.Fatal("expected hover")
	}
	if hover.Range == nil {
		t.Fatal("expected range")
	}
	if hover.Range.Start.Line != 1 || hover.Range.End.Line != 1 {
		t.Errorf("wrong line in range: start=%d, end=%d", hover.Range.Start.Line, hover.Range.End.Line)
	}
	// Range should cover "title"
	if hover.Range.Start.Character != 0 {
		t.Errorf("range start = %d, want 0", hover.Range.Start.Character)
	}
	if hover.Range.End.Character != 5 { // "title" ends at position 5
		t.Errorf("range end = %d, want 5", hover.Range.End.Character)
	}
}

func TestGetAdmonitionHover(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		lineNum      int
		col          int
		wantHover    bool
		wantContains []string
	}{
		{
			name:      "hover on note type",
			line:      "!!! note \"Title\"",
			lineNum:   5,
			col:       5, // On "note"
			wantHover: true,
			wantContains: []string{
				"**Note**",
				"Additional information",
				"Color:",
				"Usage:",
			},
		},
		{
			name:      "hover on warning type",
			line:      "!!! warning",
			lineNum:   0,
			col:       6, // On "warning"
			wantHover: true,
			wantContains: []string{
				"**Warning**",
				"Potential issues",
			},
		},
		{
			name:      "hover on tip type with ???",
			line:      "??? tip \"Helpful\"",
			lineNum:   2,
			col:       5, // On "tip"
			wantHover: true,
			wantContains: []string{
				"**Tip**",
				"Helpful suggestions",
			},
		},
		{
			name:      "hover on danger type with ???+",
			line:      "???+ danger",
			lineNum:   3,
			col:       7, // On "danger"
			wantHover: true,
			wantContains: []string{
				"**Danger**",
				"data loss",
			},
		},
		{
			name:      "hover on unknown admonition type",
			line:      "!!! unknowntype",
			lineNum:   0,
			col:       8, // On "unknowntype"
			wantHover: true,
			wantContains: []string{
				"**unknowntype**",
				"Unknown admonition type",
			},
		},
		{
			name:      "hover not on type (on marker)",
			line:      "!!! note",
			lineNum:   0,
			col:       1, // On "!!!"
			wantHover: false,
		},
		{
			name:      "hover not on admonition line",
			line:      "Regular text here",
			lineNum:   0,
			col:       5,
			wantHover: false,
		},
		{
			name:      "hover on indented admonition",
			line:      "    !!! info \"Nested\"",
			lineNum:   0,
			col:       9, // On "info"
			wantHover: true,
			wantContains: []string{
				"**Info**",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{}

			hover := s.getAdmonitionHover(tt.line, tt.lineNum, tt.col)

			if tt.wantHover {
				if hover == nil {
					t.Fatal("expected hover, got nil")
				}
				for _, want := range tt.wantContains {
					if !strings.Contains(hover.Contents.Value, want) {
						t.Errorf("hover content missing %q\nGot: %s", want, hover.Contents.Value)
					}
				}
			} else if hover != nil {
				t.Errorf("expected no hover, got: %s", hover.Contents.Value)
			}
		})
	}
}

func TestGetAdmonitionHover_Range(t *testing.T) {
	s := &Server{}

	hover := s.getAdmonitionHover("!!! warning \"Title\"", 5, 7)
	if hover == nil {
		t.Fatal("expected hover")
	}
	if hover.Range == nil {
		t.Fatal("expected range")
	}
	if hover.Range.Start.Line != 5 || hover.Range.End.Line != 5 {
		t.Errorf("wrong line: start=%d, end=%d", hover.Range.Start.Line, hover.Range.End.Line)
	}
	// Range should cover "warning" (positions 4-11)
	if hover.Range.Start.Character != 4 {
		t.Errorf("range start = %d, want 4", hover.Range.Start.Character)
	}
	if hover.Range.End.Character != 11 {
		t.Errorf("range end = %d, want 11", hover.Range.End.Character)
	}
}

func TestGetWikilinkAtPosition_Hover(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		col      int
		lineNum  int
		wantSlug string
		wantNil  bool
	}{
		{
			name:     "basic wikilink",
			line:     "See [[my-post]] for more",
			col:      8,
			lineNum:  0,
			wantSlug: "my-post",
		},
		{
			name:     "wikilink with display text",
			line:     "See [[my-post|My Post]] here",
			col:      8,
			lineNum:  0,
			wantSlug: "my-post",
		},
		{
			name:    "cursor outside wikilink",
			line:    "See [[my-post]] for more",
			col:     2, // Before [[
			lineNum: 0,
			wantNil: true,
		},
		{
			name:    "no wikilink on line",
			line:    "Regular text without links",
			col:     5,
			lineNum: 0,
			wantNil: true,
		},
		{
			name:     "multiple wikilinks - first one",
			line:     "[[first]] and [[second]]",
			col:      3,
			lineNum:  0,
			wantSlug: "first",
		},
		{
			name:     "multiple wikilinks - second one",
			line:     "[[first]] and [[second]]",
			col:      18,
			lineNum:  0,
			wantSlug: "second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug, rangeResult := getWikilinkAtPosition(tt.line, tt.col, tt.lineNum)

			if tt.wantNil {
				if slug != "" {
					t.Errorf("expected empty slug, got %q", slug)
				}
				if rangeResult != nil {
					t.Error("expected nil range")
				}
			} else {
				if slug != tt.wantSlug {
					t.Errorf("slug = %q, want %q", slug, tt.wantSlug)
				}
				if rangeResult == nil {
					t.Error("expected non-nil range")
				}
			}
		})
	}
}

func TestAdmonitionLineRegex(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantMatch  bool
		wantType   string
		wantMarker string
	}{
		{
			name:       "basic !!! note",
			line:       "!!! note",
			wantMatch:  true,
			wantType:   "note",
			wantMarker: "!!!",
		},
		{
			name:       "!!! with title",
			line:       "!!! warning \"Be careful\"",
			wantMatch:  true,
			wantType:   "warning",
			wantMarker: "!!!",
		},
		{
			name:       "??? marker",
			line:       "??? tip",
			wantMatch:  true,
			wantType:   "tip",
			wantMarker: "???",
		},
		{
			name:       "???+ marker",
			line:       "???+ info \"Open by default\"",
			wantMatch:  true,
			wantType:   "info",
			wantMarker: "???+",
		},
		{
			name:       "indented admonition",
			line:       "    !!! note \"Nested\"",
			wantMatch:  true,
			wantType:   "note",
			wantMarker: "!!!",
		},
		{
			name:      "not an admonition",
			line:      "Regular text",
			wantMatch: false,
		},
		{
			name:      "incomplete marker",
			line:      "!! note",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := admonitionLineRegex.FindStringSubmatch(tt.line)
			gotMatch := match != nil

			if gotMatch != tt.wantMatch {
				t.Errorf("match = %v, want %v", gotMatch, tt.wantMatch)
				return
			}

			if tt.wantMatch && len(match) >= 4 {
				if match[2] != tt.wantMarker {
					t.Errorf("marker = %q, want %q", match[2], tt.wantMarker)
				}
				if match[3] != tt.wantType {
					t.Errorf("type = %q, want %q", match[3], tt.wantType)
				}
			}
		})
	}
}

func TestHoverContentsFormat(t *testing.T) {
	s := &Server{}

	// Test frontmatter hover returns markdown
	content := "---\ntitle: Test\n---"
	lines := strings.Split(content, "\n")
	hover := s.getFrontmatterHover(content, lines, 1, 3)
	if hover == nil {
		t.Fatal("expected hover")
	}
	if hover.Contents.Kind != "markdown" {
		t.Errorf("contents kind = %q, want markdown", hover.Contents.Kind)
	}

	// Test admonition hover returns markdown
	hover = s.getAdmonitionHover("!!! note", 0, 5)
	if hover == nil {
		t.Fatal("expected hover")
	}
	if hover.Contents.Kind != "markdown" {
		t.Errorf("contents kind = %q, want markdown", hover.Contents.Kind)
	}
}
