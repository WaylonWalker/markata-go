package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/example/markata-go/pkg/lifecycle"
)

// TestRedirectsPlugin_Name tests the plugin name.
func TestRedirectsPlugin_Name(t *testing.T) {
	p := NewRedirectsPlugin()
	if got := p.Name(); got != "redirects" {
		t.Errorf("Name() = %q, want %q", got, "redirects")
	}
}

// TestRedirectsPlugin_ParseRedirects tests redirect file parsing.
func TestRedirectsPlugin_ParseRedirects(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []Redirect
	}{
		{
			name: "basic redirects",
			content: `/old-path    /new-path
/legacy-url  /current-url`,
			want: []Redirect{
				{Original: "/old-path", New: "/new-path"},
				{Original: "/legacy-url", New: "/current-url"},
			},
		},
		{
			name: "with comments",
			content: `# This is a comment
/old    /new
# Another comment
/foo    /bar`,
			want: []Redirect{
				{Original: "/old", New: "/new"},
				{Original: "/foo", New: "/bar"},
			},
		},
		{
			name: "skip wildcards",
			content: `/old/*    /new/*
/valid    /path
/another/*  /dest/*`,
			want: []Redirect{
				{Original: "/valid", New: "/path"},
			},
		},
		{
			name: "skip malformed lines",
			content: `/only-source
/valid    /destination
incomplete
/another  /target`,
			want: []Redirect{
				{Original: "/valid", New: "/destination"},
				{Original: "/another", New: "/target"},
			},
		},
		{
			name: "skip non-absolute paths",
			content: `old-path    /new-path
/valid    /destination
/source   relative-dest`,
			want: []Redirect{
				{Original: "/valid", New: "/destination"},
			},
		},
		{
			name:    "empty content",
			content: "",
			want:    nil,
		},
		{
			name: "only comments and whitespace",
			content: `# Comment 1
  
# Comment 2
   `,
			want: nil,
		},
		{
			name: "extra whitespace handling",
			content: `   /old    /new   
	/tabbed	/path	`,
			want: []Redirect{
				{Original: "/old", New: "/new"},
				{Original: "/tabbed", New: "/path"},
			},
		},
		{
			name: "multiple spaces between paths",
			content: `/source      /destination
/a          /b`,
			want: []Redirect{
				{Original: "/source", New: "/destination"},
				{Original: "/a", New: "/b"},
			},
		},
		{
			name: "extra fields ignored",
			content: `/old    /new    301
/foo    /bar    302    extra`,
			want: []Redirect{
				{Original: "/old", New: "/new"},
				{Original: "/foo", New: "/bar"},
			},
		},
	}

	p := NewRedirectsPlugin()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.parseRedirects(tt.content)

			if len(got) != len(tt.want) {
				t.Errorf("parseRedirects() returned %d redirects, want %d", len(got), len(tt.want))
				t.Errorf("got: %+v", got)
				t.Errorf("want: %+v", tt.want)
				return
			}

			for i, r := range got {
				if r.Original != tt.want[i].Original {
					t.Errorf("redirect[%d].Original = %q, want %q", i, r.Original, tt.want[i].Original)
				}
				if r.New != tt.want[i].New {
					t.Errorf("redirect[%d].New = %q, want %q", i, r.New, tt.want[i].New)
				}
			}
		})
	}
}

// TestRedirectsPlugin_Write tests the full write workflow.
func TestRedirectsPlugin_Write(t *testing.T) {
	// Create temporary directories
	tmpDir := t.TempDir()
	staticDir := filepath.Join(tmpDir, "static")
	outputDir := filepath.Join(tmpDir, "output")
	os.MkdirAll(staticDir, 0755)
	os.MkdirAll(outputDir, 0755)

	// Create redirects file
	redirectsContent := `# Blog migration
/old-post    /new-post
/legacy      /current
/blog/2023/article    /posts/article
`
	redirectsFile := filepath.Join(staticDir, "_redirects")
	if err := os.WriteFile(redirectsFile, []byte(redirectsContent), 0644); err != nil {
		t.Fatalf("failed to create redirects file: %v", err)
	}

	// Setup manager
	m := lifecycle.NewManager()
	cfg := m.Config()
	cfg.OutputDir = outputDir

	// Create and configure plugin
	p := NewRedirectsPlugin()
	p.SetConfig(RedirectsConfig{
		RedirectsFile: redirectsFile,
	})

	// Run write
	if err := p.Write(m); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify output files
	expectedFiles := []struct {
		path     string
		contains []string
	}{
		{
			path: filepath.Join(outputDir, "old-post", "index.html"),
			contains: []string{
				`url='/new-post'`,
				`href="/new-post"`,
				`/old-post`,
				"Page Moved",
			},
		},
		{
			path: filepath.Join(outputDir, "legacy", "index.html"),
			contains: []string{
				`url='/current'`,
				`href="/current"`,
			},
		},
		{
			path: filepath.Join(outputDir, "blog", "2023", "article", "index.html"),
			contains: []string{
				`url='/posts/article'`,
				`href="/posts/article"`,
			},
		},
	}

	for _, ef := range expectedFiles {
		content, err := os.ReadFile(ef.path)
		if err != nil {
			t.Errorf("expected file %s not found: %v", ef.path, err)
			continue
		}

		contentStr := string(content)
		for _, substr := range ef.contains {
			if !strings.Contains(contentStr, substr) {
				t.Errorf("file %s missing expected content %q", ef.path, substr)
			}
		}

		// Verify it's valid HTML
		if !strings.Contains(contentStr, "<!DOCTYPE html>") {
			t.Errorf("file %s missing DOCTYPE", ef.path)
		}
	}
}

