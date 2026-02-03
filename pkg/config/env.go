package config

import (
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

const envPrefix = "MARKATA_GO_"

// Common string constants used in environment variable processing.
const (
	envKeyURL         = "url"
	envKeyConcurrency = "concurrency"
	envKeyJSON        = "json"
)

// ApplyEnvOverrides applies environment variable overrides to a config.
// Environment variables are expected to follow the format MARKATA_GO_*.
// Nested keys use underscores: MARKATA_GO_FEEDS_DEFAULTS_ITEMS_PER_PAGE
// Boolean values: "true", "1", "yes" -> true; "false", "0", "no" -> false
// List values: comma-separated strings
func ApplyEnvOverrides(config *models.Config) error {
	env := os.Environ()
	overrides := make(map[string]string)

	for _, e := range env {
		if strings.HasPrefix(e, envPrefix) {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimPrefix(parts[0], envPrefix)
				overrides[key] = parts[1]
			}
		}
	}

	// Apply simple overrides
	for key, value := range overrides {
		applyEnvOverride(config, key, value)
	}

	return nil
}

// applyEnvOverride applies a single environment variable override.
//
//nolint:gocyclo // This is a switch statement mapping env vars to config fields, complexity is unavoidable.
func applyEnvOverride(config *models.Config, key, value string) {
	// Normalize the key to lowercase for comparison
	keyLower := strings.ToLower(key)

	switch keyLower {
	case "output_dir":
		config.OutputDir = value
	case envKeyURL:
		config.URL = value
	case "title":
		config.Title = value
	case "description":
		config.Description = value
	case "author":
		config.Author = value
	case "assets_dir":
		config.AssetsDir = value
	case "templates_dir":
		config.TemplatesDir = value
	case envKeyConcurrency:
		if v, err := strconv.Atoi(value); err == nil {
			config.Concurrency = v
		}
	case "hooks":
		config.Hooks = parseStringList(value)
	case "disabled_hooks":
		config.DisabledHooks = parseStringList(value)
	case "glob_patterns":
		config.GlobConfig.Patterns = parseStringList(value)
	case "glob_use_gitignore":
		config.GlobConfig.UseGitignore = parseBool(value)
	case "markdown_extensions":
		config.MarkdownConfig.Extensions = parseStringList(value)
	case "feed_defaults_items_per_page", "feeds_defaults_items_per_page":
		if v, err := strconv.Atoi(value); err == nil {
			config.FeedDefaults.ItemsPerPage = v
		}
	case "feed_defaults_orphan_threshold", "feeds_defaults_orphan_threshold":
		if v, err := strconv.Atoi(value); err == nil {
			config.FeedDefaults.OrphanThreshold = v
		}
	case "feed_defaults_formats_html", "feeds_defaults_formats_html":
		config.FeedDefaults.Formats.HTML = parseBool(value)
	case "feed_defaults_formats_rss", "feeds_defaults_formats_rss":
		config.FeedDefaults.Formats.RSS = parseBool(value)
	case "feed_defaults_formats_atom", "feeds_defaults_formats_atom":
		config.FeedDefaults.Formats.Atom = parseBool(value)
	case "feed_defaults_formats_json", "feeds_defaults_formats_json":
		config.FeedDefaults.Formats.JSON = parseBool(value)
	case "feed_defaults_formats_markdown", "feeds_defaults_formats_markdown":
		config.FeedDefaults.Formats.Markdown = parseBool(value)
	case "feed_defaults_formats_text", "feeds_defaults_formats_text":
		config.FeedDefaults.Formats.Text = parseBool(value)
	case "feed_defaults_formats_sitemap", "feeds_defaults_formats_sitemap":
		config.FeedDefaults.Formats.Sitemap = parseBool(value)
	case "feed_defaults_syndication_max_items", "feeds_defaults_syndication_max_items":
		if v, err := strconv.Atoi(value); err == nil {
			config.FeedDefaults.Syndication.MaxItems = v
		}
	case "feed_defaults_syndication_include_content", "feeds_defaults_syndication_include_content":
		config.FeedDefaults.Syndication.IncludeContent = parseBool(value)
	// Pagefind search settings
	case "search_pagefind_auto_install":
		if config.Extra == nil {
			config.Extra = make(map[string]interface{})
		}
		if searchConfig, ok := config.Extra["search"].(models.SearchConfig); ok {
			autoInstall := parseBool(value)
			searchConfig.Pagefind.AutoInstall = &autoInstall
			config.Extra["search"] = searchConfig
		}
	case "search_pagefind_cache_dir":
		if config.Extra == nil {
			config.Extra = make(map[string]interface{})
		}
		if searchConfig, ok := config.Extra["search"].(models.SearchConfig); ok {
			searchConfig.Pagefind.CacheDir = value
			config.Extra["search"] = searchConfig
		}
	case "search_pagefind_version":
		if config.Extra == nil {
			config.Extra = make(map[string]interface{})
		}
		if searchConfig, ok := config.Extra["search"].(models.SearchConfig); ok {
			searchConfig.Pagefind.Version = value
			config.Extra["search"] = searchConfig
		}
	case "search_pagefind_bundle_dir":
		if config.Extra == nil {
			config.Extra = make(map[string]interface{})
		}
		if searchConfig, ok := config.Extra["search"].(models.SearchConfig); ok {
			searchConfig.Pagefind.BundleDir = value
			config.Extra["search"] = searchConfig
		}
	case "search_pagefind_verbose":
		if config.Extra == nil {
			config.Extra = make(map[string]interface{})
		}
		if searchConfig, ok := config.Extra["search"].(models.SearchConfig); ok {
			verbose := parseBool(value)
			searchConfig.Pagefind.Verbose = &verbose
			config.Extra["search"] = searchConfig
		}
	}
}

