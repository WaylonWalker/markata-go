package builddag

import "context"

// TaskID identifies a task in a compiled build graph.
type TaskID string

// ArtifactID identifies an artifact flowing between tasks.
type ArtifactID struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`
}

// String returns a stable human-readable artifact identifier.
func (id ArtifactID) String() string {
	return id.Kind + ":" + id.Key
}

// artifactLess compares identity fields, not the potentially ambiguous display
// string (kind and key may themselves contain colons).
func artifactLess(a, b ArtifactID) bool {
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	return a.Key < b.Key
}

// Scope describes the ownership boundary of a task.
type Scope string

const (
	ScopeSite     Scope = "site"
	ScopePost     Scope = "post"
	ScopeFeed     Scope = "feed"
	ScopeArtifact Scope = "artifact"
)

// TaskFunc performs one task. The first executor is intentionally serial.
type TaskFunc func(context.Context) error

// TaskSpec declares one computation boundary, its artifact dependencies, and
// any mutable/shared resources it reads or writes.
type TaskSpec struct {
	ID           TaskID          `json:"id"`
	Group        string          `json:"group,omitempty"`
	Requires     []ArtifactID    `json:"requires,omitempty"`
	Provides     []ArtifactID    `json:"provides,omitempty"`
	Resources    []ResourceClaim `json:"resources,omitempty"`
	Scope        Scope           `json:"scope,omitempty"`
	Version      string          `json:"version,omitempty"`
	Exclusive    bool            `json:"exclusive,omitempty"`
	ParallelSafe bool            `json:"parallel_safe,omitempty"`
	Func         TaskFunc        `json:"-"`
}
