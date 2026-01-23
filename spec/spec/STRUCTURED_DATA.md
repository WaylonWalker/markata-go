# Structured Data Specification

This document specifies the structured data system for markata-go, providing JSON-LD Schema.org markup, OpenGraph meta tags, and Twitter Cards for SEO and social media optimization.

## Overview

The structured data system automatically generates:
- **JSON-LD** - Schema.org structured data for search engine rich results
- **OpenGraph** - Meta tags for Facebook, LinkedIn, and general social sharing
- **Twitter Cards** - Meta tags for Twitter/X sharing previews

## Configuration

### Site-Level Configuration

Configure structured data in `markata-go.toml`:

```toml
[markata-go.seo]
# Twitter/X handle for twitter:site (without @)
twitter_handle = "username"

# Default OG image for posts without images
default_image = "/images/og-default.jpg"

# Site logo URL for Schema.org Organization
logo_url = "/images/logo.png"

[markata-go.seo.structured_data]
# Enable/disable structured data generation (default: true)
enabled = true

# Schema.org Organization/Person for publisher
[markata-go.seo.structured_data.publisher]
type = "Organization"  # or "Person"
name = "Site Name"
url = "https://example.com"
logo = "/images/logo.png"

# Default author for posts without explicit author
[markata-go.seo.structured_data.default_author]
type = "Person"
name = "Author Name"
url = "https://example.com/about"
```

### Frontmatter Fields

Posts can override structured data via frontmatter:

```yaml
---
title: "Post Title"
description: "Post description for meta tags"
date: 2024-01-15
modified: 2024-01-16  # For dateModified
author: "Author Name"  # Override default author
image: "/images/post.jpg"  # OG/Twitter image
tags: ["tag1", "tag2"]

# Optional structured data overrides
schema_type: "BlogPosting"  # Default: auto-detected
social_image: "/images/og-custom.jpg"  # Override OG image specifically
---
```

## Data Models

### StructuredDataConfig

```go
type StructuredDataConfig struct {
    // Enabled controls generation (default: true)
    Enabled *bool `toml:"enabled"`
    
    // Publisher is the site publisher info
    Publisher *EntityConfig `toml:"publisher"`
    
    // DefaultAuthor is used when posts don't specify author
    DefaultAuthor *EntityConfig `toml:"default_author"`
}

type EntityConfig struct {
    // Type is "Person" or "Organization"
    Type string `toml:"type"`
    
    // Name is the entity name
    Name string `toml:"name"`
    
    // URL is the entity's web page
    URL string `toml:"url"`
    
    // Logo is the logo URL (Organizations only)
    Logo string `toml:"logo"`
}
```

### Generated Structured Data

The plugin stores generated data in `post.Extra`:

```go
post.Extra["structured_data"] = &StructuredData{
    JSONLD:      jsonLDString,      // JSON-LD script content
    OpenGraph:   []MetaTag{...},    // OG meta tags
    TwitterCard: []MetaTag{...},    // Twitter meta tags
}
```

## JSON-LD Schema

### BlogPosting Schema

Generated for all blog posts:

```json
{
  "@context": "https://schema.org",
  "@type": "BlogPosting",
  "headline": "Post Title",
  "description": "Post description",
  "datePublished": "2024-01-15T00:00:00Z",
  "dateModified": "2024-01-16T00:00:00Z",
  "author": {
    "@type": "Person",
    "name": "Author Name",
    "url": "https://example.com/about"
  },
  "publisher": {
    "@type": "Organization",
    "name": "Site Name",
    "logo": {
      "@type": "ImageObject",
      "url": "https://example.com/logo.png"
    }
  },
  "mainEntityOfPage": {
    "@type": "WebPage",
    "@id": "https://example.com/post-slug/"
  },
  "image": "https://example.com/images/post.jpg",
  "keywords": ["tag1", "tag2"]
}
```

### WebSite Schema

Generated for the homepage:

```json
{
  "@context": "https://schema.org",
  "@type": "WebSite",
  "name": "Site Title",
  "description": "Site description",
  "url": "https://example.com",
  "publisher": {
    "@type": "Organization",
    "name": "Site Name"
  }
}
```

## OpenGraph Tags

Generated meta tags:

| Tag | Source |
|-----|--------|
| `og:title` | post.Title or config.Title |
| `og:description` | post.Description or config.Description |
| `og:type` | "article" for posts, "website" for pages |
| `og:url` | Canonical URL |
| `og:site_name` | config.Title |
| `og:image` | post.Extra["image"] or seo.default_image |
| `og:image:width` | 1200 (if image set) |
| `og:image:height` | 630 (if image set) |
| `og:locale` | "en_US" (or config.lang) |
| `article:published_time` | post.Date (articles only) |
| `article:modified_time` | post.Extra["modified"] (if set) |
| `article:author` | Author URL |
| `article:tag` | Each tag separately |

## Twitter Card Tags

Generated meta tags:

| Tag | Source |
|-----|--------|
| `twitter:card` | "summary_large_image" or "summary" |
| `twitter:site` | @{seo.twitter_handle} |
| `twitter:creator` | @{author handle} or @{seo.twitter_handle} |
| `twitter:title` | post.Title |
| `twitter:description` | post.Description (truncated to 200 chars) |
| `twitter:image` | Same as og:image |

## Plugin Lifecycle

The StructuredDataPlugin runs in the **Transform** stage:

1. Runs after frontmatter parsing (needs title, description, date)
2. Runs before template rendering (adds data to post.Extra)

### Priority

Priority: 500 (middle of transform stage, after description plugin)

## Template Integration

Templates access structured data via `post` context:

```html
{# JSON-LD #}
{% if post.structured_data.jsonld %}
<script type="application/ld+json">
{{ post.structured_data.jsonld | safe }}
</script>
{% endif %}

{# OpenGraph #}
{% for meta in post.structured_data.opengraph %}
<meta property="{{ meta.property }}" content="{{ meta.content }}">
{% endfor %}

{# Twitter Cards #}
{% for meta in post.structured_data.twitter %}
<meta name="{{ meta.name }}" content="{{ meta.content }}">
{% endfor %}
```

### Default Template Updates

The default `base.html` template will be updated to automatically include structured data when available.

## URL Handling

All URLs in structured data are made absolute:

- Relative URLs (e.g., `/images/post.jpg`) are prefixed with `config.URL`
- Protocol-relative URLs are prefixed with `https:`
- Absolute URLs are used as-is

## Intelligent Defaults

The plugin provides sensible defaults:

| Field | Default |
|-------|---------|
| `og:type` | "article" for posts with date, "website" otherwise |
| `schema_type` | "BlogPosting" for posts with date, "WebPage" otherwise |
| `og:image` | seo.default_image if set |
| `author` | seo.structured_data.default_author |
| `dateModified` | Same as datePublished if not set |

## Error Handling

- Missing required fields (title) - Skip structured data for that post, log warning
- Invalid URLs - Skip the invalid URL field, continue with others
- Missing config - Use sensible defaults, structured data still generated

## Testing Requirements

1. Unit tests for JSON-LD generation
2. Unit tests for OpenGraph tag generation
3. Unit tests for Twitter Card generation
4. Integration test with full build pipeline
5. Validation against Schema.org validator output format

## Future Enhancements

Potential future additions (not in v1):

- BreadcrumbList schema
- FAQPage schema
- HowTo schema
- Event schema
- Product schema
- Review schema
- Custom schema injection via frontmatter
