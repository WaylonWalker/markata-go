// Package tests provides integration tests for markata-go.
package tests

import (
	"errors"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/encryption"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/plugins"
)

// =============================================================================
// Encryption Integration Tests
// =============================================================================

// privateMarker is a unique string embedded in private post content.
// Tests verify this string NEVER appears in plaintext in the build output.
const privateMarker = "SUPER_SECRET_MARKER_TEXT_12345"

// buildWithEncryption creates a manager configured for encryption testing.
// It registers glob, load, render markdown, encryption, and publish plugins.
func buildWithEncryption(t *testing.T, site *testSite, modelsConfig *models.Config) (*lifecycle.Manager, error) {
	t.Helper()

	m := lifecycle.NewManager()

	cfg := &lifecycle.Config{
		ContentDir:   site.contentDir,
		OutputDir:    site.outputDir,
		GlobPatterns: []string{"**/*.md"},
		Extra:        make(map[string]interface{}),
	}
	cfg.Extra["url"] = "https://example.com"
	cfg.Extra["title"] = "Test Site"

	if modelsConfig != nil {
		cfg.Extra["models_config"] = modelsConfig
	}

	m.SetConfig(cfg)

	m.RegisterPlugin(plugins.NewGlobPlugin())
	m.RegisterPlugin(plugins.NewLoadPlugin())
	m.RegisterPlugin(plugins.NewRenderMarkdownPlugin())
	m.RegisterPlugin(plugins.NewEncryptionPlugin())
	m.RegisterPlugin(plugins.NewPublishHTMLPlugin())

	err := m.Run()
	return m, err
}

// TestIntegration_Encryption_PrivateContentNeverPlaintext verifies that private
// content is encrypted and the marker text never appears in plaintext output.
func TestIntegration_Encryption_PrivateContentNeverPlaintext(t *testing.T) {
	// Set encryption key
	t.Setenv("MARKATA_GO_ENCRYPTION_KEY_DEFAULT", "test-password-xyz") // pragma: allowlist secret

	site := newTestSite(t)

	// Private post with marker text
	site.addPost("secret-post.md", `---
title: Secret Post
slug: secret-post
published: true
private: true
---
This post contains `+privateMarker+` that should be encrypted.`)

	// Public post (no marker text)
	site.addPost("public-post.md", `---
title: Public Post
slug: public-post
published: true
---
This is a public post with nothing secret.`)

	// Another private post with explicit secret_key
	site.addPost("private-with-key.md", `---
title: Private With Key
slug: private-with-key
published: true
private: true
secret_key: default
---
Another private post with `+privateMarker+` inside it.`)

	modelsConfig := models.NewConfig()
	// Encryption is enabled by default with DefaultKey="default"

	m, err := buildWithEncryption(t, site, modelsConfig)
	if err != nil {
		t.Fatalf("build should succeed with encryption key set: %v", err)
	}

	// Verify private posts were encrypted
	for _, post := range m.Posts() {
		if !post.Private {
			continue
		}

		// The marker text must NOT appear in plaintext in ArticleHTML
		if strings.Contains(post.ArticleHTML, privateMarker) {
			t.Errorf("SECURITY: private post %q contains marker text in plaintext ArticleHTML", post.Path)
		}

		// Must have encryption wrapper
		if !strings.Contains(post.ArticleHTML, `class="encrypted-content"`) {
			t.Errorf("private post %q missing encrypted-content wrapper", post.Path)
		}
		if !strings.Contains(post.ArticleHTML, `data-encrypted="`) {
			t.Errorf("private post %q missing data-encrypted attribute", post.Path)
		}

		// Must be marked for script inclusion
		if hasEncrypted, ok := post.Extra["has_encrypted_content"].(bool); !ok || !hasEncrypted {
			t.Errorf("private post %q should have has_encrypted_content=true", post.Path)
		}
	}

	// Verify public post is NOT encrypted
	for _, post := range m.Posts() {
		if post.Private {
			continue
		}
		if strings.Contains(post.ArticleHTML, `class="encrypted-content"`) {
			t.Errorf("public post %q should NOT have encrypted-content wrapper", post.Path)
		}
		if strings.Contains(post.ArticleHTML, "data-encrypted=") {
			t.Errorf("public post %q should NOT have data-encrypted attribute", post.Path)
		}
	}

	// Verify the encrypted content can actually be decrypted
	for _, post := range m.Posts() {
		if !post.Private {
			continue
		}
		encryptedData := extractEncryptedData(t, post.ArticleHTML)
		if encryptedData == "" {
			t.Errorf("private post %q has no encrypted data", post.Path)
			continue
		}

		decrypted, decErr := encryption.Decrypt(encryptedData, "test-password-xyz") // pragma: allowlist secret
		if decErr != nil {
			t.Errorf("failed to decrypt private post %q: %v", post.Path, decErr)
			continue
		}

		// The decrypted content SHOULD contain the marker
		if !strings.Contains(string(decrypted), privateMarker) {
			t.Errorf("decrypted content of %q should contain marker text", post.Path)
		}
	}

	// Check output files if they were written
	if site.fileExists("secret-post/index.html") {
		content := site.readFile("secret-post/index.html")
		if strings.Contains(content, privateMarker) {
			t.Error("SECURITY: output file secret-post/index.html contains marker in plaintext")
		}
	}
	if site.fileExists("private-with-key/index.html") {
		content := site.readFile("private-with-key/index.html")
		if strings.Contains(content, privateMarker) {
			t.Error("SECURITY: output file private-with-key/index.html contains marker in plaintext")
		}
	}
}

