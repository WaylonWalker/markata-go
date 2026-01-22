package models

// Link represents a hyperlink found in a post's HTML content.
// It tracks both the source (where the link is from) and target (where it points to).
type Link struct {
	// SourceURL is the absolute URL of the source post
	SourceURL string `json:"source_url" yaml:"source_url" toml:"source_url"`

	// SourcePost is a reference to the source post object
	SourcePost *Post `json:"-" yaml:"-" toml:"-"`

	// TargetPost is a reference to the target post (nil if external)
	TargetPost *Post `json:"-" yaml:"-" toml:"-"`

	// RawTarget is the original href value as found in the HTML
	RawTarget string `json:"raw_target" yaml:"raw_target" toml:"raw_target"`

	// TargetURL is the resolved absolute URL
	TargetURL string `json:"target_url" yaml:"target_url" toml:"target_url"`

	// TargetDomain is the domain extracted from target_url
	TargetDomain string `json:"target_domain" yaml:"target_domain" toml:"target_domain"`

	// IsInternal is true if the link points to the same site
	IsInternal bool `json:"is_internal" yaml:"is_internal" toml:"is_internal"`

	// IsSelf is true if the link points to the same post
	IsSelf bool `json:"is_self" yaml:"is_self" toml:"is_self"`

	// SourceText is the cleaned link text from the source anchor
	SourceText string `json:"source_text" yaml:"source_text" toml:"source_text"`

	// TargetText is the cleaned link text from the target (if available)
	TargetText string `json:"target_text" yaml:"target_text" toml:"target_text"`
}

// NewLink creates a new Link with the given source and target information.
func NewLink(sourceURL, rawTarget, targetURL, targetDomain string, isInternal bool) *Link {
	return &Link{
		SourceURL:    sourceURL,
		RawTarget:    rawTarget,
		TargetURL:    targetURL,
		TargetDomain: targetDomain,
		IsInternal:   isInternal,
		IsSelf:       false,
	}
}

// SourceSlug returns the slug of the source post, or empty string if nil.
func (l *Link) SourceSlug() string {
	if l.SourcePost == nil {
		return ""
	}
	return l.SourcePost.Slug
}

// TargetSlug returns the slug of the target post, or empty string if nil.
func (l *Link) TargetSlug() string {
	if l.TargetPost == nil {
		return ""
	}
	return l.TargetPost.Slug
}

// SourceTitle returns the title of the source post, or empty string if nil.
func (l *Link) SourceTitle() string {
	if l.SourcePost == nil || l.SourcePost.Title == nil {
		return ""
	}
	return *l.SourcePost.Title
}

// TargetTitle returns the title of the target post, or empty string if nil.
func (l *Link) TargetTitle() string {
	if l.TargetPost == nil || l.TargetPost.Title == nil {
		return ""
	}
	return *l.TargetPost.Title
}
