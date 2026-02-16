//go:build cgo

package config

import (
	"strings"
	"testing"
)

func TestSetYamlValue_UpdatesNestedKey(t *testing.T) {
	input := []byte("markata-go:\n  theme:\n    palette: light\n")
	path, err := ParseKeyPath("theme.palette")
	if err != nil {
		t.Fatalf("ParseKeyPath error: %v", err)
	}
	path = normalizeConfigPath(path)
	updated, err := setYamlValue(input, path, "dark")
	if err != nil {
		t.Fatalf("setYamlValue error: %v", err)
	}
	if !strings.Contains(string(updated), "palette: dark") {
		t.Fatalf("unexpected output: %s", string(updated))
	}
}