// TestIntegration_Encryption_BuildFailsWithoutKey verifies the build fails with
// a CriticalError when private posts exist but no encryption key is available.
func TestIntegration_Encryption_BuildFailsWithoutKey(t *testing.T) {
	// Ensure NO encryption keys are set
	t.Setenv("MARKATA_GO_ENCRYPTION_KEY_DEFAULT", "") // pragma: allowlist secret

	site := newTestSite(t)

	site.addPost("secret.md", `---
title: Secret
slug: secret
published: true
private: true
---
This must not be published without encryption.`)

	modelsConfig := models.NewConfig()

	_, err := buildWithEncryption(t, site, modelsConfig)
	if err == nil {
		t.Fatal("build should FAIL when private posts exist but no encryption key is set")
	}

	// Verify it's a CriticalError via HookErrors.HasCritical()
	var hookErrors *lifecycle.HookErrors
	if !errors.As(err, &hookErrors) {
		t.Fatalf("error should be *lifecycle.HookErrors, got: %T", err)
	}
	if !hookErrors.HasCritical() {
		t.Error("HookErrors should contain at least one critical error")
	}

	// Error message should mention the missing key
	if !strings.Contains(err.Error(), "secret.md") {
		t.Errorf("error should reference the affected post, got: %v", err)
	}
}

