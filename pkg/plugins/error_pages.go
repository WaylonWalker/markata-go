package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// ErrorPagesPlugin generates static error pages (404.html) during build.
// The 404 page includes:
// - A prefilled search form based on the URL path
// - Client-side slug matching to suggest similar pages
// - Optional Pagefind integration for enhanced search
type ErrorPagesPlugin struct{}

// Compile-time interface verification.
var _ lifecycle.Plugin = (*ErrorPagesPlugin)(nil)
var _ lifecycle.WritePlugin = (*ErrorPagesPlugin)(nil)

// NewErrorPagesPlugin creates a new ErrorPagesPlugin.
func NewErrorPagesPlugin() *ErrorPagesPlugin {
	return &ErrorPagesPlugin{}
}

// Name returns the plugin name.
func (p *ErrorPagesPlugin) Name() string {
	return "error_pages"
}

// Write generates static error pages during the Write stage.
func (p *ErrorPagesPlugin) Write(m *lifecycle.Manager) error {
	// Get the full config
	lcConfig := m.Config()
	if lcConfig == nil || lcConfig.Extra == nil {
		return nil
	}

	cfg, ok := lcConfig.Extra["models_config"].(*models.Config)
	if !ok || cfg == nil {
		// No config available, skip
		return nil
	}

	// Check if 404 page is enabled
	if !cfg.ErrorPages.Is404Enabled() {
		return nil
	}

	// Generate posts index for client-side matching
	if err := p.generatePostsIndex(m, cfg); err != nil {
		return err
	}

	// Generate 404 page
	return p.generate404Page(m, cfg)
}

// postIndexEntry is a lightweight post entry for the 404 search index.
type postIndexEntry struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

// generatePostsIndex creates a lightweight JSON index of all posts for client-side matching.
func (p *ErrorPagesPlugin) generatePostsIndex(m *lifecycle.Manager, cfg *models.Config) error {
	posts := m.Posts()
	entries := make([]postIndexEntry, 0, len(posts))

	for _, post := range posts {
		if post == nil || post.Slug == "" {
			continue
		}

		entry := postIndexEntry{
			Slug: post.Slug,
			URL:  "/" + post.Slug + "/",
		}

		if post.Title != nil {
			entry.Title = *post.Title
		} else {
			entry.Title = post.Slug
		}

		if post.Description != nil {
			entry.Description = *post.Description
		}

		entries = append(entries, entry)
	}

	// Marshal to JSON
	data, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("marshaling posts index: %w", err)
	}

	// Write to output directory
	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = "output"
	}

	indexPath := filepath.Join(outputDir, "_404-index.json")
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		return fmt.Errorf("creating output directory for 404 index: %w", err)
	}

	if err := os.WriteFile(indexPath, data, 0o600); err != nil {
		return fmt.Errorf("writing 404 index: %w", err)
	}

	return nil
}

// generate404Page creates the static 404.html file.
// It renders through the post template so users get their full site experience
// (search, sidebars, recent posts, etc.) and uses client-side JS for suggestions.
func (p *ErrorPagesPlugin) generate404Page(_ *lifecycle.Manager, cfg *models.Config) error {
	// Create template engine
	templatesDir := cfg.TemplatesDir
	if templatesDir == "" {
		templatesDir = PluginNameTemplates
	}
	engine, err := templates.NewEngineWithTheme(templatesDir, cfg.Theme.Name)
	if err != nil {
		return fmt.Errorf("creating template engine for 404 page: %w", err)
	}

	// Create a synthetic post for the 404 page
	title := "Page Not Found"
	description := "The requested page could not be found."
	post := &models.Post{
		Slug:        "404",
		Title:       &title,
		Description: &description,
	}

	// Generate the 404 page body content with client-side fuzzy search
	body := generate404Body(cfg)

	// Create template context using the post template
	ctx := templates.NewContext(post, body, cfg)

	// Determine template name - use post template for full site experience
	templateName := cfg.ErrorPages.Custom404Template
	if templateName == "" {
		templateName = "post.html"
	}

	// Render the 404 template
	html, err := engine.Render(templateName, ctx)
	if err != nil {
		return fmt.Errorf("rendering 404 template: %w", err)
	}

	// Write to output directory
	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = "output"
	}

	outputPath := filepath.Join(outputDir, "404.html")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("creating output directory for 404 page: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(html), 0o600); err != nil {
		return fmt.Errorf("writing 404.html: %w", err)
	}

	return nil
}

