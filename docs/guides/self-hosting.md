---
title: "Self-Hosting Guide"
description: "Deploy markata-go sites on your own servers using Docker, systemd, or manual setups"
date: 2024-01-23
published: true
tags:
  - documentation
  - deployment
  - self-hosting
  - docker
  - systemd
---

# Self-Hosting Guide

This guide covers deploying markata-go sites on your own infrastructure. Whether you prefer Docker containers, systemd services, or manual setups, you'll find everything you need to self-host your static site.

## Overview

Self-hosting gives you complete control over your site's infrastructure. markata-go supports several deployment methods:

| Method | Best For | Complexity |
|--------|----------|------------|
| Docker Compose | Quick setup, portability | Low |
| systemd | Linux servers, integration | Medium |
| Manual | Custom setups, learning | High |

**Prerequisites:**
- A Linux server (VPS, dedicated, or home server)
- Domain name pointed to your server
- Basic command line knowledge

## Quick Start with Docker

The fastest way to self-host is using Docker Compose:

```bash
# Clone your site or create a new one
git clone https://github.com/you/your-site.git
cd your-site

# Copy the deployment files
cp -r deploy/docker ./docker-deploy
cd docker-deploy

# Configure environment
cp .env.example .env
# Edit .env with your domain: SITE_DOMAIN=example.com

# Start production stack
docker compose -f docker-compose.prod.yml up -d
```

Your site will be available at `https://your-domain.com` with automatic HTTPS.

