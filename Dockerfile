# Multi-stage Dockerfile for markata-go
# Produces a minimal, secure container image for production use

# =============================================================================
# Build Stage
# =============================================================================
FROM golang:1.24-alpine AS builder

# Install ca-certificates for HTTPS support in final image
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build arguments for version injection
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

# Build static binary with version information
# CGO_ENABLED=0 ensures fully static binary
# -s -w strips debug information for smaller binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w \
        -X github.com/WaylonWalker/markata-go/cmd/markata-go/cmd.Version=${VERSION} \
        -X github.com/WaylonWalker/markata-go/cmd/markata-go/cmd.Commit=${COMMIT} \
        -X github.com/WaylonWalker/markata-go/cmd/markata-go/cmd.Date=${BUILD_DATE}" \
    -o markata-go \
    ./cmd/markata-go

# Verify binary was built
RUN ./markata-go version

# =============================================================================
# Builder Runtime Stage
# =============================================================================
FROM alpine:3.20 AS builder-runtime

# Labels for container registry metadata
LABEL org.opencontainers.image.title="markata-go-builder"
LABEL org.opencontainers.image.description="Builder image for markata-go with shell and publish tooling"
LABEL org.opencontainers.image.url="https://github.com/WaylonWalker/markata-go"
LABEL org.opencontainers.image.source="https://github.com/WaylonWalker/markata-go"
LABEL org.opencontainers.image.vendor="Waylon Walker"
LABEL org.opencontainers.image.licenses="MIT"

# Install shell, TLS roots, and build/publish tooling
# Note: nodejs/npm/@mermaid-js/mermaid-cli are NOT needed here because
# markata-go's "chromium" mode uses chromedp (Go-native CDP) directly,
# and pagefind is installed as a standalone binary below.
RUN apk add --no-cache \
    ca-certificates \
    coreutils \
    findutils \
    gawk \
    git \
    libavif-apps \
    libwebp-tools \
    chromium \
    openssh-client \
    rsync \
    tzdata

# Install pagefind as a standalone binary (musl-static, no Node.js needed)
ARG PAGEFIND_VERSION=v1.4.0
RUN wget -q "https://github.com/Pagefind/pagefind/releases/download/${PAGEFIND_VERSION}/pagefind-${PAGEFIND_VERSION}-x86_64-unknown-linux-musl.tar.gz" \
        -O /tmp/pagefind.tar.gz \
    && tar xzf /tmp/pagefind.tar.gz -C /usr/local/bin/ pagefind \
    && chmod +x /usr/local/bin/pagefind \
    && rm /tmp/pagefind.tar.gz

# Copy the binary
COPY --from=builder /app/markata-go /usr/local/bin/markata-go

# Set working directory for user content
WORKDIR /site

# Default command opens a shell for scripts
CMD ["sh"]

# =============================================================================
# Runtime Stage
# =============================================================================
# Use scratch for the smallest possible runtime image
FROM scratch

# Labels for container registry metadata
LABEL org.opencontainers.image.title="markata-go"
LABEL org.opencontainers.image.description="A plugin-driven static site generator written in Go"
LABEL org.opencontainers.image.url="https://github.com/WaylonWalker/markata-go"
LABEL org.opencontainers.image.source="https://github.com/WaylonWalker/markata-go"
LABEL org.opencontainers.image.vendor="Waylon Walker"
LABEL org.opencontainers.image.licenses="MIT"

# Copy CA certificates for HTTPS support
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy the binary
COPY --from=builder /app/markata-go /usr/local/bin/markata-go

# Set working directory for user content
WORKDIR /site

# Run as non-root user (numeric UID/GID)
USER 65532:65532

# Default entrypoint
ENTRYPOINT ["markata-go"]

# Default command shows help
CMD ["--help"]
