package lsp

import (
	"testing"
)

func TestGetAdmonitionContext(t *testing.T) {
	tests := []struct {
		name            string
		line            string
		col             int
		wantMarker      string
		wantTypePrefix  string
		wantMarkerStart int
		wantTypeStart   int
		wantInContext   bool
	}{
		{
			name:            "after !!! marker only",
			line:            "!!! ",
			col:             4,
			wantMarker:      "!!!",
			wantTypePrefix:  "",
			wantMarkerStart: 0,
			wantTypeStart:   4,
			wantInContext:   true,
		},
		{
			name:            "partial type after !!!",
			line:            "!!! no",
			col:             6,
			wantMarker:      "!!!",
			wantTypePrefix:  "no",
			wantMarkerStart: 0,
			wantTypeStart:   4,
			wantInContext:   true,
		},
		{
			name:            "full type after !!!",
			line:            "!!! note",
			col:             8,
			wantMarker:      "!!!",
			wantTypePrefix:  "note",
			wantMarkerStart: 0,
			wantTypeStart:   4,
			wantInContext:   true,
		},
		{
			name:            "after ??? marker",
			line:            "??? ",
			col:             4,
			wantMarker:      "???",
			wantTypePrefix:  "",
			wantMarkerStart: 0,
			wantTypeStart:   4,
			wantInContext:   true,
		},
		{
			name:            "after ???+ marker",
			line:            "???+ ",
			col:             5,
			wantMarker:      "???+",
			wantTypePrefix:  "",
			wantMarkerStart: 0,
			wantTypeStart:   5,
			wantInContext:   true,
		},
		{
			name:            "partial type after ???+",
			line:            "???+ war",
			col:             8,
			wantMarker:      "???+",
			wantTypePrefix:  "war",
			wantMarkerStart: 0,
			wantTypeStart:   5,
			wantInContext:   true,
		},
		{
			name:            "with leading spaces",
			line:            "  !!! tip",
			col:             9,
			wantMarker:      "!!!",
			wantTypePrefix:  "tip",
			wantMarkerStart: 2,
			wantTypeStart:   6,
			wantInContext:   true,
		},
		{
			name:          "not an admonition - regular text",
			line:          "This is not an admonition",
			col:           10,
			wantInContext: false,
		},
		{
			name:          "not an admonition - incomplete marker",
			line:          "!! note",
			col:           7,
			wantInContext: false,
		},
		{
			name:          "not an admonition - marker mid-line",
			line:          "Some text !!! note",
			col:           18,
			wantInContext: false,
		},
		{
			name:          "empty line",
			line:          "",
			col:           0,
			wantInContext: false,
		},
		{
			name:            "marker without space yet",
			line:            "!!!",
			col:             3,
			wantMarker:      "!!!",
			wantTypePrefix:  "",
			wantMarkerStart: 0,
			wantTypeStart:   4,
			wantInContext:   true,
		},
		{
			name:            "??? without space",
			line:            "???",
			col:             3,
			wantMarker:      "???",
			wantTypePrefix:  "",
			wantMarkerStart: 0,
			wantTypeStart:   4,
			wantInContext:   true,
		},
		{
			name:            "???+ without space",
			line:            "???+",
			col:             4,
			wantMarker:      "???+",
			wantTypePrefix:  "",
			wantMarkerStart: 0,
			wantTypeStart:   5,
			wantInContext:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, inContext := getAdmonitionContext(tt.line, tt.col)

			if inContext != tt.wantInContext {
				t.Errorf("inContext = %v, want %v", inContext, tt.wantInContext)
				return
			}

			if !tt.wantInContext {
				return
			}

			if ctx.Marker != tt.wantMarker {
				t.Errorf("marker = %q, want %q", ctx.Marker, tt.wantMarker)
			}
			if ctx.TypePrefix != tt.wantTypePrefix {
				t.Errorf("typePrefix = %q, want %q", ctx.TypePrefix, tt.wantTypePrefix)
			}
			if ctx.MarkerStart != tt.wantMarkerStart {
				t.Errorf("markerStart = %d, want %d", ctx.MarkerStart, tt.wantMarkerStart)
			}
			if ctx.TypeStart != tt.wantTypeStart {
				t.Errorf("typeStart = %d, want %d", ctx.TypeStart, tt.wantTypeStart)
			}
		})
	}
}

func TestGetAdmonitionType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantNil  bool
		wantName string
	}{
		{
			name:     "note type",
			input:    "note",
			wantNil:  false,
			wantName: "note",
		},
		{
			name:     "warning type",
			input:    "warning",
			wantNil:  false,
			wantName: "warning",
		},
		{
			name:     "case insensitive",
			input:    "NOTE",
			wantNil:  false,
			wantName: "note",
		},
		{
			name:     "tip type",
			input:    "tip",
			wantNil:  false,
			wantName: "tip",
		},
		{
			name:    "unknown type",
			input:   "unknown",
			wantNil: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adType := GetAdmonitionType(tt.input)

			if tt.wantNil && adType != nil {
				t.Errorf("expected nil, got %v", adType)
				return
			}
			if !tt.wantNil && adType == nil {
				t.Error("expected non-nil, got nil")
				return
			}
			if adType != nil && adType.Name != tt.wantName {
				t.Errorf("name = %q, want %q", adType.Name, tt.wantName)
			}
		})
	}
}

func TestAllAdmonitionTypes(t *testing.T) {
	types := AllAdmonitionTypes()

	// Check we have all expected types
	expectedTypes := []string{
		"note", "info", "tip", "hint", "success",
		"warning", "caution", "important", "danger", "error",
		"bug", "example", "quote", "abstract", "aside",
	}

	if len(types) != len(expectedTypes) {
		t.Errorf("got %d types, want %d", len(types), len(expectedTypes))
	}

	// Check all expected types exist
	typeMap := make(map[string]bool)
	for _, at := range types {
		typeMap[at.Name] = true
	}

	for _, expected := range expectedTypes {
		if !typeMap[expected] {
			t.Errorf("missing expected type: %s", expected)
		}
	}
}

