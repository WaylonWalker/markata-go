package builddag

import (
	"container/heap"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
)

// Builder collects task declarations and explicitly declared external inputs.
type Builder struct {
	tasks    []TaskSpec
	external map[ArtifactID]bool
}

// NewBuilder creates an empty graph builder.
func NewBuilder() *Builder {
	return &Builder{external: make(map[ArtifactID]bool)}
}

// AddTask appends a task declaration. Compile performs validation.
func (b *Builder) AddTask(task TaskSpec) {
	b.tasks = append(b.tasks, task)
}

// AddExternal declares an artifact supplied outside the graph.
func (b *Builder) AddExternal(id ArtifactID) {
	b.external[id] = true
}

// Graph is an immutable compiled task graph.
type Graph struct {
	tasks    map[TaskID]TaskSpec
	deps     map[TaskID]map[TaskID]bool
	order    []TaskID
	external map[ArtifactID]bool
}

// Compile validates declarations and returns an immutable graph.
func (b *Builder) Compile() (*Graph, error) {
	tasks := append([]TaskSpec(nil), b.tasks...)
	for i := range tasks {
		tasks[i] = cloneTask(tasks[i])
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })

	byID := make(map[TaskID]TaskSpec, len(tasks))
	providers := make(map[ArtifactID]TaskID)
	for i := range tasks {
		task := &tasks[i]
		if task.ID == "" {
			return nil, fmt.Errorf("builddag: task ID is empty")
		}
		if _, exists := byID[task.ID]; exists {
			return nil, fmt.Errorf("builddag: duplicate task ID %q", task.ID)
		}
		if err := validateTaskResources(*task); err != nil {
			return nil, fmt.Errorf("builddag: task %q: %w", task.ID, err)
		}
		byID[task.ID] = *task

		for _, artifact := range task.Provides {
			if !validArtifact(artifact) {
				return nil, fmt.Errorf("builddag: task %q provides invalid artifact %q", task.ID, artifact.String())
			}
			if previous, exists := providers[artifact]; exists {
				return nil, fmt.Errorf(
					"builddag: duplicate provider for artifact %s: tasks %q and %q",
					artifact.String(), previous, task.ID,
				)
			}
			providers[artifact] = task.ID
		}
	}

	deps := make(map[TaskID]map[TaskID]bool, len(tasks))
	for i := range tasks {
		task := &tasks[i]
		dependencies := make(map[TaskID]bool)
		for _, artifact := range task.Requires {
			if !validArtifact(artifact) {
				return nil, fmt.Errorf("builddag: task %q requires invalid artifact %q", task.ID, artifact.String())
			}
			if provider, exists := providers[artifact]; exists {
				dependencies[provider] = true
				continue
			}
			if !b.external[artifact] {
				return nil, fmt.Errorf(
					"builddag: task %q requires missing provider for artifact %s",
					task.ID, artifact.String(),
				)
			}
		}
		deps[task.ID] = dependencies
	}

	order, err := topologicalOrder(tasks, deps)
	if err != nil {
		return nil, err
	}

	external := make(map[ArtifactID]bool, len(b.external))
	for id := range b.external {
		external[id] = true
	}

	return &Graph{
		tasks:    byID,
		deps:     deps,
		order:    order,
		external: external,
	}, nil
}

func validateTaskResources(task TaskSpec) error {
	if task.Exclusive && task.ParallelSafe {
		return fmt.Errorf("exclusive task cannot be parallel-safe")
	}
	if task.ParallelSafe && len(task.Resources) == 0 {
		return fmt.Errorf("parallel-safe task must declare resource claims")
	}

	seen := make(map[ResourceID]AccessMode, len(task.Resources))
	for _, claim := range task.Resources {
		if !validResourceClaim(claim) {
			return fmt.Errorf("invalid resource claim %q", claim.String())
		}
		if previous, exists := seen[claim.Resource]; exists {
			return fmt.Errorf(
				"resource %s declared more than once (%s and %s)",
				claim.Resource.String(), previous, claim.Access,
			)
		}
		seen[claim.Resource] = claim.Access
	}
	return nil
}

func validArtifact(id ArtifactID) bool {
	return id.Kind != "" && id.Key != ""
}

func topologicalOrder(tasks []TaskSpec, deps map[TaskID]map[TaskID]bool) ([]TaskID, error) {
	remaining := make(map[TaskID]int, len(tasks))
	children := make(map[TaskID][]TaskID, len(tasks))
	ready := make(taskIDHeap, 0, len(tasks))

	for i := range tasks {
		id := tasks[i].ID
		remaining[id] = len(deps[id])
		for dependency := range deps[id] {
			children[dependency] = append(children[dependency], id)
		}
		if remaining[id] == 0 {
			ready = append(ready, id)
		}
	}
	heap.Init(&ready)

	order := make([]TaskID, 0, len(tasks))
	for len(ready) > 0 {
		id, ok := heap.Pop(&ready).(TaskID)
		if !ok {
			return nil, fmt.Errorf("builddag: invalid task ID in ready heap")
		}
		order = append(order, id)

		for _, child := range children[id] {
			remaining[child]--
			if remaining[child] == 0 {
				heap.Push(&ready, child)
			}
		}
	}

	if len(order) != len(tasks) {
		return nil, fmt.Errorf("builddag: task dependency cycle: %v", findCycle(tasks, deps))
	}
	return order, nil
}

