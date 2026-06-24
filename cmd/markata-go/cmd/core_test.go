package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/buildstats"
	"github.com/spf13/cobra"
)

func TestResolveConfigBaseDir(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "markata-go.toml")
	if err := os.WriteFile(configPath, []byte("title = \"test\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}

	got := resolveConfigBaseDir(configPath)
	want := filepath.Clean(configDir)
	if got != want {
		t.Fatalf("resolveConfigBaseDir() = %q, want %q", got, want)
	}
}

func TestResolveConfigRelativePath(t *testing.T) {
	baseDir := t.TempDir()
	absOut := filepath.Join(t.TempDir(), "out")

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "relative path", path: "output", want: filepath.Join(baseDir, "output")},
		{name: "absolute path", path: absOut, want: absOut},
		{name: "empty path", path: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveConfigRelativePath(baseDir, tt.path); got != tt.want {
				t.Fatalf("resolveConfigRelativePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestPrintBuildResult_IncludesBenchmarkSummary(t *testing.T) {
	stdout := bytes.NewBuffer(nil)
	command := &cobra.Command{Use: "build"}
	command.SetOut(stdout)
	currentCmd = command
	defer func() { currentCmd = nil }()

	printBuildResult(&BuildResult{
		PostsProcessed: 12,
		FeedsGenerated: 3,
		Duration:       9.87,
		Benchmark: buildstats.Summary{
			Total: 10 * time.Second,
			Resources: buildstats.ResourceBreakdown{
				CPU:           2 * time.Second,
				NetworkWait:   5 * time.Second,
				DiskReadWait:  750 * time.Millisecond,
				DiskWriteWait: 1250 * time.Millisecond,
				Idle:          1 * time.Second,
			},
			Hotspots: []buildstats.Hotspot{
				{Stage: "collect", Plugin: "blogroll", Duration: 3 * time.Second},
				{Stage: "render", Plugin: "link_avatars", Duration: 2 * time.Second},
			},
			Requests: []buildstats.RequestTiming{
				{Stage: "collect", Plugin: "blogroll", Method: "GET", URL: "https://example.com/feed.xml", Duration: 4 * time.Second, Status: 200},
			},
		},
	})

	output := stdout.String()
	for _, want := range []string{
		"Build completed successfully!",
		"Posts processed: 12",
		"Feeds generated: 3",
		"Resource profile: estimated wall time",
		"CPU",
		"Network wait",
		"Disk read",
		"Disk write",
		"Idle",
		"Hotspots:",
		"collect/blogroll",
		"render/link_avatars",
		"Slowest requests:",
		"GET https://example.com/feed.xml",
		"Duration: 9.87s",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}
