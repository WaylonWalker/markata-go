package lifecycle

import (
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/WaylonWalker/markata-go/pkg/filter"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// Cache is an interface for caching data between stages.
// Implementations must be thread-safe for concurrent access.
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Delete(key string)
	Clear()
}

// memoryCache is a simple in-memory cache implementation.
type memoryCache struct {
	mu    sync.RWMutex
	items map[string]interface{}
}

func newMemoryCache() *memoryCache {
	return &memoryCache{
		items: make(map[string]interface{}),
	}
}

func (c *memoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.items[key]
	return v, ok
}

func (c *memoryCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value
}

func (c *memoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

func (c *memoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]interface{})
}

// Config holds markata configuration.
type Config struct {
	// ContentDir is the directory containing content files.
	ContentDir string

	// OutputDir is the directory for generated output.
	OutputDir string

	// GlobPatterns are patterns to match content files.
	GlobPatterns []string

	// Extra holds additional configuration options.
	Extra map[string]interface{}
}

// NewConfig creates a new Config with default values.
func NewConfig() *Config {
	return &Config{
		ContentDir:   ".",
		OutputDir:    "output",
		GlobPatterns: []string{"**/*.md"},
		Extra:        make(map[string]interface{}),
	}
}

// Feed represents a generated feed (RSS, Atom, etc.).
type Feed struct {
	// Name is the feed identifier.
	Name string

	// Title is the feed title.
	Title string

	// Posts are the posts included in the feed.
	Posts []*models.Post

	// Content is the generated feed content.
	Content string

	// Path is the output path for the feed.
	Path string
}

// Manager orchestrates the lifecycle stages and plugin execution.
type Manager struct {
	// plugins is the list of registered plugins.
	plugins []Plugin

	// config holds the markata configuration.
	config *Config

	// posts holds the processed posts.
	posts []*models.Post

	// files holds discovered content file paths.
	files []string

	// feeds holds generated feeds.
	feeds []*Feed

	// currentStage tracks the currently executing stage.
	currentStage Stage

	// stagesRun tracks which stages have completed.
	stagesRun map[Stage]bool

	// cache provides caching between stages.
	cache Cache

	// mu protects concurrent access to manager state.
	mu sync.RWMutex

	// warnings collects non-critical errors.
	warnings []*HookError

	// concurrency controls the number of concurrent goroutines for parallel processing.
	concurrency int
}

// NewManager creates a new lifecycle Manager with default settings.
// Concurrency is auto-detected from CPU cores, capped at 16.
func NewManager() *Manager {
	concurrency := runtime.NumCPU()
	if concurrency > 16 {
		concurrency = 16 // Cap to avoid excessive goroutine overhead
	}
	if concurrency < 1 {
		concurrency = 1
	}

	return &Manager{
		plugins:     make([]Plugin, 0),
		config:      NewConfig(),
		posts:       make([]*models.Post, 0),
		files:       make([]string, 0),
		feeds:       make([]*Feed, 0),
		stagesRun:   make(map[Stage]bool),
		cache:       newMemoryCache(),
		warnings:    make([]*HookError, 0),
		concurrency: concurrency,
	}
}

// RegisterPlugin adds a plugin to the manager.
// Plugins are executed in registration order within the same priority level.
func (m *Manager) RegisterPlugin(p Plugin) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plugins = append(m.plugins, p)
}

// RegisterPlugins adds multiple plugins to the manager.
func (m *Manager) RegisterPlugins(plugins ...Plugin) {
	for _, p := range plugins {
		m.RegisterPlugin(p)
	}
}

// Plugins returns a copy of the registered plugins.
func (m *Manager) Plugins() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Plugin, len(m.plugins))
	copy(result, m.plugins)
	return result
}

// Config returns the current configuration.
func (m *Manager) Config() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// SetConfig sets the configuration.
func (m *Manager) SetConfig(config *Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}

// Posts returns a copy of the posts slice.
func (m *Manager) Posts() []*models.Post {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.Post, len(m.posts))
	copy(result, m.posts)
	return result
}

// SetPosts sets the posts slice.
func (m *Manager) SetPosts(posts []*models.Post) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.posts = posts
}

