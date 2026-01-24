package migrate

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CheckTemplates scans a templates directory for compatibility issues.
func CheckTemplates(templatesDir string) ([]TemplateIssue, error) {
	var issues []TemplateIssue

	err := filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only check HTML and template files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".html" && ext != ".jinja" && ext != ".jinja2" && ext != ".j2" {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		fileIssues, err := checkTemplateFile(path)
		if err != nil {
			issues = append(issues, TemplateIssue{
				File:     path,
				Line:     0,
				Issue:    err.Error(),
				Severity: "error",
			})
			return nil
		}

		issues = append(issues, fileIssues...)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return issues, nil
}

// checkTemplateFile checks a single template file for compatibility issues.
func checkTemplateFile(path string) ([]TemplateIssue, error) {
	var issues []TemplateIssue

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check for various template patterns
		lineIssues := checkTemplateLine(path, lineNum, line)
		issues = append(issues, lineIssues...)
	}

	if err := scanner.Err(); err != nil {
		return issues, err
	}

	return issues, nil
}

// checkTemplateLine checks a single line for template compatibility issues.
func checkTemplateLine(file string, lineNum int, line string) []TemplateIssue {
	var issues []TemplateIssue

	// Check for unsupported Jinja2 features

	// 1. do statement
	if doPattern.MatchString(line) {
		issues = append(issues, TemplateIssue{
			File:       file,
			Line:       lineNum,
			Issue:      "{% do %} statement is not supported in pongo2",
			Severity:   "error",
			Suggestion: "Use {% set %} or restructure the template logic",
		})
	}

	// 2. Macro definitions
	if macroPattern.MatchString(line) {
		issues = append(issues, TemplateIssue{
			File:       file,
			Line:       lineNum,
			Issue:      "{% macro %} is not supported in pongo2",
			Severity:   "error",
			Suggestion: "Convert macros to include templates with variables",
		})
	}

	// 3. Call blocks
	if callPattern.MatchString(line) {
		issues = append(issues, TemplateIssue{
			File:       file,
			Line:       lineNum,
			Issue:      "{% call %} blocks are not supported in pongo2",
			Severity:   "error",
			Suggestion: "Restructure to use includes or standard blocks",
		})
	}

	// 4. Python expressions in templates
	if pythonExprPattern.MatchString(line) {
		issues = append(issues, TemplateIssue{
			File:       file,
			Line:       lineNum,
			Issue:      "Python expressions (list comprehensions, etc.) are not supported",
			Severity:   "error",
			Suggestion: "Pre-compute these values in Go code and pass to template",
		})
	}

	// 5. post.markata access
	if markataAccessPattern.MatchString(line) {
		issues = append(issues, TemplateIssue{
			File:       file,
			Line:       lineNum,
			Issue:      "post.markata access is not available in markata-go",
			Severity:   "warning",
			Suggestion: "Use 'config' directly for config access, 'feeds' for feeds",
		})
	}

	// 6. article_html (renamed to content)
	if articleHTMLPattern.MatchString(line) {
		issues = append(issues, TemplateIssue{
			File:       file,
			Line:       lineNum,
			Issue:      "post.article_html has been renamed",
			Severity:   "warning",
			Suggestion: "Use post.content instead of post.article_html",
		})
	}

	// 7. Import statements
	if importPattern.MatchString(line) {
		issues = append(issues, TemplateIssue{
			File:       file,
			Line:       lineNum,
			Issue:      "{% import %} is not fully supported in pongo2",
			Severity:   "warning",
			Suggestion: "Use {% include %} instead, passing variables explicitly",
		})
	}

	// 8. from...import statements
	if fromImportPattern.MatchString(line) {
		issues = append(issues, TemplateIssue{
			File:       file,
			Line:       lineNum,
			Issue:      "{% from...import %} is not supported in pongo2",
			Severity:   "error",
			Suggestion: "Use {% include %} with explicit variable passing",
		})
	}

	// 9. Check for Python string methods
	if pythonMethodPattern.MatchString(line) {
		issues = append(issues, TemplateIssue{
			File:       file,
			Line:       lineNum,
			Issue:      "Python string methods are not supported",
			Severity:   "warning",
			Suggestion: "Use pongo2 filters like |lower, |upper, |title instead",
		})
	}

	// 10. Check for complex with statements
	if complexWithPattern.MatchString(line) {
		issues = append(issues, TemplateIssue{
			File:       file,
			Line:       lineNum,
			Issue:      "{% with %} with multiple assignments may not work as expected",
			Severity:   "warning",
			Suggestion: "Use multiple {% set %} statements instead",
		})
	}

	return issues
}

// Pre-compiled patterns for performance
var (
	doPattern            = regexp.MustCompile(`\{%\s*do\s+`)
	macroPattern         = regexp.MustCompile(`\{%\s*macro\s+`)
	callPattern          = regexp.MustCompile(`\{%\s*call\s+`)
	pythonExprPattern    = regexp.MustCompile(`\{\{.*\[.*for.*in.*\].*\}\}`)
	markataAccessPattern = regexp.MustCompile(`post\.markata`)
	articleHTMLPattern   = regexp.MustCompile(`post\.article_html|article_html`)
	importPattern        = regexp.MustCompile(`\{%\s*import\s+`)
	fromImportPattern    = regexp.MustCompile(`\{%\s*from\s+.*\s+import\s+`)
	pythonMethodPattern  = regexp.MustCompile(`\{\{.*\.(lower|upper|strip|split|join|replace|format)\(\).*\}\}`)
	complexWithPattern   = regexp.MustCompile(`\{%\s*with\s+\w+\s*=\s*[^,]+,\s*\w+\s*=`)
)

// TemplateVariable represents a template variable and its migration status.
type TemplateVariable struct {
	Old        string
	New        string
	Deprecated bool
	Message    string
}

// GetVariableMigrations returns the list of variable migrations needed.
func GetVariableMigrations() []TemplateVariable {
	return []TemplateVariable{
		{
			Old:        "post.markata.config",
			New:        "config",
			Deprecated: true,
			Message:    "Access config directly without going through post.markata",
		},
		{
			Old:        "post.markata.feeds",
			New:        "feeds",
			Deprecated: true,
			Message:    "Access feeds directly without going through post.markata",
		},
		{
			Old:        "post.article_html",
			New:        "post.content",
			Deprecated: true,
			Message:    "Renamed from article_html to content",
		},
		{
			Old:        "markata.config",
			New:        "config",
			Deprecated: true,
			Message:    "Access config directly",
		},
		{
			Old:        "markata.feeds",
			New:        "feeds",
			Deprecated: true,
			Message:    "Access feeds directly",
		},
	}
}

// GetFilterMigrations returns the list of filter migrations for templates.
func GetFilterMigrations() []TemplateVariable {
	return []TemplateVariable{
		{
			Old:     ".lower()",
			New:     "|lower",
			Message: "Use pongo2 filter syntax",
		},
		{
			Old:     ".upper()",
			New:     "|upper",
			Message: "Use pongo2 filter syntax",
		},
		{
			Old:     ".title()",
			New:     "|title",
			Message: "Use pongo2 filter syntax",
		},
		{
			Old:     ".strip()",
			New:     "|trim",
			Message: "Use pongo2 filter syntax",
		},
		{
			Old:     "|safe",
			New:     "|safe",
			Message: "Filter is the same, but note pongo2 escapes by default",
		},
	}
}
