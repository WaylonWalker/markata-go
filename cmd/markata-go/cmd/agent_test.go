package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/agentskill"
	"github.com/spf13/cobra"
)

func TestNormalizeAgentInstallTarget(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "default empty", input: "", want: agentTargetAgents},
		{name: "agents", input: "agents", want: agentTargetAgents},
		{name: "claude", input: "claude", want: agentTargetClaude},
		{name: "unknown", input: "cursor", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeAgentInstallTarget(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeAgentInstallTarget() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("normalizeAgentInstallTarget() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInstallAgentSkill_DryRunDoesNotWriteFiles(t *testing.T) {
	root := t.TempDir()
	installedFiles, err := installAgentSkill(root, agentTargetAgents, agentskill.SiteSkillName, true, false, "test")
	if err != nil {
		t.Fatalf("installAgentSkill() error = %v", err)
	}
	if len(installedFiles) == 0 {
		t.Fatal("expected bundled skill files")
	}

	if _, err := os.Stat(filepath.Join(root, ".agents")); !os.IsNotExist(err) {
		t.Fatalf("expected dry run to avoid creating files, got err=%v", err)
	}
}

func TestInstallAgentSkill_WritesBundledFiles(t *testing.T) {
	root := t.TempDir()
	installedFiles, err := installAgentSkill(root, agentTargetClaude, agentskill.SiteSkillName, false, false, "test")
	if err != nil {
		t.Fatalf("installAgentSkill() error = %v", err)
	}
	if len(installedFiles) == 0 {
		t.Fatal("expected bundled skill files")
	}

	skillFile := filepath.Join(root, ".claude", "skills", agentskill.SiteSkillName, "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("ReadFile(SKILL.md) error = %v", err)
	}
	if !strings.Contains(string(content), "markata-go-site") {
		t.Fatalf("expected installed skill content, got %q", string(content))
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "skills", agentskill.SiteSkillName, "topics", "template-management.md")); err != nil {
		t.Fatalf("expected topic file to be installed: %v", err)
	}
}

func TestInstallAgentSkill_ExistingFileRequiresForce(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".agents", "skills", agentskill.SiteSkillName)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte("existing"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := installAgentSkill(root, agentTargetAgents, agentskill.SiteSkillName, false, false, "test")
	if err == nil {
		t.Fatal("expected conflict error when files already exist")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("expected --force guidance, got %v", err)
	}
}

// TestRunAgentInstallCommand_DryRunUsesCommandWriter verifies that command output
// goes through the cobra command writer, not directly to os.Stdout.
//
// NOTE: This test mutates package-level flag variables and restores them via
// defer. It must NOT use t.Parallel().
func TestRunAgentInstallCommand_DryRunUsesCommandWriter(t *testing.T) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	command := &cobra.Command{Use: "install"}
	command.SetOut(stdout)
	command.SetErr(stderr)

	originalTarget := agentInstallTarget
	originalName := agentInstallName
	originalDryRun := agentInstallDryRun
	originalForce := agentInstallForce
	defer func() {
		agentInstallTarget = originalTarget
		agentInstallName = originalName
		agentInstallDryRun = originalDryRun
		agentInstallForce = originalForce
		currentCmd = nil
	}()

	agentInstallTarget = agentTargetAgents
	agentInstallName = agentskill.SiteSkillName
	agentInstallDryRun = true
	agentInstallForce = false

	if err := runAgentInstallCommand(command, []string{t.TempDir()}); err != nil {
		t.Fatalf("runAgentInstallCommand() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "Dry run: would install") {
		t.Fatalf("expected output in command writer, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

// TestRunAgentUpdateCommand_DryRunUsesCommandWriter verifies update reports through the cobra writer.
//
// NOTE: This test mutates package-level flag variables and restores them via
// defer. It must NOT use t.Parallel().
func TestRunAgentUpdateCommand_DryRunUsesCommandWriter(t *testing.T) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	command := &cobra.Command{Use: "update"}
	command.SetOut(stdout)
	command.SetErr(stderr)

	originalTarget := agentUpdateTarget
	originalName := agentUpdateName
	originalDryRun := agentUpdateDryRun
	defer func() {
		agentUpdateTarget = originalTarget
		agentUpdateName = originalName
		agentUpdateDryRun = originalDryRun
		currentCmd = nil
	}()

	agentUpdateTarget = agentTargetAgents
	agentUpdateName = agentskill.SiteSkillName
	agentUpdateDryRun = true

	if err := runAgentUpdateCommand(command, []string{t.TempDir()}); err != nil {
		t.Fatalf("runAgentUpdateCommand() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "Dry run: would update") {
		t.Fatalf("expected update dry-run output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

// TestRunAgentUpdateCommand_OverwritesInstalledFiles verifies update rewrites an existing install.
//
// NOTE: This test mutates package-level flag variables and restores them via
// defer. It must NOT use t.Parallel().
func TestRunAgentUpdateCommand_OverwritesInstalledFiles(t *testing.T) {
	root := t.TempDir()
	_, err := installAgentSkill(root, agentTargetAgents, agentskill.SiteSkillName, false, false, Version)
	if err != nil {
		t.Fatalf("installAgentSkill() error = %v", err)
	}

	skillFile := filepath.Join(root, ".agents", "skills", agentskill.SiteSkillName, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("modified"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	command := &cobra.Command{Use: "update"}
	command.SetOut(stdout)
	command.SetErr(stderr)

	originalTarget := agentUpdateTarget
	originalName := agentUpdateName
	originalDryRun := agentUpdateDryRun
	defer func() {
		agentUpdateTarget = originalTarget
		agentUpdateName = originalName
		agentUpdateDryRun = originalDryRun
		currentCmd = nil
	}()

	agentUpdateTarget = agentTargetAgents
	agentUpdateName = agentskill.SiteSkillName
	agentUpdateDryRun = false

	if err := runAgentUpdateCommand(command, []string{root}); err != nil {
		t.Fatalf("runAgentUpdateCommand() error = %v", err)
	}

	content, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(content), "markata-go-site") {
		t.Fatalf("expected SKILL.md to be restored by update, got %q", string(content))
	}
	if !strings.Contains(stdout.String(), "Updated ") {
		t.Fatalf("expected update output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestInstallAgentSkill_WritesManifest(t *testing.T) {
	root := t.TempDir()
	_, err := installAgentSkill(root, agentTargetAgents, agentskill.SiteSkillName, false, false, "0.5.0-test")
	if err != nil {
		t.Fatalf("installAgentSkill() error = %v", err)
	}

	skillDir := filepath.Join(root, ".agents", "skills", agentskill.SiteSkillName)
	manifest, err := agentskill.ReadManifest(skillDir)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Version != "0.5.0-test" {
		t.Errorf("Version = %q, want %q", manifest.Version, "0.5.0-test")
	}
	if manifest.Target != agentTargetAgents {
		t.Errorf("Target = %q, want %q", manifest.Target, agentTargetAgents)
	}
	if len(manifest.Files) == 0 {
		t.Error("expected manifest to contain file hashes")
	}
}

func TestInstallAgentSkill_DryRunDoesNotWriteManifest(t *testing.T) {
	root := t.TempDir()
	_, err := installAgentSkill(root, agentTargetAgents, agentskill.SiteSkillName, true, false, "0.5.0-test")
	if err != nil {
		t.Fatalf("installAgentSkill() error = %v", err)
	}

	skillDir := filepath.Join(root, ".agents", "skills", agentskill.SiteSkillName)
	_, err = agentskill.ReadManifest(skillDir)
	if err == nil {
		t.Fatal("expected manifest to not exist after dry run")
	}
}

// TestRunAgentDoctorCommand_DriftReturnsExitCode verifies doctor reports drift via ExitCodeError.
//
// NOTE: This test mutates package-level flag variables and restores them via
// defer. It must NOT use t.Parallel().
func TestRunAgentDoctorCommand_DriftReturnsExitCode(t *testing.T) {
	root := t.TempDir()
	_, err := installAgentSkill(root, agentTargetAgents, agentskill.SiteSkillName, false, false, "older-version")
	if err != nil {
		t.Fatalf("installAgentSkill() error = %v", err)
	}

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	command := &cobra.Command{Use: "doctor"}
	command.SetOut(stdout)
	command.SetErr(stderr)

	originalTarget := agentDoctorTarget
	originalName := agentDoctorName
	defer func() {
		agentDoctorTarget = originalTarget
		agentDoctorName = originalName
		currentCmd = nil
	}()

	agentDoctorTarget = agentTargetAgents
	agentDoctorName = agentskill.SiteSkillName

	err = runAgentDoctorCommand(command, []string{root})
	if err == nil {
		t.Fatal("expected drift exit code error")
	}

	var exitErr *ExitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitCodeError, got %T", err)
	}
	if exitErr.ExitCode() != doctorExitDrift {
		t.Fatalf("ExitCode() = %d, want %d", exitErr.ExitCode(), doctorExitDrift)
	}
	if !strings.Contains(stdout.String(), "Skill has") {
		t.Fatalf("expected drift output, got %q", stdout.String())
	}
}

// TestRunAgentDoctorCommand_MissingManifestReturnsExitCode verifies doctor handles pre-manifest installs.
//
// NOTE: This test mutates package-level flag variables and restores them via
// defer. It must NOT use t.Parallel().
func TestRunAgentDoctorCommand_MissingManifestReturnsExitCode(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".agents", "skills", agentskill.SiteSkillName)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("test"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	command := &cobra.Command{Use: "doctor"}
	command.SetOut(stdout)
	command.SetErr(stderr)

	originalTarget := agentDoctorTarget
	originalName := agentDoctorName
	defer func() {
		agentDoctorTarget = originalTarget
		agentDoctorName = originalName
		currentCmd = nil
	}()

	agentDoctorTarget = agentTargetAgents
	agentDoctorName = agentskill.SiteSkillName

	err := runAgentDoctorCommand(command, []string{root})
	if err == nil {
		t.Fatal("expected exit code error for missing manifest")
	}

	var exitErr *ExitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitCodeError, got %T", err)
	}
	if exitErr.ExitCode() != doctorExitDrift {
		t.Fatalf("ExitCode() = %d, want %d", exitErr.ExitCode(), doctorExitDrift)
	}
	if !strings.Contains(stdout.String(), "has no manifest") {
		t.Fatalf("expected missing manifest output, got %q", stdout.String())
	}
}

// TestRunAgentDoctorCommand_UpToDate verifies doctor reports no drift immediately after install.
//
// NOTE: This test mutates package-level flag variables and restores them via
// defer. It must NOT use t.Parallel().
func TestRunAgentDoctorCommand_UpToDate(t *testing.T) {
	root := t.TempDir()

	// Install first.
	_, err := installAgentSkill(root, agentTargetAgents, agentskill.SiteSkillName, false, false, Version)
	if err != nil {
		t.Fatalf("installAgentSkill() error = %v", err)
	}

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	command := &cobra.Command{Use: "doctor"}
	command.SetOut(stdout)
	command.SetErr(stderr)

	originalTarget := agentDoctorTarget
	originalName := agentDoctorName
	defer func() {
		agentDoctorTarget = originalTarget
		agentDoctorName = originalName
		currentCmd = nil
	}()

	agentDoctorTarget = agentTargetAgents
	agentDoctorName = agentskill.SiteSkillName

	if err := runAgentDoctorCommand(command, []string{root}); err != nil {
		t.Fatalf("runAgentDoctorCommand() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "up to date") {
		t.Errorf("expected 'up to date' in output, got %q", stdout.String())
	}
}

// TestRunAgentRemoveCommand_RemovesInstalledSkill verifies remove deletes the skill directory.
//
// NOTE: This test mutates package-level flag variables and restores them via
// defer. It must NOT use t.Parallel().
func TestRunAgentRemoveCommand_RemovesInstalledSkill(t *testing.T) {
	root := t.TempDir()
	_, err := installAgentSkill(root, agentTargetClaude, agentskill.SiteSkillName, false, false, Version)
	if err != nil {
		t.Fatalf("installAgentSkill() error = %v", err)
	}

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	command := &cobra.Command{Use: "remove"}
	command.SetOut(stdout)
	command.SetErr(stderr)

	originalTarget := agentRemoveTarget
	originalName := agentRemoveName
	defer func() {
		agentRemoveTarget = originalTarget
		agentRemoveName = originalName
		currentCmd = nil
	}()

	agentRemoveTarget = agentTargetClaude
	agentRemoveName = agentskill.SiteSkillName

	if err := runAgentRemoveCommand(command, []string{root}); err != nil {
		t.Fatalf("runAgentRemoveCommand() error = %v", err)
	}

	skillDir := filepath.Join(root, ".claude", "skills", agentskill.SiteSkillName)
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatalf("expected skill dir to be removed, got err=%v", err)
	}
	if !strings.Contains(stdout.String(), "Removed .claude/skills/") {
		t.Fatalf("expected remove output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

// TestAgentRemoveCmd_HasUninstallAlias verifies uninstall is exposed as an alias.
func TestAgentRemoveCmd_HasUninstallAlias(t *testing.T) {
	found := false
	for _, alias := range agentRemoveCmd.Aliases {
		if alias == "uninstall" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected uninstall alias on agent remove command")
	}
}

func TestValidateSkillName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid name", input: "markata-go-site", wantErr: false},
		{name: "valid simple", input: "my-skill", wantErr: false},
		{name: "empty", input: "", wantErr: true},
		{name: "dot dot", input: "..", wantErr: true},
		{name: "dot dot prefix", input: "..foo", wantErr: true},
		{name: "single dot", input: ".", wantErr: true},
		{name: "forward slash", input: "foo/bar", wantErr: true},
		{name: "backslash", input: "foo\\bar", wantErr: true},
		{name: "traversal path", input: "../../etc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSkillName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSkillName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestRunAgentRemoveCommand_RejectsTraversalName(t *testing.T) {
	root := t.TempDir()

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	command := &cobra.Command{Use: "remove"}
	command.SetOut(stdout)
	command.SetErr(stderr)

	originalTarget := agentRemoveTarget
	originalName := agentRemoveName
	defer func() {
		agentRemoveTarget = originalTarget
		agentRemoveName = originalName
		currentCmd = nil
	}()

	agentRemoveTarget = agentTargetAgents
	agentRemoveName = "../.."

	err := runAgentRemoveCommand(command, []string{root})
	if err == nil {
		t.Fatal("expected error for path traversal name")
	}
	if !strings.Contains(err.Error(), "path separator") && !strings.Contains(err.Error(), "path traversal") {
		t.Fatalf("expected path validation error, got %v", err)
	}
}

func TestRunAgentRemoveCommand_RejectsNonSkillDirectory(t *testing.T) {
	root := t.TempDir()
	// Create a directory that is NOT a skill (no SKILL.md, no .manifest.json).
	notASkill := filepath.Join(root, ".agents", "skills", "not-a-skill")
	if err := os.MkdirAll(notASkill, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(notASkill, "random.txt"), []byte("hi"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	command := &cobra.Command{Use: "remove"}
	command.SetOut(stdout)
	command.SetErr(stderr)

	originalTarget := agentRemoveTarget
	originalName := agentRemoveName
	defer func() {
		agentRemoveTarget = originalTarget
		agentRemoveName = originalName
		currentCmd = nil
	}()

	agentRemoveTarget = agentTargetAgents
	agentRemoveName = "not-a-skill"

	err := runAgentRemoveCommand(command, []string{root})
	if err == nil {
		t.Fatal("expected error for non-skill directory")
	}
	if !strings.Contains(err.Error(), "does not appear to be an installed skill") {
		t.Fatalf("expected skill marker error, got %v", err)
	}

	// Verify the directory was NOT deleted.
	if _, statErr := os.Stat(notASkill); os.IsNotExist(statErr) {
		t.Fatal("expected non-skill directory to be preserved")
	}
}
