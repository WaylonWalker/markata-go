package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/WaylonWalker/markata-go/pkg/config"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Common string constants to avoid goconst warnings.
const (
	formatJSON  = "json"
	formatTOML  = "toml"
	formatYAML  = "yaml"
	extYAML     = ".yaml"
	extYML      = ".yml"
	boolStrTrue = "true"
)

// configCmd represents the config command group.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration commands",
	Long: `Commands for managing markata-go configuration.

Subcommands:
  show     - Display the resolved configuration
  get      - Get a specific configuration value
  set      - Set a configuration value
  validate - Validate the configuration file
  init     - Create a new configuration file`,
}

// configShowCmd shows the resolved configuration.
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display resolved configuration",
	Long: `Display the fully resolved configuration with all defaults applied.

The configuration is merged from:
  1. Default values
  2. Config file (discovered or specified)
  3. Environment variables

Example usage:
  markata-go config show              # Show as YAML
  markata-go config show --json       # Show as JSON
  markata-go config show --toml       # Show as TOML
  markata-go config show --annotate   # Show with source annotations
  markata-go config show --diff       # Show only user-provided values`,
	RunE: runConfigShowCommand,
}

// configGetCmd gets a specific configuration value.
var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get specific config value",
	Long: `Get a specific configuration value by key.

Supports dot notation for nested values.

Example usage:
  markata-go config get output_dir
  markata-go config get glob.patterns
  markata-go config get feed_defaults.items_per_page`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGetCommand,
}

// configValidateCmd validates the configuration.
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Long: `Validate the configuration file and report any errors or warnings.

Exit codes:
  0 - Configuration is valid (warnings may be present)
  1 - Configuration has errors

Example usage:
  markata-go config validate
  markata-go config validate -c custom-config.toml`,
	RunE: runConfigValidateCommand,
}

// configInitCmd creates a new configuration file.
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create new config file",
	Long: `Create a new configuration file with sensible defaults.

Supports TOML, YAML, and JSON formats (detected from filename).

Example usage:
  markata-go config init                    # Creates markata-go.toml
  markata-go config init markata-go.yaml    # Creates YAML config
  markata-go config init --force            # Overwrite existing file`,
	RunE: runConfigInitCommand,
}

// configSetCmd sets a configuration value.
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set configuration value",
	Long: `Set a configuration value in the config file.

Supports dot notation for nested values. Values are automatically
type-detected (string, int, bool, array/object via JSON).

Example usage:
  markata-go config set output_dir "dist"
  markata-go config set url "https://example.com"
  markata-go config set concurrency 4
  markata-go config set glob.use_gitignore true
  markata-go config set glob.patterns '["posts/**/*.md", "pages/*.md"]'
  markata-go config set feed_defaults.items_per_page 15

Flags:
  --dry-run   Show what would be changed without writing
  --backup    Create a backup before modifying the config file`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSetCommand,
}

var (
	// configFormat specifies the output format for config show.
	configFormat string

	// configShowAnnotate shows source of each config value.
	configShowAnnotate bool

	// configShowDiff shows only user-provided values that differ from defaults.
	configShowDiff bool

	// configInitForce overwrites existing config file.
	configInitForce bool

	// configSetDryRun shows what would be changed without writing.
	configSetDryRun bool

	// configSetBackup creates a backup before modifying.
	configSetBackup bool
)

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configInitCmd)

	configShowCmd.Flags().StringVar(&configFormat, "format", "yaml", "output format (yaml, json, toml)")
	configShowCmd.Flags().Bool("json", false, "output as JSON (shorthand for --format=json)")
	configShowCmd.Flags().Bool("toml", false, "output as TOML (shorthand for --format=toml)")
	configShowCmd.Flags().BoolVar(&configShowAnnotate, "annotate", false, "show source of each config value (default vs user config)")
	configShowCmd.Flags().BoolVar(&configShowDiff, "diff", false, "show only user-provided values that differ from defaults")

	configInitCmd.Flags().BoolVar(&configInitForce, "force", false, "overwrite existing file")

	configSetCmd.Flags().BoolVar(&configSetDryRun, "dry-run", false, "show what would be changed without writing")
	configSetCmd.Flags().BoolVar(&configSetBackup, "backup", false, "create backup before modifying")
}

