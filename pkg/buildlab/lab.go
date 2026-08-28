package buildlab

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/runtimeenv"
)

const ResultSchemaVersion = 1

const (
	verdictFail = "fail"
	verdictPass = "pass"
)

// BuildCommand describes one real Markata-Go process.  Binary should normally
// be an absolute path to a built executable so the baseline and candidate
// identity can be recorded without relying on process state.
type BuildCommand struct {
	Binary    string
	Args      []string
	OutputDir string
	Timeout   time.Duration
	Env       []string
}

// Workspace is an isolated copy of a fixture and its process-local state.
// IsolationConfig redirects built-in Markata-Go cache settings into this
// workspace and is appended after fixture-provided merge configs.
type Workspace struct {
	Root            string
	SiteDir         string
	HomeDir         string
	XDGCacheDir     string
	TempDir         string
	MarkataCache    string
	IsolationConfig string
}

// NewWorkspace copies source inputs into a private site and creates the
// directories used by home, XDG cache, temporary files, and Markata cache
// state. Known generated build state is not copied.
func NewWorkspace(fixture, parent string) (Workspace, error) {
	return newWorkspace(fixture, parent, buildLabFixtureExclusions())
}

func newWorkspace(fixture, parent string, excluded []string) (Workspace, error) {
	if fixture == "" {
		return Workspace{}, fmt.Errorf("fixture path is empty")
	}
	if parent == "" {
		parent = os.TempDir()
	}
	root, err := os.MkdirTemp(parent, "markata-buildlab-")
	if err != nil {
		return Workspace{}, fmt.Errorf("create workspace: %w", err)
	}
	fail := func(e error) (Workspace, error) {
		_ = os.RemoveAll(root)
		return Workspace{}, e
	}
	w := Workspace{
		Root:            root,
		SiteDir:         filepath.Join(root, "site"),
		HomeDir:         filepath.Join(root, "home"),
		XDGCacheDir:     filepath.Join(root, "xdg-cache"),
		TempDir:         filepath.Join(root, "tmp"),
		MarkataCache:    filepath.Join(root, "markata-cache"),
		IsolationConfig: filepath.Join(root, "buildlab-isolation.json"),
	}
	if err := copyDir(fixture, w.SiteDir, excluded); err != nil {
		return fail(fmt.Errorf("copy fixture: %w", err))
	}
	for _, dir := range []string{w.HomeDir, w.XDGCacheDir, w.TempDir, w.MarkataCache} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fail(fmt.Errorf("create workspace state %s: %w", dir, err))
		}
	}
	if err := writeIsolationConfig(w.IsolationConfig, w.MarkataCache); err != nil {
		return fail(fmt.Errorf("write workspace isolation config: %w", err))
	}
	return w, nil
}

func buildLabFixtureExclusions(commands ...BuildCommand) []string {
	excluded := []string{".markata", ".markata.cache", ".markata-cache", "cache", "output"}
	for _, command := range commands {
		output := command.OutputDir
		if output == "" {
			output = outputArg(command.Args)
		}
		if output == "" || filepath.IsAbs(output) {
			continue
		}
		clean := filepath.Clean(filepath.FromSlash(output))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			continue
		}
		excluded = append(excluded, clean)
	}
	return excluded
}

