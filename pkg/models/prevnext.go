package models

// PrevNextContext contains navigation context for a post.
type PrevNextContext struct {
	// FeedSlug is the feed or series slug used for navigation
	FeedSlug string `json:"feed_slug" yaml:"feed_slug" toml:"feed_slug"`

	// FeedTitle is the feed or series title
	FeedTitle string `json:"feed_title" yaml:"feed_title" toml:"feed_title"`

	// Position is the 1-indexed position of the post in the sequence
	Position int `json:"position" yaml:"position" toml:"position"`

	// Total is the total number of posts in the sequence
	Total int `json:"total" yaml:"total" toml:"total"`

	// Prev is the previous post in the sequence (nil if first)
	Prev *Post `json:"prev,omitempty" yaml:"prev,omitempty" toml:"prev,omitempty"`

	// Next is the next post in the sequence (nil if last)
	Next *Post `json:"next,omitempty" yaml:"next,omitempty" toml:"next,omitempty"`
}

// HasPrev returns true if there is a previous post.
func (c *PrevNextContext) HasPrev() bool {
	return c.Prev != nil
}

// HasNext returns true if there is a next post.
func (c *PrevNextContext) HasNext() bool {
	return c.Next != nil
}

// IsFirst returns true if this is the first post in the sequence.
func (c *PrevNextContext) IsFirst() bool {
	return c.Position == 1
}

// IsLast returns true if this is the last post in the sequence.
func (c *PrevNextContext) IsLast() bool {
	return c.Position == c.Total
}

// PrevNextStrategy defines how prev/next links are calculated.
type PrevNextStrategy string

const (
	// StrategyFirstFeed uses the first feed the post appears in.
	StrategyFirstFeed PrevNextStrategy = "first_feed"

	// StrategyExplicitFeed always uses the configured default_feed.
	StrategyExplicitFeed PrevNextStrategy = "explicit_feed"

	// StrategySeries uses the post's series frontmatter, falling back to first_feed.
	StrategySeries PrevNextStrategy = "series"

	// StrategyFrontmatter uses the post's prevnext_feed frontmatter, falling back to first_feed.
	StrategyFrontmatter PrevNextStrategy = "frontmatter"
)

// IsValid returns true if the strategy is a recognized value.
func (s PrevNextStrategy) IsValid() bool {
	switch s {
	case StrategyFirstFeed, StrategyExplicitFeed, StrategySeries, StrategyFrontmatter:
		return true
	}
	return false
}

// PrevNextConfig holds configuration for the prevnext plugin.
type PrevNextConfig struct {
	// Enabled controls whether the plugin is active (default: true)
	Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`

	// Strategy determines how prev/next links are resolved
	Strategy PrevNextStrategy `json:"strategy" yaml:"strategy" toml:"strategy"`

	// DefaultFeed is the feed slug to use when strategy is "explicit_feed"
	DefaultFeed string `json:"default_feed" yaml:"default_feed" toml:"default_feed"`
}

// NewPrevNextConfig creates a new PrevNextConfig with default values.
func NewPrevNextConfig() PrevNextConfig {
	return PrevNextConfig{
		Enabled:     true,
		Strategy:    StrategyFirstFeed,
		DefaultFeed: "",
	}
}

// ApplyDefaults ensures all required fields have sensible defaults.
func (c *PrevNextConfig) ApplyDefaults() {
	if c.Strategy == "" {
		c.Strategy = StrategyFirstFeed
	}
}
