// Package contentedit provides markdown file editing with frontmatter handling.
package contentedit

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	// ErrInvalidFrontmatter indicates frontmatter is not valid YAML
	ErrInvalidFrontmatter = fmt.Errorf("invalid frontmatter")
	// ErrFileNotFound indicates the file does not exist
	ErrFileNotFound = fmt.Errorf("file not found")
	// ErrConflict indicates the file was modified since last read
	ErrConflict        = fmt.Errorf("file conflict")
	slugCleanupPattern = regexp.MustCompile(`[^a-z0-9-]+`)
)

const (
	delimiter = "---"
)

// Post represents a markdown file ready for editing
type Post struct {
	Path        string
	Slug        string
	PreviewURL  string
	Frontmatter string
	Body        string
	Hash        string
	Exists      bool
	GitStatus   string // "modified", "staged", "untracked", "tracked"

	// Cached parsed values
	title     string
	date      string
	published bool
	loaded    bool
}

// Title returns the title from frontmatter
func (p *Post) GetTitle() string {
	if p.loaded {
		return p.title
	}
	p.parseFrontmatter()
	return p.title
}

// Date returns the date from frontmatter
func (p *Post) GetDate() string {
	if p.loaded {
		return p.date
	}
	p.parseFrontmatter()
	return p.date
}

// Published returns the published status from frontmatter
func (p *Post) IsPublished() bool {
	if p.loaded {
		return p.published
	}
	p.parseFrontmatter()
	return p.published
}

// parseFrontmatter extracts common fields from frontmatter
func (p *Post) parseFrontmatter() {
	if p.Frontmatter == "" {
		p.loaded = true
		return
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(p.Frontmatter), &data); err != nil {
		p.loaded = true
		return
	}

	if v, ok := data["title"].(string); ok {
		p.title = v
	}
	if v, ok := data["date"].(string); ok {
		p.date = v
	}
	if v, ok := data["published"].(bool); ok {
		p.published = v
	} else if v, ok := data["draft"].(bool); ok {
		p.published = !v
	}

	p.loaded = true
}

// LoadPost loads a markdown file and splits it into frontmatter and body
func LoadPost(path string) (*Post, error) {
	// Verify file exists
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)

	// Split frontmatter and body
	frontmatter, body, err := extractFrontmatter(content)
	if err != nil {
		return nil, err
	}

	// Extract slug from path
	slug := extractSlugFromContent(path, frontmatter)

	return &Post{
		Path:        path,
		Slug:        slug,
		PreviewURL:  "/" + slug + "/",
		Frontmatter: frontmatter,
		Body:        body,
		Hash:        ContentHash(content),
		Exists:      true,
	}, nil
}

// NewPost creates a new unsaved post draft.
func NewPost(path, frontmatter, body string) *Post {
	slug := extractSlugFromContent(path, frontmatter)
	content := BuildContent(frontmatter, body)
	previewURL := "/"
	if slug != "" {
		previewURL = "/" + slug + "/"
	}

	return &Post{
		Path:        path,
		Slug:        slug,
		PreviewURL:  previewURL,
		Frontmatter: frontmatter,
		Body:        body,
		Hash:        ContentHash(content),
		Exists:      false,
	}
}

// extractFrontmatter splits markdown content into frontmatter and body
func extractFrontmatter(content string) (frontmatter, body string, err error) {
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// Check if content starts with frontmatter delimiter
	if !strings.HasPrefix(content, delimiter) {
		// No frontmatter
		return "", content, nil
	}

	// Find the end of the opening delimiter line
	afterOpening := content[len(delimiter):]

	// The opening delimiter must be on its own line
	if afterOpening != "" && afterOpening[0] != '\n' {
		return "", content, nil
	}

	// Skip the newline after opening delimiter
	afterOpening = strings.TrimPrefix(afterOpening, "\n")

	// Handle empty frontmatter case
	if strings.HasPrefix(afterOpening, delimiter) {
		remaining := afterOpening[len(delimiter):]
		remaining = strings.TrimPrefix(remaining, "\n")
		return "", remaining, nil
	}

	// Find the closing delimiter
	closingIdx := strings.Index(afterOpening, "\n"+delimiter)
	if closingIdx == -1 {
		// Check if content ends with the delimiter on its own line
		if strings.HasSuffix(afterOpening, "\n"+delimiter) {
			closingIdx = len(afterOpening) - len(delimiter) - 1
		} else {
			return "", "", ErrInvalidFrontmatter
		}
	}

	// Extract frontmatter and body
	frontmatter = afterOpening[:closingIdx]
	remaining := afterOpening[closingIdx+1:]
	remaining = strings.TrimPrefix(remaining, delimiter)
	remaining = strings.TrimPrefix(remaining, "\n")
	body = remaining

	return frontmatter, body, nil
}

