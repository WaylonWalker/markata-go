package builddag

import (
	"strings"
	"testing"
)

func TestGraphCompileRejectsExclusiveParallelSafeTask(t *testing.T) {
	builder := NewBuilder()
	builder.AddTask(TaskSpec{
		ID:           "task",
		Exclusive:    true,
		ParallelSafe: true,
		Resources: []ResourceClaim{{
			Resource: ResourceID{Kind: ResourcePost, Key: "post-a"},
			Access:   AccessWrite,
		}},
	})

	_, err := builder.Compile()
	if err == nil || !strings.Contains(err.Error(), "exclusive task cannot be parallel-safe") {
		t.Fatalf("Compile() error = %v, want exclusive/parallel-safe conflict", err)
	}
}

func TestGraphCompileRequiresClaimsForParallelSafeTask(t *testing.T) {
	builder := NewBuilder()
	builder.AddTask(TaskSpec{ID: "task", ParallelSafe: true})

	_, err := builder.Compile()
	if err == nil || !strings.Contains(err.Error(), "must declare resource claims") {
		t.Fatalf("Compile() error = %v, want missing resource claims", err)
	}
}

func TestGraphCompileRejectsInvalidResourceClaim(t *testing.T) {
	tests := []ResourceClaim{
		{Resource: ResourceID{Kind: ResourcePost, Key: ""}, Access: AccessRead},
		{Resource: ResourceID{Kind: "", Key: "post-a"}, Access: AccessRead},
		{Resource: ResourceID{Kind: ResourcePost, Key: "post-a"}, Access: "execute"},
	}

	for _, claim := range tests {
		builder := NewBuilder()
		builder.AddTask(TaskSpec{ID: "task", Resources: []ResourceClaim{claim}})
		_, err := builder.Compile()
		if err == nil || !strings.Contains(err.Error(), "invalid resource claim") {
			t.Fatalf("Compile() claim=%+v error = %v, want invalid resource claim", claim, err)
		}
	}
}

func TestGraphCompileRejectsDuplicateResourceClaim(t *testing.T) {
	resource := ResourceID{Kind: ResourceCache, Key: "render/post-a"}
	builder := NewBuilder()
	builder.AddTask(TaskSpec{
		ID: "task",
		Resources: []ResourceClaim{
			{Resource: resource, Access: AccessRead},
			{Resource: resource, Access: AccessWrite},
		},
	})

	_, err := builder.Compile()
	if err == nil || !strings.Contains(err.Error(), "declared more than once") {
		t.Fatalf("Compile() error = %v, want duplicate resource declaration", err)
	}
}

func TestResourceClaimOrderDoesNotChangeDigest(t *testing.T) {
	post := ResourceClaim{Resource: ResourceID{Kind: ResourcePost, Key: "post-a"}, Access: AccessWrite}
	cache := ResourceClaim{Resource: ResourceID{Kind: ResourceCache, Key: "render/post-a"}, Access: AccessRead}

	first := NewBuilder()
	first.AddTask(TaskSpec{ID: "render", Resources: []ResourceClaim{post, cache}})
	firstGraph, err := first.Compile()
	if err != nil {
		t.Fatalf("compile first graph: %v", err)
	}
	firstDigest, err := firstGraph.Digest()
	if err != nil {
		t.Fatalf("digest first graph: %v", err)
	}

	second := NewBuilder()
	second.AddTask(TaskSpec{ID: "render", Resources: []ResourceClaim{cache, post}})
	secondGraph, err := second.Compile()
	if err != nil {
		t.Fatalf("compile second graph: %v", err)
	}
	secondDigest, err := secondGraph.Digest()
	if err != nil {
		t.Fatalf("digest second graph: %v", err)
	}

	if firstDigest != secondDigest {
		t.Fatalf("resource declaration order changed digest: %s != %s", firstDigest, secondDigest)
	}

	serialized, err := firstGraph.Serialize()
	if err != nil {
		t.Fatalf("serialize graph: %v", err)
	}
	if !strings.Contains(string(serialized), `"resources"`) {
		t.Fatalf("serialized graph does not expose resource claims: %s", serialized)
	}
}

func TestLegacyTaskWithoutClaimsRemainsValid(t *testing.T) {
	builder := NewBuilder()
	builder.AddTask(TaskSpec{ID: "legacy", Exclusive: true})
	graph, err := builder.Compile()
	if err != nil {
		t.Fatalf("Compile() legacy task: %v", err)
	}
	serialized, err := graph.Serialize()
	if err != nil {
		t.Fatalf("Serialize() legacy task: %v", err)
	}
	if strings.Contains(string(serialized), `"resources"`) {
		t.Fatalf("legacy task unexpectedly serialized empty resources: %s", serialized)
	}
}
