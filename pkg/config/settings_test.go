package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

func settingsByKey(t *testing.T) map[string]SettingField {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Title = "Example"
	fields := make(map[string]SettingField)
	for _, f := range Settings(cfg) {
		if _, dup := fields[f.Key]; dup {
			t.Fatalf("duplicate setting key %q", f.Key)
		}
		fields[f.Key] = f
	}
	return fields
}

func TestSettings_DescribesConfig(t *testing.T) {
	fields := settingsByKey(t)

	title, ok := fields["title"]
	if !ok || title.Kind != SettingString || !title.Editable || title.Value != "Example" {
		t.Fatalf("title = %+v", title)
	}
	if title.Section != "" {
		t.Fatalf("top-level title should have no section: %+v", title)
	}
	if fields["theme.palette"].Section != "theme" {
		t.Fatalf("theme.palette section = %q", fields["theme.palette"].Section)
	}

	palette, ok := fields["theme.palette"]
	if !ok || !palette.Editable || len(palette.Options) == 0 {
		t.Fatalf("theme.palette = %+v", palette)
	}

	fontpack := fields["theme.fontpack"]
	if !containsString(fontpack.Options, "typewriter") {
		t.Fatalf("theme.fontpack options missing typewriter: %v", fontpack.Options)
	}

	slugMode := fields["glob.slug_mode"]
	if !containsString(slugMode.Options, "flat") || !containsString(slugMode.Options, "path") {
		t.Fatalf("glob.slug_mode options = %v", slugMode.Options)
	}

	nav, ok := fields["nav"]
	if !ok || nav.Kind != SettingComplex || nav.Editable {
		t.Fatalf("nav = %+v", nav)
	}

	webmention, ok := fields["webmentions.webmention_io_token"]
	if ok && (!webmention.Sensitive || webmention.Editable || webmention.Value != nil) {
		t.Fatalf("token field should be hidden: %+v", webmention)
	}
	for key, f := range fields {
		if f.Sensitive && (f.Editable || f.Value != nil) {
			t.Errorf("sensitive field %s exposes a value or is editable", key)
		}
		if f.Unsupported && f.Editable {
			t.Errorf("unsupported field %s is editable", key)
		}
	}
}

func TestSettings_EditableFieldsRoundTrip(t *testing.T) {
	for key, f := range settingsByKey(t) {
		if f.Editable && !settingParses(f) {
			t.Errorf("editable setting %s is not read back by the config loader", key)
		}
	}
}

func TestLookupSetting(t *testing.T) {
	if f, ok := LookupSetting("theme.palette"); !ok || f.Key != "theme.palette" {
		t.Fatalf("LookupSetting(theme.palette) = %+v, %v", f, ok)
	}
	if got := len(mustLookup(t, "theme.palette").SettingPath()); got != 2 {
		t.Fatalf("SettingPath length = %d", got)
	}
	for _, key := range []string{"nav", "does.not.exist", ""} {
		if _, ok := LookupSetting(key); ok {
			t.Errorf("LookupSetting(%q) should fail", key)
		}
	}
}

func TestSettingValue(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Title = "Hello"
	if v, ok := SettingValue(cfg, "title"); !ok || v != "Hello" {
		t.Fatalf("SettingValue(title) = %v, %v", v, ok)
	}
	if _, ok := SettingValue(cfg, "missing.key"); ok {
		t.Fatal("SettingValue(missing.key) should fail")
	}
}