func runConfigShowCommand(cmd *cobra.Command, _ []string) error {
	// Handle format shorthands
	jsonFlag, err := cmd.Flags().GetBool(formatJSON)
	if err != nil {
		return fmt.Errorf("failed to get json flag: %w", err)
	}
	if jsonFlag {
		configFormat = formatJSON
	}
	tomlFlag, err := cmd.Flags().GetBool(formatTOML)
	if err != nil {
		return fmt.Errorf("failed to get toml flag: %w", err)
	}
	if tomlFlag {
		configFormat = formatTOML
	}

	// Handle --annotate and --diff modes
	if configShowAnnotate || configShowDiff {
		return runConfigShowWithSources(cfgFile)
	}

	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Output in requested format
	switch strings.ToLower(configFormat) {
	case formatJSON:
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))

	case formatTOML:
		if err := toml.NewEncoder(os.Stdout).Encode(cfg); err != nil {
			return fmt.Errorf("failed to marshal TOML: %w", err)
		}

	case formatYAML:
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		fmt.Print(string(data))

	default:
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		fmt.Print(string(data))
	}

	return nil
}

// runConfigShowWithSources handles --annotate and --diff flags.
// It compares user config with defaults to show value sources.
func runConfigShowWithSources(configPath string) error {
	// Get merged config as a map
	merged, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	mergedMap, err := configToMap(merged)
	if err != nil {
		return fmt.Errorf("failed to convert config to map: %w", err)
	}

	// Load user config file directly (without merging with defaults)
	var userMap map[string]interface{}
	var userConfigFile string

	if configPath != "" {
		userConfigFile = configPath
	} else {
		userConfigFile, _ = config.Discover() //nolint:errcheck // discovery failure is ok, we'll handle empty string
	}

	if userConfigFile != "" {
		data, err := os.ReadFile(userConfigFile)
		if err == nil {
			format := formatFromPath(userConfigFile)
			wrapper, err := parseConfigToMap(data, format)
			if err == nil {
				// Extract the inner markata-go config
				if inner, ok := wrapper["markata-go"].(map[string]interface{}); ok {
					userMap = inner
				}
			}
		}
	}

	if configShowDiff {
		// Show only values that differ from defaults
		return showDiffConfig(userMap, userConfigFile)
	}

	// Show annotated config
	return showAnnotatedConfig(mergedMap, userMap, userConfigFile)
}

// configToMap converts a Config struct to a map[string]interface{}.
func configToMap(cfg *models.Config) (map[string]interface{}, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// showDiffConfig prints only user-provided values that differ from defaults.
func showDiffConfig(userMap map[string]interface{}, userConfigFile string) error {
	if len(userMap) == 0 {
		fmt.Println("# No user configuration found")
		fmt.Println("# All values are defaults")
		return nil
	}

	if userConfigFile != "" {
		fmt.Printf("# User configuration from: %s\n", userConfigFile)
	}
	fmt.Println("# Values below differ from defaults:")
	fmt.Println()

	// Output the user config as YAML with comments
	data, err := yaml.Marshal(userMap)
	if err != nil {
		return fmt.Errorf("failed to marshal diff: %w", err)
	}
	fmt.Print(string(data))

	return nil
}

// showAnnotatedConfig prints the merged config with source annotations.
func showAnnotatedConfig(merged, user map[string]interface{}, userConfigFile string) error {
	// Print header
	fmt.Println("# Configuration with source annotations")
	if userConfigFile != "" {
		fmt.Printf("# User config: %s\n", userConfigFile)
	} else {
		fmt.Println("# User config: (none found, using defaults)")
	}
	fmt.Println()

	// Get YAML representation of merged config
	data, err := yaml.Marshal(merged)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Split into lines and annotate each
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			fmt.Println()
			continue
		}

		// Parse the line to extract key path
		trimmed := strings.TrimLeft(line, " ")
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "#") {
			// Array item or comment - just print
			fmt.Println(line)
			continue
		}

		// Try to extract key from "key: value" format
		colonIdx := strings.Index(trimmed, ":")
		if colonIdx == -1 {
			fmt.Println(line)
			continue
		}

		key := trimmed[:colonIdx]
		indent := len(line) - len(trimmed)

		// Determine source based on whether it's in user config
		source := "default"
		if isKeyInMap(user, key) {
			if userConfigFile != "" {
				source = "user: " + filepath.Base(userConfigFile)
			} else {
				source = "user"
			}
		}

		// Calculate padding for alignment
		padding := 40 - len(line)
		if padding < 2 {
			padding = 2
		}

		// Print with annotation
		// Only annotate leaf values (lines with values after the colon)
		valueAfterColon := strings.TrimSpace(trimmed[colonIdx+1:])
		if valueAfterColon != "" && !strings.HasPrefix(valueAfterColon, "|") && !strings.HasPrefix(valueAfterColon, ">") {
			fmt.Printf("%s%s# %s\n", line, strings.Repeat(" ", padding), source)
		} else {
			// It's a parent key (object/map) - check if any child is from user
			if indent == 0 && isKeyInMap(user, key) {
				fmt.Printf("%s%s# %s (partial)\n", line, strings.Repeat(" ", padding), source)
			} else {
				fmt.Println(line)
			}
		}
	}

	return nil
}