func TestGetAdmonitionCompletions(t *testing.T) {
	tests := []struct {
		name       string
		ctx        *AdmonitionContext
		wantCount  int
		wantFirst  string
		wantFilter string
	}{
		{
			name: "no prefix returns all types",
			ctx: &AdmonitionContext{
				Marker:     "!!!",
				TypePrefix: "",
				TypeStart:  4,
			},
			wantCount: 15, // All admonition types
		},
		{
			name: "prefix 'n' filters to note",
			ctx: &AdmonitionContext{
				Marker:     "!!!",
				TypePrefix: "n",
				TypeStart:  4,
			},
			wantCount:  1,
			wantFirst:  "note",
			wantFilter: "note",
		},
		{
			name: "prefix 'wa' filters to warning",
			ctx: &AdmonitionContext{
				Marker:     "!!!",
				TypePrefix: "wa",
				TypeStart:  4,
			},
			wantCount:  1,
			wantFirst:  "warning",
			wantFilter: "warning",
		},
		{
			name: "prefix 'e' matches error and example",
			ctx: &AdmonitionContext{
				Marker:     "!!!",
				TypePrefix: "e",
				TypeStart:  4,
			},
			wantCount: 2, // error and example
		},
		{
			name: "prefix 'xyz' matches nothing",
			ctx: &AdmonitionContext{
				Marker:     "!!!",
				TypePrefix: "xyz",
				TypeStart:  4,
			},
			wantCount: 0,
		},
		{
			name: "case insensitive prefix",
			ctx: &AdmonitionContext{
				Marker:     "!!!",
				TypePrefix: "NOTE",
				TypeStart:  4,
			},
			wantCount: 1,
			wantFirst: "note",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := CompletionParams{
				Position: Position{Line: 0, Character: tt.ctx.TypeStart + len(tt.ctx.TypePrefix)},
			}
			items := getAdmonitionCompletions(tt.ctx, params)

			if len(items) != tt.wantCount {
				t.Errorf("got %d items, want %d", len(items), tt.wantCount)
				for _, item := range items {
					t.Logf("  - %s", item.Label)
				}
			}

			if tt.wantFirst != "" && len(items) > 0 && items[0].Label != tt.wantFirst {
				t.Errorf("first item label = %q, want %q", items[0].Label, tt.wantFirst)
			}

			if tt.wantFilter != "" && len(items) > 0 && items[0].FilterText != tt.wantFilter {
				t.Errorf("first item filterText = %q, want %q", items[0].FilterText, tt.wantFilter)
			}

			// Check all items have required fields
			for i, item := range items {
				if item.Label == "" {
					t.Errorf("item %d has empty label", i)
				}
				if item.Kind != CompletionItemKindKeyword {
					t.Errorf("item %d kind = %d, want %d (Keyword)", i, item.Kind, CompletionItemKindKeyword)
				}
				if item.InsertText == "" {
					t.Errorf("item %d has empty insertText", i)
				}
				if item.InsertTextFormat != InsertTextFormatSnippet {
					t.Errorf("item %d format = %d, want %d (Snippet)", i, item.InsertTextFormat, InsertTextFormatSnippet)
				}
				if item.Documentation == nil {
					t.Errorf("item %d has nil documentation", i)
				}
			}
		})
	}
}

func TestAdmonitionCompletionHasCorrectSnippet(t *testing.T) {
	ctx := &AdmonitionContext{
		Marker:     "!!!",
		TypePrefix: "note",
		TypeStart:  4,
	}
	params := CompletionParams{
		Position: Position{Line: 0, Character: 8},
	}

	items := getAdmonitionCompletions(ctx, params)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	// Check the snippet format
	expected := "note \"${1:Note}\""
	if items[0].InsertText != expected {
		t.Errorf("insertText = %q, want %q", items[0].InsertText, expected)
	}
}

func TestAdmonitionTypeMetadata(t *testing.T) {
	// Verify key types have expected metadata
	tests := []struct {
		name        string
		wantColor   string
		wantIcon    string
		wantDescLen int // minimum description length
	}{
		{"note", "#448aff", "pencil", 10},
		{"warning", "#ff9100", "exclamation-triangle", 10},
		{"danger", "#ff5252", "bolt", 10},
		{"tip", "#00bfa5", "lightbulb", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adType := GetAdmonitionType(tt.name)
			if adType == nil {
				t.Fatalf("type %q not found", tt.name)
			}

			if adType.Color != tt.wantColor {
				t.Errorf("color = %q, want %q", adType.Color, tt.wantColor)
			}
			if adType.Icon != tt.wantIcon {
				t.Errorf("icon = %q, want %q", adType.Icon, tt.wantIcon)
			}
			if len(adType.Description) < tt.wantDescLen {
				t.Errorf("description too short: %q", adType.Description)
			}
		})
	}
}

func TestFormatAdmonitionDocumentation(t *testing.T) {
	adType := GetAdmonitionType("warning")
	if adType == nil {
		t.Fatal("warning type not found")
	}

	doc := formatAdmonitionDocumentation(adType)

	// Check that the documentation contains expected parts
	expectedParts := []string{
		"**Warning**",
		adType.Description,
		"Color:",
		adType.Color,
		"Usage:",
		"!!! warning",
	}

	for _, part := range expectedParts {
		if !contains(doc, part) {
			t.Errorf("documentation missing expected part: %q", part)
		}
	}
}

// contains checks if s contains substr (case-sensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
