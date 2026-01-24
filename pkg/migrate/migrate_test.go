package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		changes  int
	}{
		{
			name:     "empty expression",
			input:    "",
			expected: "",
			changes:  0,
		},
		{
			name:     "no changes needed",
			input:    "published == True",
			expected: "published == True",
			changes:  0,
		},
		{
			name:     "quoted True to unquoted",
			input:    "published == 'True'",
			expected: "published == True",
			changes:  1,
		},
		{
			name:     "quoted False to unquoted",
			input:    "draft == 'False'",
			expected: "draft == False",
			changes:  1,
		},
		{
			name:     "double quoted True",
			input:    `published == "True"`,
			expected: "published == True",
			changes:  1,
		},
		{
			name:     "lowercase true to True",
			input:    "published == 'true'",
			expected: "published == True",
			changes:  1,
		},
		{
			name:     "in operator with two items",
			input:    "templateKey in ['blog-post', 'til']",
			expected: "templateKey == 'blog-post' or templateKey == 'til'",
			changes:  1,
		},
		{
			name:     "in operator with three items",
			input:    "status in ['draft', 'review', 'published']",
			expected: "status == 'draft' or status == 'review' or status == 'published'",
			changes:  1,
		},
		{
			name:     "missing space before operator",
			input:    "date<=today",
			expected: "date <= today",
			changes:  2, // space before and after
		},
		{
			name:     "missing space after operator",
			input:    "count >=10",
			expected: "count >= 10",
			changes:  1,
		},
		{
			name:     "is None to == None",
			input:    "image is None",
			expected: "image == None",
			changes:  1,
		},
		{
			name:     "is not None to != None",
			input:    "image is not None",
			expected: "image != None",
			changes:  1,
		},
		{
			name:     "complex expression",
			input:    "published == 'True' and templateKey in ['blog', 'til']",
			expected: "published == True and templateKey == 'blog' or templateKey == 'til'",
			changes:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, changes := Filter(tt.input)
			if result != tt.expected {
				t.Errorf("Filter(%q) = %q, want %q", tt.input, result, tt.expected)
			}
			if len(changes) != tt.changes {
				t.Errorf("Filter(%q) made %d changes, want %d", tt.input, len(changes), tt.changes)
			}
		})
	}
}

func TestValidateFilter(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{
			name:    "empty expression",
			expr:    "",
			wantErr: false,
		},
		{
			name:    "valid simple expression",
			expr:    "published == True",
			wantErr: false,
		},
		{
			name:    "valid complex expression",
			expr:    "published == True and date <= today",
			wantErr: false,
		},
		{
			name:    "invalid - unclosed string",
			expr:    "title == 'hello",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilter(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilter(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
		})
	}
}

