package aesthetic

import (
	"strings"
	"testing"
)

func TestNewAesthetic(t *testing.T) {
	a := NewAesthetic("brutal")

	if a.Name != "brutal" {
		t.Errorf("Name = %q, want %q", a.Name, "brutal")
	}
	if a.Tokens.Radius == nil {
		t.Error("Tokens.Radius map is nil")
	}
	if a.Tokens.Spacing == nil {
		t.Error("Tokens.Spacing map is nil")
	}
	if a.Tokens.Border == nil {
		t.Error("Tokens.Border map is nil")
	}
	if a.Tokens.Shadow == nil {
		t.Error("Tokens.Shadow map is nil")
	}
	if a.Tokens.Typography == nil {
		t.Error("Tokens.Typography map is nil")
	}
}

func TestAesthetic_CSS(t *testing.T) {
	a := &Aesthetic{
		Name:        "brutal",
		Description: "Brutalist design aesthetic",
		Tokens: Tokens{
			Radius: map[string]string{
				"none": "0",
				"sm":   "0",
				"md":   "0",
				"lg":   "0",
			},
			Spacing: &SpacingTokens{
				Scale: 0.75,
			},
			Border: map[string]string{
				"width_thin":   "2px",
				"width_normal": "3px",
				"width_thick":  "4px",
				"style":        "solid",
			},
			Shadow: map[string]string{
				"sm": "none",
				"md": "none",
				"lg": "none",
			},
			Typography: map[string]string{
				"font_primary":  "var(--font-mono)",
				"leading_scale": "1.0",
			},
		},
	}

	css := a.GenerateCSS()

	// Should contain :root selector
	if !strings.Contains(css, ":root") {
		t.Error("CSS should contain :root selector")
	}

	// Should contain radius variables
	if !strings.Contains(css, "--radius") {
		t.Error("CSS should contain radius variables")
	}

	// Should contain border variables
	if !strings.Contains(css, "--border") {
		t.Error("CSS should contain border variables")
	}

	// Should contain shadow variables
	if !strings.Contains(css, "--shadow") {
		t.Error("CSS should contain shadow variables")
	}
}

func TestAesthetic_CSS_Minified(t *testing.T) {
	a := &Aesthetic{
		Name: "minimal",
		Tokens: Tokens{
			Radius: map[string]string{
				"sm": "0",
			},
			Border: map[string]string{
				"width_normal": "1px",
			},
		},
	}

	css := a.GenerateCSSMinified()

	// Minified CSS should not have unnecessary whitespace
	lines := strings.Split(strings.TrimSpace(css), "\n")
	if len(lines) > 2 {
		t.Errorf("Minified CSS should have few lines, got %d", len(lines))
	}

	// Should still contain key definitions
	if !strings.Contains(css, "--radius") || !strings.Contains(css, "--border") {
		t.Error("Minified CSS should contain variables")
	}
}

func TestAesthetic_DefaultValues(t *testing.T) {
	tests := []struct {
		name      string
		aesthetic *Aesthetic
		tokenType string
		tokenKey  string
		wantVal   string
	}{
		{
			name: "default radius none",
			aesthetic: &Aesthetic{
				Name: "test",
				Tokens: Tokens{
					Radius: map[string]string{},
				},
			},
			tokenType: "radius",
			tokenKey:  "none",
			wantVal:   "0",
		},
		{
			name: "default border width",
			aesthetic: &Aesthetic{
				Name: "test",
				Tokens: Tokens{
					Border: map[string]string{},
				},
			},
			tokenType: "border",
			tokenKey:  "width_normal",
			wantVal:   "1px",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.aesthetic.GetWithDefault(tt.tokenType, tt.tokenKey, tt.wantVal)
			if got != tt.wantVal {
				t.Errorf("GetWithDefault(%q, %q) = %q, want %q", tt.tokenType, tt.tokenKey, got, tt.wantVal)
			}
		})
	}
}

func TestAesthetic_Get(t *testing.T) {
	a := &Aesthetic{
		Name: "test",
		Tokens: Tokens{
			Radius: map[string]string{
				"sm": "4px",
				"md": "8px",
			},
			Border: map[string]string{
				"width_normal": "2px",
				"style":        "solid",
			},
			Shadow: map[string]string{
				"sm": "0 1px 2px rgba(0,0,0,0.1)",
			},
		},
	}

	tests := []struct {
		tokenType string
		tokenKey  string
		want      string
	}{
		{"radius", "sm", "4px"},
		{"radius", "md", "8px"},
		{"border", "width_normal", "2px"},
		{"border", "style", "solid"},
		{"shadow", "sm", "0 1px 2px rgba(0,0,0,0.1)"},
		{"radius", "unknown", ""},
		{"unknown", "sm", ""},
	}

	for _, tt := range tests {
		name := tt.tokenType + "." + tt.tokenKey
		t.Run(name, func(t *testing.T) {
			got := a.Get(tt.tokenType, tt.tokenKey)
			if got != tt.want {
				t.Errorf("Get(%q, %q) = %q, want %q", tt.tokenType, tt.tokenKey, got, tt.want)
			}
		})
	}
}

