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

- [Getting Started](./getting-started.md) - Full tutorial
- [Configuration](./guides/configuration.md) - Customize your site
- [Frontmatter](./guides/frontmatter.md) - Content metadata
- [Feeds](./guides/feeds.md) - Create archives and RSS
- [Templates](./guides/templates.md) - Customize appearance
- [Deployment](./guides/deployment.md) - Go live
