package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	// newDir is the directory for new posts (overrides template placement).
	newDir string

	// newDraft creates the post as a draft.
	newDraft bool

	// newTags is a comma-separated list of tags.
	newTags string

	// newTemplate specifies the content template to use.
	newTemplate string

	// newList lists available templates.
	newList bool
)

// ContentTemplate represents a content template with its configuration.
type ContentTemplate struct {
	Name        string
	Directory   string
	Frontmatter map[string]interface{}
	Body        string
	Source      string // "builtin", "config", or "file"
}

// builtinTemplates returns the default built-in content templates.
func builtinTemplates() map[string]ContentTemplate {
	return map[string]ContentTemplate{
		"post": {
			Name:      "post",
			Directory: "posts",
			Frontmatter: map[string]interface{}{
				"template": "post",
			},
			Body:   "Write your content here...",
			Source: "builtin",
		},
		"page": {
			Name:      "page",
			Directory: "pages",
			Frontmatter: map[string]interface{}{
				"template": "page",
			},
			Body:   "Write your page content here...",
			Source: "builtin",
		},
		"docs": {
			Name:      "docs",
			Directory: "docs",
			Frontmatter: map[string]interface{}{
				"template": "docs",
			},
			Body:   "Write your documentation here...",
			Source: "builtin",
		},
		"article": {
			Name:      "article",
			Directory: "pages/article",
			Frontmatter: map[string]interface{}{
				"template": "article",
			},
			Body:   "Write your article here...",
			Source: "builtin",
		},
		"note": {
			Name:      "note",
			Directory: "pages/note",
			Frontmatter: map[string]interface{}{
				"template": "note",
			},
			Body:   "A quick note...",
			Source: "builtin",
		},
		"photo": {
			Name:      "photo",
			Directory: "pages/photo",
			Frontmatter: map[string]interface{}{
				"template": "photo",
				"image":    "",
			},
			Body:   "Photo caption...",
			Source: "builtin",
		},
		"video": {
			Name:      "video",
			Directory: "pages/video",
			Frontmatter: map[string]interface{}{
				"template":  "video",
				"video_url": "",
			},
			Body:   "Video description...",
			Source: "builtin",
		},
		"link": {
			Name:      "link",
			Directory: "pages/link",
			Frontmatter: map[string]interface{}{
				"template": "link",
				"url":      "",
			},
			Body:   "Why I'm sharing this link...",
			Source: "builtin",
		},
		"quote": {
			Name:      "quote",
			Directory: "pages/quote",
			Frontmatter: map[string]interface{}{
				"template":     "quote",
				"quote_author": "",
			},
			Body:   "> The quote goes here...",
			Source: "builtin",
		},
		"guide": {
			Name:      "guide",
			Directory: "pages/guide",
			Frontmatter: map[string]interface{}{
				"template": "guide",
			},
			Body:   "## Introduction\n\nWrite your guide here...",
			Source: "builtin",
		},
		"inline": {
			Name:      "inline",
			Directory: "pages/inline",
			Frontmatter: map[string]interface{}{
				"template": "inline",
			},
			Body:   "Inline content...",
			Source: "builtin",
		},
	}
}

// newCmd represents the new command.
var newCmd = &cobra.Command{
	Use:   "new [title]",
	Short: "Create a new content file",
	Long: `Create a new markdown content file with frontmatter template.

The command generates a new markdown file with:
  - Title set from the argument (or prompted if not provided)
  - Slug generated from the title
  - Current date
  - Draft status (configurable)
  - Tags (optional)
  - Template-specific frontmatter and placement

Template System:
  Templates control the default frontmatter and output directory for new content.
  Built-in templates: post, page, docs, article, note, photo, video, link, quote, guide, inline

  Custom templates can be defined:
  1. In markata-go.toml under [content_templates]
  2. As markdown files in the content-templates/ directory

Example usage:
  markata-go new "My First Post"                    # Create posts/my-first-post.md (default: post)
  markata-go new "About" --template page            # Create pages/about.md
  markata-go new "Getting Started" --template docs  # Create docs/getting-started.md
  markata-go new "Hello World" --dir blog           # Override directory: blog/hello-world.md
  markata-go new --list                             # List available templates
  markata-go new                                    # Interactive mode`,
	Args: cobra.MaximumNArgs(1),
	RunE: runNewCommand,
}

func init() {
	rootCmd.AddCommand(newCmd)

	newCmd.Flags().StringVar(&newDir, "dir", "", "directory for new content (overrides template placement)")
	newCmd.Flags().BoolVar(&newDraft, "draft", true, "create as draft")
	newCmd.Flags().StringVar(&newTags, "tags", "", "comma-separated list of tags")
	newCmd.Flags().StringVarP(&newTemplate, "template", "t", "post", "content template to use")
	newCmd.Flags().BoolVarP(&newList, "list", "l", false, "list available templates")
}

