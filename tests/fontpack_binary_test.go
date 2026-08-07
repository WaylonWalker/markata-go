package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstalledBinaryUsesEmbeddedFontpackOutsideCheckout(t *testing.T) {
	if testing.Short() {
		t.Skip("installed binary integration test")
	}
	_, thisFile, _, _ := runtime.Caller(0)
	repo := filepath.Dir(filepath.Dir(thisFile))
	binary := filepath.Join(t.TempDir(), "markata-go")
	build := exec.Command("go", "build", "-o", binary, "./cmd/markata-go")
	build.Dir = repo
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build standalone binary: %v\n%s", err, output)
	}
	site := t.TempDir()
	if err := os.MkdirAll(filepath.Join(site, "pages"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(site, "markata-go.toml"), []byte("[markata-go]\nfontpack = \"field-notebook\"\noutput_dir = \"output\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(site, "pages", "note.md"), []byte("---\ntitle: Embedded fonts\npublished: true\n---\n\nHello from an installed binary.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, "build", "--fast", "--config", filepath.Join(site, "markata-go.toml"))
	cmd.Dir = site // deliberately outside the Markata source checkout
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("standalone build: %v\n%s", err, output)
	}
	css, err := os.ReadFile(filepath.Join(site, "output", "css", "fonts.css"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(css), "@font-face") {
		t.Fatal("generated font CSS has no @font-face declaration")
	}
	entries, err := os.ReadDir(filepath.Join(site, "output", "assets", "fonts"))
	if err != nil {
		t.Fatal(err)
	}
	fontCount := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".woff2") {
			fontCount++
		}
	}
	if fontCount == 0 {
		t.Fatal("standalone build emitted no WOFF2 assets")
	}
}
