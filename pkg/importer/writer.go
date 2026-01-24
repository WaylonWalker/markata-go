package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Writer handles writing imported posts as markdown files.
type Writer struct {
	outputDir string
}

// NewWriter creates a new Writer for the specified output directory.
func NewWriter(outputDir string) *Writer {
	return &Writer{outputDir: outputDir}
}

// WriteResult contains the result of writing posts.
type WriteResult struct {
	// Written is the number of files written
	Written int

	// Skipped is the number of files skipped (already exist)
	Skipped int

	// Paths contains the paths of written files
	Paths []string

	// Errors contains any non-fatal errors encountered
	Errors []error
}

// Write writes imported posts as markdown files.
func (w *Writer) Write(posts []*ImportedPost, dryRun bool) (*WriteResult, error) {
	result := &WriteResult{}

	// Create output directory if it doesn't exist
	if !dryRun {
		if err := os.MkdirAll(w.outputDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	for _, post := range posts {
		path, err := w.writePost(post, dryRun)
		if err != nil {
			if os.IsExist(err) {
				result.Skipped++
				continue
			}
			result.Errors = append(result.Errors, fmt.Errorf("failed to write %s: %w", post.Slug, err))
			continue
		}

		result.Written++
		result.Paths = append(result.Paths, path)
	}

	return result, nil
}

// writePost writes a single post as a markdown file.
func (w *Writer) writePost(post *ImportedPost, dryRun bool) (string, error) {
	filename := post.Slug + ".md"
	path := filepath.Join(w.outputDir, filename)

	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return path, os.ErrExist
	}

	if dryRun {
		return path, nil
	}

	content := formatPostAsMarkdown(post)

	// Write file with readable permissions (0o644 is appropriate for content files)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint:gosec // content files should be readable
		return "", err
	}

	return path, nil
}

// formatPostAsMarkdown formats an imported post as a markdown file with frontmatter.
func formatPostAsMarkdown(post *ImportedPost) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: %q\n", post.Title))
	sb.WriteString(fmt.Sprintf("date: %s\n", post.Published.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("source_url: %q\n", post.SourceURL))
	sb.WriteString(fmt.Sprintf("source_type: %s\n", post.SourceType))
	sb.WriteString(fmt.Sprintf("imported: %s\n", post.Imported.Format("2006-01-02")))

	if post.Author != "" {
		sb.WriteString(fmt.Sprintf("author: %q\n", post.Author))
	}

	if post.Summary != "" {
		// Escape quotes in summary for YAML
		summary := strings.ReplaceAll(post.Summary, "\"", "\\\"")
		sb.WriteString(fmt.Sprintf("description: %q\n", summary))
	}

	if len(post.Tags) > 0 {
		sb.WriteString("tags:\n")
		// Always add imported and source_type as tags
		tagsWritten := make(map[string]bool)
		sb.WriteString("  - imported\n")
		tagsWritten["imported"] = true
		sb.WriteString(fmt.Sprintf("  - %s\n", post.SourceType))
		tagsWritten[post.SourceType] = true

		for _, tag := range post.Tags {
			tag = strings.TrimSpace(tag)
			if tag != "" && !tagsWritten[tag] {
				sb.WriteString(fmt.Sprintf("  - %s\n", tag))
				tagsWritten[tag] = true
			}
		}
	} else {
		// Always add imported and source_type tags
		sb.WriteString("tags:\n")
		sb.WriteString("  - imported\n")
		sb.WriteString(fmt.Sprintf("  - %s\n", post.SourceType))
	}

	sb.WriteString("published: true\n")
	sb.WriteString("---\n\n")

	// Use content text, not HTML
	content := post.Content
	if content == "" {
		content = stripHTML(post.ContentHTML)
	}
	sb.WriteString(content)
	sb.WriteString("\n")

	return sb.String()
}
