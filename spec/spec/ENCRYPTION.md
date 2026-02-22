# Encryption Specification

## Overview

The encryption system protects private post content using AES-256-GCM client-side encryption. Encrypted posts are served as ciphertext; decryption happens entirely in the visitor's browser.

**Core invariant:** Private posts must never be published with plaintext content. If a private post cannot be encrypted, the build must fail.

## Privacy Boundary

**Encryption protects content, not metadata.** The post body (`Content`, `ArticleHTML`) is private and encrypted. Frontmatter metadata -- title, description, tags, dates, slug, avatar -- is public and remains in cleartext.

This is by design:

- **Title** is preserved for page cards, feed listings, HTML `<title>`, navigation, and SEO.
- **Description** is preserved if explicitly set in frontmatter (the author chose to make it public). Auto-generated descriptions are suppressed for private posts.
- **Tags and dates** are preserved for site structure, filtering, and feed membership.
- **Slug and URL** are preserved so the page is routable and linkable.

Plugins that generate output from post data follow this boundary: they may use frontmatter fields freely but must not expose body content. The `scrubPrivateMetadata` function enforces this by clearing `Content` and content-derived fields (inlinks/outlinks text) while preserving frontmatter-provided fields.

## Configuration

### `[encryption]` Table

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | `bool` | `true` | Whether encryption processing is active |
| `default_key` | `string` | `"default"` | Key name used when a post has no explicit key |
| `decryption_hint` | `string` | `""` | Help text shown to visitors next to the password prompt |
| `private_tags` | `map[string]string` | `{}` | Maps tag names (or templateKey values) to encryption key names |
| `enforce_strength` | `bool` | `true` | Require keys to meet the configured strength policy before encrypting any private post |
| `min_estimated_crack_time` | `string` | `"10y"` | Minimum estimated crack time for each password (supports `y`, `d`, `h`, `m`, `s` units) |
| `min_password_length` | `int` | `14` | Minimum password length enforced for every encryption key |

### Environment Variables

Encryption keys are loaded from environment variables with the prefix `MARKATA_GO_ENCRYPTION_KEY_`:

```
MARKATA_GO_ENCRYPTION_KEY_{NAME} = password
```

Key name lookup is case-insensitive: `MARKATA_GO_ENCRYPTION_KEY_DEFAULT` matches key name `"default"`.

Config-level overrides:

| Variable | Overrides |
|----------|-----------|
| `MARKATA_GO_ENCRYPTION_ENABLED` | `encryption.enabled` |
| `MARKATA_GO_ENCRYPTION_DEFAULT_KEY` | `encryption.default_key` |
| `MARKATA_GO_ENCRYPTION_DECRYPTION_HINT` | `encryption.decryption_hint` |
| `MARKATA_GO_ENCRYPTION_ENFORCE_STRENGTH` | `encryption.enforce_strength` |
| `MARKATA_GO_ENCRYPTION_MIN_ESTIMATED_CRACK_TIME` | `encryption.min_estimated_crack_time` |
| `MARKATA_GO_ENCRYPTION_MIN_PASSWORD_LENGTH` | `encryption.min_password_length` |

### `.env` File Support

A `.env` file in the project root is loaded automatically during config loading (before config file parsing). Real environment variables take precedence over `.env` values.

## Password Strength Policy

All encryption keys used by private posts must satisfy the configured strength policy **before** any encryption occurs. The defaults are strict: `enforce_strength = true`, `min_estimated_crack_time = "10y"`, and `min_password_length = 14`. The policy evaluates each password without ever logging or storing the plaintext.

### Duration parsing

- The `min_estimated_crack_time` value supports `y` (365 days), `d` (24 hours), `h`, `m`, and `s` units and may combine them (e.g., `1y6d`).
- Standard `time.ParseDuration` units (`ns`, `us`, `ms`, `s`, `m`, `h`) are also accepted when you omit year/day units.

### Estimator assumptions

- The estimator assumes the attacker can make 10,000,000,000 guesses per second (10¹⁰) using specialized hardware.
- Entropy is computed as `log₂(charset_sizeᶰ)`, where `n` is the password length and `charset_size` reflects the union of lowercase, uppercase, digits, and symbol characters actually present.
- The resulting estimate is compared against `min_estimated_crack_time`. Passwords with too-short length or insufficient entropy cause a policy violation.

### Enforcement

- If any private post references a key whose password violates the policy, the build aborts with `EncryptionBuildError` (a `CriticalError`). The error lists the affected posts, the key name, and the policy reason — it never includes the plaintext password or hint.
- Strength enforcement runs before encryption so no private content is ever published if the policy fails.

## Data Model

### Post Fields

| Field | Type | Source | Description |
|-------|------|--------|-------------|
| `Private` | `bool` | frontmatter `private` | Whether the post is private |
| `SecretKey` | `string` | frontmatter `secret_key` / `private_key` / `encryption_key` | Which encryption key to use |

`SecretKey` frontmatter aliases are checked in priority order: `secret_key` > `private_key` > `encryption_key`. The first non-empty value wins.

### Post Extra Fields (Set by Plugin)

| Key | Type | Description |
|-----|------|-------------|
| `has_encrypted_content` | `bool` | `true` when post content has been encrypted |
| `encryption_key_name` | `string` | The key name that was used for encryption |

## Plugin Behavior

### Two-Phase Lifecycle

