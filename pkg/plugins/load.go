package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// Pre-compiled regex patterns for date normalization.
// These are compiled once at package init instead of per-call.
var (
	// timeFixRegex fixes malformed time components like "8:011:00" -> "08:11:00"
	timeFixRegex = regexp.MustCompile(`(\d{1,2}):0*(\d{1,2}):0*(\d{1,2})`)

	// singleDigitHourRegex normalizes single-digit hours in time component
	singleDigitHourRegex = regexp.MustCompile(`([ T])(\d):(\d{2})`)

	// startSingleDigitHourRegex matches time at start of string
	startSingleDigitHourRegex = regexp.MustCompile(`^\d:\d{2}`)
)

// LoadPlugin parses markdown files into Post objects.
type LoadPlugin struct{}

// NewLoadPlugin creates a new LoadPlugin.
func NewLoadPlugin() *LoadPlugin {
	return &LoadPlugin{}
}

// Name returns the plugin identifier.
func (p *LoadPlugin) Name() string {
	return "load"
}

// Load reads and parses all discovered files into Post objects.
// Files are loaded in parallel using a worker pool for improved I/O performance.
// Uses ModTime-based caching to skip re-parsing unchanged files.
func (p *LoadPlugin) Load(m *lifecycle.Manager) error {
	files := m.Files()
	config := m.Config()
	baseDir := config.ContentDir
	if baseDir == "" {
		baseDir = "."
	}

	if cachedPosts := lifecycle.GetServeCachedPosts(m); len(cachedPosts) > 0 {
		return p.loadFromCachedPosts(m, files, baseDir, cachedPosts)
	}

	// Get build cache for ModTime-based skipping
	cache := GetBuildCache(m)
	if lifecycle.IsServeFullRebuild(m) || cache == nil {
		return p.loadAllFiles(m, files, baseDir, cache)
	}

	affected := lifecycle.GetServeAffectedPaths(m)
	if len(affected) == 0 {
		return p.loadAllFiles(m, files, baseDir, cache)
	}

	restored, _, err := p.restoreFromCacheOrLoadChanged(files, baseDir, cache, affected)
	if err != nil {
		return err
	}

	m.SetPosts(restored)
	return nil
}

func (p *LoadPlugin) loadFromCachedPosts(
	m *lifecycle.Manager,
	files []string,
	baseDir string,
	cachedPosts map[string]*models.Post,
) error {
	cache := GetBuildCache(m)
	posts := make([]*models.Post, 0, len(files))
	affected := lifecycle.GetServeAffectedPaths(m)
	useFastCache := lifecycle.IsServeFastMode(m) && len(affected) > 0
	for _, file := range files {
		post, err := p.loadCachedPost(file, baseDir, cachedPosts, cache, useFastCache, affected)
		if err != nil {
			return err
		}
		if post != nil {
			posts = append(posts, post)
		}
	}
	m.SetPosts(posts)
	return nil
}

func (p *LoadPlugin) loadCachedPost(
	file string,
	baseDir string,
	cachedPosts map[string]*models.Post,
	cache *buildcache.Cache,
	useFastCache bool,
	affected map[string]bool,
) (*models.Post, error) {
	if existing, ok := cachedPosts[file]; ok && !useFastCache {
		fullPath := file
		if !filepath.IsAbs(file) {
			fullPath = filepath.Join(baseDir, file)
		}
		stat, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to stat %s: %w", file, err)
		}
		if cache != nil {
			if cachedData := cache.GetCachedPostData(file, stat.ModTime().UnixNano()); cachedData != nil {
				return p.restorePostFromCache(cachedData, cache), nil
			}
		}
		return existing, nil
	}

	if useFastCache {
		if len(affected) > 0 && affected[file] {
			return p.loadFile(file, baseDir, cache)
		}
		if cache != nil {
			if cachedData := cache.GetCachedPostDataLatest(file); cachedData != nil {
				return p.restorePostFromCache(cachedData, cache), nil
			}
		}
	}

	return p.loadFile(file, baseDir, cache)
}

