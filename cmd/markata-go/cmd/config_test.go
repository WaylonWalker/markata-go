package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{"string", "hello", "hello"},
		{"string with spaces", "hello world", "hello world"},
		{"bool true", "true", true},
		{"bool false", "false", false},
		{"bool TRUE", "TRUE", true},
		{"bool FALSE", "FALSE", false},
		{"int", "42", 42},
		{"int negative", "-5", -5},
		{"float", "3.14", 3.14},
		{"json array", `["a", "b", "c"]`, []interface{}{"a", "b", "c"}},
		{"json object", `{"key": "value"}`, map[string]interface{}{"key": "value"}},
		{"string that looks like number", "007", 7}, // Note: leading zeros are parsed as int
		{"url", "https://example.com", "https://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseValue(tt.input)

			// For slices and maps, compare as JSON strings
			switch expected := tt.expected.(type) {
			case []interface{}:
				gotSlice, ok := got.([]interface{})
				if !ok {
					t.Errorf("parseValue(%q) = %T, want []interface{}", tt.input, got)
					return
				}
				if len(gotSlice) != len(expected) {
					t.Errorf("parseValue(%q) = %v, want %v", tt.input, got, tt.expected)
				}
			case map[string]interface{}:
				gotMap, ok := got.(map[string]interface{})
				if !ok {
					t.Errorf("parseValue(%q) = %T, want map[string]interface{}", tt.input, got)
					return
				}
				if len(gotMap) != len(expected) {
					t.Errorf("parseValue(%q) = %v, want %v", tt.input, got, tt.expected)
				}
			default:
				if got != tt.expected {
					t.Errorf("parseValue(%q) = %v (%T), want %v (%T)", tt.input, got, got, tt.expected, tt.expected)
				}
			}
		})
	}
}

func TestSetMapValue(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    interface{}
		wantErr  bool
		checkKey string
		checkVal interface{}
	}{
		{
			name:     "top-level key",
			key:      "title",
			value:    "New Title",
			checkKey: "title",
			checkVal: "New Title",
		},
		{
			name:     "nested key",
			key:      "glob.patterns",
			value:    []interface{}{"**/*.md"},
			checkKey: "glob.patterns",
			checkVal: []interface{}{"**/*.md"},
		},
		{
			name:     "deep nested key",
			key:      "feed_defaults.formats.html",
			value:    true,
			checkKey: "feed_defaults.formats.html",
			checkVal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := map[string]interface{}{
				"markata-go": map[string]interface{}{},
			}

			err := setMapValue(m, tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("setMapValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				got := getMapValue(m, tt.checkKey)
				if got == nil {
					t.Errorf("getMapValue(%q) = nil, want %v", tt.checkKey, tt.checkVal)
				}
			}
		})
	}
}

func TestFormatFromPath(t *testing.T) {
	tests := []struct {
		path   string
		expect string
	}{
		{"config.toml", formatTOML},
		{"config.yaml", formatYAML},
		{"config.yml", formatYAML},
		{"config.json", formatJSON},
		{"config.txt", formatTOML}, // default
		{"config.TOML", formatTOML},
		{"config.YAML", formatYAML},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := formatFromPath(tt.path)
			if got != tt.expect {
				t.Errorf("formatFromPath(%q) = %q, want %q", tt.path, got, tt.expect)
			}
		})
	}
}

func TestParseConfigToMap(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		format  string
		wantErr bool
	}{
		{
			name: "toml config",
			data: `[markata-go]
title = "Test"
`,
			format:  formatTOML,
			wantErr: false,
		},
		{
			name: "yaml config",
			data: `markata-go:
  title: Test
`,
			format:  formatYAML,
			wantErr: false,
		},
		{
			name:    "json config",
			data:    `{"markata-go": {"title": "Test"}}`,
			format:  formatJSON,
			wantErr: false,
		},
		{
			name:    "invalid toml",
			data:    `[invalid`,
			format:  formatTOML,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseConfigToMap([]byte(tt.data), tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConfigToMap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigSetIntegration(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		initialContent string
		format         string
		key            string
		value          string
		checkContains  string
	}{
		{
			name: "set toml string",
			initialContent: `[markata-go]
title = "Old"
`,
			format:        formatTOML,
			key:           "title",
			value:         "New Title",
			checkContains: `title = "New Title"`,
		},
		{
			name: "set toml int",
			initialContent: `[markata-go]
concurrency = 2
`,
			format:        formatTOML,
			key:           "concurrency",
			value:         "8",
			checkContains: "concurrency = 8",
		},
		{
			name: "set yaml string",
			initialContent: `markata-go:
  title: Old
`,
			format:        formatYAML,
			key:           "title",
			value:         "New Title",
			checkContains: "title: New Title",
		},
		{
			name:           "set json bool",
			initialContent: `{"markata-go": {"verbose": false}}`,
			format:         formatJSON,
			key:            "verbose",
			value:          "true",
			checkContains:  `"verbose": true`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			ext := "." + tt.format
			if tt.format == formatYAML {
				ext = ".yaml"
			}
			configPath := filepath.Join(tmpDir, "config"+ext)
			if err := os.WriteFile(configPath, []byte(tt.initialContent), 0o600); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Read and parse
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("Failed to read config: %v", err)
			}

			configMap, err := parseConfigToMap(data, tt.format)
			if err != nil {
				t.Fatalf("Failed to parse config: %v", err)
			}

			// Set value
			parsedValue := parseValue(tt.value)
			if err := setMapValue(configMap, tt.key, parsedValue); err != nil {
				t.Fatalf("Failed to set value: %v", err)
			}

			// Marshal back
			newData, err := marshalConfigMap(configMap, tt.format)
			if err != nil {
				t.Fatalf("Failed to marshal config: %v", err)
			}

			// Write and verify
			if err := os.WriteFile(configPath, newData, 0o600); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}

			result, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("Failed to read result: %v", err)
			}
			if !strings.Contains(string(result), tt.checkContains) {
				t.Errorf("Result does not contain %q:\n%s", tt.checkContains, result)
			}
		})
	}
}

func TestFormatValueForDisplay(t *testing.T) {
	tests := []struct {
		name   string
		value  interface{}
		expect string
	}{
		{"nil", nil, "<not set>"},
		{"string", "hello", `"hello"`},
		{"int", 42, "42"},
		{"bool", true, "true"},
		{"slice", []interface{}{"a", "b"}, `["a","b"]`},
		{"map", map[string]interface{}{"key": "value"}, `{"key":"value"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValueForDisplay(tt.value)
			if got != tt.expect {
				t.Errorf("formatValueForDisplay(%v) = %q, want %q", tt.value, got, tt.expect)
			}
		})
	}
}
