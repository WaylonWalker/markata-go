package models

import (
	"testing"
)

// =============================================================================
// StructuredData Tests
// =============================================================================

func TestNewStructuredData(t *testing.T) {
	sd := NewStructuredData()

	if sd == nil {
		t.Fatal("NewStructuredData() returned nil")
	}
	if sd.JSONLD != "" {
		t.Errorf("JSONLD: got %q, want empty string", sd.JSONLD)
	}
	if sd.OpenGraph == nil {
		t.Error("OpenGraph should be initialized")
	}
	if len(sd.OpenGraph) != 0 {
		t.Errorf("OpenGraph: got %d items, want 0", len(sd.OpenGraph))
	}
	if sd.Twitter == nil {
		t.Error("Twitter should be initialized")
	}
	if len(sd.Twitter) != 0 {
		t.Errorf("Twitter: got %d items, want 0", len(sd.Twitter))
	}
}

func TestStructuredData_AddOpenGraph(t *testing.T) {
	tests := []struct {
		name        string
		property    string
		content     string
		wantAdded   bool
		description string
	}{
		{
			name:        "add valid OpenGraph tag",
			property:    "og:title",
			content:     "My Page Title",
			wantAdded:   true,
			description: "valid tag should be added",
		},
		{
			name:        "skip empty content",
			property:    "og:title",
			content:     "",
			wantAdded:   false,
			description: "empty content should not be added",
		},
		{
			name:        "add tag with special characters",
			property:    "og:description",
			content:     "Content with <html> & \"quotes\"",
			wantAdded:   true,
			description: "special characters should be preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := NewStructuredData()
			initialCount := len(sd.OpenGraph)

			sd.AddOpenGraph(tt.property, tt.content)

			if tt.wantAdded {
				if len(sd.OpenGraph) != initialCount+1 {
					t.Errorf("AddOpenGraph() should have added tag, got %d items", len(sd.OpenGraph))
				}
				lastTag := sd.OpenGraph[len(sd.OpenGraph)-1]
				if lastTag.Property != tt.property {
					t.Errorf("Property: got %q, want %q", lastTag.Property, tt.property)
				}
				if lastTag.Content != tt.content {
					t.Errorf("Content: got %q, want %q", lastTag.Content, tt.content)
				}
			} else if len(sd.OpenGraph) != initialCount {
				t.Errorf("AddOpenGraph() should not have added tag, got %d items", len(sd.OpenGraph))
			}
		})
	}
}

func TestStructuredData_AddOpenGraph_Multiple(t *testing.T) {
	sd := NewStructuredData()

	sd.AddOpenGraph("og:title", "Title")
	sd.AddOpenGraph("og:description", "Description")
	sd.AddOpenGraph("og:image", "https://example.com/image.jpg")

	if len(sd.OpenGraph) != 3 {
		t.Errorf("OpenGraph count: got %d, want 3", len(sd.OpenGraph))
	}
}

func TestStructuredData_AddTwitter(t *testing.T) {
	tests := []struct {
		name        string
		tagName     string
		content     string
		wantAdded   bool
		description string
	}{
		{
			name:        "add valid Twitter card tag",
			tagName:     "twitter:card",
			content:     "summary_large_image",
			wantAdded:   true,
			description: "valid tag should be added",
		},
		{
			name:        "skip empty content",
			tagName:     "twitter:title",
			content:     "",
			wantAdded:   false,
			description: "empty content should not be added",
		},
		{
			name:        "add twitter creator",
			tagName:     "twitter:creator",
			content:     "@username",
			wantAdded:   true,
			description: "twitter handles should be preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := NewStructuredData()
			initialCount := len(sd.Twitter)

			sd.AddTwitter(tt.tagName, tt.content)

			if tt.wantAdded {
				if len(sd.Twitter) != initialCount+1 {
					t.Errorf("AddTwitter() should have added tag, got %d items", len(sd.Twitter))
				}
				lastTag := sd.Twitter[len(sd.Twitter)-1]
				if lastTag.Name != tt.tagName {
					t.Errorf("Name: got %q, want %q", lastTag.Name, tt.tagName)
				}
				if lastTag.Content != tt.content {
					t.Errorf("Content: got %q, want %q", lastTag.Content, tt.content)
				}
			} else if len(sd.Twitter) != initialCount {
				t.Errorf("AddTwitter() should not have added tag, got %d items", len(sd.Twitter))
			}
		})
	}
}

