// Package blogroll provides functionality for managing blogroll metadata.
package blogroll

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// AvatarSource indicates where an avatar URL was discovered.
type AvatarSource string

const (
	// AvatarSourceConfig indicates avatar was configured explicitly.
	AvatarSourceConfig AvatarSource = "config"
	// AvatarSourceHCard indicates avatar was discovered from h-card u-photo.
	AvatarSourceHCard AvatarSource = "h-card"
	// AvatarSourceWebFinger indicates avatar was discovered from WebFinger rel=avatar.
	AvatarSourceWebFinger AvatarSource = "webfinger"
	// AvatarSourceWellKnown indicates avatar was discovered from /.well-known/avatar.
	AvatarSourceWellKnown AvatarSource = "well-known"
	// AvatarSourceFeed indicates avatar was discovered from feed logo/icon.
	AvatarSourceFeed AvatarSource = "feed"
	// AvatarSourceOpenGraph indicates avatar was discovered from og:image.
	AvatarSourceOpenGraph AvatarSource = "opengraph"
	// AvatarSourceFavicon indicates avatar was discovered from favicon.
	AvatarSourceFavicon AvatarSource = "favicon"
)

// AvatarResult contains the discovered avatar URL and its source.
type AvatarResult struct {
	URL    string       `json:"url"`
	Source AvatarSource `json:"source"`
}

// WebFingerResponse represents a WebFinger JRD response.
type WebFingerResponse struct {
	Subject string          `json:"subject"`
	Aliases []string        `json:"aliases,omitempty"`
	Links   []WebFingerLink `json:"links,omitempty"`
}

// WebFingerLink represents a link in a WebFinger response.
type WebFingerLink struct {
	Rel  string `json:"rel"`
	Type string `json:"type,omitempty"`
	Href string `json:"href,omitempty"`
}

const (
	// webFingerAvatarRel is the standard rel value for avatar links.
	webFingerAvatarRel = "http://webfinger.net/rel/avatar"
)

// DiscoverAvatar attempts to discover an avatar URL for a site using multiple methods.
// It tries in this order:
// 1. h-card u-photo from the site's homepage
// 2. WebFinger rel=avatar (if resource handle is provided)
// 3. /.well-known/avatar endpoint
//
// Returns nil if no avatar is discovered.
func (u *Updater) DiscoverAvatar(ctx context.Context, siteURL, resource string) (*AvatarResult, error) {
	// 1. Try h-card u-photo from homepage
	if result, err := u.discoverHCardAvatar(ctx, siteURL); err == nil && result != nil {
		return result, nil
	}

	// 2. Try WebFinger rel=avatar if we have a resource
	if resource != "" {
		if result, err := u.discoverWebFingerAvatar(ctx, siteURL, resource); err == nil && result != nil {
			return result, nil
		}
	} else {
		// Try WebFinger with site URL as resource (low probability but worth trying)
		if result, err := u.discoverWebFingerAvatar(ctx, siteURL, siteURL); err == nil && result != nil {
			return result, nil
		}
	}

	// 3. Try /.well-known/avatar endpoint
	if result, err := u.discoverWellKnownAvatar(ctx, siteURL); err == nil && result != nil {
		return result, nil
	}

	return nil, nil
}

// discoverHCardAvatar fetches the site homepage and extracts u-photo from h-card.
func (u *Updater) discoverHCardAvatar(ctx context.Context, siteURL string) (*AvatarResult, error) {
	body, err := u.fetchURL(ctx, siteURL, "text/html,application/xhtml+xml")
	if err != nil {
		return nil, fmt.Errorf("fetch site: %w", err)
	}

	avatarURL := extractHCardPhoto(string(body), siteURL)
	if avatarURL != "" {
		return &AvatarResult{
			URL:    avatarURL,
			Source: AvatarSourceHCard,
		}, nil
	}

	return nil, nil
}

// discoverWebFingerAvatar queries WebFinger for the avatar rel link.
func (u *Updater) discoverWebFingerAvatar(ctx context.Context, siteURL, resource string) (*AvatarResult, error) {
	// Parse site URL to get host
	parsed, err := url.Parse(siteURL)
	if err != nil {
		return nil, fmt.Errorf("parse site URL: %w", err)
	}

	// Build WebFinger URL
	webfingerURL := fmt.Sprintf("%s://%s/.well-known/webfinger?resource=%s",
		parsed.Scheme, parsed.Host, url.QueryEscape(resource))

	body, err := u.fetchURL(ctx, webfingerURL, "application/jrd+json, application/json")
	if err != nil {
		return nil, fmt.Errorf("fetch webfinger: %w", err)
	}

	var jrd WebFingerResponse
	if err := json.Unmarshal(body, &jrd); err != nil {
		return nil, fmt.Errorf("parse webfinger: %w", err)
	}

	// Look for avatar rel link
	for _, link := range jrd.Links {
		if link.Rel == webFingerAvatarRel && link.Href != "" {
			return &AvatarResult{
				URL:    link.Href,
				Source: AvatarSourceWebFinger,
			}, nil
		}
	}

	return nil, nil
}

