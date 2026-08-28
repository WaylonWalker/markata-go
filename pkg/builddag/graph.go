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

func NewBuilder() *Builder                   { return &Builder{external: make(map[ArtifactID]bool)} }
func (b *Builder) AddTask(t TaskSpec)        { b.tasks = append(b.tasks, t) }
func (b *Builder) AddExternal(id ArtifactID) { b.external[id] = true }
func (b *Builder) Compile() (*Graph, error) {
	tasks := append([]TaskSpec(nil), b.tasks...)
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	byID := make(map[TaskID]TaskSpec, len(tasks))
	provider := map[ArtifactID]TaskID{}
	for i := range tasks {
		t := tasks[i]
		if t.ID == "" {
			return nil, fmt.Errorf("task ID is empty")
		}
		if _, ok := byID[t.ID]; ok {
			return nil, fmt.Errorf("duplicate task ID %q", t.ID)
		}
		byID[t.ID] = t
		for _, a := range t.Provides {
			if a.Kind == "" || a.Key == "" {
				return nil, fmt.Errorf("task %q provides invalid artifact %q", t.ID, a.String())
			}
			if p, ok := provider[a]; ok {
				return nil, fmt.Errorf("duplicate provider for artifact %s: tasks %q and %q", a, p, t.ID)
			}
			provider[a] = t.ID
		}
	}
	deps := make(map[TaskID]map[TaskID]bool)
	reverse := make(map[TaskID][]TaskID)
	consumers := make(map[ArtifactID][]TaskID)
	for i := range tasks {
		t := tasks[i]
		deps[t.ID] = map[TaskID]bool{}
		for _, a := range t.Requires {
			if a.Kind == "" || a.Key == "" {
				return nil, fmt.Errorf("task %q requires invalid artifact %q", t.ID, a.String())
			}
			if p, ok := provider[a]; ok {
				deps[t.ID][p] = true
			} else if !b.external[a] {
				return nil, fmt.Errorf("task %q requires missing provider for artifact %s (declare it external)", t.ID, a)
			}
			consumers[a] = append(consumers[a], t.ID)
		}
	}
	for id, ds := range deps {
		for d := range ds {
			reverse[d] = append(reverse[d], id)
		}
	}
	order, err := topo(tasks, deps)
	if err != nil {
		return nil, err
	}
	external := make(map[ArtifactID]bool, len(b.external))
	for id := range b.external {
		external[id] = true
	}
	return &Graph{tasks: byID, provider: provider, deps: deps, reverse: reverse, consumers: consumers, order: order, external: external}, nil
}
func topo(tasks []TaskSpec, deps map[TaskID]map[TaskID]bool) ([]TaskID, error) {
	left := map[TaskID]int{}
	for i := range tasks {
		t := tasks[i]
		left[t.ID] = len(deps[t.ID])
	}
	ready := []TaskID{}
	for i := range tasks {
		t := tasks[i]
		if left[t.ID] == 0 {
			ready = append(ready, t.ID)
		}
	}
	out := []TaskID{}
	for len(ready) > 0 {
		sort.Slice(ready, func(i, j int) bool { return ready[i] < ready[j] })
		id := ready[0]
		ready = ready[1:]
		out = append(out, id)
		for child, ds := range deps {
			if ds[id] {
				left[child]--
				if left[child] == 0 {
					ready = append(ready, child)
				}
			}
		}
	}
	if len(out) != len(tasks) {
		cycle := findCycle(tasks, deps)
		return nil, fmt.Errorf("task dependency cycle: %v", cycle)
	}
	return out, nil
}
func findCycle(tasks []TaskSpec, deps map[TaskID]map[TaskID]bool) []TaskID {
	state := map[TaskID]int{}
	path := []TaskID{}
	var walk func(TaskID) []TaskID
	walk = func(id TaskID) []TaskID {
		state[id] = 1
		path = append(path, id)
		dependencies := make([]TaskID, 0, len(deps[id]))
		for d := range deps[id] {
			dependencies = append(dependencies, d)
		}
		sort.Slice(dependencies, func(i, j int) bool { return dependencies[i] < dependencies[j] })
		for _, d := range dependencies {
			if state[d] == 1 {
				for i, v := range path {
					if v == d {
						return append(path[i:], d)
					}
				}
			}
			if state[d] == 0 {
				if c := walk(d); c != nil {
					return c
				}
			}
		}
		path = path[:len(path)-1]
		state[id] = 2
		return nil
	}
	for i := range tasks {
		t := tasks[i]
		if state[t.ID] == 0 {
			if c := walk(t.ID); c != nil {
				return c
			}
		}
	}
	return nil
}

// Graph is an immutable compiled task graph.
type Graph struct {
	tasks     map[TaskID]TaskSpec
	provider  map[ArtifactID]TaskID
	deps      map[TaskID]map[TaskID]bool
	reverse   map[TaskID][]TaskID
	consumers map[ArtifactID][]TaskID
	order     []TaskID
	external  map[ArtifactID]bool
}

func (g *Graph) Order() []TaskID                 { return append([]TaskID(nil), g.order...) }
func (g *Graph) Task(id TaskID) (TaskSpec, bool) { t, ok := g.tasks[id]; return t, ok }
func (g *Graph) Serialize() ([]byte, error) {
	type entry struct {
		ID           TaskID   `json:"id"`
		Group        string   `json:"group,omitempty"`
		Requires     []string `json:"requires,omitempty"`
		Provides     []string `json:"provides,omitempty"`
		Scope        Scope    `json:"scope,omitempty"`
		Version      string   `json:"version,omitempty"`
		Exclusive    bool     `json:"exclusive,omitempty"`
		ParallelSafe bool     `json:"parallel_safe,omitempty"`
	}
	e := make([]entry, 0, len(g.order))
	for _, id := range g.order {
		t := g.tasks[id]
		r := ids(t.Requires)
		p := ids(t.Provides)
		sort.Strings(r)
		sort.Strings(p)
		e = append(e, entry{ID: t.ID, Group: t.Group, Requires: r, Provides: p,
			Scope: t.Scope, Version: t.Version, Exclusive: t.Exclusive,
			ParallelSafe: t.ParallelSafe})
	}
	external := make([]string, 0, len(g.external))
	for id := range g.external {
		external = append(external, id.String())
	}
	sort.Strings(external)
	return json.Marshal(struct {
		External []string `json:"external,omitempty"`
		Tasks    []entry  `json:"tasks"`
	}{External: external, Tasks: e})
}
func ids(a []ArtifactID) []string {
	r := make([]string, len(a))
	for i, v := range a {
		r[i] = v.String()
	}
	return r
}
func (g *Graph) Digest() (string, error) {
	b, e := g.Serialize()
	if e != nil {
		return "", e
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
