package authors

import (
	"fmt"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

// PageGenerator handles creation of author page posts
type PageGenerator struct {
	urlPattern string
}

// NewPageGenerator creates a new PageGenerator
func NewPageGenerator(urlPattern string) *PageGenerator {
	if urlPattern == "" {
		urlPattern = "/authors/{author}/" // Default pattern
	}
	return &PageGenerator{
		urlPattern: urlPattern,
	}
}

// GenerateAuthorPage creates a pseudo-post for an author bio page
func (pg *PageGenerator) GenerateAuthorPage(author *models.Author, posts []*models.Post) *models.Post {
	// Generate URL from pattern
	url := strings.ReplaceAll(pg.urlPattern, "{author}", author.ID)

	// Split URL into path and href parts
	parts := strings.Split(url, "/")
	slug := parts[len(parts)-1]
	href := "/" + slug + "/"
	if slug == "" {
		href = "/" // Root author page
	}

	authorPost := &models.Post{
		Path:      "authors/" + author.ID,
		Slug:      slug,
		Href:      href,
		Title:     &author.Name,
		Published: true,
		Content:   "", // Will be populated by templates plugin
	}

	// Store author information in Extra for template access
	authorPost.Set("author", author)
	authorPost.Set("posts", posts)
	authorPost.Set("author_id", author.ID)

	return authorPost
}

// GenerateAuthorIndex creates an index page listing all authors
func (pg *PageGenerator) GenerateAuthorIndex(authors map[string]models.Author) *models.Post {
	indexPost := &models.Post{
		Path:      "authors/index",
		Slug:      "authors/index",
		Href:      "/authors/",
		Title:     AuthorsIndexTitle,
		Published: true,
		Content:   "", // Will be populated by templates plugin
	}

	// Store authors list in Extra for template access
	authorList := make([]*models.Author, 0, len(authors))
	for _, author := range authors {
		authorList = append(authorList, author)
	}
	indexPost.Set("authors", authorList)
	indexPost.Set("title", &AuthorsIndexTitle)

	return indexPost
}

// Constants for author page generation
const AuthorsIndexTitle = "Authors"
