package models

// MentionsConfig configures the @mentions resolution plugin.
type MentionsConfig struct {
	// Enabled controls whether mentions processing is active (default: true)
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty" toml:"enabled,omitempty"`

	// CSSClass is the CSS class applied to mention links (default: "mention")
	CSSClass string `json:"css_class,omitempty" yaml:"css_class,omitempty" toml:"css_class,omitempty"`

	// FromPosts configures mention sources from internal posts
	FromPosts []MentionPostSource `json:"from_posts,omitempty" yaml:"from_posts,omitempty" toml:"from_posts,omitempty"`
}

// MentionPostSource configures a source of @mentions from internal posts.
// This allows resolving @handles from posts like contact pages or team member pages.
type MentionPostSource struct {
	// Filter is a filter expression to select which posts to extract handles from
	// Example: "'contact' in tags" or "template == 'team-member.html'"
	Filter string `json:"filter" yaml:"filter" toml:"filter"`

	// HandleField is the frontmatter field containing the handle (default: uses slug)
	// Example: "handle" for frontmatter like `handle: alice`
	HandleField string `json:"handle_field,omitempty" yaml:"handle_field,omitempty" toml:"handle_field,omitempty"`

	// AliasesField is the frontmatter field containing handle aliases (optional)
	// Example: "aliases" for frontmatter like `aliases: [alices, asmith]`
	AliasesField string `json:"aliases_field,omitempty" yaml:"aliases_field,omitempty" toml:"aliases_field,omitempty"`
}

// NewMentionsConfig creates a new MentionsConfig with default values.
func NewMentionsConfig() MentionsConfig {
	enabled := true
	return MentionsConfig{
		Enabled:   &enabled,
		CSSClass:  "mention",
		FromPosts: []MentionPostSource{},
	}
}

// IsEnabled returns whether mentions processing is enabled.
// Defaults to true if not explicitly set.
func (m *MentionsConfig) IsEnabled() bool {
	if m.Enabled == nil {
		return true
	}
	return *m.Enabled
}

// GetCSSClass returns the CSS class for mention links.
// Defaults to "mention" if not set.
func (m *MentionsConfig) GetCSSClass() string {
	if m.CSSClass == "" {
		return "mention"
	}
	return m.CSSClass
}
