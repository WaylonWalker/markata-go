package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/WaylonWalker/markata-go/pkg/services"
	"github.com/WaylonWalker/markata-go/pkg/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive terminal UI",
	Long: `Launch an interactive terminal user interface for browsing and managing
your markata-go site content.

The TUI provides:
  - Post list with filtering and sorting
  - Tag and feed browsing
  - Quick editing via $EDITOR
  - Build triggering

Navigation:
  j/k or ↑/↓  Move selection
  Enter       View post details
  /           Filter posts
  :           Command mode
  q           Quit`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, _ []string) error {
	// Create the manager using the existing helper
	manager, err := createManager(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create services app
	app := services.NewApp(manager)

	// Load posts through Collect stage for full TUI functionality
	// This runs Transform (for stats, auto-titles) and Collect (for feeds)
	if err := app.Build.LoadForTUI(cmd.Context()); err != nil {
		return fmt.Errorf("failed to load posts: %w", err)
	}

	// Load theme from configuration
	paletteName := tui.GetPaletteNameFromConfig(manager.Config().Extra)
	colors := tui.LoadColors(paletteName)
	theme := tui.NewTheme(colors)

	// Create and run TUI with theme
	model := tui.NewModelWithTheme(app, theme)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
