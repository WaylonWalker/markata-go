package searchapi

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/search"
)

// Handler serves the bleve search API.
// The index is built lazily on first search and reused across requests.
// It is only rebuilt when posts change (detected via count + path hash).
type Handler struct {
	posts       []*models.Post
	postsByPath map[string]*models.Post
	cacheDir    string
	config      Config
	mu          sync.RWMutex
	idx         *search.Index
	postsHash   string // cheap fingerprint of current post set
	searchSem   chan struct{}
}

// Config controls search API behavior.
type Config struct {
	DefaultLimit int
	MaxLimit     int
	DefaultFuzzy bool
	CORSOrigins  []string
	IndexName    string // process-specific index name to avoid conflicts
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		DefaultLimit: 20,
		MaxLimit:     100,
		DefaultFuzzy: false,
		CORSOrigins:  []string{"*"},
	}
}

// SearchResponse is the JSON response envelope.
type SearchResponse struct {
	Query   string         `json:"query"`
	Total   int            `json:"total"`
	Fuzzy   bool           `json:"fuzzy"`
	Limit   int            `json:"limit"`
	Results []SearchResult `json:"results"`
}

// SearchResult is a single search hit in the API response.
type SearchResult struct {
	Title       string   `json:"title"`
	Path        string   `json:"path"`
	Slug        string   `json:"slug"`
	Href        string   `json:"href"`
	Description string   `json:"description,omitempty"`
	Date        string   `json:"date,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Score       float64  `json:"score"`
	WordCount   int      `json:"word_count,omitempty"`
	ReadTime    string   `json:"read_time,omitempty"`
	Private     bool     `json:"private,omitempty"`
}

// NewHandler creates a new search API handler.
// The bleve index is built lazily on the first search request,
// not at construction time, to avoid blocking server startup.
func NewHandler(posts []*models.Post, cacheDir string, cfg Config) *Handler {
	// Limit concurrent searches to prevent resource exhaustion.
	// Default to 4 concurrent searches; large sites (3000+ posts) with
	// bleve can use significant memory per query.
	maxConcurrent := 4
	return &Handler{
		posts:       posts,
		postsByPath: search.PostsByPath(posts),
		cacheDir:    cacheDir,
		config:      cfg,
		postsHash:   postsFingerprint(posts),
		searchSem:   make(chan struct{}, maxConcurrent),
	}
}

// UpdatePosts replaces the post list (e.g., after a rebuild).
// Only rebuilds the index if the post set actually changed.
func (h *Handler) UpdatePosts(posts []*models.Post) {
	newHash := postsFingerprint(posts)

	h.mu.Lock()
	defer h.mu.Unlock()

	if newHash == h.postsHash {
		return // nothing changed
	}

	h.posts = posts
	h.postsByPath = search.PostsByPath(posts)
	h.postsHash = newHash

	// Close old index so it rebuilds on next search
	if h.idx != nil {
		h.idx.Close()
		h.idx = nil
	}
}

// Close releases the underlying search index.
func (h *Handler) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.idx != nil {
		h.idx.Close()
		h.idx = nil
	}
}

// ServeHTTP handles GET requests for search.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS
	origin := r.Header.Get("Origin")
	if origin != "" && h.corsAllowed(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	queryStr := r.URL.Query().Get("q")
	if queryStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing required parameter: q"})
		return
	}

	opts, limit, fuzzy := h.parseQueryOptions(r)

	// Acquire search semaphore to limit concurrent searches
	select {
	case h.searchSem <- struct{}{}:
		defer func() { <-h.searchSem }()
	default:
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "too many concurrent searches, try again"})
		return
	}

	idx, postsByPath, err := h.getOrBuildIndex()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "search index unavailable"})
		return
	}

	results, err := idx.Search(queryStr, opts, postsByPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "search failed"})
		return
	}

	writeJSON(w, http.StatusOK, buildResponse(queryStr, results, fuzzy, limit))
}

// parseQueryOptions extracts search parameters from the HTTP request.
func (h *Handler) parseQueryOptions(r *http.Request) (search.QueryOptions, int, bool) {
	q := r.URL.Query()

	fuzzy := h.config.DefaultFuzzy
	if f := q.Get("fuzzy"); f != "" {
		fuzzy = f == "true" || f == "1"
	}

	limit := h.config.DefaultLimit
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > h.config.MaxLimit {
		limit = h.config.MaxLimit
	}

	opts := search.QueryOptions{
		Limit: limit,
		Fuzzy: fuzzy,
	}

	if tags := q.Get("tags"); tags != "" {
		opts.Tags = strings.Split(tags, ",")
	}
	if from := q.Get("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			opts.DateFrom = &t
		}
	}
	if to := q.Get("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			opts.DateTo = &t
		}
	}

	published := true
	opts.Published = &published

	return opts, limit, fuzzy
}

func buildResponse(queryStr string, results []search.Result, fuzzy bool, limit int) SearchResponse {
	resp := SearchResponse{
		Query:   queryStr,
		Total:   len(results),
		Fuzzy:   fuzzy,
		Limit:   limit,
		Results: make([]SearchResult, len(results)),
	}

	for i, hit := range results {
		sr := SearchResult{
			Path:    hit.Post.Path,
			Slug:    hit.Post.Slug,
			Href:    hit.Post.Href,
			Tags:    hit.Post.Tags,
			Score:   hit.Score,
			Private: hit.Post.Private,
		}
		if hit.Post.Title != nil {
			sr.Title = *hit.Post.Title
		}
		if hit.Post.Description != nil {
			sr.Description = *hit.Post.Description
		}
		if hit.Post.Date != nil {
			sr.Date = hit.Post.Date.Format(time.RFC3339)
		}
		if hit.Post.Extra != nil {
			if wc, ok := hit.Post.Extra["word_count"].(int); ok {
				sr.WordCount = wc
				sr.ReadTime = readTime(wc)
			}
		}
		resp.Results[i] = sr
	}
	return resp
}

// getOrBuildIndex returns the current index, building it if needed.
// The index is reused across requests and only rebuilt when posts change.
func (h *Handler) getOrBuildIndex() (*search.Index, map[string]*models.Post, error) {
	// Fast path: index already exists
	h.mu.RLock()
	if h.idx != nil {
		idx := h.idx
		postsByPath := h.postsByPath
		h.mu.RUnlock()
		return idx, postsByPath, nil
	}
	h.mu.RUnlock()

	// Slow path: build index
	h.mu.Lock()
	defer h.mu.Unlock()

	// Double-check after acquiring write lock
	if h.idx != nil {
		return h.idx, h.postsByPath, nil
	}

	idx, err := search.BuildIfNeededNamed(h.cacheDir, h.config.IndexName, h.posts)
	if err != nil {
		return nil, nil, err
	}
	h.idx = idx
	return h.idx, h.postsByPath, nil
}

func (h *Handler) corsAllowed(origin string) bool {
	for _, allowed := range h.config.CORSOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

func readTime(words int) string {
	minutes := words / 200
	if minutes < 1 {
		return "1 min"
	}
	return fmt.Sprintf("%d min", minutes)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	//nolint:errcheck // HTTP response writer errors are handled by the framework
	enc.Encode(v)
}

// postsFingerprint creates a cheap hash of the post set for change detection.
// Uses count + sorted paths — NOT content (content hashing is too expensive
// to run every 2 seconds on 3000+ posts).
func postsFingerprint(posts []*models.Post) string {
	if len(posts) == 0 {
		return "empty"
	}
	h := sha256.New()
	fmt.Fprintf(h, "n=%d\n", len(posts))
	paths := make([]string, len(posts))
	for i, p := range posts {
		paths[i] = p.Path
	}
	sort.Strings(paths)
	for _, p := range paths {
		fmt.Fprintln(h, p)
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
