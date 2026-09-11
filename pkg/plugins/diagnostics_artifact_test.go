package plugins

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/diagnostics"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
)

func TestDiagnosticsArtifactPlugin_Priority(t *testing.T) {
	plugin := NewDiagnosticsArtifactPlugin()
	if got := plugin.Priority(lifecycle.StageCleanup); got != lifecycle.PriorityLast+100 {
		t.Fatalf("Priority(StageCleanup) = %d, want %d", got, lifecycle.PriorityLast+100)
	}
	if got := plugin.Priority(lifecycle.StageWrite); got != lifecycle.PriorityDefault {
		t.Fatalf("Priority(StageWrite) = %d, want %d", got, lifecycle.PriorityDefault)
	}
}

func TestDiagnosticsArtifactPlugin_WritesAfterCompleteLifecycle(t *testing.T) {
	outputDir := t.TempDir()
	contentDir := t.TempDir()
	m := lifecycle.NewManager()
	m.SetConfig(&lifecycle.Config{
		ContentDir: contentDir,
		OutputDir:  outputDir,
		Extra: map[string]interface{}{
			"markata_version": "test-version",
			"markata_commit":  "test-commit",
		},
	})
	m.RegisterPlugin(NewDiagnosticsArtifactPlugin())
	m.SetFiles([]string{"post.md"})
	m.ContentLedger().MarkLoaded("post.md")

	if err := m.Run(); err != nil {
		t.Fatalf("Manager.Run() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outputDir, diagnostics.DefaultArtifactPath))
	if err != nil {
		t.Fatalf("read diagnostics artifact: %v", err)
	}
	artifact, err := diagnostics.ParseArtifact(data)
	if err != nil {
		t.Fatalf("ParseArtifact() error = %v", err)
	}
	if artifact.Generator.Version != "test-version" || artifact.Generator.Commit != "test-commit" {
		t.Fatalf("generator = %#v", artifact.Generator)
	}
	if len(artifact.Entries) != 1 || artifact.Entries[0].Path != "post.md" {
		t.Fatalf("entries = %#v", artifact.Entries)
	}
}

func TestDiagnosticsArtifactPlugin_SkipsIncompleteLifecycle(t *testing.T) {
	outputDir := t.TempDir()
	m := lifecycle.NewManager()
	m.Config().OutputDir = outputDir

	if err := m.RunTo(lifecycle.StageWrite); err != nil {
		t.Fatalf("RunTo(StageWrite) error = %v", err)
	}
	if err := NewDiagnosticsArtifactPlugin().Cleanup(m); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, diagnostics.DefaultArtifactPath)); !os.IsNotExist(err) {
		t.Fatalf("incomplete lifecycle created an artifact, stat error = %v", err)
	}
}

func TestDiagnosticsArtifactPlugin_PublishesBuildWithWarnings(t *testing.T) {
	outputDir := t.TempDir()
	destination := filepath.Join(outputDir, diagnostics.DefaultArtifactPath)

	m := lifecycle.NewManager()
	m.Config().OutputDir = outputDir
	m.RegisterPlugins(&diagnosticsArtifactRenderFailure{}, NewDiagnosticsArtifactPlugin())
	if err := m.Run(); err != nil {
		t.Fatalf("Manager.Run() error = %v, want warning-only completion", err)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read diagnostics artifact: %v", err)
	}
	if _, err := diagnostics.ParseArtifact(data); err != nil {
		t.Fatalf("ParseArtifact() error = %v", err)
	}
	if len(m.Warnings()) != 1 {
		t.Fatalf("warnings = %d, want 1", len(m.Warnings()))
	}
}

func TestDiagnosticsArtifactPlugin_PreservesArtifactAfterFailedBuild(t *testing.T) {
	outputDir := t.TempDir()
	destination := filepath.Join(outputDir, diagnostics.DefaultArtifactPath)
	if err := writeDiagnosticsArtifact(destination, []byte("previous artifact\n")); err != nil {
		t.Fatalf("write previous artifact: %v", err)
	}

	m := lifecycle.NewManager()
	m.Config().OutputDir = outputDir
	m.RegisterPlugins(&diagnosticsArtifactCriticalRenderFailure{}, NewDiagnosticsArtifactPlugin())
	if err := m.Run(); err == nil {
		t.Fatal("Manager.Run() succeeded after critical render failure")
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read previous artifact: %v", err)
	}
	if string(data) != "previous artifact\n" {
		t.Fatalf("artifact changed after failed build: %q", data)
	}
}

