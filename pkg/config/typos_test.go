package config

import (
	"strings"
	"testing"
)

func TestConfigKeyTypoSuggestsNestedKey(t *testing.T) {
	_, err := LoadFromString("[markata-go.theme]\npalete = 'ayu-dark'\n", FormatTOML)
	if err == nil || !strings.Contains(err.Error(), "theme.palette") {
		t.Fatalf("error = %v; want theme.palette suggestion", err)
	}
}

func TestConfigKeyTypoSuggestsTopLevelKey(t *testing.T) {
	_, err := LoadFromString("[markata-go]\noutput_dr = 'public'\n", FormatTOML)
	if err == nil || !strings.Contains(err.Error(), "output_dir") {
		t.Fatalf("error = %v; want output_dir suggestion", err)
	}
}

func TestConfigUnknownPluginSectionRemainsValid(t *testing.T) {
	for _, section := range []string{"stats", "webmentions"} {
		_, err := LoadFromString("[markata-go."+section+"]\ncount = 3\n", FormatTOML)
		if err != nil {
			t.Fatalf("custom plugin section %q: %v", section, err)
		}
	}
}
