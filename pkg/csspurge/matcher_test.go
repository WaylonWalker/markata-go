package csspurge

import (
	"strings"
	"testing"
)

func TestPurgeCSS(t *testing.T) {
	tests := []struct {
		name          string
		css           string
		usedClasses   []string
		usedIDs       []string
		usedElements  []string
		preserve      []string
		wantKeptRules int
		wantRemoved   bool
		checkOutput   func(t *testing.T, output string)
	}{
		{
			name:          "keep used class",
			css:           `.foo { color: red; } .bar { color: blue; }`,
			usedClasses:   []string{"foo"},
			wantKeptRules: 1,
			wantRemoved:   true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, ".foo") {
					t.Error("output should contain .foo")
				}
				if strings.Contains(output, ".bar") {
					t.Error("output should not contain .bar")
				}
			},
		},
		{
			name:          "keep used ID",
			css:           `#header { height: 60px; } #footer { height: 40px; }`,
			usedIDs:       []string{"header"},
			wantKeptRules: 1,
			wantRemoved:   true,
		},
		{
			name:          "keep used element",
			css:           `div { margin: 0; } span { padding: 0; }`,
			usedElements:  []string{"div"},
			wantKeptRules: 1,
			wantRemoved:   true,
		},
		{
			name:          "keep preserved pattern",
			css:           `.js-toggle { display: none; } .unused { color: red; }`,
			usedClasses:   []string{},
			preserve:      []string{"js-*"},
			wantKeptRules: 1,
			wantRemoved:   true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, ".js-toggle") {
					t.Error("output should contain .js-toggle")
				}
			},
		},
		{
			name:          "keep keyframes always",
			css:           `@keyframes spin { from { transform: rotate(0); } to { transform: rotate(360deg); } }`,
			usedClasses:   []string{},
			wantKeptRules: 1,
			wantRemoved:   false,
		},
		{
			name:          "keep font-face always",
			css:           `@font-face { font-family: 'Custom'; src: url('font.woff2'); }`,
			usedClasses:   []string{},
			wantKeptRules: 1,
			wantRemoved:   false,
		},
		{
			name:          "keep import always",
			css:           `@import url('other.css');`,
			usedClasses:   []string{},
			wantKeptRules: 1,
			wantRemoved:   false,
		},
		{
			name:          "purge inside media query",
			css:           `@media (max-width: 600px) { .used { display: block; } .unused { display: none; } }`,
			usedClasses:   []string{"used"},
			wantKeptRules: 1, // The @media rule itself
			wantRemoved:   true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, ".used") {
					t.Error("output should contain .used")
				}
				if strings.Contains(output, ".unused") {
					t.Error("output should not contain .unused")
				}
			},
		},
		{
			name:          "remove empty media query",
			css:           `@media (max-width: 600px) { .unused { display: none; } } .used { color: red; }`,
			usedClasses:   []string{"used"},
			wantKeptRules: 1,
			wantRemoved:   true,
			checkOutput: func(t *testing.T, output string) {
				if strings.Contains(output, "@media") {
					t.Error("output should not contain empty @media")
				}
			},
		},
		{
			name:          "keep universal selector",
			css:           `* { box-sizing: border-box; } .unused { color: red; }`,
			usedClasses:   []string{},
			wantKeptRules: 1,
			wantRemoved:   true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "*") {
					t.Error("output should contain universal selector")
				}
			},
		},
		{
			name:          "comma-separated selectors - one used",
			css:           `h1, h2, .title { font-weight: bold; }`,
			usedElements:  []string{"h1"},
			wantKeptRules: 1,
			wantRemoved:   false,
		},
		{
			name:          "comma-separated selectors - none used",
			css:           `.a, .b, .c { color: red; }`,
			usedClasses:   []string{},
			wantKeptRules: 0,
			wantRemoved:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			used := NewUsedSelectors()
			for _, c := range tt.usedClasses {
				used.Classes[c] = true
			}
			for _, id := range tt.usedIDs {
				used.IDs[id] = true
			}
			for _, elem := range tt.usedElements {
				used.Elements[elem] = true
			}

			opts := PurgeOptions{
				Preserve: tt.preserve,
			}

			output, stats := PurgeCSS(tt.css, used, opts)

			if stats.KeptRules != tt.wantKeptRules {
				t.Errorf("kept %d rules, want %d", stats.KeptRules, tt.wantKeptRules)
			}

			hasRemoved := stats.RemovedRules > 0
			if hasRemoved != tt.wantRemoved {
				t.Errorf("removed = %v, want %v (removed %d rules)", hasRemoved, tt.wantRemoved, stats.RemovedRules)
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestPurgeStats(t *testing.T) {
	used := NewUsedSelectors()
	used.Classes["used"] = true

	css := `.used { color: red; } .unused1 { color: blue; } .unused2 { color: green; }`
	_, stats := PurgeCSS(css, used, PurgeOptions{})

	if stats.TotalRules != 3 {
		t.Errorf("TotalRules = %d, want 3", stats.TotalRules)
	}
	if stats.KeptRules != 1 {
		t.Errorf("KeptRules = %d, want 1", stats.KeptRules)
	}
	if stats.RemovedRules != 2 {
		t.Errorf("RemovedRules = %d, want 2", stats.RemovedRules)
	}
	if stats.OriginalSize == 0 {
		t.Error("OriginalSize should not be 0")
	}
	if stats.PurgedSize >= stats.OriginalSize {
		t.Error("PurgedSize should be less than OriginalSize")
	}

	savings := stats.SavingsPercent()
	if savings <= 0 || savings >= 100 {
		t.Errorf("SavingsPercent() = %f, want between 0 and 100", savings)
	}
}

func TestDefaultPreservePatterns(t *testing.T) {
	patterns := DefaultPreservePatterns()
	if len(patterns) == 0 {
		t.Error("DefaultPreservePatterns() should return patterns")
	}

	// Check some expected patterns
	expected := []string{"js-*", "htmx-*", "active", "hidden", "dark", "light"}
	patternSet := make(map[string]bool)
	for _, p := range patterns {
		patternSet[p] = true
	}

	for _, e := range expected {
		if !patternSet[e] {
			t.Errorf("expected pattern %q not found", e)
		}
	}
}

func TestMatchesPreservePatterns(t *testing.T) {
	patterns := []string{"js-*", "htmx-*", "active"}

	tests := []struct {
		name string
		want bool
	}{
		{"js-toggle", true},
		{"js-", true},
		{"htmx-request", true},
		{"active", true},
		{"inactive", false},
		{"other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesPreservePatterns(tt.name, patterns)
			if got != tt.want {
				t.Errorf("matchesPreservePatterns(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsSelectorUsed(t *testing.T) {
	used := NewUsedSelectors()
	used.Classes["foo"] = true
	used.IDs["bar"] = true
	used.Elements["div"] = true

	tests := []struct {
		selector string
		want     bool
	}{
		{".foo", true},
		{".bar", false},
		{"#bar", true},
		{"#baz", false},
		{"div", true},
		{"span", false},
		{"div.foo", true},
		{"div.unused", false},
		{".foo, .unused", true},       // One of multiple matches
		{".unused1, .unused2", false}, // None match
	}

	for _, tt := range tests {
		t.Run(tt.selector, func(t *testing.T) {
			got := isSelectorUsed(tt.selector, used, nil)
			if got != tt.want {
				t.Errorf("isSelectorUsed(%q) = %v, want %v", tt.selector, got, tt.want)
			}
		})
	}
}
