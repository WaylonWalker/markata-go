package services

import "time"

// SortOrder defines the sort direction.
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// ListOptions configures list operations.
type ListOptions struct {
	// Filter is a filter expression (same syntax as lifecycle.Manager.Filter)
	Filter string

	// Tags filters posts by tags (AND logic - post must have all tags)
	Tags []string

	// DateRange filters posts by date range
	DateRange *DateRange

	// Published filters by published status (nil = all)
	Published *bool

	// Draft filters by draft status (nil = all)
	Draft *bool

	// SortBy is the field to sort by (e.g., "date", "title")
	SortBy string

	// SortOrder is the sort direction (default: desc for date, asc for title)
	SortOrder SortOrder

	// Offset is the number of items to skip (for pagination)
	Offset int

	// Limit is the maximum number of items to return (0 = no limit)
	Limit int
}

// DateRange defines a date range filter.
type DateRange struct {
	Start *time.Time
	End   *time.Time
}

// SearchOptions configures search operations.
type SearchOptions struct {
	// Fields to search in (default: title, description, content)
	Fields []string

	// CaseSensitive enables case-sensitive search
	CaseSensitive bool

	// Fuzzy enables fuzzy matching
	Fuzzy bool

	// Limit is the maximum number of results (0 = no limit)
	Limit int
}

// TagInfo represents a tag with metadata.
type TagInfo struct {
	// Name is the tag name
	Name string

	// Count is the number of posts with this tag
	Count int

	// Slug is the URL-safe version of the tag
	Slug string
}

// BuildOptions configures build operations.
type BuildOptions struct {
	// Watch enables watch mode for continuous rebuilding
	Watch bool

	// Serve enables the development server
	Serve bool

	// Port is the server port (default: 8080)
	Port int

	// Clean removes output directory before building
	Clean bool

	// Concurrency sets parallel processing level
	Concurrency int
}

// BuildResult contains the result of a build operation.
type BuildResult struct {
	// Success indicates if the build completed successfully
	Success bool

	// Duration is the build time
	Duration time.Duration

	// PostsProcessed is the number of posts processed
	PostsProcessed int

	// FilesWritten is the number of files written
	FilesWritten int

	// Errors contains any errors that occurred
	Errors []error

	// Warnings contains non-critical warnings
	Warnings []string
}

// BuildEvent represents a build progress event.
type BuildEvent struct {
	// Type is the event type
	Type BuildEventType

	// Stage is the current build stage
	Stage string

	// Message is the event message
	Message string

	// Progress is the progress percentage (0-100)
	Progress int

	// Error is the error if Type is BuildEventError
	Error error
}

// BuildEventType identifies the type of build event.
type BuildEventType string

const (
	BuildEventStart    BuildEventType = "start"
	BuildEventStage    BuildEventType = "stage"
	BuildEventProgress BuildEventType = "progress"
	BuildEventComplete BuildEventType = "complete"
	BuildEventError    BuildEventType = "error"
)
