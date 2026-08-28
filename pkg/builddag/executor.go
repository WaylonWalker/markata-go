package builddag

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sort"
)

// TaskState retains a task result, including dynamic dependency edges.
type TaskState struct {
	Result       TaskResult        `json:"result"`
	Completed    bool              `json:"completed"`
	Version      string            `json:"version,omitempty"`
	InputDigests map[string]string `json:"input_digests,omitempty"`
}

// ExecutionState is serializable scheduler state.
type ExecutionState struct {
	SchemaVersion int                  `json:"schema_version"`
	GraphDigest   string               `json:"graph_digest"`
	Tasks         map[TaskID]TaskState `json:"tasks"`
}

// Executor runs a graph serially. MaxParallel values above one are rejected.
type Executor struct {
	MaxParallel    int
	Seed           int64
	RandomizeReady bool
}

func NewExecutor(maxParallel int) (*Executor, error) {
	if maxParallel == 0 {
		maxParallel = 1
	}
	if maxParallel != 1 {
		return nil, fmt.Errorf("builddag: MaxParallel must be 1, got %d", maxParallel)
	}
	return &Executor{MaxParallel: 1}, nil
}

//nolint:gocyclo // Scheduling, cache validation, and task result validation form one atomic execution loop.
func (e *Executor) Execute(ctx context.Context, g *Graph, store ArtifactStore, state *ExecutionState) (*ExecutionState, error) {
	if e == nil || e.MaxParallel != 1 {
		return nil, fmt.Errorf("builddag: serial executor requires MaxParallel=1")
	}
	if state == nil {
		state = &ExecutionState{SchemaVersion: 1, Tasks: map[TaskID]TaskState{}}
	}
	if state.Tasks == nil {
		state.Tasks = map[TaskID]TaskState{}
	}
	if state.SchemaVersion == 0 {
		state.SchemaVersion = 1
	}
	if g == nil {
		return nil, fmt.Errorf("builddag: graph is nil")
	}
	if store == nil {
		return nil, fmt.Errorf("builddag: artifact store is nil")
	}
	graphDigest, err := g.Digest()
	if err != nil {
		return nil, fmt.Errorf("builddag: graph digest: %w", err)
	}
	if state.GraphDigest != "" && state.GraphDigest != graphDigest {
		state.Tasks = map[TaskID]TaskState{}
	}
	state.GraphDigest = graphDigest
	//nolint:gosec // A seeded PRNG is intentional for reproducible ready-queue exploration.
	rng := rand.New(rand.NewSource(e.Seed))
	done := map[TaskID]bool{}
	for len(done) < len(g.tasks) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		ready := []TaskID{}
		for _, id := range g.order {
			if done[id] {
				continue
			}
			ok := true
			for d := range g.deps[id] {
				if !done[d] {
					ok = false
				}
			}
			if ok {
				ready = append(ready, id)
			}
		}
		if len(ready) == 0 {
			return nil, fmt.Errorf("builddag: no ready task")
		}
		if e.RandomizeReady && len(ready) > 1 {
			ready = []TaskID{ready[rng.Intn(len(ready))]}
		}
		id := ready[0]
		t := g.tasks[id]
		inputs, err := requiredInputs(t, store)
		if err != nil {
			return nil, err
		}
		if cached, ok := state.Tasks[id]; ok && cached.Completed &&
			cached.Version == t.Version &&
			inputDigestsMatch(cached.InputDigests, inputs) &&
			validateTaskResult(t, cached.Result) == nil {
			for _, a := range cached.Result.Artifacts {
				store.Put(a)
			}
			done[id] = true
			continue
		}
		delete(state.Tasks, id)
		if t.Func == nil {
			return nil, fmt.Errorf("task %q has no function", id)
		}
		result, err := t.Func(ctx, TaskContext{Store: store, Inputs: inputs})
		if err != nil {
			return nil, fmt.Errorf("task %q: %w", id, err)
		}
		if err := validateTaskResult(t, result); err != nil {
			return nil, fmt.Errorf("task %q: %w", id, err)
		}
		for _, a := range result.Artifacts {
			store.Put(a)
		}
		state.Tasks[id] = TaskState{
			Result: result, Completed: true, Version: t.Version,
			InputDigests: digestInputs(inputs),
		}
		done[id] = true
	}
	return state, nil
}

