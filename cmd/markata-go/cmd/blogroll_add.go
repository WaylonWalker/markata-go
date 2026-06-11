package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/blogroll"
	"github.com/WaylonWalker/markata-go/pkg/config"
	"github.com/WaylonWalker/markata-go/pkg/models"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// Constants for blogroll add command.
const (
	defaultCategory     = "Uncategorized"
	promptDefaultYesStr = "Y/n"
	promptDefaultNoStr  = "y/N"
	youtubeFeedBaseURL  = "https://www.youtube.com/feeds/videos.xml?channel_id="
)

var (
	youtubeChannelIDPattern      = regexp.MustCompile(`UC[a-zA-Z0-9_-]{22}`)
	youtubeVideoIDPattern        = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)
	youtubeOEmbedEndpoint        = "https://www.youtube.com/oembed"
	blogrollAddHTTPClientFactory = func(timeout time.Duration) *http.Client { return &http.Client{Timeout: timeout} }
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
	rawInput := args[0]

	// Load config and check for duplicates
	configPath, cfg, err := loadConfigForBlogrollAdd()
	if err != nil {
		return err
	}

	feedURL, err := normalizeBlogrollAddInput(cfg, rawInput)
	if err != nil {
		return err
	}

	// Validate URL format
	if err := validateFeedURL(feedURL); err != nil {
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
		feedValues, err = promptForFeedValues(cfg, metadata, feedValues)
		if err != nil {
			return err
		}
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

type youtubeOEmbedResponse struct {
	AuthorURL string `json:"author_url"`
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

type metadataChoice struct {
	label       string
	title       string
	description string
	tags        []string
}

// validateFeedURL validates that the URL is a valid HTTP/HTTPS URL.
func validateFeedURL(feedURL string) error {
	parsedURL, err := url.Parse(feedURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return fmt.Errorf("invalid feed URL: must be a valid HTTP/HTTPS URL")
	}
	return nil
}

func normalizeBlogrollAddInput(cfg *models.Config, input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("invalid feed URL: must be a valid HTTP/HTTPS URL")
	}

	if strings.HasPrefix(strings.ToLower(trimmed), "yt:") {
		handle := strings.TrimSpace(trimmed[3:])
		if handle == "" {
			return "", fmt.Errorf("invalid youtube shortcut: expected yt:<handle>")
		}
		trimmed = "https://www.youtube.com/@" + strings.TrimPrefix(handle, "@")
	}

	parsedURL, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid feed URL: must be a valid HTTP/HTTPS URL")
	}

	if !isYouTubeURL(parsedURL) {
		return trimmed, nil
	}

	if strings.Contains(parsedURL.Path, "/feeds/videos.xml") {
		return trimmed, nil
	}

	if channelID := extractYouTubeChannelIDFromURL(trimmed); channelID != "" {
		return youtubeFeedBaseURL + channelID, nil
	}

	if isYouTubeVideoURL(parsedURL) {
		channelURL, err := fetchYouTubeAuthorURL(cfg, trimmed)
		if err != nil {
			return "", err
		}
		return resolveYouTubeChannelFeedURL(cfg, channelURL)
	}

	if isYouTubeChannelLikeURL(parsedURL) {
		return resolveYouTubeChannelFeedURL(cfg, trimmed)
	}

	return trimmed, nil
}

func resolveYouTubeChannelFeedURL(cfg *models.Config, rawURL string) (string, error) {
	if channelID := extractYouTubeChannelIDFromURL(rawURL); channelID != "" {
		return youtubeFeedBaseURL + channelID, nil
	}

	channelID, err := fetchYouTubeChannelID(cfg, rawURL)
	if err != nil {
		return "", err
	}

	return youtubeFeedBaseURL + channelID, nil
}

func fetchYouTubeAuthorURL(cfg *models.Config, videoURL string) (string, error) {
	timeout := blogrollAddTimeout(cfg)
	client := blogrollAddHTTPClientFactory(timeout)

	oembedURL := youtubeOEmbedEndpoint + "?url=" + url.QueryEscape(videoURL) + "&format=json"
	body, err := fetchBlogrollAddURL(client, oembedURL, "application/json")
	if err != nil {
		return "", fmt.Errorf("resolve youtube video channel: %w", err)
	}

	var response youtubeOEmbedResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("parse youtube oembed response: %w", err)
	}
	if response.AuthorURL == "" {
		return "", fmt.Errorf("resolve youtube video channel: author URL not found")
	}

	return response.AuthorURL, nil
}

