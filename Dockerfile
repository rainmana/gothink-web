# Multi-stage build for GoThink MCP Server
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the HTTP server binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o gothink-http \
    ./cmd/gothink-http

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/gothink-http .

# Copy mental models examples
COPY --from=builder /build/examples ./examples

# Create non-root user
RUN addgroup -g 1001 -S gothink && \
    adduser -S gothink -u 1001 -G gothink

# Change ownership
RUN chown -R gothink:gothink /app

USER gothink

# Expose port (will be overridden by cloud platforms)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-8080}/health || exit 1

CMD ["./gothink-http"]
