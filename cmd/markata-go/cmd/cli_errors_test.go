package cmd

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCLICommandAliases(t *testing.T) {
	for alias, want := range map[string]string{"s": "serve", "serv": "serve", "b": "build", "ls": "list"} {
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
	for _, name := range []string{"ser", "sevre", "dev", "preview"} {
		err := unknownCommandError(rootCmd, name)
		if ExitCodeForError(err) != exitCodeUsage || !strings.Contains(err.Error(), "markata-go serve") {
			t.Errorf("%q error = %v; want usage error suggesting serve", name, err)
		}
	}
	familyMessage := unknownCommandError(rootCmd, "ser").Error()
	if strings.Count(familyMessage, "serve") != 1 || !strings.Contains(familyMessage, "aliases: s, serv") {
		t.Fatalf("command family suggestion = %q; want one serve result with aliases", familyMessage)
	}
	if command, _, err := rootCmd.Find([]string{"dev"}); err == nil && command.Name() == "serve" {
		t.Fatal("semantic suggestion dev must not execute serve")
	}
	err := runRootCommand(rootCmd, []string{"sevre"})
	if ExitCodeForError(err) != exitCodeUsage || !strings.Contains(err.Error(), "markata-go serve") {
		t.Fatalf("error = %v; want suggestion", err)
	}
	err = runRootCommand(rootCmd, []string{"xyzzy"})
	if strings.Contains(err.Error(), "cannot infer") || !strings.Contains(err.Error(), "markata-go --help") {
		t.Fatalf("error = %v; want contextual recovery without intent claim", err)
	}
}

func TestServePortAndListenDiagnostics(t *testing.T) {
	if err := validateServePort(65536); ExitCodeForError(err) != exitCodeUsage || !strings.Contains(FormatError(err), "1 to 65535") {
		t.Fatalf("out-of-range port diagnostic = %v", err)
	}
	if got := FormatError(clarifyFlagError(serveCmd, errors.New(`invalid argument "nonsense" for "--port" flag: strconv.ParseInt: parsing "nonsense": invalid syntax`))); !strings.Contains(got, "whole number from 1 to 65535") || !strings.Contains(got, "--port 8001") {
		t.Fatalf("port parse diagnostic = %q", got)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	addr := listener.Addr().String()
	_, err = net.Listen("tcp", addr)
	if err == nil {
		t.Fatal("second listener unexpectedly bound")
	}
	message := FormatError(diagnoseServeListenError(err, addr))
	if !strings.Contains(message, "already in use") || !strings.Contains(message, "Another Markata server may already be serving") || strings.Contains(message, "Serving at") {
		t.Fatalf("occupied-port diagnostic = %q", message)
	}

	windowsCollision := errors.New("listen tcp 127.0.0.1:8000: bind: only one usage of each socket address")
	if message := FormatError(diagnoseServeListenError(windowsCollision, "127.0.0.1:8000")); !strings.Contains(message, "port 8000 is already in use") {
		t.Fatalf("Windows occupied-port diagnostic = %q", message)
	}
}

func TestInvalidListChoicesAreActionable(t *testing.T) {
	_, err := parseListFormat("jsoon")
	message := FormatError(err)
	if ExitCodeForError(err) != exitCodeUsage || !strings.Contains(message, "invalid value \"jsoon\" for --format") || !strings.Contains(message, "Valid values: table, json, csv, path") || !strings.Contains(message, "json") {
		t.Fatalf("format diagnostic = %q", message)
	}
	_, err = parseSortOrder("dec")
	if !strings.Contains(FormatError(err), "desc") {
		t.Fatalf("order diagnostic = %q", FormatError(err))
	}
}

func TestUnknownConfigSubcommandDoesNotShowConfiguration(t *testing.T) {
	err := configCmd.Args(configCmd, []string{"sho"})
	if ExitCodeForError(err) != exitCodeUsage || !strings.Contains(err.Error(), "markata-go config show") {
		t.Fatalf("error = %v; want config show suggestion", err)
	}
}

func TestHelpTypoReturnsSuggestion(t *testing.T) {
	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"sevre"}, "markata-go serve"},
		{[]string{"config", "sho"}, "markata-go config show"},
	} {
		err := runHelpCommand(nil, test.args)
		if ExitCodeForError(err) != exitCodeUsage || !strings.Contains(err.Error(), test.want) {
			t.Errorf("help %v error = %v; want %s", test.args, err, test.want)
		}
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

func TestKnownConfigKeyErrorDoesNotClaimAmbiguity(t *testing.T) {
	message := configKeyError("markata-go.output_dir").Error()
	if !strings.Contains(message, "not set") || strings.Contains(message, "cannot infer") {
		t.Fatalf("message = %q", message)
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

func TestFormatErrorNoColorEnvironment(t *testing.T) {
	previousNoColor, previousForceColor, previousLogFormat := noColor, forceColor, logFormat
	noColor, forceColor, logFormat = false, false, "auto"
	t.Setenv("NO_COLOR", "1")
	t.Cleanup(func() { noColor, forceColor, logFormat = previousNoColor, previousForceColor, previousLogFormat })
	message := FormatError(unknownCommandError(rootCmd, "xyzzy"))
	if strings.Contains(message, "\x1b[") {
		t.Fatalf("NO_COLOR output contains ANSI escape: %q", message)
	}
}
