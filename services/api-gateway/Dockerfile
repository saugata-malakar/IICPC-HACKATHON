# ─────────────────────────────────────────────────────────────────────────────
# Multi-stage build for API Gateway
# ─────────────────────────────────────────────────────────────────────────────

FROM golang:1.22-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files and source
COPY go.mod ./
COPY . .

# Download deps and build
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=2.0.0" \
    -o api-gateway \
    main.go

# ─────────────────────────────────────────────────────────────────────────────
# Runtime stage
# ─────────────────────────────────────────────────────────────────────────────

FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /build/api-gateway /app/api-gateway

# Non-root user
RUN addgroup -g 1000 iicpc && \
    adduser -D -u 1000 -G iicpc iicpc && \
    chown -R iicpc:iicpc /app

USER iicpc

EXPOSE 8080 9090

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/api-gateway"]
