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
type ContentIndexPlugin struct {
	initialSource    sourcegit.State
	initialSourceErr error
	capturedSource   bool
}

func NewContentIndexPlugin() *ContentIndexPlugin { return &ContentIndexPlugin{} }
func (p *ContentIndexPlugin) Name() string       { return "content_index" }
func (p *ContentIndexPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageWrite {
		return lifecycle.PriorityLast
	}
	return lifecycle.PriorityDefault
}

// Configure captures source state before content discovery starts. The final
// write compares this snapshot with a second snapshot before publishing.
func (p *ContentIndexPlugin) Configure(m *lifecycle.Manager) error {
	cfg, ok, err := contentIndexConfig(m.Config().Extra)
	if err != nil {
		return err
	}
	if !ok || !cfg.Enabled {
		return nil
	}
	p.initialSource, p.initialSourceErr = sourcegit.Read(context.Background(), m.Config().ContentDir)
	p.capturedSource = true
	return nil
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
	stateBefore, beforeErr := p.initialSource, p.initialSourceErr
	if !p.capturedSource {
		stateBefore, beforeErr = sourcegit.Read(context.Background(), m.Config().ContentDir)
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
	stateAfter, afterErr := sourcegit.Read(context.Background(), m.Config().ContentDir)
	if beforeErr == nil && afterErr == nil {
		if !stateBefore.Equal(stateAfter) {
			return fmt.Errorf("source Git state changed during Content Index build")
		}
		index.Source.Commit = stateAfter.Commit
		index.Source.Dirty = stateAfter.Dirty
	}
	data, err := contentindex.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshal content index: %w", err)
	}
	destination, err := contentIndexDestination(m.Config().OutputDir, cfg.Output)
	if err != nil {
		return err
	}
	if strings.EqualFold(filepath.Clean(destination), filepath.Join(filepath.Clean(m.Config().OutputDir), "_headers")) {
		return fmt.Errorf("content_index.output cannot be %q; it is reserved for static headers", cfg.Output)
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
	if err := writeContentIndexHeaders(m.Config().OutputDir, destination); err != nil {
		return fmt.Errorf("write content index CORS headers: %w", err)
	}
	return nil
}

// writeContentIndexHeaders emits the Cloudflare Pages static-header sidecar
// for the public artifact. The artifact contains public metadata only, so it
// is safe and useful for browser consumers on any origin. Keep the rule
// specific to the artifact instead of widening CORS for the complete site.
func writeContentIndexHeaders(outputDir, destination string) error {
	relative, err := filepath.Rel(outputDir, destination)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		// A Pages header file can only control files inside the deployed output.
		// Absolute destinations outside it are supported for artifact generation,
		// but must not receive a misleading rule in output/_headers.
		return nil
	}
	route := "/" + filepath.ToSlash(relative)
	if strings.ContainsAny(route, "*?[]{}:\r\n") {
		// Cloudflare Pages treats these characters as route patterns. Do not
		// turn a user-selected artifact name into a broader CORS grant.
		return nil
	}
	headersPath := filepath.Join(outputDir, "_headers")
	existing, err := os.ReadFile(headersPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", headersPath, err)
	}
	lines := strings.Split(strings.ReplaceAll(string(existing), "\r\n", "\n"), "\n")
	for index, line := range lines {
		if line != route {
			continue
		}
		end := index + 1
		for end < len(lines) && (lines[end] == "" || strings.HasPrefix(lines[end], " ") || strings.HasPrefix(lines[end], "\t")) {
			end++
		}
		for _, header := range lines[index+1 : end] {
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(header)), "access-control-allow-origin:") {
				return nil // Preserve an explicitly configured restrictive policy.
			}
		}
		lines = append(lines[:index+1], append([]string{"  Access-Control-Allow-Origin: *"}, lines[index+1:]...)...)
		return writeHeadersFile(headersPath, strings.Join(lines, "\n"))
	}
	block := fmt.Sprintf("%s\n  Access-Control-Allow-Origin: *\n", route)
	content := append([]byte(nil), existing...)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content = append(content, '\n')
	}
	content = append(content, []byte(block)...)
	return writeHeadersFile(headersPath, string(content))
}

func writeHeadersFile(path, content string) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".headers-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary headers file: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set headers permissions: %w", err)
	}
	if _, err := temporary.WriteString(content); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary headers file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary headers file: %w", err)
	}
	if runtime.GOOS == tailwindOSWindows {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove existing headers file: %w", err)
		}
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("replace headers file: %w", err)
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
