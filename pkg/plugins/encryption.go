// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"fmt"
	"os"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/encryption"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// EncryptionEnvPrefix is the prefix for encryption key environment variables.
const EncryptionEnvPrefix = "MARKATA_GO_ENCRYPTION_KEY_"

// EncryptionBuildError is returned when a private post cannot be encrypted.
// It implements lifecycle.CriticalError to halt the build and prevent
// exposing unencrypted private content.
type EncryptionBuildError struct {
	Posts []string
	Msg   string
}

func (e *EncryptionBuildError) Error() string {
	return fmt.Sprintf("encryption error: %s (posts: %s)", e.Msg, strings.Join(e.Posts, ", "))
}

// IsCritical marks this error as a build-halting error.
// Private posts must never be published without encryption.
func (e *EncryptionBuildError) IsCritical() bool {
	return true
}

// EncryptionPlugin encrypts content for private posts.
// It runs during the Render stage (after markdown is converted to HTML) to encrypt
// the ArticleHTML content.
//
// # Encryption is enabled by default
//
// The plugin is enabled by default with default_key="default". Users only need to
// set MARKATA_GO_ENCRYPTION_KEY_DEFAULT in their environment or .env file.
//
// # How It Works
//
//  1. Posts with private: true are automatically encrypted
//  2. Posts with tags matching [encryption.private_tags] are treated as private
//  3. Frontmatter secret_key (or aliases: private_key, encryption_key) specifies which key to use
//  4. If no key is specified, the default_key is used
//  5. The build FAILS if a private post has no available encryption key
//
// # Client-Side Decryption
//
// The encrypted content is wrapped in a div with data attributes.
// The client-side JavaScript uses Web Crypto API with matching PBKDF2 parameters.
type EncryptionPlugin struct {
	enabled        bool
	defaultKey     string
	decryptionHint string
	privateTags    map[string]string // tag -> key name
	// keys maps key names to passwords (loaded from env vars)
	keys map[string]string
}

