# FastCache API Server Docker Image
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the API server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-server examples/api-server/main.go

# Build the load tester
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o load-tester tools/load-tester/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates and curl for health checks
RUN apk --no-cache add ca-certificates curl

# Create non-root user
RUN addgroup -g 1000 -S fastcache && \
    adduser -u 1000 -S fastcache -G fastcache

WORKDIR /home/fastcache

# Copy binaries from builder
COPY --from=builder /app/api-server .
COPY --from=builder /app/load-tester .

# Copy example configuration
COPY examples/docker/config.json .

# Change ownership
RUN chown -R fastcache:fastcache /home/fastcache

# Switch to non-root user
USER fastcache

# Expose port
EXPOSE 8080 8090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default command
CMD ["./api-server"]