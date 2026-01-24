package services

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// postService implements PostService using lifecycle.Manager.
type postService struct {
	manager *lifecycle.Manager
}

// newPostService creates a new PostService.
func newPostService(m *lifecycle.Manager) PostService {
	return &postService{manager: m}
}

// List returns posts matching the given options.
func (s *postService) List(_ context.Context, opts ListOptions) ([]*models.Post, error) {
	posts := s.manager.Posts()

	// Apply filter expression
	if opts.Filter != "" {
		filtered, err := s.manager.Filter(opts.Filter)
		if err != nil {
			return nil, err
		}
		posts = filtered
	}

	// Apply tag filter (AND logic)
	if len(opts.Tags) > 0 {
		posts = filterByTags(posts, opts.Tags)
	}

	// Apply published filter
	if opts.Published != nil {
		posts = filterByPublished(posts, *opts.Published)
	}

	// Apply draft filter
	if opts.Draft != nil {
		posts = filterByDraft(posts, *opts.Draft)
	}

	// Apply date range filter
	if opts.DateRange != nil {
		posts = filterByDateRange(posts, opts.DateRange)
	}

	// Apply sorting
	if opts.SortBy != "" {
		sortPosts(posts, opts.SortBy, opts.SortOrder)
	}

	// Apply pagination
	if opts.Offset > 0 && opts.Offset < len(posts) {
		posts = posts[opts.Offset:]
	}
	if opts.Limit > 0 && opts.Limit < len(posts) {
		posts = posts[:opts.Limit]
	}

	return posts, nil
}

// Get returns a single post by path.
func (s *postService) Get(_ context.Context, path string) (*models.Post, error) {
	posts := s.manager.Posts()
	for _, p := range posts {
		if p.Path == path {
			return p, nil
		}
	}
	return nil, nil
}

// Search returns posts matching a text query.
func (s *postService) Search(_ context.Context, query string, opts SearchOptions) ([]*models.Post, error) {
	posts := s.manager.Posts()
	query = strings.ToLower(query)

	var results []*models.Post
	for _, p := range posts {
		if matchesSearch(p, query, opts) {
			results = append(results, p)
		}
	}

	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}

// Count returns the total number of posts matching options.
func (s *postService) Count(ctx context.Context, opts ListOptions) (int, error) {
	posts, err := s.List(ctx, opts)
	if err != nil {
		return 0, err
	}
	return len(posts), nil
}

// Helper functions

func filterByTags(posts []*models.Post, tags []string) []*models.Post {
	var result []*models.Post
	for _, p := range posts {
		if hasAllTags(p.Tags, tags) {
			result = append(result, p)
		}
	}
	return result
}

func hasAllTags(postTags, required []string) bool {
	tagSet := make(map[string]bool)
	for _, t := range postTags {
		tagSet[strings.ToLower(t)] = true
	}
	for _, req := range required {
		if !tagSet[strings.ToLower(req)] {
			return false
		}
	}
	return true
}

func filterByPublished(posts []*models.Post, published bool) []*models.Post {
	var result []*models.Post
	for _, p := range posts {
		if p.Published == published {
			result = append(result, p)
		}
	}
	return result
}

func filterByDraft(posts []*models.Post, draft bool) []*models.Post {
	var result []*models.Post
	for _, p := range posts {
		if p.Draft == draft {
			result = append(result, p)
		}
	}
	return result
}

func filterByDateRange(posts []*models.Post, dr *DateRange) []*models.Post {
	result := make([]*models.Post, 0, len(posts))
	for _, p := range posts {
		if p.Date == nil {
			continue
		}
		if dr.Start != nil && p.Date.Before(*dr.Start) {
			continue
		}
		if dr.End != nil && p.Date.After(*dr.End) {
			continue
		}
		result = append(result, p)
	}
	return result
}

func sortPosts(posts []*models.Post, field string, order SortOrder) {
	sort.SliceStable(posts, func(i, j int) bool {
		var cmp int
		switch strings.ToLower(field) {
		case "date":
			cmp = compareDates(posts[i].Date, posts[j].Date)
		case "title":
			cmp = compareTitles(posts[i].Title, posts[j].Title)
		case "path":
			cmp = strings.Compare(posts[i].Path, posts[j].Path)
		case "words":
			cmp = compareWordCounts(posts[i], posts[j])
		default:
			return false
		}
		if order == SortDesc {
			return cmp > 0
		}
		return cmp < 0
	})
}

func compareDates(a, b *time.Time) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	if a.Before(*b) {
		return -1
	}
	if a.After(*b) {
		return 1
	}
	return 0
}

func compareTitles(a, b *string) int {
	as, bs := "", ""
	if a != nil {
		as = *a
	}
	if b != nil {
		bs = *b
	}
	return strings.Compare(strings.ToLower(as), strings.ToLower(bs))
}

func compareWordCounts(a, b *models.Post) int {
	wcA := getWordCount(a)
	wcB := getWordCount(b)
	if wcA < wcB {
		return -1
	}
	if wcA > wcB {
		return 1
	}
	return 0
}

func getWordCount(p *models.Post) int {
	if p.Extra == nil {
		return 0
	}
	if wc, ok := p.Extra["word_count"].(int); ok {
		return wc
	}
	return 0
}

func matchesSearch(p *models.Post, query string, _ SearchOptions) bool {
	// Search in title
	if p.Title != nil && strings.Contains(strings.ToLower(*p.Title), query) {
		return true
	}
	// Search in description
	if p.Description != nil && strings.Contains(strings.ToLower(*p.Description), query) {
		return true
	}
	// Search in content
	if strings.Contains(strings.ToLower(p.Content), query) {
		return true
	}
	// Search in tags
	for _, tag := range p.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}
