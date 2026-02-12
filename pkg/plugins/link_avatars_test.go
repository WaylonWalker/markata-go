package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestLinkAvatarsPlugin_Name(t *testing.T) {
	p := NewLinkAvatarsPlugin()
	if got := p.Name(); got != "link_avatars" {
		t.Errorf("Name() = %q, want %q", got, "link_avatars")
	}
}

func TestLinkAvatarsPlugin_DefaultConfig(t *testing.T) {
	p := NewLinkAvatarsPlugin()
	cfg := p.Config()

	if cfg.Enabled != true {
		t.Errorf("default Enabled = %v, want true", cfg.Enabled)
	}
	if cfg.Selector != "a[href^='http']" {
		t.Errorf("default Selector = %q, want %q", cfg.Selector, "a[href^='http']")
	}
	if cfg.Service != "duckduckgo" {
		t.Errorf("default Service = %q, want %q", cfg.Service, "duckduckgo")
	}
	if cfg.Size != 16 {
		t.Errorf("default Size = %d, want %d", cfg.Size, 16)
	}
	if cfg.Position != "before" {
		t.Errorf("default Position = %q, want %q", cfg.Position, "before")
	}
	if len(cfg.IgnoreClasses) != 1 || cfg.IgnoreClasses[0] != "no-avatar" {
		t.Errorf("default IgnoreClasses = %v, want %v", cfg.IgnoreClasses, []string{"no-avatar"})
	}
}

func TestLinkAvatarsPlugin_Configure(t *testing.T) {
	tests := []struct {
		name     string
		extra    map[string]interface{}
		expected LinkAvatarsConfig
	}{
		{
			name:  "nil_extra",
			extra: nil,
			expected: LinkAvatarsConfig{
				Enabled:       true,
				Selector:      "a[href^='http']",
				Service:       "duckduckgo",
				IgnoreClasses: []string{"no-avatar"},
				Size:          16,
				Position:      "before",
			},
		},
		{
			name: "enabled_only",
			extra: map[string]interface{}{
				"link_avatars": map[string]interface{}{
					"enabled": true,
				},
			},
			expected: LinkAvatarsConfig{
				Enabled:       true,
				Selector:      "a[href^='http']",
				Service:       "duckduckgo",
				IgnoreClasses: []string{"no-avatar"},
				Size:          16,
				Position:      "before",
			},
		},
		{
			name: "custom_service",
			extra: map[string]interface{}{
				"link_avatars": map[string]interface{}{
					"enabled": true,
					"service": "google",
					"size":    14,
				},
			},
			expected: LinkAvatarsConfig{
				Enabled:       true,
				Selector:      "a[href^='http']",
				Service:       "google",
				IgnoreClasses: []string{"no-avatar"},
				Size:          14,
				Position:      "before",
			},
		},
		{
			name: "ignore_lists",
			extra: map[string]interface{}{
				"link_avatars": map[string]interface{}{
					"enabled":          true,
					"ignore_domains":   []interface{}{"example.com", "test.org"},
					"ignore_classes":   []interface{}{"no-avatar"},
					"ignore_selectors": []interface{}{"nav a", ".footer a"},
				},
			},
			expected: LinkAvatarsConfig{
				Enabled:         true,
				Selector:        "a[href^='http']",
				Service:         "duckduckgo",
				Size:            16,
				Position:        "before",
				IgnoreDomains:   []string{"example.com", "test.org"},
				IgnoreClasses:   []string{"no-avatar"},
				IgnoreSelectors: []string{"nav a", ".footer a"},
			},
		},
		{
			name: "position_after",
			extra: map[string]interface{}{
				"link_avatars": map[string]interface{}{
					"enabled":  true,
					"position": "after",
				},
			},
			expected: LinkAvatarsConfig{
				Enabled:       true,
				Selector:      "a[href^='http']",
				Service:       "duckduckgo",
				IgnoreClasses: []string{"no-avatar"},
				Size:          16,
				Position:      "after",
			},
		},
		{
			name: "custom_template",
			extra: map[string]interface{}{
				"link_avatars": map[string]interface{}{
					"enabled":  true,
					"service":  "custom",
					"template": "https://favicon.example.com/?url={origin}",
				},
			},
			expected: LinkAvatarsConfig{
				Enabled:       true,
				Selector:      "a[href^='http']",
				Service:       "custom",
				Template:      "https://favicon.example.com/?url={origin}",
				IgnoreClasses: []string{"no-avatar"},
				Size:          16,
				Position:      "before",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewLinkAvatarsPlugin()
			m := lifecycle.NewManager()
			cfg := m.Config()
			cfg.Extra = tt.extra

			err := p.Configure(m)
			if err != nil {
				t.Fatalf("Configure() error = %v", err)
			}

			got := p.Config()
			if got.Enabled != tt.expected.Enabled {
				t.Errorf("Enabled = %v, want %v", got.Enabled, tt.expected.Enabled)
			}
			if got.Selector != tt.expected.Selector {
				t.Errorf("Selector = %q, want %q", got.Selector, tt.expected.Selector)
			}
			if got.Service != tt.expected.Service {
				t.Errorf("Service = %q, want %q", got.Service, tt.expected.Service)
			}
			if got.Size != tt.expected.Size {
				t.Errorf("Size = %d, want %d", got.Size, tt.expected.Size)
			}
			if got.Position != tt.expected.Position {
				t.Errorf("Position = %q, want %q", got.Position, tt.expected.Position)
			}
			if got.Template != tt.expected.Template {
				t.Errorf("Template = %q, want %q", got.Template, tt.expected.Template)
			}
			if !stringSliceEqual(got.IgnoreDomains, tt.expected.IgnoreDomains) {
				t.Errorf("IgnoreDomains = %v, want %v", got.IgnoreDomains, tt.expected.IgnoreDomains)
			}
			if !stringSliceEqual(got.IgnoreClasses, tt.expected.IgnoreClasses) {
				t.Errorf("IgnoreClasses = %v, want %v", got.IgnoreClasses, tt.expected.IgnoreClasses)
			}
			if !stringSliceEqual(got.IgnoreSelectors, tt.expected.IgnoreSelectors) {
				t.Errorf("IgnoreSelectors = %v, want %v", got.IgnoreSelectors, tt.expected.IgnoreSelectors)
			}
		})
	}
}

