---
title: "Quickstart"
description: "Get a markata-go site running in under 5 minutes"
date: 2024-01-15
published: true
tags:
  - documentation
  - getting-started
---

# Quickstart

Get a site running in under 5 minutes.

## Install

Install the latest release on Linux or macOS:

```bash
curl -sSL https://raw.githubusercontent.com/WaylonWalker/markata-go/main/install.sh | bash
```

Other supported installation methods are covered in the [[installation|Installation Guide]].

Verify the binary is available:

```bash
markata-go version
```

## Create Your Site

```bash
mkdir my-site
cd my-site
markata-go config init
markata-go new "Hello World"
```

This creates:
- `markata-go.toml` - Site configuration
- `pages/post/hello-world.md` - Your first post

## Preview

```bash
markata-go serve
```

Open [http://localhost:8000](http://localhost:8000) in your browser.

## Build

```bash
markata-go build
```

Output is written to `./output/` by default.

## Next Steps

- [[getting-started|Getting Started]] - Full tutorial
- [[configuration-guide|Configuration]] - Customize your site
- [[frontmatter-guide|Frontmatter]] - Content metadata
- [[feeds-guide|Feeds]] - Create archives and RSS
- [[templates-guide|Templates]] - Customize appearance
- [[deployment-guide|Deployment]] - Go live