func (p *LoadPlugin) loadAllFiles(m *lifecycle.Manager, files []string, baseDir string, cache *buildcache.Cache) error {
	if cache == nil {
		return p.loadSequential(m, files, baseDir, cache)
	}

	posts := make([]*models.Post, 0, len(files))
	var firstErr error
	affected := lifecycle.GetServeAffectedPaths(m)
	useFastCache := lifecycle.IsServeFastMode(m) && len(affected) > 0

	for _, file := range files {
		if useFastCache {
			if len(affected) > 0 && affected[file] {
				post, err := p.loadFile(file, baseDir, cache)
				if err != nil {
					if firstErr == nil {
						firstErr = err
					}
					continue
				}
				posts = append(posts, post)
				continue
			}
			cachedData := cache.GetCachedPostDataLatest(file)
			if cachedData != nil {
				posts = append(posts, p.restorePostFromCache(cachedData, cache))
				continue
			}
		}

		fullPath := file
		if !filepath.IsAbs(file) {
			fullPath = filepath.Join(baseDir, file)
		}
		stat, err := os.Stat(fullPath)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to stat %s: %w", file, err)
			}
			continue
		}
		cachedData := cache.GetCachedPostData(file, stat.ModTime().UnixNano())
		if cachedData != nil {
			posts = append(posts, p.restorePostFromCache(cachedData, cache))
			continue
		}
		post, err := p.loadFile(file, baseDir, cache)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		posts = append(posts, post)
	}
	if firstErr != nil {
		return firstErr
	}
	for _, post := range posts {
		m.AddPost(post)
	}
	return nil
}

func (p *LoadPlugin) restoreFromCacheOrLoadChanged(
	files []string,
	baseDir string,
	cache *buildcache.Cache,
	affected map[string]bool,
) (posts, changedPosts []*models.Post, err error) {
	if cache == nil {
		return nil, nil, fmt.Errorf("build cache unavailable")
	}

	posts = make([]*models.Post, 0, len(files))
	var firstErr error

	for _, file := range files {
		if affected[file] {
			post, err := p.loadFile(file, baseDir, cache)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			posts = append(posts, post)
			changedPosts = append(changedPosts, post)
			continue
		}

		fullPath := file
		if !filepath.IsAbs(file) {
			fullPath = filepath.Join(baseDir, file)
		}
		stat, err := os.Stat(fullPath)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to stat %s: %w", file, err)
			}
			continue
		}
		modTime := stat.ModTime().UnixNano()
		cachedData := cache.GetCachedPostData(file, modTime)
		if cachedData == nil {
			post, err := p.loadFile(file, baseDir, cache)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			posts = append(posts, post)
			continue
		}
		posts = append(posts, p.restorePostFromCache(cachedData, cache))
	}

	if firstErr != nil {
		return nil, nil, firstErr
	}

	return posts, changedPosts, nil
}

// loadFile loads a single file, using cache if ModTime is unchanged.
func (p *LoadPlugin) loadFile(file, baseDir string, cache *buildcache.Cache) (*models.Post, error) {
	// Construct full path
	fullPath := file
	if !filepath.IsAbs(file) {
		fullPath = filepath.Join(baseDir, file)
	}

	// Stat file for ModTime
	stat, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %s: %w", file, err)
	}
	modTime := stat.ModTime().UnixNano()

	// Check cache for unchanged file
	if cache != nil {
		if cachedData := cache.GetCachedPostData(file, modTime); cachedData != nil {
			// Restore Post from cached data
			post := p.restorePostFromCache(cachedData, cache)
			return post, nil
		}
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", file, err)
	}

	// Parse the file
	post, err := p.parseFile(file, string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", file, err)
	}

	// Cache the parsed post
	if cache != nil {
		postData := p.postToCachedData(post)
		//nolint:errcheck // caching is best-effort
		cache.CachePostData(file, modTime, postData)
		contentHash := buildcache.ContentHash(post.Content)
		//nolint:errcheck // caching is best-effort
		cache.CacheArticleHTML(file, contentHash, post.ArticleHTML)
		cache.SetPostSlug(file, post.Slug)
	}

	if cache != nil {
		feedHash := computePostFeedItemHash(post)
		tagHash := computePostTagIndexHash(post)
		gardenHash := computePostGardenHash(post)
		feedChanged, _, _ := cache.UpdatePostSemanticHashes(file, feedHash, tagHash, gardenHash)
		if feedChanged {
			cache.MarkFeedSlugChanged(post.Slug)
		}
	}

	return post, nil
}

