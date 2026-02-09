package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/WaylonWalker/markata-go/pkg/config"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/palettes"
)

// initWizardState holds the state gathered during the huh wizard.
type initWizardState struct {
	// Site info
	Title       string
	Description string
	Author      string
	URL         string

	// Feature selections
	ConfigureFeatures bool
	SelectedFeatures  []string

	// Theme config
	Palette string

	// SEO config
	TwitterHandle string
	DefaultImage  string
	LogoURL       string

	// Post formats
	EnableMarkdown bool
	EnableOG       bool

	// Advanced feeds
	EnableAtom bool
	EnableJSON bool

	// Vend options
	VendAssets         bool
	SelectedVendAssets []string

	// First post
	CreateFirstPost bool
	PostTitle       string
}

// createHuhTheme creates a huh theme based on the configured palette.
// If no palette is configured or loading fails, returns the default Charm theme.
func createHuhTheme(paletteName string) *huh.Theme {
	if paletteName == "" {
		return huh.ThemeCharm()
	}

	loader := palettes.NewLoader()
	palette, err := loader.Load(paletteName)
	if err != nil {
		return huh.ThemeCharm()
	}

	// Create a custom theme based on the palette
	theme := huh.ThemeCharm()

	// Map palette colors to huh theme
	if hex := palette.Resolve("accent"); hex != "" {
		accentColor := lipgloss.Color(hex)
		theme.Focused.Title = theme.Focused.Title.Foreground(accentColor)
		theme.Focused.SelectedOption = theme.Focused.SelectedOption.Foreground(accentColor)
		theme.Focused.FocusedButton = theme.Focused.FocusedButton.Background(accentColor)
	}

	if hex := palette.Resolve("text-muted"); hex != "" {
		mutedColor := lipgloss.Color(hex)
		theme.Focused.Description = theme.Focused.Description.Foreground(mutedColor)
		theme.Blurred.Title = theme.Blurred.Title.Foreground(mutedColor)
	}

	if hex := palette.Resolve("link"); hex != "" {
		linkColor := lipgloss.Color(hex)
		theme.Focused.Option = theme.Focused.Option.Foreground(linkColor)
	}

	return theme
}

// getPaletteOptions returns huh options for palette selection with search support.
func getPaletteOptions() []huh.Option[string] {
	loader := palettes.NewLoader()
	availablePalettes, err := loader.Discover()
	if err != nil || len(availablePalettes) == 0 {
		return []huh.Option[string]{
			huh.NewOption("default-light", "default-light"),
			huh.NewOption("default-dark", "default-dark"),
		}
	}

	// Sort palettes by name for consistent display
	sort.Slice(availablePalettes, func(i, j int) bool {
		return availablePalettes[i].Name < availablePalettes[j].Name
	})

	options := make([]huh.Option[string], 0, len(availablePalettes))
	for _, p := range availablePalettes {
		label := fmt.Sprintf("%s (%s)", p.Name, p.Variant)
		options = append(options, huh.NewOption(label, p.Name))
	}

	return options
}