// isKeyInMap checks if a key exists in the map (handles nested maps).
func isKeyInMap(m map[string]interface{}, key string) bool {
	if m == nil {
		return false
	}
	_, exists := m[key]
	return exists
}

func runConfigGetCommand(_ *cobra.Command, args []string) error {
	key := args[0]

	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get value by key (supports dot notation)
	value, err := getConfigValue(cfg, key)
	if err != nil {
		return err
	}

	// Print value
	switch v := value.(type) {
	case string:
		fmt.Println(v)
	case []string:
		for _, s := range v {
			fmt.Println(s)
		}
	default:
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			fmt.Printf("%v\n", v)
		} else {
			fmt.Println(string(data))
		}
	}

	return nil
}

// getConfigValue retrieves a configuration value by key with dot notation support.
func getConfigValue(cfg *models.Config, key string) (interface{}, error) {
	// Map of top-level keys to their values
	lowKey := strings.ToLower(key)
	if val, err := getTopLevelConfigValue(cfg, lowKey); err == nil {
		return val, nil
	}

	// Handle nested keys
	return getNestedConfigValue(cfg, lowKey)
}

// getTopLevelConfigValue returns top-level config values.
func getTopLevelConfigValue(cfg *models.Config, key string) (interface{}, error) {
	switch key {
	case "output_dir":
		return cfg.OutputDir, nil
	case "url":
		return cfg.URL, nil
	case "title":
		return cfg.Title, nil
	case "description":
		return cfg.Description, nil
	case "author":
		return cfg.Author, nil
	case "assets_dir":
		return cfg.AssetsDir, nil
	case "templates_dir":
		return cfg.TemplatesDir, nil
	case "hooks":
		return cfg.Hooks, nil
	case "disabled_hooks":
		return cfg.DisabledHooks, nil
	case "concurrency":
		return cfg.Concurrency, nil
	default:
		return nil, fmt.Errorf("not a top-level key")
	}
}

// getNestedConfigValue returns nested config values (with dot notation).
func getNestedConfigValue(cfg *models.Config, key string) (interface{}, error) {
	parts := strings.SplitN(key, ".", 2)
	switch strings.ToLower(parts[0]) {
	case "glob":
		if len(parts) == 1 {
			return cfg.GlobConfig, nil
		}
		if strings.EqualFold(parts[1], "patterns") {
			return cfg.GlobConfig.Patterns, nil
		}
		if strings.EqualFold(parts[1], "use_gitignore") {
			return cfg.GlobConfig.UseGitignore, nil
		}
	case "markdown":
		if len(parts) == 1 {
			return cfg.MarkdownConfig, nil
		}
		if strings.EqualFold(parts[1], "extensions") {
			return cfg.MarkdownConfig.Extensions, nil
		}
	case "feed_defaults":
		if len(parts) == 1 {
			return cfg.FeedDefaults, nil
		}
		if strings.EqualFold(parts[1], "items_per_page") {
			return cfg.FeedDefaults.ItemsPerPage, nil
		}
		if strings.EqualFold(parts[1], "orphan_threshold") {
			return cfg.FeedDefaults.OrphanThreshold, nil
		}
	case "feeds":
		return cfg.Feeds, nil
	}

	return nil, fmt.Errorf("unknown config key: %s", key)
}

