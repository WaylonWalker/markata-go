package cmd

import (
	"bytes"
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
	installedFiles, err := installAgentSkill(root, agentTargetAgents, agentskill.SiteSkillName, true, false)
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
	installedFiles, err := installAgentSkill(root, agentTargetClaude, agentskill.SiteSkillName, false, false)
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

	_, err := installAgentSkill(root, agentTargetAgents, agentskill.SiteSkillName, false, false)
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
