// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"os"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/encryption"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// EncryptionEnvPrefix is the prefix for encryption key environment variables.
const EncryptionEnvPrefix = "MARKATA_GO_ENCRYPTION_KEY_"

// EncryptionPlugin encrypts content for private posts that have a secret_key specified.
// It runs during the Render stage (after markdown is converted to HTML) to encrypt
// the ArticleHTML content.
//
// # How It Works
//
// 1. Checks if encryption is enabled in config
// 2. For posts with private: true and secret_key: "key_name":
//   - Looks up the encryption password from MARKATA_GO_ENCRYPTION_KEY_{KEY_NAME}
//   - Encrypts post.ArticleHTML using AES-256-GCM
//   - Replaces ArticleHTML with a wrapper div containing the encrypted content
//   - Marks the post for client-side decryption script inclusion
//
// # Client-Side Decryption
//
// The encrypted content is wrapped in a div with data attributes:
//
//	<div class="encrypted-content" data-encrypted="base64..." data-hint="...">
//	  <p>This content is encrypted. Enter the password to view.</p>
//	  <input type="password" placeholder="Password">
//	  <button>Decrypt</button>
//	</div>
//
// The client-side JavaScript uses Web Crypto API with matching PBKDF2 parameters.
type EncryptionPlugin struct {
	enabled        bool
	defaultKey     string
	decryptionHint string
	// keys maps key names to passwords (loaded from env vars)
	keys map[string]string
}

// NewEncryptionPlugin creates a new EncryptionPlugin.
func NewEncryptionPlugin() *EncryptionPlugin {
	return &EncryptionPlugin{
		keys: make(map[string]string),
	}
}

// Name returns the unique name of the plugin.
func (p *EncryptionPlugin) Name() string {
	return "encryption"
}

// Configure loads encryption configuration and encryption keys from environment.
func (p *EncryptionPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()

	// Check for models.Config via Extra or direct config access
	if modelsConfig, ok := getModelsConfig(config); ok {
		p.enabled = modelsConfig.Encryption.Enabled
		p.defaultKey = modelsConfig.Encryption.DefaultKey
		p.decryptionHint = modelsConfig.Encryption.DecryptionHint
	}

	// Also check Extra for backward compatibility
	if config.Extra != nil {
		if enabled, ok := config.Extra["encryption_enabled"].(bool); ok {
			p.enabled = enabled
		}
		if defaultKey, ok := config.Extra["encryption_default_key"].(string); ok {
			p.defaultKey = defaultKey
		}
		if hint, ok := config.Extra["encryption_decryption_hint"].(string); ok {
			p.decryptionHint = hint
		}
	}

	if !p.enabled {
		return nil
	}

	// Load encryption keys from environment variables
	p.loadKeysFromEnvironment()

	return nil
}

// loadKeysFromEnvironment scans environment for MARKATA_GO_ENCRYPTION_KEY_* variables.
func (p *EncryptionPlugin) loadKeysFromEnvironment() {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, EncryptionEnvPrefix) {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				// Extract key name (lowercase for case-insensitive lookup)
				keyName := strings.ToLower(strings.TrimPrefix(parts[0], EncryptionEnvPrefix))
				password := parts[1]
				if password != "" {
					p.keys[keyName] = password
				}
			}
		}
	}
}

// getKeyPassword returns the password for a given key name.
// Falls back to default key if the specific key is not found.
func (p *EncryptionPlugin) getKeyPassword(keyName string) (string, error) {
	// Try exact key first (case-insensitive)
	if password, ok := p.keys[strings.ToLower(keyName)]; ok {
		return password, nil
	}

	// Try default key
	if p.defaultKey != "" {
		if password, ok := p.keys[strings.ToLower(p.defaultKey)]; ok {
			return password, nil
		}
	}

	return "", fmt.Errorf("encryption key %q not found in environment (expected %s%s)",
		keyName, EncryptionEnvPrefix, strings.ToUpper(keyName))
}

// Priority returns the plugin priority for the given stage.
// Encryption should run late in the Render stage, after markdown is rendered to HTML.
func (p *EncryptionPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageRender {
		return lifecycle.PriorityLast // Run after all other render plugins
	}
	return lifecycle.PriorityDefault
}

