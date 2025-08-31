FROM docker.io/library/golang:1.24-alpine AS builder

WORKDIR /app

# Install git for Go dependencies
RUN apk add --no-cache git

# Copy go.mod and go.sum files first for better layer caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with version information
ARG VERSION=0.0.0-dev

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w \
  -X http-mock-server/pkg/version.Version=${VERSION}" \
  -o /app/server ./cmd;

# Use distroless for minimal image size and better security
FROM gcr.io/distroless/static-debian12

WORKDIR /app

# Copy the binary and config from builder
COPY --from=builder /app/server /app/

# Run as non-root user (distroless has nonroot:nonroot with uid:gid 65532:65532)
USER nonroot:nonroot

# Set environment variables
ENV PORT=8080

# Expose port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/app/server"]
