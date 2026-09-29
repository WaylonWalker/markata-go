package diagnostics

import (
	"strings"
	"testing"
	"time"
)

func TestArtifactExecutorMetadata(t *testing.T) {
	builtAt := time.Date(2026, time.September, 28, 22, 0, 0, 0, time.UTC)
	for _, executor := range []string{ArtifactExecutorLegacy, ArtifactExecutorDAG} {
		t.Run(executor, func(t *testing.T) {
			data, err := MarshalArtifact(ContentLedgerSnapshot{Entries: []ContentDisposition{}}, ArtifactBuildInfo{
				BuiltAt:  builtAt,
				Executor: executor,
			})
			if err != nil {
				t.Fatalf("MarshalArtifact() = %v", err)
			}
			artifact, err := ParseArtifact(data)
			if err != nil {
				t.Fatalf("ParseArtifact() = %v", err)
			}
			if artifact.Executor != executor {
				t.Fatalf("executor = %q, want %q", artifact.Executor, executor)
			}
		})
	}
}

func TestArtifactExecutorMetadataRejectsUnknownValue(t *testing.T) {
	_, err := MarshalArtifact(ContentLedgerSnapshot{Entries: []ContentDisposition{}}, ArtifactBuildInfo{
		BuiltAt:  time.Date(2026, time.September, 28, 22, 0, 0, 0, time.UTC),
		Executor: "parallel-magic",
	})
	if err == nil || !strings.Contains(err.Error(), "executor") {
		t.Fatalf("MarshalArtifact() error = %v, want executor validation error", err)
	}
}

func TestParseArtifactAllowsOlderV1WithoutExecutor(t *testing.T) {
	data := []byte(`{"$schema":"markata://schemas/content-diagnostics/v1","schema":"markata.content-diagnostics","schema_version":1,"generator":{"name":"markata-go","version":"test"},"built_at":"2026-09-28T22:00:00Z","summary":{},"entries":[]}`)
	artifact, err := ParseArtifact(data)
	if err != nil {
		t.Fatalf("ParseArtifact() = %v", err)
	}
	if artifact.Executor != "" {
		t.Fatalf("executor = %q, want empty for older v1 artifact", artifact.Executor)
	}
}
