# ------------------------------------------------------------------------------------
# Stage 1: Builder (Build the Go application)
# ------------------------------------------------------------------------------------
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

ARG HTTP_PROXY
ENV http_proxy=$HTTP_PROXY

ARG HTTPS_PROXY
ENV https_proxy=$HTTPS_PROXY

# Install wire for dependency injection
RUN go install github.com/google/wire/cmd/wire@latest

# Set working directory
WORKDIR /build

# Copy go mod files first (for better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# example copy the env
#RUN cp /app/.env.stage.example /app/.env

# Generate wire code and build the application
RUN cd internal/apps && wire
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o main ./cmd

# ------------------------------------------------------------------------------------
# Stage 2: Runtime (Lightweight final image)
# ------------------------------------------------------------------------------------
FROM new-nexus.bri.co.id/mks/base/alpine:3.16 AS runtime

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /build/ /app/

# Change ownership to non-root user
RUN chown -R www-data:www-data /app
RUN chmod -R 775 /app

# Expose port (default from config)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

USER www-data

ENV http_proxy ""
ENV https_proxy ""

# Run the application
CMD ["./main"]
