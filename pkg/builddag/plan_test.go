package builddag

import "testing"

func TestCompositeDigestIsDeterministicAndOrderSensitive(t *testing.T) {
	segments := []SegmentDigest{
		{Name: "load", Digest: "aaa", TaskCount: 3},
		{Name: "transform", Digest: "bbb", TaskCount: 8},
	}

	first, err := CompositeDigest(segments)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CompositeDigest(append([]SegmentDigest(nil), segments...))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("identical staged plans produced different digests: %s != %s", first, second)
	}

	reversed, err := CompositeDigest([]SegmentDigest{segments[1], segments[0]})
	if err != nil {
		t.Fatal(err)
	}
	if first == reversed {
		t.Fatalf("segment order did not affect composite digest: %s", first)
	}
}

func TestCompositeDigestIncludesTaskCounts(t *testing.T) {
	first, err := CompositeDigest([]SegmentDigest{{Name: "transform", Digest: "same", TaskCount: 2}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := CompositeDigest([]SegmentDigest{{Name: "transform", Digest: "same", TaskCount: 3}})
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("task count did not affect composite digest: %s", first)
	}
}

func TestCompositeDigestRejectsInvalidSegments(t *testing.T) {
	tests := [][]SegmentDigest{
		{{Name: "", Digest: "aaa", TaskCount: 1}},
		{{Name: "load", Digest: "", TaskCount: 1}},
		{{Name: "load", Digest: "aaa", TaskCount: -1}},
	}
	for _, segments := range tests {
		if _, err := CompositeDigest(segments); err == nil {
			t.Fatalf("CompositeDigest(%+v) succeeded, want error", segments)
		}
	}
}