// TestIntegration_Encryption_PrivateTags verifies that posts are automatically
// marked private based on configured private_tags.
func TestIntegration_Encryption_PrivateTags(t *testing.T) {
	t.Setenv("MARKATA_GO_ENCRYPTION_KEY_DEFAULT", "test-password-xyz") // pragma: allowlist secret
	t.Setenv("MARKATA_GO_ENCRYPTION_KEY_PERSONAL", "personal-pass")    // pragma: allowlist secret

	site := newTestSite(t)

	// Post tagged "diary" -- should be auto-private via private_tags config
	site.addPost("diary-entry.md", `---
title: Diary Entry
slug: diary-entry
published: true
tags:
  - diary
  - reflection
---
My diary has `+privateMarker+` in it.`)

	// Public post with no matching tags
	site.addPost("blog-post.md", `---
title: Blog Post
slug: blog-post
published: true
tags:
  - golang
  - tutorial
---
This is a public blog post.`)

	modelsConfig := models.NewConfig()
	modelsConfig.Encryption.PrivateTags = map[string]string{
		"diary": "personal",
	}

	m, err := buildWithEncryption(t, site, modelsConfig)
	if err != nil {
		t.Fatalf("build should succeed: %v", err)
	}

	// Find the diary post and verify it was encrypted
	var diaryPost *models.Post
	var blogPost *models.Post
	for _, post := range m.Posts() {
		if strings.Contains(post.Path, "diary-entry") {
			diaryPost = post
		}
		if strings.Contains(post.Path, "blog-post") {
			blogPost = post
		}
	}

	if diaryPost == nil {
		t.Fatal("diary post not found in build output")
	}
	if !diaryPost.Private {
		t.Error("diary post should be marked private via private_tags")
	}
	if strings.Contains(diaryPost.ArticleHTML, privateMarker) {
		t.Error("SECURITY: diary post contains marker text in plaintext")
	}
	if !strings.Contains(diaryPost.ArticleHTML, `class="encrypted-content"`) {
		t.Error("diary post should be encrypted")
	}

	if blogPost == nil {
		t.Fatal("blog post not found in build output")
	}
	if blogPost.Private {
		t.Error("blog post should NOT be marked private")
	}
}

// TestIntegration_Encryption_FrontmatterAliases verifies that private_key and
// encryption_key frontmatter fields work as aliases for secret_key.
func TestIntegration_Encryption_FrontmatterAliases(t *testing.T) {
	t.Setenv("MARKATA_GO_ENCRYPTION_KEY_DEFAULT", "test-password-xyz") // pragma: allowlist secret

	site := newTestSite(t)

	// Post using secret_key
	site.addPost("with-secret-key.md", `---
title: With SecretKey
slug: with-secret-key
published: true
private: true
secret_key: default
---
Secret content A `+privateMarker)

	// Post using private_key alias
	site.addPost("with-private-key.md", `---
title: With PrivateKey
slug: with-private-key
published: true
private: true
private_key: default
---
Secret content B `+privateMarker)

	// Post using encryption_key alias
	site.addPost("with-encryption-key.md", `---
title: With EncryptionKey
slug: with-encryption-key
published: true
private: true
encryption_key: default
---
Secret content C `+privateMarker)

	modelsConfig := models.NewConfig()

	m, err := buildWithEncryption(t, site, modelsConfig)
	if err != nil {
		t.Fatalf("build should succeed: %v", err)
	}

	// All three posts should be encrypted
	for _, post := range m.Posts() {
		if !post.Private {
			continue
		}
		if strings.Contains(post.ArticleHTML, privateMarker) {
			t.Errorf("SECURITY: post %q contains marker in plaintext (alias may not work)", post.Path)
		}
		if !strings.Contains(post.ArticleHTML, `class="encrypted-content"`) {
			t.Errorf("post %q should be encrypted", post.Path)
		}
	}
}