// parseBool parses a string into a boolean.
// "true", "1", "yes" -> true
// "false", "0", "no" -> false
// All comparisons are case-insensitive.
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	}
	return false
}

// parseStringList parses a comma-separated string into a slice.
func parseStringList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// GetEnvValue returns the value of an environment variable with the MARKATA_GO_ prefix.
func GetEnvValue(key string) (string, bool) {
	return os.LookupEnv(envPrefix + strings.ToUpper(key))
}

// SetEnvValue sets an environment variable with the MARKATA_GO_ prefix.
// This is primarily useful for testing.
func SetEnvValue(key, value string) error {
	return os.Setenv(envPrefix+strings.ToUpper(key), value)
}

// UnsetEnvValue unsets an environment variable with the MARKATA_GO_ prefix.
// This is primarily useful for testing.
func UnsetEnvValue(key string) error {
	return os.Unsetenv(envPrefix + strings.ToUpper(key))
}

// FromEnv creates a Config entirely from environment variables.
// This is useful when no config file is available.
func FromEnv() *models.Config {
	config := DefaultConfig()
	_ = ApplyEnvOverrides(config) //nolint:errcheck // Best-effort env override
	return config
}

// StructToEnvKeys returns a map of environment variable keys for a struct.
// This is useful for documentation and debugging.
func StructToEnvKeys(prefix string, v interface{}) map[string]string {
	result := make(map[string]string)
	structToEnvKeysRecursive(prefix, reflect.TypeOf(v), result)
	return result
}

func structToEnvKeysRecursive(prefix string, t reflect.Type, result map[string]string) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get the field name for the environment variable
		name := field.Name
		if tag := field.Tag.Get("json"); tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] != "" && parts[0] != "-" {
				name = parts[0]
			}
		}

		envKey := prefix + strings.ToUpper(name)

		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		switch fieldType.Kind() {
		case reflect.Struct:
			structToEnvKeysRecursive(envKey+"_", fieldType, result)
		case reflect.Slice:
			result[envKey] = "comma-separated list"
		case reflect.Bool:
			result[envKey] = "true/false/1/0/yes/no"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			result[envKey] = "integer"
		case reflect.String:
			result[envKey] = "string"
		default:
			// Other types (uint, float, complex, etc.) are not currently supported for env vars
		}
	}
}
