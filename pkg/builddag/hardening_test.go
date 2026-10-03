package builddag

import (
	"bytes"
	"context"
	"reflect"
	"slices"
	"testing"
)

func TestCompiledDeclarationsAreImmutable(t *testing.T) {
	external := ArtifactID{Kind: "source", Key: "config"}
	produced := ArtifactID{Kind: "post", Key: "rendered"}
	resource := ResourceClaim{Resource: ResourceID{Kind: ResourcePost, Key: "example"}, Access: AccessWrite}
	calls := []TaskID{}
	first := TaskSpec{
		ID: "a", Group: "render", Scope: ScopePost, Version: "v1",
		Requires: []ArtifactID{external}, Provides: []ArtifactID{produced},
		Resources: []ResourceClaim{resource}, ParallelSafe: true,
		Func: func(context.Context) error { calls = append(calls, "a"); return nil },
	}
	second := TaskSpec{
		ID: "b", Requires: []ArtifactID{produced},
		Func: func(context.Context) error { calls = append(calls, "b"); return nil },
	}
	builder := NewBuilder()
	builder.AddExternal(external)
	builder.AddTask(second)
	builder.AddTask(first)
	graph, err := builder.Compile()
	if err != nil {
		t.Fatal(err)
	}
	wantOrder := graph.Order()
	wantSerialized, wantReport, wantDigest := graphSnapshot(t, graph)

	// Mutate backing arrays, not merely the local TaskSpec slice headers.
	first.Requires[0] = ArtifactID{Kind: "bad", Key: "requirement"}
	first.Provides[0] = ArtifactID{Kind: "bad", Key: "provider"}
	first.Resources[0] = ResourceClaim{Resource: ResourceID{Kind: ResourceCache, Key: "other"}, Access: AccessRead}
	second.Requires[0] = first.Provides[0]
	// Destructive external-set mutation must be isolated, as must appended inputs.
	delete(builder.external, external)
	builder.AddExternal(ArtifactID{Kind: "source", Key: "later"})
	builder.AddTask(TaskSpec{ID: "later"})

	task, ok := graph.Task("a")
	if !ok {
		t.Fatal("compiled task missing")
	}
	task.Requires[0] = first.Requires[0]
	task.Provides[0] = first.Provides[0]
	task.Resources[0] = first.Resources[0]
	task.Scope = ScopeSite
	task.Group = "changed"
	task.Version = "changed"
	task.ParallelSafe = false
	task.Func = nil
	order := graph.Order()
	order[0] = "changed"

	gotTask, ok := graph.Task("a")
	if !ok || !reflect.DeepEqual(gotTask.Requires, []ArtifactID{external}) ||
		!reflect.DeepEqual(gotTask.Provides, []ArtifactID{produced}) ||
		!reflect.DeepEqual(gotTask.Resources, []ResourceClaim{resource}) ||
		gotTask.Scope != ScopePost || gotTask.Group != "render" || gotTask.Version != "v1" || !gotTask.ParallelSafe {
		t.Fatalf("compiled metadata changed: %+v", gotTask)
	}
	serialized, report, digest := graphSnapshot(t, graph)
	if !bytes.Equal(serialized, wantSerialized) || !bytes.Equal(report, wantReport) || digest != wantDigest ||
		!reflect.DeepEqual(graph.Order(), wantOrder) {
		t.Fatal("compiled order, serialization, report, or digest changed after input/accessor mutation")
	}
	executor, err := NewExecutor(1)
	if err != nil {
		t.Fatal(err)
	}
	result, err := executor.Execute(context.Background(), graph)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Order, wantOrder) || !reflect.DeepEqual(calls, wantOrder) || result.TaskCount != 2 {
		t.Fatalf("execution changed: result=%+v calls=%v", result, calls)
	}
	// Diagnostic results are owned by the caller too.
	reported, err := graph.Report(ReportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	reported.Tasks[0].Resources[0] = first.Resources[0]
	gotTask, _ = graph.Task("a")
	if gotTask.Resources[0] != resource {
		t.Fatal("report metadata aliases graph")
	}
}

