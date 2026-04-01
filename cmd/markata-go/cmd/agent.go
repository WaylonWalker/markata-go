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

var (
	agentInstallTarget string
	agentInstallName   string
	agentInstallForce  bool
	agentInstallDryRun bool
)

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.AddCommand(agentInstallCmd)

	agentInstallCmd.Flags().StringVar(&agentInstallTarget, "target", agentTargetAgents, "install target: agents or claude")
	agentInstallCmd.Flags().StringVar(&agentInstallName, "name", agentskill.SiteSkillName, "installed skill directory name")
	agentInstallCmd.Flags().BoolVar(&agentInstallForce, "force", false, "overwrite bundled skill files if they already exist")
	agentInstallCmd.Flags().BoolVar(&agentInstallDryRun, "dry-run", false, "show what would be installed without writing files")
}

func runAgentInstallCommand(cmd *cobra.Command, args []string) error {
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

	target, err := normalizeAgentInstallTarget(agentInstallTarget)
	if err != nil {
		return err
	}

	installedFiles, err := installAgentSkill(root, target, agentInstallName, agentInstallDryRun, agentInstallForce)
	if err != nil {
		return err
	}

	destination := agentSkillInstallDir(root, target, agentInstallName)
	if agentInstallDryRun {
		outlnf("Dry run: would install %d file(s) to %s", len(installedFiles), destination)
	} else {
		outlnf("Installed %d file(s) to %s", len(installedFiles), destination)
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

func installAgentSkill(root, target, skillName string, dryRun, force bool) ([]string, error) {
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

	return installedFiles, nil
}
