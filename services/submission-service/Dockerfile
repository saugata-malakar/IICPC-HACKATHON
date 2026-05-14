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

# Build the application
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o submission-service .

# ─── Run Stage ──────────────────────────────────────────────────────────────
FROM alpine:latest

WORKDIR /app

# Install docker client (to interact with host docker)
RUN apk add --no-cache docker-cli

# Copy binary from builder
COPY --from=builder /app/submission-service .

# Create uploads directory
RUN mkdir uploads

# Environment variables
ENV PORT=8001
ENV GIN_MODE=release

EXPOSE 8001

CMD ["./submission-service"]