// runHuhNewProjectWizard runs the interactive huh wizard for new projects.
func runHuhNewProjectWizard(theme *huh.Theme) (*initWizardState, error) {
	state := &initWizardState{
		Title:       "My Site",
		Description: "A site built with markata-go",
		URL:         "https://example.com",
		Palette:     "default-light",
	}

	// Group 1: Basic site information
	siteInfoGroup := huh.NewGroup(
		huh.NewNote().
			Title("Welcome to markata-go!").
			Description("Let's set up your new static site. Press Enter to continue."),
		huh.NewInput().
			Title("Site Title").
			Description("The name of your site").
			Value(&state.Title).
			Placeholder("My Site"),
		huh.NewInput().
			Title("Description").
			Description("A brief description of your site").
			Value(&state.Description).
			Placeholder("A site built with markata-go"),
		huh.NewInput().
			Title("Author").
			Description("Your name or organization").
			Value(&state.Author).
			Placeholder(""),
		huh.NewInput().
			Title("URL").
			Description("Your site's URL (used for RSS feeds and sitemaps)").
			Value(&state.URL).
			Placeholder("https://example.com"),
	)

	// Group 2: Feature configuration
	featureGroup := huh.NewGroup(
		huh.NewConfirm().
			Title("Configure additional features?").
			Description("You can configure themes, SEO, post formats, and feeds").
			Value(&state.ConfigureFeatures),
	)

	// Build the initial form
	form := huh.NewForm(siteInfoGroup, featureGroup).WithTheme(theme)
	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("wizard canceled: %w", err)
	}

	// If user wants to configure features, run additional forms
	if state.ConfigureFeatures {
		if err := runFeatureSelectionForm(state, theme); err != nil {
			return nil, err
		}
	}

	// Vending and first post options
	if err := runFinalOptionsForm(state, theme); err != nil {
		return nil, err
	}

	return state, nil
}

// runFeatureSelectionForm runs the feature selection wizard.
func runFeatureSelectionForm(state *initWizardState, theme *huh.Theme) error {
	featureOptions := []huh.Option[string]{
		huh.NewOption("Theme/Palette system (color schemes)", featureTheme),
		huh.NewOption("SEO metadata (Twitter, Open Graph)", featureSEO),
		huh.NewOption("Post output formats (markdown source, OG cards)", featurePostFormats),
		huh.NewOption("Advanced feeds (Atom, JSON Feed)", featureAdvancedFeed),
	}

	selectGroup := huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("Select features to configure").
			Description("Use space to select, enter to confirm").
			Options(featureOptions...).
			Value(&state.SelectedFeatures),
	)

	form := huh.NewForm(selectGroup).WithTheme(theme)
	if err := form.Run(); err != nil {
		return fmt.Errorf("feature selection canceled: %w", err)
	}

	// Configure each selected feature
	for _, feature := range state.SelectedFeatures {
		if err := runFeatureConfigForm(state, feature, theme); err != nil {
			return err
		}
	}

	return nil
}

// runFeatureConfigForm runs the configuration form for a specific feature.
func runFeatureConfigForm(state *initWizardState, feature string, theme *huh.Theme) error {
	switch feature {
	case featureTheme:
		return runThemeConfigForm(state, theme)
	case featureSEO:
		return runSEOConfigForm(state, theme)
	case featurePostFormats:
		return runPostFormatsConfigForm(state, theme)
	case featureAdvancedFeed:
		return runFeedsConfigForm(state, theme)
	}
	return nil
}

// runThemeConfigForm runs the theme/palette configuration form.
func runThemeConfigForm(state *initWizardState, theme *huh.Theme) error {
	paletteOptions := getPaletteOptions()

	group := huh.NewGroup(
		huh.NewNote().
			Title("Theme Configuration").
			Description("Choose a color palette for your site"),
		huh.NewSelect[string]().
			Title("Palette").
			Description("Select a color palette (type to search)").
			Options(paletteOptions...).
			Value(&state.Palette).
			Height(10),
	)

	form := huh.NewForm(group).WithTheme(theme)
	return form.Run()
}

// runSEOConfigForm runs the SEO configuration form.
func runSEOConfigForm(state *initWizardState, theme *huh.Theme) error {
	group := huh.NewGroup(
		huh.NewNote().
			Title("SEO Configuration").
			Description("Configure metadata for social sharing and search engines"),
		huh.NewInput().
			Title("Twitter/X Handle").
			Description("Without the @ symbol").
			Value(&state.TwitterHandle).
			Placeholder("yourhandle"),
		huh.NewInput().
			Title("Default Open Graph Image").
			Description("URL to the default image for social sharing").
			Value(&state.DefaultImage).
			Placeholder("https://example.com/og-image.png"),
		huh.NewInput().
			Title("Site Logo URL").
			Description("URL to your site logo (for Schema.org)").
			Value(&state.LogoURL).
			Placeholder("https://example.com/logo.png"),
	)

	form := huh.NewForm(group).WithTheme(theme)
	return form.Run()
}

