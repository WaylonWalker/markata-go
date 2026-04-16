package search

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	bleve "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

// Index wraps a bleve index with post-aware operations.
type Index struct {
	idx bleve.Index
	dir string
	mu  sync.RWMutex
}

const analyzerKeyword = "keyword"

// Result represents a single search hit with relevance score.
type Result struct {
	Post  *models.Post
	Score float64
}

// QueryOptions configures a search query.
type QueryOptions struct {
	Limit     int
	Fuzzy     bool
	Fuzziness int
	Fields    []string // empty = all fields
	DateFrom  *time.Time
	DateTo    *time.Time
	Tags      []string
	Published *bool
}

// postDoc is the bleve-indexed document structure.
type postDoc struct {
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Path        string    `json:"path"`
	Slug        string    `json:"slug"`
	Date        time.Time `json:"date"`
	Published   bool      `json:"published"`
	WordCount   int       `json:"word_count"`
}

const (
	indexDir  = "search.bleve"
	hashFile  = "search.hash"
	batchSize = 500
)

// DefaultDir returns the default index directory inside the cache dir.
func DefaultDir(cacheDir string) string {
	return filepath.Join(cacheDir, indexDir)
}

// Open opens an existing bleve index. Returns an error if no index exists.
func Open(dir string) (*Index, error) {
	idx, err := bleve.Open(dir)
	if err != nil {
		return nil, fmt.Errorf("open search index: %w", err)
	}
	return &Index{idx: idx, dir: dir}, nil
}

// Build creates a new bleve index from posts.
// If dir already exists, it is removed and rebuilt.
func Build(dir string, posts []*models.Post) (*Index, error) {
	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf("remove old index: %w", err)
	}

	m := buildMapping()
	idx, err := bleve.New(dir, m)
	if err != nil {
		return nil, fmt.Errorf("create search index: %w", err)
	}

	si := &Index{idx: idx, dir: dir}
	if err := si.indexPosts(posts); err != nil {
		idx.Close()
		return nil, err
	}

	// Index synonym definitions (best-effort — search works without them)
	if synErr := indexSynonyms(idx); synErr != nil {
		fmt.Fprintf(os.Stderr, "warning: synonym indexing failed: %v\n", synErr)
	}

	return si, nil
}

// BuildIfNeeded creates or opens an index, rebuilding only if the content hash changed.
func BuildIfNeeded(cacheDir string, posts []*models.Post) (*Index, error) {
	dir := DefaultDir(cacheDir)
	currentHash := ContentHash(posts)
	storedHash := readHashFile(filepath.Join(cacheDir, hashFile))

	if currentHash == storedHash {
		si, err := Open(dir)
		if err == nil {
			return si, nil
		}
		// Index corrupted or missing — fall through to rebuild
	}

	si, err := Build(dir, posts)
	if err != nil {
		return nil, err
	}

	// Persist hash for next run (best-effort, non-fatal)
	if mkErr := os.MkdirAll(cacheDir, 0o755); mkErr == nil {
		_ = os.WriteFile(filepath.Join(cacheDir, hashFile), []byte(currentHash), 0o600) //nolint:errcheck // best-effort cache
	}
	return si, nil
}

// Close closes the index.
func (si *Index) Close() error {
	si.mu.Lock()
	defer si.mu.Unlock()
	return si.idx.Close()
}

// Search performs a ranked full-text search.
func (si *Index) Search(query string, opts QueryOptions, postsByPath map[string]*models.Post) ([]Result, error) {
	si.mu.RLock()
	defer si.mu.RUnlock()

	q := buildQuery(query, opts)

	req := bleve.NewSearchRequest(q)
	if opts.Limit > 0 {
		req.Size = opts.Limit
	} else {
		req.Size = 1000
	}
	req.Fields = []string{"path"}

	searchResult, err := si.idx.Search(req)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	results := make([]Result, 0, len(searchResult.Hits))
	for _, hit := range searchResult.Hits {
		path, ok := hit.Fields["path"].(string)
		if !ok {
			continue
		}
		post := postsByPath[path]
		if post == nil {
			continue
		}
		results = append(results, Result{Post: post, Score: hit.Score})
	}
	return results, nil
}

// indexPosts batch-indexes all posts into the bleve index.
func (si *Index) indexPosts(posts []*models.Post) error {
	batch := si.idx.NewBatch()
	for i, post := range posts {
		doc := toPostDoc(post)
		if err := batch.Index(post.Path, doc); err != nil {
			return fmt.Errorf("index post %s: %w", post.Path, err)
		}
		if (i+1)%batchSize == 0 {
			if err := si.idx.Batch(batch); err != nil {
				return fmt.Errorf("batch index: %w", err)
			}
			batch = si.idx.NewBatch()
		}
	}
	if batch.Size() > 0 {
		if err := si.idx.Batch(batch); err != nil {
			return fmt.Errorf("batch index: %w", err)
		}
	}
	return nil
}

func toPostDoc(p *models.Post) postDoc {
	doc := postDoc{
		Content:   p.Content,
		Tags:      p.Tags,
		Path:      p.Path,
		Slug:      p.Slug,
		Published: p.Published,
	}
	if p.Title != nil {
		doc.Title = *p.Title
	}
	if p.Description != nil {
		doc.Description = *p.Description
	}
	if p.Date != nil {
		doc.Date = *p.Date
	}
	if p.Extra != nil {
		if wc, ok := p.Extra["word_count"].(int); ok {
			doc.WordCount = wc
		}
	}
	return doc
}