func TestLinkAvatarsPlugin_WriteDisabled(t *testing.T) {
	p := NewLinkAvatarsPlugin()
	// Explicitly disable the plugin (default is now enabled)
	p.SetConfig(LinkAvatarsConfig{
		Enabled:  false,
		Selector: "a[href^='http']",
		Service:  "duckduckgo",
		Size:     16,
		Position: "before",
	})
	m := lifecycle.NewManager()

	tmpDir := t.TempDir()
	cfg := m.Config()
	cfg.OutputDir = tmpDir

	err := p.Write(m)
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}

	// Assets should not be created
	jsPath := filepath.Join(tmpDir, "assets", "markata", "link-avatars.js")
	if _, err := os.Stat(jsPath); !os.IsNotExist(err) {
		t.Errorf("link-avatars.js should not exist when disabled")
	}
}

func TestLinkAvatarsPlugin_WriteEnabled(t *testing.T) {
	p := NewLinkAvatarsPlugin()
	p.SetConfig(LinkAvatarsConfig{
		Enabled:  true,
		Selector: "a[href^='http']",
		Service:  "duckduckgo",
		Size:     16,
		Position: "before",
	})

	m := lifecycle.NewManager()
	tmpDir := t.TempDir()
	cfg := m.Config()
	cfg.OutputDir = tmpDir

	err := p.Write(m)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Check JavaScript file was created
	jsPath := filepath.Join(tmpDir, "assets", "markata", "link-avatars.js")
	jsContent, err := os.ReadFile(jsPath)
	if err != nil {
		t.Fatalf("Failed to read link-avatars.js: %v", err)
	}
	if len(jsContent) == 0 {
		t.Error("link-avatars.js is empty")
	}

	// Verify JS contains expected content
	jsStr := string(jsContent)
	if !strings.Contains(jsStr, "Link Avatars") {
		t.Error("JS should contain header comment")
	}
	if !strings.Contains(jsStr, "duckduckgo") {
		t.Error("JS should contain service name")
	}
	if !strings.Contains(jsStr, "getFaviconURL") {
		t.Error("JS should contain getFaviconURL function")
	}

	// Check CSS file was created
	cssPath := filepath.Join(tmpDir, "assets", "markata", "link-avatars.css")
	cssContent, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("Failed to read link-avatars.css: %v", err)
	}
	if len(cssContent) == 0 {
		t.Error("link-avatars.css is empty")
	}

	// Verify CSS contains expected content
	cssStr := string(cssContent)
	if !strings.Contains(cssStr, ".has-avatar") {
		t.Error("CSS should contain .has-avatar class")
	}
	if !strings.Contains(cssStr, "--favicon-url") {
		t.Error("CSS should contain --favicon-url variable")
	}
	if !strings.Contains(cssStr, "16px") {
		t.Error("CSS should contain icon size")
	}
}

func TestLinkAvatarsPlugin_WriteCustomSize(t *testing.T) {
	p := NewLinkAvatarsPlugin()
	p.SetConfig(LinkAvatarsConfig{
		Enabled:  true,
		Selector: "a[href^='http']",
		Service:  "google",
		Size:     24,
		Position: "after",
	})

	m := lifecycle.NewManager()
	tmpDir := t.TempDir()
	cfg := m.Config()
	cfg.OutputDir = tmpDir

	err := p.Write(m)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Check CSS has custom size
	cssPath := filepath.Join(tmpDir, "assets", "markata", "link-avatars.css")
	cssContent, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("Failed to read link-avatars.css: %v", err)
	}

	cssStr := string(cssContent)
	if !strings.Contains(cssStr, "24px") {
		t.Errorf("CSS should contain custom icon size 24px, got: %s", cssStr)
	}
}