// AddPost adds a post to the posts slice.
func (m *Manager) AddPost(post *models.Post) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.posts = append(m.posts, post)
}

// Files returns a copy of the discovered file paths.
func (m *Manager) Files() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]string, len(m.files))
	copy(result, m.files)
	return result
}

// SetFiles sets the discovered file paths.
func (m *Manager) SetFiles(files []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files = files
}

// AddFile adds a file path to the files slice.
func (m *Manager) AddFile(file string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files = append(m.files, file)
}

// Feeds returns a copy of the feeds slice.
func (m *Manager) Feeds() []*Feed {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*Feed, len(m.feeds))
	copy(result, m.feeds)
	return result
}

// SetFeeds sets the feeds slice.
func (m *Manager) SetFeeds(feeds []*Feed) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.feeds = feeds
}

// AddFeed adds a feed to the feeds slice.
func (m *Manager) AddFeed(feed *Feed) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.feeds = append(m.feeds, feed)
}

// Cache returns the cache instance.
func (m *Manager) Cache() Cache {
	return m.cache
}

// SetCache sets a custom cache implementation.
func (m *Manager) SetCache(cache Cache) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache = cache
}

// CurrentStage returns the currently executing stage.
func (m *Manager) CurrentStage() Stage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentStage
}

// HasRun returns true if the given stage has completed.
func (m *Manager) HasRun(stage Stage) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stagesRun[stage]
}

// Warnings returns collected non-critical errors.
func (m *Manager) Warnings() []*HookError {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*HookError, len(m.warnings))
	copy(result, m.warnings)
	return result
}

// SetConcurrency sets the concurrency level for parallel processing.
func (m *Manager) SetConcurrency(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n < 1 {
		n = 1
	}
	m.concurrency = n
}

// Concurrency returns the concurrency level.
func (m *Manager) Concurrency() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.concurrency
}

// Run executes all lifecycle stages in order.
func (m *Manager) Run() error {
	return m.RunTo(StageCleanup)
}

// RunTo executes lifecycle stages up to and including the specified stage.
// Already completed stages are skipped.
func (m *Manager) RunTo(stage Stage) error {
	if !IsValidStage(stage) {
		return fmt.Errorf("invalid stage: %s", stage)
	}

	stages := StagesUpTo(stage)

	for _, s := range stages {
		if m.HasRun(s) {
			continue
		}

		if err := m.runStage(s); err != nil {
			return err
		}
	}

	return nil
}

// runStage executes a single lifecycle stage.
func (m *Manager) runStage(stage Stage) error {
	m.mu.Lock()
	m.currentStage = stage
	m.mu.Unlock()

	var hookErrors *HookErrors

	switch stage {
	case StageConfigure:
		hookErrors = runConfigureHooks(m)
	case StageValidate:
		hookErrors = runValidateHooks(m)
	case StageGlob:
		hookErrors = runGlobHooks(m)
	case StageLoad:
		hookErrors = runLoadHooks(m)
	case StageTransform:
		hookErrors = runTransformHooks(m)
	case StageRender:
		hookErrors = runRenderHooks(m)
	case StageCollect:
		hookErrors = runCollectHooks(m)
	case StageWrite:
		hookErrors = runWriteHooks(m)
	case StageCleanup:
		hookErrors = runCleanupHooks(m)
	default:
		return fmt.Errorf("unknown stage: %s", stage)
	}

	// Collect warnings
	m.mu.Lock()
	for _, err := range hookErrors.Errors {
		if !err.Critical {
			m.warnings = append(m.warnings, err)
		}
	}
	m.mu.Unlock()

	// Return error if any critical errors occurred
	if hookErrors.HasCritical() {
		return hookErrors
	}

	// Mark stage as complete
	m.mu.Lock()
	m.stagesRun[stage] = true
	m.mu.Unlock()

	return nil
}

// Reset clears the manager state, allowing stages to be run again.
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.posts = make([]*models.Post, 0)
	m.files = make([]string, 0)
	m.feeds = make([]*Feed, 0)
	m.stagesRun = make(map[Stage]bool)
	m.warnings = make([]*HookError, 0)
	m.currentStage = ""
	m.cache.Clear()
}

