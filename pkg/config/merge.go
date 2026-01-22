package config

import (
	"github.com/example/markata-go/pkg/models"
)

// MergeConfigs merges override configuration into base configuration.
// The override values take precedence over base values.
// For nested objects, a deep merge is performed.
// Arrays replace by default (use *_append fields for appending).
func MergeConfigs(base, override *models.Config) *models.Config {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	result := &models.Config{}
	*result = *base

	// String fields - override if non-empty
	if override.OutputDir != "" {
		result.OutputDir = override.OutputDir
	}
	if override.URL != "" {
		result.URL = override.URL
	}
	if override.Title != "" {
		result.Title = override.Title
	}
	if override.Description != "" {
		result.Description = override.Description
	}
	if override.Author != "" {
		result.Author = override.Author
	}
	if override.AssetsDir != "" {
		result.AssetsDir = override.AssetsDir
	}
	if override.TemplatesDir != "" {
		result.TemplatesDir = override.TemplatesDir
	}

	// Slice fields - replace if non-nil and non-empty
	if len(override.Hooks) > 0 {
		result.Hooks = override.Hooks
	}
	if len(override.DisabledHooks) > 0 {
		result.DisabledHooks = override.DisabledHooks
	}

	// Integer fields - override if non-zero
	if override.Concurrency != 0 {
		result.Concurrency = override.Concurrency
	}

	// Nested structs - deep merge
	result.GlobConfig = mergeGlobConfig(base.GlobConfig, override.GlobConfig)
	result.MarkdownConfig = mergeMarkdownConfig(base.MarkdownConfig, override.MarkdownConfig)
	result.FeedDefaults = mergeFeedDefaults(base.FeedDefaults, override.FeedDefaults)

	// Feeds array - replace if non-empty
	if len(override.Feeds) > 0 {
		result.Feeds = override.Feeds
	}

	return result
}

// mergeGlobConfig merges GlobConfig values.
func mergeGlobConfig(base, override models.GlobConfig) models.GlobConfig {
	result := base

	if len(override.Patterns) > 0 {
		result.Patterns = override.Patterns
	}

	// UseGitignore is a bool, always use override value if it differs from zero value
	// Since we can't distinguish between "explicitly set to false" and "not set",
	// we always take the override value
	result.UseGitignore = override.UseGitignore

	return result
}

// mergeMarkdownConfig merges MarkdownConfig values.
func mergeMarkdownConfig(base, override models.MarkdownConfig) models.MarkdownConfig {
	result := base

	if len(override.Extensions) > 0 {
		result.Extensions = override.Extensions
	}

	return result
}

// mergeFeedDefaults merges FeedDefaults values.
func mergeFeedDefaults(base, override models.FeedDefaults) models.FeedDefaults {
	result := base

	if override.ItemsPerPage != 0 {
		result.ItemsPerPage = override.ItemsPerPage
	}
	if override.OrphanThreshold != 0 {
		result.OrphanThreshold = override.OrphanThreshold
	}

	result.Formats = mergeFeedFormats(base.Formats, override.Formats)
	result.Templates = mergeFeedTemplates(base.Templates, override.Templates)
	result.Syndication = mergeSyndicationConfig(base.Syndication, override.Syndication)

	return result
}

// mergeFeedFormats merges FeedFormats values.
// If the override has any format enabled, it replaces the base entirely.
// This allows explicitly disabling formats by setting only the desired ones.
func mergeFeedFormats(base, override models.FeedFormats) models.FeedFormats {
	// Check if override has any format set to true
	if override.HTML || override.RSS || override.Atom || override.JSON || override.Markdown || override.Text {
		// Override is "active" - use it entirely
		return override
	}
	// Override has no formats enabled, keep base
	return base
}

// mergeFeedTemplates merges FeedTemplates values.
func mergeFeedTemplates(base, override models.FeedTemplates) models.FeedTemplates {
	result := base

	if override.HTML != "" {
		result.HTML = override.HTML
	}
	if override.RSS != "" {
		result.RSS = override.RSS
	}
	if override.Atom != "" {
		result.Atom = override.Atom
	}
	if override.JSON != "" {
		result.JSON = override.JSON
	}
	if override.Card != "" {
		result.Card = override.Card
	}

	return result
}

// mergeSyndicationConfig merges SyndicationConfig values.
func mergeSyndicationConfig(base, override models.SyndicationConfig) models.SyndicationConfig {
	result := base

	if override.MaxItems != 0 {
		result.MaxItems = override.MaxItems
	}
	// For IncludeContent, we take the override value
	result.IncludeContent = override.IncludeContent || base.IncludeContent

	return result
}

// MergeSlice merges two slices with optional append behavior.
// If append is true, the override slice is appended to the base slice.
// Otherwise, the override slice replaces the base slice.
func MergeSlice[T any](base, override []T, appendMode bool) []T {
	if len(override) == 0 {
		return base
	}
	if appendMode {
		result := make([]T, len(base)+len(override))
		copy(result, base)
		copy(result[len(base):], override)
		return result
	}
	return override
}

// AppendHooks appends hooks to the configuration's Hooks slice.
func AppendHooks(config *models.Config, hooks ...string) {
	config.Hooks = MergeSlice(config.Hooks, hooks, true)
}

// AppendDisabledHooks appends hooks to the configuration's DisabledHooks slice.
func AppendDisabledHooks(config *models.Config, hooks ...string) {
	config.DisabledHooks = MergeSlice(config.DisabledHooks, hooks, true)
}

// AppendGlobPatterns appends patterns to the configuration's GlobConfig.Patterns slice.
func AppendGlobPatterns(config *models.Config, patterns ...string) {
	config.GlobConfig.Patterns = MergeSlice(config.GlobConfig.Patterns, patterns, true)
}

// AppendFeeds appends feeds to the configuration's Feeds slice.
func AppendFeeds(config *models.Config, feeds ...models.FeedConfig) {
	config.Feeds = MergeSlice(config.Feeds, feeds, true)
}
