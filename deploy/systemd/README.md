# systemd Deployment for markata-go

This directory contains systemd service files for running markata-go on Linux servers.

## Quick Start

### Automated Installation

```bash
# Run the installation script
sudo ./install.sh
```

The script will:
1. Create a dedicated service user
2. Set up the site directory
3. Install Go (if needed)
4. Install markata-go
5. Install and configure systemd services

### Manual Installation

```bash
# 1. Create service user
sudo useradd -r -s /bin/false -d /var/www/mysite markata

# 2. Create directories
sudo mkdir -p /var/www/mysite/{public,.go}
sudo chown -R markata:markata /var/www/mysite

# 3. Install markata-go
sudo -u markata bash -c '
    export HOME=/var/www/mysite
    export GOPATH=/var/www/mysite/.go
    go install github.com/WaylonWalker/markata-go/cmd/markata-go@latest
'

# 4. Copy and edit service files
sudo cp markata-go.service /etc/systemd/system/
sudo cp markata-go-watch.service /etc/systemd/system/
# Edit paths in /etc/systemd/system/markata-go*.service

# 5. Reload systemd
sudo systemctl daemon-reload
```

## Services

### markata-go.service (Build on Demand)

One-shot service that builds the site when started. Use this with an external web server (nginx, Caddy) serving the static files.

```bash
# Enable and start
sudo systemctl enable markata-go
sudo systemctl start markata-go

# Rebuild the site
sudo systemctl restart markata-go

# Check build status
sudo systemctl status markata-go
sudo journalctl -u markata-go
```

### markata-go-watch.service (Continuous Watch)

Long-running service that watches for file changes and rebuilds automatically. Also serves a preview on port 8000.

```bash
# Enable and start
sudo systemctl enable markata-go-watch
sudo systemctl start markata-go-watch

# View live logs
sudo journalctl -u markata-go-watch -f

# Restart after config changes
sudo systemctl restart markata-go-watch
```

## Configuration

### Service File Customization

Edit the service files in `/etc/systemd/system/` to customize:

```ini
[Service]
# Site content directory
WorkingDirectory=/var/www/mysite

# Site URL for absolute links
Environment=MARKATA_GO_URL=https://example.com

# Output directory
ExecStart=/var/www/mysite/.go/bin/markata-go build --clean -o /var/www/mysite/public
```

### After Editing Services

```bash
# Reload systemd configuration
sudo systemctl daemon-reload

# Restart the service
sudo systemctl restart markata-go
```

## Integration with Web Servers

### nginx

```nginx
server {
    listen 80;
    server_name example.com;

    root /var/www/mysite/public;
    index index.html;

    # MIME types for txt/md files
    location ~ \.(txt|md)$ {
        default_type text/plain;
        charset utf-8;
    }

    # Try exact file first (for /robots.txt), then index.html, then directory
    location / {
        try_files $uri $uri/index.html $uri/ =404;
    }
}
```

**Testing nginx configuration:**

```bash
# Test syntax
sudo nginx -t

# Reload
sudo systemctl reload nginx

# Verify txt files work
curl -I http://localhost/robots.txt
```

### Caddy

```caddyfile
example.com {
    root * /var/www/mysite/public
    file_server

    # MIME types for txt/md files
    @txtmd path *.txt *.md
    header @txtmd Content-Type "text/plain; charset=utf-8"

    # Try exact file first (for /robots.txt), then index.html
    try_files {path} {path}/index.html {path}/
}
```

## Automated Rebuilds

### Timer-based Rebuilds

Create a timer for periodic rebuilds:

```bash
# /etc/systemd/system/markata-go.timer
[Unit]
Description=Rebuild markata-go site hourly

[Timer]
OnCalendar=hourly
Persistent=true

[Install]
WantedBy=timers.target
```

Enable the timer:

```bash
sudo systemctl enable markata-go.timer
sudo systemctl start markata-go.timer

# Check timer status
sudo systemctl list-timers
```

### Webhook-triggered Rebuilds

For Git-based workflows, trigger rebuilds via webhook:

```bash
# Simple webhook script
#!/bin/bash
# /usr/local/bin/rebuild-site.sh
cd /var/www/mysite
git pull
sudo systemctl restart markata-go
```

## Logging

View logs using journalctl:

```bash
# All logs
sudo journalctl -u markata-go

# Follow logs in real-time
sudo journalctl -u markata-go -f

# Logs since last boot
sudo journalctl -u markata-go -b

# Logs from last hour
sudo journalctl -u markata-go --since "1 hour ago"
```

## Troubleshooting

### Service Won't Start

```bash
# Check status and recent logs
sudo systemctl status markata-go
sudo journalctl -u markata-go -n 50

# Verify paths and permissions
ls -la /var/www/mysite
sudo -u markata ls -la /var/www/mysite/.go/bin/
```

### Permission Denied Errors

```bash
# Fix ownership
sudo chown -R markata:markata /var/www/mysite

# Check SELinux (if enabled)
sudo setenforce 0  # Temporarily disable for testing
```

### Build Fails

```bash
# Run manually as service user
sudo -u markata bash -c '
    export HOME=/var/www/mysite
    export GOPATH=/var/www/mysite/.go
    export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
    cd /var/www/mysite
    markata-go build --clean -v
'
```

### Go Not Found

```bash
# Verify Go installation
which go
go version

# Add to PATH if needed
export PATH=$PATH:/usr/local/go/bin
```

## Security Notes

The service files include security hardening:

- **NoNewPrivileges**: Prevents privilege escalation
- **ProtectSystem=strict**: Read-only file system except allowed paths
- **ProtectHome=yes**: No access to home directories
- **PrivateTmp=yes**: Isolated /tmp directory
- **Resource limits**: Memory and CPU caps (watch service)

## Uninstalling

```bash
# Stop and disable services
sudo systemctl stop markata-go markata-go-watch
sudo systemctl disable markata-go markata-go-watch

# Remove service files
sudo rm /etc/systemd/system/markata-go.service
sudo rm /etc/systemd/system/markata-go-watch.service
sudo systemctl daemon-reload

# Optionally remove user and data
sudo userdel markata
sudo rm -rf /var/www/mysite
```

## See Also

- [Self-Hosting Guide](../../docs/guides/self-hosting.md)
- [Docker Deployment](../docker/README.md)
- [Deployment Guide](../../docs/guides/deployment.md)
