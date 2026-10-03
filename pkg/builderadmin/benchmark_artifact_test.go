package builderadmin

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBenchmarkArtifactPath(t *testing.T) {
	t.Parallel()

	historyDir := t.TempDir()
	got, err := benchmarkArtifactPath(historyDir, "build-123")
	if err != nil {
		t.Fatalf("benchmarkArtifactPath: %v", err)
	}
	want := filepath.Join(historyDir, benchmarkArtifactDirName, "build-123.json")
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}

	for _, id := range []string{"", "../secret", "nested/build", `nested\\build`} {
		if _, err := benchmarkArtifactPath(historyDir, id); err == nil {
			t.Errorf("benchmarkArtifactPath(%q) unexpectedly succeeded", id)
		}
	}
}

func TestReadBuildBenchmarkArtifact(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "benchmark.json")
	contents := `{
  "posts_processed": 2722,
  "feeds_generated": 18,
  "duration_seconds": 810.4,
  "warnings": ["example warning"],
  "benchmark": {
    "Total": 810400000000,
    "Resources": {"CPU": 191000000000, "NetworkWait": 104000000000},
    "Hotspots": [{"Stage":"write", "Plugin":"publish_feeds", "Duration":184200000000}],
    "Requests": [],
    "Stages": []
  },
  "content": {},
  "future_field": "ignored"
}`
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	artifact, err := readBuildBenchmarkArtifact(path)
	if err != nil {
		t.Fatalf("readBuildBenchmarkArtifact: %v", err)
	}
	if artifact.PostsProcessed != 2722 || artifact.FeedsGenerated != 18 {
		t.Fatalf("counts = posts:%d feeds:%d", artifact.PostsProcessed, artifact.FeedsGenerated)
	}
	if artifact.Duration != 810.4 {
		t.Fatalf("duration = %v", artifact.Duration)
	}
	if artifact.Benchmark.Total != 810400*time.Millisecond {
		t.Fatalf("benchmark total = %s", artifact.Benchmark.Total)
	}
	if len(artifact.Benchmark.Hotspots) != 1 || artifact.Benchmark.Hotspots[0].Plugin != "publish_feeds" {
		t.Fatalf("hotspots = %#v", artifact.Benchmark.Hotspots)
	}
}

func TestEnsureBenchmarkArtifactDir(t *testing.T) {
	t.Parallel()

	historyDir := t.TempDir()
	dir, err := ensureBenchmarkArtifactDir(historyDir)
	if err != nil {
		t.Fatalf("ensureBenchmarkArtifactDir: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat benchmark dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", dir)
	}
}
