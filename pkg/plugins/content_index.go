package plugins

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/contentindex"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/sourcegit"
)

// ContentIndexPlugin writes the opt-in Markata Content Index after feed
// membership has been resolved.
type ContentIndexPlugin struct{}

func NewContentIndexPlugin() *ContentIndexPlugin { return &ContentIndexPlugin{} }
func (p *ContentIndexPlugin) Name() string       { return "content_index" }
func (p *ContentIndexPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageWrite {
		return lifecycle.PriorityLast
	}
	return lifecycle.PriorityDefault
}

//nolint:gocyclo // The writer coordinates the complete build-time artifact pipeline.
func (p *ContentIndexPlugin) Write(m *lifecycle.Manager) (err error) {
	defer func() {
		if err != nil {
			err = &contentIndexWriteError{err: err}
		}
	}()
	cfg, ok, configErr := contentIndexConfig(m.Config().Extra)
	if configErr != nil {
		return configErr
	}
	if !ok || !cfg.Enabled {
		return nil
	}
	if cfg.SchemaVersion != 0 && cfg.SchemaVersion != contentindex.CurrentVersion {
		return fmt.Errorf("content_index.schema_version %d is unsupported; latest is %d", cfg.SchemaVersion, contentindex.CurrentVersion)
	}

	posts := m.Posts()
	posts = slices.DeleteFunc(posts, func(post *models.Post) bool { return post == nil })
	sort.SliceStable(posts, func(i, j int) bool { return posts[i].Path < posts[j].Path })
	public := make(map[string]bool, len(posts))
	for _, post := range posts {
		if post != nil && !post.Private && !post.Draft && !post.Skip {
			public[post.Path] = true
		}
	}
	feeds := make(map[string][]string)
	for _, feed := range m.Feeds() {
		if feed == nil {
			continue
		}
		for _, post := range feed.Posts {
			if post != nil && public[post.Path] {
				feeds[post.Path] = append(feeds[post.Path], feed.Name)
			}
		}
	}
	for path := range feeds {
		sort.Strings(feeds[path])
	}

	docs := make([]contentindex.Document, 0, len(public))
	for _, post := range posts {
		if post == nil || !public[post.Path] {
			continue
		}
		docs = append(docs, documentFromPost(post, feeds[post.Path]))
	}
	version := "dev"
	if v, ok := m.Config().Extra["markata_version"].(string); ok && v != "" {
		version = v
	}
	index := contentindex.Index{Schema: contentindex.Schema, SchemaVersion: contentindex.CurrentVersion, Scope: contentindex.PublicScope, Generator: contentindex.Generator{Name: contentindex.GeneratorName, Version: version}, DocumentCount: len(docs), Documents: docs}
	if state, err := sourcegit.Read(context.Background(), m.Config().ContentDir); err == nil {
		index.Source.Commit = state.Commit
		index.Source.Dirty = state.Dirty
	}
	data, err := contentindex.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshal content index: %w", err)
	}
	destination, err := contentIndexDestination(m.Config().OutputDir, cfg.Output)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create content index directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".content-index-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary content index: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set content index permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary content index: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary content index: %w", err)
	}
	if err := replaceContentIndex(temporaryName, destination); err != nil {
		return fmt.Errorf("write content index: %w", err)
	}
	return nil
}

func contentIndexDestination(outputDir, configured string) (string, error) {
	destination := configured
	if destination == "" {
		destination = "content-index.json"
	}
	if filepath.IsAbs(destination) {
		return destination, nil
	}
	clean := filepath.Clean(destination)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("content_index.output must stay within output_dir: %q", configured)
	}
	return filepath.Join(outputDir, clean), nil
}

func replaceContentIndex(source, destination string) error {
	if runtime.GOOS == tailwindOSWindows {
		if err := os.Remove(destination); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return os.Rename(source, destination)
}

type contentIndexWriteError struct{ err error }

func (e *contentIndexWriteError) Error() string    { return e.err.Error() }
func (e *contentIndexWriteError) Unwrap() error    { return e.err }
func (e *contentIndexWriteError) IsCritical() bool { return true }

func documentFromPost(post *models.Post, feedNames []string) contentindex.Document {
	return contentindex.Document{Path: post.Path, Slug: post.Slug, Href: post.Href, Title: post.Title, TitleText: titleText(post), Date: post.Date, Modified: post.Modified, Published: post.Published, Draft: post.Draft, Private: post.Private, Template: post.Template, Tags: sortedStrings(post.Tags), Description: post.Description, Feeds: append([]string(nil), feedNames...), Aliases: aliases(post)}
}

func titleText(post *models.Post) *string {
	if post.Title == nil && !post.TitleTextDerived && post.TitleText == "" {
		return nil
	}
	v := post.PlainTitle()
	return &v
}

func aliases(post *models.Post) []string {
	values, ok := post.Extra["aliases"]
	if !ok {
		return nil
	}
	result := make([]string, 0)
	switch v := values.(type) {
	case []string:
		result = append(result, v...)
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
	}
	return sortedStrings(result)
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

type contentIndexOptions struct {
	Enabled       bool
	Output        string
	SchemaVersion int
}

func contentIndexConfig(extra map[string]interface{}) (contentIndexOptions, bool, error) {
	value, ok := extra["content_index"]
	if !ok {
		return contentIndexOptions{}, false, nil
	}
	result := contentIndexOptions{Enabled: true}
	switch cfg := value.(type) {
	case map[string]interface{}:
		if enabled, exists := cfg["enabled"]; exists {
			value, valid := enabled.(bool)
			if !valid {
				return contentIndexOptions{}, false, fmt.Errorf("content_index.enabled must be boolean")
			}
			result.Enabled = value
		}
		if output, exists := cfg["output"]; exists {
			value, valid := output.(string)
			if !valid {
				return contentIndexOptions{}, false, fmt.Errorf("content_index.output must be a string")
			}
			result.Output = value
		}
		if value, exists := cfg["schema_version"]; exists {
			version, valid := parseOptionalInt(value)
			if !valid {
				return contentIndexOptions{}, false, fmt.Errorf("content_index.schema_version must be an integer")
			}
			result.SchemaVersion = version
		}
	case contentIndexOptions:
		result = cfg
	default:
		return contentIndexOptions{}, false, fmt.Errorf("content_index must be a table")
	}
	return result, true, nil
}

func parseOptionalInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		if math.Trunc(v) != v {
			return 0, false
		}
		return int(v), true
	default:
		return 0, false
	}
}