// TestRedirectsPlugin_Write_MissingFile tests handling of missing redirects file.
func TestRedirectsPlugin_Write_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	os.MkdirAll(outputDir, 0755)

	m := lifecycle.NewManager()
	cfg := m.Config()
	cfg.OutputDir = outputDir

	p := NewRedirectsPlugin()
	p.SetConfig(RedirectsConfig{
		RedirectsFile: filepath.Join(tmpDir, "nonexistent", "_redirects"),
	})

	// Should not error on missing file
	if err := p.Write(m); err != nil {
		t.Errorf("Write() should not error on missing file, got: %v", err)
	}
}

// TestRedirectsPlugin_Write_EmptyFile tests handling of empty redirects file.
func TestRedirectsPlugin_Write_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	os.MkdirAll(outputDir, 0755)

	// Create empty redirects file
	redirectsFile := filepath.Join(tmpDir, "_redirects")
	if err := os.WriteFile(redirectsFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create redirects file: %v", err)
	}

	m := lifecycle.NewManager()
	cfg := m.Config()
	cfg.OutputDir = outputDir

	p := NewRedirectsPlugin()
	p.SetConfig(RedirectsConfig{
		RedirectsFile: redirectsFile,
	})

	if err := p.Write(m); err != nil {
		t.Errorf("Write() error on empty file = %v", err)
	}

	// Output dir should have no redirect files
	entries, _ := os.ReadDir(outputDir)
	if len(entries) > 0 {
		t.Errorf("expected no files in output dir for empty redirects, got %d entries", len(entries))
	}
}

// TestRedirectsPlugin_Write_CustomTemplate tests custom template support.
func TestRedirectsPlugin_Write_CustomTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	os.MkdirAll(outputDir, 0755)

	// Create redirects file
	redirectsFile := filepath.Join(tmpDir, "_redirects")
	if err := os.WriteFile(redirectsFile, []byte("/old /new"), 0644); err != nil {
		t.Fatalf("failed to create redirects file: %v", err)
	}

	// Create custom template
	customTemplate := `<!DOCTYPE html>
<html>
<head><meta http-equiv="Refresh" content="0; url='{{ .New }}'" /></head>
<body>CUSTOM: {{ .Original }} -> {{ .New }}</body>
</html>`
	templateFile := filepath.Join(tmpDir, "redirect.html")
	if err := os.WriteFile(templateFile, []byte(customTemplate), 0644); err != nil {
		t.Fatalf("failed to create template file: %v", err)
	}

	m := lifecycle.NewManager()
	cfg := m.Config()
	cfg.OutputDir = outputDir

	p := NewRedirectsPlugin()
	p.SetConfig(RedirectsConfig{
		RedirectsFile:    redirectsFile,
		RedirectTemplate: templateFile,
	})

	if err := p.Write(m); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify custom template was used
	content, err := os.ReadFile(filepath.Join(outputDir, "old", "index.html"))
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "CUSTOM: /old -> /new") {
		t.Errorf("custom template not used, got: %s", string(content))
	}
}

// TestRedirectsPlugin_Configure tests configuration loading.
func TestRedirectsPlugin_Configure(t *testing.T) {
	tests := []struct {
		name     string
		extra    map[string]interface{}
		wantFile string
		wantTmpl string
	}{
		{
			name:     "default config",
			extra:    nil,
			wantFile: "static/_redirects",
			wantTmpl: "",
		},
		{
			name: "top-level redirects config",
			extra: map[string]interface{}{
				"redirects": map[string]interface{}{
					"redirects_file":    "custom/_redirects",
					"redirect_template": "custom/redirect.html",
				},
			},
			wantFile: "custom/_redirects",
			wantTmpl: "custom/redirect.html",
		},
		{
			name: "nested markata-go config",
			extra: map[string]interface{}{
				"markata-go": map[string]interface{}{
					"redirects": map[string]interface{}{
						"redirects_file": "nested/_redirects",
					},
				},
			},
			wantFile: "nested/_redirects",
			wantTmpl: "",
		},
		{
			name: "markata-go takes precedence",
			extra: map[string]interface{}{
				"redirects": map[string]interface{}{
					"redirects_file": "top-level/_redirects",
				},
				"markata-go": map[string]interface{}{
					"redirects": map[string]interface{}{
						"redirects_file": "nested/_redirects",
					},
				},
			},
			wantFile: "nested/_redirects",
			wantTmpl: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := lifecycle.NewManager()
			cfg := m.Config()
			cfg.Extra = tt.extra

			p := NewRedirectsPlugin()
			if err := p.Configure(m); err != nil {
				t.Fatalf("Configure() error = %v", err)
			}

			got := p.GetConfig()
			if got.RedirectsFile != tt.wantFile {
				t.Errorf("RedirectsFile = %q, want %q", got.RedirectsFile, tt.wantFile)
			}
			if got.RedirectTemplate != tt.wantTmpl {
				t.Errorf("RedirectTemplate = %q, want %q", got.RedirectTemplate, tt.wantTmpl)
			}
		})
	}
}

