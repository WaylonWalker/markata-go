package models

import "testing"

func TestNewLink(t *testing.T) {
	link := NewLink(
		"https://example.com/post/",
		"/other/",
		"https://example.com/other/",
		"example.com",
		true,
	)

	if link.SourceURL != "https://example.com/post/" {
		t.Errorf("expected SourceURL 'https://example.com/post/', got %q", link.SourceURL)
	}
	if link.RawTarget != "/other/" {
		t.Errorf("expected RawTarget '/other/', got %q", link.RawTarget)
	}
	if link.TargetURL != "https://example.com/other/" {
		t.Errorf("expected TargetURL 'https://example.com/other/', got %q", link.TargetURL)
	}
	if link.TargetDomain != "example.com" {
		t.Errorf("expected TargetDomain 'example.com', got %q", link.TargetDomain)
	}
	if !link.IsInternal {
		t.Error("expected IsInternal to be true")
	}
	if link.IsSelf {
		t.Error("expected IsSelf to be false by default")
	}
}

func TestLink_SourceSlug(t *testing.T) {
	// Test with nil SourcePost
	link := &Link{}
	if slug := link.SourceSlug(); slug != "" {
		t.Errorf("expected empty string for nil SourcePost, got %q", slug)
	}

	// Test with SourcePost
	link.SourcePost = &Post{Slug: "my-post"}
	if slug := link.SourceSlug(); slug != "my-post" {
		t.Errorf("expected 'my-post', got %q", slug)
	}
}

func TestLink_TargetSlug(t *testing.T) {
	// Test with nil TargetPost
	link := &Link{}
	if slug := link.TargetSlug(); slug != "" {
		t.Errorf("expected empty string for nil TargetPost, got %q", slug)
	}

	// Test with TargetPost
	link.TargetPost = &Post{Slug: "target-post"}
	if slug := link.TargetSlug(); slug != "target-post" {
		t.Errorf("expected 'target-post', got %q", slug)
	}
}

func TestLink_SourceTitle(t *testing.T) {
	// Test with nil SourcePost
	link := &Link{}
	if title := link.SourceTitle(); title != "" {
		t.Errorf("expected empty string for nil SourcePost, got %q", title)
	}

	// Test with SourcePost but nil Title
	link.SourcePost = &Post{}
	if title := link.SourceTitle(); title != "" {
		t.Errorf("expected empty string for nil Title, got %q", title)
	}

	// Test with SourcePost and Title
	postTitle := "My Post Title"
	link.SourcePost = &Post{Title: &postTitle}
	if title := link.SourceTitle(); title != "My Post Title" {
		t.Errorf("expected 'My Post Title', got %q", title)
	}
}

func TestLink_TargetTitle(t *testing.T) {
	// Test with nil TargetPost
	link := &Link{}
	if title := link.TargetTitle(); title != "" {
		t.Errorf("expected empty string for nil TargetPost, got %q", title)
	}

	// Test with TargetPost but nil Title
	link.TargetPost = &Post{}
	if title := link.TargetTitle(); title != "" {
		t.Errorf("expected empty string for nil Title, got %q", title)
	}

	// Test with TargetPost and Title
	postTitle := "Target Post Title"
	link.TargetPost = &Post{Title: &postTitle}
	if title := link.TargetTitle(); title != "Target Post Title" {
		t.Errorf("expected 'Target Post Title', got %q", title)
	}
}

func TestLink_ExternalLink(t *testing.T) {
	link := NewLink(
		"https://mysite.com/post/",
		"https://external.com/page",
		"https://external.com/page",
		"external.com",
		false,
	)

	if link.IsInternal {
		t.Error("expected IsInternal to be false for external link")
	}
	if link.TargetDomain != "external.com" {
		t.Errorf("expected TargetDomain 'external.com', got %q", link.TargetDomain)
	}
}

func TestLink_SelfLink(t *testing.T) {
	post := &Post{Slug: "my-post"}

	link := &Link{
		SourcePost: post,
		TargetPost: post,
		IsSelf:     true,
	}

	if !link.IsSelf {
		t.Error("expected IsSelf to be true")
	}
	if link.SourceSlug() != link.TargetSlug() {
		t.Error("expected source and target slugs to be equal for self-link")
	}
}
