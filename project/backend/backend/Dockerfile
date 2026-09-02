# Smart Fish Feeder Backend API Dockerfile
# Optimized for Railway deployment from the monorepo root

# Build stage
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy backend module files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source code from the monorepo
COPY /. .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=1.0.0" \
    -o main cmd/server/main.go

# Production stage - use distroless for smaller image
FROM gcr.io/distroless/static-debian12:nonroot

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder
COPY --from=builder /app/main /main

# Copy config files (optional, Railway uses env vars)
COPY --from=builder /app/configs /configs

# Railway sets PORT environment variable
# The app reads from SFF_SERVER_PORT or defaults to 8080
ENV SFF_SERVER_HOST=0.0.0.0

# Expose default port (Railway overrides this)
EXPOSE 8080

# Run as non-root user (distroless nonroot)
USER nonroot:nonroot

# Run the application
ENTRYPOINT ["/main"]
