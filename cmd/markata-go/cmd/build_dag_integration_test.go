package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/buildlab"
)

func TestBuildCLI_DAGEquivalentAfterLinkedTargetMutations(t *testing.T) {
	t.Helper()
	requireLinuxBuildLab(t)
	repoRoot := moduleRoot(t)
	fixture := filepath.Join(repoRoot, "cmd", "markata-go", "cmd", "testdata", "dag-site")
	if _, err := os.Stat(fixture); err != nil {
		t.Fatal(err)
	}
	binary := buildTestBinary(t)

	result, err := buildlab.RunScenario(context.Background(), buildlab.ScenarioRunConfig{
		Fixture: fixture,
		Scenario: buildlab.Scenario{
			ID:      "cli-dag-linked-target",
			Version: "1",
			Operations: []buildlab.Operation{
				{Type: buildlab.OpBuild},
				{Type: buildlab.OpWriteFile, Path: "content/future-target.md", Content: "---\ntitle: Future target\ndate: 2026-08-03\npublished: true\n---\n\n# Future target\n\nA target created after the initial build.\n"},
				{Type: buildlab.OpBuild},
				{Type: buildlab.OpReplaceExact, Path: "content/target.md", Old: "Original target content.", New: "Updated target content."},
				{Type: buildlab.OpBuild},
				{Type: buildlab.OpDelete, Path: "content/target.md"},
				{Type: buildlab.OpBuild},
				{Type: buildlab.OpWriteFile, Path: "content/target.md", Content: "---\ntitle: Target post\ndate: 2026-08-01\npublished: true\n---\n\n# Target post\n\nRecreated target content.\n"},
				{Type: buildlab.OpBuild},
				{Type: buildlab.OpRename, Path: "content/target.md", Dest: "content/renamed.md"},
				{Type: buildlab.OpBuild},
			},
		},
		Baseline: buildlab.BuildCommand{
			Binary:    binary,
			Args:      []string{"build", "--no-color", "--no-input", "-c", "markata-go.toml"},
			OutputDir: "output",
			Timeout:   5 * time.Minute,
			Env:       []string{"MARKATA_GO_ENCRYPTION_ENABLED=false"},
		},
		Candidate: buildlab.BuildCommand{
			Binary:    binary,
			Args:      []string{"build", "--no-color", "--no-input", "-c", "markata-go.toml", "--dag"},
			OutputDir: "output",
			Timeout:   5 * time.Minute,
			Env:       []string{"MARKATA_GO_ENCRYPTION_ENABLED=false"},
		},
		CheckDeterminism: true,
		GOMAXPROCS:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != buildLabPassVerdict {
		t.Fatalf("verdict = %q, result = %+v", result.Verdict, result)
	}
	if len(result.Checkpoints) != 6 {
		t.Fatalf("checkpoints = %d, want 6", len(result.Checkpoints))
	}
	for _, checkpoint := range result.Checkpoints {
		if !checkpoint.Correctness.DifferentialEqual {
			t.Fatalf("differential equality failed at operation %d: %+v", checkpoint.OperationIndex, checkpoint.Correctness.DifferentialDiff)
		}
		if checkpoint.Correctness.IncrementalApplicable && !checkpoint.Correctness.IncrementalEqual {
			t.Fatalf("incremental equality failed at operation %d: %+v", checkpoint.OperationIndex, checkpoint.Correctness)
		}
		if !checkpoint.Correctness.DeterministicEqual {
			t.Fatalf("determinism equality failed at operation %d: %+v", checkpoint.OperationIndex, checkpoint.Correctness.DeterminismDiff)
		}
	}
}

func TestBuildLab_IgnoresAmbientSiteDirectory(t *testing.T) {
	requireLinuxBuildLab(t)
	repoRoot := moduleRoot(t)
	fixture := filepath.Join(repoRoot, "cmd", "markata-go", "cmd", "testdata", "dag-site")
	trap := t.TempDir()
	t.Setenv("MARKATA_GO_SITE_DIR", trap)
	binary := buildTestBinary(t)

	result, err := buildlab.RunScenario(context.Background(), buildlab.ScenarioRunConfig{
		Fixture: fixture,
		Scenario: buildlab.Scenario{
			ID:         "buildlab-site-dir-isolation",
			Version:    "1",
			Operations: []buildlab.Operation{{Type: buildlab.OpBuild}},
		},
		Baseline: buildlab.BuildCommand{
			Binary: binary,
			Args:   []string{"build", "--no-color", "--no-input", "-c", "markata-go.toml"},
			Env:    []string{"MARKATA_GO_SITE_DIR=" + trap, "MARKATA_GO_ENCRYPTION_ENABLED=false"},
		},
		Candidate: buildlab.BuildCommand{
			Binary: binary,
			Args:   []string{"build", "--no-color", "--no-input", "-c", "markata-go.toml", "--dag"},
			Env:    []string{"MARKATA_GO_SITE_DIR=" + trap, "MARKATA_GO_ENCRYPTION_ENABLED=false"},
		},
		GOMAXPROCS: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != buildLabPassVerdict {
		t.Fatalf("ambient site directory escaped the workspace: %+v", result)
	}
}

func TestBuildLab_IsolatesAbsoluteConfiguredCache(t *testing.T) {
	requireLinuxBuildLab(t)
	repoRoot := moduleRoot(t)
	sourceFixture := filepath.Join(repoRoot, "cmd", "markata-go", "cmd", "testdata", "dag-site")
	fixture := t.TempDir()
	if err := buildlab.CopyFixture(sourceFixture, fixture); err != nil {
		t.Fatal(err)
	}

	externalCache := t.TempDir()
	sentinel := filepath.Join(externalCache, "sentinel")
	if err := os.WriteFile(sentinel, []byte("keep me\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dotEnvCache := t.TempDir()
	dotEnvSentinel := filepath.Join(dotEnvCache, "sentinel")
	if err := os.WriteFile(dotEnvSentinel, []byte("keep dotenv cache\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, ".env"), []byte("MARKATA_GO_SEARCH_PAGEFIND_CACHE_DIR="+dotEnvCache+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fakeTools := t.TempDir()
	fakePagefind := filepath.Join(fakeTools, "pagefind")
	const fakePagefindScript = `#!/bin/sh
site=""
bundle="_pagefind"
while [ "$#" -gt 0 ]; do
  case "$1" in
    --site) site="$2"; shift 2 ;;
    --output-subdir) bundle="$2"; shift 2 ;;
    *) shift ;;
  esac
done
mkdir -p "$site/$bundle"
if [ -n "${MARKATA_GO_SEARCH_PAGEFIND_CACHE_DIR:-}" ]; then
  mkdir -p "$MARKATA_GO_SEARCH_PAGEFIND_CACHE_DIR"
  printf 'dotenv override escaped\n' > "$MARKATA_GO_SEARCH_PAGEFIND_CACHE_DIR/escaped"
fi
`
	//nolint:gosec // The test requires an executable fake Pagefind command.
	if err := os.WriteFile(fakePagefind, []byte(fakePagefindScript), 0o700); err != nil {
		t.Fatal(err)
	}
	overlay := filepath.Join(fixture, "cache-overlay.toml")
	cachePath := func(name string) string { return strconv.Quote(filepath.Join(externalCache, name)) }
	overlayData := "[markata-go]\n" +
		"cache_dir = " + cachePath("build") + "\n\n" +
		"[markata-go.assets]\ncache_dir = " + cachePath("assets") + "\n\n" +
		"[markata-go.blogroll]\ncache_dir = " + cachePath("blogroll") + "\n\n" +
		"[markata-go.embeds]\ncache_dir = " + cachePath("embeds") + "\n\n" +
		"[markata-go.image_optimization]\ncache_dir = " + cachePath("image-optimization") + "\n\n" +
		"[markata-go.mentions]\ncache_dir = " + cachePath("mentions") + "\n\n" +
		"[markata-go.search]\nenabled = true\n\n" +
		"[markata-go.search.pagefind]\ncache_dir = " + cachePath("pagefind") + "\n\n" +
		"auto_install = false\n\n" +
		"[markata-go.tailwind]\ncache_dir = " + cachePath("tailwind") + "\n\n" +
		"[markata-go.webmentions]\ncache_dir = " + cachePath("webmentions") + "\n"
	if err := os.WriteFile(overlay, []byte(overlayData), 0o600); err != nil {
		t.Fatal(err)
	}

	binary := buildTestBinary(t)
	environment := []string{
		"MARKATA_GO_ENCRYPTION_ENABLED=false",
		"PATH=" + fakeTools + string(os.PathListSeparator) + os.Getenv("PATH"),
	}
	baseArgs := []string{"build", "--no-color", "--no-input", "-c", "markata-go.toml", "-m", "cache-overlay.toml"}
	candidateArgs := append(append([]string(nil), baseArgs...), "--dag")
	result, err := buildlab.RunScenario(context.Background(), buildlab.ScenarioRunConfig{
		Fixture: fixture,
		Scenario: buildlab.Scenario{
			ID:      "cli-dag-cache-isolation",
			Version: "1",
			Operations: []buildlab.Operation{
				{Type: buildlab.OpBuild},
				{Type: buildlab.OpReplaceExact, Path: "content/target.md", Old: "Original target content.", New: "Updated target content."},
				{Type: buildlab.OpBuild},
			},
		},
		Baseline: buildlab.BuildCommand{
			Binary: binary, Args: baseArgs, OutputDir: "output", Timeout: 5 * time.Minute,
			Env: environment,
		},
		Candidate: buildlab.BuildCommand{
			Binary: binary, Args: candidateArgs, OutputDir: "output", Timeout: 5 * time.Minute,
			Env: environment,
		},
		CheckDeterminism: true,
		GOMAXPROCS:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != buildLabPassVerdict || len(result.Checkpoints) != 2 {
		t.Fatalf("result = %+v", result)
	}
	checkpoint := result.Checkpoints[1]
	if !checkpoint.Correctness.IncrementalApplicable || !checkpoint.Correctness.IncrementalEqual {
		t.Fatalf("incremental correctness = %+v", checkpoint.Correctness)
	}
	data, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep me\n" {
		t.Fatalf("external cache sentinel changed: %q", data)
	}
	entries, err := os.ReadDir(externalCache)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(sentinel) {
		t.Fatalf("external cache contents = %+v", entries)
	}
	dotEnvEntries, err := os.ReadDir(dotEnvCache)
	if err != nil {
		t.Fatal(err)
	}
	if len(dotEnvEntries) != 1 || dotEnvEntries[0].Name() != filepath.Base(dotEnvSentinel) {
		t.Fatalf("dotenv cache contents = %+v", dotEnvEntries)
	}
}

func buildTestBinary(t *testing.T) string {
	t.Helper()
	repoRoot := moduleRoot(t)
	binary := filepath.Join(t.TempDir(), "markata-go")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "./cmd/markata-go")
	build.Dir = repoRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build markata-go: %v\n%s", err, output)
	}
	return binary
}

func requireLinuxBuildLab(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("Build Lab process groups are implemented on Linux")
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate go.mod")
		}
		dir = parent
	}
}
