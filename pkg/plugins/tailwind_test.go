package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

type stubTailwindInstaller struct {
	path string
	err  error
}

func (s stubTailwindInstaller) Install() (string, error) {
	return s.path, s.err
}

func TestTailwindPlugin_DefaultContentArgs(t *testing.T) {
	plugin := NewTailwindPlugin()
	config := &lifecycle.Config{
		ContentDir:   "/repo",
		OutputDir:    "/repo/output",
		GlobPatterns: []string{"pages/**/*.md", "posts/**/*.md"},
	}

	args := plugin.defaultContentArgs(config, true)
	if len(args) != 2 || args[0] != "--content" {
		t.Fatalf("defaultContentArgs() = %v, want [--content patterns]", args)
	}

	patterns := strings.Split(args[1], ",")
	want := []string{"/repo/pages/**/*.md", "/repo/posts/**/*.md", "/repo/output/**/*.html"}
	for _, needle := range want {
		found := false
		for _, pattern := range patterns {
			if pattern == needle {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("patterns %v missing %q", patterns, needle)
		}
	}
}

func TestTailwindPlugin_ResolveBuildConfigFile_GeneratesContentConfig(t *testing.T) {
	plugin := NewTailwindPlugin()
	plugin.config = models.NewTailwindConfig()

	config := &lifecycle.Config{
		ContentDir:   "/repo",
		OutputDir:    "/repo/output",
		GlobPatterns: []string{"pages/**/*.md", "posts/**/*.md"},
	}

	configPath, cleanup, err := plugin.resolveBuildConfigFile(config, true)
	if err != nil {
		t.Fatalf("resolveBuildConfigFile() error = %v", err)
	}
	defer cleanup()

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", configPath, err)
	}
	text := string(data)
	for _, needle := range []string{"module.exports", "/repo/pages/**/*.md", "/repo/posts/**/*.md", "/repo/output/**/*.html"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("generated config missing %q:\n%s", needle, text)
		}
	}
}

func TestTailwindPlugin_IncludeAssetPath_RejectsAbsoluteOutput(t *testing.T) {
	plugin := NewTailwindPlugin()
	config := &lifecycle.Config{Extra: map[string]interface{}{"assets_dir": "/repo/static"}}

	got := plugin.includeAssetPath(config, "/tmp/markata-tailwind.css")
	if got != "" {
		t.Fatalf("includeAssetPath() = %q, want empty string for absolute output outside assets dir", got)
	}
}

func TestTailwindPlugin_ResolveBuildInput_GeneratesDefaultInput(t *testing.T) {
	plugin := NewTailwindPlugin()
	plugin.config = models.NewTailwindConfig()

	tmpDir := t.TempDir()
	config := &lifecycle.Config{Extra: map[string]interface{}{"assets_dir": filepath.Join(tmpDir, "static")}}

	inputPath, cleanup, err := plugin.resolveBuildInput(config)
	if err != nil {
		t.Fatalf("resolveBuildInput() error = %v", err)
	}
	defer cleanup()

	data, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", inputPath, err)
	}
	if string(data) != tailwindDefaultInputCSS {
		t.Fatalf("generated input = %q, want %q", string(data), tailwindDefaultInputCSS)
	}
}

func TestTailwindPlugin_FindOrInstallTailwind_PrefersManagedBinary(t *testing.T) {
	plugin := NewTailwindPlugin()
	plugin.config = models.NewTailwindConfig()
	plugin.config.Version = "v3.4.19"

	origLookPath := tailwindLookPath
	origInstaller := newTailwindInstaller
	t.Cleanup(func() {
		tailwindLookPath = origLookPath
		newTailwindInstaller = origInstaller
	})

	lookups := 0
	tailwindLookPath = func(_ string) (string, error) {
		lookups++
		return "/usr/bin/tailwindcss", nil
	}

	installed := false
	newTailwindInstaller = func(config TailwindInstallerConfig) tailwindInstaller {
		installed = true
		if config.Version != "v3.4.19" {
			t.Fatalf("installer version = %q, want v3.4.19", config.Version)
		}
		return stubTailwindInstaller{path: "/managed/tailwindcss"}
	}

	path, err := plugin.findOrInstallTailwind()
	if err != nil {
		t.Fatalf("findOrInstallTailwind() error = %v", err)
	}
	if path != "/managed/tailwindcss" {
		t.Fatalf("findOrInstallTailwind() = %q, want /managed/tailwindcss", path)
	}
	if !installed {
		t.Fatal("expected managed installer to be used")
	}
	if lookups != 0 {
		t.Fatalf("expected PATH lookup to be skipped when auto-install enabled, got %d lookups", lookups)
	}
}

func TestTailwindPlugin_FindOrInstallTailwind_UsesPathWhenAutoInstallDisabled(t *testing.T) {
	plugin := NewTailwindPlugin()
	plugin.config = models.NewTailwindConfig()
	autoInstall := false
	plugin.config.AutoInstall = &autoInstall

	origLookPath := tailwindLookPath
	origInstaller := newTailwindInstaller
	t.Cleanup(func() {
		tailwindLookPath = origLookPath
		newTailwindInstaller = origInstaller
	})

	tailwindLookPath = func(_ string) (string, error) {
		return "/usr/bin/tailwindcss", nil
	}
	newTailwindInstaller = func(_ TailwindInstallerConfig) tailwindInstaller {
		t.Fatal("installer should not be used when auto_install is false")
		return stubTailwindInstaller{}
	}

	path, err := plugin.findOrInstallTailwind()
	if err != nil {
		t.Fatalf("findOrInstallTailwind() error = %v", err)
	}
	if path != "/usr/bin/tailwindcss" {
		t.Fatalf("findOrInstallTailwind() = %q, want /usr/bin/tailwindcss", path)
	}
}
