package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// Format constants for config file types.
const (
	formatTOML = "toml"
	formatYAML = "yaml"
	formatJSON = "json"
)

// MigrationResult contains the results of a migration operation.
type MigrationResult struct {
	// InputFile is the source config file path
	InputFile string

	// OutputFile is the target config file path
	OutputFile string

	// Changes is the list of configuration changes made
	Changes []ConfigChange

	// FilterMigrations is the list of filter expression migrations
	FilterMigrations []FilterMigration

	// Warnings is the list of non-blocking issues
	Warnings []Warning

	// Errors is the list of blocking issues
	Errors []MigrationError

	// TemplateIssues is the list of template compatibility issues
	TemplateIssues []TemplateIssue

	// MigratedConfig is the resulting configuration (as generic map)
	MigratedConfig map[string]interface{}

	// Timestamp when migration was performed
	Timestamp time.Time
}

// ConfigChange represents a single configuration change.
type ConfigChange struct {
	// Type is the change type: "namespace", "rename", "transform", "remove"
	Type string

	// Path is the config path (e.g., "markata.nav")
	Path string

	// OldValue is the original value
	OldValue interface{}

	// NewValue is the migrated value
	NewValue interface{}

	// Description explains the change
	Description string
}

// FilterMigration represents a filter expression migration.
type FilterMigration struct {
	// Feed is the feed name this filter belongs to
	Feed string

	// Original is the original filter expression
	Original string

	// Migrated is the migrated filter expression
	Migrated string

	// Changes lists specific transformations applied
	Changes []string

	// Valid indicates if the migrated filter is valid
	Valid bool

	// Error contains any migration error
	Error string
}

// Warning represents a non-blocking migration issue.
type Warning struct {
	// Category groups related warnings
	Category string // "config", "filter", "template", "plugin"

	// Message describes the warning
	Message string

	// Path is the config path or file path
	Path string

	// Suggestion provides actionable guidance
	Suggestion string
}

// MigrationError represents a blocking migration issue.
type MigrationError struct {
	// Category groups related errors
	Category string

	// Message describes the error
	Message string

	// Path is the config path or file path
	Path string

	// Fatal indicates if migration cannot continue
	Fatal bool
}

// TemplateIssue represents a template compatibility issue.
type TemplateIssue struct {
	// File is the template file path
	File string

	// Line is the line number
	Line int

	// Issue describes the compatibility issue
	Issue string

	// Severity is "error", "warning", or "info"
	Severity string

	// Suggestion provides fix guidance
	Suggestion string
}

