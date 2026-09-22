package palettes

import (
	"strings"
	"testing"
)

// derivedReadabilityChecks mirrors the default-theme readability checks so a
// derived counterpart is held to the same bar as a hand-authored palette.
var derivedReadabilityChecks = []struct {
	fg, bg   string
	minRatio float64
}{
	{"text-primary", "bg-primary", 4.5},
	{"text-primary", "bg-surface", 4.5},
	{"text-primary", "bg-elevated", 4.5},
	{"text-secondary", "bg-primary", 4.5},
	{"text-secondary", "bg-surface", 4.5},
	{"text-muted", "bg-primary", 4.5},
	{"text-muted", "bg-secondary", 4.5},
	{"text-muted", "bg-surface", 4.5},
	{"text-muted", "bg-elevated", 4.5},
	{"link", "bg-primary", 4.5},
	{"link", "bg-surface", 4.5},
}

func TestDeriveCounterpart_EveryBuiltinIsReadable(t *testing.T) {
	t.Parallel()
	for _, name := range BuiltinNames() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			base, err := LoadBuiltin(name)
			if err != nil {
				t.Fatal(err)
			}
			derived := DeriveCounterpart(base)
			if derived.Variant == base.Variant {
				t.Fatalf("derived variant %q should differ from base %q", derived.Variant, base.Variant)
			}
			if errs := derived.Validate(); len(errs) > 0 {
				t.Fatalf("derived palette invalid: %v", errs)
			}
			bg, err := ParseHexColor(derived.Resolve("bg-primary"))
			if err != nil {
				t.Fatalf("derived background is not a color: %v", err)
			}
			if l := bg.ToHSL().L; derived.Variant == VariantLight && l < 0.85 {
				t.Errorf("light background too dark: %s (L=%.2f)", bg.Hex(), l)
			} else if derived.Variant == VariantDark && l > 0.25 {
				t.Errorf("dark background too light: %s (L=%.2f)", bg.Hex(), l)
			}
			for _, check := range derivedReadabilityChecks {
				fgHex, bgHex := derived.Resolve(check.fg), derived.Resolve(check.bg)
				if fgHex == "" || bgHex == "" {
					continue
				}
				ratio, err := ContrastRatioFromHex(fgHex, bgHex)
				if err != nil {
					t.Fatal(err)
				}
				if ratio < check.minRatio {
					t.Errorf("%s on %s: %.2f < %.1f (%s on %s)", check.fg, check.bg, ratio, check.minRatio, fgHex, bgHex)
				}
			}
		})
	}
}

func TestLoader_DerivesMissingCounterpart(t *testing.T) {
	loader := NewLoaderWithPaths(nil)
	p, err := loader.Load("dracula-light")
	if err != nil {
		t.Fatalf("expected derived dracula-light: %v", err)
	}
	if p.Variant != VariantLight {
		t.Errorf("variant = %q, want light", p.Variant)
	}
	if !strings.EqualFold(normalizeFileName(p.Name), "dracula-light") {
		t.Errorf("name = %q, want Dracula Light", p.Name)
	}
	// Explicit palettes are never shadowed by derivation.
	if p, err := loader.Load("everforest-light"); err != nil || p.Author == "" && strings.Contains(p.Description, "derived") {
		t.Errorf("everforest-light should load the real palette, got %+v err=%v", p, err)
	}
	// A suffix that matches the base variant is not derived.
	if _, err := loader.Load("dracula-dark"); err == nil {
		t.Error("dracula-dark should not exist (dracula is already dark)")
	}
}

func TestDetectVariants_AlwaysReturnsBothModes(t *testing.T) {
	loader := NewLoaderWithPaths(nil)
	for _, name := range BuiltinNames() {
		v := detectVariantsWithLoader(name, loader)
		if v.Light == "" || v.Dark == "" {
			t.Errorf("%s: light=%q dark=%q", name, v.Light, v.Dark)
			continue
		}
		if v.Light == v.Dark {
			t.Errorf("%s: light and dark resolve to the same palette %q", name, v.Light)
		}
		for _, n := range []string{v.Light, v.Dark} {
			if _, err := loader.Load(n); err != nil {
				t.Errorf("%s: variant %q not loadable: %v", name, n, err)
			}
		}
	}
	v := detectVariantsWithLoader("dracula", loader)
	if v.Light != "dracula-light" || v.Dark != "dracula" {
		t.Errorf("dracula variants = %+v", v)
	}
	v = detectVariantsWithLoader("catppuccin-mocha", loader)
	if v.Light != "catppuccin-latte" {
		t.Errorf("known family mapping should win, got %+v", v)
	}
}

func TestDiscover_IncludesDerivedCounterparts(t *testing.T) {
	loader := NewLoaderWithPaths(nil)
	infos, err := loader.Discover()
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]PaletteInfo{}
	for _, info := range infos {
		byName[normalizeFileName(info.Name)] = info
	}
	if info, ok := byName["dracula-light"]; !ok || !info.Derived {
		t.Errorf("expected derived dracula-light in discovery, got %+v", info)
	}
	if _, ok := byName["catppuccin-mocha-light"]; ok {
		t.Error("catppuccin-mocha already has catppuccin-latte; no derived counterpart expected")
	}
	if info := byName["everforest-light"]; info.Derived {
		t.Error("explicit everforest-light must not be flagged derived")
	}
}
