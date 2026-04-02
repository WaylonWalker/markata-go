package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/agentskill"
	"github.com/spf13/cobra"
)

const (
	agentTargetAgents = "agents"
	agentTargetClaude = "claude"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Install agent integrations for markata-go sites",
	Long: `Manage installable agent integrations for markata-go sites.

The initial workflow installs a bundled, topic-based skill into a site repository.
The command group is intentionally generic so future subcommands can add export
and MCP-oriented integrations without changing the bundled skill format.

Supported install targets:
  agents  - installs to .agents/skills/<name>/
  claude  - installs to .claude/skills/<name>/`,
}

var agentInstallCmd = &cobra.Command{
	Use:   "install [site-path]",
	Short: "Install the bundled markata-go site skill",
	Long: `Install the bundled markata-go site skill into a repository.

By default the skill is installed into the current directory using the portable
.agents/skills layout. Use --target claude to install into Claude Code's
.claude/skills layout instead.

Examples:
  markata-go agent install
  markata-go agent install --target claude
  markata-go agent install ../my-site --dry-run
  markata-go agent install . --force`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAgentInstallCommand,
}

var agentDoctorCmd = &cobra.Command{
	Use:   "doctor [site-path]",
	Short: "Check installed skill for drift against the current binary",
	Long: `Compare installed skill files against the versions bundled in the current
markata-go binary. Reports files that are modified, missing, or newly added
since the skill was installed.

Examples:
  markata-go agent doctor
  markata-go agent doctor --target claude
  markata-go agent doctor ../my-site`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAgentDoctorCommand,
}

var agentUpdateCmd = &cobra.Command{
	Use:   "update [site-path]",
	Short: "Update an installed markata-go site skill",
	Long: `Update the installed markata-go site skill in place.

This is the user-friendly equivalent of running 'markata-go agent install --force'.

Examples:
  markata-go agent update
  markata-go agent update --target claude
  markata-go agent update --dry-run
  markata-go agent update ../my-site`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAgentUpdateCommand,
}

var agentRemoveCmd = &cobra.Command{
	Use:     "remove [site-path]",
	Aliases: []string{"uninstall"},
	Short:   "Remove an installed markata-go site skill",
	Long: `Remove the installed markata-go site skill from a repository.

Examples:
  markata-go agent remove
  markata-go agent uninstall
  markata-go agent remove --target claude
  markata-go agent remove ../my-site`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAgentRemoveCommand,
}

var (
	agentInstallTarget string
	agentInstallName   string
	agentInstallForce  bool
	agentInstallDryRun bool

	agentDoctorTarget string
	agentDoctorName   string

	agentUpdateTarget string
	agentUpdateName   string
	agentUpdateDryRun bool

	agentRemoveTarget string
	agentRemoveName   string
)

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.AddCommand(agentInstallCmd)
	agentCmd.AddCommand(agentUpdateCmd)
	agentCmd.AddCommand(agentDoctorCmd)
	agentCmd.AddCommand(agentRemoveCmd)

	agentInstallCmd.Flags().StringVar(&agentInstallTarget, "target", agentTargetAgents, "install target: agents or claude")
	agentInstallCmd.Flags().StringVar(&agentInstallName, "name", agentskill.SiteSkillName, "installed skill directory name")
	agentInstallCmd.Flags().BoolVar(&agentInstallForce, "force", false, "overwrite bundled skill files if they already exist")
	agentInstallCmd.Flags().BoolVar(&agentInstallDryRun, "dry-run", false, "show what would be installed without writing files")

	agentUpdateCmd.Flags().StringVar(&agentUpdateTarget, "target", agentTargetAgents, "install target: agents or claude")
	agentUpdateCmd.Flags().StringVar(&agentUpdateName, "name", agentskill.SiteSkillName, "installed skill directory name")
	agentUpdateCmd.Flags().BoolVar(&agentUpdateDryRun, "dry-run", false, "show what would be updated without writing files")

	agentDoctorCmd.Flags().StringVar(&agentDoctorTarget, "target", agentTargetAgents, "install target: agents or claude")
	agentDoctorCmd.Flags().StringVar(&agentDoctorName, "name", agentskill.SiteSkillName, "installed skill directory name")

	agentRemoveCmd.Flags().StringVar(&agentRemoveTarget, "target", agentTargetAgents, "install target: agents or claude")
	agentRemoveCmd.Flags().StringVar(&agentRemoveName, "name", agentskill.SiteSkillName, "installed skill directory name")
}

