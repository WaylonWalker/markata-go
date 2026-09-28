package builddag

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

func validArtifact(id ArtifactID) bool {
	return id.Kind != "" && id.Key != ""
}

func topologicalOrder(tasks []TaskSpec, deps map[TaskID]map[TaskID]bool) ([]TaskID, error) {
	remaining := make(map[TaskID]int, len(tasks))
	ready := make([]TaskID, 0, len(tasks))

	for i := range tasks {
		id := tasks[i].ID
		remaining[id] = len(deps[id])
		if remaining[id] == 0 {
			ready = append(ready, id)
		}
	}

	order := make([]TaskID, 0, len(tasks))
	for len(ready) > 0 {
		sort.Slice(ready, func(i, j int) bool { return ready[i] < ready[j] })
		id := ready[0]
		ready = ready[1:]
		order = append(order, id)

		for i := range tasks {
			child := tasks[i].ID
			if !deps[child][id] {
				continue
			}
			remaining[child]--
			if remaining[child] == 0 {
				ready = append(ready, child)
			}
		}
	}

	if len(order) != len(tasks) {
		return nil, fmt.Errorf("builddag: task dependency cycle: %v", findCycle(tasks, deps))
	}
	return order, nil
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

// Task returns a task declaration by ID.
func (g *Graph) Task(id TaskID) (TaskSpec, bool) {
	if g == nil {
		return TaskSpec{}, false
	}
	task, ok := g.tasks[id]
	return task, ok
}

// Serialize returns the stable graph declaration without executable functions.
func (g *Graph) Serialize() ([]byte, error) {
	if g == nil {
		return nil, fmt.Errorf("builddag: graph is nil")
	}

	type entry struct {
		ID           TaskID       `json:"id"`
		Group        string       `json:"group,omitempty"`
		Requires     []ArtifactID `json:"requires,omitempty"`
		Provides     []ArtifactID `json:"provides,omitempty"`
		Scope        Scope        `json:"scope,omitempty"`
		Version      string       `json:"version,omitempty"`
		Exclusive    bool         `json:"exclusive,omitempty"`
		ParallelSafe bool         `json:"parallel_safe,omitempty"`
	}

	entries := make([]entry, 0, len(g.order))
	for _, id := range g.order {
		task := g.tasks[id]
		requires := append([]ArtifactID(nil), task.Requires...)
		provides := append([]ArtifactID(nil), task.Provides...)
		sort.Slice(requires, func(i, j int) bool { return requires[i].String() < requires[j].String() })
		sort.Slice(provides, func(i, j int) bool { return provides[i].String() < provides[j].String() })
		entries = append(entries, entry{
			ID:           task.ID,
			Group:        task.Group,
			Requires:     requires,
			Provides:     provides,
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
	sort.Slice(external, func(i, j int) bool { return external[i].String() < external[j].String() })

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
