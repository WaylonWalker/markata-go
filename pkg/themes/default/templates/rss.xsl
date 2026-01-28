<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:atom="http://www.w3.org/2005/Atom"
    xmlns:dc="http://purl.org/dc/elements/1.1/"
    xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
    <xsl:output method="html" version="1.0" encoding="UTF-8" indent="yes" />

    <xsl:template match="/">
        <html lang="en">
        <head>
            <meta charset="UTF-8" />
            <meta name="viewport" content="width=device-width, initial-scale=1.0" />
            <title><xsl:value-of select="/rss/channel/title" /> - RSS Feed</title>

            <!-- Theme CSS -->
            <link rel="stylesheet" href="/css/variables.css" />
            <link rel="stylesheet" href="/css/main.css" />

            <!-- Feed-specific styles -->
            <style>
                .feed-container {
                    max-width: 65ch;
                    margin: 0 auto;
                    padding: 2rem 1rem;
                }

                .feed-header {
                    margin-bottom: 2rem;
                    padding-bottom: 1rem;
                    border-bottom: 1px solid var(--color-border, #e5e7eb);
                }

                .feed-title {
                    font-size: 2rem;
                    margin: 0 0 0.5rem 0;
                    color: var(--color-text, #1f2937);
                }

                .feed-description {
                    color: var(--color-text-muted, #6b7280);
                    margin: 0;
                }

                .feed-notice {
                    background: var(--color-surface, #f9fafb);
                    border: 1px solid var(--color-border, #e5e7eb);
                    border-radius: 0.5rem;
                    padding: 1rem;
                    margin-bottom: 2rem;
                }

                .feed-notice-title {
                    font-weight: 600;
                    margin-bottom: 0.5rem;
                    display: flex;
                    align-items: center;
                    gap: 0.5rem;
                }

                .feed-notice p {
                    margin: 0.5rem 0;
                    color: var(--color-text-muted, #6b7280);
                    font-size: 0.875rem;
                }

                .feed-url {
                    font-family: ui-monospace, monospace;
                    background: var(--color-background, #fff);
                    padding: 0.5rem;
                    border-radius: 0.25rem;
                    font-size: 0.875rem;
                    word-break: break-all;
                    border: 1px solid var(--color-border, #e5e7eb);
                }

                .feed-items {
                    list-style: none;
                    padding: 0;
                    margin: 0;
                }

                .feed-item {
                    padding: 1.5rem 0;
                    border-bottom: 1px solid var(--color-border, #e5e7eb);
                }

                .feed-item:last-child {
                    border-bottom: none;
                }

                .feed-item-title {
                    font-size: 1.25rem;
                    margin: 0 0 0.5rem 0;
                }

                .feed-item-title a {
                    color: var(--color-primary, #3b82f6);
                    text-decoration: none;
                }

                .feed-item-title a:hover {
                    text-decoration: underline;
                }

                .feed-item-description {
                    color: var(--color-text-muted, #6b7280);
                    margin: 0 0 0.5rem 0;
                    line-height: 1.6;
                }

                .feed-item-meta {
                    font-size: 0.875rem;
                    color: var(--color-text-muted, #6b7280);
                }

                details {
                    margin-top: 0.5rem;
                }

                summary {
                    cursor: pointer;
                    color: var(--color-primary, #3b82f6);
                    font-size: 0.875rem;
                }

                details ul {
                    margin: 0.5rem 0;
                    padding-left: 1.5rem;
                }

                details li {
                    margin: 0.25rem 0;
                }

                details a {
                    color: var(--color-primary, #3b82f6);
                }

                @media (prefers-color-scheme: dark) {
                    .feed-title {
                        color: var(--color-text, #f9fafb);
                    }
                    .feed-notice {
                        background: var(--color-surface, #1f2937);
                        border-color: var(--color-border, #374151);
                    }
                    .feed-url {
                        background: var(--color-background, #111827);
                        border-color: var(--color-border, #374151);
                    }
                    .feed-item {
                        border-color: var(--color-border, #374151);
                    }
                }
            </style>
        </head>

        <body>
            <div class="feed-container">
                <header class="feed-header">
                    <h1 class="feed-title">
                        <xsl:value-of select="/rss/channel/title" />
                    </h1>
                    <p class="feed-description">
                        <xsl:value-of select="/rss/channel/description" />
                    </p>
                </header>

                <div class="feed-notice">
                    <div class="feed-notice-title">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M4 11a9 9 0 0 1 9 9" />
                            <path d="M4 4a16 16 0 0 1 16 16" />
                            <circle cx="5" cy="19" r="1" />
                        </svg>
                        This is an RSS Feed
                    </div>
                    <p>Subscribe by copying this URL into your feed reader:</p>
                    <div class="feed-url">
                        <xsl:value-of select="/rss/channel/link" />rss.xml
                    </div>
                    <details>
                        <summary>What is RSS? Popular feed readers</summary>
                        <p>RSS lets you subscribe to websites and get updates in one place. Here are some popular readers:</p>
                        <ul>
                            <li><a href="https://feedly.com" target="_blank">Feedly</a> (Web, iOS, Android)</li>
                            <li><a href="https://netnewswire.com" target="_blank">NetNewsWire</a> (Mac, iOS - Free)</li>
                            <li><a href="https://newsblur.com" target="_blank">NewsBlur</a> (Web, iOS, Android)</li>
                            <li><a href="https://www.inoreader.com" target="_blank">Inoreader</a> (Web, iOS, Android)</li>
                        </ul>
                    </details>
                </div>

                <h2>Recent Posts</h2>
                <ul class="feed-items">
                    <xsl:for-each select="/rss/channel/item">
                        <li class="feed-item">
                            <h3 class="feed-item-title">
                                <a target="_blank">
                                    <xsl:attribute name="href">
                                        <xsl:value-of select="link" />
                                    </xsl:attribute>
                                    <xsl:value-of select="title" />
                                </a>
                            </h3>
                            <p class="feed-item-description">
                                <xsl:value-of select="description" />
                            </p>
                            <div class="feed-item-meta">
                                Published: <xsl:value-of select="pubDate" />
                            </div>
                        </li>
                    </xsl:for-each>
                </ul>
            </div>
        </body>
        </html>
    </xsl:template>
</xsl:stylesheet>