func runAgentInstallCommand(cmd *cobra.Command, args []string) error {
	currentCmd = cmd
	return runAgentWriteCommand(args, agentInstallTarget, agentInstallName, agentInstallDryRun, agentInstallForce, "install")
}

func runAgentUpdateCommand(cmd *cobra.Command, args []string) error {
	currentCmd = cmd
	return runAgentWriteCommand(args, agentUpdateTarget, agentUpdateName, agentUpdateDryRun, true, "update")
}

func runAgentWriteCommand(args []string, targetFlag, name string, dryRun, force bool, operation string) error {
	sitePath := "."
	if len(args) > 0 {
		sitePath = args[0]
	}

	root, err := filepath.Abs(sitePath)
	if err != nil {
		return fmt.Errorf("resolve site path %q: %w", sitePath, err)
	}

	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("stat site path %q: %w", root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("site path %q is not a directory", root)
	}

	target, err := normalizeAgentInstallTarget(targetFlag)
	if err != nil {
		return err
	}

	installedFiles, err := installAgentSkill(root, target, name, dryRun, force, Version)
	if err != nil {
		return err
	}

	destination := agentSkillInstallDir(root, target, name)
	if dryRun {
		outlnf("Dry run: would %s %d file(s) %s %s", operation, len(installedFiles), prepositionForAgentOperation(operation), destination)
	} else {
		outlnf("%s %d file(s) %s %s", pastTenseForAgentOperation(operation), len(installedFiles), prepositionForAgentOperation(operation), destination)
	}

	for _, filePath := range installedFiles {
		relPath, relErr := filepath.Rel(root, filePath)
		if relErr != nil {
			relPath = filePath
		}
		outlnf("- %s", filepath.ToSlash(relPath))
	}

	return nil
}

func prepositionForAgentOperation(operation string) string {
	switch operation {
	case "update":
		return "in"
	default:
		return "to"
	}
}

func pastTenseForAgentOperation(operation string) string {
	switch operation {
	case "update":
		return "Updated"
	default:
		return "Installed"
	}
}

func normalizeAgentInstallTarget(target string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "", agentTargetAgents:
		return agentTargetAgents, nil
	case agentTargetClaude:
		return agentTargetClaude, nil
	default:
		return "", fmt.Errorf("unknown agent install target %q; use 'agents' or 'claude'", target)
	}
}

func agentSkillInstallDir(root, target, skillName string) string {
	switch target {
	case agentTargetClaude:
		return filepath.Join(root, ".claude", "skills", skillName)
	default:
		return filepath.Join(root, ".agents", "skills", skillName)
	}
}

func installAgentSkill(root, target, skillName string, dryRun, force bool, version string) ([]string, error) {
	bundledFS, err := agentskill.SiteSkill()
	if err != nil {
		return nil, fmt.Errorf("load bundled skill: %w", err)
	}

	relativeFiles, err := agentskill.ListFiles()
	if err != nil {
		return nil, fmt.Errorf("list bundled skill files: %w", err)
	}

	destinationRoot := agentSkillInstallDir(root, target, skillName)
	installedFiles := make([]string, 0, len(relativeFiles))
	for _, relativePath := range relativeFiles {
		installedFiles = append(installedFiles, filepath.Join(destinationRoot, filepath.FromSlash(relativePath)))
	}

	if !force {
		for _, filePath := range installedFiles {
			_, statErr := os.Stat(filePath)
			if statErr == nil {
				return nil, fmt.Errorf("refusing to overwrite existing file %q; re-run with --force", filePath)
			}
			if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
				return nil, fmt.Errorf("stat destination %q: %w", filePath, statErr)
			}
		}
	}

	if dryRun {
		return installedFiles, nil
	}

	for i, relativePath := range relativeFiles {
		data, readErr := fs.ReadFile(bundledFS, relativePath)
		if readErr != nil {
			return nil, fmt.Errorf("read bundled skill file %q: %w", relativePath, readErr)
		}

		filePath := installedFiles[i]
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			return nil, fmt.Errorf("create destination directory for %q: %w", filePath, err)
		}
		if err := os.WriteFile(filePath, data, 0o644); err != nil { //nolint:gosec // installed skill files are non-sensitive markdown
			return nil, fmt.Errorf("write bundled skill file %q: %w", filePath, err)
		}
	}

	manifest, err := agentskill.ComputeBundledManifest(version, target)
	if err != nil {
		return nil, fmt.Errorf("compute manifest: %w", err)
	}
	if err := agentskill.WriteManifest(manifest, destinationRoot); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	return installedFiles, nil
}

