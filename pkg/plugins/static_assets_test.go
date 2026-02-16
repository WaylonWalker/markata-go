package plugins

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
)

func TestStaticAssets_CreateHashedCopies_UsesRegistryHash(t *testing.T) {
	plugin := NewStaticAssetsPlugin()

	outputDir := filepath.Join(t.TempDir(), "output")
	cssDir := filepath.Join(outputDir, "css")
	if err := os.MkdirAll(cssDir, 0o755); err != nil {
		t.Fatalf("create css dir: %v", err)
	}

	content := []byte("body { color: red; }")
	origPath := filepath.Join(cssDir, "main.css")
	if err := os.WriteFile(origPath, content, 0o600); err != nil {
		t.Fatalf("write original file: %v", err)
	}

	manager := lifecycle.NewManager()
	config := lifecycle.NewConfig()
	config.OutputDir = outputDir
	manager.SetConfig(config)

	actualHash := fmt.Sprintf("%x", sha256.Sum256(content))[:8]
	wantedHash := "deadbeef"
	if wantedHash == actualHash {
		wantedHash = "feedbeef"
	}
	manager.SetAssetHash("css/main.css", wantedHash)

	if err := plugin.createHashedCopies(manager, outputDir); err != nil {
		t.Fatalf("create hashed copies: %v", err)
	}

	hashedPath := filepath.Join(cssDir, fmt.Sprintf("main.%s.css", wantedHash))
	data, err := os.ReadFile(hashedPath)
	if err != nil {
		t.Fatalf("read hashed file: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("hashed content = %q, want %q", string(data), string(content))
	}

	if wantedHash != actualHash {
		unexpectedPath := filepath.Join(cssDir, fmt.Sprintf("main.%s.css", actualHash))
		if _, err := os.Stat(unexpectedPath); err == nil {
			t.Errorf("unexpected hashed file created: %s", unexpectedPath)
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stat unexpected hash: %v", err)
		}
	}
}
