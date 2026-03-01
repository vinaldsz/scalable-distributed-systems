# Multi-stage build for Go binary
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy source code from src directory (context is parent dir)
COPY src/go.mod .
COPY src/*.go .

# Build the binary
RUN go build -o search-service .

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install curl for health checks
RUN apk add --no-cache curl

# Copy binary from builder
COPY --from=builder /app/search-service .

# Expose port
EXPOSE 8080 8081

# Run with argument for version (broken or fixed)
ENTRYPOINT ["./search-service"]
CMD ["broken"]