func TestAesthetic_Clone(t *testing.T) {
	original := &Aesthetic{
		Name:        "original",
		Description: "Original aesthetic",
		Tokens: Tokens{
			Radius: map[string]string{
				"sm": "4px",
			},
			Border: map[string]string{
				"width_normal": "1px",
			},
			Shadow: map[string]string{
				"sm": "0 1px 2px rgba(0,0,0,0.1)",
			},
		},
	}

	clone := original.Clone()

	// Verify it's a deep copy
	if clone.Name != original.Name {
		t.Errorf("Clone Name = %q, want %q", clone.Name, original.Name)
	}

	// Modify clone and verify original unchanged
	clone.Name = "modified"
	clone.Tokens.Radius["sm"] = "8px"
	clone.Tokens.Border["width_normal"] = "2px"

	if original.Name == "modified" {
		t.Error("Modifying clone affected original Name")
	}
	if original.Tokens.Radius["sm"] != "4px" {
		t.Error("Modifying clone affected original Radius")
	}
	if original.Tokens.Border["width_normal"] != "1px" {
		t.Error("Modifying clone affected original Border")
	}
}

func TestAesthetic_Validate(t *testing.T) {
	tests := []struct {
		name      string
		aesthetic *Aesthetic
		wantErrs  int
	}{
		{
			name: "valid aesthetic",
			aesthetic: &Aesthetic{
				Name:        "valid",
				Description: "A valid aesthetic",
				Tokens: Tokens{
					Radius: map[string]string{
						"sm": "4px",
					},
					Border: map[string]string{
						"width_normal": "1px",
					},
				},
			},
			wantErrs: 0,
		},
		{
			name: "missing name",
			aesthetic: &Aesthetic{
				Name: "",
				Tokens: Tokens{
					Radius: map[string]string{
						"sm": "4px",
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "nil tokens maps",
			aesthetic: &Aesthetic{
				Name: "test",
				Tokens: Tokens{
					Radius: nil,
					Border: nil,
					Shadow: nil,
				},
			},
			wantErrs: 0, // nil maps are allowed, will use defaults
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.aesthetic.Validate()
			if len(errs) != tt.wantErrs {
				t.Errorf("Validate() returned %d errors, want %d", len(errs), tt.wantErrs)
				for _, err := range errs {
					t.Logf("  error: %v", err)
				}
			}
		})
	}
}

func TestAesthetic_Merge(t *testing.T) {
	base := &Aesthetic{
		Name: "base",
		Tokens: Tokens{
			Radius: map[string]string{
				"sm": "4px",
				"md": "8px",
			},
			Border: map[string]string{
				"width_normal": "1px",
				"style":        "solid",
			},
		},
	}

	override := &Aesthetic{
		Name: "custom",
		Tokens: Tokens{
			Radius: map[string]string{
				"sm": "0", // Override sm to 0
			},
			Shadow: map[string]string{
				"sm": "none", // Add new shadow section
			},
		},
	}

	merged := base.Merge(override)

	// Name should come from override
	if merged.Name != "custom" {
		t.Errorf("Merged Name = %q, want %q", merged.Name, "custom")
	}

	// radius.sm should be overridden to 0
	if merged.Tokens.Radius["sm"] != "0" {
		t.Errorf("radius.sm = %q, want %q", merged.Tokens.Radius["sm"], "0")
	}

	// radius.md should be preserved from base
	if merged.Tokens.Radius["md"] != "8px" {
		t.Errorf("radius.md = %q, want %q", merged.Tokens.Radius["md"], "8px")
	}

	// border should be preserved (override has nil Border)
	if merged.Tokens.Border["width_normal"] != "1px" {
		t.Errorf("border.width_normal = %q, want %q", merged.Tokens.Border["width_normal"], "1px")
	}

	// shadow.sm should be added from override
	if merged.Tokens.Shadow["sm"] != "none" {
		t.Errorf("shadow.sm = %q, want %q", merged.Tokens.Shadow["sm"], "none")
	}
}

func TestAesthetic_SpacingScale(t *testing.T) {
	tests := []struct {
		name  string
		scale float64
		base  float64
		want  float64
	}{
		{"brutal compact", 0.75, 1.0, 0.75},
		{"minimal spacious", 1.5, 1.0, 1.5},
		{"default", 1.0, 1.0, 1.0},
		{"elevated generous", 1.25, 1.0, 1.25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Aesthetic{
				Name: tt.name,
				Tokens: Tokens{
					Spacing: &SpacingTokens{
						Scale: tt.scale,
					},
				},
			}

			got := a.GetSpacingScale()
			if got != tt.want {
				t.Errorf("GetSpacingScale() = %v, want %v", got, tt.want)
			}
		})
	}
}
