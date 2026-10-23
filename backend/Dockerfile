# ==========================================
# NENAV API — Production Dockerfile
# Multi-stage build, minimal runtime, non-root
#
# Reference: Master Prompt §43 — Containerization
# ==========================================

# --- Build stage ---
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Cache deps
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source
COPY backend/ .

# Build static binary for ARM64
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /build/nenav-api \
    ./cmd/api

# --- Runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata curl && \
    addgroup -S nenav && \
    adduser -S -G nenav nenav

COPY --from=builder /build/nenav-api /usr/local/bin/nenav-api
COPY backend/migrations /app/migrations

WORKDIR /app
USER nenav

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["nenav-api"]
