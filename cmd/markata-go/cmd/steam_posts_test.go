package cmd

import (
	"testing"
)

func TestEscapeYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Simple strings that don't need quoting
		{"simple", "Portal 2", "Portal 2"},
		{"alphanumeric", "Game123", "Game123"},

		// Empty string
		{"empty", "", `""`},

		// Strings with colons (common in game titles)
		{"colon", "Half-Life: Alyx", `"Half-Life: Alyx"`},
		{"multiple_colons", "Title: Part 1: Episode 2", `"Title: Part 1: Episode 2"`},

		// Strings with quotes
		{"double_quote", `Game with "quotes"`, `"Game with \"quotes\""`},

		// Strings with newlines
		{"newline", "Line 1\nLine 2", `"Line 1\nLine 2"`},

		// Strings with hash (would start YAML comment)
		{"hash", "Game #1", `"Game #1"`},

		// YAML reserved words
		{"true", "true", `"true"`},
		{"false", "false", `"false"`},
		{"null", "null", `"null"`},
		{"yes", "yes", `"yes"`},
		{"no", "no", `"no"`},
		{"on", "on", `"on"`},
		{"off", "off", `"off"`},
		{"tilde", "~", `"~"`},

		// Case insensitive reserved words
		{"TRUE", "TRUE", `"TRUE"`},
		{"False", "False", `"False"`},

		// Leading/trailing whitespace
		{"leading_space", " leading", `" leading"`},
		{"trailing_space", "trailing ", `"trailing "`},

		// Starting with special characters
		{"start_dash", "-dash", `"-dash"`},
		{"start_asterisk", "*star", `"*star"`},
		{"start_bracket", "[array", `"[array"`},
		{"start_brace", "{object", `"{object"`},
		{"start_ampersand", "&anchor", `"&anchor"`},
		{"start_exclamation", "!tag", `"!tag"`},
		{"start_pipe", "|literal", `"|literal"`},
		{"start_greater", ">folded", `">folded"`},
		{"start_percent", "%directive", `"%directive"`},

		// Multiple special characters
		{"complex", "Game: Part #1 (\"Special Edition\")", `"Game: Part #1 (\"Special Edition\")"`},

		// Backslash - doesn't require quoting by itself in YAML (only if other chars present)
		{"backslash", `path\to\file`, `path\to\file`},

		// Tab character
		{"tab", "with\ttab", `"with\ttab"`},

		// Carriage return
		{"carriage_return", "with\rreturn", `"with\rreturn"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeYAML(tt.input)
			if got != tt.expected {
				t.Errorf("escapeYAML(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEscapeYAMLArray(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{"empty", []string{}, "[]"},
		{"single", []string{"tag"}, "[tag]"},
		{"multiple", []string{"steam", "game", "rpg"}, "[steam, game, rpg]"},
		{"with_special", []string{"steam", "game:rpg"}, `[steam, "game:rpg"]`},
		{"all_special", []string{"tag: one", "tag: two"}, `["tag: one", "tag: two"]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeYAMLArray(tt.input)
			if got != tt.expected {
				t.Errorf("escapeYAMLArray(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitizeForFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "Portal 2", "portal-2"},
		{"colon", "Half-Life: Alyx", "half-life--alyx"},
		{"special_chars", "Game™ (2024)", "game---2024-"},
		{"unicode", "ゲーム", "---"},
		{"multiple_spaces", "A  B   C", "a--b---c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeForFilename(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeForFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCoalesceStr(t *testing.T) {
	value := "value"
	tests := []struct {
		name     string
		input    *string
		def      string
		expected string
	}{
		{"nil", nil, "default", "default"},
		{"non_nil", &value, "default", "value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := coalesceStr(tt.input, tt.def)
			if got != tt.expected {
				t.Errorf("coalesceStr(%v, %q) = %q, want %q", tt.input, tt.def, got, tt.expected)
			}
		})
	}
}
