package plugins

import (
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestAestheticCSSPlugin_Configure(t *testing.T) {
	plugin := NewAestheticCSSPlugin()
	m := lifecycle.NewManager()

	// Create config
	cfg := lifecycle.NewConfig()

	// Setup extra
	b := true
	themeCfg := models.NewThemeConfig()
	themeCfg.Aesthetic = "brutal"
	themeCfg.Switcher.Enabled = &b

	cfg.Extra = map[string]interface{}{
		"theme": themeCfg,
	}
	m.SetConfig(cfg)

	err := plugin.Configure(m)
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}

	hash := m.GetAssetHash("css/aesthetic.css")
	if hash == "" {
		t.Error("Expected hash to be set for css/aesthetic.css")
	}
}