func runConfigValidateCommand(_ *cobra.Command, _ []string) error {
	// Load and validate configuration
	cfg, validationErrs, err := config.LoadAndValidate(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Split errors and warnings
	actualErrors, warnings := config.SplitErrorsAndWarnings(validationErrs)

	// Print warnings
	if len(warnings) > 0 {
		fmt.Println("Warnings:")
		for _, w := range warnings {
			fmt.Printf("  - %v\n", w)
		}
		fmt.Println()
	}

	// Print errors
	if len(actualErrors) > 0 {
		fmt.Println("Errors:")
		for _, e := range actualErrors {
			fmt.Printf("  - %v\n", e)
		}
		return fmt.Errorf("configuration validation failed")
	}

	// Print success
	configPath := cfgFile
	if configPath == "" {
		var discoverErr error
		configPath, discoverErr = config.Discover()
		if discoverErr != nil {
			configPath = ""
		}
	}
	if configPath == "" {
		configPath = "(defaults)"
	}

	fmt.Printf("Configuration is valid: %s\n", configPath)

	if verbose {
		fmt.Printf("\nConfiguration summary:\n")
		fmt.Printf("  Output directory: %s\n", cfg.OutputDir)
		fmt.Printf("  Site URL: %s\n", cfg.URL)
		fmt.Printf("  Site title: %s\n", cfg.Title)
		fmt.Printf("  Glob patterns: %v\n", cfg.GlobConfig.Patterns)
		fmt.Printf("  Feeds defined: %d\n", len(cfg.Feeds))
	}

	return nil
}

func runConfigInitCommand(_ *cobra.Command, args []string) error {
	// Determine output filename
	filename := "markata-go.toml"
	if len(args) > 0 {
		filename = args[0]
	}

	// Check if file exists
	if _, err := os.Stat(filename); err == nil && !configInitForce {
		return fmt.Errorf("file already exists: %s (use --force to overwrite)", filename)
	}

	// Determine format from filename
	var content string
	switch {
	case strings.HasSuffix(filename, extYAML) || strings.HasSuffix(filename, extYML):
		content = defaultConfigYAML
	case strings.HasSuffix(filename, ".json"):
		content = defaultConfigJSON
	default:
		content = defaultConfigTOML
	}

	// Write file (0o644 is appropriate for config files that should be world-readable)
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil { //nolint:gosec // config files should be readable
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Created: %s\n", filename)
	return nil
}

func runConfigSetCommand(_ *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// Discover config file
	configPath := cfgFile
	if configPath == "" {
		var err error
		configPath, err = config.Discover()
		if err != nil {
			return fmt.Errorf("no config file found (use -c to specify one or run 'config init' first): %w", err)
		}
	}

	// Read existing config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Determine format from extension
	format := formatFromPath(configPath)

	// Parse to generic map for modification
	configMap, err := parseConfigToMap(data, format)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Parse the value with automatic type detection
	parsedValue := parseValue(value)

	// Get the old value for display
	oldValue := getMapValue(configMap, key)

	// Set the value in the map
	if err := setMapValue(configMap, key, parsedValue); err != nil {
		return fmt.Errorf("failed to set value: %w", err)
	}

	// If dry-run, just show what would change
	if configSetDryRun {
		fmt.Printf("Would update %s in %s:\n", key, configPath)
		fmt.Printf("  Old: %v\n", formatValueForDisplay(oldValue))
		fmt.Printf("  New: %v\n", formatValueForDisplay(parsedValue))
		return nil
	}

	// Create backup if requested
	if configSetBackup {
		backupPath := configPath + ".backup." + time.Now().Format("20060102-150405")
		if err := copyFile(configPath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("Created backup: %s\n", backupPath)
	}

	// Marshal back to the original format
	newData, err := marshalConfigMap(configMap, format)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write the updated config
	if err := os.WriteFile(configPath, newData, 0o644); err != nil { //nolint:gosec // config files should be readable
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Updated %s in %s\n", key, configPath)
	return nil
}

// formatFromPath determines the config format from a file path.
func formatFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".toml":
		return formatTOML
	case extYAML, extYML:
		return formatYAML
	case ".json":
		return formatJSON
	default:
		return formatTOML
	}
}

// parseConfigToMap parses config data into a generic map[string]interface{}.
// The config is wrapped in a "markata-go" key.
func parseConfigToMap(data []byte, format string) (map[string]interface{}, error) {
	var wrapper map[string]interface{}

	switch format {
	case formatTOML:
		if err := toml.Unmarshal(data, &wrapper); err != nil {
			return nil, err
		}
	case formatYAML:
		if err := yaml.Unmarshal(data, &wrapper); err != nil {
			return nil, err
		}
	case formatJSON:
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	// If the config doesn't have the markata-go wrapper, add it
	if _, ok := wrapper["markata-go"]; !ok {
		wrapper = map[string]interface{}{"markata-go": wrapper}
	}

	return wrapper, nil
}

// marshalConfigMap marshals a config map back to the specified format.
func marshalConfigMap(configMap map[string]interface{}, format string) ([]byte, error) {
	switch format {
	case formatTOML:
		var buf strings.Builder
		encoder := toml.NewEncoder(&buf)
		if err := encoder.Encode(configMap); err != nil {
			return nil, err
		}
		return []byte(buf.String()), nil
	case formatYAML:
		return yaml.Marshal(configMap)
	case formatJSON:
		return json.MarshalIndent(configMap, "", "  ")
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// parseValue parses a string value with automatic type detection.
func parseValue(value string) interface{} {
	// Check for boolean
	lower := strings.ToLower(value)
	if lower == boolStrTrue {
		return true
	}
	if lower == "false" {
		return false
	}

	// Check for integer
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}

	// Check for float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		// Only treat as float if it has a decimal point
		if strings.Contains(value, ".") {
			return floatVal
		}
	}

	// Check for JSON array or object
	if strings.HasPrefix(value, "[") || strings.HasPrefix(value, "{") {
		var jsonVal interface{}
		if err := json.Unmarshal([]byte(value), &jsonVal); err == nil {
			return jsonVal
		}
	}

	// Default to string
	return value
}

// getMapValue retrieves a value from a nested map using dot notation.
func getMapValue(m map[string]interface{}, key string) interface{} {
	// First, look inside markata-go wrapper
	inner, ok := m["markata-go"].(map[string]interface{})
	if !ok {
		return nil
	}

	parts := strings.Split(key, ".")
	current := inner

	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return nil
		}

		if i == len(parts)-1 {
			return val
		}

		current, ok = val.(map[string]interface{})
		if !ok {
			return nil
		}
	}

	return nil
}

// setMapValue sets a value in a nested map using dot notation.
func setMapValue(m map[string]interface{}, key string, value interface{}) error {
	// Get or create the markata-go wrapper
	inner, ok := m["markata-go"].(map[string]interface{})
	if !ok {
		inner = make(map[string]interface{})
		m["markata-go"] = inner
	}

	parts := strings.Split(key, ".")
	current := inner

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - set the value
			current[part] = value
			return nil
		}

		// Navigate or create intermediate maps
		next, ok := current[part]
		if !ok {
			// Create the intermediate map
			newMap := make(map[string]interface{})
			current[part] = newMap
			current = newMap
		} else {
			current, ok = next.(map[string]interface{})
			if !ok {
				return fmt.Errorf("cannot set nested key %s: %s is not a map", key, part)
			}
		}
	}

	return nil
}

