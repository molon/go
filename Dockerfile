# Dockerfile for building the patched Go with error stack traces
#
# This Dockerfile builds Go from source with the error stack trace enhancements.
# It uses the official Go image as the bootstrap compiler.
#
# Build:
#   docker build -t go-errstack .
#
# Run tests:
#   docker build --target test -t go-errstack-test .
#   docker run --rm go-errstack-test
#
# Usage:
#   docker run --rm -v $(pwd):/app -w /app go-errstack go run main.go
#   docker run --rm -v $(pwd):/app -w /app go-errstack go build -o myapp main.go

# Use Go 1.24.12 as bootstrap (Go 1.25.x requires Go 1.24+)
# Note: bookworm is used for builder because it has better build toolchain support
FROM golang:1.24.12-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy the patched Go source
WORKDIR /go-src
COPY . .

# Build Go from source
# GOROOT_BOOTSTRAP points to the pre-installed Go in the base image
ENV GOROOT_BOOTSTRAP=/usr/local/go
WORKDIR /go-src/src
RUN ./make.bash

# Create the final image based on Alpine for smaller size
# Note: This image does NOT include gcc, matching official golang:alpine behavior
# Use the test stage if you need cgo support
FROM alpine:3.21

# Install runtime dependencies (no gcc, matching golang:alpine)
RUN apk add --no-cache \
    ca-certificates \
    git \
    bash

# Copy the built Go from builder
COPY --from=builder /go-src /usr/local/go

# Set up environment
ENV GOROOT=/usr/local/go
ENV GOPATH=/go
ENV PATH=$GOROOT/bin:$GOPATH/bin:$PATH

# Create workspace directory
RUN mkdir -p /go/src /go/bin /go/pkg

WORKDIR /workspace

# Verify the installation
RUN go version

# Default command
CMD ["go", "version"]

# Test stage - includes gcc for cgo support during comprehensive tests
FROM alpine:3.21 AS test

# Install build tools for cgo support during tests
RUN apk add --no-cache \
    ca-certificates \
    bash \
    build-base

COPY --from=builder /go-src /usr/local/go

ENV GOROOT=/usr/local/go
ENV PATH=$GOROOT/bin:$PATH

WORKDIR /usr/local/go/src

# Default: run the stack trace tests (quick test)
# For comprehensive tests, override with: sh ./run.bash --no-rebuild
CMD ["go", "test", "-v", "-run", "Test.*Stack", "errors"]
