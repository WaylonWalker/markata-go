package servecontrol

import "strings"

const (
	browserThemeModeDark  = "dark"
	browserThemeModeLight = "light"
)

// BrowserTheme returns the shared semantic color roles used by browser control
// surfaces. resolve should look up a Markata palette semantic/component name.
// Missing roles use the defaults shared with Builder Admin.
func BrowserTheme(resolve func(string) string, dark bool) map[string]string {
	theme := map[string]string{
		"mode":              browserThemeModeDark,
		"background":        "#09090b",
		"panel":             "#18181b",
		"surface":           "#27272a",
		"elevated":          "#3f3f46",
		"text-primary":      "#f4f4f5",
		"text-secondary":    "#a1a1aa",
		"accent":            "#fafafa",
		"link":              "#fafafa",
		"border":            "#3f3f46",
		"focus":             "#93c5fd",
		"success":           "#22c55e",
		"warning":           "#f59e0b",
		"error":             "#ef4444",
		"info":              "#3b82f6",
		"code-background":   "#111113",
		"code-text":         "#f4f4f5",
		"button-background": "#fafafa",
		"button-text":       "#18181b",
	}
	if !dark {
		for role, color := range map[string]string{
			"background": "#f7f7f5", "panel": "#ffffff", "surface": "#f1f1ef", "elevated": "#e4e4e0",
			"text-primary": "#202124", "text-secondary": "#55565a", "accent": "#26272b", "link": "#155eef",
			"border": "#d1d3d7", "focus": "#155eef", "success": "#137a44", "warning": "#995c00",
			"error": "#b42318", "info": "#175cd3", "code-background": "#f1f2f3", "code-text": "#202124",
			"button-background": "#202124", "button-text": "#ffffff",
		} {
			theme[role] = color
		}
		theme["mode"] = browserThemeModeLight
	}
	if resolve == nil {
		return theme
	}

	roles := []struct {
		role    string
		palette string
	}{
		{"background", "bg-primary"},
		{"panel", "bg-secondary"},
		{"surface", "bg-surface"},
		{"elevated", "bg-elevated"},
		{"text-primary", "text-primary"},
		{"text-secondary", "text-muted"},
		{"accent", "accent"},
		{"link", "link"},
		{"border", "border"},
		{"focus", "border-focus"},
		{"success", "success"},
		{"warning", "warning"},
		{"error", "error"},
		{"info", "info"},
		{"code-background", "code-bg"},
		{"code-text", "code-text"},
		{"button-background", "button-primary-bg"},
		{"button-text", "button-primary-text"},
	}
	for _, item := range roles {
		if color := resolve(item.palette); color != "" {
			theme[item.role] = color
		}
	}
	if resolve("button-primary-bg") == "" {
		theme["button-background"] = theme["accent"]
	}
	if resolve("button-primary-text") == "" {
		theme["button-text"] = theme["background"]
	}
	return theme
}

// BrowserTokenStylesheet serializes the shared semantic browser theme as CSS
// custom properties. Palette values are hex colors; invalid values fall back
// to the built-in defaults before they reach a style element.
func BrowserTokenStylesheet(theme map[string]string) string {
	defaults := BrowserTheme(nil, theme["mode"] != browserThemeModeLight)
	roles := []string{
		"background", "panel", "surface", "elevated", "text-primary", "text-secondary",
		"accent", "link", "border", "focus", "success", "warning", "error", "info",
		"code-background", "code-text", "button-background", "button-text",
	}
	var css strings.Builder
	css.WriteString(":root{color-scheme:")
	mode := theme["mode"]
	if mode != browserThemeModeLight {
		mode = browserThemeModeDark
	}
	css.WriteString(mode)
	for _, role := range roles {
		color := theme[role]
		if !validBrowserHexColor(color) {
			color = defaults[role]
		}
		css.WriteString(";--markata-")
		css.WriteString(role)
		css.WriteByte(':')
		css.WriteString(color)
	}
	css.WriteString("} *{scrollbar-color:var(--markata-border) transparent;scrollbar-width:thin}")
	css.WriteString(" *::-webkit-scrollbar{width:10px;height:10px}")
	css.WriteString(" *::-webkit-scrollbar-track{background:transparent}")
	css.WriteString(" *::-webkit-scrollbar-thumb{background:var(--markata-border);border:2px solid transparent;border-radius:999px;background-clip:padding-box}")
	css.WriteString(" *::-webkit-scrollbar-thumb:hover{background:var(--markata-text-secondary);background-clip:padding-box}")
	css.WriteString(" :where(button,input,a,summary,[tabindex]):focus-visible{outline:2px solid var(--markata-focus);outline-offset:3px}")
	return css.String()
}

func validBrowserHexColor(value string) bool {
	if len(value) != 4 && len(value) != 7 {
		return false
	}
	if value[0] != '#' {
		return false
	}
	for _, char := range value[1:] {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}