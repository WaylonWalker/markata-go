package builddag

import (
	"encoding/json"
	"fmt"
	"sort"
	"unicode/utf8"
)

const (
	// GraphReportSchema identifies the stable graph inspection representation.
	GraphReportSchema = "markata.builddag-report"
	// GraphReportVersion is the current graph report schema generation.
	GraphReportVersion = 1

	defaultReportMaxTasks        = 256
	defaultReportMaxItemsPerTask = 64
	defaultReportMaxValueBytes   = 256
)

// ReportOptions bounds diagnostic graph output. Zero or negative values use
// conservative defaults so callers cannot accidentally emit an unbounded
// report for a large site.
type ReportOptions struct {
	MaxTasks        int
	MaxItemsPerTask int
	MaxValueBytes   int
}

// GraphReport is a deterministic, non-executable representation of a compiled
// graph suitable for local diagnostics and support tooling.
type GraphReport struct {
	Schema            string       `json:"schema"`
	Version           int          `json:"version"`
	Digest            string       `json:"digest"`
	TaskCount         int          `json:"task_count"`
	ReportedTaskCount int          `json:"reported_task_count"`
	Truncated         bool         `json:"truncated"`
	Order             []TaskID     `json:"order"`
	Tasks             []TaskReport `json:"tasks"`
}

// TaskReport captures the scheduling metadata for one task. Empty resource
// claims are emitted explicitly so compatibility tasks are never mistaken for
// tasks with proven ownership.
type TaskReport struct {
	ID             TaskID          `json:"id"`
	Group          string          `json:"group,omitempty"`
	Scope          Scope           `json:"scope,omitempty"`
	Version        string          `json:"task_version,omitempty"`
	Requires       []ArtifactID    `json:"requires"`
	Provides       []ArtifactID    `json:"provides"`
	Resources      []ResourceClaim `json:"resources"`
	Exclusive      bool            `json:"exclusive"`
	ParallelSafe   bool            `json:"parallel_safe"`
	ItemsTruncated bool            `json:"items_truncated"`
}

// Report returns a bounded deterministic representation of the compiled
// graph. It never includes task functions or other executable values.
func (g *Graph) Report(options ReportOptions) (GraphReport, error) {
	if g == nil {
		return GraphReport{}, fmt.Errorf("builddag: graph is nil")
	}

	options = normalizeReportOptions(options)
	digest, err := g.Digest()
	if err != nil {
		return GraphReport{}, fmt.Errorf("builddag: graph report digest: %w", err)
	}

	order := g.Order()
	limit := len(order)
	if limit > options.MaxTasks {
		limit = options.MaxTasks
	}

	report := GraphReport{
		Schema:            GraphReportSchema,
		Version:           GraphReportVersion,
		Digest:            digest,
		TaskCount:         len(order),
		ReportedTaskCount: limit,
		Truncated:         limit < len(order),
		Order:             make([]TaskID, 0, limit),
		Tasks:             make([]TaskReport, 0, limit),
	}

	for _, id := range order[:limit] {
		task, ok := g.Task(id)
		if !ok {
			return GraphReport{}, fmt.Errorf("builddag: task %q missing from compiled graph", id)
		}
		reported, itemsTruncated := reportTask(task, options)
		report.Order = append(report.Order, TaskID(boundReportValue(string(id), options.MaxValueBytes)))
		reported.ItemsTruncated = itemsTruncated
		report.Tasks = append(report.Tasks, reported)
	}
	return report, nil
}

// MarshalReport encodes Report as stable indented JSON.
func (g *Graph) MarshalReport(options ReportOptions) ([]byte, error) {
	report, err := g.Report(options)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(report, "", "  ")
}

func normalizeReportOptions(options ReportOptions) ReportOptions {
	if options.MaxTasks <= 0 {
		options.MaxTasks = defaultReportMaxTasks
	}
	if options.MaxItemsPerTask <= 0 {
		options.MaxItemsPerTask = defaultReportMaxItemsPerTask
	}
	if options.MaxValueBytes <= 0 {
		options.MaxValueBytes = defaultReportMaxValueBytes
	}
	return options
}

func reportTask(task TaskSpec, options ReportOptions) (TaskReport, bool) {
	requires, requiresTruncated := reportArtifacts(task.Requires, options)
	provides, providesTruncated := reportArtifacts(task.Provides, options)
	resources, resourcesTruncated := reportResources(task.Resources, options)

	return TaskReport{
		ID:           TaskID(boundReportValue(string(task.ID), options.MaxValueBytes)),
		Group:        boundReportValue(task.Group, options.MaxValueBytes),
		Scope:        Scope(boundReportValue(string(task.Scope), options.MaxValueBytes)),
		Version:      boundReportValue(task.Version, options.MaxValueBytes),
		Requires:     requires,
		Provides:     provides,
		Resources:    resources,
		Exclusive:    task.Exclusive,
		ParallelSafe: task.ParallelSafe,
	}, requiresTruncated || providesTruncated || resourcesTruncated
}

func reportArtifacts(artifacts []ArtifactID, options ReportOptions) ([]ArtifactID, bool) {
	items := append([]ArtifactID(nil), artifacts...)
	sort.Slice(items, func(i, j int) bool { return items[i].String() < items[j].String() })
	truncated := len(items) > options.MaxItemsPerTask
	if truncated {
		items = items[:options.MaxItemsPerTask]
	}
	result := make([]ArtifactID, len(items))
	for index, item := range items {
		result[index] = ArtifactID{
			Kind: boundReportValue(item.Kind, options.MaxValueBytes),
			Key:  boundReportValue(item.Key, options.MaxValueBytes),
		}
	}
	return result, truncated
}

func reportResources(resources []ResourceClaim, options ReportOptions) ([]ResourceClaim, bool) {
	items := append([]ResourceClaim(nil), resources...)
	sort.Slice(items, func(i, j int) bool { return items[i].String() < items[j].String() })
	truncated := len(items) > options.MaxItemsPerTask
	if truncated {
		items = items[:options.MaxItemsPerTask]
	}
	result := make([]ResourceClaim, len(items))
	for index, item := range items {
		result[index] = ResourceClaim{
			Resource: ResourceID{
				Kind: ResourceKind(boundReportValue(string(item.Resource.Kind), options.MaxValueBytes)),
				Key:  boundReportValue(item.Resource.Key, options.MaxValueBytes),
			},
			Access: AccessMode(boundReportValue(string(item.Access), options.MaxValueBytes)),
		}
	}
	return result, truncated
}

func boundReportValue(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}

	suffix := "..."
	limit := maxBytes
	if maxBytes > len(suffix) {
		limit = maxBytes - len(suffix)
	} else {
		suffix = ""
	}
	for limit > 0 && !utf8.ValidString(value[:limit]) {
		limit--
	}
	return value[:limit] + suffix
}
