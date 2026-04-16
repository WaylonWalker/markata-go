package searchapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/search"
)

// Handler serves the bleve search API.
type Handler struct {
	posts       []*models.Post
	postsByPath map[string]*models.Post
	cacheDir    string
	config      Config
	mu          sync.RWMutex
	idx         *search.Index
}

// Config controls search API behavior.
type Config struct {
	DefaultLimit int
	MaxLimit     int
	DefaultFuzzy bool
	CORSOrigins  []string
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
func NewHandler(posts []*models.Post, cacheDir string, cfg Config) *Handler {
	return &Handler{
		posts:       posts,
		postsByPath: search.PostsByPath(posts),
		cacheDir:    cacheDir,
		config:      cfg,
	}
}

// UpdatePosts replaces the post list (e.g., after a rebuild).
func (h *Handler) UpdatePosts(posts []*models.Post) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.posts = posts
	h.postsByPath = search.PostsByPath(posts)
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

	q := r.URL.Query()
	queryStr := q.Get("q")
	if queryStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing required parameter: q"})
		return
	}

	// Parse options
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

	// Tag filtering
	if tags := q.Get("tags"); tags != "" {
		opts.Tags = strings.Split(tags, ",")
	}

	// Date filtering
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

	// Published filter (default: only published)
	published := true
	opts.Published = &published

	h.mu.RLock()
	posts := h.posts
	postsByPath := h.postsByPath
	h.mu.RUnlock()

	// Build/open index
	idx, err := search.BuildIfNeeded(h.cacheDir, posts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "search index unavailable"})
		return
	}
	defer idx.Close()

	results, err := idx.Search(queryStr, opts, postsByPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "search failed"})
		return
	}

	resp := SearchResponse{
		Query:   queryStr,
		Total:   len(results),
		Fuzzy:   fuzzy,
		Limit:   limit,
		Results: make([]SearchResult, len(results)),
	}

	for i, hit := range results {
		sr := SearchResult{
			Path:  hit.Post.Path,
			Slug:  hit.Post.Slug,
			Href:  hit.Post.Href,
			Tags:  hit.Post.Tags,
			Score: hit.Score,
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
		sr.Private = hit.Post.Private
		resp.Results[i] = sr
	}

	writeJSON(w, http.StatusOK, resp)
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
	_ = enc.Encode(v)
}
