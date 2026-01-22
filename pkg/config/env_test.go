package config

import (
	"os"
	"testing"
)

func TestApplyEnvOverrides_StringFields(t *testing.T) {
	// Save and restore env
	cleanup := setEnvVars(t, map[string]string{
		"MARKATA_GO_OUTPUT_DIR":    "env-output",
		"MARKATA_GO_URL":           "https://env.example.com",
		"MARKATA_GO_TITLE":         "Env Title",
		"MARKATA_GO_DESCRIPTION":   "Env Description",
		"MARKATA_GO_AUTHOR":        "Env Author",
		"MARKATA_GO_ASSETS_DIR":    "env-assets",
		"MARKATA_GO_TEMPLATES_DIR": "env-templates",
	})
	defer cleanup()

	config := DefaultConfig()
	err := ApplyEnvOverrides(config)
	if err != nil {
		t.Fatalf("ApplyEnvOverrides() error = %v", err)
	}

	tests := []struct {
		field string
		got   string
		want  string
	}{
		{"OutputDir", config.OutputDir, "env-output"},
		{"URL", config.URL, "https://env.example.com"},
		{"Title", config.Title, "Env Title"},
		{"Description", config.Description, "Env Description"},
		{"Author", config.Author, "Env Author"},
		{"AssetsDir", config.AssetsDir, "env-assets"},
		{"TemplatesDir", config.TemplatesDir, "env-templates"},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.field, tt.got, tt.want)
		}
	}
}

func TestApplyEnvOverrides_IntFields(t *testing.T) {
	cleanup := setEnvVars(t, map[string]string{
		"MARKATA_GO_CONCURRENCY": "8",
	})
	defer cleanup()

	config := DefaultConfig()
	ApplyEnvOverrides(config)

	if config.Concurrency != 8 {
		t.Errorf("Concurrency = %d, want 8", config.Concurrency)
	}
}

func TestApplyEnvOverrides_BoolFields(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"true lowercase", "true", true},
		{"TRUE uppercase", "TRUE", true},
		{"True mixed", "True", true},
		{"1", "1", true},
		{"yes", "yes", true},
		{"YES", "YES", true},
		{"false", "false", false},
		{"FALSE", "FALSE", false},
		{"0", "0", false},
		{"no", "no", false},
		{"NO", "NO", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setEnvVars(t, map[string]string{
				"MARKATA_GO_GLOB_USE_GITIGNORE": tt.value,
			})
			defer cleanup()

			config := DefaultConfig()
			config.GlobConfig.UseGitignore = !tt.want // Set to opposite
			ApplyEnvOverrides(config)

			if config.GlobConfig.UseGitignore != tt.want {
				t.Errorf("UseGitignore = %v, want %v for input %q", config.GlobConfig.UseGitignore, tt.want, tt.value)
			}
		})
	}
}

func TestApplyEnvOverrides_ListFields(t *testing.T) {
	cleanup := setEnvVars(t, map[string]string{
		"MARKATA_GO_HOOKS":               "markdown,template,sitemap",
		"MARKATA_GO_DISABLED_HOOKS":      "seo,analytics",
		"MARKATA_GO_GLOB_PATTERNS":       "posts/**/*.md,pages/*.md",
		"MARKATA_GO_MARKDOWN_EXTENSIONS": "tables,footnotes",
	})
	defer cleanup()

	config := DefaultConfig()
	ApplyEnvOverrides(config)

	tests := []struct {
		field string
		got   []string
		want  []string
	}{
		{"Hooks", config.Hooks, []string{"markdown", "template", "sitemap"}},
		{"DisabledHooks", config.DisabledHooks, []string{"seo", "analytics"}},
		{"GlobPatterns", config.GlobConfig.Patterns, []string{"posts/**/*.md", "pages/*.md"}},
		{"MarkdownExtensions", config.MarkdownConfig.Extensions, []string{"tables", "footnotes"}},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if len(tt.got) != len(tt.want) {
				t.Errorf("%s length = %d, want %d", tt.field, len(tt.got), len(tt.want))
				return
			}
			for i, v := range tt.want {
				if tt.got[i] != v {
					t.Errorf("%s[%d] = %q, want %q", tt.field, i, tt.got[i], v)
				}
			}
		})
	}
}

