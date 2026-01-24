---
title: "Deployment Guide"
description: "Guide to deploying markata-go sites to GitHub Pages, Netlify, Vercel, and self-hosted solutions"
date: 2024-01-15
published: true
tags:
  - documentation
  - deployment
  - hosting
---

# Deployment and Hosting Guide

This guide covers deploying markata-go sites to various hosting platforms, from managed services like GitHub Pages and Netlify to self-hosted solutions using nginx or Docker.

> **Prerequisites:** Before deploying, ensure you have:
> - A working markata-go site that builds successfully (`markata-go build`)
> - [Configuration](/docs/guides/configuration/) set up with your site URL
> - [Feeds](/docs/guides/feeds/) configured for RSS/Atom if you want syndication

## Building for Production

### Build Command

Build your site for production using the `build` command:

```bash
# Standard production build
markata-go build

# Clean build (removes output directory first)
markata-go build --clean

# Build with verbose output
markata-go build -v

# Build to a custom output directory
markata-go build -o dist

# Dry run (show what would be built without writing files)
markata-go build --dry-run
```

### Output Directory Structure

After building, your output directory (default: `public/`) contains the complete static site:

```
public/
├── index.html              # Home page (from empty-slug feed)
├── blog/
│   ├── index.html          # Blog feed page 1
│   ├── page/2/index.html   # Blog feed page 2
│   ├── rss.xml             # RSS feed
│   └── atom.xml            # Atom feed
├── tags/
│   ├── go/
│   │   ├── index.html      # Tag archive
│   │   └── rss.xml         # Tag RSS feed
│   └── tutorial/
│       └── index.html
├── my-first-post/
│   └── index.html          # Individual post
├── about/
│   └── index.html          # Page
├── static/                 # Copied assets
│   ├── css/
│   ├── js/
│   └── images/
└── sitemap.xml
```

### Clean Builds

For production deployments, always use clean builds to ensure no stale files remain:

```bash
markata-go build --clean
```

Or manually remove the output directory before building:

```bash
rm -rf public && markata-go build
```

### Environment-Specific Builds

Use environment variables to customize builds for different environments:

```bash
# Production
MARKATA_GO_URL=https://example.com markata-go build

# Staging
MARKATA_GO_URL=https://staging.example.com markata-go build

# Preview/development
MARKATA_GO_URL=https://preview-123.example.com markata-go build
```

---

## Deploying to GitHub Pages

GitHub Pages is a free hosting service for static sites directly from a GitHub repository.

### GitHub Actions Workflow (Recommended)

Create `.github/workflows/deploy.yml`:

```yaml
name: Deploy to GitHub Pages

on:
  push:
    branches:
      - main
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: false

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true

      - name: Install markata-go
        run: go install github.com/example/markata-go/cmd/markata-go@latest

      - name: Build site
        run: markata-go build --clean
        env:
          MARKATA_GO_URL: ${{ vars.SITE_URL || 'https://username.github.io/repo-name' }}

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: ./public

  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    needs: build
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

### Repository Settings

1. Go to your repository's **Settings** > **Pages**
2. Under **Source**, select **GitHub Actions**
3. The workflow will deploy on every push to `main`

### Custom Domains

To use a custom domain with GitHub Pages:

1. Add a `CNAME` file to your `static/` directory:

```
example.com
```

2. Configure DNS at your domain registrar:
   - For apex domain: Add `A` records pointing to GitHub's IPs
   - For subdomain: Add a `CNAME` record pointing to `username.github.io`

3. Update your config:

```toml
[markata-go]
url = "https://example.com"
```

### Manual Deployment

For manual deployment without GitHub Actions:

```bash
# Build the site
markata-go build --clean

# Deploy using gh-pages branch
cd public
git init
git add -A
git commit -m "Deploy"
git push -f git@github.com:username/repo.git main:gh-pages
```

Or use the `gh-pages` npm package:

```bash
npm install -g gh-pages
markata-go build --clean
gh-pages -d public
```

---

## Deploying to Netlify

Netlify provides continuous deployment, serverless functions, and edge features.

### netlify.toml Configuration

Create `netlify.toml` in your repository root:

```toml
[build]
  command = "go install github.com/example/markata-go/cmd/markata-go@latest && markata-go build --clean"
  publish = "public"

[build.environment]
  GO_VERSION = "1.22"
  MARKATA_GO_URL = "https://example.netlify.app"

# Production context
[context.production]
  environment = { MARKATA_GO_URL = "https://example.com" }

# Deploy previews
[context.deploy-preview]
  command = "go install github.com/example/markata-go/cmd/markata-go@latest && markata-go build"

[context.deploy-preview.environment]
  # URL is set automatically by Netlify

# Branch deploys
[context.branch-deploy]
  command = "go install github.com/example/markata-go/cmd/markata-go@latest && markata-go build"

