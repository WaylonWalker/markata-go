package agentskill

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestComputeBundledManifest_IncludesAllFiles(t *testing.T) {
	files, err := ListFiles()
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}

	manifest, err := ComputeBundledManifest("0.1.0-test", "agents")
	if err != nil {
		t.Fatalf("ComputeBundledManifest() error = %v", err)
	}

	if manifest.Version != "0.1.0-test" {
		t.Errorf("Version = %q, want %q", manifest.Version, "0.1.0-test")
	}
	if manifest.Target != "agents" {
		t.Errorf("Target = %q, want %q", manifest.Target, "agents")
	}
	if len(manifest.Files) != len(files) {
		t.Errorf("Files count = %d, want %d", len(manifest.Files), len(files))
	}

	for _, f := range files {
		hash, ok := manifest.Files[f]
		if !ok {
			t.Errorf("missing file %q in manifest", f)
			continue
		}
		if len(hash) != 64 {
			t.Errorf("file %q hash length = %d, want 64 hex chars", f, len(hash))
		}
	}
}

func TestComputeBundledManifest_DeterministicHashes(t *testing.T) {
	m1, err := ComputeBundledManifest("v1", "agents")
	if err != nil {
		t.Fatalf("first ComputeBundledManifest() error = %v", err)
	}
	m2, err := ComputeBundledManifest("v2", "claude")
	if err != nil {
		t.Fatalf("second ComputeBundledManifest() error = %v", err)
	}

	// File hashes should be identical regardless of version/target metadata.
	for path, hash1 := range m1.Files {
		hash2, ok := m2.Files[path]
		if !ok {
			t.Errorf("missing file %q in second manifest", path)
			continue
		}
		if hash1 != hash2 {
			t.Errorf("hash mismatch for %q: %q != %q", path, hash1, hash2)
		}
	}
}

func TestWriteAndReadManifest_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	original, err := ComputeBundledManifest("0.5.0", "agents")
	if err != nil {
		t.Fatalf("ComputeBundledManifest() error = %v", err)
	}

	if err := WriteManifest(original, dir); err != nil {
		t.Fatalf("WriteManifest() error = %v", err)
	}

	// Verify the file exists.
	manifestPath := filepath.Join(dir, ManifestFileName)
	info, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatalf("manifest file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("manifest file is empty")
	}

	loaded, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}

	if loaded.Version != original.Version {
		t.Errorf("Version = %q, want %q", loaded.Version, original.Version)
	}
	if loaded.Target != original.Target {
		t.Errorf("Target = %q, want %q", loaded.Target, original.Target)
	}
	if len(loaded.Files) != len(original.Files) {
		t.Errorf("Files count = %d, want %d", len(loaded.Files), len(original.Files))
	}
	for path, hash := range original.Files {
		if loaded.Files[path] != hash {
			t.Errorf("file %q hash = %q, want %q", path, loaded.Files[path], hash)
		}
	}
}

func TestReadManifest_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadManifest(dir)
	if err == nil {
		t.Fatal("expected error reading missing manifest, got nil")
	}
}

func TestComputeDrift_NoDrift(t *testing.T) {
	dir := t.TempDir()
	manifest, err := ComputeBundledManifest("0.5.0", "agents")
	if err != nil {
		t.Fatalf("ComputeBundledManifest() error = %v", err)
	}
	if err := writeInstalledFiles(t, dir); err != nil {
		t.Fatalf("writeInstalledFiles() error = %v", err)
	}

	report, err := ComputeDrift(manifest, dir, "0.5.0")
	if err != nil {
		t.Fatalf("ComputeDrift() error = %v", err)
	}

	if report.HasDrift() {
		t.Errorf("expected no drift, got %d issue(s)", report.IssueCount())
		for _, entry := range report.Files {
			if entry.Status != FileOK {
				t.Logf("  %s: %s", entry.Status, entry.Path)
			}
		}
	}
}

