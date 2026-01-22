---
title: "Quickstart"
description: "Get a markata-go site running in under 5 minutes"
date: 2024-01-15
published: true
template: doc.html
tags:
  - documentation
  - getting-started
---

# Quickstart

Get a site running in under 5 minutes.

## Prerequisites

- **Go 1.22+** - [Install Go](https://go.dev/doc/install)

Verify your installation:

```bash
go version
```

## Install

```bash
go install github.com/example/markata-go/cmd/markata-go@latest
```

## Create Your Site

```bash
mkdir my-site
cd my-site
markata-go config init
markata-go new "Hello World"
```

This creates:
- `markata.toml` - Site configuration
- `pages/hello-world.md` - Your first post

## Preview

```bash
markata-go serve
```

Open [http://localhost:8000](http://localhost:8000) in your browser.

## Build

```bash
markata-go build
```

Output is written to `./output/`.

## Next Steps

- [[getting-started|Getting Started]] - Full tutorial
- [[configuration-guide|Configuration]] - Customize your site
- [[frontmatter-guide|Frontmatter]] - Content metadata
- [[feeds-guide|Feeds]] - Create archives and RSS
- [[templates-guide|Templates]] - Customize appearance
- [[deployment-guide|Deployment]] - Go live
