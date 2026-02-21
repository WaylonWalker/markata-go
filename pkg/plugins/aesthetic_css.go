package plugins

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/aesthetic"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// AestheticCSSPlugin generates CSS variables from the configured aesthetic.
// It runs during the Write stage and creates/overwrites css/aesthetic.css
// with the aesthetic's CSS custom properties.
type AestheticCSSPlugin struct{}

// NewAestheticCSSPlugin creates a new AestheticCSSPlugin.
func NewAestheticCSSPlugin() *AestheticCSSPlugin {
	return &AestheticCSSPlugin{}
}

// Name returns the unique name of the plugin.
func (p *AestheticCSSPlugin) Name() string {
	return "aesthetic_css"
}

// Configure generates CSS from the configured aesthetic and registers its hash
func (p *AestheticCSSPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()

	aestheticName := p.getAestheticConfig(config.Extra)
	if aestheticName == "" {
		return nil
	}

	loader := aesthetic.NewLoader()
	switcherEnabled := p.isSwitcherEnabled(config.Extra)

	var css string
	if switcherEnabled {
		css = p.generateMultiAestheticCSS(loader, config.Extra, aestheticName)
	} else {
		css = p.generateSingleAestheticCSS(loader, aestheticName)
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(css)))[:8]

	m.SetAssetHash("css/aesthetic.css", hash)
	templates.SetAssetHashes(map[string]string{"css/aesthetic.css": hash})

	log.Printf("[aesthetic_css] Registered hash %s for aesthetic.css", hash)

	return nil
}

// Write generates CSS from the configured aesthetic and writes it to the output directory.
func (p *AestheticCSSPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	outputDir := config.OutputDir
	if config.Extra != nil {
		if fast, ok := config.Extra["fast_mode"].(bool); ok && fast {
			return nil
		}
	}

	aestheticName := p.getAestheticConfig(config.Extra)
	if aestheticName == "" {
		return nil
	}

	log.Printf("[aesthetic_css] Generating CSS for aesthetic: %s", aestheticName)

	switcherEnabled := p.isSwitcherEnabled(config.Extra)
	loader := aesthetic.NewLoader()

	var css string
	if switcherEnabled {
		css = p.generateMultiAestheticCSS(loader, config.Extra, aestheticName)
	} else {
		css = p.generateSingleAestheticCSS(loader, aestheticName)
	}

	cssDir := filepath.Join(outputDir, "css")
	cssPath := filepath.Join(cssDir, "aesthetic.css")
	if existing, err := os.ReadFile(cssPath); err == nil {
		if bytes.Equal(existing, []byte(css)) {
			return nil
		}
	}

	if err := os.MkdirAll(cssDir, 0o755); err != nil {
		return fmt.Errorf("creating css directory: %w", err)
	}
	if err := os.WriteFile(cssPath, []byte(css), 0o600); err != nil {
		return fmt.Errorf("writing aesthetic CSS: %w", err)
	}

	if hash := m.GetAssetHash("css/aesthetic.css"); hash != "" {
		base := strings.TrimSuffix(filepath.Base(cssPath), filepath.Ext(cssPath))
		hashedPath := filepath.Join(cssDir, fmt.Sprintf("%s.%s.css", base, hash))
		if err := os.WriteFile(hashedPath, []byte(css), 0o600); err != nil {
			return fmt.Errorf("writing hashed aesthetic CSS: %w", err)
		}
	}

	return nil
}

func (p *AestheticCSSPlugin) isSwitcherEnabled(extra map[string]interface{}) bool {
	if extra == nil {
		return false
	}
	if themeConfig, ok := extra["theme"].(models.ThemeConfig); ok {
		return themeConfig.Switcher.IsEnabled()
	}
	if theme, ok := extra["theme"].(map[string]interface{}); ok {
		if switcher, ok := theme["switcher"].(map[string]interface{}); ok {
			if enabled, ok := switcher["enabled"].(bool); ok {
				return enabled
			}
		}
	}
	return false
}

func (p *AestheticCSSPlugin) getSwitcherConfig(extra map[string]interface{}) models.ThemeSwitcherConfig {
	if extra == nil {
		return models.NewThemeSwitcherConfig()
	}
	if themeConfig, ok := extra["theme"].(models.ThemeConfig); ok {
		return themeConfig.Switcher
	}
	return models.NewThemeSwitcherConfig()
}

func (p *AestheticCSSPlugin) generateSingleAestheticCSS(loader *aesthetic.Loader, aestheticName string) string {
	a, err := loader.Load(aestheticName)
	if err != nil {
		return ""
	}

	var buf bytes.Buffer
	buf.WriteString("@layer reset, tokens, base, components, utilities, overrides;\n\n")
	buf.WriteString("@layer tokens {\n")
	buf.WriteString(fmt.Sprintf("/* Aesthetic: %s */\n", a.Name))

	// Just use the root level CSS block directly, but strip out the :root { ... } to format it ourselves
	// or we can just append a.GenerateCSS() directly. Wait, a.GenerateCSS() returns :root { ... }
	cssContent := a.GenerateCSS()
	buf.WriteString(cssContent)

	buf.WriteString("}\n")
	return buf.String()
}