func writeIsolationConfig(path, cacheDir string) error {
	pluginCache := func(name string) string {
		return filepath.Join(cacheDir, name)
	}
	data, err := json.Marshal(map[string]any{
		"markata-go": map[string]any{
			"cache_dir": cacheDir,
			"assets": map[string]any{
				"cache_dir": pluginCache("assets"),
			},
			"blogroll": map[string]any{
				"cache_dir": pluginCache("blogroll"),
			},
			"embeds": map[string]any{
				"cache_dir": pluginCache("embeds"),
			},
			"image_optimization": map[string]any{
				"cache_dir": pluginCache("image-optimization"),
			},
			"mentions": map[string]any{
				"cache_dir": pluginCache("mentions"),
			},
			"search": map[string]any{
				"pagefind": map[string]any{
					"cache_dir": pluginCache("pagefind"),
				},
			},
			"tailwind": map[string]any{
				"cache_dir": pluginCache("tailwind"),
			},
			"webmentions": map[string]any{
				"cache_dir": pluginCache("webmentions"),
			},
		},
	})
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

// Remove deletes all files created for the workspace.
func (w Workspace) Remove() error {
	if w.Root == "" {
		return nil
	}
	return os.RemoveAll(w.Root)
}

func (w Workspace) Environment(extra []string, gomaxprocs int) []string {
	values := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok && !isolatedEnvironmentKey(key) && !sensitiveEnvironmentKey(key) {
			values[key] = value
		}
	}
	for _, entry := range extra {
		key, value, ok := strings.Cut(entry, "=")
		if ok && key != "" && !isolatedEnvironmentKey(key) && !sensitiveEnvironmentKey(key) {
			values[key] = value
		}
	}
	values["HOME"] = w.HomeDir
	values["XDG_CACHE_HOME"] = w.XDGCacheDir
	values["TMPDIR"] = w.TempDir
	values["TMP"] = w.TempDir
	values["TEMP"] = w.TempDir
	values["TZ"] = "UTC"
	values["LANG"] = "C.UTF-8"
	values["LC_ALL"] = "C.UTF-8"
	values["SOURCE_DATE_EPOCH"] = "0"
	values[runtimeenv.EnvDisableDotEnv] = "1"
	if gomaxprocs > 0 {
		values["GOMAXPROCS"] = strconv.Itoa(gomaxprocs)
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	env := make([]string, 0, len(keys))
	for _, key := range keys {
		env = append(env, key+"="+values[key])
	}
	return env
}

func isolatedEnvironmentKey(key string) bool {
	upper := strings.ToUpper(key)
	switch upper {
	case "HOME", "XDG_CACHE_HOME", "TMPDIR", "TMP", "TEMP", "TZ", "LANG", "LC_ALL", "SOURCE_DATE_EPOCH", "GOMAXPROCS", runtimeenv.EnvDisableDotEnv, "MARKATA_GO_SITE_DIR":
		return true
	}
	return strings.HasPrefix(upper, "MARKATA_GO_") && strings.Contains(upper, "CACHE")
}

func sensitiveEnvironmentKey(key string) bool {
	upper := strings.ToUpper(key)
	for _, marker := range []string{"API_KEY", "_KEY", "TOKEN", "SECRET", "PASSWORD", "CREDENTIAL", "COOKIE", "AUTH"} {
		if strings.Contains(upper, marker) {
			return true
		}
	}
	return false
}

func (c BuildCommand) run(ctx context.Context, w Workspace, clean bool, gomaxprocs int) RunResult {
	if c.Binary == "" {
		return RunResult{Err: fmt.Errorf("build command binary is empty")}
	}
	args, err := c.args(w, clean)
	if err != nil {
		return RunResult{Err: err}
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	return Run(ctx, RunConfig{
		Command: c.Binary,
		Args:    args,
		CWD:     w.SiteDir,
		Env:     w.Environment(c.Env, gomaxprocs),
		Timeout: timeout,
	})
}

func (c BuildCommand) args(w Workspace, clean bool) (args []string, err error) {
	output, err := c.outputPath(w)
	if err != nil {
		return nil, err
	}
	args = append([]string(nil), c.Args...)
	// The isolation config must follow all fixture and user merge configs so
	// its private cache directory cannot be overridden by the fixture.
	if w.IsolationConfig != "" {
		args = append(args, "--merge-config", w.IsolationConfig)
	}
	// Always override the command's output path with a workspace-local path.
	// This prevents a copied fixture containing an absolute output_dir from
	// causing baseline and candidate processes to write into the same tree.
	// Bind site selection to the copied workspace. This is defense in depth
	// against both caller-provided flags and ambient site-directory settings.
	args = append(args, "--output", output, "--site-dir", w.SiteDir)
	if clean && !hasArg(args, "--clean") {
		args = append(args, "--clean")
	}
	return args, nil
}

func (c BuildCommand) outputPath(w Workspace) (string, error) {
	output := c.OutputDir
	if output == "" {
		output = outputArg(c.Args)
	}
	if output == "" {
		output = "output"
	}
	if filepath.IsAbs(output) {
		clean := filepath.Clean(output)
		if err := validateOutputPath(w.SiteDir, clean); err != nil {
			return "", fmt.Errorf("build output directory %q escapes workspace", output)
		}
		return clean, nil
	}
	clean := filepath.Clean(filepath.Join(w.SiteDir, output))
	if err := validateOutputPath(w.SiteDir, clean); err != nil {
		return "", fmt.Errorf("build output directory %q escapes workspace", output)
	}
	return clean, nil
}

func validateOutputPath(root, output string) error {
	rel, err := filepath.Rel(root, output)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("output path escapes workspace")
	}
	part := root
	for _, component := range strings.Split(rel, string(filepath.Separator)) {
		if component == "." || component == "" {
			continue
		}
		part = filepath.Join(part, component)
		info, err := os.Lstat(part)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output path enters symlink %q", part)
		}
	}
	return nil
}

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

