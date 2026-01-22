// Package config provides configuration loading and management for markata-go.
//
// # Configuration File Formats
//
// The package supports three configuration formats:
//   - TOML (primary, recommended) using github.com/BurntSushi/toml
//   - YAML using gopkg.in/yaml.v3
//   - JSON using the standard library
//
// All formats use a top-level "markata-go" key to namespace configuration:
//
//	# TOML example
//	[markata-go]
//	output_dir = "public"
//	url = "https://example.com"
//
//	# YAML example
//	markata-go:
//	  output_dir: public
//	  url: https://example.com
//
//	# JSON example
//	{"markata-go": {"output_dir": "public", "url": "https://example.com"}}
//
// # Configuration Discovery
//
// When no explicit config path is provided, the package searches for
// configuration files in the following order:
//
//  1. ./markata-go.toml
//  2. ./markata-go.yaml
//  3. ./markata-go.yml
//  4. ./markata-go.json
//  5. ~/.config/markata-go/config.toml
//
// If no configuration file is found, default values are used.
//
// # Environment Variable Overrides
//
// Configuration values can be overridden using environment variables with
// the MARKATA_GO_ prefix. Environment variables take precedence over file
// configuration.
//
// Simple fields:
//
//	MARKATA_GO_OUTPUT_DIR=public
//	MARKATA_GO_URL=https://example.com
//
// Nested fields use underscores:
//
//	MARKATA_GO_GLOB_PATTERNS=posts/**/*.md,pages/*.md
//	MARKATA_GO_FEED_DEFAULTS_ITEMS_PER_PAGE=20
//
// Boolean values accept: "true", "1", "yes" for true; "false", "0", "no" for false.
//
// List values are comma-separated:
//
//	MARKATA_GO_HOOKS=markdown,template,sitemap
//
// # Usage
//
// Basic usage:
//
//	// Load from default locations with env overrides
//	config, err := config.Load("")
//
//	// Load from specific file
//	config, err := config.Load("/path/to/config.toml")
//
//	// Load with defaults only
//	config, err := config.LoadWithDefaults()
//
//	// Load and validate
//	config, validationErrs, err := config.LoadAndValidate("/path/to/config.toml")
//
// # Merging Configurations
//
// The MergeConfigs function performs a deep merge of two configurations:
//
//	base := config.DefaultConfig()
//	override := &models.Config{OutputDir: "custom"}
//	merged := config.MergeConfigs(base, override)
//
// String fields are overridden if non-empty. Integer fields are overridden if non-zero.
// Slices are replaced if non-empty. Feed formats are replaced if any format is enabled.
//
// # Validation
//
// Configurations can be validated to catch common errors and warnings:
//
//	errs := config.ValidateConfig(cfg)
//	if config.HasErrors(errs) {
//	    // Handle errors
//	}
//	if config.HasWarnings(errs) {
//	    // Handle warnings
//	}
//
// Validation checks include:
//   - URL format validation
//   - Concurrency must be >= 0
//   - Feed slugs are required
//   - Warning on empty glob patterns
//   - Warning on feeds with no output formats
package config
