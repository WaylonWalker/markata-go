// Package contentindex defines the versioned Markata Content Index format.
package contentindex

import "time"

const (
	Schema         = "markata.content-index"
	SchemaURL      = "markata://schemas/content-index/v1"
	CurrentVersion = 1
	GeneratorName  = "markata-go"
)

// Index is the normalized internal representation of a Content Index.
// Readers for released wire formats must normalize into this type.
type Index struct {
	Schema        string
	SchemaVersion int
	Generator     Generator
	Source        Source
	DocumentCount int
	Documents     []Document
}

type Generator struct{ Name, Version string }

// Source identifies the repository state described by the index. Commit is
// empty when the source directory is not a Git checkout or HEAD is unavailable.
type Source struct{ Commit string }

// Document contains compact, derived metadata. It never contains article body.
type Document struct {
	Path        string
	Slug        string
	Href        string
	Title       *string
	TitleText   *string
	Date        *time.Time
	Modified    *time.Time
	Published   bool
	Draft       bool
	Private     bool
	Template    string
	Tags        []string
	Description *string
	Feeds       []string
	Aliases     []string
}
