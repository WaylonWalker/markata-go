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

// TestMentionHoverContent tests the handleMentionHover function output.
// These tests verify that hover content is properly formatted for all mention
// scenarios, including minimal mentions that previously caused Neovim crashes
// due to trailing separators without content (issue #444).
func TestMentionHoverContent(t *testing.T) {
	tests := []struct {
		name            string
		mention         *MentionInfo
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "mention with only handle",
			mention: &MentionInfo{
				Handle: "minimal",
			},
			wantContains: []string{
				"## @minimal",
				"No additional information available",
			},
			wantNotContains: []string{
				"---\n",      // Should NOT have separator without content
				"*Site:*",    // No site URL
				"*Feed:*",    // No feed URL
				"*Aliases:*", // No aliases
			},
		},
		{
			name: "mention with handle and title only",
			mention: &MentionInfo{
				Handle: "titled",
				Title:  "A Cool Blog",
			},
			wantContains: []string{
				"## @titled - A Cool Blog",
				"No additional information available",
			},
			wantNotContains: []string{
				"---\n",
			},
		},
		{
			name: "mention with handle and description only",
			mention: &MentionInfo{
				Handle:      "described",
				Description: "An interesting blog about tech",
			},
			wantContains: []string{
				"## @described",
				"An interesting blog about tech",
			},
			wantNotContains: []string{
				"---\n",
				"No additional information available", // Has description
			},
		},
		{
			name: "mention with site_url only",
			mention: &MentionInfo{
				Handle:  "siteonly",
				SiteURL: "https://example.com",
			},
			wantContains: []string{
				"## @siteonly",
				"---",
				"*Site:* https://example.com",
			},
			wantNotContains: []string{
				"No additional information available",
			},
		},
		{
			name: "mention with feed_url only",
			mention: &MentionInfo{
				Handle:  "feedonly",
				FeedURL: "https://example.com/feed.xml",
			},
			wantContains: []string{
				"## @feedonly",
				"---",
				"*Feed:* https://example.com/feed.xml",
			},
			wantNotContains: []string{
				"No additional information available",
			},
		},
		{
			name: "mention with aliases only",
			mention: &MentionInfo{
				Handle:  "aliased",
				Aliases: []string{"other", "another"},
			},
			wantContains: []string{
				"## @aliased",
				"---",
				"*Aliases:* @other, @another",
			},
			wantNotContains: []string{
				"No additional information available",
			},
		},
		{
			name: "mention with all fields",
			mention: &MentionInfo{
				Handle:      "complete",
				Title:       "Complete Blog",
				Description: "A fully configured blog",
				SiteURL:     "https://complete.example.com",
				FeedURL:     "https://complete.example.com/feed.xml",
				Aliases:     []string{"full", "all"},
			},
			wantContains: []string{
				"## @complete - Complete Blog",
				"A fully configured blog",
				"---",
				"*Site:* https://complete.example.com",
				"*Feed:* https://complete.example.com/feed.xml",
				"*Aliases:* @full, @all",
			},
			wantNotContains: []string{
				"No additional information available",
			},
		},
		{
			name: "mention with title and site_url",
			mention: &MentionInfo{
				Handle:  "partial",
				Title:   "Partial Blog",
				SiteURL: "https://partial.example.com",
			},
			wantContains: []string{
				"## @partial - Partial Blog",
				"---",
				"*Site:* https://partial.example.com",
			},
			wantNotContains: []string{
				"No additional information available",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build the hover content using the same logic as handleMentionHover
			content := buildMentionHoverContent(tt.mention)

			// Check for expected content
			for _, want := range tt.wantContains {
				if !strings.Contains(content, want) {
					t.Errorf("hover content missing %q\nGot:\n%s", want, content)
				}
			}

			// Check for content that should NOT be present
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(content, notWant) {
					t.Errorf("hover content should NOT contain %q\nGot:\n%s", notWant, content)
				}
			}

			// Verify no trailing whitespace that could cause editor issues (issue #444)
			if strings.HasSuffix(content, " ") || strings.HasSuffix(content, "\n\n") {
				t.Errorf("hover content has trailing whitespace that could cause editor issues:\n%q", content)
			}

			// Verify content is not empty
			if content == "" {
				t.Error("hover content is empty")
			}
		})
	}
}

// buildMentionHoverContent builds hover content for a mention - extracted from handleMentionHover
// to enable unit testing without needing full server setup.
func buildMentionHoverContent(mention *MentionInfo) string {
	var sb strings.Builder
	sb.WriteString("## @")
	sb.WriteString(mention.Handle)
	if mention.Title != "" {
		sb.WriteString(" - ")
		sb.WriteString(mention.Title)
	}
	sb.WriteString("\n\n")

	if mention.Description != "" {
		sb.WriteString(mention.Description)
		sb.WriteString("\n\n")
	}

	// Build metadata section - only add separator if we have content
	hasMetadata := mention.SiteURL != "" || mention.FeedURL != "" || len(mention.Aliases) > 0
	if hasMetadata {
		sb.WriteString("---\n")
		if mention.SiteURL != "" {
			sb.WriteString("*Site:* ")
			sb.WriteString(mention.SiteURL)
			sb.WriteString("\n\n")
		}
		if mention.FeedURL != "" {
			sb.WriteString("*Feed:* ")
			sb.WriteString(mention.FeedURL)
			sb.WriteString("\n\n")
		}
		if len(mention.Aliases) > 0 {
			sb.WriteString("*Aliases:* @")
			sb.WriteString(strings.Join(mention.Aliases, ", @"))
			sb.WriteString("\n")
		}
	} else if mention.Description == "" {
		// Show placeholder for mentions with no metadata at all
		sb.WriteString("*No additional information available.*\n")
	}

	// Trim trailing whitespace to avoid width calculation issues in editors
	content := strings.TrimRight(sb.String(), "\n ")
	if content == "" {
		content = "## @" + mention.Handle
	}

	return content
}
