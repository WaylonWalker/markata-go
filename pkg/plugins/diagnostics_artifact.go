package plugins

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/diagnostics"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/sourcegit"
)

const diagnosticsArtifactSourceTimeout = 2 * time.Second

const diagnosticsArtifactDefaultOutputDir = "output"

// DiagnosticsArtifactPlugin persists the manager-owned content diagnostics
// snapshot after a complete successful lifecycle run.
type DiagnosticsArtifactPlugin struct{}

// NewDiagnosticsArtifactPlugin creates a diagnostics artifact plugin.
func NewDiagnosticsArtifactPlugin() *DiagnosticsArtifactPlugin {
	return &DiagnosticsArtifactPlugin{}
}

// Name returns the unique plugin name.
func (p *DiagnosticsArtifactPlugin) Name() string {
	return "diagnostics_artifact"
}

// Priority runs after other cleanup plugins, including Pagefind.
func (p *DiagnosticsArtifactPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageCleanup {
		return lifecycle.PriorityLast + 100
	}
	return lifecycle.PriorityDefault
}

// Cleanup writes the diagnostics artifact. Source Git metadata is optional:
// an unavailable Git checkout does not make an otherwise successful build
// fail or cause fabricated revision data to be published.
func (p *DiagnosticsArtifactPlugin) Cleanup(m *lifecycle.Manager) error {
	if m == nil || m.Config() == nil {
		return nil
	}
	// Only manager-driven cleanup after all prior stages may publish. This also
	// prevents a direct Cleanup call after a partial RunTo from publishing.
	if !diagnosticsLifecycleSucceeded(m) {
		return nil
	}

	config := m.Config()
	if diagnosticsNonProductionMode(config) {
		return nil
	}
	outputDir := config.OutputDir
	if outputDir == "" {
		outputDir = diagnosticsArtifactDefaultOutputDir
	}

	data, err := diagnostics.MarshalArtifact(m.ContentDiagnostics(), diagnostics.ArtifactBuildInfo{
		MarkataVersion: artifactConfigString(config, "markata_version"),
		MarkataCommit:  artifactConfigString(config, "markata_commit"),
		SourceCommit:   diagnosticsArtifactSourceCommit(config.ContentDir),
		BuiltAt:        time.Now().UTC(),
	})
	if err != nil {
		return &diagnosticsArtifactError{err: fmt.Errorf("marshal diagnostics artifact: %w", err)}
	}

	destination := filepath.Join(outputDir, diagnostics.DefaultArtifactPath)
	if err := writeDiagnosticsArtifact(destination, data); err != nil {
		return &diagnosticsArtifactError{err: err}
	}
	return nil
}

func diagnosticsNonProductionMode(config *lifecycle.Config) bool {
	if config == nil || config.Extra == nil {
		return false
	}
	fast, fastOK := config.Extra["fast_mode"].(bool)
	incremental, incrementalOK := config.Extra["incremental_mode"].(bool)
	return (fastOK && fast) || (incrementalOK && incremental)
}

func diagnosticsLifecycleSucceeded(m *lifecycle.Manager) bool {
	if m.CurrentStage() != lifecycle.StageCleanup {
		return false
	}
	for _, stage := range []lifecycle.Stage{
		lifecycle.StageConfigure,
		lifecycle.StageValidate,
		lifecycle.StageGlob,
		lifecycle.StageLoad,
		lifecycle.StageTransform,
		lifecycle.StageRender,
		lifecycle.StageCollect,
		lifecycle.StageWrite,
	} {
		if !m.HasRun(stage) {
			return false
		}
	}
	// Critical errors stop RunTo before cleanup. Non-critical hook warnings do
	// not prevent the lifecycle from completing and remain useful context in
	// the published diagnostics snapshot.
	return true
}

func artifactConfigString(config *lifecycle.Config, key string) string {
	if config == nil || config.Extra == nil {
		return ""
	}
	value, ok := config.Extra[key].(string)
	if !ok {
		return ""
	}
	return value
}

func diagnosticsArtifactSourceCommit(contentDir string) string {
	if contentDir == "" {
		contentDir = "."
	}
	ctx, cancel := context.WithTimeout(context.Background(), diagnosticsArtifactSourceTimeout)
	defer cancel()
	commit, err := sourcegit.Head(ctx, contentDir)
	if err != nil {
		return ""
	}
	return commit
}

func writeDiagnosticsArtifact(destination string, data []byte) error {
	directory := filepath.Dir(destination)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create diagnostics artifact directory: %w", err)
	}

	temporary, err := os.CreateTemp(directory, ".diagnostics-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary diagnostics artifact: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set diagnostics artifact permissions: %w", err)
	}
	written, err := temporary.Write(data)
	if err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary diagnostics artifact: %w", err)
	}
	if written != len(data) {
		_ = temporary.Close()
		return fmt.Errorf("write temporary diagnostics artifact: %w", io.ErrShortWrite)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync temporary diagnostics artifact: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary diagnostics artifact: %w", err)
	}

	if err := replaceDiagnosticsArtifact(temporaryName, destination); err != nil {
		return fmt.Errorf("replace diagnostics artifact: %w", err)
	}
	return nil
}

func replaceDiagnosticsArtifact(source, destination string) error {
	if runtime.GOOS != tailwindOSWindows {
		return os.Rename(source, destination)
	}

	// Windows cannot rename over an existing file. Move the old artifact to a
	// temporary sibling first, then restore it if installing the new artifact
	// fails. The new file is complete before this replacement begins.
	directory := filepath.Dir(destination)
	if info, err := os.Lstat(destination); err == nil && info.IsDir() {
		return fmt.Errorf("diagnostics artifact destination is a directory")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	backup, err := os.CreateTemp(directory, ".diagnostics-backup-*")
	if err != nil {
		return fmt.Errorf("create diagnostics artifact backup: %w", err)
	}
	backupName := backup.Name()
	if err := backup.Close(); err != nil {
		_ = os.Remove(backupName)
		return fmt.Errorf("close diagnostics artifact backup: %w", err)
	}
	if err := os.Remove(backupName); err != nil {
		return fmt.Errorf("prepare diagnostics artifact backup: %w", err)
	}

	hadDestination := true
	if err := os.Rename(destination, backupName); err != nil {
		if os.IsNotExist(err) {
			hadDestination = false
		} else {
			return err
		}
	}
	if err := os.Rename(source, destination); err != nil {
		if hadDestination {
			if restoreErr := os.Rename(backupName, destination); restoreErr != nil {
				return fmt.Errorf("install diagnostics artifact: %w; restore previous artifact: %w", err, restoreErr)
			}
		}
		return err
	}
	if hadDestination {
		_ = os.Remove(backupName)
	}
	return nil
}

type diagnosticsArtifactError struct {
	err error
}

func (e *diagnosticsArtifactError) Error() string {
	return e.err.Error()
}

func (e *diagnosticsArtifactError) Unwrap() error {
	return e.err
}

// IsCritical makes artifact publication failures fail the full build. The
// cleanup stage otherwise treats ordinary plugin errors as warnings.
func (e *diagnosticsArtifactError) IsCritical() bool {
	return true
}

var (
	_ lifecycle.Plugin         = (*DiagnosticsArtifactPlugin)(nil)
	_ lifecycle.CleanupPlugin  = (*DiagnosticsArtifactPlugin)(nil)
	_ lifecycle.PriorityPlugin = (*DiagnosticsArtifactPlugin)(nil)
)
