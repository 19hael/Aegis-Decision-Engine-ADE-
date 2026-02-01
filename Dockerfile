# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ade-server ./cmd/ade-server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ade-cli ./cmd/ade-cli

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 -S ade && \
    adduser -u 1000 -S ade -G ade

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/ade-server /app/
COPY --from=builder /build/ade-cli /app/
COPY --from=builder /build/policies /app/policies
COPY --from=builder /build/migrations /app/migrations

# Change ownership
RUN chown -R ade:ade /app

# Switch to non-root user
USER ade

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the server
CMD ["./ade-server"]
