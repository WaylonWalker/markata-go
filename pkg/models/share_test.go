package models

import (
	"strings"
	"testing"
)

func TestBuildShareButtonsDefaults(t *testing.T) {
	post := &Post{
		Slug:        "hello-world",
		Href:        "/hello-world/",
		Title:       stringPtr("Hello World"),
		Description: stringPtr("A friendly introduction"),
	}
	config := NewShareComponentConfig()
	buttons := BuildShareButtons(config, "https://example.com", "Site", post)
	if len(buttons) != len(DefaultSharePlatformOrder) {
		t.Fatalf("expected %d buttons, got %d", len(DefaultSharePlatformOrder), len(buttons))
	}
	if buttons[0].Key != "twitter" {
		t.Fatalf("expected first button to be twitter, got %s", buttons[0].Key)
	}
	if !strings.Contains(buttons[0].Link, "twitter.com/intent/tweet") {
		t.Fatalf("unexpected twitter link: %s", buttons[0].Link)
	}
	last := buttons[len(buttons)-1]
	if last.Action != "copy" || last.CopyText != "https://example.com/hello-world/" {
		t.Fatalf("copy button malformed: %+v", last)
	}
}

func TestBuildShareButtonsCustomPlatform(t *testing.T) {
	post := &Post{
		Slug:  "custom",
		Href:  "/custom/",
		Title: stringPtr("Custom Share"),
	}
	config := NewShareComponentConfig()
	config.Platforms = []string{"mastodon", "copy"}
	config.Custom = map[string]SharePlatformConfig{
		"mastodon": {
			Name: "Mastodon",
			Icon: "mastodon.svg",
			URL:  "https://mastodon.social/share?text={{title}}&url={{url}}",
		},
	}
	buttons := BuildShareButtons(config, "https://example.com", "Site", post)
	if len(buttons) != 2 {
		t.Fatalf("expected 2 buttons, got %d", len(buttons))
	}
	if buttons[0].Key != "mastodon" {
		t.Fatalf("expected mastodon platform first, got %s", buttons[0].Key)
	}
	if !strings.Contains(buttons[0].Link, "mastodon.social/share") {
		t.Fatalf("unexpected mastodon link: %s", buttons[0].Link)
	}
}

func TestBuildShareButtonsDisabled(t *testing.T) {
	post := &Post{Slug: "nothing", Href: "/nothing/"}
	config := NewShareComponentConfig()
	enabled := false
	config.Enabled = &enabled
	buttons := BuildShareButtons(config, "https://example.com", "Site", post)
	if len(buttons) != 0 {
		t.Fatalf("expected no buttons when disabled, got %d", len(buttons))
	}
}

func stringPtr(value string) *string {
	return &value
}
