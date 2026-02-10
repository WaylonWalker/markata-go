---
title: "Standard Web Txt Files"
description: "Generate robots.txt, llms.txt, humans.txt and other standard web txt files at their canonical URLs"
date: 2026-01-26
published: true
tags:
  - configuration
  - output
  - seo
  - standards
---

# Standard Web Txt Files

markata-go supports generating standard web txt files at their expected canonical URLs. This enables creating files like `robots.txt`, `llms.txt`, and `humans.txt` that web crawlers, AI models, and humans expect to find at the root of your site.

## Quick Start

1. **Text format is enabled by default** - no configuration needed

2. **Create a markdown file** for your txt content:

```markdown
<!-- robots.md -->
---
title: "Robots"
slug: robots
published: true
---

User-agent: *
Allow: /
Disallow: /private/
```

3. **Build your site** - markata-go generates:
   - `/robots.txt` - The canonical robots.txt file
   - `/robots/index.txt` - Redirect for backwards compatibility
   - `/robots/index.txt/index.html` - HTML redirect for static hosts

## Common Txt Files

### robots.txt

The robot exclusion standard for web crawlers:

```markdown
---
title: "Robots"
slug: robots
published: true
---

User-agent: *
Allow: /

User-agent: GPTBot
Disallow: /

Sitemap: https://example.com/sitemap.xml
```

### llms.txt

Guidance for AI language models ([llms.txt standard](https://llmstxt.org)):

```markdown
---
title: "LLMs"
slug: llms
published: true
---

# LLMs.txt

> This site welcomes AI training on its content.

## Site Information
- Name: Example Site
- URL: https://example.com
- Author: Jane Doe

## Guidelines
- Attribution appreciated but not required
- Commercial use allowed
- Respect rate limits

## Notable Content
- /blog/ - Technical articles
- /docs/ - Documentation
```

### humans.txt

Human-readable site credits ([humanstxt.org](https://humanstxt.org)):

```markdown
---
title: "Humans"
slug: humans
published: true
---

/* TEAM */
Developer: Jane Doe
Contact: jane@example.com
Twitter: @janedoe
Location: San Francisco, CA

/* SITE */
Last update: 2026/01/26
Standards: HTML5, CSS3
Software: markata-go
```

### security.txt

Security contact information ([securitytxt.org](https://securitytxt.org)):

```markdown
---
title: "Security"
slug: .well-known/security
published: true
---

Contact: security@example.com
Expires: 2027-01-01T00:00:00.000Z
Preferred-Languages: en
Canonical: https://example.com/.well-known/security.txt
```

## Auto-Generated .well-known Entries

markata-go can generate additional `.well-known` endpoints directly from your site metadata. These do not require markdown source files.

### Default Auto-Generated Entries

- `/.well-known/host-meta`
- `/.well-known/host-meta.json`
- `/.well-known/webfinger`
- `/.well-known/nodeinfo` and `/nodeinfo/2.0`
- `/.well-known/time`

### Configuration

```toml
[markata-go.well_known]
enabled = true
auto_generate = ["host-meta", "host-meta.json", "webfinger", "nodeinfo", "time"]

# Optional entries requiring config
ssh_fingerprint = "SHA256:abcdef..."
keybase_username = "username"
```

**Notes:**
- If you set `auto_generate = []`, only optional entries with explicit config are generated.
- Templates live under `templates/well-known/` and can be overridden.

## How It Works

### Reversed Redirects

For `.txt` and `.md` formats, markata-go uses **reversed redirects**:

| Canonical Location | Redirect From |
|-------------------|---------------|
| `/robots.txt` | `/robots/index.txt`, `/robots/index.txt/index.html` |
| `/llms.txt` | `/llms/index.txt`, `/llms/index.txt/index.html` |

This ensures:
- Standard files work at their expected URLs
- Backwards compatibility with directory-based URLs
- Both URL styles are accessible

### File Structure

After building, your output directory contains:

```
output/
├── robots.txt                   ← Canonical content
├── robots/
│   └── index.txt/
│       └── index.html           ← HTML redirect to /robots.txt
├── llms.txt                     ← Canonical content
├── llms/
│   └── index.txt/
│       └── index.html           ← HTML redirect to /llms.txt
└── humans.txt                   ← Canonical content
    humans/
    └── index.txt/
        └── index.html           ← HTML redirect to /humans.txt
```

## Configuration

Text output is enabled by default. To customize:

```toml
[markata-go.post_formats]
html = true       # Standard HTML (default: true)
markdown = true   # Raw markdown at /slug.md (default: true)
text = true       # Plain text at /slug.txt (default: true)
og = true         # OpenGraph cards (default: true)
```

To disable text output:

```toml
[markata-go.post_formats]
text = false
```

## Text File Format

The plain text output includes:

1. **Title** (underlined with `=`)
2. **Description** (if present)
3. **Date** (if present)
4. **Raw content** (no markdown processing)

Example output for a post:

```
My Post Title
=============

A description of the post.

Date: January 26, 2026

The actual content of the post...
```

For standard txt files like `robots.txt`, keep the content minimal:

```markdown
---
title: "Robots"
slug: robots
published: true
---

User-agent: *
Allow: /
```

This generates clean output without the title/description header:

```
Robots
======

User-agent: *
Allow: /
```

## Testing

Verify your txt files work:

```bash
# Build and serve locally
markata-go serve

# Test in another terminal
curl http://localhost:8000/robots.txt
curl http://localhost:8000/llms.txt
curl http://localhost:8000/humans.txt

# Test redirects
curl -L http://localhost:8000/robots/index.txt
```

## See Also

- [Post Output Formats](/docs/guides/post-formats/) - Full guide to all output formats
- [Configuration Guide](/docs/guides/configuration/) - Complete configuration reference
- [SEO Guide](/docs/guides/structured-data/) - Search engine optimization
