# Docker Deployment for markata-go

This directory contains Docker Compose configurations for deploying markata-go sites.

## Quick Start

### Development (Local Preview)

```bash
# From your site directory
cd deploy/docker

# Start development server with hot reload
docker compose -f docker-compose.dev.yml up

# Site available at http://localhost:8000
```

### Production (Self-Hosted)

```bash
# 1. Configure your environment
cp .env.example .env
# Edit .env with your domain and settings

# 2. Start production stack
docker compose -f docker-compose.prod.yml up -d

# Site available at https://your-domain.com (after DNS propagation)
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SITE_DOMAIN` | Your domain name | `example.com` |
| `SITE_URL` | Full site URL | `https://example.com` |
| `SITE_DIR` | Path to site content | `./../..` |
| `PORT` | Development port | `8000` |

### Development Stack

The development configuration (`docker-compose.dev.yml`) provides:

- Hot reload on file changes
- Local preview server on port 8000
- Go module caching for faster rebuilds

```bash
# Custom port
PORT=3000 docker compose -f docker-compose.dev.yml up

# Custom site directory
SITE_DIR=/path/to/my/site docker compose -f docker-compose.dev.yml up
```

### Production Stack

The production configuration (`docker-compose.prod.yml`) includes:

- Multi-stage build process
- Caddy reverse proxy with automatic HTTPS
- Security headers (HSTS, CSP, etc.)
- Gzip/Zstd compression
- Optimized caching headers
- Health checks
- Optional Watchtower for auto-updates

## Caddy Configuration

The included `Caddyfile` provides:

- **Automatic HTTPS** via Let's Encrypt
- **HTTP/2 and HTTP/3** support
- **Security headers** (HSTS, X-Frame-Options, etc.)
- **Compression** (gzip, zstd)
- **Caching** (static assets: 1 year, HTML: no-cache, feeds: 1 hour)
- **Clean URLs** handling
- **Custom 404 page**

### Customizing Caddy

Edit `Caddyfile` to:

- Add redirects
- Configure additional domains
- Adjust caching rules
- Add basic auth

Example: Adding basic auth:

```caddyfile
example.com {
    # Add before file_server
    basicauth /admin/* {
        admin $2a$14$... # Use: caddy hash-password
    }

    root * /srv
    file_server
    # ...
}
```

## Rebuilding the Site

### Manual Rebuild

```bash
# Rebuild and restart
docker compose -f docker-compose.prod.yml up -d --build

# Or rebuild just the builder service
docker compose -f docker-compose.prod.yml up builder
docker compose -f docker-compose.prod.yml restart caddy
```

### Automated Rebuilds

For automatic rebuilds on content changes, consider:

1. **Webhook-triggered rebuilds** - Set up a webhook to trigger rebuilds
2. **Scheduled rebuilds** - Use cron to periodically rebuild
3. **Git-based rebuilds** - Use CI/CD to rebuild on push

Example cron job for hourly rebuilds:

```bash
0 * * * * cd /path/to/deploy/docker && docker compose -f docker-compose.prod.yml up builder && docker compose -f docker-compose.prod.yml restart caddy
```

## Troubleshooting

### Build Fails

```bash
# View build logs
docker compose -f docker-compose.prod.yml logs builder

# Rebuild without cache
docker compose -f docker-compose.prod.yml build --no-cache
```

### HTTPS Not Working

1. Ensure your domain points to the server's IP
2. Check ports 80 and 443 are open
3. View Caddy logs:

```bash
docker compose -f docker-compose.prod.yml logs caddy
```

### Container Won't Start

```bash
# Check container status
docker compose -f docker-compose.prod.yml ps

# View all logs
docker compose -f docker-compose.prod.yml logs

# Verify volumes
docker volume ls | grep markata
```

### Clear Everything and Start Fresh

```bash
docker compose -f docker-compose.prod.yml down -v
docker compose -f docker-compose.prod.yml up -d
```

## Resource Requirements

### Minimum

- 1 CPU core
- 512 MB RAM
- 1 GB disk space

### Recommended

- 2 CPU cores
- 1 GB RAM
- 5 GB disk space (for build cache)

## Security Considerations

1. **Keep containers updated** - Enable Watchtower or manually update
2. **Use secrets** - Don't commit `.env` files with sensitive data
3. **Firewall** - Only expose ports 80 and 443
4. **Monitor logs** - Check Caddy access logs for suspicious activity

## See Also

- [Self-Hosting Guide](../../docs/guides/self-hosting.md)
- [systemd Deployment](../systemd/README.md)
- [Deployment Guide](../../docs/guides/deployment.md)
