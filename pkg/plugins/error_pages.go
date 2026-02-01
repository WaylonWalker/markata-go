package plugins

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// ErrorPagesPlugin generates static error pages (404.html) during build.
// The 404 page uses client-side JavaScript with Pagefind for fuzzy search
// suggestions, so it works fully statically without any backend.
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

	// Generate 404 page
	return p.generate404Page(m, cfg)
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
// This includes the error message and client-side JavaScript that uses
// Pagefind for fuzzy search suggestions based on the current URL.
func generate404Body(cfg *models.Config) string {
	// Get pagefind bundle directory
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

    <div id="suggestions-section" class="suggestions" style="display: none;">
        <h2>Did you mean one of these?</h2>
        <ul id="suggestions-list" class="suggestion-list"></ul>
    </div>

    <div class="error-actions">
        <p>You can also try:</p>
        <ul>
            <li><a href="/">Go to the home page</a></li>
            <li>Use the search above to find what you're looking for</li>
        </ul>
    </div>
</div>

<style>
.error-404 {
    text-align: center;
    padding: var(--spacing-lg, 2rem) 0;
}

.error-message {
    font-size: var(--font-size-lg, 1.125rem);
    color: var(--color-text-muted, #6b7280);
    margin-bottom: var(--spacing-md, 1rem);
}

.requested-path {
    margin-bottom: var(--spacing-lg, 2rem);
}

.requested-path code {
    background: var(--color-code-bg, #f3f4f6);
    padding: var(--spacing-xs, 0.25rem) var(--spacing-sm, 0.5rem);
    border-radius: var(--radius-sm, 0.25rem);
    font-family: var(--font-mono, monospace);
    color: var(--color-text, #1f2937);
}

.suggestions {
    text-align: left;
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

.suggestion-excerpt {
    display: block;
    font-size: var(--font-size-sm, 0.875rem);
    color: var(--color-text-muted, #6b7280);
    margin-top: var(--spacing-xs, 0.25rem);
}

.error-actions {
    text-align: left;
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

<script type="module">
// Client-side fuzzy search for 404 suggestions using Pagefind
(async function() {
    const path = window.location.pathname;
    document.getElementById('requested-path').textContent = path;

    // Extract search terms from the URL path
    const searchTerms = path
        .replace(/^\/+|\/+$/g, '')  // Remove leading/trailing slashes
        .replace(/[-_]/g, ' ')       // Convert dashes/underscores to spaces
        .replace(/\.[^.]+$/, '')     // Remove file extension
        .toLowerCase();

    if (!searchTerms) return;

    try {
        // Load Pagefind
        const pagefind = await import('/%s/pagefind.js');
        await pagefind.init();

        // Search for similar content
        const search = await pagefind.search(searchTerms);

        if (search.results.length === 0) return;

        // Get top 5 results
        const results = await Promise.all(
            search.results.slice(0, 5).map(r => r.data())
        );

        // Filter out the 404 page itself
        const suggestions = results.filter(r => !r.url.includes('404'));

        if (suggestions.length === 0) return;

        // Show suggestions
        const section = document.getElementById('suggestions-section');
        const list = document.getElementById('suggestions-list');

        suggestions.forEach(result => {
            const li = document.createElement('li');
            li.className = 'suggestion-item';

            const a = document.createElement('a');
            a.href = result.url;

            const title = document.createElement('span');
            title.className = 'suggestion-title';
            title.textContent = result.meta?.title || result.url;
            a.appendChild(title);

            if (result.excerpt) {
                const excerpt = document.createElement('span');
                excerpt.className = 'suggestion-excerpt';
                excerpt.innerHTML = result.excerpt;
                a.appendChild(excerpt);
            }

            li.appendChild(a);
            list.appendChild(li);
        });

        section.style.display = 'block';
    } catch (e) {
        // Pagefind not available or search failed - that's okay
        console.debug('404 suggestions unavailable:', e.message);
    }
})();
</script>`, bundleDir)
}
