package servecontrol

import (
	"strings"
	"testing"
)

func TestBrowserTheme_CompleteFallbackRoles(t *testing.T) {
	theme := BrowserTheme(nil, false)
	roles := []string{
		"mode", "background", "panel", "surface", "elevated", "text-primary", "text-secondary",
		"accent", "link", "border", "focus", "success", "warning", "error", "info",
		"code-background", "code-text", "button-background", "button-text",
	}
	for _, role := range roles {
		if theme[role] == "" {
			t.Errorf("theme role %q has no fallback", role)
		}
	}
	if theme["mode"] != "light" || theme["background"] == "#09090b" || theme["text-primary"] == "#f4f4f5" {
		t.Errorf("light fallback is not light: %+v", theme)
	}
}

func TestBrowserTheme_MapsPaletteRolesAndButtonFallbacks(t *testing.T) {
	colors := map[string]string{
		"bg-primary": "#101010", "bg-secondary": "#202020", "bg-surface": "#303030",
		"bg-elevated": "#404040", "text-primary": "#f0f0f0", "text-muted": "#a0a0a0",
		"accent": "#ff8800", "link": "#00aaff", "border": "#505050", "border-focus": "#ffffff",
		"success": "#00ff00", "warning": "#ffaa00", "error": "#ff0000", "info": "#0000ff",
		"code-bg": "#151515", "code-text": "#eeeeee",
	}
	theme := BrowserTheme(func(role string) string { return colors[role] }, true)
	if theme["background"] != colors["bg-primary"] || theme["surface"] != colors["bg-surface"] {
		t.Fatalf("background roles were not mapped: %+v", theme)
	}
	if theme["text-primary"] != colors["text-primary"] || theme["text-secondary"] != colors["text-muted"] {
		t.Fatalf("text roles were not mapped: %+v", theme)
	}
	if theme["button-background"] != colors["accent"] || theme["button-text"] != colors["bg-primary"] {
		t.Fatalf("button fallbacks = %q/%q, want accent/background", theme["button-background"], theme["button-text"])
	}
	if theme["mode"] != "dark" {
		t.Fatalf("mode = %q, want dark", theme["mode"])
	}
}

func TestBrowserTheme_UsesPaletteButtonRoles(t *testing.T) {
	theme := BrowserTheme(func(role string) string {
		return map[string]string{"accent": "#abcdef", "button-primary-bg": "#123456", "button-primary-text": "#fedcba"}[role]
	}, true)
	if theme["button-background"] != "#123456" || theme["button-text"] != "#fedcba" {
		t.Fatalf("button roles = %q/%q", theme["button-background"], theme["button-text"])
	}
}

func TestBrowserTokenStylesheet_UsesModeFallbackAndRejectsCSSInjection(t *testing.T) {
	css := BrowserTokenStylesheet(map[string]string{
		"mode": "light", "background": "#fff;body{display:none}", "focus": "#123456",
	})
	if !strings.Contains(css, "color-scheme:light") || !strings.Contains(css, "--markata-background:#f7f7f5") {
		t.Fatalf("stylesheet did not use light fallback: %s", css)
	}
	if !strings.Contains(css, "--markata-focus:#123456") || !strings.Contains(css, "focus-visible") {
		t.Fatalf("stylesheet omitted shared focus role: %s", css)
	}
	if !strings.Contains(css, "scrollbar-color:var(--markata-border) transparent") ||
		!strings.Contains(css, "::-webkit-scrollbar-thumb") ||
		!strings.Contains(css, "background:var(--markata-text-secondary)") {
		t.Fatalf("stylesheet omitted themed scrollbar rules: %s", css)
	}
	if strings.Contains(css, "display:none") {
		t.Fatalf("untrusted CSS value escaped into stylesheet: %s", css)
	}
}
