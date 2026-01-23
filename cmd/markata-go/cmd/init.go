package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/WaylonWalker/markata-go/pkg/config"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/palettes"
)

// initCmd represents the init command.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new markata-go project",
	Long: `Initialize a new markata-go project with interactive setup.

This command creates the basic project structure and configuration file
by asking you a few questions about your site.

If a configuration file already exists, you can add new features or
update site information interactively.

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

// promptYesNo displays a yes/no question and returns the boolean result.
func promptYesNo(reader *bufio.Reader, question string, defaultYes bool) bool {
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

// featureInfo describes an available feature for the wizard.
type featureInfo struct {
	Name        string
	Description string
	Configured  bool
}

// detectConfiguredFeatures checks which features are already configured.
func detectConfiguredFeatures(cfg *models.Config) map[string]bool {
	features := make(map[string]bool)
	features["theme"] = cfg.Theme.Palette != "" && cfg.Theme.Palette != "default-light"
	features["seo"] = cfg.SEO.TwitterHandle != "" || cfg.SEO.DefaultImage != ""
	features["post_formats"] = cfg.PostFormats.Markdown || cfg.PostFormats.OG
	features["advanced_feeds"] = cfg.FeedDefaults.Formats.Atom || cfg.FeedDefaults.Formats.JSON
	return features
}

// getAvailableFeatures returns the list of features that can be added.
func getAvailableFeatures(configured map[string]bool) []featureInfo {
	allFeatures := []featureInfo{
		{
			Name:        "theme",
			Description: "Theme/Palette system (color schemes)",
			Configured:  configured["theme"],
		},
		{
			Name:        "seo",
			Description: "SEO metadata (Twitter, Open Graph)",
			Configured:  configured["seo"],
		},
		{
			Name:        "post_formats",
			Description: "Post output formats (markdown source, OG cards)",
			Configured:  configured["post_formats"],
		},
		{
			Name:        "advanced_feeds",
			Description: "Advanced feeds (Atom, JSON Feed)",
			Configured:  configured["advanced_feeds"],
		},
	}

	return allFeatures
}

// promptMenuChoice displays a numbered menu and returns the selected option.
func promptMenuChoice(reader *bufio.Reader, question string, options []string) int {
	fmt.Println()
	fmt.Println(question)
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}
	fmt.Print("\nEnter choice [1]: ")

	input, err := reader.ReadString('\n')
	if err != nil || strings.TrimSpace(input) == "" {
		return 0
	}

	var choice int
	_, err = fmt.Sscanf(strings.TrimSpace(input), "%d", &choice)
	if err != nil || choice < 1 || choice > len(options) {
		return 0
	}
	return choice - 1
}

// promptFeatureSelection displays checkboxes for feature selection.
func promptFeatureSelection(reader *bufio.Reader, features []featureInfo) []string {
	fmt.Println()
	fmt.Println("Select features to add (enter numbers separated by spaces):")
	fmt.Println()

	availableIdx := []int{}
	for i, f := range features {
		status := "[ ]"
		if f.Configured {
			status = "[x]"
			fmt.Printf("  %d) %s %s (already configured)\n", i+1, status, f.Description)
		} else {
			fmt.Printf("  %d) %s %s\n", i+1, status, f.Description)
			availableIdx = append(availableIdx, i)
		}
	}

	if len(availableIdx) == 0 {
		fmt.Println("\n  All features are already configured!")
		return nil
	}

	fmt.Print("\nEnter numbers (e.g., 1 3): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil
	}

	selected := []string{}
	parts := strings.Fields(strings.TrimSpace(input))
	for _, p := range parts {
		var idx int
		if _, err := fmt.Sscanf(p, "%d", &idx); err == nil {
			if idx >= 1 && idx <= len(features) && !features[idx-1].Configured {
				selected = append(selected, features[idx-1].Name)
			}
		}
	}

	return selected
}

// backupConfig creates a backup of the existing config file.
func backupConfig(path string) error {
	backupPath := path + ".backup"
	// If backup already exists, add timestamp
	if _, err := os.Stat(backupPath); err == nil {
		backupPath = path + "." + time.Now().Format("20060102-150405") + ".backup"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config for backup: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0o644); err != nil { //nolint:gosec // backup files should be readable
		return fmt.Errorf("failed to write backup: %w", err)
	}

	fmt.Printf("  Backed up config to %s\n", backupPath)
	return nil
}

// addFeatureTheme prompts for theme/palette configuration.
func addFeatureTheme(reader *bufio.Reader, cfg *models.Config) error {
	fmt.Println()
	fmt.Println("Theme/Palette Configuration")
	fmt.Println("----------------------------")

	// List some available palettes
	loader := palettes.NewLoader()
	availablePalettes, err := loader.Discover()
	if err != nil {
		fmt.Println("(Could not discover palettes, using default)")
	} else if len(availablePalettes) > 0 {
		fmt.Println("Available palettes:")
		// Show a sample of palettes
		shown := 0
		for _, p := range availablePalettes {
			if shown < 10 {
				variant := "light"
				if p.Variant == palettes.VariantDark {
					variant = "dark"
				}
				fmt.Printf("  - %s (%s)\n", p.Name, variant)
				shown++
			}
		}
		if len(availablePalettes) > 10 {
			fmt.Printf("  ... and %d more (run 'markata-go palette list' to see all)\n", len(availablePalettes)-10)
		}
	}

	palette := prompt(reader, "\nPalette name", "default-light")
	cfg.Theme.Palette = palette

	return nil
}

// addFeatureSEO prompts for SEO configuration.
func addFeatureSEO(reader *bufio.Reader, cfg *models.Config) error {
	fmt.Println()
	fmt.Println("SEO Configuration")
	fmt.Println("-----------------")

	handle := prompt(reader, "Twitter/X handle (without @)", cfg.SEO.TwitterHandle)
	if handle != "" {
		// Remove @ if user accidentally includes it
		handle = strings.TrimPrefix(handle, "@")
		cfg.SEO.TwitterHandle = handle
	}

	defaultImage := prompt(reader, "Default Open Graph image URL", cfg.SEO.DefaultImage)
	cfg.SEO.DefaultImage = defaultImage

	logoURL := prompt(reader, "Site logo URL (for Schema.org)", cfg.SEO.LogoURL)
	cfg.SEO.LogoURL = logoURL

	return nil
}

// addFeaturePostFormats prompts for post format configuration.
func addFeaturePostFormats(reader *bufio.Reader, cfg *models.Config) error {
	fmt.Println()
	fmt.Println("Post Output Formats")
	fmt.Println("-------------------")

	if promptYesNo(reader, "Enable markdown source output? (generates /slug/index.md)", false) {
		cfg.PostFormats.Markdown = true
	}

	if promptYesNo(reader, "Enable OG card output? (generates /slug/og/index.html for social images)", false) {
		cfg.PostFormats.OG = true
	}

	return nil
}

// addFeatureAdvancedFeeds prompts for advanced feed configuration.
func addFeatureAdvancedFeeds(reader *bufio.Reader, cfg *models.Config) error {
	fmt.Println()
	fmt.Println("Advanced Feed Formats")
	fmt.Println("---------------------")
	fmt.Println("HTML and RSS feeds are enabled by default.")

	if promptYesNo(reader, "Enable Atom feed output?", false) {
		cfg.FeedDefaults.Formats.Atom = true
	}

	if promptYesNo(reader, "Enable JSON Feed output?", false) {
		cfg.FeedDefaults.Formats.JSON = true
	}

	return nil
}

// addFeature adds a specific feature to the configuration.
func addFeature(reader *bufio.Reader, feature string, cfg *models.Config) error {
	switch feature {
	case "theme":
		return addFeatureTheme(reader, cfg)
	case "seo":
		return addFeatureSEO(reader, cfg)
	case "post_formats":
		return addFeaturePostFormats(reader, cfg)
	case "advanced_feeds":
		return addFeatureAdvancedFeeds(reader, cfg)
	}
	return nil
}

// displayCurrentConfig shows the current configuration.
func displayCurrentConfig(cfg *models.Config) {
	fmt.Println()
	fmt.Println("Current Configuration")
	fmt.Println("=====================")
	fmt.Println()
	fmt.Printf("Site Information:\n")
	fmt.Printf("  Title:       %s\n", cfg.Title)
	fmt.Printf("  URL:         %s\n", cfg.URL)
	fmt.Printf("  Description: %s\n", cfg.Description)
	fmt.Printf("  Author:      %s\n", cfg.Author)
	fmt.Println()
	fmt.Printf("Directories:\n")
	fmt.Printf("  Output:      %s\n", cfg.OutputDir)
	fmt.Printf("  Templates:   %s\n", cfg.TemplatesDir)
	fmt.Printf("  Assets:      %s\n", cfg.AssetsDir)
	fmt.Println()
	fmt.Printf("Theme:\n")
	fmt.Printf("  Palette:     %s\n", cfg.Theme.Palette)
	fmt.Println()
	fmt.Printf("SEO:\n")
	fmt.Printf("  Twitter:     %s\n", valueOrNone(cfg.SEO.TwitterHandle))
	fmt.Printf("  Default Img: %s\n", valueOrNone(cfg.SEO.DefaultImage))
	fmt.Println()
	fmt.Printf("Post Formats:\n")
	fmt.Printf("  HTML:        %v\n", cfg.PostFormats.IsHTMLEnabled())
	fmt.Printf("  Markdown:    %v\n", cfg.PostFormats.Markdown)
	fmt.Printf("  OG Cards:    %v\n", cfg.PostFormats.OG)
	fmt.Println()
	fmt.Printf("Feed Formats (defaults):\n")
	fmt.Printf("  HTML:        %v\n", cfg.FeedDefaults.Formats.HTML)
	fmt.Printf("  RSS:         %v\n", cfg.FeedDefaults.Formats.RSS)
	fmt.Printf("  Atom:        %v\n", cfg.FeedDefaults.Formats.Atom)
	fmt.Printf("  JSON:        %v\n", cfg.FeedDefaults.Formats.JSON)
	fmt.Println()
}

func valueOrNone(s string) string {
	if s == "" {
		return "(not set)"
	}
	return s
}

// updateSiteInfo prompts for updated site information.
func updateSiteInfo(reader *bufio.Reader, cfg *models.Config) error {
	fmt.Println()
	fmt.Println("Update Site Information")
	fmt.Println("-----------------------")

	cfg.Title = prompt(reader, "Site title", cfg.Title)
	cfg.Description = prompt(reader, "Description", cfg.Description)
	cfg.Author = prompt(reader, "Author", cfg.Author)
	cfg.URL = prompt(reader, "URL", cfg.URL)

	return nil
}

// writeConfigTOML writes the configuration to a TOML file.
func writeConfigTOML(path string, cfg *models.Config) error {
	// Build TOML content manually for better formatting
	var sb strings.Builder

	sb.WriteString("# Markata-go configuration file\n\n")

	sb.WriteString("# Site metadata\n")
	sb.WriteString(fmt.Sprintf("title = %q\n", cfg.Title))
	sb.WriteString(fmt.Sprintf("url = %q\n", cfg.URL))
	sb.WriteString(fmt.Sprintf("description = %q\n", cfg.Description))
	sb.WriteString(fmt.Sprintf("author = %q\n", cfg.Author))
	sb.WriteString("\n")

	sb.WriteString("# Output settings\n")
	sb.WriteString(fmt.Sprintf("output_dir = %q\n", cfg.OutputDir))
	sb.WriteString(fmt.Sprintf("templates_dir = %q\n", cfg.TemplatesDir))
	sb.WriteString(fmt.Sprintf("assets_dir = %q\n", cfg.AssetsDir))
	sb.WriteString("\n")

	// Glob config
	sb.WriteString("# File discovery\n")
	sb.WriteString("[glob]\n")
	if len(cfg.GlobConfig.Patterns) > 0 {
		sb.WriteString("patterns = [")
		for i, p := range cfg.GlobConfig.Patterns {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%q", p))
		}
		sb.WriteString("]\n")
	} else {
		sb.WriteString("patterns = [\"**/*.md\"]\n")
	}
	sb.WriteString(fmt.Sprintf("use_gitignore = %v\n", cfg.GlobConfig.UseGitignore))
	sb.WriteString("\n")

	// Theme config
	if cfg.Theme.Palette != "" {
		sb.WriteString("# Theme configuration\n")
		sb.WriteString("[theme]\n")
		if cfg.Theme.Name != "" && cfg.Theme.Name != "default" {
			sb.WriteString(fmt.Sprintf("name = %q\n", cfg.Theme.Name))
		}
		sb.WriteString(fmt.Sprintf("palette = %q\n", cfg.Theme.Palette))
		sb.WriteString("\n")
	}

	// SEO config
	if cfg.SEO.TwitterHandle != "" || cfg.SEO.DefaultImage != "" || cfg.SEO.LogoURL != "" {
		sb.WriteString("# SEO configuration\n")
		sb.WriteString("[seo]\n")
		if cfg.SEO.TwitterHandle != "" {
			sb.WriteString(fmt.Sprintf("twitter_handle = %q\n", cfg.SEO.TwitterHandle))
		}
		if cfg.SEO.DefaultImage != "" {
			sb.WriteString(fmt.Sprintf("default_image = %q\n", cfg.SEO.DefaultImage))
		}
		if cfg.SEO.LogoURL != "" {
			sb.WriteString(fmt.Sprintf("logo_url = %q\n", cfg.SEO.LogoURL))
		}
		sb.WriteString("\n")
	}

	// Post formats config
	if cfg.PostFormats.Markdown || cfg.PostFormats.OG {
		sb.WriteString("# Post output formats\n")
		sb.WriteString("[post_formats]\n")
		if cfg.PostFormats.Markdown {
			sb.WriteString("markdown = true\n")
		}
		if cfg.PostFormats.OG {
			sb.WriteString("og = true\n")
		}
		sb.WriteString("\n")
	}

	// Feed defaults
	sb.WriteString("# Feed defaults\n")
	sb.WriteString("[feed_defaults]\n")
	sb.WriteString(fmt.Sprintf("items_per_page = %d\n", cfg.FeedDefaults.ItemsPerPage))
	sb.WriteString(fmt.Sprintf("orphan_threshold = %d\n", cfg.FeedDefaults.OrphanThreshold))
	sb.WriteString("\n")

	sb.WriteString("[feed_defaults.formats]\n")
	sb.WriteString(fmt.Sprintf("html = %v\n", cfg.FeedDefaults.Formats.HTML))
	sb.WriteString(fmt.Sprintf("rss = %v\n", cfg.FeedDefaults.Formats.RSS))
	sb.WriteString(fmt.Sprintf("atom = %v\n", cfg.FeedDefaults.Formats.Atom))
	sb.WriteString(fmt.Sprintf("json = %v\n", cfg.FeedDefaults.Formats.JSON))
	sb.WriteString("\n")

	// Custom feeds (commented example)
	sb.WriteString("# Define custom feeds\n")
	sb.WriteString("# [[feeds]]\n")
	sb.WriteString("# slug = \"blog\"\n")
	sb.WriteString("# title = \"Blog Posts\"\n")
	sb.WriteString("# filter = \"published == true\"\n")
	sb.WriteString("# sort = \"date\"\n")
	sb.WriteString("# reverse = true\n")

	return os.WriteFile(path, []byte(sb.String()), 0o644) //nolint:gosec // config files should be readable
}

// runExistingProjectWizard runs the wizard for an existing project.
func runExistingProjectWizard(reader *bufio.Reader, configPath string) error {
	fmt.Println()
	fmt.Println("Found existing markata-go.toml")

	// Load existing config
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load existing config: %w", err)
	}

	menuOptions := []string{
		"Add new features",
		"Update site information",
		"View current configuration",
		"Exit",
	}

	for {
		choice := promptMenuChoice(reader, "What would you like to do?", menuOptions)

		switch choice {
		case 0: // Add new features
			configured := detectConfiguredFeatures(cfg)
			features := getAvailableFeatures(configured)
			selected := promptFeatureSelection(reader, features)

			if len(selected) == 0 {
				fmt.Println("\nNo features selected.")
				continue
			}

			// Add each selected feature
			for _, feature := range selected {
				if err := addFeature(reader, feature, cfg); err != nil {
					return fmt.Errorf("failed to configure %s: %w", feature, err)
				}
			}

			// Backup and write config
			if err := backupConfig(configPath); err != nil {
				fmt.Printf("  Warning: %v\n", err)
			}

			if err := writeConfigTOML(configPath, cfg); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			fmt.Println()
			fmt.Println("  Updated markata-go.toml with new features")
			fmt.Println()
			fmt.Println("Run 'markata-go build' to apply changes!")
			return nil

		case 1: // Update site information
			if err := updateSiteInfo(reader, cfg); err != nil {
				return err
			}

			// Backup and write config
			if err := backupConfig(configPath); err != nil {
				fmt.Printf("  Warning: %v\n", err)
			}

			if err := writeConfigTOML(configPath, cfg); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			fmt.Println()
			fmt.Println("  Updated markata-go.toml")
			fmt.Println()
			fmt.Println("Run 'markata-go build' to apply changes!")
			return nil

		case 2: // View current configuration
			displayCurrentConfig(cfg)
			// Continue the loop to show menu again

		case 3: // Exit
			fmt.Println("\nExiting without changes.")
			return nil
		}
	}
}

func runInitCommand(_ *cobra.Command, _ []string) error {
	reader := bufio.NewReader(os.Stdin)

	// Check for existing config file
	configPath := "markata-go.toml"
	if _, err := os.Stat(configPath); err == nil {
		if initForce {
			// Force mode - proceed with new project setup
			fmt.Println("--force specified, overwriting existing configuration")
		} else {
			// Run the wizard for existing projects
			return runExistingProjectWizard(reader, configPath)
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
		fmt.Printf("  Created %s/\n", dir)
	}

	// Create config with defaults and user input
	cfg := models.NewConfig()
	cfg.Title = title
	cfg.Description = description
	cfg.Author = author
	cfg.URL = url
	cfg.GlobConfig.Patterns = []string{"**/*.md"}
	cfg.GlobConfig.UseGitignore = true

	// Ask about optional features for new projects
	fmt.Println()
	if promptYesNo(reader, "Would you like to configure additional features?", false) {
		configured := detectConfiguredFeatures(cfg)
		features := getAvailableFeatures(configured)
		selected := promptFeatureSelection(reader, features)

		for _, feature := range selected {
			if err := addFeature(reader, feature, cfg); err != nil {
				return fmt.Errorf("failed to configure %s: %w", feature, err)
			}
		}
	}

	// Write config file
	if err := writeConfigTOML(configPath, cfg); err != nil {
		return fmt.Errorf("failed to write markata-go.toml: %w", err)
	}
	fmt.Println("  Created markata-go.toml")

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
			fmt.Printf("  Created %s\n", fullPath)
		}
	}

	fmt.Println()
	fmt.Println("Done! Run 'markata-go serve' to start.")
	fmt.Println()

	return nil
}
