// Package testbinary provides the Markata CLI binary used by integration tests.
package testbinary

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	BinaryEnv   = "MARKATA_GO_TEST_BINARY"
	RevisionEnv = "MARKATA_GO_TEST_BINARY_REVISION"
	windowsGOOS = "windows"
)

// Resolve returns the CLI binary supplied by CI, or builds one in fallbackDir
// for local tests. CI sets the path only after building from the checked-out
// revision; the revision check prevents accidentally reusing a stale binary.
func Resolve(moduleRoot, fallbackDir string) (string, error) {
	moduleRoot, err := filepath.Abs(moduleRoot)
	if err != nil {
		return "", fmt.Errorf("resolve module root: %w", err)
	}
	if binary := os.Getenv(BinaryEnv); binary != "" {
		if err := validateProvidedBinary(moduleRoot, binary); err != nil {
			return "", err
		}
		return binary, nil
	}

	binary := filepath.Join(fallbackDir, "markata-go")
	if runtime.GOOS == windowsGOOS {
		binary += ".exe"
	}
	command := exec.Command("go", "build", "-o", binary, "./cmd/markata-go")
	command.Dir = moduleRoot
	if output, err := command.CombinedOutput(); err != nil {
		return "", fmt.Errorf("build markata-go: %w\n%s", err, output)
	}
	return binary, nil
}

func validateProvidedBinary(moduleRoot, binary string) error {
	if !filepath.IsAbs(binary) {
		return fmt.Errorf("%s must be an absolute path, got %q", BinaryEnv, binary)
	}
	info, err := os.Stat(binary)
	if err != nil {
		return fmt.Errorf("stat %s: %w", BinaryEnv, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file: %q", BinaryEnv, binary)
	}
	if runtime.GOOS != windowsGOOS && info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("%s is not executable: %q", BinaryEnv, binary)
	}

	expectedRevision := os.Getenv(RevisionEnv)
	if expectedRevision == "" {
		return fmt.Errorf("%s is set but %s is missing", BinaryEnv, RevisionEnv)
	}
	command := exec.Command("git", "rev-parse", "HEAD")
	command.Dir = moduleRoot
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("read checked-out revision for %s: %w", BinaryEnv, err)
	}
	if revision := strings.TrimSpace(string(output)); revision != expectedRevision {
		return fmt.Errorf("%s was built for revision %s, current checkout is %s", BinaryEnv, expectedRevision, revision)
	}
	return nil
}
