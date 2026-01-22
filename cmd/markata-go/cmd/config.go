package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/example/markata-go/pkg/config"
	"github.com/example/markata-go/pkg/models"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// configCmd represents the config command group.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration commands",
	Long: `Commands for managing markata-go configuration.

Subcommands:
  show     - Display the resolved configuration
  get      - Get a specific configuration value
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
  markata-go config show           # Show as YAML
  markata-go config show --json    # Show as JSON
  markata-go config show --toml    # Show as TOML`,
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

var (
	// configFormat specifies the output format for config show.
	configFormat string

	// configInitForce overwrites existing config file.
	configInitForce bool
)

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configInitCmd)

	configShowCmd.Flags().StringVar(&configFormat, "format", "yaml", "output format (yaml, json, toml)")
	configShowCmd.Flags().Bool("json", false, "output as JSON (shorthand for --format=json)")
	configShowCmd.Flags().Bool("toml", false, "output as TOML (shorthand for --format=toml)")

	configInitCmd.Flags().BoolVar(&configInitForce, "force", false, "overwrite existing file")
}

func runConfigShowCommand(cmd *cobra.Command, args []string) error {
	// Handle format shorthands
	if jsonFlag, _ := cmd.Flags().GetBool("json"); jsonFlag {
		configFormat = "json"
	}
	if tomlFlag, _ := cmd.Flags().GetBool("toml"); tomlFlag {
		configFormat = "toml"
	}

	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Output in requested format
	switch strings.ToLower(configFormat) {
	case "json":
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))

	case "toml":
		if err := toml.NewEncoder(os.Stdout).Encode(cfg); err != nil {
			return fmt.Errorf("failed to marshal TOML: %w", err)
		}

	case "yaml":
		fallthrough
	default:
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		fmt.Print(string(data))
	}

	return nil
}

func runConfigGetCommand(cmd *cobra.Command, args []string) error {
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
	switch strings.ToLower(key) {
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
	}

	// Handle nested keys
	parts := strings.SplitN(key, ".", 2)
	switch strings.ToLower(parts[0]) {
	case "glob":
		if len(parts) == 1 {
			return cfg.GlobConfig, nil
		}
		switch strings.ToLower(parts[1]) {
		case "patterns":
			return cfg.GlobConfig.Patterns, nil
		case "use_gitignore":
			return cfg.GlobConfig.UseGitignore, nil
		}
	case "markdown":
		if len(parts) == 1 {
			return cfg.MarkdownConfig, nil
		}
		switch strings.ToLower(parts[1]) {
		case "extensions":
			return cfg.MarkdownConfig.Extensions, nil
		}
	case "feed_defaults":
		if len(parts) == 1 {
			return cfg.FeedDefaults, nil
		}
		switch strings.ToLower(parts[1]) {
		case "items_per_page":
			return cfg.FeedDefaults.ItemsPerPage, nil
		case "orphan_threshold":
			return cfg.FeedDefaults.OrphanThreshold, nil
		}
	case "feeds":
		return cfg.Feeds, nil
	}

	return nil, fmt.Errorf("unknown config key: %s", key)
}

func runConfigValidateCommand(cmd *cobra.Command, args []string) error {
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
		configPath, _ = config.Discover()
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

func runConfigInitCommand(cmd *cobra.Command, args []string) error {
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
	case strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml"):
		content = defaultConfigYAML
	case strings.HasSuffix(filename, ".json"):
		content = defaultConfigJSON
	default:
		content = defaultConfigTOML
	}

	// Write file
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Created: %s\n", filename)
	return nil
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

# Feed defaults
[feed_defaults]
items_per_page = 10
orphan_threshold = 3

[feed_defaults.formats]
html = true
rss = true
atom = false
json = false

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

# Feed defaults
feed_defaults:
  items_per_page: 10
  orphan_threshold: 3
  formats:
    html: true
    rss: true
    atom: false
    json: false

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
  "feed_defaults": {
    "items_per_page": 10,
    "orphan_threshold": 3,
    "formats": {
      "html": true,
      "rss": true,
      "atom": false,
      "json": false
    }
  },
  "feeds": []
}
`
