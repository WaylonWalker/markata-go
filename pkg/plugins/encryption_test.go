package plugins

import (
	"os"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/encryption"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestEncryptionPlugin_Name(t *testing.T) {
	plugin := NewEncryptionPlugin()
	if plugin.Name() != "encryption" {
		t.Errorf("Name() = %q, want %q", plugin.Name(), "encryption")
	}
}

func TestEncryptionPlugin_ShouldEncrypt(t *testing.T) {
	plugin := NewEncryptionPlugin()
	plugin.enabled = true
	plugin.defaultKey = "default"
	plugin.keys = map[string]string{"default": "password"}

	tests := []struct {
		name       string
		post       *models.Post
		wantResult bool
	}{
		{
			name: "private post with secret key",
			post: &models.Post{
				Private:     true,
				SecretKey:   "blog",
				ArticleHTML: "<p>Content</p>",
			},
			wantResult: true,
		},
		{
			name: "private post without key uses default",
			post: &models.Post{
				Private:     true,
				ArticleHTML: "<p>Content</p>",
			},
			wantResult: true,
		},
		{
			name: "non-private post",
			post: &models.Post{
				Private:     false,
				SecretKey:   "blog",
				ArticleHTML: "<p>Content</p>",
			},
			wantResult: false,
		},
		{
			name: "draft post",
			post: &models.Post{
				Private:     true,
				Draft:       true,
				SecretKey:   "blog",
				ArticleHTML: "<p>Content</p>",
			},
			wantResult: false,
		},
		{
			name: "skipped post",
			post: &models.Post{
				Private:     true,
				Skip:        true,
				SecretKey:   "blog",
				ArticleHTML: "<p>Content</p>",
			},
			wantResult: false,
		},
		{
			name: "private post with no content",
			post: &models.Post{
				Private:     true,
				SecretKey:   "blog",
				ArticleHTML: "",
			},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := plugin.shouldEncrypt(tt.post)
			if got != tt.wantResult {
				t.Errorf("shouldEncrypt() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

func TestEncryptionPlugin_ShouldEncrypt_NoDefaultKey(t *testing.T) {
	plugin := NewEncryptionPlugin()
	plugin.enabled = true
	plugin.defaultKey = "" // No default key

	// Post without secret_key should not be encrypted
	post := &models.Post{
		Private:     true,
		ArticleHTML: "<p>Content</p>",
	}

	if plugin.shouldEncrypt(post) {
		t.Error("shouldEncrypt() should return false when no default key and no secret_key")
	}
}

func TestEncryptionPlugin_LoadKeysFromEnvironment(t *testing.T) {
	// Set test environment variables
	os.Setenv("MARKATA_GO_ENCRYPTION_KEY_BLOG", "blog-password")
	os.Setenv("MARKATA_GO_ENCRYPTION_KEY_PREMIUM", "premium-password")
	defer func() {
		os.Unsetenv("MARKATA_GO_ENCRYPTION_KEY_BLOG")
		os.Unsetenv("MARKATA_GO_ENCRYPTION_KEY_PREMIUM")
	}()

	plugin := NewEncryptionPlugin()
	plugin.loadKeysFromEnvironment()

	if len(plugin.keys) != 2 {
		t.Errorf("Expected 2 keys loaded, got %d", len(plugin.keys))
	}

	if plugin.keys["blog"] != "blog-password" {
		t.Errorf("blog key = %q, want %q", plugin.keys["blog"], "blog-password")
	}

	if plugin.keys["premium"] != "premium-password" {
		t.Errorf("premium key = %q, want %q", plugin.keys["premium"], "premium-password")
	}
}

func TestEncryptionPlugin_GetKeyPassword(t *testing.T) {
	plugin := NewEncryptionPlugin()
	plugin.keys = map[string]string{
		"blog":    "blog-password",
		"premium": "premium-password",
	}
	plugin.defaultKey = "blog"

	tests := []struct {
		name     string
		keyName  string
		wantPass string
		wantErr  bool
	}{
		{
			name:     "existing key",
			keyName:  "blog",
			wantPass: "blog-password",
			wantErr:  false,
		},
		{
			name:     "case insensitive",
			keyName:  "BLOG",
			wantPass: "blog-password",
			wantErr:  false,
		},
		{
			name:     "fallback to default",
			keyName:  "nonexistent",
			wantPass: "blog-password",
			wantErr:  false,
		},
		{
			name:     "another key",
			keyName:  "premium",
			wantPass: "premium-password",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPass, err := plugin.getKeyPassword(tt.keyName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getKeyPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotPass != tt.wantPass {
				t.Errorf("getKeyPassword() = %q, want %q", gotPass, tt.wantPass)
			}
		})
	}
}

func TestEncryptionPlugin_GetKeyPassword_NoDefaultKey(t *testing.T) {
	plugin := NewEncryptionPlugin()
	plugin.keys = map[string]string{
		"blog": "blog-password",
	}
	plugin.defaultKey = "" // No default

	// Should fail for nonexistent key with no default
	_, err := plugin.getKeyPassword("nonexistent")
	if err == nil {
		t.Error("getKeyPassword() should return error for nonexistent key with no default")
	}
}

func TestEncryptionPlugin_EncryptPost(t *testing.T) {
	plugin := NewEncryptionPlugin()
	plugin.enabled = true
	plugin.keys = map[string]string{
		"blog": "test-password-123",
	}
	plugin.decryptionHint = "Contact me for access"

	post := &models.Post{
		Path:        "test.md",
		Private:     true,
		SecretKey:   "blog",
		ArticleHTML: "<p>This is secret content</p>",
		Extra:       make(map[string]interface{}),
	}

	err := plugin.encryptPost(post)
	if err != nil {
		t.Fatalf("encryptPost() error: %v", err)
	}

	// Check that content was encrypted
	if !strings.Contains(post.ArticleHTML, `class="encrypted-content"`) {
		t.Error("ArticleHTML should contain encrypted-content class")
	}

	if !strings.Contains(post.ArticleHTML, "data-encrypted=") {
		t.Error("ArticleHTML should contain data-encrypted attribute")
	}

	if !strings.Contains(post.ArticleHTML, "Encrypted Content") {
		t.Error("ArticleHTML should contain 'Encrypted Content' title")
	}

	if !strings.Contains(post.ArticleHTML, plugin.decryptionHint) {
		t.Error("ArticleHTML should contain decryption hint")
	}

	// Check post was marked for script inclusion
	if hasEncrypted, ok := post.Extra["has_encrypted_content"].(bool); !ok || !hasEncrypted {
		t.Error("Post should have has_encrypted_content = true in Extra")
	}

	// Extract and verify the encrypted content can be decrypted
	start := strings.Index(post.ArticleHTML, `data-encrypted="`) + len(`data-encrypted="`)
	end := strings.Index(post.ArticleHTML[start:], `"`)
	encryptedData := post.ArticleHTML[start : start+end]

	decrypted, err := encryption.Decrypt(encryptedData, "test-password-123")
	if err != nil {
		t.Fatalf("Failed to decrypt content: %v", err)
	}

	if string(decrypted) != "<p>This is secret content</p>" {
		t.Errorf("Decrypted content = %q, want %q", decrypted, "<p>This is secret content</p>")
	}
}

func TestEncryptionPlugin_EncryptPost_MissingKey(t *testing.T) {
	plugin := NewEncryptionPlugin()
	plugin.enabled = true
	plugin.keys = map[string]string{} // No keys loaded

	post := &models.Post{
		Path:        "test.md",
		Private:     true,
		SecretKey:   "nonexistent",
		ArticleHTML: "<p>Content</p>",
		Extra:       make(map[string]interface{}),
	}

	err := plugin.encryptPost(post)
	if err != nil {
		t.Fatalf("encryptPost() should not return error, got: %v", err)
	}

	// Check that an error was stored in Extra
	if _, ok := post.Extra["encryption_error"]; !ok {
		t.Error("Post should have encryption_error in Extra when key is missing")
	}

	// Content should remain unchanged
	if post.ArticleHTML != "<p>Content</p>" {
		t.Error("Content should remain unchanged when key is missing")
	}
}

func TestEncryptionPlugin_Priority(t *testing.T) {
	plugin := NewEncryptionPlugin()

	// Should run at priority 50 (after markdown but before templates)
	priority := plugin.Priority(lifecycle.StageRender)
	if priority != 50 {
		t.Errorf("Priority(StageRender) = %d, want %d", priority, 50)
	}

	// Default priority for other stages
	priority = plugin.Priority(lifecycle.StageTransform)
	if priority != lifecycle.PriorityDefault {
		t.Errorf("Priority(StageTransform) = %d, want %d", priority, lifecycle.PriorityDefault)
	}
}

func TestEncryptionPlugin_EscapeHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello", "Hello"},
		{"<script>", "&lt;script&gt;"},
		{"&test", "&amp;test"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"it's", "it&#39;s"},
		{`<a href="test">&</a>`, "&lt;a href=&quot;test&quot;&gt;&amp;&lt;/a&gt;"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeHTML(tt.input)
			if got != tt.expected {
				t.Errorf("escapeHTML(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// Test that the plugin implements required interfaces
func TestEncryptionPlugin_Interfaces(_ *testing.T) {
	var _ lifecycle.Plugin = (*EncryptionPlugin)(nil)
	var _ lifecycle.ConfigurePlugin = (*EncryptionPlugin)(nil)
	var _ lifecycle.RenderPlugin = (*EncryptionPlugin)(nil)
	var _ lifecycle.PriorityPlugin = (*EncryptionPlugin)(nil)
}