type AestheticManifestEntry struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

func (p *AestheticCSSPlugin) generateMultiAestheticCSS(loader *aesthetic.Loader, extra map[string]interface{}, defaultAestheticName string) string {
	var buf bytes.Buffer

	buf.WriteString("@layer reset, tokens, base, components, utilities, overrides;\n\n")
	buf.WriteString("@layer tokens {\n")

	allAesthetics, err := loader.Discover()
	if err != nil {
		return p.generateSingleAestheticCSS(loader, defaultAestheticName)
	}

	// Filter based on switcher
	// ... we will filter just by name for now, using the palette include/exclude logic
	switcherConfig := p.getSwitcherConfig(extra)
	filteredAesthetics := p.filterAesthetics(allAesthetics, switcherConfig)

	manifest := make([]AestheticManifestEntry, 0, len(filteredAesthetics))
	for _, info := range filteredAesthetics {
		manifest = append(manifest, AestheticManifestEntry{
			Name:        info.Name,
			DisplayName: info.Name,
		})
	}

	sort.Slice(manifest, func(i, j int) bool {
		return manifest[i].Name < manifest[j].Name
	})

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		manifestJSON = []byte("[]")
	}
	escapedManifest := strings.ReplaceAll(string(manifestJSON), "'", "\\'")

	buf.WriteString("/* Global aesthetic configuration */\n")
	buf.WriteString(":root {\n")
	buf.WriteString(fmt.Sprintf("  --aesthetic-default: %q;\n", defaultAestheticName))
	buf.WriteString(fmt.Sprintf("  --aesthetic-manifest: '%s';\n", escapedManifest))
	buf.WriteString("}\n\n")

	for _, info := range filteredAesthetics {
		a, err := loader.Load(info.Name)
		if err != nil {
			continue
		}

		buf.WriteString(fmt.Sprintf("/* Aesthetic: %s */\n", info.Name))
		buf.WriteString(fmt.Sprintf("[data-aesthetic=%q] {\n", info.Name))

		// Extract variables directly to inject in [data-aesthetic] scope
		p.writeAestheticVariablesIndented(&buf, a, "  ")

		buf.WriteString("}\n\n")
	}

	if defaultAestheticName != "" {
		a, err := loader.Load(defaultAestheticName)
		if err == nil {
			buf.WriteString(fmt.Sprintf("/* Default aesthetic - %s */\n", defaultAestheticName))
			buf.WriteString(":root:not([data-aesthetic]) {\n")
			p.writeAestheticVariablesIndented(&buf, a, "  ")
			buf.WriteString("}\n\n")
		}
	}

	buf.WriteString("}\n")
	return buf.String()
}

func (p *AestheticCSSPlugin) writeAestheticVariablesIndented(buf *bytes.Buffer, a *aesthetic.Aesthetic, indent string) {
	cssLines := strings.Split(a.GenerateCSS(), "\n")
	for _, line := range cssLines {
		if strings.HasPrefix(line, "--") {
			buf.WriteString(indent + line + "\n")
		} else if strings.HasPrefix(strings.TrimSpace(line), "--") {
			buf.WriteString(indent + strings.TrimSpace(line) + "\n")
		}
	}
}

func (p *AestheticCSSPlugin) filterAesthetics(all []aesthetic.Info, switcherConfig models.ThemeSwitcherConfig) []aesthetic.Info {
	if switcherConfig.IsIncludeAll() {
		excludeSet := make(map[string]bool)
		for _, name := range switcherConfig.Exclude {
			excludeSet[strings.ToLower(name)] = true
		}

		var result []aesthetic.Info
		for _, info := range all {
			lowerName := strings.ToLower(info.Name)
			if !excludeSet[lowerName] {
				result = append(result, info)
			}
		}
		return result
	}

	includeSet := make(map[string]bool)
	for _, name := range switcherConfig.Include {
		includeSet[strings.ToLower(name)] = true
	}

	var result []aesthetic.Info
	for _, info := range all {
		lowerName := strings.ToLower(info.Name)
		if includeSet[lowerName] {
			result = append(result, info)
		}
	}
	return result
}

func (p *AestheticCSSPlugin) getAestheticConfig(extra map[string]interface{}) string {
	if extra == nil {
		return ""
	}

	if modelsConfig, ok := extra["models_config"].(*models.Config); ok {
		if modelsConfig.Theme.Aesthetic != "" {
			return modelsConfig.Theme.Aesthetic
		}
	}

	if themeConfig, ok := extra["theme"].(models.ThemeConfig); ok {
		return themeConfig.Aesthetic
	}

	theme, ok := extra["theme"].(map[string]interface{})
	if !ok {
		return ""
	}

	if asth, ok := theme["aesthetic"].(string); ok && asth != "" {
		return asth
	}

	return ""
}

func (p *AestheticCSSPlugin) Priority(_ lifecycle.Stage) int {
	return lifecycle.PriorityDefault
}

var (
	_ lifecycle.Plugin          = (*AestheticCSSPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*AestheticCSSPlugin)(nil)
	_ lifecycle.WritePlugin     = (*AestheticCSSPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*AestheticCSSPlugin)(nil)
)