// restorePostFromCache creates a Post from cached data.
func (p *LoadPlugin) restorePostFromCache(data *buildcache.CachedPostData, cache *buildcache.Cache) *models.Post {
	post := models.NewPost(data.Path)
	post.Content = data.Content
	post.Slug = data.Slug
	post.Href = data.Href
	post.Title = data.Title
	post.Date = data.Date
	post.Published = data.Published
	post.Draft = data.Draft
	post.Private = data.Private
	post.Skip = data.Skip
	post.Tags = data.Tags
	post.Description = data.Description
	post.Template = data.Template
	post.Templates = data.Templates
	post.RawFrontmatter = data.RawFrontmatter
	post.InputHash = data.InputHash
	post.Authors = data.Authors
	post.Author = data.Author
	post.SecretKey = data.SecretKey // pragma: allowlist secret
	if data.Extra != nil {
		for k, v := range data.Extra {
			post.Set(k, v)
		}
	}

	if cache != nil {
		contentHash := buildcache.ContentHash(data.Content)
		if cachedHTML := cache.GetCachedArticleHTML(data.Path, contentHash); cachedHTML != "" {
			post.ArticleHTML = cachedHTML
		}
	}
	return post
}

func computePostFeedItemHash(post *models.Post) string {
	if post == nil {
		return ""
	}
	var b strings.Builder
	if post.Title != nil {
		b.WriteString(*post.Title)
	}
	b.WriteByte('\x00')
	b.WriteString(post.Slug)
	b.WriteByte('\x00')
	b.WriteString(post.Href)
	b.WriteByte('\x00')
	if post.Date != nil {
		b.WriteString(post.Date.UTC().Format(time.RFC3339Nano))
	}
	b.WriteByte('\x00')
	if post.Description != nil {
		b.WriteString(*post.Description)
	}
	b.WriteByte('\x00')
	b.WriteString(post.ArticleHTML)
	b.WriteByte('\x00')
	if len(post.Tags) > 0 {
		tags := append([]string(nil), post.Tags...)
		sort.Strings(tags)
		for _, tag := range tags {
			b.WriteString(tag)
			b.WriteByte('\x01')
		}
	}
	b.WriteByte('\x00')
	if post.Extra != nil {
		if img, ok := post.Extra["image"].(string); ok {
			b.WriteString(img)
		}
	}
	return buildcache.ContentHash(b.String())
}

func computePostTagIndexHash(post *models.Post) string {
	if post == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(post.Slug)
	b.WriteByte('\x00')
	if post.Title != nil {
		b.WriteString(*post.Title)
	}
	b.WriteByte('\x00')
	b.WriteString(post.Href)
	b.WriteByte('\x00')
	b.WriteString(fmt.Sprintf("%t", post.Published))
	b.WriteByte('\x00')
	b.WriteString(fmt.Sprintf("%t", post.Draft))
	b.WriteByte('\x00')
	b.WriteString(fmt.Sprintf("%t", post.Private))
	b.WriteByte('\x00')
	b.WriteString(fmt.Sprintf("%t", post.Skip))
	b.WriteByte('\x00')
	if len(post.Tags) > 0 {
		tags := append([]string(nil), post.Tags...)
		sort.Strings(tags)
		for _, tag := range tags {
			b.WriteString(tag)
			b.WriteByte('\x01')
		}
	}
	return buildcache.ContentHash(b.String())
}

