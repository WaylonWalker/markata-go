// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/markata-go/pkg/lifecycle"
	"github.com/example/markata-go/pkg/models"
)

// PublishFeedsPlugin writes feeds to multiple output formats during the write stage.
type PublishFeedsPlugin struct{}

// NewPublishFeedsPlugin creates a new PublishFeedsPlugin.
func NewPublishFeedsPlugin() *PublishFeedsPlugin {
	return &PublishFeedsPlugin{}
}

// Name returns the unique name of the plugin.
func (p *PublishFeedsPlugin) Name() string {
	return "publish_feeds"
}

// Write generates and writes feed files in all configured formats.
func (p *PublishFeedsPlugin) Write(m *lifecycle.Manager) error {
	config := m.Config()
	outputDir := config.OutputDir

	// Get feed configs from cache
	var feedConfigs []models.FeedConfig
	if cached, ok := m.Cache().Get("feed_configs"); ok {
		if fcs, ok := cached.([]models.FeedConfig); ok {
			feedConfigs = fcs
		}
	}

	if len(feedConfigs) == 0 {
		return nil
	}

	for _, fc := range feedConfigs {
		if err := p.publishFeed(&fc, config, outputDir); err != nil {
			return fmt.Errorf("publishing feed %q: %w", fc.Slug, err)
		}
	}

	return nil
}

// publishFeed publishes a single feed in all configured formats.
func (p *PublishFeedsPlugin) publishFeed(fc *models.FeedConfig, config *lifecycle.Config, outputDir string) error {
	// Determine feed directory
	feedDir := filepath.Join(outputDir, fc.Slug)
	if fc.Slug == "" {
		feedDir = outputDir
	}

	// Create feed directory
	if err := os.MkdirAll(feedDir, 0755); err != nil {
		return fmt.Errorf("creating feed directory: %w", err)
	}

	// Publish HTML pages
	if fc.Formats.HTML {
		if err := p.publishHTMLPages(fc, config, feedDir); err != nil {
			return fmt.Errorf("publishing HTML: %w", err)
		}
	}

	// Publish RSS
	if fc.Formats.RSS {
		if err := p.publishRSS(fc, config, feedDir); err != nil {
			return fmt.Errorf("publishing RSS: %w", err)
		}
	}

	// Publish Atom
	if fc.Formats.Atom {
		if err := p.publishAtom(fc, config, feedDir); err != nil {
			return fmt.Errorf("publishing Atom: %w", err)
		}
	}

	// Publish JSON
	if fc.Formats.JSON {
		if err := p.publishJSON(fc, config, feedDir); err != nil {
			return fmt.Errorf("publishing JSON: %w", err)
		}
	}

	// Publish Markdown
	if fc.Formats.Markdown {
		if err := p.publishMarkdown(fc, feedDir); err != nil {
			return fmt.Errorf("publishing Markdown: %w", err)
		}
	}

	// Publish Text
	if fc.Formats.Text {
		if err := p.publishText(fc, feedDir); err != nil {
			return fmt.Errorf("publishing Text: %w", err)
		}
	}

	return nil
}

// publishHTMLPages publishes HTML pages for a paginated feed.
func (p *PublishFeedsPlugin) publishHTMLPages(fc *models.FeedConfig, config *lifecycle.Config, feedDir string) error {
	for _, page := range fc.Pages {
		// Determine output path
		var pagePath string
		if page.Number == 1 {
			pagePath = filepath.Join(feedDir, "index.html")
		} else {
			pageDir := filepath.Join(feedDir, "page", fmt.Sprintf("%d", page.Number))
			if err := os.MkdirAll(pageDir, 0755); err != nil {
				return fmt.Errorf("creating page directory: %w", err)
			}
			pagePath = filepath.Join(pageDir, "index.html")
		}

		// Generate HTML content
		html, err := p.generateFeedPageHTML(fc, &page, config)
		if err != nil {
			return fmt.Errorf("generating page %d: %w", page.Number, err)
		}

		// Write file
		if err := os.WriteFile(pagePath, []byte(html), 0644); err != nil {
			return fmt.Errorf("writing page %d: %w", page.Number, err)
		}
	}

	return nil
}

