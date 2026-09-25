package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleSettings_DryRunReturnsDiffWithoutWriting(t *testing.T) {
	site := setupThemeBakeSite(t, settingsTestFiles)
	before := readSiteFile(t, site, "config/theme.toml")
	rebuilds := 0
	serveRequestFullRebuild = func() { rebuilds++ }

	rec, resp := doSettings(t, http.MethodPost, `{"dry_run":true,"changes":[{"key":"theme.palette","value":"gruvbox-dark"}]}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("dry run status %d: %s", rec.Code, resp.Error)
	}
	if len(resp.Diffs) != 1 || resp.Diffs[0].Target != filepath.Join("config", "theme.toml") {
		t.Fatalf("diffs = %+v", resp.Diffs)
	}
	diff := resp.Diffs[0].Diff
	if !strings.Contains(diff, "-palette = \"nord-dark\"") || !strings.Contains(diff, "+palette = \"gruvbox-dark\"") {
		t.Fatalf("diff missing change:\n%s", diff)
	}
	if got := readSiteFile(t, site, "config/theme.toml"); got != before || rebuilds != 0 {
		t.Fatalf("dry run wrote or rebuilt (rebuilds=%d):\n%s", rebuilds, got)
	}
}

func TestHandleSettings_UnsetResetsToDefault(t *testing.T) {
	site := setupThemeBakeSite(t, settingsTestFiles)
	serveRequestFullRebuild = func() {}

	_, got := doSettings(t, http.MethodGet, "", nil)
	f := settingsField(t, got, "theme.palette")
	if f.Default == nil || f.Default == "nord-dark" {
		t.Fatalf("theme.palette default = %v", f.Default)
	}
	if got.Session == "" || got.SinglePage {
		t.Fatalf("session=%q single_page=%v", got.Session, got.SinglePage)
	}

	// Preview the reset without writing.
	rec, resp := doSettingsPreview(t, `{"changes":[{"key":"theme.palette","unset":true}]}`)
	if rec.Code != http.StatusOK || len(resp.Preview) != 1 || !resp.Preview[0].Unset {
		t.Fatalf("preview status %d preview %+v err %s", rec.Code, resp.Preview, resp.Error)
	}
	cfg, _, _, err := loadManagerConfig("")
	if err != nil || cfg.Theme.Palette == "nord-dark" {
		t.Fatalf("preview did not reset palette: %v %q", err, cfg.Theme.Palette)
	}

	// Values may not accompany unset, and unset keys must be defined somewhere.
	if rec, _ := doSettings(t, http.MethodPost, `{"changes":[{"key":"theme.palette","value":"x","unset":true}]}`, nil); rec.Code == http.StatusOK {
		t.Fatal("unset with a value should fail")
	}
	if rec, _ := doSettings(t, http.MethodPost, `{"changes":[{"key":"concurrency","unset":true}]}`, nil); rec.Code == http.StatusOK {
		t.Fatal("unset of an undefined key should fail")
	}

	rec, resp = doSettings(t, http.MethodPost, `{"changes":[{"key":"theme.palette","unset":true}]}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("bake status %d: %s", rec.Code, resp.Error)
	}
	if len(resp.Changed) != 1 || !resp.Changed[0].Unset || len(resp.Preview) != 0 {
		t.Fatalf("changed = %+v preview = %+v", resp.Changed, resp.Preview)
	}
	if theme := readSiteFile(t, site, "config/theme.toml"); strings.Contains(theme, "palette") {
		t.Fatalf("palette still in file:\n%s", theme)
	}
}