The encryption plugin participates in two lifecycle stages to ensure complete privacy protection:

#### Phase 1: Transform Stage (PriorityFirst / -1000)

Privacy marking runs at `PriorityFirst` (-1000) in the Transform stage -- before any other Transform or Render plugin. This ensures all downstream plugins see `post.Private == true` and can act accordingly.

**Processing:** Apply private tags. For each non-draft, non-skipped post, check if any of its tags match a `private_tags` entry. If no tag matches, also check the post's `Template` field (set from the `templateKey` or `template` frontmatter). If either matches, set `Private = true` and assign the matching key name (unless `SecretKey` is already set from frontmatter). Tag matches take priority over `templateKey` matches for key assignment.

**Rationale:** If privacy marking ran later (e.g., during Render), Transform-stage plugins like Description would auto-generate descriptions from private content before the post was marked private -- leaking plaintext into metadata.

#### Phase 2: Render Stage (Priority 50)

Encryption runs during the Render stage at priority 50 -- after markdown rendering (default priority) but before templates (priority 100).

**Processing:**

1. **Validate keys**: Find all private, non-draft, non-skipped posts. For each, resolve the key name (post's `SecretKey`, falling back to `default_key`). If no key name resolves, or the key's password is not found in the environment, record a failure.

2. **Fail on missing keys**: If any private posts failed validation, return an `EncryptionBuildError` (implements `CriticalError`). The error message lists all affected posts and the expected environment variable names.

3. **Encrypt content**: For each private post with non-empty `ArticleHTML`, encrypt the HTML using AES-256-GCM. Replace `ArticleHTML` with an encrypted wrapper containing:
   - The encrypted content as a base64 string in a `data-encrypted` attribute
   - The key name in a `data-key-name` attribute
   - A password input form with ARIA labels
   - The decryption hint (if configured)
   - A "Remember for this session" checkbox

### Cross-Plugin Privacy Protection

The following plugins respect `post.Private` to prevent leaking private content through non-article output:

| Plugin | Protection | Details |
|--------|-----------|---------|
| `publish_html` | Alternate formats suppressed | `.md`, `.txt`, and OG card outputs are skipped for private posts |
| `description` | Auto-generation skipped | Does not generate descriptions from private content |
| `embeds` | Private embed card | Shows a "Private Content" card instead of title/description/date |
| `wikilinks` | Metadata attributes suppressed | `data-title`, `data-description`, `data-date` attributes are omitted for private targets |
| `wikilink_hover` | Hover preview suppressed | No preview text or metadata shown for private targets |
| `feeds` / `atom` / `rss` / `jsonfeed` | Excluded from subscription feeds | Private posts are filtered out of RSS, Atom, and JSON Feed outputs |
| `auto_feeds` | Encrypted cards on feed pages | Tag feeds for `private_tags` set `IncludePrivate=true` so private posts appear as encrypted cards with password prompts. Non-private-tag feeds exclude private posts as usual. |

### Error Handling

`EncryptionBuildError` implements the `lifecycle.CriticalError` interface (`IsCritical() bool` returns `true`). This causes the lifecycle manager to halt the build even though the Render stage is normally non-critical.

### Skipped Posts

Posts with `Draft = true` or `Skip = true` are excluded from all encryption processing. They are not subject to key validation and are never encrypted.

### Disabled State

When `enabled = false`, the plugin's `Render()` method returns `nil` immediately. No posts are modified.

## Encryption Algorithm

- **Cipher**: AES-256-GCM
- **Key derivation**: PBKDF2 with 100,000 iterations, SHA-256, random 16-byte salt
- **IV**: Random 12 bytes
- **Output format**: Base64-encoded concatenation of salt + IV + ciphertext

Client-side decryption uses the Web Crypto API with matching parameters.

## CLI Utilities

### `encryption generate-password`

Generate a policy-compliant encryption password without invoking the full build. The command prints the generated password to stdout so it can be captured in scripts or piped into other tools.

```
markata-go encryption generate-password
markata-go encryption generate-password --length 20
```

- **Default length**: `14` (matches `min_password_length`).
- **Length flag**: `--length` allows requesting longer passwords; it is rejected if less than the configured minimum length.
- **Output**: password only to stdout (no extra text). Use shell redirection or copy/paste as needed.
- **Guarantees**: The generated password satisfies both the minimum length and estimated crack time thresholds.

### `encryption check`

Check configured key material against the active encryption policy without running a full build.

```
markata-go encryption check
markata-go encryption check --key default
```

- By default this checks all keys referenced by `default_key` and `private_tags`.
- The command exits non-zero if a required key is missing or fails policy checks.
- Output identifies key names and env var names only; plaintext passwords are never printed.

## Lint Integration

The `markata-go lint` command MUST include an encryption policy rule when encryption is enabled:

- report an error if a required encryption key is missing from environment variables,
- report an error if a required key fails `min_password_length` or `min_estimated_crack_time`,
- report a warning when `enforce_strength` is disabled, since builds will not enforce policy.

## Config Merging

When merging encryption configs (e.g., from multiple config files), the following rules apply:

- `enabled`, `default_key`, `decryption_hint`: Override takes precedence if it has any non-default values
- `private_tags`: Merged as maps; override entries take precedence over base entries for the same tag

## Cache Behavior

The `SecretKey` field is persisted in the build cache (`CachedPostData.SecretKey`). This ensures posts restored from cache retain their encryption key assignment.
