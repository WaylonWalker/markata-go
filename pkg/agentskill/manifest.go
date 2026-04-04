package agentskill

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ManifestFileName is the name of the manifest file written alongside installed skill files.
const ManifestFileName = ".manifest.json"

// Manifest records the state of an installed skill for drift detection.
type Manifest struct {
	Version     string            `json:"version"`
	InstalledAt time.Time         `json:"installed_at"`
	Target      string            `json:"target"`
	Files       map[string]string `json:"files"`
}

// ComputeBundledManifest builds a Manifest from the current bundled skill files.
// The version and target fields are set by the caller.
func ComputeBundledManifest(version, target string) (*Manifest, error) {
	skillFS, err := SiteSkill()
	if err != nil {
		return nil, fmt.Errorf("load bundled skill: %w", err)
	}

	files, err := ListFiles()
	if err != nil {
		return nil, fmt.Errorf("list bundled skill files: %w", err)
	}

	fileHashes := make(map[string]string, len(files))
	for _, relPath := range files {
		data, readErr := fs.ReadFile(skillFS, relPath)
		if readErr != nil {
			return nil, fmt.Errorf("read bundled file %q: %w", relPath, readErr)
		}
		fileHashes[relPath] = sha256Hex(data)
	}

	return &Manifest{
		Version:     version,
		InstalledAt: time.Now().UTC().Truncate(time.Second),
		Target:      target,
		Files:       fileHashes,
	}, nil
}

// WriteManifest serializes the manifest to JSON and writes it to the given directory.
func WriteManifest(m *Manifest, dir string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	data = append(data, '\n')

	dest := filepath.Join(dir, ManifestFileName)
	if err := os.WriteFile(dest, data, 0o644); err != nil { //nolint:gosec // manifest is non-sensitive metadata
		return fmt.Errorf("write manifest %q: %w", dest, err)
	}
	return nil
}

// ReadManifest reads and parses a manifest file from the given skill directory.
func ReadManifest(dir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, ManifestFileName))
	if err != nil {
		return nil, err
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

// FileStatus describes the drift state of a single file.
type FileStatus string

const (
	FileOK       FileStatus = "ok"
	FileModified FileStatus = "modified"
	FileNew      FileStatus = "new"
	FileMissing  FileStatus = "missing"
)

// DriftReport holds the comparison between installed and bundled skill files.
type DriftReport struct {
	InstalledVersion string
	CurrentVersion   string
	VersionMismatch  bool
	Files            []DriftEntry
}

// DriftEntry describes the drift state of a single file.
type DriftEntry struct {
	Path   string
	Status FileStatus
}

// HasDrift returns true if any file is not in the ok state.
func (r *DriftReport) HasDrift() bool {
	if r.VersionMismatch {
		return true
	}
	for _, entry := range r.Files {
		if entry.Status != FileOK {
			return true
		}
	}
	return false
}

// IssueCount returns the number of files that are not ok.
func (r *DriftReport) IssueCount() int {
	count := 0
	if r.VersionMismatch {
		count++
	}
	for _, entry := range r.Files {
		if entry.Status != FileOK {
			count++
		}
	}
	return count
}

// ComputeDrift compares an installed skill directory against the current bundled skill.
func ComputeDrift(installed *Manifest, skillDir, currentVersion string) (*DriftReport, error) {
	skillFS, err := SiteSkill()
	if err != nil {
		return nil, fmt.Errorf("load bundled skill: %w", err)
	}

	bundledFiles, err := ListFiles()
	if err != nil {
		return nil, fmt.Errorf("list bundled skill files: %w", err)
	}

	// Build a set of files in the manifest for lookup.
	installedSet := make(map[string]string, len(installed.Files))
	for path, hash := range installed.Files {
		installedSet[path] = hash
	}

	// Build a set of bundled files for lookup.
	bundledSet := make(map[string]string, len(bundledFiles))
	for _, relPath := range bundledFiles {
		data, readErr := fs.ReadFile(skillFS, relPath)
		if readErr != nil {
			return nil, fmt.Errorf("read bundled file %q: %w", relPath, readErr)
		}
		bundledSet[relPath] = sha256Hex(data)
	}

	report := &DriftReport{
		InstalledVersion: installed.Version,
		CurrentVersion:   currentVersion,
		VersionMismatch:  installed.Version != currentVersion,
	}

	// Check each bundled file against the installed on-disk file.
	for _, relPath := range bundledFiles {
		bundledHash := bundledSet[relPath]
		_, exists := installedSet[relPath]

		switch {
		case !exists:
			report.Files = append(report.Files, DriftEntry{Path: relPath, Status: FileNew})
		default:
			installedPath := filepath.Join(skillDir, filepath.FromSlash(relPath))
			data, readErr := os.ReadFile(installedPath)
			if readErr != nil {
				if errors.Is(readErr, os.ErrNotExist) {
					report.Files = append(report.Files, DriftEntry{Path: relPath, Status: FileMissing})
					continue
				}
				return nil, fmt.Errorf("read installed file %q: %w", installedPath, readErr)
			}

			if sha256Hex(data) != bundledHash {
				report.Files = append(report.Files, DriftEntry{Path: relPath, Status: FileModified})
				continue
			}

			report.Files = append(report.Files, DriftEntry{Path: relPath, Status: FileOK})
		}
	}

	// Check for files in the manifest that are no longer in the bundle.
	for relPath := range installedSet {
		if _, exists := bundledSet[relPath]; !exists {
			report.Files = append(report.Files, DriftEntry{Path: relPath, Status: FileMissing})
		}
	}

	sort.Slice(report.Files, func(i, j int) bool {
		return report.Files[i].Path < report.Files[j].Path
	})

	return report, nil
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
