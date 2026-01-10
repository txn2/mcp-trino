# syntax=docker/dockerfile:1

FROM alpine:3.21

# Install ca-certificates for TLS connections
RUN apk add --no-cache ca-certificates

# Copy the binary from goreleaser
COPY mcp-trino /usr/local/bin/mcp-trino

# Run as non-root user
RUN adduser -D -u 1000 mcp
USER mcp

ENTRYPOINT ["/usr/local/bin/mcp-trino"]
