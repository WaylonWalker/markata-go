package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunSearchServer_InvalidMode(t *testing.T) {
	originalMode := searchServerMode
	defer func() { searchServerMode = originalMode }()

	searchServerMode = "invalid"
	err := runSearchServer(searchServerCmd, nil)
	if err == nil {
		t.Fatal("expected invalid mode error")
	}
	if err.Error() != "invalid --mode \"invalid\" (expected runtime-index, watch-content, or read-only-index)" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunSearchServer_ReadOnlyRequiresIndexDir(t *testing.T) {
	originalMode := searchServerMode
	originalIndexDir := searchServerIndexDir
	defer func() {
		searchServerMode = originalMode
		searchServerIndexDir = originalIndexDir
	}()

	searchServerMode = "read-only-index"
	searchServerIndexDir = ""
	err := runSearchServer(searchServerCmd, nil)
	if err == nil {
		t.Fatal("expected missing index-dir error")
	}
	if err.Error() != "--index-dir is required in read-only-index mode" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchBuildIndexCommand_BuildsArtifacts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	indexDir := filepath.Join(tmpDir, "search.bleve")
	hashPath := filepath.Join(tmpDir, "search.hash")

	originalCfg := cfgFile
	originalMerge := mergeConfigFiles
	defer func() {
		cfgFile = originalCfg
		mergeConfigFiles = originalMerge
	}()

	cfgFile = filepath.Join("..", "..", "..", "benchmarks", "site", "markata-go.toml")
	mergeConfigFiles = nil

	cmd := searchBuildIndexCmd()
	cmd.SetArgs([]string{"--index-dir", indexDir, "--hash-path", hashPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute build-index: %v", err)
	}

	if info, err := os.Stat(indexDir); err != nil || !info.IsDir() {
		t.Fatalf("expected index dir to exist: %v", err)
	}
	if _, err := os.Stat(hashPath); err != nil {
		t.Fatalf("expected hash file to exist: %v", err)
	}
}
