package themes

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/palettes"
)

type themeContrastRule struct {
	selector string
	fgExpr   string
	bgExpr   string
	minRatio float64
	context  string
}

type themeColorEnv map[string]palettes.Color

var cssVarPattern = regexp.MustCompile(`--[a-zA-Z0-9_-]+`)

func TestDefaultThemeStaticContrastPairsAcrossBuiltinPalettes(t *testing.T) {
	t.Parallel()

	rules := []themeContrastRule{
		{selector: "a", fgExpr: "var(--color-link, var(--color-primary))", bgExpr: "var(--color-background)", minRatio: 4.5, context: "default links"},
		{selector: "a:hover", fgExpr: "var(--color-link-hover, var(--color-primary-dark))", bgExpr: "var(--color-background)", minRatio: 4.5, context: "hovered default links"},
		{selector: "kbd", fgExpr: "var(--color-text)", bgExpr: "var(--color-surface, var(--color-background))", minRatio: 4.5, context: "keyboard keys"},
		{selector: "kbd.key-ctrl", fgExpr: "var(--color-text)", bgExpr: "var(--color-kbd-modifier-bg, var(--color-background))", minRatio: 4.5, context: "keyboard modifier keys"},
		{selector: ".note", fgExpr: "var(--color-text)", bgExpr: "var(--color-note-bg, var(--color-background))", minRatio: 4.5, context: "note containers"},
		{selector: ".warning", fgExpr: "var(--color-text)", bgExpr: "var(--color-warning-bg, var(--color-background))", minRatio: 4.5, context: "warning containers"},
		{selector: ".info", fgExpr: "var(--color-text)", bgExpr: "var(--color-info-bg, var(--color-background))", minRatio: 4.5, context: "info containers"},
		{selector: ".success", fgExpr: "var(--color-text)", bgExpr: "var(--color-success-bg, var(--color-background))", minRatio: 4.5, context: "success containers"},
		{selector: ".error", fgExpr: "var(--color-text)", bgExpr: "var(--color-danger-bg, var(--color-background))", minRatio: 4.5, context: "error containers"},
		{selector: ".post-copy__summary", fgExpr: "var(--color-text)", bgExpr: "var(--color-background)", minRatio: 4.5, context: "post copy summary"},
		{selector: ".post-copy__summary:hover", fgExpr: "var(--color-text)", bgExpr: "var(--color-surface)", minRatio: 4.5, context: "hovered post copy summary"},
		{selector: ".feed-nav-counter", fgExpr: "var(--color-text)", bgExpr: "var(--color-background)", minRatio: 4.5, context: "feed nav counter"},
		{selector: ".webmention-count", fgExpr: "var(--color-text)", bgExpr: "var(--color-background)", minRatio: 4.5, context: "webmention count"},
	}

	for _, name := range palettes.BuiltinNames() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			palette, err := palettes.LoadBuiltin(name)
			if err != nil {
				t.Fatalf("load builtin palette %q: %v", name, err)
			}

			env := themeEnvForPalette(t, palette)
			for _, rule := range rules {
				fg, err := evalThemeColor(rule.fgExpr, env)
				if err != nil {
					t.Errorf("%s %s foreground %q: %v", name, rule.selector, rule.fgExpr, err)
					continue
				}

				bg, err := evalThemeColor(rule.bgExpr, env)
				if err != nil {
					t.Errorf("%s %s background %q: %v", name, rule.selector, rule.bgExpr, err)
					continue
				}

				ratio := palettes.ContrastRatio(fg, bg)
				if ratio < rule.minRatio {
					t.Errorf("%s %s contrast %.2f < %.1f for %s (%s on %s)",
						name, rule.selector, ratio, rule.minRatio, rule.context, fg.Hex(), bg.Hex())
				}
			}
		})
	}
}

func themeEnvForPalette(t *testing.T, palette *palettes.Palette) themeColorEnv {
	t.Helper()

	env := themeColorEnv{}
	setPaletteColor(t, env, "--color-primary", palette, "accent", 3.0)
	setPaletteColor(t, env, "--color-primary-light", palette, "accent-hover", 3.0)
	setPaletteColor(t, env, "--color-primary-dark", palette, "accent-hover", 3.0)
	setPaletteColor(t, env, "--color-text", palette, "text-primary", 4.5)
	setPaletteColor(t, env, "--color-text-secondary", palette, "text-secondary", 4.5)
	setPaletteColor(t, env, "--color-text-muted", palette, "text-muted", 4.5)
	setRawPaletteColor(t, env, "--color-background", palette, "bg-primary")
	setRawPaletteColor(t, env, "--color-surface", palette, "bg-surface")
	setRawPaletteColor(t, env, "--color-border", palette, "border")
	setPaletteColor(t, env, "--color-success", palette, "success", 3.0)
	setPaletteColor(t, env, "--color-warning", palette, "warning", 3.0)
	setPaletteColor(t, env, "--color-error", palette, "error", 3.0)
	setPaletteColor(t, env, "--color-info", palette, "info", 3.0)
	setPaletteColor(t, env, "--color-link", palette, "link", 4.5)
	setPaletteColor(t, env, "--color-link-hover", palette, "link-hover", 4.5)
	setPaletteColor(t, env, "--color-link-visited", palette, "link-visited", 4.5)
	if c, ok := env["--color-background"]; ok {
		env["--color-kbd-modifier-bg"] = c
		env["--color-note-bg"] = c
		env["--color-warning-bg"] = c
		env["--color-info-bg"] = c
		env["--color-success-bg"] = c
		env["--color-danger-bg"] = c
	}

	white, err := palettes.ParseHexColor("#ffffff")
	if err != nil {
		t.Fatalf("parse white: %v", err)
	}
	env["white"] = white

	transparent, err := palettes.ParseHexColor(palette.Resolve("bg-primary"))
	if err != nil {
		t.Fatalf("parse transparent fallback bg-primary: %v", err)
	}
	env["transparent"] = transparent

	return env
}