// extractSlug extracts a slug from a markdown file path
func extractSlug(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	if ext != "" {
		base = base[:len(base)-len(ext)]
	}
	return strings.ToLower(strings.ReplaceAll(base, " ", "-"))
}

func extractSlugFromContent(path, frontmatter string) string {
	if frontmatter != "" {
		var data map[string]interface{}
		if err := yaml.Unmarshal([]byte(frontmatter), &data); err == nil {
			if slug, ok := data["slug"].(string); ok && strings.TrimSpace(slug) != "" {
				return slugify(strings.TrimSpace(slug))
			}
			if title, ok := data["title"].(string); ok && strings.TrimSpace(title) != "" {
				return slugify(strings.TrimSpace(title))
			}
		}
	}
	return extractSlug(path)
}

func slugify(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	input = strings.ReplaceAll(input, "/", "-")
	input = strings.ReplaceAll(input, "_", "-")
	input = strings.ReplaceAll(input, " ", "-")
	input = slugCleanupPattern.ReplaceAllString(input, "-")
	for strings.Contains(input, "--") {
		input = strings.ReplaceAll(input, "--", "-")
	}
	return strings.Trim(input, "-")
}

// BuildContent renders frontmatter and body into a markdown file.
func BuildContent(frontmatter, body string) string {
	var buf strings.Builder
	if frontmatter != "" {
		frontmatter = strings.TrimRight(frontmatter, "\n")
		buf.WriteString("---\n")
		buf.WriteString(frontmatter)
		buf.WriteString("\n")
		buf.WriteString("---\n\n")
	}
	buf.WriteString(body)
	return buf.String()
}

// ContentHash returns a stable hash for content conflict detection.
func ContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// SavePost saves a post with formatted frontmatter
type SaveOptions struct {
	BaseHash string // Hash from last read, for conflict detection
}

func SavePost(post *Post, opts *SaveOptions) error {
	if post == nil {
		return fmt.Errorf("post is required")
	}
	if strings.TrimSpace(post.Path) == "" {
		return fmt.Errorf("path is required")
	}

	// Format the frontmatter (may be empty)
	formattedFM := post.Frontmatter
	if post.Frontmatter != "" {
		var err error
		formattedFM, err = FormatFrontmatter(post.Frontmatter)
		if err != nil {
			return fmt.Errorf("frontmatter validation failed: %w", err)
		}
	}

	content := BuildContent(formattedFM, post.Body)

	if opts != nil && opts.BaseHash != "" {
		if existing, err := os.ReadFile(post.Path); err == nil {
			if ContentHash(string(existing)) != opts.BaseHash {
				return ErrConflict
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	// Write to temp file first, then rename (atomic write)
	dir := filepath.Dir(post.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmpFile := filepath.Join(dir, ".tmp-"+filepath.Base(post.Path)+"-"+randStr(8))

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return err
	}

	// Rename to final location
	if err := os.Rename(tmpFile, post.Path); err != nil {
		os.Remove(tmpFile)
		return err
	}

	post.Frontmatter = formattedFM
	post.Slug = extractSlugFromContent(post.Path, formattedFM)
	post.PreviewURL = "/" + post.Slug + "/"
	post.Hash = ContentHash(content)
	post.Exists = true

	return nil
}

func randStr(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, n)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		result[i] = letters[num.Int64()]
	}
	return string(result)
}

// FormatFrontmatter parses and reformats frontmatter with consistent YAML style
func FormatFrontmatter(input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", nil
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(input), &data); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err)
	}

	buf := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buf)
	encoder.SetIndent(2)

	if err := encoder.Encode(buildOrderedYAMLMap(data)); err != nil {
		return "", err
	}

	encoder.Close()

	return strings.TrimSuffix(buf.String(), "\n"), nil
}

