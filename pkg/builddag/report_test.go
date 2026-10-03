package builddag

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestGraphReportIsDeterministicAcrossDeclarationOrder(t *testing.T) {
	post := ResourceClaim{Resource: ResourceID{Kind: ResourcePost, Key: "post-a"}, Access: AccessWrite}
	cache := ResourceClaim{Resource: ResourceID{Kind: ResourceCache, Key: "render/post-a"}, Access: AccessRead}
	artifact := ArtifactID{Kind: "stage", Key: "ready"}

	first := NewBuilder()
	first.AddTask(TaskSpec{ID: "b", Requires: []ArtifactID{artifact}, Resources: []ResourceClaim{post, cache}})
	first.AddTask(TaskSpec{ID: "a", Provides: []ArtifactID{artifact}})
	firstGraph, err := first.Compile()
	if err != nil {
		t.Fatal(err)
	}

	second := NewBuilder()
	second.AddTask(TaskSpec{ID: "a", Provides: []ArtifactID{artifact}})
	second.AddTask(TaskSpec{ID: "b", Requires: []ArtifactID{artifact}, Resources: []ResourceClaim{cache, post}})
	secondGraph, err := second.Compile()
	if err != nil {
		t.Fatal(err)
	}

	firstJSON, err := firstGraph.MarshalReport(ReportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := secondGraph.MarshalReport(ReportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatalf("reports differ:\n%s\n%s", firstJSON, secondJSON)
	}
}

func TestGraphReportMakesMissingOwnershipExplicit(t *testing.T) {
	builder := NewBuilder()
	builder.AddTask(TaskSpec{ID: "legacy", Exclusive: true})
	graph, err := builder.Compile()
	if err != nil {
		t.Fatal(err)
	}

	data, err := graph.MarshalReport(ReportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"resources": []`) {
		t.Fatalf("report does not make empty ownership explicit: %s", data)
	}
}

func TestGraphReportBoundsTasksAndTaskItems(t *testing.T) {
	builder := NewBuilder()
	builder.AddTask(TaskSpec{
		ID: "a",
		Provides: []ArtifactID{
			{Kind: "output", Key: "one"},
			{Kind: "output", Key: "two"},
			{Kind: "output", Key: "three"},
		},
		Resources: []ResourceClaim{
			{Resource: ResourceID{Kind: ResourceOutput, Key: "one"}, Access: AccessWrite},
			{Resource: ResourceID{Kind: ResourceOutput, Key: "two"}, Access: AccessWrite},
			{Resource: ResourceID{Kind: ResourceOutput, Key: "three"}, Access: AccessWrite},
		},
	})
	builder.AddTask(TaskSpec{ID: "b"})
	builder.AddTask(TaskSpec{ID: "c"})
	graph, err := builder.Compile()
	if err != nil {
		t.Fatal(err)
	}

	report, err := graph.Report(ReportOptions{MaxTasks: 2, MaxItemsPerTask: 2, MaxValueBytes: 64})
	if err != nil {
		t.Fatal(err)
	}
	if report.TaskCount != 3 || report.ReportedTaskCount != 2 || !report.Truncated {
		t.Fatalf("unexpected report bounds: %+v", report)
	}
	if len(report.Order) != 2 || len(report.Tasks) != 2 {
		t.Fatalf("reported slices not bounded: order=%d tasks=%d", len(report.Order), len(report.Tasks))
	}
	if !report.Tasks[0].ItemsTruncated || len(report.Tasks[0].Provides) != 2 || len(report.Tasks[0].Resources) != 2 {
		t.Fatalf("task items not bounded: %+v", report.Tasks[0])
	}
	if report.ValuesTruncated || report.Tasks[0].ValuesTruncated {
		t.Fatalf("short values unexpectedly marked truncated: %+v", report)
	}
}

func TestGraphReportBoundsValuesWithoutInvalidUTF8(t *testing.T) {
	builder := NewBuilder()
	builder.AddTask(TaskSpec{
		ID:    TaskID("render-😀-very-long"),
		Group: "render-😀-very-long",
		Resources: []ResourceClaim{{
			Resource: ResourceID{Kind: ResourcePost, Key: "post-😀-very-long"},
			Access:   AccessWrite,
		}},
	})
	graph, err := builder.Compile()
	if err != nil {
		t.Fatal(err)
	}

	report, err := graph.Report(ReportOptions{MaxTasks: 1, MaxItemsPerTask: 1, MaxValueBytes: 12})
	if err != nil {
		t.Fatal(err)
	}
	if !report.ValuesTruncated || !report.Tasks[0].ValuesTruncated {
		t.Fatalf("value truncation was not reported: %+v", report)
	}
	if !utf8.ValidString(string(report.Tasks[0].ID)) || !utf8.ValidString(report.Tasks[0].Group) || !utf8.ValidString(report.Tasks[0].Resources[0].Resource.Key) {
		t.Fatalf("bounded report contains invalid UTF-8: %+v", report.Tasks[0])
	}
	if len(report.Tasks[0].Group) > 12 || len(report.Tasks[0].Resources[0].Resource.Key) > 12 {
		t.Fatalf("bounded value is too large: %+v", report.Tasks[0])
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal bounded report: %v", err)
	}
	if !utf8.Valid(data) {
		t.Fatalf("report JSON is invalid UTF-8: %q", data)
	}
}
