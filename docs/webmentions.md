---
title: "WebMentions & Social Bridging"
description: "Connect your static site to the social web with WebMentions and cross-platform bridging"
date: 2026-01-24
published: true
tags:
  - webmentions
  - indieweb
  - social
  - bridging
---

# WebMentions & Social Bridging

markata-go supports the IndieWeb WebMention protocol for connecting your static site to the decentralized social web. This includes:

- **Outgoing WebMentions**: Automatically notify sites you link to
- **Social Media Bridging**: Receive likes, reposts, and replies from Twitter, Bluesky, Mastodon, and more

## Quick Start

Enable WebMentions in your `markata-go.toml`:

```toml
[markata-go.webmentions]
enabled = true

# For outgoing mentions
outgoing = true

# For social media bridging
[markata-go.webmentions.bridges]
enabled = true
bluesky = true
twitter = true
mastodon = true
```

## Outgoing WebMentions

When you link to external sites in your posts, markata-go can automatically send WebMentions to those sites (if they support the protocol).

### How It Works

1. During build, markata-go scans your posts for external links
2. For each link, it discovers the target site's WebMention endpoint
3. It sends a notification: "Hey, this URL linked to you!"
4. The target site can then display your link/comment

### Configuration

```toml
[markata-go.webmentions]
enabled = true
outgoing = true

# HTTP request timeout
timeout = "30s"

# Max concurrent requests
concurrent_requests = 5

# Cache directory to avoid re-sending
cache_dir = ".cache/webmentions"

# Custom User-Agent string
user_agent = "MySite/1.0 (WebMention; +https://mysite.com)"
```

### Caching

Sent WebMentions are cached to avoid re-sending on every build. The cache is stored in the configured `cache_dir` as JSON.

## Social Media Bridging