func requiredInputs(t TaskSpec, store ArtifactStore) (map[ArtifactID]Artifact, error) {
	inputs := make(map[ArtifactID]Artifact, len(t.Requires))
	for _, id := range t.Requires {
		value, ok := store.Get(id)
		if !ok {
			return nil, fmt.Errorf("task %q requires missing artifact %s", t.ID, id)
		}
		inputs[id] = value
	}
	return inputs, nil
}

func validateTaskResult(t TaskSpec, result TaskResult) error {
	provided := make(map[ArtifactID]bool, len(result.Artifacts))
	for _, a := range result.Artifacts {
		if a.ID.Kind == "" || a.ID.Key == "" {
			return fmt.Errorf("returned invalid artifact %q", a.ID)
		}
		if !containsArtifact(t.Provides, a.ID) {
			return fmt.Errorf("returned undeclared artifact %s", a.ID)
		}
		if provided[a.ID] {
			return fmt.Errorf("returned artifact %s more than once", a.ID)
		}
		provided[a.ID] = true
	}
	for _, a := range t.Provides {
		if !provided[a] {
			return fmt.Errorf("did not return provided artifact %s", a)
		}
	}
	seenDeps := make(map[ArtifactID]bool, len(result.DynamicDeps))
	for _, dep := range result.DynamicDeps {
		if dep.Kind == "" || dep.Key == "" {
			return fmt.Errorf("returned invalid dynamic dependency %q", dep)
		}
		if seenDeps[dep] {
			return fmt.Errorf("returned dynamic dependency %s more than once", dep)
		}
		seenDeps[dep] = true
	}
	return nil
}

func digestInputs(inputs map[ArtifactID]Artifact) map[string]string {
	digests := make(map[string]string, len(inputs))
	for id, artifact := range inputs {
		digests[id.String()] = digestArtifact(artifact)
	}
	return digests
}

func inputDigestsMatch(want map[string]string, inputs map[ArtifactID]Artifact) bool {
	got := digestInputs(inputs)
	if len(want) != len(got) {
		return false
	}
	for id, digest := range got {
		if want[id] != digest {
			return false
		}
	}
	return true
}

func digestArtifact(artifact Artifact) string {
	h := sha256.New()
	_, _ = h.Write([]byte(artifact.ID.String()))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(artifact.Data)
	return hex.EncodeToString(h.Sum(nil))
}

func containsArtifact(ids []ArtifactID, want ArtifactID) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

// Invalidate returns the transitive task closure affected by changed artifacts.
func (s *ExecutionState) Invalidate(g *Graph, changed []ArtifactID) []TaskID {
	if s == nil || g == nil {
		return nil
	}
	affected := map[TaskID]bool{}
	queue := []TaskID{}
	for _, a := range changed {
		if id, ok := g.provider[a]; ok {
			queue = append(queue, id)
		}
		queue = append(queue, g.consumers[a]...)
		for id, st := range s.Tasks {
			for _, d := range st.Result.DynamicDeps {
				if d == a {
					queue = append(queue, id)
				}
			}
		}
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if affected[id] {
			continue
		}
		affected[id] = true
		queue = append(queue, g.reverse[id]...)
	}
	out := make([]TaskID, 0, len(affected))
	for id := range affected {
		out = append(out, id)
	}
	sortTaskIDs(out)
	for _, id := range out {
		delete(s.Tasks, id)
	}
	return out
}
func sortTaskIDs(a []TaskID) {
	sort.Slice(a, func(i, j int) bool { return a[i] < a[j] })
}
