package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testIconSVG = `<svg width="24" height="24" viewBox="0 0 24 24"><path d="M1 1h22v22H1z"/></svg>`

func TestIconsPluginLoadsCanonicalAndSlashSyntax(t *testing.T) {
	root := t.TempDir()
	iconPath := filepath.Join(root, "lucide", "smile.svg")
	if err := os.MkdirAll(filepath.Dir(iconPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(iconPath, []byte(testIconSVG), 0o600); err != nil {
		t.Fatal(err)
	}

	plugin := NewIconsPlugin()
	plugin.paths = []string{root}
	if err := plugin.loadIcons(); err != nil {
		t.Fatal(err)
	}

	got := plugin.processContent("canonical :lucide-smile: and slash :lucide/smile:")
	if count := strings.Count(got, `data-icon="lucide/smile"`); count != 2 {
		t.Fatalf("expected both syntaxes to resolve, got %d icons in %q", count, got)
	}
	if strings.Contains(got, `width="24"`) || strings.Contains(got, `height="24"`) {
		t.Fatalf("expected fixed SVG dimensions to be normalized: %q", got)
	}
	if !strings.Contains(got, `width="1em" height="1em"`) {
		t.Fatalf("expected em-sized inline SVG: %q", got)
	}
}

func TestIconsPluginLeavesUnknownShortcodesAndEmojiAlone(t *testing.T) {
	plugin := NewIconsPlugin()
	plugin.icons["lucide/smile"] = iconAsset{name: "lucide/smile", svg: testIconSVG}
	plugin.icons["lucide-smile"] = plugin.icons["lucide/smile"]

	got := plugin.processContent("Known :lucide-smile: unknown :other-icon: emoji :smile:")
	if !strings.Contains(got, ":other-icon:") || !strings.Contains(got, ":smile:") {
		t.Fatalf("unknown shortcode or emoji was modified: %q", got)
	}
	if !strings.Contains(got, `data-icon="lucide/smile"`) {
		t.Fatalf("known icon was not rendered: %q", got)
	}
}

func TestIconsPluginPreservesInlineAndFencedCode(t *testing.T) {
	plugin := NewIconsPlugin()
	asset := iconAsset{name: "lucide/smile", svg: testIconSVG}
	plugin.icons["lucide-smile"] = asset

	content := "outside :lucide-smile:\n\n`inline :lucide-smile:`\n\n```md\n:lucide-smile:\n```\n"
	got := plugin.processContent(content)
	if count := strings.Count(got, `data-icon="lucide/smile"`); count != 1 {
		t.Fatalf("expected only the prose shortcode to render, got %d in %q", count, got)
	}
	if !strings.Contains(got, "`inline :lucide-smile:`") {
		t.Fatalf("inline code changed: %q", got)
	}
	if !strings.Contains(got, "```md\n:lucide-smile:\n```") {
		t.Fatalf("fenced code changed: %q", got)
	}
}

func TestIconsPluginPackAllowlist(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"lucide/smile.svg", "simple-icons/github.svg"} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(testIconSVG), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	plugin := NewIconsPlugin()
	plugin.paths = []string{root}
	plugin.packs = map[string]struct{}{"lucide": {}}
	if err := plugin.loadIcons(); err != nil {
		t.Fatal(err)
	}

	got := plugin.processContent(":lucide-smile: :simple-icons-github:")
	if !strings.Contains(got, `data-icon="lucide/smile"`) {
		t.Fatalf("allowed pack did not render: %q", got)
	}
	if !strings.Contains(got, ":simple-icons-github:") {
		t.Fatalf("disallowed pack rendered: %q", got)
	}
}

func TestIconsPluginRejectsActiveSVGContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "unsafe", "script.svg")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`<svg viewBox="0 0 1 1"><script>alert(1)</script></svg>`), 0o600); err != nil {
		t.Fatal(err)
	}

	plugin := NewIconsPlugin()
	plugin.paths = []string{root}
	if err := plugin.loadIcons(); err != nil {
		t.Fatal(err)
	}
	if got := plugin.processContent(":unsafe-script:"); got != ":unsafe-script:" {
		t.Fatalf("unsafe SVG should not render, got %q", got)
	}
}

func TestIconsPluginDefaultLucideFallbackUsesVendoredTarget(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("MARKATA_GO_OFFLINE", "1")
	iconPath := filepath.Join("static", ".icons", "lucide", "smile.svg")
	if err := os.MkdirAll(filepath.Dir(iconPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(iconPath, []byte(testIconSVG), 0o600); err != nil {
		t.Fatal(err)
	}

	plugin := NewIconsPlugin()
	got := plugin.processContent(":lucide-smile: :lucide/smile:")
	if count := strings.Count(got, `data-icon="lucide/smile"`); count != 2 {
		t.Fatalf("expected zero-config Lucide fallback to resolve both aliases, got %d in %q", count, got)
	}
	if !plugin.vendorAttempted {
		t.Fatal("expected missing Lucide lookup to invoke the default vendor path")
	}
}

func TestIconsPluginAutoVendorCanBeDisabled(t *testing.T) {
	plugin := NewIconsPlugin()
	plugin.autoVendor = false
	if got := plugin.processContent(":lucide-smile:"); got != ":lucide-smile:" {
		t.Fatalf("disabled auto vendor changed shortcode: %q", got)
	}
	if plugin.vendorAttempted {
		t.Fatal("disabled auto vendor should not attempt vendoring")
	}
}

func TestIconsPluginDoesNotVendorCodeExamples(t *testing.T) {
	plugin := NewIconsPlugin()
	content := "`inline :lucide-smile:`\n\n```md\n:lucide-smile:\n```\n"
	if got := plugin.processContent(content); got != content {
		t.Fatalf("code examples changed: %q", got)
	}
	if plugin.vendorAttempted {
		t.Fatal("code examples should not trigger default vendoring")
	}
}

func TestDefaultLucideVendorPackIsPinned(t *testing.T) {
	pack := defaultLucideVendorPack()
	if pack.name != "lucide" || pack.packageName != "lucide-static" || pack.version != defaultLucideVendorVersion {
		t.Fatalf("unexpected default Lucide pack: %+v", pack)
	}
	if err := pack.validate(); err != nil {
		t.Fatalf("default Lucide pack is invalid: %v", err)
	}
	if !strings.Contains(pack.archiveURL(), "lucide-static-"+defaultLucideVendorVersion+".tgz") {
		t.Fatalf("default Lucide archive is not pinned: %q", pack.archiveURL())
	}
}

func TestMarkdownFenceMarker(t *testing.T) {
	tests := []struct {
		line string
		mark byte
		len  int
	}{
		{"```go\n", '`', 3},
		{"  ~~~~\n", '~', 4},
		{"    ```\n", 0, 0},
		{"text ```\n", 0, 0},
	}
	for _, tt := range tests {
		mark, length := markdownFenceMarker(tt.line)
		if mark != tt.mark || length != tt.len {
			t.Errorf("markdownFenceMarker(%q) = (%q, %d), want (%q, %d)", tt.line, mark, length, tt.mark, tt.len)
		}
	}
}