func TestStructuredData_AddTwitter_Multiple(t *testing.T) {
	sd := NewStructuredData()

	sd.AddTwitter("twitter:card", "summary")
	sd.AddTwitter("twitter:title", "My Title")
	sd.AddTwitter("twitter:description", "My Description")

	if len(sd.Twitter) != 3 {
		t.Errorf("Twitter count: got %d, want 3", len(sd.Twitter))
	}
}

// =============================================================================
// BlogPosting Tests
// =============================================================================

func TestNewBlogPosting(t *testing.T) {
	headline := "My Blog Post"
	url := "https://example.com/posts/my-blog-post/"

	bp := NewBlogPosting(headline, url)

	if bp == nil {
		t.Fatal("NewBlogPosting() returned nil")
	}
	if bp.Context != "https://schema.org" {
		t.Errorf("Context: got %q, want %q", bp.Context, "https://schema.org")
	}
	if bp.Type != "BlogPosting" {
		t.Errorf("Type: got %q, want %q", bp.Type, "BlogPosting")
	}
	if bp.Headline != headline {
		t.Errorf("Headline: got %q, want %q", bp.Headline, headline)
	}
	if bp.URL != url {
		t.Errorf("URL: got %q, want %q", bp.URL, url)
	}
	if bp.MainEntityOfPage == nil {
		t.Fatal("MainEntityOfPage should not be nil")
	}
	if bp.MainEntityOfPage.Type != "WebPage" {
		t.Errorf("MainEntityOfPage.Type: got %q, want %q", bp.MainEntityOfPage.Type, "WebPage")
	}
	if bp.MainEntityOfPage.ID != url {
		t.Errorf("MainEntityOfPage.ID: got %q, want %q", bp.MainEntityOfPage.ID, url)
	}
}

func TestNewBlogPosting_EmptyValues(t *testing.T) {
	bp := NewBlogPosting("", "")

	if bp.Headline != "" {
		t.Error("Headline should be empty")
	}
	if bp.URL != "" {
		t.Error("URL should be empty")
	}
	// MainEntityOfPage should still be created
	if bp.MainEntityOfPage == nil {
		t.Fatal("MainEntityOfPage should not be nil")
	}
}

// =============================================================================
// WebSite Tests
// =============================================================================

func TestNewWebSite(t *testing.T) {
	name := "My Website"
	url := "https://example.com"

	ws := NewWebSite(name, url)

	if ws == nil {
		t.Fatal("NewWebSite() returned nil")
	}
	if ws.Context != "https://schema.org" {
		t.Errorf("Context: got %q, want %q", ws.Context, "https://schema.org")
	}
	if ws.Type != "WebSite" {
		t.Errorf("Type: got %q, want %q", ws.Type, "WebSite")
	}
	if ws.Name != name {
		t.Errorf("Name: got %q, want %q", ws.Name, name)
	}
	if ws.URL != url {
		t.Errorf("URL: got %q, want %q", ws.URL, url)
	}
	if ws.Publisher != nil {
		t.Error("Publisher should be nil by default")
	}
}

func TestNewWebSite_EmptyValues(t *testing.T) {
	ws := NewWebSite("", "")

	if ws.Name != "" {
		t.Error("Name should be empty")
	}
	if ws.URL != "" {
		t.Error("URL should be empty")
	}
}

// =============================================================================
// SchemaAgent Tests
// =============================================================================