func buildMapping() mapping.IndexMapping {
	im := bleve.NewIndexMapping()
	im.DefaultAnalyzer = "en"

	// Configure synonym source: link the "english" source to the "wordnet" collection
	// using the "en" analyzer for term normalization.
	err := im.AddSynonymSource(synonymSourceName, map[string]interface{}{
		"collection": synonymCollection,
		"analyzer":   synonymAnalyzer,
	})
	if err != nil {
		// Non-fatal: search works without synonyms
		fmt.Fprintf(os.Stderr, "warning: synonym source setup failed: %v\n", err)
	}

	dm := bleve.NewDocumentMapping()

	title := bleve.NewTextFieldMapping()
	title.Analyzer = "en"
	title.Store = false
	title.IncludeTermVectors = true
	title.SynonymSource = synonymSourceName
	dm.AddFieldMappingsAt("title", title)

	content := bleve.NewTextFieldMapping()
	content.Analyzer = "en"
	content.Store = false
	content.IncludeTermVectors = true
	content.SynonymSource = synonymSourceName
	dm.AddFieldMappingsAt("content", content)

	desc := bleve.NewTextFieldMapping()
	desc.Analyzer = "en"
	desc.Store = false
	desc.SynonymSource = synonymSourceName
	dm.AddFieldMappingsAt("description", desc)

	tags := bleve.NewTextFieldMapping()
	tags.Analyzer = analyzerKeyword
	tags.Store = false
	tags.DocValues = true
	dm.AddFieldMappingsAt("tags", tags)

	path := bleve.NewTextFieldMapping()
	path.Analyzer = analyzerKeyword
	path.Store = true
	path.Index = false
	dm.AddFieldMappingsAt("path", path)

	slug := bleve.NewTextFieldMapping()
	slug.Analyzer = analyzerKeyword
	slug.Store = false
	dm.AddFieldMappingsAt("slug", slug)

	date := bleve.NewDateTimeFieldMapping()
	date.Store = false
	date.DocValues = true
	dm.AddFieldMappingsAt("date", date)

	published := mapping.NewBooleanFieldMapping()
	published.Store = false
	published.DocValues = true
	dm.AddFieldMappingsAt("published", published)

	wordCount := bleve.NewNumericFieldMapping()
	wordCount.Store = false
	dm.AddFieldMappingsAt("word_count", wordCount)

	im.DefaultMapping = dm
	return im
}

// synonymFields are the fields with SynonymSource configured.
// MatchQuery must target specific fields for synonym expansion to work;
// the default "_all" field has no synonym source.
var synonymFields = []string{"title", "content", "description"}

func buildQuery(queryStr string, opts QueryOptions) query.Query {
	var textQuery query.Query
	if opts.Fuzzy {
		// Build per-field fuzzy queries so synonym expansion triggers
		fuzziness := opts.Fuzziness
		if fuzziness <= 0 {
			fuzziness = 1
		}
		dq := query.NewDisjunctionQuery(nil)
		for _, field := range synonymFields {
			mq := query.NewMatchQuery(queryStr)
			mq.SetFuzziness(fuzziness)
			mq.SetField(field)
			dq.AddQuery(mq)
		}
		textQuery = dq
	} else {
		// Build per-field match queries so synonym expansion triggers
		dq := query.NewDisjunctionQuery(nil)
		for _, field := range synonymFields {
			mq := query.NewMatchQuery(queryStr)
			mq.SetField(field)
			dq.AddQuery(mq)
		}
		textQuery = dq
	}

	// If no additional filters, return the text query directly
	if opts.DateFrom == nil && opts.DateTo == nil && len(opts.Tags) == 0 && opts.Published == nil {
		return textQuery
	}

	// Build boolean query with filters
	bq := query.NewBooleanQuery(nil, nil, nil)
	bq.AddMust(textQuery)

	if opts.DateFrom != nil || opts.DateTo != nil {
		var from, to time.Time
		if opts.DateFrom != nil {
			from = *opts.DateFrom
		}
		if opts.DateTo != nil {
			to = *opts.DateTo
		}
		dq := query.NewDateRangeQuery(from, to)
		dq.SetField("date")
		bq.AddMust(dq)
	}

	for _, tag := range opts.Tags {
		tq := query.NewTermQuery(tag)
		tq.SetField("tags")
		bq.AddMust(tq)
	}

	if opts.Published != nil {
		pq := query.NewBoolFieldQuery(*opts.Published)
		pq.SetField("published")
		bq.AddMust(pq)
	}

	return bq
}

// ContentHash computes a hash of all post paths and content for cache invalidation.
func ContentHash(posts []*models.Post) string {
	h := sha256.New()
	// Sort by path for deterministic hashing
	paths := make([]string, len(posts))
	for i, p := range posts {
		paths[i] = p.Path
	}
	sort.Strings(paths)

	contentMap := make(map[string]string, len(posts))
	for _, p := range posts {
		contentMap[p.Path] = p.Content
	}

	for _, path := range paths {
		fmt.Fprintf(h, "%s:%s\n", path, contentMap[path])
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func readHashFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// PostsByPath builds a lookup map from posts for result resolution.
func PostsByPath(posts []*models.Post) map[string]*models.Post {
	m := make(map[string]*models.Post, len(posts))
	for _, p := range posts {
		m[p.Path] = p
	}
	return m
}