// formatValueForDisplay formats a value for human-readable display.
func formatValueForDisplay(v interface{}) string {
	if v == nil {
		return "<not set>"
	}
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case []interface{}:
		data, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(data)
	case map[string]interface{}:
		data, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(data)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// Default configuration templates
const defaultConfigTOML = `# Markata-go configuration file

# Site metadata
title = "My Site"
url = "https://example.com"
description = "A site built with markata-go"
author = "Your Name"

# Output settings
output_dir = "output"
templates_dir = "templates"
assets_dir = "static"

# File discovery
[glob]
patterns = ["**/*.md"]
use_gitignore = true

# Post output formats
[post_formats]
html = true
markdown = true
text = true
og = true

# Feed defaults
[feed_defaults]
items_per_page = 10
orphan_threshold = 3

[feed_defaults.formats]
html = true
rss = true
atom = true
json = true
sitemap = true

# Define custom feeds
# [[feeds]]
# slug = "blog"
# title = "Blog Posts"
# filter = "published == true"
# sort = "date"
# reverse = true
`

const defaultConfigYAML = `# Markata-go configuration file

# Site metadata
title: My Site
url: https://example.com
description: A site built with markata-go
author: Your Name

# Output settings
output_dir: output
templates_dir: templates
assets_dir: static

# File discovery
glob:
  patterns:
    - "**/*.md"
  use_gitignore: true

# Post output formats
post_formats:
  html: true
  markdown: true
  text: true
  og: true

# Feed defaults
feed_defaults:
  items_per_page: 10
  orphan_threshold: 3
  formats:
    html: true
    rss: true
    atom: true
    json: true
    sitemap: true

# Define custom feeds
# feeds:
#   - slug: blog
#     title: Blog Posts
#     filter: "published == true"
#     sort: date
#     reverse: true
`

const defaultConfigJSON = `{
  "title": "My Site",
  "url": "https://example.com",
  "description": "A site built with markata-go",
  "author": "Your Name",
  "output_dir": "output",
  "templates_dir": "templates",
  "assets_dir": "static",
  "glob": {
    "patterns": ["**/*.md"],
    "use_gitignore": true
  },
  "post_formats": {
    "html": true,
    "markdown": true,
    "text": true,
    "og": true
  },
  "feed_defaults": {
    "items_per_page": 10,
    "orphan_threshold": 3,
    "formats": {
      "html": true,
      "rss": true,
      "atom": true,
      "json": true,
      "sitemap": true
    }
  },
  "feeds": []
}
`
