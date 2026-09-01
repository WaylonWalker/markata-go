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

## Source-Encrypted Markdown

Markata-go supports encrypting the Markdown body on disk so private posts can be safely stored in a git repository. Source encryption protects only the Markdown body; YAML frontmatter remains cleartext for routing, feeds, dates, titles, tags, and key selection.

Source-encrypted bodies are self-identifying. They do not require an `encrypted_source` frontmatter key. The body starts with this marker:

```markdown
<!-- markata-encrypted-source:v1 key=personal -->
BASE64_AES_GCM_CIPHERTEXT
```

The optional `key` attribute records the key name used to encrypt the source body. If it is present, the loader uses that key name for source decryption and for the post's `SecretKey`. If the marker omits `key`, the loader falls back to frontmatter `secret_key` / `private_key` / `encryption_key`, then `encryption.default_key`.

During the Load stage, encrypted source bodies are decrypted in memory before normal Markdown rendering. The decrypted Markdown body MUST NOT be written back to the source file, build output, or parsed-post cache. The final Render-stage encryption still encrypts private post HTML for browser delivery.

When encryption is disabled with `encryption.enabled = false`, source-body decryption is skipped and the encrypted marker remains ordinary Markdown content.

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

- Password strength estimation uses the `zxcvbn` algorithm (`github.com/nbutton23/zxcvbn-go`), which detects common passwords, dictionary words, keyboard walks, repeats, and predictable substitutions.
- Crack-time estimates follow zxcvbn's offline slow-hash model: 10ms per guess split across 100 attackers (about 10,000 guesses/second).
- The resulting estimate is compared against `min_estimated_crack_time`. Passwords with too-short length or insufficient estimated crack time cause a policy violation.

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

#### Load Stage Source Decryption

Before Transform and Render plugins run, the loader detects source-encrypted Markdown bodies and decrypts them in memory using the configured key material from `MARKATA_GO_ENCRYPTION_KEY_{NAME}`. This allows all downstream plugins to operate on normal Markdown while keeping the on-disk file encrypted.

If a source-encrypted body references a missing key or cannot be decrypted, loading that post fails with an actionable error that names the key source but never prints the password.

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

The canonical post wrapper retains `data-key-name` for same-key unlock and
session storage. Before an encrypted wrapper is copied into an opted-in feed,
the feed privacy projection removes that attribute. Feed entries can still be
decrypted manually, but they do not disclose key names or support cross-entry
unlocking and password storage.

### Client initialization and focus

The browser decryption initializer MUST be idempotent. It MUST initialize
encrypted wrappers that are inserted into the DOM after an outer wrapper is
decrypted, without adding duplicate event listeners to wrappers that were
already initialized.

Initialization on page load, and decryption started from a remembered session
password, MUST NOT focus a password input or change the reader's scroll
position. An explicit user submission MAY move focus to the revealed content
for accessibility, but that focus operation MUST preserve the current scroll
position.

After decrypted HTML is inserted, same-key unlock MUST inspect the live DOM so
newly revealed canonical wrappers can participate in the existing multi-post
unlock behavior. Wrappers projected into feeds intentionally omit
`data-key-name`; those wrappers remain manual, isolated decryption boundaries
and MUST NOT gain implicit session storage or cross-entry unlocking.

### Cross-Plugin Privacy Protection

The following plugins respect `post.Private` to prevent leaking private content through non-article output:

| Plugin | Protection | Details |
|--------|-----------|---------|
| `publish_html` | Alternate formats suppressed | `.md`, `.txt`, and OG card outputs are skipped for private posts |
| `description` | Auto-generation skipped | Does not generate descriptions from private content |
| `embeds` | Private embed card | Shows a "Private Content" card instead of title/description/date |
| `wikilinks` | Metadata attributes suppressed | `data-title`, `data-description`, `data-date` attributes are omitted for private targets |
| `wikilink_hover` | Hover preview suppressed | No preview text or metadata shown for private targets |
| `feeds` / `atom` / `rss` / `jsonfeed` | Excluded unless explicitly opted in | Private posts are filtered out of public feed pages plus RSS, Atom, and JSON Feed outputs unless a feed explicitly opts into `include_private=true`; when included, private entries may expose explicitly authored metadata but must render only encrypted HTML content and must not expose body-derived summaries, raw content, key names, or private media |
| `auto_feeds` | Private-tag feeds may opt in | Auto-generated tag feeds for tags listed in `private_tags` opt into private posts so encrypted post content can render in those feed outputs; other auto-generated feeds remain public-only |

### Error Handling

`EncryptionBuildError` implements the `lifecycle.CriticalError` interface (`IsCritical() bool` returns `true`). This causes the lifecycle manager to halt the build even though the Render stage is normally non-critical.

Encryption key validation MUST run during the Transform stage after private tag/templateKey assignment. This fails the build early before expensive render plugins run when keys are missing or violate policy. Render still re-validates before encryption as a safety check.

When key validation fails, the error output MUST be summarized by key and reason. Each summary entry includes:

- key name
- policy or missing-key reason
- total count of affected posts
- a short sample list (up to 3 paths)

