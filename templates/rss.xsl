<?xml version="1.0" encoding="utf-8"?>
<!--
  RSS 2.0 XSL Stylesheet for markata-go
  Transforms RSS feeds into human-readable HTML while maintaining machine compatibility.
  Inspired by pretty-feed-v3.xsl (https://github.com/genmon/aboutfeeds/blob/main/tools/pretty-feed-v3.xsl)
-->
<xsl:stylesheet version="1.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:atom="http://www.w3.org/2005/Atom">

  <xsl:output method="html" version="1.0" encoding="UTF-8" indent="yes"/>

  <xsl:template match="/">
    <html lang="en">
      <head>
        <meta charset="UTF-8"/>
        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
        <title><xsl:value-of select="/rss/channel/title"/> - RSS Feed</title>
        <link rel="stylesheet" href="/css/variables.css"/>
        <link rel="stylesheet" href="/css/main.css"/>
        <style>
          /* Feed-specific styles */
          .feed-banner {
            background: var(--color-surface, #f9fafb);
            border: 1px solid var(--color-border, #e5e7eb);
            border-radius: var(--radius-lg, 0.5rem);
            padding: var(--space-6, 1.5rem);
            margin-bottom: var(--space-8, 2rem);
          }
          .feed-banner p {
            margin: 0 0 var(--space-3, 0.75rem) 0;
            line-height: var(--leading-relaxed, 1.75);
          }
          .feed-banner p:last-child {
            margin-bottom: 0;
          }
          .feed-banner a {
            color: var(--color-primary, #3b82f6);
            text-decoration: underline;
          }
          .feed-banner a:hover {
            color: var(--color-primary-dark, #2563eb);
          }
          .feed-header {
            margin-bottom: var(--space-8, 2rem);
          }
          .feed-header h1 {
            margin: 0 0 var(--space-2, 0.5rem) 0;
            font-size: var(--text-3xl, 1.875rem);
            font-weight: 700;
            color: var(--color-text, #1f2937);
          }
          .feed-header .description {
            color: var(--color-text-muted, #6b7280);
            font-size: var(--text-lg, 1.125rem);
            margin: 0;
          }
          .feed-meta {
            display: flex;
            gap: var(--space-4, 1rem);
            margin-top: var(--space-4, 1rem);
            font-size: var(--text-sm, 0.875rem);
            color: var(--color-text-muted, #6b7280);
          }
          .posts {
            display: flex;
            flex-direction: column;
            gap: var(--space-6, 1.5rem);
          }
          .card {
            background: var(--color-surface, #f9fafb);
            border: 1px solid var(--color-border, #e5e7eb);
            border-radius: var(--radius-lg, 0.5rem);
            padding: var(--space-6, 1.5rem);
            transition: border-color 0.2s, box-shadow 0.2s;
          }
          .card:hover {
            border-color: var(--color-primary, #3b82f6);
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
          }
          .card h2 {
            margin: 0 0 var(--space-2, 0.5rem) 0;
            font-size: var(--text-xl, 1.25rem);
            font-weight: 600;
          }
          .card h2 a {
            color: var(--color-text, #1f2937);
            text-decoration: none;
          }
          .card h2 a:hover {
            color: var(--color-primary, #3b82f6);
          }
          .card time {
            display: block;
            font-size: var(--text-sm, 0.875rem);
            color: var(--color-text-muted, #6b7280);
            margin-bottom: var(--space-2, 0.5rem);
          }
          .card p {
            margin: 0;
            color: var(--color-text-muted, #6b7280);
            line-height: var(--leading-normal, 1.5);
          }
          .container {
            max-width: var(--page-width, 1200px);
            margin: 0 auto;
            padding: var(--space-8, 2rem) var(--space-4, 1rem);
          }
          body {
            font-family: var(--font-body, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif);
            background: var(--color-background, #ffffff);
            color: var(--color-text, #1f2937);
            line-height: var(--leading-normal, 1.5);
            margin: 0;
          }
          /* Dark mode support */
          @media (prefers-color-scheme: dark) {
            body {
              --color-text: #f9fafb;
              --color-text-muted: #9ca3af;
              --color-background: #111827;
              --color-surface: #1f2937;
              --color-border: #374151;
            }
          }
        </style>
      </head>
      <body>
        <div class="container">
          <div class="feed-banner">
            <p>
              <strong>This is a web feed</strong>, also known as an RSS feed.
              <strong>Subscribe</strong> by copying the URL from the address bar into your newsreader.
            </p>
            <p>
              Visit <a href="https://aboutfeeds.com">About Feeds</a> to get started with newsreaders and subscribing. It's free.
            </p>
          </div>

          <header class="feed-header">
            <h1><xsl:value-of select="/rss/channel/title"/></h1>
            <xsl:if test="/rss/channel/description">
              <p class="description"><xsl:value-of select="/rss/channel/description"/></p>
            </xsl:if>
            <div class="feed-meta">
              <span><xsl:value-of select="count(/rss/channel/item)"/> posts</span>
              <xsl:if test="/rss/channel/lastBuildDate">
                <span>Updated: <xsl:value-of select="/rss/channel/lastBuildDate"/></span>
              </xsl:if>
            </div>
          </header>

          <div class="posts">
            <xsl:for-each select="/rss/channel/item">
              <article class="card">
                <h2>
                  <a>
                    <xsl:attribute name="href">
                      <xsl:value-of select="link"/>
                    </xsl:attribute>
                    <xsl:value-of select="title"/>
                  </a>
                </h2>
                <xsl:if test="pubDate">
                  <time>
                    <xsl:attribute name="datetime">
                      <xsl:value-of select="pubDate"/>
                    </xsl:attribute>
                    <xsl:value-of select="pubDate"/>
                  </time>
                </xsl:if>
                <xsl:if test="description">
                  <p><xsl:value-of select="description"/></p>
                </xsl:if>
              </article>
            </xsl:for-each>
          </div>
        </div>
      </body>
    </html>
  </xsl:template>

</xsl:stylesheet>
