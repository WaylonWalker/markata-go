package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	// newDir is the directory for new posts.
	newDir string

	// newDraft creates the post as a draft.
	newDraft bool
)

// newCmd represents the new command.
var newCmd = &cobra.Command{
	Use:   "new [title]",
	Short: "Create a new post",
	Long: `Create a new markdown post with frontmatter template.

The command generates a new markdown file with:
  - Title set from the argument
  - Slug generated from the title
  - Current date
  - Draft status (configurable)
  - Empty tags array

Example usage:
  markata-go new "My First Post"          # Create posts/my-first-post.md
  markata-go new "Hello World" --dir blog # Create blog/hello-world.md
  markata-go new "Draft Post" --draft     # Create as draft (default)
  markata-go new "Published" --draft=false # Create as published`,
	Args: cobra.ExactArgs(1),
	RunE: runNewCommand,
}

func init() {
	rootCmd.AddCommand(newCmd)

	newCmd.Flags().StringVar(&newDir, "dir", "posts", "directory for new post")
	newCmd.Flags().BoolVar(&newDraft, "draft", true, "create as draft")
}

func runNewCommand(_ *cobra.Command, args []string) error {
	title := args[0]

	// Generate slug from title
	slug := generateSlug(title)

	// Create filename
	filename := slug + ".md"
	fullPath := filepath.Join(newDir, filename)

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Errorf("file already exists: %s", fullPath)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate frontmatter
	now := time.Now()
	content := generatePostContent(title, slug, now, newDraft)

	// Write file (0o644 is appropriate for content files that should be world-readable)
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil { //nolint:gosec // content files should be readable
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Created: %s\n", fullPath)
	if verbose {
		fmt.Printf("  Title: %s\n", title)
		fmt.Printf("  Slug: %s\n", slug)
		fmt.Printf("  Date: %s\n", now.Format("2006-01-02"))
		fmt.Printf("  Draft: %t\n", newDraft)
	}

	return nil
}

// generateSlug creates a URL-safe slug from a title.
func generateSlug(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove non-alphanumeric characters (except hyphens)
	reg := regexp.MustCompile(`[^a-z0-9\-]+`)
	slug = reg.ReplaceAllString(slug, "")

	// Collapse multiple hyphens
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	return slug
}

// generatePostContent creates the markdown content with frontmatter.
func generatePostContent(title, slug string, date time.Time, draft bool) string {
	published := !draft

	return fmt.Sprintf(`---
title: "%s"
slug: "%s"
date: %s
published: %t
draft: %t
tags: []
description: ""
---

# %s

Write your content here...
`, title, slug, date.Format("2006-01-02"), published, draft, title)
}