// doctorExitDrift is the exit code when drift is detected.
const doctorExitDrift = 1

// doctorExitError is the exit code when doctor cannot complete.
const doctorExitError = 2

func runAgentDoctorCommand(cmd *cobra.Command, args []string) error {
	currentCmd = cmd

	sitePath := "."
	if len(args) > 0 {
		sitePath = args[0]
	}

	root, err := filepath.Abs(sitePath)
	if err != nil {
		return fmt.Errorf("resolve site path %q: %w", sitePath, err)
	}

	info, err := os.Stat(root)
	if err != nil {
		return newExitCodeError(doctorExitError, fmt.Errorf("stat site path %q: %w", root, err))
	}
	if !info.IsDir() {
		return newExitCodeError(doctorExitError, fmt.Errorf("site path %q is not a directory", root))
	}

	target, err := normalizeAgentInstallTarget(agentDoctorTarget)
	if err != nil {
		return newExitCodeError(doctorExitError, err)
	}

	skillDir := agentSkillInstallDir(root, target, agentDoctorName)
	info, err = os.Stat(skillDir)
	if err != nil || !info.IsDir() {
		return newExitCodeError(doctorExitError, fmt.Errorf("skill not installed at %s; run 'markata-go agent install' first", skillDir))
	}

	manifest, err := agentskill.ReadManifest(skillDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			outlnf("Skill at %s has no manifest.", skillDir)
			outlnf("Re-install with 'markata-go agent install --force' to enable drift detection.")
			return newExitCodeError(doctorExitDrift, nil)
		}
		return newExitCodeError(doctorExitError, fmt.Errorf("read manifest: %w", err))
	}

	report, err := agentskill.ComputeDrift(manifest, skillDir, Version)
	if err != nil {
		return newExitCodeError(doctorExitError, fmt.Errorf("compute drift: %w", err))
	}

	outlnf("Skill:     %s", agentDoctorName)
	outlnf("Location:  %s", skillDir)
	outlnf("Installed: %s", manifest.Version)
	outlnf("Current:   %s", Version)
	if report.VersionMismatch {
		outlnf("Version:   mismatch")
	}
	outlnf("")

	for _, entry := range report.Files {
		switch entry.Status {
		case agentskill.FileOK:
			outlnf("  ok        %s", entry.Path)
		case agentskill.FileModified:
			outlnf("  modified  %s", entry.Path)
		case agentskill.FileNew:
			outlnf("  new       %s", entry.Path)
		case agentskill.FileMissing:
			outlnf("  missing   %s", entry.Path)
		}
	}

	outlnf("")
	if report.HasDrift() {
		outlnf("Skill has %d issue(s). Run 'markata-go agent install --force' to update.", report.IssueCount())
		return newExitCodeError(doctorExitDrift, nil)
	}

	outlnf("Skill is up to date.")
	return nil
}

func runAgentRemoveCommand(cmd *cobra.Command, args []string) error {
	currentCmd = cmd

	sitePath := "."
	if len(args) > 0 {
		sitePath = args[0]
	}

	root, err := filepath.Abs(sitePath)
	if err != nil {
		return fmt.Errorf("resolve site path %q: %w", sitePath, err)
	}

	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("stat site path %q: %w", root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("site path %q is not a directory", root)
	}

	target, err := normalizeAgentInstallTarget(agentRemoveTarget)
	if err != nil {
		return err
	}

	skillDir := agentSkillInstallDir(root, target, agentRemoveName)
	info, err = os.Stat(skillDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("skill not installed at %s", skillDir)
		}
		return fmt.Errorf("stat skill directory %q: %w", skillDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("skill path %q is not a directory", skillDir)
	}

	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("remove skill directory %q: %w", skillDir, err)
	}

	relPath, relErr := filepath.Rel(root, skillDir)
	if relErr != nil {
		relPath = skillDir
	}
	outlnf("Removed %s", filepath.ToSlash(relPath))
	return nil
}
