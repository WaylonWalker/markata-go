package authors

import (
	"fmt"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// AuthorPlugin handles multi-author support for markata-go
// Manages author validation, linking, and page generation
type AuthorPlugin struct {
	authors         map[string]models.Author
	defaultAuthor   *models.Author
	defaultAuthorID string
}

// NewAuthorPlugin creates a new AuthorPlugin
func NewAuthorPlugin() *AuthorPlugin {
	return &AuthorPlugin{
		authors: make(map[string]models.Author),
	}
}

// Name returns the plugin identifier
func (p *AuthorPlugin) Name() string {
	return "authors"
}

// getFeedDefaults retrieves feed defaults from config or returns sensible defaults
func getFeedDefaults(config *lifecycle.Config) models.FeedDefaults {
	if config.Extra == nil {
		return models.NewFeedDefaults()
	}

	if defaults, ok := config.Extra["feed_defaults"]; ok {
		if feedDefaults, ok := defaults.(models.FeedDefaults); ok {
			return feedDefaults
		}
	}

	return models.NewFeedDefaults()
}

// Configure loads author configuration and validates it
func (p *AuthorPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()

	// Validate authors configuration if present
	if config.Authors.Authors != nil {
		if err := models.ValidateAuthors(config.Authors.Authors); err != nil {
			return fmt.Errorf("authors configuration validation failed: %w", err)
		}

		// Load authors into plugin state
		p.authors = config.Authors.Authors

		// Find default author
		defaultAuthor, defaultAuthorID := models.GetDefaultAuthor(config.Authors.Authors)
		p.defaultAuthor = defaultAuthor
		p.defaultAuthorID = defaultAuthorID
	}

	return nil
}

// Transform links post author references to author objects
func (p *AuthorPlugin) Transform(m *lifecycle.Manager) error {
	posts := m.Posts()

	for _, post := range posts {
		authorIDs := post.GetAuthors()
		if len(authorIDs) > 0 {
			// Clear any existing author objects
			post.AuthorObjects = nil

			// Link each author ID to author object
			for _, authorID := range authorIDs {
				if author, exists := p.authors[authorID]; exists {
					// Clone author object to avoid modifying the original
					authorClone := author

					// Override role-specific data based on post context if needed
					// Future enhancement: support per-post role overrides

					post.AuthorObjects = append(post.AuthorObjects, authorClone)
				} else {
					// Author not found in config, use fallback
					if p.defaultAuthor != nil {
						// Use default author with partial info
						fallbackAuthor := *p.defaultAuthor
						fallbackAuthor.ID = authorID
						post.AuthorObjects = append(post.AuthorObjects, fallbackAuthor)
					}
				}
			}
		}
	}

	return nil
}

// Collect generates author pages and collections
func (p *AuthorPlugin) Collect(m *lifecycle.Manager) error {
	config := m.Config()

	// Skip author page generation if disabled
	if !config.Authors.GeneratePages {
		return nil
	}

	// Create author posts collection
	authorPosts := make(map[string][]*models.Post)
	for _, post := range m.Posts() {
		for _, authorID := range post.GetAuthors() {
			authorPosts[authorID] = append(authorPosts[authorID], post)
		}
	}

	// Generate author posts for template context
	for authorID, posts := range authorPosts {
		// Create a pseudo-post for author page generation
		author, exists := p.authors[authorID]
		if !exists {
			continue
		}

		authorPost := &models.Post{
			Path:      "authors/" + authorID,
			Slug:      "authors/" + authorID,
			Href:      "/authors/" + authorID + "/",
			Title:     &author.Name,
			Content:   "", // Will be filled by templates plugin
			Published: true,
		}

		// Set author info in Extra for template access
		authorPost.Set("author", author)
		authorPost.Set("posts", posts)

		// Add to lifecycle for rendering
		m.AddPost(authorPost)
	}

	return nil
}

// Render handles author page rendering (no additional work needed)
// Template rendering is handled by the templates plugin
func (p *AuthorPlugin) Render(m *lifecycle.Manager) error {
	// Author pages are rendered by the templates plugin
	// using the posts we created in the Collect phase
	return nil
}

// Write handles any post-author page writing (no additional work needed)
// Author pages are written by the templates plugin
func (p *AuthorPlugin) Write(m *lifecycle.Manager) error {
	// Author pages are written by the templates plugin
	return nil
}

// GetAuthor returns an author by ID
func (p *AuthorPlugin) GetAuthor(authorID string) (*models.Author, bool) {
	author, exists := p.authors[authorID]
	return &author, exists
}

// GetDefaultAuthor returns the default author
func (p *AuthorPlugin) GetDefaultAuthor() (*models.Author, bool) {
	if p.defaultAuthor != nil {
		return p.defaultAuthor, true
	}
	return nil, false
}

// GetAllAuthors returns all configured authors
func (p *AuthorPlugin) GetAllAuthors() map[string]models.Author {
	result := make(map[string]models.Author)
	for id, author := range p.authors {
		result[id] = author
	}
	return result
}
