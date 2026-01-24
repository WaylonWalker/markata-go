// Package importer provides functionality to import content from external sources.
//
// This package implements the PESOS (Publish Elsewhere, Syndicate to Own Site)
// pattern, allowing users to import their content from various external sources
// like RSS feeds and JSON Feeds into their markata-go site.
//
// # Supported Sources
//
// The package currently supports:
//   - RSS/Atom feeds via the RSSImporter
//   - JSON Feed format via the JSONFeedImporter
//
// # Usage
//
//	importer, err := NewRSSImporter(url)
//	if err != nil {
//	    return err
//	}
//	posts, err := importer.Import(opts)
package importer

import (
	"time"
)

// ImportedPost represents a post imported from an external source.
type ImportedPost struct {
	// ID is a unique identifier from the source
	ID string `json:"id" yaml:"id"`

	// SourceURL is the original URL of the post
	SourceURL string `json:"source_url" yaml:"source_url"`

	// SourceType indicates the type of source (e.g., "rss", "jsonfeed")
	SourceType string `json:"source_type" yaml:"source_type"`

	// Title is the post title
	Title string `json:"title" yaml:"title"`

	// Content is the post content (may be HTML or plain text)
	Content string `json:"content" yaml:"content"`

	// ContentHTML is the HTML content if available
	ContentHTML string `json:"content_html,omitempty" yaml:"content_html,omitempty"`

	// Published is when the post was originally published
	Published time.Time `json:"published" yaml:"published"`

	// Updated is when the post was last updated (if available)
	Updated *time.Time `json:"updated,omitempty" yaml:"updated,omitempty"`

	// Imported is when the post was imported
	Imported time.Time `json:"imported" yaml:"imported"`

	// Author is the post author
	Author string `json:"author,omitempty" yaml:"author,omitempty"`

	// Tags are any tags or categories from the source
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// Summary is a short summary or description
	Summary string `json:"summary,omitempty" yaml:"summary,omitempty"`

	// Slug is the URL-safe slug for the post
	Slug string `json:"slug" yaml:"slug"`
}

// ImportOptions configures the import operation.
type ImportOptions struct {
	// Since filters posts to only include those published after this time
	Since *time.Time

	// DryRun if true, previews imports without writing files
	DryRun bool

	// OutputDir is the directory to write imported posts
	OutputDir string

	// AddTags are additional tags to add to all imported posts
	AddTags []string
}

// Importer defines the interface for importing content from external sources.
type Importer interface {
	// Name returns the name of the importer (e.g., "rss", "jsonfeed")
	Name() string

	// Import fetches and returns posts from the source
	Import(opts ImportOptions) ([]*ImportedPost, error)

	// SourceURL returns the URL being imported from
	SourceURL() string
}

// ImportResult holds the result of an import operation.
type ImportResult struct {
	// Posts are the successfully imported posts
	Posts []*ImportedPost

	// Skipped is the number of posts skipped (e.g., already imported, filtered by date)
	Skipped int

	// Errors contains any non-fatal errors encountered
	Errors []error
}
