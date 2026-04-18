// Package search provides full-text search using bleve for ranked results.
//
// # Index Management
//
// The Index type wraps a bleve index with markdown-post-aware document
// mapping. Indexes are persisted to disk and rebuilt only when content
// changes (detected via content hash).
//
// # Usage
//
//	idx, err := search.Open(dir)
//	if err != nil {
//	    idx, err = search.Build(dir, posts)
//	}
//	results, err := idx.Search("golang", search.QueryOptions{Limit: 10})
package search