func TestHandleSettings_RefusesGlobalConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	global := filepath.Join(home, ".config", "markata-go", "config.toml")
	if err := os.MkdirAll(filepath.Dir(global), 0o755); err != nil {
		t.Fatal(err)
	}
	const content = "[markata-go]\ntitle = \"Global\"\n"
	if err := os.WriteFile(global, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	setupThemeBakeSite(t, map[string]string{"posts/a.md": "# a\n"})
	serveRequestFullRebuild = func() {}

	_, got := doSettings(t, http.MethodGet, "", nil)
	if f := settingsField(t, got, "title"); f.TargetKind != targetKindGlobal {
		t.Fatalf("title target kind = %q, want global", f.TargetKind)
	}
	rec, resp := doSettings(t, http.MethodPost, `{"changes":[{"key":"title","value":"New"}]}`, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d, want 409: %s", rec.Code, resp.Error)
	}
	data, err := os.ReadFile(global)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Fatalf("global config changed:\n%s", data)
	}
	rec, _ = doThemeBake(t, http.MethodPost, `{"palette":"gruvbox-dark"}`, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("theme bake status %d, want 409", rec.Code)
	}
}

func TestHandleThemeBake_DropsBakedKeysFromPreview(t *testing.T) {
	setupThemeBakeSite(t, settingsTestFiles)
	serveRequestFullRebuild = func() {}
	if rec, resp := doSettingsPreview(t, `{"changes":[{"key":"theme.palette","value":"gruvbox-dark"},{"key":"title","value":"P"}]}`); rec.Code != http.StatusOK {
		t.Fatalf("preview: %s", resp.Error)
	}
	rec, resp := doThemeBake(t, http.MethodPost, `{"palette":"catppuccin-mocha"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("bake status %d: %s", rec.Code, resp.Error)
	}
	preview := servePreviewChanges()
	if len(preview) != 1 || preview[0].Key != "title" {
		t.Fatalf("preview after theme bake = %+v", preview)
	}
}

func TestCreateManager_PreviewUsesSeparateCache(t *testing.T) {
	site := setupThemeBakeSite(t, map[string]string{"markata-go.toml": "[markata-go]\ntitle = \"Disk\"\n"})
	serveRequestFullRebuild = func() {}
	if rec, resp := doSettingsPreview(t, `{"changes":[{"key":"title","value":"P"}]}`); rec.Code != http.StatusOK {
		t.Fatalf("preview: %s", resp.Error)
	}
	m, err := createManager("")
	if err != nil {
		t.Fatal(err)
	}
	dir, ok := m.Config().Extra["cache_dir"].(string)
	if !ok {
		t.Fatal("cache_dir is not a string")
	}
	if filepath.Base(dir) != servePreviewCacheDir || !strings.HasPrefix(dir, site) && !strings.HasPrefix(dir, ".markata") {
		t.Fatalf("cache_dir = %q", dir)
	}
	setServePreview(nil)
	m, err = createManager("")
	if err != nil {
		t.Fatal(err)
	}
	if rawDir, exists := m.Config().Extra["cache_dir"]; exists {
		dir, ok = rawDir.(string)
		if !ok {
			t.Fatal("cache_dir is not a string")
		}
		if strings.Contains(dir, servePreviewCacheDir) {
			t.Fatalf("cache_dir without preview = %q", dir)
		}
	}
}

func TestUnifiedDiff(t *testing.T) {
	before := "a\nb\nc\nd\ne\nf\ng\n"
	after := "a\nb\nc\nD\ne\nf\ng\nh\n"
	got := unifiedDiff(before, after)
	for _, want := range []string{"-d", "+D", "+h", " c", "@@"} {
		if !strings.Contains(got, want) {
			t.Fatalf("diff missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, " a\n") {
		t.Fatalf("diff has too much context:\n%s", got)
	}
	if unifiedDiff("same\n", "same\n") != "" {
		t.Fatal("identical inputs should produce no diff")
	}
}

func TestRedactSensitiveLines(t *testing.T) {
	got := redactSensitiveLines("@@ -1,2 +1,2 @@\n api_token = \"abc123\"\n-title = \"Old\"\n+title = \"New\"")
	if strings.Contains(got, "abc123") || !strings.Contains(got, "+title = \"New\"") {
		t.Fatalf("redacted diff:\n%s", got)
	}
}
