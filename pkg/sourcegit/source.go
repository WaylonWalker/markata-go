// Package sourcegit provides the canonical Git state used for source builds.
package sourcegit

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// State describes the checked-out source tree. Dirty is nil when Git state is
// unavailable, rather than pretending that an unavailable tree is clean.
type State struct {
	Commit string
	Dirty  *bool
}

// Command creates a Git command rooted at sourceDir using the same safe
// directory handling as the builder-admin source checkout.
func Command(ctx context.Context, sourceDir string, args ...string) *exec.Cmd {
	gitArgs := append([]string{"-c", "safe.directory=" + sourceDir, "-C", sourceDir}, args...)
	return exec.CommandContext(ctx, "git", gitArgs...)
}

// Read returns HEAD and whether Git reports any tracked, staged, deleted, or
// untracked files in the source tree.
func Read(ctx context.Context, sourceDir string) (State, error) {
	commit, err := Head(ctx, sourceDir)
	if err != nil {
		return State{}, err
	}
	statusOutput, err := Command(ctx, sourceDir, "status", "--porcelain", "--untracked-files=all").Output()
	if err != nil {
		return State{}, fmt.Errorf("read git worktree status: %w", err)
	}
	dirty := strings.TrimSpace(string(statusOutput)) != ""
	return State{Commit: commit, Dirty: &dirty}, nil
}

// Head returns only the checked-out commit without scanning worktree status.
func Head(ctx context.Context, sourceDir string) (string, error) {
	output, err := Command(ctx, sourceDir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("read git HEAD: %w", err)
	}
	commit := strings.TrimSpace(string(output))
	if commit == "" {
		return "", fmt.Errorf("git returned an empty HEAD")
	}
	return commit, nil
}
