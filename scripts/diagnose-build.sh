#!/bin/bash
# Diagnostic script to identify build inconsistencies
# Run this on both machines and compare output

echo "=== Markata-Go Build Diagnostics ==="
echo ""

# Environment info
echo "--- Environment ---"
echo "Hostname: $(hostname)"
echo "User: $USER"
echo "Date: $(date)"
echo "Go version: $(go version)"
echo "Working directory: $(pwd)"
echo ""

# Git info
echo "--- Git Status ---"
if [ -d .git ]; then
    echo "Commit: $(git log -1 --oneline)"
    echo "Branch: $(git branch --show-current)"
    UNCOMMITTED=$(git status --short | wc -l)
    echo "Uncommitted files: $UNCOMMITTED"
    if [ $UNCOMMITTED -gt 0 ]; then
        echo "Warning: Uncommitted changes detected!"
        git status --short
    fi
else
    echo "Not a git repository"
fi
echo ""

# Binary info
echo "--- Binary Information ---"
if [ -f ~/go/bin/markata-go ]; then
    echo "Binary: ~/go/bin/markata-go"
    ls -lh ~/go/bin/markata-go
    echo "SHA256: $(sha256sum ~/go/bin/markata-go | awk '{print $1}')"
    echo "Modified: $(stat -c %y ~/go/bin/markata-go)"
else
    echo "No binary at ~/go/bin/markata-go"
fi

if [ -f ./markata-go ]; then
    echo ""
    echo "Binary: ./markata-go"
    ls -lh ./markata-go
    echo "SHA256: $(sha256sum ./markata-go | awk '{print $1}')"
    echo "Modified: $(stat -c %y ./markata-go)"
fi
echo ""

# Template checksums (embedded - these go into the binary)
echo "--- Embedded Template Checksums ---"
for template in \
    "pkg/themes/default/templates/feed.html" \
    "pkg/themes/default/templates/post.html" \
    "pkg/themes/default/templates/partials/cards/article-card.html" \
    "pkg/themes/default/templates/partials/cards/default-card.html" \
    "pkg/themes/default/templates/partials/cards/card-router.html"
do
    if [ -f "$template" ]; then
        echo "$(sha256sum "$template" | awk '{print $1}')  $template"
    else
        echo "MISSING: $template"
    fi
done
echo ""

# Template checksums (project overrides)
echo "--- Project Template Overrides ---"
for template in \
    "templates/feed.html" \
    "templates/post.html" \
    "templates/partials/cards/article-card.html"
do
    if [ -f "$template" ]; then
        echo "$(sha256sum "$template" | awk '{print $1}')  $template"
    else
        echo "Not overridden: $template"
    fi
done
echo ""

# Cache status
echo "--- Cache Status ---"
if [ -d .markata ]; then
    echo "Markata cache exists: .markata/"
    ls -lh .markata/ 2>/dev/null | head -10
else
    echo "No markata cache"
fi
echo ""

# Config file
echo "--- Config File ---"
if [ -f markata-go.toml ]; then
    echo "Config: markata-go.toml"
    echo "SHA256: $(sha256sum markata-go.toml | awk '{print $1}')"
    echo "Size: $(stat -c %s markata-go.toml) bytes"
else
    echo "No markata-go.toml found"
fi
echo ""

# Recent build output sample
echo "--- Sample Build Output ---"
if [ -f public/index.html ]; then
    echo "Output exists: public/"
    echo "Index size: $(stat -c %s public/index.html) bytes"
    # Look for a card in the output
    if grep -q 'class="card' public/index.html 2>/dev/null; then
        echo "Cards found in output:"
        grep -o 'class="card[^"]*"' public/index.html | head -5
    else
        echo "No cards found in index.html"
    fi
else
    echo "No output at public/"
fi
echo ""

echo "=== End Diagnostics ==="
