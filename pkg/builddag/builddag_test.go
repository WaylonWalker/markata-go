package builddag

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func aid(kind, key string) ArtifactID { return ArtifactID{Kind: kind, Key: key} }

func TestArtifactID_RoundTrip(t *testing.T) {
	want := aid("post", "one")
	got, err := ParseArtifactID(want.String())
	if err != nil || got != want {
		t.Fatalf("round trip = %#v, %v", got, err)
	}
	if _, err := ParseArtifactID("missing-separator"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestBuilder_Validation(t *testing.T) {
	x := aid("input", "x")
	cases := []struct {
		name     string
		tasks    []TaskSpec
		external bool
		want     string
	}{
		{"duplicate task", []TaskSpec{{ID: "a"}, {ID: "a"}}, false, "duplicate task ID"},
		{"duplicate provider", []TaskSpec{{ID: "a", Provides: []ArtifactID{x}}, {ID: "b", Provides: []ArtifactID{x}}}, false, "duplicate provider"},
		{"missing provider", []TaskSpec{{ID: "a", Requires: []ArtifactID{x}}}, false, "missing provider"},
		{"cycle", []TaskSpec{{ID: "a", Requires: []ArtifactID{aid("v", "b")}, Provides: []ArtifactID{aid("v", "a")}}, {ID: "b", Requires: []ArtifactID{aid("v", "a")}, Provides: []ArtifactID{aid("v", "b")}}}, false, "cycle"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder()
			for _, task := range tc.tasks {
				b.AddTask(task)
			}
			if tc.external {
				b.AddExternal(x)
			}
			_, err := b.Compile()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
	b := NewBuilder()
	b.AddExternal(x)
	b.AddTask(TaskSpec{ID: "a", Requires: []ArtifactID{x}})
	if _, err := b.Compile(); err != nil {
		t.Fatal(err)
	}
}

func TestGraph_OrderSerializationDigestStable(t *testing.T) {
	a, b, c := aid("x", "a"), aid("x", "b"), aid("x", "c")
	makeGraph := func(reverse bool) *Graph {
		bld := NewBuilder()
		tasks := []TaskSpec{{ID: "finish", Requires: []ArtifactID{b}, Provides: []ArtifactID{c}}, {ID: "start", Provides: []ArtifactID{a}}, {ID: "middle", Requires: []ArtifactID{a}, Provides: []ArtifactID{b}}}
		if reverse {
			for i := len(tasks) - 1; i >= 0; i-- {
				bld.AddTask(tasks[i])
			}
		} else {
			for _, task := range tasks {
				bld.AddTask(task)
			}
		}
		g, err := bld.Compile()
		if err != nil {
			t.Fatal(err)
		}
		return g
	}
	g1, g2 := makeGraph(false), makeGraph(true)
	if !reflect.DeepEqual(g1.Order(), []TaskID{"start", "middle", "finish"}) {
		t.Fatal(g1.Order())
	}
	s1, err := g1.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	s2, err := g2.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(s1, s2) {
		t.Fatal("serialization is not stable")
	}
	d1, err := g1.Digest()
	if err != nil {
		t.Fatal(err)
	}
	d2, err := g2.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if d1 != d2 {
		t.Fatal("digest is not stable")
	}
}

func TestExecutor_SerialRandomAndPersistence(t *testing.T) {
	x, y := aid("x", "x"), aid("x", "y")
	order := []TaskID{}
	task := func(id TaskID, out ArtifactID) TaskSpec {
		return TaskSpec{ID: id, Provides: []ArtifactID{out}, Func: func(context.Context, TaskContext) (TaskResult, error) {
			order = append(order, id)
			return TaskResult{Artifacts: []Artifact{{ID: out, Data: []byte(string(id))}}}, nil
		}}
	}
	b := NewBuilder()
	b.AddTask(task("b", x))
	b.AddTask(task("a", y))
	g, err := b.Compile()
	if err != nil {
		t.Fatal(err)
	}
	e, err := NewExecutor(1)
	if err != nil {
		t.Fatal(err)
	}
	state, err := e.Execute(context.Background(), g, NewMemoryStore(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 {
		t.Fatal(order)
	}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	var restored ExecutionState
	if err = json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	order = nil
	store := NewMemoryStore()
	if _, err = e.Execute(context.Background(), g, store, &restored); err != nil {
		t.Fatal(err)
	}
	if len(order) != 0 {
		t.Fatal("cached state was not restored")
	}
	if _, err = NewExecutor(2); err == nil {
		t.Fatal("expected concurrency rejection")
	}
	if restored.GraphDigest == "" {
		t.Fatal("graph digest was not persisted")
	}
	_ = x
	_ = y
}

func TestExecutor_RandomizesIndependentReadyTasks(t *testing.T) {
	makeGraph := func() *Graph {
		b := NewBuilder()
		ids := []TaskID{"a", "b"}
		for i := range ids {
			id := ids[i]
			out := aid("out", string(id))
			b.AddTask(TaskSpec{ID: id, Provides: []ArtifactID{out}, Func: func(context.Context, TaskContext) (TaskResult, error) {
				return TaskResult{Artifacts: []Artifact{{ID: out}}}, nil
			}})
		}
		graph, err := b.Compile()
		if err != nil {
			t.Fatal(err)
		}
		return graph
	}

	orders := make(map[string]bool)
	for seed := int64(0); seed < 32; seed++ {
		var order []TaskID
		graph := makeGraph()
		for id := range graph.tasks {
			task := graph.tasks[id]
			original := task.Func
			capturedID := id
			task.Func = func(ctx context.Context, tc TaskContext) (TaskResult, error) {
				order = append(order, capturedID)
				return original(ctx, tc)
			}
			graph.tasks[id] = task
		}
		executor := &Executor{MaxParallel: 1, Seed: seed, RandomizeReady: true}
		if _, err := executor.Execute(context.Background(), graph, NewMemoryStore(), nil); err != nil {
			t.Fatal(err)
		}
		orders[string(order[0])+string(order[1])] = true
	}
	if len(orders) < 2 {
		t.Fatalf("randomized ready tasks produced only one order: %v", orders)
	}
}

func TestExecutionState_InvalidateIncludesDynamicDeps(t *testing.T) {
	input, out := aid("file", "input"), aid("file", "out")
	b := NewBuilder()
	b.AddExternal(input)
	b.AddTask(TaskSpec{ID: "build", Requires: []ArtifactID{input}, Provides: []ArtifactID{out}, Func: func(context.Context, TaskContext) (TaskResult, error) {
		return TaskResult{Artifacts: []Artifact{{ID: out}}, DynamicDeps: []ArtifactID{aid("file", "extra")}}, nil
	}})
	g, err := b.Compile()
	if err != nil {
		t.Fatal(err)
	}
	store := NewMemoryStore()
	store.Put(Artifact{ID: input, Data: []byte("input")})
	state, err := (&Executor{MaxParallel: 1}).Execute(context.Background(), g, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	var restored ExecutionState
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	if got := restored.Invalidate(g, []ArtifactID{aid("file", "extra")}); !reflect.DeepEqual(got, []TaskID{"build"}) {
		t.Fatal(got)
	}
	var restoredInput ExecutionState
	if err := json.Unmarshal(raw, &restoredInput); err != nil {
		t.Fatal(err)
	}
	if got := restoredInput.Invalidate(g, []ArtifactID{input}); !reflect.DeepEqual(got, []TaskID{"build"}) {
		t.Fatalf("external input invalidation = %v", got)
	}
}

func TestExecutor_RequiresDeclaredInputs(t *testing.T) {
	input, output := aid("file", "input"), aid("file", "output")
	b := NewBuilder()
	b.AddExternal(input)
	b.AddTask(TaskSpec{
		ID: "build", Requires: []ArtifactID{input}, Provides: []ArtifactID{output},
		Func: func(context.Context, TaskContext) (TaskResult, error) {
			return TaskResult{Artifacts: []Artifact{{ID: output}}}, nil
		},
	})
	g, err := b.Compile()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&Executor{MaxParallel: 1}).Execute(context.Background(), g, NewMemoryStore(), nil); err == nil || !strings.Contains(err.Error(), "missing artifact") {
		t.Fatalf("missing input error = %v", err)
	}
}

func TestExecutionState_SaveAndLoad(t *testing.T) {
	path := t.TempDir() + "/state.json"
	want := &ExecutionState{SchemaVersion: 1, GraphDigest: "graph", Tasks: map[TaskID]TaskState{
		"task": {Completed: true, Version: "v1", Result: TaskResult{DynamicDeps: []ArtifactID{aid("post", "target")}}},
	}}
	if err := SaveExecutionState(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadExecutionState(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("state = %#v, want %#v", got, want)
	}
}
