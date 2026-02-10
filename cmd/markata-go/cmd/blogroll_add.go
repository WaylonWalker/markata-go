package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/WaylonWalker/markata-go/pkg/blogroll"
	"github.com/WaylonWalker/markata-go/pkg/config"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// Constants for blogroll add command.
const (
	defaultCategory     = "Uncategorized"
	promptDefaultYesStr = "Y/n"
	promptDefaultNoStr  = "y/N"
)

// Flags for blogroll add command.
var (
	blogrollAddTitle       string
	blogrollAddDescription string
	blogrollAddCategory    string
	blogrollAddTags        []string
	blogrollAddSiteURL     string
	blogrollAddHandle      string
	blogrollAddActive      bool
	blogrollAddDryRun      bool
	blogrollAddNoPrompt    bool
)

// blogrollAddCmd adds a new feed to the blogroll.
var blogrollAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a new external feed to blogroll",
	Long: `Add a new RSS/Atom feed to your blogroll by providing the feed URL.

The command will:
1. Fetch and validate the feed
2. Extract metadata (title, description, site URL)
3. Prompt for optional fields (category, tags)
4. Add the feed to your config

Example:
  markata-go blogroll add https://example.com/rss.xml
  markata-go blogroll add https://example.com/atom.xml --category tech
  markata-go blogroll add https://example.com/feed --no-prompt
  markata-go blogroll add https://example.com/feed --dry-run
  markata-go blogroll add https://example.com/feed --title "My Blog" --tags "go,programming"`,
	Args: cobra.ExactArgs(1),
	RunE: runBlogrollAdd,
}

func init() {
	blogrollCmd.AddCommand(blogrollAddCmd)

	blogrollAddCmd.Flags().StringVar(&blogrollAddTitle, "title", "", "feed title (auto-fetched if not set)")
	blogrollAddCmd.Flags().StringVar(&blogrollAddDescription, "description", "", "feed description (auto-fetched if not set)")
	blogrollAddCmd.Flags().StringVar(&blogrollAddCategory, "category", "", "category for grouping (default: "+defaultCategory+")")
	blogrollAddCmd.Flags().StringSliceVar(&blogrollAddTags, "tags", nil, "comma-separated tags")
	blogrollAddCmd.Flags().StringVar(&blogrollAddSiteURL, "site-url", "", "main website URL (auto-fetched if not set)")
	blogrollAddCmd.Flags().StringVar(&blogrollAddHandle, "handle", "", "handle for @mentions (auto-generated if not set)")
	blogrollAddCmd.Flags().BoolVar(&blogrollAddActive, "active", true, "include in reader page")
	blogrollAddCmd.Flags().BoolVar(&blogrollAddDryRun, "dry-run", false, "preview changes without modifying config")
	blogrollAddCmd.Flags().BoolVar(&blogrollAddNoPrompt, "no-prompt", false, "skip interactive prompts, use defaults")
}

func runBlogrollAdd(_ *cobra.Command, args []string) error {
	feedURL := args[0]

	// Validate URL format
	if err := validateFeedURL(feedURL); err != nil {
		return err
	}

	// Load config and check for duplicates
	configPath, cfg, err := loadConfigForBlogrollAdd()
	if err != nil {
		return err
	}

	if err := checkDuplicateFeedURL(cfg, feedURL); err != nil {
		return err
	}

	// Fetch feed metadata
	metadata, err := fetchFeedMetadataForAdd(cfg, feedURL)
	if err != nil {
		return err
	}

	// Build feed values from flags and metadata
	feedValues := buildFeedValues(metadata, feedURL)

	// Check for duplicate handle
	if err := checkDuplicateHandle(cfg, feedValues.handle, feedValues.title); err != nil {
		return err
	}

	// Display fetched info
	displayFetchedInfo(feedValues.title, feedValues.description)

	// Interactive prompts if not disabled
	if !blogrollAddNoPrompt {
		feedValues = promptForFeedValues(feedValues)
	}

	// Build the feed config
	feedConfig := buildFeedConfig(feedURL, feedValues, metadata)

	// Display what will be added
	fmt.Println()
	printFeedConfig(feedConfig)

	if blogrollAddDryRun {
		fmt.Println("\n(dry-run mode - no changes written)")
		return nil
	}

	// Add feed to config and write
	return saveFeedToConfig(configPath, cfg, feedConfig)
}

// feedValues holds the values for building a feed config.
type feedValues struct {
	title       string
	description string
	siteURL     string
	category    string
	handle      string
	tags        []string
	active      bool
}

// validateFeedURL validates that the URL is a valid HTTP/HTTPS URL.
func validateFeedURL(feedURL string) error {
	parsedURL, err := url.Parse(feedURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return fmt.Errorf("invalid feed URL: must be a valid HTTP/HTTPS URL")
	}
	return nil
}