The full list of post paths should not be printed in the error message.

### Skipped Posts

Posts with `Draft = true` or `Skip = true` are excluded from all encryption processing. They are not subject to key validation and are never encrypted.

An explicit `private: false` is also an opt-out from `encryption.private_tags` and
template-key privacy defaults. This allows an individual post to remain public
even when its tag or template normally marks posts private.

### Disabled State

When `enabled = false`, the plugin's `Render()` method returns `nil` immediately. No posts are modified.

## Encryption Algorithm

- **Cipher**: AES-256-GCM
- **Key derivation**: PBKDF2 with 100,000 iterations, SHA-256, random 16-byte salt
- **IV**: Random 12 bytes
- **Output format**: Base64-encoded concatenation of salt + IV + ciphertext

Client-side decryption uses the Web Crypto API with matching parameters.

## CLI Utilities

### Hidden Root Shortcuts

The CLI provides hidden root-level shortcuts for the common source-encryption workflow:

```
markata-go encrypt [flags]
markata-go decrypt [path...] [flags]
```

`encrypt` has the same behavior and flags as `encryption encrypt-posts`; `decrypt` has the same behavior and flags as `encryption decrypt-posts`. These shortcuts are intentionally omitted from standard help and user documentation, while remaining available for interactive use and automation.

Bulk-command status labels, paths, and key names use the active CLI theme when output is a terminal. `--color` forces color, while `--no-color`, `--log-format plain`, `NO_COLOR`, and non-terminal output use plain text.

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

### `encryption encrypt-post`

Encrypt a single private Markdown source body. By default the command rewrites the file in place.

```
markata-go encryption encrypt-post path/to/post.md
markata-go encryption encrypt-post path/to/post.md --stdout
```

- The post must already be private by frontmatter or `private_tags`.
- Key selection follows the build rules: explicit frontmatter key, then matching `private_tags`, then `default_key`.
- `--stdout` writes the transformed Markdown document to stdout and does not modify the file.
- Already encrypted source bodies are skipped unless `--force` is passed.

### `encryption encrypt-posts`

Encrypt all private Markdown source bodies matched by the active content glob configuration.

```
markata-go encryption encrypt-posts
markata-go encryption encrypt-posts --workers 4
markata-go encryption encrypt-posts --dry-run
```

- The command rewrites matching files in place by default.
- `--dry-run` reports which files would be encrypted without modifying the filesystem.
- `--workers` bounds concurrent source-body preparation. The default (`0`) uses `GOMAXPROCS`; set `--workers 1` to process serially.
- Posts that are draft, skipped, public, or already source-encrypted are not rewritten.
- Missing or weak keys cause the command to fail before writing changed files.

### `encryption decrypt-posts`

Decrypt source-encrypted Markdown bodies back to plaintext. This is the inverse of `encryption encrypt-posts`.

```
markata-go encryption decrypt-posts
markata-go encryption decrypt-posts --dry-run
markata-go encryption decrypt-posts path/to/post.md
markata-go encryption decrypt-posts content/private/
markata-go encryption decrypt-posts --workers 4
```

- With no path arguments the command scans the active content glob configuration.
- Explicit path arguments MAY be files or directories; directories are scanned recursively for `.md` files.
- The command rewrites matching files in place by default, replacing the encrypted body with plaintext and preserving the original frontmatter block verbatim.
- `--dry-run` reports which files would be decrypted without modifying the filesystem.
- Files whose bodies are not source-encrypted are counted as skipped and never rewritten.
- The key name is read from the encrypted source marker (`key=...`), falling back to `encryption.default_key`.
- The password is read from `MARKATA_GO_ENCRYPTION_KEY_<KEY>`. A missing env var or an incorrect password MUST fail before any file is written.
- Key strength policy is NOT enforced for decryption; only the correct password matters.
- `--workers` bounds concurrent source-body preparation. The default (`0`) uses `GOMAXPROCS`; set `--workers 1` to process serially.

### Source encryption round trips

The source-encryption commands store authenticated ciphertext in the dedicated
`.markata/source-encryption-cache.json` file. After `decrypt-posts`, an
unchanged body passed to `encrypt-posts` is decrypted and compared with the
current body before its original ciphertext and nonce are reused. Only changed
source bodies are re-encrypted, so a decrypt/edit/encrypt cycle does not create
repository-wide ciphertext churn. The cache contains no plaintext, password,
or password-derived verifier. Cache failures are non-fatal; encryption falls
back to fresh ciphertext and reports a warning.

### Source Command Write Safety

`encrypt-posts` and `decrypt-posts` prepare every candidate document concurrently, then perform writes only after all preparation succeeds. Before writing, the commands verify that each source still matches the content read during preparation. Each replacement uses an atomic rename; a write failure restores documents already written and reports any restoration failure. Consequently, a malformed post, missing key, invalid password, encryption/decryption failure, or detected concurrent edit leaves source files unchanged. Command output remains in the input-file order regardless of worker completion order.

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

Parsed-post cache entries MUST NOT be written for source-encrypted Markdown files because they would contain decrypted Markdown bodies. Source-encrypted files are parsed on each build.