Social media bridging allows you to receive interactions from Twitter, Bluesky, Mastodon, and other platforms as WebMentions. This is powered by services like [Bridgy Fed](https://fed.brid.gy/).

### Supported Platforms

| Platform | Interactions | Detection |
|----------|-------------|-----------|
| Bluesky | Likes, reposts, replies | bsky.app URLs, Bridgy |
| Twitter/X | Likes, retweets, replies | twitter.com/x.com URLs |
| Mastodon | Favorites, boosts, replies | Fediverse patterns |
| GitHub | Stars, comments, issues | github.com URLs |
| Flickr | Favorites, comments | flickr.com URLs |

### Setting Up Bridging

1. **Sign up for webmention.io** to receive incoming mentions
2. **Connect Bridgy Fed** to your social accounts at [fed.brid.gy](https://fed.brid.gy/)
3. **Configure markata-go**:

```toml
[markata-go.webmentions]
enabled = true
webmention_io_token = "your_token_here"

[markata-go.webmentions.bridges]
enabled = true
bridgy_fediverse = true

# Enable specific platforms
bluesky = true
twitter = true
mastodon = true
github = true
flickr = false
```

### Platform Detection

markata-go automatically detects the source platform of incoming WebMentions by analyzing:

1. **URL patterns**: `brid.gy/publish/bluesky`, `bsky.app`, etc.
2. **Domain matching**: Known Mastodon instances, GitHub, etc.
3. **Content patterns**: Platform-specific indicators in mention content

Each detected mention is enriched with:
- **Platform name**: "bluesky", "twitter", "mastodon", etc.
- **Handle**: Platform-specific username (e.g., `@alice.bsky.social`)
- **Original URL**: Link back to the original interaction

### Filtering Bridged Mentions

You can filter incoming mentions by platform, interaction type, or content:

```toml
[markata-go.webmentions.bridges.filters]
# Only accept from these platforms
platforms = ["bluesky", "mastodon"]

# Only accept these interaction types
# Options: "like", "repost", "reply", "bookmark", "mention"
interaction_types = ["like", "repost", "reply"]

# Minimum content length for replies
min_content_length = 10

# Block specific domains
blocked_domains = ["spam-site.com", "bad-actor.net"]
```

## Template Integration

Display WebMentions in your templates:

```html
{% if post.webmentions %}
<section class="webmentions">
  <h3>Reactions</h3>

  {% for mention in post.webmentions %}
  <article class="webmention webmention--{{ mention.platform }}">
    <!-- Platform badge -->
    {% if mention.platform != "web" %}
    <span class="platform-badge">
      {% if mention.platform == "bluesky" %}Bluesky
      {% elif mention.platform == "twitter" %}Twitter
      {% elif mention.platform == "mastodon" %}Mastodon
      {% elif mention.platform == "github" %}GitHub
      {% endif %}
    </span>
    {% endif %}

    <!-- Author info -->
    <div class="author">
      {% if mention.author.photo %}
      <img src="{{ mention.author.photo }}" alt="{{ mention.author.name }}">
      {% endif %}
      <a href="{{ mention.author.url }}">
        {{ mention.handle | default(mention.author.name) }}
      </a>
    </div>

    <!-- Interaction type -->
    {% if mention.wm_property == "like-of" %}
      <p>Liked this post</p>
    {% elif mention.wm_property == "repost-of" %}
      <p>Reposted this post</p>
    {% elif mention.wm_property == "in-reply-to" %}
      <blockquote>{{ mention.content.text }}</blockquote>
    {% endif %}

    <!-- Link to original -->
    {% if mention.original_url %}
    <a href="{{ mention.original_url }}">View on {{ mention.platform }}</a>
    {% endif %}
  </article>
  {% endfor %}
</section>
{% endif %}
```

### Platform-Specific Styling

```css
/* Platform colors */
.platform-badge--bluesky { background: #0085ff; }
.platform-badge--twitter { background: #1da1f2; }
.platform-badge--mastodon { background: #6364ff; }
.platform-badge--github { background: #333; }

/* Platform borders */
.webmention--bluesky { border-left: 3px solid #0085ff; }
.webmention--twitter { border-left: 3px solid #1da1f2; }
.webmention--mastodon { border-left: 3px solid #6364ff; }
.webmention--github { border-left: 3px solid #333; }
```

## Full Configuration Reference

```toml
[markata-go.webmentions]
# Enable the webmentions plugin
enabled = true

# Send outgoing webmentions when linking to external sites
outgoing = true

# HTTP request timeout for sending/receiving
timeout = "30s"

# Directory to cache sent webmentions (avoids re-sending)
cache_dir = ".cache/webmentions"

# Maximum concurrent HTTP requests
concurrent_requests = 5

# User-Agent string for HTTP requests
user_agent = "markata-go/1.0 (WebMention; +https://github.com/WaylonWalker/markata-go)"

# API token for webmention.io (for receiving mentions)
webmention_io_token = ""

[markata-go.webmentions.bridges]
# Enable social media bridging detection
enabled = false

# Use Bridgy Fediverse for bridging
bridgy_fediverse = true

# Platform toggles
bluesky = true
twitter = true
mastodon = true
github = true
flickr = false

[markata-go.webmentions.bridges.filters]
# Limit to specific platforms (empty = all enabled)
platforms = []

# Limit to specific interaction types (empty = all)
# Options: "like", "repost", "reply", "bookmark", "mention"
interaction_types = []

# Minimum content length for replies
min_content_length = 0

# Block mentions from these domains
blocked_domains = []
```

## Troubleshooting

### WebMentions not being sent

1. Ensure `enabled = true` and `outgoing = true`
2. Check that target sites have WebMention endpoints
3. Review the cache directory for sent status
4. Check HTTP timeout settings

### Bridged mentions not appearing

1. Verify your webmention.io token is correct
2. Check that Bridgy is connected to your social accounts
3. Ensure the platform is enabled in `bridges` config
4. Check filters aren't blocking the mentions

### Platform not detected correctly

The platform detection uses URL patterns. If a mention isn't being detected:

1. Check the source URL in the raw WebMention data
2. Ensure the platform is enabled in config
3. File an issue if detection should be improved

## Resources

- [WebMention Spec](https://www.w3.org/TR/webmention/)
- [Bridgy Fed](https://fed.brid.gy/) - Social media bridging
- [webmention.io](https://webmention.io/) - Hosted WebMention receiver
- [IndieWeb Wiki](https://indieweb.org/Webmention)
