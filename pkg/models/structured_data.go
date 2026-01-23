package models

// StructuredData holds generated structured data for a post.
// This is stored in post.Extra["structured_data"] after processing.
type StructuredData struct {
	// JSONLD is the JSON-LD script content (without script tags)
	JSONLD string `json:"jsonld" yaml:"jsonld" toml:"jsonld"`

	// OpenGraph contains OpenGraph meta tags
	OpenGraph []OpenGraphTag `json:"opengraph" yaml:"opengraph" toml:"opengraph"`

	// Twitter contains Twitter Card meta tags
	Twitter []TwitterTag `json:"twitter" yaml:"twitter" toml:"twitter"`
}

// OpenGraphTag represents an OpenGraph meta tag.
type OpenGraphTag struct {
	// Property is the og: property name (e.g., "og:title")
	Property string `json:"property" yaml:"property" toml:"property"`

	// Content is the tag content value
	Content string `json:"content" yaml:"content" toml:"content"`
}

// TwitterTag represents a Twitter Card meta tag.
type TwitterTag struct {
	// Name is the twitter: name (e.g., "twitter:card")
	Name string `json:"name" yaml:"name" toml:"name"`

	// Content is the tag content value
	Content string `json:"content" yaml:"content" toml:"content"`
}

// NewStructuredData creates a new empty StructuredData.
func NewStructuredData() *StructuredData {
	return &StructuredData{
		OpenGraph: []OpenGraphTag{},
		Twitter:   []TwitterTag{},
	}
}

// AddOpenGraph adds an OpenGraph tag.
func (s *StructuredData) AddOpenGraph(property, content string) {
	if content == "" {
		return
	}
	s.OpenGraph = append(s.OpenGraph, OpenGraphTag{
		Property: property,
		Content:  content,
	})
}

// AddTwitter adds a Twitter Card tag.
func (s *StructuredData) AddTwitter(name, content string) {
	if content == "" {
		return
	}
	s.Twitter = append(s.Twitter, TwitterTag{
		Name:    name,
		Content: content,
	})
}

// BlogPosting represents a Schema.org BlogPosting for JSON-LD.
type BlogPosting struct {
	Context          string       `json:"@context"`
	Type             string       `json:"@type"`
	Headline         string       `json:"headline"`
	Description      string       `json:"description,omitempty"`
	DatePublished    string       `json:"datePublished,omitempty"`
	DateModified     string       `json:"dateModified,omitempty"`
	Author           *SchemaAgent `json:"author,omitempty"`
	Publisher        *SchemaAgent `json:"publisher,omitempty"`
	MainEntityOfPage *WebPage     `json:"mainEntityOfPage,omitempty"`
	Image            string       `json:"image,omitempty"`
	Keywords         []string     `json:"keywords,omitempty"`
	URL              string       `json:"url,omitempty"`
}

// WebPage represents a Schema.org WebPage for JSON-LD.
type WebPage struct {
	Type string `json:"@type"`
	ID   string `json:"@id"`
}

// WebSite represents a Schema.org WebSite for JSON-LD.
type WebSite struct {
	Context     string       `json:"@context"`
	Type        string       `json:"@type"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	URL         string       `json:"url"`
	Publisher   *SchemaAgent `json:"publisher,omitempty"`
}

// SchemaAgent represents a Schema.org Person or Organization.
type SchemaAgent struct {
	Type string       `json:"@type"`
	Name string       `json:"name"`
	URL  string       `json:"url,omitempty"`
	Logo *ImageObject `json:"logo,omitempty"`
}

// ImageObject represents a Schema.org ImageObject.
type ImageObject struct {
	Type string `json:"@type"`
	URL  string `json:"url"`
}

// NewBlogPosting creates a new BlogPosting with required fields.
func NewBlogPosting(headline, url string) *BlogPosting {
	return &BlogPosting{
		Context:  "https://schema.org",
		Type:     "BlogPosting",
		Headline: headline,
		URL:      url,
		MainEntityOfPage: &WebPage{
			Type: "WebPage",
			ID:   url,
		},
	}
}

// NewWebSite creates a new WebSite with required fields.
func NewWebSite(name, url string) *WebSite {
	return &WebSite{
		Context: "https://schema.org",
		Type:    "WebSite",
		Name:    name,
		URL:     url,
	}
}

// NewSchemaAgent creates a new SchemaAgent (Person or Organization).
func NewSchemaAgent(agentType, name string) *SchemaAgent {
	if agentType == "" {
		agentType = "Organization"
	}
	return &SchemaAgent{
		Type: agentType,
		Name: name,
	}
}

// WithURL sets the URL on a SchemaAgent and returns it for chaining.
func (a *SchemaAgent) WithURL(url string) *SchemaAgent {
	a.URL = url
	return a
}

// WithLogo sets the logo on a SchemaAgent and returns it for chaining.
func (a *SchemaAgent) WithLogo(logoURL string) *SchemaAgent {
	if logoURL != "" {
		a.Logo = &ImageObject{
			Type: "ImageObject",
			URL:  logoURL,
		}
	}
	return a
}
