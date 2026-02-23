# Build stage
FROM golang:1.25-alpine AS builder
# Set working directory
WORKDIR /app
# Copy go mod files
COPY go.mod ./
# Download dependencies
RUN go mod download
# Copy source code
COPY . .
# Build the application for AMD64/x86_64 architecture
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o product-api .
# Final stage
FROM alpine:latest
# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates
WORKDIR /root/
# Copy the binary from builder
COPY --from=builder /app/product-api .
# Expose port 8080
EXPOSE 8080
# Run the application
CMD ["./product-api"]