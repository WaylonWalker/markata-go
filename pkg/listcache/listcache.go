package listcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/plugins"
)

const (
	CacheVersion    = 1
	DefaultCacheDir = ".markata/cache"
	CacheFileName   = "list.json"
)

type Options struct {
	CacheDir   string
	ConfigHash string
}

type Cache struct {
	Version      int                   `json:"version"`
	ConfigHash   string                `json:"config_hash"`
	GeneratedAt  time.Time             `json:"generated_at"`
	ContentDir   string                `json:"content_dir"`
	GlobPatterns []string              `json:"glob_patterns"`
	Files        map[string]FileInfo   `json:"files"`
	Posts        map[string]CachedPost `json:"posts"`
	Feeds        []CachedFeed          `json:"feeds"`
}

type FileInfo struct {
	ModTime int64 `json:"mod_time"`
	Size    int64 `json:"size"`
}

type CachedPost struct {
	Path        string            `json:"path"`
	Content     string            `json:"content"`
	Slug        string            `json:"slug"`
	Href        string            `json:"href"`
	Title       *string           `json:"title,omitempty"`
	Date        *time.Time        `json:"date,omitempty"`
	Published   bool              `json:"published"`
	Draft       bool              `json:"draft"`
	Private     bool              `json:"private"`
	Skip        bool              `json:"skip"`
	Tags        []string          `json:"tags,omitempty"`
	Description *string           `json:"description,omitempty"`
	Template    string            `json:"template"`
	Templates   map[string]string `json:"templates,omitempty"`
	Authors     []string          `json:"authors,omitempty"`
	Author      *string           `json:"author,omitempty"`
	SecretKey   string            `json:"secret_key,omitempty"`
	Extra       map[string]any    `json:"extra,omitempty"`
	WordCount   int               `json:"word_count"`
	ReadingTime int               `json:"reading_time"`
	CharCount   int               `json:"char_count"`
}

type CachedFeed struct {
	Name      string   `json:"name"`
	Title     string   `json:"title"`
	Path      string   `json:"path"`
	PostPaths []string `json:"post_paths"`
}

func SetOptions(m *lifecycle.Manager, opts Options) {
	m.Cache().Set("list_cache_options", opts)
}

func OptionsFromManager(m *lifecycle.Manager) (Options, bool) {
	if cached, ok := m.Cache().Get("list_cache_options"); ok {
		if opts, ok := cached.(Options); ok {
			return opts, true
		}
	}
	return Options{}, false
}

func LoadOrRefresh(ctx context.Context, m *lifecycle.Manager, opts Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	cacheDir := opts.CacheDir
	if cacheDir == "" {
		cacheDir = DefaultCacheDir
	}

	cachePath := filepath.Join(cacheDir, CacheFileName)
	cache, err := loadCache(cachePath)
	if err != nil {
		return err
	}

	contentDir := m.Config().ContentDir
	if contentDir == "" {
		contentDir = "."
	}

	files, err := discoverFiles(m)
	if err != nil {
		return err
	}

	if cache.Version != CacheVersion || cache.ConfigHash != opts.ConfigHash {
		cache = newCache(opts.ConfigHash, contentDir, m.Config().GlobPatterns)
	}

	currentFiles, changedFiles, err := diffFiles(files, contentDir, cache.Files)
	if err != nil {
		return err
	}

	postsByPath := make(map[string]*models.Post, len(files))
	for _, file := range files {
		if cached, ok := cache.Posts[file]; ok && !changedFiles[file] {
			postsByPath[file] = cachedPostToModel(cached)
		}
	}

	if len(changedFiles) > 0 {
		updated, err := loadChangedPosts(contentDir, changedFiles)
		if err != nil {
			return err
		}
		applyTransforms(m.Config(), updated)
		for _, post := range updated {
			postsByPath[post.Path] = post
		}
	}

	posts := make([]*models.Post, 0, len(files))
	for _, file := range files {
		if post, ok := postsByPath[file]; ok {
			posts = append(posts, post)
		}
	}

	m.SetPosts(posts)
	useCachedFeeds := len(changedFiles) == 0 && cache.ConfigHash == opts.ConfigHash && len(cache.Feeds) > 0
	if useCachedFeeds {
		m.SetFeeds(cachedFeedsToModel(cache.Feeds, postsByPath))
	} else {
		if err := rebuildFeeds(m); err != nil {
			return err
		}
	}

	cache.Files = currentFiles
	cache.Posts = make(map[string]CachedPost, len(posts))
	for _, post := range posts {
		cache.Posts[post.Path] = modelToCachedPost(post)
	}
	cache.Feeds = modelToCachedFeeds(m.Feeds())
	cache.GeneratedAt = time.Now()
	cache.ContentDir = contentDir
	cache.GlobPatterns = append([]string{}, m.Config().GlobPatterns...)

	return saveCache(cachePath, cache)
}

