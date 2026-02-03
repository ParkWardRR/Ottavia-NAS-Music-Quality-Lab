# Ottavia - Music Quality Lab
# Multi-stage build for minimal image size

# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev nodejs npm

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy package.json for npm
COPY package.json package-lock.json* ./
RUN npm ci --only=production

# Copy source
COPY . .

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

# Generate templates
RUN templ generate

# Build CSS
RUN npm run css:build

# Build binary
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o ottavia ./cmd/server

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ffmpeg \
    ca-certificates \
    tzdata \
    sqlite

# Create non-root user
RUN addgroup -g 1000 ottavia && \
    adduser -u 1000 -G ottavia -h /app -D ottavia

WORKDIR /app

# Copy binary and static files
COPY --from=builder /app/ottavia .
COPY --from=builder /app/web/static ./web/static

# Create data directories
RUN mkdir -p /data/artifacts /data/temp && \
    chown -R ottavia:ottavia /app /data

# Switch to non-root user
USER ottavia

# Environment
ENV SEVILLE_PORT=8080 \
    SEVILLE_DB_DSN=/data/ottavia.db \
    SEVILLE_ARTIFACTS_PATH=/data/artifacts \
    SEVILLE_TEMP_PATH=/data/temp

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/health || exit 1

# Run
ENTRYPOINT ["./ottavia"]
