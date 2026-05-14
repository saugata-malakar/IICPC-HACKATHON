# ─────────────────────────────────────────────────────────────────────────────
# Multi-stage build for Telemetry Ingester (Rust)
# ─────────────────────────────────────────────────────────────────────────────

FROM rust:1.77-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache \
    musl-dev \
    pkgconfig \
    openssl-dev \
    cmake \
    make \
    g++ \
    clang \
    llvm \
    linux-headers

# Copy manifests
COPY Cargo.toml ./

# Build dependencies (cached layer)
RUN mkdir src && \
    echo "fn main() {}" > src/main.rs && \
    cargo build --release && \
    rm -rf src

# Copy source
COPY src ./src

# Build application
RUN cargo build --release --bin telemetry-ingester

# ─────────────────────────────────────────────────────────────────────────────
# Runtime stage
# ─────────────────────────────────────────────────────────────────────────────

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    libgcc

COPY --from=builder /build/target/release/telemetry-ingester /app/telemetry-ingester

RUN addgroup -g 1000 iicpc && \
    adduser -D -u 1000 -G iicpc iicpc && \
    chown -R iicpc:iicpc /app

USER iicpc

EXPOSE 8004

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8004/health || exit 1

ENTRYPOINT ["/app/telemetry-ingester"]
