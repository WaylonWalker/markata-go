package testbinary

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolve_UsesProvidedBinaryFromCurrentRevision(t *testing.T) {
	moduleRoot, revision := currentCheckout(t)

	binary := filepath.Join(t.TempDir(), "markata-go")
	mode := os.FileMode(0o600)
	if runtime.GOOS != "windows" {
		mode = 0o700
	}
	if err := os.WriteFile(binary, []byte("test executable"), mode); err != nil {
		t.Fatal(err)
	}
	t.Setenv(BinaryEnv, binary)
	t.Setenv(RevisionEnv, revision)

	got, err := Resolve(moduleRoot, t.TempDir())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got != binary {
		t.Fatalf("Resolve() = %q, want supplied binary %q", got, binary)
	}
}

func TestResolve_RejectsBinaryFromDifferentRevision(t *testing.T) {
	moduleRoot, _ := currentCheckout(t)
	binary := filepath.Join(t.TempDir(), "markata-go")
	mode := os.FileMode(0o600)
	if runtime.GOOS != "windows" {
		mode = 0o700
	}
	if err := os.WriteFile(binary, []byte("test executable"), mode); err != nil {
		t.Fatal(err)
	}
	t.Setenv(BinaryEnv, binary)
	t.Setenv(RevisionEnv, "stale-checkout-revision")

	if _, err := Resolve(moduleRoot, t.TempDir()); err == nil {
		t.Fatal("Resolve() succeeded with a binary from a different revision")
	}
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func currentCheckout(t *testing.T) (moduleRoot, revision string) {
	t.Helper()
	moduleRoot = filepath.Clean(filepath.Join(mustGetwd(t), "..", ".."))
	command := exec.Command("git", "rev-parse", "HEAD")
	command.Dir = moduleRoot
	output, err := command.Output()
	if err != nil {
		t.Skipf("test requires a Git checkout: %v", err)
	}
	return moduleRoot, strings.TrimSpace(string(output))
}