func computePostGardenHash(post *models.Post) string {
	if post == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(post.Slug)
	b.WriteByte('\x00')
	if post.Title != nil {
		b.WriteString(*post.Title)
	}
	b.WriteByte('\x00')
	b.WriteString(post.Href)
	b.WriteByte('\x00')
	if post.Date != nil {
		b.WriteString(post.Date.UTC().Format(time.RFC3339Nano))
	}
	b.WriteByte('\x00')
	if post.Description != nil {
		b.WriteString(*post.Description)
	}
	b.WriteByte('\x00')
	b.WriteString(fmt.Sprintf("%t", post.Published))
	b.WriteByte('\x00')
	b.WriteString(fmt.Sprintf("%t", post.Draft))
	b.WriteByte('\x00')
	b.WriteString(fmt.Sprintf("%t", post.Private))
	b.WriteByte('\x00')
	b.WriteString(fmt.Sprintf("%t", post.Skip))
	b.WriteByte('\x00')
	if len(post.Tags) > 0 {
		tags := append([]string(nil), post.Tags...)
		sort.Strings(tags)
		for _, tag := range tags {
			b.WriteString(tag)
			b.WriteByte('\x01')
		}
	}
	b.WriteByte('\x00')
	if len(post.Outlinks) > 0 {
		links := make([]string, 0, len(post.Outlinks))
		for _, link := range post.Outlinks {
			links = append(links, link.RawTarget)
		}
		sort.Strings(links)
		for _, href := range links {
			b.WriteString(href)
			b.WriteByte('\x01')
		}
	}
	return buildcache.ContentHash(b.String())
}

// postToCachedData converts a Post to cacheable data.
func (p *LoadPlugin) postToCachedData(post *models.Post) *buildcache.CachedPostData {
	return &buildcache.CachedPostData{
		Path:           post.Path,
		Content:        post.Content,
		Slug:           post.Slug,
		Href:           post.Href,
		Title:          post.Title,
		Date:           post.Date,
		Published:      post.Published,
		Draft:          post.Draft,
		Private:        post.Private,
		Skip:           post.Skip,
		Tags:           post.Tags,
		Description:    post.Description,
		Template:       post.Template,
		Templates:      post.Templates,
		RawFrontmatter: post.RawFrontmatter,
		InputHash:      post.InputHash,
		Authors:        post.Authors,
		Author:         post.Author,
		SecretKey:      post.SecretKey, // pragma: allowlist secret
		Extra:          post.Extra,
	}
}

// loadSequential loads files one at a time (used for small file counts).
func (p *LoadPlugin) loadSequential(m *lifecycle.Manager, files []string, baseDir string, cache *buildcache.Cache) error {
	for _, file := range files {
		post, err := p.loadFile(file, baseDir, cache)
		if err != nil {
			return err
		}
		m.AddPost(post)
	}

	return nil
}

// ParsePostFromContent parses a markdown file content into a Post.
// This is a lightweight helper for cache-backed workflows that need
// frontmatter parsing and slug generation without running the full lifecycle.
func ParsePostFromContent(path, content string) (*models.Post, error) {
	loader := &LoadPlugin{}
	return loader.parseFile(path, content)
}