// runPostFormatsConfigForm runs the post formats configuration form.
func runPostFormatsConfigForm(state *initWizardState, theme *huh.Theme) error {
	group := huh.NewGroup(
		huh.NewNote().
			Title("Post Output Formats").
			Description("Configure additional output formats for your posts"),
		huh.NewConfirm().
			Title("Enable Markdown source output?").
			Description("Generates /slug.md alongside HTML").
			Value(&state.EnableMarkdown),
		huh.NewConfirm().
			Title("Enable OG card output?").
			Description("Generates /slug/og/index.html for social image generation").
			Value(&state.EnableOG),
	)

	form := huh.NewForm(group).WithTheme(theme)
	return form.Run()
}

// runFeedsConfigForm runs the advanced feeds configuration form.
func runFeedsConfigForm(state *initWizardState, theme *huh.Theme) error {
	group := huh.NewGroup(
		huh.NewNote().
			Title("Advanced Feed Formats").
			Description("HTML and RSS feeds are enabled by default"),
		huh.NewConfirm().
			Title("Enable Atom feed output?").
			Value(&state.EnableAtom),
		huh.NewConfirm().
			Title("Enable JSON Feed output?").
			Value(&state.EnableJSON),
	)

	form := huh.NewForm(group).WithTheme(theme)
	return form.Run()
}

// runFinalOptionsForm runs the final options form (vending, first post).
func runFinalOptionsForm(state *initWizardState, theme *huh.Theme) error {
	group := huh.NewGroup(
		huh.NewConfirm().
			Title("Vend built-in assets for customization?").
			Description("Export templates, palettes, and layouts for local editing").
			Value(&state.VendAssets),
	)

	form := huh.NewForm(group).WithTheme(theme)
	if err := form.Run(); err != nil {
		return err
	}

	// If vending, ask what to vend
	if state.VendAssets {
		vendOptions := []huh.Option[string]{
			huh.NewOption("Content templates (for 'markata-go new')", "templates"),
			huh.NewOption("Color palettes (TOML files)", "palettes"),
			huh.NewOption("HTML layout templates (Jinja2/Pongo2)", "layouts"),
		}

		vendGroup := huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select assets to vend").
				Description("Use space to select, enter to confirm").
				Options(vendOptions...).
				Value(&state.SelectedVendAssets),
		)

		vendForm := huh.NewForm(vendGroup).WithTheme(theme)
		if err := vendForm.Run(); err != nil {
			return err
		}
	}

	// Ask about first post
	postGroup := huh.NewGroup(
		huh.NewConfirm().
			Title("Create your first post?").
			Value(&state.CreateFirstPost).
			Affirmative("Yes").
			Negative("No"),
	)

	postForm := huh.NewForm(postGroup).WithTheme(theme)
	if err := postForm.Run(); err != nil {
		return err
	}

	if state.CreateFirstPost {
		state.PostTitle = "Hello World"
		titleGroup := huh.NewGroup(
			huh.NewInput().
				Title("Post title").
				Value(&state.PostTitle).
				Placeholder("Hello World"),
		)

		titleForm := huh.NewForm(titleGroup).WithTheme(theme)
		if err := titleForm.Run(); err != nil {
			return err
		}
	}

	return nil
}

