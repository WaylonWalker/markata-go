// Package sourcegit provides the canonical Git state used for source builds.
package sourcegit

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// State describes the checked-out source tree. Dirty is nil when Git state is
// unavailable, rather than pretending that an unavailable tree is clean.
type State struct {
	Commit string
	Dirty  *bool
	// fingerprint is internal build-snapshot data. It is not part of the wire
	// format because a dirty source is never revision-equivalent.
	fingerprint string
}

// Command creates a Git command rooted at sourceDir using the same safe
// directory handling as the builder-admin source checkout.
func Command(ctx context.Context, sourceDir string, args ...string) *exec.Cmd {
	gitArgs := append([]string{"-c", "safe.directory=" + sourceDir, "-C", sourceDir}, args...)
	return exec.CommandContext(ctx, "git", gitArgs...)
}

// Read returns HEAD and whether Git reports any tracked, staged, deleted,
// untracked, or ignored files in the source tree.
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
	// A build may disable Markata's .gitignore filtering. Treat ignored files
	// as dirty as well, because Git cannot prove that they are not inputs.
	ignoredOutput, err := Command(ctx, sourceDir, "status", "--porcelain=v1", "-z", "--ignored", "--untracked-files=all").Output()
	if err != nil {
		return State{}, fmt.Errorf("read ignored git worktree status: %w", err)
	}
	ignoredSources, err := ignoredMarkdownFiles(sourceDir, ignoredOutput)
	if err != nil {
		return State{}, err
	}
	if len(ignoredSources) > 0 {
		dirty = true
	}
	fingerprint, err := snapshotFingerprint(ctx, sourceDir, statusOutput, ignoredSources)
	if err != nil {
		return State{}, err
	}
	finalCommit, err := Head(ctx, sourceDir)
	if err != nil {
		return State{}, err
	}
	if finalCommit != commit {
		return State{}, fmt.Errorf("source Git HEAD changed while reading status")
	}
	return State{Commit: commit, Dirty: &dirty, fingerprint: fingerprint}, nil
}

// Equal reports whether two complete source snapshots are identical.
func (s State) Equal(other State) bool {
	if s.Commit != other.Commit {
		return false
	}
	if s.fingerprint != other.fingerprint {
		return false
	}
	if s.Dirty == nil || other.Dirty == nil {
		return s.Dirty == nil && other.Dirty == nil
	}
	return *s.Dirty == *other.Dirty
}

func ignoredMarkdownFiles(sourceDir string, status []byte) ([]string, error) {
	var result []string
	for _, record := range splitNUL(status) {
		if !strings.HasPrefix(record, "!! ") {
			continue
		}
		name := strings.TrimPrefix(record, "!! ")
		path := filepath.Join(sourceDir, filepath.FromSlash(name))
		if strings.HasSuffix(name, "/") {
			if err := filepath.WalkDir(path, func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !entry.IsDir() && isMarkdownSource(path) {
					relative, err := filepath.Rel(sourceDir, path)
					if err != nil {
						return err
					}
					result = append(result, relative)
				}
				return nil
			}); err != nil {
				return nil, fmt.Errorf("scan ignored source directory %q: %w", name, err)
			}
		} else if isMarkdownSource(name) {
			result = append(result, name)
		}
	}
	return result, nil
}

func snapshotFingerprint(ctx context.Context, sourceDir string, statusOutput []byte, ignoredSources []string) (string, error) {
	hash := sha256.New()
	_, _ = hash.Write(statusOutput)
	diff, err := Command(ctx, sourceDir, "diff", "HEAD", "--binary").Output()
	if err != nil {
		return "", fmt.Errorf("read git worktree diff: %w", err)
	}
	_, _ = hash.Write(diff)
	untracked, err := Command(ctx, sourceDir, "ls-files", "--others", "--exclude-standard", "-z").Output()
	if err != nil {
		return "", fmt.Errorf("read untracked source files: %w", err)
	}
	for _, name := range splitNUL(untracked) {
		if name == "" {
			continue
		}
		_, _ = io.WriteString(hash, name)
		file, err := os.Open(filepath.Join(sourceDir, filepath.FromSlash(name)))
		if err != nil {
			return "", fmt.Errorf("read untracked source file %q: %w", name, err)
		}
		if _, err := io.Copy(hash, file); err != nil {
			_ = file.Close()
			return "", fmt.Errorf("hash untracked source file %q: %w", name, err)
		}
		if err := file.Close(); err != nil {
			return "", fmt.Errorf("close untracked source file %q: %w", name, err)
		}
	}
	for _, path := range ignoredSources {
		name := path
		_, _ = io.WriteString(hash, name)
		file, err := os.Open(filepath.Join(sourceDir, filepath.FromSlash(name)))
		if err != nil {
			return "", fmt.Errorf("read ignored source file %q: %w", name, err)
		}
		if _, err := io.Copy(hash, file); err != nil {
			_ = file.Close()
			return "", fmt.Errorf("hash ignored source file %q: %w", name, err)
		}
		if err := file.Close(); err != nil {
			return "", fmt.Errorf("close ignored source file %q: %w", name, err)
		}
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func isMarkdownSource(name string) bool {
	return strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".markdown") || strings.HasSuffix(name, ".mdx")
}

func splitNUL(value []byte) []string {
	return strings.Split(strings.TrimSuffix(string(value), "\x00"), "\x00")
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
