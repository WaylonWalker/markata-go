package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/buildlab"
)

// This test is deliberately observational. Current main may report product
// failures for linked-target invalidation; those failures are evidence, not a
// failure of this harness. A harness failure remains a test failure.
func TestBuildLab_LinkedAndFixtureMutationsCharacterizeProduct(t *testing.T) {
	requireLinuxBuildLab(t)
	fixture := filepath.Join(moduleRoot(t), "cmd", "markata-go", "cmd", "testdata", "buildlab-site")
	binary := buildTestBinary(t)
	result, runErr := buildlab.RunScenario(context.Background(), buildlab.ScenarioRunConfig{
		Fixture: fixture,
		Scenario: buildlab.Scenario{ID: "cli-buildlab-fixture-mutations", Version: "1", Operations: []buildlab.Operation{
			{Type: buildlab.OpClearCache}, {Type: buildlab.OpClearOutput}, {Type: buildlab.OpBuild}, {Type: buildlab.OpBuild},
			{Type: buildlab.OpReplaceExact, Path: "content/target.md", Old: "Original target content.", New: "Updated target content."}, {Type: buildlab.OpBuild},
			{Type: buildlab.OpWriteFile, Path: "content/future-target.md", Content: fixtureFutureTarget}, {Type: buildlab.OpBuild},
			{Type: buildlab.OpDelete, Path: "content/target.md"}, {Type: buildlab.OpBuild},
			{Type: buildlab.OpWriteFile, Path: "content/target.md", Content: fixtureRecreatedTarget}, {Type: buildlab.OpBuild},
			{Type: buildlab.OpRename, Path: "content/target.md", Dest: "content/renamed.md"}, {Type: buildlab.OpBuild},
			{Type: buildlab.OpSetConfig, Path: "markata-go.toml", Key: "title", Value: "Changed by scenario"}, {Type: buildlab.OpBuild},
		}},
		Baseline:         buildlab.BuildCommand{Binary: binary, Args: []string{"build", "-c", "markata-go.toml"}, OutputDir: "output", Timeout: 5 * time.Minute, Env: []string{"MARKATA_GO_ENCRYPTION_ENABLED=false"}},
		Candidate:        buildlab.BuildCommand{Binary: binary, Args: []string{"build", "-c", "markata-go.toml"}, OutputDir: "output", Timeout: 5 * time.Minute, Env: []string{"MARKATA_GO_ENCRYPTION_ENABLED=false"}},
		Classes:          map[string]buildlab.OutputClass{".well-known/time": buildlab.ClassVolatile},
		CheckDeterminism: true, GOMAXPROCS: 1,
	})
	for checkpointIndex := range result.Checkpoints {
		checkpoint := &result.Checkpoints[checkpointIndex]
		observations := []buildlab.RunObservation{checkpoint.Baseline, checkpoint.CandidateClean, checkpoint.CandidateIncremental}
		if checkpoint.CandidateDeterminism != nil {
			observations = append(observations, *checkpoint.CandidateDeterminism)
		}
		for _, observation := range observations {
			if observation.FailureClass == buildlab.FailureHarness {
				t.Fatalf("harness failure at operation %d: %+v (run error: %v)", checkpoint.OperationIndex, observation, runErr)
			}
			if observation.FailureClass == buildlab.FailureProduct {
				t.Logf("observed product failure at operation %d: error=%q stderr_sha256=%s", checkpoint.OperationIndex, observation.Error, observation.StderrSHA256)
			}
		}
	}
	if runErr != nil && !hasBuildLabProductFailure(result) {
		t.Fatalf("Build Lab run error was not represented as a product failure: %v; failure_class=%q", runErr, result.FailureClass)
	}
	if result.Verdict == buildLabPassVerdict {
		t.Logf("current main passed linked-target characterization")
	} else {
		t.Logf("current main product result: verdict=%s, failure_class=%s, checkpoints=%d", result.Verdict, result.FailureClass, len(result.Checkpoints))
		for checkpointIndex := range result.Checkpoints {
			checkpoint := &result.Checkpoints[checkpointIndex]
			for _, diagnostic := range checkpoint.Diagnostics {
				t.Logf("operation %d: %s: %s", checkpoint.OperationIndex, diagnostic.Scope, diagnostic.Message)
			}
		}
	}
	if len(result.Checkpoints) != 8 {
		t.Fatalf("checkpoints = %d, want 8", len(result.Checkpoints))
	}
}

