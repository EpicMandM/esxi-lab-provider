# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY api/go.mod api/go.sum ./
RUN go mod download

# Copy source code
COPY api/ ./

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /scheduler \
    ./cmd/server/main.go

# Create directories for the final stage
RUN mkdir -p /app-dirs/config /app-dirs/data

# Final stage - distroless static
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy binary from builder
COPY --from=builder /scheduler /app/scheduler

# Copy directory structure from builder
COPY --from=builder --chown=nonroot:nonroot /app-dirs/config /app/config
COPY --from=builder --chown=nonroot:nonroot /app-dirs/data /app/data

# Copy config files
COPY --chown=nonroot:nonroot api/data/user_config.toml /app/data/user_config.toml

# NOTE: Secrets should be mounted at runtime:
# - Service account: mount to /app/config/service-account.json
# - SMTP password: set via SMTP_PASSWORD environment variable

USER nonroot:nonroot

ENTRYPOINT ["/app/scheduler"]
