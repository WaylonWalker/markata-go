// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// SearchcraftPlugin syncs posts to a Searchcraft Core instance after builds.
type SearchcraftPlugin struct {
	client *http.Client
}

// NewSearchcraftPlugin constructs a SearchcraftPlugin with a default HTTP client.
func NewSearchcraftPlugin() *SearchcraftPlugin {
	return &SearchcraftPlugin{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the plugin identifier.
func (p *SearchcraftPlugin) Name() string {
	return "searchcraft"
}

// Cleanup syncs changed documents with Searchcraft Core.
func (p *SearchcraftPlugin) Cleanup(m *lifecycle.Manager) error {
	cfg := m.Config()
	sc := getSearchcraftConfig(cfg)
	if !sc.IsEnabled() {
		return nil
	}
	if sc.IngestKey == "" {
		fmt.Println("[searchcraft] WARNING: ingest_key is not configured; skipping Searchcraft sync")
		return nil
	}
	if sc.Endpoint == "" {
		fmt.Println("[searchcraft] WARNING: endpoint is empty; skipping Searchcraft sync")
		return nil
	}
	if sc.SkipOnFastMode && isFastMode(cfg) {
		fmt.Println("[searchcraft] Skipping Searchcraft sync in fast mode")
		return nil
	}
	modelsCfg, _ := getModelsConfig(cfg)
	siteName := resolveSiteName(modelsCfg, cfg)
	index := sc.ResolvedIndex
	if index == "" {
		index = sc.ResolveIndexName(siteName)
	}
	if index == "" {
		fmt.Println("[searchcraft] WARNING: resolved index name is empty; skipping sync")
		return nil
	}
	posts := m.Posts()
	docs := make([]searchcraftDocument, 0, len(posts))
	for _, post := range posts {
		if !shouldIndexPost(post, sc) {
			continue
		}
		doc := buildSearchcraftDocument(post, modelsCfg, cfg, sc)
		if doc != nil {
			docs = append(docs, *doc)
		}
	}
	cachePath := searchcraftCachePath(cfg)
	cache, err := loadSearchcraftCache(cachePath)
	if err != nil {
		fmt.Printf("[searchcraft] WARNING: failed to load cache: %v\n", err)
		cache = newSearchcraftCache()
	}
	current := make(map[string]bool, len(docs))
	docsToIngest := make([]searchcraftDocument, 0, len(docs))
	now := time.Now()
	for _, doc := range docs {
		hash := computeDocumentHash(doc)
		current[doc.ID] = true
		if entry, ok := cache.Entries[doc.ID]; ok && entry.DocumentHash == hash {
			continue
		}
		docsToIngest = append(docsToIngest, doc)
		cache.Entries[doc.ID] = searchcraftCacheEntry{DocumentHash: hash, UpdatedAt: now}
	}
	deleted := []string{}
	if sc.DeleteMissing {
		for id := range cache.Entries {
			if !current[id] {
				deleted = append(deleted, id)
			}
		}
	}
	if len(docsToIngest) > 0 {
		if err := p.sendDocuments(index, sc, docsToIngest); err != nil {
			fmt.Printf("[searchcraft] WARNING: failed to ingest docs: %v\n", err)
		} else {
			fmt.Printf("[searchcraft] synced %d documents to %s\n", len(docsToIngest), index)
		}
	}
	if len(deleted) > 0 {
		removed := 0
		for _, id := range deleted {
			if err := p.deleteDocument(index, sc, id); err != nil {
				fmt.Printf("[searchcraft] WARNING: delete %s failed: %v\n", id, err)
				continue
			}
			delete(cache.Entries, id)
			removed++
		}
		if removed > 0 {
			fmt.Printf("[searchcraft] removed %d stale documents from %s\n", removed, index)
		}
	}
	if err := cache.Save(cachePath); err != nil {
		fmt.Printf("[searchcraft] WARNING: failed to save cache: %v\n", err)
	}
	return nil
}

// Priority ensures the plugin runs late in cleanup.
func (p *SearchcraftPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageCleanup {
		return lifecycle.PriorityLast
	}
	return lifecycle.PriorityDefault
}

func (p *SearchcraftPlugin) sendDocuments(index string, cfg models.SearchcraftConfig, docs []searchcraftDocument) error {
	if len(docs) == 0 {
		return nil
	}
	if err := p.ensureIndex(index, cfg); err != nil {
		return err
	}
	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	batchSize := cfg.BatchSizeOrDefault()
	for start := 0; start < len(docs); start += batchSize {
		end := start + batchSize
		if end > len(docs) {
			end = len(docs)
		}
		if err := p.postDocuments(endpoint, index, cfg, docs[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (p *SearchcraftPlugin) ensureIndex(index string, cfg models.SearchcraftConfig) error {
	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	urlPath := fmt.Sprintf("%s/index/%s", endpoint, url.PathEscape(index))
	req, err := http.NewRequest(http.MethodGet, urlPath, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", cfg.IngestKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 400 {
		return nil
	}
	if resp.StatusCode != http.StatusNotFound {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("searchcraft index check failed: %s", strings.TrimSpace(string(msg)))
	}

	payload := map[string]any{
		"override_if_exists": false,
		"index": map[string]any{
			"name":              index,
			"language":          "en",
			"auto_commit_delay": 1,
			"search_fields":     []string{"title", "summary", "body", "content", "tags", "authors"},
			"fields": map[string]any{
				"id":           map[string]any{"type": "text", "required": true, "stored": true, "indexed": false},
				"title":        map[string]any{"type": "text", "stored": true},
				"summary":      map[string]any{"type": "text", "stored": true},
				"body":         map[string]any{"type": "text", "stored": true},
				"content":      map[string]any{"type": "text", "stored": true},
				"tags":         map[string]any{"type": "text", "stored": true, "multi": true},
				"url":          map[string]any{"type": "text", "stored": true},
				"path":         map[string]any{"type": "text", "stored": true},
				"site":         map[string]any{"type": "text", "stored": true},
				"published_at": map[string]any{"type": "datetime", "stored": true, "indexed": true, "fast": true},
				"modified_at":  map[string]any{"type": "datetime", "stored": true, "indexed": true, "fast": true},
				"authors":      map[string]any{"type": "text", "stored": true, "multi": true},
				"template":     map[string]any{"type": "text", "stored": true},
				"feed":         map[string]any{"type": "text", "stored": true},
				"published":    map[string]any{"type": "bool", "stored": true, "fast": true},
				"draft":        map[string]any{"type": "bool", "stored": true, "fast": true},
				"private":      map[string]any{"type": "bool", "stored": true, "fast": true},
			},
			"weight_multipliers": map[string]float64{"title": 2.0, "summary": 1.5, "tags": 1.2, "body": 1.0, "content": 0.7},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	createReq, err := http.NewRequest(http.MethodPost, endpoint+"/index", bytes.NewReader(body))
	if err != nil {
		return err
	}
	createReq.Header.Set("Authorization", cfg.IngestKey)
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := p.client.Do(createReq)
	if err != nil {
		return err
	}
	defer createResp.Body.Close()
	if createResp.StatusCode >= 400 {
		msg, _ := io.ReadAll(createResp.Body)
		return fmt.Errorf("searchcraft index create failed: %s", strings.TrimSpace(string(msg)))
	}
	fmt.Printf("[searchcraft] created index %s\n", index)
	return nil
}

func (p *SearchcraftPlugin) postDocuments(endpoint, index string, cfg models.SearchcraftConfig, docs []searchcraftDocument) error {
	urlPath := fmt.Sprintf("%s/index/%s/documents", endpoint, url.PathEscape(index))
	body, err := json.Marshal(docs)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, urlPath, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", cfg.IngestKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("searchcraft ingest failed: %s", strings.TrimSpace(string(msg)))
	}
	return nil
}

func (p *SearchcraftPlugin) deleteDocument(index string, cfg models.SearchcraftConfig, id string) error {
	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	urlPath := fmt.Sprintf("%s/index/%s/documents/query", endpoint, url.PathEscape(index))
	escapedID := strings.ReplaceAll(strings.ReplaceAll(id, "\\", "\\\\"), "\"", "\\\"")
	payload := fmt.Sprintf(`{"query":{"exact":{"ctx":"id:%s"}}}`, escapedID)
	req, err := http.NewRequest(http.MethodDelete, urlPath, strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", cfg.IngestKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("searchcraft delete failed: %s", strings.TrimSpace(string(msg)))
	}
	return nil
}

func shouldIndexPost(post *models.Post, cfg models.SearchcraftConfig) bool {
	if post == nil {
		return false
	}
	if post.Skip {
		return false
	}
	if post.Private && !cfg.IncludePrivate {
		return false
	}
	if post.Draft && !cfg.IncludeDrafts {
		return false
	}
	if !post.Published && !cfg.IncludeDrafts {
		return false
	}
	return true
}

func buildSearchcraftDocument(post *models.Post, modelsCfg *models.Config, cfg *lifecycle.Config, sc models.SearchcraftConfig) *searchcraftDocument {
	if post == nil {
		return nil
	}
	id := post.Slug
	if id == "" {
		id = strings.TrimPrefix(post.Path, "./")
	}
	if id == "" {
		return nil
	}
	title := ""
	if post.Title != nil {
		title = *post.Title
	}
	summary := ""
	if post.Description != nil {
		summary = *post.Description
	}
	body := post.ArticleHTML
	content := post.Content
	tags := append([]string{}, post.Tags...)
	sort.Strings(tags)
	authors := post.GetAuthors()
	sort.Strings(authors)
	urlValue := buildPostURL(modelsCfg, cfg, post)
	siteName := resolveSiteName(modelsCfg, cfg)
	publishedAt := formatTime(post.Date)
	modifiedAt := formatTime(post.Modified)
	return &searchcraftDocument{
		ID:          id,
		Title:       title,
		Summary:     summary,
		Body:        body,
		Content:     content,
		Tags:        tags,
		URL:         urlValue,
		Path:        post.Href,
		Site:        siteName,
		PublishedAt: publishedAt,
		ModifiedAt:  modifiedAt,
		Authors:     authors,
		Template:    post.Template,
		Feed:        post.PrevNextFeed,
		Published:   post.Published,
		Draft:       post.Draft,
		Private:     post.Private,
	}
}

type searchcraftDocument struct {
	ID          string   `json:"id"`
	Title       string   `json:"title,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Body        string   `json:"body,omitempty"`
	Content     string   `json:"content,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	URL         string   `json:"url,omitempty"`
	Path        string   `json:"path,omitempty"`
	Site        string   `json:"site,omitempty"`
	PublishedAt string   `json:"published_at,omitempty"`
	ModifiedAt  string   `json:"modified_at,omitempty"`
	Authors     []string `json:"authors,omitempty"`
	Template    string   `json:"template,omitempty"`
	Feed        string   `json:"feed,omitempty"`
	Published   bool     `json:"published"`
	Draft       bool     `json:"draft"`
	Private     bool     `json:"private"`
}

func buildPostURL(modelsCfg *models.Config, cfg *lifecycle.Config, post *models.Post) string {
	base := ""
	if modelsCfg != nil && modelsCfg.URL != "" {
		base = strings.TrimRight(modelsCfg.URL, "/")
	} else if cfg != nil && cfg.Extra != nil {
		if val, ok := cfg.Extra["url"].(string); ok {
			base = strings.TrimRight(val, "/")
		}
	}
	if base == "" {
		return post.Href
	}
	return base + post.Href
}

func formatTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func computeDocumentHash(doc searchcraftDocument) string {
	h := sha256.New()
	writeString(h, doc.Title)
	writeString(h, doc.Summary)
	writeString(h, doc.Body)
	writeString(h, doc.Content)
	writeString(h, doc.URL)
	writeString(h, doc.Path)
	writeString(h, doc.Site)
	writeString(h, doc.PublishedAt)
	writeString(h, doc.ModifiedAt)
	for _, tag := range doc.Tags {
		writeString(h, tag)
	}
	for _, author := range doc.Authors {
		writeString(h, author)
	}
	writeBool(h, doc.Published)
	writeBool(h, doc.Draft)
	writeBool(h, doc.Private)
	writeString(h, doc.Template)
	writeString(h, doc.Feed)
	return hex.EncodeToString(h.Sum(nil))
}

func writeString(h hash.Hash, value string) {
	h.Write([]byte(value))
}

func writeBool(h hash.Hash, value bool) {
	if value {
		h.Write([]byte{1})
		return
	}
	h.Write([]byte{0})
}

type searchcraftCache struct {
	Entries map[string]searchcraftCacheEntry `json:"entries"`
}

type searchcraftCacheEntry struct {
	DocumentHash string    `json:"document_hash"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func newSearchcraftCache() *searchcraftCache {
	return &searchcraftCache{Entries: make(map[string]searchcraftCacheEntry)}
}

func loadSearchcraftCache(path string) (*searchcraftCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newSearchcraftCache(), nil
		}
		return nil, err
	}
	var cache searchcraftCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return newSearchcraftCache(), nil
	}
	if cache.Entries == nil {
		cache.Entries = make(map[string]searchcraftCacheEntry)
	}
	return &cache, nil
}

func (c *searchcraftCache) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func searchcraftCachePath(cfg *lifecycle.Config) string {
	output := "output"
	if cfg != nil && cfg.OutputDir != "" {
		output = cfg.OutputDir
	}
	return filepath.Join(output, "..", ".markata", "searchcraft-cache.json")
}

func getSearchcraftConfig(cfg *lifecycle.Config) models.SearchcraftConfig {
	if cfg != nil && cfg.Extra != nil {
		if sc, ok := cfg.Extra["searchcraft"].(models.SearchcraftConfig); ok {
			return sc
		}
	}
	return models.NewSearchcraftConfig()
}

func resolveSiteName(modelsCfg *models.Config, cfg *lifecycle.Config) string {
	if modelsCfg != nil && modelsCfg.Title != "" {
		return modelsCfg.Title
	}
	if cfg != nil && cfg.Extra != nil {
		if title, ok := cfg.Extra["title"].(string); ok {
			return title
		}
	}
	return "site"
}

func isFastMode(cfg *lifecycle.Config) bool {
	if cfg == nil || cfg.Extra == nil {
		return false
	}
	if fast, ok := cfg.Extra["fast_mode"].(bool); ok {
		return fast
	}
	return false
}
