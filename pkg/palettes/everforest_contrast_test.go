package palettes

import "testing"

func TestEverforestDarkThemeContrastPairs(t *testing.T) {
	palette, err := LoadBuiltin("everforest-dark")
	if err != nil {
		t.Fatalf("failed to load everforest-dark: %v", err)
	}

	checks := []ContrastCheckSpec{
		{"text-primary", "bg-primary", 4.5, "AA"},
		{"text-primary", "bg-surface", 4.5, "AA"},
		{"text-secondary", "bg-primary", 4.5, "AA"},
		{"text-muted", "bg-primary", 4.5, "AA"},
		{"text-muted", "bg-secondary", 4.5, "AA"},
		{"text-muted", "bg-surface", 4.5, "AA"},
		{"link", "bg-primary", 4.5, "AA"},
		{"info", "bg-primary", 3.0, "UI"},
		{"success", "bg-primary", 3.0, "UI"},
		{"warning", "bg-primary", 3.0, "UI"},
		{"error", "bg-primary", 3.0, "UI"},
		{"code-text", "code-bg", 4.5, "AA"},
		{"code-comment", "code-bg", 3.0, "AA Large"},
		{"button-primary-text", "button-primary-bg", 4.5, "AA"},
		{"button-secondary-text", "button-secondary-bg", 4.5, "AA"},
	}

	for _, result := range palette.CheckContrastWith(checks) {
		if result.Passed {
			continue
		}
		t.Errorf("everforest-dark %s on %s failed: %.2f < %.1f", result.Foreground, result.Background, result.Ratio, result.Required)
	}
}
