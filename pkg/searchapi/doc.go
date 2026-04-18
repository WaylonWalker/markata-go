// Package searchapi provides a read-only HTTP API for bleve full-text search.
//
// # Privacy
//
// Private posts are indexed with a narrow safe subset.
// Only explicit frontmatter title, explicit frontmatter description, and navigational fields are exposed.
// Body content, tags, media, and derived content fields are never exposed by the search API.
// Draft and skipped posts are excluded entirely.
//
// # Endpoints
//
// GET /api/search?q=<query>&fuzzy=true&limit=20&tags=go,cli&from=2024-01-01&to=2024-12-31
//
// Response:
//
//	{
//	  "query": "search term",
//	  "total": 5,
//	  "results": [
//	    {
//	      "title": "Post Title",
//	      "path": "posts/example.md",
//	      "slug": "example",
//	      "href": "/example",
//	      "description": "A short description",
//	      "date": "2024-01-15T00:00:00Z",
//	      "tags": ["go", "cli"],
//	      "score": 1.234,
//	      "word_count": 500,
//	      "read_time": "3 min"
//	    }
//	  ]
//	}
package searchapi
