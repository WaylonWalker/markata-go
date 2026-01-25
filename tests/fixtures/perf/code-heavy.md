---
title: "Code Heavy Post"
description: "A post with many code blocks for syntax highlighting benchmarks"
date: 2024-01-15
published: true
tags:
  - benchmark
  - code
---

# Code Heavy Post

This post focuses on code block rendering performance.

## Go

```go
package lifecycle

import (
	"fmt"
	"sync"
)

// Manager orchestrates the lifecycle stages and plugin execution.
type Manager struct {
	plugins     []Plugin
	config      *Config
	posts       []*Post
	mu          sync.RWMutex
	concurrency int
}

// NewManager creates a new lifecycle Manager.
func NewManager() *Manager {
	return &Manager{
		plugins:     make([]Plugin, 0),
		config:      NewConfig(),
		posts:       make([]*Post, 0),
		concurrency: 4,
	}
}

// Run executes all lifecycle stages in order.
func (m *Manager) Run() error {
	stages := []Stage{
		StageConfigure,
		StageValidate,
		StageGlob,
		StageLoad,
		StageTransform,
		StageRender,
		StageCollect,
		StageWrite,
		StageCleanup,
	}

	for _, stage := range stages {
		if err := m.runStage(stage); err != nil {
			return fmt.Errorf("stage %s failed: %w", stage, err)
		}
	}

	return nil
}
```

## Python

```python
from typing import List, Optional
from dataclasses import dataclass
import asyncio

@dataclass
class Post:
    path: str
    title: str
    content: str
    published: bool = False
    tags: List[str] = None

    def __post_init__(self):
        if self.tags is None:
            self.tags = []

class SiteBuilder:
    def __init__(self, config: dict):
        self.config = config
        self.posts: List[Post] = []

    async def build(self) -> None:
        """Build the entire site."""
        await self.load_posts()
        await self.render_posts()
        await self.write_output()

    async def load_posts(self) -> None:
        """Load all markdown posts."""
        # Implementation here
        pass

    async def render_posts(self) -> None:
        """Render markdown to HTML."""
        tasks = [self.render_post(p) for p in self.posts]
        await asyncio.gather(*tasks)

    async def render_post(self, post: Post) -> None:
        """Render a single post."""
        pass

    async def write_output(self) -> None:
        """Write rendered HTML to output directory."""
        pass

if __name__ == "__main__":
    builder = SiteBuilder({"output_dir": "public"})
    asyncio.run(builder.build())
```

## JavaScript/TypeScript

```typescript
interface Post {
  path: string;
  title: string;
  content: string;
  published: boolean;
  tags: string[];
  date?: Date;
}

interface Config {
  outputDir: string;
  templatesDir: string;
  contentDir: string;
}

class SiteBuilder {
  private config: Config;
  private posts: Post[] = [];

  constructor(config: Config) {
    this.config = config;
  }

  async build(): Promise<void> {
    console.log("Building site...");
    await this.loadPosts();
    await this.renderPosts();
    await this.writeOutput();
    console.log("Build complete!");
  }

  private async loadPosts(): Promise<void> {
    // Load all markdown files
    const files = await this.glob("**/*.md");
    for (const file of files) {
      const post = await this.parsePost(file);
      this.posts.push(post);
    }
  }

  private async renderPosts(): Promise<void> {
    const promises = this.posts.map(post => this.renderPost(post));
    await Promise.all(promises);
  }

  private async renderPost(post: Post): Promise<string> {
    // Render markdown to HTML
    return "";
  }

  private async writeOutput(): Promise<void> {
    // Write HTML files
  }

  private async glob(pattern: string): Promise<string[]> {
    return [];
  }

  private async parsePost(path: string): Promise<Post> {
    return {
      path,
      title: "",
      content: "",
      published: false,
      tags: [],
    };
  }
}

// Usage
const builder = new SiteBuilder({
  outputDir: "public",
  templatesDir: "templates",
  contentDir: "content",
});

builder.build().catch(console.error);
```

## Rust

```rust
use std::collections::HashMap;
use std::path::PathBuf;
use tokio::fs;

#[derive(Debug, Clone)]
pub struct Post {
    pub path: PathBuf,
    pub title: String,
    pub content: String,
    pub published: bool,
    pub tags: Vec<String>,
}

pub struct SiteBuilder {
    config: Config,
    posts: Vec<Post>,
}

#[derive(Debug)]
pub struct Config {
    pub output_dir: PathBuf,
    pub content_dir: PathBuf,
    pub templates_dir: PathBuf,
}

impl SiteBuilder {
    pub fn new(config: Config) -> Self {
        Self {
            config,
            posts: Vec::new(),
        }
    }

    pub async fn build(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        self.load_posts().await?;
        self.render_posts().await?;
        self.write_output().await?;
        Ok(())
    }

    async fn load_posts(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        // Load markdown files
        Ok(())
    }

    async fn render_posts(&self) -> Result<(), Box<dyn std::error::Error>> {
        // Render to HTML
        Ok(())
    }

    async fn write_output(&self) -> Result<(), Box<dyn std::error::Error>> {
        // Write files
        Ok(())
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let config = Config {
        output_dir: PathBuf::from("public"),
        content_dir: PathBuf::from("content"),
        templates_dir: PathBuf::from("templates"),
    };

    let mut builder = SiteBuilder::new(config);
    builder.build().await?;

    Ok(())
}
```

## SQL

```sql
-- Create tables for a blog database
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(255) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL
);

CREATE TABLE post_tags (
    post_id INTEGER REFERENCES posts(id) ON DELETE CASCADE,
    tag_id INTEGER REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, tag_id)
);

-- Query published posts with tags
SELECT
    p.id,
    p.title,
    p.slug,
    p.created_at,
    array_agg(t.name) as tags
FROM posts p
LEFT JOIN post_tags pt ON p.id = pt.post_id
LEFT JOIN tags t ON pt.tag_id = t.id
WHERE p.published = TRUE
GROUP BY p.id, p.title, p.slug, p.created_at
ORDER BY p.created_at DESC;
```

## YAML

```yaml
markata-go:
  output_dir: public
  url: https://example.com
  title: My Site

  glob:
    patterns:
      - "posts/**/*.md"
      - "pages/*.md"
    use_gitignore: true

  feeds:
    - slug: blog
      title: Blog
      filter: "published == True"
      sort: date
      reverse: true
      formats:
        html: true
        rss: true
        atom: true
```

## Shell

```bash
#!/bin/bash

# Build script for markata-go site

set -euo pipefail

echo "Building site..."

# Clean output directory
rm -rf public/

# Run the build
markata-go build

# Generate sitemap
markata-go sitemap

# Optimize images (if available)
if command -v optipng &> /dev/null; then
    find public -name "*.png" -exec optipng -o5 {} \;
fi

echo "Build complete!"
echo "Output in: public/"
```

This post helps benchmark syntax highlighting performance across multiple languages.
