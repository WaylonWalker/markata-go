package blogroll

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestExtractHCardPhoto(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		baseURL  string
		expected string
	}{
		{
			name: "basic h-card with u-photo img",
			html: `<div class="h-card">
				<img class="u-photo" src="/avatar.jpg" alt="Avatar">
				<span class="p-name">John Doe</span>
			</div>`,
			baseURL:  "https://example.com",
			expected: "https://example.com/avatar.jpg",
		},
		{
			name: "h-card with u-photo in different class order",
			html: `<div class="h-card">
				<img src="/photo.png" class="avatar u-photo" alt="">
			</div>`,
			baseURL:  "https://example.com",
			expected: "https://example.com/photo.png",
		},
		{
			name: "h-card with absolute u-photo URL",
			html: `<div class="h-card">
				<img class="u-photo" src="https://cdn.example.com/avatar.jpg">
			</div>`,
			baseURL:  "https://example.com",
			expected: "https://cdn.example.com/avatar.jpg",
		},
		{
			name: "h-card with u-photo as link",
			html: `<div class="h-card">
				<a class="u-photo" href="/images/me.png">Photo</a>
			</div>`,
			baseURL:  "https://example.com",
			expected: "https://example.com/images/me.png",
		},
		{
			name: "u-photo outside h-card (fallback)",
			html: `<div>
				<img class="u-photo" src="/fallback.jpg">
			</div>`,
			baseURL:  "https://example.com",
			expected: "https://example.com/fallback.jpg",
		},
		{
			name:     "no u-photo found",
			html:     `<div class="h-card"><span class="p-name">Name</span></div>`,
			baseURL:  "https://example.com",
			expected: "",
		},
		{
			name: "multiple h-cards - first one wins",
			html: `<div class="h-card">
				<img class="u-photo" src="/first.jpg">
			</div>
			<div class="h-card">
				<img class="u-photo" src="/second.jpg">
			</div>`,
			baseURL:  "https://example.com",
			expected: "https://example.com/first.jpg",
		},
		{
			name: "h-card with data attribute",
			html: `<div class="h-card" data-u-photo="/data-photo.jpg">
				<span class="p-name">Name</span>
			</div>`,
			baseURL:  "https://example.com",
			expected: "https://example.com/data-photo.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHCardPhoto(tt.html, tt.baseURL)
			if result != tt.expected {
				t.Errorf("extractHCardPhoto() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractUPhoto(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		baseURL  string
		expected string
	}{
		{
			name:     "img with u-photo class",
			content:  `<img class="u-photo" src="/photo.jpg">`,
			baseURL:  "https://example.com",
			expected: "https://example.com/photo.jpg",
		},
		{
			name:     "img with src before class",
			content:  `<img src="/photo.jpg" class="u-photo">`,
			baseURL:  "https://example.com",
			expected: "https://example.com/photo.jpg",
		},
		{
			name:     "link with u-photo class",
			content:  `<a class="u-photo" href="/photo.jpg">Photo</a>`,
			baseURL:  "https://example.com",
			expected: "https://example.com/photo.jpg",
		},
		{
			name:     "link with href before class",
			content:  `<a href="/photo.jpg" class="profile u-photo">Photo</a>`,
			baseURL:  "https://example.com",
			expected: "https://example.com/photo.jpg",
		},
		{
			name:     "no u-photo",
			content:  `<img src="/photo.jpg" class="avatar">`,
			baseURL:  "https://example.com",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractUPhoto(tt.content, tt.baseURL)
			if result != tt.expected {
				t.Errorf("extractUPhoto() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestWebFingerAvatarDiscovery(t *testing.T) {
	// Create a test server that returns WebFinger responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/webfinger" {
			w.Header().Set("Content-Type", "application/jrd+json")
			_, _ = w.Write([]byte(`{
				"subject": "acct:user@example.com",
				"links": [
					{
						"rel": "http://webfinger.net/rel/profile-page",
						"type": "text/html",
						"href": "https://example.com/"
					},
					{
						"rel": "http://webfinger.net/rel/avatar",
						"href": "https://example.com/avatar.png"
					}
				]
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	updater := NewUpdater(5 * time.Second)
	ctx := context.Background()

	result, err := updater.discoverWebFingerAvatar(ctx, server.URL, "acct:user@example.com")
	if err != nil {
		t.Fatalf("discoverWebFingerAvatar() error = %v", err)
	}

	if result == nil {
		t.Fatal("discoverWebFingerAvatar() returned nil")
	}

	if result.URL != "https://example.com/avatar.png" {
		t.Errorf("discoverWebFingerAvatar() URL = %q, want %q", result.URL, "https://example.com/avatar.png")
	}

	if result.Source != AvatarSourceWebFinger {
		t.Errorf("discoverWebFingerAvatar() Source = %q, want %q", result.Source, AvatarSourceWebFinger)
	}
}

func TestWebFingerNoAvatarLink(t *testing.T) {
	// Create a test server that returns WebFinger without avatar
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/webfinger" {
			w.Header().Set("Content-Type", "application/jrd+json")
			_, _ = w.Write([]byte(`{
				"subject": "acct:user@example.com",
				"links": [
					{
						"rel": "http://webfinger.net/rel/profile-page",
						"type": "text/html",
						"href": "https://example.com/"
					}
				]
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	updater := NewUpdater(5 * time.Second)
	ctx := context.Background()

	result, err := updater.discoverWebFingerAvatar(ctx, server.URL, "acct:user@example.com")
	if err != nil {
		t.Fatalf("discoverWebFingerAvatar() error = %v", err)
	}

	if result != nil {
		t.Errorf("discoverWebFingerAvatar() = %v, want nil", result)
	}
}

func TestHCardAvatarDiscovery(t *testing.T) {
	// Create a test server that returns HTML with h-card
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<div class="h-card">
	<img class="u-photo" src="/avatar.jpg" alt="Avatar">
	<span class="p-name">Test User</span>
</div>
</body>
</html>`))
	}))
	defer server.Close()

	updater := NewUpdater(5 * time.Second)
	ctx := context.Background()

	result, err := updater.discoverHCardAvatar(ctx, server.URL)
	if err != nil {
		t.Fatalf("discoverHCardAvatar() error = %v", err)
	}

	if result == nil {
		t.Fatal("discoverHCardAvatar() returned nil")
	}

	expectedURL := server.URL + "/avatar.jpg"
	if result.URL != expectedURL {
		t.Errorf("discoverHCardAvatar() URL = %q, want %q", result.URL, expectedURL)
	}

	if result.Source != AvatarSourceHCard {
		t.Errorf("discoverHCardAvatar() Source = %q, want %q", result.Source, AvatarSourceHCard)
	}
}

func TestWellKnownAvatarDiscovery(t *testing.T) {
	// Create a test server that serves /.well-known/avatar
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/avatar" {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("fake image content"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	updater := NewUpdater(5 * time.Second)
	ctx := context.Background()

	result, err := updater.discoverWellKnownAvatar(ctx, server.URL)
	if err != nil {
		t.Fatalf("discoverWellKnownAvatar() error = %v", err)
	}

	if result == nil {
		t.Fatal("discoverWellKnownAvatar() returned nil")
	}

	expectedURL := server.URL + "/.well-known/avatar"
	if result.URL != expectedURL {
		t.Errorf("discoverWellKnownAvatar() URL = %q, want %q", result.URL, expectedURL)
	}

	if result.Source != AvatarSourceWellKnown {
		t.Errorf("discoverWellKnownAvatar() Source = %q, want %q", result.Source, AvatarSourceWellKnown)
	}
}

func TestDiscoverAvatar_PriorityOrder(t *testing.T) {
	// Create a test server with all avatar sources
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			// Homepage with h-card
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<div class="h-card"><img class="u-photo" src="/hcard-avatar.jpg"></div>`))
		case "/.well-known/webfinger":
			w.Header().Set("Content-Type", "application/jrd+json")
			_, _ = w.Write([]byte(`{
				"subject": "acct:user@example.com",
				"links": [{"rel": "http://webfinger.net/rel/avatar", "href": "/wf-avatar.jpg"}]
			}`))
		case "/.well-known/avatar":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("avatar"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	updater := NewUpdater(5 * time.Second)
	ctx := context.Background()

	// h-card should be preferred (first in priority)
	result, err := updater.DiscoverAvatar(ctx, server.URL, "")
	if err != nil {
		t.Fatalf("DiscoverAvatar() error = %v", err)
	}

	if result == nil {
		t.Fatal("DiscoverAvatar() returned nil")
	}

	// Should return h-card avatar (first priority)
	if result.Source != AvatarSourceHCard {
		t.Errorf("DiscoverAvatar() Source = %q, want %q (h-card is first priority)", result.Source, AvatarSourceHCard)
	}
}

func TestIsValidAvatarURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/avatar.jpg", true},
		{"https://example.com/avatar.png", true},
		{"https://example.com/avatar.gif", true},
		{"https://example.com/avatar.webp", true},
		{"https://example.com/avatar.svg", true},
		{"https://example.com/avatar.ico", true},
		{"https://example.com/avatar", true}, // Dynamic URLs allowed
		{"https://example.com/style.css", false},
		{"https://example.com/script.js", false},
		{"https://example.com/page.html", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := isValidAvatarURL(tt.url)
			if result != tt.expected {
				t.Errorf("isValidAvatarURL(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestMetadataWithAvatar(t *testing.T) {
	// Create a test server that returns both site metadata and avatar
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>Test Site</title>
	<meta property="og:title" content="Test Site">
	<meta property="og:description" content="A test site">
	<meta property="og:image" content="/og-image.jpg">
</head>
<body>
<div class="h-card">
	<img class="u-photo" src="/avatar.jpg">
	<span class="p-name">Test User</span>
</div>
</body>
</html>`))
		case "/feed.xml":
			w.Header().Set("Content-Type", "application/rss+xml")
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0">
<channel>
	<title>Test Feed</title>
	<link>` + serverURL + `</link>
	<description>A test feed</description>
</channel>
</rss>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	updater := NewUpdater(5 * time.Second)
	ctx := context.Background()

	metadata, err := updater.FetchMetadata(ctx, server.URL+"/feed.xml")
	if err != nil {
		t.Fatalf("FetchMetadata() error = %v", err)
	}

	// Check that avatar was discovered
	if metadata.AvatarURL == "" {
		t.Error("FetchMetadata() AvatarURL is empty, expected h-card avatar")
	}

	expectedAvatar := server.URL + "/avatar.jpg"
	if metadata.AvatarURL != expectedAvatar {
		t.Errorf("FetchMetadata() AvatarURL = %q, want %q", metadata.AvatarURL, expectedAvatar)
	}

	if metadata.AvatarSource != AvatarSourceHCard {
		t.Errorf("FetchMetadata() AvatarSource = %q, want %q", metadata.AvatarSource, AvatarSourceHCard)
	}

	// ImageURL should be the og:image (not the avatar)
	expectedImage := server.URL + "/og-image.jpg"
	if metadata.ImageURL != expectedImage {
		t.Errorf("FetchMetadata() ImageURL = %q, want %q", metadata.ImageURL, expectedImage)
	}
}