// applyWizardState applies the wizard state to create the project.
func applyWizardState(state *initWizardState, force bool) error {
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

	// Create config
	cfg := models.NewConfig()
	cfg.Title = state.Title
	cfg.Description = state.Description
	cfg.Author = state.Author
	cfg.URL = state.URL
	cfg.GlobConfig.Patterns = []string{"**/*.md"}
	cfg.GlobConfig.UseGitignore = true

	// Apply feature configurations
	for _, feature := range state.SelectedFeatures {
		switch feature {
		case featureTheme:
			cfg.Theme.Palette = state.Palette
		case featureSEO:
			cfg.SEO.TwitterHandle = strings.TrimPrefix(state.TwitterHandle, "@")
			cfg.SEO.DefaultImage = state.DefaultImage
			cfg.SEO.LogoURL = state.LogoURL
		case featurePostFormats:
			cfg.PostFormats.Markdown = state.EnableMarkdown
			cfg.PostFormats.OG = state.EnableOG
		case featureAdvancedFeed:
			cfg.FeedDefaults.Formats.Atom = state.EnableAtom
			cfg.FeedDefaults.Formats.JSON = state.EnableJSON
		}
	}

	// Write config file
	configPath := defaultConfigFilename
	if err := writeConfigTOML(configPath, cfg); err != nil {
		return fmt.Errorf("failed to write %s: %w", defaultConfigFilename, err)
	}
	fmt.Printf("  Created %s\n", defaultConfigFilename)

	// Vend assets if requested
	if state.VendAssets && len(state.SelectedVendAssets) > 0 {
		if err := runVendAssets(force, state.SelectedVendAssets); err != nil {
			return err
		}
		fmt.Println()
		fmt.Println("Assets vendored! Local files will override built-in defaults.")
	}

	// Create first post if requested
	if state.CreateFirstPost && state.PostTitle != "" {
		if err := createFirstPost(state.PostTitle, force); err != nil {
			return err
		}
	}

	fmt.Println()
	fmt.Println("Done! Run 'markata-go serve' to start.")
	fmt.Println()

	return nil
}

// runHuhExistingProjectWizard runs the wizard for existing projects using huh.
func runHuhExistingProjectWizard(configPath string, theme *huh.Theme, force bool) error {
	fmt.Println()
	fmt.Println("Found existing markata-go.toml")

	// Load existing config
	cfg, err := loadConfigForWizard(configPath)
	if err != nil {
		return fmt.Errorf("failed to load existing config: %w", err)
	}

	for {
		var choice string
		menuOptions := []huh.Option[string]{
			huh.NewOption("Add new features", "features"),
			huh.NewOption("Update site information", "site_info"),
			huh.NewOption("Vend built-in assets (templates, palettes, layouts)", "vend"),
			huh.NewOption("View current configuration", "view"),
			huh.NewOption("Exit", "exit"),
		}

		menuGroup := huh.NewGroup(
			huh.NewSelect[string]().
				Title("What would you like to do?").
				Options(menuOptions...).
				Value(&choice),
		)

		menuForm := huh.NewForm(menuGroup).WithTheme(theme)
		if err := menuForm.Run(); err != nil {
			return err
		}

		switch choice {
		case "features":
			if err := runAddFeaturesWizard(cfg, configPath, theme); err != nil {
				return err
			}
			return nil

		case "site_info":
			if err := runUpdateSiteInfoWizard(cfg, configPath, theme); err != nil {
				return err
			}
			return nil

		case "vend":
			if err := runVendWizard(theme, force); err != nil {
				return err
			}
			return nil

		case "view":
			displayCurrentConfig(cfg)
			// Continue the loop

		case "exit":
			fmt.Println("\nExiting without changes.")
			return nil
		}
	}
}

