package config

import "testing"

func TestFontpackConfigLoadsAcrossFormats(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		data   string
	}{
		{"toml", FormatTOML, "[markata-go]\nfontpack = \"field-notebook\"\n"},
		{"yaml", FormatYAML, "markata-go:\n  fontpack: field-notebook\n"},
		{"json", FormatJSON, `{"markata-go":{"fontpack":"field-notebook"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadFromString(tt.data, tt.format)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Fontpack != "field-notebook" {
				t.Fatalf("Fontpack = %q, want field-notebook", cfg.Fontpack)
			}
		})
	}
}
