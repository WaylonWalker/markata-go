package builddag

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestGraphCompileStableOrderAndDigest(t *testing.T) {
	artifactA := ArtifactID{Kind: "stage", Key: "a"}
	artifactB := ArtifactID{Kind: "stage", Key: "b"}
	builder := NewBuilder()
	builder.AddTask(TaskSpec{ID: "b", Requires: []ArtifactID{artifactA}, Provides: []ArtifactID{artifactB}})
	builder.AddTask(TaskSpec{ID: "a", Provides: []ArtifactID{artifactA}})
	graph, err := builder.Compile()
	if err != nil {
		t.Fatalf("compile graph: %v", err)
	}
	if got, want := graph.Order(), []TaskID{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	first, err := graph.Digest()
	if err != nil {
		t.Fatalf("digest graph: %v", err)
	}

	other := NewBuilder()
	other.AddTask(TaskSpec{ID: "a", Provides: []ArtifactID{artifactA}})
	other.AddTask(TaskSpec{ID: "b", Requires: []ArtifactID{artifactA}, Provides: []ArtifactID{artifactB}})
	otherGraph, err := other.Compile()
	if err != nil {
		t.Fatalf("compile reordered graph: %v", err)
	}
	second, err := otherGraph.Digest()
	if err != nil {
		t.Fatalf("digest reordered graph: %v", err)
	}
	if first != second {
		t.Fatalf("digest changed with declaration order: %s != %s", first, second)
	}
}

func TestGraphCompileRejectsMissingProvider(t *testing.T) {
	builder := NewBuilder()
	builder.AddTask(TaskSpec{ID: "consumer", Requires: []ArtifactID{{Kind: "stage", Key: "missing"}}})
	_, err := builder.Compile()
	if err == nil || !strings.Contains(err.Error(), "missing provider") {
		t.Fatalf("Compile() error = %v, want missing provider", err)
	}
}

func TestGraphCompileAllowsDeclaredExternal(t *testing.T) {
	external := ArtifactID{Kind: "source", Key: "config"}
	builder := NewBuilder()
	builder.AddExternal(external)
	builder.AddTask(TaskSpec{ID: "configure", Requires: []ArtifactID{external}, Provides: []ArtifactID{{Kind: "stage", Key: "configure"}}})
	if _, err := builder.Compile(); err != nil {
		t.Fatalf("Compile() with external input: %v", err)
	}
}

func TestGraphCompileRejectsDuplicateProvider(t *testing.T) {
	artifact := ArtifactID{Kind: "stage", Key: "same"}
	builder := NewBuilder()
	builder.AddTask(TaskSpec{ID: "one", Provides: []ArtifactID{artifact}})
	builder.AddTask(TaskSpec{ID: "two", Provides: []ArtifactID{artifact}})
	_, err := builder.Compile()
	if err == nil || !strings.Contains(err.Error(), "duplicate provider") {
		t.Fatalf("Compile() error = %v, want duplicate provider", err)
	}
}

func TestGraphCompileRejectsCycle(t *testing.T) {
	a := ArtifactID{Kind: "stage", Key: "a"}
	b := ArtifactID{Kind: "stage", Key: "b"}
	builder := NewBuilder()
	builder.AddTask(TaskSpec{ID: "a", Requires: []ArtifactID{b}, Provides: []ArtifactID{a}})
	builder.AddTask(TaskSpec{ID: "b", Requires: []ArtifactID{a}, Provides: []ArtifactID{b}})
	_, err := builder.Compile()
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("Compile() error = %v, want cycle", err)
	}
}

func TestExecutorRunsSerialTopologicalOrder(t *testing.T) {
	a := ArtifactID{Kind: "stage", Key: "a"}
	b := ArtifactID{Kind: "stage", Key: "b"}
	calls := make([]TaskID, 0, 2)
	builder := NewBuilder()
	builder.AddTask(TaskSpec{ID: "b", Requires: []ArtifactID{a}, Provides: []ArtifactID{b}, Func: func(context.Context) error {
		calls = append(calls, "b")
		return nil
	}})
	builder.AddTask(TaskSpec{ID: "a", Provides: []ArtifactID{a}, Func: func(context.Context) error {
		calls = append(calls, "a")
		return nil
	}})
	graph, err := builder.Compile()
	if err != nil {
		t.Fatalf("compile graph: %v", err)
	}
	executor, err := NewExecutor(1)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	result, err := executor.Execute(context.Background(), graph)
	if err != nil {
		t.Fatalf("execute graph: %v", err)
	}
	if got, want := calls, []TaskID{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %v, want %v", got, want)
	}
	if result.TaskCount != 2 {
		t.Fatalf("task count = %d, want 2", result.TaskCount)
	}
}

func TestExecutorRejectsParallelism(t *testing.T) {
	if _, err := NewExecutor(2); err == nil {
		t.Fatal("NewExecutor(2) succeeded, want serial-only error")
	}
}