func TestNewSchemaAgent(t *testing.T) {
	tests := []struct {
		name      string
		agentType string
		agentName string
		wantType  string
		wantName  string
	}{
		{
			name:      "person agent",
			agentType: "Person",
			agentName: "John Doe",
			wantType:  "Person",
			wantName:  "John Doe",
		},
		{
			name:      "organization agent",
			agentType: "Organization",
			agentName: "Acme Corp",
			wantType:  "Organization",
			wantName:  "Acme Corp",
		},
		{
			name:      "empty type defaults to Organization",
			agentType: "",
			agentName: "Default Org",
			wantType:  "Organization",
			wantName:  "Default Org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewSchemaAgent(tt.agentType, tt.agentName)

			if agent == nil {
				t.Fatal("NewSchemaAgent() returned nil")
			}
			if agent.Type != tt.wantType {
				t.Errorf("Type: got %q, want %q", agent.Type, tt.wantType)
			}
			if agent.Name != tt.wantName {
				t.Errorf("Name: got %q, want %q", agent.Name, tt.wantName)
			}
			if agent.URL != "" {
				t.Errorf("URL should be empty, got %q", agent.URL)
			}
			if agent.Logo != nil {
				t.Error("Logo should be nil by default")
			}
		})
	}
}

func TestSchemaAgent_WithURL(t *testing.T) {
	agent := NewSchemaAgent("Person", "John Doe")
	url := "https://example.com/about"

	result := agent.WithURL(url)

	// Should return the same agent for chaining
	if result != agent {
		t.Error("WithURL() should return the same agent")
	}
	if agent.URL != url {
		t.Errorf("URL: got %q, want %q", agent.URL, url)
	}
}

func TestSchemaAgent_WithURL_Chaining(t *testing.T) {
	url := "https://example.com"

	agent := NewSchemaAgent("Organization", "Acme").WithURL(url)

	if agent.URL != url {
		t.Errorf("URL: got %q, want %q", agent.URL, url)
	}
}

func TestSchemaAgent_WithLogo(t *testing.T) {
	tests := []struct {
		name     string
		logoURL  string
		wantLogo bool
		wantURL  string
	}{
		{
			name:     "valid logo URL",
			logoURL:  "https://example.com/logo.png",
			wantLogo: true,
			wantURL:  "https://example.com/logo.png",
		},
		{
			name:     "empty logo URL does not create ImageObject",
			logoURL:  "",
			wantLogo: false,
			wantURL:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewSchemaAgent("Organization", "Acme")
			result := agent.WithLogo(tt.logoURL)

			// Should return the same agent for chaining
			if result != agent {
				t.Error("WithLogo() should return the same agent")
			}

			if tt.wantLogo {
				if agent.Logo == nil {
					t.Fatal("Logo should not be nil")
				}
				if agent.Logo.Type != "ImageObject" {
					t.Errorf("Logo.Type: got %q, want %q", agent.Logo.Type, "ImageObject")
				}
				if agent.Logo.URL != tt.wantURL {
					t.Errorf("Logo.URL: got %q, want %q", agent.Logo.URL, tt.wantURL)
				}
			} else if agent.Logo != nil {
				t.Error("Logo should be nil for empty URL")
			}
		})
	}
}

func TestSchemaAgent_Chaining(t *testing.T) {
	agent := NewSchemaAgent("Organization", "Acme Corp").
		WithURL("https://acme.com").
		WithLogo("https://acme.com/logo.png")

	if agent.Type != "Organization" {
		t.Errorf("Type: got %q, want %q", agent.Type, "Organization")
	}
	if agent.Name != "Acme Corp" {
		t.Errorf("Name: got %q, want %q", agent.Name, "Acme Corp")
	}
	if agent.URL != "https://acme.com" {
		t.Errorf("URL: got %q, want %q", agent.URL, "https://acme.com")
	}
	if agent.Logo == nil {
		t.Fatal("Logo should not be nil")
	}
	if agent.Logo.URL != "https://acme.com/logo.png" {
		t.Errorf("Logo.URL: got %q, want %q", agent.Logo.URL, "https://acme.com/logo.png")
	}
}