func TestComputeDrift_NewFile(t *testing.T) {
	dir := t.TempDir()
	// Create a manifest with one file missing from the real bundle.
	manifest, err := ComputeBundledManifest("0.4.0", "agents")
	if err != nil {
		t.Fatalf("ComputeBundledManifest() error = %v", err)
	}
	if err := writeInstalledFiles(t, dir); err != nil {
		t.Fatalf("writeInstalledFiles() error = %v", err)
	}

	// Remove one file to simulate a bundle that has grown.
	var removed string
	for path := range manifest.Files {
		removed = path
		delete(manifest.Files, path)
		break
	}

	report, err := ComputeDrift(manifest, dir, "0.5.0")
	if err != nil {
		t.Fatalf("ComputeDrift() error = %v", err)
	}

	if !report.HasDrift() {
		t.Fatal("expected drift for new file, got none")
	}

	found := false
	for _, entry := range report.Files {
		if entry.Path == removed && entry.Status == FileNew {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected file %q to be reported as new", removed)
	}
}

func TestComputeDrift_ModifiedFile(t *testing.T) {
	dir := t.TempDir()
	manifest, err := ComputeBundledManifest("0.4.0", "agents")
	if err != nil {
		t.Fatalf("ComputeBundledManifest() error = %v", err)
	}
	if err := writeInstalledFiles(t, dir); err != nil {
		t.Fatalf("writeInstalledFiles() error = %v", err)
	}

	// Modify one installed file to simulate local drift.
	var modified string
	for path := range manifest.Files {
		modified = path
		break
	}
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(modified)), []byte("modified"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	report, err := ComputeDrift(manifest, dir, "0.5.0")
	if err != nil {
		t.Fatalf("ComputeDrift() error = %v", err)
	}

	if !report.HasDrift() {
		t.Fatal("expected drift for modified file, got none")
	}

	found := false
	for _, entry := range report.Files {
		if entry.Path == modified && entry.Status == FileModified {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected file %q to be reported as modified", modified)
	}
}

func TestComputeDrift_MissingFile(t *testing.T) {
	dir := t.TempDir()
	manifest, err := ComputeBundledManifest("0.4.0", "agents")
	if err != nil {
		t.Fatalf("ComputeBundledManifest() error = %v", err)
	}
	if err := writeInstalledFiles(t, dir); err != nil {
		t.Fatalf("writeInstalledFiles() error = %v", err)
	}

	var missing string
	for path := range manifest.Files {
		missing = path
		break
	}
	if err := os.Remove(filepath.Join(dir, filepath.FromSlash(missing))); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	report, err := ComputeDrift(manifest, dir, "0.5.0")
	if err != nil {
		t.Fatalf("ComputeDrift() error = %v", err)
	}

	if !report.HasDrift() {
		t.Fatal("expected drift for missing file, got none")
	}

	found := false
	for _, entry := range report.Files {
		if entry.Path == missing && entry.Status == FileMissing {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected %q to be reported as missing", missing)
	}
}

func TestComputeDrift_VersionMismatch(t *testing.T) {
	dir := t.TempDir()
	manifest, err := ComputeBundledManifest("0.4.0", "agents")
	if err != nil {
		t.Fatalf("ComputeBundledManifest() error = %v", err)
	}
	if err := writeInstalledFiles(t, dir); err != nil {
		t.Fatalf("writeInstalledFiles() error = %v", err)
	}

	report, err := ComputeDrift(manifest, dir, "0.5.0")
	if err != nil {
		t.Fatalf("ComputeDrift() error = %v", err)
	}
	if !report.VersionMismatch {
		t.Fatal("expected version mismatch")
	}
	if !report.HasDrift() {
		t.Fatal("expected drift when versions differ")
	}
}

func writeInstalledFiles(t *testing.T, dir string) error {
	t.Helper()
	skillFS, err := SiteSkill()
	if err != nil {
		return err
	}
	files, err := ListFiles()
	if err != nil {
		return err
	}
	for _, relPath := range files {
		data, err := fs.ReadFile(skillFS, relPath)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func TestDriftReport_IssueCount(t *testing.T) {
	report := &DriftReport{
		Files: []DriftEntry{
			{Path: "a.md", Status: FileOK},
			{Path: "b.md", Status: FileModified},
			{Path: "c.md", Status: FileNew},
			{Path: "d.md", Status: FileOK},
			{Path: "e.md", Status: FileMissing},
		},
	}

	if got := report.IssueCount(); got != 3 {
		t.Errorf("IssueCount() = %d, want 3", got)
	}
	if !report.HasDrift() {
		t.Error("HasDrift() = false, want true")
	}
}

func TestDriftReport_NoDrift(t *testing.T) {
	report := &DriftReport{
		Files: []DriftEntry{
			{Path: "a.md", Status: FileOK},
			{Path: "b.md", Status: FileOK},
		},
	}

	if report.HasDrift() {
		t.Error("HasDrift() = true, want false")
	}
	if got := report.IssueCount(); got != 0 {
		t.Errorf("IssueCount() = %d, want 0", got)
	}
}
