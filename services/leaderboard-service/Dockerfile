# ─────────────────────────────────────────────────────────────────────────────
# Multi-stage build for Leaderboard Service
# ─────────────────────────────────────────────────────────────────────────────

FROM golang:1.22-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod ./
COPY . .

RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=2.0.0" \
    -o leaderboard-service \
    main.go

# ─────────────────────────────────────────────────────────────────────────────
# Runtime stage
# ─────────────────────────────────────────────────────────────────────────────

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /build/leaderboard-service /app/leaderboard-service

RUN addgroup -g 1000 iicpc && \
    adduser -D -u 1000 -G iicpc iicpc && \
    chown -R iicpc:iicpc /app

USER iicpc

EXPOSE 8003

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8003/health || exit 1

ENTRYPOINT ["/app/leaderboard-service"]