# Headers
[[headers]]
  for = "/*"
  [headers.values]
    X-Frame-Options = "DENY"
    X-Content-Type-Options = "nosniff"
    Referrer-Policy = "strict-origin-when-cross-origin"

[[headers]]
  for = "/static/*"
  [headers.values]
    Cache-Control = "public, max-age=31536000, immutable"

[[headers]]
  for = "/*.html"
  [headers.values]
    Cache-Control = "public, max-age=0, must-revalidate"

# Redirects
[[redirects]]
  from = "/old-path/*"
  to = "/new-path/:splat"
  status = 301

[[redirects]]
  from = "/api/*"
  to = "https://api.example.com/:splat"
  status = 200
  force = true

# SPA fallback (if needed)
# [[redirects]]
#   from = "/*"
#   to = "/index.html"
#   status = 200
```

### Build Settings via UI

Alternatively, configure through the Netlify dashboard:

1. **Base directory**: (leave empty or set if site is in subdirectory)
2. **Build command**: `go install github.com/example/markata-go/cmd/markata-go@latest && markata-go build --clean`
3. **Publish directory**: `public`
4. **Environment variables**:
   - `GO_VERSION`: `1.22`
   - `MARKATA_GO_URL`: `https://your-site.netlify.app`

### Deploy via CLI

Install the Netlify CLI and deploy manually:

```bash
# Install CLI
npm install -g netlify-cli

# Login
netlify login

# Initialize (first time)
netlify init

# Deploy preview
markata-go build --clean
netlify deploy --dir=public

# Deploy to production
netlify deploy --dir=public --prod
```

---

## Deploying to Vercel

Vercel offers zero-configuration deployments with excellent performance.

### vercel.json Configuration

Create `vercel.json` in your repository root:

```json
{
  "buildCommand": "go install github.com/example/markata-go/cmd/markata-go@latest && markata-go build --clean",
  "outputDirectory": "public",
  "installCommand": "echo 'No npm dependencies'",
  "framework": null,
  "headers": [
    {
      "source": "/static/(.*)",
      "headers": [
        {
          "key": "Cache-Control",
          "value": "public, max-age=31536000, immutable"
        }
      ]
    },
    {
      "source": "/(.*).html",
      "headers": [
        {
          "key": "Cache-Control",
          "value": "public, max-age=0, must-revalidate"
        }
      ]
    }
  ],
  "redirects": [
    {
      "source": "/old-path/:path*",
      "destination": "/new-path/:path*",
      "permanent": true
    }
  ],
  "rewrites": [
    {
      "source": "/api/:path*",
      "destination": "https://api.example.com/:path*"
    }
  ]
}
```

### Environment Variables

Set environment variables in the Vercel dashboard or via CLI:

```bash
# Via CLI
vercel env add MARKATA_GO_URL production
# Enter: https://example.com

vercel env add MARKATA_GO_URL preview
# Enter: (leave blank to use Vercel's preview URL)
```

### Deploy via CLI

```bash
# Install CLI
npm install -g vercel

# Login
vercel login

# Deploy preview
markata-go build --clean
vercel --cwd public

# Deploy to production
vercel --cwd public --prod
```

### Using vercel.json with Go

Since Vercel needs Go installed, use their Go runtime:

```json
{
  "build": {
    "env": {
      "GO_VERSION": "1.22"
    }
  },
  "buildCommand": "curl -sSL https://go.dev/dl/go1.22.0.linux-amd64.tar.gz | tar -C /usr/local -xzf - && export PATH=$PATH:/usr/local/go/bin && go install github.com/example/markata-go/cmd/markata-go@latest && ~/go/bin/markata-go build --clean",
  "outputDirectory": "public"
}
```

---

## Deploying to Cloudflare Pages

Cloudflare Pages offers fast global CDN deployment with Workers integration.

### Build Configuration

Configure via the Cloudflare dashboard:

1. **Framework preset**: None
2. **Build command**:
   ```
   curl -sSL https://go.dev/dl/go1.22.0.linux-amd64.tar.gz | tar -xzf - && ./go/bin/go install github.com/example/markata-go/cmd/markata-go@latest && ~/go/bin/markata-go build --clean
   ```
3. **Build output directory**: `public`
4. **Root directory**: (leave empty)

### Environment Variables

Set in the Cloudflare dashboard under **Settings** > **Environment variables**:

| Variable | Production | Preview |
|----------|------------|---------|
| `MARKATA_GO_URL` | `https://example.com` | (leave empty) |
| `GO_VERSION` | `1.22` | `1.22` |

### _headers File

Create `static/_headers` for custom headers (copied to output):

```
/*
  X-Frame-Options: DENY
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin

/static/*
  Cache-Control: public, max-age=31536000, immutable

/*.html
  Cache-Control: public, max-age=0, must-revalidate
```