func newCache(configHash, contentDir string, patterns []string) Cache {
	return Cache{
		Version:      CacheVersion,
		ConfigHash:   configHash,
		GeneratedAt:  time.Time{},
		ContentDir:   contentDir,
		GlobPatterns: append([]string{}, patterns...),
		Files:        make(map[string]FileInfo),
		Posts:        make(map[string]CachedPost),
		Feeds:        []CachedFeed{},
	}
}

func loadCache(path string) (Cache, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Cache{}, nil
		}
		return Cache{}, fmt.Errorf("read list cache: %w", err)
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	dec.UseNumber()

	var cache Cache
	if err := dec.Decode(&cache); err != nil && !errors.Is(err, io.EOF) {
		return Cache{}, fmt.Errorf("decode list cache: %w", err)
	}

	if cache.Files == nil {
		cache.Files = make(map[string]FileInfo)
	}
	if cache.Posts == nil {
		cache.Posts = make(map[string]CachedPost)
	}
	if cache.Feeds == nil {
		cache.Feeds = []CachedFeed{}
	}

	return cache, nil
}

func saveCache(path string, cache Cache) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create list cache dir: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("write list cache: %w", err)
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cache); err != nil {
		return fmt.Errorf("encode list cache: %w", err)
	}
	return nil
}

func discoverFiles(m *lifecycle.Manager) ([]string, error) {
	glob := plugins.NewGlobPlugin()
	if err := glob.Configure(m); err != nil {
		return nil, err
	}
	if err := glob.Glob(m); err != nil {
		return nil, err
	}
	return m.Files(), nil
}

func diffFiles(files []string, contentDir string, cached map[string]FileInfo) (map[string]FileInfo, map[string]bool, error) {
	current := make(map[string]FileInfo, len(files))
	changed := make(map[string]bool)

	for _, file := range files {
		fullPath := filepath.Join(contentDir, file)
		stat, err := os.Stat(fullPath)
		if err != nil {
			return nil, nil, fmt.Errorf("stat %s: %w", file, err)
		}

		info := FileInfo{ModTime: stat.ModTime().UnixNano(), Size: stat.Size()}
		current[file] = info
		if cachedInfo, ok := cached[file]; !ok || cachedInfo.ModTime != info.ModTime || cachedInfo.Size != info.Size {
			changed[file] = true
		}
	}

	return current, changed, nil
}