// loadTemplates discovers and loads all available content templates.
func loadTemplates() map[string]ContentTemplate {
	templates := builtinTemplates()

	// Load templates from config if available
	cfg := loadConfigSafe()
	if cfg != nil {
		// Apply placement overrides from config
		for name, dir := range cfg.ContentTemplates.Placement {
			if t, exists := templates[name]; exists {
				t.Directory = dir
				templates[name] = t
			}
		}

		// Add/override templates from config
		for _, ct := range cfg.ContentTemplates.Templates {
			templates[ct.Name] = ContentTemplate{
				Name:        ct.Name,
				Directory:   ct.Directory,
				Frontmatter: ct.Frontmatter,
				Body:        ct.Body,
				Source:      "config",
			}
		}

		// Load templates from content-templates directory
		templatesDir := cfg.ContentTemplates.Directory
		if templatesDir == "" {
			templatesDir = "content-templates"
		}
		loadTemplatesFromDir(templatesDir, templates)
	} else {
		// No config, try default directory
		loadTemplatesFromDir("content-templates", templates)
	}

	return templates
}

// loadTemplatesFromDir loads templates from markdown files in a directory.
func loadTemplatesFromDir(dir string, templates map[string]ContentTemplate) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Directory doesn't exist or can't be read - that's fine
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		template := parseTemplateFile(name, string(content))
		template.Source = "file"
		templates[name] = template
	}
}

// parseTemplateFile parses a markdown template file with frontmatter.
func parseTemplateFile(name, content string) ContentTemplate {
	template := ContentTemplate{
		Name:        name,
		Directory:   name, // Default directory is same as template name
		Frontmatter: make(map[string]interface{}),
		Body:        "",
	}

	// Check for frontmatter
	if !strings.HasPrefix(content, "---") {
		template.Body = strings.TrimSpace(content)
		return template
	}

	// Find end of frontmatter
	parts := strings.SplitN(content[3:], "---", 2)
	if len(parts) < 2 {
		template.Body = strings.TrimSpace(content)
		return template
	}

	// Parse frontmatter YAML
	frontmatterYAML := strings.TrimSpace(parts[0])
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &template.Frontmatter); err == nil {
		// Extract directory from frontmatter if present
		if dir, ok := template.Frontmatter["_directory"].(string); ok {
			template.Directory = dir
			delete(template.Frontmatter, "_directory")
		}
	}

	// Body is everything after frontmatter, but preserve the template markers
	template.Body = strings.TrimSpace(parts[1])

	return template
}

// loadConfigSafe attempts to load the config without errors.
func loadConfigSafe() *configWrapper {
	// Try to find and parse config file
	configPaths := []string{
		"markata-go.toml",
		"markata-go.yaml",
		"markata-go.yml",
		"markata-go.json",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			cfg, err := parseConfigFile(path)
			if err == nil {
				return cfg
			}
		}
	}
	return nil
}

// configWrapper wraps the content templates config for safe loading.
type configWrapper struct {
	ContentTemplates struct {
		Directory string            `yaml:"directory" toml:"directory" json:"directory"`
		Placement map[string]string `yaml:"placement" toml:"placement" json:"placement"`
		Templates []struct {
			Name        string                 `yaml:"name" toml:"name" json:"name"`
			Directory   string                 `yaml:"directory" toml:"directory" json:"directory"`
			Frontmatter map[string]interface{} `yaml:"frontmatter" toml:"frontmatter" json:"frontmatter"`
			Body        string                 `yaml:"body" toml:"body" json:"body"`
		} `yaml:"templates" toml:"templates" json:"templates"`
	} `yaml:"content_templates" toml:"content_templates" json:"content_templates"`
}

// parseConfigFile parses a config file to extract content templates config.
func parseConfigFile(path string) (*configWrapper, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg configWrapper
	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(content, &cfg)
	case ".toml":
		err = toml.Unmarshal(content, &cfg)
	case ".json":
		err = json.Unmarshal(content, &cfg)
	default:
		return nil, fmt.Errorf("unsupported config format: %s", ext)
	}

	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// interactiveInput holds the collected input from interactive mode.
type interactiveInput struct {
	title    string
	template ContentTemplate
	tags     []string
}

// runInteractiveMode prompts the user for input when no title is provided.
func runInteractiveMode(cmd *cobra.Command, templates map[string]ContentTemplate) (*interactiveInput, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()

	// Get title
	title := promptNew(reader, "Title", "")
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}

	template := templates[newTemplate]

	// Get template (only prompt if not explicitly set via flag)
	if !cmd.Flags().Changed("template") {
		templateNames := make([]string, 0, len(templates))
		for name := range templates {
			templateNames = append(templateNames, name)
		}
		sort.Strings(templateNames)
		fmt.Printf("Available templates: %s\n", strings.Join(templateNames, ", "))
		templateInput := promptNew(reader, "Template", "post")
		if t, ok := templates[templateInput]; ok {
			newTemplate = templateInput
			template = t
		}
	}

	// Get directory (only prompt if not explicitly set via flag)
	if !cmd.Flags().Changed("dir") {
		dirDefault := template.Directory
		newDir = promptNew(reader, "Directory", dirDefault)
	}

	// Get tags (only prompt if not explicitly set via flag)
	var tags []string
	if !cmd.Flags().Changed("tags") {
		tagsInput := promptNew(reader, "Tags (comma-separated)", "")
		if tagsInput != "" {
			tags = parseTags(tagsInput)
		}
	} else if newTags != "" {
		tags = parseTags(newTags)
	}

	// Get draft status (only prompt if not explicitly set via flag)
	if !cmd.Flags().Changed("draft") {
		newDraft = promptYesNoNew(reader, "Create as draft?", true)
	}

	fmt.Println()

	return &interactiveInput{
		title:    title,
		template: template,
		tags:     tags,
	}, nil
}