func fetchYouTubeChannelID(cfg *models.Config, rawURL string) (string, error) {
	timeout := blogrollAddTimeout(cfg)
	client := blogrollAddHTTPClientFactory(timeout)
	body, err := fetchBlogrollAddURL(client, rawURL, "text/html,application/xhtml+xml")
	if err != nil {
		return "", fmt.Errorf("resolve youtube channel feed: %w", err)
	}

	channelID := extractYouTubeChannelIDFromHTML(string(body))
	if channelID == "" {
		return "", fmt.Errorf("resolve youtube channel feed: channel ID not found")
	}

	return channelID, nil
}

func fetchBlogrollAddURL(client *http.Client, targetURL, accept string) ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, targetURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	req.Header.Set("User-Agent", "markata-go/1.0 (Blogroll Add)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return body, nil
}

func blogrollAddTimeout(cfg *models.Config) time.Duration {
	timeout := 30 * time.Second
	if cfg != nil && cfg.Blogroll.Timeout > 0 {
		timeout = time.Duration(cfg.Blogroll.Timeout) * time.Second
	}
	return timeout
}

func isYouTubeURL(parsedURL *url.URL) bool {
	host := strings.ToLower(parsedURL.Host)
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "m.")
	return host == "youtube.com" || host == "youtu.be"
}

func isYouTubeVideoURL(parsedURL *url.URL) bool {
	host := strings.ToLower(parsedURL.Host)
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "m.")

	if host == "youtu.be" {
		videoID := strings.Trim(strings.TrimPrefix(parsedURL.Path, "/"), "/")
		return youtubeVideoIDPattern.MatchString(videoID)
	}

	if strings.HasPrefix(parsedURL.Path, "/watch") {
		return youtubeVideoIDPattern.MatchString(parsedURL.Query().Get("v"))
	}

	if strings.HasPrefix(parsedURL.Path, "/shorts/") {
		videoID := strings.TrimPrefix(parsedURL.Path, "/shorts/")
		videoID = strings.Trim(videoID, "/")
		return youtubeVideoIDPattern.MatchString(videoID)
	}

	return false
}

func isYouTubeChannelLikeURL(parsedURL *url.URL) bool {
	return strings.HasPrefix(parsedURL.Path, "/@") || strings.HasPrefix(parsedURL.Path, "/channel/") || strings.HasPrefix(parsedURL.Path, "/user/") || strings.HasPrefix(parsedURL.Path, "/c/")
}