// TestRedirectsPlugin_Priority tests the plugin priority.
func TestRedirectsPlugin_Priority(t *testing.T) {
	p := NewRedirectsPlugin()

	if got := p.Priority(lifecycle.StageWrite); got != lifecycle.PriorityLate {
		t.Errorf("Priority(StageWrite) = %d, want %d (PriorityLate)", got, lifecycle.PriorityLate)
	}

	if got := p.Priority(lifecycle.StageRender); got != lifecycle.PriorityDefault {
		t.Errorf("Priority(StageRender) = %d, want %d (PriorityDefault)", got, lifecycle.PriorityDefault)
	}
}

// TestRedirectsPlugin_Write_Caching tests that caching prevents regeneration.
func TestRedirectsPlugin_Write_Caching(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	os.MkdirAll(outputDir, 0755)

	// Create redirects file
	redirectsFile := filepath.Join(tmpDir, "_redirects")
	if err := os.WriteFile(redirectsFile, []byte("/old /new"), 0644); err != nil {
		t.Fatalf("failed to create redirects file: %v", err)
	}

	m := lifecycle.NewManager()
	cfg := m.Config()
	cfg.OutputDir = outputDir

	p := NewRedirectsPlugin()
	p.SetConfig(RedirectsConfig{
		RedirectsFile: redirectsFile,
	})

	// First write
	if err := p.Write(m); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify file was created
	outputPath := filepath.Join(outputDir, "old", "index.html")
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected file %s not found after first write", outputPath)
	}

	// Delete the output file
	os.Remove(outputPath)

	// Second write (should be cached)
	if err := p.Write(m); err != nil {
		t.Fatalf("Write() error on second call = %v", err)
	}

	// File should NOT be recreated due to caching
	if _, err := os.Stat(outputPath); err == nil {
		t.Errorf("file %s should not exist due to caching", outputPath)
	}
}

// TestRedirectsPlugin_Write_NestedPaths tests handling of deeply nested paths.
func TestRedirectsPlugin_Write_NestedPaths(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	os.MkdirAll(outputDir, 0755)

	// Create redirects file with nested paths
	redirectsContent := `/a/b/c/d/e    /target
/deep/nested/path/here    /simple`
	redirectsFile := filepath.Join(tmpDir, "_redirects")
	if err := os.WriteFile(redirectsFile, []byte(redirectsContent), 0644); err != nil {
		t.Fatalf("failed to create redirects file: %v", err)
	}

	m := lifecycle.NewManager()
	cfg := m.Config()
	cfg.OutputDir = outputDir

	p := NewRedirectsPlugin()
	p.SetConfig(RedirectsConfig{
		RedirectsFile: redirectsFile,
	})

	if err := p.Write(m); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify nested directories were created
	expectedPaths := []string{
		filepath.Join(outputDir, "a", "b", "c", "d", "e", "index.html"),
		filepath.Join(outputDir, "deep", "nested", "path", "here", "index.html"),
	}

	for _, path := range expectedPaths {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s not found", path)
		}
	}
}

// TestRedirect_Struct tests the Redirect struct.
func TestRedirect_Struct(t *testing.T) {
	r := Redirect{
		Original: "/old-path",
		New:      "/new-path",
	}

	if r.Original != "/old-path" {
		t.Errorf("Original = %q, want %q", r.Original, "/old-path")
	}
	if r.New != "/new-path" {
		t.Errorf("New = %q, want %q", r.New, "/new-path")
	}
}

// TestHashContent tests the hash function.
func TestHashContent(t *testing.T) {
	// Same content should produce same hash
	h1 := hashContent([]byte("test content"))
	h2 := hashContent([]byte("test content"))
	if h1 != h2 {
		t.Errorf("same content produced different hashes: %d != %d", h1, h2)
	}

	// Different content should produce different hash
	h3 := hashContent([]byte("different content"))
	if h1 == h3 {
		t.Errorf("different content produced same hash: %d", h1)
	}

	// Empty content should work
	h4 := hashContent([]byte(""))
	if h4 == 0 {
		t.Errorf("empty content produced zero hash")
	}
}
