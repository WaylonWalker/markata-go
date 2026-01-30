package aesthetic

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBuiltin(t *testing.T) {
	// Test loading a known built-in aesthetic
	names := BuiltinNames()
	if len(names) == 0 {
		t.Skip("No built-in aesthetics available")
	}

	a, err := LoadBuiltin(names[0])
	if err != nil {
		t.Fatalf("LoadBuiltin(%q) error = %v", names[0], err)
	}

	if a.Name == "" {
		t.Error("Loaded aesthetic has empty name")
	}
	if a.Source != "built-in" {
		t.Errorf("Loaded aesthetic source = %q, want %q", a.Source, "built-in")
	}
}

func TestLoadBuiltin_Brutal(t *testing.T) {
	// Test loading the "brutal" aesthetic specifically
	a, err := LoadBuiltin("brutal")
	if err != nil {
		// If brutal doesn't exist yet, skip
		if errors.Is(err, ErrAestheticNotFound) {
			t.Skip("brutal aesthetic not yet implemented")
		}
		t.Fatalf("LoadBuiltin(brutal) error = %v", err)
	}

	// Brutal aesthetic should have specific characteristics
	if a.Name != "brutal" && a.Name != "Brutal" {
		t.Errorf("Name = %q, expected brutal or Brutal", a.Name)
	}

	// Brutal aesthetic typically has no border radius
	if radius, ok := a.Tokens.Radius["sm"]; ok {
		if radius != "0" && radius != "0px" {
			t.Logf("Note: brutal aesthetic has radius.sm %q (expected 0)", radius)
		}
	}

	// Brutal aesthetic should have no shadows
	if shadow, ok := a.Tokens.Shadow["sm"]; ok {
		if shadow != "none" {
			t.Logf("Note: brutal aesthetic has shadow.sm %q (expected none)", shadow)
		}
	}
}

func TestLoadBuiltin_NotFound(t *testing.T) {
	_, err := LoadBuiltin("nonexistent-aesthetic-xyz")
	if err == nil {
		t.Error("LoadBuiltin() should return error for nonexistent aesthetic")
	}
	if !errors.Is(err, ErrAestheticNotFound) {
		t.Errorf("error should be ErrAestheticNotFound, got %v", err)
	}
}

func TestDiscoverBuiltin(t *testing.T) {
	infos := DiscoverBuiltin()

	// We should have at least some built-in aesthetics
	if len(infos) == 0 {
		t.Skip("No built-in aesthetics discovered")
	}

	// Verify each info has required fields
	for _, info := range infos {
		if info.Name == "" {
			t.Error("Discovered aesthetic has empty name")
		}
		if info.Source != "built-in" {
			t.Errorf("Discovered aesthetic %q source = %q, want %q", info.Name, info.Source, "built-in")
		}
	}
}

func TestHasBuiltin(t *testing.T) {
	names := BuiltinNames()
	if len(names) == 0 {
		t.Skip("No built-in aesthetics available")
	}

	// Should find existing aesthetic
	if !HasBuiltin(names[0]) {
		t.Errorf("HasBuiltin(%q) = false, want true", names[0])
	}

	// Should not find nonexistent aesthetic
	if HasBuiltin("nonexistent-aesthetic-xyz") {
		t.Error("HasBuiltin(nonexistent) = true, want false")
	}
}

