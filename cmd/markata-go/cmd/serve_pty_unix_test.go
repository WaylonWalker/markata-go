//go:build linux || darwin || freebsd || openbsd || netbsd || dragonfly

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
)

func TestServeTUIOwnsPTYDuringBuildAndResize(t *testing.T) {
	root := markataGoModuleRoot(t)
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "markata-go")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binaryPath, "./cmd/markata-go")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build markata-go: %v\n%s", err, output)
	}

	siteDir := filepath.Join(tempDir, "site")
	contentDir := filepath.Join(siteDir, "content")
	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := `[markata-go]
title = "PTY Serve Test"
url = "http://localhost:8000"
output_dir = "output"

[markata-go.glob]
patterns = ["content/**/*.md"]
use_gitignore = false

[markata-go.search]
enabled = false

[markata-go.blogroll]
enabled = false

[markata-go.mentions]
enabled = false

[markata-go.webmention]
enabled = false

[markata-go.garden]
enabled = false

[markata-go.tailwind]
include = false
build = false

[[markata-go.feeds]]
slug = "all"
title = "All posts"
filter = "published == true"
sort = "title"
`
	if err := os.WriteFile(filepath.Join(siteDir, "markata-go.toml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 24; i++ {
		post := fmt.Sprintf("---\ntitle: PTY post %02d\npublished: true\nauthors: [missing]\n---\n\n# PTY post %02d\n\nBuild output must stay inside the TUI.\n", i, i)
		path := filepath.Join(contentDir, fmt.Sprintf("post-%02d.md", i))
		if err := os.WriteFile(path, []byte(post), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected listener address type %T", listener.Addr())
	}
	port := address.Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binaryPath, "serve", "--config", "markata-go.toml", "--port", fmt.Sprint(port), "--fast")
	cmd.Dir = siteDir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "MARKATA_GO_ENCRYPTION_ENABLED=false")
	terminal, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 32, Cols: 112})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
				t.Logf("kill Serve process during cleanup: %v", err)
			}
			if _, err := cmd.Process.Wait(); err != nil && !errors.Is(err, os.ErrProcessDone) {
				t.Logf("wait for Serve process during cleanup: %v", err)
			}
		}
		if err := terminal.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			t.Logf("close PTY during cleanup: %v", err)
		}
	})

	var output bytes.Buffer
	var outputMu sync.Mutex
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		chunk := make([]byte, 32*1024)
		for {
			n, err := terminal.Read(chunk)
			if n > 0 {
				outputMu.Lock()
				_, _ = output.Write(chunk[:n])
				outputMu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
	snapshotOutput := func() string {
		outputMu.Lock()
		defer outputMu.Unlock()
		return output.String()
	}
	waitForOutput := func(needle string, start int, timeout time.Duration) bool {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			current := snapshotOutput()
			if len(current) >= start && strings.Contains(current[start:], needle) {
				return true
			}
			time.Sleep(50 * time.Millisecond)
		}
		current := snapshotOutput()
		return len(current) >= start && strings.Contains(current[start:], needle)
	}

	// Wait for the initial job to be visible before editing a watched file. This
	// ensures setup completed and the following build is watcher-triggered.
	if !waitForOutput("Initial build", 0, 15*time.Second) {
		t.Fatal("initial build never appeared in the TUI")
	}
	beforeProbe := len(snapshotOutput())
	probe := filepath.Join(contentDir, "watch-probe.md")
	if err := os.WriteFile(probe, []byte("---\ntitle: Watch probe\npublished: true\n---\n\n# Probe\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !waitForOutput("Rebuild", beforeProbe, 10*time.Second) {
		t.Fatal("watcher change did not create a rebuild job")
	}
	for _, size := range []pty.Winsize{
		{Rows: 14, Cols: 44}, {Rows: 42, Cols: 132}, {Rows: 10, Cols: 38},
		{Rows: 24, Cols: 96}, {Rows: 9, Cols: 34}, {Rows: 42, Cols: 132},
		{Rows: 12, Cols: 48}, {Rows: 36, Cols: 120}, {Rows: 9, Cols: 38},
		{Rows: 32, Cols: 112},
	} {
		if err := pty.Setsize(terminal, &size); err != nil {
			t.Fatalf("resize PTY to %dx%d: %v", size.Cols, size.Rows, err)
		}
		time.Sleep(120 * time.Millisecond)
	}
	jobsView := snapshotOutput()
	for _, marker := range []string{"[authors]", "[mentions]", "[render_markdown]", "[build_cache]"} {
		if strings.Contains(jobsView, marker) {
			t.Errorf("ordinary plugin output %q escaped into the Jobs TUI view", marker)
		}
	}
	beforeLogs := len(snapshotOutput())
	if _, err := terminal.WriteString("l"); err != nil {
		t.Fatal(err)
	}
	if !waitForOutput("LOGS", beforeLogs, 2*time.Second) {
		t.Fatal("Logs view was not rendered")
	}
	for _, size := range []pty.Winsize{{Rows: 10, Cols: 40}, {Rows: 36, Cols: 120}, {Rows: 9, Cols: 34}, {Rows: 32, Cols: 112}} {
		if err := pty.Setsize(terminal, &size); err != nil {
			t.Fatalf("resize Logs view to %dx%d: %v", size.Cols, size.Rows, err)
		}
		time.Sleep(120 * time.Millisecond)
	}
	for _, view := range []struct {
		key    []byte
		header string
	}{
		{key: []byte("w"), header: "WARNINGS"},
		{key: []byte{27}},
		{key: []byte{27}},
		{key: []byte("p"), header: "PAGES"},
		{key: []byte("\r"), header: "· PAGE ·"},
		{key: []byte{27}},
		{key: []byte("f"), header: "FEEDS"},
		{key: []byte("\r"), header: "· FEED ·"},
	} {
		beforeView := len(snapshotOutput())
		if _, err := terminal.Write(view.key); err != nil {
			t.Fatal(err)
		}
		time.Sleep(120 * time.Millisecond)
		if err := pty.Setsize(terminal, &pty.Winsize{Rows: 10, Cols: 40}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(120 * time.Millisecond)
		if err := pty.Setsize(terminal, &pty.Winsize{Rows: 36, Cols: 120}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(120 * time.Millisecond)
		if view.header != "" && !waitForOutput(view.header, beforeView, 2*time.Second) {
			t.Errorf("key %q did not render the %s view", view.key, view.header)
		}
	}

	if _, err := terminal.WriteString("q"); err != nil {
		t.Fatal(err)
	}
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	select {
	case err := <-wait:
		if err != nil {
			t.Fatalf("serve exited with error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Serve did not quit after q")
	}
	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("PTY output reader did not stop")
	}

	captured := snapshotOutput()
	if !strings.Contains(captured, "?1049h") || !strings.Contains(captured, "?1049l") {
		t.Fatalf("TUI did not enter and restore the alternate screen")
	}
}

func markataGoModuleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