// writeContentFile creates the content file with the given parameters.
func writeContentFile(title, slug, outputDir string, draft bool, tags []string, template ContentTemplate) error {
	filename := slug + ".md"
	fullPath := filepath.Join(outputDir, filename)

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Errorf("file already exists: %s", fullPath)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate content
	now := time.Now()
	content := generateTemplatedContent(title, slug, now, draft, tags, template)

	// Write file (0o644 is appropriate for content files that should be world-readable)
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil { //nolint:gosec // content files should be readable
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Created: %s\n", fullPath)
	if verbose {
		fmt.Printf("  Template: %s\n", newTemplate)
		fmt.Printf("  Title: %s\n", title)
		fmt.Printf("  Slug: %s\n", slug)
		fmt.Printf("  Date: %s\n", now.Format("2006-01-02"))
		fmt.Printf("  Draft: %t\n", draft)
		if len(tags) > 0 {
			fmt.Printf("  Tags: %s\n", strings.Join(tags, ", "))
		}
	}

	return nil
}

func runNewCommand(cmd *cobra.Command, args []string) error {
	// Handle --list flag
	if newList {
		return listTemplates()
	}

	// Load templates
	templates := loadTemplates()

	// Validate template exists
	template, exists := templates[newTemplate]
	if !exists {
		availableNames := make([]string, 0, len(templates))
		for name := range templates {
			availableNames = append(availableNames, name)
		}
		sort.Strings(availableNames)
		return fmt.Errorf("unknown template %q; available templates: %s", newTemplate, strings.Join(availableNames, ", "))
	}

	var title string
	var tags []string

	// If no title provided, run interactive mode
	if len(args) == 0 {
		input, err := runInteractiveMode(cmd, templates)
		if err != nil {
			return err
		}
		title = input.title
		template = input.template
		tags = input.tags
	} else {
		title = args[0]
		if newTags != "" {
			tags = parseTags(newTags)
		}
	}

	// Determine output directory
	outputDir := template.Directory
	if cmd.Flags().Changed("dir") || newDir != "" {
		outputDir = newDir
	}
	if outputDir == "" {
		outputDir = template.Directory
	}

	// Generate slug from title
	slug := generateSlug(title)

	return writeContentFile(title, slug, outputDir, newDraft, tags, template)
}

// listTemplates prints available templates.
func listTemplates() error {
	templates := loadTemplates()

	if len(templates) == 0 {
		fmt.Println("No templates available.")
		return nil
	}

	// Sort template names
	names := make([]string, 0, len(templates))
	for name := range templates {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Println("Available content templates:")
	fmt.Println()
	for _, name := range names {
		t := templates[name]
		fmt.Printf("  %-12s -> %s/  (%s)\n", name, t.Directory, t.Source)
	}
	fmt.Println()
	fmt.Println("Use --template <name> or -t <name> to select a template.")
	return nil
}

// generateTemplatedContent creates the markdown content with template-specific frontmatter.
func generateTemplatedContent(title, slug string, date time.Time, draft bool, tags []string, template ContentTemplate) string {
	published := !draft

	// Build frontmatter map
	fm := make(map[string]interface{})

	// Add template-specific frontmatter first (can be overridden)
	for k, v := range template.Frontmatter {
		fm[k] = v
	}

	// Add standard fields (these override template defaults)
	fm["title"] = title
	fm["slug"] = slug
	fm["date"] = date.Format("2006-01-02")
	fm["published"] = published
	fm["draft"] = draft

	// Handle tags
	if len(tags) > 0 {
		fm["tags"] = tags
	} else if _, exists := fm["tags"]; !exists {
		fm["tags"] = []string{}
	}

	// Add description if not present
	if _, exists := fm["description"]; !exists {
		fm["description"] = ""
	}

	// Serialize frontmatter to YAML
	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		// Fallback to simple format
		return generatePostContentWithTags(title, slug, date, draft, tags)
	}

	// Build content
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(fmBytes)
	sb.WriteString("---\n\n")
	sb.WriteString("# ")
	sb.WriteString(title)
	sb.WriteString("\n\n")

	// Use template body or default
	body := template.Body
	if body == "" {
		body = "Write your content here..."
	}
	sb.WriteString(body)
	sb.WriteString("\n")

	return sb.String()
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
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultVal
	}
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
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}
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
