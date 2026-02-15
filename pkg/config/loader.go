package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

// Supported config file names in discovery order
var configFileNames = []string{
	"markata-go.toml",
	"markata-go.yaml",
	"markata-go.yml",
	"markata-go.json",
}

// Format represents a configuration file format.
type Format string

const (
	FormatTOML Format = "toml"
	FormatYAML Format = "yaml"
	FormatJSON Format = "json"
)

// ErrConfigNotFound is returned when no config file is found.
var ErrConfigNotFound = errors.New("no configuration file found")

// Load loads configuration from the specified file path.
// If configPath is empty, it will attempt to discover a config file.
// Environment variable overrides are applied after loading the file.
// A .env file in the current directory is loaded first (if present)
// so that encryption keys and other settings can be stored there.
func Load(configPath string) (*models.Config, error) {
	// Load .env file before anything else so env vars are available
	// for both config parsing and encryption key lookups.
	// Errors loading .env are non-fatal (file may not exist).
	_ = LoadDotEnv() //nolint:errcheck // .env loading is best-effort

	var config *models.Config
	var err error

	if configPath == "" {
		// Try to discover a config file
		configPath, err = Discover()
		if err != nil {
			if errors.Is(err, ErrConfigNotFound) {
				// No config file found, use defaults with env overrides
				return LoadWithDefaults()
			}
			return nil, err
		}
	}

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Determine format from extension
	format := formatFromPath(configPath)

	// Parse based on format
	switch format {
	case FormatTOML:
		config, err = ParseTOML(data)
	case FormatYAML:
		config, err = ParseYAML(data)
	case FormatJSON:
		config, err = ParseJSON(data)
	default:
		return nil, fmt.Errorf("unsupported config format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	// Merge with defaults to fill in any missing values
	defaults := DefaultConfig()
	config = MergeConfigs(defaults, config)

	// Apply environment variable overrides
	if err := ApplyEnvOverrides(config); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	return config, nil
}

// Discover searches for a configuration file in the standard locations.
// It returns the path to the first config file found, or ErrConfigNotFound.
//
// Discovery order:
//  1. ./markata-go.toml
//  2. ./markata-go.yaml (or .yml)
//  3. ./markata-go.json
//  4. ~/.config/markata-go/config.toml
func Discover() (string, error) {
	// Check current directory first
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	for _, name := range configFileNames {
		path := filepath.Join(cwd, name)
		if fileExists(path) {
			return path, nil
		}
	}

	// Check user config directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userConfigPath := filepath.Join(homeDir, ".config", "markata-go", "config.toml")
		if fileExists(userConfigPath) {
			return userConfigPath, nil
		}
	}

	return "", ErrConfigNotFound
}

// LoadWithDefaults returns a configuration with default values and
// environment variable overrides applied.
func LoadWithDefaults() (*models.Config, error) {
	config := DefaultConfig()

	if err := ApplyEnvOverrides(config); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	return config, nil
}

// LoadWithMerge loads a base configuration and merges it with one or more
// override configurations. Override configs are applied in order, with later
// configs taking precedence over earlier ones.
//
// This is useful for environment-specific overrides, e.g.:
//   - markata-go.toml (base config)
//   - markata-go.local.toml (local overrides, gitignored)
//   - fast-markata-go.toml (fast build overrides)
//
// Example usage:
//
//	// Load base + local overrides
//	config, err := LoadWithMerge("markata-go.toml", "markata-go.local.toml")
//
//	// Load with multiple overrides
//	config, err := LoadWithMerge("markata-go.toml", "fast-markata-go.toml", "debug.toml")
func LoadWithMerge(basePath string, overridePaths ...string) (*models.Config, error) {
	// Load .env file first
	_ = LoadDotEnv() //nolint:errcheck // .env loading is best-effort

	// Load base config (or use empty config if no base path provided)
	var baseConfig *models.Config
	var err error
	if basePath != "" {
		baseConfig, err = LoadSingleConfig(basePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load base config %s: %w", basePath, err)
		}
	} else {
		baseConfig = &models.Config{}
	}

	// Merge with defaults first
	baseConfig = MergeConfigs(DefaultConfig(), baseConfig)

	// Apply each override config in order
	for _, overridePath := range overridePaths {
		if overridePath == "" {
			continue
		}

		overrideConfig, err := LoadSingleConfig(overridePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load override config %s: %w", overridePath, err)
		}

		// Merge override into base (override takes precedence)
		baseConfig = MergeConfigs(baseConfig, overrideConfig)
	}

	// Apply environment variable overrides last (highest precedence)
	if err := ApplyEnvOverrides(baseConfig); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	return baseConfig, nil
}

// LoadSingleConfig loads a single config file without merging defaults or env overrides.
// This is useful for loading partial configs that will be merged with others.
func LoadSingleConfig(configPath string) (*models.Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	format := formatFromPath(configPath)

	var config *models.Config
	switch format {
	case FormatTOML:
		config, err = ParseTOML(data)
	case FormatYAML:
		config, err = ParseYAML(data)
	case FormatJSON:
		config, err = ParseJSON(data)
	default:
		return nil, fmt.Errorf("unsupported config format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return config, nil
}

// LoadFromString parses configuration from a string with the specified format.
func LoadFromString(data string, format Format) (*models.Config, error) {
	var config *models.Config
	var err error

	switch format {
	case FormatTOML:
		config, err = ParseTOML([]byte(data))
	case FormatYAML:
		config, err = ParseYAML([]byte(data))
	case FormatJSON:
		config, err = ParseJSON([]byte(data))
	default:
		return nil, fmt.Errorf("unsupported config format: %s", format)
	}

	if err != nil {
		return nil, err
	}

	// Merge with defaults
	defaults := DefaultConfig()
	config = MergeConfigs(defaults, config)

	return config, nil
}

// MustLoad loads configuration and panics on error.
// This is useful for initialization code where config loading failure
// should be fatal.
func MustLoad(configPath string) *models.Config {
	config, err := Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return config
}

// LoadAndValidate loads and validates configuration.
// Returns the config and any validation errors/warnings.
func LoadAndValidate(configPath string) (*models.Config, []error, error) {
	config, err := Load(configPath)
	if err != nil {
		return nil, nil, err
	}

	validationErrs := ValidateConfig(config)
	return config, validationErrs, nil
}

// LoadAndValidateWithPositions loads configuration and validates it with
// position tracking for enhanced error messages.
// Returns the config, detailed errors with file positions, and any loading error.
func LoadAndValidateWithPositions(configPath string) (*models.Config, *ConfigErrors, error) {
	var actualPath string
	var err error

	if configPath == "" {
		// Try to discover a config file
		actualPath, err = Discover()
		if err != nil {
			if errors.Is(err, ErrConfigNotFound) {
				// No config file found, use defaults with env overrides
				config, loadErr := LoadWithDefaults()
				if loadErr != nil {
					return nil, nil, loadErr
				}
				// Validate without position tracking since there's no file
				configErrors := ValidateConfigWithPositions(config, nil)
				return config, configErrors, nil
			}
			return nil, nil, err
		}
	} else {
		actualPath = configPath
	}

	// Read the file for both parsing and position tracking
	data, err := os.ReadFile(actualPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read config file %s: %w", actualPath, err)
	}

	// Create position tracker for the config file
	tracker := NewPositionTracker(data, actualPath)

	// Determine format and parse
	format := formatFromPath(actualPath)
	var config *models.Config

	switch format {
	case FormatTOML:
		config, err = ParseTOML(data)
	case FormatYAML:
		config, err = ParseYAML(data)
	case FormatJSON:
		config, err = ParseJSON(data)
	default:
		return nil, nil, fmt.Errorf("unsupported config format: %s", format)
	}

	if err != nil {
		// Create a config error for parse failures
		configErrors := &ConfigErrors{}
		configErrors.Add(&ConfigError{
			File:    actualPath,
			Message: fmt.Sprintf("failed to parse configuration: %v", err),
			Field:   "syntax",
		})
		return nil, configErrors, fmt.Errorf("failed to parse config file %s: %w", actualPath, err)
	}

	// Merge with defaults
	defaults := DefaultConfig()
	config = MergeConfigs(defaults, config)

	// Apply environment variable overrides
	if err := ApplyEnvOverrides(config); err != nil {
		return nil, nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	// Validate with position tracking
	configErrors := ValidateConfigWithPositions(config, tracker)

	return config, configErrors, nil
}

// formatFromPath determines the config format from a file path.
func formatFromPath(path string) Format {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".toml":
		return FormatTOML
	case ".yaml", ".yml":
		return FormatYAML
	case ".json":
		return FormatJSON
	default:
		return FormatTOML // Default to TOML
	}
}

// fileExists returns true if the file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// Path holds the result of config file discovery with additional metadata.
type Path struct {
	Path   string // Full path to the config file
	Format Format // Format of the config file
	Source string // Where it was found: "cli", "cwd", "user"
}

// DiscoverAll finds all config files in the standard locations.
// This is useful for debugging or showing available configs.
func DiscoverAll() []Path {
	var found []Path

	cwd, err := os.Getwd()
	if err == nil {
		for _, name := range configFileNames {
			path := filepath.Join(cwd, name)
			if fileExists(path) {
				found = append(found, Path{
					Path:   path,
					Format: formatFromPath(path),
					Source: "cwd",
				})
			}
		}
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		userConfigPath := filepath.Join(homeDir, ".config", "markata-go", "config.toml")
		if fileExists(userConfigPath) {
			found = append(found, Path{
				Path:   userConfigPath,
				Format: FormatTOML,
				Source: "user",
			})
		}
	}

	return found
}