func TestDiagnosticsArtifactPlugin_SkipsNonProductionBuildModes(t *testing.T) {
	for _, mode := range []string{"fast_mode", "incremental_mode"} {
		t.Run(mode, func(t *testing.T) {
			outputDir := t.TempDir()
			m := lifecycle.NewManager()
			m.Config().OutputDir = outputDir
			m.Config().Extra[mode] = true
			m.RegisterPlugin(NewDiagnosticsArtifactPlugin())

			if err := m.Run(); err != nil {
				t.Fatalf("Manager.Run() error = %v", err)
			}
			if _, err := os.Stat(filepath.Join(outputDir, diagnostics.DefaultArtifactPath)); !os.IsNotExist(err) {
				t.Fatalf("%s build created an artifact, stat error = %v", mode, err)
			}
		})
	}
}

func TestDiagnosticsArtifactPlugin_WriteFailureIsCritical(t *testing.T) {
	outputDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outputDir, ".markata"), []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("create conflicting output path: %v", err)
	}

	m := lifecycle.NewManager()
	m.Config().OutputDir = outputDir
	m.RegisterPlugin(NewDiagnosticsArtifactPlugin())
	err := m.Run()
	if err == nil {
		t.Fatal("Manager.Run() succeeded after artifact write failure")
	}
	var hookErrors *lifecycle.HookErrors
	if !errors.As(err, &hookErrors) || !hookErrors.HasCritical() {
		t.Fatalf("error = %v, want critical hook error", err)
	}
	if m.HasRun(lifecycle.StageCleanup) {
		t.Fatal("cleanup stage was marked complete after critical artifact failure")
	}
}

func TestWriteDiagnosticsArtifact_ReplacesExistingFile(t *testing.T) {
	destination := filepath.Join(t.TempDir(), diagnostics.DefaultArtifactPath)
	if err := writeDiagnosticsArtifact(destination, []byte("old\n")); err != nil {
		t.Fatalf("write old artifact: %v", err)
	}
	if err := writeDiagnosticsArtifact(destination, []byte("new\n")); err != nil {
		t.Fatalf("write new artifact: %v", err)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if string(data) != "new\n" {
		t.Fatalf("artifact = %q, want new content", data)
	}
}

func TestWriteDiagnosticsArtifact_RemovesTemporaryFileAfterReplacementFailure(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, diagnostics.DefaultArtifactPath)
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatalf("create conflicting artifact directory: %v", err)
	}

	if err := writeDiagnosticsArtifact(destination, []byte("artifact")); err == nil {
		t.Fatal("writeDiagnosticsArtifact() succeeded with a directory destination")
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(destination), ".diagnostics-*.tmp"))
	if err != nil {
		t.Fatalf("find temporary artifacts: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary artifacts remain after failed replacement: %v", matches)
	}
}

type diagnosticsArtifactRenderFailure struct{}

func (p *diagnosticsArtifactRenderFailure) Name() string {
	return "diagnostics_artifact_test_failure"
}

func (p *diagnosticsArtifactRenderFailure) Render(_ *lifecycle.Manager) error {
	return errors.New("expected render warning")
}

var _ lifecycle.RenderPlugin = (*diagnosticsArtifactRenderFailure)(nil)

type diagnosticsArtifactCriticalRenderFailure struct{}

func (p *diagnosticsArtifactCriticalRenderFailure) Name() string {
	return "diagnostics_artifact_test_critical_failure"
}

func (p *diagnosticsArtifactCriticalRenderFailure) Render(_ *lifecycle.Manager) error {
	return diagnosticsArtifactCriticalError{}
}

type diagnosticsArtifactCriticalError struct{}

func (diagnosticsArtifactCriticalError) Error() string {
	return "expected critical render failure"
}

func (diagnosticsArtifactCriticalError) IsCritical() bool {
	return true
}

var _ lifecycle.RenderPlugin = (*diagnosticsArtifactCriticalRenderFailure)(nil)
var _ lifecycle.CriticalError = diagnosticsArtifactCriticalError{}