// generate404Body creates the HTML body content for the 404 page.
// Includes a prefilled search form and client-side slug matching.
func generate404Body(cfg *models.Config) string {
	// Get pagefind bundle directory for optional enhanced search
	bundleDir := cfg.Search.Pagefind.BundleDir
	if bundleDir == "" {
		bundleDir = defaultBundleDir
	}

	return fmt.Sprintf(`<div class="error-404">
    <p class="error-message">
        The page you're looking for doesn't exist or has been moved.
    </p>
    <p class="requested-path">
        Looking for: <code id="requested-path"></code>
    </p>

    <!-- Prefilled search form -->
    <div class="search-form-section">
        <h2>Search for it</h2>
        <form id="search-form" class="search-form" action="/" method="get">
            <input
                type="text"
                id="search-input"
                name="q"
                class="search-input"
                placeholder="Search..."
                autocomplete="off"
            >
            <button type="submit" class="search-button">Search</button>
        </form>
    </div>

    <!-- Suggestions based on URL slug matching -->
    <div id="suggestions-section" class="suggestions" style="display: none;">
        <h2>Did you mean one of these?</h2>
        <ul id="suggestions-list" class="suggestion-list"></ul>
    </div>

    <div class="error-actions">
        <p>Or try:</p>
        <ul>
            <li><a href="/">Go to the home page</a></li>
        </ul>
    </div>
</div>

<style>
.error-404 {
    max-width: 600px;
    margin: 0 auto;
    padding: var(--spacing-lg, 2rem) var(--spacing-md, 1rem);
}

.error-message {
    font-size: var(--font-size-lg, 1.125rem);
    color: var(--color-text-muted, #6b7280);
    margin-bottom: var(--spacing-md, 1rem);
    text-align: center;
}

.requested-path {
    margin-bottom: var(--spacing-lg, 2rem);
    text-align: center;
}

.requested-path code {
    background: var(--color-code-bg, #f3f4f6);
    padding: var(--spacing-xs, 0.25rem) var(--spacing-sm, 0.5rem);
    border-radius: var(--radius-sm, 0.25rem);
    font-family: var(--font-mono, monospace);
    color: var(--color-text, #1f2937);
}

.search-form-section {
    margin: var(--spacing-xl, 2rem) 0;
}

.search-form-section h2 {
    font-size: var(--font-size-lg, 1.125rem);
    margin-bottom: var(--spacing-md, 1rem);
    color: var(--color-heading, #111827);
}

.search-form {
    display: flex;
    gap: var(--spacing-sm, 0.5rem);
}

.search-input {
    flex: 1;
    padding: var(--spacing-sm, 0.75rem) var(--spacing-md, 1rem);
    font-size: var(--font-size-base, 1rem);
    border: 2px solid var(--color-border, #e5e7eb);
    border-radius: var(--radius-md, 0.5rem);
    background: var(--color-bg, #ffffff);
    color: var(--color-text, #1f2937);
    transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.search-input:focus {
    outline: none;
    border-color: var(--color-primary, #3b82f6);
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.search-button {
    padding: var(--spacing-sm, 0.75rem) var(--spacing-lg, 1.5rem);
    font-size: var(--font-size-base, 1rem);
    font-weight: 600;
    color: white;
    background: var(--color-primary, #3b82f6);
    border: none;
    border-radius: var(--radius-md, 0.5rem);
    cursor: pointer;
    transition: background-color 0.2s ease;
}

.search-button:hover {
    background: var(--color-primary-dark, #2563eb);
}

.suggestions {
    margin: var(--spacing-xl, 2rem) 0;
    padding: var(--spacing-lg, 1.5rem);
    background: var(--color-surface, #f9fafb);
    border-radius: var(--radius-md, 0.5rem);
    border: 1px solid var(--color-border, #e5e7eb);
}

.suggestions h2 {
    font-size: var(--font-size-lg, 1.125rem);
    margin-bottom: var(--spacing-md, 1rem);
    color: var(--color-heading, #111827);
}

.suggestion-list {
    list-style: none;
    padding: 0;
    margin: 0;
}

.suggestion-item {
    margin-bottom: var(--spacing-sm, 0.5rem);
}

.suggestion-item a {
    display: block;
    padding: var(--spacing-md, 1rem);
    background: var(--color-bg, #ffffff);
    border-radius: var(--radius-sm, 0.25rem);
    text-decoration: none;
    transition: background-color 0.2s ease, transform 0.2s ease;
    border: 1px solid var(--color-border, #e5e7eb);
}

.suggestion-item a:hover {
    background: var(--color-surface-hover, #f3f4f6);
    transform: translateX(4px);
}

.suggestion-title {
    display: block;
    font-weight: 600;
    color: var(--color-primary, #3b82f6);
}

.suggestion-description {
    display: block;
    font-size: var(--font-size-sm, 0.875rem);
    color: var(--color-text-muted, #6b7280);
    margin-top: var(--spacing-xs, 0.25rem);
}

.suggestion-match {
    display: inline-block;
    font-size: var(--font-size-xs, 0.75rem);
    color: var(--color-text-muted, #9ca3af);
    margin-left: var(--spacing-sm, 0.5rem);
}

.error-actions {
    margin-top: var(--spacing-xl, 2rem);
}

.error-actions ul {
    list-style-position: inside;
    padding-left: 0;
}

.error-actions li {
    margin-bottom: var(--spacing-xs, 0.25rem);
}

.error-actions a {
    color: var(--color-primary, #3b82f6);
}

/* Dark mode */
@media (prefers-color-scheme: dark) {
    .requested-path code {
        background: var(--color-code-bg, #374151);
    }

    .search-input {
        background: var(--color-bg, #111827);
        border-color: var(--color-border, #374151);
        color: var(--color-text, #f3f4f6);
    }

    .suggestions {
        background: var(--color-surface, #1f2937);
        border-color: var(--color-border, #374151);
    }

    .suggestion-item a {
        background: var(--color-bg, #111827);
        border-color: var(--color-border, #374151);
    }

    .suggestion-item a:hover {
        background: var(--color-surface-hover, #374151);
    }
}
</style>

<script>
// 404 Page - Client-side slug matching and search prefill
(function() {
    'use strict';

    const path = window.location.pathname;
    const requestedPathEl = document.getElementById('requested-path');
    const searchInput = document.getElementById('search-input');
    const searchForm = document.getElementById('search-form');
    const suggestionsSection = document.getElementById('suggestions-section');
    const suggestionsList = document.getElementById('suggestions-list');

    // Show the requested path
    if (requestedPathEl) {
        requestedPathEl.textContent = path;
    }

    // Extract search terms from URL path
    function extractSearchTerms(urlPath) {
        return urlPath
            .replace(/^\/+|\/+$/g, '')   // Remove leading/trailing slashes
            .replace(/[-_]/g, ' ')        // Convert dashes/underscores to spaces
            .replace(/\.[^.]+$/, '')      // Remove file extension
            .replace(/\s+/g, ' ')         // Normalize whitespace
            .trim();
    }

    // Prefill the search input
    const searchTerms = extractSearchTerms(path);
    if (searchInput && searchTerms) {
        searchInput.value = searchTerms;
    }

    // Handle form submission - redirect to home with search param
    // This works with pagefind's URL-based search initialization
    if (searchForm) {
        searchForm.addEventListener('submit', function(e) {
            e.preventDefault();
            const query = searchInput.value.trim();
            if (query) {
                // Redirect to home with search query in hash (pagefind style)
                window.location.href = '/#search=' + encodeURIComponent(query);
            } else {
                window.location.href = '/';
            }
        });
    }

    // Levenshtein distance for fuzzy matching
    function levenshtein(a, b) {
        if (a.length === 0) return b.length;
        if (b.length === 0) return a.length;

        const matrix = [];
        for (let i = 0; i <= b.length; i++) {
            matrix[i] = [i];
        }
        for (let j = 0; j <= a.length; j++) {
            matrix[0][j] = j;
        }

        for (let i = 1; i <= b.length; i++) {
            for (let j = 1; j <= a.length; j++) {
                if (b.charAt(i - 1) === a.charAt(j - 1)) {
                    matrix[i][j] = matrix[i - 1][j - 1];
                } else {
                    matrix[i][j] = Math.min(
                        matrix[i - 1][j - 1] + 1, // substitution
                        matrix[i][j - 1] + 1,     // insertion
                        matrix[i - 1][j] + 1      // deletion
                    );
                }
            }
        }
        return matrix[b.length][a.length];
    }

    // Normalize a slug for comparison
    function normalizeSlug(slug) {
        return slug
            .toLowerCase()
            .replace(/^\/+|\/+$/g, '')
            .replace(/[-_]/g, '')
            .replace(/\s+/g, '');
    }

    // Find similar posts based on slug
    function findSimilarPosts(posts, targetPath, maxResults) {
        const targetSlug = normalizeSlug(targetPath);
        const targetWords = extractSearchTerms(targetPath).toLowerCase().split(/\s+/);

        const scored = posts.map(post => {
            const postSlug = normalizeSlug(post.slug);
            const postTitle = (post.title || '').toLowerCase();

            // Calculate slug distance
            const slugDistance = levenshtein(targetSlug, postSlug);

            // Calculate word overlap score (bonus for matching words)
            let wordMatchScore = 0;
            targetWords.forEach(word => {
                if (word.length >= 3) {
                    if (postSlug.includes(word)) wordMatchScore += 2;
                    if (postTitle.includes(word)) wordMatchScore += 1;
                }
            });

            // Combined score (lower is better, subtract word matches as bonus)
            const score = slugDistance - (wordMatchScore * 2);

            return { post, score, slugDistance };
        });

        // Sort by score (lower is better) and filter reasonable matches
        return scored
            .filter(s => s.slugDistance <= Math.max(targetSlug.length * 0.6, 5))
            .sort((a, b) => a.score - b.score)
            .slice(0, maxResults)
            .map(s => s.post);
    }

    // Render suggestions
    function renderSuggestions(posts) {
        if (!posts || posts.length === 0) return;

        suggestionsList.innerHTML = '';
        posts.forEach(post => {
            const li = document.createElement('li');
            li.className = 'suggestion-item';

            const a = document.createElement('a');
            a.href = post.url;

            const title = document.createElement('span');
            title.className = 'suggestion-title';
            title.textContent = post.title || post.slug;
            a.appendChild(title);

            if (post.description) {
                const desc = document.createElement('span');
                desc.className = 'suggestion-description';
                desc.textContent = post.description;
                a.appendChild(desc);
            }

            li.appendChild(a);
            suggestionsList.appendChild(li);
        });

        suggestionsSection.style.display = 'block';
    }

    // Load posts index and find suggestions
    async function loadAndSuggest() {
        try {
            const response = await fetch('/_404-index.json');
            if (!response.ok) return;

            const posts = await response.json();
            const similar = findSimilarPosts(posts, path, 5);
            renderSuggestions(similar);
        } catch (e) {
            console.debug('404 suggestions unavailable:', e.message);
        }
    }

    // Try Pagefind first for richer results, fall back to our index
    async function init() {
        try {
            const pagefind = await import('/%s/pagefind.js');
            await pagefind.init();

            const search = await pagefind.search(searchTerms);
            if (search.results.length > 0) {
                const results = await Promise.all(
                    search.results.slice(0, 5).map(r => r.data())
                );
                const suggestions = results
                    .filter(r => !r.url.includes('404'))
                    .map(r => ({
                        url: r.url,
                        title: r.meta?.title || r.url,
                        description: r.excerpt ? r.excerpt.replace(/<[^>]*>/g, '') : ''
                    }));

                if (suggestions.length > 0) {
                    renderSuggestions(suggestions);
                    return;
                }
            }
        } catch (e) {
            // Pagefind not available, fall back to our index
            console.debug('Pagefind not available, using fallback:', e.message);
        }

        // Fall back to our lightweight index
        await loadAndSuggest();
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
</script>`, bundleDir)
}
