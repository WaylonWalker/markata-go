package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/buildlab"
	"github.com/spf13/cobra"
)

var (
	buildLabFixture          string
	buildLabBaseline         string
	buildLabCandidate        string
	buildLabBuildConfig      string
	buildLabScenario         string
	buildLabResult           string
	buildLabSeed             int64
	buildLabTimeout          time.Duration
	buildLabGOMAXPROCS       int
	buildLabCheckDeterminism bool
	buildLabVolatile         string
	buildLabFast             bool
	buildLabCandidateDAG     bool
	buildLabDAGRandom        bool
	buildLabExternalTools    string
	buildLabEnvironment      string
)

const buildLabPassVerdict = "pass"

var buildLabCmd = &cobra.Command{
	Use:   "buildlab",
	Short: "Run isolated baseline and incremental build comparisons",
}

var buildLabRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a Build Lab scenario",
	Args:  cobra.NoArgs,
	RunE:  runBuildLabCommand,
}

func init() {
	rootCmd.AddCommand(buildLabCmd)
	buildLabCmd.AddCommand(buildLabRunCmd)

	flags := buildLabRunCmd.Flags()
	flags.StringVar(&buildLabFixture, "fixture", "", "site fixture directory")
	flags.StringVar(&buildLabBaseline, "baseline", "", "baseline markata-go binary (defaults to this binary)")
	flags.StringVar(&buildLabCandidate, "candidate", "", "candidate markata-go binary (defaults to this binary)")
	flags.StringVar(&buildLabBuildConfig, "build-config", "markata-go.toml", "config path relative to the fixture")
	flags.StringVar(&buildLabScenario, "scenario", "", "scenario JSON file; defaults to a clean build")
	flags.StringVar(&buildLabResult, "result", "", "write structured result JSON to this path")
	flags.Int64Var(&buildLabSeed, "seed", 1, "scenario seed")
	flags.DurationVar(&buildLabTimeout, "timeout", 10*time.Minute, "per-build timeout")
	flags.IntVar(&buildLabGOMAXPROCS, "gomaxprocs", 1, "GOMAXPROCS used by isolated builds")
	flags.BoolVar(&buildLabCheckDeterminism, "check-determinism", true, "compare two clean candidate builds")
	flags.StringVar(&buildLabVolatile, "volatile", ".well-known/time", "comma-separated output paths whose bytes may vary")
	flags.BoolVar(&buildLabFast, "fast", false, "add --fast to baseline and candidate builds")
	flags.BoolVar(&buildLabCandidateDAG, "candidate-dag", true, "add --dag to candidate builds")
	flags.BoolVar(&buildLabDAGRandom, "candidate-dag-random-ready", false, "add --dag-random-ready to candidate builds")
	flags.StringVar(&buildLabExternalTools, "tool-version", "", "comma-separated tool=version metadata to record")
	flags.StringVar(&buildLabEnvironment, "env", "", "comma-separated non-secret KEY=value entries for each build")
}