func TestCoerceSettingValue(t *testing.T) {
	str := SettingField{Key: "s", Kind: SettingString}
	boolean := SettingField{Key: "b", Kind: SettingBool}
	integer := SettingField{Key: "i", Kind: SettingInt, intBits: 8}
	float := SettingField{Key: "f", Kind: SettingFloat}
	list := SettingField{Key: "l", Kind: SettingList}
	complexField := SettingField{Key: "c", Kind: SettingComplex}

	tests := []struct {
		name    string
		field   SettingField
		raw     any
		want    any
		wantErr bool
	}{
		{"string", str, "hi", "hi", false},
		{"string wrong type", str, 1.0, nil, true},
		{"string control char", str, "a\x00b", nil, true},
		{"string newline ok", str, "a\nb", "a\nb", false},
		{"bool", boolean, true, true, false},
		{"bool wrong type", boolean, "true", nil, true},
		{"int", integer, 12.0, int64(12), false},
		{"int fraction", integer, 1.5, nil, true},
		{"int out of range", integer, 300.0, nil, true},
		{"float", float, 1.25, 1.25, false},
		{"float wrong type", float, "1", nil, true},
		{"list", list, []any{"a", "b"}, []string{"a", "b"}, false},
		{"list non-string", list, []any{"a", 1.0}, nil, true},
		{"list wrong type", list, "a", nil, true},
		{"complex", complexField, "x", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CoerceSettingValue(tt.field, tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !SettingsEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}

	if _, err := CoerceSettingValue(complexField, "x"); !errors.Is(err, ErrUnknownSetting) {
		t.Fatalf("complex error = %v, want ErrUnknownSetting", err)
	}
}

func mustLookup(t *testing.T, key string) SettingField {
	t.Helper()
	f, ok := LookupSetting(key)
	if !ok {
		t.Fatalf("setting %s not found", key)
	}
	return f
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func TestLoadWithMergeOptions_Overlay(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/markata-go.toml"
	if err := os.WriteFile(path, []byte("[markata-go]\ntitle = \"Disk\"\n\n[markata-go.theme]\npalette = \"nord-dark\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	overlay := SettingsOverlay([]BakeSetting{
		{Path: []string{"title"}, Value: "Preview"},
		{Path: []string{"theme", "fontpack"}, Value: "typewriter"},
		{Path: []string{"glob", "patterns"}, Value: []string{"notes/*.md"}},
	})
	cfg, err := LoadWithMergeOptions(LoadOptions{DisableDotEnv: true, DisableEnvOverrides: true, Overlay: overlay}, path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Title != "Preview" || cfg.Theme.Palette != "nord-dark" || cfg.Theme.Fontpack != "typewriter" {
		t.Fatalf("overlay not applied: title=%q palette=%q fontpack=%q", cfg.Title, cfg.Theme.Palette, cfg.Theme.Fontpack)
	}
	if got := cfg.GlobConfig.Patterns; len(got) != 1 || got[0] != "notes/*.md" {
		t.Fatalf("patterns = %v", got)
	}
	if SettingsOverlay(nil) != nil {
		t.Fatal("empty overlay should be nil")
	}
}

func TestLoadWithMergeOptions_Remove(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/markata-go.toml"
	if err := os.WriteFile(path, []byte("[markata-go]\ntitle = \"Disk\"\nconcurrency = 3\n\n[markata-go.theme]\npalette = \"nord-dark\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	defaults, err := LoadWithMergeOptions(LoadOptions{DisableDotEnv: true, DisableEnvOverrides: true}, dir+"/missing.toml")
	if err != nil {
		defaults = nil
	}
	cfg, err := LoadWithMergeOptions(LoadOptions{
		DisableDotEnv:       true,
		DisableEnvOverrides: true,
		Remove:              [][]string{{"theme", "palette"}, {"concurrency"}, {"not", "there"}},
	}, path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Title != "Disk" {
		t.Fatalf("title = %q, want untouched", cfg.Title)
	}
	if cfg.Theme.Palette == "nord-dark" || cfg.Concurrency == 3 {
		t.Fatalf("remove not applied: palette=%q concurrency=%d", cfg.Theme.Palette, cfg.Concurrency)
	}
	if defaults != nil && cfg.Theme.Palette != defaults.Theme.Palette {
		t.Fatalf("palette = %q, want default %q", cfg.Theme.Palette, defaults.Theme.Palette)
	}
}

// quotedListPattern finds doc text listing two or more quoted values, such
// as `"left", "right"` or `"pagefind" (default) or "bleve"`.
var quotedListPattern = regexp.MustCompile(`"[a-z][a-z0-9_-]*"(?: \(default\))?(?:,| or) "[a-z][a-z0-9_-]*"`)

// freeFormSettings list values only as examples; any value is valid.
var freeFormSettings = map[string]bool{
	"components.share.position": true,
}

func TestSettings_DiscreteValuesHaveOptions(t *testing.T) {
	for key, f := range settingsByKey(t) {
		if !f.Editable || f.Kind != SettingString || freeFormSettings[key] {
			continue
		}
		doc := f.Doc
		if i := strings.Index(strings.ToLower(doc), "example"); i >= 0 {
			doc = doc[:i]
		}
		if quotedListPattern.MatchString(doc) && !f.Closed {
			t.Errorf("%s documents discrete values but has no closed options: %s", key, f.Doc)
		}
		if f.Closed && len(f.Options) < 2 {
			t.Errorf("%s is closed with options %v", key, f.Options)
		}
	}
}

func TestCoerceSettingValue_EnforcesOptionsAndRanges(t *testing.T) {
	cfg := DefaultConfig()
	tests := []struct {
		key     string
		raw     any
		wantErr bool
	}{
		{"glob.slug_mode", "path", false},
		{"glob.slug_mode", "bogus", true},
		{"layout.name", "docs", false},
		{"layout.name", "wiki", true},
		{"layout.name", "", false},
		{"components.nav.position", "sidebar", false},
		{"components.nav.position", "top", true},
		{"feed_defaults.pagination_type", "htmx-infinite", false},
		{"feed_defaults.pagination_type", "scroll", true},
		{"markdown.highlight.theme", "monokai", false},
		{"markdown.highlight.theme", "not-a-style", true},
		{"theme.palette", "nord-dark", false},
		{"theme.palette", "not-a-palette", true},
		{"theme.fontpack", "typewriter", false},
		{"theme.fontpack", "not-a-pack", true},
		{"theme.aesthetic", "not-an-aesthetic", true},
		{"theme.motif.color_mix", 0.5, false},
		{"theme.motif.color_mix", 1.5, true},
		{"theme.texture.scale", 4.0, true},
		{"title", "anything goes", false},
	}
	for _, tt := range tests {
		t.Run(tt.key+"="+fmt.Sprint(tt.raw), func(t *testing.T) {
			field, ok := LookupSettingIn(cfg, tt.key)
			if !ok {
				t.Fatalf("setting %s not editable", tt.key)
			}
			_, err := CoerceSettingValue(field, tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	custom := DefaultConfig()
	custom.FontpacksFile = "fonts.toml"
	field, _ := LookupSettingIn(custom, "theme.fontpack")
	if _, err := CoerceSettingValue(field, "house-pack"); err != nil {
		t.Fatalf("custom catalog should accept any fontpack: %v", err)
	}
}
