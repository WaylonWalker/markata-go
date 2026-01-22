package palettes

import (
	"testing"
)

func TestNewPalette(t *testing.T) {
	p := NewPalette("test", VariantDark)

	if p.Name != "test" {
		t.Errorf("Name = %q, want %q", p.Name, "test")
	}
	if p.Variant != VariantDark {
		t.Errorf("Variant = %q, want %q", p.Variant, VariantDark)
	}
	if p.Colors == nil {
		t.Error("Colors map is nil")
	}
	if p.Semantic == nil {
		t.Error("Semantic map is nil")
	}
	if p.Components == nil {
		t.Error("Components map is nil")
	}
}

func TestPalette_Resolve(t *testing.T) {
	p := &Palette{
		Name:    "test",
		Variant: VariantDark,
		Colors: map[string]string{
			"red":   "#ff0000",
			"blue":  "#0000ff",
			"green": "#00ff00",
		},
		Semantic: map[string]string{
			"text-primary": "red",
			"accent":       "blue",
		},
		Components: map[string]string{
			"button-bg": "accent",
			"link":      "green",
		},
	}

	tests := []struct {
		name     string
		colorRef string
		want     string
	}{
		{"raw color", "red", "#ff0000"},
		{"raw color normalized", "blue", "#0000ff"},
		{"semantic referencing raw", "text-primary", "#ff0000"},
		{"semantic referencing raw 2", "accent", "#0000ff"},
		{"component referencing semantic", "button-bg", "#0000ff"},
		{"component referencing raw", "link", "#00ff00"},
		{"unknown color", "unknown", ""},
		{"direct hex", "#aabbcc", "#aabbcc"},
		{"short hex", "#abc", "#aabbcc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.Resolve(tt.colorRef)
			if got != tt.want {
				t.Errorf("Resolve(%q) = %q, want %q", tt.colorRef, got, tt.want)
			}
		})
	}
}

func TestPalette_Resolve_CircularReference(t *testing.T) {
	p := &Palette{
		Name:    "test",
		Variant: VariantDark,
		Colors:  map[string]string{},
		Semantic: map[string]string{
			"a": "b",
		},
		Components: map[string]string{
			"b": "a", // Circular: b -> a -> b
		},
	}

	// Should return empty string for circular references
	got := p.Resolve("a")
	if got != "" {
		t.Errorf("Resolve() with circular reference should return empty, got %q", got)
	}
}

func TestPalette_ResolveAll(t *testing.T) {
	p := &Palette{
		Name:    "test",
		Variant: VariantDark,
		Colors: map[string]string{
			"red":  "#ff0000",
			"blue": "#0000ff",
		},
		Semantic: map[string]string{
			"primary": "red",
		},
		Components: map[string]string{
			"button": "primary",
		},
	}

	resolved, err := p.ResolveAll()
	if err != nil {
		t.Fatalf("ResolveAll() error = %v", err)
	}

	if len(resolved) != 4 {
		t.Errorf("ResolveAll() returned %d colors, want 4", len(resolved))
	}

	if resolved["red"] != "#ff0000" {
		t.Errorf("resolved[red] = %q, want #ff0000", resolved["red"])
	}
	if resolved["primary"] != "#ff0000" {
		t.Errorf("resolved[primary] = %q, want #ff0000", resolved["primary"])
	}
	if resolved["button"] != "#ff0000" {
		t.Errorf("resolved[button] = %q, want #ff0000", resolved["button"])
	}
}

func TestPalette_Validate(t *testing.T) {
	tests := []struct {
		name     string
		palette  *Palette
		wantErrs int
	}{
		{
			name: "valid palette",
			palette: &Palette{
				Name:    "test",
				Variant: VariantDark,
				Colors: map[string]string{
					"red": "#ff0000",
				},
				Semantic: map[string]string{
					"primary": "red",
				},
				Components: map[string]string{
					"button": "primary",
				},
			},
			wantErrs: 0,
		},
		{
			name: "missing name",
			palette: &Palette{
				Name:    "",
				Variant: VariantDark,
				Colors:  map[string]string{},
			},
			wantErrs: 1,
		},
		{
			name: "invalid variant",
			palette: &Palette{
				Name:    "test",
				Variant: "invalid",
				Colors:  map[string]string{},
			},
			wantErrs: 1,
		},
		{
			name: "invalid hex color",
			palette: &Palette{
				Name:    "test",
				Variant: VariantDark,
				Colors: map[string]string{
					"bad": "not-a-color",
				},
			},
			wantErrs: 1,
		},
		{
			name: "semantic references unknown",
			palette: &Palette{
				Name:    "test",
				Variant: VariantDark,
				Colors:  map[string]string{},
				Semantic: map[string]string{
					"primary": "unknown",
				},
			},
			wantErrs: 1,
		},
		{
			name: "semantic references semantic",
			palette: &Palette{
				Name:    "test",
				Variant: VariantDark,
				Colors: map[string]string{
					"red": "#ff0000",
				},
				Semantic: map[string]string{
					"primary":   "red",
					"secondary": "primary", // Not allowed
				},
			},
			wantErrs: 1,
		},
		{
			name: "component references component",
			palette: &Palette{
				Name:    "test",
				Variant: VariantDark,
				Colors: map[string]string{
					"red": "#ff0000",
				},
				Semantic: map[string]string{
					"primary": "red",
				},
				Components: map[string]string{
					"button":  "primary",
					"button2": "button", // Not allowed
				},
			},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.palette.Validate()
			if len(errs) != tt.wantErrs {
				t.Errorf("Validate() returned %d errors, want %d", len(errs), tt.wantErrs)
				for _, err := range errs {
					t.Logf("  error: %v", err)
				}
			}
		})
	}
}

func TestPalette_Clone(t *testing.T) {
	original := &Palette{
		Name:        "original",
		Variant:     VariantDark,
		Author:      "Test Author",
		Description: "Test Description",
		Colors: map[string]string{
			"red": "#ff0000",
		},
		Semantic: map[string]string{
			"primary": "red",
		},
		Components: map[string]string{
			"button": "primary",
		},
	}

	clone := original.Clone()

	// Verify it's a deep copy
	if clone.Name != original.Name {
		t.Errorf("Clone Name = %q, want %q", clone.Name, original.Name)
	}

	// Modify clone and verify original unchanged
	clone.Name = "modified"
	clone.Colors["red"] = "#00ff00"

	if original.Name == "modified" {
		t.Error("Modifying clone affected original Name")
	}
	if original.Colors["red"] != "#ff0000" {
		t.Error("Modifying clone affected original Colors")
	}
}

func TestIsHexColor(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"#fff", true},
		{"#ffffff", true},
		{"#FFF", true},
		{"#FFFFFF", true},
		{"fff", true},
		{"ffffff", true},
		{"#ffff", true},     // RGBA short
		{"#ffffffff", true}, // RGBA long
		{"", false},
		{"#ff", false},
		{"#fffff", false},
		{"#gggggg", false},
		{"red", false},
		{"rgb(0,0,0)", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isHexColor(tt.input)
			if got != tt.want {
				t.Errorf("isHexColor(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeHexColor(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"#fff", "#ffffff"},
		{"#FFF", "#ffffff"},
		{"fff", "#ffffff"},
		{"#ffffff", "#ffffff"},
		{"#FFFFFF", "#ffffff"},
		{"ffffff", "#ffffff"},
		{"#abc", "#aabbcc"},
		{"#ABC", "#aabbcc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeHexColor(tt.input)
			if got != tt.want {
				t.Errorf("normalizeHexColor(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
