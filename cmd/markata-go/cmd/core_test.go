package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/buildstats"
	"github.com/WaylonWalker/markata-go/pkg/models"
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

func TestCreateManagerProjectsAffectedConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "markata-go.toml")
	config := `[markata-go]
[markata-go.head]
text = "typed head"
[markata-go.theme_calendar]
enabled = false
[markata-go.error_pages]
enable_404 = false
[markata-go.resource_hints]
enabled = false
[markata-go.markdown.highlight]
enabled = false
theme = "monokai"
`
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	manager, err := createManager(configPath)
	if err != nil {
		t.Fatal(err)
	}

	extra := manager.Config().Extra
	if head, ok := extra["head"].(models.HeadConfig); !ok || head.Text != "typed head" {
		t.Errorf("head extra = %#v, want typed head config", extra["head"])
	}
	if calendar, ok := extra["theme_calendar"].(models.ThemeCalendarConfig); !ok || calendar.IsEnabled() {
		t.Errorf("theme_calendar extra = %#v, want disabled typed config", extra["theme_calendar"])
	}
	if errorPages, ok := extra["error_pages"].(models.ErrorPagesConfig); !ok || errorPages.Is404Enabled() {
		t.Errorf("error_pages extra = %#v, want disabled typed config", extra["error_pages"])
	}
	if resourceHints, ok := extra["resource_hints"].(models.ResourceHintsConfig); !ok || resourceHints.IsEnabled() {
		t.Errorf("resource_hints extra = %#v, want disabled typed config", extra["resource_hints"])
	}
	if markdown, ok := extra["markdown"].(models.MarkdownConfig); !ok || markdown.Highlight.IsEnabled() || markdown.Highlight.Theme != "monokai" {
		t.Errorf("markdown extra = %#v, want typed highlight config", extra["markdown"])
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
