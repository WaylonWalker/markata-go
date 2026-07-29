package cmd

import (
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/WaylonWalker/markata-go/pkg/logging"
	"github.com/spf13/cobra"
)

func TestCLISectionRule_UsesPreferredTerminalWidth(t *testing.T) {
	oldTerminalWidth := cliTerminalWidth
	defer func() { cliTerminalWidth = oldTerminalWidth }()
	cliTerminalWidth = func(io.Writer) int { return 120 }

	rule := cliSectionRule("Neovim")
	if got := utf8.RuneCountInString(rule); got != cliPreferredRuleWidth {
		t.Errorf("rule width = %d, want %d", got, cliPreferredRuleWidth)
	}
	if !strings.HasPrefix(rule, "───") {
		t.Errorf("rule = %q, want Unicode box-drawing separator", rule)
	}
}

func TestCLISectionRule_UsesCompactFormOnNarrowTerminal(t *testing.T) {
	oldTerminalWidth := cliTerminalWidth
	defer func() { cliTerminalWidth = oldTerminalWidth }()
	cliTerminalWidth = func(io.Writer) int { return 20 }

	if got, want := cliSectionRule("Helix"), "─ Helix ─"; got != want {
		t.Errorf("rule = %q, want %q", got, want)
	}
}

func TestCLISectionRule_TruncatesLongLabelsOnNarrowTerminal(t *testing.T) {
	oldTerminalWidth := cliTerminalWidth
	defer func() { cliTerminalWidth = oldTerminalWidth }()
	cliTerminalWidth = func(io.Writer) int { return 20 }

	rule := cliSectionRule("Theme/Palette Configuration")
	if got := utf8.RuneCountInString(rule); got > 20 {
		t.Errorf("rule width = %d, want at most 20", got)
	}
	if !strings.Contains(rule, "…") {
		t.Errorf("rule = %q, want truncated label", rule)
	}
}

func TestCLISectionRule_TruncatesLongLabelsAtTerminalBoundary(t *testing.T) {
	oldTerminalWidth := cliTerminalWidth
	defer func() { cliTerminalWidth = oldTerminalWidth }()
	cliTerminalWidth = func(io.Writer) int { return cliNarrowRuleWidth }

	rule := cliSectionRule("A separator label that is wider than the terminal")
	if got := utf8.RuneCountInString(rule); got > cliNarrowRuleWidth {
		t.Errorf("rule width = %d, want at most %d", got, cliNarrowRuleWidth)
	}
}

func TestRenderCommandHelp_IncludesAliases(t *testing.T) {
	oldCmd := currentCmd
	defer func() { currentCmd = oldCmd }()
	command := &cobra.Command{Use: "remove", Short: "Remove a skill", Aliases: []string{"uninstall"}}
	output := &strings.Builder{}
	command.SetOut(output)

	renderCommandHelp(command)
	if !strings.Contains(output.String(), "remove, uninstall") {
		t.Errorf("help output = %q, want aliases", output.String())
	}
}

func TestRenderHelpDescription_UsesThemedUnicodeSections(t *testing.T) {
	oldCmd := currentCmd
	defer func() { currentCmd = oldCmd }()
	command := &cobra.Command{Use: "help"}
	output := &strings.Builder{}
	command.SetOut(output)
	currentCmd = command

	renderHelpDescription(output, "Overview text\n\nExamples:\n  markata-go build", logging.DefaultTheme())
	if !strings.Contains(output.String(), "─") || !strings.Contains(output.String(), "Examples") {
		t.Errorf("help output = %q, want Unicode Examples section", output.String())
	}
}