func extractYouTubeChannelIDFromURL(rawURL string) string {
	if matches := youtubeChannelIDPattern.FindAllString(rawURL, -1); len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func extractYouTubeChannelIDFromHTML(html string) string {
	if idx := strings.Index(html, youtubeFeedBaseURL); idx >= 0 {
		candidate := html[idx+len(youtubeFeedBaseURL):]
		if match := youtubeChannelIDPattern.FindString(candidate); match != "" {
			return match
		}
	}

	return extractYouTubeChannelIDFromURL(html)
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
		title = firstNonEmpty(metadata.FeedTitle, metadata.Title)
	}

	description := blogrollAddDescription
	if description == "" {
		description = firstNonEmpty(metadata.FeedDescription, metadata.Description)
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

	tags := blogrollAddTags
	if len(tags) == 0 {
		tags = copyTags(firstNonEmptyTags(metadata.FeedTags, metadata.Tags))
	}

	return feedValues{
		title:       title,
		description: description,
		siteURL:     siteURL,
		category:    category,
		handle:      handle,
		tags:        tags,
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
func promptForFeedValues(cfg *models.Config, metadata *blogroll.Metadata, fv feedValues) (feedValues, error) {
	if inputIsTerminal() && outputIsTerminal() {
		selected, err := promptForMetadataChoice(cfg, metadata, fv)
		if err != nil {
			return fv, err
		}
		fv = selected
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println()

	fv.title = promptBlogroll(reader, "Title", fv.title)
	fv.description = promptBlogroll(reader, "Description", fv.description)
	fv.category = promptBlogroll(reader, "Category", fv.category)

	if len(blogrollAddTags) == 0 {
		tagsInput := promptBlogroll(reader, "Tags (comma-separated)", strings.Join(fv.tags, ", "))
		fv.tags = parseTagsBlogroll(tagsInput)
	}

	fv.siteURL = promptBlogroll(reader, "Site URL", fv.siteURL)
	fv.handle = promptBlogroll(reader, "Handle (for @mentions)", fv.handle)
	fv.active = promptYesNoBlogroll(reader, "Include in reader page?", fv.active)

	return fv, nil
}

func promptForMetadataChoice(cfg *models.Config, metadata *blogroll.Metadata, fv feedValues) (feedValues, error) {
	choices := buildMetadataChoices(metadata)
	if len(choices) < 2 {
		return fv, nil
	}

	selectedLabel := choices[0].label
	options := make([]huh.Option[string], 0, len(choices))
	for _, choice := range choices {
		options = append(options, huh.NewOption(choice.label, choice.label))
	}

	noteDescription := make([]string, 0, len(choices))
	for _, choice := range choices {
		noteDescription = append(noteDescription, formatMetadataChoice(choice))
	}

	paletteName := ""
	if cfg != nil {
		paletteName = cfg.Theme.Palette
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Metadata Source").
				Description(strings.Join(noteDescription, "\n\n")+"\n\nChoose a starting point. You can still edit the fields next."),
			huh.NewSelect[string]().
				Title("Use which metadata?").
				Options(options...).
				Value(&selectedLabel),
		),
	).WithTheme(createHuhTheme(paletteName))

	if err := form.Run(); err != nil {
		return fv, err
	}

	for _, choice := range choices {
		if choice.label != selectedLabel {
			continue
		}
		if blogrollAddTitle == "" {
			fv.title = choice.title
		}
		if blogrollAddDescription == "" {
			fv.description = choice.description
		}
		if len(blogrollAddTags) == 0 {
			fv.tags = copyTags(choice.tags)
		}
		break
	}

	return fv, nil
}

func buildMetadataChoices(metadata *blogroll.Metadata) []metadataChoice {
	feed := metadataChoice{
		label:       "Feed metadata",
		title:       metadata.FeedTitle,
		description: metadata.FeedDescription,
		tags:        copyTags(metadata.FeedTags),
	}
	site := metadataChoice{
		label:       "Site metadata",
		title:       metadata.Title,
		description: metadata.Description,
		tags:        copyTags(metadata.Tags),
	}

	choices := make([]metadataChoice, 0, 2)
	if hasMetadataChoice(feed) {
		choices = append(choices, feed)
	}
	if hasMetadataChoice(site) && !metadataChoicesEqual(feed, site) {
		choices = append(choices, site)
	}

	return choices
}

func hasMetadataChoice(choice metadataChoice) bool {
	return choice.title != "" || choice.description != "" || len(choice.tags) > 0
}

func metadataChoicesEqual(a, b metadataChoice) bool {
	if a.title != b.title || a.description != b.description || len(a.tags) != len(b.tags) {
		return false
	}
	for i := range a.tags {
		if a.tags[i] != b.tags[i] {
			return false
		}
	}
	return true
}

func formatMetadataChoice(choice metadataChoice) string {
	parts := []string{choice.label + ":"}
	parts = append(parts, "title="+formatMetadataField(choice.title))
	parts = append(parts, "description="+formatMetadataField(truncateMetadataField(choice.description, 80)))
	if len(choice.tags) == 0 {
		parts = append(parts, "tags=(none)")
	} else {
		parts = append(parts, "tags="+strings.Join(choice.tags, ", "))
	}
	return strings.Join(parts, "\n")
}

func truncateMetadataField(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen-3] + "..."
}

func formatMetadataField(value string) string {
	if value == "" {
		return "(empty)"
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyTags(options ...[]string) []string {
	for _, option := range options {
		if len(option) > 0 {
			return option
		}
	}
	return nil
}

func copyTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	return append([]string{}, tags...)
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