// taskIDHeap preserves the smallest-ready-task policy without sorting the
// entire frontier after each task. Children need not be sorted before pushing.
type taskIDHeap []TaskID

func (h taskIDHeap) Len() int           { return len(h) }
func (h taskIDHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h taskIDHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *taskIDHeap) Push(value any) {
	id, ok := value.(TaskID)
	if !ok {
		panic("builddag: invalid task ID pushed to ready heap")
	}
	*h = append(*h, id)
}
func (h *taskIDHeap) Pop() any {
	last := len(*h) - 1
	id := (*h)[last]
	(*h)[last] = ""
	*h = (*h)[:last]
	return id
}

func findCycle(tasks []TaskSpec, deps map[TaskID]map[TaskID]bool) []TaskID {
	state := make(map[TaskID]uint8, len(tasks))
	stack := make([]TaskID, 0, len(tasks))

	var visit func(TaskID) []TaskID
	visit = func(id TaskID) []TaskID {
		state[id] = 1
		stack = append(stack, id)

		dependencies := make([]TaskID, 0, len(deps[id]))
		for dependency := range deps[id] {
			dependencies = append(dependencies, dependency)
		}
		sort.Slice(dependencies, func(i, j int) bool { return dependencies[i] < dependencies[j] })

		for _, dependency := range dependencies {
			switch state[dependency] {
			case 0:
				if cycle := visit(dependency); cycle != nil {
					return cycle
				}
			case 1:
				for i, current := range stack {
					if current == dependency {
						cycle := append([]TaskID(nil), stack[i:]...)
						return append(cycle, dependency)
					}
				}
			}
		}

		stack = stack[:len(stack)-1]
		state[id] = 2
		return nil
	}

	for i := range tasks {
		id := tasks[i].ID
		if state[id] != 0 {
			continue
		}
		if cycle := visit(id); cycle != nil {
			return cycle
		}
	}
	return nil
}

// Order returns the deterministic topological task order.
func (g *Graph) Order() []TaskID {
	if g == nil {
		return nil
	}
	return append([]TaskID(nil), g.order...)
}

// Task returns a task declaration by ID with independently owned metadata
// slices. It does not copy state captured by the task's function.
func (g *Graph) Task(id TaskID) (TaskSpec, bool) {
	if g == nil {
		return TaskSpec{}, false
	}
	task, ok := g.tasks[id]
	return cloneTask(task), ok
}

func cloneTask(task TaskSpec) TaskSpec {
	task.Requires = slices.Clone(task.Requires)
	task.Provides = slices.Clone(task.Provides)
	task.Resources = slices.Clone(task.Resources)
	return task
}

// Serialize returns the stable graph declaration without executable functions.
func (g *Graph) Serialize() ([]byte, error) {
	if g == nil {
		return nil, fmt.Errorf("builddag: graph is nil")
	}

	type entry struct {
		ID           TaskID          `json:"id"`
		Group        string          `json:"group,omitempty"`
		Requires     []ArtifactID    `json:"requires,omitempty"`
		Provides     []ArtifactID    `json:"provides,omitempty"`
		Resources    []ResourceClaim `json:"resources,omitempty"`
		Scope        Scope           `json:"scope,omitempty"`
		Version      string          `json:"version,omitempty"`
		Exclusive    bool            `json:"exclusive,omitempty"`
		ParallelSafe bool            `json:"parallel_safe,omitempty"`
	}

	entries := make([]entry, 0, len(g.order))
	for _, id := range g.order {
		task := g.tasks[id]
		requires := append([]ArtifactID(nil), task.Requires...)
		provides := append([]ArtifactID(nil), task.Provides...)
		resources := append([]ResourceClaim(nil), task.Resources...)
		sort.Slice(requires, func(i, j int) bool { return artifactLess(requires[i], requires[j]) })
		sort.Slice(provides, func(i, j int) bool { return artifactLess(provides[i], provides[j]) })
		sort.Slice(resources, func(i, j int) bool { return resourceClaimLess(resources[i], resources[j]) })
		entries = append(entries, entry{
			ID:           task.ID,
			Group:        task.Group,
			Requires:     requires,
			Provides:     provides,
			Resources:    resources,
			Scope:        task.Scope,
			Version:      task.Version,
			Exclusive:    task.Exclusive,
			ParallelSafe: task.ParallelSafe,
		})
	}

	external := make([]ArtifactID, 0, len(g.external))
	for id := range g.external {
		external = append(external, id)
	}
	sort.Slice(external, func(i, j int) bool { return artifactLess(external[i], external[j]) })

	return json.Marshal(struct {
		External []ArtifactID `json:"external,omitempty"`
		Tasks    []entry      `json:"tasks"`
	}{
		External: external,
		Tasks:    entries,
	})
}

// Digest returns a stable SHA-256 digest of the graph declaration.
func (g *Graph) Digest() (string, error) {
	serialized, err := g.Serialize()
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(serialized)
	return hex.EncodeToString(digest[:]), nil
}