### _redirects File

Create `static/_redirects` for redirects:

```
/old-path/* /new-path/:splat 301
/blog/old-post /blog/new-post 301

# Proxy example
/api/* https://api.example.com/:splat 200
```

### Wrangler CLI Deployment

```bash
# Install wrangler
npm install -g wrangler

# Login
wrangler login

# Build and deploy
markata-go build --clean
wrangler pages deploy public --project-name=my-site
```

---

## Self-Hosting

### Using nginx

nginx configuration for serving a markata-go site:

```nginx
server {
    listen 80;
    listen [::]:80;
    server_name example.com www.example.com;

    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name example.com www.example.com;

    # SSL configuration
    ssl_certificate /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;
    ssl_session_timeout 1d;
    ssl_session_cache shared:SSL:50m;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers off;

    # HSTS
    add_header Strict-Transport-Security "max-age=63072000" always;

    # Document root
    root /var/www/example.com/public;
    index index.html;

    # Security headers
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml application/json application/javascript
               application/rss+xml application/atom+xml image/svg+xml;

    # Static assets caching
    location /static/ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # HTML files - no caching
    location ~* \.html$ {
        expires -1;
        add_header Cache-Control "no-store, no-cache, must-revalidate";
    }

    # XML feeds
    location ~* \.(xml|rss|atom)$ {
        expires 1h;
        add_header Cache-Control "public";
    }

    # Clean URLs - try file, then directory, then 404
    location / {
        try_files $uri $uri/ =404;
    }

    # Custom error pages
    error_page 404 /404.html;
    location = /404.html {
        internal;
    }
}
```

### Using Caddy

Caddy provides automatic HTTPS and simpler configuration:

```caddyfile
example.com {
    root * /var/www/example.com/public
    file_server

    # Compression
    encode gzip

    # Security headers
    header {
        X-Frame-Options "DENY"
        X-Content-Type-Options "nosniff"
        Referrer-Policy "strict-origin-when-cross-origin"
        Strict-Transport-Security "max-age=63072000"
    }

    # Static assets caching
    @static path /static/*
    header @static Cache-Control "public, max-age=31536000, immutable"

    # HTML no-cache
    @html path *.html
    header @html Cache-Control "no-cache, no-store, must-revalidate"

    # Handle clean URLs
    try_files {path} {path}/ {path}/index.html

    # Custom 404
    handle_errors {
        rewrite * /404.html
        file_server
    }
}

# Redirect www to non-www
www.example.com {
    redir https://example.com{uri} permanent
}
```

### Docker Deployment

#### Dockerfile

Create a multi-stage `Dockerfile`:

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Install markata-go
RUN go install github.com/example/markata-go/cmd/markata-go@latest

# Copy source
COPY . .

# Build site
ARG MARKATA_GO_URL=https://example.com
ENV MARKATA_GO_URL=$MARKATA_GO_URL

RUN markata-go build --clean

# Production stage
FROM nginx:alpine

# Copy nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy built site
COPY --from=builder /build/public /usr/share/nginx/html

# Healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost/ || exit 1

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
```

#### Docker nginx.conf

Create `nginx.conf` for the container:

```nginx
server {
    listen 80;
    server_name _;

    root /usr/share/nginx/html;
    index index.html;

    # Gzip
    gzip on;
    gzip_vary on;
    gzip_types text/plain text/css application/json application/javascript
               text/xml application/xml application/rss+xml image/svg+xml;

    # Security headers
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;

    # Static caching
    location /static/ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # Clean URLs
    location / {
        try_files $uri $uri/ =404;
    }

    error_page 404 /404.html;
}
```

#### docker-compose.yml

For easy deployment with Docker Compose:

```yaml
version: '3.8'

services:
  site:
    build:
      context: .
      args:
        MARKATA_GO_URL: https://example.com
    ports:
      - "80:80"
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost/"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Optional: Caddy reverse proxy with auto-HTTPS
  caddy:
    image: caddy:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy_data:/data
      - caddy_config:/config
    depends_on:
      - site

volumes:
  caddy_data:
  caddy_config:
```

#### Build and Run

```bash
# Build image
docker build -t my-site --build-arg MARKATA_GO_URL=https://example.com .

# Run container
docker run -d -p 80:80 --name my-site my-site

# Or with docker-compose
docker-compose up -d
```

---

## CI/CD Best Practices

### Caching Dependencies

#### GitHub Actions

```yaml
- name: Setup Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.22'
    cache: true  # Caches Go modules automatically

- name: Cache markata-go binary
  uses: actions/cache@v4
  with:
    path: ~/go/bin/markata-go
    key: markata-go-${{ runner.os }}-${{ hashFiles('go.sum') }}

- name: Install markata-go
  run: |
    if [ ! -f ~/go/bin/markata-go ]; then
      go install github.com/example/markata-go/cmd/markata-go@latest
    fi