func TestApplyEnvOverrides_NestedFields(t *testing.T) {
	cleanup := setEnvVars(t, map[string]string{
		"MARKATA_GO_FEED_DEFAULTS_ITEMS_PER_PAGE":              "25",
		"MARKATA_GO_FEED_DEFAULTS_ORPHAN_THRESHOLD":            "5",
		"MARKATA_GO_FEED_DEFAULTS_FORMATS_HTML":                "true",
		"MARKATA_GO_FEED_DEFAULTS_FORMATS_RSS":                 "false",
		"MARKATA_GO_FEED_DEFAULTS_FORMATS_ATOM":                "true",
		"MARKATA_GO_FEED_DEFAULTS_SYNDICATION_MAX_ITEMS":       "100",
		"MARKATA_GO_FEED_DEFAULTS_SYNDICATION_INCLUDE_CONTENT": "true",
	})
	defer cleanup()

	config := DefaultConfig()
	ApplyEnvOverrides(config)

	if config.FeedDefaults.ItemsPerPage != 25 {
		t.Errorf("FeedDefaults.ItemsPerPage = %d, want 25", config.FeedDefaults.ItemsPerPage)
	}
	if config.FeedDefaults.OrphanThreshold != 5 {
		t.Errorf("FeedDefaults.OrphanThreshold = %d, want 5", config.FeedDefaults.OrphanThreshold)
	}
	if !config.FeedDefaults.Formats.HTML {
		t.Error("FeedDefaults.Formats.HTML should be true")
	}
	if config.FeedDefaults.Formats.RSS {
		t.Error("FeedDefaults.Formats.RSS should be false")
	}
	if !config.FeedDefaults.Formats.Atom {
		t.Error("FeedDefaults.Formats.Atom should be true")
	}
	if config.FeedDefaults.Syndication.MaxItems != 100 {
		t.Errorf("FeedDefaults.Syndication.MaxItems = %d, want 100", config.FeedDefaults.Syndication.MaxItems)
	}
	if !config.FeedDefaults.Syndication.IncludeContent {
		t.Error("FeedDefaults.Syndication.IncludeContent should be true")
	}
}

