# ─── Build Stage ────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy dependency files
COPY go.mod ./
# RUN go mod download

# Copy source code
COPY . .

# Download dependencies and build
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o chaos-engine .

# ─── Run Stage ──────────────────────────────────────────────────────────────
FROM alpine:latest

WORKDIR /app

# Install docker client, iptables, tc (netem)
RUN apk add --no-cache docker-cli iptables iproute2-tc stress-ng

# Copy binary from builder
COPY --from=builder /app/chaos-engine .

# Environment variables
ENV PORT=8005

EXPOSE 8005

CMD ["./chaos-engine"]