func buildOrderedYAMLMap(data map[string]interface{}) *yaml.Node {
	priority := map[string]int{
		"title":       0,
		"slug":        1,
		"date":        2,
		"published":   3,
		"draft":       4,
		"description": 5,
		"tags":        6,
		"templateKey": 7,
		"layout":      8,
	}
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		pi, iok := priority[keys[i]]
		pj, jok := priority[keys[j]]
		if iok && jok {
			return pi < pj
		}
		if iok != jok {
			return iok
		}
		return keys[i] < keys[j]
	})

	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, key := range keys {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
			interfaceToYAMLNode(data[key]),
		)
	}
	return node
}

func interfaceToYAMLNode(value interface{}) *yaml.Node {
	switch typed := value.(type) {
	case map[string]interface{}:
		return buildOrderedYAMLMap(typed)
	case map[interface{}]interface{}:
		converted := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			converted[fmt.Sprint(key)] = item
		}
		return buildOrderedYAMLMap(converted)
	case []interface{}:
		node := &yaml.Node{Kind: yaml.SequenceNode}
		for _, item := range typed {
			node.Content = append(node.Content, interfaceToYAMLNode(item))
		}
		return node
	case []string:
		node := &yaml.Node{Kind: yaml.SequenceNode}
		for _, item := range typed {
			node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: item})
		}
		return node
	case bool:
		if typed {
			return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"}
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "false"}
	case string:
		node := &yaml.Node{Kind: yaml.ScalarNode, Value: typed}
		if strings.Contains(typed, "\n") {
			node.Style = yaml.LiteralStyle
		}
		return node
	case time.Time:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!timestamp", Value: typed.UTC().Format(time.RFC3339)}
	default:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprint(value)}
	}
}

// ValidateFrontmatter returns validation errors for frontmatter
func ValidateFrontmatter(input string) error {
	if strings.TrimSpace(input) == "" {
		return nil
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(input), &data); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err)
	}

	// Check known fields have reasonable types
	knownFields := map[string]func(interface{}) error{
		"title": func(v interface{}) error {
			if v != nil {
				if _, ok := v.(string); !ok {
					return fmt.Errorf("title must be a string")
				}
			}
			return nil
		},
		"date": func(v interface{}) error {
			if v != nil {
				if _, ok := v.(string); !ok {
					return fmt.Errorf("date must be a string")
				}
			}
			return nil
		},
		"published": func(v interface{}) error {
			if v != nil {
				if _, ok := v.(bool); !ok {
					return fmt.Errorf("published must be a boolean")
				}
			}
			return nil
		},
		"draft": func(v interface{}) error {
			if v != nil {
				if _, ok := v.(bool); !ok {
					return fmt.Errorf("draft must be a boolean")
				}
			}
			return nil
		},
		"tags": func(v interface{}) error {
			if v != nil {
				if _, ok := v.([]interface{}); !ok {
					return fmt.Errorf("tags must be a list")
				}
			}
			return nil
		},
		"description": func(v interface{}) error {
			if v != nil {
				if _, ok := v.(string); !ok {
					return fmt.Errorf("description must be a string")
				}
			}
			return nil
		},
		"slug": func(v interface{}) error {
			if v != nil {
				if _, ok := v.(string); !ok {
					return fmt.Errorf("slug must be a string")
				}
			}
			return nil
		},
	}

	for k, v := range data {
		if validate, ok := knownFields[k]; ok {
			if err := validate(v); err != nil {
				return err
			}
		}
	}

	return nil
}

// IsValidPath validates that a path is within allowed content directories
var validPathRegex = regexp.MustCompile(`^(\.\./|[a-zA-Z]:/)`)

func IsValidPath(path string) bool {
	// Reject paths with traversal attempts
	if strings.Contains(path, "..") {
		return false
	}
	// Reject absolute paths (unless on Windows with drive letter)
	if strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "/.") {
		return false
	}
	return true
}

// ListPosts returns all markdown files in the content directory
func ListPosts(contentDir string) ([]*Post, error) {
	var posts []*Post

	err := filepath.Walk(contentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-markdown files
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".markdown" {
			return nil
		}

		post, err := LoadPost(path)
		if err != nil {
			// Skip files that can't be loaded
			return nil
		}

		posts = append(posts, post)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return posts, nil
}