// Render encrypts content for private posts with encryption keys.
func (p *EncryptionPlugin) Render(m *lifecycle.Manager) error {
	if !p.enabled {
		return nil
	}

	// Find posts that need encryption
	posts := m.FilterPosts(func(post *models.Post) bool {
		return p.shouldEncrypt(post)
	})

	if len(posts) == 0 {
		return nil
	}

	// Check if we have any keys loaded
	if len(p.keys) == 0 {
		// Warn but don't fail - user may have forgotten to set env vars
		for _, post := range posts {
			post.Set("encryption_warning", "Post marked for encryption but no encryption keys found in environment")
		}
		return nil
	}

	return m.ProcessPostsSliceConcurrently(posts, func(post *models.Post) error {
		return p.encryptPost(post)
	})
}

// shouldEncrypt determines if a post should have its content encrypted.
func (p *EncryptionPlugin) shouldEncrypt(post *models.Post) bool {
	if post.Skip || post.Draft {
		return false
	}

	// Must be private
	if !post.Private {
		return false
	}

	// Must have a secret_key or default key must be set
	if post.SecretKey == "" && p.defaultKey == "" {
		return false
	}

	// Must have content to encrypt
	if post.ArticleHTML == "" {
		return false
	}

	return true
}

// encryptPost encrypts the post's ArticleHTML content.
func (p *EncryptionPlugin) encryptPost(post *models.Post) error {
	keyName := post.SecretKey
	if keyName == "" {
		keyName = p.defaultKey
	}

	password, err := p.getKeyPassword(keyName)
	if err != nil {
		// Store warning and skip encryption for this post
		post.Set("encryption_error", err.Error())
		return nil
	}

	// Encrypt the article HTML
	encryptedContent, err := encryption.Encrypt([]byte(post.ArticleHTML), password)
	if err != nil {
		return fmt.Errorf("failed to encrypt post %q: %w", post.Path, err)
	}

	// Build the decryption hint HTML
	hintHTML := ""
	if p.decryptionHint != "" {
		hintHTML = fmt.Sprintf(`<p class="encrypted-content__hint">%s</p>`, escapeHTML(p.decryptionHint))
	}

	// Replace ArticleHTML with encrypted wrapper
	post.ArticleHTML = fmt.Sprintf(`<div class="encrypted-content" data-encrypted="%s">
  <div class="encrypted-content__locked">
    <div class="encrypted-content__icon">🔒</div>
    <h3 class="encrypted-content__title">Encrypted Content</h3>
    <p class="encrypted-content__message">This content is encrypted. Enter the password to decrypt and view.</p>
    %s
    <div class="encrypted-content__form">
      <input type="password" class="encrypted-content__input" placeholder="Enter password" autocomplete="off">
      <button type="button" class="encrypted-content__button">Decrypt</button>
    </div>
    <p class="encrypted-content__error" style="display: none;"></p>
  </div>
  <div class="encrypted-content__decrypted" style="display: none;"></div>
</div>`, encryptedContent, hintHTML)

	// Mark post as having encrypted content (for template to include decryption script)
	post.Set("has_encrypted_content", true)
	post.Set("encryption_key_name", keyName)

	return nil
}

// escapeHTML escapes special characters for safe HTML output.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// getModelsConfig attempts to extract models.Config from lifecycle.Config.
// This handles the conversion between the lifecycle config and models config.
func getModelsConfig(config *lifecycle.Config) (*models.Config, bool) {
	if config == nil || config.Extra == nil {
		return nil, false
	}

	// The models.Config is typically stored in Extra["_models_config"]
	// or we can access individual encryption fields
	if modelsConfig, ok := config.Extra["_models_config"].(*models.Config); ok {
		return modelsConfig, true
	}

	return nil, false
}

// Ensure EncryptionPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*EncryptionPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*EncryptionPlugin)(nil)
	_ lifecycle.RenderPlugin    = (*EncryptionPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*EncryptionPlugin)(nil)
)