// TestIntegration_Encryption_FrontmatterKeyOverridesTagKey verifies that
// a frontmatter secret_key takes precedence over a tag-level key.
func TestIntegration_Encryption_FrontmatterKeyOverridesTagKey(t *testing.T) {
	t.Setenv("MARKATA_GO_ENCRYPTION_KEY_DEFAULT", "default-pass")   // pragma: allowlist secret
	t.Setenv("MARKATA_GO_ENCRYPTION_KEY_PERSONAL", "personal-pass") // pragma: allowlist secret
	t.Setenv("MARKATA_GO_ENCRYPTION_KEY_CUSTOM", "custom-pass")     // pragma: allowlist secret

	site := newTestSite(t)

	// Post with diary tag AND explicit secret_key -- frontmatter should win
	site.addPost("override.md", `---
title: Override Test
slug: override
published: true
private: true
secret_key: custom
tags:
  - diary
---
Override content `+privateMarker)

	modelsConfig := models.NewConfig()
	modelsConfig.Encryption.PrivateTags = map[string]string{
		"diary": "personal",
	}

	m, err := buildWithEncryption(t, site, modelsConfig)
	if err != nil {
		t.Fatalf("build should succeed: %v", err)
	}

	var overridePost *models.Post
	for _, post := range m.Posts() {
		if strings.Contains(post.Path, "override") {
			overridePost = post
		}
	}

	if overridePost == nil {
		t.Fatal("override post not found")
	}

	// The key should be "custom" (frontmatter), not "personal" (tag)
	keyName, ok := overridePost.Extra["encryption_key_name"].(string)
	if !ok {
		t.Fatal("override post should have encryption_key_name in Extra")
	}
	if keyName != "custom" {
		t.Errorf("encryption_key_name = %q, want %q (frontmatter should override tag)", keyName, "custom")
	}

	// Verify encryption happened with the custom key
	encryptedData := extractEncryptedData(t, overridePost.ArticleHTML)
	if encryptedData == "" {
		t.Fatal("override post should have encrypted data")
	}

	// Should decrypt with custom-pass, not personal-pass
	decrypted, decErr := encryption.Decrypt(encryptedData, "custom-pass") // pragma: allowlist secret
	if decErr != nil {
		t.Fatalf("should decrypt with custom key password: %v", decErr)
	}
	if !strings.Contains(string(decrypted), privateMarker) {
		t.Error("decrypted content should contain marker")
	}
}

// TestIntegration_Encryption_DisabledSkipsEncryption verifies that when
// encryption is disabled, private posts pass through unmodified.
func TestIntegration_Encryption_DisabledSkipsEncryption(t *testing.T) {
	site := newTestSite(t)

	site.addPost("secret.md", `---
title: Secret
slug: secret
published: true
private: true
---
Content here.`)

	modelsConfig := models.NewConfig()
	modelsConfig.Encryption.Enabled = false

	m, err := buildWithEncryption(t, site, modelsConfig)
	if err != nil {
		t.Fatalf("build should succeed with encryption disabled: %v", err)
	}

	for _, post := range m.Posts() {
		if strings.Contains(post.ArticleHTML, `class="encrypted-content"`) {
			t.Error("encryption should not run when disabled")
		}
	}
}

// TestIntegration_Encryption_DraftAndSkippedPostsIgnored verifies that
// draft and skipped posts are not subject to encryption requirements.
func TestIntegration_Encryption_DraftAndSkippedPostsIgnored(t *testing.T) {
	// No encryption key -- normally this would fail if private posts exist
	t.Setenv("MARKATA_GO_ENCRYPTION_KEY_DEFAULT", "") // pragma: allowlist secret

	site := newTestSite(t)

	// Draft private post -- should be ignored
	site.addPost("draft.md", `---
title: Draft
slug: draft
published: true
private: true
draft: true
---
Draft content.`)

	// Non-private published post
	site.addPost("public.md", `---
title: Public
slug: public
published: true
---
Public content.`)

	modelsConfig := models.NewConfig()

	_, err := buildWithEncryption(t, site, modelsConfig)
	if err != nil {
		t.Fatalf("build should succeed when only draft/skipped posts are private: %v", err)
	}
}

// =============================================================================
// Helpers
// =============================================================================

// extractEncryptedData extracts the base64-encoded encrypted data from the
// encrypted-content div's data-encrypted attribute.
func extractEncryptedData(t *testing.T, html string) string {
	t.Helper()
	marker := `data-encrypted="`
	start := strings.Index(html, marker)
	if start == -1 {
		return ""
	}
	start += len(marker)
	end := strings.Index(html[start:], `"`)
	if end == -1 {
		return ""
	}
	return html[start : start+end]
}