```

#### GitLab CI

```yaml
variables:
  GOPATH: $CI_PROJECT_DIR/.go

cache:
  key: ${CI_COMMIT_REF_SLUG}
  paths:
    - .go/pkg/mod/
    - .go/bin/

build:
  image: golang:1.22
  script:
    - go install github.com/example/markata-go/cmd/markata-go@latest
    - markata-go build --clean
  artifacts:
    paths:
      - public/
```

### Environment-Specific Builds

Create separate workflows or jobs for different environments:

```yaml
name: Deploy

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true

      - name: Install markata-go
        run: go install github.com/example/markata-go/cmd/markata-go@latest

      - name: Build (Production)
        if: github.ref == 'refs/heads/main'
        run: markata-go build --clean
        env:
          MARKATA_GO_URL: https://example.com

      - name: Build (Preview)
        if: github.event_name == 'pull_request'
        run: markata-go build --clean
        env:
          MARKATA_GO_URL: https://preview-${{ github.event.number }}.example.com

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: site
          path: public/
```

### Preview Deployments

Automatically deploy preview environments for pull requests:

#### Netlify (automatic)

Netlify automatically creates deploy previews for PRs.

#### Vercel (automatic)

Vercel automatically creates preview deployments for PRs.

#### GitHub Pages with PR Previews

```yaml
name: Preview

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  preview:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true

      - name: Build
        run: |
          go install github.com/example/markata-go/cmd/markata-go@latest
          markata-go build --clean
        env:
          MARKATA_GO_URL: https://username.github.io/repo/pr-${{ github.event.number }}

      - name: Deploy Preview
        uses: peaceiris/actions-gh-pages@v4
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./public
          destination_dir: pr-${{ github.event.number }}

      - name: Comment PR
        uses: actions/github-script@v7
        with:
          script: |
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: '🚀 Preview deployed to https://username.github.io/repo/pr-${{ github.event.number }}/'
            })
```

### Build Validation

Add validation steps before deployment:

```yaml
- name: Build site
  run: markata-go build --clean

- name: Validate HTML
  run: |
    npm install -g html-validate
    html-validate "public/**/*.html" || true

- name: Check for broken links
  run: |
    npm install -g linkinator
    linkinator public --recurse --skip "^(?!https?://example.com)" || true

- name: Validate feeds
  run: |
    # Validate RSS
    xmllint --noout public/blog/rss.xml
    # Validate Atom
    xmllint --noout public/blog/atom.xml
```

### Complete CI/CD Workflow

A full workflow with caching, validation, and deployment:

```yaml
name: Build and Deploy

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history for git dates

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true

      - name: Install markata-go
        run: go install github.com/example/markata-go/cmd/markata-go@latest

      - name: Validate config
        run: markata-go config validate

      - name: Build site
        run: markata-go build --clean -v
        env:
          MARKATA_GO_URL: ${{ github.event_name == 'push' && 'https://example.com' || '' }}

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: ./public

  deploy:
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

---

## Troubleshooting

### Common Issues

**Build fails with "command not found"**

Ensure Go is in your PATH and markata-go is installed:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
go install github.com/example/markata-go/cmd/markata-go@latest
```

**Assets not loading**

Check that your `url` config matches your deployment URL:

```toml
[markata-go]
url = "https://example.com"  # Must match actual deployment URL
```

**RSS/Atom feeds show wrong URLs**

The base URL must be set correctly for feeds to work:

```bash
MARKATA_GO_URL=https://example.com markata-go build
```

**404 errors on page refresh**

Ensure your server is configured to serve `index.html` for directory requests. All hosting platforms in this guide handle this automatically.

### Debugging Deployments

Use verbose mode to debug build issues:

```bash
markata-go build --clean -v
```

Check the resolved configuration:

```bash
markata-go config show --sources
```

Validate before deploying:

```bash
markata-go config validate
```

---

## Next Steps

Congratulations on deploying your site! Here are ways to enhance it further:

**Share your content:**
- [Syndication Guide](/docs/guides/syndication/) - Automatically share posts to Mastodon, Twitter, and other platforms

**Add interactivity:**
- [Search Guide](/docs/guides/search/) - Add client-side search to your site
- [Dynamic Content Guide](/docs/guides/dynamic-content/) - Integrate with JavaScript frameworks

**Extend functionality:**
- [Plugin Development Guide](/docs/guides/plugin-development/) - Create custom plugins for advanced features

---

## See Also

- [Configuration Guide](/docs/guides/configuration/) - Full configuration reference
- [Feeds Guide](/docs/guides/feeds/) - RSS, Atom, and JSON feed configuration
- [Troubleshooting](/docs/troubleshooting/) - Common issues and solutions
- [Quick Reference](/docs/guides/quick-reference/) - CLI commands and config snippets
