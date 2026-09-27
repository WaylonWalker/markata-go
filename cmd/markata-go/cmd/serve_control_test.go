package cmd

import (
	"bytes"
	stdlog "log"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/servecontrol"
	"github.com/spf13/cobra"
)

func TestShouldRunServeTUI_RequiresRealTerminal(t *testing.T) {
	input, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	if shouldRunServeTUI(false, false, input, os.Stdout) {
		t.Fatal("/dev/null must not count as an interactive input")
	}
	if shouldRunServeTUI(true, false, os.Stdin, os.Stdout) {
		t.Fatal("--no-tui must disable the terminal UI")
	}
	if shouldRunServeTUI(false, true, os.Stdin, os.Stdout) {
		t.Fatal("--no-input must disable the terminal UI")
	}
	if shouldRunServeTUI(false, false, strings.NewReader(""), os.Stdout) {
		t.Fatal("redirected input must disable the terminal UI")
	}
}

func TestServeLoggerManagerSetupKeepsTUITerminalOwned(t *testing.T) {
	previousWriter, previousFlags, previousPrefix := stdlog.Writer(), stdlog.Flags(), stdlog.Prefix()
	previousCommand := currentCmd
	previousTheme := currentLogTheme
	t.Cleanup(func() {
		stdlog.SetOutput(previousWriter)
		stdlog.SetFlags(previousFlags)
		stdlog.SetPrefix(previousPrefix)
		currentCmd = previousCommand
		currentLogTheme = previousTheme
		clearServeControl()
	})

	var terminal bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetErr(&terminal)
	currentCmd = cmd
	runtime := servecontrol.NewRuntime()
	setServeControl(runtime, true, nil)

	// Manager setup reconfigures the global logger. The Serve wrapper must
	// immediately restore the observer/discard policy after that reconfiguration.
	configureServeLoggerForManager(lifecycle.NewManager())
	stdlog.Printf("[authors] resolved 3 authors")

	if got := terminal.String(); got != "" {
		t.Fatalf("plugin log escaped to the TUI terminal: %q", got)
	}
	logs := runtime.Snapshot().Logs
	if len(logs) != 1 || logs[0].Message != "[authors] resolved 3 authors" {
		t.Fatalf("runtime logs = %#v, want the observed plugin entry", logs)
	}
}

func TestServeLoopbackHost(t *testing.T) {
	for _, host := range []string{"localhost", "127.0.0.1", "::1"} {
		if !serveLoopbackHost(host) {
			t.Errorf("%q should be loopback", host)
		}
	}
	for _, host := range []string{"0.0.0.0", "example.com", "192.168.1.2"} {
		if serveLoopbackHost(host) {
			t.Errorf("%q should not be loopback", host)
		}
	}
}

func TestStartHTTPServer_UsesPreboundListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	server, _ := startHTTPServer(listener, http.NotFoundHandler())
	if server == nil {
		t.Fatal("server was not started from its prebound listener")
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminCommand_IsDiscoverable(t *testing.T) {
	if adminCmd.Name() != "admin" || len(adminCmd.Aliases) == 0 || adminCmd.Aliases[0] != "a" {
		t.Fatal("local admin command and alias are missing")
	}
	for _, name := range []string{"port", "host", "watch"} {
		if adminCmd.Flags().Lookup(name) == nil {
			t.Errorf("admin flag %q is missing", name)
		}
	}
	for _, name := range []string{"admin", "no-tui"} {
		if serveCmd.Flags().Lookup(name) == nil {
			t.Errorf("serve flag %q is missing", name)
		}
	}
}