type BuildIdentity struct {
	Binary       string `json:"binary"`
	BinarySHA256 string `json:"binary_sha256,omitempty"`
	GitSHA       string `json:"git_sha,omitempty"`
}

type FixtureIdentity struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type EnvironmentIdentity struct {
	Timezone             string            `json:"timezone"`
	Locale               string            `json:"locale"`
	SourceDateEpoch      string            `json:"source_date_epoch"`
	GOMAXPROCS           int               `json:"gomaxprocs"`
	GOOS                 string            `json:"goos"`
	GOARCH               string            `json:"goarch"`
	GoVersion            string            `json:"go_version"`
	ExternalToolVersions map[string]string `json:"external_tool_versions,omitempty"`
}

func defaultEnvironment(gomaxprocs int, tools map[string]string) EnvironmentIdentity {
	if gomaxprocs <= 0 {
		gomaxprocs = runtime.GOMAXPROCS(0)
	}
	copyTools := make(map[string]string, len(tools))
	for key, value := range tools {
		copyTools[key] = value
	}
	return EnvironmentIdentity{
		Timezone: "UTC", Locale: "C.UTF-8", SourceDateEpoch: "0",
		GOMAXPROCS: gomaxprocs, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
		GoVersion: runtime.Version(), ExternalToolVersions: copyTools,
	}
}

type RunObservation struct {
	ExitCode        int     `json:"exit_code"`
	Successful      bool    `json:"successful"`
	TimedOut        bool    `json:"timed_out"`
	DurationSeconds float64 `json:"duration_seconds"`
	ManifestDigest  string  `json:"manifest_digest,omitempty"`
	StdoutSHA256    string  `json:"stdout_sha256,omitempty"`
	StderrSHA256    string  `json:"stderr_sha256,omitempty"`
	Error           string  `json:"error,omitempty"`
}

type CorrectnessResult struct {
	DifferentialEqual     bool         `json:"differential_equal"`
	IncrementalEqual      bool         `json:"incremental_equal"`
	IncrementalApplicable bool         `json:"incremental_applicable"`
	DeterministicEqual    bool         `json:"deterministic_equal"`
	DifferentialDiff      ManifestDiff `json:"differential_diff,omitempty"`
	IncrementalDiff       ManifestDiff `json:"incremental_diff,omitempty"`
	DeterminismDiff       ManifestDiff `json:"determinism_diff,omitempty"`
}

type Checkpoint struct {
	OperationIndex       int               `json:"operation_index"`
	Operation            string            `json:"operation"`
	Baseline             RunObservation    `json:"baseline"`
	CandidateClean       RunObservation    `json:"candidate_clean"`
	CandidateDeterminism *RunObservation   `json:"candidate_determinism,omitempty"`
	CandidateIncremental RunObservation    `json:"candidate_incremental"`
	Correctness          CorrectnessResult `json:"correctness"`
}

// Result is stable, machine-readable evidence from a scenario run.  Timing
// and process hashes are observations; correctness fields are the gate.
type Result struct {
	SchemaVersion int                 `json:"schema_version"`
	Baseline      BuildIdentity       `json:"baseline"`
	Candidate     BuildIdentity       `json:"candidate"`
	Fixture       FixtureIdentity     `json:"fixture"`
	Scenario      Scenario            `json:"scenario"`
	Environment   EnvironmentIdentity `json:"environment"`
	Checkpoints   []Checkpoint        `json:"checkpoints"`
	Verdict       string              `json:"verdict"`
}

