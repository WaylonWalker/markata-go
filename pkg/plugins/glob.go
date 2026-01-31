package plugins

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/bmatcuk/doublestar/v4"
)

// GlobPlugin discovers content files using glob patterns.
type GlobPlugin struct {
	// patterns are the glob patterns to match files against.
	// Supports ** for recursive matching (doublestar patterns).
	patterns []string

	// useGitignore determines whether to parse and respect .gitignore.
	useGitignore bool

	// gitignorePatterns holds parsed gitignore patterns.
	gitignorePatterns []string
}

// NewGlobPlugin creates a new GlobPlugin with default settings.
func NewGlobPlugin() *GlobPlugin {
	return &GlobPlugin{
		patterns:     []string{"**/*.md"},
		useGitignore: true,
	}
}

// Name returns the plugin identifier.
func (p *GlobPlugin) Name() string {
	return "glob"
}

// Configure reads configuration from the manager and initializes the plugin.
func (p *GlobPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()

	// Get glob patterns from config
	if len(config.GlobPatterns) > 0 {
		p.patterns = config.GlobPatterns
	}

	// Check for useGitignore setting in Extra config
	if extra := config.Extra; extra != nil {
		if useGitignore, ok := extra["use_gitignore"].(bool); ok {
			p.useGitignore = useGitignore
		}
	}

	// Parse .gitignore if enabled
	if p.useGitignore {
		if err := p.loadGitignore(config.ContentDir); err != nil {
			// Don't fail if .gitignore doesn't exist
			if !os.IsNotExist(err) {
				return err
			}
		}
	}

	return nil
}

// loadGitignore reads and parses .gitignore patterns.
func (p *GlobPlugin) loadGitignore(baseDir string) error {
	gitignorePath := filepath.Join(baseDir, ".gitignore")
	file, err := os.Open(gitignorePath)
	if err != nil {
		return err
	}
	defer file.Close()

	p.gitignorePatterns = make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p.gitignorePatterns = append(p.gitignorePatterns, line)
	}

	return scanner.Err()
}

// isIgnored checks if a path matches any gitignore pattern.
func (p *GlobPlugin) isIgnored(path string) bool {
	if !p.useGitignore || len(p.gitignorePatterns) == 0 {
		return false
	}

	// Normalize path separators
	normalizedPath := filepath.ToSlash(path)

	for _, pattern := range p.gitignorePatterns {
		// Handle negation patterns (patterns starting with !)
		if strings.HasPrefix(pattern, "!") {
			continue // Skip negation for now in ignore check
		}

		// Normalize the pattern
		normalizedPattern := filepath.ToSlash(pattern)

		// Handle directory patterns (ending with /)
		normalizedPattern = strings.TrimSuffix(normalizedPattern, "/")

		// Try different matching strategies

		// 1. Direct match with the pattern
		matched, err := doublestar.Match(normalizedPattern, normalizedPath)
		if err == nil && matched {
			return true
		}

		// 2. Pattern as prefix (for directory patterns)
		if strings.HasPrefix(normalizedPath, normalizedPattern+"/") {
			return true
		}

		// 3. Match against just the filename
		filename := filepath.Base(normalizedPath)
		matched, err = doublestar.Match(normalizedPattern, filename)
		if err == nil && matched {
			return true
		}

		// 4. Try with **/ prefix for patterns that should match anywhere
		if !strings.HasPrefix(normalizedPattern, "**/") && !strings.HasPrefix(normalizedPattern, "/") {
			matched, err = doublestar.Match("**/"+normalizedPattern, normalizedPath)
			if err == nil && matched {
				return true
			}
		}
	}

	return false
}

// Glob discovers content files matching the configured patterns.
// Uses cached file list when patterns haven't changed.
func (p *GlobPlugin) Glob(m *lifecycle.Manager) error {
	config := m.Config()
	baseDir := config.ContentDir
	if baseDir == "" {
		baseDir = "."
	}

	// Convert to absolute path for consistent matching
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return err
	}

	// Check for cached file list
	cache := GetBuildCache(m)
	patternHash := buildcache.HashContent(strings.Join(p.patterns, "\n"))

	if cache != nil {
		cachedFiles, cachedHash := cache.GetGlobCache()
		if cachedHash == patternHash && len(cachedFiles) > 0 {
			// Patterns unchanged - use cached list, just check for new files
			newFiles := p.findNewFiles(absBaseDir, cachedFiles)
			if len(newFiles) == 0 {
				m.SetFiles(cachedFiles)
				return nil
			}
			// Append new files and re-sort
			allFiles := make([]string, 0, len(cachedFiles)+len(newFiles))
			allFiles = append(allFiles, cachedFiles...)
			allFiles = append(allFiles, newFiles...)
			sort.Strings(allFiles)
			cache.SetGlobCache(allFiles, patternHash)
			m.SetFiles(allFiles)
			return nil
		}
	}

	// Full scan
	files := p.scanFiles(absBaseDir)

	// Cache for next build
	if cache != nil {
		cache.SetGlobCache(files, patternHash)
	}

	m.SetFiles(files)
	return nil
}

// scanFiles performs full glob scan.
func (p *GlobPlugin) scanFiles(absBaseDir string) []string {
	fileSet := make(map[string]struct{})

	for _, pattern := range p.patterns {
		fullPattern := pattern
		if !filepath.IsAbs(pattern) {
			fullPattern = filepath.Join(absBaseDir, pattern)
		}

		matches, err := doublestar.FilepathGlob(fullPattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			relPath, err := filepath.Rel(absBaseDir, match)
			if err != nil {
				relPath = match
			}

			if p.isIgnored(relPath) {
				continue
			}

			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}

			fileSet[relPath] = struct{}{}
		}
	}

	files := make([]string, 0, len(fileSet))
	for file := range fileSet {
		files = append(files, file)
	}
	sort.Strings(files)
	return files
}

// findNewFiles returns files matching patterns that aren't in the cached list.
func (p *GlobPlugin) findNewFiles(absBaseDir string, cached []string) []string {
	known := make(map[string]struct{}, len(cached))
	for _, f := range cached {
		known[f] = struct{}{}
	}

	var newFiles []string
	for _, pattern := range p.patterns {
		fullPattern := pattern
		if !filepath.IsAbs(pattern) {
			fullPattern = filepath.Join(absBaseDir, pattern)
		}

		matches, err := doublestar.FilepathGlob(fullPattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			relPath, err := filepath.Rel(absBaseDir, match)
			if err != nil {
				relPath = match
			}

			if _, exists := known[relPath]; exists {
				continue
			}
			if p.isIgnored(relPath) {
				continue
			}

			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}

			newFiles = append(newFiles, relPath)
		}
	}
	return newFiles
}

// SetPatterns sets the glob patterns to use for file discovery.
func (p *GlobPlugin) SetPatterns(patterns []string) {
	p.patterns = patterns
}

// SetUseGitignore enables or disables gitignore support.
func (p *GlobPlugin) SetUseGitignore(use bool) {
	p.useGitignore = use
}