// discoverWellKnownAvatar checks for /.well-known/avatar endpoint.
func (u *Updater) discoverWellKnownAvatar(ctx context.Context, siteURL string) (*AvatarResult, error) {
	// Parse site URL to get host
	parsed, err := url.Parse(siteURL)
	if err != nil {
		return nil, fmt.Errorf("parse site URL: %w", err)
	}

	// Try /.well-known/avatar
	avatarURL := fmt.Sprintf("%s://%s/.well-known/avatar", parsed.Scheme, parsed.Host)

	// We just need to check if the URL exists and returns an image or redirect
	// The URL itself becomes the avatar URL if successful
	body, err := u.fetchURL(ctx, avatarURL, "image/*, text/html")
	if err != nil {
		return nil, fmt.Errorf("fetch well-known avatar: %w", err)
	}

	// If we got content, the URL is valid
	if len(body) > 0 {
		return &AvatarResult{
			URL:    avatarURL,
			Source: AvatarSourceWellKnown,
		}, nil
	}

	return nil, nil
}

// extractHCardPhoto extracts the u-photo from an h-card in HTML content.
// It follows these rules:
// 1. Prefer representative h-card (rel="me" or class contains "p-author")
// 2. Look for u-photo property (img.u-photo or background-image)
// 3. Resolve relative URLs against baseURL
func extractHCardPhoto(htmlContent, baseURL string) string {
	// Strategy:
	// 1. Find all h-card elements
	// 2. Within each h-card, look for u-photo
	// 3. Prefer h-cards that appear to be "representative" (first one, or one with rel="me")

	// Simple regex-based extraction
	// Look for patterns like:
	// <div class="h-card">...<img class="u-photo" src="...">...</div>
	// <div class="h-card">...<img class="... u-photo ..." src="...">...</div>

	// First, try to find h-card elements
	hcardPattern := regexp.MustCompile(`(?is)<(?:div|article|section|span)[^>]*class="[^"]*\bh-card\b[^"]*"[^>]*>(.*?)</(?:div|article|section|span)>`)
	hcards := hcardPattern.FindAllStringSubmatch(htmlContent, -1)

	for _, match := range hcards {
		if len(match) > 1 {
			hcardContent := match[1]

			// Look for u-photo img within the h-card
			photoURL := extractUPhoto(hcardContent, baseURL)
			if photoURL != "" {
				return photoURL
			}
		}
	}

	// Fallback: look for any u-photo on the page (not inside h-card)
	return extractUPhoto(htmlContent, baseURL)
}

// extractUPhoto extracts a u-photo URL from HTML content.
func extractUPhoto(content, baseURL string) string {
	// Pattern 1: <img class="u-photo" src="...">
	imgPattern := regexp.MustCompile(`(?i)<img[^>]*class="[^"]*\bu-photo\b[^"]*"[^>]*src="([^"]+)"`)
	if matches := imgPattern.FindStringSubmatch(content); len(matches) > 1 {
		return resolveURL(matches[1], baseURL)
	}

	// Pattern 1b: <img src="..." class="u-photo">
	imgPattern2 := regexp.MustCompile(`(?i)<img[^>]*src="([^"]+)"[^>]*class="[^"]*\bu-photo\b[^"]*"`)
	if matches := imgPattern2.FindStringSubmatch(content); len(matches) > 1 {
		return resolveURL(matches[1], baseURL)
	}

	// Pattern 2: <a class="u-photo" href="...">
	linkPattern := regexp.MustCompile(`(?i)<a[^>]*class="[^"]*\bu-photo\b[^"]*"[^>]*href="([^"]+)"`)
	if matches := linkPattern.FindStringSubmatch(content); len(matches) > 1 {
		return resolveURL(matches[1], baseURL)
	}

	// Pattern 2b: <a href="..." class="u-photo">
	linkPattern2 := regexp.MustCompile(`(?i)<a[^>]*href="([^"]+)"[^>]*class="[^"]*\bu-photo\b[^"]*"`)
	if matches := linkPattern2.FindStringSubmatch(content); len(matches) > 1 {
		return resolveURL(matches[1], baseURL)
	}

	// Pattern 3: data-u-photo attribute
	dataPattern := regexp.MustCompile(`(?i)data-u-photo="([^"]+)"`)
	if matches := dataPattern.FindStringSubmatch(content); len(matches) > 1 {
		return resolveURL(matches[1], baseURL)
	}

	return ""
}

// isValidAvatarURL checks if a URL looks like a valid avatar image.
func isValidAvatarURL(avatarURL string) bool {
	if avatarURL == "" {
		return false
	}

	lower := strings.ToLower(avatarURL)

	// Check for common image extensions
	imageExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".ico"}
	for _, ext := range imageExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}

	// Also accept URLs without extensions (might be served dynamically)
	// But filter out obviously non-image URLs
	nonImagePatterns := []string{".css", ".js", ".html", ".xml", ".json"}
	for _, pattern := range nonImagePatterns {
		if strings.HasSuffix(lower, pattern) {
			return false
		}
	}

	return true
}