func setPaletteColor(t *testing.T, env themeColorEnv, cssName string, palette *palettes.Palette, key string, minRatio float64) {
	t.Helper()

	fgHex := palette.Resolve(key)
	if fgHex == "" {
		return
	}
	bgHex := palette.Resolve("bg-primary")
	if bgHex == "" {
		setHexColor(t, env, cssName, fgHex)
		return
	}

	fg, errFg := palettes.ParseHexColor(fgHex)
	bg, errBg := palettes.ParseHexColor(bgHex)
	if errFg != nil || errBg != nil {
		setHexColor(t, env, cssName, fgHex)
		return
	}

	adjusted, _ := fg.AdjustForContrast(bg, minRatio)
	env[cssName] = adjusted
}

func setRawPaletteColor(t *testing.T, env themeColorEnv, cssName string, palette *palettes.Palette, key string) {
	t.Helper()

	if value := palette.Resolve(key); value != "" {
		setHexColor(t, env, cssName, value)
	}
}

func setHexColor(t *testing.T, env themeColorEnv, cssName, hex string) {
	t.Helper()

	c, err := palettes.ParseHexColor(hex)
	if err != nil {
		t.Fatalf("parse %s=%s: %v", cssName, hex, err)
	}
	env[cssName] = c
}

func evalThemeColor(expr string, env themeColorEnv) (palettes.Color, error) {
	expr = strings.TrimSpace(expr)
	if strings.HasPrefix(expr, "#") {
		return palettes.ParseHexColor(expr)
	}
	if c, ok := env[expr]; ok {
		return c, nil
	}
	if strings.HasPrefix(expr, "var(") {
		return evalCSSVar(expr, env)
	}
	if strings.HasPrefix(expr, "color-mix(") {
		return evalColorMix(expr, env)
	}

	return palettes.Color{}, fmt.Errorf("unsupported color expression %q", expr)
}

func evalCSSVar(expr string, env themeColorEnv) (palettes.Color, error) {
	vars := cssVarPattern.FindAllString(expr, -1)
	for _, name := range vars {
		if c, ok := env[name]; ok {
			return c, nil
		}
	}
	return palettes.Color{}, fmt.Errorf("no known CSS variable in %q", expr)
}

func evalColorMix(expr string, env themeColorEnv) (palettes.Color, error) {
	inner := strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(expr), ")"), "color-mix(")
	parts := splitTopLevel(inner, ',')
	if len(parts) != 3 {
		return palettes.Color{}, fmt.Errorf("unsupported color-mix parts in %q", expr)
	}
	if !strings.EqualFold(strings.TrimSpace(parts[0]), "in srgb") {
		return palettes.Color{}, fmt.Errorf("unsupported color-mix space in %q", expr)
	}

	c1, p1, err := evalColorMixPart(parts[1], env)
	if err != nil {
		return palettes.Color{}, err
	}
	c2, p2, err := evalColorMixPart(parts[2], env)
	if err != nil {
		return palettes.Color{}, err
	}

	switch {
	case p1 == 0 && p2 == 0:
		p1 = 0.5
		p2 = 0.5
	case p1 == 0:
		p1 = 1 - p2
	case p2 == 0:
		p2 = 1 - p1
	}
	total := p1 + p2
	if total == 0 {
		return palettes.Color{}, fmt.Errorf("zero color-mix total in %q", expr)
	}
	p1 /= total
	p2 /= total

	return palettes.Color{
		R: uint8(float64(c1.R)*p1 + float64(c2.R)*p2 + 0.5),
		G: uint8(float64(c1.G)*p1 + float64(c2.G)*p2 + 0.5),
		B: uint8(float64(c1.B)*p1 + float64(c2.B)*p2 + 0.5),
	}, nil
}

func evalColorMixPart(part string, env themeColorEnv) (palettes.Color, float64, error) {
	fields := strings.Fields(strings.TrimSpace(part))
	if len(fields) == 0 {
		return palettes.Color{}, 0, fmt.Errorf("empty color-mix part")
	}

	colorExpr := fields[0]
	if strings.HasPrefix(colorExpr, "var(") && !strings.Contains(colorExpr, ")") {
		for i := 1; i < len(fields); i++ {
			colorExpr += " " + fields[i]
			if strings.Contains(fields[i], ")") {
				fields = append([]string{colorExpr}, fields[i+1:]...)
				break
			}
		}
	}

	colorValue, err := evalThemeColor(colorExpr, env)
	if err != nil {
		return palettes.Color{}, 0, err
	}

	if len(fields) < 2 || !strings.HasSuffix(fields[1], "%") {
		return colorValue, 0, nil
	}
	percent, err := strconv.ParseFloat(strings.TrimSuffix(fields[1], "%"), 64)
	if err != nil {
		return palettes.Color{}, 0, fmt.Errorf("parse color-mix percent %q: %w", fields[1], err)
	}
	return colorValue, percent / 100, nil
}

func splitTopLevel(s string, sep rune) []string {
	parts := []string{}
	start := 0
	depth := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case sep:
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}
