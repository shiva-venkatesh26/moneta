# Moneta - Local-first code memory system
# Multi-stage build for minimal image size

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies (needed for CGO/SQLite)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files first for layer caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source
COPY . .

# Build with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o moneta ./cmd/moneta

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN adduser -D -g '' moneta

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/moneta /usr/local/bin/moneta

# Create data directory
RUN mkdir -p /data && chown moneta:moneta /data

# Switch to non-root user
USER moneta

# Set data directory
ENV MONETA_DATA_DIR=/data

# Expose server port
EXPOSE 3456

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3456/health || exit 1

# Default command: start server
ENTRYPOINT ["moneta"]
CMD ["serve", "--host", "0.0.0.0"]
