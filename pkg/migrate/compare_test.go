package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompare(t *testing.T) {
	// Create temp directories for testing
	oldDir := t.TempDir()
	newDir := t.TempDir()

	// Create some test files in old dir
	createTestFile(t, oldDir, "index.html", "<html>index</html>")
	createTestFile(t, oldDir, "about.html", "<html>about</html>")
	createTestFile(t, oldDir, "blog/post1.html", "<html>post1</html>")
	createTestFile(t, oldDir, "blog/post2.html", "<html>post2</html>")
	createTestFile(t, oldDir, "feed.xml", "<rss>feed</rss>")

	// Create some test files in new dir (with differences)
	createTestFile(t, newDir, "index.html", "<html>index</html>")
	createTestFile(t, newDir, "about.html", "<html>about</html>")
	createTestFile(t, newDir, "blog/post1.html", "<html>post1</html>")
	// post2.html and feed.xml are intentionally not created in newDir
	createTestFile(t, newDir, "sitemap.xml", "<sitemap>sitemap</sitemap>")
	createTestFile(t, newDir, "robots.txt", "User-agent: *")

	// Compare with default options (HTML only)
	opts := CompareOptions{
		Extensions: []string{".html"},
	}

	result, err := Compare(oldDir, newDir, opts)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	// Check old file count (HTML only)
	if len(result.OldFiles) != 4 {
		t.Errorf("OldFiles count = %d, want 4", len(result.OldFiles))
	}

	// Check new file count (HTML only)
	if len(result.NewFiles) != 3 {
		t.Errorf("NewFiles count = %d, want 3", len(result.NewFiles))
	}

	// Check missing in new
	if len(result.MissingInNew) != 1 {
		t.Errorf("MissingInNew count = %d, want 1", len(result.MissingInNew))
	} else if result.MissingInNew[0] != "/blog/post2.html" {
		t.Errorf("MissingInNew[0] = %s, want /blog/post2.html", result.MissingInNew[0])
	}

	// Check new only (none for HTML since sitemap.xml and robots.txt are not HTML)
	if len(result.NewOnly) != 0 {
		t.Errorf("NewOnly count = %d, want 0 (HTML only)", len(result.NewOnly))
	}

	// Check common files
	if len(result.Common) != 3 {
		t.Errorf("Common count = %d, want 3", len(result.Common))
	}
}

func TestCompare_AllFiles(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	createTestFile(t, oldDir, "index.html", "<html>index</html>")
	createTestFile(t, oldDir, "feed.xml", "<rss>feed</rss>")

	createTestFile(t, newDir, "index.html", "<html>index</html>")
	createTestFile(t, newDir, "sitemap.xml", "<sitemap>sitemap</sitemap>")

	// Compare all files (empty extensions)
	opts := CompareOptions{
		Extensions: []string{},
	}

	result, err := Compare(oldDir, newDir, opts)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	// Should include all files
	if len(result.OldFiles) != 2 {
		t.Errorf("OldFiles count = %d, want 2", len(result.OldFiles))
	}

	if len(result.NewFiles) != 2 {
		t.Errorf("NewFiles count = %d, want 2", len(result.NewFiles))
	}

	// feed.xml is missing in new
	if len(result.MissingInNew) != 1 {
		t.Errorf("MissingInNew count = %d, want 1", len(result.MissingInNew))
	}

	// sitemap.xml is new only
	if len(result.NewOnly) != 1 {
		t.Errorf("NewOnly count = %d, want 1", len(result.NewOnly))
	}
}

func TestCompare_WithIgnorePatterns(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	createTestFile(t, oldDir, "index.html", "<html>index</html>")
	createTestFile(t, oldDir, "assets/style.css", "body { }")
	createTestFile(t, oldDir, "assets/script.js", "console.log('test')")

	createTestFile(t, newDir, "index.html", "<html>index</html>")
	// Assets are different in new dir
	createTestFile(t, newDir, "assets/app.css", "body { }")

	// Compare with ignore pattern for assets
	opts := CompareOptions{
		Extensions:     []string{},
		IgnorePatterns: []string{"assets/**"},
	}

	result, err := Compare(oldDir, newDir, opts)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	// Should only have HTML files (assets ignored)
	if len(result.OldFiles) != 1 {
		t.Errorf("OldFiles count = %d, want 1 (assets ignored)", len(result.OldFiles))
	}

	if len(result.NewFiles) != 1 {
		t.Errorf("NewFiles count = %d, want 1 (assets ignored)", len(result.NewFiles))
	}

	// No differences when assets are ignored
	if result.HasDifferences() {
		t.Error("HasDifferences() = true, want false (assets ignored)")
	}
}

