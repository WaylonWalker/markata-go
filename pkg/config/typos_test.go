package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigKeyTypoSuggestsNestedKey(t *testing.T) {
	_, err := LoadFromString("[markata-go.theme]\npalete = 'ayu-dark'\n", FormatTOML)
	if err == nil || !strings.Contains(err.Error(), "theme.palette") {
		t.Fatalf("error = %v; want theme.palette suggestion", err)
	}
}

func TestGetValueFromFileReadsExistingKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "markata-go.toml")
	if err := os.WriteFile(path, []byte("[markata-go]\noutput_dir = 'public'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"output_dir", "markata-go.output_dir"} {
		value, err := GetValueFromFile(path, key)
		if err != nil || value != "public" {
			t.Errorf("GetValueFromFile(%q) = %v, %v; want public", key, value, err)
		}
	}
	_, err := GetValueFromFile(path, "missing_key")
	if !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("missing key error = %v; want ErrKeyNotFound", err)
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
