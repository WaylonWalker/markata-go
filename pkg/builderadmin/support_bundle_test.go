package builderadmin

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestWriteSupportBundle(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := writeSupportBundle(&output, SupportBundleInput{
		BuildID:            "build-123",
		DiagnosticMarkdown: []byte("# Diagnostic\n\nBuild was slow.\n"),
		BenchmarkJSON:      []byte(`{"benchmark":{"Total":810400000000}}`),
		TraceJSON:          []byte(`{"spans":[]}`),
	})
	if err != nil {
		t.Fatalf("writeSupportBundle: %v", err)
	}

	archive, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatalf("open support bundle: %v", err)
	}
	files := make(map[string][]byte, len(archive.File))
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		files[file.Name] = data
	}

	for _, name := range []string{"manifest.json", "diagnostic.md", "benchmark.json", "trace.json"} {
		if _, ok := files[name]; !ok {
			t.Errorf("bundle missing %s", name)
		}
	}
	if _, ok := files["build.log"]; ok {
		t.Fatal("bundle unexpectedly contains raw build log")
	}

	var manifest SupportBundleManifest
	if err := json.Unmarshal(files["manifest.json"], &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if manifest.SchemaVersion != supportBundleSchemaVersion || manifest.BuildID != "build-123" {
		t.Fatalf("manifest = %#v", manifest)
	}
	if len(manifest.Files) != 3 {
		t.Fatalf("manifest file count = %d, want 3", len(manifest.Files))
	}
}

func TestWriteSupportBundleRejectsOversizedEntry(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := writeSupportBundle(&output, SupportBundleInput{
		BuildID:            "build-123",
		DiagnosticMarkdown: bytes.Repeat([]byte("x"), maxSupportBundleEntryBytes+1),
	})
	if err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("error = %v, want size-limit error", err)
	}
}

func TestWriteSupportBundleRejectsUnsafeBuildID(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	if err := writeSupportBundle(&output, SupportBundleInput{BuildID: "../secret"}); err == nil {
		t.Fatal("unsafe build id unexpectedly accepted")
	}
}