// CanonicalJSON returns deterministic JSON for a result with checkpoint order
// preserved from the declarative scenario.
func (r Result) CanonicalJSON() ([]byte, error) {
	if r.SchemaVersion == 0 {
		r.SchemaVersion = ResultSchemaVersion
	}
	return json.Marshal(r)
}

func (r Result) Digest() (string, error) {
	b, err := r.CanonicalJSON()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// ScenarioRunConfig describes a baseline/candidate comparison.
type ScenarioRunConfig struct {
	Fixture          string
	Scenario         Scenario
	Baseline         BuildCommand
	Candidate        BuildCommand
	ParentDir        string
	Classes          map[string]OutputClass
	Comparators      map[string]func(FileRecord, FileRecord) bool
	CheckDeterminism bool
	GOMAXPROCS       int
	ExternalTools    map[string]string
}

// RunScenario executes each build operation and compares all three required
// paths.  A failed command is represented in Result and makes the verdict
// fail; callers can persist the result even when a candidate is broken.
//
//nolint:gocyclo // Scenario validation and checkpoint orchestration are one public operation.
func RunScenario(ctx context.Context, cfg ScenarioRunConfig) (result Result, runErr error) {
	if cfg.Fixture == "" {
		return Result{}, fmt.Errorf("fixture path is empty")
	}
	if cfg.Scenario.Version == "" {
		cfg.Scenario.Version = "1"
	}
	result = Result{
		SchemaVersion: ResultSchemaVersion,
		Baseline:      identityForCommand(cfg.Baseline),
		Candidate:     identityForCommand(cfg.Candidate),
		Fixture:       FixtureIdentity{Path: cfg.Fixture},
		Scenario:      cfg.Scenario,
		Environment:   defaultEnvironment(cfg.GOMAXPROCS, cfg.ExternalTools),
		Checkpoints:   []Checkpoint{},
		Verdict:       verdictFail,
	}
	fixtureManifest, err := BuildManifest(cfg.Fixture, nil)
	if err != nil {
		return result, fmt.Errorf("fixture manifest: %w", err)
	}
	fixtureDigest, err := fixtureManifest.Digest()
	if err != nil {
		return result, fmt.Errorf("fixture digest: %w", err)
	}
	result.Fixture.Digest = fixtureDigest
	excluded := buildLabFixtureExclusions(cfg.Baseline, cfg.Candidate)
	source, err := newWorkspace(cfg.Fixture, cfg.ParentDir, excluded)
	if err != nil {
		return result, err
	}
	defer func() {
		if err := cleanupWorkspace(source); err != nil {
			result.Verdict = verdictFail
			runErr = errors.Join(runErr, err)
		}
	}()

	builtOnce := false
	pendingMutation := false
	pendingMutationIndex := -1
	for operationIndex := range cfg.Scenario.Operations {
		operation := cfg.Scenario.Operations[operationIndex]
		if operation.Type != OpBuild {
			if err := applyWorkspaceOperation(source, operation, cfg.Candidate); err != nil {
				return result, &ScenarioError{Operation: operationIndex, Err: err}
			}
			if isSemanticMutation(operation.Type) {
				if pendingMutation {
					return result, &ScenarioError{Operation: operationIndex, Err: fmt.Errorf("mutation must be followed by a build")}
				}
				pendingMutation = true
				pendingMutationIndex = operationIndex
			}
			continue
		}

		checkpoint, checkpointErr := runScenarioCheckpoint(ctx, cfg, source, operationIndex, operation, builtOnce, result.Environment.GOMAXPROCS)
		result.Checkpoints = append(result.Checkpoints, checkpoint)
		if checkpointErr != nil {
			return result, &ScenarioError{Operation: operationIndex, Err: checkpointErr}
		}
		builtOnce = true
		pendingMutation = false
		pendingMutationIndex = -1
	}
	if pendingMutation {
		return result, &ScenarioError{Operation: pendingMutationIndex, Err: fmt.Errorf("scenario ends with a mutation without a following build")}
	}
	if len(result.Checkpoints) == 0 {
		return result, fmt.Errorf("scenario contains no build operation")
	}
	result.Verdict = verdictPass
	for i := range result.Checkpoints {
		checkpoint := &result.Checkpoints[i]
		if !checkpoint.Baseline.Successful || !checkpoint.CandidateClean.Successful ||
			!checkpoint.Correctness.DifferentialEqual || !checkpoint.Correctness.DeterministicEqual ||
			(checkpoint.Correctness.IncrementalApplicable &&
				(!checkpoint.CandidateIncremental.Successful || !checkpoint.Correctness.IncrementalEqual)) {
			result.Verdict = verdictFail
			break
		}
	}
	return result, nil
}

// runScenarioCheckpoint compares the candidate's incremental output with
// clean baseline and candidate builds for one build operation. Derived clean
// workspaces are released after each manifest is captured so large generated
// output trees do not accumulate across the checkpoint.
func runScenarioCheckpoint(
	ctx context.Context,
	cfg ScenarioRunConfig,
	source Workspace,
	operationIndex int,
	operation Operation,
	incrementalApplicable bool,
	gomaxprocs int,
) (checkpoint Checkpoint, runErr error) {
	var incremental RunResult
	if incrementalApplicable {
		incremental = cfg.Candidate.run(ctx, source, false, gomaxprocs)
	} else {
		incremental = cfg.Candidate.run(ctx, source, true, gomaxprocs)
	}
	incrementalManifest, incrementalManifestErr := manifestForRun(source, cfg.Candidate, cfg.Classes)
	incrementalObservation := observe(incremental, incrementalManifest, incrementalManifestErr)
	if !incrementalApplicable {
		incrementalObservation = RunObservation{}
	}

	excluded := buildLabFixtureExclusions(cfg.Baseline, cfg.Candidate)
	baselineObservation, baselineManifest, err := runCleanWorkspace(ctx, source.SiteDir, cfg.ParentDir, excluded, cfg.Baseline, cfg.Classes, gomaxprocs)
	if err != nil {
		return checkpoint, err
	}
	candidateCleanObservation, candidateCleanManifest, err := runCleanWorkspace(ctx, source.SiteDir, cfg.ParentDir, excluded, cfg.Candidate, cfg.Classes, gomaxprocs)
	if err != nil {
		return checkpoint, err
	}

	correctness := CorrectnessResult{IncrementalApplicable: incrementalApplicable}
	if baselineObservation.Successful && candidateCleanObservation.Successful {
		correctness.DifferentialDiff = CompareManifests(baselineManifest, candidateCleanManifest, cfg.Comparators)
		correctness.DifferentialEqual = correctness.DifferentialDiff.Equal()
	}
	if incrementalApplicable && incremental.Successful() && candidateCleanObservation.Successful {
		correctness.IncrementalDiff = CompareManifests(candidateCleanManifest, incrementalManifest, cfg.Comparators)
		correctness.IncrementalEqual = correctness.IncrementalDiff.Equal()
	}
	correctness.DeterministicEqual = !cfg.CheckDeterminism
	var determinismObservation *RunObservation
	if cfg.CheckDeterminism {
		observation, determinismManifest, err := runCleanWorkspace(ctx, source.SiteDir, cfg.ParentDir, excluded, cfg.Candidate, cfg.Classes, gomaxprocs)
		if err != nil {
			return checkpoint, err
		}
		determinismObservation = &observation
		if candidateCleanObservation.Successful && observation.Successful {
			correctness.DeterminismDiff = CompareManifests(candidateCleanManifest, determinismManifest, cfg.Comparators)
			correctness.DeterministicEqual = correctness.DeterminismDiff.Equal()
		}
	}
	checkpoint = Checkpoint{
		OperationIndex: operationIndex, Operation: string(operation.Type),
		Baseline: baselineObservation, CandidateClean: candidateCleanObservation,
		CandidateDeterminism: determinismObservation, CandidateIncremental: incrementalObservation,
		Correctness: correctness,
	}
	return checkpoint, nil
}

func runCleanWorkspace(
	ctx context.Context,
	fixture, parent string,
	excluded []string,
	command BuildCommand,
	classes map[string]OutputClass,
	gomaxprocs int,
) (observation RunObservation, manifest Manifest, runErr error) {
	workspace, err := newWorkspace(fixture, parent, excluded)
	if err != nil {
		return observation, manifest, err
	}
	defer func() {
		runErr = errors.Join(runErr, cleanupWorkspace(workspace))
	}()
	run := command.run(ctx, workspace, true, gomaxprocs)
	manifest, manifestErr := manifestForRun(workspace, command, classes)
	observation = observe(run, manifest, manifestErr)
	return observation, manifest, nil
}

func isSemanticMutation(operation OperationType) bool {
	switch operation {
	case OpWriteFile, OpReplaceExact, OpDelete, OpRename, OpCopy, OpSetConfig, OpTouch:
		return true
	default:
		return false
	}
}

func cleanupWorkspace(w Workspace) error {
	if err := w.Remove(); err != nil {
		return fmt.Errorf("remove Build Lab workspace %q: %w", w.Root, err)
	}
	return nil
}

func applyWorkspaceOperation(w Workspace, operation Operation, outputCommand BuildCommand) error {
	if operation.Type == OpClearOutput {
		output, err := outputCommand.outputPath(w)
		if err != nil {
			return err
		}
		if err := os.RemoveAll(output); err != nil {
			return fmt.Errorf("clear workspace output: %w", err)
		}
		return nil
	}
	if err := ApplyOperation(w.SiteDir, operation); err != nil {
		return err
	}
	if operation.Type != OpClearCache || w.MarkataCache == "" {
		return nil
	}
	if err := os.RemoveAll(w.MarkataCache); err != nil {
		return fmt.Errorf("clear workspace cache: %w", err)
	}
	if err := os.MkdirAll(w.MarkataCache, 0o755); err != nil {
		return fmt.Errorf("recreate workspace cache: %w", err)
	}
	return nil
}

func identityForCommand(command BuildCommand) BuildIdentity {
	identity := BuildIdentity{Binary: command.Binary}
	if hash, err := hashFile(command.Binary); err == nil {
		identity.BinarySHA256 = hash
	}
	return identity
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func manifestForRun(w Workspace, command BuildCommand, classes map[string]OutputClass) (Manifest, error) {
	output, err := command.outputPath(w)
	if err != nil {
		return Manifest{Records: []FileRecord{}}, err
	}
	if info, statErr := os.Stat(output); statErr != nil {
		return Manifest{Records: []FileRecord{}}, fmt.Errorf("output directory %q: %w", output, statErr)
	} else if !info.IsDir() {
		return Manifest{Records: []FileRecord{}}, fmt.Errorf("output path %q is not a directory", output)
	}
	return BuildManifest(output, classes)
}

func outputArg(args []string) string {
	for i, arg := range args {
		if (arg == "--output" || arg == "-o") && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(arg, "--output=") {
			return strings.TrimPrefix(arg, "--output=")
		}
	}
	return ""
}

func observe(run RunResult, manifest Manifest, manifestErr error) RunObservation {
	digest := ""
	if value, err := manifest.Digest(); err == nil {
		digest = value
	}
	if manifestErr != nil {
		if run.Err == nil {
			run.Err = manifestErr
		} else {
			run.Err = errors.Join(run.Err, manifestErr)
		}
	}
	return RunObservation{
		ExitCode: run.ExitCode, Successful: run.Successful(), TimedOut: run.TimedOut,
		DurationSeconds: run.Duration.Seconds(), ManifestDigest: digest,
		StdoutSHA256: hashBytes(run.Stdout), StderrSHA256: hashBytes(run.Stderr),
		Error: errorString(run.Err),
	}
}

func hashBytes(value []byte) string {
	h := sha256.Sum256(value)
	return hex.EncodeToString(h[:])
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// WriteResult writes canonical result JSON and creates its parent directory.
func WriteResult(path string, result Result) error {
	data, err := result.CanonicalJSON()
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
