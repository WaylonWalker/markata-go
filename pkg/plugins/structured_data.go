// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"encoding/json"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// StructuredDataPlugin generates JSON-LD Schema.org markup, OpenGraph meta tags,
// and Twitter Cards for SEO and social media optimization.
type StructuredDataPlugin struct{}

// NewStructuredDataPlugin creates a new StructuredDataPlugin.
func NewStructuredDataPlugin() *StructuredDataPlugin {
	return &StructuredDataPlugin{}
}

// Name returns the unique name of the plugin.
func (p *StructuredDataPlugin) Name() string {
	return "structured_data"
}

// Priority returns the plugin priority for the given stage.
// Runs in mid-transform, after description plugin has run.
func (p *StructuredDataPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageTransform {
		return lifecycle.PriorityDefault // 500
	}
	return lifecycle.PriorityDefault
}

// Transform generates structured data for each post.
func (p *StructuredDataPlugin) Transform(m *lifecycle.Manager) error {
	config := m.Config()

	// Check if structured data is enabled
	seoConfig := getSEOConfig(config)
	if !seoConfig.StructuredData.IsEnabled() {
		return nil
	}

	return m.ProcessPostsConcurrently(func(post *models.Post) error {
		if post.Skip || post.Draft {
			return nil
		}

		// Skip posts without titles (required for structured data)
		if post.Title == nil || *post.Title == "" {
			return nil
		}

		return p.generateStructuredData(post, config, &seoConfig)
	})
}

// generateStructuredData creates all structured data for a post.
func (p *StructuredDataPlugin) generateStructuredData(post *models.Post, config *lifecycle.Config, seoConfig *models.SEOConfig) error {
	sd := models.NewStructuredData()

	// Generate JSON-LD
	jsonLD, err := p.generateJSONLD(post, config, seoConfig)
	if err == nil && jsonLD != "" {
		sd.JSONLD = jsonLD
	}

	// Generate OpenGraph tags
	p.generateOpenGraph(sd, post, config, seoConfig)

	// Generate Twitter Card tags
	p.generateTwitterCard(sd, post, config, seoConfig)

	// Store in post.Extra
	post.Set("structured_data", sd)

	return nil
}

