# Multi-stage Dockerfile for markata-go
# Produces a minimal, secure container image for production use

# =============================================================================
# Build Stage
# =============================================================================
FROM golang:1.22-alpine AS builder

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
# Runtime Stage
# =============================================================================
# Using distroless static image for minimal attack surface
# - No shell, no package manager
# - Only statically-linked binaries can run
# - nonroot variant runs as non-root user by default
FROM gcr.io/distroless/static-debian12:nonroot

# Labels for container registry metadata
LABEL org.opencontainers.image.title="markata-go"
LABEL org.opencontainers.image.description="A plugin-driven static site generator written in Go"
LABEL org.opencontainers.image.url="https://github.com/WaylonWalker/markata-go"
LABEL org.opencontainers.image.source="https://github.com/WaylonWalker/markata-go"
LABEL org.opencontainers.image.vendor="Waylon Walker"
LABEL org.opencontainers.image.licenses="MIT"

# Copy CA certificates for HTTPS support
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/markata-go /usr/local/bin/markata-go

# Set working directory for user content
WORKDIR /site

# Run as non-root user (provided by distroless:nonroot)
USER nonroot:nonroot

# Default entrypoint
ENTRYPOINT ["markata-go"]

# Default command shows help
CMD ["--help"]
