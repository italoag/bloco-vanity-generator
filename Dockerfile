# Build arguments
ARG GO_VERSION=1.25.10
ARG ALPINE_VERSION=3.20

# Build stage
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

# Build arguments for version info
ARG VERSION=dev
ARG GIT_REV=unknown
ARG BUILD_DATE=unknown

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata

# Set working directory
WORKDIR /build

# Copy go modules files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=$(go env GOARCH) \
    go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${GIT_REV} -X main.date=${BUILD_DATE}" \
    -o bloco-eth \
    ./cmd/bloco-eth

# Verify the binary
RUN ./bloco-eth --help

# Runtime stage
FROM alpine:${ALPINE_VERSION}

# Re-declare build arguments for use in this stage
ARG VERSION=dev
ARG GIT_REV=unknown
ARG BUILD_DATE=unknown

# Install runtime dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata

# Create non-root user
RUN addgroup -g 1001 -S bloco-eth && \
    adduser -u 1001 -S bloco-eth -G bloco-eth

# Create necessary directories
RUN mkdir -p /app /home/bloco-eth && \
    chown -R bloco-eth:bloco-eth /app /home/bloco-eth

# Copy binary from builder
COPY --from=builder /build/bloco-eth /app/bloco-eth

# Copy CA certificates and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Set working directory
WORKDIR /home/bloco-eth

# Switch to non-root user
USER bloco-eth

# Set environment variables
ENV PATH="/app:${PATH}" \
    HOME="/home/bloco-eth" \
    USER="bloco-eth" \
    VERSION="${VERSION}" \
    GIT_REV="${GIT_REV}" \
    BUILD_DATE="${BUILD_DATE}"

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD bloco-eth --help || exit 1

# Add labels
LABEL org.opencontainers.image.title="bloco-eth" \
    org.opencontainers.image.description="An Ethereum like Wallet Generator" \
    org.opencontainers.image.url="https://github.com/italoag/bloco-wallet-generator" \
    org.opencontainers.image.source="https://github.com/italoag/bloco-wallet-generator" \
    org.opencontainers.image.version="${VERSION}" \
    org.opencontainers.image.revision="${GIT_REV}" \
    org.opencontainers.image.created="${BUILD_DATE}" \
    org.opencontainers.image.licenses="MIT" \
    org.opencontainers.image.vendor="Italo A. G."

# Default command
ENTRYPOINT ["bloco-eth"]
CMD ["--help"]