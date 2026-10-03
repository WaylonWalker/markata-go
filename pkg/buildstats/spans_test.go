package buildstats

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestStartSpanRecordsHierarchyAndSanitizesAttributes(t *testing.T) {
	profile := Start()
	SetActiveStage("render")
	SetActivePlugin("render_markdown")

	ctx, root := StartSpan(
		context.Background(),
		" render.post ",
		StringAttribute("post", "posts/example.md"),
		StringAttribute("authorization", "Bearer do-not-record"),
	)
	_, child := StartSpan(ctx, "template.execute", StringAttribute("template", "post.html"))
	child.EndError(errors.New("token=do-not-record"))
	child.End()
	root.End()

	summary := profile.Stop()
	if len(summary.Spans) != 2 {
		t.Fatalf("span count = %d, want 2", len(summary.Spans))
	}

	var rootTiming, childTiming SpanTiming
	for _, span := range summary.Spans {
		switch span.Name {
		case "render.post":
			rootTiming = span
		case "template.execute":
			childTiming = span
		}
	}
	if rootTiming.ID == "" {
		t.Fatal("root span was not recorded")
	}
	if childTiming.ParentID != rootTiming.ID {
		t.Fatalf("child parent = %q, want %q", childTiming.ParentID, rootTiming.ID)
	}
	if rootTiming.Stage != "render" || rootTiming.Plugin != "render_markdown" {
		t.Fatalf("root attribution = %q/%q", rootTiming.Stage, rootTiming.Plugin)
	}
	if rootTiming.Status != "ok" {
		t.Fatalf("root status = %q, want ok", rootTiming.Status)
	}
	if childTiming.Status != "error" {
		t.Fatalf("child status = %q, want error", childTiming.Status)
	}
	if got := rootTiming.Attributes["post"]; got != "posts/example.md" {
		t.Fatalf("post attribute = %q", got)
	}
	if _, ok := rootTiming.Attributes["authorization"]; ok {
		t.Fatal("sensitive authorization attribute was retained")
	}
	if rootTiming.Duration < 0 || childTiming.Duration < 0 {
		t.Fatalf("span durations must be non-negative: root=%s child=%s", rootTiming.Duration, childTiming.Duration)
	}
}

func TestStartSpanWithoutActiveProfileIsNoop(t *testing.T) {
	ctx := context.Background()
	gotCtx, span := StartSpan(ctx, "outside-build", StringAttribute("key", "value"))
	if gotCtx != ctx {
		t.Fatal("no-op span changed context")
	}
	span.End()
	span.EndError(errors.New("ignored"))
}

func TestStartSpanConcurrentCompletion(t *testing.T) {
	profile := Start()
	ctx, root := StartSpan(context.Background(), "root")
	var workers sync.WaitGroup
	for range 32 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			_, child := StartSpan(ctx, "child")
			child.End()
			child.EndError(errors.New("ignored second completion"))
		}()
	}
	workers.Wait()
	root.End()
	summary := profile.Stop()
	if len(summary.Spans) != 33 {
		t.Fatalf("span count = %d, want 33", len(summary.Spans))
	}
	ids := make(map[string]bool)
	for _, span := range summary.Spans {
		if ids[span.ID] {
			t.Fatalf("duplicate span ID %q", span.ID)
		}
		ids[span.ID] = true
		if span.Name == "child" && span.ParentID != root.id {
			t.Fatalf("child parent = %q, want %q", span.ParentID, root.id)
		}
		if span.Status != "ok" {
			t.Fatalf("status = %q, want ok", span.Status)
		}
	}
}