func TestBuildLabCLI_EmitsMachineReadableResultAndUsesOrdinaryBuild(t *testing.T) {
	requireLinuxBuildLab(t)
	fixture := t.TempDir()
	if err := os.WriteFile(filepath.Join(fixture, "markata-go.toml"), []byte("[markata-go]\noutput_dir = \"output\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(t.TempDir(), "build.sh")
	const scriptText = `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --dag*) exit 91 ;;
    --output|-o) out="$2"; shift 2 ;;
    --output=*) out="${1#--output=}"; shift ;;
    *) shift ;;
  esac
done
mkdir -p "$out"
`
	if err := os.WriteFile(script, []byte(scriptText), 0o700); err != nil { //nolint:gosec // The test fixture must be executable as a build command.
		t.Fatal(err)
	}
	scenario := filepath.Join(t.TempDir(), "scenario.json")
	if err := os.WriteFile(scenario, []byte(`{"id":"cli-contract","version":"1","seed":42,"operations":[{"type":"build"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	resultPath := filepath.Join(t.TempDir(), "result.json")
	command := exec.Command(buildTestBinary(t), "buildlab", "run", "--fixture", fixture, "--baseline", script, "--candidate", script, "--scenario", scenario, "--check-determinism=false", "--result", resultPath) //nolint:gosec // Test-controlled fixture and executable paths are intentional.
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("CLI failed: %v\nstderr=%s\nstdout=%s", err, stderr.String(), stdout.String())
	}
	var got, fileResult buildlab.Result
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &got); err != nil {
		t.Fatalf("stdout is not JSON: %v", err)
	}
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fileResult); err != nil {
		t.Fatal(err)
	}
	if got.Verdict != buildLabPassVerdict || got.Scenario.Seed != 42 || fileResult.Verdict != got.Verdict {
		t.Fatalf("stdout/file result = %+v / %+v", got, fileResult)
	}
}

func TestBuildLabCLI_EmitsSetupFailureAndFinalizesCPUProfile(t *testing.T) {
	requireLinuxBuildLab(t)
	binary := buildTestBinary(t)
	profile := filepath.Join(t.TempDir(), "buildlab.prof")
	missingFixture := filepath.Join(t.TempDir(), "missing-fixture")
	command := exec.Command(binary, "--cpuprofile", profile, "buildlab", "run", "--fixture", missingFixture)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err == nil {
		t.Fatal("Build Lab setup failure unexpectedly succeeded")
	}
	var result buildlab.Result
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); err != nil {
		t.Fatalf("setup failure stdout is not JSON: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	if result.Verdict != "fail" || result.FailureClass != buildlab.FailureHarness || len(result.Diagnostics) != 1 {
		t.Fatalf("setup failure result = %+v", result)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Build Lab harness (setup)")) {
		t.Fatalf("setup diagnostics missing from stderr: %s", stderr.String())
	}
	info, err := os.Stat(profile)
	if err != nil {
		t.Fatalf("CPU profile was not written: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("CPU profile is empty after failed Build Lab command")
	}
}

const fixtureFutureTarget = "---\ntitle: Future target\ndate: 2026-08-03\npublished: true\n---\n\n# Future target\n\nCreated after the initial build.\n"
const fixtureRecreatedTarget = "---\ntitle: Target post\ndate: 2026-08-01\npublished: true\n---\n\n# Target post\n\nRecreated target content.\n"

func hasBuildLabProductFailure(result buildlab.Result) bool {
	for checkpointIndex := range result.Checkpoints {
		checkpoint := &result.Checkpoints[checkpointIndex]
		observations := []buildlab.RunObservation{checkpoint.Baseline, checkpoint.CandidateClean, checkpoint.CandidateIncremental}
		if checkpoint.CandidateDeterminism != nil {
			observations = append(observations, *checkpoint.CandidateDeterminism)
		}
		for _, observation := range observations {
			if observation.FailureClass == buildlab.FailureProduct {
				return true
			}
		}
	}
	return false
}

func buildTestBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "markata-go")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	command := exec.Command("go", "build", "-o", binary, "./cmd/markata-go")
	command.Dir = moduleRoot(t)
	if output, err := command.CombinedOutput(); err != nil {
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
