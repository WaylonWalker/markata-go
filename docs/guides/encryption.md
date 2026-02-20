---
title: "Encryption and Private Posts"
description: "Protect sensitive content with client-side encryption using AES-256-GCM"
date: 2024-01-15
published: true
slug: /docs/guides/encryption/
tags:
  - documentation
  - encryption
  - privacy
  - security
---

# Encryption and Private Posts

markata-go can encrypt post content so that only visitors with the correct password can read it. Encryption uses AES-256-GCM and runs entirely client-side -- your server never sees the decrypted content.

> **Key principle:** Private posts are **never published in plaintext**. If the encryption key is missing, the build fails rather than exposing your content.

## Quick Start

1. Create a `.env` file in your project root:

```bash
MARKATA_GO_ENCRYPTION_KEY_DEFAULT=your-secret-password
```

2. Mark a post as private in its frontmatter:

```yaml
---
title: My Private Post
private: true
---
Your secret content here.
```

3. Build your site:

```bash
markata-go build
```

The post's HTML content is replaced with an encrypted blob and a password prompt. Visitors enter the password in their browser to decrypt and view the content.

## How It Works

1. During the build, the encryption plugin finds all posts with `private: true`
2. It encrypts the rendered HTML using AES-256-GCM with a password derived via PBKDF2
3. The encrypted content is embedded as a base64 string in a `data-encrypted` attribute
4. A password form is rendered in place of the content
5. Client-side JavaScript uses the Web Crypto API to decrypt on password entry

**The build fails if any private post cannot be encrypted.** This prevents accidentally publishing sensitive content.

## Configuration

Encryption is **enabled by default** with `default_key = "default"`. You only need to set the environment variable.

### Config File Options

```toml
[encryption]
enabled = true                           # default: true
default_key = "default"                  # default: "default"
decryption_hint = "DM me for access"     # optional hint shown to visitors

[encryption.private_tags]
diary = "personal"                       # tag "diary" encrypts with key "personal"
draft-ideas = "default"                  # tag "draft-ideas" encrypts with key "default"
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable encryption processing |
| `default_key` | string | `"default"` | Key name used when a post doesn't specify one |
| `decryption_hint` | string | `""` | Help text shown next to the password prompt |
| `private_tags` | map | `{}` | Maps tag names to encryption key names |

### Environment Variables

Encryption keys are loaded from environment variables with the prefix `MARKATA_GO_ENCRYPTION_KEY_`:

```bash
# The default key (used when no specific key is set on a post)
MARKATA_GO_ENCRYPTION_KEY_DEFAULT=my-password

# Named keys for different access levels
MARKATA_GO_ENCRYPTION_KEY_PERSONAL=personal-password
MARKATA_GO_ENCRYPTION_KEY_PREMIUM=premium-password
```

You can also override config options via environment:

```bash
MARKATA_GO_ENCRYPTION_ENABLED=true
MARKATA_GO_ENCRYPTION_DEFAULT_KEY=default
MARKATA_GO_ENCRYPTION_DECRYPTION_HINT="Contact me for access"
```

### .env File Support

Place a `.env` file in your project root. It is loaded automatically during the build:

```bash
# .env
MARKATA_GO_ENCRYPTION_KEY_DEFAULT=my-secret-password
MARKATA_GO_ENCRYPTION_KEY_PERSONAL=another-password
```

Rules:
- Lines starting with `#` are comments
- Values can be quoted with single or double quotes
- Real environment variables take precedence over `.env` values
- The `.env` file should be in your `.gitignore`

## Making Posts Private

There are three ways to make a post private:

### 1. Frontmatter `private: true`

The simplest approach. The post is encrypted with the default key:

```yaml
---
title: My Secret Post
private: true
---
```

### 2. Frontmatter with a specific key

Use `secret_key` (or its aliases `private_key`, `encryption_key`) to encrypt with a named key:

```yaml
---
title: Premium Content
private: true
secret_key: premium
---
```

This looks for `MARKATA_GO_ENCRYPTION_KEY_PREMIUM` in the environment.

All three frontmatter fields are equivalent -- use whichever name you prefer:

| Field | Example |
|-------|---------|
| `secret_key` | `secret_key: premium` |
| `private_key` | `private_key: premium` |
| `encryption_key` | `encryption_key: premium` |

If multiple are set, `secret_key` takes priority, then `private_key`, then `encryption_key`.

### 3. Private tags

Configure tags that automatically mark posts as private:

```toml
[encryption.private_tags]
diary = "personal"
journal = "personal"
```

Any post tagged `diary` or `journal` is automatically treated as private and encrypted with the `personal` key. You don't need to set `private: true` in the frontmatter.

The `private_tags` check matches against both the post's `tags` list and its `templateKey` (or `template`) frontmatter field. This is useful for content that uses `templateKey` as its primary categorization, such as gratitude journals or diary entries that may not have explicit tags.

```toml
[encryption.private_tags]
gratitude = "default"    # Matches posts with tag "gratitude" OR templateKey "gratitude"
```

**Priority rules:**
- If a tag matches, the tag's key is used
- If only `templateKey` matches, its key is used
- **Frontmatter key overrides both:** If a post has a frontmatter `secret_key`, it takes priority over any tag or templateKey match

## Build Behavior

### Missing Keys Fail the Build

If any private post has no available encryption key, the build **fails with a critical error** listing all affected posts and the expected environment variables:

```
encryption error: private posts found without available encryption keys.
Build halted to prevent exposing private content
(posts: diary/2024-01-15.md (key "personal": set MARKATA_GO_ENCRYPTION_KEY_PERSONAL in environment or .env))
```

This is intentional. Private content must never be published unencrypted.

### Draft and Skipped Posts

Posts with `draft: true` or `skip: true` are excluded from encryption checks. They are not published at all, so they don't need encryption.

### Disabling Encryption

To disable encryption entirely:

```toml
[encryption]
enabled = false
```

When disabled, the encryption plugin does nothing. Private posts pass through unmodified (they are still rendered but not encrypted). Use this only for local development.

## Multiple Access Levels

You can use different keys for different audiences:

```toml
[encryption]
default_key = "default"
decryption_hint = "Contact me for the password"

[encryption.private_tags]
diary = "personal"
premium = "subscribers"
```

```bash
# .env
MARKATA_GO_ENCRYPTION_KEY_DEFAULT=general-password
MARKATA_GO_ENCRYPTION_KEY_PERSONAL=my-eyes-only
MARKATA_GO_ENCRYPTION_KEY_SUBSCRIBERS=subscriber-password
```

Then in your posts:

```yaml
# Uses default key
---
private: true
---

# Uses personal key (via tag)
---
tags: [diary]
---

# Uses subscribers key (explicit)
---
private: true
secret_key: subscribers
---
```

## Client-Side Decryption

The encrypted content includes:
- A lock icon and "Encrypted Content" heading
- The decryption hint (if configured)
- A password input field
- A "Remember for this session" checkbox (uses sessionStorage)

When the correct password is entered, JavaScript decrypts the content in-browser using the Web Crypto API with matching PBKDF2 parameters. The decrypted HTML replaces the password form.

If "Remember for this session" is checked, the password is stored in sessionStorage (cleared when the browser tab closes). This allows navigating between encrypted posts without re-entering the password for posts using the same key.

## Security Notes

- **AES-256-GCM** encryption with random IVs
- **PBKDF2** key derivation with 100,000 iterations
- Encryption happens at build time; decryption happens client-side
- The server only ever serves encrypted content
- Passwords are never transmitted to the server
- Session storage is opt-in and per-tab only
