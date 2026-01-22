// Package plugins provides lifecycle plugins for markata-go.
package plugins

// String constants used throughout the plugins package.
// These constants help avoid magic strings and satisfy goconst linter.
const (
	// BoolTrue is the string representation of true.
	BoolTrue = "true"

	// AdmonitionTypeAside is the aside admonition type.
	AdmonitionTypeAside = "aside"

	// PositionLeft is the left position value.
	PositionLeft = "left"

	// PositionStart is the start position value.
	PositionStart = "start"

	// PositionEnd is the end position value.
	PositionEnd = "end"

	// PluginNameTemplates is the templates plugin name.
	PluginNameTemplates = "templates"

	// ThemeDefault is the default theme name.
	ThemeDefault = "default"

	// DefaultSiteURL is the default site URL used when none is configured.
	DefaultSiteURL = "https://example.com"
)