// parseFile parses a markdown file's content into a Post object.
func (p *LoadPlugin) parseFile(path, content string) (*models.Post, error) {
	// Parse frontmatter and get raw frontmatter for hashing
	metadata, body, rawFrontmatter, err := ParseFrontmatterWithRaw(content)
	if err != nil {
		return nil, err
	}

	// Create post with defaults
	post := models.NewPost(path)
	post.Content = body
	post.RawFrontmatter = rawFrontmatter

	// Apply metadata to post
	if err := p.applyMetadata(post, metadata); err != nil {
		return nil, err
	}

	// Generate slug if not explicitly set in frontmatter
	// If slug was explicitly set (even to empty string), respect it
	if !post.Has("_slug_explicit") && post.Slug == "" {
		post.GenerateSlug()
	}

	// Generate href from slug
	post.GenerateHref()

	// Compute input hash (content + frontmatter + template)
	// Template may be resolved later, so we use what we have now
	post.InputHash = buildcache.ComputePostInputHash(body, rawFrontmatter, post.Template)

	return post, nil
}

// applyMetadata applies parsed frontmatter metadata to a Post.
func (p *LoadPlugin) applyMetadata(post *models.Post, metadata map[string]interface{}) error {
	// Known fields to extract
	knownFields := map[string]bool{
		"title":          true,
		"date":           true,
		"published":      true,
		"draft":          true,
		"private":        true,
		"skip":           true,
		"tags":           true,
		"description":    true,
		"template":       true,
		"templates":      true,
		"slug":           true,
		"secret_key":     true,
		"private_key":    true,
		"encryption_key": true,
		"author":         true,
		"authors":        true,
		"by":             true,
		"writer":         true,
	}

	// Title
	if title := GetString(metadata, "title"); title != "" {
		post.Title = &title
	}

	// Date - handle various formats
	if dateVal, ok := metadata["date"]; ok {
		date, err := parseDate(dateVal)
		if err != nil {
			return fmt.Errorf("invalid date: %w", err)
		}
		post.Date = &date
	}

	// Published
	post.Published = GetBool(metadata, "published", false)

	// Draft
	post.Draft = GetBool(metadata, "draft", false)

	// Private
	post.Private = GetBool(metadata, "private", false)

	// Skip
	post.Skip = GetBool(metadata, "skip", false)

	// Tags
	if tags := GetStringSlice(metadata, "tags"); tags != nil {
		post.Tags = tags
	}

	// Description
	if desc := GetString(metadata, "description"); desc != "" {
		post.Description = &desc
	}

	// Template - support both 'template' and 'templateKey' for Python markata compatibility
	if template := GetString(metadata, "template"); template != "" {
		post.Template = template
	} else if template := GetString(metadata, "templateKey"); template != "" {
		post.Template = template
	}

	// Templates - per-format template overrides
	if templatesVal, ok := metadata["templates"]; ok {
		post.Templates = parseTemplatesMap(templatesVal)
	}

	// Slug - support custom slugs including explicit empty string for homepage
	if slugVal, exists := metadata["slug"]; exists {
		slug := normalizeCustomSlug(GetString(metadata, "slug"))
		post.Slug = slug
		// Mark that slug was explicitly set (prevents auto-generation)
		post.Set("_slug_explicit", true)
		_ = slugVal // Exists check used, value handled via GetString
	}

	// SecretKey - for encrypted posts  // pragma: allowlist secret
	// Supports aliases: secret_key, private_key, encryption_key (first non-empty wins)
	if secretKey := GetString(metadata, "secret_key"); secretKey != "" {
		post.SecretKey = secretKey // pragma: allowlist secret
	} else if privateKey := GetString(metadata, "private_key"); privateKey != "" {
		post.SecretKey = privateKey // pragma: allowlist secret
	} else if encryptionKey := GetString(metadata, "encryption_key"); encryptionKey != "" {
		post.SecretKey = encryptionKey // pragma: allowlist secret
	}

	// Author fields - support both legacy 'author' and new 'authors' arrays
	// Also support aliases: 'by' and 'writer' map to 'author'
	// SetAuthors handles: strings, []string, and mixed []interface{} with {id, role} maps
	if authorsVal, ok := metadata["authors"]; ok {
		post.SetAuthors(authorsVal)
	} else if author := GetString(metadata, "author"); author != "" {
		// Legacy single author field
		post.Author = &author
	} else if by := GetString(metadata, "by"); by != "" {
		// Alias: 'by' -> 'author'
		post.Author = &by
	} else if writer := GetString(metadata, "writer"); writer != "" {
		// Alias: 'writer' -> 'author'
		post.Author = &writer
	}

	// Store unknown fields in Extra
	for key, value := range metadata {
		if !knownFields[key] {
			post.Set(key, value)
		}
	}

	return nil
}

