package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCLICommandAliases(t *testing.T) {
	for alias, want := range map[string]string{"s": "serve", "b": "build", "ls": "list"} {
		command, _, err := rootCmd.Find([]string{alias})
		if err != nil || command.Name() != want {
			t.Errorf("%s resolves to %v, %v; want %s", alias, command, err, want)
		}
	}
	if serveCmd.Flags().Lookup("bind") == nil {
		t.Fatal("serve --bind is missing")
	}
}

func TestServeBindAliasAndConflict(t *testing.T) {
	oldHost, oldBind := serveHost, serveBind
	t.Cleanup(func() { serveHost, serveBind = oldHost, oldBind })
	newCommand := func() *cobra.Command {
		command := &cobra.Command{Use: "serve"}
		command.Flags().StringVar(&serveHost, "host", "localhost", "")
		command.Flags().StringVar(&serveBind, "bind", "localhost", "")
		return command
	}
	command := newCommand()
	if err := command.Flags().Set("bind", "0.0.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := resolveServeHost(command); err != nil || serveHost != "0.0.0.0" {
		t.Fatalf("host = %q, error = %v; want bind address", serveHost, err)
	}
	command = newCommand()
	if err := command.Flags().Set("bind", "0.0.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := command.Flags().Set("host", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if err := resolveServeHost(command); ExitCodeForError(err) != exitCodeUsage {
		t.Fatalf("error = %v; want usage error for conflicting addresses", err)
	}
}

func TestUnknownRootCommandSuggestsAndDoesNotBuild(t *testing.T) {
	err := runRootCommand(rootCmd, []string{"sevre"})
	if ExitCodeForError(err) != exitCodeUsage || !strings.Contains(err.Error(), "markata-go serve") {
		t.Fatalf("error = %v; want usage error suggesting serve", err)
	}
	err = runRootCommand(rootCmd, []string{"xyzzy"})
	if !strings.Contains(err.Error(), "cannot infer") || !strings.Contains(err.Error(), "markata-go --help") {
		t.Fatalf("error = %v; want honest fallback", err)
	}
}

func TestUnknownConfigSubcommandDoesNotShowConfiguration(t *testing.T) {
	err := configCmd.Args(configCmd, []string{"sho"})
	if ExitCodeForError(err) != exitCodeUsage || !strings.Contains(err.Error(), "markata-go config show") {
		t.Fatalf("error = %v; want config show suggestion", err)
	}
}

func TestUnknownCommandGroupChildIsUsageError(t *testing.T) {
	prepareCommandGroups(rootCmd)
	for _, group := range []*cobra.Command{paletteCmd, agentCmd, readerCmd} {
		err := group.Args(group, []string{"lst"})
		if ExitCodeForError(err) != exitCodeUsage || !strings.Contains(err.Error(), "unknown command") {
			t.Errorf("%s error = %v; want usage error", group.Name(), err)
		}
	}
}

func TestUnknownFlagSuggestsClosestFlag(t *testing.T) {
	err := clarifyFlagError(serveCmd, errors.New("unknown flag: --hst"))
	if !strings.Contains(err.Error(), "--host") || !strings.Contains(err.Error(), "markata-go serve --help") {
		t.Fatalf("error = %v; want host suggestion and exact help path", err)
	}
}

func TestConfigGetSuggestsExistingKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "markata-go.toml")
	if err := os.WriteFile(path, []byte("[markata-go]\noutput_dir = 'public'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	previous := cfgFile
	cfgFile = path
	t.Cleanup(func() { cfgFile = previous })
	err := runConfigGetCommand(configGetCmd, []string{"output_dr"})
	if ExitCodeForError(err) != exitCodeUsage || !strings.Contains(err.Error(), "output_dir") {
		t.Fatalf("error = %v; want output_dir suggestion", err)
	}
}

func TestConfigSetSuggestionPreservesExtensionKeys(t *testing.T) {
	if safeConfigSetSuggestion("webmentions.enabled", "webmention.enabled") {
		t.Fatal("a different plugin section must remain writable")
	}
	if !safeConfigSetSuggestion("theme.palete", "theme.palette") {
		t.Fatal("a typo within a known section should be diagnosed")
	}
	if !safeConfigSetSuggestion("output_dr", "output_dir") {
		t.Fatal("a close multiword top-level key should be diagnosed")
	}
}

func TestFormatErrorPlain(t *testing.T) {
	previous := noColor
	noColor = true
	t.Cleanup(func() { noColor = previous })
	message := FormatError(unknownCommandError(rootCmd, "sevre"))
	if strings.Contains(message, "\x1b[") || !strings.Contains(message, "Error: unknown command") || !strings.Contains(message, "Next:") {
		t.Fatalf("formatted error = %q", message)
	}
}

func TestFormatErrorForcedColor(t *testing.T) {
	previousNoColor, previousForceColor, previousLogFormat := noColor, forceColor, logFormat
	noColor, forceColor, logFormat = false, true, "auto"
	t.Cleanup(func() {
		noColor, forceColor, logFormat = previousNoColor, previousForceColor, previousLogFormat
	})
	message := FormatError(unknownCommandError(rootCmd, "sevre"))
	if !strings.Contains(message, "\x1b[") {
		t.Fatalf("formatted error = %q; want color", message)
	}
}
