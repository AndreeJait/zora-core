# ==========================================
# Stage 1: Build
# ==========================================
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server ./cmd/http
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/worker ./cmd/worker

# ==========================================
# Stage 2: Runtime
# ==========================================
FROM alpine:3.21

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    && rm -rf /var/cache/apk/*

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/bin/server /app/server
COPY --from=builder /app/bin/migrate /app/migrate
COPY --from=builder /app/bin/worker /app/worker

# Copy config and migration files
COPY --from=builder /app/files /app/files

# Expose HTTP port
EXPOSE 8080

ENTRYPOINT ["/app/server"]
CMD ["--config", "/app/files/config/app.yaml"]
