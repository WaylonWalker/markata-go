package cmd

import (
	"bufio"
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

	// newTags is a comma-separated list of tags.
	newTags string
)

// newCmd represents the new command.
var newCmd = &cobra.Command{
	Use:   "new [title]",
	Short: "Create a new post",
	Long: `Create a new markdown post with frontmatter template.

The command generates a new markdown file with:
  - Title set from the argument (or prompted if not provided)
  - Slug generated from the title
  - Current date
  - Draft status (configurable)
  - Tags (optional)

Example usage:
  markata-go new "My First Post"              # Create posts/my-first-post.md
  markata-go new "Hello World" --dir blog     # Create blog/hello-world.md
  markata-go new "Draft Post" --draft         # Create as draft (default)
  markata-go new "Published" --draft=false    # Create as published
  markata-go new "Go Tutorial" --tags "go,tutorial"  # Create with tags
  markata-go new                              # Interactive mode`,
	Args: cobra.MaximumNArgs(1),
	RunE: runNewCommand,
}

func init() {
	rootCmd.AddCommand(newCmd)

	newCmd.Flags().StringVar(&newDir, "dir", "posts", "directory for new post")
	newCmd.Flags().BoolVar(&newDraft, "draft", true, "create as draft")
	newCmd.Flags().StringVar(&newTags, "tags", "", "comma-separated list of tags")
}

func runNewCommand(cmd *cobra.Command, args []string) error {
	var title string
	var tags []string

	// Parse tags from flag if provided
	if newTags != "" {
		tags = parseTags(newTags)
	}

	// If no title provided, run interactive mode
	if len(args) == 0 {
		reader := bufio.NewReader(os.Stdin)

		fmt.Println()

		// Get title
		title = promptNew(reader, "Post title", "")
		if title == "" {
			return fmt.Errorf("post title is required")
		}

		// Get directory (only prompt if not explicitly set via flag)
		if !cmd.Flags().Changed("dir") {
			newDir = promptNew(reader, "Directory", "posts")
		}

		// Get tags (only prompt if not explicitly set via flag)
		if !cmd.Flags().Changed("tags") {
			tagsInput := promptNew(reader, "Tags (comma-separated)", "")
			if tagsInput != "" {
				tags = parseTags(tagsInput)
			}
		}

		// Get draft status (only prompt if not explicitly set via flag)
		if !cmd.Flags().Changed("draft") {
			newDraft = promptYesNoNew(reader, "Create as draft?", true)
		}

		fmt.Println()
	} else {
		title = args[0]
	}

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
	content := generatePostContentWithTags(title, slug, now, newDraft, tags)

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
		if len(tags) > 0 {
			fmt.Printf("  Tags: %s\n", strings.Join(tags, ", "))
		}
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
	return generatePostContentWithTags(title, slug, date, draft, nil)
}

// generatePostContentWithTags creates the markdown content with frontmatter and optional tags.
func generatePostContentWithTags(title, slug string, date time.Time, draft bool, tags []string) string {
	published := !draft

	// Format tags as YAML array
	tagsYAML := "[]"
	if len(tags) > 0 {
		var quotedTags []string
		for _, tag := range tags {
			quotedTags = append(quotedTags, fmt.Sprintf("%q", tag))
		}
		tagsYAML = "[" + strings.Join(quotedTags, ", ") + "]"
	}

	return fmt.Sprintf(`---
title: "%s"
slug: "%s"
date: %s
published: %t
draft: %t
tags: %s
description: ""
---

# %s

Write your content here...
`, title, slug, date.Format("2006-01-02"), published, draft, tagsYAML, title)
}

// promptNew displays a question and returns the user's response or a default value.
func promptNew(reader *bufio.Reader, question, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", question, defaultVal)
	} else {
		fmt.Printf("%s: ", question)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

// promptYesNoNew displays a yes/no question and returns the boolean result.
func promptYesNoNew(reader *bufio.Reader, question string, defaultYes bool) bool {
	defaultStr := "y/N"
	if defaultYes {
		defaultStr = "Y/n"
	}
	fmt.Printf("%s (%s): ", question, defaultStr)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}

// parseTags splits a comma-separated tag string into a slice of trimmed tags.
func parseTags(tagsStr string) []string {
	var tags []string
	for _, tag := range strings.Split(tagsStr, ",") {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}
