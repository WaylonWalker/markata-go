// Package templates provides a template engine for markata-go using pongo2 (Jinja2-like syntax).
package templates

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/flosch/pongo2/v6"
)

// Engine provides template rendering capabilities using pongo2.
type Engine struct {
	set *pongo2.TemplateSet
	dir string
	mu  sync.RWMutex

	// templateCache stores compiled templates for reuse
	templateCache map[string]*pongo2.Template

	// searchPaths is the ordered list of directories to search for templates
	// Resolution order: project templates -> theme templates -> default theme
	searchPaths []string

	// themeName is the current theme name
	themeName string
}

// NewEngine creates a new template engine with the given templates directory.
// The templates directory is used as the base path for template loading.
// If templatesDir is empty, templates can only be rendered from strings.
func NewEngine(templatesDir string) (*Engine, error) {
	return NewEngineWithTheme(templatesDir, "default")
}

// NewEngineWithTheme creates a new template engine with theme support.
func NewEngineWithTheme(templatesDir, themeName string) (*Engine, error) {
	e := &Engine{
		dir:           templatesDir,
		templateCache: make(map[string]*pongo2.Template),
		searchPaths:   make([]string, 0),
		themeName:     themeName,
	}

	if e.themeName == "" {
		e.themeName = "default"
	}

	// Register custom filters
	registerFilters()

	// Build search paths
	e.buildSearchPaths(templatesDir)

	// Create a basic template set (we'll handle loading ourselves)
	e.set = pongo2.NewSet("default", pongo2.MustNewLocalFileSystemLoader(""))

	return e, nil
}

// buildSearchPaths constructs the ordered list of template directories.
func (e *Engine) buildSearchPaths(templatesDir string) {
	e.searchPaths = make([]string, 0)

	// 1. Project templates (highest priority)
	if templatesDir != "" {
		if _, err := os.Stat(templatesDir); err == nil {
			e.searchPaths = append(e.searchPaths, templatesDir)
		}
	}

	// 2. Current theme templates
	if e.themeName != "" && e.themeName != "default" {
		themeTemplatesDir := filepath.Join("themes", e.themeName, "templates")
		if _, err := os.Stat(themeTemplatesDir); err == nil {
			e.searchPaths = append(e.searchPaths, themeTemplatesDir)
		}
	}

	// 3. Default theme templates (fallback)
	defaultThemeDir := filepath.Join("themes", "default", "templates")
	if _, err := os.Stat(defaultThemeDir); err == nil {
		e.searchPaths = append(e.searchPaths, defaultThemeDir)
	}

	// If no paths found, use the original templatesDir even if it doesn't exist
	if len(e.searchPaths) == 0 && templatesDir != "" {
		e.searchPaths = append(e.searchPaths, templatesDir)
	}
}

// Render renders a template file with the given context.
// templateName is the path relative to the templates directory.
func (e *Engine) Render(templateName string, ctx Context) (string, error) {
	tpl, err := e.LoadTemplate(templateName)
	if err != nil {
		return "", fmt.Errorf("failed to load template %q: %w", templateName, err)
	}

	result, err := tpl.Execute(ctx.ToPongo2())
	if err != nil {
		return "", fmt.Errorf("failed to execute template %q: %w", templateName, err)
	}

	return result, nil
}

// RenderString renders a template string with the given context.
// This is useful for inline templates in markdown content (jinja_md).
func (e *Engine) RenderString(templateStr string, ctx Context) (string, error) {
	tpl, err := pongo2.FromString(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template string: %w", err)
	}

	result, err := tpl.Execute(ctx.ToPongo2())
	if err != nil {
		return "", fmt.Errorf("failed to execute template string: %w", err)
	}

	return result, nil
}

