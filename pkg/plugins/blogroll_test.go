package plugins

import (
	"testing"
)

// =============================================================================
// extractFirstImageFromHTML Tests
// =============================================================================

func TestExtractFirstImageFromHTML_BasicImage(t *testing.T) {
	html := `<p>Some text</p><img src="https://example.com/image.jpg" alt="test">`
	result := extractFirstImageFromHTML(html)
	if result != "https://example.com/image.jpg" {
		t.Errorf("extractFirstImageFromHTML() = %q, want %q", result, "https://example.com/image.jpg")
	}
}

func TestExtractFirstImageFromHTML_SingleQuotes(t *testing.T) {
	html := `<img src='https://example.com/image.png'>`
	result := extractFirstImageFromHTML(html)
	if result != "https://example.com/image.png" {
		t.Errorf("extractFirstImageFromHTML() = %q, want %q", result, "https://example.com/image.png")
	}
}

func TestExtractFirstImageFromHTML_NoImage(t *testing.T) {
	html := `<p>No images here</p>`
	result := extractFirstImageFromHTML(html)
	if result != "" {
		t.Errorf("extractFirstImageFromHTML() = %q, want empty string", result)
	}
}

func TestExtractFirstImageFromHTML_HTMLEntities(t *testing.T) {
	// This tests the bug fix for Atom feeds that encode content as HTML entities
	html := `&lt;p&gt;Some text&lt;/p&gt;&lt;img src="https://example.com/atom-image.jpg" alt="test"&gt;`
	result := extractFirstImageFromHTML(html)
	if result != "https://example.com/atom-image.jpg" {
		t.Errorf("extractFirstImageFromHTML() with HTML entities = %q, want %q", result, "https://example.com/atom-image.jpg")
	}
}

func TestExtractFirstImageFromHTML_NestedHTMLEntities(t *testing.T) {
	// Test deeply encoded content (edge case)
	html := `&lt;img src=&quot;https://example.com/deeply-encoded.jpg&quot;&gt;`
	result := extractFirstImageFromHTML(html)
	if result != "https://example.com/deeply-encoded.jpg" {
		t.Errorf("extractFirstImageFromHTML() with nested entities = %q, want %q", result, "https://example.com/deeply-encoded.jpg")
	}
}

func TestExtractFirstImageFromHTML_MultipleImages(t *testing.T) {
	// Should return the first image only
	html := `<img src="first.jpg"><img src="second.jpg">`
	result := extractFirstImageFromHTML(html)
	if result != "first.jpg" {
		t.Errorf("extractFirstImageFromHTML() = %q, want %q", result, "first.jpg")
	}
}

// =============================================================================
// generateFallbackImageURL Tests
// =============================================================================

func TestGenerateFallbackImageURL_Basic(t *testing.T) {
	template := "https://shots.example.com/shot/?url={url}&width=1200"
	entryURL := "https://blog.example.com/my-post"
	result := generateFallbackImageURL(template, entryURL)
	expected := "https://shots.example.com/shot/?url=https%3A%2F%2Fblog.example.com%2Fmy-post&width=1200"
	if result != expected {
		t.Errorf("generateFallbackImageURL() = %q, want %q", result, expected)
	}
}

func TestGenerateFallbackImageURL_WithSpecialChars(t *testing.T) {
	template := "https://screenshot.service/{url}"
	entryURL := "https://example.com/post?foo=bar&baz=qux"
	result := generateFallbackImageURL(template, entryURL)
	expected := "https://screenshot.service/https%3A%2F%2Fexample.com%2Fpost%3Ffoo%3Dbar%26baz%3Dqux"
	if result != expected {
		t.Errorf("generateFallbackImageURL() = %q, want %q", result, expected)
	}
}

func TestGenerateFallbackImageURL_NoPlaceholder(t *testing.T) {
	// If template has no {url} placeholder, return as-is
	template := "https://default-image.com/fallback.png"
	entryURL := "https://blog.example.com/post"
	result := generateFallbackImageURL(template, entryURL)
	if result != template {
		t.Errorf("generateFallbackImageURL() = %q, want %q", result, template)
	}
}

func TestGenerateFallbackImageURL_EmptyEntryURL(t *testing.T) {
	template := "https://shots.example.com/?url={url}"
	entryURL := ""
	result := generateFallbackImageURL(template, entryURL)
	expected := "https://shots.example.com/?url="
	if result != expected {
		t.Errorf("generateFallbackImageURL() = %q, want %q", result, expected)
	}
}

func TestGenerateFallbackImageURL_URLWithUnicode(t *testing.T) {
	template := "https://shots.example.com/?url={url}"
	entryURL := "https://example.com/post/日本語"
	result := generateFallbackImageURL(template, entryURL)
	// Unicode characters should be percent-encoded
	expected := "https://shots.example.com/?url=https%3A%2F%2Fexample.com%2Fpost%2F%E6%97%A5%E6%9C%AC%E8%AA%9E"
	if result != expected {
		t.Errorf("generateFallbackImageURL() = %q, want %q", result, expected)
	}
}