func TestApplyEnvOverrides_CaseInsensitive(t *testing.T) {
	// Test that keys are case-insensitive
	cleanup := setEnvVars(t, map[string]string{
		"MARKATA_GO_OUTPUT_DIR": "lower",
	})
	defer cleanup()

	config := DefaultConfig()
	ApplyEnvOverrides(config)

	if config.OutputDir != "lower" {
		t.Errorf("OutputDir = %q, want %q", config.OutputDir, "lower")
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"Yes", true},
		{"false", false},
		{"False", false},
		{"FALSE", false},
		{"0", false},
		{"no", false},
		{"NO", false},
		{"No", false},
		{"", false},
		{"invalid", false},
		{"  true  ", true},
		{"  false  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseBool(tt.input)
			if got != tt.want {
				t.Errorf("parseBool(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseStringList(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"single", []string{"single"}},
		{"", nil},
		{"  a  ,  b  ,  c  ", []string{"a", "b", "c"}},
		{"a,,b", []string{"a", "b"}},
		{",a,b,", []string{"a", "b"}},
		{"**/*.md,posts/**/*.md", []string{"**/*.md", "posts/**/*.md"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseStringList(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseStringList(%q) = %v (len=%d), want %v (len=%d)", tt.input, got, len(got), tt.want, len(tt.want))
				return
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("parseStringList(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGetEnvValue(t *testing.T) {
	cleanup := setEnvVars(t, map[string]string{
		"MARKATA_GO_TEST_KEY": "test-value",
	})
	defer cleanup()

	val, ok := GetEnvValue("TEST_KEY")
	if !ok {
		t.Error("GetEnvValue() should find TEST_KEY")
	}
	if val != "test-value" {
		t.Errorf("GetEnvValue() = %q, want %q", val, "test-value")
	}

	_, ok = GetEnvValue("NONEXISTENT")
	if ok {
		t.Error("GetEnvValue() should not find NONEXISTENT")
	}
}

func TestSetEnvValue(t *testing.T) {
	err := SetEnvValue("NEW_KEY", "new-value")
	if err != nil {
		t.Fatalf("SetEnvValue() error = %v", err)
	}
	defer os.Unsetenv("MARKATA_GO_NEW_KEY")

	val, ok := GetEnvValue("NEW_KEY")
	if !ok {
		t.Error("SetEnvValue() should have set the key")
	}
	if val != "new-value" {
		t.Errorf("SetEnvValue() value = %q, want %q", val, "new-value")
	}
}

func TestUnsetEnvValue(t *testing.T) {
	os.Setenv("MARKATA_GO_TO_UNSET", "value")

	err := UnsetEnvValue("TO_UNSET")
	if err != nil {
		t.Fatalf("UnsetEnvValue() error = %v", err)
	}

	_, ok := GetEnvValue("TO_UNSET")
	if ok {
		t.Error("UnsetEnvValue() should have unset the key")
	}
}

func TestConfigFromEnv(t *testing.T) {
	cleanup := setEnvVars(t, map[string]string{
		"MARKATA_GO_OUTPUT_DIR": "from-env",
		"MARKATA_GO_TITLE":      "Env Config",
	})
	defer cleanup()

	config := ConfigFromEnv()

	if config.OutputDir != "from-env" {
		t.Errorf("OutputDir = %q, want %q", config.OutputDir, "from-env")
	}
	if config.Title != "Env Config" {
		t.Errorf("Title = %q, want %q", config.Title, "Env Config")
	}
	// Should still have defaults for non-overridden fields
	defaults := DefaultConfig()
	if config.TemplatesDir != defaults.TemplatesDir {
		t.Errorf("TemplatesDir = %q, want default %q", config.TemplatesDir, defaults.TemplatesDir)
	}
}

func TestApplyEnvOverrides_InvalidInt(t *testing.T) {
	cleanup := setEnvVars(t, map[string]string{
		"MARKATA_GO_CONCURRENCY": "not-a-number",
	})
	defer cleanup()

	config := DefaultConfig()
	config.Concurrency = 4 // Set a value
	ApplyEnvOverrides(config)

	// Should remain unchanged when env value is invalid
	if config.Concurrency != 4 {
		t.Errorf("Concurrency = %d, want 4 (should be unchanged for invalid input)", config.Concurrency)
	}
}

func TestApplyEnvOverrides_EmptyList(t *testing.T) {
	cleanup := setEnvVars(t, map[string]string{
		"MARKATA_GO_HOOKS": "",
	})
	defer cleanup()

	config := DefaultConfig()
	config.Hooks = []string{"original"}
	ApplyEnvOverrides(config)

	// Empty string should result in nil list, overwriting the original
	if len(config.Hooks) != 0 {
		t.Errorf("Hooks = %v, want empty", config.Hooks)
	}
}

func TestApplyEnvOverrides_AlternateKeyFormats(t *testing.T) {
	// Test that both feed_defaults and feeds_defaults work
	cleanup := setEnvVars(t, map[string]string{
		"MARKATA_GO_FEEDS_DEFAULTS_ITEMS_PER_PAGE": "50",
	})
	defer cleanup()

	config := DefaultConfig()
	ApplyEnvOverrides(config)

	if config.FeedDefaults.ItemsPerPage != 50 {
		t.Errorf("FeedDefaults.ItemsPerPage = %d, want 50", config.FeedDefaults.ItemsPerPage)
	}
}

func TestStructToEnvKeys(t *testing.T) {
	type TestStruct struct {
		Name   string   `json:"name"`
		Count  int      `json:"count"`
		Active bool     `json:"active"`
		Tags   []string `json:"tags"`
	}

	keys := StructToEnvKeys("PREFIX_", TestStruct{})

	expectedKeys := []string{"PREFIX_NAME", "PREFIX_COUNT", "PREFIX_ACTIVE", "PREFIX_TAGS"}
	for _, key := range expectedKeys {
		if _, ok := keys[key]; !ok {
			t.Errorf("StructToEnvKeys() missing key %q", key)
		}
	}
}

// Helper function to set environment variables for tests
func setEnvVars(t *testing.T, vars map[string]string) func() {
	t.Helper()

	// Save original values
	originals := make(map[string]string)
	exists := make(map[string]bool)
	for k := range vars {
		if v, ok := os.LookupEnv(k); ok {
			originals[k] = v
			exists[k] = true
		}
	}

	// Set new values
	for k, v := range vars {
		os.Setenv(k, v)
	}

	// Return cleanup function
	return func() {
		for k := range vars {
			if exists[k] {
				os.Setenv(k, originals[k])
			} else {
				os.Unsetenv(k)
			}
		}
	}
}