// parseDate attempts to parse a date value from various formats.
func parseDate(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		return parseDateString(v)
	default:
		return time.Time{}, fmt.Errorf("unsupported date type: %T", value)
	}
}

// parseDateString parses a date string using common formats.
func parseDateString(s string) (time.Time, error) {
	// Normalize the date string first
	s = normalizeDateString(s)

	// Common date formats to try (most specific first)
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006/01/02",
		"01/02/2006 15:04:05",
		"01/02/2006 15:04",
		"01/02/2006",
		"02-01-2006",
		"January 2, 2006",
		"Jan 2, 2006",
		"2 January 2006",
		"2 Jan 2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}

// normalizeDateString normalizes a date string to handle common variations.
// It handles:
// - Single-digit hours/minutes (e.g., "1:00:00" -> "01:00:00")
// - Malformed time components (e.g., "8:011:00" -> "8:11:00")
// - Extra whitespace
func normalizeDateString(s string) string {
	// Trim whitespace
	s = strings.TrimSpace(s)

	// Early return if no time component
	if !strings.Contains(s, ":") {
		return s
	}

	// Fix malformed time components like "8:011:00" -> "08:11:00"
	// This regex finds time components with potentially extra leading zeros
	s = timeFixRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := timeFixRegex.FindStringSubmatch(match)
		if len(parts) == 4 {
			h, err := strconv.Atoi(parts[1])
			if err != nil {
				return match
			}
			m, err := strconv.Atoi(parts[2])
			if err != nil {
				return match
			}
			sec, err := strconv.Atoi(parts[3])
			if err != nil {
				return match
			}
			return fmt.Sprintf("%02d:%02d:%02d", h, m, sec)
		}
		return match
	})

	// Normalize single-digit hours in time component
	// Match patterns like " 1:00" or "T1:00" and pad the hour
	s = singleDigitHourRegex.ReplaceAllString(s, "${1}0${2}:${3}")

	// Handle time at start of string or after date with space
	if startSingleDigitHourRegex.MatchString(s) {
		s = "0" + s
	}

	return s
}

// normalizeCustomSlug normalizes a slug from frontmatter.
// It handles:
//   - "/" or "" -> "" (homepage)
//   - "/docs/page" -> "docs/page" (strip leading slash)
//   - "docs/page/" -> "docs/page" (strip trailing slash)
//   - Preserves internal structure for nested paths
func normalizeCustomSlug(slug string) string {
	// Trim whitespace
	slug = strings.TrimSpace(slug)

	// "/" means homepage (empty slug)
	if slug == "/" {
		return ""
	}

	// Strip leading and trailing slashes
	slug = strings.Trim(slug, "/")

	return slug
}

// parseTemplatesMap parses the templates field from frontmatter.
// It handles both map[string]interface{} and map[string]string formats.
func parseTemplatesMap(val interface{}) map[string]string {
	result := make(map[string]string)

	switch v := val.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if str, ok := value.(string); ok {
				result[key] = str
			}
		}
	case map[string]string:
		for key, value := range v {
			result[key] = value
		}
	case map[interface{}]interface{}:
		// Handle YAML's default map type
		for key, value := range v {
			if keyStr, ok := key.(string); ok {
				if valStr, ok := value.(string); ok {
					result[keyStr] = valStr
				}
			}
		}
	}

	return result
}
