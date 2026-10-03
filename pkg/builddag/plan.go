package builddag

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// SegmentDigest identifies one already-compiled graph segment in execution
// order. Staged DAG builds use segments so later graphs can be materialized
// from state produced by earlier lifecycle barriers.
type SegmentDigest struct {
	Name      string `json:"name"`
	Digest    string `json:"digest"`
	TaskCount int    `json:"task_count"`
}

// CompositeDigest returns a deterministic SHA-256 identity for an ordered set
// of compiled graph segments. Segment order is significant because it is part
// of the build plan's execution semantics.
func CompositeDigest(segments []SegmentDigest) (string, error) {
	for index, segment := range segments {
		if segment.Name == "" {
			return "", fmt.Errorf("builddag: segment %d name is empty", index)
		}
		if segment.Digest == "" {
			return "", fmt.Errorf("builddag: segment %q digest is empty", segment.Name)
		}
		if segment.TaskCount < 0 {
			return "", fmt.Errorf("builddag: segment %q task count is negative", segment.Name)
		}
	}

	data, err := json.Marshal(struct {
		Version  int             `json:"version"`
		Segments []SegmentDigest `json:"segments"`
	}{
		Version:  1,
		Segments: append([]SegmentDigest(nil), segments...),
	})
	if err != nil {
		return "", fmt.Errorf("builddag: marshal staged plan: %w", err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}
