package builderadmin

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

const (
	supportBundleSchemaVersion = 1
	maxSupportBundleEntryBytes = 2 << 20
)

// SupportBundleInput contains already-sanitized diagnostic artifacts for one
// build. Raw logs are deliberately not accepted here; they need a dedicated
// sanitizer before they can safely enter a portable support bundle.
type SupportBundleInput struct {
	BuildID            string
	DiagnosticMarkdown []byte
	BenchmarkJSON      []byte
	TraceJSON          []byte
}

// SupportBundleManifest describes the portable contents of a support bundle.
type SupportBundleManifest struct {
	SchemaVersion int                 `json:"schema_version"`
	BuildID       string              `json:"build_id"`
	Files         []SupportBundleFile `json:"files"`
}

// SupportBundleFile describes one bounded file in a support bundle.
type SupportBundleFile struct {
	Name string `json:"name"`
	Size int    `json:"size_bytes"`
}

type supportBundleEntry struct {
	name string
	data []byte
}

func writeSupportBundle(dst io.Writer, input SupportBundleInput) error {
	if dst == nil {
		return fmt.Errorf("support bundle destination is required")
	}
	buildID := strings.TrimSpace(input.BuildID)
	if buildID == "" || filepath.Base(buildID) != buildID || strings.ContainsAny(buildID, `/\\`) {
		return fmt.Errorf("invalid build id %q", input.BuildID)
	}

	entries := []supportBundleEntry{
		{name: "diagnostic.md", data: input.DiagnosticMarkdown},
		{name: "benchmark.json", data: input.BenchmarkJSON},
		{name: "trace.json", data: input.TraceJSON},
	}
	manifest := SupportBundleManifest{SchemaVersion: supportBundleSchemaVersion, BuildID: buildID}
	filtered := entries[:0]
	for _, entry := range entries {
		if len(entry.data) == 0 {
			continue
		}
		if len(entry.data) > maxSupportBundleEntryBytes {
			return fmt.Errorf("support bundle entry %q is %d bytes; limit is %d", entry.name, len(entry.data), maxSupportBundleEntryBytes)
		}
		filtered = append(filtered, entry)
		manifest.Files = append(manifest.Files, SupportBundleFile{Name: entry.name, Size: len(entry.data)})
	}

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode support bundle manifest: %w", err)
	}

	archive := zip.NewWriter(dst)
	if err := writeSupportBundleEntry(archive, "manifest.json", append(manifestJSON, '\n')); err != nil {
		_ = archive.Close()
		return err
	}
	for _, entry := range filtered {
		if err := writeSupportBundleEntry(archive, entry.name, entry.data); err != nil {
			_ = archive.Close()
			return err
		}
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close support bundle: %w", err)
	}
	return nil
}

func writeSupportBundleEntry(archive *zip.Writer, name string, data []byte) error {
	writer, err := archive.Create(name)
	if err != nil {
		return fmt.Errorf("create support bundle entry %q: %w", name, err)
	}
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("write support bundle entry %q: %w", name, err)
	}
	return nil
}