See [Docker Deployment README](https://github.com/WaylonWalker/markata-go/tree/main/deploy/docker) for detailed configuration options.

## Docker Compose Configurations

### Development Mode

For local development with hot reload:

```bash
cd deploy/docker
docker compose -f docker-compose.dev.yml up
```

Features:
- Hot reload on file changes
- Local preview at http://localhost:8000
- Go module caching

### Production Mode

For production with Caddy reverse proxy:

```bash
cd deploy/docker

# Configure
cp .env.example .env
vim .env  # Set SITE_DOMAIN and SITE_URL

# Deploy
docker compose -f docker-compose.prod.yml up -d
```

Features:
- Automatic HTTPS via Let's Encrypt
- HTTP/2 and HTTP/3 support
- Security headers
- Gzip/Zstd compression
- Optimized caching

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SITE_DOMAIN` | Your domain name | `example.com` |
| `SITE_URL` | Full URL for feeds/links | `https://example.com` |
| `SITE_DIR` | Path to site content | `./../..` |
| `PORT` | Development port | `8000` |

## systemd Services

For traditional Linux server deployments, use systemd services.

### Quick Setup

```bash
cd deploy/systemd
sudo ./install.sh
```

The installer prompts for:
- Site directory (default: `/var/www/mysite`)
- Site URL (default: `https://example.com`)
- Service user (default: `markata`)

### Manual Setup

```bash
# Create service user
sudo useradd -r -s /bin/false -d /var/www/mysite markata

# Create directories
sudo mkdir -p /var/www/mysite/{public,.go}
sudo chown -R markata:markata /var/www/mysite

# Install markata-go
sudo -u markata bash -c '
    export GOPATH=/var/www/mysite/.go
    go install github.com/WaylonWalker/markata-go/cmd/markata-go@latest
'

# Install services
sudo cp deploy/systemd/markata-go.service /etc/systemd/system/
sudo cp deploy/systemd/markata-go-watch.service /etc/systemd/system/
# Edit paths in service files

sudo systemctl daemon-reload
```

### Available Services

**markata-go.service** - One-time build:
```bash
sudo systemctl enable markata-go
sudo systemctl start markata-go

# Rebuild site
sudo systemctl restart markata-go
```

**markata-go-watch.service** - Auto-rebuild on changes:
```bash
sudo systemctl enable markata-go-watch
sudo systemctl start markata-go-watch

# View logs
sudo journalctl -u markata-go-watch -f
```

See [systemd Deployment README](https://github.com/WaylonWalker/markata-go/tree/main/deploy/systemd) for detailed setup and troubleshooting.

## Web Server Configuration

### Using nginx

If using systemd with nginx:

```nginx
server {
    listen 80;
    listen [::]:80;
    server_name example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name example.com;

    ssl_certificate /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;

    root /var/www/mysite/public;
    index index.html;

    # Security headers
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;

    # Caching
    location /static/ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    location / {
        try_files $uri $uri/ =404;
    }
}
```

### Using Caddy

Simpler alternative with automatic HTTPS:

```caddyfile
example.com {
    root * /var/www/mysite/public
    file_server
    encode gzip

    header {
        X-Frame-Options "DENY"
        X-Content-Type-Options "nosniff"
    }

    @static path /static/*
    header @static Cache-Control "public, max-age=31536000, immutable"

    try_files {path} {path}/ {path}/index.html
}
```

## Automated Rebuilds

### Git Webhooks

Rebuild on push using a webhook endpoint:

```bash
#!/bin/bash
# /usr/local/bin/rebuild-site.sh
cd /var/www/mysite
git pull origin main
sudo systemctl restart markata-go
```

### Scheduled Rebuilds

Using systemd timer:

```ini
# /etc/systemd/system/markata-go.timer
[Unit]
Description=Rebuild site hourly

[Timer]
OnCalendar=hourly
Persistent=true

[Install]
WantedBy=timers.target
```

```bash
sudo systemctl enable markata-go.timer
sudo systemctl start markata-go.timer
```

### CI/CD Integration

Build in CI and deploy via rsync:

```yaml
# .github/workflows/deploy.yml
deploy:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - run: |
        go install github.com/WaylonWalker/markata-go/cmd/markata-go@latest
        markata-go build --clean
    - run: |
        rsync -avz --delete public/ user@server:/var/www/mysite/public/
```

## Security Checklist

Before going live:

- [ ] Firewall configured (only ports 80, 443, and SSH)
- [ ] SSH key authentication only (disable password auth)
- [ ] Automatic security updates enabled
- [ ] HTTPS working with valid certificate
- [ ] Security headers configured
- [ ] Service running as non-root user
- [ ] File permissions restricted

### Firewall Setup (ufw)

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow http
sudo ufw allow https
sudo ufw enable
```

## Monitoring

### Basic Health Checks

```bash
# Check if site is responding
curl -I https://example.com

# Check systemd service
sudo systemctl status markata-go

# View recent logs
sudo journalctl -u markata-go -n 100
```

### Uptime Monitoring

Consider using:
- [Uptime Robot](https://uptimerobot.com/) (free tier available)
- [Healthchecks.io](https://healthchecks.io/) for cron monitoring
- Self-hosted: [Uptime Kuma](https://github.com/louislam/uptime-kuma)

## Troubleshooting

### Site Not Loading

1. Check if the service is running:
   ```bash
   sudo systemctl status markata-go
   docker compose ps
   ```

2. Check if web server is running:
   ```bash
   sudo systemctl status nginx  # or caddy
   ```

3. Check firewall:
   ```bash
   sudo ufw status
   ```

### HTTPS Not Working

1. Verify DNS is pointing to your server
2. Check Let's Encrypt logs:
   ```bash
   # Caddy
   docker compose logs caddy
   # Certbot
   sudo certbot certificates
   ```

### Build Failures

1. Check build logs:
   ```bash
   sudo journalctl -u markata-go
   docker compose logs builder
   ```

2. Run build manually:
   ```bash
   cd /var/www/mysite
   markata-go build --clean -v
   ```

## Resource Requirements

| Setup | CPU | RAM | Disk |
|-------|-----|-----|------|
| Minimum | 1 core | 512 MB | 1 GB |
| Recommended | 2 cores | 1 GB | 5 GB |
| With build cache | 2 cores | 2 GB | 10 GB |

## Next Steps

- [Configuration Guide](/docs/guides/configuration/) - Customize your site
- [Deployment Guide](/docs/guides/deployment/) - Other hosting options
- [Themes Guide](/docs/guides/themes/) - Customize appearance
