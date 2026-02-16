package config

import (
	"strings"
	"testing"
)

func TestSetTomlValue_PreservesLayout(t *testing.T) {
	input := []byte(`title = "Site"

[markata-go]
# keep me
theme = "default"

[markata-go.theme]
palette = "light"
`)

	path, err := ParseKeyPath("theme.palette")
	if err != nil {
		t.Fatalf("ParseKeyPath error: %v", err)
	}
	path = normalizeConfigPath(path)
	updated, err := setTomlValue(input, path, "dark")
	if err != nil {
		t.Fatalf("setTomlValue error: %v", err)
	}
	result := string(updated)
	if !containsAll(result, []string{"# keep me", "palette = \"dark\""}) {
		t.Fatalf("unexpected output:\n%s", result)
	}
}

func containsAll(content string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(content, part) {
			return false
		}
	}
	return true
}
