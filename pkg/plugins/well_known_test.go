package plugins

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestWellKnownPlugin_Write_DefaultEntries(t *testing.T) {
	outputDir := t.TempDir()
	wellKnownConfig := models.NewWellKnownConfig()
	config := &lifecycle.Config{
		OutputDir: outputDir,
		Extra: map[string]interface{}{
			"url":         "https://example.com",
			"title":       "Example Site",
			"description": "Example description",
			"author":      "Jane Doe",
			"well_known":  wellKnownConfig,
		},
	}

	m := lifecycle.NewManager()
	m.SetConfig(config)

	plugin := NewWellKnownPlugin()
	plugin.now = func() time.Time {
		return time.Date(2026, time.February, 4, 12, 34, 56, 0, time.UTC)
	}

	if err := plugin.Write(m); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	expected := []string{
		".well-known/host-meta",
		".well-known/host-meta.json",
		".well-known/webfinger",
		".well-known/nodeinfo",
		".well-known/time",
		"nodeinfo/2.0",
	}

	for _, rel := range expected {
		path := filepath.Join(outputDir, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}

	content, err := os.ReadFile(filepath.Join(outputDir, filepath.FromSlash(".well-known/time")))
	if err != nil {
		t.Fatalf("reading time file: %v", err)
	}
	if string(content) != "2026-02-04T12:34:56Z\n" {
		t.Fatalf("time content = %q, want %q", string(content), "2026-02-04T12:34:56Z\n")
	}
}

func TestWellKnownPlugin_Write_OptionalEntriesOnly(t *testing.T) {
	outputDir := t.TempDir()
	enabled := true
	wellKnownConfig := models.WellKnownConfig{
		Enabled:         &enabled,
		AutoGenerate:    []string{},
		SSHFingerprint:  "SHA256:abcdef",
		KeybaseUsername: "alice",
	}
	config := &lifecycle.Config{
		OutputDir: outputDir,
		Extra: map[string]interface{}{
			"url":        "https://example.com",
			"well_known": wellKnownConfig,
		},
	}

	m := lifecycle.NewManager()
	m.SetConfig(config)

	plugin := NewWellKnownPlugin()
	plugin.now = func() time.Time { return time.Date(2026, time.February, 4, 0, 0, 0, 0, time.UTC) }

	if err := plugin.Write(m); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	sshfpPath := filepath.Join(outputDir, filepath.FromSlash(".well-known/sshfp"))
	if _, err := os.Stat(sshfpPath); err != nil {
		t.Fatalf("expected %s to exist: %v", sshfpPath, err)
	}

	keybasePath := filepath.Join(outputDir, filepath.FromSlash(".well-known/keybase.txt"))
	if _, err := os.Stat(keybasePath); err != nil {
		t.Fatalf("expected %s to exist: %v", keybasePath, err)
	}

	if _, err := os.Stat(filepath.Join(outputDir, filepath.FromSlash(".well-known/host-meta"))); err == nil {
		t.Fatalf("did not expect host-meta to be generated when auto_generate is empty")
	}
}