func TestLinkAvatarsPlugin_GenerateJavaScript(t *testing.T) {
	p := NewLinkAvatarsPlugin()
	p.SetConfig(LinkAvatarsConfig{
		Enabled:         true,
		Selector:        "article a[href^='http']",
		Service:         "custom",
		Template:        "https://my-service.com/favicon?url={host}",
		IgnoreDomains:   []string{"example.com"},
		IgnoreClasses:   []string{"no-avatar"},
		IgnoreSelectors: []string{"nav a"},
		Size:            20,
		Position:        "after",
	})

	js := p.generateJavaScript()

	// Check that config is embedded
	if !strings.Contains(js, `"selector":"article a[href^='http']"`) {
		t.Error("JS should contain custom selector")
	}
	if !strings.Contains(js, `"service":"custom"`) {
		t.Error("JS should contain custom service")
	}
	if !strings.Contains(js, `"template":"https://my-service.com/favicon?url={host}"`) {
		t.Error("JS should contain custom template")
	}
	if !strings.Contains(js, `"example.com"`) {
		t.Error("JS should contain ignore domain")
	}
	if !strings.Contains(js, `"no-avatar"`) {
		t.Error("JS should contain ignore class")
	}
	if !strings.Contains(js, `"nav a"`) {
		t.Error("JS should contain ignore selector")
	}
}

func TestLinkAvatarsPlugin_GenerateCSS(t *testing.T) {
	p := NewLinkAvatarsPlugin()
	p.SetConfig(LinkAvatarsConfig{
		Enabled:  true,
		Size:     18,
		Position: "before",
	})

	css := p.generateCSS()

	// Check CSS structure
	if !strings.Contains(css, "a.has-avatar {") {
		t.Error("CSS should contain base .has-avatar rule")
	}
	if !strings.Contains(css, "::before") {
		t.Error("CSS should contain ::before pseudo-element")
	}
	if !strings.Contains(css, "::after") {
		t.Error("CSS should contain ::after pseudo-element")
	}
	if !strings.Contains(css, "18px") {
		t.Error("CSS should contain custom size")
	}
	if !strings.Contains(css, "background-image: var(--favicon-url)") {
		t.Error("CSS should use --favicon-url variable")
	}
}

func TestLinkAvatarsPlugin_HeadInjection(t *testing.T) {
	p := NewLinkAvatarsPlugin()

	m := lifecycle.NewManager()
	tmpDir := t.TempDir()
	cfg := m.Config()
	cfg.OutputDir = tmpDir
	cfg.Extra = make(map[string]interface{})

	// Create a models.Config and store it in Extra (as done by createManager)
	modelsConfig := &models.Config{}
	cfg.Extra["models_config"] = modelsConfig

	// Set link_avatars config in Extra (as done by config loader)
	cfg.Extra["link_avatars"] = map[string]interface{}{
		"enabled": true,
	}

	// Run Configure (which now handles head injection)
	err := p.Configure(m)
	if err != nil {
		t.Fatalf("Configure() error = %v", err)
	}

	// Check that head tags were injected into models_config
	if len(modelsConfig.Head.Link) != 1 {
		t.Errorf("Expected 1 link tag, got %d", len(modelsConfig.Head.Link))
	}
	if len(modelsConfig.Head.Script) != 1 {
		t.Errorf("Expected 1 script tag, got %d", len(modelsConfig.Head.Script))
	}

	// Check that the head was also updated in Extra (for ToModelsConfig)
	updatedHead, ok := cfg.Extra["head"].(models.HeadConfig)
	if !ok {
		t.Fatal("head should be set in Extra")
	}
	if len(updatedHead.Link) != 1 {
		t.Errorf("Expected 1 link tag in Extra[head], got %d", len(updatedHead.Link))
	}
	if len(updatedHead.Script) != 1 {
		t.Errorf("Expected 1 script tag in Extra[head], got %d", len(updatedHead.Script))
	}

	// Verify the correct paths
	if updatedHead.Link[0].Href != "/assets/markata/link-avatars.css" {
		t.Errorf("CSS href = %q, want %q", updatedHead.Link[0].Href, "/assets/markata/link-avatars.css")
	}
	if updatedHead.Script[0].Src != "/assets/markata/link-avatars.js" {
		t.Errorf("JS src = %q, want %q", updatedHead.Script[0].Src, "/assets/markata/link-avatars.js")
	}
}

// stringSliceEqual checks if two string slices are equal.
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