// loadConfigForBlogrollAdd loads the config file for adding a feed.
func loadConfigForBlogrollAdd() (string, *models.Config, error) {
	configPath := cfgFile
	if configPath == "" {
		var err error
		configPath, err = config.Discover()
		if err != nil {
			return "", nil, fmt.Errorf("no config file found: run 'markata-go init' first")
		}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load config: %w", err)
	}

	return configPath, cfg, nil
}

// checkDuplicateFeedURL checks if a feed with the given URL already exists.
func checkDuplicateFeedURL(cfg *models.Config, feedURL string) error {
	for i := range cfg.Blogroll.Feeds {
		if cfg.Blogroll.Feeds[i].URL == feedURL {
			return fmt.Errorf("feed with URL %q already exists (use 'blogroll update' to refresh metadata)", feedURL)
		}
	}
	return nil
}

// fetchFeedMetadataForAdd fetches metadata from the feed URL.
func fetchFeedMetadataForAdd(cfg *models.Config, feedURL string) (*blogroll.Metadata, error) {
	fmt.Printf("Fetching feed from %s...\n", feedURL)

	timeout := time.Duration(cfg.Blogroll.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	updater := blogroll.NewUpdater(timeout)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	metadata, err := updater.FetchMetadata(ctx, feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed metadata: %w", err)
	}

	return metadata, nil
}

// buildFeedValues builds the feed values from flags and metadata.
func buildFeedValues(metadata *blogroll.Metadata, feedURL string) feedValues {
	title := blogrollAddTitle
	if title == "" {
		title = metadata.Title
		if title == "" {
			title = metadata.FeedTitle
		}
	}

	description := blogrollAddDescription
	if description == "" {
		description = metadata.Description
		if description == "" {
			description = metadata.FeedDescription
		}
	}

	siteURL := blogrollAddSiteURL
	if siteURL == "" {
		siteURL = metadata.SiteURL
	}

	category := blogrollAddCategory
	if category == "" {
		category = defaultCategory
	}

	handle := blogrollAddHandle
	if handle == "" {
		handle = generateHandle(title, feedURL)
	}

	return feedValues{
		title:       title,
		description: description,
		siteURL:     siteURL,
		category:    category,
		handle:      handle,
		tags:        blogrollAddTags,
		active:      blogrollAddActive,
	}
}

// checkDuplicateHandle checks if a handle is already in use.
func checkDuplicateHandle(cfg *models.Config, handle, title string) error {
	for i := range cfg.Blogroll.Feeds {
		feed := &cfg.Blogroll.Feeds[i]
		if feed.Handle != "" && feed.Handle == handle {
			return fmt.Errorf("handle %q already in use by %q. Use --handle to specify a different handle", handle, feed.Title)
		}
	}
	_ = title // used in error message but handle check is separate
	return nil
}

// displayFetchedInfo displays the fetched feed info.
func displayFetchedInfo(title, description string) {
	if title != "" || description != "" {
		fmt.Printf("\nFetched feed: %q", title)
		if description != "" {
			truncDesc := description
			if len(truncDesc) > 60 {
				truncDesc = truncDesc[:57] + "..."
			}
			fmt.Printf(" - %q", truncDesc)
		}
		fmt.Println()
	}
}

// promptForFeedValues prompts the user for feed values interactively.
func promptForFeedValues(fv feedValues) feedValues {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println()

	fv.title = promptBlogroll(reader, "Title", fv.title)
	fv.description = promptBlogroll(reader, "Description", fv.description)
	fv.category = promptBlogroll(reader, "Category", fv.category)

	if fv.tags == nil {
		tagsInput := promptBlogroll(reader, "Tags (comma-separated)", "")
		if tagsInput != "" {
			fv.tags = parseTagsBlogroll(tagsInput)
		}
	}

	fv.siteURL = promptBlogroll(reader, "Site URL", fv.siteURL)
	fv.handle = promptBlogroll(reader, "Handle (for @mentions)", fv.handle)
	fv.active = promptYesNoBlogroll(reader, "Include in reader page?", fv.active)

	return fv
}

// buildFeedConfig builds the ExternalFeedConfig from feed values.
func buildFeedConfig(feedURL string, fv feedValues, metadata *blogroll.Metadata) models.ExternalFeedConfig {
	feedConfig := models.ExternalFeedConfig{
		URL:         feedURL,
		Title:       fv.title,
		Description: fv.description,
		Category:    fv.category,
		Tags:        fv.tags,
		SiteURL:     fv.siteURL,
		Handle:      fv.handle,
		Active:      &fv.active,
	}

	// Add image URL if fetched
	if metadata.ImageURL != "" {
		feedConfig.ImageURL = metadata.ImageURL
	}

	return feedConfig
}

// saveFeedToConfig adds the feed to the config and writes it.
func saveFeedToConfig(configPath string, cfg *models.Config, feedConfig models.ExternalFeedConfig) error {
	cfg.Blogroll.Feeds = append(cfg.Blogroll.Feeds, feedConfig)

	// Enable blogroll if not already enabled
	if !cfg.Blogroll.Enabled {
		cfg.Blogroll.Enabled = true
		fmt.Println("\nEnabled blogroll in config (was disabled)")
	}

	// Write updated config
	if err := writeConfigUpdate(configPath, cfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("\nFeed added to %s\n", configPath)
	return nil
}

// promptBlogroll displays a prompt and returns user input or default.
func promptBlogroll(reader *bufio.Reader, question, defaultVal string) string {
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

// promptYesNoBlogroll displays a yes/no prompt.
func promptYesNoBlogroll(reader *bufio.Reader, question string, defaultYes bool) bool {
	defaultStr := promptDefaultNoStr
	if defaultYes {
		defaultStr = promptDefaultYesStr
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
	return input == "y" || input == statusYes
}

// generateHandle creates a handle from title or URL.
func generateHandle(title, feedURL string) string {
	// Try to generate from title first
	if title != "" {
		handle := strings.ToLower(title)
		// Remove common suffixes
		handle = strings.TrimSuffix(handle, "'s blog")
		handle = strings.TrimSuffix(handle, " blog")
		// Replace spaces with nothing and remove non-alphanumeric
		var result strings.Builder
		for _, r := range handle {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				result.WriteRune(r)
			}
		}
		if result.Len() > 0 {
			return result.String()
		}
	}

	// Fall back to domain name
	parsedURL, err := url.Parse(feedURL)
	if err != nil {
		return ""
	}

	host := parsedURL.Host
	host = strings.TrimPrefix(host, "www.")

	// Extract first part of domain
	parts := strings.Split(host, ".")
	if len(parts) > 0 {
		return parts[0]
	}

	return ""
}

// parseTagsBlogroll parses a comma-separated string into a slice of tags.
func parseTagsBlogroll(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}

// printFeedConfig displays the feed configuration that will be added.
func printFeedConfig(feed models.ExternalFeedConfig) {
	ext := strings.ToLower(filepath.Ext(cfgFile))
	if ext == "" {
		// Auto-detect from discovered config
		configPath, err := config.Discover()
		if err == nil {
			ext = strings.ToLower(filepath.Ext(configPath))
		}
	}

	switch ext {
	case extYAML, extYML:
		printFeedConfigYAML(feed)
	default:
		printFeedConfigTOML(feed)
	}
}

// printFeedConfigTOML prints the feed config in TOML format.
func printFeedConfigTOML(feed models.ExternalFeedConfig) {
	fmt.Println("[[blogroll.feeds]]")
	fmt.Printf("url = %q\n", feed.URL)
	if feed.Title != "" {
		fmt.Printf("title = %q\n", feed.Title)
	}
	if feed.Description != "" {
		fmt.Printf("description = %q\n", feed.Description)
	}
	if feed.Category != "" && feed.Category != defaultCategory {
		fmt.Printf("category = %q\n", feed.Category)
	}
	if len(feed.Tags) > 0 {
		fmt.Printf("tags = %s\n", formatTOMLArray(feed.Tags))
	}
	if feed.SiteURL != "" {
		fmt.Printf("site_url = %q\n", feed.SiteURL)
	}
	if feed.Handle != "" {
		fmt.Printf("handle = %q\n", feed.Handle)
	}
	if feed.ImageURL != "" {
		fmt.Printf("image_url = %q\n", feed.ImageURL)
	}
	if feed.Active != nil && !*feed.Active {
		fmt.Printf("active = %v\n", *feed.Active)
	}
}

// printFeedConfigYAML prints the feed config in YAML format.
func printFeedConfigYAML(feed models.ExternalFeedConfig) {
	fmt.Println("- url:", feed.URL)
	if feed.Title != "" {
		fmt.Println("  title:", feed.Title)
	}
	if feed.Description != "" {
		fmt.Println("  description:", feed.Description)
	}
	if feed.Category != "" && feed.Category != defaultCategory {
		fmt.Println("  category:", feed.Category)
	}
	if len(feed.Tags) > 0 {
		fmt.Print("  tags: [")
		for i, t := range feed.Tags {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(t)
		}
		fmt.Println("]")
	}
	if feed.SiteURL != "" {
		fmt.Println("  site_url:", feed.SiteURL)
	}
	if feed.Handle != "" {
		fmt.Println("  handle:", feed.Handle)
	}
	if feed.ImageURL != "" {
		fmt.Println("  image_url:", feed.ImageURL)
	}
	if feed.Active != nil && !*feed.Active {
		fmt.Printf("  active: %v\n", *feed.Active)
	}
}

// formatTOMLArray formats a string slice as a TOML array.
func formatTOMLArray(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("%q", item)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
