# Multi-stage Dockerfile for artifact-cli
# This Dockerfile creates a minimal production image with the artifact-cli binary

# Build stage
FROM golang:1.25-alpine AS builder
# Set working directory
WORKDIR /app
# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata
# Copy go mod files
COPY go.mod go.sum ./
# Download dependencies
RUN go mod download
# Copy source code
COPY . .
# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o artifact-cli cmd/root.go

# Production stage
FROM alpine:3.19
# Labels for metadata
LABEL maintainer="educates-team" \
      description="A CLI tool to push, pull, and sync folders as OCI artifacts" \
      version="1.0.0" \
      org.opencontainers.image.title="artifact-cli" \
      org.opencontainers.image.description="A CLI tool to push, pull, and sync folders as OCI artifacts" \
      org.opencontainers.image.vendor="educates-team" \
      org.opencontainers.image.version="1.0.0" \
      org.opencontainers.image.source="https://github.com/educates/educates-artifact-cli"
# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && rm -rf /var/cache/apk/*
# Create non-root user for security
RUN addgroup -g 1001 -S artifact && \
    adduser -u 1001 -S artifact -G artifact
# Set working directory
WORKDIR /app
# Copy the binary from builder stage
COPY --from=builder /app/artifact-cli /usr/local/bin/artifact-cli
# Make binary executable
RUN chmod +x /usr/local/bin/artifact-cli
# Switch to non-root user
USER artifact
# Set default working directory for user operations
WORKDIR /home/artifact
# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD artifact-cli --help > /dev/null || exit 1
# Default command
ENTRYPOINT ["artifact-cli"]
CMD ["--help"]