// runAddFeaturesWizard runs the feature addition wizard for existing projects.
func runAddFeaturesWizard(cfg *models.Config, configPath string, theme *huh.Theme) error {
	configured := detectConfiguredFeatures(cfg)
	features := getAvailableFeatures(configured)

	// Build options excluding already configured features
	var options []huh.Option[string]
	for _, f := range features {
		if !f.Configured {
			options = append(options, huh.NewOption(f.Description, f.Name))
		}
	}

	if len(options) == 0 {
		fmt.Println("\nAll features are already configured!")
		return nil
	}

	var selectedFeatures []string
	group := huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("Select features to add").
			Description("Use space to select, enter to confirm").
			Options(options...).
			Value(&selectedFeatures),
	)

	form := huh.NewForm(group).WithTheme(theme)
	if err := form.Run(); err != nil {
		return err
	}

	if len(selectedFeatures) == 0 {
		fmt.Println("\nNo features selected.")
		return nil
	}

	// Create a temporary state to hold feature configs
	state := &initWizardState{
		Palette: cfg.Theme.Palette,
	}

	// Configure each selected feature
	for _, feature := range selectedFeatures {
		if err := runFeatureConfigForm(state, feature, theme); err != nil {
			return err
		}
	}

	// Apply configurations to cfg
	for _, feature := range selectedFeatures {
		switch feature {
		case featureTheme:
			cfg.Theme.Palette = state.Palette
		case featureSEO:
			cfg.SEO.TwitterHandle = strings.TrimPrefix(state.TwitterHandle, "@")
			cfg.SEO.DefaultImage = state.DefaultImage
			cfg.SEO.LogoURL = state.LogoURL
		case featurePostFormats:
			cfg.PostFormats.Markdown = state.EnableMarkdown
			cfg.PostFormats.OG = state.EnableOG
		case featureAdvancedFeed:
			cfg.FeedDefaults.Formats.Atom = state.EnableAtom
			cfg.FeedDefaults.Formats.JSON = state.EnableJSON
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
}

// runUpdateSiteInfoWizard runs the site info update wizard.
func runUpdateSiteInfoWizard(cfg *models.Config, configPath string, theme *huh.Theme) error {
	group := huh.NewGroup(
		huh.NewNote().
			Title("Update Site Information"),
		huh.NewInput().
			Title("Site Title").
			Value(&cfg.Title),
		huh.NewInput().
			Title("Description").
			Value(&cfg.Description),
		huh.NewInput().
			Title("Author").
			Value(&cfg.Author),
		huh.NewInput().
			Title("URL").
			Value(&cfg.URL),
	)

	form := huh.NewForm(group).WithTheme(theme)
	if err := form.Run(); err != nil {
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
}

// runVendWizard runs the asset vending wizard.
func runVendWizard(theme *huh.Theme, force bool) error {
	vendOptions := []huh.Option[string]{
		huh.NewOption("Content templates (for 'markata-go new')", "templates"),
		huh.NewOption("Color palettes (TOML files)", "palettes"),
		huh.NewOption("HTML layout templates (Jinja2/Pongo2)", "layouts"),
	}

	var selected []string
	group := huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("Select assets to vend").
			Description("Use space to select, enter to confirm").
			Options(vendOptions...).
			Value(&selected),
	)

	form := huh.NewForm(group).WithTheme(theme)
	if err := form.Run(); err != nil {
		return err
	}

	if len(selected) == 0 {
		fmt.Println("\nNo assets selected.")
		return nil
	}

	if err := runVendAssets(force, selected); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Assets vendored successfully!")
	fmt.Println("Local files will now override built-in defaults.")

	return nil
}

// loadConfigForWizard loads config for the wizard, using the config package.
func loadConfigForWizard(configPath string) (*models.Config, error) {
	return config.Load(configPath)
}

// createFirstPost creates the first post for a new project.
func createFirstPost(postTitle string, force bool) error {
	slug := slugify(postTitle)
	filename := slug + ".md"
	fullPath := filepath.Join("posts", filename)

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil && !force {
		fmt.Printf("  ! Post already exists: %s (skipped)\n", fullPath)
		return nil
	}

	now := time.Now()
	content := fmt.Sprintf(`---
title: %q
slug: %s
date: %q
published: false
draft: true
templateKey: post
tags: []
description: ""
---

# %s

Write your content here...
`, postTitle, slug, now.Format("2006-01-02"), postTitle)

	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil { //nolint:gosec // content files should be readable
		return fmt.Errorf("failed to write post: %w", err)
	}
	fmt.Printf("  Created %s\n", fullPath)
	return nil
}

// slugify converts a title to a URL-safe slug.
func slugify(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	// Remove special characters
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return strings.Trim(result.String(), "-")
}