// generateJSONLD creates JSON-LD Schema.org markup for a post.
func (p *StructuredDataPlugin) generateJSONLD(post *models.Post, config *lifecycle.Config, seoConfig *models.SEOConfig) (string, error) {
	siteURL := getSiteURL(config)
	postURL := siteURL + post.Href

	// Create BlogPosting schema
	bp := models.NewBlogPosting(*post.Title, postURL)

	// Add description
	if post.Description != nil {
		bp.Description = *post.Description
	}

	// Add dates
	if post.Date != nil {
		bp.DatePublished = post.Date.Format("2006-01-02T15:04:05Z07:00")
		bp.DateModified = bp.DatePublished

		// Check for modified date in Extra
		if modified, ok := post.Extra["modified"]; ok {
			if modStr, ok := modified.(string); ok {
				bp.DateModified = modStr
			}
		}
	}

	// Add image
	imageURL := p.getPostImage(post, seoConfig)
	if imageURL != "" {
		bp.Image = p.makeAbsoluteURL(imageURL, siteURL)
	}

	// Add keywords from tags
	if len(post.Tags) > 0 {
		bp.Keywords = post.Tags
	}

	// Add author
	bp.Author = p.getAuthor(post, config, seoConfig)

	// Add publisher
	bp.Publisher = p.getPublisher(config, seoConfig)

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(bp, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// generateOpenGraph creates OpenGraph meta tags.
func (p *StructuredDataPlugin) generateOpenGraph(sd *models.StructuredData, post *models.Post, config *lifecycle.Config, seoConfig *models.SEOConfig) {
	siteURL := getSiteURL(config)
	siteTitle := getSiteTitle(config)
	siteDescription := getSiteDescription(config)
	postURL := siteURL + post.Href

	// Required tags
	sd.AddOpenGraph("og:title", *post.Title)
	sd.AddOpenGraph("og:url", postURL)
	sd.AddOpenGraph("og:site_name", siteTitle)

	// Type - article for posts with dates, website otherwise
	if post.Date != nil {
		sd.AddOpenGraph("og:type", "article")
	} else {
		sd.AddOpenGraph("og:type", "website")
	}

	// Description
	if post.Description != nil {
		sd.AddOpenGraph("og:description", *post.Description)
	} else if siteDescription != "" {
		sd.AddOpenGraph("og:description", siteDescription)
	}

	// Image
	imageURL := p.getPostImage(post, seoConfig)
	if imageURL != "" {
		absImageURL := p.makeAbsoluteURL(imageURL, siteURL)
		sd.AddOpenGraph("og:image", absImageURL)
		sd.AddOpenGraph("og:image:width", "1200")
		sd.AddOpenGraph("og:image:height", "630")
	}

	// Locale
	sd.AddOpenGraph("og:locale", "en_US")

	// Article-specific tags
	if post.Date != nil {
		sd.AddOpenGraph("article:published_time", post.Date.Format("2006-01-02T15:04:05Z07:00"))

		// Modified time
		if modified, ok := post.Extra["modified"]; ok {
			if modStr, ok := modified.(string); ok {
				sd.AddOpenGraph("article:modified_time", modStr)
			}
		}

		// Author URL
		author := p.getAuthor(post, config, seoConfig)
		if author != nil && author.URL != "" {
			sd.AddOpenGraph("article:author", author.URL)
		}

		// Tags
		for _, tag := range post.Tags {
			sd.AddOpenGraph("article:tag", tag)
		}
	}
}

// generateTwitterCard creates Twitter Card meta tags.
func (p *StructuredDataPlugin) generateTwitterCard(sd *models.StructuredData, post *models.Post, config *lifecycle.Config, seoConfig *models.SEOConfig) {
	siteURL := getSiteURL(config)

	// Card type - summary_large_image if we have an image
	imageURL := p.getPostImage(post, seoConfig)
	if imageURL != "" {
		sd.AddTwitter("twitter:card", "summary_large_image")
		sd.AddTwitter("twitter:image", p.makeAbsoluteURL(imageURL, siteURL))
	} else {
		sd.AddTwitter("twitter:card", "summary")
	}

	// Site handle
	if seoConfig.TwitterHandle != "" {
		sd.AddTwitter("twitter:site", "@"+seoConfig.TwitterHandle)
	}

	// Creator handle (use author's twitter if available, otherwise site handle)
	creatorHandle := p.getTwitterHandle(post, seoConfig)
	if creatorHandle != "" {
		sd.AddTwitter("twitter:creator", "@"+creatorHandle)
	}

	// Title
	sd.AddTwitter("twitter:title", *post.Title)

	// Description (truncated to 200 chars for Twitter)
	if post.Description != nil {
		desc := *post.Description
		if len(desc) > 200 {
			desc = desc[:197] + "..."
		}
		sd.AddTwitter("twitter:description", desc)
	}
}

// getPostImage returns the image URL for a post.
// Checks frontmatter image, social_image, then falls back to default.
func (p *StructuredDataPlugin) getPostImage(post *models.Post, seoConfig *models.SEOConfig) string {
	// Check for social_image override first
	if socialImage, ok := post.Extra["social_image"]; ok {
		if imgStr, ok := socialImage.(string); ok && imgStr != "" {
			return imgStr
		}
	}

	// Check for image in frontmatter
	if image, ok := post.Extra["image"]; ok {
		if imgStr, ok := image.(string); ok && imgStr != "" {
			return imgStr
		}
	}

	// Fall back to default image
	return seoConfig.DefaultImage
}

// getAuthor returns the author SchemaAgent for a post.
func (p *StructuredDataPlugin) getAuthor(post *models.Post, config *lifecycle.Config, seoConfig *models.SEOConfig) *models.SchemaAgent {
	// Check for author in frontmatter
	var authorName string
	if author, ok := post.Extra["author"]; ok {
		if authorStr, ok := author.(string); ok && authorStr != "" {
			authorName = authorStr
		}
	}

	// If we have a custom author name, create a basic agent
	if authorName != "" {
		return models.NewSchemaAgent("Person", authorName)
	}

	// Use default author from config
	if seoConfig.StructuredData.DefaultAuthor != nil {
		da := seoConfig.StructuredData.DefaultAuthor
		return models.NewSchemaAgent(da.Type, da.Name).WithURL(da.URL)
	}

	// Fall back to site author
	siteAuthor := getSiteAuthor(config)
	if siteAuthor != "" {
		return models.NewSchemaAgent("Person", siteAuthor)
	}

	return nil
}

// getPublisher returns the publisher SchemaAgent for the site.
func (p *StructuredDataPlugin) getPublisher(config *lifecycle.Config, seoConfig *models.SEOConfig) *models.SchemaAgent {
	siteURL := getSiteURL(config)

	// Use publisher from config
	if seoConfig.StructuredData.Publisher != nil {
		pub := seoConfig.StructuredData.Publisher
		agent := models.NewSchemaAgent(pub.Type, pub.Name).WithURL(pub.URL)
		if pub.Logo != "" {
			agent.WithLogo(p.makeAbsoluteURL(pub.Logo, siteURL))
		} else if seoConfig.LogoURL != "" {
			agent.WithLogo(p.makeAbsoluteURL(seoConfig.LogoURL, siteURL))
		}
		return agent
	}

	// Fall back to site title as Organization
	siteTitle := getSiteTitle(config)
	if siteTitle != "" {
		agent := models.NewSchemaAgent("Organization", siteTitle).WithURL(siteURL)
		if seoConfig.LogoURL != "" {
			agent.WithLogo(p.makeAbsoluteURL(seoConfig.LogoURL, siteURL))
		}
		return agent
	}

	return nil
}

// getTwitterHandle returns the Twitter handle for the post author or site.
func (p *StructuredDataPlugin) getTwitterHandle(post *models.Post, seoConfig *models.SEOConfig) string {
	// Check for author's twitter handle in frontmatter
	if twitterHandle, ok := post.Extra["twitter"]; ok {
		if handleStr, ok := twitterHandle.(string); ok && handleStr != "" {
			// Remove @ if present
			return strings.TrimPrefix(handleStr, "@")
		}
	}

	// Fall back to site handle
	return seoConfig.TwitterHandle
}

// makeAbsoluteURL converts a relative URL to an absolute URL.
func (p *StructuredDataPlugin) makeAbsoluteURL(url, siteURL string) string {
	if url == "" {
		return ""
	}

	// Already absolute
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}

	// Protocol-relative
	if strings.HasPrefix(url, "//") {
		return "https:" + url
	}

	// Relative URL - prepend site URL
	siteURL = strings.TrimSuffix(siteURL, "/")
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}
	return siteURL + url
}

// getSEOConfig retrieves the SEOConfig from lifecycle.Config.Extra.
func getSEOConfig(config *lifecycle.Config) models.SEOConfig {
	if config.Extra != nil {
		if seo, ok := config.Extra["seo"].(models.SEOConfig); ok {
			return seo
		}
	}
	return models.NewSEOConfig()
}

// Ensure StructuredDataPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*StructuredDataPlugin)(nil)
	_ lifecycle.TransformPlugin = (*StructuredDataPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*StructuredDataPlugin)(nil)
)