// LoadTemplate loads and caches a template by name.
// The template is loaded from the search paths in order.
func (e *Engine) LoadTemplate(name string) (*pongo2.Template, error) {
	e.mu.RLock()
	if tpl, ok := e.templateCache[name]; ok {
		e.mu.RUnlock()
		return tpl, nil
	}
	e.mu.RUnlock()

	// Find the template in search paths
	templatePath := e.FindTemplate(name)
	if templatePath == "" {
		return nil, fmt.Errorf("template %q not found in search paths %v", name, e.searchPaths)
	}

	// Get absolute path
	absPath, err := filepath.Abs(templatePath)
	if err != nil {
		absPath = templatePath
	}

	// Create a template set with a multi-directory loader for proper include/extends support
	tplSet := pongo2.NewSet(name, &searchPathLoader{
		searchPaths: e.searchPaths,
	})

	// Load the template using absolute path
	tpl, err := tplSet.FromFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %q: %w", name, err)
	}

	// Cache the compiled template
	e.mu.Lock()
	e.templateCache[name] = tpl
	e.mu.Unlock()

	return tpl, nil
}

// searchPathLoader implements pongo2.TemplateLoader for multi-directory support.
type searchPathLoader struct {
	searchPaths []string
}

func (l *searchPathLoader) Abs(base, name string) string {
	// If name is already absolute, return it
	if filepath.IsAbs(name) {
		return name
	}

	// If base is provided, try relative to base directory first
	if base != "" {
		baseDir := filepath.Dir(base)
		candidate := filepath.Join(baseDir, name)
		if _, err := os.Stat(candidate); err == nil {
			absPath, _ := filepath.Abs(candidate)
			return absPath
		}
	}

	// Search through all search paths
	for _, dir := range l.searchPaths {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			absPath, _ := filepath.Abs(candidate)
			return absPath
		}
	}

	// Return name with first search path as fallback (will fail gracefully)
	if len(l.searchPaths) > 0 {
		absPath, _ := filepath.Abs(filepath.Join(l.searchPaths[0], name))
		return absPath
	}
	return name
}

func (l *searchPathLoader) Get(path string) (io.Reader, error) {
	// If path is already absolute, just try to open it
	if filepath.IsAbs(path) {
		file, err := os.Open(path)
		if err == nil {
			return file, nil
		}
	}

	// Search through all search paths
	for _, dir := range l.searchPaths {
		candidate := filepath.Join(dir, path)
		file, err := os.Open(candidate)
		if err == nil {
			return file, nil
		}
	}

	// Last resort: try to open path as-is
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("template %q not found in search paths", path)
	}
	return file, nil
}

// TemplateExists checks if a template file exists in any search path.
func (e *Engine) TemplateExists(name string) bool {
	for _, dir := range e.searchPaths {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// FindTemplate returns the full path to the template, searching through all paths.
func (e *Engine) FindTemplate(name string) string {
	for _, dir := range e.searchPaths {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// ClearCache clears the template cache.
// This is useful during development when templates are being modified.
func (e *Engine) ClearCache() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.templateCache = make(map[string]*pongo2.Template)
}

// Dir returns the templates directory path.
func (e *Engine) Dir() string {
	return e.dir
}

// SearchPaths returns the ordered list of template search paths.
func (e *Engine) SearchPaths() []string {
	return e.searchPaths
}

// ThemeName returns the current theme name.
func (e *Engine) ThemeName() string {
	return e.themeName
}

// SetTheme sets the theme and rebuilds search paths.
func (e *Engine) SetTheme(themeName string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.themeName = themeName
	if e.themeName == "" {
		e.themeName = "default"
	}

	// Clear cache and rebuild search paths
	e.templateCache = make(map[string]*pongo2.Template)
	e.buildSearchPaths(e.dir)
}

// SetDir sets the templates directory and reinitializes the loader.
func (e *Engine) SetDir(dir string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.dir = dir
	e.templateCache = make(map[string]*pongo2.Template)

	// Rebuild search paths
	e.buildSearchPaths(dir)

	return nil
}