//nolint:gocyclo // This command intentionally keeps CLI parsing and execution in one reportable flow.
func runBuildLabCommand(_ *cobra.Command, _ []string) error {
	if buildLabFixture == "" {
		return fmt.Errorf("--fixture is required")
	}
	baseline, err := absoluteCommandPath(buildLabBaseline)
	if err != nil {
		return fmt.Errorf("baseline binary: %w", err)
	}
	candidate, err := absoluteCommandPath(buildLabCandidate)
	if err != nil {
		return fmt.Errorf("candidate binary: %w", err)
	}
	scenario, err := loadBuildLabScenario(buildLabScenario)
	if err != nil {
		return err
	}
	scenario.Seed = buildLabSeed
	if scenario.ID == "" {
		scenario.ID = "cli-smoke"
	}
	if scenario.Version == "" {
		scenario.Version = "1"
	}

	baselineArgs := []string{"build"}
	if buildLabFast {
		baselineArgs = append(baselineArgs, "--fast")
	}
	if buildLabBuildConfig != "" {
		baselineArgs = append(baselineArgs, "-c", buildLabBuildConfig)
	}
	for _, merge := range mergeConfigFiles {
		if merge != "" {
			baselineArgs = append(baselineArgs, "-m", merge)
		}
	}
	candidateArgs := append([]string(nil), baselineArgs...)
	if buildLabCandidateDAG {
		candidateArgs = append(candidateArgs, "--dag")
		if buildLabDAGRandom {
			candidateArgs = append(candidateArgs, "--dag-random-ready", "--dag-seed", fmt.Sprint(buildLabSeed))
		}
	}
	classes := make(map[string]buildlab.OutputClass)
	for _, path := range strings.Split(buildLabVolatile, ",") {
		path = strings.TrimSpace(path)
		if path != "" {
			classes[path] = buildlab.ClassVolatile
		}
	}
	externalTools, err := parseBuildLabToolVersions(buildLabExternalTools)
	if err != nil {
		return err
	}
	environment, err := parseBuildLabEnvironment(buildLabEnvironment)
	if err != nil {
		return err
	}
	result, runErr := buildlab.RunScenario(context.Background(), buildlab.ScenarioRunConfig{
		Fixture:          buildLabFixture,
		Scenario:         scenario,
		Baseline:         buildlab.BuildCommand{Binary: baseline, Args: baselineArgs, OutputDir: "output", Timeout: buildLabTimeout, Env: environment},
		Candidate:        buildlab.BuildCommand{Binary: candidate, Args: candidateArgs, OutputDir: "output", Timeout: buildLabTimeout, Env: environment},
		Classes:          classes,
		CheckDeterminism: buildLabCheckDeterminism,
		GOMAXPROCS:       buildLabGOMAXPROCS,
		ExternalTools:    externalTools,
	})

	if buildLabResult != "" {
		if err := buildlab.WriteResult(buildLabResult, result); err != nil {
			return fmt.Errorf("write Build Lab result: %w", err)
		}
	}
	data, marshalErr := result.CanonicalJSON()
	if marshalErr != nil {
		return marshalErr
	}
	fmt.Fprintln(outWriter(), string(data))
	if runErr != nil {
		return runErr
	}
	if result.Verdict != buildLabPassVerdict {
		return fmt.Errorf("build lab verdict: %s", result.Verdict)
	}
	return nil
}

func parseBuildLabToolVersions(value string) (map[string]string, error) {
	tools := make(map[string]string)
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key, version, ok := strings.Cut(item, "=")
		key = strings.TrimSpace(key)
		version = strings.TrimSpace(version)
		if !ok || key == "" || version == "" {
			return nil, fmt.Errorf("invalid --tool-version value %q; use name=version", item)
		}
		tools[key] = version
	}
	return tools, nil
}

func parseBuildLabEnvironment(value string) ([]string, error) {
	entries := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key, envValue, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid --env value %q; use KEY=value", item)
		}
		entries = append(entries, strings.TrimSpace(key)+"="+envValue)
	}
	return entries, nil
}

func absoluteCommandPath(value string) (string, error) {
	if value == "" {
		value, err := exec.LookPath(os.Args[0])
		if err != nil {
			return "", err
		}
		return filepath.Abs(value)
	}
	if !strings.ContainsRune(value, filepath.Separator) {
		if path, err := exec.LookPath(value); err == nil {
			return filepath.Abs(path)
		}
	}
	path, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	return path, nil
}

func loadBuildLabScenario(path string) (buildlab.Scenario, error) {
	if path == "" {
		return buildlab.Scenario{
			ID:      "cli-smoke",
			Version: "1",
			Operations: []buildlab.Operation{
				{Type: buildlab.OpClearCache},
				{Type: buildlab.OpClearOutput},
				{Type: buildlab.OpBuild},
				{Type: buildlab.OpBuild},
			},
		}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return buildlab.Scenario{}, fmt.Errorf("read scenario: %w", err)
	}
	var scenario buildlab.Scenario
	if err := json.Unmarshal(data, &scenario); err != nil {
		return buildlab.Scenario{}, fmt.Errorf("parse scenario: %w", err)
	}
	return scenario, nil
}
