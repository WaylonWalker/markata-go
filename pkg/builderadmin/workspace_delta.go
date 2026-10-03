package builderadmin

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type workspacePublicationStats struct {
	LinkedFiles         int64
	CopiedFiles         int64
	CopiedBytes         int64
	ComparedBytes       int64
	CachedSourceHashes  int64
	CachedReleaseHashes int64
	ManifestSaved       bool
}

// stageIncrementalWorkspace retains an independent mutable workspace. Only
// immutable releases share file inodes; failed or later builds cannot mutate them.
func stageIncrementalWorkspace(workspace, releasesDir, releaseID, baselineID string) (string, workspacePublicationStats, error) {
	return stageIncrementalWorkspaceWithLink(workspace, releasesDir, releaseID, baselineID, func(root *os.Root, old, next string) error { return root.Link(old, next) })
}

func stageIncrementalWorkspaceWithLink(workspace, releasesDir, releaseID, baselineID string, link func(*os.Root, string, string) error) (string, workspacePublicationStats, error) {
	var stats workspacePublicationStats
	if workspace == "" || releasesDir == "" || !plainReleaseID(releaseID) {
		return "", stats, fmt.Errorf("workspace, releases directory, and plain release ID are required")
	}
	if pathsOverlap(workspace, releasesDir) {
		return "", stats, fmt.Errorf("workspace and releases directory must not overlap")
	}
	info, err := os.Stat(workspace)
	if err != nil {
		return "", stats, fmt.Errorf("stat build workspace: %w", err)
	}
	if !info.IsDir() {
		return "", stats, fmt.Errorf("build workspace is not a directory: %s", workspace)
	}
	if err := os.MkdirAll(releasesDir, 0o755); err != nil {
		return "", stats, err
	}
	root, err := os.OpenRoot(releasesDir)
	if err != nil {
		return "", stats, err
	}
	defer root.Close()
	if _, err := root.Lstat(releaseID); err == nil {
		return "", stats, fmt.Errorf("release already exists: %s", releaseID)
	} else if !os.IsNotExist(err) {
		return "", stats, fmt.Errorf("stat release destination: %w", err)
	}
	if !plainReleaseID(baselineID) {
		baselineID = ""
	}
	staging, err := os.MkdirTemp(releasesDir, ".staging-"+releaseID+"-")
	if err != nil {
		return "", stats, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(staging)
		}
	}()
	publisher := workspaceDeltaPublisher{workspace: workspace, staging: staging, baselineID: baselineID, root: root, link: link, previous: loadPublicationManifest(workspace, baselineID), records: make(map[string]publicationRecord)}
	err = publisher.copyTree()
	stats = publisher.stats
	stageName := filepath.Base(staging)
	if err != nil {
		return "", stats, fmt.Errorf("stage incremental workspace: %w", err)
	}
	if err := publisher.finishDirectories(); err != nil {
		return "", stats, err
	}
	if err := os.Chmod(staging, 0o755); err != nil {
		return "", stats, err
	}
	if err := root.Rename(stageName, releaseID); err != nil {
		return "", stats, err
	}
	committed = true
	stats.ManifestSaved = savePublicationManifest(workspace, releaseID, publisher.records)
	return filepath.Join(releasesDir, releaseID), stats, nil
}

type workspaceDeltaPublisher struct {
	workspace, staging, baselineID string
	root                           *os.Root
	link                           func(*os.Root, string, string) error
	sourceBuffer, baselineBuffer   []byte
	stats                          workspacePublicationStats
	directories                    []workspaceDirectory
	previous                       publicationManifest
	records                        map[string]publicationRecord
}

func (p *workspaceDeltaPublisher) copyEntry(path string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	rel, err := filepath.Rel(p.workspace, path)
	if err != nil || rel == "." {
		return err
	}
	next := filepath.Join(filepath.Base(p.staging), rel)
	target := filepath.Join(p.staging, rel)
	info, err := entry.Info()
	if err != nil {
		return err
	}
	if entry.Type()&os.ModeSymlink != 0 {
		destination, err := os.Readlink(path)
		if err != nil {
			return err
		}
		return os.Symlink(destination, target)
	}
	if entry.IsDir() {
		if err := os.Mkdir(target, 0o700); err != nil {
			return err
		}
		p.directories = append(p.directories, workspaceDirectory{path: target, info: info})
		return nil
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("unsupported workspace entry %s (%s)", path, info.Mode())
	}
	return p.copyRegular(path, target, next, rel, info)
}

func pathsOverlap(first, second string) bool {
	first, err := filepath.Abs(first)
	if err != nil {
		return true
	}
	second, err = filepath.Abs(second)
	if err != nil {
		return true
	}
	for _, pair := range [][2]string{{first, second}, {second, first}} {
		rel, err := filepath.Rel(pair[0], pair[1])
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func plainReleaseID(id string) bool {
	return id != "" && filepath.IsLocal(id) && filepath.Base(id) == id && id[0] != '.'
}

type workspaceDirectory struct {
	path string
	info fs.FileInfo
}

func (p *workspaceDeltaPublisher) finishDirectories() error {
	for i := len(p.directories) - 1; i >= 0; i-- {
		directory := p.directories[i]
		if err := os.Chmod(directory.path, directory.info.Mode().Perm()); err != nil {
			return err
		}
		if err := os.Chtimes(directory.path, directory.info.ModTime(), directory.info.ModTime()); err != nil {
			return err
		}
	}
	return nil
}

const workspacePublicationWorkers = 8

type workspaceCopyJob struct {
	path  string
	entry fs.DirEntry
}

type workspaceCopyResult struct {
	stats   workspacePublicationStats
	records map[string]publicationRecord
	err     error
}

func (p *workspaceDeltaPublisher) copyTree() error {
	jobs := make(chan workspaceCopyJob, workspacePublicationWorkers)
	results := make(chan workspaceCopyResult, workspacePublicationWorkers)
	for range workspacePublicationWorkers {
		go func() {
			worker := workspaceDeltaPublisher{workspace: p.workspace, staging: p.staging, baselineID: p.baselineID, root: p.root, link: p.link, sourceBuffer: make([]byte, 64<<10), baselineBuffer: make([]byte, 64<<10), previous: p.previous, records: make(map[string]publicationRecord)}
			var workerErr error
			for job := range jobs {
				if workerErr == nil {
					workerErr = worker.copyEntry(job.path, job.entry, nil)
				}
			}
			results <- workspaceCopyResult{stats: worker.stats, records: worker.records, err: workerErr}
		}()
	}
	walkErr := filepath.WalkDir(p.workspace, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return p.copyEntry(path, entry, nil)
		}
		jobs <- workspaceCopyJob{path: path, entry: entry}
		return nil
	})
	close(jobs)
	for range workspacePublicationWorkers {
		result := <-results
		p.stats.LinkedFiles += result.stats.LinkedFiles
		p.stats.CopiedFiles += result.stats.CopiedFiles
		p.stats.CopiedBytes += result.stats.CopiedBytes
		p.stats.ComparedBytes += result.stats.ComparedBytes
		p.stats.CachedSourceHashes += result.stats.CachedSourceHashes
		p.stats.CachedReleaseHashes += result.stats.CachedReleaseHashes
		for path, record := range result.records {
			p.records[path] = record
		}
		if walkErr == nil {
			walkErr = result.err
		}
	}
	return walkErr
}