func TestCompare_InvalidDirectories(t *testing.T) {
	validDir := t.TempDir()

	// Test with non-existent old directory
	_, err := Compare("/nonexistent/path", validDir, CompareOptions{})
	if err == nil {
		t.Error("expected error for non-existent old directory")
	}

	// Test with non-existent new directory
	_, err = Compare(validDir, "/nonexistent/path", CompareOptions{})
	if err == nil {
		t.Error("expected error for non-existent new directory")
	}

	// Test with file instead of directory
	tempFile := filepath.Join(validDir, "file.txt")
	if err := os.WriteFile(tempFile, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = Compare(tempFile, validDir, CompareOptions{})
	if err == nil {
		t.Error("expected error for file instead of directory")
	}
}

func TestCompare_EmptyDirectories(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	result, err := Compare(oldDir, newDir, CompareOptions{})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if len(result.OldFiles) != 0 {
		t.Errorf("OldFiles count = %d, want 0", len(result.OldFiles))
	}

	if len(result.NewFiles) != 0 {
		t.Errorf("NewFiles count = %d, want 0", len(result.NewFiles))
	}

	if result.HasDifferences() {
		t.Error("HasDifferences() = true, want false for empty directories")
	}
}

func TestCompareResult_Report(t *testing.T) {
	result := &CompareResult{
		OldDir:       "markout",
		NewDir:       "public",
		OldFiles:     []string{"/index.html", "/about.html", "/old-post.html"},
		NewFiles:     []string{"/index.html", "/about.html", "/sitemap.xml"},
		MissingInNew: []string{"/old-post.html"},
		NewOnly:      []string{"/sitemap.xml"},
		Common:       []string{"/index.html", "/about.html"},
	}

	report := result.Report()

	// Check report contains expected content
	expectedContent := []string{
		"Migration Comparison Report",
		"markout/",
		"public/",
		"3 files",
		"Missing in new site",
		"/old-post.html",
		"New files not in old",
		"/sitemap.xml",
		"Summary",
	}

	for _, expected := range expectedContent {
		if !containsString(report, expected) {
			t.Errorf("report missing expected content: %q", expected)
		}
	}
}

func TestCompareResult_Report_NoChanges(t *testing.T) {
	result := &CompareResult{
		OldDir:   "old",
		NewDir:   "new",
		OldFiles: []string{"/index.html"},
		NewFiles: []string{"/index.html"},
		Common:   []string{"/index.html"},
	}

	report := result.Report()

	if !containsString(report, "No differences found") {
		t.Error("report should indicate no differences")
	}
}

func TestCompareResult_ExitCode(t *testing.T) {
	tests := []struct {
		name     string
		result   CompareResult
		expected int
	}{
		{
			name: "no differences",
			result: CompareResult{
				OldFiles: []string{"/index.html"},
				NewFiles: []string{"/index.html"},
				Common:   []string{"/index.html"},
			},
			expected: 0,
		},
		{
			name: "missing files",
			result: CompareResult{
				MissingInNew: []string{"/old.html"},
			},
			expected: 1,
		},
		{
			name: "new files only",
			result: CompareResult{
				NewOnly: []string{"/new.html"},
			},
			expected: 1,
		},
		{
			name: "both missing and new",
			result: CompareResult{
				MissingInNew: []string{"/old.html"},
				NewOnly:      []string{"/new.html"},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.ExitCode(); got != tt.expected {
				t.Errorf("ExitCode() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestCompareResult_JSONReport(t *testing.T) {
	result := &CompareResult{
		OldDir:       "old",
		NewDir:       "new",
		OldFiles:     []string{"/index.html", "/about.html"},
		NewFiles:     []string{"/index.html"},
		MissingInNew: []string{"/about.html"},
		NewOnly:      []string{},
		Common:       []string{"/index.html"},
	}

	report := result.JSONReport()

	// Check required fields
	if report["old_dir"] != "old" {
		t.Errorf("old_dir = %v, want 'old'", report["old_dir"])
	}
	if report["new_dir"] != "new" {
		t.Errorf("new_dir = %v, want 'new'", report["new_dir"])
	}
	if report["old_file_count"] != 2 {
		t.Errorf("old_file_count = %v, want 2", report["old_file_count"])
	}
	if report["new_file_count"] != 1 {
		t.Errorf("new_file_count = %v, want 1", report["new_file_count"])
	}
	if report["missing_count"] != 1 {
		t.Errorf("missing_count = %v, want 1", report["missing_count"])
	}
	if report["has_differences"] != true {
		t.Errorf("has_differences = %v, want true", report["has_differences"])
	}
	if report["exit_code"] != 1 {
		t.Errorf("exit_code = %v, want 1", report["exit_code"])
	}
}

func TestDefaultCompareOptions(t *testing.T) {
	opts := DefaultCompareOptions()

	if len(opts.Extensions) == 0 {
		t.Error("default extensions should not be empty")
	}

	// Should default to HTML
	foundHTML := false
	for _, ext := range opts.Extensions {
		if ext == ".html" {
			foundHTML = true
			break
		}
	}
	if !foundHTML {
		t.Error("default extensions should include .html")
	}

	// Should have some default ignore patterns
	if len(opts.IgnorePatterns) == 0 {
		t.Error("default ignore patterns should not be empty")
	}
}

func TestCompare_MultipleExtensions(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	createTestFile(t, oldDir, "index.html", "<html>index</html>")
	createTestFile(t, oldDir, "feed.xml", "<rss>feed</rss>")
	createTestFile(t, oldDir, "style.css", "body { }")

	createTestFile(t, newDir, "index.html", "<html>index</html>")
	createTestFile(t, newDir, "feed.xml", "<rss>feed</rss>")
	createTestFile(t, newDir, "style.css", "body { }")

	// Compare HTML and XML files
	opts := CompareOptions{
		Extensions: []string{".html", ".xml"},
	}

	result, err := Compare(oldDir, newDir, opts)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	// Should include HTML and XML, but not CSS
	if len(result.OldFiles) != 2 {
		t.Errorf("OldFiles count = %d, want 2 (HTML and XML only)", len(result.OldFiles))
	}

	if result.HasDifferences() {
		t.Error("HasDifferences() = true, want false")
	}
}

// Helper function to create test files
func createTestFile(t *testing.T, baseDir, path, content string) {
	t.Helper()
	fullPath := filepath.Join(baseDir, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to create file %s: %v", fullPath, err)
	}
}