// Filter returns posts matching the given expression.
// The expression supports the AST-based filter syntax:
//   - "published == True" - field equals value
//   - "draft != True" - field not equals value
//   - "'go' in tags" - value in slice (or "tags contains go" for legacy syntax)
//   - "date <= today" - date comparisons with special values
//   - Multiple conditions can be combined with "and" or "or"
//   - Supports "not" for negation
func (m *Manager) Filter(expr string) ([]*models.Post, error) {
	if expr == "" {
		return m.Posts(), nil
	}

	// Parse once (AST-based)
	f, err := filter.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid filter expression: %w", err)
	}

	// Use MatchAllWithErrors to get both results and any evaluation errors
	posts := m.Posts()
	results, errs := f.MatchAllWithErrors(posts)
	if len(errs) > 0 {
		// Return first error encountered
		return nil, errs[0]
	}

	return results, nil
}

// Map extracts field values from posts, with optional filtering and sorting.
// Parameters:
//   - field: the field to extract (supports dot notation for nested fields)
//   - filterExpr: optional filter expression (same syntax as Filter)
//   - sortField: field to sort by (empty for no sorting)
//   - reverse: if true, sort in descending order
func (m *Manager) Map(field, filterExpr, sortField string, reverse bool) ([]interface{}, error) {
	posts, err := m.Filter(filterExpr)
	if err != nil {
		return nil, err
	}

	// Sort if requested
	if sortField != "" {
		sort.SliceStable(posts, func(i, j int) bool {
			vi := getPostField(posts[i], sortField)
			vj := getPostField(posts[j], sortField)
			cmp := compareValues(vi, vj)
			if reverse {
				return cmp > 0
			}
			return cmp < 0
		})
	}

	// Extract field values
	result := make([]interface{}, len(posts))
	for i, post := range posts {
		result[i] = getPostField(post, field)
	}

	return result, nil
}

// getPostField retrieves a field value from a post using reflection.
func getPostField(post *models.Post, field string) interface{} {
	// Check Extra fields first
	if post.Extra != nil {
		if v, ok := post.Extra[field]; ok {
			return v
		}
	}

	// Use reflection for struct fields
	v := reflect.ValueOf(post).Elem()
	t := v.Type()

	// Try to find field by name (case-insensitive)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if strings.EqualFold(f.Name, field) {
			fv := v.Field(i)
			if fv.Kind() == reflect.Ptr {
				if fv.IsNil() {
					return nil
				}
				return fv.Elem().Interface()
			}
			return fv.Interface()
		}
	}

	return nil
}

// compareValues compares two values for sorting.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareValues(a, b interface{}) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Compare strings
	as, aok := a.(string)
	bs, bok := b.(string)
	if aok && bok {
		return strings.Compare(as, bs)
	}

	// Compare as formatted strings
	return strings.Compare(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
}

// ProcessPostsConcurrently processes posts concurrently using a bounded worker pool.
// The worker pool is sized to Concurrency(), ensuring that regardless of post count,
// only a fixed number of goroutines are spawned. This eliminates scheduler overhead
// and memory churn for large builds.
//
// Error handling: If any post fails to process, the function continues processing
// remaining posts and returns an aggregated error containing the count of failures
// and the first error encountered.
func (m *Manager) ProcessPostsConcurrently(fn func(*models.Post) error) error {
	posts := m.Posts()
	if len(posts) == 0 {
		return nil
	}

	numWorkers := m.Concurrency()
	if numWorkers > len(posts) {
		numWorkers = len(posts)
	}

	jobs := make(chan *models.Post, len(posts))
	errCh := make(chan error, len(posts))

	var wg sync.WaitGroup

	// Start fixed number of workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for post := range jobs {
				if err := fn(post); err != nil {
					errCh <- fmt.Errorf("processing %s: %w", post.Path, err)
				}
			}
		}()
	}

	// Send all posts to the jobs channel
	for _, post := range posts {
		jobs <- post
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()
	close(errCh)

	// Collect errors
	errs := make([]error, 0)
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d posts failed to process; first error: %w", len(errs), errs[0])
	}

	return nil
}