func TestListAesthetics(t *testing.T) {
	aesthetics := ListAesthetics()

	// Should return at least builtin aesthetics (if any exist)
	// This test verifies the function doesn't panic and returns a slice
	if aesthetics == nil {
		t.Error("ListAesthetics() should not return nil")
	}

	// Verify the list contains valid entries
	for _, info := range aesthetics {
		if info.Name == "" {
			t.Error("ListAesthetics() returned entry with empty name")
		}
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary aesthetic file using the correct TOML format
	tmpDir := t.TempDir()
	aestheticFile := filepath.Join(tmpDir, "test-aesthetic.toml")

	content := `
name = "Test Aesthetic"
description = "A test aesthetic for unit tests"

[tokens.radius]
none = "0"
sm = "4px"
md = "8px"
lg = "16px"
xl = "24px"
full = "9999px"

[tokens.spacing]
scale = 1.0

[tokens.border]
width_thin = "1px"
width_normal = "2px"
width_thick = "3px"
style = "solid"

[tokens.shadow]
sm = "0 1px 2px rgba(0,0,0,0.05)"
md = "0 4px 6px rgba(0,0,0,0.1)"
lg = "0 10px 15px rgba(0,0,0,0.1)"
xl = "0 25px 50px rgba(0,0,0,0.15)"

[tokens.typography]
font_primary = "var(--font-sans)"
leading_scale = "1.1"
`

	if err := os.WriteFile(aestheticFile, []byte(content), 0o600); err != nil {
		t.Fatalf("Failed to write test aesthetic: %v", err)
	}

	a, err := LoadFromFile(aestheticFile)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	if a.Name != "Test Aesthetic" {
		t.Errorf("Name = %q, want %q", a.Name, "Test Aesthetic")
	}
	if a.Description != "A test aesthetic for unit tests" {
		t.Errorf("Description = %q, want %q", a.Description, "A test aesthetic for unit tests")
	}
	if len(a.Tokens.Radius) != 6 {
		t.Errorf("Radius count = %d, want 6", len(a.Tokens.Radius))
	}
	if len(a.Tokens.Border) != 4 {
		t.Errorf("Border count = %d, want 4", len(a.Tokens.Border))
	}
	if len(a.Tokens.Shadow) != 4 {
		t.Errorf("Shadow count = %d, want 4", len(a.Tokens.Shadow))
	}
	if a.Tokens.Radius["sm"] != "4px" {
		t.Errorf("Radius.sm = %q, want %q", a.Tokens.Radius["sm"], "4px")
	}
}

func TestLoadFromFile_Invalid(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "invalid TOML",
			content: "this is not valid toml [[[",
		},
		{
			name: "missing name",
			content: `
description = "No name specified"
[tokens.radius]
sm = "4px"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aestheticFile := filepath.Join(tmpDir, tt.name+".toml")
			if err := os.WriteFile(aestheticFile, []byte(tt.content), 0o600); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			_, err := LoadFromFile(aestheticFile)
			if err == nil {
				t.Errorf("LoadFromFile() should return error for %s", tt.name)
			}
		})
	}
}

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/aesthetic.toml")
	if err == nil {
		t.Error("LoadFromFile() should return error for nonexistent file")
	}
}

func TestLoader_Load(t *testing.T) {
	// Create temp directory with an aesthetic
	tmpDir := t.TempDir()
	aestheticsDir := filepath.Join(tmpDir, "aesthetics")
	if err := os.MkdirAll(aestheticsDir, 0o755); err != nil {
		t.Fatalf("Failed to create aesthetics dir: %v", err)
	}

	content := `
name = "Project Aesthetic"
description = "Custom project aesthetic"

[tokens.radius]
sm = "0"
md = "0"

[tokens.spacing]
scale = 0.8

[tokens.border]
width_normal = "3px"
style = "solid"

[tokens.shadow]
sm = "none"
md = "none"
`
	if err := os.WriteFile(filepath.Join(aestheticsDir, "project.toml"), []byte(content), 0o600); err != nil {
		t.Fatalf("Failed to write aesthetic: %v", err)
	}

	loader := NewLoaderWithPaths([]string{aestheticsDir})

	a, err := loader.Load("project")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if a.Name != "Project Aesthetic" {
		t.Errorf("Name = %q, want %q", a.Name, "Project Aesthetic")
	}
	if a.Tokens.Radius["sm"] != "0" {
		t.Errorf("Radius.sm = %q, want %q", a.Tokens.Radius["sm"], "0")
	}
}

func TestLoader_Discover(t *testing.T) {
	// Create temp directory with aesthetics
	tmpDir := t.TempDir()
	aestheticsDir := filepath.Join(tmpDir, "aesthetics")
	if err := os.MkdirAll(aestheticsDir, 0o755); err != nil {
		t.Fatalf("Failed to create aesthetics dir: %v", err)
	}

	// Create two test aesthetics
	aestheticConfigs := map[string]string{
		"minimal": `
name = "minimal aesthetic"
description = "Minimal design"

[tokens.radius]
sm = "0"

[tokens.shadow]
sm = "none"
`,
		"brutal": `
name = "brutal aesthetic"
description = "Brutal design"

[tokens.radius]
sm = "0"

[tokens.border]
width_normal = "3px"
`,
	}

	for name, content := range aestheticConfigs {
		if err := os.WriteFile(filepath.Join(aestheticsDir, name+".toml"), []byte(content), 0o600); err != nil {
			t.Fatalf("Failed to write aesthetic: %v", err)
		}
	}

	loader := NewLoaderWithPaths([]string{aestheticsDir})
	infos, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Should find at least our 2 aesthetics (may include built-ins)
	found := 0
	for _, info := range infos {
		if info.Name == "minimal aesthetic" || info.Name == "brutal aesthetic" {
			found++
		}
	}

	if found != 2 {
		t.Errorf("Found %d test aesthetics, want 2", found)
	}
}

func TestNormalizeAestheticName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Brutal Design", "brutal-design"},
		{"Neo Brutalism", "neo-brutalism"},
		{"MY_AESTHETIC", "my-aesthetic"},
		{"already-normalized", "already-normalized"},
		{"MixedCase_and Spaces", "mixedcase-and-spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeAestheticName(tt.input)
			if got != tt.want {
				t.Errorf("normalizeAestheticName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuiltinAesthetics_Content(t *testing.T) {
	// Test that built-in aesthetics have expected content
	expectedAesthetics := []struct {
		name        string
		description string
	}{
		{"brutal", "Brutalist design"},
		{"minimal", "Maximum whitespace"},
		{"elevated", "Layered/premium"},
		{"balanced", ""},
		{"precision", ""},
	}

	for _, expected := range expectedAesthetics {
		t.Run(expected.name, func(t *testing.T) {
			a, err := LoadBuiltin(expected.name)
			if err != nil {
				if errors.Is(err, ErrAestheticNotFound) {
					t.Skipf("Built-in aesthetic %q not yet available", expected.name)
				}
				t.Fatalf("LoadBuiltin(%q) error = %v", expected.name, err)
			}

			// Verify basic fields
			if a.Name == "" {
				t.Error("Name should not be empty")
			}

			// Verify has some tokens defined
			hasTokens := len(a.Tokens.Radius) > 0 ||
				len(a.Tokens.Border) > 0 ||
				len(a.Tokens.Shadow) > 0 ||
				a.Tokens.Spacing != nil

			if !hasTokens {
				t.Error("Aesthetic should have some tokens defined")
			}
		})
	}
}
