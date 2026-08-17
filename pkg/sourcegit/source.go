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

// ReadOptions controls which ignored files contribute to the source
// fingerprint. Ignored files are build inputs when Git-ignore filtering is
// disabled, but generated ignored trees must not be walked when filtering is
// enabled.
type ReadOptions struct {
	IncludeIgnoredContent bool
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
	return ReadWithOptions(ctx, sourceDir, ReadOptions{IncludeIgnoredContent: true})
}

// ReadWithOptions returns the source state using the requested ignored-file
// policy.
func ReadWithOptions(ctx context.Context, sourceDir string, options ReadOptions) (State, error) {
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
	ignoredMode := "--ignored=matching"
	if options.IncludeIgnoredContent {
		ignoredMode = "--ignored"
	}
	ignoredOutput, err := Command(ctx, sourceDir, "status", "--porcelain=v1", "-z", ignoredMode, "--untracked-files=all").Output()
	if err != nil {
		return State{}, fmt.Errorf("read ignored git worktree status: %w", err)
	}
	ignoredSources := ignoredMarkdownFiles(ignoredOutput)
	if options.IncludeIgnoredContent && strings.Contains(string(ignoredOutput), "!! ") {
		dirty = true
	} else if !options.IncludeIgnoredContent && len(ignoredSources) > 0 {
		// An ignored directory such as generated output is not a build input
		// when Git-ignore filtering is enabled. Direct ignored Markdown files
		// remain visible so the source contract can still report them.
		dirty = true
	}
	fingerprint, err := snapshotFingerprint(ctx, sourceDir, statusOutput, ignoredSources, options)
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

func ignoredMarkdownFiles(status []byte) []string {
	var result []string
	for _, record := range splitNUL(status) {
		if !strings.HasPrefix(record, "!! ") {
			continue
		}
		name := strings.TrimPrefix(record, "!! ")
		// Ignore whole directories without walking them. They are already
		// represented as dirty, and walking generated output trees can dominate
		// build time. Direct ignored Markdown files retain byte-level tracking.
		if !strings.HasSuffix(name, "/") && isMarkdownSource(name) {
			result = append(result, name)
		}
	}
	return result
}

func snapshotFingerprint(ctx context.Context, sourceDir string, statusOutput []byte, ignoredSources []string, options ReadOptions) (string, error) {
	hash := sha256.New()
	_, _ = hash.Write(statusOutput)
	// Hash the changed paths and their current bytes instead of asking Git to
	// render a complete binary patch. The latter is needlessly expensive for a
	// large dirty checkout, while path + bytes preserves the same snapshot
	// distinction for tracked content changes.
	rawDiff, err := Command(ctx, sourceDir, "diff", "HEAD", "--raw", "-z", "--").Output()
	if err != nil {
		return "", fmt.Errorf("read changed source metadata: %w", err)
	}
	_, _ = hash.Write(rawDiff)
	changed, err := Command(ctx, sourceDir, "diff", "HEAD", "--name-only", "-z", "--").Output()
	if err != nil {
		return "", fmt.Errorf("read changed source paths: %w", err)
	}
	for _, name := range splitNUL(changed) {
		if name == "" {
			continue
		}
		if err := hashSourceFile(ctx, hash, sourceDir, name, "tracked", options); err != nil {
			return "", err
		}
	}
	untracked, err := Command(ctx, sourceDir, "ls-files", "--others", "--exclude-standard", "-z").Output()
	if err != nil {
		return "", fmt.Errorf("read untracked source files: %w", err)
	}
	for _, name := range splitNUL(untracked) {
		if name == "" {
			continue
		}
		if err := hashSourceFile(ctx, hash, sourceDir, name, "untracked", options); err != nil {
			return "", err
		}
	}
	for _, path := range ignoredSources {
		if err := hashSourceFile(ctx, hash, sourceDir, path, "ignored", options); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func hashSourceFile(ctx context.Context, hash io.Writer, sourceDir, name, kind string, options ReadOptions) error {
	if _, err := io.WriteString(hash, kind+"\x00"+name+"\x00"); err != nil {
		return fmt.Errorf("hash %s source file %q: %w", kind, name, err)
	}
	path := filepath.Join(sourceDir, filepath.FromSlash(name))
	info, lstatErr := os.Lstat(path)
	if lstatErr != nil && !os.IsNotExist(lstatErr) {
		return fmt.Errorf("stat %s source file %q: %w", kind, name, lstatErr)
	}
	if lstatErr == nil && info.Mode()&os.ModeSymlink != 0 {
		if _, err := fmt.Fprintf(hash, "mode\x00%#o\x00", info.Mode()); err != nil {
			return fmt.Errorf("hash mode for source file %q: %w", name, err)
		}
		target, err := os.Readlink(path)
		if err != nil {
			return fmt.Errorf("readlink %s source file %q: %w", kind, name, err)
		}
		if _, err := io.WriteString(hash, "symlink\x00"+target+"\x00"); err != nil {
			return fmt.Errorf("hash symlink source file %q: %w", name, err)
		}
		if targetInfo, err := os.Stat(path); err == nil && targetInfo.Mode().IsRegular() {
			return hashFileContents(hash, path, name, kind)
		}
		return nil
	}
	if lstatErr == nil && info.IsDir() {
		return hashSubmodule(ctx, hash, path, name, options)
	}
	if lstatErr == nil {
		if _, err := fmt.Fprintf(hash, "mode\x00%#o\x00", info.Mode()); err != nil {
			return fmt.Errorf("hash mode for source file %q: %w", name, err)
		}
	}
	return hashFileContents(hash, path, name, kind)
}

func hashSubmodule(ctx context.Context, hash io.Writer, path, name string, options ReadOptions) error {
	head, err := Command(ctx, path, "rev-parse", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("read submodule HEAD %q: %w", name, err)
	}
	status, err := Command(ctx, path, "status", "--porcelain", "--untracked-files=all").Output()
	if err != nil {
		return fmt.Errorf("read submodule status %q: %w", name, err)
	}
	diff, err := Command(ctx, path, "diff", "HEAD", "--binary").Output()
	if err != nil {
		return fmt.Errorf("read submodule diff %q: %w", name, err)
	}
	for _, part := range [][]byte{head, status, diff} {
		if _, err := hash.Write(part); err != nil {
			return fmt.Errorf("hash submodule %q: %w", name, err)
		}
	}
	ignoredMode := "--ignored=matching"
	if options.IncludeIgnoredContent {
		ignoredMode = "--ignored"
	}
	ignoredStatus, err := Command(ctx, path, "status", "--porcelain=v1", "-z", ignoredMode, "--untracked-files=all").Output()
	if err != nil {
		return fmt.Errorf("read submodule ignored files %q: %w", name, err)
	}
	ignoredSources := ignoredMarkdownFiles(ignoredStatus)
	if _, err := hash.Write(ignoredStatus); err != nil {
		return fmt.Errorf("hash submodule ignored files %q: %w", name, err)
	}
	for _, child := range ignoredSources {
		if err := hashSourceFile(ctx, hash, path, child, "submodule-ignored", options); err != nil {
			return err
		}
	}
	untracked, err := Command(ctx, path, "ls-files", "--others", "--exclude-standard", "-z").Output()
	if err != nil {
		return fmt.Errorf("read submodule untracked files %q: %w", name, err)
	}
	for _, child := range splitNUL(untracked) {
		if child == "" {
			continue
		}
		if err := hashSourceFile(ctx, hash, path, child, "submodule-untracked", options); err != nil {
			return err
		}
	}
	return nil
}

func hashFileContents(hash io.Writer, path, name, kind string) error {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s source file %q: %w", kind, name, err)
	}
	if _, err := io.Copy(hash, file); err != nil {
		_ = file.Close()
		return fmt.Errorf("hash %s source file %q: %w", kind, name, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s source file %q: %w", kind, name, err)
	}
	return nil
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
