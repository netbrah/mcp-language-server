# Multi-stage build for optimized Docker image

# Stage 1: Build the Go application
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mcp-language-server .

# Stage 2: Download and extract clangd
FROM alpine:latest AS clangd-downloader

# Download clangd release
# Using version 19.1.5 as a stable recent release
ARG CLANGD_VERSION=19.1.5
RUN apk add --no-cache wget unzip

WORKDIR /tmp
RUN wget https://github.com/clangd/clangd/releases/download/${CLANGD_VERSION}/clangd-linux-${CLANGD_VERSION}.zip && \
    unzip clangd-linux-${CLANGD_VERSION}.zip && \
    chmod +x clangd_${CLANGD_VERSION}/bin/clangd

# Stage 3: Final runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates libstdc++ libgcc

WORKDIR /app

# Copy the built application
COPY --from=builder /build/mcp-language-server /app/mcp-language-server

# Copy clangd binaries
COPY --from=clangd-downloader /tmp/clangd_*/bin/clangd /usr/local/bin/clangd
COPY --from=clangd-downloader /tmp/clangd_*/lib/clang /usr/local/lib/clang

# Create directory for workspace mounting
RUN mkdir -p /workspace

# Expose HTTP port
EXPOSE 8080

# Set default environment variables
ENV LOG_LEVEL=INFO
ENV TRANSPORT=http
ENV PORT=8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/health || exit 1

# Default command uses HTTP transport
# Users can override to use stdio by setting TRANSPORT=stdio
ENTRYPOINT ["/bin/sh", "-c", "/app/mcp-language-server --workspace /workspace --lsp clangd --transport ${TRANSPORT} --port ${PORT}"]