// HasErrors returns true if there are any migration errors.
func (r *MigrationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if there are any warnings.
func (r *MigrationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// ExitCode returns the appropriate exit code for the migration result.
func (r *MigrationResult) ExitCode() int {
	if r.HasErrors() {
		for _, err := range r.Errors {
			if err.Fatal {
				return 3
			}
		}
		return 2
	}
	if r.HasWarnings() {
		return 1
	}
	return 0
}

// Config migrates a Python markata config file to markata-go format.
// If outputPath is empty, it returns the result without writing.
func Config(inputPath, outputPath string) (*MigrationResult, error) {
	result := &MigrationResult{
		InputFile:  inputPath,
		OutputFile: outputPath,
		Timestamp:  time.Now(),
	}

	// Read input file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Determine format from extension
	format := detectFormat(inputPath)

	// Parse config
	var rawConfig map[string]interface{}
	switch format {
	case formatTOML:
		if _, err := toml.Decode(string(data), &rawConfig); err != nil {
			return nil, fmt.Errorf("failed to parse TOML: %w", err)
		}
	case formatYAML:
		if err := yaml.Unmarshal(data, &rawConfig); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config format: %s", format)
	}

	// Migrate the configuration
	migratedConfig := migrateConfigMap(rawConfig, result)
	result.MigratedConfig = migratedConfig

	// Write output if path provided
	if outputPath != "" {
		outFormat := detectFormat(outputPath)
		if err := writeConfig(migratedConfig, outputPath, outFormat); err != nil {
			return result, fmt.Errorf("failed to write config: %w", err)
		}
	}

	return result, nil
}

// ConfigFromMap migrates a config map directly (useful for testing).
func ConfigFromMap(rawConfig map[string]interface{}) (*MigrationResult, error) {
	result := &MigrationResult{
		Timestamp: time.Now(),
	}

	migratedConfig := migrateConfigMap(rawConfig, result)
	result.MigratedConfig = migratedConfig

	return result, nil
}

// migrateConfigMap performs the actual config migration.
func migrateConfigMap(rawConfig map[string]interface{}, result *MigrationResult) map[string]interface{} {
	migrated := make(map[string]interface{})

	// Look for markata namespace (could be "markata" or "tool.markata")
	var markataConfig map[string]interface{}
	var found bool

	// Check for [markata]
	if mc, ok := rawConfig["markata"].(map[string]interface{}); ok {
		markataConfig = mc
		found = true
		result.Changes = append(result.Changes, ConfigChange{
			Type:        "namespace",
			Path:        "markata",
			OldValue:    "markata",
			NewValue:    "markata-go",
			Description: "Namespace change from [markata] to [markata-go]",
		})
	}

	// Check for [tool.markata] (pyproject.toml style)
	if tool, ok := rawConfig["tool"].(map[string]interface{}); ok {
		if mc, ok := tool["markata"].(map[string]interface{}); ok {
			markataConfig = mc
			found = true
			result.Changes = append(result.Changes, ConfigChange{
				Type:        "namespace",
				Path:        "tool.markata",
				OldValue:    "tool.markata",
				NewValue:    "markata-go",
				Description: "Namespace change from [tool.markata] to [markata-go]",
			})
		}
	}

	if !found {
		result.Warnings = append(result.Warnings, Warning{
			Category:   "config",
			Message:    "No [markata] or [tool.markata] section found",
			Path:       "",
			Suggestion: "Ensure your config has a [markata] section",
		})
		return migrated
	}

	// Migrate config under markata-go namespace
	migratedSection := migrateSection(markataConfig, "", result)
	migrated["markata-go"] = migratedSection

	return migrated
}

// migrateSection migrates a configuration section.
func migrateSection(section map[string]interface{}, path string, result *MigrationResult) map[string]interface{} {
	migrated := make(map[string]interface{})

	for key, value := range section {
		fullPath := key
		if path != "" {
			fullPath = path + "." + key
		}

		// Handle key renames
		newKey, renamed := renameKey(key)
		if renamed {
			result.Changes = append(result.Changes, ConfigChange{
				Type:        "rename",
				Path:        fullPath,
				OldValue:    key,
				NewValue:    newKey,
				Description: fmt.Sprintf("Key renamed from '%s' to '%s'", key, newKey),
			})
		}

		// Handle special transformations
		switch key {
		case "nav":
			// Convert nav map to array
			if navMap, ok := value.(map[string]interface{}); ok {
				navArray := migrateNavMap(navMap)
				migrated[newKey] = navArray
				result.Changes = append(result.Changes, ConfigChange{
					Type:        "transform",
					Path:        fullPath,
					OldValue:    navMap,
					NewValue:    navArray,
					Description: fmt.Sprintf("Nav converted from map to array (%d items)", len(navArray)),
				})
				continue
			}
		case "feeds":
			// Migrate feed filters
			if feedsArray, ok := value.([]interface{}); ok {
				migratedFeeds := migrateFeedsArray(feedsArray, result)
				migrated[newKey] = migratedFeeds
				continue
			}
			if feedsMap, ok := value.(map[string]interface{}); ok {
				migratedFeeds := migrateFeedsMap(feedsMap, result)
				migrated[newKey] = migratedFeeds
				continue
			}
		case "hooks":
			// Check for unsupported hooks
			if hooks, ok := value.([]interface{}); ok {
				checkUnsupportedHooks(hooks, result)
			}
		}

		// Handle nested sections
		if nested, ok := value.(map[string]interface{}); ok {
			migrated[newKey] = migrateSection(nested, fullPath, result)
		} else {
			migrated[newKey] = value
		}
	}

	return migrated
}

// renameKey returns the new key name and whether it was renamed.
func renameKey(key string) (string, bool) {
	renames := map[string]string{
		"glob_patterns":    "patterns",
		"author_name":      "author",
		"site_name":        "title",
		"site_description": "description",
		"output":           "output_dir",
		"color_theme":      "palette",
	}

	if newKey, ok := renames[key]; ok {
		return newKey, true
	}
	return key, false
}

// migrateNavMap converts a Python markata nav map to markata-go nav array.
func migrateNavMap(navMap map[string]interface{}) []map[string]interface{} {
	navArray := make([]map[string]interface{}, 0, len(navMap))
	caser := cases.Title(language.English)

	for label, url := range navMap {
		navItem := map[string]interface{}{
			"label": caser.String(label),
			"url":   url,
		}
		navArray = append(navArray, navItem)
	}

	return navArray
}

// migrateFeedsArray migrates a feeds array.
func migrateFeedsArray(feeds []interface{}, result *MigrationResult) []interface{} {
	var migratedFeeds []interface{}

	for _, feed := range feeds {
		if feedMap, ok := feed.(map[string]interface{}); ok {
			migratedFeed := migrateFeedConfig(feedMap, result)
			migratedFeeds = append(migratedFeeds, migratedFeed)
		} else {
			migratedFeeds = append(migratedFeeds, feed)
		}
	}

	return migratedFeeds
}

// migrateFeedsMap migrates a feeds map (where keys are feed names).
func migrateFeedsMap(feeds map[string]interface{}, result *MigrationResult) []interface{} {
	var migratedFeeds []interface{}

	for name, feedConfig := range feeds {
		if feedMap, ok := feedConfig.(map[string]interface{}); ok {
			feedMap["slug"] = name
			migratedFeed := migrateFeedConfig(feedMap, result)
			migratedFeeds = append(migratedFeeds, migratedFeed)
		}
	}

	return migratedFeeds
}

// migrateFeedConfig migrates a single feed configuration.
func migrateFeedConfig(feed map[string]interface{}, result *MigrationResult) map[string]interface{} {
	migrated := make(map[string]interface{})

	for key, value := range feed {
		if key == "filter" {
			if filterStr, ok := value.(string); ok {
				feedName := ""
				if name, ok := feed["slug"].(string); ok {
					feedName = name
				} else if name, ok := feed["name"].(string); ok {
					feedName = name
				}

				migratedFilter, changes := Filter(filterStr)
				migrated[key] = migratedFilter

				filterMigration := FilterMigration{
					Feed:     feedName,
					Original: filterStr,
					Migrated: migratedFilter,
					Changes:  changes,
					Valid:    true,
				}

				// Validate the migrated filter
				if err := ValidateFilter(migratedFilter); err != nil {
					filterMigration.Valid = false
					filterMigration.Error = err.Error()
				}

				result.FilterMigrations = append(result.FilterMigrations, filterMigration)
			}
		} else {
			migrated[key] = value
		}
	}

	return migrated
}

// checkUnsupportedHooks checks for hooks that aren't supported in markata-go.
func checkUnsupportedHooks(hooks []interface{}, result *MigrationResult) {
	unsupported := []string{
		"rich_output",
		"console",
		"custom_python",
	}

	for _, hook := range hooks {
		hookName, ok := hook.(string)
		if !ok {
			continue
		}

		for _, u := range unsupported {
			if hookName == u {
				result.Warnings = append(result.Warnings, Warning{
					Category:   "plugin",
					Message:    fmt.Sprintf("Hook '%s' is not supported in markata-go", hookName),
					Path:       "hooks",
					Suggestion: "Remove this hook or find an alternative",
				})
			}
		}
	}
}

// detectFormat determines the config format from file extension.
func detectFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".toml":
		return formatTOML
	case ".yaml", ".yml":
		return formatYAML
	case ".json":
		return formatJSON
	default:
		return formatTOML
	}
}

// writeConfig writes the config to a file in the specified format.
func writeConfig(config map[string]interface{}, path, format string) error {
	var data []byte
	var err error

	switch format {
	case formatTOML:
		var buf strings.Builder
		enc := toml.NewEncoder(&buf)
		enc.Indent = ""
		if err := enc.Encode(config); err != nil {
			return fmt.Errorf("failed to encode TOML: %w", err)
		}
		data = []byte(buf.String())
	case formatYAML:
		data, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to encode YAML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	return os.WriteFile(path, data, 0o600)
}
