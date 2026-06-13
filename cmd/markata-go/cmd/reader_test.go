package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunReaderUpdateCommand_UsesCommandWriterAndRefreshesCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		if _, err := w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Example Feed</title>
    <link>https://example.com/</link>
    <description>Example description</description>
    <item>
      <title>Fresh entry</title>
      <link>https://example.com/fresh-entry</link>
      <guid>fresh-entry</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 GMT</pubDate>
      <description>Hello reader</description>
    </item>
  </channel>
</rss>`)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache", "blogroll")
	configPath := filepath.Join(tmpDir, "markata-go.toml")
	configContent := strings.Join([]string{
		"[markata-go]",
		`title = "Test"`,
		`url = "https://example.com"`,
		"",
		"[markata-go.blogroll]",
		"enabled = true",
		`cache_dir = "` + filepath.ToSlash(cacheDir) + `"`,
		`cache_duration = "24h"`,
		"",
		"[[markata-go.blogroll.feeds]]",
		`url = "` + server.URL + `"`,
	}, "\n")
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	command := &cobra.Command{Use: "update"}
	command.SetOut(stdout)
	command.SetErr(stderr)

	originalCfgFile := cfgFile
	originalConcurrency := readerUpdateConcurrency
	defer func() {
		cfgFile = originalCfgFile
		readerUpdateConcurrency = originalConcurrency
		currentCmd = nil
	}()

	cfgFile = configPath
	readerUpdateConcurrency = 3
	currentCmd = command

	if err := runReaderUpdateCommand(command, nil); err != nil {
		t.Fatalf("runReaderUpdateCommand() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "Reader cache updated: 1 refreshed, 0 stale fallback, 0 failed, 1 entries") {
		t.Fatalf("expected refresh summary in stdout, got %q", stdout.String())
	}
	if !strings.Contains(filepath.ToSlash(stdout.String()), filepath.ToSlash(filepath.Join("cache", "blogroll"))) {
		t.Fatalf("expected resolved cache dir in stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Refreshing reader cache for 1 feed(s) with concurrency 3...") {
		t.Fatalf("expected progress start on stderr, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "[1/1] refreshed Example Feed") {
		t.Fatalf("expected per-feed progress on stderr, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Reader cache refresh complete: 1/1 feeds processed") {
		t.Fatalf("expected completion progress on stderr, got %q", stderr.String())
	}

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("read cache dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 cache file, got %d", len(entries))
	}

	cacheData, err := os.ReadFile(filepath.Join(cacheDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("read cache file: %v", err)
	}
	if !strings.Contains(string(cacheData), "Fresh entry") {
		t.Fatalf("expected cache file to contain refreshed entry, got %q", string(cacheData))
	}
}

func TestRunReaderUpdateCommand_DisabledBlogrollExplainsConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "markata-go.toml")
	configContent := strings.Join([]string{
		"[markata-go]",
		`title = "Test"`,
		`url = "https://example.com"`,
	}, "\n")
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	command := &cobra.Command{Use: "update"}
	command.SetOut(stdout)
	command.SetErr(stderr)

	originalCfgFile := cfgFile
	originalConcurrency := readerUpdateConcurrency
	defer func() {
		cfgFile = originalCfgFile
		readerUpdateConcurrency = originalConcurrency
		currentCmd = nil
	}()

	cfgFile = configPath
	readerUpdateConcurrency = 0
	currentCmd = command

	if err := runReaderUpdateCommand(command, nil); err != nil {
		t.Fatalf("runReaderUpdateCommand() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "Blogroll reader is not enabled in configuration.") {
		t.Fatalf("expected config guidance in stdout, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output for disabled config, got %q", stderr.String())
	}
}

func TestRunReaderUpdateCommand_RejectsNegativeConcurrency(t *testing.T) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	command := &cobra.Command{Use: "update"}
	command.SetOut(stdout)
	command.SetErr(stderr)

	originalCfgFile := cfgFile
	originalConcurrency := readerUpdateConcurrency
	defer func() {
		cfgFile = originalCfgFile
		readerUpdateConcurrency = originalConcurrency
		currentCmd = nil
	}()

	readerUpdateConcurrency = -1
	currentCmd = command

	err := runReaderUpdateCommand(command, nil)
	if err == nil {
		t.Fatal("expected error for negative concurrency")
	}
	if !strings.Contains(err.Error(), "--concurrency must be 0 or greater") {
		t.Fatalf("unexpected error: %v", err)
	}
}