func graphSnapshot(t *testing.T, graph *Graph) (serializedBytes, reportBytes []byte, planDigest string) {
	t.Helper()
	serialized, err := graph.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	report, err := graph.MarshalReport(ReportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	digest, err := graph.Digest()
	if err != nil {
		t.Fatal(err)
	}
	return serialized, report, digest
}

func TestTaskMetadataPreservesNilAndEmptySlices(t *testing.T) {
	builder := NewBuilder()
	builder.AddTask(TaskSpec{ID: "nil"})
	builder.AddTask(TaskSpec{ID: "empty", Requires: []ArtifactID{}, Provides: []ArtifactID{}, Resources: []ResourceClaim{}})
	graph, err := builder.Compile()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []TaskID{"nil", "empty"} {
		task, ok := graph.Task(id)
		if !ok {
			t.Fatalf("task %q missing", id)
		}
		wantNil := id == "nil"
		if (task.Requires == nil) != wantNil || (task.Provides == nil) != wantNil || (task.Resources == nil) != wantNil {
			t.Fatalf("task %q nil/empty metadata changed: %+v", id, task)
		}
	}
}

func TestStructuredCanonicalOrdering(t *testing.T) {
	artifacts := []ArtifactID{{Kind: "post", Key: "rendered:example"}, {Kind: "post:rendered", Key: "example"}}
	external := []ArtifactID{{Kind: "source", Key: "raw:example"}, {Kind: "source:raw", Key: "example"}}
	if artifacts[0].String() != artifacts[1].String() || external[0].String() != external[1].String() {
		t.Fatal("test requires colliding display strings")
	}
	// Kind takes precedence over access; opaque keys retain punctuation.
	resources := []ResourceClaim{
		{Resource: ResourceID{Kind: ResourceCache, Key: "render:example"}, Access: AccessWrite},
		{Resource: ResourceID{Kind: ResourcePost, Key: "example:render"}, Access: AccessRead},
	}
	var wantSerialized, wantReport []byte
	var wantDigest string
	for iteration := 0; iteration < 50; iteration++ {
		inputs, outputs, claims := slices.Clone(external), slices.Clone(artifacts), slices.Clone(resources)
		if iteration%2 != 0 {
			slices.Reverse(inputs)
			slices.Reverse(outputs)
			slices.Reverse(claims)
		}
		tasks := []TaskSpec{
			{ID: "producer", Requires: inputs, Provides: outputs, Resources: claims},
			{ID: "consumer", Requires: outputs},
		}
		if iteration%2 != 0 {
			slices.Reverse(tasks)
		}
		builder := NewBuilder()
		for _, input := range inputs {
			builder.AddExternal(input)
		}
		for _, task := range tasks {
			builder.AddTask(task)
		}
		graph, err := builder.Compile()
		if err != nil {
			t.Fatal(err)
		}
		serialized, report, digest := graphSnapshot(t, graph)
		if iteration == 0 {
			wantSerialized, wantReport, wantDigest = serialized, report, digest
		}
		if !bytes.Equal(serialized, wantSerialized) || !bytes.Equal(report, wantReport) || digest != wantDigest {
			t.Fatalf("canonical output changed on iteration %d", iteration)
		}
		// Repeated reads exercise external map iteration independently of builds.
		repeatedSerialized, repeatedReport, repeatedDigest := graphSnapshot(t, graph)
		if !bytes.Equal(repeatedSerialized, wantSerialized) || !bytes.Equal(repeatedReport, wantReport) || repeatedDigest != wantDigest {
			t.Fatalf("repeated canonical output changed on iteration %d", iteration)
		}
		bounded, err := graph.Report(ReportOptions{MaxItemsPerTask: 1})
		if err != nil {
			t.Fatal(err)
		}
		if bounded.Tasks[0].Provides[0] != artifacts[0] || bounded.Tasks[0].Requires[0] != external[0] ||
			bounded.Tasks[0].Resources[0] != resources[0] || !bounded.Tasks[0].ItemsTruncated {
			t.Fatalf("bounded report did not sort structured fields before truncation: %+v", bounded.Tasks[0])
		}
	}
}

func TestResourceClaimStructuredOrder(t *testing.T) {
	base := ResourceClaim{Resource: ResourceID{Kind: ResourceCache, Key: "a"}, Access: AccessRead}
	for _, later := range []ResourceClaim{
		{Resource: ResourceID{Kind: ResourcePost, Key: "a"}, Access: AccessRead},
		{Resource: ResourceID{Kind: ResourceCache, Key: "a:b"}, Access: AccessRead},
		{Resource: base.Resource, Access: AccessWrite},
	} {
		if !resourceClaimLess(base, later) || resourceClaimLess(later, base) || resourceClaimLess(base, base) {
			t.Fatalf("invalid structured comparison: %+v vs %+v", base, later)
		}
	}
}

func TestTopologicalSmallestReadyOrder(t *testing.T) {
	tests := []struct {
		name  string
		deps  map[TaskID][]TaskID
		order []TaskID
	}{
		{"fan-out", map[TaskID][]TaskID{"b": nil, "a": {"b"}, "e": {"b"}, "c": nil}, []TaskID{"b", "a", "c", "e"}},
		{"fan-in", map[TaskID][]TaskID{"a": nil, "c": nil, "b": {"a", "c"}, "d": nil}, []TaskID{"a", "c", "b", "d"}},
		{"disconnected", map[TaskID][]TaskID{"z": nil, "c": nil, "a": nil}, []TaskID{"a", "c", "z"}},
		{"mixed", map[TaskID][]TaskID{"b": nil, "a": {"b"}, "aa": {"a"}, "d": {"b"}, "e": {"a", "d", "d"}, "c": nil, "z": nil}, []TaskID{"b", "a", "aa", "c", "d", "e", "z"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for iteration := 0; iteration < 20; iteration++ {
				builder := NewBuilder()
				for id, dependencies := range test.deps {
					task := TaskSpec{ID: id, Provides: []ArtifactID{{Kind: "task", Key: string(id)}}}
					for _, dependency := range dependencies {
						task.Requires = append(task.Requires, ArtifactID{Kind: "task", Key: string(dependency)})
					}
					builder.AddTask(task)
				}
				graph, err := builder.Compile()
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(graph.Order(), test.order) {
					t.Fatalf("order = %v, want %v", graph.Order(), test.order)
				}
			}
		})
	}
}

func TestCycleDiagnosticsRemainStable(t *testing.T) {
	for iteration := 0; iteration < 20; iteration++ {
		builder := NewBuilder()
		for id, dependencies := range map[TaskID][]TaskID{"a": {"c", "b"}, "b": {"a"}, "c": {"c"}, "ready": nil} {
			task := TaskSpec{ID: id, Provides: []ArtifactID{{Kind: "task", Key: string(id)}}}
			for _, dependency := range dependencies {
				task.Requires = append(task.Requires, ArtifactID{Kind: "task", Key: string(dependency)})
			}
			builder.AddTask(task)
		}
		_, err := builder.Compile()
		if err == nil || err.Error() != "builddag: task dependency cycle: [a b a]" {
			t.Fatalf("cycle error = %v, want stable [a b a] diagnostic", err)
		}
	}
}
