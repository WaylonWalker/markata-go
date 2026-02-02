#!/bin/bash
# Clean rebuild script for markata-go
# Ensures all caches are cleared and binary is rebuilt from scratch

set -e  # Exit on error

echo "=== Markata-Go Clean Rebuild Script ==="
echo ""

# Store current directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

echo "Working directory: $PROJECT_ROOT"
echo ""

# 1. Show git status
echo "Step 1: Checking git status..."
git status --short
COMMIT=$(git log -1 --oneline)
echo "Current commit: $COMMIT"
echo ""

# 2. Clean Go caches
echo "Step 2: Cleaning Go build cache..."
go clean -cache
echo "✓ Go cache cleaned"
echo ""

# 3. Remove old binaries
echo "Step 3: Removing old binaries..."
rm -f ./markata-go
rm -f ~/go/bin/markata-go
echo "✓ Old binaries removed"
echo ""

# 4. Remove markata build cache
echo "Step 4: Removing markata build cache..."
rm -rf .markata
rm -rf output
echo "✓ Markata caches removed"
echo ""

# 5. Verify template checksums (for debugging)
echo "Step 5: Template checksums (for verification)..."
echo "Embedded templates:"
sha256sum pkg/themes/default/templates/partials/cards/article-card.html | awk '{print $1}' | head -c 16
echo " - article-card.html"
sha256sum pkg/themes/default/templates/feed.html | awk '{print $1}' | head -c 16
echo " - feed.html"
echo ""

# 6. Rebuild binary
echo "Step 6: Building markata-go (this may take a minute)..."
go install -a ./cmd/markata-go
echo "✓ Binary installed to ~/go/bin/markata-go"
echo ""

# 7. Verify installation
if [ -f ~/go/bin/markata-go ]; then
    echo "Step 7: Verifying installation..."
    BINARY_SIZE=$(stat -c %s ~/go/bin/markata-go)
    BINARY_DATE=$(stat -c %y ~/go/bin/markata-go | cut -d. -f1)
    echo "Binary size: $(numfmt --to=iec-i --suffix=B $BINARY_SIZE)"
    echo "Binary date: $BINARY_DATE"
    echo "✓ Installation verified"
else
    echo "✗ ERROR: Binary not found at ~/go/bin/markata-go"
    exit 1
fi

echo ""
echo "=== Rebuild Complete ==="
echo ""
echo "Next steps:"
echo "  1. Run: markata-go build --clean"
echo "  2. Check output in: output/"
echo ""
echo "For debugging, share these checksums:"
sha256sum pkg/themes/default/templates/partials/cards/article-card.html | awk '{print "  article-card: " $1}'
sha256sum pkg/themes/default/templates/feed.html | awk '{print "  feed.html:    " $1}'
echo "  commit:       $COMMIT"