// NewEncryptionPlugin creates a new EncryptionPlugin.
func NewEncryptionPlugin() *EncryptionPlugin {
	return &EncryptionPlugin{
		keys:        make(map[string]string),
		privateTags: make(map[string]string),
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
		if modelsConfig.Encryption.PrivateTags != nil {
			for tag, key := range modelsConfig.Encryption.PrivateTags {
				p.privateTags[strings.ToLower(tag)] = key
			}
		}
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
// Encryption should run in the middle of the Render stage:
// - After markdown is rendered to HTML (PriorityDefault/PriorityEarly)
// - Before templates wrap the HTML (PriorityLate)
func (p *EncryptionPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageRender {
		// Run after markdown rendering but before templates
		// Templates run at PriorityLate (100), so we run at 50
		return 50
	}
	return lifecycle.PriorityDefault
}

// applyPrivateTags marks posts as private based on configured private_tags.
// If a post has a tag matching a private_tags entry, it is marked Private=true
// and assigned the tag's key (unless the post already has a frontmatter secret_key).
func (p *EncryptionPlugin) applyPrivateTags(posts []*models.Post) {
	if len(p.privateTags) == 0 {
		return
	}

	for _, post := range posts {
		if post.Skip || post.Draft {
			continue
		}
		for _, tag := range post.Tags {
			tagLower := strings.ToLower(tag)
			if keyName, ok := p.privateTags[tagLower]; ok {
				post.Private = true
				// Only set key from tag if frontmatter didn't specify one
				if post.SecretKey == "" {
					post.SecretKey = keyName // pragma: allowlist secret
				}
				break // one matching tag is enough
			}
		}
	}
}

// Render encrypts content for private posts with encryption keys.
// Returns a CriticalError if any private post cannot be encrypted,
// preventing unencrypted private content from being published.
func (p *EncryptionPlugin) Render(m *lifecycle.Manager) error {
	if !p.enabled {
		return nil
	}

	// Apply private tags to mark posts as private based on their tags
	p.applyPrivateTags(m.Posts())

	// Find all private posts (whether they need encryption or not)
	privatePosts := m.FilterPosts(func(post *models.Post) bool {
		return !post.Skip && !post.Draft && post.Private
	})

	if len(privatePosts) == 0 {
		return nil
	}

	// Check that every private post can be encrypted.
	// This is a safety check: we must NEVER expose private content unencrypted.
	var failedPosts []string
	for _, post := range privatePosts {
		keyName := post.SecretKey
		if keyName == "" {
			keyName = p.defaultKey
		}
		if keyName == "" {
			// No key name at all - no way to encrypt
			failedPosts = append(failedPosts, fmt.Sprintf("%s (no encryption key specified and no default key configured)", post.Path))
			continue
		}
		if _, err := p.getKeyPassword(keyName); err != nil {
			failedPosts = append(failedPosts, fmt.Sprintf("%s (key %q: set %s%s in environment or .env)",
				post.Path, keyName, EncryptionEnvPrefix, strings.ToUpper(keyName)))
		}
	}

	if len(failedPosts) > 0 {
		return &EncryptionBuildError{
			Posts: failedPosts,
			Msg:   "private posts found without available encryption keys. Build halted to prevent exposing private content",
		}
	}

	// Filter to posts that actually have content to encrypt
	postsToEncrypt := m.FilterPosts(func(post *models.Post) bool {
		return p.shouldEncrypt(post)
	})

	if len(postsToEncrypt) == 0 {
		return nil
	}

	return m.ProcessPostsSliceConcurrently(postsToEncrypt, func(post *models.Post) error {
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
		// This should not happen because we already validated all keys in Render(),
		// but if it does, fail hard to protect private content.
		return &EncryptionBuildError{
			Posts: []string{post.Path},
			Msg:   fmt.Sprintf("encryption key %q not found", keyName),
		}
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

	// Generate unique ID for accessibility (based on post path hash)
	inputID := fmt.Sprintf("decrypt-input-%d", hashString(post.Path))
	checkboxID := fmt.Sprintf("decrypt-remember-%d", hashString(post.Path))

	// Replace ArticleHTML with encrypted wrapper
	// Includes:
	// - data-key-name for multi-post unlock feature
	// - ARIA labels for accessibility
	// - Remember me checkbox for explicit session storage opt-in
	post.ArticleHTML = fmt.Sprintf(`<div class="encrypted-content" data-encrypted="%s" data-key-name="%s" role="region" aria-label="Encrypted content">
  <div class="encrypted-content__locked">
    <div class="encrypted-content__icon" aria-hidden="true">🔒</div>
    <h3 class="encrypted-content__title" id="encrypt-title-%d">Encrypted Content</h3>
    <p class="encrypted-content__message">This content is encrypted. Enter the password to decrypt and view.</p>
    %s
    <div class="encrypted-content__form" role="form" aria-labelledby="encrypt-title-%d">
      <label for="%s" class="sr-only">Password</label>
      <input type="password" id="%s" class="encrypted-content__input" placeholder="Enter password" autocomplete="off" aria-describedby="encrypt-error-%d">
      <button type="button" class="encrypted-content__button" aria-busy="false">Decrypt</button>
    </div>
    <label class="encrypted-content__remember-label">
      <input type="checkbox" id="%s" class="encrypted-content__remember" aria-describedby="remember-desc-%d">
      <span>Remember for this session</span>
    </label>
    <span id="remember-desc-%d" class="sr-only">If checked, the password will be saved in your browser for this session only. It will be cleared when you close the browser.</span>
    <p id="encrypt-error-%d" class="encrypted-content__error" style="display: none;" role="alert" aria-live="assertive"></p>
  </div>
  <div class="encrypted-content__decrypted" style="display: none;" tabindex="-1"></div>
</div>`, encryptedContent, escapeHTML(keyName), hashString(post.Path), hintHTML,
		hashString(post.Path), inputID, inputID, hashString(post.Path),
		checkboxID, hashString(post.Path), hashString(post.Path), hashString(post.Path))

	// Mark post as having encrypted content (for template to include decryption script)
	post.Set("has_encrypted_content", true)
	post.Set("encryption_key_name", keyName)

	// Invalidate the post map cache so templates pick up the new Extra fields
	templates.InvalidatePost(post)

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

// hashString generates a simple numeric hash for a string.
// Used for generating unique IDs in HTML elements.
func hashString(s string) uint32 {
	var hash uint32
	for _, c := range s {
		hash = hash*31 + uint32(c)
	}
	return hash
}

// getModelsConfig attempts to extract models.Config from lifecycle.Config.
// This handles the conversion between the lifecycle config and models config.
func getModelsConfig(config *lifecycle.Config) (*models.Config, bool) {
	if config == nil || config.Extra == nil {
		return nil, false
	}

	// The models.Config is stored in Extra["models_config"] (set by core.go)
	if modelsConfig, ok := config.Extra["models_config"].(*models.Config); ok {
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
