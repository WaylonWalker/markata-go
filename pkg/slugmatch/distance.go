// Package slugmatch provides utilities for finding similar slugs/URLs.
//
// # Overview
//
// This package implements string similarity algorithms to suggest
// alternative posts when a requested URL is not found (404 pages).
//
// # Usage
//
//	similar := slugmatch.FindSimilarSlugs("/posts/my-pst", posts, 5)
//	for _, post := range similar {
//	    fmt.Printf("Did you mean: %s?\n", post.Slug)
//	}
package slugmatch

import (
	"sort"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

// LevenshteinDistance calculates the minimum number of single-character edits
// (insertions, deletions, or substitutions) required to change one string into another.
//
// This is useful for finding "similar" strings when a user makes a typo in a URL.
//
// Example:
//
//	LevenshteinDistance("kitten", "sitting") // returns 3
//	LevenshteinDistance("hello", "hallo")    // returns 1
func LevenshteinDistance(a, b string) int {
	if a == "" {
		return len(b)
	}
	if b == "" {
		return len(a)
	}

	// Create a matrix to store distances
	// We only need two rows at a time to save memory
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)

	// Initialize the first row
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}

	// Fill in the matrix
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			// Minimum of: deletion, insertion, substitution
			curr[j] = min(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}
		// Swap rows
		prev, curr = curr, prev
	}

	return prev[len(b)]
}

// NormalizedDistance returns the Levenshtein distance normalized to [0, 1].
// 0 means identical strings, 1 means completely different.
func NormalizedDistance(a, b string) float64 {
	maxLen := max(len(a), len(b))
	if maxLen == 0 {
		return 0
	}
	return float64(LevenshteinDistance(a, b)) / float64(maxLen)
}

// slugMatch holds a post and its distance score for sorting.
type slugMatch struct {
	post     *models.Post
	distance int
}

// FindSimilarSlugs finds posts with slugs similar to the target path.
// It returns up to maxResults posts, sorted by similarity (most similar first).
//
// The target is typically a URL path like "/posts/my-post" and posts are
// compared by their Slug field.
//
// Example:
//
//	posts := manager.Posts()
//	similar := FindSimilarSlugs("/posts/my-pst", posts, 5)
func FindSimilarSlugs(target string, posts []*models.Post, maxResults int) []*models.Post {
	if len(posts) == 0 || maxResults <= 0 {
		return nil
	}

	// Normalize the target path
	target = normalizePath(target)

	// Calculate distances for all posts
	matches := make([]slugMatch, 0, len(posts))
	for _, post := range posts {
		if post == nil || post.Slug == "" {
			continue
		}

		// Compare against normalized slug
		slug := normalizePath(post.Slug)
		distance := LevenshteinDistance(target, slug)

		// Only include if reasonably similar (distance < half the target length)
		// This filters out completely unrelated posts
		maxDistance := max(len(target)/2, 5)
		if distance <= maxDistance {
			matches = append(matches, slugMatch{post: post, distance: distance})
		}
	}

	// Sort by distance (ascending)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].distance < matches[j].distance
	})

	// Return top results
	results := make([]*models.Post, 0, min(maxResults, len(matches)))
	for i := 0; i < len(matches) && i < maxResults; i++ {
		results = append(results, matches[i].post)
	}

	return results
}

// FindSimilarByTitle finds posts with titles similar to a search query.
// This is useful when the slug doesn't match but the user might be looking
// for a post by its title.
func FindSimilarByTitle(query string, posts []*models.Post, maxResults int) []*models.Post {
	if len(posts) == 0 || maxResults <= 0 || query == "" {
		return nil
	}

	// Normalize the query
	query = strings.ToLower(strings.TrimSpace(query))

	// Calculate distances for all posts
	matches := make([]slugMatch, 0, len(posts))
	for _, post := range posts {
		if post == nil {
			continue
		}

		title := ""
		if post.Title != nil {
			title = strings.ToLower(*post.Title)
		}
		if title == "" {
			continue
		}

		distance := LevenshteinDistance(query, title)

		// Only include if reasonably similar
		maxDistance := max(len(query)/2, 5)
		if distance <= maxDistance {
			matches = append(matches, slugMatch{post: post, distance: distance})
		}
	}

	// Sort by distance (ascending)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].distance < matches[j].distance
	})

	// Return top results
	results := make([]*models.Post, 0, min(maxResults, len(matches)))
	for i := 0; i < len(matches) && i < maxResults; i++ {
		results = append(results, matches[i].post)
	}

	return results
}

// normalizePath normalizes a URL path for comparison.
// It removes leading/trailing slashes, converts to lowercase,
// and removes common prefixes like "posts/".
func normalizePath(path string) string {
	// Remove leading/trailing slashes
	path = strings.Trim(path, "/")

	// Convert to lowercase
	path = strings.ToLower(path)

	// Remove common prefixes for better matching
	// e.g., "/posts/my-post" -> "my-post"
	prefixes := []string{"posts/", "blog/", "articles/", "pages/"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
			break
		}
	}

	return path
}