// generateFeedPageHTML generates HTML for a feed page.
func (p *PublishFeedsPlugin) generateFeedPageHTML(fc *models.FeedConfig, page *models.FeedPage, config *lifecycle.Config) (string, error) {
	siteURL := getSiteURL(config)
	siteTitle := getSiteTitle(config)

	title := fc.Title
	if title == "" {
		title = siteTitle
	}

	// Simple default template
	tmplStr := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <link rel="alternate" type="application/rss+xml" title="RSS Feed" href="rss.xml">
    <link rel="alternate" type="application/atom+xml" title="Atom Feed" href="atom.xml">
    <link rel="alternate" type="application/feed+json" title="JSON Feed" href="feed.json">
</head>
<body>
    <header>
        <h1>{{.Title}}</h1>
        {{if .Description}}<p>{{.Description}}</p>{{end}}
    </header>
    <main>
        <ul class="posts">
        {{range .Posts}}
            <li class="post">
                <a href="{{.Href}}">
                    <h2>{{if .Title}}{{.Title}}{{else}}{{.Slug}}{{end}}</h2>
                </a>
                {{if .Date}}<time datetime="{{.Date.Format "2006-01-02"}}">{{.Date.Format "January 2, 2006"}}</time>{{end}}
                {{if .Description}}<p>{{.Description}}</p>{{end}}
            </li>
        {{end}}
        </ul>
    </main>
    <nav class="pagination">
        {{if .HasPrev}}<a href="{{.PrevURL}}" rel="prev">Previous</a>{{end}}
        <span>Page {{.Number}}</span>
        {{if .HasNext}}<a href="{{.NextURL}}" rel="next">Next</a>{{end}}
    </nav>
    <footer>
        <p><a href="{{.SiteURL}}">{{.SiteTitle}}</a></p>
    </footer>
</body>
</html>`

	tmpl, err := template.New("feed").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	// Prepare template data
	data := struct {
		Title       string
		Description string
		Posts       []*models.Post
		HasPrev     bool
		HasNext     bool
		PrevURL     string
		NextURL     string
		Number      int
		SiteURL     string
		SiteTitle   string
	}{
		Title:       title,
		Description: fc.Description,
		Posts:       page.Posts,
		HasPrev:     page.HasPrev,
		HasNext:     page.HasNext,
		PrevURL:     page.PrevURL,
		NextURL:     page.NextURL,
		Number:      page.Number,
		SiteURL:     siteURL,
		SiteTitle:   siteTitle,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}

// publishRSS generates and writes an RSS feed.
func (p *PublishFeedsPlugin) publishRSS(fc *models.FeedConfig, config *lifecycle.Config, feedDir string) error {
	rss, err := GenerateRSSFromFeedConfig(fc, config)
	if err != nil {
		return err
	}

	rssPath := filepath.Join(feedDir, "rss.xml")
	return os.WriteFile(rssPath, []byte(rss), 0644)
}

// publishAtom generates and writes an Atom feed.
func (p *PublishFeedsPlugin) publishAtom(fc *models.FeedConfig, config *lifecycle.Config, feedDir string) error {
	atom, err := GenerateAtomFromFeedConfig(fc, config)
	if err != nil {
		return err
	}

	atomPath := filepath.Join(feedDir, "atom.xml")
	return os.WriteFile(atomPath, []byte(atom), 0644)
}

// publishJSON generates and writes a JSON feed.
func (p *PublishFeedsPlugin) publishJSON(fc *models.FeedConfig, config *lifecycle.Config, feedDir string) error {
	jsonFeed, err := GenerateJSONFeedFromFeedConfig(fc, config)
	if err != nil {
		return err
	}

	jsonPath := filepath.Join(feedDir, "feed.json")
	return os.WriteFile(jsonPath, []byte(jsonFeed), 0644)
}

// publishMarkdown generates and writes a Markdown feed listing.
func (p *PublishFeedsPlugin) publishMarkdown(fc *models.FeedConfig, feedDir string) error {
	var sb strings.Builder

	// Title
	title := fc.Title
	if title == "" {
		title = "Posts"
	}
	sb.WriteString("# " + title + "\n\n")

	// Description
	if fc.Description != "" {
		sb.WriteString(fc.Description + "\n\n")
	}

	// Posts list
	for _, post := range fc.Posts {
		postTitle := post.Slug
		if post.Title != nil {
			postTitle = *post.Title
		}

		sb.WriteString("- [" + postTitle + "](" + post.Href + ")")

		if post.Date != nil {
			sb.WriteString(" - " + post.Date.Format("2006-01-02"))
		}

		sb.WriteString("\n")
	}

	mdPath := filepath.Join(feedDir, "index.md")
	return os.WriteFile(mdPath, []byte(sb.String()), 0644)
}

// publishText generates and writes a plain text feed listing.
func (p *PublishFeedsPlugin) publishText(fc *models.FeedConfig, feedDir string) error {
	var sb strings.Builder

	// Title
	title := fc.Title
	if title == "" {
		title = "Posts"
	}
	sb.WriteString(title + "\n")
	sb.WriteString(strings.Repeat("=", len(title)) + "\n\n")

	// Description
	if fc.Description != "" {
		sb.WriteString(fc.Description + "\n\n")
	}

	// Posts list
	for _, post := range fc.Posts {
		postTitle := post.Slug
		if post.Title != nil {
			postTitle = *post.Title
		}

		if post.Date != nil {
			sb.WriteString(post.Date.Format("2006-01-02") + " - ")
		}

		sb.WriteString(postTitle + "\n")
		sb.WriteString("  " + post.Href + "\n\n")
	}

	txtPath := filepath.Join(feedDir, "index.txt")
	return os.WriteFile(txtPath, []byte(sb.String()), 0644)
}

// Ensure PublishFeedsPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin      = (*PublishFeedsPlugin)(nil)
	_ lifecycle.WritePlugin = (*PublishFeedsPlugin)(nil)
)