func loadChangedPosts(contentDir string, changed map[string]bool) ([]*models.Post, error) {
	posts := make([]*models.Post, 0, len(changed))
	for path := range changed {
		fullPath := filepath.Join(contentDir, path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		post, err := plugins.ParsePostFromContent(path, string(content))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func applyTransforms(cfg *lifecycle.Config, posts []*models.Post) {
	if len(posts) == 0 {
		return
	}

	m := lifecycle.NewManager()
	m.SetConfig(cfg)
	m.SetPosts(posts)

	autoTitle := plugins.NewAutoTitlePlugin()
	_ = autoTitle.Transform(m)

	description := plugins.NewDescriptionPlugin()
	_ = description.Configure(m)
	_ = description.Transform(m)

	stats := plugins.NewStatsPlugin()
	_ = stats.Configure(m)
	_ = stats.Transform(m)
}

func rebuildFeeds(m *lifecycle.Manager) error {
	baseFeeds := baseFeedConfigs(m)
	if baseFeeds != nil {
		m.Config().Extra["feeds"] = baseFeeds
	}

	if err := plugins.NewSeriesPlugin().Collect(m); err != nil {
		return err
	}
	if err := plugins.NewAutoFeedsPlugin().Collect(m); err != nil {
		return err
	}
	if err := plugins.NewFeedsPlugin().Collect(m); err != nil {
		return err
	}
	return nil
}

func baseFeedConfigs(m *lifecycle.Manager) []models.FeedConfig {
	if cached, ok := m.Cache().Get("list_cache_base_feeds"); ok {
		if feeds, ok := cached.([]models.FeedConfig); ok {
			return cloneFeedConfigs(feeds)
		}
	}

	feedsVal, ok := m.Config().Extra["feeds"].([]models.FeedConfig)
	if !ok {
		return nil
	}
	clone := cloneFeedConfigs(feedsVal)
	m.Cache().Set("list_cache_base_feeds", clone)
	return cloneFeedConfigs(clone)
}

func cloneFeedConfigs(feeds []models.FeedConfig) []models.FeedConfig {
	if feeds == nil {
		return nil
	}
	clone := make([]models.FeedConfig, len(feeds))
	for i := range feeds {
		clone[i] = feeds[i]
	}
	return clone
}

func modelToCachedPost(post *models.Post) CachedPost {
	return CachedPost{
		Path:        post.Path,
		Content:     post.Content,
		Slug:        post.Slug,
		Href:        post.Href,
		Title:       post.Title,
		Date:        post.Date,
		Published:   post.Published,
		Draft:       post.Draft,
		Private:     post.Private,
		Skip:        post.Skip,
		Tags:        append([]string{}, post.Tags...),
		Description: post.Description,
		Template:    post.Template,
		Templates:   cloneStringMap(post.Templates),
		Authors:     append([]string{}, post.Authors...),
		Author:      post.Author,
		SecretKey:   post.SecretKey, // pragma: allowlist secret
		Extra:       cloneAnyMap(post.Extra),
		WordCount:   getExtraInt(post.Extra, "word_count"),
		ReadingTime: getExtraInt(post.Extra, "reading_time"),
		CharCount:   getExtraInt(post.Extra, "char_count"),
	}
}

func cachedPostToModel(cached CachedPost) *models.Post {
	post := models.NewPost(cached.Path)
	post.Content = cached.Content
	post.Slug = cached.Slug
	post.Href = cached.Href
	post.Title = cached.Title
	post.Date = cached.Date
	post.Published = cached.Published
	post.Draft = cached.Draft
	post.Private = cached.Private
	post.Skip = cached.Skip
	post.Tags = append([]string{}, cached.Tags...)
	post.Description = cached.Description
	post.Template = cached.Template
	post.Templates = cloneStringMap(cached.Templates)
	post.Authors = append([]string{}, cached.Authors...)
	post.Author = cached.Author
	post.SecretKey = cached.SecretKey // pragma: allowlist secret
	post.Extra = normalizeExtraMap(cached.Extra)
	post.Set("word_count", cached.WordCount)
	post.Set("reading_time", cached.ReadingTime)
	post.Set("char_count", cached.CharCount)
	return post
}

func modelToCachedFeeds(feeds []*lifecycle.Feed) []CachedFeed {
	result := make([]CachedFeed, 0, len(feeds))
	for _, feed := range feeds {
		paths := make([]string, 0, len(feed.Posts))
		for _, post := range feed.Posts {
			paths = append(paths, post.Path)
		}
		result = append(result, CachedFeed{
			Name:      feed.Name,
			Title:     feed.Title,
			Path:      feed.Path,
			PostPaths: paths,
		})
	}
	return result
}

func cachedFeedsToModel(feeds []CachedFeed, postsByPath map[string]*models.Post) []*lifecycle.Feed {
	result := make([]*lifecycle.Feed, 0, len(feeds))
	for _, feed := range feeds {
		posts := make([]*models.Post, 0, len(feed.PostPaths))
		for _, path := range feed.PostPaths {
			if post, ok := postsByPath[path]; ok {
				posts = append(posts, post)
			}
		}
		result = append(result, &lifecycle.Feed{
			Name:  feed.Name,
			Title: feed.Title,
			Posts: posts,
			Path:  feed.Path,
		})
	}
	return result
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	clone := make(map[string]string, len(in))
	for k, v := range in {
		clone[k] = v
	}
	return clone
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	clone := make(map[string]any, len(in))
	for k, v := range in {
		clone[k] = v
	}
	return clone
}

func normalizeExtraMap(in map[string]any) map[string]any {
	if in == nil {
		return make(map[string]any)
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = normalizeValue(v)
	}
	return out
}

func normalizeValue(value any) any {
	switch v := value.(type) {
	case json.Number:
		if strings.Contains(v.String(), ".") {
			if f, err := v.Float64(); err == nil {
				return f
			}
			return v.String()
		}
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
		return v.String()
	case map[string]any:
		return normalizeExtraMap(v)
	case []any:
		items := make([]any, len(v))
		for i := range v {
			items[i] = normalizeValue(v[i])
		}
		return items
	default:
		return value
	}
}

func getExtraInt(extra map[string]any, key string) int {
	if extra == nil {
		return 0
	}
	val, ok := extra[key]
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
	}
	return 0
}
