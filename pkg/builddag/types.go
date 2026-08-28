package builddag

import (
	"context"
	"fmt"
	"strings"
)

// TaskID identifies a task.
type TaskID string

// ArtifactID identifies an artifact by kind and key.
type ArtifactID struct{ Kind, Key string }

// String returns the canonical, stable representation of an artifact ID.
func (id ArtifactID) String() string { return id.Kind + ":" + id.Key }

// ParseArtifactID parses the representation produced by String.
func ParseArtifactID(s string) (ArtifactID, error) {
	kind, key, ok := strings.Cut(s, ":")
	if !ok || kind == "" || key == "" || strings.Contains(key, ":") {
		return ArtifactID{}, fmt.Errorf("invalid artifact ID %q", s)
	}
	return ArtifactID{Kind: kind, Key: key}, nil
}

// Scope limits the ownership of a task or artifact.
type Scope string

const (
	ScopeSite     Scope = "site"
	ScopePost     Scope = "post"
	ScopeFeed     Scope = "feed"
	ScopeArtifact Scope = "artifact"
)

// Artifact is a value stored between tasks. Bytes are copied by the memory store.
type Artifact struct {
	ID   ArtifactID `json:"id"`
	Data []byte     `json:"data,omitempty"`
}

// ArtifactStore is the executor's small persistence boundary.
type ArtifactStore interface {
	Get(ArtifactID) (Artifact, bool)
	Put(Artifact)
}

// TaskContext exposes only the declared inputs and artifact store to a task.
type TaskContext struct {
	Store  ArtifactStore
	Inputs map[ArtifactID]Artifact
}

func (c TaskContext) Get(id ArtifactID) (Artifact, bool) { a, ok := c.Inputs[id]; return a, ok }

// TaskFunc executes a task serially.
type TaskFunc func(context.Context, TaskContext) (TaskResult, error)

// TaskSpec declares one computation boundary.
type TaskSpec struct {
	ID       TaskID       `json:"id"`
	Group    string       `json:"group,omitempty"`
	Requires []ArtifactID `json:"requires,omitempty"`
	Provides []ArtifactID `json:"provides,omitempty"`
	Scope    Scope        `json:"scope,omitempty"`
	Version  string       `json:"version,omitempty"`
	// Exclusive marks compatibility tasks that may touch legacy mutable state.
	// It is metadata in the serial executor and a scheduling constraint for a
	// future executor.  Legacy adapters set it to true conservatively.
	Exclusive bool `json:"exclusive,omitempty"`
	// ParallelSafe documents whether a task is safe to run concurrently.  It is
	// deliberately not used to enable concurrency in the first executor.
	ParallelSafe bool     `json:"parallel_safe,omitempty"`
	Func         TaskFunc `json:"-"`
}

// TaskResult is the output of a task, including dependencies discovered at runtime.
type TaskResult struct {
	Artifacts   []Artifact   `json:"artifacts"`
	DynamicDeps []ArtifactID `json:"dynamic_deps,omitempty"`
}