func TestConfigFromMap(t *testing.T) {
	tests := []struct {
		name            string
		input           map[string]interface{}
		expectedChanges int
		checkOutput     func(t *testing.T, result *MigrationResult)
	}{
		{
			name:            "empty config",
			input:           map[string]interface{}{},
			expectedChanges: 0,
			checkOutput: func(t *testing.T, result *MigrationResult) {
				if len(result.Warnings) == 0 {
					t.Error("expected warning for missing markata section")
				}
			},
		},
		{
			name: "basic markata config",
			input: map[string]interface{}{
				"markata": map[string]interface{}{
					"output": "dist",
					"title":  "My Site",
				},
			},
			expectedChanges: 2, // namespace + rename
			checkOutput: func(t *testing.T, result *MigrationResult) {
				mg, ok := result.MigratedConfig["markata-go"].(map[string]interface{})
				if !ok {
					t.Fatal("markata-go section not found")
				}
				if mg["output_dir"] != "dist" {
					t.Errorf("output_dir = %v, want 'dist'", mg["output_dir"])
				}
			},
		},
		{
			name: "config with glob_patterns rename",
			input: map[string]interface{}{
				"markata": map[string]interface{}{
					"glob_patterns": []interface{}{"**/*.md"},
				},
			},
			expectedChanges: 2, // namespace + rename
			checkOutput: func(t *testing.T, result *MigrationResult) {
				mg, ok := result.MigratedConfig["markata-go"].(map[string]interface{})
				if !ok {
					t.Fatal("markata-go section not found")
				}
				patterns, ok := mg["patterns"].([]interface{})
				if !ok {
					t.Errorf("patterns not found or wrong type")
				}
				if len(patterns) != 1 || patterns[0] != "**/*.md" {
					t.Errorf("patterns = %v, want [**/*.md]", patterns)
				}
			},
		},
		{
			name: "config with nav map",
			input: map[string]interface{}{
				"markata": map[string]interface{}{
					"nav": map[string]interface{}{
						"home":  "/",
						"about": "/about",
					},
				},
			},
			expectedChanges: 2, // namespace + nav transform
			checkOutput: func(t *testing.T, result *MigrationResult) {
				mg, ok := result.MigratedConfig["markata-go"].(map[string]interface{})
				if !ok {
					t.Fatal("markata-go section not found")
				}
				nav, ok := mg["nav"].([]map[string]interface{})
				if !ok {
					t.Errorf("nav not found or wrong type")
				}
				if len(nav) != 2 {
					t.Errorf("nav has %d items, want 2", len(nav))
				}
			},
		},
		{
			name: "config with feed filter",
			input: map[string]interface{}{
				"markata": map[string]interface{}{
					"feeds": []interface{}{
						map[string]interface{}{
							"slug":   "blog",
							"filter": "published == 'True'",
						},
					},
				},
			},
			expectedChanges: 1, // namespace
			checkOutput: func(t *testing.T, result *MigrationResult) {
				if len(result.FilterMigrations) != 1 {
					t.Errorf("expected 1 filter migration, got %d", len(result.FilterMigrations))
					return
				}
				fm := result.FilterMigrations[0]
				if fm.Original != "published == 'True'" {
					t.Errorf("original filter = %q, want \"published == 'True'\"", fm.Original)
				}
				if fm.Migrated != "published == True" {
					t.Errorf("migrated filter = %q, want \"published == True\"", fm.Migrated)
				}
			},
		},
		{
			name: "pyproject.toml style",
			input: map[string]interface{}{
				"tool": map[string]interface{}{
					"markata": map[string]interface{}{
						"output": "public",
					},
				},
			},
			expectedChanges: 2, // namespace + rename
			checkOutput: func(t *testing.T, result *MigrationResult) {
				if _, ok := result.MigratedConfig["markata-go"]; !ok {
					t.Error("markata-go section not found")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConfigFromMap(tt.input)
			if err != nil {
				t.Fatalf("ConfigFromMap() error = %v", err)
			}
			if len(result.Changes) < tt.expectedChanges {
				t.Errorf("got %d changes, want at least %d", len(result.Changes), tt.expectedChanges)
			}
			if tt.checkOutput != nil {
				tt.checkOutput(t, result)
			}
		})
	}
}

func TestMigrationResult_Report(t *testing.T) {
	result := &MigrationResult{
		InputFile: "markata.toml",
		Changes: []ConfigChange{
			{Type: "namespace", Path: "markata", Description: "Namespace change"},
		},
		FilterMigrations: []FilterMigration{
			{Feed: "blog", Original: "published == 'True'", Migrated: "published == True", Changes: []string{"Boolean literal"}, Valid: true},
		},
		Warnings: []Warning{
			{Category: "plugin", Message: "Hook not supported"},
		},
	}

	report := result.Report()

	// Check report contains expected sections
	expectedSections := []string{
		"Migration Report",
		"SUMMARY",
		"CONFIGURATION CHANGES",
		"FILTER MIGRATIONS",
		"WARNINGS",
		"NEXT STEPS",
	}

	for _, section := range expectedSections {
		if !containsString(report, section) {
			t.Errorf("report missing section: %s", section)
		}
	}
}

func TestMigrationResult_ExitCode(t *testing.T) {
	tests := []struct {
		name     string
		result   MigrationResult
		expected int
	}{
		{
			name:     "no issues",
			result:   MigrationResult{},
			expected: 0,
		},
		{
			name: "warnings only",
			result: MigrationResult{
				Warnings: []Warning{{Message: "test"}},
			},
			expected: 1,
		},
		{
			name: "non-fatal errors",
			result: MigrationResult{
				Errors: []MigrationError{{Message: "test", Fatal: false}},
			},
			expected: 2,
		},
		{
			name: "fatal error",
			result: MigrationResult{
				Errors: []MigrationError{{Message: "test", Fatal: true}},
			},
			expected: 3,
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

func TestCheckTemplates(t *testing.T) {
	// Create temp directory with test templates
	tempDir := t.TempDir()

	// Create a template with issues
	templateContent := `{% extends "base.html" %}
{% macro render_item(item) %}
  <li>{{ item }}</li>
{% endmacro %}
{% do items.append('new') %}
{{ post.markata.config.title }}
{{ post.article_html }}
{{ [x for x in items] }}`

	templatePath := filepath.Join(tempDir, "test.html")
	if err := os.WriteFile(templatePath, []byte(templateContent), 0o600); err != nil {
		t.Fatalf("failed to create test template: %v", err)
	}

	issues, err := CheckTemplates(tempDir)
	if err != nil {
		t.Fatalf("CheckTemplates() error = %v", err)
	}

	// Should find multiple issues
	expectedIssueTypes := []string{
		"macro",
		"do",
		"post.markata",
		"article_html",
	}

	for _, issueType := range expectedIssueTypes {
		found := false
		for _, issue := range issues {
			if containsString(issue.Issue, issueType) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find issue containing %q", issueType)
		}
	}
}

func TestAnalyzeFilter(t *testing.T) {
	tests := []struct {
		name         string
		expr         string
		wantWarnings int
	}{
		{
			name:         "valid expression",
			expr:         "published == True",
			wantWarnings: 0,
		},
		{
			name:         "lambda expression",
			expr:         "lambda x: x.published",
			wantWarnings: 1,
		},
		{
			name:         "string method",
			expr:         "title.lower() == 'test'",
			wantWarnings: 1,
		},
		{
			name:         "len function",
			expr:         "len(tags) > 0",
			wantWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := AnalyzeFilter(tt.expr)
			if len(warnings) != tt.wantWarnings {
				t.Errorf("AnalyzeFilter(%q) returned %d warnings, want %d", tt.expr, len(warnings), tt.wantWarnings)
			}
		})
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && (s[0:len(substr)] == substr || containsString(s[1:], substr)))
}
