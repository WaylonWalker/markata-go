package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// initCmd represents the init command.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new markata-go project",
	Long: `Initialize a new markata-go project with interactive setup.

This command creates the basic project structure and configuration file
by asking you a few questions about your site.

Example usage:
  markata-go init           # Interactive project setup
  markata-go init --force   # Overwrite existing files`,
	RunE: runInitCommand,
}

var (
	// initForce overwrites existing files.
	initForce bool
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing files")
}

// prompt displays a question and returns the user's response or a default value.
func prompt(reader *bufio.Reader, question, defaultVal string) string {
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

// promptYesNo displays a yes/no question and returns the boolean result.
func promptYesNo(reader *bufio.Reader, question string, defaultYes bool) bool {
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

func runInitCommand(_ *cobra.Command, _ []string) error {
	reader := bufio.NewReader(os.Stdin)

	// Check for existing config file
	if !initForce {
		if _, err := os.Stat("markata-go.toml"); err == nil {
			return fmt.Errorf("markata-go.toml already exists (use --force to overwrite)")
		}
	}

	fmt.Println()
	fmt.Println("Welcome to markata-go!")
	fmt.Println()

	// Gather site information
	title := prompt(reader, "Site title", "My Site")
	description := prompt(reader, "Description", "A site built with markata-go")
	author := prompt(reader, "Author", "")
	url := prompt(reader, "URL", "https://example.com")

	fmt.Println()
	fmt.Println("Creating project structure...")

	// Create directories
	dirs := []string{"posts", "static"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Printf("  ✓ Created %s/\n", dir)
	}

	// Generate and write config file
	configContent := generateInitConfig(title, description, author, url)
	if err := os.WriteFile("markata-go.toml", []byte(configContent), 0o644); err != nil { //nolint:gosec // config files should be readable
		return fmt.Errorf("failed to write markata-go.toml: %w", err)
	}
	fmt.Println("  ✓ Created markata-go.toml")

	fmt.Println()

	// Offer to create first post
	if promptYesNo(reader, "Create your first post?", true) {
		postTitle := prompt(reader, "Post title", "Hello World")

		slug := generateSlug(postTitle)
		filename := slug + ".md"
		fullPath := filepath.Join("posts", filename)

		// Check if file already exists
		if _, err := os.Stat(fullPath); err == nil && !initForce {
			fmt.Printf("  ! Post already exists: %s (skipped)\n", fullPath)
		} else {
			now := time.Now()
			content := generatePostContent(title, slug, now, false)
			if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil { //nolint:gosec // content files should be readable
				return fmt.Errorf("failed to write post: %w", err)
			}
			fmt.Printf("  ✓ Created %s\n", fullPath)
		}
	}

	fmt.Println()
	fmt.Println("Done! Run 'markata-go serve' to start.")
	fmt.Println()

	return nil
}

// generateInitConfig creates a TOML config string from the provided values.
func generateInitConfig(title, description, author, url string) string {
	return fmt.Sprintf(`# Markata-go configuration file

# Site metadata
title = %q
url = %q
description = %q
author = %q

# Output settings
output_dir = "output"
templates_dir = "templates"
assets_dir = "static"

# File discovery
[glob]
patterns = ["**/*.md"]
use_gitignore = true

# Feed defaults
[feed_defaults]
items_per_page = 10
orphan_threshold = 3

[feed_defaults.formats]
html = true
rss = true
atom = false
json = false

# Define custom feeds
# [[feeds]]
# slug = "blog"
# title = "Blog Posts"
# filter = "published == true"
# sort = "date"
# reverse = true
`, title, url, description, author)
}
